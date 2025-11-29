package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"investorcenter-api/database"
	"investorcenter-api/models"
	"investorcenter-api/services"

	"github.com/gin-gonic/gin"
)

// FinancialsHandler handles financial statement API requests
type FinancialsHandler struct {
	service *services.FinancialsService
}

// NewFinancialsHandler creates a new financials handler
func NewFinancialsHandler() *FinancialsHandler {
	return &FinancialsHandler{
		service: services.NewFinancialsService(),
	}
}

// parseFinancialsParams extracts query parameters for financials endpoints
func parseFinancialsParams(c *gin.Context) (timeframe models.Timeframe, limit int, fiscalYear *int, sort string) {
	// Timeframe: quarterly (default), annual, ttm
	tf := c.DefaultQuery("timeframe", "quarterly")
	switch strings.ToLower(tf) {
	case "annual":
		timeframe = models.TimeframeAnnual
	case "ttm", "trailing_twelve_months":
		timeframe = models.TimeframeTTM
	default:
		timeframe = models.TimeframeQuarterly
	}

	// Limit: default 8, max 40
	limit = 8
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		if l > 40 {
			l = 40
		}
		limit = l
	}

	// Fiscal year filter
	if fy, err := strconv.Atoi(c.Query("fiscal_year")); err == nil {
		fiscalYear = &fy
	}

	// Sort order: asc or desc (default)
	sort = c.DefaultQuery("sort", "desc")
	if sort != "asc" {
		sort = "desc"
	}

	return
}

// GetIncomeStatements handles GET /api/v1/stocks/:ticker/financials/income
func (h *FinancialsHandler) GetIncomeStatements(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))

	// Check database connection
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Financial statements service is temporarily unavailable",
		})
		return
	}

	timeframe, limit, _, _ := parseFinancialsParams(c)

	response, err := h.service.GetIncomeStatements(c.Request.Context(), ticker, timeframe, limit)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Financial data not found",
			"message": err.Error(),
			"ticker":  ticker,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response,
		"meta": gin.H{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// GetBalanceSheets handles GET /api/v1/stocks/:ticker/financials/balance
func (h *FinancialsHandler) GetBalanceSheets(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))

	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Financial statements service is temporarily unavailable",
		})
		return
	}

	timeframe, limit, _, _ := parseFinancialsParams(c)

	response, err := h.service.GetBalanceSheets(c.Request.Context(), ticker, timeframe, limit)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Financial data not found",
			"message": err.Error(),
			"ticker":  ticker,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response,
		"meta": gin.H{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// GetCashFlowStatements handles GET /api/v1/stocks/:ticker/financials/cashflow
func (h *FinancialsHandler) GetCashFlowStatements(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))

	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Financial statements service is temporarily unavailable",
		})
		return
	}

	timeframe, limit, _, _ := parseFinancialsParams(c)

	response, err := h.service.GetCashFlowStatements(c.Request.Context(), ticker, timeframe, limit)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Financial data not found",
			"message": err.Error(),
			"ticker":  ticker,
		})
		return
	}

	// Enrich cash flow data with calculated fields
	for i := range response.Periods {
		response.Periods[i].Data = services.EnrichCashFlowData(response.Periods[i].Data)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response,
		"meta": gin.H{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// GetRatios handles GET /api/v1/stocks/:ticker/financials/ratios
func (h *FinancialsHandler) GetRatios(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))

	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Financial statements service is temporarily unavailable",
		})
		return
	}

	timeframe, limit, _, _ := parseFinancialsParams(c)

	response, err := h.service.GetRatios(c.Request.Context(), ticker, timeframe, limit)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Financial data not found",
			"message": err.Error(),
			"ticker":  ticker,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response,
		"meta": gin.H{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// RefreshFinancials handles POST /api/v1/stocks/:ticker/financials/refresh
func (h *FinancialsHandler) RefreshFinancials(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))

	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Financial statements service is temporarily unavailable",
		})
		return
	}

	err := h.service.RefreshFinancials(c.Request.Context(), ticker)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to refresh financial data",
			"message": err.Error(),
			"ticker":  ticker,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Financial data refreshed successfully",
		"ticker":  ticker,
		"meta": gin.H{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// GetAllFinancials handles GET /api/v1/stocks/:ticker/financials
// Returns a summary of all available financial statements
func (h *FinancialsHandler) GetAllFinancials(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))

	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Financial statements service is temporarily unavailable",
		})
		return
	}

	timeframe, limit, _, _ := parseFinancialsParams(c)

	// Fetch all three statement types
	income, incomeErr := h.service.GetIncomeStatements(c.Request.Context(), ticker, timeframe, limit)
	balance, balanceErr := h.service.GetBalanceSheets(c.Request.Context(), ticker, timeframe, limit)
	cashflow, cashflowErr := h.service.GetCashFlowStatements(c.Request.Context(), ticker, timeframe, limit)

	// If all failed, return error
	if incomeErr != nil && balanceErr != nil && cashflowErr != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Financial data not found",
			"message": "No financial statements available for this ticker",
			"ticker":  ticker,
		})
		return
	}

	// Enrich cash flow data
	if cashflow != nil {
		for i := range cashflow.Periods {
			cashflow.Periods[i].Data = services.EnrichCashFlowData(cashflow.Periods[i].Data)
		}
	}

	// Get metadata from the first successful response
	var metadata models.FinancialsMetadata
	if income != nil {
		metadata = income.Metadata
	} else if balance != nil {
		metadata = balance.Metadata
	} else if cashflow != nil {
		metadata = cashflow.Metadata
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"ticker":    ticker,
			"timeframe": timeframe,
			"metadata":  metadata,
			"income":    getPeriodsOrNull(income),
			"balance":   getPeriodsOrNull(balance),
			"cashflow":  getPeriodsOrNull(cashflow),
		},
		"meta": gin.H{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// getPeriodsOrNull returns the periods from a response or nil if response is nil
func getPeriodsOrNull(response *models.FinancialsResponse) []models.FinancialPeriod {
	if response == nil {
		return nil
	}
	return response.Periods
}

// ValidateTickerExists checks if a ticker exists in the database
func ValidateTickerExists(ticker string) bool {
	_, err := database.GetTickerIDBySymbol(ticker)
	return err == nil
}
