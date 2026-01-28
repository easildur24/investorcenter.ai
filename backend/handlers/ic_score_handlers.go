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
	"investorcenter-api/services"

	"github.com/gin-gonic/gin"
)

// fmpClient is a package-level FMP client instance
var fmpClient *services.FMPClient

// polygonClient is a package-level Polygon client instance
var polygonClient *services.PolygonClient

func init() {
	fmpClient = services.NewFMPClient()
	polygonClient = services.NewPolygonClient()
}

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

	// Ensure scores is never nil (sqlx may set it to nil if no rows found)
	if scores == nil {
		scores = make([]models.ICScoreListItem, 0)
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

	// Get total tickers count for context
	var totalStocks int
	database.DB.Get(&totalStocks, "SELECT COUNT(*) FROM tickers")

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

// GetFinancialMetrics retrieves financial metrics for a ticker
// Uses FMP API as primary source with database as fallback
// GET /api/v1/stocks/:ticker/financials
func GetFinancialMetrics(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))

	if ticker == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticker symbol is required"})
		return
	}

	// Check database connection
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Financial metrics service is temporarily unavailable",
		})
		return
	}

	// Try to fetch from FMP API (real-time TTM ratios)
	var fmpData *services.FMPRatiosTTM
	var fmpErr error
	if fmpClient != nil && fmpClient.APIKey != "" {
		fmpData, fmpErr = fmpClient.GetRatiosTTM(ticker)
		if fmpErr != nil {
			log.Printf("FMP API error for %s (falling back to DB): %v", ticker, fmpErr)
		}
	}

	// Query latest financial data from database (with YoY comparison)
	// Note: ORDER BY uses CASE to prioritize rows with actual data over NULL rows
	// (there can be duplicate rows where some have NULL margins/ratios)
	query := `
		WITH latest AS (
			SELECT *
			FROM financials
			WHERE ticker = $1
			ORDER BY period_end_date DESC,
			         CASE WHEN gross_margin IS NOT NULL THEN 0 ELSE 1 END,
			         CASE WHEN roe IS NOT NULL THEN 0 ELSE 1 END
			LIMIT 1
		),
		prior_year AS (
			SELECT revenue, eps_diluted, period_end_date
			FROM financials
			WHERE ticker = $1
			  AND period_end_date <= (SELECT period_end_date - INTERVAL '11 months' FROM latest)
			ORDER BY period_end_date DESC
			LIMIT 1
		)
		SELECT
			l.ticker, l.period_end_date, l.fiscal_year, l.fiscal_quarter,
			l.gross_margin, l.operating_margin, l.net_margin,
			l.roe, l.roa, l.debt_to_equity, l.current_ratio, l.quick_ratio,
			l.pe_ratio, l.pb_ratio, l.ps_ratio, l.shares_outstanding, l.statement_type,
			l.revenue as current_revenue, l.eps_diluted as current_eps,
			p.revenue as prior_revenue, p.eps_diluted as prior_eps
		FROM latest l
		LEFT JOIN prior_year p ON TRUE
	`

	var result struct {
		Ticker            string   `db:"ticker"`
		PeriodEndDate     *string  `db:"period_end_date"`
		FiscalYear        *int     `db:"fiscal_year"`
		FiscalQuarter     *int     `db:"fiscal_quarter"`
		GrossMargin       *float64 `db:"gross_margin"`
		OperatingMargin   *float64 `db:"operating_margin"`
		NetMargin         *float64 `db:"net_margin"`
		ROE               *float64 `db:"roe"`
		ROA               *float64 `db:"roa"`
		DebtToEquity      *float64 `db:"debt_to_equity"`
		CurrentRatio      *float64 `db:"current_ratio"`
		QuickRatio        *float64 `db:"quick_ratio"`
		PERatio           *float64 `db:"pe_ratio"`
		PBRatio           *float64 `db:"pb_ratio"`
		PSRatio           *float64 `db:"ps_ratio"`
		SharesOutstanding *int64   `db:"shares_outstanding"`
		StatementType     *string  `db:"statement_type"`
		CurrentRevenue    *int64   `db:"current_revenue"`
		CurrentEPS        *float64 `db:"current_eps"`
		PriorRevenue      *int64   `db:"prior_revenue"`
		PriorEPS          *float64 `db:"prior_eps"`
	}

	err := database.DB.Get(&result, query, ticker)
	dbHasData := err == nil

	// If both FMP and DB failed, return error
	if fmpData == nil && !dbHasData {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Financial data not found",
				"message": fmt.Sprintf("No financial data available for %s", ticker),
				"ticker":  ticker,
			})
			return
		}
		log.Printf("Error fetching financial metrics for %s: %v", ticker, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch financial metrics",
			"message": "An error occurred while retrieving financial data",
		})
		return
	}

	// Merge FMP + DB data (FMP as primary, DB as fallback)
	merged := services.MergeWithDBData(
		fmpData,
		result.GrossMargin, result.OperatingMargin, result.NetMargin,
		result.ROE, result.ROA,
		result.DebtToEquity, result.CurrentRatio, result.QuickRatio,
		result.PERatio, result.PBRatio, result.PSRatio,
	)

	// Calculate YoY growth rates (always from DB)
	var revenueGrowthYoY *float64
	var earningsGrowthYoY *float64

	if result.PriorRevenue != nil && result.CurrentRevenue != nil && *result.PriorRevenue > 0 {
		growth := (float64(*result.CurrentRevenue) - float64(*result.PriorRevenue)) / float64(*result.PriorRevenue) * 100
		revenueGrowthYoY = &growth
	}

	if result.PriorEPS != nil && result.CurrentEPS != nil && *result.PriorEPS != 0 {
		growth := (*result.CurrentEPS - *result.PriorEPS) / abs(*result.PriorEPS) * 100
		earningsGrowthYoY = &growth
	}

	// Determine data source for logging/debugging
	dataSource := "database"
	if merged.FMPAvailable {
		dataSource = "fmp+database"
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"ticker":              ticker,
			"period_end_date":     result.PeriodEndDate,
			"fiscal_year":         result.FiscalYear,
			"fiscal_quarter":      result.FiscalQuarter,
			"gross_margin":        merged.GrossMargin,
			"operating_margin":    merged.OperatingMargin,
			"net_margin":          merged.NetMargin,
			"roe":                 merged.ROE,
			"roa":                 merged.ROA,
			"debt_to_equity":      merged.DebtToEquity,
			"current_ratio":       merged.CurrentRatio,
			"quick_ratio":         merged.QuickRatio,
			"pe_ratio":            merged.PERatio,
			"pb_ratio":            merged.PBRatio,
			"ps_ratio":            merged.PSRatio,
			"revenue_growth_yoy":  revenueGrowthYoY,
			"earnings_growth_yoy": earningsGrowthYoY,
			"shares_outstanding":  result.SharesOutstanding,
			"statement_type":      result.StatementType,
		},
		"meta": gin.H{
			"ticker":      ticker,
			"data_source": dataSource,
		},
		"debug": gin.H{
			"sources": merged.Sources,
		},
	})
}

