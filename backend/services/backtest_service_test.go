package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"investorcenter-api/models"
)

// ---------------------------------------------------------------------------
// NewBacktestService
// ---------------------------------------------------------------------------

func TestNewBacktestService_DefaultURL(t *testing.T) {
	os.Unsetenv("IC_SCORE_API_URL")
	svc := NewBacktestService()
	require.NotNil(t, svc)
	assert.Equal(t, "http://localhost:8001", svc.icScoreAPIURL)
}

func TestNewBacktestService_CustomURL(t *testing.T) {
	os.Setenv("IC_SCORE_API_URL", "http://custom-api:9000")
	defer os.Unsetenv("IC_SCORE_API_URL")

	svc := NewBacktestService()
	require.NotNil(t, svc)
	assert.Equal(t, "http://custom-api:9000", svc.icScoreAPIURL)
}

func TestNewBacktestService_LongTimeout(t *testing.T) {
	svc := NewBacktestService()
	assert.Equal(t, 30*time.Minute, svc.httpClient.Timeout)
}

// ---------------------------------------------------------------------------
// ValidateConfig — pure function
// ---------------------------------------------------------------------------

func TestValidateConfig_Valid(t *testing.T) {
	svc := NewBacktestService()

	config := models.BacktestConfig{
		StartDate:          "2020-01-01",
		EndDate:            "2024-01-01",
		RebalanceFrequency: "monthly",
		Universe:           "sp500",
		TransactionCostBps: 10,
		SlippageBps:        5,
	}

	err := svc.ValidateConfig(config)
	assert.NoError(t, err)
}

func TestValidateConfig_InvalidStartDate(t *testing.T) {
	svc := NewBacktestService()

	config := models.BacktestConfig{
		StartDate: "not-a-date",
		EndDate:   "2024-01-01",
	}

	err := svc.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "start_date")
}

func TestValidateConfig_InvalidEndDate(t *testing.T) {
	svc := NewBacktestService()

	config := models.BacktestConfig{
		StartDate: "2020-01-01",
		EndDate:   "not-a-date",
	}

	err := svc.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "end_date")
}

func TestValidateConfig_EndBeforeStart(t *testing.T) {
	svc := NewBacktestService()

	config := models.BacktestConfig{
		StartDate:          "2024-01-01",
		EndDate:            "2020-01-01",
		RebalanceFrequency: "monthly",
		Universe:           "sp500",
	}

	err := svc.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "end_date must be after start_date")
}

func TestValidateConfig_PeriodTooShort(t *testing.T) {
	svc := NewBacktestService()

	config := models.BacktestConfig{
		StartDate:          "2024-01-01",
		EndDate:            "2024-06-01",
		RebalanceFrequency: "monthly",
		Universe:           "sp500",
	}

	err := svc.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 1 year")
}

func TestValidateConfig_PeriodTooLong(t *testing.T) {
	svc := NewBacktestService()

	config := models.BacktestConfig{
		StartDate:          "2010-01-01",
		EndDate:            "2024-01-01", // 14 years
		RebalanceFrequency: "monthly",
		Universe:           "sp500",
	}

	err := svc.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceed 10 years")
}

func TestValidateConfig_FutureEndDate(t *testing.T) {
	svc := NewBacktestService()

	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")

	config := models.BacktestConfig{
		StartDate:          "2020-01-01",
		EndDate:            futureDate,
		RebalanceFrequency: "monthly",
		Universe:           "sp500",
	}

	err := svc.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "future")
}

func TestValidateConfig_InvalidFrequency(t *testing.T) {
	svc := NewBacktestService()

	config := models.BacktestConfig{
		StartDate:          "2020-01-01",
		EndDate:            "2024-01-01",
		RebalanceFrequency: "biweekly",
		Universe:           "sp500",
	}

	err := svc.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rebalance_frequency")
}

func TestValidateConfig_ValidFrequencies(t *testing.T) {
	svc := NewBacktestService()

	for _, freq := range []string{"daily", "weekly", "monthly", "quarterly"} {
		t.Run(freq, func(t *testing.T) {
			config := models.BacktestConfig{
				StartDate:          "2020-01-01",
				EndDate:            "2024-01-01",
				RebalanceFrequency: freq,
				Universe:           "sp500",
			}

			err := svc.ValidateConfig(config)
			assert.NoError(t, err)
		})
	}
}

func TestValidateConfig_InvalidUniverse(t *testing.T) {
	svc := NewBacktestService()

	config := models.BacktestConfig{
		StartDate:          "2020-01-01",
		EndDate:            "2024-01-01",
		RebalanceFrequency: "monthly",
		Universe:           "nasdaq100",
	}

	err := svc.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "universe")
}

func TestValidateConfig_ValidUniverses(t *testing.T) {
	svc := NewBacktestService()

	for _, univ := range []string{"sp500", "sp1500", "all"} {
		t.Run(univ, func(t *testing.T) {
			config := models.BacktestConfig{
				StartDate:          "2020-01-01",
				EndDate:            "2024-01-01",
				RebalanceFrequency: "monthly",
				Universe:           univ,
			}

			err := svc.ValidateConfig(config)
			assert.NoError(t, err)
		})
	}
}

func TestValidateConfig_NegativeTransactionCost(t *testing.T) {
	svc := NewBacktestService()

	config := models.BacktestConfig{
		StartDate:          "2020-01-01",
		EndDate:            "2024-01-01",
		RebalanceFrequency: "monthly",
		Universe:           "sp500",
		TransactionCostBps: -5,
	}

	err := svc.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transaction_cost_bps")
}

