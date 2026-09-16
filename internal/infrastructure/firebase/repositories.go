package firebase

import (
	"context"
	"fmt"
	"sort"
	"time"

	"kawan-threads/internal/domain/entity"
	"kawan-threads/internal/domain/repository"
)

// ---------------------------------------------------------------------------
// ContentFirebaseRepository
// ---------------------------------------------------------------------------

// ContentFirebaseRepository implements repository.ContentRepository using
// Firebase RTDB at path threads/content/{id}.
type ContentFirebaseRepository struct {
	client *FirebaseClient
}

func NewContentRepository(client *FirebaseClient) *ContentFirebaseRepository {
	return &ContentFirebaseRepository{client: client}
}

func (r *ContentFirebaseRepository) Save(ctx context.Context, content *entity.Content) error {
	return r.client.Set(ctx, fmt.Sprintf("threads/content/%s", content.ID), content)
}

func (r *ContentFirebaseRepository) FindByID(ctx context.Context, id string) (*entity.Content, error) {
	var c entity.Content
	if err := r.client.Get(ctx, fmt.Sprintf("threads/content/%s", id), &c); err != nil {
		return nil, err
	}
	if c.ID == "" {
		return nil, nil
	}
	return &c, nil
}

func (r *ContentFirebaseRepository) FindAll(ctx context.Context, filter repository.ContentFilter) ([]*entity.Content, error) {
	var raw map[string]entity.Content
	if err := r.client.GetAll(ctx, "threads/content", &raw); err != nil {
		return nil, err
	}

	result := make([]*entity.Content, 0, len(raw))
	for _, c := range raw {
		c := c // capture
		if filter.Status != nil && c.Status != *filter.Status {
			continue
		}
		if filter.Pillar != nil && c.Pillar != *filter.Pillar {
			continue
		}
		result = append(result, &c)
	}

	// Deterministic ordering by creation time descending.
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	// Pagination.
	if filter.Offset > 0 {
		if filter.Offset >= len(result) {
			return []*entity.Content{}, nil
		}
		result = result[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(result) {
		result = result[:filter.Limit]
	}

	return result, nil
}

func (r *ContentFirebaseRepository) UpdateStatus(ctx context.Context, id string, status entity.ContentStatus) error {
	return r.client.Update(ctx, fmt.Sprintf("threads/content/%s", id), map[string]interface{}{
		"status":     status,
		"updated_at": time.Now().UTC(),
	})
}

func (r *ContentFirebaseRepository) Update(ctx context.Context, content *entity.Content) error {
	content.UpdatedAt = time.Now().UTC()
	return r.client.Set(ctx, fmt.Sprintf("threads/content/%s", content.ID), content)
}

func (r *ContentFirebaseRepository) Delete(ctx context.Context, id string) error {
	return r.client.Delete(ctx, fmt.Sprintf("threads/content/%s", id))
}

// ---------------------------------------------------------------------------
// ContentVersionFirebaseRepository
// ---------------------------------------------------------------------------

// ContentVersionFirebaseRepository implements repository.ContentVersionRepository
// using Firebase RTDB at path threads/content/{contentId}/versions/{versionId}.
type ContentVersionFirebaseRepository struct {
	client *FirebaseClient
}

func NewContentVersionRepository(client *FirebaseClient) *ContentVersionFirebaseRepository {
	return &ContentVersionFirebaseRepository{client: client}
}

func (r *ContentVersionFirebaseRepository) Save(ctx context.Context, version *entity.ContentVersion) error {
	path := fmt.Sprintf("threads/content/%s/versions/%s", version.ContentID, version.ID)
	return r.client.Set(ctx, path, version)
}

func (r *ContentVersionFirebaseRepository) FindByContentID(ctx context.Context, contentID string) ([]*entity.ContentVersion, error) {
	var raw map[string]entity.ContentVersion
	if err := r.client.GetAll(ctx, fmt.Sprintf("threads/content/%s/versions", contentID), &raw); err != nil {
		return nil, err
	}

	result := make([]*entity.ContentVersion, 0, len(raw))
	for _, v := range raw {
		v := v
		result = append(result, &v)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Version < result[j].Version
	})
	return result, nil
}

// ---------------------------------------------------------------------------
// QueueFirebaseRepository
// ---------------------------------------------------------------------------

// QueueFirebaseRepository implements repository.QueueRepository using
// Firebase RTDB at path threads/queue/{contentId}.
type QueueFirebaseRepository struct {
	client *FirebaseClient
}

func NewQueueRepository(client *FirebaseClient) *QueueFirebaseRepository {
	return &QueueFirebaseRepository{client: client}
}

func (r *QueueFirebaseRepository) Enqueue(ctx context.Context, item *entity.QueueItem) error {
	return r.client.Set(ctx, fmt.Sprintf("threads/queue/%s", item.ContentID), item)
}

func (r *QueueFirebaseRepository) Dequeue(ctx context.Context, limit int) ([]*entity.QueueItem, error) {
	all, err := r.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	// Sort by priority descending, then by QueuedAt ascending.
	sort.Slice(all, func(i, j int) bool {
		if all[i].Priority != all[j].Priority {
			return all[i].Priority > all[j].Priority
		}
		return all[i].QueuedAt.Before(all[j].QueuedAt)
	})

	if limit > 0 && limit < len(all) {
		return all[:limit], nil
	}
	return all, nil
}

func (r *QueueFirebaseRepository) FindByContentID(ctx context.Context, contentID string) (*entity.QueueItem, error) {
	var item entity.QueueItem
	if err := r.client.Get(ctx, fmt.Sprintf("threads/queue/%s", contentID), &item); err != nil {
		return nil, err
	}
	if item.ContentID == "" {
		return nil, nil
	}
	return &item, nil
}

func (r *QueueFirebaseRepository) Remove(ctx context.Context, contentID string) error {
	return r.client.Delete(ctx, fmt.Sprintf("threads/queue/%s", contentID))
}

func (r *QueueFirebaseRepository) UpdatePriority(ctx context.Context, contentID string, priority int) error {
	return r.client.Update(ctx, fmt.Sprintf("threads/queue/%s", contentID), map[string]interface{}{
		"priority": priority,
	})
}

func (r *QueueFirebaseRepository) FindAll(ctx context.Context) ([]*entity.QueueItem, error) {
	var raw map[string]entity.QueueItem
	if err := r.client.GetAll(ctx, "threads/queue", &raw); err != nil {
		return nil, err
	}
	result := make([]*entity.QueueItem, 0, len(raw))
	for _, item := range raw {
		item := item
		result = append(result, &item)
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// ScheduleFirebaseRepository
// ---------------------------------------------------------------------------

// ScheduleFirebaseRepository implements repository.ScheduleRepository using
// Firebase RTDB at path threads/schedules/{id}.
type ScheduleFirebaseRepository struct {
	client *FirebaseClient
}

func NewScheduleRepository(client *FirebaseClient) *ScheduleFirebaseRepository {
	return &ScheduleFirebaseRepository{client: client}
}

func (r *ScheduleFirebaseRepository) Save(ctx context.Context, schedule *entity.Schedule) error {
	return r.client.Set(ctx, fmt.Sprintf("threads/schedules/%s", schedule.ID), schedule)
}

func (r *ScheduleFirebaseRepository) FindByID(ctx context.Context, id string) (*entity.Schedule, error) {
	var s entity.Schedule
	if err := r.client.Get(ctx, fmt.Sprintf("threads/schedules/%s", id), &s); err != nil {
		return nil, err
	}
	if s.ID == "" {
		return nil, nil
	}
	return &s, nil
}

func (r *ScheduleFirebaseRepository) FindScheduled(ctx context.Context, before time.Time) ([]*entity.Schedule, error) {
	all, err := r.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*entity.Schedule, 0)
	for _, s := range all {
		if s.ScheduledAt.Before(before) || s.ScheduledAt.Equal(before) {
			result = append(result, s)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ScheduledAt.Before(result[j].ScheduledAt)
	})
	return result, nil
}

func (r *ScheduleFirebaseRepository) FindAll(ctx context.Context) ([]*entity.Schedule, error) {
	var raw map[string]entity.Schedule
	if err := r.client.GetAll(ctx, "threads/schedules", &raw); err != nil {
		return nil, err
	}
	result := make([]*entity.Schedule, 0, len(raw))
	for _, s := range raw {
		s := s
		result = append(result, &s)
	}
	return result, nil
}

func (r *ScheduleFirebaseRepository) Delete(ctx context.Context, id string) error {
	return r.client.Delete(ctx, fmt.Sprintf("threads/schedules/%s", id))
}

// ---------------------------------------------------------------------------
// PublishedPostFirebaseRepository
// ---------------------------------------------------------------------------

// PublishedPostFirebaseRepository implements repository.PublishedPostRepository
// using Firebase RTDB at path threads/posts/{id}.
type PublishedPostFirebaseRepository struct {
	client *FirebaseClient
}

func NewPublishedPostRepository(client *FirebaseClient) *PublishedPostFirebaseRepository {
	return &PublishedPostFirebaseRepository{client: client}
}

func (r *PublishedPostFirebaseRepository) Save(ctx context.Context, post *entity.PublishedPost) error {
	return r.client.Set(ctx, fmt.Sprintf("threads/posts/%s", post.ID), post)
}

func (r *PublishedPostFirebaseRepository) FindByContentID(ctx context.Context, contentID string) (*entity.PublishedPost, error) {
	var raw map[string]entity.PublishedPost
	if err := r.client.GetAll(ctx, "threads/posts", &raw); err != nil {
		return nil, err
	}
	for _, post := range raw {
		if post.ContentID == contentID {
			p := post
			return &p, nil
		}
	}
	return nil, nil
}

func (r *PublishedPostFirebaseRepository) FindAll(ctx context.Context) ([]*entity.PublishedPost, error) {
	var raw map[string]entity.PublishedPost
	if err := r.client.GetAll(ctx, "threads/posts", &raw); err != nil {
		return nil, err
	}
	result := make([]*entity.PublishedPost, 0, len(raw))
	for _, p := range raw {
		p := p
		result = append(result, &p)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].PublishedAt.After(result[j].PublishedAt)
	})
	return result, nil
}

