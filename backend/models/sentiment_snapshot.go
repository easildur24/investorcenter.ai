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
	SnapshotTime time.Time `json:"snapshot_time" db:"snapshot_time"`
	TimeRange    string    `json:"time_range" db:"time_range"` // "1d", "7d", "14d", "30d"

	// Mention metrics
	MentionCount  int `json:"mention_count" db:"mention_count"`
	TotalUpvotes  int `json:"total_upvotes" db:"total_upvotes"`
	TotalComments int `json:"total_comments" db:"total_comments"`
	UniquePosts   int `json:"unique_posts" db:"unique_posts"`

	// Sentiment breakdown
	BullishCount int `json:"bullish_count" db:"bullish_count"`
	NeutralCount int `json:"neutral_count" db:"neutral_count"`
	BearishCount int `json:"bearish_count" db:"bearish_count"`
	// Pct fields are fractions 0.0–1.0 (NOT percentages 0–100).
	// Multiply by 100 when presenting to the frontend.
	BullishPct     float64 `json:"bullish_pct" db:"bullish_pct"`
	NeutralPct     float64 `json:"neutral_pct" db:"neutral_pct"`
	BearishPct     float64 `json:"bearish_pct" db:"bearish_pct"`
	SentimentScore float64 `json:"sentiment_score" db:"sentiment_score"` // -1.0 to +1.0
	SentimentLabel string  `json:"sentiment_label" db:"sentiment_label"` // "bullish"/"bearish"/"neutral"

	// Velocity metrics
	MentionVelocity1h *float64 `json:"mention_velocity_1h,omitempty" db:"mention_velocity_1h"`
	// SentimentVelocity24h is the delta between the current snapshot's sentiment_score
	// and the previous snapshot's value. Despite the "24h" name, it measures change
	// since the last snapshot (which runs hourly), not a fixed 24h window.
	SentimentVelocity24h *float64 `json:"sentiment_velocity_24h,omitempty" db:"sentiment_velocity_24h"`

	// Composite score (non-circular ranking formula)
	CompositeScore float64 `json:"composite_score" db:"composite_score"`

	// Subreddit distribution: {"wallstreetbets": 45, "stocks": 30, ...}
	SubredditDistribution json.RawMessage `json:"subreddit_distribution,omitempty" db:"subreddit_distribution"`

	// Pre-computed ranking
	Rank         *int `json:"rank,omitempty" db:"rank"`
	PreviousRank *int `json:"previous_rank,omitempty" db:"previous_rank"`
	RankChange   *int `json:"rank_change,omitempty" db:"rank_change"` // always integer

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// SentimentTimeSeriesPoint represents a single time-series data point
// from the ticker_sentiment_history hypertable.
type SentimentTimeSeriesPoint struct {
	Time           time.Time `json:"time" db:"time"`
	Ticker         string    `json:"ticker" db:"ticker"`
	SentimentScore float64   `json:"sentiment_score" db:"sentiment_score"`
	// Pct fields are fractions 0.0–1.0 (NOT percentages 0–100).
	BullishPct     float64  `json:"bullish_pct" db:"bullish_pct"`
	BearishPct     *float64 `json:"bearish_pct,omitempty" db:"bearish_pct"` // NULL for pre-041 rows
	NeutralPct     *float64 `json:"neutral_pct,omitempty" db:"neutral_pct"` // NULL for pre-041 rows
	MentionCount   int      `json:"mention_count" db:"mention_count"`
	CompositeScore float64  `json:"composite_score" db:"composite_score"`
}

// ValidTimeRanges for snapshot queries
var ValidTimeRanges = map[string]bool{
	"1d":  true,
	"7d":  true,
	"14d": true,
	"30d": true,
}
