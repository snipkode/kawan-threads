// Package filestore provides a local, JSON-file-backed implementation of all
// repository interfaces. It is intended as a development/demo datastore so the
// whole platform (API + worker + web) can run without Firebase credentials.
//
// Concurrency model:
//
//   - Every operation takes the store's mutex (write lock) — with per-row
//     filesystem persistence this is fast enough for a dev/demo datastore and
//     avoids read/write races entirely.
//   - Before operating, the latest database is reloaded from disk so that
//     changes made by ANOTHER process (e.g. the worker vs the API) are always
//     visible. Writes are applied to the freshly-loaded snapshot and then
//     flushed atomically (temp file + rename).
//
// This gives last-writer-wins semantics per operation across processes, which
// is more than sufficient for local development while remaining fully
// consistent within a single process.
package filestore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"kawan-threads/internal/domain/entity"
	"kawan-threads/internal/domain/repository"
)

// database mirrors the collection layout persisted in the JSON file.
type database struct {
	Content     map[string]entity.Content         `json:"content"`
	Versions    map[string]entity.ContentVersion  `json:"versions"`
	Queue       map[string]entity.QueueItem       `json:"queue"`
	Schedules   map[string]entity.Schedule        `json:"schedules"`
	Posts       map[string]entity.PublishedPost   `json:"posts"`
	Performance map[string]entity.PostPerformance `json:"performance"`
	Topics      map[string]entity.Topic           `json:"topics"`
	History     map[string]entity.History         `json:"history"`
	Experiments map[string]entity.Experiment      `json:"experiments"`
}

func newDatabase() *database {
	return &database{
		Content:     map[string]entity.Content{},
		Versions:    map[string]entity.ContentVersion{},
		Queue:       map[string]entity.QueueItem{},
		Schedules:   map[string]entity.Schedule{},
		Posts:       map[string]entity.PublishedPost{},
		Performance: map[string]entity.PostPerformance{},
		Topics:      map[string]entity.Topic{},
		History:     map[string]entity.History{},
		Experiments: map[string]entity.Experiment{},
	}
}

// Store is the root of the file-backed datastore. Each exported field is a
// repository implementation satisfying its corresponding domain interface.
type Store struct {
	mu   sync.Mutex
	path string
	db   *database

	Content        *ContentRepository
	ContentVersion *ContentVersionRepository
	Queue          *QueueRepository
	Schedule       *ScheduleRepository
	PublishedPost  *PublishedPostRepository
	Performance    *PostPerformanceRepository
	Topic          *TopicRepository
	History        *HistoryRepository
	Experiment     *ExperimentRepository
}

// Open loads (or creates) the file-backed store at path. It is safe to call
// from more than one process; the latest on-disk state is re-read on every
// operation.
func Open(path string) (*Store, error) {
	s := &Store{
		path: path,
		db:   newDatabase(),
	}
	s.ensureCollections()
	s.Content = &ContentRepository{s: s}
	s.ContentVersion = &ContentVersionRepository{s: s}
	s.Queue = &QueueRepository{s: s}
	s.Schedule = &ScheduleRepository{s: s}
	s.PublishedPost = &PublishedPostRepository{s: s}
	s.Performance = &PostPerformanceRepository{s: s}
	s.Topic = &TopicRepository{s: s}
	s.History = &HistoryRepository{s: s}
	s.Experiment = &ExperimentRepository{s: s}

	return s, nil
}

// ensureCollections normalises nil maps (covers empty/partial files).
func (s *Store) ensureCollections() {
	if s.db.Content == nil {
		s.db.Content = map[string]entity.Content{}
	}
	if s.db.Versions == nil {
		s.db.Versions = map[string]entity.ContentVersion{}
	}
	if s.db.Queue == nil {
		s.db.Queue = map[string]entity.QueueItem{}
	}
	if s.db.Schedules == nil {
		s.db.Schedules = map[string]entity.Schedule{}
	}
	if s.db.Posts == nil {
		s.db.Posts = map[string]entity.PublishedPost{}
	}
	if s.db.Performance == nil {
		s.db.Performance = map[string]entity.PostPerformance{}
	}
	if s.db.Topics == nil {
		s.db.Topics = map[string]entity.Topic{}
	}
	if s.db.History == nil {
		s.db.History = map[string]entity.History{}
	}
	if s.db.Experiments == nil {
		s.db.Experiments = map[string]entity.Experiment{}
	}
}

