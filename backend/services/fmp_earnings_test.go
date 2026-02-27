package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helpers float64Ptr and newFMPTestClient are defined in fmp_httptest_test.go

// ============================================================================
// ComputeSurprisePercent Tests
// ============================================================================

func TestComputeSurprisePercent_Beat(t *testing.T) {
	result := ComputeSurprisePercent(float64Ptr(1.64), float64Ptr(1.60))
	require.NotNil(t, result)
	assert.InDelta(t, 2.5, *result, 0.01) // (1.64-1.60)/1.60*100 = 2.5%
}

func TestComputeSurprisePercent_Miss(t *testing.T) {
	result := ComputeSurprisePercent(float64Ptr(1.50), float64Ptr(1.60))
	require.NotNil(t, result)
	assert.InDelta(t, -6.25, *result, 0.01) // (1.50-1.60)/1.60*100 = -6.25%
}

func TestComputeSurprisePercent_NilActual(t *testing.T) {
	result := ComputeSurprisePercent(nil, float64Ptr(1.60))
	assert.Nil(t, result)
}

func TestComputeSurprisePercent_NilEstimate(t *testing.T) {
	result := ComputeSurprisePercent(float64Ptr(1.64), nil)
	assert.Nil(t, result)
}

func TestComputeSurprisePercent_BothNil(t *testing.T) {
	result := ComputeSurprisePercent(nil, nil)
	assert.Nil(t, result)
}

func TestComputeSurprisePercent_ZeroEstimate(t *testing.T) {
	// When estimate is zero, return nil (division undefined)
	result := ComputeSurprisePercent(float64Ptr(0.05), float64Ptr(0))
	assert.Nil(t, result)
}

func TestComputeSurprisePercent_BothZero(t *testing.T) {
	// Both zero: estimate is still zero → nil
	result := ComputeSurprisePercent(float64Ptr(0), float64Ptr(0))
	assert.Nil(t, result)
}

func TestComputeSurprisePercent_NegativeEstimate(t *testing.T) {
	// EPS can be negative; surprise should use absolute value of estimate
	result := ComputeSurprisePercent(float64Ptr(-0.50), float64Ptr(-0.80))
	require.NotNil(t, result)
	// (-0.50 - (-0.80)) / |-0.80| * 100 = 0.30/0.80*100 = 37.5%
	assert.InDelta(t, 37.5, *result, 0.01)
}

// ============================================================================
// ComputeBeat Tests
// ============================================================================

func TestComputeBeat_Beat(t *testing.T) {
	result := ComputeBeat(float64Ptr(1.64), float64Ptr(1.60))
	require.NotNil(t, result)
	assert.True(t, *result)
}

func TestComputeBeat_Miss(t *testing.T) {
	result := ComputeBeat(float64Ptr(1.50), float64Ptr(1.60))
	require.NotNil(t, result)
	assert.False(t, *result)
}

func TestComputeBeat_Equal(t *testing.T) {
	result := ComputeBeat(float64Ptr(1.60), float64Ptr(1.60))
	require.NotNil(t, result)
	assert.False(t, *result) // equal is not a beat
}

func TestComputeBeat_NilActual(t *testing.T) {
	result := ComputeBeat(nil, float64Ptr(1.60))
	assert.Nil(t, result)
}

func TestComputeBeat_NilEstimate(t *testing.T) {
	result := ComputeBeat(float64Ptr(1.64), nil)
	assert.Nil(t, result)
}

// ============================================================================
// ToFiscalQuarter Tests
// ============================================================================

func TestToFiscalQuarter_Q1(t *testing.T) {
	assert.Equal(t, "Q1 '26", ToFiscalQuarter("2026-02-15"))
	assert.Equal(t, "Q1 '25", ToFiscalQuarter("2025-01-01"))
	assert.Equal(t, "Q1 '26", ToFiscalQuarter("2026-03-31"))
}

func TestToFiscalQuarter_Q2(t *testing.T) {
	assert.Equal(t, "Q2 '26", ToFiscalQuarter("2026-04-15"))
	assert.Equal(t, "Q2 '25", ToFiscalQuarter("2025-06-30"))
}

