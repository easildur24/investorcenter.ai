package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// newFMPTestClient creates an FMPClient pointing at the given test server URL.
func newFMPTestClient(serverURL string) *FMPClient {
	return &FMPClient{
		APIKey: "test-key",
		Client: &http.Client{Timeout: 5 * time.Second},
	}
}

// saveFMPBaseURL saves the current FMPBaseURL and returns a restore function.
func saveFMPBaseURL() func() {
	orig := FMPBaseURL
	return func() { FMPBaseURL = orig }
}

// ===========================================================================
// GetRatiosTTM
// ===========================================================================

func TestFMP_GetRatiosTTM_Success(t *testing.T) {
	pe := 25.5
	gm := 0.45

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/ratios-ttm")
		assert.Equal(t, "AAPL", r.URL.Query().Get("symbol"))
		assert.Equal(t, "test-key", r.URL.Query().Get("apikey"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]FMPRatiosTTM{
			{
				Symbol:                  "AAPL",
				PriceToEarningsRatioTTM: &pe,
				GrossProfitMarginTTM:    &gm,
			},
		})
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	result, err := client.GetRatiosTTM("AAPL")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "AAPL", result.Symbol)
	require.NotNil(t, result.PriceToEarningsRatioTTM)
	assert.InDelta(t, 25.5, *result.PriceToEarningsRatioTTM, 0.01)
	require.NotNil(t, result.GrossProfitMarginTTM)
	assert.InDelta(t, 0.45, *result.GrossProfitMarginTTM, 0.01)
}

func TestFMP_GetRatiosTTM_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]FMPRatiosTTM{})
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	_, err := client.GetRatiosTTM("FAKE")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no FMP data found")
}

func TestFMP_GetRatiosTTM_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	_, err := client.GetRatiosTTM("AAPL")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestFMP_GetRatiosTTM_BadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	_, err := client.GetRatiosTTM("AAPL")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestFMP_GetRatiosTTM_ConnectionError(t *testing.T) {
	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = "http://localhost:99999"

	client := &FMPClient{
		APIKey: "test-key",
		Client: &http.Client{Timeout: 1 * time.Second},
	}

	_, err := client.GetRatiosTTM("AAPL")
	assert.Error(t, err)
}

// ===========================================================================
// GetKeyMetricsTTM
// ===========================================================================

func TestFMP_GetKeyMetricsTTM_Success(t *testing.T) {
	mc := 3000000000000.0
	ev := 3100000000000.0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/key-metrics-ttm")
		assert.Equal(t, "MSFT", r.URL.Query().Get("symbol"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]FMPKeyMetricsTTM{
			{
				Symbol:             "MSFT",
				MarketCapTTM:       &mc,
				EnterpriseValueTTM: &ev,
			},
		})
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	result, err := client.GetKeyMetricsTTM("MSFT")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "MSFT", result.Symbol)
	require.NotNil(t, result.MarketCapTTM)
	assert.InDelta(t, 3e12, *result.MarketCapTTM, 1e9)
}

func TestFMP_GetKeyMetricsTTM_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]FMPKeyMetricsTTM{})
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	_, err := client.GetKeyMetricsTTM("FAKE")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no FMP key-metrics-ttm data")
}

func TestFMP_GetKeyMetricsTTM_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	_, err := client.GetKeyMetricsTTM("AAPL")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 503")
}

// ===========================================================================
// GetFinancialGrowth
// ===========================================================================

func TestFMP_GetFinancialGrowth_Success(t *testing.T) {
	rg := 0.15
	eg := 0.22

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/financial-growth")
		assert.Equal(t, "AAPL", r.URL.Query().Get("symbol"))
		assert.Equal(t, "5", r.URL.Query().Get("limit"))
		assert.Equal(t, "annual", r.URL.Query().Get("period"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]FMPFinancialGrowth{
			{Symbol: "AAPL", Date: "2024-09-28", RevenueGrowth: &rg, EPSGrowth: &eg},
			{Symbol: "AAPL", Date: "2023-09-30", RevenueGrowth: &rg},
		})
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	results, err := client.GetFinancialGrowth("AAPL", 5)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "AAPL", results[0].Symbol)
	require.NotNil(t, results[0].RevenueGrowth)
	assert.InDelta(t, 0.15, *results[0].RevenueGrowth, 0.001)
}

func TestFMP_GetFinancialGrowth_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]FMPFinancialGrowth{})
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	results, err := client.GetFinancialGrowth("FAKE", 5)
	require.NoError(t, err) // empty is not an error for this endpoint
	assert.Empty(t, results)
}

func TestFMP_GetFinancialGrowth_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	_, err := client.GetFinancialGrowth("AAPL", 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 429")
}

