package services

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"investorcenter-api/models"
)

// ---------------------------------------------------------------------------
// ParseFiscalPeriod — pure function
// ---------------------------------------------------------------------------

func TestParseFiscalPeriod(t *testing.T) {
	tests := []struct {
		input string
		want  *int
	}{
		{"Q1", intPointer(1)},
		{"Q2", intPointer(2)},
		{"Q3", intPointer(3)},
		{"Q4", intPointer(4)},
		{"FY", nil},
		{"TTM", nil},
		{"", nil},
		{"Q5", nil},
		{"annual", nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseFiscalPeriod(tt.input)
			if tt.want == nil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, *tt.want, *got)
			}
		})
	}
}

func intPointer(i int) *int { return &i }

// ---------------------------------------------------------------------------
// ParseTimeframe — pure function
// ---------------------------------------------------------------------------

func TestParseTimeframe(t *testing.T) {
	tests := []struct {
		input string
		want  models.Timeframe
	}{
		{"Q1", models.TimeframeQuarterly},
		{"Q2", models.TimeframeQuarterly},
		{"Q3", models.TimeframeQuarterly},
		{"Q4", models.TimeframeQuarterly},
		{"FY", models.TimeframeAnnual},
		{"TTM", models.TimeframeTTM},
		{"", models.TimeframeQuarterly},        // default
		{"unknown", models.TimeframeQuarterly}, // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseTimeframe(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// CalculateYoYChange — pure function
// ---------------------------------------------------------------------------

func TestCalculateYoYChange_Normal(t *testing.T) {
	result := CalculateYoYChange(110, 100)
	require.NotNil(t, result)
	assert.InDelta(t, 0.10, *result, 0.001)
}

func TestCalculateYoYChange_Decline(t *testing.T) {
	result := CalculateYoYChange(90, 100)
	require.NotNil(t, result)
	assert.InDelta(t, -0.10, *result, 0.001)
}

func TestCalculateYoYChange_ZeroPrevious(t *testing.T) {
	result := CalculateYoYChange(100, 0)
	assert.Nil(t, result, "should return nil when previous is zero")
}

func TestCalculateYoYChange_NegativeToPositive(t *testing.T) {
	result := CalculateYoYChange(50, -100)
	require.NotNil(t, result)
	// (50 - (-100)) / abs(-100) = 150/100 = 1.5
	assert.InDelta(t, 1.5, *result, 0.001)
}

func TestCalculateYoYChange_BothNegative(t *testing.T) {
	result := CalculateYoYChange(-50, -100)
	require.NotNil(t, result)
	// (-50 - (-100)) / abs(-100) = 50/100 = 0.5
	assert.InDelta(t, 0.5, *result, 0.001)
}

func TestCalculateYoYChange_NoChange(t *testing.T) {
	result := CalculateYoYChange(100, 100)
	require.NotNil(t, result)
	assert.InDelta(t, 0.0, *result, 0.001)
}

func TestCalculateYoYChange_LargeGrowth(t *testing.T) {
	result := CalculateYoYChange(1000, 100)
	require.NotNil(t, result)
	assert.InDelta(t, 9.0, *result, 0.001) // 900% growth
}

// ---------------------------------------------------------------------------
// absFloat64 — pure function
// ---------------------------------------------------------------------------

func TestAbsFloat64_Positive(t *testing.T) {
	assert.Equal(t, 5.5, absFloat64(5.5))
}

func TestAbsFloat64_Negative(t *testing.T) {
	assert.Equal(t, 5.5, absFloat64(-5.5))
}

func TestAbsFloat64_Zero(t *testing.T) {
	assert.Equal(t, 0.0, absFloat64(0.0))
}

// ---------------------------------------------------------------------------
// CalculateYoYChanges — pure function
// ---------------------------------------------------------------------------

func TestCalculateYoYChanges_Normal(t *testing.T) {
	current := models.FinancialData{
		"revenue":    float64(110000),
		"net_income": float64(22000),
	}
	previous := models.FinancialData{
		"revenue":    float64(100000),
		"net_income": float64(20000),
	}

	changes := CalculateYoYChanges(current, previous)
	require.NotNil(t, changes["revenue"])
	assert.InDelta(t, 0.10, *changes["revenue"], 0.001)
	require.NotNil(t, changes["net_income"])
	assert.InDelta(t, 0.10, *changes["net_income"], 0.001)
}

func TestCalculateYoYChanges_SkipsMetadataFields(t *testing.T) {
	current := models.FinancialData{
		"revenue":       float64(110000),
		"revenue_label": "Total Revenue",
		"revenue_unit":  "USD",
	}
	previous := models.FinancialData{
		"revenue":       float64(100000),
		"revenue_label": "Total Revenue",
		"revenue_unit":  "USD",
	}

	changes := CalculateYoYChanges(current, previous)
	// Only revenue should have a change, not _label or _unit keys
	assert.NotNil(t, changes["revenue"])
	assert.Nil(t, changes["revenue_label"])
	assert.Nil(t, changes["revenue_unit"])
}

func TestCalculateYoYChanges_MissingPrevious(t *testing.T) {
	current := models.FinancialData{
		"revenue":    float64(110000),
		"new_metric": float64(5000),
	}
	previous := models.FinancialData{
		"revenue": float64(100000),
		// new_metric doesn't exist in previous
	}

	changes := CalculateYoYChanges(current, previous)
	assert.NotNil(t, changes["revenue"])
	assert.Nil(t, changes["new_metric"], "should be nil when previous doesn't have the key")
}

func TestCalculateYoYChanges_ZeroPrevious(t *testing.T) {
	current := models.FinancialData{
		"revenue": float64(110000),
	}
	previous := models.FinancialData{
		"revenue": float64(0),
	}

	changes := CalculateYoYChanges(current, previous)
	assert.Nil(t, changes["revenue"], "should be nil when previous value is zero")
}

func TestCalculateYoYChanges_NonFloat64Values(t *testing.T) {
	current := models.FinancialData{
		"revenue": "not a number",
		"count":   int64(100),
	}
	previous := models.FinancialData{
		"revenue": float64(100000),
		"count":   int64(50),
	}

	changes := CalculateYoYChanges(current, previous)
	// Neither should produce a change (type mismatch)
	assert.Nil(t, changes["revenue"])
	assert.Nil(t, changes["count"])
}

func TestCalculateYoYChanges_Empty(t *testing.T) {
	changes := CalculateYoYChanges(models.FinancialData{}, models.FinancialData{})
	assert.Empty(t, changes)
}

// ---------------------------------------------------------------------------
// convertToFinancialData — pure function
// ---------------------------------------------------------------------------

func TestConvertToFinancialData_Normal(t *testing.T) {
	values := map[string]models.PolygonFinancialValue{
		"revenues": {
			Value: 394328000000.0,
			Label: "Revenues",
			Unit:  "USD",
		},
		"net_income_loss": {
			Value: 99803000000.0,
			Label: "Net Income (Loss)",
			Unit:  "USD",
		},
	}

	data := convertToFinancialData(values)

	assert.Equal(t, 394328000000.0, data["revenues"])
	assert.Equal(t, "Revenues", data["revenues_label"])
	assert.Equal(t, "USD", data["revenues_unit"])
	assert.Equal(t, 99803000000.0, data["net_income_loss"])
	assert.Equal(t, "Net Income (Loss)", data["net_income_loss_label"])
	assert.Equal(t, "USD", data["net_income_loss_unit"])
}

func TestConvertToFinancialData_Empty(t *testing.T) {
	data := convertToFinancialData(map[string]models.PolygonFinancialValue{})
	assert.Empty(t, data)
}

func TestConvertToFinancialData_NilMap(t *testing.T) {
	data := convertToFinancialData(nil)
	assert.NotNil(t, data, "should return empty map, not nil")
	assert.Empty(t, data)
}

// ---------------------------------------------------------------------------
// NewRateLimiter — constructor
// ---------------------------------------------------------------------------

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(10, time.Minute)
	require.NotNil(t, rl)
	assert.Equal(t, 10, rl.maxRequests)
	assert.Equal(t, time.Minute, rl.window)
	assert.Equal(t, 0, rl.requestCount)
}

// ---------------------------------------------------------------------------
// NewPolygonFinancialsClient — constructor
// ---------------------------------------------------------------------------

func TestNewPolygonFinancialsClient(t *testing.T) {
	client := NewPolygonFinancialsClient()
	require.NotNil(t, client)
	require.NotNil(t, client.PolygonClient)
	require.NotNil(t, client.rateLimiter)
	assert.Equal(t, 5, client.rateLimiter.maxRequests)
}

// ---------------------------------------------------------------------------
// buildFinancialsURL — pure function
// ---------------------------------------------------------------------------

func TestBuildFinancialsURL_BasicParams(t *testing.T) {
	client := NewPolygonFinancialsClient()
	params := FinancialsRequestParams{
		Ticker:    "AAPL",
		Timeframe: "quarterly",
		Limit:     10,
	}

	url := client.buildFinancialsURL(EndpointIncomeStatements, params)

	assert.Contains(t, url, "vX/reference/financials")
	assert.Contains(t, url, "ticker=AAPL")
	assert.Contains(t, url, "timeframe=quarterly")
	assert.Contains(t, url, "limit=10")
}

func TestBuildFinancialsURL_WithFiscalYear(t *testing.T) {
	client := NewPolygonFinancialsClient()
	fy := 2023
	params := FinancialsRequestParams{
		Ticker:     "MSFT",
		Timeframe:  "annual",
		FiscalYear: &fy,
		Limit:      5,
	}

	url := client.buildFinancialsURL(EndpointBalanceSheets, params)

	assert.Contains(t, url, "ticker=MSFT")
	assert.Contains(t, url, "timeframe=annual")
	assert.Contains(t, url, "fiscal_year=2023")
}

func TestBuildFinancialsURL_WithFiscalQuarter(t *testing.T) {
	client := NewPolygonFinancialsClient()
	fq := 3
	params := FinancialsRequestParams{
		Ticker:        "GOOG",
		FiscalQuarter: &fq,
	}

	url := client.buildFinancialsURL(EndpointCashFlowStatements, params)

	assert.Contains(t, url, "fiscal_period=Q3")
}

func TestBuildFinancialsURL_DefaultLimit(t *testing.T) {
	client := NewPolygonFinancialsClient()
	params := FinancialsRequestParams{
		Ticker: "AMZN",
		// Limit is 0 (default)
	}

	url := client.buildFinancialsURL(EndpointIncomeStatements, params)

	assert.Contains(t, url, "limit=100") // default limit
}

func TestBuildFinancialsURL_DefaultSort(t *testing.T) {
	client := NewPolygonFinancialsClient()
	params := FinancialsRequestParams{
		Ticker: "TSLA",
		// Sort is empty (default)
	}

	url := client.buildFinancialsURL(EndpointIncomeStatements, params)

	assert.Contains(t, url, "sort=period_of_report_date")
	assert.Contains(t, url, "order=desc")
}

func TestBuildFinancialsURL_CustomSort(t *testing.T) {
	client := NewPolygonFinancialsClient()
	params := FinancialsRequestParams{
		Ticker: "META",
		Sort:   "filing_date.asc",
	}

	url := client.buildFinancialsURL(EndpointRatios, params)

	assert.Contains(t, url, "sort=filing_date.asc")
	// Should NOT contain the default order when custom sort is provided
	assert.False(t, strings.Contains(url, "order=desc"))
}

func TestBuildFinancialsURL_TickerUppercased(t *testing.T) {
	client := NewPolygonFinancialsClient()
	params := FinancialsRequestParams{
		Ticker: "aapl", // lowercase
	}

	url := client.buildFinancialsURL(EndpointIncomeStatements, params)

	assert.Contains(t, url, "ticker=AAPL")
}

func TestBuildFinancialsURL_EmptyTicker(t *testing.T) {
	client := NewPolygonFinancialsClient()
	params := FinancialsRequestParams{
		// Ticker is empty
		Limit: 10,
	}

	url := client.buildFinancialsURL(EndpointIncomeStatements, params)

	// Should not contain ticker= at all
	assert.NotContains(t, url, "ticker=")
	assert.Contains(t, url, "limit=10")
}

// ---------------------------------------------------------------------------
// ConvertPolygonToFinancialStatement — pure function
// ---------------------------------------------------------------------------

func TestConvertPolygonToFinancialStatement_Basic(t *testing.T) {
	data := models.PolygonFinancialsData{
		StartDate:    "2024-01-01",
		EndDate:      "2024-03-31",
		FilingDate:   "2024-05-01",
		FiscalYear:   "2024",
		FiscalPeriod: "Q1",
		CIK:          "0000320193",
		Financials: models.PolygonFinancialsItems{
			IncomeStatement: map[string]models.PolygonFinancialValue{
				"revenues": {Value: 90753000000, Label: "Revenues", Unit: "USD"},
			},
		},
	}

	stmt, err := ConvertPolygonToFinancialStatement(data, 42, models.StatementTypeIncome)
	require.NoError(t, err)
	require.NotNil(t, stmt)

	assert.Equal(t, 42, stmt.TickerID)
	assert.Equal(t, models.StatementTypeIncome, stmt.StatementType)
	assert.Equal(t, models.TimeframeQuarterly, stmt.Timeframe)
	assert.Equal(t, 2024, stmt.FiscalYear)
	require.NotNil(t, stmt.FiscalQuarter)
	assert.Equal(t, 1, *stmt.FiscalQuarter)
	assert.Equal(t, "2024-03-31", stmt.PeriodEnd.Format("2006-01-02"))
	require.NotNil(t, stmt.PeriodStart)
	assert.Equal(t, "2024-01-01", stmt.PeriodStart.Format("2006-01-02"))
	require.NotNil(t, stmt.FiledDate)
	assert.Equal(t, "2024-05-01", stmt.FiledDate.Format("2006-01-02"))
	require.NotNil(t, stmt.CIK)
	assert.Equal(t, "0000320193", *stmt.CIK)
}

func TestConvertPolygonToFinancialStatement_AnnualPeriod(t *testing.T) {
	data := models.PolygonFinancialsData{
		EndDate:      "2024-09-28",
		FiscalYear:   "2024",
		FiscalPeriod: "FY",
		Financials: models.PolygonFinancialsItems{
			IncomeStatement: map[string]models.PolygonFinancialValue{
				"revenues": {Value: 391035000000, Label: "Revenues", Unit: "USD"},
			},
		},
	}

	stmt, err := ConvertPolygonToFinancialStatement(data, 1, models.StatementTypeIncome)
	require.NoError(t, err)
	assert.Equal(t, models.TimeframeAnnual, stmt.Timeframe)
	assert.Nil(t, stmt.FiscalQuarter, "FY should have nil quarter")
}

func TestConvertPolygonToFinancialStatement_InvalidEndDate(t *testing.T) {
	data := models.PolygonFinancialsData{
		EndDate:    "not-a-date",
		FiscalYear: "2024",
	}

	_, err := ConvertPolygonToFinancialStatement(data, 1, models.StatementTypeIncome)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "end date")
}

