package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"investorcenter-api/services"
)

// ---------------------------------------------------------------------------
// RunBacktest — validation tests
// ---------------------------------------------------------------------------

func TestRunBacktest_Mock_InvalidJSON(t *testing.T) {
	handler := NewBacktestHandler(services.NewBacktestService())

	r := setupMockRouterNoAuth()
	r.POST("/backtest", handler.RunBacktest)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/backtest", bytes.NewBufferString("bad json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// SubmitBacktestJob — validation tests
// ---------------------------------------------------------------------------

func TestSubmitBacktestJob_Mock_InvalidJSON(t *testing.T) {
	handler := NewBacktestHandler(services.NewBacktestService())

	r := setupMockRouterNoAuth()
	r.POST("/backtest/jobs", handler.SubmitBacktestJob)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/backtest/jobs", bytes.NewBufferString("bad"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// GetBacktestJobStatus — validation tests
// ---------------------------------------------------------------------------

func TestGetBacktestJobStatus_Mock_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// GetBacktestJob queries database.DB
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "config", "status", "created_at", "started_at", "completed_at", "error", "user_id"}),
	)

	handler := NewBacktestHandler(services.NewBacktestService())

	r := setupMockRouterNoAuth()
	r.GET("/backtest/jobs/:jobId", handler.GetBacktestJobStatus)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/backtest/jobs/nonexistent-id", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// GetBacktestJobResult — validation tests
// ---------------------------------------------------------------------------

func TestGetBacktestJobResult_Mock_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// GetJobResult queries database.DB
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "config", "status", "created_at", "started_at", "completed_at", "error", "user_id"}),
	)

	handler := NewBacktestHandler(services.NewBacktestService())

	r := setupMockRouterNoAuth()
	r.GET("/backtest/jobs/:jobId/result", handler.GetBacktestJobResult)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/backtest/jobs/nonexistent/result", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// GetLatestBacktest — test when no backtests exist
// ---------------------------------------------------------------------------

func TestGetLatestBacktest_Mock_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// GetRecentCompletedBacktest queries database.DB
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "config", "status", "created_at", "started_at", "completed_at", "error", "user_id", "result"}),
	)

	handler := NewBacktestHandler(services.NewBacktestService())

	r := setupMockRouterNoAuth()
	r.GET("/backtest/latest", handler.GetLatestBacktest)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/backtest/latest", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// GetBacktestCharts — test when no backtests exist
// ---------------------------------------------------------------------------

func TestGetBacktestCharts_Mock_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "config", "status", "created_at", "started_at", "completed_at", "error", "user_id", "result"}),
	)

	handler := NewBacktestHandler(services.NewBacktestService())

	r := setupMockRouterNoAuth()
	r.GET("/backtest/charts", handler.GetBacktestCharts)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/backtest/charts", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// GetDefaultConfig — test
// ---------------------------------------------------------------------------

func TestGetDefaultConfig_Mock_Success(t *testing.T) {
	handler := NewBacktestHandler(services.NewBacktestService())

	r := setupMockRouterNoAuth()
	r.GET("/backtest/config/default", handler.GetDefaultConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/backtest/config/default", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// GetUserBacktests — auth tests
// ---------------------------------------------------------------------------

func TestGetUserBacktests_Mock_NoAuth(t *testing.T) {
	handler := NewBacktestHandler(services.NewBacktestService())

	r := setupMockRouterNoAuth()
	r.GET("/backtest/history", handler.GetUserBacktests)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/backtest/history", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------------------------------------------------------------------------
// RunQuickBacktest — test when no cache exists
// ---------------------------------------------------------------------------

func TestRunQuickBacktest_Mock_NoCache(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// GetCachedBacktestResult queries database.DB
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "config", "status", "created_at", "started_at", "completed_at", "error", "user_id", "result"}),
	)

	handler := NewBacktestHandler(services.NewBacktestService())

	r := setupMockRouterNoAuth()
	r.GET("/backtest/quick", handler.RunQuickBacktest)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/backtest/quick", nil)
	r.ServeHTTP(w, req)

	// Will fail because the IC Score API is not available
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
