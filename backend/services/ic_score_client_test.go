package services

import (
	"encoding/json"
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
// NewICScoreClient
// ---------------------------------------------------------------------------

func TestNewICScoreClient_DefaultURL(t *testing.T) {
	os.Unsetenv("IC_SCORE_API_URL")
	client := NewICScoreClient()
	require.NotNil(t, client)
	assert.Equal(t, "http://ic-score-service:8000", client.baseURL)
}

func TestNewICScoreClient_CustomURL(t *testing.T) {
	os.Setenv("IC_SCORE_API_URL", "http://localhost:9999")
	defer os.Unsetenv("IC_SCORE_API_URL")

	client := NewICScoreClient()
	require.NotNil(t, client)
	assert.Equal(t, "http://localhost:9999", client.baseURL)
}

func TestNewICScoreClient_HasTimeout(t *testing.T) {
	client := NewICScoreClient()
	require.NotNil(t, client.httpClient)
	assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
}

// ---------------------------------------------------------------------------
// GetAnnualFinancials — mock server
// ---------------------------------------------------------------------------

func TestGetAnnualFinancials_Success(t *testing.T) {
	revenue := int64(394328000000)
	netIncome := int64(99803000000)
	grossMargin := 0.438

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/financials/AAPL/annual")
		assert.Equal(t, "income_statement", r.URL.Query().Get("statement_type"))

		response := ICScoreAPIResponse{
			Ticker:        "AAPL",
			StatementType: "income_statement",
			Timeframe:     "annual",
			Periods: []ICScoreFinancialPeriod{
				{
					FiscalYear:    2024,
					PeriodEndDate: "2024-09-28",
					Revenue:       &revenue,
					NetIncome:     &netIncome,
					GrossMargin:   &grossMargin,
				},
			},
			Metadata: ICScoreFinancialMetadata{CompanyName: "Apple Inc."},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &ICScoreClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	result, err := client.GetAnnualFinancials("AAPL", "income_statement", 5)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "AAPL", result.Ticker)
	assert.Equal(t, "income_statement", result.StatementType)
	assert.Len(t, result.Periods, 1)
	assert.Equal(t, int64(394328000000), *result.Periods[0].Revenue)
	assert.Equal(t, "Apple Inc.", result.Metadata.CompanyName)
}

func TestGetAnnualFinancials_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &ICScoreClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	_, err := client.GetAnnualFinancials("FAKE", "income_statement", 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no annual financial data found")
}

func TestGetAnnualFinancials_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := &ICScoreClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	_, err := client.GetAnnualFinancials("AAPL", "income_statement", 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestGetAnnualFinancials_ConnectionError(t *testing.T) {
	client := &ICScoreClient{
		baseURL:    "http://localhost:99999",
		httpClient: &http.Client{Timeout: 1 * time.Second},
	}

	_, err := client.GetAnnualFinancials("AAPL", "income_statement", 5)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// GetTTMFinancials — mock server
// ---------------------------------------------------------------------------

func TestGetTTMFinancials_Success(t *testing.T) {
	revenue := int64(390000000000)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/financials/AAPL/ttm")

		response := ICScoreAPIResponse{
			Ticker:        "AAPL",
			StatementType: "income_statement",
			Timeframe:     "ttm",
			Periods: []ICScoreFinancialPeriod{
				{
					FiscalYear:    2024,
					PeriodEndDate: "2024-12-28",
					Revenue:       &revenue,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &ICScoreClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	result, err := client.GetTTMFinancials("AAPL", "income_statement", 3)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "ttm", result.Timeframe)
}

func TestGetTTMFinancials_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &ICScoreClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	_, err := client.GetTTMFinancials("FAKE", "income_statement", 3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no TTM financial data found")
}

// ---------------------------------------------------------------------------
// GetNews — mock server
// ---------------------------------------------------------------------------

func TestGetNews_Success(t *testing.T) {
	sentimentScore := 75.0
	sentimentLabel := "Positive"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/news/AAPL")

		response := ICScoreNewsResponse{
			Ticker: "AAPL",
			Count:  2,
			Articles: []ICScoreNewsArticle{
				{
					ID:             1,
					Title:          "Apple Reports Record Revenue",
					URL:            "https://example.com/news/1",
					Source:         "Reuters",
					PublishedAt:    "2024-01-15T10:00:00Z",
					Tickers:        []string{"AAPL"},
					SentimentScore: &sentimentScore,
					SentimentLabel: &sentimentLabel,
				},
				{
					ID:          2,
					Title:       "New iPhone Sales",
					URL:         "https://example.com/news/2",
					Source:      "Bloomberg",
					PublishedAt: "2024-01-14T09:00:00Z",
					Tickers:     []string{"AAPL"},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &ICScoreClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	result, err := client.GetNews("AAPL", 10, 7)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "AAPL", result.Ticker)
	assert.Equal(t, 2, result.Count)
	assert.Len(t, result.Articles, 2)
	assert.Equal(t, "Apple Reports Record Revenue", result.Articles[0].Title)
	assert.NotNil(t, result.Articles[0].SentimentScore)
	assert.Equal(t, 75.0, *result.Articles[0].SentimentScore)
}

// ---------------------------------------------------------------------------
// ConvertToFinancialPeriods — pure function
// ---------------------------------------------------------------------------

func TestConvertToFinancialPeriods_IncomeStatement(t *testing.T) {
	revenue := int64(394328000000)
	netIncome := int64(99803000000)
	grossMargin := 0.438
	quarter := 4

	apiResponse := &ICScoreAPIResponse{
		Ticker:        "AAPL",
		StatementType: "income_statement",
		Periods: []ICScoreFinancialPeriod{
			{
				FiscalYear:    2024,
				FiscalQuarter: &quarter,
				PeriodEndDate: "2024-09-28",
				Revenue:       &revenue,
				NetIncome:     &netIncome,
				GrossMargin:   &grossMargin,
			},
		},
	}

	periods := ConvertToFinancialPeriods(apiResponse, models.StatementTypeIncome)
	require.Len(t, periods, 1)

	p := periods[0]
	assert.Equal(t, 2024, p.FiscalYear)
	assert.Equal(t, "2024-09-28", p.PeriodEnd)
	assert.Equal(t, int64(394328000000), p.Data["revenue"])
	assert.Equal(t, int64(99803000000), p.Data["net_income"])
	// Gross margin is multiplied by 100
	assert.InDelta(t, 43.8, p.Data["gross_margin"], 0.01)
}

func TestConvertToFinancialPeriods_BalanceSheet(t *testing.T) {
	totalAssets := int64(352583000000)
	totalLiabilities := int64(290437000000)
	roe := 0.1567

	apiResponse := &ICScoreAPIResponse{
		Periods: []ICScoreFinancialPeriod{
			{
				FiscalYear:       2024,
				PeriodEndDate:    "2024-09-28",
				TotalAssets:      &totalAssets,
				TotalLiabilities: &totalLiabilities,
				ROE:              &roe,
			},
		},
	}

	periods := ConvertToFinancialPeriods(apiResponse, models.StatementTypeBalanceSheet)
	require.Len(t, periods, 1)

	p := periods[0]
	assert.Equal(t, int64(352583000000), p.Data["total_assets"])
	assert.Equal(t, int64(290437000000), p.Data["total_liabilities"])
	// ROE is multiplied by 100
	assert.InDelta(t, 15.67, p.Data["return_on_equity"], 0.01)
}

func TestConvertToFinancialPeriods_CashFlow(t *testing.T) {
	opCashFlow := int64(110543000000)
	freeCashFlow := int64(99584000000)
	capex := int64(-10959000000)

	apiResponse := &ICScoreAPIResponse{
		Periods: []ICScoreFinancialPeriod{
			{
				FiscalYear:        2024,
				PeriodEndDate:     "2024-09-28",
				OperatingCashFlow: &opCashFlow,
				FreeCashFlow:      &freeCashFlow,
				Capex:             &capex,
			},
		},
	}

	periods := ConvertToFinancialPeriods(apiResponse, models.StatementTypeCashFlow)
	require.Len(t, periods, 1)

	p := periods[0]
	assert.Equal(t, int64(110543000000), p.Data["net_cash_flow_from_operating_activities"])
	assert.Equal(t, int64(99584000000), p.Data["free_cash_flow"])
	assert.Equal(t, int64(-10959000000), p.Data["capital_expenditure"])
}

func TestConvertToFinancialPeriods_EmptyPeriods(t *testing.T) {
	apiResponse := &ICScoreAPIResponse{
		Periods: []ICScoreFinancialPeriod{},
	}

	periods := ConvertToFinancialPeriods(apiResponse, models.StatementTypeIncome)
	assert.Empty(t, periods)
}

func TestConvertToFinancialPeriods_NilFields(t *testing.T) {
	apiResponse := &ICScoreAPIResponse{
		Periods: []ICScoreFinancialPeriod{
			{
				FiscalYear:    2024,
				PeriodEndDate: "2024-09-28",
				// All fields nil
			},
		},
	}

	periods := ConvertToFinancialPeriods(apiResponse, models.StatementTypeIncome)
	require.Len(t, periods, 1)
	assert.Empty(t, periods[0].Data, "all nil fields should produce empty data map")
}

func TestConvertToFinancialPeriods_SharesOutstanding(t *testing.T) {
	shares := int64(15550061000)

	apiResponse := &ICScoreAPIResponse{
		Periods: []ICScoreFinancialPeriod{
			{
				FiscalYear:        2024,
				PeriodEndDate:     "2024-09-28",
				SharesOutstanding: &shares,
			},
		},
	}

	// Shares outstanding is added to all statement types
	for _, stType := range []models.StatementType{
		models.StatementTypeIncome,
		models.StatementTypeBalanceSheet,
		models.StatementTypeCashFlow,
	} {
		periods := ConvertToFinancialPeriods(apiResponse, stType)
		require.Len(t, periods, 1)
		assert.Equal(t, int64(15550061000), periods[0].Data["shares_outstanding"])
	}
}