// reloadLocked re-reads the latest database from disk into s.db. The caller
// must hold s.mu. Missing/empty/unparseable files fall back to the current
// in-memory state (their first occurrence is treated as an empty store).
func (s *Store) reloadLocked() error {
	if s.path == "" {
		return nil
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("filestore: reading %s: %w", s.path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		s.db = newDatabase()
		s.ensureCollections()
		return nil
	}
	fresh := newDatabase()
	if err := json.Unmarshal(data, fresh); err != nil {
		// An unparseable file means a concurrent writer is mid-rename or the
		// file is corrupted; keep current state rather than failing the call.
		return nil
	}
	s.db = fresh
	s.ensureCollections()
	return nil
}

// snapshot returns the latest database after reloading from disk. Caller must
// hold s.mu.
func (s *Store) snapshot() (*database, error) {
	if err := s.reloadLocked(); err != nil {
		return nil, err
	}
	return s.db, nil
}

// flush atomically persists the in-memory database to disk. Caller holds s.mu.
func (s *Store) flush() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("filestore: mkdir: %w", err)
	}
	data, err := json.MarshalIndent(s.db, "", "  ")
	if err != nil {
		return fmt.Errorf("filestore: marshal: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("filestore: write tmp: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("filestore: rename: %w", err)
	}
	return nil
}

// clone deep-copies a value via a JSON round trip so callers can never mutate
// the stored copy.
func clone[T any](v T) (T, error) {
	var out T
	raw, err := json.Marshal(v)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// ContentRepository
// ---------------------------------------------------------------------------

type ContentRepository struct{ s *Store }

func (r *ContentRepository) Save(_ context.Context, content *entity.Content) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return err
	}
	db.Content[content.ID] = *content
	return r.s.flush()
}

// FindByID returns a copy of the content, or (nil, nil) when absent — matching
// the Firebase implementation's semantics.
func (r *ContentRepository) FindByID(_ context.Context, id string) (*entity.Content, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return nil, err
	}
	c, ok := db.Content[id]
	if !ok {
		return nil, nil
	}
	out, err := clone(c)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *ContentRepository) FindAll(_ context.Context, filter repository.ContentFilter) ([]*entity.Content, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return nil, err
	}
	result := make([]*entity.Content, 0, len(db.Content))
	for _, c := range db.Content {
		if filter.Status != nil && c.Status != *filter.Status {
			continue
		}
		if filter.Pillar != nil && c.Pillar != *filter.Pillar {
			continue
		}
		cp, err := clone(c)
		if err != nil {
			return nil, err
		}
		result = append(result, &cp)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return paginate(result, filter.Offset, filter.Limit), nil
}

func (r *ContentRepository) UpdateStatus(_ context.Context, id string, status entity.ContentStatus) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return err
	}
	c, ok := db.Content[id]
	if !ok {
		return nil
	}
	c.Status = status
	c.UpdatedAt = time.Now().UTC()
	db.Content[id] = c
	return r.s.flush()
}

func (r *ContentRepository) Update(_ context.Context, content *entity.Content) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return err
	}
	existing, ok := db.Content[content.ID]
	if !ok {
		return nil
	}
	content.CreatedAt = existing.CreatedAt
	content.UpdatedAt = time.Now().UTC()
	db.Content[content.ID] = *content
	return r.s.flush()
}

func (r *ContentRepository) Delete(_ context.Context, id string) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return err
	}
	delete(db.Content, id)
	return r.s.flush()
}

// ---------------------------------------------------------------------------
// ContentVersionRepository
// ---------------------------------------------------------------------------

type ContentVersionRepository struct{ s *Store }

func (r *ContentVersionRepository) Save(_ context.Context, version *entity.ContentVersion) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return err
	}
	db.Versions[key(version.ContentID, version.ID)] = *version
	return r.s.flush()
}

func (r *ContentVersionRepository) FindByContentID(_ context.Context, contentID string) ([]*entity.ContentVersion, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return nil, err
	}
	result := make([]*entity.ContentVersion, 0)
	for _, v := range db.Versions {
		if v.ContentID != contentID {
			continue
		}
		cp, err := clone(v)
		if err != nil {
			return nil, err
		}
		result = append(result, &cp)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Version < result[j].Version
	})
	return result, nil
}

// ---------------------------------------------------------------------------
// QueueRepository
// ---------------------------------------------------------------------------

type QueueRepository struct{ s *Store }

