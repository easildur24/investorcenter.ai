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

// GetFinancialMetrics retrieves financial metrics for a ticker from the financials table
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

	// Query latest financial data with YoY comparison
	query := `
		WITH latest AS (
			SELECT *
			FROM financials
			WHERE ticker = $1
			ORDER BY period_end_date DESC
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
	if err != nil {
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

	// Calculate YoY growth rates
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

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"ticker":              result.Ticker,
			"period_end_date":     result.PeriodEndDate,
			"fiscal_year":         result.FiscalYear,
			"fiscal_quarter":      result.FiscalQuarter,
			"gross_margin":        result.GrossMargin,
			"operating_margin":    result.OperatingMargin,
			"net_margin":          result.NetMargin,
			"roe":                 result.ROE,
			"roa":                 result.ROA,
			"debt_to_equity":      result.DebtToEquity,
			"current_ratio":       result.CurrentRatio,
			"quick_ratio":         result.QuickRatio,
			"pe_ratio":            result.PERatio,
			"pb_ratio":            result.PBRatio,
			"ps_ratio":            result.PSRatio,
			"revenue_growth_yoy":  revenueGrowthYoY,
			"earnings_growth_yoy": earningsGrowthYoY,
			"shares_outstanding":  result.SharesOutstanding,
			"statement_type":      result.StatementType,
		},
		"meta": gin.H{
			"ticker": ticker,
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
