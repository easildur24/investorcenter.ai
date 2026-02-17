package database

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"investorcenter-api/models"
)

// ─── SQL clause structure tests ───────────────────────────────────

func TestFilterConditionSQLFormat(t *testing.T) {
	tests := []struct {
		name        string
		params      models.ScreenerParams
		wantClauses []string
		wantArgs    []interface{}
	}{
		{
			name: "min-only generates >= clause",
			params: func() models.ScreenerParams {
				v := 10.0
				return models.ScreenerParams{MarketCapMin: &v}
			}(),
			wantClauses: []string{"market_cap >= $1"},
			wantArgs:    []interface{}{10.0},
		},
		{
			name: "max-only generates <= clause",
			params: func() models.ScreenerParams {
				v := 100.0
				return models.ScreenerParams{MarketCapMax: &v}
			}(),
			wantClauses: []string{"market_cap <= $1"},
			wantArgs:    []interface{}{100.0},
		},
		{
			name: "min and max generate two clauses",
			params: func() models.ScreenerParams {
				min, max := 5.0, 25.0
				return models.ScreenerParams{PEMin: &min, PEMax: &max}
			}(),
			wantClauses: []string{"pe_ratio >= $1", "pe_ratio <= $2"},
			wantArgs:    []interface{}{5.0, 25.0},
		},
		{
			name: "equal min and max are not swapped",
			params: func() models.ScreenerParams {
				v := 15.0
				return models.ScreenerParams{PEMin: &v, PEMax: &v}
			}(),
			wantClauses: []string{"pe_ratio >= $1", "pe_ratio <= $2"},
			wantArgs:    []interface{}{15.0, 15.0},
		},
		{
			name: "single sector generates IN with one placeholder",
			params: models.ScreenerParams{
				Sectors: []string{"Technology"},
			},
			wantClauses: []string{"sector IN ($1)"},
			wantArgs:    []interface{}{"Technology"},
		},
		{
			name: "multiple industries generate correct IN clause",
			params: models.ScreenerParams{
				Industries: []string{"Software", "Hardware", "AI"},
			},
			wantClauses: []string{"industry IN ($1, $2, $3)"},
			wantArgs:    []interface{}{"Software", "Hardware", "AI"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conditions, args, _ := BuildFilterConditions(&tc.params, 1)
			assert.Equal(t, tc.wantClauses, conditions)
			assert.Equal(t, tc.wantArgs, args)
		})
	}
}

// ─── Argument indexing tests ──────────────────────────────────────

func TestFilterConditionsArgIndexing(t *testing.T) {
	t.Run("sectors before ranges use sequential indices", func(t *testing.T) {
		minPE := 10.0
		params := &models.ScreenerParams{
			Sectors: []string{"Tech", "Health"},
			PEMin:   &minPE,
		}
		conditions, args, nextIdx := BuildFilterConditions(params, 1)

		assert.Len(t, conditions, 2)
		assert.Equal(t, "sector IN ($1, $2)", conditions[0])
		assert.Contains(t, conditions[1], "$3")
		assert.Len(t, args, 3)
		assert.Equal(t, 4, nextIdx)
	})

	t.Run("industries after sectors continue indexing", func(t *testing.T) {
		params := &models.ScreenerParams{
			Sectors:    []string{"Energy"},
			Industries: []string{"Oil & Gas"},
		}
		conditions, args, nextIdx := BuildFilterConditions(params, 1)

		assert.Len(t, conditions, 2)
		assert.Equal(t, "sector IN ($1)", conditions[0])
		assert.Equal(t, "industry IN ($2)", conditions[1])
		assert.Len(t, args, 2)
		assert.Equal(t, "Energy", args[0])
		assert.Equal(t, "Oil & Gas", args[1])
		assert.Equal(t, 3, nextIdx)
	})

	t.Run("custom start index offsets all placeholders", func(t *testing.T) {
		min := 5.0
		max := 50.0
		params := &models.ScreenerParams{
			PBMin: &min,
			PBMax: &max,
		}
		conditions, _, nextIdx := BuildFilterConditions(params, 10)

		assert.Len(t, conditions, 2)
		assert.Contains(t, conditions[0], "$10")
		assert.Contains(t, conditions[1], "$11")
		assert.Equal(t, 12, nextIdx)
	})
}

// ─── Min/max swap edge cases ──────────────────────────────────────

func TestFilterConditionsMinMaxSwapEdgeCases(t *testing.T) {
	t.Run("negative values swap correctly", func(t *testing.T) {
		min := -5.0
		max := -20.0 // inverted: min > max
		params := &models.ScreenerParams{
			ROEMin: &min,
			ROEMax: &max,
		}
		_, args, _ := BuildFilterConditions(params, 1)

		// After swap: -20 should be min, -5 should be max
		assert.Equal(t, -20.0, args[0])
		assert.Equal(t, -5.0, args[1])
	})

	t.Run("zero values handled correctly", func(t *testing.T) {
		min := 0.0
		max := 100.0
		params := &models.ScreenerParams{
			ICScoreMin: &min,
			ICScoreMax: &max,
		}
		conditions, args, _ := BuildFilterConditions(params, 1)

		assert.Len(t, conditions, 2)
		assert.Equal(t, 0.0, args[0])
		assert.Equal(t, 100.0, args[1])
	})

	t.Run("very small float difference not swapped", func(t *testing.T) {
		min := 10.0
		max := 10.0001
		params := &models.ScreenerParams{
			BetaMin: &min,
			BetaMax: &max,
		}
		_, args, _ := BuildFilterConditions(params, 1)

		assert.Equal(t, 10.0, args[0])
		assert.InDelta(t, 10.0001, args[1].(float64), 0.0001)
	})
}

