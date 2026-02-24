package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"investorcenter-api/database"
	"investorcenter-api/models"

	"github.com/gin-gonic/gin"
)

// GetTrendingSentiment returns trending tickers by social media activity.
// Reads from ticker_sentiment_snapshots (V2) instead of social_posts (V1).
//
// Query params:
//   - period: "24h" or "7d" (default: "24h")
//   - limit: number of results (default: 20, max: 50)
//
// Example: GET /api/sentiment/trending?period=24h&limit=20
func GetTrendingSentiment(c *gin.Context) {
	period := c.DefaultQuery("period", "24h")
	if period != "24h" && period != "7d" {
		period = "24h"
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	// Map API period to snapshot time_range
	timeRange := "1d"
	if period == "7d" {
		timeRange = "7d"
	}

	snapshots, err := database.GetLatestSnapshots(timeRange, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch trending sentiment",
			"details": err.Error(),
		})
		return
	}

	// Batch-fetch company names
	symbols := make([]string, len(snapshots))
	for i, s := range snapshots {
		symbols[i] = s.Ticker
	}
	companyNames, _ := database.GetCompanyNames(symbols)

	// Transform snapshots to TrendingTicker response
	tickers := make([]models.TrendingTicker, 0, len(snapshots))
	for _, s := range snapshots {
		t := models.TrendingTicker{
			Ticker:    s.Ticker,
			Score:     s.SentimentScore,
			Label:     s.SentimentLabel,
			PostCount: s.MentionCount,
		}
		if name, ok := companyNames[s.Ticker]; ok {
			t.CompanyName = name
		}
		if s.MentionVelocity1h != nil {
			t.MentionDelta = *s.MentionVelocity1h
		}
		if s.Rank != nil {
			t.Rank = *s.Rank
		}
		tickers = append(tickers, t)
	}

	c.JSON(http.StatusOK, &models.TrendingResponse{
		Period:    period,
		Tickers:   tickers,
		UpdatedAt: time.Now(),
	})
}

// GetTickerSentiment returns sentiment analysis for a specific ticker.
// Reads from ticker_sentiment_snapshots (V2) instead of social_posts (V1).
//
// URL param: ticker (required)
//
// Example: GET /api/sentiment/AAPL
func GetTickerSentiment(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))
	if ticker == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Ticker symbol is required",
		})
		return
	}

	// Get 7d snapshot for main metrics
	snapshot7d, err := database.GetTickerSnapshot(ticker, "7d")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch ticker sentiment",
			"details": err.Error(),
		})
		return
	}

	// Handle no data: return empty response
	if snapshot7d == nil {
		c.JSON(http.StatusOK, &models.SentimentResponse{
			Ticker:        ticker,
			Label:         "neutral",
			TopSubreddits: []models.SubredditCount{},
			LastUpdated:   time.Now(),
		})
		return
	}

	// Get 1d snapshot for 24h post count
	var postCount24h int
	snapshot1d, err := database.GetTickerSnapshot(ticker, "1d")
	if err == nil && snapshot1d != nil {
		postCount24h = snapshot1d.MentionCount
	}

	// Get company name
	companyNames, _ := database.GetCompanyNames([]string{ticker})
	companyName := ""
	if name, ok := companyNames[ticker]; ok {
		companyName = name
	}

	// Parse subreddit distribution JSONB into top 5
	topSubreddits := parseTopSubreddits(snapshot7d.SubredditDistribution, 5)

	// Build response
	var rank, rankChange int
	if snapshot7d.Rank != nil {
		rank = *snapshot7d.Rank
	}
	if snapshot7d.RankChange != nil {
		rankChange = *snapshot7d.RankChange
	}

	response := &models.SentimentResponse{
		Ticker:      ticker,
		CompanyName: companyName,
		Score:       snapshot7d.SentimentScore,
		Label:       snapshot7d.SentimentLabel,
		Breakdown: models.SentimentBreakdown{
			Bullish: snapshot7d.BullishPct * 100,
			Bearish: snapshot7d.BearishPct * 100,
			Neutral: snapshot7d.NeutralPct * 100,
		},
		PostCount24h:  postCount24h,
		PostCount7d:   snapshot7d.MentionCount,
		Rank:          rank,
		RankChange:    rankChange,
		TopSubreddits: topSubreddits,
		LastUpdated:   snapshot7d.SnapshotTime,
	}

	c.JSON(http.StatusOK, response)
}

