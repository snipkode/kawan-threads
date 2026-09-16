package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// contextKey is an unexported type for context keys defined in this package.
type contextKey string

// RequestIDKey is the context key under which the request ID is stored.
const RequestIDKey contextKey = "request_id"

// RequestIDHeader is the HTTP header name used to propagate request IDs.
const RequestIDHeader = "X-Request-ID"

// RequestID is a middleware that reads the X-Request-ID header from the
// incoming request.  If it is absent or empty a new UUID v4 is generated.
// The resulting ID is:
//   - stored in the request context under RequestIDKey,
//   - written back in the X-Request-ID response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if id == "" {
			id = uuid.New().String()
		}

		// Propagate the ID in the response so callers can correlate.
		w.Header().Set(RequestIDHeader, id)

		// Store it in the context for downstream handlers and loggers.
		ctx := context.WithValue(r.Context(), RequestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves the request ID stored in ctx by the RequestID
// middleware.  Returns an empty string if no ID was set.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}