func (r *PublishedPostFirebaseRepository) FindByIdempotencyKey(ctx context.Context, key string) (*entity.PublishedPost, error) {
	var raw map[string]entity.PublishedPost
	if err := r.client.GetAll(ctx, "threads/posts", &raw); err != nil {
		return nil, err
	}
	for _, post := range raw {
		if post.IdempotencyKey == key {
			p := post
			return &p, nil
		}
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// PostPerformanceFirebaseRepository
// ---------------------------------------------------------------------------

// PostPerformanceFirebaseRepository implements repository.PostPerformanceRepository
// using Firebase RTDB at path threads/performance/{postId}.
type PostPerformanceFirebaseRepository struct {
	client *FirebaseClient
}

func NewPostPerformanceRepository(client *FirebaseClient) *PostPerformanceFirebaseRepository {
	return &PostPerformanceFirebaseRepository{client: client}
}

func (r *PostPerformanceFirebaseRepository) Save(ctx context.Context, perf *entity.PostPerformance) error {
	return r.client.Set(ctx, fmt.Sprintf("threads/performance/%s", perf.PostID), perf)
}

func (r *PostPerformanceFirebaseRepository) FindByPostID(ctx context.Context, postID string) (*entity.PostPerformance, error) {
	var p entity.PostPerformance
	if err := r.client.Get(ctx, fmt.Sprintf("threads/performance/%s", postID), &p); err != nil {
		return nil, err
	}
	if p.PostID == "" {
		return nil, nil
	}
	return &p, nil
}

func (r *PostPerformanceFirebaseRepository) FindAll(ctx context.Context) ([]*entity.PostPerformance, error) {
	var raw map[string]entity.PostPerformance
	if err := r.client.GetAll(ctx, "threads/performance", &raw); err != nil {
		return nil, err
	}
	result := make([]*entity.PostPerformance, 0, len(raw))
	for _, p := range raw {
		p := p
		result = append(result, &p)
	}
	return result, nil
}

func (r *PostPerformanceFirebaseRepository) FindByPillar(ctx context.Context, pillar entity.ContentPillar) ([]*entity.PostPerformance, error) {
	all, err := r.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*entity.PostPerformance, 0)
	for _, p := range all {
		if p.Pillar == pillar {
			result = append(result, p)
		}
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// TopicFirebaseRepository
// ---------------------------------------------------------------------------

// TopicFirebaseRepository implements repository.TopicRepository using
// Firebase RTDB at path threads/topics/{id}.
type TopicFirebaseRepository struct {
	client *FirebaseClient
}

func NewTopicRepository(client *FirebaseClient) *TopicFirebaseRepository {
	return &TopicFirebaseRepository{client: client}
}

func (r *TopicFirebaseRepository) Save(ctx context.Context, topic *entity.Topic) error {
	return r.client.Set(ctx, fmt.Sprintf("threads/topics/%s", topic.ID), topic)
}

func (r *TopicFirebaseRepository) FindAll(ctx context.Context) ([]*entity.Topic, error) {
	var raw map[string]entity.Topic
	if err := r.client.GetAll(ctx, "threads/topics", &raw); err != nil {
		return nil, err
	}
	result := make([]*entity.Topic, 0, len(raw))
	for _, t := range raw {
		t := t
		result = append(result, &t)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

func (r *TopicFirebaseRepository) FindByID(ctx context.Context, id string) (*entity.Topic, error) {
	var t entity.Topic
	if err := r.client.Get(ctx, fmt.Sprintf("threads/topics/%s", id), &t); err != nil {
		return nil, err
	}
	if t.ID == "" {
		return nil, nil
	}
	return &t, nil
}

// ---------------------------------------------------------------------------
// HistoryFirebaseRepository
// ---------------------------------------------------------------------------

// HistoryFirebaseRepository implements repository.HistoryRepository using
// Firebase RTDB at path threads/history/{id}.
type HistoryFirebaseRepository struct {
	client *FirebaseClient
}

func NewHistoryRepository(client *FirebaseClient) *HistoryFirebaseRepository {
	return &HistoryFirebaseRepository{client: client}
}

func (r *HistoryFirebaseRepository) Save(ctx context.Context, h *entity.History) error {
	return r.client.Set(ctx, fmt.Sprintf("threads/history/%s", h.ID), h)
}

func (r *HistoryFirebaseRepository) FindByContentID(ctx context.Context, contentID string) ([]*entity.History, error) {
	var raw map[string]entity.History
	if err := r.client.GetAll(ctx, "threads/history", &raw); err != nil {
		return nil, err
	}
	result := make([]*entity.History, 0)
	for _, h := range raw {
		if h.ContentID == contentID {
			h := h
			result = append(result, &h)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

// ---------------------------------------------------------------------------
// ExperimentFirebaseRepository
// ---------------------------------------------------------------------------

// ExperimentFirebaseRepository implements repository.ExperimentRepository
// using Firebase RTDB at path threads/experiments/{id}.
type ExperimentFirebaseRepository struct {
	client *FirebaseClient
}

func NewExperimentRepository(client *FirebaseClient) *ExperimentFirebaseRepository {
	return &ExperimentFirebaseRepository{client: client}
}

func (r *ExperimentFirebaseRepository) Save(ctx context.Context, exp *entity.Experiment) error {
	return r.client.Set(ctx, fmt.Sprintf("threads/experiments/%s", exp.ID), exp)
}

func (r *ExperimentFirebaseRepository) FindByID(ctx context.Context, id string) (*entity.Experiment, error) {
	var e entity.Experiment
	if err := r.client.Get(ctx, fmt.Sprintf("threads/experiments/%s", id), &e); err != nil {
		return nil, err
	}
	if e.ID == "" {
		return nil, nil
	}
	return &e, nil
}

func (r *ExperimentFirebaseRepository) FindAll(ctx context.Context) ([]*entity.Experiment, error) {
	var raw map[string]entity.Experiment
	if err := r.client.GetAll(ctx, "threads/experiments", &raw); err != nil {
		return nil, err
	}
	result := make([]*entity.Experiment, 0, len(raw))
	for _, e := range raw {
		e := e
		result = append(result, &e)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartedAt.Before(result[j].StartedAt)
	})
	return result, nil
}