// ===========================================================================
// GetAnalystEstimates
// ===========================================================================

func TestFMP_GetAnalystEstimates_Success(t *testing.T) {
	epsAvg := 6.5
	revAvg := 400000000000.0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/analyst-estimates")
		assert.Equal(t, "AAPL", r.URL.Query().Get("symbol"))
		assert.Equal(t, "4", r.URL.Query().Get("limit"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]FMPAnalystEstimate{
			{
				Symbol:              "AAPL",
				Date:                "2025-09-30",
				EstimatedEPSAvg:     &epsAvg,
				EstimatedRevenueAvg: &revAvg,
			},
		})
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	results, err := client.GetAnalystEstimates("AAPL", 4)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	require.NotNil(t, results[0].EstimatedEPSAvg)
	assert.InDelta(t, 6.5, *results[0].EstimatedEPSAvg, 0.01)
}

func TestFMP_GetAnalystEstimates_BadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{broken"))
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	_, err := client.GetAnalystEstimates("AAPL", 4)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

// ===========================================================================
// GetScore
// ===========================================================================

func TestFMP_GetScore_Success(t *testing.T) {
	zScore := 3.8
	fScore := 7

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/score")
		assert.Equal(t, "AAPL", r.URL.Query().Get("symbol"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]FMPScore{
			{
				Symbol:         "AAPL",
				AltmanZScore:   &zScore,
				PiotroskiScore: &fScore,
			},
		})
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	result, err := client.GetScore("AAPL")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.AltmanZScore)
	assert.InDelta(t, 3.8, *result.AltmanZScore, 0.01)
	require.NotNil(t, result.PiotroskiScore)
	assert.Equal(t, 7, *result.PiotroskiScore)
}

func TestFMP_GetScore_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]FMPScore{})
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	_, err := client.GetScore("FAKE")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no FMP score data")
}

func TestFMP_GetScore_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	_, err := client.GetScore("AAPL")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 403")
}

// ===========================================================================
// GetDividendHistory
// ===========================================================================

func TestFMP_GetDividendHistory_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/historical-price-eod/dividend/AAPL")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"historical": []FMPDividendHistorical{
				{Symbol: "AAPL", Date: "2024-11-08", Dividend: 0.25, AdjDividend: 0.25},
				{Symbol: "AAPL", Date: "2024-08-12", Dividend: 0.25, AdjDividend: 0.25},
				{Symbol: "AAPL", Date: "2024-05-10", Dividend: 0.24, AdjDividend: 0.24},
				{Symbol: "AAPL", Date: "2024-02-09", Dividend: 0.24, AdjDividend: 0.24},
			},
		})
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	results, err := client.GetDividendHistory("AAPL")
	require.NoError(t, err)
	assert.Len(t, results, 4)
	assert.Equal(t, "2024-11-08", results[0].Date)
	assert.InDelta(t, 0.25, results[0].Dividend, 0.001)
}

func TestFMP_GetDividendHistory_EmptyHistorical(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"historical": []FMPDividendHistorical{},
		})
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	results, err := client.GetDividendHistory("FAKE")
	require.NoError(t, err) // empty is not an error
	assert.Empty(t, results)
}

func TestFMP_GetDividendHistory_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	_, err := client.GetDividendHistory("AAPL")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 502")
}

// ===========================================================================
// GetGradesSummary
// ===========================================================================

func TestFMP_GetGradesSummary_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/grades-summary")
		assert.Equal(t, "AAPL", r.URL.Query().Get("symbol"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]FMPGradesSummary{
			{
				Symbol:     "AAPL",
				StrongBuy:  12,
				Buy:        20,
				Hold:       5,
				Sell:       1,
				StrongSell: 0,
				Consensus:  "Buy",
			},
		})
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	result, err := client.GetGradesSummary("AAPL")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "AAPL", result.Symbol)
	assert.Equal(t, 12, result.StrongBuy)
	assert.Equal(t, 20, result.Buy)
	assert.Equal(t, "Buy", result.Consensus)
}

func TestFMP_GetGradesSummary_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]FMPGradesSummary{})
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	_, err := client.GetGradesSummary("FAKE")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no FMP grades-summary data")
}

// ===========================================================================
// GetPriceTargetConsensus
// ===========================================================================

func TestFMP_GetPriceTargetConsensus_Success(t *testing.T) {
	high := 250.0
	low := 180.0
	consensus := 220.0
	median := 215.0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/price-target-consensus")
		assert.Equal(t, "AAPL", r.URL.Query().Get("symbol"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]FMPPriceTargetConsensus{
			{
				Symbol:          "AAPL",
				TargetHigh:      &high,
				TargetLow:       &low,
				TargetConsensus: &consensus,
				TargetMedian:    &median,
			},
		})
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	result, err := client.GetPriceTargetConsensus("AAPL")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.TargetHigh)
	assert.InDelta(t, 250.0, *result.TargetHigh, 0.01)
	require.NotNil(t, result.TargetLow)
	assert.InDelta(t, 180.0, *result.TargetLow, 0.01)
	require.NotNil(t, result.TargetConsensus)
	assert.InDelta(t, 220.0, *result.TargetConsensus, 0.01)
}

