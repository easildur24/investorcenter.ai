package models

import (
	"time"
)

// SocialPostSortOption represents available sort options for posts
type SocialPostSortOption string

const (
	SortByRecent     SocialPostSortOption = "recent"
	SortByEngagement SocialPostSortOption = "engagement"
	SortByBullish    SocialPostSortOption = "bullish"
	SortByBearish    SocialPostSortOption = "bearish"
)

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

// SentimentResponse for GET /api/sentiment/:ticker
type SentimentResponse struct {
	Ticker        string             `json:"ticker"`
	CompanyName   string             `json:"company_name,omitempty"` // Company name from stocks table
	Score         float64            `json:"score"`                  // -1 to +1
	Label         string             `json:"label"`                  // "bullish", "bearish", "neutral"
	Breakdown     SentimentBreakdown `json:"breakdown"`              // Percentage breakdown
	PostCount24h  int                `json:"post_count_24h"`         // Posts in last 24 hours
	PostCount7d   int                `json:"post_count_7d"`          // Posts in last 7 days
	Rank          int                `json:"rank"`                   // Current rank by activity
	RankChange    int                `json:"rank_change"`            // Change from previous period (+/- or 0)
	TopSubreddits []SubredditCount   `json:"top_subreddits"`         // Most active subreddits
	LastUpdated   time.Time          `json:"last_updated"`
}

// SubredditCount represents post count per subreddit
type SubredditCount struct {
	Subreddit string `json:"subreddit"`
	Count     int    `json:"count"`
}

// SentimentHistoryPoint represents a single data point in sentiment history
type SentimentHistoryPoint struct {
	Date      string  `json:"date"`  // YYYY-MM-DD
	Score     float64 `json:"score"` // -1 to +1
	PostCount int     `json:"post_count"`
	Bullish   int     `json:"bullish"`
	Bearish   int     `json:"bearish"`
	Neutral   int     `json:"neutral"`
}

// SentimentHistoryResponse for GET /api/sentiment/:ticker/history
type SentimentHistoryResponse struct {
	Ticker  string                  `json:"ticker"`
	Period  string                  `json:"period"` // "7d", "30d", "90d"
	History []SentimentHistoryPoint `json:"history"`
}

// TrendingTicker represents a ticker in the trending list
type TrendingTicker struct {
	Ticker       string  `json:"ticker"`
	CompanyName  string  `json:"company_name,omitempty"` // Company name from stocks table
	Score        float64 `json:"score"`                  // -1 to +1
	Label        string  `json:"label"`                  // "bullish", "bearish", "neutral"
	PostCount    int     `json:"post_count"`             // Posts in period
	MentionDelta float64 `json:"mention_delta"`          // % change from previous period
	Rank         int     `json:"rank"`
}

// TrendingResponse for GET /api/sentiment/trending
type TrendingResponse struct {
	Period    string           `json:"period"` // "24h", "7d"
	Tickers   []TrendingTicker `json:"tickers"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// RepresentativePost is a curated post for display
type RepresentativePost struct {
	ID                  int64    `json:"id"`
	Title               string   `json:"title"`
	BodyPreview         *string  `json:"body_preview,omitempty"`
	URL                 string   `json:"url"`
	Source              string   `json:"source"`
	Subreddit           string   `json:"subreddit"`
	Upvotes             int      `json:"upvotes"`
	CommentCount        int      `json:"comment_count"`
	AwardCount          int      `json:"award_count"`
	Sentiment           string   `json:"sentiment"`
	SentimentConfidence *float64 `json:"sentiment_confidence,omitempty"`
	Flair               *string  `json:"flair,omitempty"`
	PostedAt            string   `json:"posted_at"` // ISO 8601 string for frontend
}

// RepresentativePostsResponse for GET /api/sentiment/:ticker/posts
type RepresentativePostsResponse struct {
	Ticker string               `json:"ticker"`
	Posts  []RepresentativePost `json:"posts"`
	Total  int                  `json:"total"`
	Sort   string               `json:"sort"` // Sort option used: recent, engagement, bullish, bearish
}

// GetSentimentLabel converts a sentiment score to a human-readable label
func GetSentimentLabel(score float64) string {
	if score >= 0.2 {
		return "bullish"
	} else if score <= -0.2 {
		return "bearish"
	}
	return "neutral"
}
