package entity

import "time"

// ---------------------------------------------------------------------------
// Enumerations
// ---------------------------------------------------------------------------

// ContentStatus represents the lifecycle state of a Content item.
type ContentStatus string

const (
	StatusDraft      ContentStatus = "DRAFT"
	StatusRejected   ContentStatus = "REJECTED"
	StatusQueued     ContentStatus = "QUEUED"
	StatusScheduled  ContentStatus = "SCHEDULED"
	StatusPublishing ContentStatus = "PUBLISHING"
	StatusPublished  ContentStatus = "PUBLISHED"
	StatusFailed     ContentStatus = "FAILED"
)

// ContentPillar represents the thematic pillar of content.
type ContentPillar string

const (
	PillarEdukasiHukum ContentPillar = "EDUKASI_HUKUM"
	PillarTipsHukum    ContentPillar = "TIPS_HUKUM"
	PillarCeritaWarga  ContentPillar = "CERITA_WARGA"
	PillarMythVsFact   ContentPillar = "MYTH_VS_FACT"
	PillarCommunity    ContentPillar = "COMMUNITY"
	PillarEvent        ContentPillar = "EVENT"
	PillarTraining     ContentPillar = "TRAINING"
	PillarMembership   ContentPillar = "MEMBERSHIP"
	PillarEngagement   ContentPillar = "ENGAGEMENT"
)

// ContentFormat describes whether content is a single post or a thread series.
type ContentFormat string

const (
	FormatSinglePost   ContentFormat = "SINGLE_POST"
	FormatThreadSeries ContentFormat = "THREAD_SERIES"
)

// HookType classifies the style of the opening hook.
type HookType string

const (
	HookQuestion       HookType = "QUESTION"
	HookRelatable      HookType = "RELATABLE"
	HookCuriosity      HookType = "CURIOSITY"
	HookStory          HookType = "STORY"
	HookMyth           HookType = "MYTH"
	HookProblem        HookType = "PROBLEM"
	HookSurprisingFact HookType = "SURPRISING_FACT"
	HookHotTake        HookType = "HOT_TAKE"
)

// SchedulerType identifies how a post was scheduled.
type SchedulerType string

const (
	SchedulerAMAB   SchedulerType = "AMAB"
	SchedulerManual SchedulerType = "MANUAL"
)

// ---------------------------------------------------------------------------
// Value Objects
// ---------------------------------------------------------------------------

// QualityScore holds the AI-evaluated quality dimensions for a content item.
type QualityScore struct {
	Hook         int `json:"hook"`
	Conversation int `json:"conversation"`
	Readability  int `json:"readability"`
	Originality  int `json:"originality"`
	Relevance    int `json:"relevance"`
	Shareability int `json:"shareability"`
}

// ---------------------------------------------------------------------------
// Entities
// ---------------------------------------------------------------------------

// HookVariant is an alternative opening hook that an admin can choose from.
type HookVariant struct {
	Hook     string   `json:"hook"`
	HookType HookType `json:"hook_type"`
}

// Content is the core domain entity representing a piece of AI-generated
// social-media content for the Threads platform.
type Content struct {
	ID                   string        `json:"id"`
	Pillar               ContentPillar `json:"pillar"`
	Topic                string        `json:"topic"`
	Audience             string        `json:"audience"`
	Tone                 string        `json:"tone"`
	Format               ContentFormat `json:"format"`
	CTA                  string        `json:"cta"`
	Hook                 string        `json:"hook"`
	HookVariants         []HookVariant `json:"hook_variants,omitempty"`
	Body                 string        `json:"body"`
	ConversationQuestion string        `json:"conversation_question"`
	HookType             HookType      `json:"hook_type"`
	ContentType          string        `json:"content_type"`
	Quality              QualityScore  `json:"quality"`
	Status               ContentStatus `json:"status"`
	Source               string        `json:"source"`
	Version              int           `json:"version"`
	PromptVersion        string        `json:"prompt_version"`
	Model                string        `json:"model"`
	CreatedAt            time.Time     `json:"created_at"`
	UpdatedAt            time.Time     `json:"updated_at"`
	ApprovedAt           *time.Time    `json:"approved_at,omitempty"`
	ApprovedBy           string        `json:"approved_by,omitempty"`
	RejectedAt           *time.Time    `json:"rejected_at,omitempty"`
	RejectedBy           string        `json:"rejected_by,omitempty"`
	RejectionReason      string        `json:"rejection_reason,omitempty"`
}