func TestConvertPolygonToFinancialStatement_InvalidFiscalYear(t *testing.T) {
	data := models.PolygonFinancialsData{
		EndDate:    "2024-03-31",
		FiscalYear: "not-a-year",
	}

	_, err := ConvertPolygonToFinancialStatement(data, 1, models.StatementTypeIncome)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fiscal year")
}

func TestConvertPolygonToFinancialStatement_EmptyCIK(t *testing.T) {
	data := models.PolygonFinancialsData{
		EndDate:      "2024-03-31",
		FiscalYear:   "2024",
		FiscalPeriod: "Q1",
		CIK:          "", // empty
		Financials:   models.PolygonFinancialsItems{},
	}

	stmt, err := ConvertPolygonToFinancialStatement(data, 1, models.StatementTypeIncome)
	require.NoError(t, err)
	assert.Nil(t, stmt.CIK, "empty CIK should be nil pointer")
}

func TestConvertPolygonToFinancialStatement_BalanceSheet(t *testing.T) {
	data := models.PolygonFinancialsData{
		EndDate:      "2024-03-31",
		FiscalYear:   "2024",
		FiscalPeriod: "Q1",
		Financials: models.PolygonFinancialsItems{
			BalanceSheet: map[string]models.PolygonFinancialValue{
				"assets": {Value: 352583000000, Label: "Total Assets", Unit: "USD"},
			},
		},
	}

	stmt, err := ConvertPolygonToFinancialStatement(data, 1, models.StatementTypeBalanceSheet)
	require.NoError(t, err)
	assert.Equal(t, models.StatementTypeBalanceSheet, stmt.StatementType)
	assert.Equal(t, 352583000000.0, stmt.Data["assets"])
}

