// Package usecase contains application-layer business logic.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"kawan-threads/internal/domain/entity"
	"kawan-threads/internal/domain/repository"
)

// Sentinel errors returned by ApprovalUseCase methods.
var (
	ErrContentNotFound       = errors.New("content not found")
	ErrNotDraft              = errors.New("content status must be DRAFT")
	ErrAlreadyQueued         = errors.New("content is already in the queue")
	ErrMissingRequiredFields = errors.New("content is missing required fields (hook, body)")
)

// ApprovalUseCase encapsulates all approval-flow business logic.
type ApprovalUseCase struct {
	ContentRepo  repository.ContentRepository
	VersionRepo  repository.ContentVersionRepository
	QueueRepo    repository.QueueRepository
	HistoryRepo  repository.HistoryRepository
	Logger       *slog.Logger
	AutoApproval bool
}

// ContentPreview is the combined view returned by GetApprovalPreview.
type ContentPreview struct {
	Content  *entity.Content          `json:"content"`
	Versions []*entity.ContentVersion `json:"versions"`
	History  []*entity.History        `json:"history"`
}

// ---------------------------------------------------------------------------
// ApproveContent
// ---------------------------------------------------------------------------

// ApproveContent transitions a DRAFT content item to QUEUED and enqueues it.
// It is idempotent in the sense that if the content is already in the queue
// an ErrAlreadyQueued error is returned rather than creating a duplicate entry.
func (uc *ApprovalUseCase) ApproveContent(ctx context.Context, contentID, approvedBy string) (*entity.QueueItem, error) {
	// 1. Load content by ID.
	content, err := uc.ContentRepo.FindByID(ctx, contentID)
	if err != nil {
		return nil, fmt.Errorf("loading content: %w", err)
	}
	if content == nil {
		return nil, ErrContentNotFound
	}

	// 2. Verify status == DRAFT.
	if content.Status != entity.StatusDraft {
		return nil, fmt.Errorf("%w: current status is %s", ErrNotDraft, content.Status)
	}

	// 3. Validate required fields.
	if content.Hook == "" || content.Body == "" {
		return nil, ErrMissingRequiredFields
	}

	// 4. Idempotency check — abort if already queued.
	existing, err := uc.QueueRepo.FindByContentID(ctx, contentID)
	if err != nil {
		return nil, fmt.Errorf("checking queue: %w", err)
	}
	if existing != nil {
		return nil, ErrAlreadyQueued
	}

	// 5. Update content fields.
	now := time.Now().UTC()
	content.Status = entity.StatusQueued
	content.ApprovedAt = &now
	content.ApprovedBy = approvedBy
	content.UpdatedAt = now

	// 6. Persist the updated content.
	if err := uc.ContentRepo.Update(ctx, content); err != nil {
		return nil, fmt.Errorf("updating content: %w", err)
	}

	// 7. Determine queue priority from pillar.
	priority := pillarPriority(content.Pillar)

	// Create the QueueItem.
	queueItem := &entity.QueueItem{
		ID:            uuid.New().String(),
		ContentID:     contentID,
		Status:        entity.StatusQueued,
		Priority:      priority,
		ApprovedAt:    now,
		QueuedAt:      now,
		SchedulerType: entity.SchedulerAMAB,
	}

	// 8. Enqueue the item.
	if err := uc.QueueRepo.Enqueue(ctx, queueItem); err != nil {
		return nil, fmt.Errorf("enqueueing content: %w", err)
	}

	// 9. Save audit history entry.
	history := &entity.History{
		ID:        uuid.New().String(),
		ContentID: contentID,
		Action:    "APPROVED",
		OldStatus: entity.StatusDraft,
		NewStatus: entity.StatusQueued,
		Actor:     approvedBy,
		Note:      "Content approved and added to publish queue",
		CreatedAt: now,
	}
	if err := uc.HistoryRepo.Save(ctx, history); err != nil {
		// Non-fatal: log and continue.
		uc.Logger.Error("failed to save approval history", "content_id", contentID, "error", err)
	}

	// 10. Structured log events.
	uc.Logger.Info("content_approved",
		"content_id", contentID,
		"approved_by", approvedBy,
		"pillar", string(content.Pillar),
	)
	uc.Logger.Info("content_queued",
		"content_id", contentID,
		"queue_item_id", queueItem.ID,
		"priority", priority,
	)

	// 11. Return the new queue item.
	return queueItem, nil
}

// ---------------------------------------------------------------------------
// RejectContent
// ---------------------------------------------------------------------------

