package ycharts

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Helper: build a gin test router with a handler, optionally injecting user_id
// ---------------------------------------------------------------------------

// setupRouter creates a test router for a handler with URL params.
// If userID is non-empty, it is set in gin context (simulating auth middleware).
func setupRouter(method, path string, handler gin.HandlerFunc, userID string) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if userID != "" {
			c.Set("user_id", userID)
		}
		c.Next()
	})

	switch method {
	case "POST":
		r.POST(path, handler)
	default:
		r.POST(path, handler)
	}
	return r
}

func doPost(router *gin.Engine, url string, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", url, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	return w
}

func parseResp(w *httptest.ResponseRecorder) map[string]interface{} {
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp
}

// ===========================================================================
// PostKeyStats tests
// ===========================================================================

func TestPostKeyStats_MissingUserContext(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/key_stats/:ticker", PostKeyStats, "")
	w := doPost(router, "/ingest/ycharts/key_stats/AAPL", `{"collected_at":"2026-02-12T20:30:00Z","source_url":"https://example.com"}`)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	resp := parseResp(w)
	assert.Equal(t, "Unauthorized", resp["error"])
}

func TestPostKeyStats_TickerTooLong(t *testing.T) {
	longTicker := strings.Repeat("A", 21)
	router := setupRouter("POST", "/ingest/ycharts/key_stats/:ticker", PostKeyStats, "user-1")
	w := doPost(router, "/ingest/ycharts/key_stats/"+longTicker, `{"collected_at":"2026-02-12T20:30:00Z","source_url":"https://example.com"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "Ticker must be 1-20 characters")
}

func TestPostKeyStats_InvalidJSON(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/key_stats/:ticker", PostKeyStats, "user-1")
	w := doPost(router, "/ingest/ycharts/key_stats/AAPL", "not-json")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "Invalid JSON")
}

func TestPostKeyStats_MissingCollectedAt(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/key_stats/:ticker", PostKeyStats, "user-1")
	w := doPost(router, "/ingest/ycharts/key_stats/AAPL", `{"source_url":"https://example.com"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "collected_at is required")
}

func TestPostKeyStats_EmptyCollectedAt(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/key_stats/:ticker", PostKeyStats, "user-1")
	w := doPost(router, "/ingest/ycharts/key_stats/AAPL", `{"collected_at":"","source_url":"https://example.com"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "collected_at is required")
}

func TestPostKeyStats_MissingSourceURL(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/key_stats/:ticker", PostKeyStats, "user-1")
	w := doPost(router, "/ingest/ycharts/key_stats/AAPL", `{"collected_at":"2026-02-12T20:30:00Z"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "source_url is required")
}

func TestPostKeyStats_InvalidCollectedAtFormat(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/key_stats/:ticker", PostKeyStats, "user-1")
	w := doPost(router, "/ingest/ycharts/key_stats/AAPL", `{"collected_at":"2026-02-12","source_url":"https://example.com"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "RFC3339")
}

func TestPostKeyStats_InvalidCollectedAtGarbage(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/key_stats/:ticker", PostKeyStats, "user-1")
	w := doPost(router, "/ingest/ycharts/key_stats/AAPL", `{"collected_at":"not-a-date","source_url":"https://example.com"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "RFC3339")
}

func TestPostKeyStats_EmptyBody(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/key_stats/:ticker", PostKeyStats, "user-1")
	w := doPost(router, "/ingest/ycharts/key_stats/AAPL", "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostKeyStats_TickerUppercased(t *testing.T) {
	// Validation passes through ticker validation (lowercase is uppercased).
	// The handler should still fail later at schema validation (no schema file in test),
	// but ticker validation itself should not reject lowercase input.
	router := setupRouter("POST", "/ingest/ycharts/key_stats/:ticker", PostKeyStats, "user-1")
	w := doPost(router, "/ingest/ycharts/key_stats/aapl", `{"collected_at":"2026-02-12T20:30:00Z","source_url":"https://example.com"}`)

	// Should NOT be a ticker validation error — will fail later at schema validation
	resp := parseResp(w)
	if w.Code == http.StatusBadRequest {
		assert.NotContains(t, resp["error"], "Ticker must be 1-20 characters")
	}
}

// ===========================================================================
// PostFinancials tests
// ===========================================================================

func TestPostFinancials_MissingUserContext(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/financials/:statement/:ticker", PostFinancials, "")
	w := doPost(router, "/ingest/ycharts/financials/income_statement/AAPL", `{}`)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	resp := parseResp(w)
	assert.Equal(t, "Unauthorized", resp["error"])
}

func TestPostFinancials_InvalidStatementType(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/financials/:statement/:ticker", PostFinancials, "user-1")
	w := doPost(router, "/ingest/ycharts/financials/invalid_stmt/AAPL", `{}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "Invalid statement type")
	assert.Contains(t, resp["error"], "income_statement, balance_sheet, cash_flow")
}

func TestPostFinancials_ValidStatementTypes(t *testing.T) {
	validTypes := []string{"income_statement", "balance_sheet", "cash_flow"}
	for _, st := range validTypes {
		t.Run(st, func(t *testing.T) {
			router := setupRouter("POST", "/ingest/ycharts/financials/:statement/:ticker", PostFinancials, "user-1")
			// Send a body that will pass statement validation but fail at the next check
			w := doPost(router, "/ingest/ycharts/financials/"+st+"/AAPL", `{"source_url":"https://example.com"}`)

			// Should NOT be a statement type error
			resp := parseResp(w)
			assert.NotContains(t, resp["error"], "Invalid statement type")
		})
	}
}

func TestPostFinancials_TickerTooLong(t *testing.T) {
	longTicker := strings.Repeat("A", 21)
	router := setupRouter("POST", "/ingest/ycharts/financials/:statement/:ticker", PostFinancials, "user-1")
	w := doPost(router, "/ingest/ycharts/financials/income_statement/"+longTicker, `{}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "Ticker must be 1-20 characters")
}

func TestPostFinancials_InvalidJSON(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/financials/:statement/:ticker", PostFinancials, "user-1")
	w := doPost(router, "/ingest/ycharts/financials/income_statement/AAPL", "not-json")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "Invalid JSON")
}

func TestPostFinancials_MissingPeriod(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/financials/:statement/:ticker", PostFinancials, "user-1")
	w := doPost(router, "/ingest/ycharts/financials/income_statement/AAPL", `{"source_url":"https://example.com","period_type":"quarterly"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "period is required")
}

func TestPostFinancials_InvalidPeriodFormat(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/financials/:statement/:ticker", PostFinancials, "user-1")

	cases := []struct {
		name   string
		period string
	}{
		{"full date", "2025-09-15"},
		{"year only", "2025"},
		{"garbage", "not-a-period"},
		{"wrong separator", "2025/09"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"period":"` + tc.period + `","period_type":"quarterly","source_url":"https://example.com"}`
			w := doPost(router, "/ingest/ycharts/financials/income_statement/AAPL", body)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			resp := parseResp(w)
			assert.Contains(t, resp["error"], "period must be in YYYY-MM format")
		})
	}
}

func TestPostFinancials_MissingPeriodType(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/financials/:statement/:ticker", PostFinancials, "user-1")
	w := doPost(router, "/ingest/ycharts/financials/income_statement/AAPL", `{"period":"2025-09","source_url":"https://example.com"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "period_type is required")
}

func TestPostFinancials_InvalidPeriodType(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/financials/:statement/:ticker", PostFinancials, "user-1")
	w := doPost(router, "/ingest/ycharts/financials/income_statement/AAPL", `{"period":"2025-09","period_type":"monthly","source_url":"https://example.com"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "period_type must be one of: quarterly, annual, ttm")
}

func TestPostFinancials_MissingSourceURL(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/financials/:statement/:ticker", PostFinancials, "user-1")
	w := doPost(router, "/ingest/ycharts/financials/income_statement/AAPL", `{"period":"2025-09","period_type":"quarterly"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "source_url is required")
}

func TestPostFinancials_EmptyBody(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/financials/:statement/:ticker", PostFinancials, "user-1")
	w := doPost(router, "/ingest/ycharts/financials/income_statement/AAPL", "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ===========================================================================
// PostAnalystEstimates tests
// ===========================================================================

func TestPostAnalystEstimates_MissingUserContext(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/analyst_estimates/:ticker", PostAnalystEstimates, "")
	w := doPost(router, "/ingest/ycharts/analyst_estimates/AAPL", `{}`)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	resp := parseResp(w)
	assert.Equal(t, "Unauthorized", resp["error"])
}

func TestPostAnalystEstimates_TickerTooLong(t *testing.T) {
	longTicker := strings.Repeat("A", 21)
	router := setupRouter("POST", "/ingest/ycharts/analyst_estimates/:ticker", PostAnalystEstimates, "user-1")
	w := doPost(router, "/ingest/ycharts/analyst_estimates/"+longTicker, `{}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "Ticker must be 1-20 characters")
}

func TestPostAnalystEstimates_InvalidJSON(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/analyst_estimates/:ticker", PostAnalystEstimates, "user-1")
	w := doPost(router, "/ingest/ycharts/analyst_estimates/AAPL", "not-json")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "Invalid JSON")
}

func TestPostAnalystEstimates_MissingAsOfDate(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/analyst_estimates/:ticker", PostAnalystEstimates, "user-1")
	w := doPost(router, "/ingest/ycharts/analyst_estimates/AAPL", `{"source_url":"https://example.com"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "as_of_date is required")
}

func TestPostAnalystEstimates_EmptyAsOfDate(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/analyst_estimates/:ticker", PostAnalystEstimates, "user-1")
	w := doPost(router, "/ingest/ycharts/analyst_estimates/AAPL", `{"as_of_date":"","source_url":"https://example.com"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "as_of_date is required")
}

func TestPostAnalystEstimates_InvalidAsOfDateFormat(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/analyst_estimates/:ticker", PostAnalystEstimates, "user-1")

	cases := []struct {
		name string
		date string
	}{
		{"RFC3339 timestamp", "2026-02-12T20:30:00Z"},
		{"year-month only", "2026-02"},
		{"garbage", "not-a-date"},
		{"slash format", "2026/02/12"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"as_of_date":"` + tc.date + `","source_url":"https://example.com"}`
			w := doPost(router, "/ingest/ycharts/analyst_estimates/AAPL", body)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			resp := parseResp(w)
			assert.Contains(t, resp["error"], "as_of_date must be in YYYY-MM-DD format")
		})
	}
}

func TestPostAnalystEstimates_MissingSourceURL(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/analyst_estimates/:ticker", PostAnalystEstimates, "user-1")
	w := doPost(router, "/ingest/ycharts/analyst_estimates/AAPL", `{"as_of_date":"2026-02-12"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "source_url is required")
}

func TestPostAnalystEstimates_EmptyBody(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/analyst_estimates/:ticker", PostAnalystEstimates, "user-1")
	w := doPost(router, "/ingest/ycharts/analyst_estimates/AAPL", "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ===========================================================================
// PostValuation tests
// ===========================================================================

func TestPostValuation_MissingUserContext(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/valuation/:ticker", PostValuation, "")
	w := doPost(router, "/ingest/ycharts/valuation/AAPL", `{}`)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	resp := parseResp(w)
	assert.Equal(t, "Unauthorized", resp["error"])
}

func TestPostValuation_TickerTooLong(t *testing.T) {
	longTicker := strings.Repeat("A", 21)
	router := setupRouter("POST", "/ingest/ycharts/valuation/:ticker", PostValuation, "user-1")
	w := doPost(router, "/ingest/ycharts/valuation/"+longTicker, `{}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "Ticker must be 1-20 characters")
}

func TestPostValuation_InvalidJSON(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/valuation/:ticker", PostValuation, "user-1")
	w := doPost(router, "/ingest/ycharts/valuation/AAPL", "not-json")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "Invalid JSON")
}

func TestPostValuation_MissingAsOfDate(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/valuation/:ticker", PostValuation, "user-1")
	w := doPost(router, "/ingest/ycharts/valuation/AAPL", `{"source_url":"https://example.com"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "as_of_date is required")
}

func TestPostValuation_EmptyAsOfDate(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/valuation/:ticker", PostValuation, "user-1")
	w := doPost(router, "/ingest/ycharts/valuation/AAPL", `{"as_of_date":"","source_url":"https://example.com"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "as_of_date is required")
}

func TestPostValuation_InvalidAsOfDateFormat(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/valuation/:ticker", PostValuation, "user-1")

	cases := []struct {
		name string
		date string
	}{
		{"RFC3339 timestamp", "2026-02-12T20:30:00Z"},
		{"year-month only", "2026-02"},
		{"garbage", "not-a-date"},
		{"slash format", "2026/02/12"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"as_of_date":"` + tc.date + `","source_url":"https://example.com"}`
			w := doPost(router, "/ingest/ycharts/valuation/AAPL", body)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			resp := parseResp(w)
			assert.Contains(t, resp["error"], "as_of_date must be in YYYY-MM-DD format")
		})
	}
}

func TestPostValuation_MissingSourceURL(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/valuation/:ticker", PostValuation, "user-1")
	w := doPost(router, "/ingest/ycharts/valuation/AAPL", `{"as_of_date":"2026-02-12"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "source_url is required")
}

func TestPostValuation_EmptyBody(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/valuation/:ticker", PostValuation, "user-1")
	w := doPost(router, "/ingest/ycharts/valuation/AAPL", "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ===========================================================================
// PostPerformance tests
// ===========================================================================

func TestPostPerformance_MissingUserContext(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/performance/:ticker", PostPerformance, "")
	w := doPost(router, "/ingest/ycharts/performance/AAPL", `{}`)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	resp := parseResp(w)
	assert.Equal(t, "Unauthorized", resp["error"])
}

func TestPostPerformance_TickerTooLong(t *testing.T) {
	longTicker := strings.Repeat("A", 21)
	router := setupRouter("POST", "/ingest/ycharts/performance/:ticker", PostPerformance, "user-1")
	w := doPost(router, "/ingest/ycharts/performance/"+longTicker, `{}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "Ticker must be 1-20 characters")
}

func TestPostPerformance_InvalidJSON(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/performance/:ticker", PostPerformance, "user-1")
	w := doPost(router, "/ingest/ycharts/performance/AAPL", "not-json")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "Invalid JSON")
}

func TestPostPerformance_MissingAsOfDate(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/performance/:ticker", PostPerformance, "user-1")
	w := doPost(router, "/ingest/ycharts/performance/AAPL", `{"source_url":"https://example.com"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "as_of_date is required")
}

func TestPostPerformance_EmptyAsOfDate(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/performance/:ticker", PostPerformance, "user-1")
	w := doPost(router, "/ingest/ycharts/performance/AAPL", `{"as_of_date":"","source_url":"https://example.com"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "as_of_date is required")
}

func TestPostPerformance_InvalidAsOfDateFormat(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/performance/:ticker", PostPerformance, "user-1")

	cases := []struct {
		name string
		date string
	}{
		{"RFC3339 timestamp", "2026-02-12T20:30:00Z"},
		{"year-month only", "2026-02"},
		{"garbage", "not-a-date"},
		{"slash format", "2026/02/12"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"as_of_date":"` + tc.date + `","source_url":"https://example.com"}`
			w := doPost(router, "/ingest/ycharts/performance/AAPL", body)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			resp := parseResp(w)
			assert.Contains(t, resp["error"], "as_of_date must be in YYYY-MM-DD format")
		})
	}
}

func TestPostPerformance_MissingSourceURL(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/performance/:ticker", PostPerformance, "user-1")
	w := doPost(router, "/ingest/ycharts/performance/AAPL", `{"as_of_date":"2026-02-12"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResp(w)
	assert.Contains(t, resp["error"], "source_url is required")
}

func TestPostPerformance_EmptyBody(t *testing.T) {
	router := setupRouter("POST", "/ingest/ycharts/performance/:ticker", PostPerformance, "user-1")
	w := doPost(router, "/ingest/ycharts/performance/AAPL", "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ===========================================================================
// Cross-cutting: regex pattern tests
// ===========================================================================

func TestPeriodRegex(t *testing.T) {
	t.Run("matches valid YYYY-MM formats", func(t *testing.T) {
		assert.True(t, periodRegex.MatchString("2025-09"))
		assert.True(t, periodRegex.MatchString("2000-01"))
		assert.True(t, periodRegex.MatchString("1999-12"))
	})

	t.Run("rejects invalid formats", func(t *testing.T) {
		assert.False(t, periodRegex.MatchString("2025-9"))
		assert.False(t, periodRegex.MatchString("25-09"))
		assert.False(t, periodRegex.MatchString("2025/09"))
		assert.False(t, periodRegex.MatchString("2025-09-01"))
		assert.False(t, periodRegex.MatchString(""))
		assert.False(t, periodRegex.MatchString("abcd-ef"))
	})
}

func TestAsOfDateRegex(t *testing.T) {
	t.Run("matches valid YYYY-MM-DD formats", func(t *testing.T) {
		assert.True(t, asOfDateRegex.MatchString("2026-02-12"))
		assert.True(t, asOfDateRegex.MatchString("2000-01-01"))
		assert.True(t, asOfDateRegex.MatchString("1999-12-31"))
	})

	t.Run("rejects invalid formats", func(t *testing.T) {
		assert.False(t, asOfDateRegex.MatchString("2026-2-12"))
		assert.False(t, asOfDateRegex.MatchString("26-02-12"))
		assert.False(t, asOfDateRegex.MatchString("2026/02/12"))
		assert.False(t, asOfDateRegex.MatchString("2026-02"))
		assert.False(t, asOfDateRegex.MatchString(""))
		assert.False(t, asOfDateRegex.MatchString("abcd-ef-gh"))
		assert.False(t, asOfDateRegex.MatchString("2026-02-12T00:00:00Z"))
	})
}

// ===========================================================================
// Financial statement schema map tests
// ===========================================================================

func TestFinancialStatementSchemas(t *testing.T) {
	t.Run("has exactly 3 valid statement types", func(t *testing.T) {
		assert.Len(t, financialStatementSchemas, 3)
	})

	t.Run("contains income_statement", func(t *testing.T) {
		path, ok := financialStatementSchemas["income_statement"]
		assert.True(t, ok)
		assert.Contains(t, path, "income_statement")
	})

	t.Run("contains balance_sheet", func(t *testing.T) {
		path, ok := financialStatementSchemas["balance_sheet"]
		assert.True(t, ok)
		assert.Contains(t, path, "balance_sheet")
	})

	t.Run("contains cash_flow", func(t *testing.T) {
		path, ok := financialStatementSchemas["cash_flow"]
		assert.True(t, ok)
		assert.Contains(t, path, "cash_flow")
	})

	t.Run("does not contain invalid type", func(t *testing.T) {
		_, ok := financialStatementSchemas["invalid"]
		assert.False(t, ok)
	})
}