func TestValidateConfig_ExcessiveTransactionCost(t *testing.T) {
	svc := NewBacktestService()

	config := models.BacktestConfig{
		StartDate:          "2020-01-01",
		EndDate:            "2024-01-01",
		RebalanceFrequency: "monthly",
		Universe:           "sp500",
		TransactionCostBps: 150,
	}

	err := svc.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transaction_cost_bps")
}

func TestValidateConfig_NegativeSlippage(t *testing.T) {
	svc := NewBacktestService()

	config := models.BacktestConfig{
		StartDate:          "2020-01-01",
		EndDate:            "2024-01-01",
		RebalanceFrequency: "monthly",
		Universe:           "sp500",
		SlippageBps:        -1,
	}

	err := svc.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "slippage_bps")
}

// ---------------------------------------------------------------------------
// GetDefaultBacktestConfig
// ---------------------------------------------------------------------------

func TestGetDefaultBacktestConfig(t *testing.T) {
	svc := NewBacktestService()
	config := svc.GetDefaultBacktestConfig()

	assert.Equal(t, "monthly", config.RebalanceFrequency)
	assert.Equal(t, "sp500", config.Universe)
	assert.Equal(t, "SPY", config.Benchmark)
	assert.Equal(t, float64(10), config.TransactionCostBps)
	assert.Equal(t, float64(5), config.SlippageBps)
	assert.True(t, config.UseSmoothedScores)
	assert.False(t, config.ExcludeFinancials)
	assert.False(t, config.ExcludeUtilities)

	// Start date should be ~5 years ago
	startDate, err := time.Parse("2006-01-02", config.StartDate)
	require.NoError(t, err)
	fiveYearsAgo := time.Now().AddDate(-5, 0, 0)
	assert.WithinDuration(t, fiveYearsAgo, startDate, 48*time.Hour)

	// End date should be ~today
	endDate, err := time.Parse("2006-01-02", config.EndDate)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), endDate, 48*time.Hour)
}

// ---------------------------------------------------------------------------
// GenerateCharts — pure function
// ---------------------------------------------------------------------------

func TestGenerateCharts(t *testing.T) {
	svc := NewBacktestService()

	summary := &models.BacktestSummary{
		DecilePerformance: []models.DecilePerformance{
			{Decile: 1, AnnualizedReturn: 0.15},
			{Decile: 2, AnnualizedReturn: 0.10},
			{Decile: 9, AnnualizedReturn: -0.02},
			{Decile: 10, AnnualizedReturn: -0.08},
		},
	}

	charts := svc.GenerateCharts(summary)
	require.NotNil(t, charts)

	// Check decile bar chart
	assert.Len(t, charts.DecileBarChart.Labels, 4)
	assert.Equal(t, "D1", charts.DecileBarChart.Labels[0])
	assert.Equal(t, "D10", charts.DecileBarChart.Labels[3])

	require.Len(t, charts.DecileBarChart.Datasets, 1)
	dataset := charts.DecileBarChart.Datasets[0]

	data := dataset["data"].([]float64)
	assert.Len(t, data, 4)
	assert.InDelta(t, 15.0, data[0], 0.01) // 0.15 * 100
	assert.InDelta(t, -8.0, data[3], 0.01) // -0.08 * 100

	colors := dataset["backgroundColor"].([]string)
	assert.Equal(t, "#10b981", colors[0]) // Positive = green
	assert.Equal(t, "#ef4444", colors[3]) // Negative = red
}

func TestGenerateCharts_EmptyDeciles(t *testing.T) {
	svc := NewBacktestService()

	summary := &models.BacktestSummary{
		DecilePerformance: []models.DecilePerformance{},
	}

	charts := svc.GenerateCharts(summary)
	require.NotNil(t, charts)
	assert.Empty(t, charts.DecileBarChart.Labels)
}

// ---------------------------------------------------------------------------
// RunBacktest — mock server
// ---------------------------------------------------------------------------

func TestRunBacktest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/backtest", r.URL.Path)

		var config models.BacktestConfig
		json.NewDecoder(r.Body).Decode(&config)
		assert.Equal(t, "sp500", config.Universe)

		summary := models.BacktestSummary{
			SpreadCAGR:     0.14,
			HitRate:        0.65,
			Benchmark:      "SPY",
			StartDate:      config.StartDate,
			EndDate:        config.EndDate,
			NumPeriods:     60,
			TopVsBenchmark: 0.05,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(summary)
	}))
	defer server.Close()

	svc := &BacktestService{
		icScoreAPIURL: server.URL,
		httpClient:    &http.Client{Timeout: 5 * time.Second},
	}

	config := models.BacktestConfig{
		StartDate:          "2020-01-01",
		EndDate:            "2024-01-01",
		RebalanceFrequency: "monthly",
		Universe:           "sp500",
		Benchmark:          "SPY",
	}

	result, err := svc.RunBacktest(config)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 0.14, result.SpreadCAGR)
	assert.Equal(t, 0.65, result.HitRate)
	assert.Equal(t, "SPY", result.Benchmark)
}

func TestRunBacktest_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "backtest engine crashed")
	}))
	defer server.Close()

	svc := &BacktestService{
		icScoreAPIURL: server.URL,
		httpClient:    &http.Client{Timeout: 5 * time.Second},
	}

	_, err := svc.RunBacktest(models.BacktestConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestRunBacktest_ConnectionError(t *testing.T) {
	svc := &BacktestService{
		icScoreAPIURL: "http://localhost:99999",
		httpClient:    &http.Client{Timeout: 1 * time.Second},
	}

	_, err := svc.RunBacktest(models.BacktestConfig{})
	assert.Error(t, err)
}
