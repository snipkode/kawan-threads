package usecase

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"kawan-threads/internal/domain/entity"
	"kawan-threads/internal/domain/repository"
)

// ---------------------------------------------------------------------------
// In-memory repositories
// ---------------------------------------------------------------------------

type memStore struct {
	mu       sync.Mutex
	content  map[string]*entity.Content
	queue    map[string]*entity.QueueItem
	versions map[string][]*entity.ContentVersion
	history  map[string][]*entity.History
}

func newMemStore() *memStore {
	return &memStore{
		content:  make(map[string]*entity.Content),
		queue:    make(map[string]*entity.QueueItem),
		versions: make(map[string][]*entity.ContentVersion),
		history:  make(map[string][]*entity.History),
	}
}

type memContentRepo struct{ store *memStore }

func (m *memContentRepo) Save(_ context.Context, c *entity.Content) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()
	m.store.content[c.ID] = c
	return nil
}
func (m *memContentRepo) FindByID(_ context.Context, id string) (*entity.Content, error) {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()
	return m.store.content[id], nil
}
func (m *memContentRepo) FindAll(_ context.Context, _ repository.ContentFilter) ([]*entity.Content, error) {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()
	out := make([]*entity.Content, 0, len(m.store.content))
	for _, c := range m.store.content {
		out = append(out, c)
	}
	return out, nil
}
func (m *memContentRepo) UpdateStatus(_ context.Context, id string, status entity.ContentStatus) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()
	if c, ok := m.store.content[id]; ok {
		c.Status = status
	}
	return nil
}
func (m *memContentRepo) Update(_ context.Context, c *entity.Content) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()
	m.store.content[c.ID] = c
	return nil
}
func (m *memContentRepo) Delete(_ context.Context, id string) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()
	delete(m.store.content, id)
	return nil
}

var _ repository.ContentRepository = (*memContentRepo)(nil)

type memVersionRepo struct{ store *memStore }

func (m *memVersionRepo) Save(_ context.Context, v *entity.ContentVersion) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()
	m.store.versions[v.ContentID] = append(m.store.versions[v.ContentID], v)
	return nil
}
func (m *memVersionRepo) FindByContentID(_ context.Context, id string) ([]*entity.ContentVersion, error) {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()
	return m.store.versions[id], nil
}

var _ repository.ContentVersionRepository = (*memVersionRepo)(nil)

type memQueueRepo struct{ store *memStore }

func (m *memQueueRepo) Enqueue(_ context.Context, item *entity.QueueItem) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()
	m.store.queue[item.ContentID] = item
	return nil
}
func (m *memQueueRepo) Dequeue(_ context.Context, limit int) ([]*entity.QueueItem, error) {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()
	return nil, nil
}
func (m *memQueueRepo) FindByContentID(_ context.Context, id string) (*entity.QueueItem, error) {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()
	return m.store.queue[id], nil
}
func (m *memQueueRepo) Remove(_ context.Context, id string) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()
	delete(m.store.queue, id)
	return nil
}
func (m *memQueueRepo) UpdatePriority(_ context.Context, id string, p int) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()
	if q, ok := m.store.queue[id]; ok {
		q.Priority = p
	}
	return nil
}
func (m *memQueueRepo) FindAll(_ context.Context) ([]*entity.QueueItem, error) {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()
	out := make([]*entity.QueueItem, 0, len(m.store.queue))
	for _, q := range m.store.queue {
		out = append(out, q)
	}
	return out, nil
}

var _ repository.QueueRepository = (*memQueueRepo)(nil)

type memHistoryRepo struct{ store *memStore }

func (m *memHistoryRepo) Save(_ context.Context, h *entity.History) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()
	m.store.history[h.ContentID] = append(m.store.history[h.ContentID], h)
	return nil
}
func (m *memHistoryRepo) FindByContentID(_ context.Context, id string) ([]*entity.History, error) {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()
	return m.store.history[id], nil
}

var _ repository.HistoryRepository = (*memHistoryRepo)(nil)

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func newApprovalUC() (*ApprovalUseCase, *memStore) {
	store := newMemStore()
	return &ApprovalUseCase{
		ContentRepo: &memContentRepo{store},
		VersionRepo: &memVersionRepo{store},
		QueueRepo:   &memQueueRepo{store},
		HistoryRepo: &memHistoryRepo{store},
		Logger:      slog.Default(),
	}, store
}

func draftContent(id string) *entity.Content {
	return &entity.Content{
		ID:     id,
		Pillar: entity.PillarCommunity,
		Topic:  "Hak warga",
		Hook:   "Banyak orang baru belajar hukum setelah masalah datang.",
		Body:   "Padahal memahami hak itu penting sejak awal.",
		CTA:    "Kenalan dengan KAWAN.",
		Status: entity.StatusDraft,
	}
}

