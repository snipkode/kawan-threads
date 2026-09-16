package filestore

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"kawan-threads/internal/domain/entity"
	"kawan-threads/internal/domain/repository"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "db.json"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return s
}

func TestContentRepository_CRUDAndFilter(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	a := &entity.Content{ID: "a", Pillar: entity.PillarEdukasiHukum, Status: entity.StatusDraft, Hook: "A", CreatedAt: time.Now().Add(-2 * time.Hour)}
	b := &entity.Content{ID: "b", Pillar: entity.PillarEvent, Status: entity.StatusQueued, Hook: "B", CreatedAt: time.Now().Add(-time.Hour)}
	c := &entity.Content{ID: "c", Pillar: entity.PillarEdukasiHukum, Status: entity.StatusDraft, Hook: "C", CreatedAt: time.Now()}

	for _, item := range []*entity.Content{a, b, c} {
		if err := s.Content.Save(ctx, item); err != nil {
			t.Fatalf("Save(%s): %v", item.ID, err)
		}
	}

	byID, err := s.Content.FindByID(ctx, "a")
	if err != nil || byID == nil || byID.Hook != "A" {
		t.Fatalf("FindByID = %v, %v", byID, err)
	}

	missing, err := s.Content.FindByID(ctx, "missing")
	if err != nil || missing != nil {
		t.Fatalf("FindByID(missing) = %v, %v; want nil,nil", missing, err)
	}

	// Filter by status, sorted by CreatedAt desc.
	draft := entity.StatusDraft
	byStatus, _ := s.Content.FindAll(ctx, repository.ContentFilter{Status: &draft, Limit: 0})
	if len(byStatus) != 2 || byStatus[0].ID != "c" || byStatus[1].ID != "a" {
		t.Fatalf("FindAll(status) order wrong: %+v", byStatus)
	}

	// Limit + offset.
	all, _ := s.Content.FindAll(ctx, repository.ContentFilter{Limit: 1, Offset: 1})
	if len(all) != 1 || all[0].ID != "b" {
		t.Fatalf("FindAll(paginate) = %+v", all)
	}

	// UpdateStatus persists.
	if err := s.Content.UpdateStatus(ctx, "a", entity.StatusScheduled); err != nil {
		t.Fatal(err)
	}
	updated, _ := s.Content.FindByID(ctx, "a")
	if updated.Status != entity.StatusScheduled || updated.UpdatedAt.IsZero() {
		t.Fatalf("UpdateStatus not reflected: %+v", updated)
	}

	// Update merges + bumps timestamp, preserves CreatedAt.
	orig := updated.CreatedAt
	updated.Hook = "A2"
	if err := s.Content.Update(ctx, updated); err != nil {
		t.Fatal(err)
	}
	again, _ := s.Content.FindByID(ctx, "a")
	if again.Hook != "A2" || !again.CreatedAt.Equal(orig) {
		t.Fatalf("Update not merged: %+v", again)
	}

	// Delete.
	if err := s.Content.Delete(ctx, "c"); err != nil {
		t.Fatal(err)
	}
	afterDelete, _ := s.Content.FindAll(ctx, repository.ContentFilter{Limit: 0})
	if len(afterDelete) != 2 {
		t.Fatalf("after delete len = %d", len(afterDelete))
	}
}

func TestContentRepository_ReturnsCopies(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	got, err := s.Content.FindByID(ctx, "nope")
	if err != nil || got != nil {
		t.Fatalf("FindByID missing = %v, %v", got, err)
	}

	src := &entity.Content{
		ID: "x", Status: entity.StatusDraft,
		HookVariants: []entity.HookVariant{{Hook: "v1", HookType: entity.HookQuestion}},
	}
	if err := s.Content.Save(ctx, src); err != nil {
		t.Fatal(err)
	}

	got, _ = s.Content.FindByID(ctx, "x")
	if len(got.HookVariants) != 1 || got.HookVariants[0].Hook != "v1" {
		t.Fatalf("hook variants not round-tripped: %+v", got.HookVariants)
	}
	got.HookVariants[0].Hook = "mutated"
	again, _ := s.Content.FindByID(ctx, "x")
	if again.HookVariants[0].Hook == "mutated" {
		t.Fatal("returned content shares backing storage")
	}
}

