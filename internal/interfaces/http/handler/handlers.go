package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"kawan-threads/internal/application/usecase"
	"kawan-threads/internal/domain/entity"
	"kawan-threads/internal/domain/port"
	"kawan-threads/internal/domain/repository"
)

// ---------------------------------------------------------------------------
// Health
// ---------------------------------------------------------------------------

// HealthHandler handles GET /api/health.
type HealthHandler struct {
	StartTime time.Time
}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{StartTime: time.Now()}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	WriteSuccess(w, map[string]interface{}{
		"status": "ok",
		"uptime": time.Since(h.StartTime).String(),
		"time":   time.Now().UTC(),
	})
}

// ---------------------------------------------------------------------------
// Dashboard
// ---------------------------------------------------------------------------

// DashboardHandler handles GET /api/dashboard.
type DashboardHandler struct {
	*BaseHandler
}

func NewDashboardHandler(base *BaseHandler) *DashboardHandler {
	return &DashboardHandler{base}
}

func (h *DashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Draft count
	draftStatus := entity.StatusDraft
	publishedStatus := entity.StatusPublished
	drafts, err := h.ContentRepo.FindAll(ctx, repository.ContentFilter{Status: &draftStatus, Limit: 0})
	if err != nil {
		WriteInternalError(w, "failed to fetch drafts")
		return
	}

	published, err := h.ContentRepo.FindAll(ctx, repository.ContentFilter{Status: &publishedStatus, Limit: 0})
	if err != nil {
		WriteInternalError(w, "failed to fetch published content")
		return
	}

	// Queue count
	queueItems, err := h.QueueRepo.FindAll(ctx)
	if err != nil {
		WriteInternalError(w, "failed to fetch queue")
		return
	}

	// Scheduled count
	scheduledItems, err := h.ScheduleRepo.FindAll(ctx)
	if err != nil {
		WriteInternalError(w, "failed to fetch schedules")
		return
	}

	// Aggregate performance totals.
	perfs, err := h.PerformanceRepo.FindAll(ctx)
	if err != nil {
		WriteInternalError(w, "failed to fetch analytics")
		return
	}
	var totalViews, totalReplies, totalReposts, totalLikes int64
	for _, p := range perfs {
		totalViews += p.Views
		totalLikes += p.Likes
		totalReplies += p.Replies
		totalReposts += p.Reposts
	}

	WriteSuccess(w, map[string]interface{}{
		"draft_count":     len(drafts),
		"published_count": len(published),
		"queue_count":     len(queueItems),
		"scheduled_count": len(scheduledItems),
		"total_views":     totalViews,
		"total_likes":     totalLikes,
		"total_replies":   totalReplies,
		"total_reposts":   totalReposts,
		"generated_at":    time.Now().UTC(),
	})
}

// ---------------------------------------------------------------------------
// Content
// ---------------------------------------------------------------------------

// ContentHandler handles all /api/content routes.
type ContentHandler struct {
	*BaseHandler
	ContentUseCase  *usecase.ContentUseCase
	ApprovalUseCase *usecase.ApprovalUseCase
}

func NewContentHandler(base *BaseHandler, c *usecase.ContentUseCase, a *usecase.ApprovalUseCase) *ContentHandler {
	return &ContentHandler{BaseHandler: base, ContentUseCase: c, ApprovalUseCase: a}
}

func (h *ContentHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	filter := repository.ContentFilter{}

	if s := r.URL.Query().Get("status"); s != "" {
		st := entity.ContentStatus(s)
		filter.Status = &st
	}
	if p := r.URL.Query().Get("pillar"); p != "" {
		pl := entity.ContentPillar(p)
		filter.Pillar = &pl
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			filter.Limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil {
			filter.Offset = v
		}
	}

	items, err := h.ContentRepo.FindAll(ctx, filter)
	if err != nil {
		WriteInternalError(w, "failed to fetch content")
		return
	}
	WriteSuccess(w, items)
}

func (h *ContentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")
	if id == "" {
		WriteBadRequest(w, "id is required")
		return
	}

	c, err := h.ContentRepo.FindByID(ctx, id)
	if err != nil {
		WriteInternalError(w, "failed to fetch content")
		return
	}
	if c == nil {
		WriteNotFound(w, "content not found")
		return
	}
	WriteSuccess(w, c)
}

