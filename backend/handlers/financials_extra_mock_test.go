package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// GetAllFinancials — additional DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetAllFinancials_Mock_AllFail(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// All three statement queries fail
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("income error"))
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("balance error"))
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("cashflow error"))

	handler := NewFinancialsHandler()
	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/financials", handler.GetAllFinancials)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/financials", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Financial data not found")
}

// ---------------------------------------------------------------------------
// RefreshFinancials — additional DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestRefreshFinancials_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// The refresh function queries the database for ticker_id first
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("refresh error"))

	handler := NewFinancialsHandler()
	r := setupMockRouterNoAuth()
	r.POST("/stocks/:ticker/financials/refresh", handler.RefreshFinancials)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/stocks/AAPL/financials/refresh", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to refresh financial data")
}

// ---------------------------------------------------------------------------
// GetCashFlowStatements — additional tests with parameters
// ---------------------------------------------------------------------------

func TestGetCashFlowStatements_Mock_WithTimeframeParam(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Service calls database.DB query
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("not found"))

	handler := NewFinancialsHandler()
	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/financials/cashflow", handler.GetCashFlowStatements)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/financials/cashflow?timeframe=annual&limit=4", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// GetIncomeStatements — additional tests with query params
// ---------------------------------------------------------------------------

func TestGetIncomeStatements_Mock_AnnualTimeframe(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("not found"))

	handler := NewFinancialsHandler()
	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/financials/income", handler.GetIncomeStatements)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/financials/income?timeframe=annual", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetIncomeStatements_Mock_TTMTimeframe(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("not found"))

	handler := NewFinancialsHandler()
	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/financials/income", handler.GetIncomeStatements)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/financials/income?timeframe=ttm", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// GetBalanceSheets — additional tests with query params
// ---------------------------------------------------------------------------

func TestGetBalanceSheets_Mock_AnnualTimeframe(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("not found"))

	handler := NewFinancialsHandler()
	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/financials/balance", handler.GetBalanceSheets)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/financials/balance?timeframe=annual&limit=20&sort=asc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// GetRatios — additional tests with query params
// ---------------------------------------------------------------------------

func TestGetRatios_Mock_WithParams(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("not found"))

	handler := NewFinancialsHandler()
	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/financials/ratios", handler.GetRatios)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/financials/ratios?timeframe=annual&limit=50&fiscal_year=2023", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
