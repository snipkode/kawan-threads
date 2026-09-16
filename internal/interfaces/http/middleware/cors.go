// Package middleware provides reusable HTTP middleware components.
package middleware

import (
	"net/http"
	"strings"
)

// CORS returns a middleware that sets the appropriate CORS headers on every
// response.  allowedOrigins may contain exact origins or "*".
func CORS(allowedOrigins ...string) func(http.Handler) http.Handler {
	originSet := make(map[string]struct{}, len(allowedOrigins))
	wildcardAll := false
	for _, o := range allowedOrigins {
		if o == "*" {
			wildcardAll = true
		}
		originSet[strings.ToLower(o)] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Determine whether to reflect this origin or use "*".
			allowOrigin := ""
			if wildcardAll {
				allowOrigin = "*"
			} else if origin != "" {
				if _, ok := originSet[strings.ToLower(origin)]; ok {
					allowOrigin = origin
					w.Header().Set("Vary", "Origin")
				}
			}

			if allowOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
				w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			// Handle pre-flight requests.
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
