package scheduler

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"kawan-threads/internal/domain/entity"
	"kawan-threads/internal/domain/repository"
)

// ---------------------------------------------------------------------------
// In-memory PostPerformanceRepository for AMAB tests
// ---------------------------------------------------------------------------

type memPerformanceRepo struct {
	items []*entity.PostPerformance
}

func (m *memPerformanceRepo) Save(_ context.Context, _ *entity.PostPerformance) error { return nil }
func (m *memPerformanceRepo) FindByPostID(_ context.Context, _ string) (*entity.PostPerformance, error) {
	return nil, nil
}
func (m *memPerformanceRepo) FindAll(_ context.Context) ([]*entity.PostPerformance, error) {
	return m.items, nil
}
func (m *memPerformanceRepo) FindByPillar(_ context.Context, p entity.ContentPillar) ([]*entity.PostPerformance, error) {
	out := make([]*entity.PostPerformance, 0)
	for _, i := range m.items {
		if i.Pillar == p {
			out = append(out, i)
		}
	}
	return out, nil
}

var _ repository.PostPerformanceRepository = (*memPerformanceRepo)(nil)

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestAMAB_ExplorationWhenFewSamples(t *testing.T) {
	repo := &memPerformanceRepo{items: []*entity.PostPerformance{
		{Pillar: entity.PillarCommunity, PublishedAt: time.Now().Add(-time.Hour), Views: 100, Replies: 5, Reposts: 2, Quotes: 1},
	}}
	amab := NewAMABScorer(repo, slog.Default(), 0.20)

	for i := 0; i < 10; i++ {
		score, err := amab.Score(context.Background(), "c1", entity.PillarCommunity, entity.HookQuestion, 19)
		if err != nil {
			t.Fatalf("score error: %v", err)
		}
		if score < 0 || score > 1 {
			t.Fatalf("score out of [0,1]: %f", score)
		}
	}
}

func TestAMAB_ExploitationConvergesOnBestHour(t *testing.T) {
	now := time.Now()
	items := make([]*entity.PostPerformance, 0, 12)
	// 6 samples at hour 19 with high views.
	for i := 0; i < 6; i++ {
		items = append(items, &entity.PostPerformance{
			Pillar: entity.PillarEdukasiHukum, PublishedAt: now.Add(-time.Duration(i) * time.Hour).Add(time.Hour),
			Views: 1000, Replies: 90, Reposts: 30, Quotes: 10,
		})
	}
	// 6 samples at hour 12 with low views.
	for i := 0; i < 6; i++ {
		items = append(items, &entity.PostPerformance{
			Pillar: entity.PillarEdukasiHukum, PublishedAt: now.Add(-time.Duration(i) * time.Hour),
			Views: 50, Replies: 2, Reposts: 0, Quotes: 0,
		})
	}
	// Ensure the hour-19 samples really are at hour 19-ish and hour 12 at 12-ish.
	base := time.Date(now.Year(), now.Month(), now.Day(), 19, 30, 0, 0, time.UTC)
	for i := 0; i < 6; i++ {
		items[i].PublishedAt = base.Add(-time.Duration(i) * 7 * 24 * time.Hour)
	}
	base12 := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC)
	for i := 0; i < 6; i++ {
		items[6+i].PublishedAt = base12.Add(-time.Duration(i) * 7 * 24 * time.Hour)
	}

	repo := &memPerformanceRepo{items: items}
	amab := NewAMABScorer(repo, slog.Default(), 0.0) // full exploitation

	score19, err := amab.Score(context.Background(), "c1", entity.PillarEdukasiHukum, entity.HookQuestion, 19)
	if err != nil {
		t.Fatalf("score19 error: %v", err)
	}
	score12, err := amab.Score(context.Background(), "c1", entity.PillarEdukasiHukum, entity.HookQuestion, 12)
	if err != nil {
		t.Fatalf("score12 error: %v", err)
	}

	if score19 <= score12 {
		t.Fatalf("expected hour 19 (high views) to outscore hour 12, got 19=%f 12=%f", score19, score12)
	}
}

func TestAMAB_CheckDiversityFromContents(t *testing.T) {
	amab := NewAMABScorer(&memPerformanceRepo{}, slog.Default(), 0.20)

	community := &entity.Content{Pillar: entity.PillarCommunity, Topic: "Hak warga", HookType: entity.HookQuestion}
	education := &entity.Content{Pillar: entity.PillarEdukasiHukum, Topic: "Hak buruh", HookType: entity.HookRelatable}

	recent := []*entity.Content{community, education}

	// A third community pillar inside the last 2 → fail diversity.
	candidateCommunity := &entity.Content{Pillar: entity.PillarCommunity, Topic: "Akses keadilan", HookType: entity.HookCuriosity}
	if amab.checkDiversityFromContents(recent, candidateCommunity) {
		t.Fatal("expected diversity check to reject duplicate pillar within last 2")
	}

	// A different pillar passes.
	candidateEvent := &entity.Content{Pillar: entity.PillarEvent, Topic: "Webinar hukum", HookType: entity.HookStory}
	if !amab.checkDiversityFromContents(recent, candidateEvent) {
		t.Fatal("expected diversity check to accept different pillar")
	}

	// Same topic within last 3 → fail.
	candidateSameTopic := &entity.Content{Pillar: entity.PillarCeritaWarga, Topic: "Hak warga", HookType: entity.HookProblem}
	if amab.checkDiversityFromContents(recent, candidateSameTopic) {
		t.Fatal("expected diversity check to reject duplicate topic within last 3")
	}
}

func TestAMAB_GetStrategyRecommendation_NoData(t *testing.T) {
	amab := NewAMABScorer(&memPerformanceRepo{}, slog.Default(), 0.20)
	rec, err := amab.GetStrategyRecommendation(context.Background())
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if rec.AnalyzedPosts != 0 || rec.SuggestedAction == "" {
		t.Fatalf("unexpected empty-data recommendation: %+v", rec)
	}
}

func TestAMAB_SelectPostingWindow_ReturnsFutureTime(t *testing.T) {
	amab := NewAMABScorer(&memPerformanceRepo{}, slog.Default(), 0.0)
	now := time.Now()
	w, err := amab.SelectPostingWindow(context.Background(), now, time.UTC)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !w.After(now) {
		t.Fatalf("window %v should be after now %v", w, now)
	}
}
