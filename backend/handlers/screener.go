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

const (
	// defaultScreenerLimit is the page size when no limit is specified.
	defaultScreenerLimit = 100
	// maxScreenerLimit caps the maximum rows a single request can return.
	maxScreenerLimit = 20000
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

// floatParam is a declarative definition for parsing a float64 query parameter.
type floatParam struct {
	key    string                                           // URL query key
	setter func(params *models.ScreenerParams, val float64) // Sets the value on ScreenerParams
}

// rangeParams defines all min/max float query parameters supported by the screener.
// Adding a new range filter requires:
//  1. Add fields to ScreenerParams (models/stock.go)
//  2. Add entry to RangeFilters (database/filter_registry.go)
//  3. Add entries here for URL param parsing
var rangeParams = []floatParam{
	// Market data
	{key: "market_cap_min", setter: func(p *models.ScreenerParams, v float64) { p.MarketCapMin = &v }},
	{key: "market_cap_max", setter: func(p *models.ScreenerParams, v float64) { p.MarketCapMax = &v }},

	// Valuation
	{key: "pe_min", setter: func(p *models.ScreenerParams, v float64) { p.PEMin = &v }},
	{key: "pe_max", setter: func(p *models.ScreenerParams, v float64) { p.PEMax = &v }},
	{key: "pb_min", setter: func(p *models.ScreenerParams, v float64) { p.PBMin = &v }},
	{key: "pb_max", setter: func(p *models.ScreenerParams, v float64) { p.PBMax = &v }},
	{key: "ps_min", setter: func(p *models.ScreenerParams, v float64) { p.PSMin = &v }},
	{key: "ps_max", setter: func(p *models.ScreenerParams, v float64) { p.PSMax = &v }},

	// Profitability
	{key: "roe_min", setter: func(p *models.ScreenerParams, v float64) { p.ROEMin = &v }},
	{key: "roe_max", setter: func(p *models.ScreenerParams, v float64) { p.ROEMax = &v }},
	{key: "roa_min", setter: func(p *models.ScreenerParams, v float64) { p.ROAMin = &v }},
	{key: "roa_max", setter: func(p *models.ScreenerParams, v float64) { p.ROAMax = &v }},
	{key: "gross_margin_min", setter: func(p *models.ScreenerParams, v float64) { p.GrossMarginMin = &v }},
	{key: "gross_margin_max", setter: func(p *models.ScreenerParams, v float64) { p.GrossMarginMax = &v }},
	{key: "net_margin_min", setter: func(p *models.ScreenerParams, v float64) { p.NetMarginMin = &v }},
	{key: "net_margin_max", setter: func(p *models.ScreenerParams, v float64) { p.NetMarginMax = &v }},

	// Financial health
	{key: "de_min", setter: func(p *models.ScreenerParams, v float64) { p.DebtToEquityMin = &v }},
	{key: "de_max", setter: func(p *models.ScreenerParams, v float64) { p.DebtToEquityMax = &v }},
	{key: "current_ratio_min", setter: func(p *models.ScreenerParams, v float64) { p.CurrentRatioMin = &v }},
	{key: "current_ratio_max", setter: func(p *models.ScreenerParams, v float64) { p.CurrentRatioMax = &v }},

	// Growth
	{key: "revenue_growth_min", setter: func(p *models.ScreenerParams, v float64) { p.RevenueGrowthMin = &v }},
	{key: "revenue_growth_max", setter: func(p *models.ScreenerParams, v float64) { p.RevenueGrowthMax = &v }},
	{key: "eps_growth_min", setter: func(p *models.ScreenerParams, v float64) { p.EPSGrowthMin = &v }},
	{key: "eps_growth_max", setter: func(p *models.ScreenerParams, v float64) { p.EPSGrowthMax = &v }},

	// Dividends
	{key: "dividend_yield_min", setter: func(p *models.ScreenerParams, v float64) { p.DividendYieldMin = &v }},
	{key: "dividend_yield_max", setter: func(p *models.ScreenerParams, v float64) { p.DividendYieldMax = &v }},
	{key: "payout_ratio_min", setter: func(p *models.ScreenerParams, v float64) { p.PayoutRatioMin = &v }},
	{key: "payout_ratio_max", setter: func(p *models.ScreenerParams, v float64) { p.PayoutRatioMax = &v }},
	{key: "consec_div_years_min", setter: func(p *models.ScreenerParams, v float64) { p.ConsecutiveDivYearsMin = &v }},

	// Risk
	{key: "beta_min", setter: func(p *models.ScreenerParams, v float64) { p.BetaMin = &v }},
	{key: "beta_max", setter: func(p *models.ScreenerParams, v float64) { p.BetaMax = &v }},

	// Fair value
	{key: "dcf_upside_min", setter: func(p *models.ScreenerParams, v float64) { p.DCFUpsideMin = &v }},
	{key: "dcf_upside_max", setter: func(p *models.ScreenerParams, v float64) { p.DCFUpsideMax = &v }},

	// IC Score
	{key: "ic_score_min", setter: func(p *models.ScreenerParams, v float64) { p.ICScoreMin = &v }},
	{key: "ic_score_max", setter: func(p *models.ScreenerParams, v float64) { p.ICScoreMax = &v }},

	// IC Score sub-factors
	{key: "value_score_min", setter: func(p *models.ScreenerParams, v float64) { p.ValueScoreMin = &v }},
	{key: "value_score_max", setter: func(p *models.ScreenerParams, v float64) { p.ValueScoreMax = &v }},
	{key: "growth_score_min", setter: func(p *models.ScreenerParams, v float64) { p.GrowthScoreMin = &v }},
	{key: "growth_score_max", setter: func(p *models.ScreenerParams, v float64) { p.GrowthScoreMax = &v }},
	{key: "profitability_score_min", setter: func(p *models.ScreenerParams, v float64) { p.ProfitabilityScoreMin = &v }},
	{key: "profitability_score_max", setter: func(p *models.ScreenerParams, v float64) { p.ProfitabilityScoreMax = &v }},
	{key: "financial_health_score_min", setter: func(p *models.ScreenerParams, v float64) { p.FinancialHealthScoreMin = &v }},
	{key: "financial_health_score_max", setter: func(p *models.ScreenerParams, v float64) { p.FinancialHealthScoreMax = &v }},
	{key: "momentum_score_min", setter: func(p *models.ScreenerParams, v float64) { p.MomentumScoreMin = &v }},
	{key: "momentum_score_max", setter: func(p *models.ScreenerParams, v float64) { p.MomentumScoreMax = &v }},
	{key: "analyst_score_min", setter: func(p *models.ScreenerParams, v float64) { p.AnalystScoreMin = &v }},
	{key: "analyst_score_max", setter: func(p *models.ScreenerParams, v float64) { p.AnalystScoreMax = &v }},
	{key: "insider_score_min", setter: func(p *models.ScreenerParams, v float64) { p.InsiderScoreMin = &v }},
	{key: "insider_score_max", setter: func(p *models.ScreenerParams, v float64) { p.InsiderScoreMax = &v }},
	{key: "institutional_score_min", setter: func(p *models.ScreenerParams, v float64) { p.InstitutionalScoreMin = &v }},
	{key: "institutional_score_max", setter: func(p *models.ScreenerParams, v float64) { p.InstitutionalScoreMax = &v }},
	{key: "sentiment_score_min", setter: func(p *models.ScreenerParams, v float64) { p.SentimentScoreMin = &v }},
	{key: "sentiment_score_max", setter: func(p *models.ScreenerParams, v float64) { p.SentimentScoreMax = &v }},
	{key: "technical_score_min", setter: func(p *models.ScreenerParams, v float64) { p.TechnicalScoreMin = &v }},
	{key: "technical_score_max", setter: func(p *models.ScreenerParams, v float64) { p.TechnicalScoreMax = &v }},
}

// parseScreenerParams extracts and validates query parameters
func parseScreenerParams(c *gin.Context) models.ScreenerParams {
	params := models.ScreenerParams{
		Page:  1,
		Limit: defaultScreenerLimit,
		Sort:  "market_cap",
		Order: "desc",
		// AssetType is parsed below but not used in the query — the
		// screener_data materialized view already filters to asset_type='CS'.
		// Kept for future use if the screener expands to ETFs.
		AssetType: "CS",
	}

	// Page
	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && page > 0 {
		params.Page = page
	}

	// Limit (capped at maxScreenerLimit)
	if raw := c.Query("limit"); raw != "" {
		if limit, err := strconv.Atoi(raw); err == nil && limit > 0 {
			if limit > maxScreenerLimit {
				limit = maxScreenerLimit
			}
			params.Limit = limit
		}
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
		for i := range params.Sectors {
			params.Sectors[i] = strings.TrimSpace(params.Sectors[i])
		}
	}

	// Industries (comma-separated)
	if industries := c.Query("industries"); industries != "" {
		params.Industries = strings.Split(industries, ",")
		for i := range params.Industries {
			params.Industries[i] = strings.TrimSpace(params.Industries[i])
		}
	}

	// Parse all range filters declaratively
	for _, fp := range rangeParams {
		if raw := c.Query(fp.key); raw != "" {
			if val, err := strconv.ParseFloat(raw, 64); err == nil {
				fp.setter(&params, val)
			}
		}
	}

	// Asset type (validated against allowlist)
	if assetType := c.Query("asset_type"); assetType != "" {
		validAssetTypes := map[string]bool{
			"CS": true, "ETF": true, "ADRC": true, "ADRW": true,
			"WARRANT": true, "RIGHT": true, "UNIT": true,
			"PFD": true, "FUND": true, "SP": true, "OS": true,
			"crypto": true,
		}
		if validAssetTypes[assetType] {
			params.AssetType = assetType
		}
		// Invalid values silently fall back to default "CS"
	}

	return params
}
