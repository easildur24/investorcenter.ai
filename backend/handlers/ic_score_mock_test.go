package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// GetICScore — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetICScore_Mock_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)

	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/ic-score", GetICScore)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/ic-score", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "IC Score not found")
}

func TestGetICScore_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("connection error"))

	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/ic-score", GetICScore)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/ic-score", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to fetch IC Score")
}

func TestGetICScore_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "ticker", "date", "overall_score",
		"value_score", "growth_score", "profitability_score", "financial_health_score",
		"momentum_score", "analyst_consensus_score", "insider_activity_score",
		"institutional_score", "news_sentiment_score", "technical_score",
		"rating", "sector_percentile", "confidence_level", "data_completeness",
		"created_at",
	}).AddRow(
		1, "AAPL", now, 85.5,
		80.0, 90.0, 85.0, 88.0,
		75.0, 82.0, 70.0,
		78.0, 65.0, 72.0,
		"Strong Buy", 92.5, "high", 95.0,
		now,
	)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/ic-score", GetICScore)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/ic-score", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "AAPL")
}

// ---------------------------------------------------------------------------
// GetICScores — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetICScores_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("query error"))

	r := setupMockRouterNoAuth()
	r.GET("/ic-scores", GetICScores)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ic-scores", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to fetch IC Scores")
}

func TestGetICScores_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// Main query returns scores
	rows := sqlmock.NewRows([]string{
		"ticker", "overall_score", "rating", "data_completeness", "created_at",
	}).AddRow("AAPL", 85.5, "Strong Buy", 95.0, now).
		AddRow("MSFT", 82.0, "Buy", 90.0, now)
	mock.ExpectQuery("SELECT .+ FROM").WillReturnRows(rows)

	// Count query
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// Total stocks count
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))

	r := setupMockRouterNoAuth()
	r.GET("/ic-scores", GetICScores)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ic-scores?limit=10&offset=0&sort=overall_score&order=desc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "AAPL")
	assert.Contains(t, w.Body.String(), "MSFT")
}

func TestGetICScores_Mock_WithSearch(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"ticker", "overall_score", "rating", "data_completeness", "created_at",
	}).AddRow("AAPL", 85.5, "Strong Buy", 95.0, now)
	mock.ExpectQuery("SELECT .+ FROM").WillReturnRows(rows)

	// Count query with search param
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	// Total stocks count
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))

	r := setupMockRouterNoAuth()
	r.GET("/ic-scores", GetICScores)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ic-scores?search=AAPL", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetICScores_Mock_EmptyResult(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{
		"ticker", "overall_score", "rating", "data_completeness", "created_at",
	})
	mock.ExpectQuery("SELECT .+ FROM").WillReturnRows(rows)

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))

	r := setupMockRouterNoAuth()
	r.GET("/ic-scores", GetICScores)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ic-scores", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"data":[]`)
}

// ---------------------------------------------------------------------------
// GetFinancialMetrics — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetFinancialMetrics_Mock_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// DB returns no rows (fmpClient has no API key in test environment)
	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)

	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/financials", GetFinancialMetrics)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/financials", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Financial data not found")
}

func TestGetFinancialMetrics_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("connection refused"))

	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/financials", GetFinancialMetrics)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/financials", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to fetch financial metrics")
}

// ---------------------------------------------------------------------------
// GetRiskMetrics — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetRiskMetrics_IC_Mock_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)

	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/risk", GetRiskMetrics)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/risk", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Risk metrics not found")
}

func TestGetRiskMetrics_IC_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("connection error"))

	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/risk", GetRiskMetrics)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/risk", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to fetch risk metrics")
}

func TestGetRiskMetrics_IC_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{
		"time", "ticker", "period", "alpha", "beta", "sharpe_ratio", "sortino_ratio",
		"std_dev", "max_drawdown", "var_5", "annualized_return", "downside_deviation", "data_points",
	}).AddRow(
		"2024-01-01", "AAPL", "1Y", 0.05, 1.1, 1.5, 2.0,
		0.25, -0.15, -0.02, 0.12, 0.18, 252,
	)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/risk", GetRiskMetrics)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/risk?period=1Y", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "AAPL")
}

// ---------------------------------------------------------------------------
// GetTechnicalIndicators — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetTechnicalIndicators_IC_Mock_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)

	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/technical", GetTechnicalIndicators)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/technical", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Technical indicators not found")
}

func TestGetTechnicalIndicators_IC_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("query error"))

	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/technical", GetTechnicalIndicators)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/technical", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to fetch technical indicators")
}

func TestGetTechnicalIndicators_IC_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{
		"calculation_date", "ticker", "current_price",
		"sma_50", "sma_200", "ema_12", "ema_26",
		"rsi_14", "macd", "macd_signal", "macd_histogram",
		"bb_upper", "bb_middle", "bb_lower",
		"volume_ma_20", "return_1m", "return_3m", "return_6m", "return_12m",
	}).AddRow(
		"2024-01-01", "AAPL", 185.50,
		180.0, 170.0, 183.0, 178.0,
		55.0, 2.5, 1.8, 0.7,
		190.0, 180.0, 170.0,
		50000000.0, 5.0, 10.0, 15.0, 25.0,
	)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/technical", GetTechnicalIndicators)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/technical", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "AAPL")
}

// ---------------------------------------------------------------------------
// GetICScoreHistory — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetICScoreHistory_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("query error"))

	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/ic-score/history", GetICScoreHistory)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/ic-score/history", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to fetch IC Score history")
}

func TestGetICScoreHistory_Mock_EmptyResult(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{
		"id", "ticker", "date", "overall_score",
		"value_score", "growth_score", "profitability_score", "financial_health_score",
		"momentum_score", "analyst_consensus_score", "insider_activity_score",
		"institutional_score", "news_sentiment_score", "technical_score",
		"rating", "sector_percentile", "confidence_level", "data_completeness",
		"created_at",
	})
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/ic-score/history", GetICScoreHistory)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/ic-score/history?days=30", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"count":0`)
}

func TestGetICScoreHistory_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	rows := sqlmock.NewRows([]string{
		"id", "ticker", "date", "overall_score",
		"value_score", "growth_score", "profitability_score", "financial_health_score",
		"momentum_score", "analyst_consensus_score", "insider_activity_score",
		"institutional_score", "news_sentiment_score", "technical_score",
		"rating", "sector_percentile", "confidence_level", "data_completeness",
		"created_at",
	}).AddRow(
		1, "AAPL", yesterday, 85.5,
		80.0, 90.0, 85.0, 88.0,
		75.0, 82.0, 70.0,
		78.0, 65.0, 72.0,
		"Strong Buy", 92.5, "high", 95.0,
		yesterday,
	).AddRow(
		2, "AAPL", now, 86.0,
		81.0, 91.0, 86.0, 89.0,
		76.0, 83.0, 71.0,
		79.0, 66.0, 73.0,
		"Strong Buy", 93.0, "high", 96.0,
		now,
	)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/ic-score/history", GetICScoreHistory)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/ic-score/history?days=90", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"count":2`)
}

// ---------------------------------------------------------------------------
// GetComprehensiveFinancialMetrics — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetComprehensiveFinancialMetrics_Mock_NilDB(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	origDB := getDatabaseDB()
	setDatabaseDBNil()
	defer restoreDatabaseDB(origDB)

	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/metrics", GetComprehensiveFinancialMetrics)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/metrics", nil)
	r.ServeHTTP(w, req)

	// Even with nil DB, this handler continues (using FMP client as primary)
	// and returns 200 with whatever data is available
	assert.Equal(t, http.StatusOK, w.Code)
}
