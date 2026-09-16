// Package publishing provides a worker that publishes scheduled content
// to the Threads platform.
package publishing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"kawan-threads/internal/config"
	"kawan-threads/internal/domain/entity"
	"kawan-threads/internal/domain/port"
	"kawan-threads/internal/domain/repository"
	"kawan-threads/internal/infrastructure/threads"
	applogger "kawan-threads/internal/logger"
)

// ---------------------------------------------------------------------------
// Publisher
// ---------------------------------------------------------------------------

// Publisher polls the schedule repository and publishes due content to the
// Threads platform.
type Publisher struct {
	ContentRepo   repository.ContentRepository
	ScheduleRepo  repository.ScheduleRepository
	PublishedRepo repository.PublishedPostRepository
	HistoryRepo   repository.HistoryRepository
	ThreadsPort   port.ThreadsPort
	Logger        *slog.Logger
	MaxRetry      int
	Timezone      *time.Location
	// Backoff holds the retry wait durations for rate-limit handling.
	// When nil, the default schedule (5s, 30s, 2m) is used.
	Backoff []time.Duration
}

// defaultBackoff is the exponential-recovery schedule on rate limits.
var defaultBackoff = []time.Duration{5 * time.Second, 30 * time.Second, 2 * time.Minute}

// NewPublisher constructs a Publisher with all dependencies.
func NewPublisher(
	cfg *config.Config,
	contentRepo repository.ContentRepository,
	scheduleRepo repository.ScheduleRepository,
	publishedRepo repository.PublishedPostRepository,
	historyRepo repository.HistoryRepository,
	threadsPort port.ThreadsPort,
	logger *slog.Logger,
) *Publisher {
	tz, err := time.LoadLocation(cfg.Scheduler.Timezone)
	if err != nil {
		logger.Warn("invalid timezone, falling back to UTC", "timezone", cfg.Scheduler.Timezone, "error", err)
		tz = time.UTC
	}

	return &Publisher{
		ContentRepo:   contentRepo,
		ScheduleRepo:  scheduleRepo,
		PublishedRepo: publishedRepo,
		HistoryRepo:   historyRepo,
		ThreadsPort:   threadsPort,
		Logger:        logger,
		MaxRetry:      cfg.Settings.MaxRetry,
		Timezone:      tz,
	}
}

// Run starts the publishing loop, ticking every 30 seconds. It blocks until
// ctx is cancelled.
func (p *Publisher) Run(ctx context.Context) {
	p.Logger.Info("publisher worker started", "interval", "30s", "max_retry", p.MaxRetry)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.Logger.Info("publisher worker stopping")
			return
		case <-ticker.C:
			p.runCycle(ctx)
		}
	}
}

// runCycle finds all schedules due for publishing and processes each one.
func (p *Publisher) runCycle(ctx context.Context) {
	now := time.Now().In(p.Timezone)

	due, err := p.ScheduleRepo.FindScheduled(ctx, now.UTC())
	if err != nil {
		p.Logger.Error("publisher: FindScheduled failed", "error", err)
		return
	}

	for _, schedule := range due {
		if err := p.publishOne(ctx, schedule); err != nil {
			p.Logger.Error("publisher: publishOne failed",
				"content_id", schedule.ContentID,
				"schedule_id", schedule.ID,
				"error", err,
			)
		}
	}
}