func TestToFiscalQuarter_Q3(t *testing.T) {
	assert.Equal(t, "Q3 '25", ToFiscalQuarter("2025-07-01"))
	assert.Equal(t, "Q3 '25", ToFiscalQuarter("2025-09-30"))
}

func TestToFiscalQuarter_Q4(t *testing.T) {
	assert.Equal(t, "Q4 '25", ToFiscalQuarter("2025-10-30"))
	assert.Equal(t, "Q4 '25", ToFiscalQuarter("2025-12-31"))
}

func TestToFiscalQuarter_InvalidDate(t *testing.T) {
	assert.Equal(t, "not-a-date", ToFiscalQuarter("not-a-date"))
	assert.Equal(t, "", ToFiscalQuarter(""))
}

// ============================================================================
// TransformEarnings Tests
// ============================================================================

func TestTransformEarnings_FullPipeline(t *testing.T) {
	records := []FMPEarningsRecord{
		{
			// Future quarter (upcoming)
			Symbol:           "AAPL",
			Date:             "2099-06-15",
			EPSActual:        nil,
			EPSEstimated:     float64Ptr(2.35),
			RevenueActual:    nil,
			RevenueEstimated: float64Ptr(124100000000),
		},
		{
			// Past quarter (beat)
			Symbol:           "AAPL",
			Date:             "2024-10-30",
			EPSActual:        float64Ptr(1.64),
			EPSEstimated:     float64Ptr(1.60),
			RevenueActual:    float64Ptr(94930000000),
			RevenueEstimated: float64Ptr(94500000000),
		},
		{
			// Past quarter (miss)
			Symbol:           "AAPL",
			Date:             "2024-07-25",
			EPSActual:        float64Ptr(1.35),
			EPSEstimated:     float64Ptr(1.40),
			RevenueActual:    float64Ptr(85800000000),
			RevenueEstimated: float64Ptr(86000000000),
		},
	}

	resp := TransformEarnings(records)
	require.NotNil(t, resp)

	// Should have 3 results
	assert.Len(t, resp.Earnings, 3)

	// First record should be upcoming
	assert.True(t, resp.Earnings[0].IsUpcoming)
	assert.Nil(t, resp.Earnings[0].EPSSurprisePercent)
	assert.Nil(t, resp.Earnings[0].EPSBeat)

	// Second record should be a beat
	assert.False(t, resp.Earnings[1].IsUpcoming)
	require.NotNil(t, resp.Earnings[1].EPSBeat)
	assert.True(t, *resp.Earnings[1].EPSBeat)
	require.NotNil(t, resp.Earnings[1].EPSSurprisePercent)
	assert.InDelta(t, 2.5, *resp.Earnings[1].EPSSurprisePercent, 0.01)

	// Third record should be a miss
	require.NotNil(t, resp.Earnings[2].EPSBeat)
	assert.False(t, *resp.Earnings[2].EPSBeat)
	require.NotNil(t, resp.Earnings[2].EPSSurprisePercent)
	assert.True(t, *resp.Earnings[2].EPSSurprisePercent < 0)

	// NextEarnings should be the upcoming record
	require.NotNil(t, resp.NextEarnings)
	assert.Equal(t, "2099-06-15", resp.NextEarnings.Date)
	assert.True(t, resp.NextEarnings.IsUpcoming)

	// MostRecentEarnings should be the first past record
	require.NotNil(t, resp.MostRecentEarnings)
	assert.Equal(t, "2024-10-30", resp.MostRecentEarnings.Date)
	assert.False(t, resp.MostRecentEarnings.IsUpcoming)

	// Beat rate: 1 EPS beat out of 2, 1 revenue beat out of 2
	require.NotNil(t, resp.BeatRate)
	assert.Equal(t, 1, resp.BeatRate.EPSBeats)
	assert.Equal(t, 1, resp.BeatRate.RevenueBeats)
	assert.Equal(t, 2, resp.BeatRate.TotalQuarters)
	assert.Equal(t, 2, resp.BeatRate.TotalRevenueQuarters)
}

