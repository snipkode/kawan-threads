// Package analytics provides a worker that collects published-post insights
// from the Threads API and feeds them back into the AMAB strategy.
package analytics

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"kawan-threads/internal/domain/entity"
	"kawan-threads/internal/domain/port"
	"kawan-threads/internal/domain/repository"
	"kawan-threads/internal/infrastructure/scheduler"
	applogger "kawan-threads/internal/logger"
)

// WorkerOptions configures the analytics worker.
type WorkerOptions struct {
	PublishedRepo    repository.PublishedPostRepository
	ContentRepo      repository.ContentRepository
	PerformanceRepo  repository.PostPerformanceRepository
	HistoryRepo      repository.HistoryRepository
	ThreadsPort      port.ThreadsPort
	UserID           string
	Logger           *slog.Logger
	TickInterval     time.Duration
	ProcessBatchSize int
}

// Worker polls for published posts that lack performance data, fetches
// insights from Threads, stores them, and refreshes the AMAB strategy.
type Worker struct {
	opts WorkerOptions
	amab *scheduler.AMABScorer
}

// NewWorker constructs an analytics Worker.
func NewWorker(opts WorkerOptions) (*Worker, error) {
	if opts.ThreadsPort == nil {
		return nil, fmt.Errorf("analytics worker: ThreadsPort is required")
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.TickInterval <= 0 {
		opts.TickInterval = 10 * time.Minute
	}
	if opts.ProcessBatchSize <= 0 {
		opts.ProcessBatchSize = 10
	}
	return &Worker{
		opts: opts,
		amab: scheduler.NewAMABScorer(opts.PerformanceRepo, opts.Logger, 0.20),
	}, nil
}

// Run blocks until ctx is cancelled, collecting analytics on each tick.
func (w *Worker) Run(ctx context.Context) {
	w.opts.Logger.Info("analytics worker started",
		"interval", w.opts.TickInterval.String(),
		"batch_size", w.opts.ProcessBatchSize,
	)
	ticker := time.NewTicker(w.opts.TickInterval)
	defer ticker.Stop()

	// Run one pass immediately so insights are collected shortly after startup.
	w.runCycle(ctx)

	for {
		select {
		case <-ctx.Done():
			w.opts.Logger.Info("analytics worker stopping")
			return
		case <-ticker.C:
			w.runCycle(ctx)
		}
	}
}

// threadsConfigurer is implemented by a ThreadsPort that can report whether
// it holds the minimum credentials to call the Threads API.
type threadsConfigurer interface {
	Configured() bool
}

// runCycle collects insights for published posts that have no performance
// record yet, then refreshes the AMAB strategy recommendation.
func (w *Worker) runCycle(ctx context.Context) {
	if c, ok := w.opts.ThreadsPort.(threadsConfigurer); ok && !c.Configured() {
		w.opts.Logger.Debug("analytics: Threads not configured, skipping cycle")
		return
	}

	posts, err := w.opts.PublishedRepo.FindAll(ctx)
	if err != nil {
		w.opts.Logger.Error("analytics: load published posts failed", "error", err)
		return
	}

	collected := 0
	processed := 0
	for _, post := range posts {
		if w.opts.ProcessBatchSize > 0 && processed >= w.opts.ProcessBatchSize {
			break
		}
		processed++

		// Skip posts already measured.
		existing, err := w.opts.PerformanceRepo.FindByPostID(ctx, post.ID)
		if err != nil {
			w.opts.Logger.Error("analytics: performance lookup failed", "post_id", post.ID, "error", err)
			continue
		}
		if existing != nil {
			continue
		}

		insights, err := w.opts.ThreadsPort.GetPostInsights(ctx, w.opts.UserID, post.ThreadsPostID)
		if err != nil {
			w.opts.Logger.Warn("analytics: insights fetch failed",
				"post_id", post.ID, "threads_post_id", post.ThreadsPostID, "error", err)
			continue
		}

		content, cErr := w.opts.ContentRepo.FindByID(ctx, post.ContentID)
		if cErr != nil || content == nil {
			w.opts.Logger.Warn("analytics: content lookup failed for post", "content_id", post.ContentID, "error", cErr)
			content = &entity.Content{}
		}

		perf := &entity.PostPerformance{
			PostID:      post.ID,
			ContentID:   post.ContentID,
			Pillar:      content.Pillar,
			Topic:       content.Topic,
			HookType:    content.HookType,
			Format:      content.Format,
			ScheduledAt: post.PublishedAt.Add(-24 * time.Hour), // best-effort; schedule is deleted after publish
			PublishedAt: post.PublishedAt,
			Views:       insights.Views,
			Likes:       insights.Likes,
			Replies:     insights.Replies,
			Reposts:     insights.Reposts,
			Quotes:      insights.Quotes,
			CollectedAt: time.Now().UTC(),
		}

		if err := w.opts.PerformanceRepo.Save(ctx, perf); err != nil {
			w.opts.Logger.Error("analytics: save performance failed", "post_id", post.ID, "error", err)
			continue
		}

		applogger.LogEvent(w.opts.Logger, applogger.EventAnalyticsCollected,
			"post_id", post.ID,
			"content_id", post.ContentID,
			"views", insights.Views,
			"replies", insights.Replies,
			"reposts", insights.Reposts,
			"quotes", insights.Quotes,
		)
		collected++
	}

	if collected == 0 {
		return
	}

	// Refresh the AMAB strategy now that analytics have been collected.
	rec, recErr := w.amab.GetStrategyRecommendation(ctx)
	if recErr != nil {
		w.opts.Logger.Warn("analytics: strategy recommendation failed", "error", recErr)
		return
	}
	applogger.LogEvent(w.opts.Logger, applogger.EventAmabUpdated,
		"top_pillar", string(rec.TopPillar),
		"best_hours", rec.BestHours,
		"analyzed_posts", rec.AnalyzedPosts,
	)
	w.opts.Logger.Info("analytics: amab strategy updated", "collected", collected)
}
