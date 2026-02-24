package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"investorcenter-api/database"
	"investorcenter-api/services"

	"github.com/gin-gonic/gin"
)

// polygonClientForReddit is a package-level client reused across heatmap requests,
// avoiding per-request construction of the HTTP client and connection pool.
var polygonClientForReddit = services.NewPolygonClient()

// GetRedditHeatmap returns trending tickers based on Reddit popularity
// Query params:
//   - days: number of days to aggregate (default: 7)
//   - top: limit number of results (default: 50, max: 100)
//
// Example: GET /api/v1/reddit/heatmap?days=7&top=20
func GetRedditHeatmap(c *gin.Context) {
	// Parse query parameters
	daysStr := c.DefaultQuery("days", "7")
	topStr := c.DefaultQuery("top", "50")

	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 {
		days = 7
	}
	if days > 90 {
		days = 90 // Cap at 90 days
	}

	limit, err := strconv.Atoi(topStr)
	if err != nil || limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100 // Cap at 100 results
	}

	// Fetch heatmap data from database
	heatmapData, err := database.GetRedditHeatmap(days, limit)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusOK, gin.H{
				"data": []interface{}{},
				"meta": gin.H{
					"days":  days,
					"limit": limit,
					"count": 0,
				},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch Reddit heatmap data",
			"details": err.Error(),
		})
		return
	}

	// BUG-003: Enrich with real-time price data from Polygon
	enrichWithPriceData(heatmapData)

	// Get latest date with data
	latestDate, err := database.GetLatestRedditDate()
	if err != nil && err != sql.ErrNoRows {
		// Log error but don't fail the request
		latestDate = heatmapData[0].Date
	}

	c.JSON(http.StatusOK, gin.H{
		"data": heatmapData,
		"meta": gin.H{
			"days":       days,
			"limit":      limit,
			"count":      len(heatmapData),
			"latestDate": latestDate,
		},
	})
}

// enrichWithPriceData fetches real-time prices for all tickers and attaches them
// to the heatmap data. Errors are logged but not propagated — missing price data
// is better than failing the entire request.
func enrichWithPriceData(data []database.RedditHeatmapData) {
	if len(data) == 0 {
		return
	}

	// Collect unique symbols (deduplicate to avoid wasting Polygon API quota)
	seen := make(map[string]bool, len(data))
	symbols := make([]string, 0, len(data))
	for _, item := range data {
		if !seen[item.TickerSymbol] {
			seen[item.TickerSymbol] = true
			symbols = append(symbols, item.TickerSymbol)
		}
	}

	// Fetch prices in bulk (reuse package-level client)
	quotes, err := polygonClientForReddit.GetMultipleQuotes(symbols)
	if err != nil {
		log.Printf("Warning: Failed to enrich Reddit heatmap with price data: %v\n", err)
		return
	}

	// Attach price data to each row
	for i := range data {
		if quote, ok := quotes[data[i].TickerSymbol]; ok {
			price := quote.Price
			changePct := quote.ChangePercent
			data[i].Price = &price
			data[i].PriceChangePct = &changePct
		}
	}
}

// GetRedditPipelineHealth returns pipeline freshness data for the DataFreshnessIndicator.
// BUG-004: Enables the frontend to show how fresh the data is and warn when stale.
// No auth required.
func GetRedditPipelineHealth(c *gin.Context) {
	health, err := database.GetRedditPipelineHealth()
	if err != nil {
		log.Printf("Error fetching pipeline health: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch pipeline health",
		})
		return
	}
	c.JSON(http.StatusOK, health)
}

// GetTickerRedditHistory returns Reddit popularity history for a specific ticker
// Query params:
//   - days: number of days of history (default: 30)
//
// Example: GET /api/v1/reddit/ticker/AAPL/history?days=30
func GetTickerRedditHistory(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Symbol is required",
		})
		return
	}

	// Parse days parameter
	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 {
		days = 30
	}
	if days > 90 {
		days = 90 // Cap at 90 days
	}

	// Fetch ticker history from database
	history, err := database.GetTickerRedditHistory(symbol, days)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"error":  "No Reddit data found for this ticker",
				"symbol": symbol,
				"days":   days,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch Reddit history",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": history,
		"meta": gin.H{
			"symbol": symbol,
			"days":   days,
			"count":  len(history.History),
		},
	})
}
