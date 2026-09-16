// Package runtimeconfig provides a persistent, runtime-editable configuration
// store. Values are seeded from environment configuration at startup and can
// then be changed at run time through the Settings API (and the web UI).
// Persisted overrides are shared between the API process (writer) and the
// worker process (reader) through the SettingsRepository.
//
// Only infrastructure concerns (datastore mode, Firebase service account,
// port, …) remain environment-only; everything else is editable at run time.
package runtimeconfig

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"kawan-threads/internal/config"
	"kawan-threads/internal/domain/repository"
)

// Keys of every value that can be managed at run time. Values marked (locked)
// are read from the environment only and exposed read-only for visibility.
const (
	GeminiAPIKey              = "gemini_api_key"
	GeminiModel               = "gemini_model"
	ThreadsClientID           = "threads_client_id"
	ThreadsClientSecret       = "threads_client_secret"
	ThreadsRedirectURI        = "threads_redirect_uri"
	ThreadsAccessToken        = "threads_access_token"
	ThreadsUserID             = "threads_user_id"
	SchedulerTimezone         = "timezone"
	SchedulerIntervalMinutes  = "scheduler_interval_minutes"
	AutoApproval              = "auto_approval"
	AutoPublish               = "auto_publish"
	MaxPostsPerDay            = "max_posts_per_day"
	MinPostIntervalMinutes    = "min_post_interval_minutes"
	ExplorationRate           = "exploration_rate"
	MaxRetry                  = "max_retry"
	DataStore                 = "data_store"    // locked: env only
	AppEnv                    = "app_env"       // locked: env only
	FirebaseDatabaseURL       = "db_url"        // locked: env only
	FirebaseServiceAccountSet = "sa_configured" // locked: env only (bool)

	// Autopilot — daily background content generation
	AutopilotEnabled    = "autopilot_enabled"
	AutopilotDailyCount = "autopilot_daily_count"  // total drafts per day (default 3)
	AutopilotGoal       = "autopilot_goal"          // "website_visit" | "training_signup" | "community_growth"
	AutopilotRunHour    = "autopilot_run_hour"      // hour (0-23 local time) to trigger generation, default 7
)

// intKeys are normalised as integers (clamped >= 0).
var intKeys = map[string]bool{
	SchedulerIntervalMinutes: true,
	MaxPostsPerDay:           true,
	MinPostIntervalMinutes:   true,
	MaxRetry:                 true,
	AutopilotDailyCount:      true,
	AutopilotRunHour:         true,
}

// floatKeys are normalised as floats (clamped to [0, 1]).
var floatKeys = map[string]bool{
	ExplorationRate: true,
}

// Store is a concurrency-safe runtime configuration holder.
type Store struct {
	mu     sync.RWMutex
	values map[string]interface{}
	seed   map[string]interface{}
	repo   repository.SettingsRepository
	logger *slog.Logger
}

// New constructs a Store seeded from the environment configuration. Persisted
// overrides are loaded separately via Load.
func New(repo repository.SettingsRepository, cfg *config.Config, logger *slog.Logger) *Store {
	if logger == nil {
		logger = slog.Default()
	}
	seed := map[string]interface{}{
		GeminiAPIKey:             cfg.Gemini.APIKey,
		GeminiModel:              cfg.Gemini.Model,
		ThreadsClientID:          cfg.Threads.ClientID,
		ThreadsClientSecret:      cfg.Threads.ClientSecret,
		ThreadsRedirectURI:       cfg.Threads.RedirectURI,
		ThreadsAccessToken:       cfg.Threads.AccessToken,
		ThreadsUserID:            cfg.Threads.UserID,
		SchedulerTimezone:        cfg.Scheduler.Timezone,
		SchedulerIntervalMinutes: cfg.Scheduler.IntervalMinutes,
		AutoApproval:             cfg.Settings.AutoApproval,
		AutoPublish:              cfg.Settings.AutoPublish,
		MaxPostsPerDay:           cfg.Settings.MaxPostsPerDay,
		MinPostIntervalMinutes:   cfg.Settings.MinPostIntervalMinutes,
		ExplorationRate:          cfg.Settings.ExplorationRate,
		MaxRetry:                 cfg.Settings.MaxRetry,
		// Autopilot defaults
		AutopilotEnabled:    false,
		AutopilotDailyCount: 3,
		AutopilotGoal:       "website_visit",
		AutopilotRunHour:    7,
	}
	values := make(map[string]interface{}, len(seed)+4)
	for k, v := range seed {
		values[k] = v
	}
	values[DataStore] = cfg.Store.DataStore
	values[AppEnv] = cfg.App.Env
	values[FirebaseDatabaseURL] = cfg.Firebase.DatabaseURL
	values[FirebaseServiceAccountSet] = cfg.Firebase.ServiceAccountBase64 != ""

	return &Store{values: values, seed: seed, repo: repo, logger: logger}
}

// Load overlays any persisted settings on top of the environment seed.
// Persisted values win; keys never seen before are ignored.
func (s *Store) Load(ctx context.Context) {
	persisted, err := s.repo.GetSettings(ctx)
	if err != nil {
		s.logger.Warn("runtimeconfig: load persisted settings failed (using env seed)", "error", err)
		return
	}
	s.overlay(persisted)
}

// Reload re-reads persisted settings and overlays them. Workers call this on
// every tick so UI changes made in the API process take effect.
func (s *Store) Reload(ctx context.Context) {
	s.Load(ctx)
}