// Helper function for absolute value
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// GetComprehensiveFinancialMetrics retrieves all financial metrics for a ticker
// Uses FMP API endpoints (ratios-ttm, key-metrics-ttm, financial-growth, analyst-estimates, score)
// GET /api/v1/stocks/:ticker/metrics
func GetComprehensiveFinancialMetrics(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))

	if ticker == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticker symbol is required"})
		return
	}

	// Get current stock price for Forward P/E and Forward Dividend Yield calculations
	var currentPrice float64 = 0
	if database.DB != nil {
		var priceResult struct {
			Price *float64 `db:"current_price"`
		}
		priceQuery := `SELECT current_price FROM tickers WHERE symbol = $1 AND active = true`
		if err := database.DB.Get(&priceResult, priceQuery, ticker); err == nil && priceResult.Price != nil {
			currentPrice = *priceResult.Price
		}
	}

	// Fallback: fetch real-time price from Polygon API if database value is NULL
	if currentPrice == 0 && polygonClient != nil {
		log.Printf("Database current_price is NULL for %s, fetching real-time price from Polygon API", ticker)
		if priceData, err := polygonClient.GetStockRealTimePrice(ticker); err == nil && priceData != nil {
			currentPriceFloat, _ := priceData.Price.Float64()
			currentPrice = currentPriceFloat
			log.Printf("✓ Fetched real-time price for %s from Polygon API: $%.2f", ticker, currentPrice)
		} else {
			log.Printf("⚠️ Failed to fetch real-time price from Polygon API for %s: %v", ticker, err)
		}
	}

	// Final fallback: derive current price from P/E ratio and EPS if all other methods failed
	// We'll set this after getting FMP data if currentPrice is still 0

	// Fetch all FMP data in parallel
	var allMetrics *services.FMPAllMetrics
	if fmpClient != nil && fmpClient.APIKey != "" {
		allMetrics = fmpClient.GetAllMetrics(ticker)

		// Log any errors for debugging
		for endpoint, err := range allMetrics.Errors {
			log.Printf("FMP %s error for %s: %v", endpoint, ticker, err)
		}
	}

	// Merge all FMP data
	merged := services.MergeAllData(allMetrics, currentPrice)

	// If current price is still 0, try to derive it from P/E ratio and EPS
	if currentPrice == 0 && merged.PERatio != nil && *merged.PERatio > 0 && merged.EPSDiluted != nil && *merged.EPSDiluted > 0 {
		currentPrice = *merged.PERatio * *merged.EPSDiluted
		log.Printf("Derived current price for %s from P/E (%.2f) × EPS (%.2f) = $%.2f", ticker, *merged.PERatio, *merged.EPSDiluted, currentPrice)
	}

	// Fetch database fallbacks for missing metrics from fundamental_metrics_extended
	// Note: The database stores percentages as percentage values (e.g., 31.18 for 31.18%), not decimals
	if database.DB != nil {
		var dbFallback struct {
			// Profitability
			GrossMargin     *float64 `db:"gross_margin"`
			OperatingMargin *float64 `db:"operating_margin"`
			NetMargin       *float64 `db:"net_margin"`
			EBITDAMargin    *float64 `db:"ebitda_margin"`
			ROE             *float64 `db:"roe"`
			ROA             *float64 `db:"roa"`
			ROIC            *float64 `db:"roic"`
			// Growth
			RevenueGrowthYoY    *float64 `db:"revenue_growth_yoy"`
			RevenueGrowth3YCAGR *float64 `db:"revenue_growth_3y_cagr"`
			RevenueGrowth5YCAGR *float64 `db:"revenue_growth_5y_cagr"`
			EPSGrowthYoY        *float64 `db:"eps_growth_yoy"`
			EPSGrowth3YCAGR     *float64 `db:"eps_growth_3y_cagr"`
			EPSGrowth5YCAGR     *float64 `db:"eps_growth_5y_cagr"`
			FCFGrowthYoY        *float64 `db:"fcf_growth_yoy"`
			// Valuation
			EnterpriseValue *float64 `db:"enterprise_value"`
			EVToRevenue     *float64 `db:"ev_to_revenue"`
			EVToEBITDA      *float64 `db:"ev_to_ebitda"`
			EVToFCF         *float64 `db:"ev_to_fcf"`
			// Liquidity
			CurrentRatio *float64 `db:"current_ratio"`
			QuickRatio   *float64 `db:"quick_ratio"`
			// Leverage
			DebtToEquity     *float64 `db:"debt_to_equity"`
			InterestCoverage *float64 `db:"interest_coverage"`
			NetDebtToEBITDA  *float64 `db:"net_debt_to_ebitda"`
			// Dividends
			DividendYield            *float64 `db:"dividend_yield"`
			PayoutRatio              *float64 `db:"payout_ratio"`
			ConsecutiveDividendYears *int     `db:"consecutive_dividend_years"`
		}
		fallbackQuery := `
			SELECT m.gross_margin, m.operating_margin, m.net_margin, m.ebitda_margin,
			       m.roe, m.roa, m.roic,
			       m.revenue_growth_yoy, m.revenue_growth_3y_cagr, m.revenue_growth_5y_cagr,
			       m.eps_growth_yoy, m.eps_growth_3y_cagr, m.eps_growth_5y_cagr, m.fcf_growth_yoy,
			       m.enterprise_value, m.ev_to_revenue, m.ev_to_ebitda, m.ev_to_fcf,
			       m.current_ratio, m.quick_ratio,
			       m.debt_to_equity, m.interest_coverage, m.net_debt_to_ebitda,
			       m.dividend_yield, m.payout_ratio, m.consecutive_dividend_years
			FROM fundamental_metrics_extended m
			WHERE m.ticker = $1
			ORDER BY m.calculation_date DESC
			LIMIT 1
		`
		if err := database.DB.Get(&dbFallback, fallbackQuery, ticker); err == nil {
			// Profitability fallbacks
			if merged.GrossMargin == nil && dbFallback.GrossMargin != nil {
				merged.GrossMargin = dbFallback.GrossMargin
				merged.Sources.GrossMargin = services.SourceDatabase
			}
			if merged.OperatingMargin == nil && dbFallback.OperatingMargin != nil {
				merged.OperatingMargin = dbFallback.OperatingMargin
				merged.Sources.OperatingMargin = services.SourceDatabase
			}
			if merged.NetMargin == nil && dbFallback.NetMargin != nil {
				merged.NetMargin = dbFallback.NetMargin
				merged.Sources.NetMargin = services.SourceDatabase
			}
			if merged.EBITDAMargin == nil && dbFallback.EBITDAMargin != nil {
				merged.EBITDAMargin = dbFallback.EBITDAMargin
				merged.Sources.EBITDAMargin = services.SourceDatabase
			}
			if merged.ROE == nil && dbFallback.ROE != nil {
				merged.ROE = dbFallback.ROE
				merged.Sources.ROE = services.SourceDatabase
			}
			if merged.ROA == nil && dbFallback.ROA != nil {
				merged.ROA = dbFallback.ROA
				merged.Sources.ROA = services.SourceDatabase
			}
			if merged.ROIC == nil && dbFallback.ROIC != nil {
				merged.ROIC = dbFallback.ROIC
				merged.Sources.ROIC = services.SourceDatabase
			}
			// Growth fallbacks
			if merged.RevenueGrowthYoY == nil && dbFallback.RevenueGrowthYoY != nil {
				merged.RevenueGrowthYoY = dbFallback.RevenueGrowthYoY
				merged.Sources.RevenueGrowthYoY = services.SourceDatabase
			}
			if merged.RevenueGrowth3YCAGR == nil && dbFallback.RevenueGrowth3YCAGR != nil {
				merged.RevenueGrowth3YCAGR = dbFallback.RevenueGrowth3YCAGR
				merged.Sources.RevenueGrowth3Y = services.SourceDatabase
			}
			if merged.RevenueGrowth5YCAGR == nil && dbFallback.RevenueGrowth5YCAGR != nil {
				merged.RevenueGrowth5YCAGR = dbFallback.RevenueGrowth5YCAGR
				merged.Sources.RevenueGrowth5Y = services.SourceDatabase
			}
			if merged.EPSGrowthYoY == nil && dbFallback.EPSGrowthYoY != nil {
				merged.EPSGrowthYoY = dbFallback.EPSGrowthYoY
				merged.Sources.EPSGrowthYoY = services.SourceDatabase
			}
			if merged.EPSGrowth3YCAGR == nil && dbFallback.EPSGrowth3YCAGR != nil {
				merged.EPSGrowth3YCAGR = dbFallback.EPSGrowth3YCAGR
			}
			if merged.EPSGrowth5YCAGR == nil && dbFallback.EPSGrowth5YCAGR != nil {
				merged.EPSGrowth5YCAGR = dbFallback.EPSGrowth5YCAGR
				merged.Sources.EPSGrowth5Y = services.SourceDatabase
			}
			if merged.FCFGrowthYoY == nil && dbFallback.FCFGrowthYoY != nil {
				merged.FCFGrowthYoY = dbFallback.FCFGrowthYoY
			}
			// Valuation fallbacks
			if merged.EnterpriseValue == nil && dbFallback.EnterpriseValue != nil {
				merged.EnterpriseValue = dbFallback.EnterpriseValue
			}
			if merged.EVToSales == nil && dbFallback.EVToRevenue != nil {
				merged.EVToSales = dbFallback.EVToRevenue
				merged.Sources.EVToSales = services.SourceDatabase
			}
			if merged.EVToEBITDA == nil && dbFallback.EVToEBITDA != nil {
				merged.EVToEBITDA = dbFallback.EVToEBITDA
				merged.Sources.EVToEBITDA = services.SourceDatabase
			}
			if merged.EVToFCF == nil && dbFallback.EVToFCF != nil {
				merged.EVToFCF = dbFallback.EVToFCF
				merged.Sources.EVToFCF = services.SourceDatabase
			}
			// Liquidity fallbacks
			if merged.CurrentRatio == nil && dbFallback.CurrentRatio != nil {
				merged.CurrentRatio = dbFallback.CurrentRatio
				merged.Sources.CurrentRatio = services.SourceDatabase
			}
			if merged.QuickRatio == nil && dbFallback.QuickRatio != nil {
				merged.QuickRatio = dbFallback.QuickRatio
				merged.Sources.QuickRatio = services.SourceDatabase
			}
			// Leverage fallbacks
			if merged.DebtToEquity == nil && dbFallback.DebtToEquity != nil {
				merged.DebtToEquity = dbFallback.DebtToEquity
				merged.Sources.DebtToEquity = services.SourceDatabase
			}
			if merged.InterestCoverage == nil && dbFallback.InterestCoverage != nil {
				merged.InterestCoverage = dbFallback.InterestCoverage
				merged.Sources.InterestCoverage = services.SourceDatabase
			}
			if merged.NetDebtToEBITDA == nil && dbFallback.NetDebtToEBITDA != nil {
				merged.NetDebtToEBITDA = dbFallback.NetDebtToEBITDA
			}
			// Dividend fallbacks
			if merged.DividendYield == nil && dbFallback.DividendYield != nil {
				merged.DividendYield = dbFallback.DividendYield
				merged.Sources.DividendYield = services.SourceDatabase
			}
			if merged.PayoutRatio == nil && dbFallback.PayoutRatio != nil {
				merged.PayoutRatio = dbFallback.PayoutRatio
				merged.Sources.PayoutRatio = services.SourceDatabase
			}
			if merged.ConsecutiveDividendYears == nil && dbFallback.ConsecutiveDividendYears != nil {
				merged.ConsecutiveDividendYears = dbFallback.ConsecutiveDividendYears
			}
		}

		// Fetch Market Cap from tickers table if not available from FMP
		if merged.MarketCap == nil {
			var marketCapResult struct {
				MarketCap *float64 `db:"market_cap"`
			}
			marketCapQuery := `SELECT market_cap FROM tickers WHERE symbol = $1 AND active = true`
			if err := database.DB.Get(&marketCapResult, marketCapQuery, ticker); err == nil && marketCapResult.MarketCap != nil {
				merged.MarketCap = marketCapResult.MarketCap
				merged.Sources.MarketCap = services.SourceDatabase
			}
		}
	}

	// Calculate derived metrics as fallbacks if primary sources are missing

	// PEG Ratio: P/E / EPS Growth Rate (use 5Y EPS CAGR if available)
	if merged.PEGRatio == nil && merged.PERatio != nil && *merged.PERatio > 0 {
		// Try using 5-year EPS CAGR first, then YoY
		var epsGrowth *float64
		if merged.EPSGrowth5YCAGR != nil && *merged.EPSGrowth5YCAGR > 0 {
			epsGrowth = merged.EPSGrowth5YCAGR
		} else if merged.EPSGrowthYoY != nil && *merged.EPSGrowthYoY > 0 {
			epsGrowth = merged.EPSGrowthYoY
		}
		if epsGrowth != nil {
			pegRatio := *merged.PERatio / *epsGrowth
			merged.PEGRatio = &pegRatio
			merged.Sources.PEGRatio = services.SourceCalculated
			// Add PEG interpretation
			interp, _ := services.GetPEGInterpretation(pegRatio)
			merged.PEGInterpretation = &interp
		}
	}

	// Earnings Yield: inverse of P/E ratio (1 / PE * 100)
	if merged.EarningsYield == nil && merged.PERatio != nil && *merged.PERatio > 0 {
		earningsYield := (1.0 / *merged.PERatio) * 100
		merged.EarningsYield = &earningsYield
		merged.Sources.EarningsYield = services.SourceCalculated
	}

	// FCF Yield: inverse of Price to FCF ratio (1 / P/FCF * 100)
	if merged.FCFYield == nil && merged.PriceToFCF != nil && *merged.PriceToFCF > 0 {
		fcfYield := (1.0 / *merged.PriceToFCF) * 100
		merged.FCFYield = &fcfYield
		merged.Sources.FCFYield = services.SourceCalculated
	}

	// Forward P/E calculated fallback: Use projected EPS from historical growth
	if merged.ForwardPE == nil && merged.PERatio != nil && *merged.PERatio > 0 && merged.EPSDiluted != nil && *merged.EPSDiluted > 0 {
		// Use EPS growth rate to project next year's EPS
		var epsGrowthRate float64 = 0.10 // Default 10% growth assumption
		if merged.EPSGrowth5YCAGR != nil && *merged.EPSGrowth5YCAGR > 0 {
			epsGrowthRate = *merged.EPSGrowth5YCAGR / 100 // Convert percentage to decimal
		} else if merged.EPSGrowthYoY != nil && *merged.EPSGrowthYoY > 0 {
			epsGrowthRate = *merged.EPSGrowthYoY / 100
		}

		// Project next year's EPS: CurrentEPS * (1 + growth rate)
		projectedEPS := *merged.EPSDiluted * (1 + epsGrowthRate)

		// Calculate Forward P/E from current price and projected EPS
		if currentPrice > 0 && projectedEPS > 0 {
			forwardPE := currentPrice / projectedEPS
			merged.ForwardPE = &forwardPE
			merged.Sources.ForwardPE = services.SourceCalculated
		}
	}

	// EV/EBITDA calculated fallback: Derive EBITDA from available metrics
	if merged.EVToEBITDA == nil && merged.EnterpriseValue != nil && *merged.EnterpriseValue > 0 {
		var ebitda *float64

		// Method 1: Calculate from Revenue * EBITDA Margin
		if merged.RevenuePerShare != nil && merged.EBITDAMargin != nil && *merged.RevenuePerShare > 0 && *merged.EBITDAMargin > 0 {
			// EBITDA = Revenue * EBITDA Margin / 100
			// Need to multiply by shares outstanding, but we can use Market Cap / Price as proxy
			if merged.MarketCap != nil && *merged.MarketCap > 0 && currentPrice > 0 {
				sharesOutstanding := *merged.MarketCap / currentPrice
				revenue := *merged.RevenuePerShare * sharesOutstanding
				ebitdaValue := revenue * (*merged.EBITDAMargin / 100)
				ebitda = &ebitdaValue
			}
		}

		// Method 2: Calculate from Net Income and margins if Method 1 failed
		if ebitda == nil && merged.NetMargin != nil && merged.EBITDAMargin != nil &&
			*merged.NetMargin > 0 && *merged.EBITDAMargin > 0 &&
			merged.RevenuePerShare != nil && *merged.RevenuePerShare > 0 {
			if merged.MarketCap != nil && *merged.MarketCap > 0 && currentPrice > 0 {
				sharesOutstanding := *merged.MarketCap / currentPrice
				revenue := *merged.RevenuePerShare * sharesOutstanding
				ebitdaValue := revenue * (*merged.EBITDAMargin / 100)
				ebitda = &ebitdaValue
			}
		}

		// Calculate EV/EBITDA if we derived EBITDA
		if ebitda != nil && *ebitda > 0 {
			evToEBITDA := *merged.EnterpriseValue / *ebitda
			merged.EVToEBITDA = &evToEBITDA
			merged.Sources.EVToEBITDA = services.SourceCalculated
		}
	}

	// EV/Sales calculated fallback: EV / Revenue (using Market Cap as proxy when EV is available)
	if merged.EVToSales == nil && merged.EnterpriseValue != nil && merged.PSRatio != nil && merged.MarketCap != nil && *merged.MarketCap > 0 {
		// EV/Sales = EV / Revenue, and P/S = Price / Revenue per share = Market Cap / Revenue
		// So Revenue = Market Cap / P/S, and EV/Sales = EV / (Market Cap / P/S) = EV * P/S / Market Cap
		evToSales := (*merged.EnterpriseValue * *merged.PSRatio) / *merged.MarketCap
		merged.EVToSales = &evToSales
		merged.Sources.EVToSales = services.SourceCalculated
	}

	// If no data available at all, return error
	if !merged.FMPAvailable && allMetrics != nil && len(allMetrics.Errors) == 6 {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Financial data not found",
			"message": fmt.Sprintf("No financial data available for %s from FMP", ticker),
			"ticker":  ticker,
		})
		return
	}

	// Build response with all metrics organized by category
	response := gin.H{
		// === VALUATION ===
		"valuation": gin.H{
			"pe_ratio":           merged.PERatio,
			"forward_pe":         merged.ForwardPE,
			"pb_ratio":           merged.PBRatio,
			"ps_ratio":           merged.PSRatio,
			"price_to_fcf":       merged.PriceToFCF,
			"price_to_ocf":       merged.PriceToOCF,
			"peg_ratio":          merged.PEGRatio,
			"peg_interpretation": merged.PEGInterpretation,
			"enterprise_value":   merged.EnterpriseValue,
			"ev_to_sales":        merged.EVToSales,
			"ev_to_ebitda":       merged.EVToEBITDA,
			"ev_to_ebit":         merged.EVToEBIT,
			"ev_to_fcf":          merged.EVToFCF,
			"earnings_yield":     merged.EarningsYield,
			"fcf_yield":          merged.FCFYield,
			"market_cap":         merged.MarketCap,
		},

		// === PROFITABILITY ===
		"profitability": gin.H{
			"gross_margin":     merged.GrossMargin,
			"operating_margin": merged.OperatingMargin,
			"net_margin":       merged.NetMargin,
			"ebitda_margin":    merged.EBITDAMargin,
			"ebit_margin":      merged.EBITMargin,
			"fcf_margin":       merged.FCFMargin,
			"pretax_margin":    merged.PretaxMargin,
			"roe":              merged.ROE,
			"roa":              merged.ROA,
			"roic":             merged.ROIC,
			"roce":             merged.ROCE,
		},

		// === LIQUIDITY ===
		"liquidity": gin.H{
			"current_ratio":   merged.CurrentRatio,
			"quick_ratio":     merged.QuickRatio,
			"cash_ratio":      merged.CashRatio,
			"working_capital": merged.WorkingCapital,
		},

		// === LEVERAGE ===
		"leverage": gin.H{
			"debt_to_equity":     merged.DebtToEquity,
			"debt_to_assets":     merged.DebtToAssets,
			"debt_to_ebitda":     merged.DebtToEBITDA,
			"debt_to_capital":    merged.DebtToCapital,
			"interest_coverage":  merged.InterestCoverage,
			"net_debt_to_ebitda": merged.NetDebtToEBITDA,
			"net_debt":           merged.NetDebt,
		},

		// === EFFICIENCY ===
		"efficiency": gin.H{
			"asset_turnover":             merged.AssetTurnover,
			"inventory_turnover":         merged.InventoryTurnover,
			"receivables_turnover":       merged.ReceivablesTurnover,
			"payables_turnover":          merged.PayablesTurnover,
			"fixed_asset_turnover":       merged.FixedAssetTurnover,
			"days_sales_outstanding":     merged.DaysOfSalesOutstanding,
			"days_inventory_outstanding": merged.DaysOfInventoryOutstanding,
			"days_payables_outstanding":  merged.DaysOfPayablesOutstanding,
			"cash_conversion_cycle":      merged.CashConversionCycle,
		},

		// === GROWTH ===
		"growth": gin.H{
			"revenue_growth_yoy":          merged.RevenueGrowthYoY,
			"revenue_growth_3y_cagr":      merged.RevenueGrowth3YCAGR,
			"revenue_growth_5y_cagr":      merged.RevenueGrowth5YCAGR,
			"gross_profit_growth_yoy":     merged.GrossProfitGrowthYoY,
			"operating_income_growth_yoy": merged.OperatingIncomeGrowthYoY,
			"net_income_growth_yoy":       merged.NetIncomeGrowthYoY,
			"eps_growth_yoy":              merged.EPSGrowthYoY,
			"eps_growth_3y_cagr":          merged.EPSGrowth3YCAGR,
			"eps_growth_5y_cagr":          merged.EPSGrowth5YCAGR,
			"fcf_growth_yoy":              merged.FCFGrowthYoY,
			"book_value_growth_yoy":       merged.BookValueGrowthYoY,
			"dividend_growth_5y_cagr":     merged.DividendGrowth5YCAGR,
		},

		// === PER SHARE ===
		"per_share": gin.H{
			"eps_diluted":             merged.EPSDiluted,
			"book_value_per_share":    merged.BookValuePerShare,
			"tangible_book_per_share": merged.TangibleBookPerShare,
			"revenue_per_share":       merged.RevenuePerShare,
			"operating_cf_per_share":  merged.OperatingCFPerShare,
			"fcf_per_share":           merged.FCFPerShare,
			"cash_per_share":          merged.CashPerShare,
			"dividend_per_share":      merged.DividendPerShare,
			"graham_number":           merged.GrahamNumber,
		},

		// === DIVIDENDS ===
		"dividends": gin.H{
			"dividend_yield":             merged.DividendYield,
			"forward_dividend_yield":     merged.ForwardDividendYield,
			"payout_ratio":               merged.PayoutRatio,
			"payout_interpretation":      merged.PayoutInterpretation,
			"fcf_payout_ratio":           merged.FCFPayoutRatio,
			"consecutive_dividend_years": merged.ConsecutiveDividendYears,
			"ex_dividend_date":           merged.ExDividendDate,
			"payment_date":               merged.PaymentDate,
			"dividend_frequency":         merged.DividendFrequency,
		},

		// === QUALITY SCORES ===
		"quality_scores": gin.H{
			"altman_z_score":             merged.AltmanZScore,
			"altman_z_interpretation":    merged.AltmanZInterpretation,
			"altman_z_description":       merged.AltmanZDescription,
			"piotroski_f_score":          merged.PiotroskiFScore,
			"piotroski_f_interpretation": merged.PiotroskiFInterpretation,
			"piotroski_f_description":    merged.PiotroskiFDescription,
		},

		// === FORWARD ESTIMATES ===
		"forward_estimates": gin.H{
			"forward_eps":          merged.ForwardEPS,
			"forward_eps_high":     merged.ForwardEPSHigh,
			"forward_eps_low":      merged.ForwardEPSLow,
			"forward_revenue":      merged.ForwardRevenue,
			"forward_ebitda":       merged.ForwardEBITDA,
			"forward_net_income":   merged.ForwardNetIncome,
			"num_analysts_eps":     merged.NumAnalystsEPS,
			"num_analysts_revenue": merged.NumAnalystsRevenue,
		},
	}

	// Collect errors for debugging
	var errors []string
	if allMetrics != nil {
		for endpoint, err := range allMetrics.Errors {
			errors = append(errors, fmt.Sprintf("%s: %v", endpoint, err))
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response,
		"meta": gin.H{
			"ticker":        ticker,
			"fmp_available": merged.FMPAvailable,
			"current_price": currentPrice,
		},
		"debug": gin.H{
			"sources": merged.Sources,
			"errors":  errors,
		},
	})
}