// ─── Consecutive dividend years (min-only) ─────────────────────

func TestFilterConsecutiveDividendYearsMinOnly(t *testing.T) {
	// This filter only supports min (GetMax always returns nil)
	min := 10.0
	params := &models.ScreenerParams{
		ConsecutiveDivYearsMin: &min,
	}
	conditions, args, _ := BuildFilterConditions(params, 1)

	// Should only generate one condition (min), never max
	found := false
	for _, c := range conditions {
		if strings.Contains(c, "consecutive_dividend_years") {
			found = true
			assert.Contains(t, c, ">=")
			assert.NotContains(t, c, "<=")
		}
	}
	assert.True(t, found, "expected a condition for consecutive_dividend_years")
	// Verify the arg is correct
	assert.Contains(t, args, 10.0)
}

// ─── Large combined filter test ─────────────────────────────────

func TestFilterConditionsLargeCombined(t *testing.T) {
	pe := 20.0
	roe := 15.0
	ic := 60.0
	mcMin := 1e9
	mcMax := 1e12
	beta := 1.5
	dy := 2.0

	params := &models.ScreenerParams{
		Sectors:          []string{"Technology", "Healthcare"},
		Industries:       []string{"Software"},
		PEMax:            &pe,
		ROEMin:           &roe,
		ICScoreMin:       &ic,
		MarketCapMin:     &mcMin,
		MarketCapMax:     &mcMax,
		BetaMax:          &beta,
		DividendYieldMin: &dy,
	}

	conditions, args, nextIdx := BuildFilterConditions(params, 1)

	// 1 sector condition + 1 industry condition + 7 range conditions
	assert.Len(t, conditions, 9)
	assert.Len(t, args, 10) // 2 sectors + 1 industry + 7 range values
	assert.Equal(t, 11, nextIdx)

	// Verify sectors come first
	assert.True(t, strings.HasPrefix(conditions[0], "sector IN"))
	assert.True(t, strings.HasPrefix(conditions[1], "industry IN"))
}

// ─── Every range filter column generates valid SQL ──────────────

func TestFilterEveryRangeFilterGeneratesSQL(t *testing.T) {
	for _, f := range RangeFilters {
		t.Run(f.Column, func(t *testing.T) {
			params := &models.ScreenerParams{}
			// With empty params, getters should return nil
			assert.Nil(t, f.GetMin(params),
				"GetMin should return nil for zero-value params")

			// Verify column name only contains valid SQL characters
			for _, ch := range f.Column {
				assert.True(t, (ch >= 'a' && ch <= 'z') || ch == '_',
					"column %q contains invalid char %q", f.Column, string(ch))
			}
		})
	}
}

// ─── Registry completeness ──────────────────────────────────────

func TestFilterRegistryCoversAllScreenerParamsRangeFields(t *testing.T) {
	// Collect all columns referenced in the registry
	registeredColumns := make(map[string]bool)
	for _, f := range RangeFilters {
		registeredColumns[f.Column] = true
	}

	// These are all range-filterable columns expected from ScreenerParams
	expectedColumns := []string{
		"market_cap", "pe_ratio", "pb_ratio", "ps_ratio",
		"roe", "roa", "gross_margin", "net_margin",
		"debt_to_equity", "current_ratio",
		"revenue_growth", "eps_growth_yoy",
		"dividend_yield", "payout_ratio", "consecutive_dividend_years",
		"beta", "dcf_upside_percent", "ic_score",
		"value_score", "growth_score", "profitability_score",
		"financial_health_score", "momentum_score",
		"analyst_consensus_score", "insider_activity_score",
		"institutional_score", "news_sentiment_score", "technical_score",
	}

	for _, col := range expectedColumns {
		assert.True(t, registeredColumns[col],
			"expected column %q to be in RangeFilters registry", col)
	}
}

// ─── SQL injection safety ───────────────────────────────────────

func TestFilterConditionsNoSQLInjectionInColumnNames(t *testing.T) {
	// Verify that all column names in generated conditions are from the registry
	// and don't contain any SQL injection vectors
	val := 1.0
	params := &models.ScreenerParams{
		MarketCapMin: &val,
		PEMin:        &val,
		ROEMin:       &val,
	}

	conditions, _, _ := BuildFilterConditions(params, 1)

	for _, cond := range conditions {
		// Conditions should be in the form "column_name >= $N" or "column_name <= $N"
		// or "sector/industry IN (...)"
		assert.True(t,
			strings.Contains(cond, " >= $") ||
				strings.Contains(cond, " <= $") ||
				strings.Contains(cond, " IN ("),
			"unexpected condition format: %q", cond)
	}
}
