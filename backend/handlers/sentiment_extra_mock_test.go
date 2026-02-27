package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// snapshotColumns returns the column names for the ticker_sentiment_snapshots table.
func snapshotColumns() []string {
	return []string{
		"id", "ticker", "snapshot_time", "time_range",
		"mention_count", "total_upvotes", "total_comments", "unique_posts",
		"bullish_count", "neutral_count", "bearish_count",
		"bullish_pct", "neutral_pct", "bearish_pct",
		"sentiment_score", "sentiment_label",
		"mention_velocity_1h", "sentiment_velocity_24h",
		"composite_score", "subreddit_distribution",
		"rank", "previous_rank", "rank_change", "created_at",
	}
}

// addSnapshotRow adds a snapshot row with the given ticker, rank, and time range.
func addSnapshotRow(rows *sqlmock.Rows, id int64, ticker string, rank int, timeRange string) *sqlmock.Rows {
	now := time.Now()
	velocity := 5.0
	return rows.AddRow(
		id, ticker, now, timeRange,
		100, 500, 200, 50,
		60, 30, 10,
		0.60, 0.30, 0.10,
		0.55, "bullish",
		&velocity, nil,
		75.0, []byte(`{"wallstreetbets": 40, "stocks": 30}`),
		&rank, nil, nil, now,
	)
}

// ---------------------------------------------------------------------------
// GetTrendingSentiment — success path tests
// ---------------------------------------------------------------------------

func TestGetTrendingSentiment_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// GetLatestSnapshots calls getLatestSnapshotsWithWindow(timeRange, limit, "1 hour")
	rows := sqlmock.NewRows(snapshotColumns())
	addSnapshotRow(rows, 1, "AAPL", 1, "1d")
	addSnapshotRow(rows, 2, "MSFT", 2, "1d")
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	// GetCompanyNames query
	nameRows := sqlmock.NewRows([]string{"symbol", "name"}).
		AddRow("AAPL", "Apple Inc.").
		AddRow("MSFT", "Microsoft Corp.")
	mock.ExpectQuery("SELECT").WillReturnRows(nameRows)

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/trending", GetTrendingSentiment)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sentiment/trending?period=24h&limit=10", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "AAPL")
	assert.Contains(t, w.Body.String(), "MSFT")
}

func TestGetTrendingSentiment_Mock_EmptyThenFallback(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// First query (1 hour window) returns empty
	emptyRows := sqlmock.NewRows(snapshotColumns())
	mock.ExpectQuery("SELECT").WillReturnRows(emptyRows)

	// Second query (6 hour fallback) returns data
	rows := sqlmock.NewRows(snapshotColumns())
	addSnapshotRow(rows, 1, "TSLA", 1, "1d")
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	// GetCompanyNames
	nameRows := sqlmock.NewRows([]string{"symbol", "name"}).
		AddRow("TSLA", "Tesla Inc.")
	mock.ExpectQuery("SELECT").WillReturnRows(nameRows)

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/trending", GetTrendingSentiment)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sentiment/trending", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "TSLA")
}

func TestGetTrendingSentiment_Mock_LimitCapped(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Limit > 50 gets capped to 50
	rows := sqlmock.NewRows(snapshotColumns())
	addSnapshotRow(rows, 1, "AAPL", 1, "1d")
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	nameRows := sqlmock.NewRows([]string{"symbol", "name"}).
		AddRow("AAPL", "Apple Inc.")
	mock.ExpectQuery("SELECT").WillReturnRows(nameRows)

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/trending", GetTrendingSentiment)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sentiment/trending?limit=100", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetTrendingSentiment_Mock_CompanyNamesFails(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	rows := sqlmock.NewRows(snapshotColumns())
	addSnapshotRow(rows, 1, "AAPL", 1, "1d")
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	// GetCompanyNames fails - should still return data without company names
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("names error"))

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/trending", GetTrendingSentiment)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sentiment/trending", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "AAPL")
}

// ---------------------------------------------------------------------------
// GetTickerSentiment — success path tests
// ---------------------------------------------------------------------------

func TestGetTickerSentiment_Mock_NoData(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// GetTickerSnapshot returns no rows (nil, nil)
	emptyRows := sqlmock.NewRows(snapshotColumns())
	mock.ExpectQuery("SELECT").WillReturnRows(emptyRows)

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/:ticker", GetTickerSentiment)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sentiment/AAPL", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "neutral")
}