func (r *QueueRepository) Enqueue(_ context.Context, item *entity.QueueItem) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return err
	}
	db.Queue[item.ContentID] = *item
	return r.s.flush()
}

func (r *QueueRepository) Dequeue(_ context.Context, limit int) ([]*entity.QueueItem, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return nil, err
	}
	items := make([]*entity.QueueItem, 0, len(db.Queue))
	for _, i := range db.Queue {
		cp, err := clone(i)
		if err != nil {
			return nil, err
		}
		items = append(items, &cp)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Priority != items[j].Priority {
			return items[i].Priority > items[j].Priority
		}
		return items[i].QueuedAt.Before(items[j].QueuedAt)
	})
	if limit > 0 && limit < len(items) {
		return items[:limit], nil
	}
	return items, nil
}

func (r *QueueRepository) FindByContentID(_ context.Context, contentID string) (*entity.QueueItem, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return nil, err
	}
	item, ok := db.Queue[contentID]
	if !ok {
		return nil, nil
	}
	out, err := clone(item)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *QueueRepository) Remove(_ context.Context, contentID string) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return err
	}
	delete(db.Queue, contentID)
	return r.s.flush()
}

func (r *QueueRepository) UpdatePriority(_ context.Context, contentID string, priority int) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return err
	}
	item, ok := db.Queue[contentID]
	if !ok {
		return nil
	}
	item.Priority = priority
	db.Queue[contentID] = item
	return r.s.flush()
}

func (r *QueueRepository) FindAll(_ context.Context) ([]*entity.QueueItem, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return nil, err
	}
	result := make([]*entity.QueueItem, 0, len(db.Queue))
	for _, i := range db.Queue {
		cp, err := clone(i)
		if err != nil {
			return nil, err
		}
		result = append(result, &cp)
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// ScheduleRepository
// ---------------------------------------------------------------------------

type ScheduleRepository struct{ s *Store }

func (r *ScheduleRepository) Save(_ context.Context, schedule *entity.Schedule) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return err
	}
	db.Schedules[schedule.ID] = *schedule
	return r.s.flush()
}

func (r *ScheduleRepository) FindByID(_ context.Context, id string) (*entity.Schedule, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return nil, err
	}
	s, ok := db.Schedules[id]
	if !ok {
		return nil, nil
	}
	out, err := clone(s)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *ScheduleRepository) FindScheduled(_ context.Context, before time.Time) ([]*entity.Schedule, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return nil, err
	}
	result := make([]*entity.Schedule, 0)
	for _, s := range db.Schedules {
		if s.ScheduledAt.Before(before) || s.ScheduledAt.Equal(before) {
			cp, err := clone(s)
			if err != nil {
				return nil, err
			}
			result = append(result, &cp)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ScheduledAt.Before(result[j].ScheduledAt)
	})
	return result, nil
}

func (r *ScheduleRepository) FindAll(_ context.Context) ([]*entity.Schedule, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return nil, err
	}
	result := make([]*entity.Schedule, 0, len(db.Schedules))
	for _, s := range db.Schedules {
		cp, err := clone(s)
		if err != nil {
			return nil, err
		}
		result = append(result, &cp)
	}
	return result, nil
}

func (r *ScheduleRepository) Delete(_ context.Context, id string) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return err
	}
	delete(db.Schedules, id)
	return r.s.flush()
}

// ---------------------------------------------------------------------------
// PublishedPostRepository
// ---------------------------------------------------------------------------

type PublishedPostRepository struct{ s *Store }

func (r *PublishedPostRepository) Save(_ context.Context, post *entity.PublishedPost) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return err
	}
	db.Posts[post.ID] = *post
	return r.s.flush()
}

func (r *PublishedPostRepository) FindByContentID(_ context.Context, contentID string) (*entity.PublishedPost, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return nil, err
	}
	for _, p := range db.Posts {
		if p.ContentID == contentID {
			out, err := clone(p)
			if err != nil {
				return nil, err
			}
			return &out, nil
		}
	}
	return nil, nil
}

func (r *PublishedPostRepository) FindAll(_ context.Context) ([]*entity.PublishedPost, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return nil, err
	}
	result := make([]*entity.PublishedPost, 0, len(db.Posts))
	for _, p := range db.Posts {
		cp, err := clone(p)
		if err != nil {
			return nil, err
		}
		result = append(result, &cp)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].PublishedAt.After(result[j].PublishedAt)
	})
	return result, nil
}