// GetPreview returns the content together with its versions and audit history.
func (h *ContentHandler) GetPreview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")
	if id == "" {
		WriteBadRequest(w, "id is required")
		return
	}

	preview, err := h.ApprovalUseCase.GetApprovalPreview(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrContentNotFound):
			WriteNotFound(w, "content not found")
		default:
			WriteInternalError(w, "failed to load preview")
		}
		return
	}
	WriteSuccess(w, preview)
}

func (h *ContentHandler) Generate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req port.GenerateContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "invalid request body")
		return
	}

	if h.ContentUseCase == nil {
		WriteError(w, http.StatusServiceUnavailable, "AI provider not configured", "AI_UNAVAILABLE")
		return
	}

	content, err := h.ContentUseCase.GenerateContent(ctx, req)
	if err != nil {
		h.Logger.Error("content generation failed", "error", err)
		WriteInternalError(w, "content generation failed")
		return
	}
	WriteCreated(w, content)
}

func (h *ContentHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	var req usecase.EditContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "invalid request body")
		return
	}

	if h.ContentUseCase == nil {
		WriteError(w, http.StatusServiceUnavailable, "content service not configured", "SERVICE_UNAVAILABLE")
		return
	}

	// Support selecting an AI-generated hook variant.
	if req.SelectedHookVariant != "" {
		existing, err := h.ContentRepo.FindByID(ctx, id)
		if err != nil || existing == nil {
			WriteNotFound(w, "content not found")
			return
		}
		for _, v := range existing.HookVariants {
			if v.Hook == req.SelectedHookVariant {
				req.Hook = v.Hook
				req.HookType = string(v.HookType)
				break
			}
		}
	}

	content, err := h.ContentUseCase.UpdateContent(ctx, id, req)
	if err != nil {
		WriteInternalError(w, "failed to update content")
		return
	}
	WriteSuccess(w, content)
}

func (h *ContentHandler) Regenerate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	existing, err := h.ContentRepo.FindByID(ctx, id)
	if err != nil || existing == nil {
		WriteNotFound(w, "content not found")
		return
	}

	if h.ContentUseCase == nil {
		WriteError(w, http.StatusServiceUnavailable, "AI provider not configured", "AI_UNAVAILABLE")
		return
	}

	req := port.GenerateContentRequest{
		Pillar:   existing.Pillar,
		Topic:    existing.Topic,
		Audience: existing.Audience,
		Tone:     existing.Tone,
		Format:   existing.Format,
		CTA:      existing.CTA,
	}

	content, err := h.ContentUseCase.RegenerateContent(ctx, id, req)
	if err != nil {
		WriteInternalError(w, "regeneration failed")
		return
	}
	WriteSuccess(w, content)
}

func (h *ContentHandler) Approve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	if h.ApprovalUseCase == nil {
		WriteError(w, http.StatusServiceUnavailable, "approval service not configured", "SERVICE_UNAVAILABLE")
		return
	}

	item, err := h.ApprovalUseCase.ApproveContent(ctx, id, "operator")
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrContentNotFound):
			WriteNotFound(w, "content not found")
		case errors.Is(err, usecase.ErrNotDraft):
			WriteError(w, http.StatusConflict, err.Error(), "INVALID_STATUS")
		case errors.Is(err, usecase.ErrAlreadyQueued):
			WriteError(w, http.StatusConflict, err.Error(), "ALREADY_QUEUED")
		default:
			h.Logger.Error("approve content failed", "error", err)
			WriteInternalError(w, "failed to approve content")
		}
		return
	}
	WriteSuccess(w, item)
}

func (h *ContentHandler) Reject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	if h.ApprovalUseCase == nil {
		WriteError(w, http.StatusServiceUnavailable, "approval service not configured", "SERVICE_UNAVAILABLE")
		return
	}

	if err := h.ApprovalUseCase.RejectContent(ctx, id, "operator", body.Reason); err != nil {
		switch {
		case errors.Is(err, usecase.ErrContentNotFound):
			WriteNotFound(w, "content not found")
		case errors.Is(err, usecase.ErrNotDraft):
			WriteError(w, http.StatusConflict, err.Error(), "INVALID_STATUS")
		default:
			h.Logger.Error("reject content failed", "error", err)
			WriteInternalError(w, "failed to reject content")
		}
		return
	}
	WriteSuccess(w, map[string]string{"id": id})
}

// ---------------------------------------------------------------------------
// Queue
// ---------------------------------------------------------------------------

