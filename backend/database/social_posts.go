package database

import (
	"database/sql"
	"fmt"
	"investorcenter-api/models"
	"time"
)

// UpsertSocialPost inserts or updates a social post
func UpsertSocialPost(post *models.SocialPost) error {
	query := `
		INSERT INTO social_posts (
			external_post_id, source, ticker, subreddit, title, body_preview,
			url, upvotes, comment_count, award_count, sentiment,
			sentiment_confidence, flair, posted_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
		ON CONFLICT (external_post_id) DO UPDATE SET
			upvotes = EXCLUDED.upvotes,
			comment_count = EXCLUDED.comment_count,
			award_count = EXCLUDED.award_count,
			sentiment = EXCLUDED.sentiment,
			sentiment_confidence = EXCLUDED.sentiment_confidence,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`

	err := DB.QueryRow(query,
		post.ExternalPostID,
		post.Source,
		post.Ticker,
		post.Subreddit,
		post.Title,
		post.BodyPreview,
		post.URL,
		post.Upvotes,
		post.CommentCount,
		post.AwardCount,
		post.Sentiment,
		post.SentimentConfidence,
		post.Flair,
		post.PostedAt,
	).Scan(&post.ID, &post.CreatedAt, &post.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to upsert social post: %w", err)
	}
	return nil
}

