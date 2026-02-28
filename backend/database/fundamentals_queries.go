package database

import (
	"database/sql"
	"fmt"

	"investorcenter-api/models"
)

// GetStockMetricsMap returns all available metrics for a stock from fundamental_metrics_extended
// and valuation_ratios as a map keyed by metric name. Returns nil if no data exists.
func GetStockMetricsMap(ticker string) (map[string]*float64, *models.StockMetricsRow, error) {
	if DB == nil {
		return nil, nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT
			m.gross_margin::float8 as gross_margin,
			m.operating_margin::float8 as operating_margin,
			m.net_margin::float8 as net_margin,
			m.ebitda_margin::float8 as ebitda_margin,
			m.roe::float8 as roe,
			m.roa::float8 as roa,
			m.roic::float8 as roic,
			m.revenue_growth_yoy::float8 as revenue_growth_yoy,
			m.eps_growth_yoy::float8 as eps_growth_yoy,
			m.fcf_growth_yoy::float8 as fcf_growth_yoy,
			m.ev_to_revenue::float8 as ev_to_revenue,
			m.ev_to_ebitda::float8 as ev_to_ebitda,
			m.ev_to_fcf::float8 as ev_to_fcf,
			m.current_ratio::float8 as current_ratio,
			m.quick_ratio::float8 as quick_ratio,
			m.debt_to_equity::float8 as debt_to_equity,
			m.interest_coverage::float8 as interest_coverage,
			m.net_debt_to_ebitda::float8 as net_debt_to_ebitda,
			m.dividend_yield::float8 as dividend_yield,
			m.payout_ratio::float8 as payout_ratio,
			v.ttm_pe_ratio::float8 as pe_ratio,
			v.ttm_pb_ratio::float8 as pb_ratio,
			v.ttm_ps_ratio::float8 as ps_ratio,
			v.stock_price::float8 as stock_price
		FROM fundamental_metrics_extended m
		LEFT JOIN valuation_ratios v ON m.ticker = v.ticker AND m.calculation_date = v.calculation_date
		WHERE UPPER(m.ticker) = UPPER($1)
		ORDER BY m.calculation_date DESC
		LIMIT 1
	`

	var row models.StockMetricsRow
	err := DB.Get(&row, query, ticker)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("failed to get stock metrics: %w", err)
	}

	return row.ToMap(), &row, nil
}

// GetEnrichedIndustryPeers returns peers from the same industry with enriched metrics.
// Peers are filtered by market cap proximity (0.25x to 4x) and sorted by market cap closeness.
func GetEnrichedIndustryPeers(industry string, marketCap float64, excludeTicker string, limit int) ([]models.EnrichedPeer, error) {
	return getEnrichedPeers("industry", industry, marketCap, excludeTicker, limit)
}

// GetEnrichedSectorPeers is a fallback when industry-level peers are insufficient.
// Uses sector-level matching with the same enrichment.
func GetEnrichedSectorPeers(sector string, marketCap float64, excludeTicker string, limit int) ([]models.EnrichedPeer, error) {
	return getEnrichedPeers("sector", sector, marketCap, excludeTicker, limit)
}

// getEnrichedPeers is a shared helper that fetches peers filtered by the given column
// (industry or sector) with enriched IC Score, valuation, and fundamental metrics via LATERAL JOINs.
func getEnrichedPeers(filterColumn, filterValue string, marketCap float64, excludeTicker string, limit int) ([]models.EnrichedPeer, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Allowlist filter column to prevent SQL injection (column names cannot be parameterized)
	switch filterColumn {
	case "industry", "sector":
	default:
		return nil, fmt.Errorf("invalid peer filter column: %s", filterColumn)
	}

	query := fmt.Sprintf(`
		SELECT
			p.symbol, p.name, p.industry, p.market_cap,
			i.ic_score,
			v.pe_ratio,
			m.roe, m.revenue_growth_yoy, m.net_margin, m.debt_to_equity
		FROM (
			SELECT symbol, name, COALESCE(industry, '') as industry, market_cap::float8 as market_cap
			FROM tickers
			WHERE %s = $1
				AND UPPER(symbol) != UPPER($2)
				AND market_cap IS NOT NULL
				AND market_cap BETWEEN $3 * 0.25 AND $3 * 4.0
				AND asset_type = 'stock'
			ORDER BY ABS(market_cap - $3) ASC
			LIMIT $4
		) p
		LEFT JOIN LATERAL (
			SELECT overall_score::float8 as ic_score
			FROM ic_scores WHERE ticker = p.symbol
			ORDER BY date DESC LIMIT 1
		) i ON true
		LEFT JOIN LATERAL (
			SELECT ttm_pe_ratio::float8 as pe_ratio
			FROM valuation_ratios WHERE ticker = p.symbol
			ORDER BY calculation_date DESC LIMIT 1
		) v ON true
		LEFT JOIN LATERAL (
			SELECT roe::float8, revenue_growth_yoy::float8, net_margin::float8, debt_to_equity::float8
			FROM fundamental_metrics_extended WHERE ticker = p.symbol
			ORDER BY calculation_date DESC LIMIT 1
		) m ON true
	`, filterColumn)

	var peers []models.EnrichedPeer
	err := DB.Select(&peers, query, filterValue, excludeTicker, marketCap, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s peers: %w", filterColumn, err)
	}

	return peers, nil
}

// GetFairValueMetrics retrieves fair value estimates from fundamental_metrics_extended
// along with the stock price from valuation_ratios.
func GetFairValueMetrics(ticker string) (*models.FairValueMetrics, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT
			m.ticker,
			m.dcf_fair_value::float8 as dcf_fair_value,
			m.epv_fair_value::float8 as epv_fair_value,
			m.graham_number::float8 as graham_number,
			m.dcf_upside_percent::float8 as dcf_upside_percent,
			m.wacc::float8 as wacc,
			v.stock_price::float8 as stock_price
		FROM fundamental_metrics_extended m
		LEFT JOIN valuation_ratios v
			ON m.ticker = v.ticker AND m.calculation_date = v.calculation_date
		WHERE UPPER(m.ticker) = UPPER($1)
		ORDER BY m.calculation_date DESC
		LIMIT 1
	`

	var fv models.FairValueMetrics
	err := DB.Get(&fv, query, ticker)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get fair value metrics: %w", err)
	}

	return &fv, nil
}

