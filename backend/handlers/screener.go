package handlers

import (
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"investorcenter-api/database"
	"investorcenter-api/models"

	"github.com/gin-gonic/gin"
)

// GetScreenerStocks handles the stock screener endpoint
// GET /api/v1/screener/stocks
func GetScreenerStocks(c *gin.Context) {
	// Check database connection
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Screener service is temporarily unavailable",
		})
		return
	}

	// Parse query parameters
	params := parseScreenerParams(c)

	// Fetch stocks from database
	stocks, total, err := database.GetScreenerStocks(params)
	if err != nil {
		log.Printf("Error fetching screener stocks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch stocks",
			"message": "An error occurred while retrieving screener data",
		})
		return
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))

	// Build response
	response := models.ScreenerResponse{
		Data: stocks,
		Meta: models.ScreenerMeta{
			Total:      total,
			Page:       params.Page,
			Limit:      params.Limit,
			TotalPages: totalPages,
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
		},
	}

	c.JSON(http.StatusOK, response)
}

// parseScreenerParams extracts and validates query parameters
func parseScreenerParams(c *gin.Context) models.ScreenerParams {
	params := models.ScreenerParams{
		Page:      1,
		Limit:     20000, // High default for client-side filtering
		Sort:      "market_cap",
		Order:     "desc",
		AssetType: "CS",
	}

	// Page
	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && page > 0 {
		params.Page = page
	}

	// Limit (max 20000 for client-side filtering screener)
	if limit, err := strconv.Atoi(c.DefaultQuery("limit", "20000")); err == nil && limit > 0 {
		if limit > 20000 {
			limit = 20000
		}
		params.Limit = limit
	}

	// Sort field
	sort := c.DefaultQuery("sort", "market_cap")
	if _, ok := database.ValidScreenerSortColumns[sort]; ok {
		params.Sort = sort
	}

	// Sort order
	order := strings.ToLower(c.DefaultQuery("order", "desc"))
	if order == "asc" || order == "desc" {
		params.Order = order
	}

	// Sectors (comma-separated)
	if sectors := c.Query("sectors"); sectors != "" {
		params.Sectors = strings.Split(sectors, ",")
		// Trim whitespace from each sector
		for i := range params.Sectors {
			params.Sectors[i] = strings.TrimSpace(params.Sectors[i])
		}
	}

	// Market cap filters
	if val, err := strconv.ParseFloat(c.Query("market_cap_min"), 64); err == nil {
		params.MarketCapMin = &val
	}
	if val, err := strconv.ParseFloat(c.Query("market_cap_max"), 64); err == nil {
		params.MarketCapMax = &val
	}

	// P/E ratio filters
	if val, err := strconv.ParseFloat(c.Query("pe_min"), 64); err == nil {
		params.PEMin = &val
	}
	if val, err := strconv.ParseFloat(c.Query("pe_max"), 64); err == nil {
		params.PEMax = &val
	}

	// Dividend yield filters
	if val, err := strconv.ParseFloat(c.Query("dividend_yield_min"), 64); err == nil {
		params.DividendYieldMin = &val
	}
	if val, err := strconv.ParseFloat(c.Query("dividend_yield_max"), 64); err == nil {
		params.DividendYieldMax = &val
	}

	// Revenue growth filters
	if val, err := strconv.ParseFloat(c.Query("revenue_growth_min"), 64); err == nil {
		params.RevenueGrowthMin = &val
	}
	if val, err := strconv.ParseFloat(c.Query("revenue_growth_max"), 64); err == nil {
		params.RevenueGrowthMax = &val
	}

	// IC Score filters
	if val, err := strconv.ParseFloat(c.Query("ic_score_min"), 64); err == nil {
		params.ICScoreMin = &val
	}
	if val, err := strconv.ParseFloat(c.Query("ic_score_max"), 64); err == nil {
		params.ICScoreMax = &val
	}

	// Asset type
	if assetType := c.Query("asset_type"); assetType != "" {
		params.AssetType = assetType
	}

	return params
}
