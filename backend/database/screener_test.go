package database

import (
	"testing"

	"investorcenter-api/models"
)

// TestValidScreenerSortColumns verifies that all sort column mappings
// reference actual columns in the screener_data materialized view.
// This prevents regressions like mapping ic_score -> market_cap
// or beta -> roe, which caused those columns to display as 0.
func TestValidScreenerSortColumns(t *testing.T) {
	// These are the actual column names in the screener_data materialized view
	// (defined in ic-score-service/migrations/019_expand_screener_materialized_view.sql)
	validViewColumns := map[string]bool{
		// Core identity
		"symbol":     true,
		"name":       true,
		"sector":     true,
		"industry":   true,
		"market_cap": true,
		"asset_type": true,
		"active":     true,

		// Price
		"price": true,

		// Valuation
		"pe_ratio": true,
		"pb_ratio": true,
		"ps_ratio": true,

		// Profitability
		"roe":              true,
		"roa":              true,
		"gross_margin":     true,
		"operating_margin": true,
		"net_margin":       true,

		// Financial health
		"debt_to_equity": true,
		"current_ratio":  true,

		// Growth
		"revenue_growth": true,
		"eps_growth_yoy": true,

		// Dividends
		"dividend_yield":             true,
		"payout_ratio":               true,
		"consecutive_dividend_years": true,

		// Risk
		"beta": true,

		// Fair value
		"dcf_upside_percent": true,

		// IC Score
		"ic_score":  true,
		"ic_rating": true,

		// IC Score sub-factors
		"value_score":             true,
		"growth_score":            true,
		"profitability_score":     true,
		"financial_health_score":  true,
		"momentum_score":          true,
		"analyst_consensus_score": true,
		"insider_activity_score":  true,
		"institutional_score":     true,
		"news_sentiment_score":    true,
		"technical_score":         true,

		// IC Score metadata
		"ic_sector_percentile": true,
		"lifecycle_stage":      true,
		"ic_data_completeness": true,

		// View metadata
		"refreshed_at": true,
	}

	for apiKey, dbColumn := range ValidScreenerSortColumns {
		t.Run(apiKey, func(t *testing.T) {
			if !validViewColumns[dbColumn] {
				t.Errorf("sort key %q maps to column %q which does not exist in screener_data view", apiKey, dbColumn)
			}
		})
	}
}

// TestSortColumnSelfMapping verifies that critical columns map to themselves
// (i.e., no placeholder mappings that point to the wrong column).
func TestSortColumnSelfMapping(t *testing.T) {
	criticalMappings := []struct {
		apiKey     string
		wantColumn string
	}{
		// Core
		{"symbol", "symbol"},
		{"market_cap", "market_cap"},
		{"price", "price"},

		// Valuation
		{"pe_ratio", "pe_ratio"},
		{"pb_ratio", "pb_ratio"},
		{"ps_ratio", "ps_ratio"},

		// Profitability
		{"roe", "roe"},
		{"roa", "roa"},
		{"gross_margin", "gross_margin"},
		{"net_margin", "net_margin"},

		// Financial health
		{"debt_to_equity", "debt_to_equity"},
		{"current_ratio", "current_ratio"},

		// Growth
		{"revenue_growth", "revenue_growth"},
		{"eps_growth_yoy", "eps_growth_yoy"},

		// Dividends
		{"dividend_yield", "dividend_yield"},
		{"payout_ratio", "payout_ratio"},

		// Risk
		{"beta", "beta"},

		// Fair value
		{"dcf_upside_percent", "dcf_upside_percent"},

		// IC Score
		{"ic_score", "ic_score"},

		// IC Score sub-factors
		{"value_score", "value_score"},
		{"growth_score", "growth_score"},
		{"profitability_score", "profitability_score"},
		{"financial_health_score", "financial_health_score"},
		{"momentum_score", "momentum_score"},
		{"analyst_consensus_score", "analyst_consensus_score"},
		{"insider_activity_score", "insider_activity_score"},
		{"institutional_score", "institutional_score"},
		{"news_sentiment_score", "news_sentiment_score"},
		{"technical_score", "technical_score"},
	}

	for _, tc := range criticalMappings {
		t.Run(tc.apiKey, func(t *testing.T) {
			got, ok := ValidScreenerSortColumns[tc.apiKey]
			if !ok {
				t.Errorf("sort key %q not found in ValidScreenerSortColumns", tc.apiKey)
				return
			}
			if got != tc.wantColumn {
				t.Errorf("sort key %q maps to %q, want %q", tc.apiKey, got, tc.wantColumn)
			}
		})
	}
}