func TestTransformEarnings_Empty(t *testing.T) {
	resp := TransformEarnings([]FMPEarningsRecord{})
	require.NotNil(t, resp)
	assert.Empty(t, resp.Earnings)
	assert.Nil(t, resp.NextEarnings)
	assert.Nil(t, resp.BeatRate)
}

func TestTransformEarnings_NoUpcoming_MostRecentPopulated(t *testing.T) {
	// Use far-past sentinel dates to make intent explicit
	records := []FMPEarningsRecord{
		{
			Symbol:           "MSFT",
			Date:             "2000-10-22",
			EPSActual:        float64Ptr(3.30),
			EPSEstimated:     float64Ptr(3.10),
			RevenueActual:    float64Ptr(65600000000),
			RevenueEstimated: float64Ptr(64500000000),
		},
		{
			Symbol:           "MSFT",
			Date:             "2000-07-23",
			EPSActual:        float64Ptr(2.95),
			EPSEstimated:     float64Ptr(2.93),
			RevenueActual:    float64Ptr(64700000000),
			RevenueEstimated: float64Ptr(64200000000),
		},
	}

	resp := TransformEarnings(records)
	require.NotNil(t, resp)

	// NextEarnings should be nil when no future-dated record exists
	assert.Nil(t, resp.NextEarnings)

	// MostRecentEarnings should be the first past record
	require.NotNil(t, resp.MostRecentEarnings)
	assert.Equal(t, "2000-10-22", resp.MostRecentEarnings.Date)
	assert.False(t, resp.MostRecentEarnings.IsUpcoming)
}

func TestTransformEarnings_FiscalQuarterLabels(t *testing.T) {
	records := []FMPEarningsRecord{
		{Symbol: "TEST", Date: "2025-01-15", EPSActual: float64Ptr(1.0), EPSEstimated: float64Ptr(1.0)},
		{Symbol: "TEST", Date: "2025-04-15", EPSActual: float64Ptr(1.0), EPSEstimated: float64Ptr(1.0)},
		{Symbol: "TEST", Date: "2025-07-15", EPSActual: float64Ptr(1.0), EPSEstimated: float64Ptr(1.0)},
		{Symbol: "TEST", Date: "2025-10-15", EPSActual: float64Ptr(1.0), EPSEstimated: float64Ptr(1.0)},
	}

	resp := TransformEarnings(records)
	assert.Equal(t, "Q1 '25", resp.Earnings[0].FiscalQuarter)
	assert.Equal(t, "Q2 '25", resp.Earnings[1].FiscalQuarter)
	assert.Equal(t, "Q3 '25", resp.Earnings[2].FiscalQuarter)
	assert.Equal(t, "Q4 '25", resp.Earnings[3].FiscalQuarter)
}

func TestTransformEarnings_NilEstimates(t *testing.T) {
	// Small-cap with no analyst coverage
	records := []FMPEarningsRecord{
		{
			Symbol:           "TINY",
			Date:             "2024-10-15",
			EPSActual:        float64Ptr(0.12),
			EPSEstimated:     nil,
			RevenueActual:    float64Ptr(5000000),
			RevenueEstimated: nil,
		},
	}

	resp := TransformEarnings(records)
	assert.Len(t, resp.Earnings, 1)
	assert.Nil(t, resp.Earnings[0].EPSSurprisePercent)
	assert.Nil(t, resp.Earnings[0].EPSBeat)
	assert.Nil(t, resp.Earnings[0].RevenueSurprisePercent)
	assert.Nil(t, resp.Earnings[0].RevenueBeat)
	// No qualified quarters → nil beat rate
	assert.Nil(t, resp.BeatRate)
}

// ============================================================================
// FMP API Method Tests (using httptest)
// ============================================================================

