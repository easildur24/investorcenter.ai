package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// ConvertToPercentage — pure function
// ---------------------------------------------------------------------------

func TestConvertToPercentage(t *testing.T) {
	tests := []struct {
		name string
		val  *float64
		want *float64
	}{
		{"nil returns nil", nil, nil},
		{"zero", float64Ptr(0), float64Ptr(0)},
		{"positive decimal", float64Ptr(0.47), float64Ptr(47.0)},
		{"one hundred percent", float64Ptr(1.0), float64Ptr(100.0)},
		{"negative", float64Ptr(-0.15), float64Ptr(-15.0)},
		{"small fraction", float64Ptr(0.001), float64Ptr(0.1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertToPercentage(tt.val)
			if tt.want == nil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.InDelta(t, *tt.want, *got, 0.0001)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CalculateCAGR — pure function
// ---------------------------------------------------------------------------

func TestCalculateCAGR(t *testing.T) {
	tests := []struct {
		name       string
		startValue float64
		endValue   float64
		years      int
		wantNil    bool
		wantApprox float64
	}{
		{"basic 10% growth over 1 year", 100, 110, 1, false, 10.0},
		{"100% growth over 1 year", 100, 200, 1, false, 100.0},
		{"doubling over 7 years", 100, 200, 7, false, 10.41}, // (2^(1/7) - 1) * 100
		{"no growth", 100, 100, 5, false, 0.0},
		{"negative growth", 100, 80, 1, false, -20.0},
		{"zero start value", 0, 100, 5, true, 0},
		{"negative start value", -100, 100, 5, true, 0},
		{"zero years", 100, 200, 0, true, 0},
		{"negative years", 100, 200, -1, true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateCAGR(tt.startValue, tt.endValue, tt.years)
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.InDelta(t, tt.wantApprox, *got, 0.1)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetZScoreInterpretation — pure function
// ---------------------------------------------------------------------------

func TestGetZScoreInterpretation(t *testing.T) {
	tests := []struct {
		score        float64
		wantZone     string
		wantNonEmpty bool
	}{
		{3.5, "safe", true},
		{3.0, "safe", true},  // exactly 3.0 is > 2.99
		{2.99, "grey", true}, // exactly 2.99 is NOT > 2.99
		{2.5, "grey", true},
		{1.82, "grey", true},
		{1.81, "distress", true}, // exactly 1.81 is NOT > 1.81
		{1.0, "distress", true},
		{0.0, "distress", true},
		{-1.0, "distress", true},
	}

	for _, tt := range tests {
		t.Run(tt.wantZone, func(t *testing.T) {
			zone, desc := GetZScoreInterpretation(tt.score)
			assert.Equal(t, tt.wantZone, zone)
			assert.NotEmpty(t, desc)
		})
	}
}

// ---------------------------------------------------------------------------
// GetFScoreInterpretation — pure function
// ---------------------------------------------------------------------------

func TestGetFScoreInterpretation(t *testing.T) {
	tests := []struct {
		score     int
		wantLevel string
	}{
		{9, "strong"},
		{8, "strong"},
		{7, "average"},
		{5, "average"},
		{4, "weak"},
		{0, "weak"},
	}

	for _, tt := range tests {
		t.Run(tt.wantLevel, func(t *testing.T) {
			level, desc := GetFScoreInterpretation(tt.score)
			assert.Equal(t, tt.wantLevel, level)
			assert.NotEmpty(t, desc)
		})
	}
}

// ---------------------------------------------------------------------------
// GetPEGInterpretation — pure function
// ---------------------------------------------------------------------------

func TestGetPEGInterpretation(t *testing.T) {
	tests := []struct {
		peg      float64
		wantZone string
	}{
		{0.5, "undervalued"},
		{0.99, "undervalued"},
		{1.0, "fair"},
		{1.5, "fair"},
		{1.6, "high"},
		{2.0, "high"},
		{2.1, "overvalued"},
		{5.0, "overvalued"},
	}

	for _, tt := range tests {
		t.Run(tt.wantZone, func(t *testing.T) {
			zone, desc := GetPEGInterpretation(tt.peg)
			assert.Equal(t, tt.wantZone, zone)
			assert.NotEmpty(t, desc)
		})
	}
}

// ---------------------------------------------------------------------------
// GetPayoutRatioInterpretation — pure function
// ---------------------------------------------------------------------------

func TestGetPayoutRatioInterpretation(t *testing.T) {
	tests := []struct {
		ratio    float64
		wantZone string
	}{
		{10, "very_safe"},
		{29, "very_safe"},
		{30, "safe"},
		{49, "safe"},
		{50, "moderate"},
		{74, "moderate"},
		{75, "at_risk"},
		{100, "at_risk"},
	}

	for _, tt := range tests {
		t.Run(tt.wantZone, func(t *testing.T) {
			zone, desc := GetPayoutRatioInterpretation(tt.ratio)
			assert.Equal(t, tt.wantZone, zone)
			assert.NotEmpty(t, desc)
		})
	}
}

// ---------------------------------------------------------------------------
// coalesce / coalesceWithSource — pure functions
// ---------------------------------------------------------------------------

// coalesce returns the first non-nil value (test helper)
func coalesce(values ...*float64) *float64 {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

func TestCoalesce(t *testing.T) {
	a := 1.0
	b := 2.0

	tests := []struct {
		name   string
		values []*float64
		want   *float64
	}{
		{"all nil", []*float64{nil, nil, nil}, nil},
		{"first non-nil", []*float64{&a, &b}, &a},
		{"skip nils", []*float64{nil, &b}, &b},
		{"single value", []*float64{&a}, &a},
		{"empty", []*float64{}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := coalesce(tt.values...)
			if tt.want == nil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, *tt.want, *got)
			}
		})
	}
}

func TestCoalesceWithSource(t *testing.T) {
	fmpVal := 42.0
	dbVal := 24.0

	tests := []struct {
		name       string
		fmpVal     *float64
		dbVal      *float64
		wantVal    *float64
		wantSource DataSource
	}{
		{"FMP preferred", &fmpVal, &dbVal, &fmpVal, SourceFMP},
		{"FMP only", &fmpVal, nil, &fmpVal, SourceFMP},
		{"DB fallback", nil, &dbVal, &dbVal, SourceDatabase},
		{"both nil", nil, nil, nil, SourceNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, source := coalesceWithSource(tt.fmpVal, tt.dbVal)
			assert.Equal(t, tt.wantSource, source)
			if tt.wantVal == nil {
				assert.Nil(t, val)
			} else {
				require.NotNil(t, val)
				assert.Equal(t, *tt.wantVal, *val)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// sourceFor — pure function
// ---------------------------------------------------------------------------

func TestSourceFor(t *testing.T) {
	val := 42.0

	assert.Equal(t, SourceDatabase, sourceFor(&val, SourceDatabase))
	assert.Equal(t, SourceFMP, sourceFor(&val, SourceFMP))
	assert.Equal(t, SourceNone, sourceFor(nil, SourceDatabase))
	assert.Equal(t, SourceNone, sourceFor(nil, SourceFMP))
}

// ---------------------------------------------------------------------------
// countConsecutiveDividendYears — pure function
// ---------------------------------------------------------------------------

func TestCountConsecutiveDividendYears(t *testing.T) {
	tests := []struct {
		name      string
		dividends []FMPDividendHistorical
		want      int
	}{
		{"empty", []FMPDividendHistorical{}, 0},
		{
			"single year",
			[]FMPDividendHistorical{
				{Date: "2024-01-15"},
				{Date: "2024-04-15"},
				{Date: "2024-07-15"},
				{Date: "2024-10-15"},
			},
			1,
		},
		{
			"multiple years",
			[]FMPDividendHistorical{
				{Date: "2024-01-15"},
				{Date: "2023-10-15"},
				{Date: "2023-07-15"},
				{Date: "2022-04-15"},
				{Date: "2021-01-15"},
			},
			4,
		},
		{
			"short dates ignored",
			[]FMPDividendHistorical{
				{Date: "202"},
			},
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countConsecutiveDividendYears(tt.dividends)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// estimateDividendFrequency — pure function
// ---------------------------------------------------------------------------

func TestEstimateDividendFrequency(t *testing.T) {
	now := time.Now()

	t.Run("less than 2 dividends returns unknown", func(t *testing.T) {
		result := estimateDividendFrequency([]FMPDividendHistorical{
			{Date: now.Format("2006-01-02")},
		})
		assert.Equal(t, "unknown", result)
	})

	t.Run("quarterly dividends", func(t *testing.T) {
		divs := make([]FMPDividendHistorical, 4)
		for i := 0; i < 4; i++ {
			divs[i] = FMPDividendHistorical{
				Date: now.AddDate(0, -3*i, 0).Format("2006-01-02"),
			}
		}
		result := estimateDividendFrequency(divs)
		assert.Equal(t, "quarterly", result)
	})

	t.Run("monthly dividends", func(t *testing.T) {
		divs := make([]FMPDividendHistorical, 12)
		for i := 0; i < 12; i++ {
			divs[i] = FMPDividendHistorical{
				Date: now.AddDate(0, -i, 0).Format("2006-01-02"),
			}
		}
		result := estimateDividendFrequency(divs)
		assert.Equal(t, "monthly", result)
	})

	t.Run("annual dividends", func(t *testing.T) {
		divs := []FMPDividendHistorical{
			{Date: now.AddDate(0, -3, 0).Format("2006-01-02")},
			{Date: now.AddDate(-1, -3, 0).Format("2006-01-02")},
		}
		result := estimateDividendFrequency(divs)
		assert.Equal(t, "annual", result)
	})

	t.Run("no recent dividends returns irregular", func(t *testing.T) {
		divs := []FMPDividendHistorical{
			{Date: now.AddDate(-3, 0, 0).Format("2006-01-02")},
			{Date: now.AddDate(-4, 0, 0).Format("2006-01-02")},
		}
		result := estimateDividendFrequency(divs)
		assert.Equal(t, "irregular", result)
	})
}

// ---------------------------------------------------------------------------
// MergeAllData — integration-style pure function test
// ---------------------------------------------------------------------------

func TestMergeAllData_NilFMP(t *testing.T) {
	merged := MergeAllData(nil, 100.0)
	require.NotNil(t, merged)
	assert.False(t, merged.FMPAvailable)
}

func TestMergeAllData_EmptyFMP(t *testing.T) {
	fmp := &FMPAllMetrics{
		Errors: make(map[string]error),
	}
	merged := MergeAllData(fmp, 100.0)
	require.NotNil(t, merged)
	assert.False(t, merged.FMPAvailable)
}

func TestMergeAllData_WithRatios(t *testing.T) {
	pe := 25.0
	pb := 5.0
	gm := 0.45
	roe := 0.25

	fmp := &FMPAllMetrics{
		RatiosTTM: &FMPRatiosTTM{
			PriceToEarningsRatioTTM: &pe,
			PriceToBookRatioTTM:     &pb,
			GrossProfitMarginTTM:    &gm,
			ReturnOnEquityTTM:       &roe,
		},
		Errors: make(map[string]error),
	}

	merged := MergeAllData(fmp, 100.0)
	require.NotNil(t, merged)
	assert.True(t, merged.FMPAvailable)

	// Valuation
	assert.Equal(t, &pe, merged.PERatio)
	assert.Equal(t, &pb, merged.PBRatio)

	// Profitability (should be converted to percentage)
	require.NotNil(t, merged.GrossMargin)
	assert.InDelta(t, 45.0, *merged.GrossMargin, 0.01)

	require.NotNil(t, merged.ROE)
	assert.InDelta(t, 25.0, *merged.ROE, 0.01)
}

func TestMergeAllData_WithEstimates_ForwardPE(t *testing.T) {
	epsAvg := 5.0

	fmp := &FMPAllMetrics{
		Estimates: []FMPAnalystEstimate{
			{EstimatedEPSAvg: &epsAvg},
		},
		Errors: make(map[string]error),
	}

	merged := MergeAllData(fmp, 100.0) // price = 100
	require.NotNil(t, merged)

	// Forward PE = price / forward EPS = 100 / 5 = 20
	require.NotNil(t, merged.ForwardPE)
	assert.InDelta(t, 20.0, *merged.ForwardPE, 0.01)
	assert.Equal(t, SourceCalculated, merged.Sources.ForwardPE)
}

func TestMergeAllData_WithScores(t *testing.T) {
	zScore := 3.5
	fScore := 8

	fmp := &FMPAllMetrics{
		Score: &FMPScore{
			AltmanZScore:   &zScore,
			PiotroskiScore: &fScore,
		},
		Errors: make(map[string]error),
	}

	merged := MergeAllData(fmp, 100.0)
	require.NotNil(t, merged)

	assert.Equal(t, &zScore, merged.AltmanZScore)
	assert.Equal(t, &fScore, merged.PiotroskiFScore)

	require.NotNil(t, merged.AltmanZInterpretation)
	assert.Equal(t, "safe", *merged.AltmanZInterpretation)

	require.NotNil(t, merged.PiotroskiFInterpretation)
	assert.Equal(t, "strong", *merged.PiotroskiFInterpretation)
}

func TestMergeAllData_WithGrowth(t *testing.T) {
	revGrowth := 0.15
	epsGrowth := 0.20

	fmp := &FMPAllMetrics{
		Growth: []FMPFinancialGrowth{
			{
				RevenueGrowth: &revGrowth,
				EPSGrowth:     &epsGrowth,
			},
		},
		Errors: make(map[string]error),
	}

	merged := MergeAllData(fmp, 100.0)
	require.NotNil(t, merged)

	require.NotNil(t, merged.RevenueGrowthYoY)
	assert.InDelta(t, 15.0, *merged.RevenueGrowthYoY, 0.01)

	require.NotNil(t, merged.EPSGrowthYoY)
	assert.InDelta(t, 20.0, *merged.EPSGrowthYoY, 0.01)
}

func TestMergeAllData_WithGradesSummary(t *testing.T) {
	fmp := &FMPAllMetrics{
		GradesSummary: &FMPGradesSummary{
			StrongBuy:  10,
			Buy:        15,
			Hold:       5,
			Sell:       2,
			StrongSell: 1,
			Consensus:  "Buy",
		},
		Errors: make(map[string]error),
	}

	merged := MergeAllData(fmp, 100.0)
	require.NotNil(t, merged)

	assert.Equal(t, 10, *merged.AnalystRatingStrongBuy)
	assert.Equal(t, 15, *merged.AnalystRatingBuy)
	assert.Equal(t, 5, *merged.AnalystRatingHold)
	assert.Equal(t, 2, *merged.AnalystRatingSell)
	assert.Equal(t, 1, *merged.AnalystRatingStrongSell)
	assert.Equal(t, "Buy", *merged.AnalystConsensus)
}

func TestMergeAllData_WithPriceTargets(t *testing.T) {
	high := 200.0
	low := 80.0
	consensus := 150.0
	median := 145.0

	fmp := &FMPAllMetrics{
		PriceTargetConsensus: &FMPPriceTargetConsensus{
			TargetHigh:      &high,
			TargetLow:       &low,
			TargetConsensus: &consensus,
			TargetMedian:    &median,
		},
		Errors: make(map[string]error),
	}

	merged := MergeAllData(fmp, 100.0)
	require.NotNil(t, merged)

	assert.Equal(t, &high, merged.TargetHigh)
	assert.Equal(t, &low, merged.TargetLow)
	assert.Equal(t, &consensus, merged.TargetConsensus)
	assert.Equal(t, &median, merged.TargetMedian)
}

func TestMergeAllData_CalculatedFCFPayoutRatio(t *testing.T) {
	dps := 2.0
	fcfps := 10.0

	fmp := &FMPAllMetrics{
		RatiosTTM: &FMPRatiosTTM{
			DividendPerShareTTM:     &dps,
			FreeCashFlowPerShareTTM: &fcfps,
		},
		Errors: make(map[string]error),
	}

	merged := MergeAllData(fmp, 100.0)
	require.NotNil(t, merged)

	// FCF Payout = (DPS / FCF per share) * 100 = (2/10) * 100 = 20%
	require.NotNil(t, merged.FCFPayoutRatio)
	assert.InDelta(t, 20.0, *merged.FCFPayoutRatio, 0.01)
}

func TestMergeAllData_CalculatedForwardDividendYield(t *testing.T) {
	dps := 3.0

	fmp := &FMPAllMetrics{
		RatiosTTM: &FMPRatiosTTM{
			DividendPerShareTTM: &dps,
		},
		Errors: make(map[string]error),
	}

	merged := MergeAllData(fmp, 100.0) // price = 100
	require.NotNil(t, merged)

	// Forward Dividend Yield = (DPS / Price) * 100 = (3/100) * 100 = 3%
	require.NotNil(t, merged.ForwardDividendYield)
	assert.InDelta(t, 3.0, *merged.ForwardDividendYield, 0.01)
}

// ---------------------------------------------------------------------------
// MergeWithDBData — legacy merge
// ---------------------------------------------------------------------------

func TestMergeWithDBData_FMPOnly(t *testing.T) {
	pe := 20.0
	gm := 0.45

	fmp := &FMPRatiosTTM{
		PriceToEarningsRatioTTM: &pe,
		GrossProfitMarginTTM:    &gm,
	}

	merged := MergeWithDBData(fmp, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NotNil(t, merged)
	assert.True(t, merged.FMPAvailable)
	assert.Equal(t, &pe, merged.PERatio)
	assert.Equal(t, SourceFMP, merged.Sources.PERatio)
}

func TestMergeWithDBData_DBFallback(t *testing.T) {
	dbPE := 18.0
	dbGM := 40.0

	merged := MergeWithDBData(nil, &dbGM, nil, nil, nil, nil, nil, nil, nil, &dbPE, nil, nil)
	require.NotNil(t, merged)
	assert.False(t, merged.FMPAvailable)
	assert.Equal(t, &dbPE, merged.PERatio)
	assert.Equal(t, &dbGM, merged.GrossMargin)
	assert.Equal(t, SourceDatabase, merged.Sources.PERatio)
}

func TestMergeWithDBData_FMPThenDB(t *testing.T) {
	fmpPE := 25.0
	dbPE := 18.0

	fmp := &FMPRatiosTTM{
		PriceToEarningsRatioTTM: &fmpPE,
	}

	merged := MergeWithDBData(fmp, nil, nil, nil, nil, nil, nil, nil, nil, &dbPE, nil, nil)
	require.NotNil(t, merged)
	// FMP should win
	assert.Equal(t, &fmpPE, merged.PERatio)
	assert.Equal(t, SourceFMP, merged.Sources.PERatio)
}

// ---------------------------------------------------------------------------
// NewFMPClient
// ---------------------------------------------------------------------------

func TestNewFMPClient_NoKey(t *testing.T) {
	t.Setenv("FMP_API_KEY", "")
	client := NewFMPClient()
	require.NotNil(t, client)
	assert.Empty(t, client.APIKey)
}

func TestNewFMPClient_WithKey(t *testing.T) {
	t.Setenv("FMP_API_KEY", "test-key-123")
	client := NewFMPClient()
	require.NotNil(t, client)
	assert.Equal(t, "test-key-123", client.APIKey)
}

func TestFMPClient_GetRatiosTTM_NoKey(t *testing.T) {
	client := &FMPClient{APIKey: ""}
	_, err := client.GetRatiosTTM("AAPL")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestFMPClient_GetKeyMetricsTTM_NoKey(t *testing.T) {
	client := &FMPClient{APIKey: ""}
	_, err := client.GetKeyMetricsTTM("AAPL")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestFMPClient_GetFinancialGrowth_NoKey(t *testing.T) {
	client := &FMPClient{APIKey: ""}
	_, err := client.GetFinancialGrowth("AAPL", 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestFMPClient_GetAnalystEstimates_NoKey(t *testing.T) {
	client := &FMPClient{APIKey: ""}
	_, err := client.GetAnalystEstimates("AAPL", 4)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestFMPClient_GetScore_NoKey(t *testing.T) {
	client := &FMPClient{APIKey: ""}
	_, err := client.GetScore("AAPL")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestFMPClient_GetDividendHistory_NoKey(t *testing.T) {
	client := &FMPClient{APIKey: ""}
	_, err := client.GetDividendHistory("AAPL")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestFMPClient_GetGradesSummary_NoKey(t *testing.T) {
	client := &FMPClient{APIKey: ""}
	_, err := client.GetGradesSummary("AAPL")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestFMPClient_GetPriceTargetConsensus_NoKey(t *testing.T) {
	client := &FMPClient{APIKey: ""}
	_, err := client.GetPriceTargetConsensus("AAPL")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

// ---------------------------------------------------------------------------
// DataSource constants
// ---------------------------------------------------------------------------

func TestDataSourceConstants(t *testing.T) {
	assert.Equal(t, DataSource("fmp"), SourceFMP)
	assert.Equal(t, DataSource("database"), SourceDatabase)
	assert.Equal(t, DataSource("calculated"), SourceCalculated)
	assert.Equal(t, DataSource(""), SourceNone)
}