// TestSortColumnCharacterValidation ensures all sort columns pass
// the defense-in-depth regex check (only lowercase letters and underscores).
func TestSortColumnCharacterValidation(t *testing.T) {
	for apiKey, dbColumn := range ValidScreenerSortColumns {
		t.Run(apiKey, func(t *testing.T) {
			for _, ch := range dbColumn {
				if !((ch >= 'a' && ch <= 'z') || ch == '_') {
					t.Errorf("sort key %q maps to column %q which contains invalid character %q", apiKey, dbColumn, string(ch))
				}
			}
		})
	}
}

// TestNoPlaceholderSortMappings ensures no sort column maps to a
// different column as a "placeholder" — every key must map to
// its own corresponding database column.
func TestNoPlaceholderSortMappings(t *testing.T) {
	// Known legitimate aliases where the API key differs from the DB column
	legitimateAliases := map[string]string{
		// None currently — all should map to their own column name
		// If a legitimate alias is needed in the future, add it here
	}

	for apiKey, dbColumn := range ValidScreenerSortColumns {
		t.Run(apiKey, func(t *testing.T) {
			if expected, isAlias := legitimateAliases[apiKey]; isAlias {
				if dbColumn != expected {
					t.Errorf("sort key %q: expected alias %q but got %q", apiKey, expected, dbColumn)
				}
				return
			}

			// For non-aliases, the column name should match the API key
			if apiKey != dbColumn {
				t.Errorf("sort key %q maps to %q — possible placeholder mapping. If this is intentional, add it to legitimateAliases", apiKey, dbColumn)
			}
		})
	}
}

// TestRangeFilterRegistryColumns verifies that every column referenced in
// the RangeFilters registry exists in the screener_data materialized view.
func TestRangeFilterRegistryColumns(t *testing.T) {
	// Columns in screener_data (from migration 019)
	validViewColumns := map[string]bool{
		"market_cap": true, "price": true,
		"pe_ratio": true, "pb_ratio": true, "ps_ratio": true,
		"roe": true, "roa": true, "gross_margin": true,
		"operating_margin": true, "net_margin": true,
		"debt_to_equity": true, "current_ratio": true,
		"revenue_growth": true, "eps_growth_yoy": true,
		"dividend_yield": true, "payout_ratio": true,
		"consecutive_dividend_years": true,
		"beta":                       true, "dcf_upside_percent": true,
		"ic_score": true, "ic_rating": true,
		"value_score": true, "growth_score": true,
		"profitability_score": true, "financial_health_score": true,
		"momentum_score": true, "analyst_consensus_score": true,
		"insider_activity_score": true, "institutional_score": true,
		"news_sentiment_score": true, "technical_score": true,
		"sector": true, "industry": true,
	}

	for i, f := range RangeFilters {
		t.Run(f.Column, func(t *testing.T) {
			if !validViewColumns[f.Column] {
				t.Errorf("RangeFilter[%d] column %q not in screener_data view", i, f.Column)
			}
		})
	}
}

// TestRangeFilterRegistryColumnsSortable verifies that every column in the
// filter registry is also sortable (present in ValidScreenerSortColumns).
func TestRangeFilterRegistryColumnsSortable(t *testing.T) {
	for i, f := range RangeFilters {
		t.Run(f.Column, func(t *testing.T) {
			if _, ok := ValidScreenerSortColumns[f.Column]; !ok {
				t.Errorf("RangeFilter[%d] column %q is filterable but not sortable (not in ValidScreenerSortColumns)", i, f.Column)
			}
		})
	}
}