// publishOne processes a single scheduled item through the full publish
// lifecycle.
func (p *Publisher) publishOne(ctx context.Context, schedule *entity.Schedule) error {
	// 1. Load content.
	content, err := p.ContentRepo.FindByID(ctx, schedule.ContentID)
	if err != nil {
		return fmt.Errorf("load content %s: %w", schedule.ContentID, err)
	}
	if content == nil {
		return fmt.Errorf("content %s not found", schedule.ContentID)
	}

	// 2. Verify status is SCHEDULED.
	if content.Status != entity.StatusScheduled {
		p.Logger.Warn("publisher: content not in SCHEDULED state, skipping",
			"content_id", content.ID,
			"status", content.Status,
		)
		return nil
	}

	// 3. Idempotency check.
	idempotencyKey := content.ID + ":" + schedule.ID
	existing, err := p.PublishedRepo.FindByIdempotencyKey(ctx, idempotencyKey)
	if err != nil {
		return fmt.Errorf("idempotency check: %w", err)
	}
	if existing != nil {
		p.Logger.Info("publisher: already published, marking PUBLISHED",
			"content_id", content.ID,
			"idempotency_key", idempotencyKey,
		)
		return p.ContentRepo.UpdateStatus(ctx, content.ID, entity.StatusPublished)
	}

	// 4. Mark PUBLISHING.
	content.Status = entity.StatusPublishing
	content.UpdatedAt = time.Now().UTC()
	if err := p.ContentRepo.Update(ctx, content); err != nil {
		return fmt.Errorf("set PUBLISHING: %w", err)
	}

	// 5. Log publish_started.
	applogger.LogEvent(p.Logger, applogger.EventPublishStarted,
		"content_id", content.ID,
		"schedule_id", schedule.ID,
	)

	// 6. Compose post text.
	text := composeText(content)

	// 7 & 8. Create container and publish with retry on rate-limit.
	threadsPostID, publishErr := p.createAndPublish(ctx, text, content, schedule)
	if publishErr != nil {
		// Determine whether we have exhausted retries.
		return p.handlePublishFailure(ctx, content, schedule, publishErr)
	}

	// 9 & 10. Save PublishedPost.
	now := time.Now().UTC()
	publishedPost := &entity.PublishedPost{
		ID:             uuid.New().String(),
		ContentID:      content.ID,
		ThreadsPostID:  threadsPostID,
		PublishedAt:    now,
		IdempotencyKey: idempotencyKey,
		RetryCount:     0,
	}
	if err := p.PublishedRepo.Save(ctx, publishedPost); err != nil {
		return fmt.Errorf("save PublishedPost: %w", err)
	}

	// 11. Mark PUBLISHED.
	content.Status = entity.StatusPublished
	content.UpdatedAt = now
	if err := p.ContentRepo.Update(ctx, content); err != nil {
		return fmt.Errorf("set PUBLISHED: %w", err)
	}

	// 12. Delete schedule.
	if err := p.ScheduleRepo.Delete(ctx, schedule.ID); err != nil {
		// Non-fatal: log but don't fail the whole operation.
		p.Logger.Warn("publisher: failed to delete schedule after publish",
			"schedule_id", schedule.ID, "error", err)
	}

	// 13. Save history: SCHEDULED → PUBLISHED.
	_ = p.saveHistory(ctx, content.ID,
		entity.StatusScheduled, entity.StatusPublished,
		"system", "published successfully",
	)

	// 14. Log publish_success.
	applogger.LogEvent(p.Logger, applogger.EventPublishSuccess,
		"content_id", content.ID,
		"threads_post_id", threadsPostID,
		"schedule_id", schedule.ID,
	)

	return nil
}

// createAndPublish calls the Threads API with exponential backoff on
// RateLimitError. Returns the threads post ID on success.
func (p *Publisher) createAndPublish(
	ctx context.Context,
	text string,
	content *entity.Content,
	schedule *entity.Schedule,
) (string, error) {
	// Determine the user ID. The publisher uses the adapter's configured
	// user ID by calling CreateTextContainer with an empty userID so the
	// adapter fills it in; however the port signature requires passing userID.
	// We obtain it from the schedule's actor or fall back to a configured value.
	// Because port.ThreadsPort takes userID as a parameter, pass the adapter's
	// stored userID (the adapter itself always uses its own token anyway).
	userID := p.resolveUserID()

	backoffs := p.backoffSchedule()

	var (
		creationID    string
		threadsPostID string
		lastErr       error
	)

	for attempt := 0; attempt <= p.MaxRetry; attempt++ {
		if attempt > 0 {
			wait := p.backoffDuration(attempt-1, backoffs)
			p.Logger.Info("publisher: retrying after backoff",
				"content_id", content.ID,
				"attempt", attempt,
				"backoff", wait,
			)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(wait):
			}
		}

		creationID, lastErr = p.ThreadsPort.CreateTextContainer(ctx, userID, text)
		if lastErr != nil {
			if isRateLimitError(lastErr) {
				p.Logger.Warn("publisher: rate limited on CreateTextContainer",
					"content_id", content.ID, "attempt", attempt)
				continue
			}
			return "", lastErr
		}

		threadsPostID, lastErr = p.ThreadsPort.PublishContainer(ctx, userID, creationID)
		if lastErr != nil {
			if isRateLimitError(lastErr) {
				p.Logger.Warn("publisher: rate limited on PublishContainer",
					"content_id", content.ID, "attempt", attempt)
				continue
			}
			return "", lastErr
		}

		// Success.
		return threadsPostID, nil
	}

	return "", fmt.Errorf("max retries (%d) exceeded: %w", p.MaxRetry, lastErr)
}

