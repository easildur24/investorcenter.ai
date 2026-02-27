package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// GetTrendingSentiment — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetTrendingSentiment_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// GetLatestSnapshots queries ticker_sentiment_snapshots
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/trending", GetTrendingSentiment)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sentiment/trending", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to fetch trending sentiment")
}

func TestGetTrendingSentiment_Mock_DBError_7d(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/trending", GetTrendingSentiment)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sentiment/trending?period=7d&limit=10", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetTrendingSentiment_Mock_InvalidPeriod(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Invalid period defaults to "24h" → time_range "1d"
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/trending", GetTrendingSentiment)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sentiment/trending?period=invalid&limit=abc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------------------------------------------------------------------------
// GetTickerSentiment — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetTickerSentiment_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// GetTickerSnapshot queries ticker_sentiment_snapshots
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/:ticker", GetTickerSentiment)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sentiment/AAPL", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to fetch ticker sentiment")
}

// ---------------------------------------------------------------------------
// GetTickerSentimentHistory — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetTickerSentimentHistory_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/:ticker/history", GetTickerSentimentHistory)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sentiment/AAPL/history", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to fetch sentiment history")
}

func TestGetTickerSentimentHistory_Mock_InvalidDays(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Invalid days defaults to 7
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/:ticker/history", GetTickerSentimentHistory)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sentiment/AAPL/history?days=abc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetTickerSentimentHistory_Mock_DaysCapped(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Days > 90 gets capped
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/:ticker/history", GetTickerSentimentHistory)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sentiment/AAPL/history?days=200", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------------------------------------------------------------------------
// GetTickerPosts — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetTickerPosts_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/:ticker/posts", GetTickerPosts)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sentiment/AAPL/posts", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to fetch posts")
}

func TestGetTickerPosts_Mock_WithParams(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/:ticker/posts", GetTickerPosts)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sentiment/AAPL/posts?limit=5&sort=engagement", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetTickerPosts_Mock_SortBullish(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/:ticker/posts", GetTickerPosts)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sentiment/AAPL/posts?sort=bullish", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetTickerPosts_Mock_SortBearish(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/:ticker/posts", GetTickerPosts)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sentiment/AAPL/posts?sort=bearish", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetTickerPosts_Mock_InvalidLimit(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/:ticker/posts", GetTickerPosts)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sentiment/AAPL/posts?limit=abc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetTickerPosts_Mock_LimitCapped(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/:ticker/posts", GetTickerPosts)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sentiment/AAPL/posts?limit=100", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