func (r *PublishedPostRepository) FindByIdempotencyKey(_ context.Context, key string) (*entity.PublishedPost, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return nil, err
	}
	for _, p := range db.Posts {
		if p.IdempotencyKey == key {
			out, err := clone(p)
			if err != nil {
				return nil, err
			}
			return &out, nil
		}
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// PostPerformanceRepository
// ---------------------------------------------------------------------------

type PostPerformanceRepository struct{ s *Store }

func (r *PostPerformanceRepository) Save(_ context.Context, perf *entity.PostPerformance) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return err
	}
	db.Performance[perf.PostID] = *perf
	return r.s.flush()
}

func (r *PostPerformanceRepository) FindByPostID(_ context.Context, postID string) (*entity.PostPerformance, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return nil, err
	}
	p, ok := db.Performance[postID]
	if !ok {
		return nil, nil
	}
	out, err := clone(p)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *PostPerformanceRepository) FindAll(_ context.Context) ([]*entity.PostPerformance, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return nil, err
	}
	result := make([]*entity.PostPerformance, 0, len(db.Performance))
	for _, p := range db.Performance {
		cp, err := clone(p)
		if err != nil {
			return nil, err
		}
		result = append(result, &cp)
	}
	return result, nil
}

func (r *PostPerformanceRepository) FindByPillar(_ context.Context, pillar entity.ContentPillar) ([]*entity.PostPerformance, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return nil, err
	}
	result := make([]*entity.PostPerformance, 0)
	for _, p := range db.Performance {
		if p.Pillar != pillar {
			continue
		}
		cp, err := clone(p)
		if err != nil {
			return nil, err
		}
		result = append(result, &cp)
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// TopicRepository
// ---------------------------------------------------------------------------

type TopicRepository struct{ s *Store }

func (r *TopicRepository) Save(_ context.Context, topic *entity.Topic) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return err
	}
	db.Topics[topic.ID] = *topic
	return r.s.flush()
}

func (r *TopicRepository) FindAll(_ context.Context) ([]*entity.Topic, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return nil, err
	}
	result := make([]*entity.Topic, 0, len(db.Topics))
	for _, t := range db.Topics {
		cp, err := clone(t)
		if err != nil {
			return nil, err
		}
		result = append(result, &cp)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

func (r *TopicRepository) FindByID(_ context.Context, id string) (*entity.Topic, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return nil, err
	}
	t, ok := db.Topics[id]
	if !ok {
		return nil, nil
	}
	out, err := clone(t)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------------------------------------------------------------------------
// HistoryRepository
// ---------------------------------------------------------------------------

type HistoryRepository struct{ s *Store }

func (r *HistoryRepository) Save(_ context.Context, h *entity.History) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return err
	}
	db.History[h.ID] = *h
	return r.s.flush()
}

func (r *HistoryRepository) FindByContentID(_ context.Context, contentID string) ([]*entity.History, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return nil, err
	}
	result := make([]*entity.History, 0)
	for _, h := range db.History {
		if h.ContentID != contentID {
			continue
		}
		cp, err := clone(h)
		if err != nil {
			return nil, err
		}
		result = append(result, &cp)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

// ---------------------------------------------------------------------------
// ExperimentRepository
// ---------------------------------------------------------------------------

type ExperimentRepository struct{ s *Store }

func (r *ExperimentRepository) Save(_ context.Context, exp *entity.Experiment) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return err
	}
	db.Experiments[exp.ID] = *exp
	return r.s.flush()
}

func (r *ExperimentRepository) FindByID(_ context.Context, id string) (*entity.Experiment, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return nil, err
	}
	e, ok := db.Experiments[id]
	if !ok {
		return nil, nil
	}
	out, err := clone(e)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *ExperimentRepository) FindAll(_ context.Context) ([]*entity.Experiment, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	db, err := r.s.snapshot()
	if err != nil {
		return nil, err
	}
	result := make([]*entity.Experiment, 0, len(db.Experiments))
	for _, e := range db.Experiments {
		cp, err := clone(e)
		if err != nil {
			return nil, err
		}
		result = append(result, &cp)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartedAt.Before(result[j].StartedAt)
	})
	return result, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func key(parts ...string) string {
	return strings.Join(parts, "::")
}

func paginate[T any](list []T, offset, limit int) []T {
	if offset > 0 {
		if offset >= len(list) {
			return []T{}
		}
		list = list[offset:]
	}
	if limit > 0 && limit < len(list) {
		list = list[:limit]
	}
	return list
}
