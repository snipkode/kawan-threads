// Package threads provides an implementation of port.ThreadsPort that
// communicates with the official Threads Graph API.
package threads

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"kawan-threads/internal/application/runtimeconfig"
	"kawan-threads/internal/domain/port"
)

const (
	apiBase  = "https://graph.threads.net/v1.0"
	authBase = "https://www.threads.net/oauth/authorize"
	tokenURL = "https://graph.threads.net/oauth/access_token"
)

// ---------------------------------------------------------------------------
// Error types
// ---------------------------------------------------------------------------

// RateLimitError is returned when the Threads API responds with HTTP 429.
// RetryAfter holds the number of seconds the caller should wait before
// retrying (parsed from the Retry-After header; zero when absent).
type RateLimitError struct {
	RetryAfter int
	Message    string
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("threads rate limit exceeded: retry after %d seconds", e.RetryAfter)
	}
	return fmt.Sprintf("threads rate limit exceeded: %s", e.Message)
}

// ---------------------------------------------------------------------------
// ThreadsAdapter
// ---------------------------------------------------------------------------

// ThreadsAdapter implements port.ThreadsPort using the official Threads API.
// All credentials are read from the runtime settings store so they can be
// configured through the UI — including the access token, which the OAuth
// callback persists after a successful exchange.
type ThreadsAdapter struct {
	settings   *runtimeconfig.Store
	httpClient *http.Client
	logger     *slog.Logger
}

