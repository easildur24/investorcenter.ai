package models

import (
	"time"
)

// SocialPost represents a social media post with sentiment data
type SocialPost struct {
	ID                  int64     `json:"id" db:"id"`
	ExternalPostID      string    `json:"external_post_id" db:"external_post_id"`
	Source              string    `json:"source" db:"source"`
	Ticker              string    `json:"ticker" db:"ticker"`
	Subreddit           string    `json:"subreddit" db:"subreddit"`
	Title               string    `json:"title" db:"title"`
	BodyPreview         *string   `json:"body_preview,omitempty" db:"body_preview"`
	URL                 string    `json:"url" db:"url"`
	Upvotes             int       `json:"upvotes" db:"upvotes"`
	CommentCount        int       `json:"comment_count" db:"comment_count"`
	AwardCount          int       `json:"award_count" db:"award_count"`
	Sentiment           *string   `json:"sentiment,omitempty" db:"sentiment"`
	SentimentConfidence *float64  `json:"sentiment_confidence,omitempty" db:"sentiment_confidence"`
	Flair               *string   `json:"flair,omitempty" db:"flair"`
	PostedAt            time.Time `json:"posted_at" db:"posted_at"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// SocialPostSortOption represents available sort options for posts
type SocialPostSortOption string

const (
	SortByRecent     SocialPostSortOption = "recent"
	SortByEngagement SocialPostSortOption = "engagement"
	SortByBullish    SocialPostSortOption = "bullish"
	SortByBearish    SocialPostSortOption = "bearish"
)

// SocialPostsRequest for fetching posts
type SocialPostsRequest struct {
	Ticker    string               `json:"ticker" binding:"required"`
	Sort      SocialPostSortOption `json:"sort" binding:"omitempty,oneof=recent engagement bullish bearish"`
	Limit     int                  `json:"limit" binding:"omitempty,min=1,max=50"`
	Subreddit string               `json:"subreddit,omitempty"`
}

// SocialPostsResponse wraps the posts response
type SocialPostsResponse struct {
	Ticker     string       `json:"ticker"`
	Posts      []SocialPost `json:"posts"`
	TotalCount int          `json:"total_count"`
	Sort       string       `json:"sort"`
}

// SentimentLexiconTerm represents a term in the sentiment lexicon
type SentimentLexiconTerm struct {
	ID        int       `json:"id" db:"id"`
	Term      string    `json:"term" db:"term"`
	Sentiment string    `json:"sentiment" db:"sentiment"`
	Weight    float64   `json:"weight" db:"weight"`
	Category  *string   `json:"category,omitempty" db:"category"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// SocialDataSource represents a configurable social media data source
type SocialDataSource struct {
	ID         int                    `json:"id" db:"id"`
	SourceName string                 `json:"source_name" db:"source_name"`
	IsEnabled  bool                   `json:"is_enabled" db:"is_enabled"`
	Config     map[string]interface{} `json:"config" db:"config"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at" db:"updated_at"`
}

// TickerSentimentSummary represents aggregated sentiment for a ticker
type TickerSentimentSummary struct {
	Ticker          string   `json:"ticker"`
	SentimentScore  *float64 `json:"sentiment_score,omitempty"`
	BullishCount    int      `json:"bullish_count"`
	BearishCount    int      `json:"bearish_count"`
	NeutralCount    int      `json:"neutral_count"`
	TotalPosts      int      `json:"total_posts"`
	TopSubreddits   []string `json:"top_subreddits,omitempty"`
	TrendDirection  string   `json:"trend_direction"`
	PopularityScore float64  `json:"popularity_score"`
}

// SentimentBreakdown for display in the UI
type SentimentBreakdown struct {
	Bullish float64 `json:"bullish"` // Percentage 0-100
	Bearish float64 `json:"bearish"` // Percentage 0-100
	Neutral float64 `json:"neutral"` // Percentage 0-100
}