// QueueHandler handles all /api/queue routes.
type QueueHandler struct {
	*BaseHandler
	ApprovalUseCase *usecase.ApprovalUseCase
}

func NewQueueHandler(base *BaseHandler, a *usecase.ApprovalUseCase) *QueueHandler {
	return &QueueHandler{base, a}
}

func (h *QueueHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	items, err := h.QueueRepo.FindAll(ctx)
	if err != nil {
		WriteInternalError(w, "failed to fetch queue")
		return
	}

	// Enrich queue items with content info for the UI list.
	byID := map[string]*entity.Content{}
	allContent, err := h.ContentRepo.FindAll(ctx, repository.ContentFilter{Limit: 0})
	if err == nil {
		for _, c := range allContent {
			byID[c.ID] = c
		}
	}

	out := make([]map[string]interface{}, 0, len(items))
	for _, q := range items {
		row := map[string]interface{}{
			"id":             q.ID,
			"content_id":     q.ContentID,
			"status":         q.Status,
			"priority":       q.Priority,
			"queued_at":      q.QueuedAt,
			"scheduler_type": q.SchedulerType,
		}
		if c := byID[q.ContentID]; c != nil {
			row["hook"] = c.Hook
			row["body"] = c.Body
			row["pillar"] = c.Pillar
			row["topic"] = c.Topic
		}
		out = append(out, row)
	}
	WriteSuccess(w, out)
}

func (h *QueueHandler) Remove(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	if h.ApprovalUseCase != nil {
		if err := h.ApprovalUseCase.RemoveFromQueue(ctx, id); err != nil {
			WriteInternalError(w, "failed to remove from queue")
			return
		}
		WriteSuccess(w, map[string]string{"id": id})
		return
	}
	if err := h.QueueRepo.Remove(ctx, id); err != nil {
		WriteInternalError(w, "failed to remove from queue")
		return
	}
	WriteSuccess(w, map[string]string{"id": id})
}

func (h *QueueHandler) UpdatePriority(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	var body struct {
		Priority int `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteBadRequest(w, "invalid request body")
		return
	}

	if h.ApprovalUseCase != nil {
		if err := h.ApprovalUseCase.UpdateQueuePriority(ctx, id, body.Priority); err != nil {
			WriteInternalError(w, "failed to update priority")
			return
		}
		WriteSuccess(w, map[string]interface{}{"id": id, "priority": body.Priority})
		return
	}
	if err := h.QueueRepo.UpdatePriority(ctx, id, body.Priority); err != nil {
		WriteInternalError(w, "failed to update priority")
		return
	}
	WriteSuccess(w, map[string]interface{}{"id": id, "priority": body.Priority})
}

// ---------------------------------------------------------------------------
// Schedule
// ---------------------------------------------------------------------------

// ScheduleHandler handles schedule-related routes.
type ScheduleHandler struct {
	*BaseHandler
}

func NewScheduleHandler(base *BaseHandler) *ScheduleHandler {
	return &ScheduleHandler{base}
}

func (h *ScheduleHandler) ScheduleContent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	var body struct {
		ScheduledAt time.Time `json:"scheduled_at"`
		Timezone    string    `json:"timezone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteBadRequest(w, "invalid request body")
		return
	}
	if body.Timezone == "" {
		body.Timezone = "Asia/Jakarta"
	}

	c, err := h.ContentRepo.FindByID(ctx, id)
	if err != nil || c == nil {
		WriteNotFound(w, "content not found")
		return
	}

	now := time.Now().UTC()
	schedule := &entity.Schedule{
		ID:            uuid.New().String(),
		ContentID:     id,
		ScheduledAt:   body.ScheduledAt,
		Timezone:      body.Timezone,
		SchedulerType: entity.SchedulerManual,
		ScheduledBy:   "operator",
		CreatedAt:     now,
	}

	if err := h.ScheduleRepo.Save(ctx, schedule); err != nil {
		WriteInternalError(w, "failed to create schedule")
		return
	}

	if err := h.ContentRepo.UpdateStatus(ctx, id, entity.StatusScheduled); err != nil {
		WriteInternalError(w, "failed to update content status")
		return
	}

	WriteCreated(w, schedule)
}

