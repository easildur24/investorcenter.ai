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
