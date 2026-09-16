package port

import "context"

// ThreadsInsights holds the analytics metrics retrieved from the Threads API
// for a single post.
type ThreadsInsights struct {
	Views   int64 `json:"views"`
	Likes   int64 `json:"likes"`
	Replies int64 `json:"replies"`
	Reposts int64 `json:"reposts"`
	Quotes  int64 `json:"quotes"`
}

// ThreadsPort is the outbound port for all interactions with the Threads
// social-media API.
type ThreadsPort interface {
	// CreateTextContainer stages text content on Threads and returns the
	// creation ID that must be supplied to PublishContainer.
	CreateTextContainer(ctx context.Context, userID string, text string) (string, error)

	// PublishContainer publishes a previously staged container and returns
	// the resulting Threads post ID.
	PublishContainer(ctx context.Context, userID string, creationID string) (string, error)

	// GetPostInsights retrieves performance metrics for the given post.
	GetPostInsights(ctx context.Context, userID string, postID string) (*ThreadsInsights, error)

	// GetAuthURL returns the OAuth2 authorisation URL that the user must visit
	// to authorise the application.
	GetAuthURL() string

	// ExchangeCode exchanges an OAuth2 authorisation code for an access token.
	ExchangeCode(ctx context.Context, code string) (string, error)

	// RefreshToken exchanges a long-lived token for a fresh one.
	RefreshToken(ctx context.Context, token string) (string, error)
}
