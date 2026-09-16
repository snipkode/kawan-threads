package publishing

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"kawan-threads/internal/domain/entity"
	"kawan-threads/internal/domain/port"
	"kawan-threads/internal/infrastructure/threads"
)

// fakeThreadsPort simulates the Threads API with configurable behaviour.
type fakeThreadsPort struct {
	failFirstN  int
	createCalls int
	pubCalls    int
	postID      string
	rateLimited bool
}

func (f *fakeThreadsPort) CreateTextContainer(_ context.Context, _, _ string) (string, error) {
	f.createCalls++
	if f.rateLimited && f.createCalls <= f.failFirstN {
		return "", &threads.RateLimitError{RetryAfter: 0, Message: "too many requests"}
	}
	return "creation_xyz", nil
}

func (f *fakeThreadsPort) PublishContainer(_ context.Context, _, _ string) (string, error) {
	f.pubCalls++
	return f.postID, nil
}

func (f *fakeThreadsPort) GetPostInsights(_ context.Context, _, _ string) (*port.ThreadsInsights, error) {
	return nil, nil
}

func (f *fakeThreadsPort) GetAuthURL() string { return "" }

func (f *fakeThreadsPort) ExchangeCode(_ context.Context, _ string) (string, error) { return "", nil }

func (f *fakeThreadsPort) RefreshToken(_ context.Context, _ string) (string, error) { return "", nil }

func newTestPublisher(maxRetry int, f *fakeThreadsPort) *Publisher {
	return &Publisher{
		ThreadsPort: f,
		Logger:      slog.Default(),
		MaxRetry:    maxRetry,
		Timezone:    time.UTC,
		Backoff:     []time.Duration{time.Millisecond, 2 * time.Millisecond, 5 * time.Millisecond},
	}
}

func contentFixture() *entity.Content {
	return &entity.Content{
		ID:                   "c1",
		Hook:                 "Hangat kah?",
		Body:                 "Tubuh.",
		CTA:                  "Gabung KAWAN.",
		ConversationQuestion: "Setuju?",
		Status:               entity.StatusScheduled,
	}
}

func TestComposeText_JoinsSections(t *testing.T) {
	got := composeText(contentFixture())
	want := "Hangat kah?\n\nTubuh.\n\nGabung KAWAN.\n\nSetuju?"
	if got != want {
		t.Fatalf("unexpected text:\n got: %q\nwant: %q", got, want)
	}
}

func TestComposeText_SkipsEmpty(t *testing.T) {
	c := &entity.Content{ID: "x", Hook: "Only hook"}
	got := composeText(c)
	if got != "Only hook" {
		t.Fatalf("expected only hook, got %q", got)
	}
}

func TestPublisher_RetriesRateLimitThenSucceeds(t *testing.T) {
	f := &fakeThreadsPort{failFirstN: 1, rateLimited: true, postID: "post_1"}
	p := newTestPublisher(3, f)

	postID, err := p.createAndPublish(context.Background(), "text", contentFixture(), &entity.Schedule{})
	if err != nil {
		t.Fatalf("createAndPublish failed: %v", err)
	}
	if postID != "post_1" {
		t.Fatalf("unexpected post id: %s", postID)
	}
	if f.createCalls != 2 {
		t.Fatalf("expected 2 create calls (1 rate-limited + 1 success), got %d", f.createCalls)
	}
}

func TestPublisher_GivesUpAfterMaxRetry(t *testing.T) {
	f := &fakeThreadsPort{failFirstN: 99, rateLimited: true}
	p := newTestPublisher(2, f)

	_, err := p.createAndPublish(context.Background(), "text", contentFixture(), &entity.Schedule{})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if f.createCalls != 3 { // initial + 2 retries
		t.Fatalf("expected 3 create calls, got %d", f.createCalls)
	}
	var rle *threads.RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("expected RateLimitError, got %v", err)
	}
}

func TestPublisher_BackoffDuration(t *testing.T) {
	p := newTestPublisher(3, nil)
	backoffs := []time.Duration{5 * time.Second, 30 * time.Second, 2 * time.Minute}

	if got := p.backoffDuration(0, backoffs); got != 5*time.Second {
		t.Fatalf("attempt 0 backoff = %v, want 5s", got)
	}
	if got := p.backoffDuration(1, backoffs); got != 30*time.Second {
		t.Fatalf("attempt 1 backoff = %v, want 30s", got)
	}
	if got := p.backoffDuration(5, backoffs); got != 2*time.Minute {
		t.Fatalf("out-of-range backoff should cap at 2m, got %v", got)
	}
}
