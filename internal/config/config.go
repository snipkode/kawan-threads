package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration.
type Config struct {
	App       AppConfig
	Store     StoreConfig
	Firebase  FirebaseConfig
	Gemini    GeminiConfig
	Threads   ThreadsConfig
	Scheduler SchedulerConfig
	Settings  SettingsConfig
	Frontend  FrontendConfig
}

// AppConfig holds general application settings.
type AppConfig struct {
	Env  string
	Port string
}

// StoreConfig selects the active datastore: "file" uses the local JSON file
// datastore (development/demo, requires no credentials), "firebase" (default)
// uses Firebase Realtime Database.
type StoreConfig struct {
	DataStore string
	DataFile  string
}

// FirebaseConfig holds Firebase configuration.
type FirebaseConfig struct {
	DatabaseURL          string
	ServiceAccountBase64 string
}

// GeminiConfig holds Gemini AI configuration.
type GeminiConfig struct {
	APIKey string
	Model  string
}

// ThreadsConfig holds Threads API configuration.
type ThreadsConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	AccessToken  string
	UserID       string
}

// SchedulerConfig holds scheduler settings.
type SchedulerConfig struct {
	Timezone        string
	IntervalMinutes int
}

// SettingsConfig holds operational settings with defaults.
type SettingsConfig struct {
	AutoApproval           bool
	AutoPublish            bool
	MaxPostsPerDay         int
	MinPostIntervalMinutes int
	ExplorationRate        float64
	MaxRetry               int
}

// FrontendConfig holds non-secret frontend config.
type FrontendConfig struct {
	ViteAPIBaseURL string
}

// LoadConfig loads configuration from a .env file (if present) and then
// from environment variables (env vars take precedence over .env).
func LoadConfig() (*Config, error) {
	// Load .env file into environment if it exists (won't override existing env vars)
	if err := loadDotEnv(".env"); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading .env: %w", err)
	}

	cfg := &Config{}

	// --- App ---
	cfg.App.Env = getEnvOrDefault("APP_ENV", "development")
	cfg.App.Port = getEnvOrDefault("PORT", "8080")

	// --- Store ---
	cfg.Store.DataStore = getEnvOrDefault("DATA_STORE", "firebase")
	cfg.Store.DataFile = getEnvOrDefault("DATA_FILE", "data/db.json")

	// --- Firebase ---
	cfg.Firebase.DatabaseURL = os.Getenv("FIREBASE_DATABASE_URL")
	cfg.Firebase.ServiceAccountBase64 = os.Getenv("FIREBASE_SERVICE_ACCOUNT_BASE64")

	// --- Gemini ---
	cfg.Gemini.APIKey = os.Getenv("GEMINI_API_KEY")
	cfg.Gemini.Model = getEnvOrDefault("GEMINI_MODEL", "gemini-1.5-flash")

	// --- Threads ---
	cfg.Threads.ClientID = os.Getenv("THREADS_CLIENT_ID")
	cfg.Threads.ClientSecret = os.Getenv("THREADS_CLIENT_SECRET")
	cfg.Threads.RedirectURI = os.Getenv("THREADS_REDIRECT_URI")
	cfg.Threads.AccessToken = os.Getenv("THREADS_ACCESS_TOKEN")
	cfg.Threads.UserID = os.Getenv("THREADS_USER_ID")

	// --- Scheduler ---
	cfg.Scheduler.Timezone = getEnvOrDefault("SCHEDULER_TIMEZONE", "Asia/Jakarta")
	cfg.Scheduler.IntervalMinutes = parseInt(os.Getenv("SCHEDULER_INTERVAL_MINUTES"), 5)

	// --- Settings ---
	cfg.Settings.AutoApproval = parseBool(os.Getenv("AUTO_APPROVAL"), false)
	cfg.Settings.AutoPublish = parseBool(os.Getenv("AUTO_PUBLISH"), false)
	cfg.Settings.MaxPostsPerDay = parseInt(os.Getenv("MAX_POSTS_PER_DAY"), 5)
	cfg.Settings.MinPostIntervalMinutes = parseInt(os.Getenv("MIN_POST_INTERVAL_MINUTES"), 90)
	cfg.Settings.ExplorationRate = parseFloat(os.Getenv("EXPLORATION_RATE"), 0.20)
	cfg.Settings.MaxRetry = parseInt(os.Getenv("MAX_RETRY"), 3)

	// --- Frontend ---
	cfg.Frontend.ViteAPIBaseURL = os.Getenv("VITE_API_BASE_URL")

	return cfg, nil
}

// loadDotEnv reads a .env file and sets environment variables that are not
// already set. This preserves existing environment variable values.
func loadDotEnv(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip blank lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on first '='
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Strip optional surrounding quotes
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		// Only set if not already set in the environment
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("setting env %s: %w", key, err)
			}
		}
	}

	return scanner.Err()
}

// getEnvOrDefault returns the environment variable value or a default.
func getEnvOrDefault(key, defaultVal string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return defaultVal
}

// parseBool parses a boolean string, returning defaultVal on failure.
func parseBool(s string, defaultVal bool) bool {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return defaultVal
	}
	return v
}

// parseInt parses an integer string, returning defaultVal on failure.
func parseInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

// parseFloat parses a float64 string, returning defaultVal on failure.
func parseFloat(s string, defaultVal float64) float64 {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return defaultVal
	}
	return v
}
