package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"investorcenter-api/models"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// newPolygonFinancialsTestClient creates a PolygonFinancialsClient pointing at
// the mock server. It uses a fast rate-limiter so tests don't block.
func newPolygonFinancialsTestClient() *PolygonFinancialsClient {
	return &PolygonFinancialsClient{
		PolygonClient: &PolygonClient{
			APIKey: "test-key",
			Client: &http.Client{Timeout: 5 * time.Second},
		},
		rateLimiter: NewRateLimiter(100, time.Minute), // generous for tests
	}
}

// sampleFinancialsResponse builds a minimal valid PolygonFinancialsResponse.
func sampleFinancialsResponse() models.PolygonFinancialsResponse {
	return models.PolygonFinancialsResponse{
		Status:    "OK",
		RequestID: "req-123",
		Count:     1,
		Results: []models.PolygonFinancialsData{
			{
				Ticker:       "AAPL",
				CIK:          "0000320193",
				CompanyName:  "Apple Inc.",
				StartDate:    "2024-01-01",
				EndDate:      "2024-03-31",
				FilingDate:   "2024-05-01",
				FiscalPeriod: "Q1",
				FiscalYear:   "2024",
				Financials: models.PolygonFinancialsItems{
					IncomeStatement: map[string]models.PolygonFinancialValue{
						"revenues": {Value: 90753000000, Label: "Revenues", Unit: "USD"},
					},
					BalanceSheet: map[string]models.PolygonFinancialValue{
						"assets": {Value: 352583000000, Label: "Total Assets", Unit: "USD"},
					},
					CashFlowStatement: map[string]models.PolygonFinancialValue{
						"net_cash_flow": {Value: 28000000000, Label: "Net Cash Flow", Unit: "USD"},
					},
				},
			},
		},
	}
}

// ===========================================================================
// GetIncomeStatements
// ===========================================================================

func TestPolygonFinancials_HTTP_GetIncomeStatements_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/vX/reference/financials")
		assert.Equal(t, "AAPL", r.URL.Query().Get("ticker"))
		assert.Equal(t, "quarterly", r.URL.Query().Get("timeframe"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sampleFinancialsResponse())
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonFinancialsTestClient()
	params := FinancialsRequestParams{
		Ticker:    "AAPL",
		Timeframe: "quarterly",
		Limit:     5,
	}

	result, err := client.GetIncomeStatements(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "OK", result.Status)
	assert.Len(t, result.Results, 1)
	assert.Equal(t, "AAPL", result.Results[0].Ticker)
	assert.Equal(t, "Q1", result.Results[0].FiscalPeriod)
}

func TestPolygonFinancials_HTTP_GetIncomeStatements_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonFinancialsTestClient()
	params := FinancialsRequestParams{Ticker: "AAPL", Timeframe: "quarterly"}

	_, err := client.GetIncomeStatements(context.Background(), params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestPolygonFinancials_HTTP_GetIncomeStatements_BadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonFinancialsTestClient()
	params := FinancialsRequestParams{Ticker: "AAPL", Timeframe: "quarterly"}

	_, err := client.GetIncomeStatements(context.Background(), params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestPolygonFinancials_HTTP_GetIncomeStatements_ConnectionError(t *testing.T) {
	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = "http://localhost:99999"

	client := &PolygonFinancialsClient{
		PolygonClient: &PolygonClient{
			APIKey: "test-key",
			Client: &http.Client{Timeout: 1 * time.Second},
		},
		rateLimiter: NewRateLimiter(100, time.Minute),
	}
	params := FinancialsRequestParams{Ticker: "AAPL", Timeframe: "quarterly"}

	_, err := client.GetIncomeStatements(context.Background(), params)
	assert.Error(t, err)
}

// ===========================================================================
// GetBalanceSheets
// ===========================================================================

func TestPolygonFinancials_HTTP_GetBalanceSheets_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sampleFinancialsResponse())
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonFinancialsTestClient()
	params := FinancialsRequestParams{
		Ticker:    "AAPL",
		Timeframe: "annual",
		Limit:     10,
	}

	result, err := client.GetBalanceSheets(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Results, 1)
}

func TestPolygonFinancials_HTTP_GetBalanceSheets_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("unavailable"))
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonFinancialsTestClient()
	params := FinancialsRequestParams{Ticker: "AAPL", Timeframe: "annual"}

	_, err := client.GetBalanceSheets(context.Background(), params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 503")
}

// ===========================================================================
// GetCashFlowStatements
// ===========================================================================

func TestPolygonFinancials_HTTP_GetCashFlowStatements_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sampleFinancialsResponse())
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonFinancialsTestClient()
	params := FinancialsRequestParams{
		Ticker:    "MSFT",
		Timeframe: "quarterly",
		Limit:     5,
	}

	result, err := client.GetCashFlowStatements(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Results, 1)
}

