// Package firebase provides a lightweight Firebase Realtime Database client
// that communicates via the RTDB REST API, authenticated with a Google service
// account.  It avoids the heavier gRPC stack of the Admin SDK and compiles
// cleanly with the dependencies already declared in go.mod.
package firebase

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"kawan-threads/internal/config"
)

// firebaseScopes are the OAuth2 scopes required for RTDB REST access.
var firebaseScopes = []string{
	"https://www.googleapis.com/auth/firebase.database",
	"https://www.googleapis.com/auth/userinfo.email",
}

// FirebaseClient wraps a token-sourced HTTP client and provides simple CRUD
// helpers against a Firebase Realtime Database via the REST API.
type FirebaseClient struct {
	baseURL    string // e.g. https://my-project.firebaseio.com
	httpClient *http.Client
}

// NewFirebaseClient creates a FirebaseClient from the application config.
// The service account JSON is expected as a base64-encoded string in
// cfg.Firebase.ServiceAccountBase64.
func NewFirebaseClient(cfg *config.Config) (*FirebaseClient, error) {
	if cfg.Firebase.DatabaseURL == "" {
		return nil, fmt.Errorf("firebase: FIREBASE_DATABASE_URL is required")
	}
	if cfg.Firebase.ServiceAccountBase64 == "" {
		return nil, fmt.Errorf("firebase: FIREBASE_SERVICE_ACCOUNT_BASE64 is required")
	}

	// Decode the base64-encoded service account JSON.
	saJSON, err := base64.StdEncoding.DecodeString(cfg.Firebase.ServiceAccountBase64)
	if err != nil {
		// Try URL-safe base64 as a fallback.
		saJSON, err = base64.URLEncoding.DecodeString(cfg.Firebase.ServiceAccountBase64)
		if err != nil {
			return nil, fmt.Errorf("firebase: decoding service account base64: %w", err)
		}
	}

	// Build a token source from the service account credentials.
	creds, err := google.CredentialsFromJSON(context.Background(), saJSON, firebaseScopes...)
	if err != nil {
		return nil, fmt.Errorf("firebase: parsing service account credentials: %w", err)
	}

	httpClient := oauth2.NewClient(context.Background(), creds.TokenSource)
	httpClient.Timeout = 30 * time.Second

	baseURL := strings.TrimRight(cfg.Firebase.DatabaseURL, "/")

	return &FirebaseClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}, nil
}

// url builds the REST URL for the given database path.
func (c *FirebaseClient) url(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return c.baseURL + path + ".json"
}

// Set writes data to path, replacing any existing value (HTTP PUT).
func (c *FirebaseClient) Set(ctx context.Context, path string, data interface{}) error {
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("firebase Set: marshalling data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.url(path), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("firebase Set: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("firebase Set: request failed: %w", err)
	}
	defer resp.Body.Close()

	return checkStatus(resp)
}

// Get retrieves the value at path and unmarshals it into dest.
// dest must be a pointer.  Returns nil (without modifying dest) when the
// node does not exist (HTTP 200 with a JSON null body).
func (c *FirebaseClient) Get(ctx context.Context, path string, dest interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(path), nil)
	if err != nil {
		return fmt.Errorf("firebase Get: building request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("firebase Get: request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := checkStatus(resp); err != nil {
		return err
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("firebase Get: reading body: %w", err)
	}

	// Firebase returns "null" when the node does not exist.
	if string(raw) == "null" {
		return nil
	}

	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("firebase Get: unmarshalling response: %w", err)
	}
	return nil
}

// GetAll retrieves all children at path as a map and unmarshals into dest.
// dest should typically be a pointer to a map or slice.
func (c *FirebaseClient) GetAll(ctx context.Context, path string, dest interface{}) error {
	return c.Get(ctx, path, dest)
}

// Delete removes the node at path (HTTP DELETE).
func (c *FirebaseClient) Delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.url(path), nil)
	if err != nil {
		return fmt.Errorf("firebase Delete: building request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("firebase Delete: request failed: %w", err)
	}
	defer resp.Body.Close()

	return checkStatus(resp)
}

// Push appends data as a new child of path using HTTP POST (Firebase generates
// the key) and returns the generated key.
func (c *FirebaseClient) Push(ctx context.Context, path string, data interface{}) (string, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("firebase Push: marshalling data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(path), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("firebase Push: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("firebase Push: request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := checkStatus(resp); err != nil {
		return "", err
	}

	// Firebase returns {"name": "-Kx..."} for POST.
	var result struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("firebase Push: decoding response: %w", err)
	}
	return result.Name, nil
}

// Update merges data into path using HTTP PATCH.
func (c *FirebaseClient) Update(ctx context.Context, path string, data interface{}) error {
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("firebase Update: marshalling data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.url(path), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("firebase Update: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("firebase Update: request failed: %w", err)
	}
	defer resp.Body.Close()

	return checkStatus(resp)
}

// checkStatus returns an error if the HTTP response indicates failure.
func checkStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("firebase: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}