// handlePublishFailure updates content status to FAILED, persists history,
// logs the failure, and re-enqueues (adds back to queue is done via
// returning a typed error; the caller can check it if needed).
func (p *Publisher) handlePublishFailure(
	ctx context.Context,
	content *entity.Content,
	schedule *entity.Schedule,
	publishErr error,
) error {
	content.Status = entity.StatusFailed
	content.UpdatedAt = time.Now().UTC()
	if updateErr := p.ContentRepo.Update(ctx, content); updateErr != nil {
		p.Logger.Error("publisher: failed to mark FAILED", "content_id", content.ID, "error", updateErr)
	}

	_ = p.saveHistory(ctx, content.ID,
		entity.StatusPublishing, entity.StatusFailed,
		"system", publishErr.Error(),
	)

	applogger.LogEvent(p.Logger, applogger.EventPublishFailed,
		"content_id", content.ID,
		"schedule_id", schedule.ID,
		"error", publishErr.Error(),
	)

	return publishErr
}

// resolveUserID returns the configured user ID from the ThreadsPort if it
// is a *threads.ThreadsAdapter, otherwise returns an empty string. The
// adapter always uses its own stored userID when making API calls, but the
// port interface requires the caller to pass it explicitly.
func (p *Publisher) resolveUserID() string {
	if a, ok := p.ThreadsPort.(*threads.ThreadsAdapter); ok {
		return a.UserID()
	}
	return ""
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// composeText assembles the final post text from a Content entity.
func composeText(c *entity.Content) string {
	parts := make([]string, 0, 4)
	if c.Hook != "" {
		parts = append(parts, c.Hook)
	}
	if c.Body != "" {
		parts = append(parts, c.Body)
	}
	if c.CTA != "" {
		parts = append(parts, c.CTA)
	}
	if c.ConversationQuestion != "" {
		parts = append(parts, c.ConversationQuestion)
	}
	return strings.Join(parts, "\n\n")
}

// saveHistory persists a History entry and silently logs any error.
func (p *Publisher) saveHistory(
	ctx context.Context,
	contentID string,
	from, to entity.ContentStatus,
	actor, note string,
) error {
	h := &entity.History{
		ID:        uuid.New().String(),
		ContentID: contentID,
		Action:    fmt.Sprintf("%s→%s", from, to),
		OldStatus: from,
		NewStatus: to,
		Actor:     actor,
		Note:      note,
		CreatedAt: time.Now().UTC(),
	}
	if err := p.HistoryRepo.Save(ctx, h); err != nil {
		p.Logger.Error("publisher: failed to save history",
			"content_id", contentID, "error", err)
		return err
	}
	return nil
}

// backoffDuration returns the backoff duration for a given attempt index,
// capped at the last element in the supplied schedule.
func (p *Publisher) backoffDuration(attempt int, schedule []time.Duration) time.Duration {
	if attempt < len(schedule) {
		return schedule[attempt]
	}
	return schedule[len(schedule)-1]
}

// backoffSchedule returns the configured backoff schedule or the default one.
func (p *Publisher) backoffSchedule() []time.Duration {
	if p.Backoff != nil && len(p.Backoff) > 0 {
		return p.Backoff
	}
	if p.MaxRetry == 0 {
		return []time.Duration{defaultBackoff[0]}
	}
	return defaultBackoff
}

// isRateLimitError reports whether err is (or wraps) a RateLimitError.
func isRateLimitError(err error) bool {
	var rle *threads.RateLimitError
	return errors.As(err, &rle)
}
