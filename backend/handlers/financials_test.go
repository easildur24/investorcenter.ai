package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"investorcenter-api/database"
	"investorcenter-api/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// parseFinancialsParams — pure function
// ---------------------------------------------------------------------------

func TestParseFinancialsParams_Defaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks/AAPL/financials/income", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	timeframe, limit, fiscalYear, sort := parseFinancialsParams(c)

	assert.Equal(t, models.TimeframeQuarterly, timeframe)
	assert.Equal(t, 8, limit)
	assert.Nil(t, fiscalYear)
	assert.Equal(t, "desc", sort)
}

func TestParseFinancialsParams_AnnualTimeframe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?timeframe=annual", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	timeframe, _, _, _ := parseFinancialsParams(c)
	assert.Equal(t, models.TimeframeAnnual, timeframe)
}

func TestParseFinancialsParams_TTMTimeframe(t *testing.T) {
	tests := []string{"ttm", "trailing_twelve_months"}
	for _, tf := range tests {
		t.Run(tf, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test?timeframe="+tf, nil)
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			timeframe, _, _, _ := parseFinancialsParams(c)
			assert.Equal(t, models.TimeframeTTM, timeframe)
		})
	}
}

func TestParseFinancialsParams_InvalidTimeframeFallsBack(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?timeframe=bogus", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	timeframe, _, _, _ := parseFinancialsParams(c)
	assert.Equal(t, models.TimeframeQuarterly, timeframe)
}

func TestParseFinancialsParams_Limit(t *testing.T) {
	tests := []struct {
		query     string
		wantLimit int
	}{
		{"limit=5", 5},
		{"limit=20", 20},
		{"limit=40", 40},
		{"limit=100", 40}, // capped at 40
		{"limit=0", 8},    // invalid, fallback
		{"limit=-1", 8},   // invalid, fallback
		{"limit=abc", 8},  // non-numeric, fallback
		{"", 8},           // missing, default
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test?"+tt.query, nil)
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			_, limit, _, _ := parseFinancialsParams(c)
			assert.Equal(t, tt.wantLimit, limit)
		})
	}
}

func TestParseFinancialsParams_FiscalYear(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?fiscal_year=2023", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	_, _, fiscalYear, _ := parseFinancialsParams(c)
	assert.NotNil(t, fiscalYear)
	assert.Equal(t, 2023, *fiscalYear)
}

func TestParseFinancialsParams_FiscalYearInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?fiscal_year=abc", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	_, _, fiscalYear, _ := parseFinancialsParams(c)
	assert.Nil(t, fiscalYear)
}

func TestParseFinancialsParams_SortAsc(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?sort=asc", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	_, _, _, sort := parseFinancialsParams(c)
	assert.Equal(t, "asc", sort)
}

func TestParseFinancialsParams_SortInvalidFallsBack(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?sort=invalid", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	_, _, _, sort := parseFinancialsParams(c)
	assert.Equal(t, "desc", sort)
}

func TestParseFinancialsParams_Combined(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?timeframe=annual&limit=12&fiscal_year=2022&sort=asc", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	timeframe, limit, fiscalYear, sort := parseFinancialsParams(c)
	assert.Equal(t, models.TimeframeAnnual, timeframe)
	assert.Equal(t, 12, limit)
	assert.NotNil(t, fiscalYear)
	assert.Equal(t, 2022, *fiscalYear)
	assert.Equal(t, "asc", sort)
}

// ---------------------------------------------------------------------------
// getPeriodsOrNull — pure function
// ---------------------------------------------------------------------------

func TestGetPeriodsOrNull_Nil(t *testing.T) {
	result := getPeriodsOrNull(nil)
	assert.Nil(t, result)
}

func TestGetPeriodsOrNull_EmptyPeriods(t *testing.T) {
	response := &models.FinancialsResponse{
		Periods: []models.FinancialPeriod{},
	}
	result := getPeriodsOrNull(response)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestGetPeriodsOrNull_WithPeriods(t *testing.T) {
	q1 := 1
	q2 := 2
	response := &models.FinancialsResponse{
		Periods: []models.FinancialPeriod{
			{FiscalYear: 2023, FiscalQuarter: &q1},
			{FiscalYear: 2023, FiscalQuarter: &q2},
		},
	}
	result := getPeriodsOrNull(response)
	assert.Len(t, result, 2)
}

// ---------------------------------------------------------------------------
// NewFinancialsHandler — constructor
// ---------------------------------------------------------------------------

func TestNewFinancialsHandler(t *testing.T) {
	handler := NewFinancialsHandler()
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.service)
}

// ---------------------------------------------------------------------------
// FinancialsHandler.GetIncomeStatements — DB not available
// ---------------------------------------------------------------------------

func TestFinancialsHandler_GetIncomeStatements_NoDB(t *testing.T) {
	original := database.DB
	database.DB = nil
	defer func() { database.DB = original }()

	handler := NewFinancialsHandler()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks/AAPL/financials/income", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: "AAPL"}}

	handler.GetIncomeStatements(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// FinancialsHandler.GetBalanceSheets — DB not available
// ---------------------------------------------------------------------------

func TestFinancialsHandler_GetBalanceSheets_NoDB(t *testing.T) {
	original := database.DB
	database.DB = nil
	defer func() { database.DB = original }()

	handler := NewFinancialsHandler()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks/AAPL/financials/balance", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: "AAPL"}}

	handler.GetBalanceSheets(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// FinancialsHandler.GetCashFlowStatements — DB not available
// ---------------------------------------------------------------------------

func TestFinancialsHandler_GetCashFlowStatements_NoDB(t *testing.T) {
	original := database.DB
	database.DB = nil
	defer func() { database.DB = original }()

	handler := NewFinancialsHandler()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks/AAPL/financials/cashflow", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: "AAPL"}}

	handler.GetCashFlowStatements(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// FinancialsHandler.GetRatios — DB not available
// ---------------------------------------------------------------------------

func TestFinancialsHandler_GetRatios_NoDB(t *testing.T) {
	original := database.DB
	database.DB = nil
	defer func() { database.DB = original }()

	handler := NewFinancialsHandler()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks/AAPL/financials/ratios", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: "AAPL"}}

	handler.GetRatios(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// FinancialsHandler.RefreshFinancials — DB not available
// ---------------------------------------------------------------------------

func TestFinancialsHandler_RefreshFinancials_NoDB(t *testing.T) {
	original := database.DB
	database.DB = nil
	defer func() { database.DB = original }()

	handler := NewFinancialsHandler()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stocks/AAPL/financials/refresh", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: "AAPL"}}

	handler.RefreshFinancials(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// FinancialsHandler.GetAllFinancials — DB not available
// ---------------------------------------------------------------------------

func TestFinancialsHandler_GetAllFinancials_NoDB(t *testing.T) {
	original := database.DB
	database.DB = nil
	defer func() { database.DB = original }()

	handler := NewFinancialsHandler()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks/AAPL/financials", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: "AAPL"}}

	handler.GetAllFinancials(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
