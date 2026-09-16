package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_AllowsBurstThenBlocks(t *testing.T) {
	rl := NewRateLimiter(3, 1) // burst 3, refill 1/s
	b := newTokenBucket(3, 1)
	now := time.Now()

	// First 3 requests are allowed.
	if !b.allow(now) {
		t.Fatal("expected 1st request to be allowed")
	}
	if !b.allow(now) {
		t.Fatal("expected 2nd request to be allowed")
	}
	if !b.allow(now) {
		t.Fatal("expected 3rd request to be allowed")
	}
	if b.allow(now) {
		t.Fatal("expected 4th request within burst to be blocked")
	}

	// After refill time passes, one token becomes available.
	if !b.allow(now.Add(time.Second)) {
		t.Fatal("expected a token to refill after 1 second")
	}
	if b.allow(now.Add(time.Second)) {
		t.Fatal("expected bucket to be empty again immediately after use")
	}

	_ = rl // keeps the constructor referenced
}

func TestRateLimitMiddleware_429WhenExceeded(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	// Very slow refill so the burst is exhausted for every subsequent request.
	mw := RateLimit(2, 0.001)(next)

	srv := httptest.NewServer(mw)
	defer srv.Close()

	statuses := []int{}
	for i := 0; i < 4; i++ {
		resp, err := http.Get(srv.URL)
		if err != nil {
			t.Fatal(err)
		}
		statuses = append(statuses, resp.StatusCode)
		resp.Body.Close()
	}

	if statuses[0] != http.StatusOK || statuses[1] != http.StatusOK {
		t.Fatalf("expected first two requests to succeed, got %v", statuses)
	}
	if statuses[2] != http.StatusTooManyRequests || statuses[3] != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on third/fourth requests, got %v", statuses)
	}
}