// ContentVersion stores a historical snapshot of a Content item's editable fields.
type ContentVersion struct {
	ID                   string    `json:"id"`
	ContentID            string    `json:"content_id"`
	Version              int       `json:"version"`
	Hook                 string    `json:"hook"`
	Body                 string    `json:"body"`
	CTA                  string    `json:"cta"`
	ConversationQuestion string    `json:"conversation_question"`
	EditedBy             string    `json:"edited_by"`
	EditedAt             time.Time `json:"edited_at"`
	Reason               string    `json:"reason"`
}

// QueueItem represents a Content item that has been approved and is waiting
// to be scheduled for publication.
type QueueItem struct {
	ID            string        `json:"id"`
	ContentID     string        `json:"content_id"`
	Status        ContentStatus `json:"status"`
	Priority      int           `json:"priority"`
	ApprovedAt    time.Time     `json:"approved_at"`
	QueuedAt      time.Time     `json:"queued_at"`
	SchedulerType SchedulerType `json:"scheduler_type"`
}

// Schedule holds a planned publication time for a Content item.
type Schedule struct {
	ID            string        `json:"id"`
	ContentID     string        `json:"content_id"`
	ScheduledAt   time.Time     `json:"scheduled_at"`
	Timezone      string        `json:"timezone"`
	SchedulerType SchedulerType `json:"scheduler_type"`
	ScheduledBy   string        `json:"scheduled_by"`
	CreatedAt     time.Time     `json:"created_at"`
}

// PublishedPost records the outcome of a successful or attempted publication.
type PublishedPost struct {
	ID             string    `json:"id"`
	ContentID      string    `json:"content_id"`
	ThreadsPostID  string    `json:"threads_post_id"`
	PublishedAt    time.Time `json:"published_at"`
	IdempotencyKey string    `json:"idempotency_key"`
	RetryCount     int       `json:"retry_count"`
}

// PostPerformance stores collected analytics metrics for a published post.
type PostPerformance struct {
	PostID      string        `json:"post_id"`
	ContentID   string        `json:"content_id"`
	Pillar      ContentPillar `json:"pillar"`
	Topic       string        `json:"topic"`
	HookType    HookType      `json:"hook_type"`
	Format      ContentFormat `json:"format"`
	ScheduledAt time.Time     `json:"scheduled_at"`
	PublishedAt time.Time     `json:"published_at"`
	Views       int64         `json:"views"`
	Likes       int64         `json:"likes"`
	Replies     int64         `json:"replies"`
	Reposts     int64         `json:"reposts"`
	Quotes      int64         `json:"quotes"`
	CollectedAt time.Time     `json:"collected_at"`
}

// Topic represents a content topic seed belonging to a pillar.
type Topic struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Pillar    ContentPillar `json:"pillar"`
	Active    bool          `json:"active"`
	CreatedAt time.Time     `json:"created_at"`
}

// Experiment represents an A/B test comparing two content variants.
type Experiment struct {
	ID         string     `json:"id"`
	ContentAID string     `json:"content_a_id"`
	ContentBID string     `json:"content_b_id"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"started_at"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
}

// History records an audit-trail entry for a Content item's state changes.
type History struct {
	ID        string        `json:"id"`
	ContentID string        `json:"content_id"`
	Action    string        `json:"action"`
	OldStatus ContentStatus `json:"old_status"`
	NewStatus ContentStatus `json:"new_status"`
	Actor     string        `json:"actor"`
	Note      string        `json:"note"`
	CreatedAt time.Time     `json:"created_at"`
}