// GetTickerSentimentHistory returns historical sentiment data for a ticker.
// Reads from ticker_sentiment_history (V2) instead of social_posts (V1).
//
// URL param: ticker (required)
// Query params:
//   - days: number of days (default: 7, max: 90)
//
// Example: GET /api/sentiment/AAPL/history?days=30
func GetTickerSentimentHistory(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))
	if ticker == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Ticker symbol is required",
		})
		return
	}

	daysStr := c.DefaultQuery("days", "7")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 {
		days = 7
	}
	if days > 90 {
		days = 90
	}

	points, err := database.GetSentimentTimeSeries(ticker, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch sentiment history",
			"details": err.Error(),
		})
		return
	}

	// Group points by date (multiple snapshots per day → pick latest)
	history := groupTimeSeriesByDate(points)

	period := fmt.Sprintf("%dd", days)
	c.JSON(http.StatusOK, &models.SentimentHistoryResponse{
		Ticker:  ticker,
		Period:  period,
		History: history,
	})
}

// GetTickerPosts returns representative social media posts for a ticker.
// Reads from reddit_posts_raw + reddit_post_tickers (V2) instead of
// social_posts (V1).
//
// URL param: ticker (required)
// Query params:
//   - limit: number of posts (default: 10, max: 20)
//   - sort: sort option (default: "recent", options: "recent", "engagement", "bullish", "bearish")
//
// Example: GET /api/sentiment/AAPL/posts?limit=10&sort=engagement
func GetTickerPosts(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))
	if ticker == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Ticker symbol is required",
		})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}

	// Parse sort parameter
	sortStr := c.DefaultQuery("sort", "recent")
	var sortOpt models.SocialPostSortOption
	switch sortStr {
	case "engagement":
		sortOpt = models.SortByEngagement
	case "bullish":
		sortOpt = models.SortByBullish
	case "bearish":
		sortOpt = models.SortByBearish
	default:
		sortOpt = models.SortByRecent
	}

	posts, err := database.GetTickerPostsV2(ticker, sortOpt, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch posts",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, posts)
}

// --- Helper functions ---

// parseTopSubreddits parses the subreddit_distribution JSONB field into
// a sorted list of SubredditCount, returning the top N entries.
func parseTopSubreddits(data json.RawMessage, topN int) []models.SubredditCount {
	if len(data) == 0 {
		return []models.SubredditCount{}
	}

	var dist map[string]int
	if err := json.Unmarshal(data, &dist); err != nil {
		return []models.SubredditCount{}
	}

	counts := make([]models.SubredditCount, 0, len(dist))
	for sub, cnt := range dist {
		counts = append(counts, models.SubredditCount{
			Subreddit: sub,
			Count:     cnt,
		})
	}

	// Sort descending by count
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].Count > counts[j].Count
	})

	if len(counts) > topN {
		counts = counts[:topN]
	}
	return counts
}

// groupTimeSeriesByDate groups multiple time-series points per day into a
// single SentimentHistoryPoint per date (using the latest point per day).
func groupTimeSeriesByDate(points []models.SentimentTimeSeriesPoint) []models.SentimentHistoryPoint {
	if len(points) == 0 {
		return []models.SentimentHistoryPoint{}
	}

	// Map date string → latest point for that date
	type dateEntry struct {
		point models.SentimentTimeSeriesPoint
		date  string
	}
	byDate := make(map[string]dateEntry)

	for _, p := range points {
		dateStr := p.Time.Format("2006-01-02")
		existing, exists := byDate[dateStr]
		if !exists || p.Time.After(existing.point.Time) {
			byDate[dateStr] = dateEntry{point: p, date: dateStr}
		}
	}

	// Convert to sorted list
	history := make([]models.SentimentHistoryPoint, 0, len(byDate))
	for _, entry := range byDate {
		p := entry.point
		mentionCount := p.MentionCount

		// Compute bullish/bearish/neutral counts from percentages
		bullish := int(p.BullishPct * float64(mentionCount))

		var bearish, neutral int
		if p.BearishPct != nil && p.NeutralPct != nil {
			bearish = int(*p.BearishPct * float64(mentionCount))
			neutral = mentionCount - bullish - bearish
		} else {
			// Fallback for old rows without bearish_pct/neutral_pct:
			// attribute remaining to neutral
			neutral = mentionCount - bullish
			bearish = 0
		}

		// Ensure no negative counts from rounding
		if neutral < 0 {
			neutral = 0
		}
		if bearish < 0 {
			bearish = 0
		}

		history = append(history, models.SentimentHistoryPoint{
			Date:      entry.date,
			Score:     p.SentimentScore,
			PostCount: mentionCount,
			Bullish:   bullish,
			Bearish:   bearish,
			Neutral:   neutral,
		})
	}

	// Sort by date ascending
	sort.Slice(history, func(i, j int) bool {
		return history[i].Date < history[j].Date
	})

	return history
}
