package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// GetRedditHeatmap — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetRedditHeatmap_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/reddit/heatmap", GetRedditHeatmap)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/reddit/heatmap", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to fetch Reddit heatmap data")
}

func TestGetRedditHeatmap_Mock_InvalidParams(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Invalid days/top params default to 7 and 50
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/reddit/heatmap", GetRedditHeatmap)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/reddit/heatmap?days=abc&top=xyz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetRedditHeatmap_Mock_DaysCapped(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Days > 90 gets capped
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/reddit/heatmap", GetRedditHeatmap)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/reddit/heatmap?days=200&top=200", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------------------------------------------------------------------------
// GetRedditPipelineHealth — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetRedditPipelineHealth_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/reddit/health", GetRedditPipelineHealth)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/reddit/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------------------------------------------------------------------------
// GetTickerRedditHistory — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetTickerRedditHistory_Mock_MissingSymbol(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupMockRouterNoAuth()
	// Register with a :symbol param, but call without symbol to get empty
	r.GET("/reddit/ticker/:symbol/history", GetTickerRedditHistory)

	// This won't match the route, so gin returns 404
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/reddit/ticker//history", nil)
	r.ServeHTTP(w, req)

	// Gin will route to 404 or the handler checks for empty symbol
	assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound)
}

func TestGetTickerRedditHistory_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/reddit/ticker/:symbol/history", GetTickerRedditHistory)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/reddit/ticker/AAPL/history", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetTickerRedditHistory_Mock_InvalidDays(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/reddit/ticker/:symbol/history", GetTickerRedditHistory)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/reddit/ticker/AAPL/history?days=abc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetTickerRedditHistory_Mock_DaysCapped(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/reddit/ticker/:symbol/history", GetTickerRedditHistory)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/reddit/ticker/AAPL/history?days=200", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
