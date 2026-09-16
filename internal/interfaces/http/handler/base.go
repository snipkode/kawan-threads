// Package handler provides shared HTTP handler types and helpers.
package handler

import (
	"log/slog"

	"kawan-threads/internal/domain/repository"
)

// BaseHandler bundles the shared dependencies injected into every handler.
type BaseHandler struct {
	Logger *slog.Logger

	// Repositories
	ContentRepo     repository.ContentRepository
	QueueRepo       repository.QueueRepository
	ScheduleRepo    repository.ScheduleRepository
	PerformanceRepo repository.PostPerformanceRepository
	TopicRepo       repository.TopicRepository
	HistoryRepo     repository.HistoryRepository
}

// NewBaseHandler constructs a BaseHandler with all dependencies.
func NewBaseHandler(
	logger *slog.Logger,
	contentRepo repository.ContentRepository,
	queueRepo repository.QueueRepository,
	scheduleRepo repository.ScheduleRepository,
	performanceRepo repository.PostPerformanceRepository,
	topicRepo repository.TopicRepository,
	historyRepo repository.HistoryRepository,
) *BaseHandler {
	return &BaseHandler{
		Logger:          logger,
		ContentRepo:     contentRepo,
		QueueRepo:       queueRepo,
		ScheduleRepo:    scheduleRepo,
		PerformanceRepo: performanceRepo,
		TopicRepo:       topicRepo,
		HistoryRepo:     historyRepo,
	}
}