func TestApproveContent_DraftToQueued(t *testing.T) {
	uc, _ := newApprovalUC()
	if err := uc.ContentRepo.Save(context.Background(), draftContent("c1")); err != nil {
		t.Fatal(err)
	}

	item, err := uc.ApproveContent(context.Background(), "c1", "operator")
	if err != nil {
		t.Fatalf("approve failed: %v", err)
	}
	if item.ContentID != "c1" || item.Priority <= 0 {
		t.Fatalf("unexpected queue item: %+v", item)
	}

	c, _ := uc.ContentRepo.FindByID(context.Background(), "c1")
	if c.Status != entity.StatusQueued {
		t.Fatalf("expected QUEUED status, got %s", c.Status)
	}
	if c.ApprovedAt == nil {
		t.Fatal("expected ApprovedAt to be set")
	}

	hist, _ := uc.HistoryRepo.FindByContentID(context.Background(), "c1")
	if len(hist) != 1 || hist[0].NewStatus != entity.StatusQueued {
		t.Fatalf("unexpected history: %+v", hist)
	}
}

func TestApproveContent_NonDraftRejected(t *testing.T) {
	uc, _ := newApprovalUC()
	c := draftContent("c1")
	c.Status = entity.StatusScheduled
	if err := uc.ContentRepo.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	_, err := uc.ApproveContent(context.Background(), "c1", "operator")
	if !errors.Is(err, ErrNotDraft) {
		t.Fatalf("expected ErrNotDraft, got %v", err)
	}
}

func TestApproveContent_Idempotency(t *testing.T) {
	uc, _ := newApprovalUC()
	if err := uc.ContentRepo.Save(context.Background(), draftContent("c1")); err != nil {
		t.Fatal(err)
	}

	if _, err := uc.ApproveContent(context.Background(), "c1", "operator"); err != nil {
		t.Fatalf("first approve failed: %v", err)
	}

	// Approving again must not create a duplicate (idempotency).
	_, err := uc.ApproveContent(context.Background(), "c1", "operator")
	if err == nil {
		t.Fatal("expected second approve to fail")
	}

	items, _ := uc.QueueRepo.FindAll(context.Background())
	if len(items) != 1 {
		t.Fatalf("expected exactly 1 queue item, got %d", len(items))
	}
}

func TestApproveContent_RequiresHookAndBody(t *testing.T) {
	uc, _ := newApprovalUC()
	c := draftContent("c1")
	c.Hook = ""
	if err := uc.ContentRepo.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	_, err := uc.ApproveContent(context.Background(), "c1", "operator")
	if !errors.Is(err, ErrMissingRequiredFields) {
		t.Fatalf("expected ErrMissingRequiredFields, got %v", err)
	}
}

func TestRejectContent_DraftToRejected(t *testing.T) {
	uc, _ := newApprovalUC()
	if err := uc.ContentRepo.Save(context.Background(), draftContent("c1")); err != nil {
		t.Fatal(err)
	}

	if err := uc.RejectContent(context.Background(), "c1", "operator", "tone too formal"); err != nil {
		t.Fatalf("reject failed: %v", err)
	}

	c, _ := uc.ContentRepo.FindByID(context.Background(), "c1")
	if c.Status != entity.StatusRejected || c.RejectionReason != "tone too formal" {
		t.Fatalf("unexpected rejected content: %+v", c)
	}
}

func TestRemoveFromQueue_RevertsToDraft(t *testing.T) {
	uc, _ := newApprovalUC()
	if err := uc.ContentRepo.Save(context.Background(), draftContent("c1")); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.ApproveContent(context.Background(), "c1", "operator"); err != nil {
		t.Fatal(err)
	}

	if err := uc.RemoveFromQueue(context.Background(), "c1"); err != nil {
		t.Fatalf("remove failed: %v", err)
	}

	c, _ := uc.ContentRepo.FindByID(context.Background(), "c1")
	if c.Status != entity.StatusDraft || c.ApprovedAt != nil {
		t.Fatalf("expected revert to DRAFT without approval: %+v", c)
	}

	_, err := uc.QueueRepo.FindByContentID(context.Background(), "c1")
	if err != nil {
		t.Fatalf("queue item should be gone: %v", err)
	}
}

func TestUpdateQueuePriority(t *testing.T) {
	uc, _ := newApprovalUC()
	if err := uc.ContentRepo.Save(context.Background(), draftContent("c1")); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.ApproveContent(context.Background(), "c1", "operator"); err != nil {
		t.Fatal(err)
	}

	if err := uc.UpdateQueuePriority(context.Background(), "c1", 100); err != nil {
		t.Fatalf("update priority failed: %v", err)
	}

	item, _ := uc.QueueRepo.FindByContentID(context.Background(), "c1")
	if item.Priority != 100 {
		t.Fatalf("expected priority 100, got %d", item.Priority)
	}
}

func TestPillarPriority(t *testing.T) {
	if got := pillarPriority(entity.PillarEvent); got != 100 {
		t.Fatalf("event priority = %d, want 100", got)
	}
	if got := pillarPriority(entity.PillarMembership); got != 80 {
		t.Fatalf("membership priority = %d, want 80", got)
	}
	if got := pillarPriority(entity.PillarCommunity); got != 50 {
		t.Fatalf("community priority = %d, want 50", got)
	}
}

var _ = time.Now
