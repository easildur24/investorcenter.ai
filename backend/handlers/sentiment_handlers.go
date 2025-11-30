package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"investorcenter-api/database"

	"github.com/gin-gonic/gin"
)

// GetTrendingSentiment returns trending tickers by social media activity
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

	trending, err := database.GetTrendingTickers(period, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch trending sentiment",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, trending)
}

// GetTickerSentiment returns sentiment analysis for a specific ticker
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

	sentiment, err := database.GetTickerSentiment(ticker)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch ticker sentiment",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, sentiment)
}

// GetTickerSentimentHistory returns historical sentiment data for a ticker
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

	history, err := database.GetSentimentHistory(ticker, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch sentiment history",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, history)
}

// GetTickerPosts returns representative social media posts for a ticker
// URL param: ticker (required)
// Query params:
//   - limit: number of posts (default: 10, max: 20)
//
// Example: GET /api/sentiment/AAPL/posts?limit=10
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

	posts, err := database.GetRepresentativePostsForAPI(ticker, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch posts",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, posts)
}
