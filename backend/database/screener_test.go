package database

import (
	"testing"
)

// TestValidScreenerSortColumns verifies that all sort column mappings
// reference actual columns in the screener_data materialized view.
// This prevents regressions like mapping ic_score -> market_cap
// or beta -> roe, which caused those columns to display as 0.
func TestValidScreenerSortColumns(t *testing.T) {
	// These are the actual column names in the screener_data materialized view
	// (defined in ic-score-service/migrations/016_create_screener_materialized_view.sql)
	validViewColumns := map[string]bool{
		"symbol":         true,
		"name":           true,
		"sector":         true,
		"industry":       true,
		"market_cap":     true,
		"asset_type":     true,
		"active":         true,
		"price":          true,
		"pe_ratio":       true,
		"pb_ratio":       true,
		"ps_ratio":       true,
		"revenue_growth": true,
		"dividend_yield": true,
		"roe":            true,
		"beta":           true,
		"ic_score":       true,
		"refreshed_at":   true,
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
		{"ic_score", "ic_score"},
		{"beta", "beta"},
		{"market_cap", "market_cap"},
		{"pe_ratio", "pe_ratio"},
		{"roe", "roe"},
		{"dividend_yield", "dividend_yield"},
		{"revenue_growth", "revenue_growth"},
		{"price", "price"},
		{"pb_ratio", "pb_ratio"},
		{"ps_ratio", "ps_ratio"},
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
