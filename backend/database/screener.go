package database

import (
	"fmt"
	"strings"

	"investorcenter-api/models"
)

// ValidScreenerSortColumns defines valid columns for sorting in the screener.
// Uses screener_data materialized view for fast queries.
// Every key must map to a real column in screener_data (see migration 019).
var ValidScreenerSortColumns = map[string]string{
	// Core identity
	"symbol":   "symbol",
	"name":     "name",
	"sector":   "sector",
	"industry": "industry",

	// Market data
	"market_cap": "market_cap",
	"price":      "price",

	// Valuation
	"pe_ratio": "pe_ratio",
	"pb_ratio": "pb_ratio",
	"ps_ratio": "ps_ratio",

	// Profitability
	"roe":              "roe",
	"roa":              "roa",
	"gross_margin":     "gross_margin",
	"operating_margin": "operating_margin",
	"net_margin":       "net_margin",

	// Financial health
	"debt_to_equity": "debt_to_equity",
	"current_ratio":  "current_ratio",

	// Growth
	"revenue_growth": "revenue_growth",
	"eps_growth_yoy": "eps_growth_yoy",

	// Dividends
	"dividend_yield":             "dividend_yield",
	"payout_ratio":               "payout_ratio",
	"consecutive_dividend_years": "consecutive_dividend_years",

	// Risk
	"beta": "beta",

	// Fair value
	"dcf_upside_percent": "dcf_upside_percent",

	// IC Score
	"ic_score": "ic_score",

	// IC Score sub-factors
	"value_score":             "value_score",
	"growth_score":            "growth_score",
	"profitability_score":     "profitability_score",
	"financial_health_score":  "financial_health_score",
	"momentum_score":          "momentum_score",
	"analyst_consensus_score": "analyst_consensus_score",
	"insider_activity_score":  "insider_activity_score",
	"institutional_score":     "institutional_score",
	"news_sentiment_score":    "news_sentiment_score",
	"technical_score":         "technical_score",
}

// GetScreenerStocks retrieves stocks for the screener with filtering and pagination.
// Uses screener_data materialized view for fast queries (<100ms vs 2-3 minutes).
func GetScreenerStocks(params models.ScreenerParams) ([]models.ScreenerStock, int, error) {
	if DB == nil {
		return nil, 0, fmt.Errorf("database not connected")
	}

	// Build WHERE conditions using the filter registry
	conditions, args, argIndex := BuildFilterConditions(&params, 1)

	// Build WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Determine sort column (allowlist-validated, safe for interpolation)
	sortColumn, ok := ValidScreenerSortColumns[params.Sort]
	if !ok {
		sortColumn = "market_cap"
	}
	// Defense-in-depth: reject any sort column not matching [a-z_]+
	for _, ch := range sortColumn {
		if !((ch >= 'a' && ch <= 'z') || ch == '_') {
			sortColumn = "market_cap"
			break
		}
	}

	// Validate order (only ASC or DESC, safe for interpolation)
	order := "DESC"
	if strings.ToUpper(params.Order) == "ASC" {
		order = "ASC"
	}

	// Count query - simple single table scan
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM screener_data %s", whereClause)

	var total int
	err := DB.Get(&total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count screener stocks: %w", err)
	}

	// Calculate offset
	offset := (params.Page - 1) * params.Limit

	// Data query - reads all columns from the expanded materialized view.
	// Note: ORDER BY column and direction cannot be parameterized in PostgreSQL.
	// Both values are validated above via allowlist (sortColumn) and strict
	// string comparison (order), making this safe from SQL injection.
	dataQuery := fmt.Sprintf(`
		SELECT
			symbol,
			name,
			sector,
			industry,
			market_cap,
			price,
			pe_ratio,
			pb_ratio,
			ps_ratio,
			roe,
			roa,
			gross_margin,
			operating_margin,
			net_margin,
			debt_to_equity,
			current_ratio,
			revenue_growth,
			eps_growth_yoy,
			dividend_yield,
			payout_ratio,
			consecutive_dividend_years,
			beta,
			dcf_upside_percent,
			ic_score,
			ic_rating,
			value_score,
			growth_score,
			profitability_score,
			financial_health_score,
			momentum_score,
			analyst_consensus_score,
			insider_activity_score,
			institutional_score,
			news_sentiment_score,
			technical_score,
			ic_sector_percentile,
			lifecycle_stage
		FROM screener_data
		%s
		ORDER BY "%s" %s NULLS LAST
		LIMIT $%d OFFSET $%d
	`, whereClause, sortColumn, order, argIndex, argIndex+1)

	args = append(args, params.Limit, offset)

	// Execute query
	stocks := make([]models.ScreenerStock, 0)
	err = DB.Select(&stocks, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch screener stocks: %w", err)
	}

	return stocks, total, nil
}