// GetRepresentativePosts returns posts for a ticker with user-selected sorting
func GetRepresentativePosts(ticker string, sort models.SocialPostSortOption, limit int) ([]models.SocialPost, error) {
	if limit <= 0 {
		limit = 10 // Default limit
	}
	if limit > 50 {
		limit = 50 // Max limit
	}

	// Build ORDER BY based on user's sort choice
	orderBy := "posted_at DESC" // default: recent
	whereClause := ""

	switch sort {
	case models.SortByEngagement:
		orderBy = "(upvotes + comment_count * 2 + award_count * 5) DESC"
	case models.SortByBullish:
		whereClause = "AND sentiment = 'bullish'"
		orderBy = "sentiment_confidence DESC NULLS LAST, upvotes DESC"
	case models.SortByBearish:
		whereClause = "AND sentiment = 'bearish'"
		orderBy = "sentiment_confidence DESC NULLS LAST, upvotes DESC"
	case models.SortByRecent:
		orderBy = "posted_at DESC"
	}

	query := fmt.Sprintf(`
		SELECT id, external_post_id, source, ticker, subreddit, title,
			   body_preview, url, upvotes, comment_count, award_count,
			   sentiment, sentiment_confidence, flair, posted_at, created_at, updated_at
		FROM social_posts
		WHERE ticker = $1
		  AND posted_at > NOW() - INTERVAL '7 days'
		  %s
		ORDER BY %s
		LIMIT $2
	`, whereClause, orderBy)

	rows, err := DB.Query(query, ticker, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query representative posts: %w", err)
	}
	defer rows.Close()

	var posts []models.SocialPost
	for rows.Next() {
		var p models.SocialPost
		var bodyPreview, sentiment, flair sql.NullString
		var sentimentConfidence sql.NullFloat64

		err := rows.Scan(
			&p.ID,
			&p.ExternalPostID,
			&p.Source,
			&p.Ticker,
			&p.Subreddit,
			&p.Title,
			&bodyPreview,
			&p.URL,
			&p.Upvotes,
			&p.CommentCount,
			&p.AwardCount,
			&sentiment,
			&sentimentConfidence,
			&flair,
			&p.PostedAt,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan social post: %w", err)
		}

		// Handle nullable fields
		if bodyPreview.Valid {
			p.BodyPreview = &bodyPreview.String
		}
		if sentiment.Valid {
			p.Sentiment = &sentiment.String
		}
		if sentimentConfidence.Valid {
			p.SentimentConfidence = &sentimentConfidence.Float64
		}
		if flair.Valid {
			p.Flair = &flair.String
		}

		posts = append(posts, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating social posts: %w", err)
	}

	return posts, nil
}

// GetPostsBySubreddit returns posts from a specific subreddit for a ticker
func GetPostsBySubreddit(ticker string, subreddit string, limit int) ([]models.SocialPost, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT id, external_post_id, source, ticker, subreddit, title,
			   body_preview, url, upvotes, comment_count, award_count,
			   sentiment, sentiment_confidence, flair, posted_at, created_at, updated_at
		FROM social_posts
		WHERE ticker = $1
		  AND subreddit = $2
		  AND posted_at > NOW() - INTERVAL '7 days'
		ORDER BY posted_at DESC
		LIMIT $3
	`

	rows, err := DB.Query(query, ticker, subreddit, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query posts by subreddit: %w", err)
	}
	defer rows.Close()

	var posts []models.SocialPost
	for rows.Next() {
		var p models.SocialPost
		var bodyPreview, sentiment, flair sql.NullString
		var sentimentConfidence sql.NullFloat64

		err := rows.Scan(
			&p.ID,
			&p.ExternalPostID,
			&p.Source,
			&p.Ticker,
			&p.Subreddit,
			&p.Title,
			&bodyPreview,
			&p.URL,
			&p.Upvotes,
			&p.CommentCount,
			&p.AwardCount,
			&sentiment,
			&sentimentConfidence,
			&flair,
			&p.PostedAt,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan social post: %w", err)
		}

		if bodyPreview.Valid {
			p.BodyPreview = &bodyPreview.String
		}
		if sentiment.Valid {
			p.Sentiment = &sentiment.String
		}
		if sentimentConfidence.Valid {
			p.SentimentConfidence = &sentimentConfidence.Float64
		}
		if flair.Valid {
			p.Flair = &flair.String
		}

		posts = append(posts, p)
	}

	return posts, nil
}

// GetTickerSentimentSummary returns aggregated sentiment data for a ticker
func GetTickerSentimentSummary(ticker string, days int) (*models.TickerSentimentSummary, error) {
	if days <= 0 {
		days = 7
	}

	query := `
		SELECT
			ticker,
			AVG(CASE WHEN sentiment = 'bullish' THEN 1
					 WHEN sentiment = 'bearish' THEN -1
					 ELSE 0 END) * 100 as sentiment_score,
			COUNT(*) FILTER (WHERE sentiment = 'bullish') as bullish_count,
			COUNT(*) FILTER (WHERE sentiment = 'bearish') as bearish_count,
			COUNT(*) FILTER (WHERE sentiment = 'neutral' OR sentiment IS NULL) as neutral_count,
			COUNT(*) as total_posts,
			ARRAY_AGG(DISTINCT subreddit ORDER BY subreddit) FILTER (WHERE subreddit IS NOT NULL) as top_subreddits
		FROM social_posts
		WHERE ticker = $1
		  AND posted_at > NOW() - $2::INTEGER * INTERVAL '1 day'
		GROUP BY ticker
	`

	var summary models.TickerSentimentSummary
	var sentimentScore sql.NullFloat64
	var subreddits []string

	err := DB.QueryRow(query, ticker, days).Scan(
		&summary.Ticker,
		&sentimentScore,
		&summary.BullishCount,
		&summary.BearishCount,
		&summary.NeutralCount,
		&summary.TotalPosts,
		&subreddits,
	)

	if err == sql.ErrNoRows {
		// Return empty summary for ticker with no posts
		return &models.TickerSentimentSummary{
			Ticker:         ticker,
			TrendDirection: "stable",
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ticker sentiment summary: %w", err)
	}

	if sentimentScore.Valid {
		summary.SentimentScore = &sentimentScore.Float64
	}
	summary.TopSubreddits = subreddits

	// Determine trend direction based on sentiment
	if sentimentScore.Valid {
		if sentimentScore.Float64 > 20 {
			summary.TrendDirection = "bullish"
		} else if sentimentScore.Float64 < -20 {
			summary.TrendDirection = "bearish"
		} else {
			summary.TrendDirection = "neutral"
		}
	} else {
		summary.TrendDirection = "stable"
	}

	return &summary, nil
}

// GetPostCountByTicker returns the number of posts for a ticker in the given time period
func GetPostCountByTicker(ticker string, days int) (int, error) {
	if days <= 0 {
		days = 7
	}

	query := `
		SELECT COUNT(*)
		FROM social_posts
		WHERE ticker = $1
		  AND posted_at > NOW() - $2::INTEGER * INTERVAL '1 day'
	`

	var count int
	err := DB.QueryRow(query, ticker, days).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get post count: %w", err)
	}

	return count, nil
}

// PruneOldPosts removes posts older than retention period
func PruneOldPosts(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 30 // Default 30 days
	}

	result, err := DB.Exec(
		"DELETE FROM social_posts WHERE posted_at < NOW() - $1 * INTERVAL '1 day'",
		retentionDays,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to prune old posts: %w", err)
	}

	return result.RowsAffected()
}

// UpdatePostSentiment updates the sentiment fields for a post
func UpdatePostSentiment(externalPostID string, sentiment string, confidence float64) error {
	query := `
		UPDATE social_posts
		SET sentiment = $1, sentiment_confidence = $2, updated_at = NOW()
		WHERE external_post_id = $3
	`

	result, err := DB.Exec(query, sentiment, confidence, externalPostID)
	if err != nil {
		return fmt.Errorf("failed to update post sentiment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("post not found: %s", externalPostID)
	}

	return nil
}

// GetPostsWithoutSentiment returns posts that need sentiment analysis
func GetPostsWithoutSentiment(limit int) ([]models.SocialPost, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, external_post_id, source, ticker, subreddit, title,
			   body_preview, url, upvotes, comment_count, award_count,
			   sentiment, sentiment_confidence, flair, posted_at, created_at, updated_at
		FROM social_posts
		WHERE sentiment IS NULL
		  AND posted_at > NOW() - INTERVAL '7 days'
		ORDER BY posted_at DESC
		LIMIT $1
	`

	rows, err := DB.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query posts without sentiment: %w", err)
	}
	defer rows.Close()

	var posts []models.SocialPost
	for rows.Next() {
		var p models.SocialPost
		var bodyPreview, sentiment, flair sql.NullString
		var sentimentConfidence sql.NullFloat64

		err := rows.Scan(
			&p.ID,
			&p.ExternalPostID,
			&p.Source,
			&p.Ticker,
			&p.Subreddit,
			&p.Title,
			&bodyPreview,
			&p.URL,
			&p.Upvotes,
			&p.CommentCount,
			&p.AwardCount,
			&sentiment,
			&sentimentConfidence,
			&flair,
			&p.PostedAt,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan social post: %w", err)
		}

		if bodyPreview.Valid {
			p.BodyPreview = &bodyPreview.String
		}
		if sentiment.Valid {
			p.Sentiment = &sentiment.String
		}
		if sentimentConfidence.Valid {
			p.SentimentConfidence = &sentimentConfidence.Float64
		}
		if flair.Valid {
			p.Flair = &flair.String
		}

		posts = append(posts, p)
	}

	return posts, nil
}