func TestGetEarnings_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/earnings")
		assert.Equal(t, "AAPL", r.URL.Query().Get("symbol"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]FMPEarningsRecord{
			{
				Symbol:       "AAPL",
				Date:         "2024-10-30",
				EPSActual:    float64Ptr(1.64),
				EPSEstimated: float64Ptr(1.60),
			},
		})
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)

	records, err := client.GetEarnings("AAPL")
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "AAPL", records[0].Symbol)
	assert.Equal(t, "2024-10-30", records[0].Date)
	require.NotNil(t, records[0].EPSActual)
	assert.Equal(t, 1.64, *records[0].EPSActual)
}

func TestGetEarnings_NoAPIKey(t *testing.T) {
	client := &FMPClient{APIKey: "", Client: http.DefaultClient}
	_, err := client.GetEarnings("AAPL")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FMP API key not configured")
}

func TestGetEarnings_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)
	_, err := client.GetEarnings("AAPL")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestGetEarnings_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]FMPEarningsRecord{})
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)
	records, err := client.GetEarnings("AAPL")
	require.NoError(t, err)
	assert.Empty(t, records)
}

func TestGetEarningsCalendar_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/earnings-calendar")
		assert.Equal(t, "2026-02-23", r.URL.Query().Get("from"))
		assert.Equal(t, "2026-03-06", r.URL.Query().Get("to"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]FMPEarningsRecord{
			{Symbol: "AAPL", Date: "2026-02-25", EPSEstimated: float64Ptr(2.35)},
			{Symbol: "MSFT", Date: "2026-02-25", EPSEstimated: float64Ptr(3.10)},
			{Symbol: "GOOG", Date: "2026-02-27", EPSEstimated: float64Ptr(1.80)},
		})
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)
	records, err := client.GetEarningsCalendar("2026-02-23", "2026-03-06")
	require.NoError(t, err)
	assert.Len(t, records, 3)
}

func TestGetEarningsCalendar_NoAPIKey(t *testing.T) {
	client := &FMPClient{APIKey: "", Client: http.DefaultClient}
	_, err := client.GetEarningsCalendar("2026-02-23", "2026-03-06")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FMP API key not configured")
}

func TestGetEarningsCalendar_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	restore := saveFMPBaseURL()
	defer restore()
	FMPBaseURL = server.URL

	client := newFMPTestClient(server.URL)
	_, err := client.GetEarningsCalendar("2026-02-23", "2026-03-06")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestTransformEarnings_IsUpcoming_DateOnly(t *testing.T) {
	// A past-dated record with nil EPSActual (small-cap, no coverage)
	// should NOT be marked upcoming — isUpcoming is purely date-based.
	records := []FMPEarningsRecord{
		{
			Symbol:       "TINY",
			Date:         "2000-01-15",
			EPSActual:    nil,
			EPSEstimated: nil,
		},
	}

	resp := TransformEarnings(records)
	require.Len(t, resp.Earnings, 1)
	assert.False(t, resp.Earnings[0].IsUpcoming, "past-dated record with nil EPSActual should not be upcoming")
}

func TestTransformEarnings_SeparateRevenueQuarterCount(t *testing.T) {
	// 2 quarters with EPS data but only 1 with revenue data.
	// TotalRevenueQuarters should be 1, not 2.
	records := []FMPEarningsRecord{
		{
			Symbol:           "TINY",
			Date:             "2000-10-15",
			EPSActual:        float64Ptr(0.50),
			EPSEstimated:     float64Ptr(0.40),
			RevenueActual:    float64Ptr(10000000),
			RevenueEstimated: float64Ptr(9000000),
		},
		{
			Symbol:           "TINY",
			Date:             "2000-07-15",
			EPSActual:        float64Ptr(0.30),
			EPSEstimated:     float64Ptr(0.25),
			RevenueActual:    nil,
			RevenueEstimated: nil,
		},
	}

	resp := TransformEarnings(records)
	require.NotNil(t, resp.BeatRate)
	assert.Equal(t, 2, resp.BeatRate.TotalQuarters)
	assert.Equal(t, 1, resp.BeatRate.TotalRevenueQuarters)
	assert.Equal(t, 2, resp.BeatRate.EPSBeats)
	assert.Equal(t, 1, resp.BeatRate.RevenueBeats)
}
