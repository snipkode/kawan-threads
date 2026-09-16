package repository

import (
	"context"
	"time"

	"kawan-threads/internal/domain/entity"
)

// ContentFilter provides filtering options when listing Content items.
type ContentFilter struct {
	Status *entity.ContentStatus
	Pillar *entity.ContentPillar
	Limit  int
	Offset int
}

// ContentRepository defines persistence operations for Content entities.
type ContentRepository interface {
	Save(ctx context.Context, content *entity.Content) error
	FindByID(ctx context.Context, id string) (*entity.Content, error)
	FindAll(ctx context.Context, filter ContentFilter) ([]*entity.Content, error)
	UpdateStatus(ctx context.Context, id string, status entity.ContentStatus) error
	Update(ctx context.Context, content *entity.Content) error
	Delete(ctx context.Context, id string) error
}

// ContentVersionRepository defines persistence operations for ContentVersion entities.
type ContentVersionRepository interface {
	Save(ctx context.Context, version *entity.ContentVersion) error
	FindByContentID(ctx context.Context, contentID string) ([]*entity.ContentVersion, error)
}

// QueueRepository defines persistence operations for QueueItem entities.
type QueueRepository interface {
	Enqueue(ctx context.Context, item *entity.QueueItem) error
	Dequeue(ctx context.Context, limit int) ([]*entity.QueueItem, error)
	FindByContentID(ctx context.Context, contentID string) (*entity.QueueItem, error)
	Remove(ctx context.Context, contentID string) error
	UpdatePriority(ctx context.Context, contentID string, priority int) error
	FindAll(ctx context.Context) ([]*entity.QueueItem, error)
}

// ScheduleRepository defines persistence operations for Schedule entities.
type ScheduleRepository interface {
	Save(ctx context.Context, schedule *entity.Schedule) error
	FindByID(ctx context.Context, id string) (*entity.Schedule, error)
	FindScheduled(ctx context.Context, before time.Time) ([]*entity.Schedule, error)
	FindAll(ctx context.Context) ([]*entity.Schedule, error)
	Delete(ctx context.Context, id string) error
}

// PublishedPostRepository defines persistence operations for PublishedPost entities.
type PublishedPostRepository interface {
	Save(ctx context.Context, post *entity.PublishedPost) error
	FindAll(ctx context.Context) ([]*entity.PublishedPost, error)
	FindByContentID(ctx context.Context, contentID string) (*entity.PublishedPost, error)
	FindByIdempotencyKey(ctx context.Context, key string) (*entity.PublishedPost, error)
}

// PostPerformanceRepository defines persistence operations for PostPerformance entities.
type PostPerformanceRepository interface {
	Save(ctx context.Context, perf *entity.PostPerformance) error
	FindByPostID(ctx context.Context, postID string) (*entity.PostPerformance, error)
	FindAll(ctx context.Context) ([]*entity.PostPerformance, error)
	FindByPillar(ctx context.Context, pillar entity.ContentPillar) ([]*entity.PostPerformance, error)
}

// TopicRepository defines persistence operations for Topic entities.
type TopicRepository interface {
	Save(ctx context.Context, topic *entity.Topic) error
	FindAll(ctx context.Context) ([]*entity.Topic, error)
	FindByID(ctx context.Context, id string) (*entity.Topic, error)
}

// HistoryRepository defines persistence operations for History entries.
type HistoryRepository interface {
	Save(ctx context.Context, h *entity.History) error
	FindByContentID(ctx context.Context, contentID string) ([]*entity.History, error)
}

// ExperimentRepository defines persistence operations for Experiment entities.
type ExperimentRepository interface {
	Save(ctx context.Context, exp *entity.Experiment) error
	FindByID(ctx context.Context, id string) (*entity.Experiment, error)
	FindAll(ctx context.Context) ([]*entity.Experiment, error)
}
