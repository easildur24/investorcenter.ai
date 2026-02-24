package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"investorcenter-api/models"
	"time"
)

// GetLatestSnapshots returns the most recent snapshot for each ticker in a given
// time range, ordered by rank (or composite_score). This is the primary read path
// for the trending page.
func GetLatestSnapshots(timeRange string, limit int) ([]models.SentimentSnapshot, error) {
	if !models.ValidTimeRanges[timeRange] {
		timeRange = "7d"
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	query := `
		WITH latest AS (
			SELECT DISTINCT ON (ticker)
				id, ticker, snapshot_time, time_range,
				mention_count, total_upvotes, total_comments, unique_posts,
				bullish_count, neutral_count, bearish_count,
				bullish_pct, neutral_pct, bearish_pct,
				sentiment_score, sentiment_label,
				mention_velocity_1h, sentiment_velocity_24h,
				composite_score, subreddit_distribution,
				rank, previous_rank, rank_change, created_at
			FROM ticker_sentiment_snapshots
			WHERE time_range = $1
			ORDER BY ticker, snapshot_time DESC
		)
		SELECT * FROM latest
		ORDER BY COALESCE(rank, 999999) ASC, composite_score DESC
		LIMIT $2
	`

	rows, err := DB.Query(query, timeRange, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest snapshots: %w", err)
	}
	defer rows.Close()

	return scanSnapshots(rows)
}

// GetTickerSnapshot returns the latest snapshot for a specific ticker and time range.
func GetTickerSnapshot(ticker string, timeRange string) (*models.SentimentSnapshot, error) {
	if !models.ValidTimeRanges[timeRange] {
		timeRange = "7d"
	}

	query := `
		SELECT
			id, ticker, snapshot_time, time_range,
			mention_count, total_upvotes, total_comments, unique_posts,
			bullish_count, neutral_count, bearish_count,
			bullish_pct, neutral_pct, bearish_pct,
			sentiment_score, sentiment_label,
			mention_velocity_1h, sentiment_velocity_24h,
			composite_score, subreddit_distribution,
			rank, previous_rank, rank_change, created_at
		FROM ticker_sentiment_snapshots
		WHERE ticker = $1 AND time_range = $2
		ORDER BY snapshot_time DESC
		LIMIT 1
	`

	var s models.SentimentSnapshot
	var mentionVelocity, sentimentVelocity sql.NullFloat64
	var subredditDist []byte
	var rank, prevRank, rankChg sql.NullInt32

	err := DB.QueryRow(query, ticker, timeRange).Scan(
		&s.ID, &s.Ticker, &s.SnapshotTime, &s.TimeRange,
		&s.MentionCount, &s.TotalUpvotes, &s.TotalComments, &s.UniquePosts,
		&s.BullishCount, &s.NeutralCount, &s.BearishCount,
		&s.BullishPct, &s.NeutralPct, &s.BearishPct,
		&s.SentimentScore, &s.SentimentLabel,
		&mentionVelocity, &sentimentVelocity,
		&s.CompositeScore, &subredditDist,
		&rank, &prevRank, &rankChg, &s.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query ticker snapshot: %w", err)
	}

	applyNullableSnapshotFields(&s, mentionVelocity, sentimentVelocity, subredditDist, rank, prevRank, rankChg)
	return &s, nil
}

// UpsertSnapshot inserts or updates a sentiment snapshot row.
// Used by the pipeline to write computed snapshots.
func UpsertSnapshot(s *models.SentimentSnapshot) error {
	query := `
		INSERT INTO ticker_sentiment_snapshots (
			ticker, snapshot_time, time_range,
			mention_count, total_upvotes, total_comments, unique_posts,
			bullish_count, neutral_count, bearish_count,
			bullish_pct, neutral_pct, bearish_pct,
			sentiment_score, sentiment_label,
			mention_velocity_1h, sentiment_velocity_24h,
			composite_score, subreddit_distribution,
			rank, previous_rank, rank_change
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19,
			$20, $21, $22
		)
		ON CONFLICT (ticker, snapshot_time, time_range) DO UPDATE SET
			mention_count = EXCLUDED.mention_count,
			total_upvotes = EXCLUDED.total_upvotes,
			total_comments = EXCLUDED.total_comments,
			unique_posts = EXCLUDED.unique_posts,
			bullish_count = EXCLUDED.bullish_count,
			neutral_count = EXCLUDED.neutral_count,
			bearish_count = EXCLUDED.bearish_count,
			bullish_pct = EXCLUDED.bullish_pct,
			neutral_pct = EXCLUDED.neutral_pct,
			bearish_pct = EXCLUDED.bearish_pct,
			sentiment_score = EXCLUDED.sentiment_score,
			sentiment_label = EXCLUDED.sentiment_label,
			mention_velocity_1h = EXCLUDED.mention_velocity_1h,
			sentiment_velocity_24h = EXCLUDED.sentiment_velocity_24h,
			composite_score = EXCLUDED.composite_score,
			subreddit_distribution = EXCLUDED.subreddit_distribution,
			rank = EXCLUDED.rank,
			previous_rank = EXCLUDED.previous_rank,
			rank_change = EXCLUDED.rank_change
		RETURNING id, created_at
	`

	return DB.QueryRow(query,
		s.Ticker, s.SnapshotTime, s.TimeRange,
		s.MentionCount, s.TotalUpvotes, s.TotalComments, s.UniquePosts,
		s.BullishCount, s.NeutralCount, s.BearishCount,
		s.BullishPct, s.NeutralPct, s.BearishPct,
		s.SentimentScore, s.SentimentLabel,
		s.MentionVelocity1h, s.SentimentVelocity24h,
		s.CompositeScore, s.SubredditDistribution,
		s.Rank, s.PreviousRank, s.RankChange,
	).Scan(&s.ID, &s.CreatedAt)
}

// InsertSentimentHistory inserts a time-series data point into the hypertable.
// Used by the pipeline after computing a snapshot.
func InsertSentimentHistory(point *models.SentimentTimeSeriesPoint) error {
	query := `
		INSERT INTO ticker_sentiment_history (time, ticker, sentiment_score, bullish_pct, mention_count, composite_score)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := DB.Exec(query,
		point.Time, point.Ticker, point.SentimentScore,
		point.BullishPct, point.MentionCount, point.CompositeScore,
	)
	if err != nil {
		return fmt.Errorf("failed to insert sentiment history: %w", err)
	}
	return nil
}

// GetSentimentTimeSeries returns historical sentiment data for charting.
// Uses the ticker_sentiment_history TimescaleDB hypertable for efficient
// time-range queries with automatic partitioning.
func GetSentimentTimeSeries(ticker string, days int) ([]models.SentimentTimeSeriesPoint, error) {
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}

	query := `
		SELECT time, ticker, sentiment_score, bullish_pct, mention_count, composite_score
		FROM ticker_sentiment_history
		WHERE ticker = $1
		  AND time > NOW() - $2::INTEGER * INTERVAL '1 day'
		ORDER BY time ASC
	`

	rows, err := DB.Query(query, ticker, days)
	if err != nil {
		return nil, fmt.Errorf("failed to query sentiment time series: %w", err)
	}
	defer rows.Close()

	var points []models.SentimentTimeSeriesPoint
	for rows.Next() {
		var p models.SentimentTimeSeriesPoint
		err := rows.Scan(&p.Time, &p.Ticker, &p.SentimentScore, &p.BullishPct, &p.MentionCount, &p.CompositeScore)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sentiment history point: %w", err)
		}
		points = append(points, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sentiment history rows: %w", err)
	}

	return points, nil
}

// PruneOldSnapshots removes snapshots older than the retention period.
// Called periodically to keep the table from growing unbounded.
func PruneOldSnapshots(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 30
	}

	result, err := DB.Exec(
		"DELETE FROM ticker_sentiment_snapshots WHERE snapshot_time < NOW() - $1 * INTERVAL '1 day'",
		retentionDays,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to prune old snapshots: %w", err)
	}

	return result.RowsAffected()
}

// --- internal helpers ---

// scanSnapshots scans multiple snapshot rows from a query result.
func scanSnapshots(rows *sql.Rows) ([]models.SentimentSnapshot, error) {
	var snapshots []models.SentimentSnapshot
	for rows.Next() {
		var s models.SentimentSnapshot
		var mentionVelocity, sentimentVelocity sql.NullFloat64
		var subredditDist []byte
		var rank, prevRank, rankChg sql.NullInt32

		err := rows.Scan(
			&s.ID, &s.Ticker, &s.SnapshotTime, &s.TimeRange,
			&s.MentionCount, &s.TotalUpvotes, &s.TotalComments, &s.UniquePosts,
			&s.BullishCount, &s.NeutralCount, &s.BearishCount,
			&s.BullishPct, &s.NeutralPct, &s.BearishPct,
			&s.SentimentScore, &s.SentimentLabel,
			&mentionVelocity, &sentimentVelocity,
			&s.CompositeScore, &subredditDist,
			&rank, &prevRank, &rankChg, &s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan snapshot row: %w", err)
		}

		applyNullableSnapshotFields(&s, mentionVelocity, sentimentVelocity, subredditDist, rank, prevRank, rankChg)
		snapshots = append(snapshots, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating snapshot rows: %w", err)
	}

	return snapshots, nil
}

// applyNullableSnapshotFields converts sql.Null* types to Go pointer fields.
func applyNullableSnapshotFields(
	s *models.SentimentSnapshot,
	mentionVelocity, sentimentVelocity sql.NullFloat64,
	subredditDist []byte,
	rank, prevRank, rankChg sql.NullInt32,
) {
	if mentionVelocity.Valid {
		v := mentionVelocity.Float64
		s.MentionVelocity1h = &v
	}
	if sentimentVelocity.Valid {
		v := sentimentVelocity.Float64
		s.SentimentVelocity24h = &v
	}
	if len(subredditDist) > 0 {
		s.SubredditDistribution = json.RawMessage(subredditDist)
	}
	if rank.Valid {
		v := int(rank.Int32)
		s.Rank = &v
	}
	if prevRank.Valid {
		v := int(prevRank.Int32)
		s.PreviousRank = &v
	}
	if rankChg.Valid {
		v := int(rankChg.Int32)
		s.RankChange = &v
	}
}

// GetLatestSnapshotTime returns the most recent snapshot_time for a time range.
// Useful for pipeline health checks.
func GetLatestSnapshotTime(timeRange string) (time.Time, error) {
	query := `SELECT MAX(snapshot_time) FROM ticker_sentiment_snapshots WHERE time_range = $1`

	var latestTime sql.NullTime
	err := DB.QueryRow(query, timeRange).Scan(&latestTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get latest snapshot time: %w", err)
	}
	if !latestTime.Valid {
		return time.Time{}, sql.ErrNoRows
	}

	return latestTime.Time, nil
}
