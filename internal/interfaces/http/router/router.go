// Package router wires all HTTP routes to their handlers using the Go 1.22
// enhanced net/http ServeMux with method+pattern routing.
package router

import (
	"net/http"

	"kawan-threads/internal/interfaces/http/handler"
	"kawan-threads/internal/interfaces/http/middleware"
)

// Deps bundles all handler instances needed to register routes.
type Deps struct {
	Health    *handler.HealthHandler
	Dashboard *handler.DashboardHandler
	Content   *handler.ContentHandler
	Queue     *handler.QueueHandler
	Schedule  *handler.ScheduleHandler
	Analytics *handler.AnalyticsHandler
	Topic     *handler.TopicHandler
	Settings  *handler.SettingsHandler
	Auth      *handler.AuthHandler
}

// New constructs a fully-configured http.Handler by registering all API
// routes and applying global middleware.
func New(deps Deps, corsOrigins ...string) http.Handler {
	mux := http.NewServeMux()

	// -----------------------------------------------------------------------
	// Health
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /api/health", deps.Health.ServeHTTP)

	// -----------------------------------------------------------------------
	// Dashboard
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /api/dashboard", deps.Dashboard.ServeHTTP)

	// -----------------------------------------------------------------------
	// Content
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /api/content", deps.Content.List)
	mux.HandleFunc("POST /api/content/generate", deps.Content.Generate)
	mux.HandleFunc("GET /api/content/{id}", deps.Content.GetByID)
	mux.HandleFunc("GET /api/content/{id}/preview", deps.Content.GetPreview)
	mux.HandleFunc("PUT /api/content/{id}", deps.Content.Update)
	mux.HandleFunc("POST /api/content/{id}/regenerate", deps.Content.Regenerate)
	mux.HandleFunc("POST /api/content/{id}/approve", deps.Content.Approve)
	mux.HandleFunc("POST /api/content/{id}/reject", deps.Content.Reject)

	// -----------------------------------------------------------------------
	// Queue
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /api/queue", deps.Queue.List)
	mux.HandleFunc("POST /api/queue/{id}/remove", deps.Queue.Remove)
	mux.HandleFunc("POST /api/queue/{id}/priority", deps.Queue.UpdatePriority)

	// -----------------------------------------------------------------------
	// Schedule
	// -----------------------------------------------------------------------
	mux.HandleFunc("POST /api/content/{id}/schedule", deps.Schedule.ScheduleContent)
	mux.HandleFunc("DELETE /api/content/{id}/schedule", deps.Schedule.DeleteSchedule)
	mux.HandleFunc("GET /api/schedules", deps.Schedule.ListSchedules)

	// -----------------------------------------------------------------------
	// Analytics
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /api/analytics", deps.Analytics.Overview)
	mux.HandleFunc("GET /api/analytics/hour", deps.Analytics.ByHour)
	mux.HandleFunc("GET /api/analytics/pillar", deps.Analytics.ByPillar)

	// -----------------------------------------------------------------------
	// Topics
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /api/topics", deps.Topic.List)
	mux.HandleFunc("POST /api/topics", deps.Topic.Create)

	// -----------------------------------------------------------------------
	// Settings
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /api/settings", deps.Settings.Get)
	mux.HandleFunc("PUT /api/settings", deps.Settings.Update)

	// -----------------------------------------------------------------------
	// Auth
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /api/auth/threads", deps.Auth.RedirectToAuth)
	mux.HandleFunc("GET /api/auth/threads/callback", deps.Auth.Callback)

	// -----------------------------------------------------------------------
	// Apply global middleware (outermost = first to execute)
	// -----------------------------------------------------------------------
	// The logger is obtained from the health handler; in production each
	// middleware receives the same shared logger passed during construction.
	var h http.Handler = mux

	// Middleware stack (applied inside-out):
	// 1. Recovery  – must wrap everything so panics are always caught
	// 2. RequestID – inject correlation ID
	// 3. CORS      – set CORS headers / handle preflight
	if len(corsOrigins) == 0 {
		corsOrigins = []string{"*"}
	}
	h = middleware.RateLimit(120, 2)(h) // burst 120, ~2 req/s sustained per IP
	h = middleware.CORS(corsOrigins...)(h)
	h = middleware.RequestID(h)

	return h
}