// BulkUpsertPosts inserts or updates multiple posts efficiently
func BulkUpsertPosts(posts []models.SocialPost) (int, error) {
	if len(posts) == 0 {
		return 0, nil
	}

	tx, err := DB.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO social_posts (
			external_post_id, source, ticker, subreddit, title, body_preview,
			url, upvotes, comment_count, award_count, sentiment,
			sentiment_confidence, flair, posted_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
		ON CONFLICT (external_post_id) DO UPDATE SET
			upvotes = EXCLUDED.upvotes,
			comment_count = EXCLUDED.comment_count,
			award_count = EXCLUDED.award_count,
			updated_at = NOW()
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	count := 0
	for _, post := range posts {
		_, err := stmt.Exec(
			post.ExternalPostID,
			post.Source,
			post.Ticker,
			post.Subreddit,
			post.Title,
			post.BodyPreview,
			post.URL,
			post.Upvotes,
			post.CommentCount,
			post.AwardCount,
			post.Sentiment,
			post.SentimentConfidence,
			post.Flair,
			post.PostedAt,
		)
		if err != nil {
			// Log error but continue with other posts
			continue
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return count, nil
}

// GetLatestPostDate returns the most recent post date for a ticker
func GetLatestPostDate(ticker string) (time.Time, error) {
	query := `SELECT MAX(posted_at) FROM social_posts WHERE ticker = $1`

	var latestDate sql.NullTime
	err := DB.QueryRow(query, ticker).Scan(&latestDate)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get latest post date: %w", err)
	}

	if !latestDate.Valid {
		return time.Time{}, sql.ErrNoRows
	}

	return latestDate.Time, nil
}

// GetPostCountByExternalID checks if a post exists by external ID
func GetPostCountByExternalID(externalID string) (int, error) {
	var count int
	err := DB.QueryRow(
		"SELECT COUNT(*) FROM social_posts WHERE external_post_id = $1",
		externalID,
	).Scan(&count)
	return count, err
}

// GetTickerSentiment returns sentiment data for the API response
func GetTickerSentiment(ticker string) (*models.SentimentResponse, error) {
	// Get sentiment breakdown and counts
	query := `
		SELECT
			COALESCE(AVG(
				CASE sentiment
					WHEN 'bullish' THEN 1
					WHEN 'bearish' THEN -1
					ELSE 0
				END
			), 0) as score,
			COUNT(*) FILTER (WHERE sentiment = 'bullish') as bullish_count,
			COUNT(*) FILTER (WHERE sentiment = 'bearish') as bearish_count,
			COUNT(*) FILTER (WHERE sentiment = 'neutral' OR sentiment IS NULL) as neutral_count,
			COUNT(*) as total_count,
			COUNT(*) FILTER (WHERE posted_at > NOW() - INTERVAL '24 hours') as count_24h,
			COUNT(*) FILTER (WHERE posted_at > NOW() - INTERVAL '7 days') as count_7d,
			MAX(posted_at) as last_updated
		FROM social_posts
		WHERE ticker = $1
		  AND posted_at > NOW() - INTERVAL '30 days'
	`

	var score float64
	var bullishCount, bearishCount, neutralCount, totalCount, count24h, count7d int
	var lastUpdated sql.NullTime

	err := DB.QueryRow(query, ticker).Scan(
		&score, &bullishCount, &bearishCount, &neutralCount,
		&totalCount, &count24h, &count7d, &lastUpdated,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticker sentiment: %w", err)
	}

	// Get company name from stocks table
	var companyName sql.NullString
	DB.QueryRow("SELECT name FROM stocks WHERE symbol = $1", ticker).Scan(&companyName)

	// Get top subreddits
	subredditQuery := `
		SELECT subreddit, COUNT(*) as count
		FROM social_posts
		WHERE ticker = $1
		  AND posted_at > NOW() - INTERVAL '7 days'
		GROUP BY subreddit
		ORDER BY count DESC
		LIMIT 5
	`

	rows, err := DB.Query(subredditQuery, ticker)
	if err != nil {
		return nil, fmt.Errorf("failed to get top subreddits: %w", err)
	}
	defer rows.Close()

	var topSubreddits []models.SubredditCount
	for rows.Next() {
		var sc models.SubredditCount
		if err := rows.Scan(&sc.Subreddit, &sc.Count); err != nil {
			continue
		}
		topSubreddits = append(topSubreddits, sc)
	}

	// Calculate rank among all tickers (by post count in last 7 days)
	rankQuery := `
		WITH ticker_ranks AS (
			SELECT ticker, COUNT(*) as post_count,
				   ROW_NUMBER() OVER (ORDER BY COUNT(*) DESC) as rank
			FROM social_posts
			WHERE posted_at > NOW() - INTERVAL '7 days'
			GROUP BY ticker
		)
		SELECT COALESCE(rank, 0) FROM ticker_ranks WHERE ticker = $1
	`
	var rank int
	DB.QueryRow(rankQuery, ticker).Scan(&rank)

	// Calculate rank change (compare current vs previous 7 day period)
	rankChangeQuery := `
		WITH current_ranks AS (
			SELECT ticker, COUNT(*) as post_count,
				   ROW_NUMBER() OVER (ORDER BY COUNT(*) DESC) as rank
			FROM social_posts
			WHERE posted_at > NOW() - INTERVAL '7 days'
			GROUP BY ticker
		),
		previous_ranks AS (
			SELECT ticker, COUNT(*) as post_count,
				   ROW_NUMBER() OVER (ORDER BY COUNT(*) DESC) as rank
			FROM social_posts
			WHERE posted_at > NOW() - INTERVAL '14 days'
			  AND posted_at <= NOW() - INTERVAL '7 days'
			GROUP BY ticker
		)
		SELECT COALESCE(p.rank, 100) - COALESCE(c.rank, 100)
		FROM current_ranks c
		LEFT JOIN previous_ranks p ON c.ticker = p.ticker
		WHERE c.ticker = $1
	`
	var rankChange int
	DB.QueryRow(rankChangeQuery, ticker).Scan(&rankChange)

	// Calculate percentages
	var breakdown models.SentimentBreakdown
	if totalCount > 0 {
		breakdown.Bullish = float64(bullishCount) / float64(totalCount) * 100
		breakdown.Bearish = float64(bearishCount) / float64(totalCount) * 100
		breakdown.Neutral = float64(neutralCount) / float64(totalCount) * 100
	}

	response := &models.SentimentResponse{
		Ticker:        ticker,
		Score:         score,
		Label:         models.GetSentimentLabel(score),
		Breakdown:     breakdown,
		PostCount24h:  count24h,
		PostCount7d:   count7d,
		Rank:          rank,
		RankChange:    rankChange,
		TopSubreddits: topSubreddits,
	}

	if companyName.Valid {
		response.CompanyName = companyName.String
	}

	if lastUpdated.Valid {
		response.LastUpdated = lastUpdated.Time
	} else {
		response.LastUpdated = time.Now()
	}

	return response, nil
}

// GetSentimentHistory returns daily sentiment data for a ticker
func GetSentimentHistory(ticker string, days int) (*models.SentimentHistoryResponse, error) {
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}

	query := `
		SELECT
			DATE(posted_at) as date,
			COALESCE(AVG(
				CASE sentiment
					WHEN 'bullish' THEN 1
					WHEN 'bearish' THEN -1
					ELSE 0
				END
			), 0) as score,
			COUNT(*) as post_count,
			COUNT(*) FILTER (WHERE sentiment = 'bullish') as bullish,
			COUNT(*) FILTER (WHERE sentiment = 'bearish') as bearish,
			COUNT(*) FILTER (WHERE sentiment = 'neutral' OR sentiment IS NULL) as neutral
		FROM social_posts
		WHERE ticker = $1
		  AND posted_at > NOW() - $2::INTEGER * INTERVAL '1 day'
		GROUP BY DATE(posted_at)
		ORDER BY date ASC
	`

	rows, err := DB.Query(query, ticker, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get sentiment history: %w", err)
	}
	defer rows.Close()

	var history []models.SentimentHistoryPoint
	for rows.Next() {
		var point models.SentimentHistoryPoint
		var date time.Time
		err := rows.Scan(&date, &point.Score, &point.PostCount, &point.Bullish, &point.Bearish, &point.Neutral)
		if err != nil {
			continue
		}
		point.Date = date.Format("2006-01-02")
		history = append(history, point)
	}

	period := fmt.Sprintf("%dd", days)
	return &models.SentimentHistoryResponse{
		Ticker:  ticker,
		Period:  period,
		History: history,
	}, nil
}

// GetTrendingTickers returns the most active tickers by social media activity
func GetTrendingTickers(period string, limit int) (*models.TrendingResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	// Determine interval based on period
	interval := "24 hours"
	previousInterval := "48 hours"
	if period == "7d" {
		interval = "7 days"
		previousInterval = "14 days"
	}

	query := fmt.Sprintf(`
		WITH current_period AS (
			SELECT
				ticker,
				COALESCE(AVG(
					CASE sentiment
						WHEN 'bullish' THEN 1
						WHEN 'bearish' THEN -1
						ELSE 0
					END
				), 0) as score,
				COUNT(*) as post_count
			FROM social_posts
			WHERE posted_at > NOW() - INTERVAL '%s'
			GROUP BY ticker
		),
		previous_period AS (
			SELECT
				ticker,
				COUNT(*) as post_count
			FROM social_posts
			WHERE posted_at > NOW() - INTERVAL '%s'
			  AND posted_at <= NOW() - INTERVAL '%s'
			GROUP BY ticker
		)
		SELECT
			c.ticker,
			COALESCE(s.name, '') as company_name,
			c.score,
			c.post_count,
			COALESCE(
				CASE WHEN p.post_count > 0
					THEN ((c.post_count::float - p.post_count::float) / p.post_count::float) * 100
					ELSE 100
				END,
				100
			) as mention_delta
		FROM current_period c
		LEFT JOIN previous_period p ON c.ticker = p.ticker
		LEFT JOIN stocks s ON c.ticker = s.symbol
		WHERE c.post_count >= 3
		ORDER BY c.post_count DESC
		LIMIT $1
	`, interval, previousInterval, interval)

	rows, err := DB.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get trending tickers: %w", err)
	}
	defer rows.Close()

	var tickers []models.TrendingTicker
	rank := 1
	for rows.Next() {
		var t models.TrendingTicker
		err := rows.Scan(&t.Ticker, &t.CompanyName, &t.Score, &t.PostCount, &t.MentionDelta)
		if err != nil {
			continue
		}
		t.Label = models.GetSentimentLabel(t.Score)
		t.Rank = rank
		rank++
		tickers = append(tickers, t)
	}

	return &models.TrendingResponse{
		Period:    period,
		Tickers:   tickers,
		UpdatedAt: time.Now(),
	}, nil
}