// RejectContent transitions a DRAFT content item to REJECTED.
func (uc *ApprovalUseCase) RejectContent(ctx context.Context, contentID, rejectedBy, reason string) error {
	// 1. Load content.
	content, err := uc.ContentRepo.FindByID(ctx, contentID)
	if err != nil {
		return fmt.Errorf("loading content: %w", err)
	}
	if content == nil {
		return ErrContentNotFound
	}

	// 2. Verify status == DRAFT.
	if content.Status != entity.StatusDraft {
		return fmt.Errorf("%w: current status is %s", ErrNotDraft, content.Status)
	}

	// 3. Update rejection fields.
	now := time.Now().UTC()
	content.Status = entity.StatusRejected
	content.RejectedAt = &now
	content.RejectedBy = rejectedBy
	content.RejectionReason = reason
	content.UpdatedAt = now

	// 4. Persist.
	if err := uc.ContentRepo.Update(ctx, content); err != nil {
		return fmt.Errorf("updating content: %w", err)
	}

	// 5. Save audit history.
	history := &entity.History{
		ID:        uuid.New().String(),
		ContentID: contentID,
		Action:    "REJECTED",
		OldStatus: entity.StatusDraft,
		NewStatus: entity.StatusRejected,
		Actor:     rejectedBy,
		Note:      reason,
		CreatedAt: now,
	}
	if err := uc.HistoryRepo.Save(ctx, history); err != nil {
		uc.Logger.Error("failed to save rejection history", "content_id", contentID, "error", err)
	}

	// 6. Log.
	uc.Logger.Info("content_rejected",
		"content_id", contentID,
		"rejected_by", rejectedBy,
		"reason", reason,
	)

	return nil
}

// ---------------------------------------------------------------------------
// GetApprovalPreview
// ---------------------------------------------------------------------------

// GetApprovalPreview returns the content item together with its version
// history and audit trail for pre-approval review.
func (uc *ApprovalUseCase) GetApprovalPreview(ctx context.Context, contentID string) (*ContentPreview, error) {
	// 1. Get content.
	content, err := uc.ContentRepo.FindByID(ctx, contentID)
	if err != nil {
		return nil, fmt.Errorf("loading content: %w", err)
	}
	if content == nil {
		return nil, ErrContentNotFound
	}

	// 2. Get content versions.
	versions, err := uc.VersionRepo.FindByContentID(ctx, contentID)
	if err != nil {
		return nil, fmt.Errorf("loading versions: %w", err)
	}

	// 3. Get audit history.
	history, err := uc.HistoryRepo.FindByContentID(ctx, contentID)
	if err != nil {
		return nil, fmt.Errorf("loading history: %w", err)
	}

	return &ContentPreview{
		Content:  content,
		Versions: versions,
		History:  history,
	}, nil
}

// ---------------------------------------------------------------------------
// GetQueue
// ---------------------------------------------------------------------------

// GetQueue returns all items currently in the publish queue.
func (uc *ApprovalUseCase) GetQueue(ctx context.Context) ([]*entity.QueueItem, error) {
	items, err := uc.QueueRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching queue: %w", err)
	}
	return items, nil
}

// ---------------------------------------------------------------------------
// RemoveFromQueue
// ---------------------------------------------------------------------------

// RemoveFromQueue removes a content item from the queue and reverts its
// status back to DRAFT so it can be re-reviewed.
func (uc *ApprovalUseCase) RemoveFromQueue(ctx context.Context, contentID string) error {
	// Verify the item is actually in the queue.
	queueItem, err := uc.QueueRepo.FindByContentID(ctx, contentID)
	if err != nil {
		return fmt.Errorf("checking queue: %w", err)
	}
	if queueItem == nil {
		return fmt.Errorf("content %s is not in the queue", contentID)
	}

	// Remove from queue.
	if err := uc.QueueRepo.Remove(ctx, contentID); err != nil {
		return fmt.Errorf("removing from queue: %w", err)
	}

	// Revert content status to DRAFT.
	content, err := uc.ContentRepo.FindByID(ctx, contentID)
	if err != nil {
		return fmt.Errorf("loading content: %w", err)
	}
	if content != nil {
		now := time.Now().UTC()
		prevStatus := content.Status
		content.Status = entity.StatusDraft
		content.ApprovedAt = nil
		content.ApprovedBy = ""
		content.UpdatedAt = now

		if err := uc.ContentRepo.Update(ctx, content); err != nil {
			return fmt.Errorf("reverting content status: %w", err)
		}

		// Save audit history.
		history := &entity.History{
			ID:        uuid.New().String(),
			ContentID: contentID,
			Action:    "REMOVED_FROM_QUEUE",
			OldStatus: prevStatus,
			NewStatus: entity.StatusDraft,
			Actor:     "system",
			Note:      "Removed from publish queue; reverted to DRAFT",
			CreatedAt: now,
		}
		if err := uc.HistoryRepo.Save(ctx, history); err != nil {
			uc.Logger.Error("failed to save queue-removal history", "content_id", contentID, "error", err)
		}
	}

	uc.Logger.Info("content_removed_from_queue", "content_id", contentID)
	return nil
}

// ---------------------------------------------------------------------------
// UpdateQueuePriority
// ---------------------------------------------------------------------------

// UpdateQueuePriority changes the scheduling priority of a queued content item.
func (uc *ApprovalUseCase) UpdateQueuePriority(ctx context.Context, contentID string, priority int) error {
	if err := uc.QueueRepo.UpdatePriority(ctx, contentID, priority); err != nil {
		return fmt.Errorf("updating queue priority: %w", err)
	}
	uc.Logger.Info("queue_priority_updated", "content_id", contentID, "priority", priority)
	return nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// pillarPriority returns a queue priority score for the given content pillar.
// Higher values are processed first.
func pillarPriority(pillar entity.ContentPillar) int {
	switch pillar {
	case entity.PillarEvent:
		return 100
	case entity.PillarMembership:
		return 80
	default:
		return 50
	}
}