// TestBuildFilterConditionsEmpty verifies that with no filters set,
// BuildFilterConditions returns no conditions and no args.
func TestBuildFilterConditionsEmpty(t *testing.T) {
	params := &models.ScreenerParams{}
	conditions, args, nextIdx := BuildFilterConditions(params, 1)

	if len(conditions) != 0 {
		t.Errorf("expected 0 conditions, got %d: %v", len(conditions), conditions)
	}
	if len(args) != 0 {
		t.Errorf("expected 0 args, got %d: %v", len(args), args)
	}
	if nextIdx != 1 {
		t.Errorf("expected nextIdx=1, got %d", nextIdx)
	}
}

// TestBuildFilterConditionsSectors verifies sector IN clause generation.
func TestBuildFilterConditionsSectors(t *testing.T) {
	params := &models.ScreenerParams{
		Sectors: []string{"Technology", "Healthcare"},
	}
	conditions, args, nextIdx := BuildFilterConditions(params, 1)

	if len(conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d: %v", len(conditions), conditions)
	}
	expected := "sector IN ($1, $2)"
	if conditions[0] != expected {
		t.Errorf("expected condition %q, got %q", expected, conditions[0])
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d: %v", len(args), args)
	}
	if args[0] != "Technology" || args[1] != "Healthcare" {
		t.Errorf("unexpected args: %v", args)
	}
	if nextIdx != 3 {
		t.Errorf("expected nextIdx=3, got %d", nextIdx)
	}
}

// TestBuildFilterConditionsIndustries verifies industry IN clause generation.
func TestBuildFilterConditionsIndustries(t *testing.T) {
	params := &models.ScreenerParams{
		Industries: []string{"Software", "Semiconductors", "Biotechnology"},
	}
	conditions, args, nextIdx := BuildFilterConditions(params, 1)

	if len(conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d: %v", len(conditions), conditions)
	}
	expected := "industry IN ($1, $2, $3)"
	if conditions[0] != expected {
		t.Errorf("expected condition %q, got %q", expected, conditions[0])
	}
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(args), args)
	}
	if nextIdx != 4 {
		t.Errorf("expected nextIdx=4, got %d", nextIdx)
	}
}

// TestBuildFilterConditionsRangeFilters verifies min/max range condition generation.
func TestBuildFilterConditionsRangeFilters(t *testing.T) {
	minPE := 5.0
	maxPE := 25.0
	minIC := 70.0

	params := &models.ScreenerParams{
		PEMin:      &minPE,
		PEMax:      &maxPE,
		ICScoreMin: &minIC,
	}
	conditions, args, nextIdx := BuildFilterConditions(params, 1)

	// pe_ratio >= $1, pe_ratio <= $2, ic_score >= $3
	if len(conditions) != 3 {
		t.Fatalf("expected 3 conditions, got %d: %v", len(conditions), conditions)
	}
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(args), args)
	}
	if args[0] != 5.0 || args[1] != 25.0 || args[2] != 70.0 {
		t.Errorf("unexpected args: %v", args)
	}
	if nextIdx != 4 {
		t.Errorf("expected nextIdx=4, got %d", nextIdx)
	}
}

// TestBuildFilterConditionsCombined verifies sectors + range filters together.
func TestBuildFilterConditionsCombined(t *testing.T) {
	minROE := 15.0
	maxBeta := 1.5

	params := &models.ScreenerParams{
		Sectors: []string{"Technology"},
		ROEMin:  &minROE,
		BetaMax: &maxBeta,
	}
	conditions, args, nextIdx := BuildFilterConditions(params, 1)

	// sector IN ($1), roe >= $2, beta <= $3
	if len(conditions) != 3 {
		t.Fatalf("expected 3 conditions, got %d: %v", len(conditions), conditions)
	}
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(args), args)
	}
	if args[0] != "Technology" {
		t.Errorf("expected first arg to be 'Technology', got %v", args[0])
	}
	if args[1] != 15.0 {
		t.Errorf("expected second arg to be 15.0, got %v", args[1])
	}
	if args[2] != 1.5 {
		t.Errorf("expected third arg to be 1.5, got %v", args[2])
	}
	if nextIdx != 4 {
		t.Errorf("expected nextIdx=4, got %d", nextIdx)
	}
}

