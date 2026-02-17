package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"investorcenter-api/database"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// abs — pure helper function
// ---------------------------------------------------------------------------

func TestAbs_Positive(t *testing.T) {
	assert.Equal(t, 5.0, abs(5.0))
}

func TestAbs_Negative(t *testing.T) {
	assert.Equal(t, 5.0, abs(-5.0))
}

func TestAbs_Zero(t *testing.T) {
	assert.Equal(t, 0.0, abs(0.0))
}

func TestAbs_LargeNegative(t *testing.T) {
	assert.Equal(t, 1000000.123, abs(-1000000.123))
}

func TestAbs_SmallNegative(t *testing.T) {
	assert.Equal(t, 0.001, abs(-0.001))
}

// ---------------------------------------------------------------------------
// GetICScore — input validation
// ---------------------------------------------------------------------------

func TestGetICScore_EmptyTicker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks//ic-score", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: ""}}

	GetICScore(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Ticker symbol is required", resp["error"])
}

func TestGetICScore_NoDB(t *testing.T) {
	original := database.DB
	database.DB = nil
	defer func() { database.DB = original }()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks/AAPL/ic-score", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: "AAPL"}}

	GetICScore(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Database not available", resp["error"])
}

// ---------------------------------------------------------------------------
// GetICScores — input validation
// ---------------------------------------------------------------------------

func TestGetICScores_NoDB(t *testing.T) {
	original := database.DB
	database.DB = nil
	defer func() { database.DB = original }()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ic-scores", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	GetICScores(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// GetFinancialMetrics — input validation
// ---------------------------------------------------------------------------

func TestGetFinancialMetrics_EmptyTicker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks//financials", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: ""}}

	GetFinancialMetrics(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Ticker symbol is required", resp["error"])
}

func TestGetFinancialMetrics_NoDB(t *testing.T) {
	original := database.DB
	database.DB = nil
	defer func() { database.DB = original }()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks/AAPL/financials", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: "AAPL"}}

	GetFinancialMetrics(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// GetComprehensiveFinancialMetrics — input validation
// ---------------------------------------------------------------------------

func TestGetComprehensiveFinancialMetrics_EmptyTicker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks//metrics", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: ""}}

	GetComprehensiveFinancialMetrics(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Ticker symbol is required", resp["error"])
}

// ---------------------------------------------------------------------------
// GetRiskMetrics — input validation
// ---------------------------------------------------------------------------

func TestGetRiskMetrics_EmptyTicker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks//risk", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: ""}}

	GetRiskMetrics(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Ticker symbol is required", resp["error"])
}

func TestGetRiskMetrics_NoDB(t *testing.T) {
	original := database.DB
	database.DB = nil
	defer func() { database.DB = original }()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks/AAPL/risk", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: "AAPL"}}

	GetRiskMetrics(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// GetTechnicalIndicators — input validation
// ---------------------------------------------------------------------------

func TestGetTechnicalIndicators_EmptyTicker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks//technical", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: ""}}

	GetTechnicalIndicators(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Ticker symbol is required", resp["error"])
}

func TestGetTechnicalIndicators_NoDB(t *testing.T) {
	original := database.DB
	database.DB = nil
	defer func() { database.DB = original }()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks/AAPL/technical", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: "AAPL"}}

	GetTechnicalIndicators(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// GetICScoreHistory — input validation
// ---------------------------------------------------------------------------

func TestGetICScoreHistory_EmptyTicker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks//ic-score/history", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: ""}}

	GetICScoreHistory(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Ticker symbol is required", resp["error"])
}

func TestGetICScoreHistory_NoDB(t *testing.T) {
	original := database.DB
	database.DB = nil
	defer func() { database.DB = original }()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks/AAPL/ic-score/history", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: "AAPL"}}

	GetICScoreHistory(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// GetICScore — ticker uppercasing
// ---------------------------------------------------------------------------

func TestGetICScore_TickerUppercased(t *testing.T) {
	original := database.DB
	database.DB = nil
	defer func() { database.DB = original }()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks/aapl/ic-score", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: "aapl"}}

	GetICScore(c)

	// Should not fail with "Ticker symbol is required" - it should get to DB check
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