// overlay merges persisted values into the current values. Only keys that
// exist are touched (persisted wins). Locked keys are never overridden.
func (s *Store) overlay(persisted map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range persisted {
		if !s.isLocked(k) {
			s.values[k] = v
		}
	}
}

// isLocked reports whether a key is environment-only.
func (s *Store) isLocked(key string) bool {
	switch key {
	case DataStore, AppEnv, FirebaseDatabaseURL, FirebaseServiceAccountSet:
		return true
	}
	return false
}

// Update merges a patch into the store, normalises known keys, persists the
// full value set, and returns the updated (deep-copied) values.
func (s *Store) Update(ctx context.Context, patch map[string]interface{}) (map[string]interface{}, error) {
	s.mu.Lock()
	normalised, _ := s.normaliseLocked(patch, true)
	s.mu.Unlock()

	s.mu.Lock()
	for k, v := range normalised {
		if !s.isLocked(k) {
			s.values[k] = v
		}
	}
	if err := s.repo.SaveSettings(ctx, s.clonedValuesLocked()); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	values := s.clonedValuesLocked()
	s.mu.Unlock()
	return values, nil
}

// All returns a deep copy of the current values (including locked, read-only
// keys) suitable for a JSON response.
func (s *Store) All() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clonedValuesLocked()
}

// ---------------------------------------------------------------------------
// Typed accessors (handle string/float64/int/json.Number persisted values)
// ---------------------------------------------------------------------------

// Str returns a string value or the fallback.
func (s *Store) Str(key, fallback string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.values[key]; ok && v != nil {
		switch t := v.(type) {
		case string:
			return t
		case bool:
			return strconv.FormatBool(t)
		case float64:
			return strconv.FormatInt(int64(t), 10)
		case int:
			return strconv.Itoa(t)
		case json.Number:
			return t.String()
		}
	}
	return fallback
}

// Int returns an int value or the fallback.
func (s *Store) Int(key string, fallback int) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.values[key]; ok && v != nil {
		switch t := v.(type) {
		case int:
			return t
		case int64:
			return int(t)
		case float64:
			return int(t)
		case json.Number:
			if i, err := t.Int64(); err == nil {
				return int(i)
			}
		case string:
			if i, err := strconv.Atoi(t); err == nil {
				return i
			}
		}
	}
	return fallback
}

// Float returns a float64 value or the fallback.
func (s *Store) Float(key string, fallback float64) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.values[key]; ok && v != nil {
		switch t := v.(type) {
		case float64:
			return t
		case int:
			return float64(t)
		case int64:
			return float64(t)
		case json.Number:
			if f, err := t.Float64(); err == nil {
				return f
			}
		case string:
			if f, err := strconv.ParseFloat(t, 64); err == nil {
				return f
			}
		}
	}
	return fallback
}

// Bool returns a bool value or the fallback.
func (s *Store) Bool(key string, fallback bool) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.values[key]; ok && v != nil {
		switch t := v.(type) {
		case bool:
			return t
		case string:
			switch strings.ToLower(strings.TrimSpace(t)) {
			case "1", "true", "yes", "on":
				return true
			case "0", "false", "no", "off":
				return false
			}
		case float64:
			return t != 0
		case int:
			return t != 0
		}
	}
	return fallback
}

// GeminiConfigured reports whether an API key is present.
func (s *Store) GeminiConfigured() bool {
	return s.Str(GeminiAPIKey, "") != ""
}

// ThreadsConfigured reports whether client ID and user ID are present.
func (s *Store) ThreadsConfigured() bool {
	return s.Str(ThreadsClientID, "") != "" && s.Str(ThreadsUserID, "") != ""
}

// ThreadsConnected reports whether an access token is present.
func (s *Store) ThreadsConnected() bool {
	return s.Str(ThreadsAccessToken, "") != ""
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// clonedValuesLocked returns a deep copy of s.values. Caller must hold a lock.
func (s *Store) clonedValuesLocked() map[string]interface{} {
	out := make(map[string]interface{}, len(s.values))
	for k, v := range s.values {
		out[k] = v
	}
	return out
}

// normaliseLocked coerces known keys to canonical types and clamps ranges.
func (s *Store) normaliseLocked(patch map[string]interface{}, ignoreUnknown bool) (map[string]interface{}, error) {
	for k, v := range patch {
		if ignoreUnknown {
			if _, ok := s.values[k]; !ok {
				continue
			}
		}
		if s.isLocked(k) {
			continue
		}
		switch {
		case intKeys[k]:
			i := s.toInt(v, 0)
			if i < 0 {
				i = 0
			}
			patch[k] = i
		case floatKeys[k]:
			f := s.toFloat(v, 0)
			if f < 0 {
				f = 0
			}
			if f > 1 {
				f = 1
			}
			patch[k] = f
		case k == AutoApproval || k == AutoPublish:
			patch[k] = s.toBool(v)
		default:
			sv, ok := v.(string)
			if !ok {
				sv = s.toStr(v)
			}
			patch[k] = sv
		}
	}
	return patch, nil
}

func (s *Store) toInt(v interface{}, def int) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return int(i)
		}
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
			return i
		}
	case bool:
		if t {
			return 1
		}
	}
	return def
}

func (s *Store) toFloat(v interface{}, def float64) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		if f, err := t.Float64(); err == nil {
			return f
		}
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil {
			return f
		}
	}
	return def
}

func (s *Store) toBool(v interface{}) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	case float64:
		return t != 0
	case int:
		return t != 0
	}
	return false
}

func (s *Store) toStr(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case json.Number:
		return t.String()
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(b)
	}
}
