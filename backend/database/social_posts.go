package database

import (
	"database/sql"
	"fmt"
	"investorcenter-api/models"
	"time"
)

// Base WHERE clause for all queries against the joined reddit tables.
// Filters out unprocessed posts, non-finance posts, and spam.
const redditPostBaseFilter = `
	r.processed_at IS NOT NULL
	AND r.is_finance_related = TRUE
	AND COALESCE(r.spam_score, 0) < 0.5
`

// GetTickerSentiment returns sentiment data for the API response
func GetTickerSentiment(ticker string) (*models.SentimentResponse, error) {
	query := `
		SELECT
			COALESCE(AVG(
				CASE t.sentiment
					WHEN 'bullish' THEN 1
					WHEN 'bearish' THEN -1
					ELSE 0
				END
			), 0) as score,
			COUNT(*) FILTER (WHERE t.sentiment = 'bullish') as bullish_count,
			COUNT(*) FILTER (WHERE t.sentiment = 'bearish') as bearish_count,
			COUNT(*) FILTER (WHERE t.sentiment = 'neutral' OR t.sentiment IS NULL) as neutral_count,
			COUNT(*) as total_count,
			COUNT(*) FILTER (WHERE r.posted_at > NOW() - INTERVAL '24 hours') as count_24h,
			COUNT(*) FILTER (WHERE r.posted_at > NOW() - INTERVAL '7 days') as count_7d,
			MAX(r.posted_at) as last_updated
		FROM reddit_post_tickers t
		JOIN reddit_posts_raw r ON t.post_id = r.id
		WHERE t.ticker = $1
		  AND r.posted_at > NOW() - INTERVAL '30 days'
		  AND ` + redditPostBaseFilter

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

	// Get company name from stocks table (best-effort, non-critical)
	var companyName sql.NullString
	_ = DB.QueryRow("SELECT name FROM tickers WHERE symbol = $1", ticker).Scan(&companyName)

	// Get top subreddits
	subredditQuery := `
		SELECT r.subreddit, COUNT(*) as count
		FROM reddit_post_tickers t
		JOIN reddit_posts_raw r ON t.post_id = r.id
		WHERE t.ticker = $1
		  AND r.posted_at > NOW() - INTERVAL '7 days'
		  AND ` + redditPostBaseFilter + `
		GROUP BY r.subreddit
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
			SELECT t.ticker, COUNT(*) as post_count,
				   ROW_NUMBER() OVER (ORDER BY COUNT(*) DESC) as rank
			FROM reddit_post_tickers t
			JOIN reddit_posts_raw r ON t.post_id = r.id
			WHERE r.posted_at > NOW() - INTERVAL '7 days'
			  AND ` + redditPostBaseFilter + `
			GROUP BY t.ticker
		)
		SELECT COALESCE(rank, 0) FROM ticker_ranks WHERE ticker = $1
	`
	var rank int
	_ = DB.QueryRow(rankQuery, ticker).Scan(&rank)

	// Calculate rank change (compare current vs previous 7 day period)
	rankChangeQuery := `
		WITH current_ranks AS (
			SELECT t.ticker, COUNT(*) as post_count,
				   ROW_NUMBER() OVER (ORDER BY COUNT(*) DESC) as rank
			FROM reddit_post_tickers t
			JOIN reddit_posts_raw r ON t.post_id = r.id
			WHERE r.posted_at > NOW() - INTERVAL '7 days'
			  AND ` + redditPostBaseFilter + `
			GROUP BY t.ticker
		),
		previous_ranks AS (
			SELECT t.ticker, COUNT(*) as post_count,
				   ROW_NUMBER() OVER (ORDER BY COUNT(*) DESC) as rank
			FROM reddit_post_tickers t
			JOIN reddit_posts_raw r ON t.post_id = r.id
			WHERE r.posted_at > NOW() - INTERVAL '14 days'
			  AND r.posted_at <= NOW() - INTERVAL '7 days'
			  AND ` + redditPostBaseFilter + `
			GROUP BY t.ticker
		)
		SELECT COALESCE(p.rank, 100) - COALESCE(c.rank, 100)
		FROM current_ranks c
		LEFT JOIN previous_ranks p ON c.ticker = p.ticker
		WHERE c.ticker = $1
	`
	var rankChange int
	_ = DB.QueryRow(rankChangeQuery, ticker).Scan(&rankChange)

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
			DATE(r.posted_at) as date,
			COALESCE(AVG(
				CASE t.sentiment
					WHEN 'bullish' THEN 1
					WHEN 'bearish' THEN -1
					ELSE 0
				END
			), 0) as score,
			COUNT(*) as post_count,
			COUNT(*) FILTER (WHERE t.sentiment = 'bullish') as bullish,
			COUNT(*) FILTER (WHERE t.sentiment = 'bearish') as bearish,
			COUNT(*) FILTER (WHERE t.sentiment = 'neutral' OR t.sentiment IS NULL) as neutral
		FROM reddit_post_tickers t
		JOIN reddit_posts_raw r ON t.post_id = r.id
		WHERE t.ticker = $1
		  AND r.posted_at > NOW() - $2::INTEGER * INTERVAL '1 day'
		  AND ` + redditPostBaseFilter + `
		GROUP BY DATE(r.posted_at)
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
				t.ticker,
				COALESCE(AVG(
					CASE t.sentiment
						WHEN 'bullish' THEN 1
						WHEN 'bearish' THEN -1
						ELSE 0
					END
				), 0) as score,
				COUNT(*) as post_count
			FROM reddit_post_tickers t
			JOIN reddit_posts_raw r ON t.post_id = r.id
			WHERE r.posted_at > NOW() - INTERVAL '%s'
			  AND `+redditPostBaseFilter+`
			GROUP BY t.ticker
		),
		previous_period AS (
			SELECT
				t.ticker,
				COUNT(*) as post_count
			FROM reddit_post_tickers t
			JOIN reddit_posts_raw r ON t.post_id = r.id
			WHERE r.posted_at > NOW() - INTERVAL '%s'
			  AND r.posted_at <= NOW() - INTERVAL '%s'
			  AND `+redditPostBaseFilter+`
			GROUP BY t.ticker
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
		LEFT JOIN tickers s ON c.ticker = s.symbol
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
	orderBy := "r.posted_at DESC" // default: recent
	whereClause := ""

	switch sort {
	case models.SortByEngagement:
		orderBy = "(r.upvotes + r.comment_count * 2) DESC"
	case models.SortByBullish:
		whereClause = "AND t.sentiment = 'bullish'"
		orderBy = "t.confidence DESC NULLS LAST, r.upvotes DESC"
	case models.SortByBearish:
		whereClause = "AND t.sentiment = 'bearish'"
		orderBy = "t.confidence DESC NULLS LAST, r.upvotes DESC"
	case models.SortByRecent:
		orderBy = "r.posted_at DESC"
	}

	query := fmt.Sprintf(`
		SELECT
			t.id, r.title, LEFT(r.body, 500) as body_preview,
			r.url, 'reddit' as source, r.subreddit,
			r.upvotes, r.comment_count, COALESCE(r.award_count, 0) as award_count,
			COALESCE(t.sentiment, 'neutral') as sentiment,
			t.confidence, r.flair, r.posted_at
		FROM reddit_post_tickers t
		JOIN reddit_posts_raw r ON t.post_id = r.id
		WHERE t.ticker = $1
		  AND r.posted_at > NOW() - INTERVAL '7 days'
		  AND `+redditPostBaseFilter+`
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
	countQuery := `
		SELECT COUNT(*)
		FROM reddit_post_tickers t
		JOIN reddit_posts_raw r ON t.post_id = r.id
		WHERE t.ticker = $1
		  AND r.posted_at > NOW() - INTERVAL '7 days'
		  AND ` + redditPostBaseFilter
	_ = DB.QueryRow(countQuery, ticker).Scan(&total)

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
