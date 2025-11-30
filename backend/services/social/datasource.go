package social

import (
	"context"
	"time"
)

// SocialDataSource is the interface all social platforms must implement
// Reddit is the first implementation; StockTwits/X can be added later
type SocialDataSource interface {
	GetName() string
	IsEnabled() bool
	FetchPosts(ctx context.Context, opts FetchOptions) ([]RawPost, error)
	GetRateLimits() RateLimitConfig
}

// FetchOptions configures how posts are fetched from a data source
type FetchOptions struct {
	Subreddit string    // Community/channel name
	Ticker    string    // Filter by ticker (optional)
	Limit     int       // Maximum posts to fetch
	Since     time.Time // Only fetch posts after this time
}

// RawPost is the platform-agnostic post structure
// NO Author field - privacy decision
type RawPost struct {
	ExternalID   string
	Source       string    // "reddit", "stocktwits", etc.
	Community    string    // Subreddit name or equivalent
	Title        string
	Body         string
	URL          string
	Upvotes      int
	CommentCount int
	AwardCount   int
	Flair        string
	PostedAt     time.Time
}

// RateLimitConfig defines rate limiting parameters for a data source
type RateLimitConfig struct {
	RequestsPerMinute int
	BurstLimit        int
}

// ProcessedPost is the post with sentiment analysis applied
type ProcessedPost struct {
	RawPost
	Ticker              string
	BodyPreview         string
	Sentiment           string  // "bullish", "bearish", "neutral"
	SentimentConfidence float64
}