// GetLatestICScore retrieves the most recent IC Score for a ticker.
func GetLatestICScore(ticker string) (*models.ICScore, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT
			id, ticker, date, overall_score,
			value_score, growth_score, profitability_score,
			financial_health_score, momentum_score,
			analyst_consensus_score, insider_activity_score,
			institutional_score, news_sentiment_score, technical_score,
			earnings_revisions_score, historical_value_score, dividend_quality_score,
			rating, sector_percentile, confidence_level, data_completeness,
			lifecycle_stage, raw_score, smoothing_applied, weights_used,
			sector_rank, sector_total,
			calculation_metadata, created_at
		FROM ic_scores
		WHERE UPPER(ticker) = UPPER($1)
		ORDER BY date DESC
		LIMIT 1
	`

	var ic models.ICScore
	err := DB.Get(&ic, query, ticker)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest IC score: %w", err)
	}

	return &ic, nil
}

// GetMetricHistory retrieves historical values for a specific metric from financial_statements.
// IMPORTANT: fieldName is used in JSONB access (fs.data->>$3). While it's parameterized and safe
// from SQL injection, callers MUST validate fieldName against models.MetricStatementMap before
// calling this function to prevent arbitrary JSONB field access.
func GetMetricHistory(ticker, statementType, fieldName, timeframe string, limit int) ([]models.MetricHistoryRow, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Defense-in-depth: validate fieldName against known metric mappings
	validField := false
	for _, m := range models.MetricStatementMap {
		if m.FieldName == fieldName {
			validField = true
			break
		}
	}
	if !validField {
		return nil, fmt.Errorf("unknown field name: %s", fieldName)
	}

	query := `
		SELECT
			fs.period_end::text as period_end,
			fs.fiscal_year,
			fs.fiscal_quarter,
			(fs.data->>$3)::float8 as value
		FROM financial_statements fs
		JOIN tickers t ON fs.ticker_id = t.id
		WHERE UPPER(t.symbol) = UPPER($1)
			AND fs.statement_type = $2
			AND fs.timeframe = $4
			AND fs.data ? $3
		ORDER BY fs.period_end DESC
		LIMIT $5
	`

	var rows []models.MetricHistoryRow
	err := DB.Select(&rows, query, ticker, statementType, fieldName, timeframe, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get metric history: %w", err)
	}

	return rows, nil
}
