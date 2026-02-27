package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// ValidateTickerExists — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestValidateTickerExists_Mock_Exists(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnRows(
		mock.NewRows([]string{"id"}).AddRow(1),
	)

	result := ValidateTickerExists("AAPL")
	assert.True(t, result)
}

func TestValidateTickerExists_Mock_NotExists(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("not found"))

	result := ValidateTickerExists("INVALID")
	assert.False(t, result)
}

// ---------------------------------------------------------------------------
// GetIncomeStatements — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetIncomeStatements_Mock_NilDB(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	origDB := getDatabaseDB()
	setDatabaseDBNil()
	defer restoreDatabaseDB(origDB)

	handler := NewFinancialsHandler()
	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/financials/income", handler.GetIncomeStatements)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/financials/income", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestGetIncomeStatements_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	handler := NewFinancialsHandler()
	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/financials/income", handler.GetIncomeStatements)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/financials/income", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// GetBalanceSheets — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetBalanceSheets_Mock_NilDB(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	origDB := getDatabaseDB()
	setDatabaseDBNil()
	defer restoreDatabaseDB(origDB)

	handler := NewFinancialsHandler()
	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/financials/balance", handler.GetBalanceSheets)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/financials/balance", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestGetBalanceSheets_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	handler := NewFinancialsHandler()
	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/financials/balance", handler.GetBalanceSheets)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/financials/balance", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// GetCashFlowStatements — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetCashFlowStatements_Mock_NilDB(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	origDB := getDatabaseDB()
	setDatabaseDBNil()
	defer restoreDatabaseDB(origDB)

	handler := NewFinancialsHandler()
	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/financials/cashflow", handler.GetCashFlowStatements)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/financials/cashflow", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestGetCashFlowStatements_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	handler := NewFinancialsHandler()
	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/financials/cashflow", handler.GetCashFlowStatements)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/financials/cashflow", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// GetRatios — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetRatios_Mock_NilDB(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	origDB := getDatabaseDB()
	setDatabaseDBNil()
	defer restoreDatabaseDB(origDB)

	handler := NewFinancialsHandler()
	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/financials/ratios", handler.GetRatios)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/financials/ratios", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestGetRatios_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	handler := NewFinancialsHandler()
	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/financials/ratios", handler.GetRatios)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/financials/ratios", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// GetAllFinancials — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetAllFinancials_Mock_NilDB(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	origDB := getDatabaseDB()
	setDatabaseDBNil()
	defer restoreDatabaseDB(origDB)

	handler := NewFinancialsHandler()
	r := setupMockRouterNoAuth()
	r.GET("/stocks/:ticker/financials", handler.GetAllFinancials)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stocks/AAPL/financials", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// RefreshFinancials — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestRefreshFinancials_Mock_NilDB(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	origDB := getDatabaseDB()
	setDatabaseDBNil()
	defer restoreDatabaseDB(origDB)

	handler := NewFinancialsHandler()
	r := setupMockRouterNoAuth()
	r.POST("/stocks/:ticker/financials/refresh", handler.RefreshFinancials)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/stocks/AAPL/financials/refresh", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