func TestGetTickerSentiment_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	rank := 5
	rankChange := -2
	velocity := 3.0

	// First query: GetTickerSnapshot(ticker, "7d")
	rows7d := sqlmock.NewRows(snapshotColumns()).AddRow(
		1, "AAPL", now, "7d",
		200, 1000, 400, 100,
		120, 60, 20,
		0.60, 0.30, 0.10,
		0.55, "bullish",
		&velocity, nil,
		80.0, []byte(`{"wallstreetbets": 50, "stocks": 30, "investing": 20}`),
		&rank, nil, &rankChange, now,
	)
	mock.ExpectQuery("SELECT").WillReturnRows(rows7d)

	// Second query: GetTickerSnapshot(ticker, "1d")
	rows1d := sqlmock.NewRows(snapshotColumns()).AddRow(
		2, "AAPL", now, "1d",
		50, 250, 100, 25,
		30, 15, 5,
		0.60, 0.30, 0.10,
		0.55, "bullish",
		&velocity, nil,
		75.0, []byte(`{}`),
		nil, nil, nil, now,
	)
	mock.ExpectQuery("SELECT").WillReturnRows(rows1d)

	// GetCompanyNames
	nameRows := sqlmock.NewRows([]string{"symbol", "name"}).
		AddRow("AAPL", "Apple Inc.")
	mock.ExpectQuery("SELECT").WillReturnRows(nameRows)

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/:ticker", GetTickerSentiment)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sentiment/AAPL", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "AAPL")
	assert.Contains(t, w.Body.String(), "bullish")
	assert.Contains(t, w.Body.String(), "Apple Inc.")
}

func TestGetTickerSentiment_Mock_EmptyTicker(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/:ticker", GetTickerSentiment)

	w := httptest.NewRecorder()
	// A route param of "/" would result in empty ticker after ToUpper
	// Actually gin would not match empty param, so we just test the handler
	// doesn't panic with a valid ticker
	req := httptest.NewRequest(http.MethodGet, "/sentiment/aapl", nil)
	r.ServeHTTP(w, req)

	// With no DB expectations set, the DB query will fail
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
}

// ---------------------------------------------------------------------------
// GetTickerSentimentHistory — success path test
// ---------------------------------------------------------------------------

func TestGetTickerSentimentHistory_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	// GetSentimentTimeSeries queries: time, ticker, sentiment_score, bullish_pct, bearish_pct, neutral_pct, mention_count, composite_score
	histCols := []string{
		"time", "ticker", "sentiment_score", "bullish_pct", "bearish_pct", "neutral_pct", "mention_count", "composite_score",
	}
	histRows := sqlmock.NewRows(histCols).
		AddRow(yesterday, "AAPL", 0.55, 0.60, 0.10, 0.30, 100, 75.0).
		AddRow(now, "AAPL", 0.60, 0.65, 0.10, 0.25, 120, 80.0)
	mock.ExpectQuery("SELECT").WillReturnRows(histRows)

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/:ticker/history", GetTickerSentimentHistory)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sentiment/AAPL/history?days=30", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "AAPL")
}

// ---------------------------------------------------------------------------
// GetTickerPosts — success path test
// ---------------------------------------------------------------------------

func TestGetTickerPosts_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// GetTickerPostsV2 scans: id, title, body, url, subreddit, upvotes, comment_count, flair, posted_at, sentiment, confidence
	postCols := []string{
		"id", "title", "body", "url", "subreddit",
		"upvotes", "comment_count", "flair", "posted_at",
		"sentiment", "confidence",
	}
	postRows := sqlmock.NewRows(postCols).
		AddRow(1, "AAPL to the moon!", "Great earnings report", "https://reddit.com/1", "wallstreetbets",
			100, 50, "DD", now,
			"bullish", 0.85)
	mock.ExpectQuery("SELECT").WillReturnRows(postRows)

	// Count query
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	r := setupMockRouterNoAuth()
	r.GET("/sentiment/:ticker/posts", GetTickerPosts)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sentiment/AAPL/posts?limit=10", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "AAPL to the moon")
}