// GetRiskMetrics retrieves risk metrics for a ticker from the risk_metrics table
// GET /api/v1/stocks/:ticker/risk?period=1Y
func GetRiskMetrics(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))
	period := c.DefaultQuery("period", "1Y")

	if ticker == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticker symbol is required"})
		return
	}

	// Check database connection
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Risk metrics service is temporarily unavailable",
		})
		return
	}

	// Query latest risk metrics
	query := `
		SELECT
			time, ticker, period, alpha, beta, sharpe_ratio, sortino_ratio,
			std_dev, max_drawdown, var_5, annualized_return, downside_deviation, data_points
		FROM risk_metrics
		WHERE ticker = $1 AND period = $2
		ORDER BY time DESC
		LIMIT 1
	`

	var result struct {
		Time              *string  `db:"time"`
		Ticker            string   `db:"ticker"`
		Period            string   `db:"period"`
		Alpha             *float64 `db:"alpha"`
		Beta              *float64 `db:"beta"`
		SharpeRatio       *float64 `db:"sharpe_ratio"`
		SortinoRatio      *float64 `db:"sortino_ratio"`
		StdDev            *float64 `db:"std_dev"`
		MaxDrawdown       *float64 `db:"max_drawdown"`
		VaR5              *float64 `db:"var_5"`
		AnnualizedReturn  *float64 `db:"annualized_return"`
		DownsideDeviation *float64 `db:"downside_deviation"`
		DataPoints        *int     `db:"data_points"`
	}

	err := database.DB.Get(&result, query, ticker, period)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Risk metrics not found",
				"message": fmt.Sprintf("No risk metrics available for %s with period %s", ticker, period),
				"ticker":  ticker,
				"period":  period,
			})
			return
		}
		log.Printf("Error fetching risk metrics for %s: %v", ticker, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch risk metrics",
			"message": "An error occurred while retrieving risk data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"ticker":             result.Ticker,
			"period":             result.Period,
			"calculation_date":   result.Time,
			"beta":               result.Beta,
			"alpha":              result.Alpha,
			"sharpe_ratio":       result.SharpeRatio,
			"sortino_ratio":      result.SortinoRatio,
			"volatility":         result.StdDev,
			"max_drawdown":       result.MaxDrawdown,
			"var_95":             result.VaR5,
			"annualized_return":  result.AnnualizedReturn,
			"downside_deviation": result.DownsideDeviation,
			"data_points":        result.DataPoints,
		},
		"meta": gin.H{
			"ticker": ticker,
			"period": period,
		},
	})
}

