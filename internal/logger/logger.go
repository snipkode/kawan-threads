package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Log event name constants matching the spec.
const (
	EventWorkerStarted           = "worker_started"
	EventSchedulerTriggered      = "scheduler_triggered"
	EventGeminiGenerationStarted = "gemini_generation_started"
	EventContentGenerated        = "content_generated"
	EventContentValidated        = "content_validated"
	EventContentRejected         = "content_rejected"
	EventContentApproved         = "content_approved"
	EventContentQueued           = "content_queued"
	EventContentScheduled        = "content_scheduled"
	EventPublishStarted          = "publish_started"
	EventPublishSuccess          = "publish_success"
	EventPublishFailed           = "publish_failed"
	EventAnalyticsCollected      = "analytics_collected"
	EventAmabUpdated             = "amab_updated"
)

// Global is the application-wide logger. Initialised to a sensible default;
// call Init to replace it with a configured instance.
var Global *slog.Logger

func init() {
	Global = New("info")
}

// Init replaces the Global logger with a new instance at the given level.
func Init(level string) {
	Global = New(level)
}

// New creates a JSON-structured slog.Logger at the requested log level.
// Valid levels: "debug", "info", "warn", "error". Defaults to "info".
func New(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: lvl,
		// Include the source file location in debug mode.
		AddSource: lvl == slog.LevelDebug,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(handler)
}

// LogEvent emits a structured log entry for a named event.
//
// Usage:
//
//	logger.LogEvent(log, logger.EventPublishSuccess,
//	    "content_id", id,
//	    "threads_post_id", postID,
//	)
//
// `fields` must be key-value pairs (string key, any value).
// An odd number of fields will add a final "!EXTRA" sentinel key.
func LogEvent(l *slog.Logger, event string, fields ...any) {
	if l == nil {
		l = Global
	}

	// Prepend the event name as a dedicated attribute so it is always the first
	// field in the JSON object and is easy to filter on.
	args := make([]any, 0, len(fields)+2)
	args = append(args, "event", event)
	args = append(args, fields...)

	l.Info("event", args...)
}
