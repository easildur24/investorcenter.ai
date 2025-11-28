package database

import (
	"fmt"
	"strings"

	"investorcenter-api/models"
)

// ValidScreenerSortColumns defines valid columns for sorting in the screener
var ValidScreenerSortColumns = map[string]string{
	"symbol":         "t.symbol",
	"name":           "t.name",
	"market_cap":     "t.market_cap",
	"price":          "lp.price",
	"pe_ratio":       "lv.ttm_pe_ratio",
	"pb_ratio":       "lv.ttm_pb_ratio",
	"ps_ratio":       "lv.ttm_ps_ratio",
	"roe":            "lm.roe",
	"revenue_growth": "lm.revenue_growth_yoy",
	"dividend_yield": "lm.dividend_yield",
	"beta":           "lm.beta",
	"ic_score":       "lic.ic_score",
}

// GetScreenerStocks retrieves stocks for the screener with filtering and pagination
func GetScreenerStocks(params models.ScreenerParams) ([]models.ScreenerStock, int, error) {
	if DB == nil {
		return nil, 0, fmt.Errorf("database not connected")
	}

	// Build the base query with CTEs
	baseQuery := `
		WITH latest_prices AS (
			SELECT DISTINCT ON (ticker)
				ticker, close as price
			FROM stock_prices
			ORDER BY ticker, time DESC
		),
		latest_valuation AS (
			SELECT DISTINCT ON (ticker)
				ticker, ttm_pe_ratio, ttm_pb_ratio, ttm_ps_ratio
			FROM valuation_ratios
			ORDER BY ticker, calculation_date DESC
		),
		latest_metrics AS (
			SELECT DISTINCT ON (ticker)
				ticker, revenue_growth_yoy, dividend_yield, roe, beta
			FROM fundamental_metrics_extended
			ORDER BY ticker, calculation_date DESC
		),
		latest_ic_scores AS (
			SELECT DISTINCT ON (ticker)
				ticker, overall_score as ic_score
			FROM ic_scores
			ORDER BY ticker, date DESC
		)
	`

	// Build WHERE conditions
	conditions := []string{"t.asset_type = $1", "t.active = true"}
	args := []interface{}{params.AssetType}
	argIndex := 2

	// Sector filter
	if len(params.Sectors) > 0 {
		placeholders := make([]string, len(params.Sectors))
		for i, sector := range params.Sectors {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, sector)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("t.sector IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Market cap filters
	if params.MarketCapMin != nil {
		conditions = append(conditions, fmt.Sprintf("t.market_cap >= $%d", argIndex))
		args = append(args, *params.MarketCapMin)
		argIndex++
	}
	if params.MarketCapMax != nil {
		conditions = append(conditions, fmt.Sprintf("t.market_cap <= $%d", argIndex))
		args = append(args, *params.MarketCapMax)
		argIndex++
	}

	// P/E ratio filters
	if params.PEMin != nil {
		conditions = append(conditions, fmt.Sprintf("lv.ttm_pe_ratio >= $%d", argIndex))
		args = append(args, *params.PEMin)
		argIndex++
	}
	if params.PEMax != nil {
		conditions = append(conditions, fmt.Sprintf("lv.ttm_pe_ratio <= $%d", argIndex))
		args = append(args, *params.PEMax)
		argIndex++
	}

	// Dividend yield filters
	if params.DividendYieldMin != nil {
		conditions = append(conditions, fmt.Sprintf("lm.dividend_yield >= $%d", argIndex))
		args = append(args, *params.DividendYieldMin)
		argIndex++
	}
	if params.DividendYieldMax != nil {
		conditions = append(conditions, fmt.Sprintf("lm.dividend_yield <= $%d", argIndex))
		args = append(args, *params.DividendYieldMax)
		argIndex++
	}

	// Revenue growth filters
	if params.RevenueGrowthMin != nil {
		conditions = append(conditions, fmt.Sprintf("lm.revenue_growth_yoy >= $%d", argIndex))
		args = append(args, *params.RevenueGrowthMin)
		argIndex++
	}
	if params.RevenueGrowthMax != nil {
		conditions = append(conditions, fmt.Sprintf("lm.revenue_growth_yoy <= $%d", argIndex))
		args = append(args, *params.RevenueGrowthMax)
		argIndex++
	}

	// IC Score filters
	if params.ICScoreMin != nil {
		conditions = append(conditions, fmt.Sprintf("lic.ic_score >= $%d", argIndex))
		args = append(args, *params.ICScoreMin)
		argIndex++
	}
	if params.ICScoreMax != nil {
		conditions = append(conditions, fmt.Sprintf("lic.ic_score <= $%d", argIndex))
		args = append(args, *params.ICScoreMax)
		argIndex++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Determine sort column
	sortColumn, ok := ValidScreenerSortColumns[params.Sort]
	if !ok {
		sortColumn = "t.market_cap"
	}

	// Validate order
	order := "DESC"
	if strings.ToUpper(params.Order) == "ASC" {
		order = "ASC"
	}

	// Count query
	countQuery := fmt.Sprintf(`
		%s
		SELECT COUNT(*)
		FROM tickers t
		LEFT JOIN latest_prices lp ON t.symbol = lp.ticker
		LEFT JOIN latest_valuation lv ON t.symbol = lv.ticker
		LEFT JOIN latest_metrics lm ON t.symbol = lm.ticker
		LEFT JOIN latest_ic_scores lic ON t.symbol = lic.ticker
		WHERE %s
	`, baseQuery, whereClause)

	var total int
	err := DB.Get(&total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count screener stocks: %w", err)
	}

	// Calculate offset
	offset := (params.Page - 1) * params.Limit

	// Data query
	dataQuery := fmt.Sprintf(`
		%s
		SELECT
			t.symbol,
			t.name,
			COALESCE(t.sector, '') as sector,
			COALESCE(t.industry, '') as industry,
			t.market_cap,
			lp.price,
			lv.ttm_pe_ratio as pe_ratio,
			lv.ttm_pb_ratio as pb_ratio,
			lv.ttm_ps_ratio as ps_ratio,
			lm.roe,
			lm.revenue_growth_yoy as revenue_growth,
			lm.dividend_yield,
			lm.beta,
			lic.ic_score
		FROM tickers t
		LEFT JOIN latest_prices lp ON t.symbol = lp.ticker
		LEFT JOIN latest_valuation lv ON t.symbol = lv.ticker
		LEFT JOIN latest_metrics lm ON t.symbol = lm.ticker
		LEFT JOIN latest_ic_scores lic ON t.symbol = lic.ticker
		WHERE %s
		ORDER BY %s %s NULLS LAST
		LIMIT $%d OFFSET $%d
	`, baseQuery, whereClause, sortColumn, order, argIndex, argIndex+1)

	args = append(args, params.Limit, offset)

	// Execute query
	stocks := make([]models.ScreenerStock, 0)
	err = DB.Select(&stocks, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch screener stocks: %w", err)
	}

	return stocks, total, nil
}