// GetTechnicalIndicators retrieves technical indicators for a ticker
// GET /api/v1/stocks/:ticker/technical
func GetTechnicalIndicators(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))

	if ticker == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticker symbol is required"})
		return
	}

	// Check database connection
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Technical indicators service is temporarily unavailable",
		})
		return
	}

	// Query latest technical indicators using pivot
	// The technical_indicators table uses indicator_name + value format
	query := `
		SELECT
			MAX(time) as calculation_date,
			ticker,
			MAX(CASE WHEN indicator_name = 'current_price' THEN value END) as current_price,
			MAX(CASE WHEN indicator_name = 'sma_50' THEN value END) as sma_50,
			MAX(CASE WHEN indicator_name = 'sma_200' THEN value END) as sma_200,
			MAX(CASE WHEN indicator_name = 'ema_12' THEN value END) as ema_12,
			MAX(CASE WHEN indicator_name = 'ema_26' THEN value END) as ema_26,
			MAX(CASE WHEN indicator_name = 'rsi' THEN value END) as rsi_14,
			MAX(CASE WHEN indicator_name = 'macd' THEN value END) as macd,
			MAX(CASE WHEN indicator_name = 'macd_signal' THEN value END) as macd_signal,
			MAX(CASE WHEN indicator_name = 'macd_histogram' THEN value END) as macd_histogram,
			MAX(CASE WHEN indicator_name = 'bb_upper' THEN value END) as bb_upper,
			MAX(CASE WHEN indicator_name = 'bb_middle' THEN value END) as bb_middle,
			MAX(CASE WHEN indicator_name = 'bb_lower' THEN value END) as bb_lower,
			MAX(CASE WHEN indicator_name = 'volume_ma_20' THEN value END) as volume_ma_20,
			MAX(CASE WHEN indicator_name = '1m_return' THEN value END) as return_1m,
			MAX(CASE WHEN indicator_name = '3m_return' THEN value END) as return_3m,
			MAX(CASE WHEN indicator_name = '6m_return' THEN value END) as return_6m,
			MAX(CASE WHEN indicator_name = '12m_return' THEN value END) as return_12m
		FROM technical_indicators
		WHERE ticker = $1
		GROUP BY ticker
	`

	var result struct {
		CalculationDate *string  `db:"calculation_date"`
		Ticker          string   `db:"ticker"`
		CurrentPrice    *float64 `db:"current_price"`
		SMA50           *float64 `db:"sma_50"`
		SMA200          *float64 `db:"sma_200"`
		EMA12           *float64 `db:"ema_12"`
		EMA26           *float64 `db:"ema_26"`
		RSI14           *float64 `db:"rsi_14"`
		MACD            *float64 `db:"macd"`
		MACDSignal      *float64 `db:"macd_signal"`
		MACDHistogram   *float64 `db:"macd_histogram"`
		BBUpper         *float64 `db:"bb_upper"`
		BBMiddle        *float64 `db:"bb_middle"`
		BBLower         *float64 `db:"bb_lower"`
		VolumeMA20      *float64 `db:"volume_ma_20"`
		Return1M        *float64 `db:"return_1m"`
		Return3M        *float64 `db:"return_3m"`
		Return6M        *float64 `db:"return_6m"`
		Return12M       *float64 `db:"return_12m"`
	}

	err := database.DB.Get(&result, query, ticker)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Technical indicators not found",
				"message": fmt.Sprintf("No technical indicators available for %s", ticker),
				"ticker":  ticker,
			})
			return
		}
		log.Printf("Error fetching technical indicators for %s: %v", ticker, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch technical indicators",
			"message": "An error occurred while retrieving technical data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"ticker":           result.Ticker,
			"calculation_date": result.CalculationDate,
			"current_price":    result.CurrentPrice,
			"sma_50":           result.SMA50,
			"sma_200":          result.SMA200,
			"ema_12":           result.EMA12,
			"ema_26":           result.EMA26,
			"rsi_14":           result.RSI14,
			"macd":             result.MACD,
			"macd_signal":      result.MACDSignal,
			"macd_histogram":   result.MACDHistogram,
			"bollinger_upper":  result.BBUpper,
			"bollinger_middle": result.BBMiddle,
			"bollinger_lower":  result.BBLower,
			"volume_ma_20":     result.VolumeMA20,
			"return_1m":        result.Return1M,
			"return_3m":        result.Return3M,
			"return_6m":        result.Return6M,
			"return_12m":       result.Return12M,
		},
		"meta": gin.H{
			"ticker": ticker,
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