// TestBuildFilterConditionsStartIndex verifies that arg indexing works
// correctly when starting from a non-1 index.
func TestBuildFilterConditionsStartIndex(t *testing.T) {
	minPrice := 50.0
	params := &models.ScreenerParams{
		MarketCapMin: &minPrice,
	}
	conditions, args, nextIdx := BuildFilterConditions(params, 5)

	if len(conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d: %v", len(conditions), conditions)
	}
	expected := "market_cap >= $5"
	if conditions[0] != expected {
		t.Errorf("expected condition %q, got %q", expected, conditions[0])
	}
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}
	if nextIdx != 6 {
		t.Errorf("expected nextIdx=6, got %d", nextIdx)
	}
}

// TestBuildFilterConditionsAllSubFactors verifies that all 10 IC Score
// sub-factor filters generate correct conditions.
func TestBuildFilterConditionsAllSubFactors(t *testing.T) {
	val := 50.0
	params := &models.ScreenerParams{
		ValueScoreMin:           &val,
		GrowthScoreMin:          &val,
		ProfitabilityScoreMin:   &val,
		FinancialHealthScoreMin: &val,
		MomentumScoreMin:        &val,
		AnalystScoreMin:         &val,
		InsiderScoreMin:         &val,
		InstitutionalScoreMin:   &val,
		SentimentScoreMin:       &val,
		TechnicalScoreMin:       &val,
	}
	conditions, args, _ := BuildFilterConditions(params, 1)

	// 10 sub-factor min filters
	if len(conditions) != 10 {
		t.Errorf("expected 10 conditions for all sub-factors, got %d: %v", len(conditions), conditions)
	}
	if len(args) != 10 {
		t.Errorf("expected 10 args, got %d", len(args))
	}
}

// TestRangeFilterRegistryNoDuplicateColumns verifies no column appears twice
// in the filter registry, which would cause double-filtering.
func TestRangeFilterRegistryNoDuplicateColumns(t *testing.T) {
	seen := map[string]int{}
	for i, f := range RangeFilters {
		if prev, ok := seen[f.Column]; ok {
			t.Errorf("column %q appears at both index %d and %d in RangeFilters", f.Column, prev, i)
		}
		seen[f.Column] = i
	}
}

// TestRangeFilterRegistryAccessors verifies that getters are not nil
// and return nil for zero-value params (no false positives).
func TestRangeFilterRegistryAccessors(t *testing.T) {
	params := &models.ScreenerParams{} // all fields are nil

	for i, f := range RangeFilters {
		t.Run(f.Column, func(t *testing.T) {
			if f.GetMin == nil {
				t.Errorf("RangeFilter[%d] %q has nil GetMin", i, f.Column)
				return
			}
			if f.GetMax == nil {
				t.Errorf("RangeFilter[%d] %q has nil GetMax", i, f.Column)
				return
			}

			// With zero-value params, getters should return nil
			minVal := f.GetMin(params)
			if minVal != nil {
				t.Errorf("RangeFilter[%d] %q GetMin returned non-nil for zero params: %v", i, f.Column, *minVal)
			}
			maxVal := f.GetMax(params)
			// consecutive_dividend_years has GetMax returning nil always
			if f.Column != "consecutive_dividend_years" && maxVal != nil {
				t.Errorf("RangeFilter[%d] %q GetMax returned non-nil for zero params: %v", i, f.Column, *maxVal)
			}
		})
	}
}