// GetRepresentativePostsForAPI returns posts formatted for the API response
func GetRepresentativePostsForAPI(ticker string, sort models.SocialPostSortOption, limit int) (*models.RepresentativePostsResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}

	// Build ORDER BY and WHERE based on sort option
	orderBy := "posted_at DESC" // default: recent
	whereClause := ""

	switch sort {
	case models.SortByEngagement:
		orderBy = "(upvotes + comment_count * 2 + award_count * 5) DESC"
	case models.SortByBullish:
		whereClause = "AND sentiment = 'bullish'"
		orderBy = "sentiment_confidence DESC NULLS LAST, upvotes DESC"
	case models.SortByBearish:
		whereClause = "AND sentiment = 'bearish'"
		orderBy = "sentiment_confidence DESC NULLS LAST, upvotes DESC"
	case models.SortByRecent:
		orderBy = "posted_at DESC"
	}

	query := fmt.Sprintf(`
		SELECT
			id, title, body_preview, url, source, subreddit,
			upvotes, comment_count, award_count,
			COALESCE(sentiment, 'neutral') as sentiment,
			sentiment_confidence, flair, posted_at
		FROM social_posts
		WHERE ticker = $1
		  AND posted_at > NOW() - INTERVAL '7 days'
		  %s
		ORDER BY %s
		LIMIT $2
	`, whereClause, orderBy)

	rows, err := DB.Query(query, ticker, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get representative posts: %w", err)
	}
	defer rows.Close()

	var posts []models.RepresentativePost
	for rows.Next() {
		var p models.RepresentativePost
		var bodyPreview, flair sql.NullString
		var sentimentConfidence sql.NullFloat64
		var postedAt time.Time

		err := rows.Scan(
			&p.ID, &p.Title, &bodyPreview, &p.URL, &p.Source, &p.Subreddit,
			&p.Upvotes, &p.CommentCount, &p.AwardCount,
			&p.Sentiment, &sentimentConfidence, &flair, &postedAt,
		)
		if err != nil {
			continue
		}

		// Handle nullable fields
		if bodyPreview.Valid {
			p.BodyPreview = &bodyPreview.String
		}
		if sentimentConfidence.Valid {
			p.SentimentConfidence = &sentimentConfidence.Float64
		}
		if flair.Valid {
			p.Flair = &flair.String
		}
		p.PostedAt = postedAt.Format(time.RFC3339)

		posts = append(posts, p)
	}

	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM social_posts WHERE ticker = $1 AND posted_at > NOW() - INTERVAL '7 days'`
	DB.QueryRow(countQuery, ticker).Scan(&total)

	// Determine sort string for response
	sortStr := "recent"
	switch sort {
	case models.SortByEngagement:
		sortStr = "engagement"
	case models.SortByBullish:
		sortStr = "bullish"
	case models.SortByBearish:
		sortStr = "bearish"
	}

	return &models.RepresentativePostsResponse{
		Ticker: ticker,
		Posts:  posts,
		Total:  total,
		Sort:   sortStr,
	}, nil
}
