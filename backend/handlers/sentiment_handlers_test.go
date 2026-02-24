package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"investorcenter-api/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// GetTickerSentiment — param validation
// ---------------------------------------------------------------------------

func TestGetTickerSentiment_EmptyTicker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sentiment/", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: ""}}

	GetTickerSentiment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Ticker symbol is required", resp["error"])
}

// ---------------------------------------------------------------------------
// GetTickerSentimentHistory — param validation
// ---------------------------------------------------------------------------

func TestGetTickerSentimentHistory_EmptyTicker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sentiment//history", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: ""}}

	GetTickerSentimentHistory(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Ticker symbol is required", resp["error"])
}

// ---------------------------------------------------------------------------
// GetTickerPosts — param validation
// ---------------------------------------------------------------------------

func TestGetTickerPosts_EmptyTicker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sentiment//posts", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: ""}}

	GetTickerPosts(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Ticker symbol is required", resp["error"])
}

// ---------------------------------------------------------------------------
// parseTopSubreddits — unit tests
// ---------------------------------------------------------------------------

func TestParseTopSubreddits_NilData(t *testing.T) {
	result := parseTopSubreddits(nil, 5)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestParseTopSubreddits_EmptyJSON(t *testing.T) {
	result := parseTopSubreddits(json.RawMessage("{}"), 5)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestParseTopSubreddits_InvalidJSON(t *testing.T) {
	result := parseTopSubreddits(json.RawMessage("{invalid"), 5)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestParseTopSubreddits_SortedByCount(t *testing.T) {
	data := json.RawMessage(`{"stocks": 10, "wallstreetbets": 50, "investing": 30}`)
	result := parseTopSubreddits(data, 5)

	require.Len(t, result, 3)
	assert.Equal(t, "wallstreetbets", result[0].Subreddit)
	assert.Equal(t, 50, result[0].Count)
	assert.Equal(t, "investing", result[1].Subreddit)
	assert.Equal(t, 30, result[1].Count)
	assert.Equal(t, "stocks", result[2].Subreddit)
	assert.Equal(t, 10, result[2].Count)
}

func TestParseTopSubreddits_TruncatedToTopN(t *testing.T) {
	data := json.RawMessage(`{"a": 5, "b": 4, "c": 3, "d": 2, "e": 1}`)
	result := parseTopSubreddits(data, 3)

	require.Len(t, result, 3)
	assert.Equal(t, 5, result[0].Count)
	assert.Equal(t, 4, result[1].Count)
	assert.Equal(t, 3, result[2].Count)
}

func TestParseTopSubreddits_FewerThanTopN(t *testing.T) {
	data := json.RawMessage(`{"stocks": 10}`)
	result := parseTopSubreddits(data, 5)

	require.Len(t, result, 1)
	assert.Equal(t, "stocks", result[0].Subreddit)
}

// ---------------------------------------------------------------------------
// groupTimeSeriesByDate — unit tests
// ---------------------------------------------------------------------------

func TestGroupTimeSeriesByDate_Empty(t *testing.T) {
	result := groupTimeSeriesByDate(nil)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestGroupTimeSeriesByDate_SinglePoint(t *testing.T) {
	bearish := 0.3
	neutral := 0.2
	points := []models.SentimentTimeSeriesPoint{
		{
			Time:           time.Date(2026, 2, 20, 14, 0, 0, 0, time.UTC),
			Ticker:         "AAPL",
			SentimentScore: 0.5,
			BullishPct:     0.5,
			BearishPct:     &bearish,
			NeutralPct:     &neutral,
			MentionCount:   10,
			CompositeScore: 0.7,
		},
	}

	result := groupTimeSeriesByDate(points)
	require.Len(t, result, 1)
	assert.Equal(t, "2026-02-20", result[0].Date)
	assert.Equal(t, 0.5, result[0].Score)
	assert.Equal(t, 10, result[0].PostCount)
	assert.Equal(t, 5, result[0].Bullish) // 0.5 * 10
	assert.Equal(t, 3, result[0].Bearish) // 0.3 * 10
	assert.Equal(t, 2, result[0].Neutral) // 10 - 5 - 3
}

func TestGroupTimeSeriesByDate_PicksLatestPerDay(t *testing.T) {
	bearish1 := 0.4
	neutral1 := 0.2
	bearish2 := 0.1
	neutral2 := 0.1
	points := []models.SentimentTimeSeriesPoint{
		{
			Time:           time.Date(2026, 2, 20, 10, 0, 0, 0, time.UTC),
			SentimentScore: 0.3,
			BullishPct:     0.4,
			BearishPct:     &bearish1,
			NeutralPct:     &neutral1,
			MentionCount:   5,
		},
		{
			Time:           time.Date(2026, 2, 20, 22, 0, 0, 0, time.UTC),
			SentimentScore: 0.8,
			BullishPct:     0.8,
			BearishPct:     &bearish2,
			NeutralPct:     &neutral2,
			MentionCount:   20,
		},
	}

	result := groupTimeSeriesByDate(points)
	require.Len(t, result, 1)
	// Should pick the 22:00 point (latest)
	assert.Equal(t, 0.8, result[0].Score)
	assert.Equal(t, 20, result[0].PostCount)
}

func TestGroupTimeSeriesByDate_MultipleDaysSorted(t *testing.T) {
	bearish := 0.2
	neutral := 0.3
	points := []models.SentimentTimeSeriesPoint{
		{
			Time:           time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC),
			SentimentScore: 0.2,
			BullishPct:     0.5,
			BearishPct:     &bearish,
			NeutralPct:     &neutral,
			MentionCount:   10,
		},
		{
			Time:           time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC),
			SentimentScore: 0.1,
			BullishPct:     0.5,
			BearishPct:     &bearish,
			NeutralPct:     &neutral,
			MentionCount:   8,
		},
		{
			Time:           time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC),
			SentimentScore: 0.3,
			BullishPct:     0.5,
			BearishPct:     &bearish,
			NeutralPct:     &neutral,
			MentionCount:   12,
		},
	}

	result := groupTimeSeriesByDate(points)
	require.Len(t, result, 3)
	// Must be sorted ascending by date
	assert.Equal(t, "2026-02-20", result[0].Date)
	assert.Equal(t, "2026-02-21", result[1].Date)
	assert.Equal(t, "2026-02-22", result[2].Date)
}

func TestGroupTimeSeriesByDate_NilPctFallback(t *testing.T) {
	// Simulate pre-migration-041 rows (no bearish_pct/neutral_pct)
	points := []models.SentimentTimeSeriesPoint{
		{
			Time:           time.Date(2026, 2, 20, 14, 0, 0, 0, time.UTC),
			SentimentScore: 0.5,
			BullishPct:     0.6,
			BearishPct:     nil,
			NeutralPct:     nil,
			MentionCount:   10,
		},
	}

	result := groupTimeSeriesByDate(points)
	require.Len(t, result, 1)
	assert.Equal(t, 6, result[0].Bullish) // round(0.6 * 10) = 6
	assert.Equal(t, 0, result[0].Bearish) // fallback: 0
	assert.Equal(t, 4, result[0].Neutral) // 10 - 6 - 0 = 4
}

func TestGroupTimeSeriesByDate_CountsSumToMentionCount(t *testing.T) {
	// 3 posts at 33.3% each — with math.Round this should sum correctly
	bearish := 1.0 / 3.0
	neutral := 1.0 / 3.0
	points := []models.SentimentTimeSeriesPoint{
		{
			Time:           time.Date(2026, 2, 20, 14, 0, 0, 0, time.UTC),
			SentimentScore: 0.0,
			BullishPct:     1.0 / 3.0,
			BearishPct:     &bearish,
			NeutralPct:     &neutral,
			MentionCount:   3,
		},
	}

	result := groupTimeSeriesByDate(points)
	require.Len(t, result, 1)
	total := result[0].Bullish + result[0].Bearish + result[0].Neutral
	assert.Equal(t, 3, total, "counts must sum to mention_count")
}

func TestGroupTimeSeriesByDate_LargeCountRounding(t *testing.T) {
	// With 100 posts: 33.3% bullish, 33.3% bearish, 33.4% neutral
	bearish := 0.333
	neutral := 0.334
	points := []models.SentimentTimeSeriesPoint{
		{
			Time:           time.Date(2026, 2, 20, 14, 0, 0, 0, time.UTC),
			SentimentScore: 0.0,
			BullishPct:     0.333,
			BearishPct:     &bearish,
			NeutralPct:     &neutral,
			MentionCount:   100,
		},
	}

	result := groupTimeSeriesByDate(points)
	require.Len(t, result, 1)

	// Bullish: round(0.333 * 100) = 33
	// Bearish: round(0.333 * 100) = 33
	// Neutral: 100 - 33 - 33 = 34
	assert.Equal(t, 33, result[0].Bullish)
	assert.Equal(t, 33, result[0].Bearish)
	assert.Equal(t, 34, result[0].Neutral)

	total := result[0].Bullish + result[0].Bearish + result[0].Neutral
	assert.Equal(t, 100, total, "counts must sum to mention_count")
}

func TestGroupTimeSeriesByDate_ZeroMentions(t *testing.T) {
	bearish := 0.0
	neutral := 0.0
	points := []models.SentimentTimeSeriesPoint{
		{
			Time:           time.Date(2026, 2, 20, 14, 0, 0, 0, time.UTC),
			SentimentScore: 0.0,
			BullishPct:     0.0,
			BearishPct:     &bearish,
			NeutralPct:     &neutral,
			MentionCount:   0,
		},
	}

	result := groupTimeSeriesByDate(points)
	require.Len(t, result, 1)
	assert.Equal(t, 0, result[0].PostCount)
	assert.Equal(t, 0, result[0].Bullish)
	assert.Equal(t, 0, result[0].Bearish)
	assert.Equal(t, 0, result[0].Neutral)
}
