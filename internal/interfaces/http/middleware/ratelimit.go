package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// tokenBucket is a fixed-capacity token bucket that refills at a constant rate.
type tokenBucket struct {
	capacity float64
	tokens   float64
	refill   float64 // tokens per second
	last     time.Time
}

func newTokenBucket(capacity float64, refillPerSec float64) *tokenBucket {
	return &tokenBucket{
		capacity: capacity,
		tokens:   capacity,
		refill:   refillPerSec,
		last:     time.Now(),
	}
}

// allow consumes one token. It returns false when the bucket is empty.
func (b *tokenBucket) allow(now time.Time) bool {
	b.tokens += now.Sub(b.last).Seconds() * b.refill
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// RateLimiter implements per-IP in-memory rate limiting.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*tokenBucket
	capacity float64
	refill   float64
}

// NewRateLimiter builds a RateLimiter with the given capacity (burst) and
// refill rate in tokens/second.
func NewRateLimiter(capacity, refillPerSec float64) *RateLimiter {
	return &RateLimiter{
		buckets:  make(map[string]*tokenBucket),
		capacity: capacity,
		refill:   refillPerSec,
	}
}

// clientIP extracts the client IP from the remote address.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// RateLimit returns middleware that allows at most `capacity` requests per
// burst with a `refillPerSec` sustained rate per IP. Exceeding the limit
// yields HTTP 429.
func RateLimit(capacity, refillPerSec float64) func(http.Handler) http.Handler {
	rl := NewRateLimiter(capacity, refillPerSec)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			now := time.Now()

			rl.mu.Lock()
			b, ok := rl.buckets[ip]
			if !ok {
				b = newTokenBucket(rl.capacity, rl.refill)
				rl.buckets[ip] = b
			}
			allowed := b.allow(now)
			rl.mu.Unlock()

			if !allowed {
				w.Header().Set("Retry-After", "1")
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"success":false,"data":null,"message":"rate limit exceeded","code":"RATE_LIMITED"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
