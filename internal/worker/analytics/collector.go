// Package analytics provides a worker that collects post performance data
// from the Threads API and persists it for AMAB feedback loop.
package analytics

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"kawan-threads/internal/domain/entity"
	"kawan-threads/internal/domain/port"
	"kawan-threads/internal/domain/repository"
	applogger "kawan-threads/internal/logger"
)

// Collector polls published posts and fetches their performance metrics.
type Collector struct {
	PublishedRepo   repository.PublishedPostRepository
	ContentRepo     repository.ContentRepository
	PerformanceRepo repository.PostPerformanceRepository
	ThreadsPort     port.ThreadsPort
	Logger          *slog.Logger
	UserID          string
}

// NewCollector creates a Collector.
func NewCollector(
	publishedRepo repository.PublishedPostRepository,
	contentRepo repository.ContentRepository,
	perfRepo repository.PostPerformanceRepository,
	threadsPort port.ThreadsPort,
	logger *slog.Logger,
	userID string,
) *Collector {
	return &Collector{
		PublishedRepo:   publishedRepo,
		ContentRepo:     contentRepo,
		PerformanceRepo: perfRepo,
		ThreadsPort:     threadsPort,
		Logger:          logger,
		UserID:          userID,
	}
}

// Run starts the analytics collection loop, ticking every hour.
func (c *Collector) Run(ctx context.Context) {
	c.Logger.Info("analytics collector started", "interval", "1h")
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.Logger.Info("analytics collector stopping")
			return
		case <-ticker.C:
			c.collect(ctx)
		}
	}
}

func (c *Collector) collect(ctx context.Context) {
	if c.ThreadsPort == nil {
		return
	}

	// Get recent published posts that need analytics.
	// In a full implementation we'd track which ones haven't been fetched yet.
	// For now we demonstrate the pattern.
	c.Logger.Debug("analytics: collection cycle started")
}

// CollectForPost fetches and stores performance data for a single published post.
func (c *Collector) CollectForPost(ctx context.Context, published *entity.PublishedPost) error {
	if c.ThreadsPort == nil {
		return nil
	}

	insights, err := c.ThreadsPort.GetPostInsights(ctx, c.UserID, published.ThreadsPostID)
	if err != nil {
		c.Logger.Warn("analytics: failed to fetch insights",
			"post_id", published.ThreadsPostID,
			"error", err,
		)
		return err
	}

	content, err := c.ContentRepo.FindByID(ctx, published.ContentID)
	if err != nil || content == nil {
		return err
	}

	perf := &entity.PostPerformance{
		PostID:      uuid.New().String(),
		ContentID:   published.ContentID,
		Pillar:      content.Pillar,
		Topic:       content.Topic,
		HookType:    content.HookType,
		Format:      content.Format,
		PublishedAt: published.PublishedAt,
		Views:       insights.Views,
		Likes:       insights.Likes,
		Replies:     insights.Replies,
		Reposts:     insights.Reposts,
		Quotes:      insights.Quotes,
		CollectedAt: time.Now().UTC(),
	}

	if err := c.PerformanceRepo.Save(ctx, perf); err != nil {
		return err
	}

	applogger.LogEvent(c.Logger, applogger.EventAnalyticsCollected,
		"content_id", published.ContentID,
		"views", insights.Views,
		"replies", insights.Replies,
		"reposts", insights.Reposts,
	)

	return nil
}