func TestConvertPolygonToFinancialStatement_CashFlow(t *testing.T) {
	data := models.PolygonFinancialsData{
		EndDate:      "2024-03-31",
		FiscalYear:   "2024",
		FiscalPeriod: "Q1",
		Financials: models.PolygonFinancialsItems{
			CashFlowStatement: map[string]models.PolygonFinancialValue{
				"net_cash_flow": {Value: 28000000000, Label: "Net Cash Flow", Unit: "USD"},
			},
		},
	}

	stmt, err := ConvertPolygonToFinancialStatement(data, 1, models.StatementTypeCashFlow)
	require.NoError(t, err)
	assert.Equal(t, models.StatementTypeCashFlow, stmt.StatementType)
	assert.Equal(t, 28000000000.0, stmt.Data["net_cash_flow"])
}

// ---------------------------------------------------------------------------
// FinancialsEndpoint constants
// ---------------------------------------------------------------------------

func TestFinancialsEndpointConstants(t *testing.T) {
	assert.Equal(t, FinancialsEndpoint("income-statements"), EndpointIncomeStatements)
	assert.Equal(t, FinancialsEndpoint("balance-sheets"), EndpointBalanceSheets)
	assert.Equal(t, FinancialsEndpoint("cash-flow-statements"), EndpointCashFlowStatements)
	assert.Equal(t, FinancialsEndpoint("ratios"), EndpointRatios)
}
