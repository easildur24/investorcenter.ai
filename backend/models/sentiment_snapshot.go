package models

import (
	"encoding/json"
	"time"
)

// SentimentSnapshot represents a pre-computed sentiment aggregation
// for a ticker in a specific time range. One row per (ticker, time_range)
// refreshed every ~15 minutes by the snapshot pipeline.
type SentimentSnapshot struct {
	ID           int64     `json:"id" db:"id"`
	Ticker       string    `json:"ticker" db:"ticker"`
	SnapshotTime time.Time `json:"snapshotTime" db:"snapshot_time"`
	TimeRange    string    `json:"timeRange" db:"time_range"` // "1d", "7d", "14d", "30d"

	// Mention metrics
	MentionCount  int `json:"mentionCount" db:"mention_count"`
	TotalUpvotes  int `json:"totalUpvotes" db:"total_upvotes"`
	TotalComments int `json:"totalComments" db:"total_comments"`
	UniquePosts   int `json:"uniquePosts" db:"unique_posts"`

	// Sentiment breakdown
	BullishCount   int     `json:"bullishCount" db:"bullish_count"`
	NeutralCount   int     `json:"neutralCount" db:"neutral_count"`
	BearishCount   int     `json:"bearishCount" db:"bearish_count"`
	BullishPct     float64 `json:"bullishPct" db:"bullish_pct"`
	NeutralPct     float64 `json:"neutralPct" db:"neutral_pct"`
	BearishPct     float64 `json:"bearishPct" db:"bearish_pct"`
	SentimentScore float64 `json:"sentimentScore" db:"sentiment_score"` // -1.0 to +1.0
	SentimentLabel string  `json:"sentimentLabel" db:"sentiment_label"` // "bullish"/"bearish"/"neutral"

	// Velocity metrics
	MentionVelocity1h    *float64 `json:"mentionVelocity1h,omitempty" db:"mention_velocity_1h"`
	SentimentVelocity24h *float64 `json:"sentimentVelocity24h,omitempty" db:"sentiment_velocity_24h"`

	// Composite score (non-circular ranking formula)
	CompositeScore float64 `json:"compositeScore" db:"composite_score"`

	// Subreddit distribution: {"wallstreetbets": 45, "stocks": 30, ...}
	SubredditDistribution json.RawMessage `json:"subredditDistribution,omitempty" db:"subreddit_distribution"`

	// Pre-computed ranking
	Rank         *int `json:"rank,omitempty" db:"rank"`
	PreviousRank *int `json:"previousRank,omitempty" db:"previous_rank"`
	RankChange   *int `json:"rankChange,omitempty" db:"rank_change"` // always integer

	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

// SentimentHistoryPoint represents a single time-series data point
// from the ticker_sentiment_history hypertable.
type SentimentTimeSeriesPoint struct {
	Time           time.Time `json:"time" db:"time"`
	Ticker         string    `json:"ticker" db:"ticker"`
	SentimentScore float64   `json:"sentimentScore" db:"sentiment_score"`
	BullishPct     float64   `json:"bullishPct" db:"bullish_pct"`
	MentionCount   int       `json:"mentionCount" db:"mention_count"`
	CompositeScore float64   `json:"compositeScore" db:"composite_score"`
}

// ValidTimeRanges for snapshot queries
var ValidTimeRanges = map[string]bool{
	"1d":  true,
	"7d":  true,
	"14d": true,
	"30d": true,
}
