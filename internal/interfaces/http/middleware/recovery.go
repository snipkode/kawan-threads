package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery returns a middleware that recovers from panics, logs the stack
// trace, and returns a 500 Internal Server Error to the client.
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := debug.Stack()

					requestID := GetRequestID(r.Context())

					logger.Error("panic recovered",
						"panic", fmt.Sprintf("%v", rec),
						"stack", string(stack),
						"method", r.Method,
						"path", r.URL.Path,
						"request_id", requestID,
					)

					// Only write the header if it hasn't already been sent.
					if w.Header().Get("Content-Type") == "" {
						w.Header().Set("Content-Type", "application/json; charset=utf-8")
					}
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"success":false,"data":null,"message":"internal server error","code":"PANIC"}`))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