func (h *ScheduleHandler) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	// Find schedule by content ID
	all, err := h.ScheduleRepo.FindAll(ctx)
	if err != nil {
		WriteInternalError(w, "failed to fetch schedules")
		return
	}

	var scheduleID string
	for _, s := range all {
		if s.ContentID == id {
			scheduleID = s.ID
			break
		}
	}

	if scheduleID == "" {
		WriteNotFound(w, "schedule not found")
		return
	}

	if err := h.ScheduleRepo.Delete(ctx, scheduleID); err != nil {
		WriteInternalError(w, "failed to delete schedule")
		return
	}

	if err := h.ContentRepo.UpdateStatus(ctx, id, entity.StatusQueued); err != nil {
		WriteInternalError(w, "failed to revert content status")
		return
	}

	WriteSuccess(w, map[string]string{"id": scheduleID})
}

func (h *ScheduleHandler) ListSchedules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	schedules, err := h.ScheduleRepo.FindAll(ctx)
	if err != nil {
		WriteInternalError(w, "failed to fetch schedules")
		return
	}

	byID := map[string]*entity.Content{}
	allContent, err := h.ContentRepo.FindAll(ctx, repository.ContentFilter{Limit: 0})
	if err == nil {
		for _, c := range allContent {
			byID[c.ID] = c
		}
	}

	out := make([]map[string]interface{}, 0, len(schedules))
	for _, s := range schedules {
		row := map[string]interface{}{
			"id":             s.ID,
			"content_id":     s.ContentID,
			"scheduled_at":   s.ScheduledAt,
			"timezone":       s.Timezone,
			"type":           s.SchedulerType,
			"scheduler_type": s.SchedulerType,
			"scheduled_by":   s.ScheduledBy,
		}
		if c := byID[s.ContentID]; c != nil {
			row["hook"] = c.Hook
			row["body"] = c.Body
			row["pillar"] = c.Pillar
		}
		out = append(out, row)
	}
	WriteSuccess(w, out)
}

// ---------------------------------------------------------------------------
// Analytics
// ---------------------------------------------------------------------------

// AnalyticsHandler handles /api/analytics routes.
type AnalyticsHandler struct {
	*BaseHandler
}

func NewAnalyticsHandler(base *BaseHandler) *AnalyticsHandler {
	return &AnalyticsHandler{base}
}

func (h *AnalyticsHandler) Overview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	perfs, err := h.PerformanceRepo.FindAll(ctx)
	if err != nil {
		WriteInternalError(w, "failed to fetch analytics")
		return
	}

	var totalViews, totalLikes, totalReplies, totalReposts int64
	for _, p := range perfs {
		totalViews += p.Views
		totalLikes += p.Likes
		totalReplies += p.Replies
		totalReposts += p.Reposts
	}

	WriteSuccess(w, map[string]interface{}{
		"total_posts":      len(perfs),
		"total_views":      totalViews,
		"total_likes":      totalLikes,
		"total_replies":    totalReplies,
		"total_reposts":    totalReposts,
		"total_engagement": totalLikes + totalReplies + totalReposts,
	})
}

func (h *AnalyticsHandler) ByHour(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	perfs, err := h.PerformanceRepo.FindAll(ctx)
	if err != nil {
		WriteInternalError(w, "failed to fetch analytics")
		return
	}

	// Aggregate by publish hour and return a sorted slice of rows.
	hourMap := make(map[int]*analyticsRow)
	for _, p := range perfs {
		hour := p.PublishedAt.Hour()
		if hourMap[hour] == nil {
			hourMap[hour] = &analyticsRow{}
		}
		hourMap[hour].add(*p)
	}

	rows := make([]map[string]interface{}, 0, len(hourMap))
	for hour := 0; hour < 24; hour++ {
		if r := hourMap[hour]; r != nil {
			rows = append(rows, r.toHourMap(hour))
		}
	}
	WriteSuccess(w, rows)
}

func (h *AnalyticsHandler) ByPillar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	perfs, err := h.PerformanceRepo.FindAll(ctx)
	if err != nil {
		WriteInternalError(w, "failed to fetch analytics")
		return
	}

	pillarMap := make(map[entity.ContentPillar]*analyticsRow)
	for _, p := range perfs {
		if pillarMap[p.Pillar] == nil {
			pillarMap[p.Pillar] = &analyticsRow{}
		}
		pillarMap[p.Pillar].add(*p)
	}

	rows := make([]map[string]interface{}, 0, len(pillarMap))
	for _, pillar := range pillarOrder {
		if r := pillarMap[pillar]; r != nil {
			rows = append(rows, r.toPillarMap(pillar))
		}
	}
	WriteSuccess(w, rows)
}