func TestFMP_GetPriceTargetConsensus_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]FMPPriceTargetConsensus{})
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	_, err := client.GetPriceTargetConsensus("FAKE")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no FMP price-target-consensus data")
}

func TestFMP_GetPriceTargetConsensus_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	_, err := client.GetPriceTargetConsensus("AAPL")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 504")
}

// ===========================================================================
// GetAllMetrics — integration-style test with mock server
// ===========================================================================

func TestFMP_GetAllMetrics_Success(t *testing.T) {
	pe := 28.0
	mc := 3000000000000.0
	rg := 0.10
	epsAvg := 7.0
	zScore := 4.0
	fScore := 8
	high := 250.0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		path := r.URL.Path
		switch {
		case contains(path, "ratios-ttm"):
			json.NewEncoder(w).Encode([]FMPRatiosTTM{
				{Symbol: "AAPL", PriceToEarningsRatioTTM: &pe},
			})
		case contains(path, "key-metrics-ttm"):
			json.NewEncoder(w).Encode([]FMPKeyMetricsTTM{
				{Symbol: "AAPL", MarketCapTTM: &mc},
			})
		case contains(path, "financial-growth"):
			json.NewEncoder(w).Encode([]FMPFinancialGrowth{
				{Symbol: "AAPL", RevenueGrowth: &rg},
			})
		case contains(path, "analyst-estimates"):
			json.NewEncoder(w).Encode([]FMPAnalystEstimate{
				{Symbol: "AAPL", EstimatedEPSAvg: &epsAvg},
			})
		case contains(path, "score"):
			json.NewEncoder(w).Encode([]FMPScore{
				{Symbol: "AAPL", AltmanZScore: &zScore, PiotroskiScore: &fScore},
			})
		case contains(path, "dividend"):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"historical": []FMPDividendHistorical{
					{Symbol: "AAPL", Date: "2024-11-08", Dividend: 0.25},
				},
			})
		case contains(path, "grades-summary"):
			json.NewEncoder(w).Encode([]FMPGradesSummary{
				{Symbol: "AAPL", StrongBuy: 10, Buy: 15, Hold: 5, Sell: 1, StrongSell: 0, Consensus: "Buy"},
			})
		case contains(path, "price-target-consensus"):
			json.NewEncoder(w).Encode([]FMPPriceTargetConsensus{
				{Symbol: "AAPL", TargetHigh: &high},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	result := client.GetAllMetrics("AAPL")
	require.NotNil(t, result)

	// Check all endpoints returned successfully
	require.NotNil(t, result.RatiosTTM)
	require.NotNil(t, result.RatiosTTM.PriceToEarningsRatioTTM)
	assert.InDelta(t, 28.0, *result.RatiosTTM.PriceToEarningsRatioTTM, 0.01)

	require.NotNil(t, result.KeyMetricsTTM)
	require.NotNil(t, result.KeyMetricsTTM.MarketCapTTM)

	assert.Len(t, result.Growth, 1)
	assert.Len(t, result.Estimates, 1)

	require.NotNil(t, result.Score)
	require.NotNil(t, result.Score.AltmanZScore)
	assert.InDelta(t, 4.0, *result.Score.AltmanZScore, 0.01)

	assert.Len(t, result.Dividends, 1)

	require.NotNil(t, result.GradesSummary)
	assert.Equal(t, "Buy", result.GradesSummary.Consensus)

	require.NotNil(t, result.PriceTargetConsensus)
	require.NotNil(t, result.PriceTargetConsensus.TargetHigh)
}

func TestFMP_GetAllMetrics_PartialFailures(t *testing.T) {
	pe := 28.0
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		path := r.URL.Path

		// Only ratios-ttm succeeds; everything else fails
		if contains(path, "ratios-ttm") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]FMPRatiosTTM{
				{Symbol: "AAPL", PriceToEarningsRatioTTM: &pe},
			})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	result := client.GetAllMetrics("AAPL")
	require.NotNil(t, result)

	// Ratios should succeed
	require.NotNil(t, result.RatiosTTM)

	// Other endpoints should have errors recorded
	assert.NotEmpty(t, result.Errors)
	assert.True(t, len(result.Errors) > 0, "should have recorded some errors")
}

// contains checks if s contains substr (helper for routing in test server).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && len(substr) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