func TestPolygonFinancials_HTTP_GetCashFlowStatements_BadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{broken"))
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonFinancialsTestClient()
	params := FinancialsRequestParams{Ticker: "MSFT", Timeframe: "quarterly"}

	_, err := client.GetCashFlowStatements(context.Background(), params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

// ===========================================================================
// GetRatios
// ===========================================================================

func TestPolygonFinancials_HTTP_GetRatios_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/vX/reference/financials")

		response := models.PolygonRatiosResponse{
			Status:    "OK",
			RequestID: "req-456",
			Count:     1,
			Results: []models.PolygonRatiosData{
				{
					Ticker:       "AAPL",
					CIK:          "0000320193",
					CompanyName:  "Apple Inc.",
					StartDate:    "2024-01-01",
					EndDate:      "2024-03-31",
					FiscalPeriod: "Q1",
					FiscalYear:   "2024",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonFinancialsTestClient()
	params := FinancialsRequestParams{
		Ticker:    "AAPL",
		Timeframe: "quarterly",
		Limit:     5,
	}

	result, err := client.GetRatios(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "OK", result.Status)
	assert.Len(t, result.Results, 1)
	assert.Equal(t, "AAPL", result.Results[0].Ticker)
}

func TestPolygonFinancials_HTTP_GetRatios_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonFinancialsTestClient()
	params := FinancialsRequestParams{Ticker: "AAPL", Timeframe: "quarterly"}

	_, err := client.GetRatios(context.Background(), params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 403")
}

func TestPolygonFinancials_HTTP_GetRatios_BadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json at all"))
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonFinancialsTestClient()
	params := FinancialsRequestParams{Ticker: "AAPL", Timeframe: "quarterly"}

	_, err := client.GetRatios(context.Background(), params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

// ===========================================================================
// GetAllFinancialsWithPagination
// ===========================================================================

func TestPolygonFinancials_HTTP_GetAllFinancialsWithPagination_SinglePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := sampleFinancialsResponse()
		resp.NextURL = nil // No more pages

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonFinancialsTestClient()
	params := FinancialsRequestParams{
		Ticker:    "AAPL",
		Timeframe: "quarterly",
		Limit:     5,
	}

	results, err := client.GetAllFinancialsWithPagination(context.Background(), params)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "AAPL", results[0].Ticker)
}

func TestPolygonFinancials_HTTP_GetAllFinancialsWithPagination_MultiPage(t *testing.T) {
	pageCount := 0
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageCount++

		resp := sampleFinancialsResponse()
		resp.Results[0].FiscalPeriod = "Q" + string(rune('0'+pageCount))

		if pageCount < 3 {
			// Point to the full absolute URL for the next page
			nextURL := serverURL + r.URL.Path + "?" + r.URL.RawQuery
			resp.NextURL = &nextURL
		} else {
			resp.NextURL = nil
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()
	serverURL = server.URL

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonFinancialsTestClient()
	params := FinancialsRequestParams{
		Ticker:    "AAPL",
		Timeframe: "quarterly",
		Limit:     100,
	}

	results, err := client.GetAllFinancialsWithPagination(context.Background(), params)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Equal(t, 3, pageCount)
}

func TestPolygonFinancials_HTTP_GetAllFinancialsWithPagination_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonFinancialsTestClient()
	params := FinancialsRequestParams{Ticker: "AAPL", Timeframe: "quarterly"}

	_, err := client.GetAllFinancialsWithPagination(context.Background(), params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestPolygonFinancials_HTTP_GetAllFinancialsWithPagination_BadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{not json}"))
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonFinancialsTestClient()
	params := FinancialsRequestParams{Ticker: "AAPL", Timeframe: "quarterly"}

	_, err := client.GetAllFinancialsWithPagination(context.Background(), params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestPolygonFinancials_HTTP_GetAllFinancialsWithPagination_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := models.PolygonFinancialsResponse{
			Status:  "OK",
			Count:   0,
			Results: []models.PolygonFinancialsData{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonFinancialsTestClient()
	params := FinancialsRequestParams{Ticker: "FAKE", Timeframe: "quarterly"}

	results, err := client.GetAllFinancialsWithPagination(context.Background(), params)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestPolygonFinancials_HTTP_GetAllFinancialsWithPagination_ConnectionError(t *testing.T) {
	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = "http://localhost:99999"

	client := &PolygonFinancialsClient{
		PolygonClient: &PolygonClient{
			APIKey: "test-key",
			Client: &http.Client{Timeout: 1 * time.Second},
		},
		rateLimiter: NewRateLimiter(100, time.Minute),
	}
	params := FinancialsRequestParams{Ticker: "AAPL", Timeframe: "quarterly"}

	_, err := client.GetAllFinancialsWithPagination(context.Background(), params)
	assert.Error(t, err)
}

// ===========================================================================
// Context cancellation
// ===========================================================================

func TestPolygonFinancials_HTTP_GetIncomeStatements_CancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a slow server
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sampleFinancialsResponse())
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonFinancialsTestClient()
	params := FinancialsRequestParams{Ticker: "AAPL", Timeframe: "quarterly"}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.GetIncomeStatements(ctx, params)
	assert.Error(t, err)
}

// ===========================================================================
// Empty NextURL string
// ===========================================================================

func TestPolygonFinancials_HTTP_GetAllFinancialsWithPagination_EmptyNextURL(t *testing.T) {
	emptyStr := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := sampleFinancialsResponse()
		resp.NextURL = &emptyStr // Empty string, not nil

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonFinancialsTestClient()
	params := FinancialsRequestParams{Ticker: "AAPL", Timeframe: "quarterly"}

	results, err := client.GetAllFinancialsWithPagination(context.Background(), params)
	require.NoError(t, err)
	assert.Len(t, results, 1, "should stop when NextURL is empty string")
}