func TestQueueRepository_PriorityOrderAndRemove(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	low := &entity.QueueItem{ContentID: "low", Priority: 20, QueuedAt: time.Now()}
	highOld := &entity.QueueItem{ContentID: "hi-old", Priority: 80, QueuedAt: time.Now().Add(-time.Hour)}
	highNew := &entity.QueueItem{ContentID: "hi-new", Priority: 80, QueuedAt: time.Now()}
	for _, i := range []*entity.QueueItem{low, highNew, highOld} {
		if err := s.Queue.Enqueue(ctx, i); err != nil {
			t.Fatal(err)
		}
	}

	items, err := s.Queue.Dequeue(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("Dequeue len = %d", len(items))
	}
	// Priority desc, then QueuedAt asc.
	if items[0].ContentID != "hi-old" || items[1].ContentID != "hi-new" || items[2].ContentID != "low" {
		t.Fatalf("Dequeue order wrong: %+v", items)
	}

	if err := s.Queue.UpdatePriority(ctx, "low", 100); err != nil {
		t.Fatal(err)
	}
	top, _ := s.Queue.Dequeue(ctx, 1)
	if top[0].ContentID != "low" {
		t.Fatalf("priority update not respected: %q", top[0].ContentID)
	}

	item, _ := s.Queue.FindByContentID(ctx, "low")
	if item == nil || item.Priority != 100 {
		t.Fatalf("FindByContentID = %+v", item)
	}

	if err := s.Queue.Remove(ctx, "low"); err != nil {
		t.Fatal(err)
	}
	if item, _ := s.Queue.FindByContentID(ctx, "low"); item != nil {
		t.Fatalf("Remove failed: %+v", item)
	}
}

func TestScheduleRepository_FindScheduled(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	now := time.Now()
	future := &entity.Schedule{ID: "f", ContentID: "f", ScheduledAt: now.Add(2 * time.Hour)}
	past := &entity.Schedule{ID: "p", ContentID: "p", ScheduledAt: now.Add(30 * time.Minute)}
	if err := s.Schedule.Save(ctx, future); err != nil {
		t.Fatal(err)
	}
	if err := s.Schedule.Save(ctx, past); err != nil {
		t.Fatal(err)
	}

	due, _ := s.Schedule.FindScheduled(ctx, now.Add(time.Hour))
	if len(due) != 1 || due[0].ID != "p" {
		t.Fatalf("FindScheduled = %+v", due)
	}

	if err := s.Schedule.Delete(ctx, "p"); err != nil {
		t.Fatal(err)
	}
	all, _ := s.Schedule.FindAll(ctx)
	if len(all) != 1 {
		t.Fatalf("after delete len = %d", len(all))
	}
}

func TestPersistenceAcrossInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db.json")
	ctx := context.Background()

	first, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Content.Save(ctx, &entity.Content{ID: "persisted", Status: entity.StatusDraft, Hook: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := first.Queue.Enqueue(ctx, &entity.QueueItem{ContentID: "persisted", Priority: 50}); err != nil {
		t.Fatal(err)
	}
	if err := first.Topic.Save(ctx, &entity.Topic{ID: "t1", Name: "Topik", Active: true}); err != nil {
		t.Fatal(err)
	}

	second, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if c, _ := second.Content.FindByID(ctx, "persisted"); c == nil || c.Hook != "x" {
		t.Fatalf("content not persisted: %+v", c)
	}
	if q, _ := second.Queue.FindByContentID(ctx, "persisted"); q == nil || q.Priority != 50 {
		t.Fatalf("queue not persisted: %+v", q)
	}
	if topics, _ := second.Topic.FindAll(ctx); len(topics) != 1 {
		t.Fatalf("topics not persisted: %+v", topics)
	}
}

func TestPublishedPost_Idempotency(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	if err := s.PublishedPost.Save(ctx, &entity.PublishedPost{
		ID: "post1", ContentID: "c1", IdempotencyKey: "key-1", PublishedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	got, _ := s.PublishedPost.FindByIdempotencyKey(ctx, "key-1")
	if got == nil || got.ID != "post1" {
		t.Fatalf("FindByIdempotencyKey = %+v", got)
	}
	byContent, _ := s.PublishedPost.FindByContentID(ctx, "c1")
	if byContent == nil {
		t.Fatalf("FindByContentID = nil")
	}
}

func TestConcurrentAccessIsSafe(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c := &entity.Content{ID: string(rune('a' + i)), Status: entity.StatusDraft, Hook: "h"}
			if err := s.Content.Save(ctx, c); err != nil {
				t.Errorf("Save: %v", err)
			}
			_, _ = s.Content.FindAll(ctx, repository.ContentFilter{Limit: 0})
		}(i)
	}
	wg.Wait()

	all, _ := s.Content.FindAll(ctx, repository.ContentFilter{Limit: 0})
	if len(all) != 20 {
		t.Fatalf("concurrent len = %d, want 20", len(all))
	}
}
