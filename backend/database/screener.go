package database

import (
	"fmt"
	"strings"

	"investorcenter-api/models"
)

// ValidScreenerSortColumns defines valid columns for sorting in the screener
// Uses screener_data materialized view for fast queries
var ValidScreenerSortColumns = map[string]string{
	"symbol":         "symbol",
	"name":           "name",
	"market_cap":     "market_cap",
	"price":          "current_price",
	"pe_ratio":       "pe_ratio",
	"pb_ratio":       "price_to_book",
	"ps_ratio":       "price_to_sales",
	"roe":            "roe",
	"revenue_growth": "revenue_growth",
	"dividend_yield": "dividend_yield",
	"beta":           "roe",        // placeholder - beta not in view
	"ic_score":       "market_cap", // placeholder - ic_score not in view
}

// GetScreenerStocks retrieves stocks for the screener with filtering and pagination
// Uses screener_data materialized view for fast queries (<100ms vs 2-3 minutes)
func GetScreenerStocks(params models.ScreenerParams) ([]models.ScreenerStock, int, error) {
	if DB == nil {
		return nil, 0, fmt.Errorf("database not connected")
	}

	// Build WHERE conditions using screener_data materialized view
	conditions := []string{}
	args := []interface{}{}
	argIndex := 1

	// Sector filter
	if len(params.Sectors) > 0 {
		placeholders := make([]string, len(params.Sectors))
		for i, sector := range params.Sectors {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, sector)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("sector IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Market cap filters
	if params.MarketCapMin != nil {
		conditions = append(conditions, fmt.Sprintf("market_cap >= $%d", argIndex))
		args = append(args, *params.MarketCapMin)
		argIndex++
	}
	if params.MarketCapMax != nil {
		conditions = append(conditions, fmt.Sprintf("market_cap <= $%d", argIndex))
		args = append(args, *params.MarketCapMax)
		argIndex++
	}

	// P/E ratio filters
	if params.PEMin != nil {
		conditions = append(conditions, fmt.Sprintf("pe_ratio >= $%d", argIndex))
		args = append(args, *params.PEMin)
		argIndex++
	}
	if params.PEMax != nil {
		conditions = append(conditions, fmt.Sprintf("pe_ratio <= $%d", argIndex))
		args = append(args, *params.PEMax)
		argIndex++
	}

	// Dividend yield filters
	if params.DividendYieldMin != nil {
		conditions = append(conditions, fmt.Sprintf("dividend_yield >= $%d", argIndex))
		args = append(args, *params.DividendYieldMin)
		argIndex++
	}
	if params.DividendYieldMax != nil {
		conditions = append(conditions, fmt.Sprintf("dividend_yield <= $%d", argIndex))
		args = append(args, *params.DividendYieldMax)
		argIndex++
	}

	// Revenue growth filters
	if params.RevenueGrowthMin != nil {
		conditions = append(conditions, fmt.Sprintf("revenue_growth >= $%d", argIndex))
		args = append(args, *params.RevenueGrowthMin)
		argIndex++
	}
	if params.RevenueGrowthMax != nil {
		conditions = append(conditions, fmt.Sprintf("revenue_growth <= $%d", argIndex))
		args = append(args, *params.RevenueGrowthMax)
		argIndex++
	}

	// IC Score filters
	if params.ICScoreMin != nil {
		conditions = append(conditions, fmt.Sprintf("ic_score >= $%d", argIndex))
		args = append(args, *params.ICScoreMin)
		argIndex++
	}
	if params.ICScoreMax != nil {
		conditions = append(conditions, fmt.Sprintf("ic_score <= $%d", argIndex))
		args = append(args, *params.ICScoreMax)
		argIndex++
	}

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

	// Data query - simple single table scan with pagination
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
			current_price as price,
			change_percent,
			pe_ratio,
			price_to_book as pb_ratio,
			price_to_sales as ps_ratio,
			roe,
			revenue_growth,
			dividend_yield,
			0.0 as beta,
			0.0 as ic_score
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