// NewThreadsAdapter constructs a ThreadsAdapter backed by the runtime settings.
func NewThreadsAdapter(settings *runtimeconfig.Store, logger *slog.Logger) *ThreadsAdapter {
	return &ThreadsAdapter{
		settings: settings,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// UserID returns the configured Threads user ID.
func (a *ThreadsAdapter) UserID() string {
	return a.settings.Str(runtimeconfig.ThreadsUserID, "")
}

// Configured reports whether the minimum credentials (client ID + user ID)
// are present to perform Threads API calls.
func (a *ThreadsAdapter) Configured() bool {
	return a.settings.Str(runtimeconfig.ThreadsClientID, "") != "" &&
		a.settings.Str(runtimeconfig.ThreadsUserID, "") != ""
}

// Connected reports whether an access token has been saved.
func (a *ThreadsAdapter) Connected() bool {
	return a.settings.Str(runtimeconfig.ThreadsAccessToken, "") != ""
}

// SaveAccessToken persists a (new or refreshed) access token through the
// runtime settings store.
func (a *ThreadsAdapter) SaveAccessToken(ctx context.Context, token string) error {
	if token == "" {
		return fmt.Errorf("threads: refusing to store an empty access token")
	}
	_, err := a.settings.Update(ctx, map[string]interface{}{runtimeconfig.ThreadsAccessToken: token})
	if err != nil {
		return fmt.Errorf("threads: persisting access token: %w", err)
	}
	return nil
}

func (a *ThreadsAdapter) clientID() string { return a.settings.Str(runtimeconfig.ThreadsClientID, "") }
func (a *ThreadsAdapter) clientSecret() string {
	return a.settings.Str(runtimeconfig.ThreadsClientSecret, "")
}
func (a *ThreadsAdapter) redirectURI() string {
	return a.settings.Str(runtimeconfig.ThreadsRedirectURI, "")
}
func (a *ThreadsAdapter) accessToken() string {
	return a.settings.Str(runtimeconfig.ThreadsAccessToken, "")
}

// ---------------------------------------------------------------------------
// port.ThreadsPort implementation
// ---------------------------------------------------------------------------

// CreateTextContainer stages text content on Threads and returns the
// creation ID that must be passed to PublishContainer.
//
// POST https://graph.threads.net/v1.0/{userID}/threads
func (a *ThreadsAdapter) CreateTextContainer(ctx context.Context, userID string, text string) (string, error) {
	endpoint := fmt.Sprintf("%s/%s/threads", apiBase, userID)

	params := url.Values{}
	params.Set("media_type", "TEXT")
	params.Set("text", text)
	params.Set("access_token", a.accessToken())

	resp, err := a.postForm(ctx, endpoint, params)
	if err != nil {
		return "", fmt.Errorf("CreateTextContainer: %w", err)
	}

	return resp.ID, nil
}

// PublishContainer publishes a previously staged container and returns the
// resulting Threads post ID.
//
// POST https://graph.threads.net/v1.0/{userID}/threads_publish
func (a *ThreadsAdapter) PublishContainer(ctx context.Context, userID string, creationID string) (string, error) {
	endpoint := fmt.Sprintf("%s/%s/threads_publish", apiBase, userID)

	params := url.Values{}
	params.Set("creation_id", creationID)
	params.Set("access_token", a.accessToken())

	resp, err := a.postForm(ctx, endpoint, params)
	if err != nil {
		return "", fmt.Errorf("PublishContainer: %w", err)
	}

	return resp.ID, nil
}

// GetPostInsights retrieves performance metrics for the given post from
// the Threads insights endpoint.
//
// GET https://graph.threads.net/v1.0/{postID}/insights
func (a *ThreadsAdapter) GetPostInsights(ctx context.Context, _ string, postID string) (*port.ThreadsInsights, error) {
	q := url.Values{}
	q.Set("metric", "views,likes,replies,reposts,quotes")
	q.Set("access_token", a.accessToken())

	endpoint := fmt.Sprintf("%s/%s/insights?%s", apiBase, postID, q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("GetPostInsights: build request: %w", err)
	}

	body, err := a.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("GetPostInsights: %w", err)
	}

	// Parse insights response:
	// {"data":[{"name":"views","values":[{"value":123}]}, ...]}
	var insightsResp struct {
		Data []struct {
			Name   string `json:"name"`
			Values []struct {
				Value int64 `json:"value"`
			} `json:"values"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &insightsResp); err != nil {
		return nil, fmt.Errorf("GetPostInsights: unmarshal response: %w", err)
	}

	insights := &port.ThreadsInsights{}
	for _, metric := range insightsResp.Data {
		if len(metric.Values) == 0 {
			continue
		}
		val := metric.Values[0].Value
		switch metric.Name {
		case "views":
			insights.Views = val
		case "likes":
			insights.Likes = val
		case "replies":
			insights.Replies = val
		case "reposts":
			insights.Reposts = val
		case "quotes":
			insights.Quotes = val
		}
	}

	return insights, nil
}

// GetAuthURL returns the OAuth2 authorization URL that the user must visit
// to authorize the application.
func (a *ThreadsAdapter) GetAuthURL() string {
	params := url.Values{}
	params.Set("client_id", a.clientID())
	params.Set("redirect_uri", a.redirectURI())
	params.Set("scope", "threads_basic,threads_content_publish,threads_read_replies,threads_manage_insights")
	params.Set("response_type", "code")
	return fmt.Sprintf("%s?%s", authBase, params.Encode())
}

// ExchangeCode exchanges an OAuth2 authorization code for a long-lived
// access token. It first retrieves a short-lived token and then upgrades it.
func (a *ThreadsAdapter) ExchangeCode(ctx context.Context, code string) (string, error) {
	// Step 1: short-lived token
	shortToken, err := a.exchangeShortLivedToken(ctx, code)
	if err != nil {
		return "", fmt.Errorf("ExchangeCode: short-lived exchange: %w", err)
	}

	// Step 2: long-lived token
	longToken, err := a.exchangeLongLivedToken(ctx, shortToken)
	if err != nil {
		return "", fmt.Errorf("ExchangeCode: long-lived exchange: %w", err)
	}

	return longToken, nil
}

// RefreshToken refreshes an existing long-lived token and returns a new one.
//
// GET https://graph.threads.net/refresh_access_token
func (a *ThreadsAdapter) RefreshToken(ctx context.Context, token string) (string, error) {
	q := url.Values{}
	q.Set("grant_type", "th_refresh_token")
	q.Set("access_token", token)

	endpoint := fmt.Sprintf("https://graph.threads.net/refresh_access_token?%s", q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("RefreshToken: build request: %w", err)
	}

	body, err := a.doRequest(req)
	if err != nil {
		return "", fmt.Errorf("RefreshToken: %w", err)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("RefreshToken: unmarshal response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("RefreshToken: empty access_token in response")
	}

	return tokenResp.AccessToken, nil
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

// apiIDResponse is used to parse {"id": "..."} responses from POST endpoints.
type apiIDResponse struct {
	ID string `json:"id"`
}

// postForm executes an application/x-www-form-urlencoded POST and returns
// the parsed id-response or a typed error.
func (a *ThreadsAdapter) postForm(ctx context.Context, endpoint string, params url.Values) (*apiIDResponse, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		strings.NewReader(params.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	body, err := a.doRequest(req)
	if err != nil {
		return nil, err
	}

	var result apiIDResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if result.ID == "" {
		return nil, fmt.Errorf("empty id in response body")
	}

	return &result, nil
}

// doRequest executes an HTTP request, handles error status codes, and
// returns the raw response body.
func (a *ThreadsAdapter) doRequest(req *http.Request) ([]byte, error) {
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := 0
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if v, parseErr := strconv.Atoi(ra); parseErr == nil {
				retryAfter = v
			}
		}
		return nil, &RateLimitError{
			RetryAfter: retryAfter,
			Message:    string(body),
		}
	}

	if resp.StatusCode >= 400 {
		// Try to parse a Threads-style error envelope.
		var apiErr struct {
			Error struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		if jsonErr := json.Unmarshal(body, &apiErr); jsonErr == nil && apiErr.Error.Message != "" {
			return nil, fmt.Errorf("threads API error %d (%s): %s",
				resp.StatusCode, apiErr.Error.Type, apiErr.Error.Message)
		}
		return nil, fmt.Errorf("threads API HTTP %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// exchangeShortLivedToken POSTs to the token endpoint and returns a
// short-lived access token.
func (a *ThreadsAdapter) exchangeShortLivedToken(ctx context.Context, code string) (string, error) {
	params := url.Values{}
	params.Set("client_id", a.clientID())
	params.Set("client_secret", a.clientSecret())
	params.Set("grant_type", "authorization_code")
	params.Set("redirect_uri", a.redirectURI())
	params.Set("code", code)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		tokenURL,
		strings.NewReader(params.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	body, err := a.doRequest(req)
	if err != nil {
		return "", err
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access_token in short-lived exchange response")
	}

	return tokenResp.AccessToken, nil
}

// exchangeLongLivedToken calls the token-exchange endpoint to upgrade a
// short-lived token to a long-lived one.
func (a *ThreadsAdapter) exchangeLongLivedToken(ctx context.Context, shortToken string) (string, error) {
	q := url.Values{}
	q.Set("grant_type", "th_exchange_token")
	q.Set("client_secret", a.clientSecret())
	q.Set("access_token", shortToken)

	endpoint := fmt.Sprintf("https://graph.threads.net/access_token?%s", q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	body, err := a.doRequest(req)
	if err != nil {
		return "", err
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access_token in long-lived exchange response")
	}

	return tokenResp.AccessToken, nil
}