type analyticsRow struct {
	Views   int64
	Likes   int64
	Replies int64
	Reposts int64
	Posts   int64
}

func (r *analyticsRow) add(p entity.PostPerformance) {
	r.Views += p.Views
	r.Likes += p.Likes
	r.Replies += p.Replies
	r.Reposts += p.Reposts
	r.Posts++
}

func (r *analyticsRow) toHourMap(hour int) map[string]interface{} {
	return map[string]interface{}{
		"hour":    hour,
		"posts":   r.Posts,
		"views":   r.Views,
		"likes":   r.Likes,
		"replies": r.Replies,
		"reposts": r.Reposts,
	}
}

func (r *analyticsRow) toPillarMap(pillar entity.ContentPillar) map[string]interface{} {
	return map[string]interface{}{
		"pillar":  pillar,
		"posts":   r.Posts,
		"views":   r.Views,
		"likes":   r.Likes,
		"replies": r.Replies,
		"reposts": r.Reposts,
	}
}

var pillarOrder = []entity.ContentPillar{
	entity.PillarEdukasiHukum,
	entity.PillarTipsHukum,
	entity.PillarCeritaWarga,
	entity.PillarMythVsFact,
	entity.PillarCommunity,
	entity.PillarEvent,
	entity.PillarTraining,
	entity.PillarMembership,
	entity.PillarEngagement,
}

// ---------------------------------------------------------------------------
// Topics
// ---------------------------------------------------------------------------

// TopicHandler handles /api/topics routes.
type TopicHandler struct {
	*BaseHandler
}

func NewTopicHandler(base *BaseHandler) *TopicHandler {
	return &TopicHandler{base}
}

func (h *TopicHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	topics, err := h.TopicRepo.FindAll(ctx)
	if err != nil {
		WriteInternalError(w, "failed to fetch topics")
		return
	}
	WriteSuccess(w, topics)
}

func (h *TopicHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var body struct {
		Name   string               `json:"name"`
		Pillar entity.ContentPillar `json:"pillar"`
		Active bool                 `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteBadRequest(w, "invalid request body")
		return
	}
	if body.Name == "" {
		WriteBadRequest(w, "name is required")
		return
	}

	topic := &entity.Topic{
		ID:        uuid.New().String(),
		Name:      body.Name,
		Pillar:    body.Pillar,
		Active:    body.Active,
		CreatedAt: time.Now().UTC(),
	}

	if err := h.TopicRepo.Save(ctx, topic); err != nil {
		WriteInternalError(w, "failed to create topic")
		return
	}
	WriteCreated(w, topic)
}

// ---------------------------------------------------------------------------
// Settings
// ---------------------------------------------------------------------------

// SettingsHandler handles /api/settings routes.
type SettingsHandler struct {
	*BaseHandler
	settings map[string]interface{}
}

func NewSettingsHandler(base *BaseHandler, initialSettings map[string]interface{}) *SettingsHandler {
	return &SettingsHandler{BaseHandler: base, settings: initialSettings}
}

func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	WriteSuccess(w, h.settings)
}

func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteBadRequest(w, "invalid request body")
		return
	}
	for k, v := range body {
		h.settings[k] = v
	}
	WriteSuccess(w, h.settings)
}

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

// AuthHandler handles Threads OAuth routes.
type AuthHandler struct {
	ThreadsPort port.ThreadsPort
}

func NewAuthHandler(tp port.ThreadsPort) *AuthHandler {
	return &AuthHandler{ThreadsPort: tp}
}

func (h *AuthHandler) RedirectToAuth(w http.ResponseWriter, r *http.Request) {
	if h.ThreadsPort == nil {
		WriteError(w, http.StatusServiceUnavailable, "Threads integration not configured", "THREADS_UNAVAILABLE")
		return
	}
	authURL := h.ThreadsPort.GetAuthURL()
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if h.ThreadsPort == nil {
		WriteError(w, http.StatusServiceUnavailable, "Threads integration not configured", "THREADS_UNAVAILABLE")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		WriteBadRequest(w, "code query parameter is required")
		return
	}

	token, err := h.ThreadsPort.ExchangeCode(r.Context(), code)
	if err != nil {
		h.ThreadsPort.GetAuthURL() // noop to avoid unused-variable lint
		WriteInternalError(w, "failed to exchange code for token")
		return
	}

	WriteSuccess(w, map[string]string{"access_token": token})
}
