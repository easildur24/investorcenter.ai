package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"investorcenter-api/database"
	"investorcenter-api/models"

	"github.com/gin-gonic/gin"
)

// GetICScore retrieves the IC Score for a specific ticker
// GET /api/v1/stocks/:ticker/ic-score
func GetICScore(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))

	if ticker == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticker symbol is required"})
		return
	}

	// Check database connection
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "IC Score service is temporarily unavailable",
		})
		return
	}

	// Query the most recent IC Score for this ticker
	var icScore models.ICScore
	query := `
		SELECT
			id, ticker, date, overall_score,
			value_score, growth_score, profitability_score, financial_health_score,
			momentum_score, analyst_consensus_score, insider_activity_score,
			institutional_score, news_sentiment_score, technical_score,
			rating, sector_percentile, confidence_level, data_completeness,
			created_at
		FROM ic_scores
		WHERE ticker = $1
		ORDER BY date DESC, created_at DESC
		LIMIT 1
	`

	err := database.DB.Get(&icScore, query, ticker)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "IC Score not found",
				"message": fmt.Sprintf("No IC Score available for %s. Score calculation may not have been run yet.", ticker),
				"ticker":  ticker,
			})
			return
		}
		log.Printf("Error fetching IC Score for %s: %v", ticker, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch IC Score",
			"message": "An error occurred while retrieving the IC Score",
		})
		return
	}

	// Convert to response format
	response := icScore.ToResponse()

	c.JSON(http.StatusOK, gin.H{
		"data": response,
		"meta": gin.H{
			"ticker":    ticker,
			"timestamp": icScore.CalculatedAt,
		},
	})
}

// GetICScores retrieves all IC Scores with pagination and filtering
// GET /api/v1/ic-scores?limit=20&offset=0&search=AAPL&sort=overall_score&order=desc
func GetICScores(c *gin.Context) {
	// Check database connection
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "IC Score service is temporarily unavailable",
		})
		return
	}

	// Parse query parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	search := strings.ToUpper(c.DefaultQuery("search", ""))
	sort := c.DefaultQuery("sort", "overall_score")
	order := c.DefaultQuery("order", "desc")

	// Validate limit
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}

	// Validate sort column
	validSortColumns := map[string]bool{
		"ticker":            true,
		"overall_score":     true,
		"rating":            true,
		"data_completeness": true,
		"created_at":        true,
	}
	if !validSortColumns[sort] {
		sort = "overall_score"
	}

	// Validate order
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	// Build query for latest scores per ticker
	whereClause := ""
	args := []interface{}{}
	if search != "" {
		whereClause = "WHERE ticker LIKE $1"
		args = append(args, search+"%")
	}

	// Query to get the latest IC Score for each ticker
	query := fmt.Sprintf(`
		WITH latest_scores AS (
			SELECT DISTINCT ON (ticker)
				ticker,
				overall_score,
				rating,
				data_completeness,
				created_at
			FROM ic_scores
			%s
			ORDER BY ticker, date DESC, created_at DESC
		)
		SELECT ticker, overall_score, rating, data_completeness, created_at
		FROM latest_scores
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sort, order, len(args)+1, len(args)+2)

	args = append(args, limit, offset)

	// Initialize with empty slice to ensure JSON returns [] not null
	scores := make([]models.ICScoreListItem, 0)
	err := database.DB.Select(&scores, query, args...)
	if err != nil {
		log.Printf("Error fetching IC Scores: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch IC Scores",
			"message": "An error occurred while retrieving IC Scores",
		})
		return
	}

	// Get total count
	countQuery := "SELECT COUNT(DISTINCT ticker) FROM ic_scores"
	if search != "" {
		countQuery += " WHERE ticker LIKE $1"
	}

	var totalCount int
	var countArgs []interface{}
	if search != "" {
		countArgs = []interface{}{search + "%"}
	}
	err = database.DB.Get(&totalCount, countQuery, countArgs...)
	if err != nil {
		log.Printf("Error counting IC Scores: %v", err)
		totalCount = 0
	}

	// Get total stocks count for context
	var totalStocks int
	database.DB.Get(&totalStocks, "SELECT COUNT(*) FROM stocks")

	c.JSON(http.StatusOK, gin.H{
		"data": scores,
		"meta": gin.H{
			"total":            totalCount,
			"limit":            limit,
			"offset":           offset,
			"total_stocks":     totalStocks,
			"coverage_percent": float64(totalCount) / float64(totalStocks) * 100,
			"search":           search,
			"sort":             sort,
			"order":            order,
		},
	})
}

// GetICScoreHistory retrieves historical IC Scores for a ticker
// GET /api/v1/stocks/:ticker/ic-score/history?days=90
func GetICScoreHistory(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))
	days, _ := strconv.Atoi(c.DefaultQuery("days", "90"))

	if ticker == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticker symbol is required"})
		return
	}

	// Check database connection
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Database not available",
		})
		return
	}

	// Validate days parameter
	if days < 1 {
		days = 90
	}
	if days > 1825 { // Max 5 years
		days = 1825
	}

	query := `
		SELECT
			id, ticker, date, overall_score,
			value_score, growth_score, profitability_score, financial_health_score,
			momentum_score, analyst_consensus_score, insider_activity_score,
			institutional_score, news_sentiment_score, technical_score,
			rating, sector_percentile, confidence_level, data_completeness,
			created_at
		FROM ic_scores
		WHERE ticker = $1 AND date >= CURRENT_DATE - $2::integer
		ORDER BY date ASC
	`

	var scores []models.ICScore
	err := database.DB.Select(&scores, query, ticker, days)
	if err != nil {
		log.Printf("Error fetching IC Score history for %s: %v", ticker, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch IC Score history",
		})
		return
	}

	// Convert to response format
	responses := make([]models.ICScoreResponse, len(scores))
	for i, score := range scores {
		responses[i] = score.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"data": responses,
		"meta": gin.H{
			"ticker": ticker,
			"days":   days,
			"count":  len(responses),
		},
	})
}
