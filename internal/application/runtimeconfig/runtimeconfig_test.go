package runtimeconfig

import (
	"context"
	"testing"

	"kawan-threads/internal/config"
)

type memSettingsRepo struct {
	values map[string]interface{}
}

func (m *memSettingsRepo) GetSettings(_ context.Context) (map[string]interface{}, error) {
	if m.values == nil {
		return map[string]interface{}{}, nil
	}
	out := make(map[string]interface{}, len(m.values))
	for k, v := range m.values {
		out[k] = v
	}
	return out, nil
}

func (m *memSettingsRepo) SaveSettings(_ context.Context, settings map[string]interface{}) error {
	m.values = settings
	return nil
}

func testConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{Env: "development", Port: "8080"},
		Store: config.StoreConfig{
			DataStore: "file",
			DataFile:  "data/db.json",
		},
		Gemini: config.GeminiConfig{
			APIKey: "env-key",
			Model:  "gemini-3.6-flash",
		},
		Threads: config.ThreadsConfig{
			ClientID:    "env-client",
			UserID:      "env-user",
			AccessToken: "env-token",
		},
		Settings: config.SettingsConfig{
			MaxPostsPerDay:         5,
			MinPostIntervalMinutes: 90,
			ExplorationRate:        0.20,
			MaxRetry:               3,
		},
	}
}

func TestSeedFromConfig(t *testing.T) {
	repo := &memSettingsRepo{}
	s := New(repo, testConfig(), nil)

	if got := s.Str(GeminiAPIKey, ""); got != "env-key" {
		t.Fatalf("gemini key = %q, want env-key", got)
	}
	if got := s.Int(MaxPostsPerDay, 0); got != 5 {
		t.Fatalf("max posts/day = %d, want 5", got)
	}
	if got := s.Float(ExplorationRate, 0); got != 0.20 {
		t.Fatalf("exploration = %v, want 0.2", got)
	}
	if !s.GeminiConfigured() {
		t.Fatal("expected gemini configured")
	}
}

func TestUpdatePersistsAndOverrides(t *testing.T) {
	repo := &memSettingsRepo{}
	s := New(repo, testConfig(), nil)
	s.Load(context.Background())

	if _, err := s.Update(context.Background(), map[string]interface{}{
		GeminiAPIKey:    "ui-key",
		MaxPostsPerDay:  "12",
		ExplorationRate: 0.75,
		AutoApproval:    "true",
	}); err != nil {
		t.Fatalf("update: %v", err)
	}

	// Normalisation: string "12" → int 12, "true" → bool, clamped rate.
	if got := s.Str(GeminiAPIKey, ""); got != "ui-key" {
		t.Fatalf("gemini key = %q, want ui-key", got)
	}
	if got := s.Int(MaxPostsPerDay, 0); got != 12 {
		t.Fatalf("max posts/day = %d, want 12", got)
	}
	if got := s.Float(ExplorationRate, 0); got != 0.75 {
		t.Fatalf("exploration = %v, want 0.75", got)
	}
	if !s.Bool(AutoApproval, false) {
		t.Fatal("auto_approval should be true")
	}

	// A fresh store (simulating the worker) must see the persisted values.
	s2 := New(repo, testConfig(), nil)
	s2.Load(context.Background())
	if got := s2.Str(GeminiAPIKey, ""); got != "ui-key" {
		t.Fatalf("worker sees gemini key = %q, want ui-key", got)
	}
	if got := s2.Int(MaxPostsPerDay, 0); got != 12 {
		t.Fatalf("worker max posts/day = %d, want 12", got)
	}
}

func TestReloadPicksUpCrossProcessChanges(t *testing.T) {
	repo := &memSettingsRepo{}
	s := New(repo, testConfig(), nil)
	s.Load(context.Background())

	// Another "process" writes directly to the repo.
	if err := repo.SaveSettings(context.Background(), map[string]interface{}{
		MaxPostsPerDay: 9,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	s.Reload(context.Background())
	if got := s.Int(MaxPostsPerDay, 0); got != 9 {
		t.Fatalf("after reload max posts/day = %d, want 9", got)
	}
	// Unrelated seed values are untouched.
	if got := s.Str(GeminiAPIKey, ""); got != "env-key" {
		t.Fatalf("gemini key = %q, want env-key", got)
	}
}

func TestLockedKeysCannotBeOverridden(t *testing.T) {
	repo := &memSettingsRepo{}
	s := New(repo, testConfig(), nil)
	if _, err := s.Update(context.Background(), map[string]interface{}{
		DataStore:    "firebase",
		GeminiAPIKey: "ok",
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if got := s.Str(DataStore, ""); got != "file" {
		t.Fatalf("data_store = %q, want locked to file", got)
	}
	if got := s.Str(GeminiAPIKey, ""); got != "ok" {
		t.Fatalf("gemini key = %q, want ok", got)
	}
}

func TestKeyTypeNormalisation(t *testing.T) {
	repo := &memSettingsRepo{}
	s := New(repo, testConfig(), nil)
	vals, err := s.Update(context.Background(), map[string]interface{}{
		ExplorationRate: 1.5, // clamp to 1.0
		MaxRetry:        -2,  // clamp to 0
		AutoPublish:     "1",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if vals[ExplorationRate] != 1.0 {
		t.Fatalf("exploration = %v, want 1.0", vals[ExplorationRate])
	}
	if vals[MaxRetry] != 0 {
		t.Fatalf("max retry = %v, want 0", vals[MaxRetry])
	}
	if !s.Bool(AutoPublish, false) {
		t.Fatal("auto_publish should be true")
	}
}
