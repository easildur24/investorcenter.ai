package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"investorcenter-api/models"
)

// GetTickerIDBySymbol retrieves the ticker ID for a given symbol
func GetTickerIDBySymbol(symbol string) (int, error) {
	var tickerID int
	query := `
		SELECT id FROM tickers
		WHERE UPPER(symbol) = UPPER($1)
		AND asset_type IN ('CS', 'stock', 'PFD', 'ETF')
		LIMIT 1
	`
	err := DB.Get(&tickerID, query, symbol)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("ticker not found: %s", symbol)
		}
		return 0, fmt.Errorf("failed to get ticker ID: %w", err)
	}
	return tickerID, nil
}

// UpsertFinancialStatement inserts or updates a financial statement
func UpsertFinancialStatement(stmt *models.FinancialStatement) error {
	// Marshal the data to JSON
	dataJSON, err := json.Marshal(stmt.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal financial data: %w", err)
	}

	query := `
		INSERT INTO financial_statements (
			ticker_id, cik, statement_type, timeframe, fiscal_year, fiscal_quarter,
			period_start, period_end, filed_date, source_filing_url, source_filing_type, data
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (ticker_id, statement_type, timeframe, fiscal_year, fiscal_quarter)
		DO UPDATE SET
			cik = EXCLUDED.cik,
			period_start = EXCLUDED.period_start,
			period_end = EXCLUDED.period_end,
			filed_date = EXCLUDED.filed_date,
			source_filing_url = EXCLUDED.source_filing_url,
			source_filing_type = EXCLUDED.source_filing_type,
			data = EXCLUDED.data,
			updated_at = NOW()
		RETURNING id
	`

	var id int
	err = DB.QueryRow(
		query,
		stmt.TickerID,
		stmt.CIK,
		stmt.StatementType,
		stmt.Timeframe,
		stmt.FiscalYear,
		stmt.FiscalQuarter,
		stmt.PeriodStart,
		stmt.PeriodEnd,
		stmt.FiledDate,
		stmt.SourceFilingURL,
		stmt.SourceFilingType,
		dataJSON,
	).Scan(&id)

	if err != nil {
		return fmt.Errorf("failed to upsert financial statement: %w", err)
	}

	stmt.ID = id
	return nil
}

// GetFinancialStatements retrieves financial statements for a ticker
func GetFinancialStatements(params models.FinancialsParams) ([]models.FinancialStatement, error) {
	// Get ticker ID
	tickerID, err := GetTickerIDBySymbol(params.Ticker)
	if err != nil {
		return nil, err
	}

	// Build query
	query := `
		SELECT
			id, ticker_id, cik, statement_type, timeframe, fiscal_year, fiscal_quarter,
			period_start, period_end, filed_date, source_filing_url, source_filing_type,
			data, created_at, updated_at
		FROM financial_statements
		WHERE ticker_id = $1
	`

	args := []interface{}{tickerID}
	argIndex := 2

	// Add statement type filter (extracted from params if needed)
	// This is handled at the handler level based on the endpoint

	// Add timeframe filter
	if params.Timeframe != "" {
		query += fmt.Sprintf(" AND timeframe = $%d", argIndex)
		args = append(args, params.Timeframe)
		argIndex++
	}

	// Add fiscal year filter
	if params.FiscalYear != nil {
		query += fmt.Sprintf(" AND fiscal_year = $%d", argIndex)
		args = append(args, *params.FiscalYear)
		argIndex++
	}

	// Add sorting
	if params.Sort == "asc" {
		query += " ORDER BY period_end ASC, fiscal_quarter ASC NULLS LAST"
	} else {
		query += " ORDER BY period_end DESC, fiscal_quarter DESC NULLS LAST"
	}

	// Add limit
	if params.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, params.Limit)
	} else {
		query += " LIMIT 8" // Default limit
	}

	var statements []models.FinancialStatement
	err = DB.Select(&statements, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get financial statements: %w", err)
	}

	return statements, nil
}

// GetFinancialStatementsByType retrieves financial statements of a specific type
func GetFinancialStatementsByType(ticker string, statementType models.StatementType, timeframe models.Timeframe, limit int) ([]models.FinancialStatement, error) {
	// Get ticker ID
	tickerID, err := GetTickerIDBySymbol(ticker)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, ticker_id, cik, statement_type, timeframe, fiscal_year, fiscal_quarter,
			period_start, period_end, filed_date, source_filing_url, source_filing_type,
			data, created_at, updated_at
		FROM financial_statements
		WHERE ticker_id = $1 AND statement_type = $2 AND timeframe = $3
		ORDER BY period_end DESC, fiscal_quarter DESC NULLS LAST
		LIMIT $4
	`

	var statements []models.FinancialStatement
	err = DB.Select(&statements, query, tickerID, statementType, timeframe, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get financial statements: %w", err)
	}

	return statements, nil
}

// GetLatestFinancialStatement retrieves the most recent financial statement
func GetLatestFinancialStatement(ticker string, statementType models.StatementType, timeframe models.Timeframe) (*models.FinancialStatement, error) {
	// Get ticker ID
	tickerID, err := GetTickerIDBySymbol(ticker)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, ticker_id, cik, statement_type, timeframe, fiscal_year, fiscal_quarter,
			period_start, period_end, filed_date, source_filing_url, source_filing_type,
			data, created_at, updated_at
		FROM financial_statements
		WHERE ticker_id = $1 AND statement_type = $2 AND timeframe = $3
		ORDER BY period_end DESC
		LIMIT 1
	`

	var stmt models.FinancialStatement
	err = DB.Get(&stmt, query, tickerID, statementType, timeframe)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest financial statement: %w", err)
	}

	return &stmt, nil
}

// GetYoYComparisonPeriod gets the statement from the same period one year ago
func GetYoYComparisonPeriod(tickerID int, statementType models.StatementType, timeframe models.Timeframe, currentFiscalYear int, currentQuarter *int) (*models.FinancialStatement, error) {
	previousYear := currentFiscalYear - 1

	query := `
		SELECT
			id, ticker_id, cik, statement_type, timeframe, fiscal_year, fiscal_quarter,
			period_start, period_end, filed_date, source_filing_url, source_filing_type,
			data, created_at, updated_at
		FROM financial_statements
		WHERE ticker_id = $1 AND statement_type = $2 AND timeframe = $3 AND fiscal_year = $4
	`

	args := []interface{}{tickerID, statementType, timeframe, previousYear}

	if currentQuarter != nil {
		query += " AND fiscal_quarter = $5"
		args = append(args, *currentQuarter)
	} else {
		query += " AND fiscal_quarter IS NULL"
	}

	query += " LIMIT 1"

	var stmt models.FinancialStatement
	err := DB.Get(&stmt, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get YoY comparison period: %w", err)
	}

	return &stmt, nil
}

// GetFinancialStatementsWithYoY retrieves financial statements with YoY comparison data
func GetFinancialStatementsWithYoY(ticker string, statementType models.StatementType, timeframe models.Timeframe, limit int) ([]models.FinancialPeriod, error) {
	statements, err := GetFinancialStatementsByType(ticker, statementType, timeframe, limit)
	if err != nil {
		return nil, err
	}

	if len(statements) == 0 {
		return nil, nil
	}

	periods := make([]models.FinancialPeriod, len(statements))

	for i, stmt := range statements {
		// Convert data to map[string]interface{}
		data := make(map[string]interface{})
		for k, v := range stmt.Data {
			data[k] = v
		}

		// Format dates
		periodEnd := stmt.PeriodEnd.Format("2006-01-02")
		var filedDate *string
		if stmt.FiledDate != nil {
			fd := stmt.FiledDate.Format("2006-01-02")
			filedDate = &fd
		}

		period := models.FinancialPeriod{
			FiscalYear:    stmt.FiscalYear,
			FiscalQuarter: stmt.FiscalQuarter,
			PeriodEnd:     periodEnd,
			FiledDate:     filedDate,
			Data:          data,
		}

		// Get YoY comparison data
		yoyStmt, err := GetYoYComparisonPeriod(stmt.TickerID, statementType, timeframe, stmt.FiscalYear, stmt.FiscalQuarter)
		if err == nil && yoyStmt != nil {
			yoyChanges := calculateYoYChanges(stmt.Data, yoyStmt.Data)
			period.YoYChange = yoyChanges
		}

		periods[i] = period
	}

	return periods, nil
}

// calculateYoYChanges calculates year-over-year percentage changes
func calculateYoYChanges(current, previous models.FinancialData) map[string]*float64 {
	changes := make(map[string]*float64)

	for key, currentVal := range current {
		// Skip metadata fields
		if strings.HasSuffix(key, "_label") || strings.HasSuffix(key, "_unit") {
			continue
		}

		currentFloat, ok := toFloat64(currentVal)
		if !ok {
			continue
		}

		previousVal, exists := previous[key]
		if !exists {
			continue
		}

		previousFloat, ok := toFloat64(previousVal)
		if !ok || previousFloat == 0 {
			continue
		}

		change := (currentFloat - previousFloat) / abs(previousFloat)
		changes[key] = &change
	}

	return changes
}

// toFloat64 converts an interface to float64
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	default:
		return 0, false
	}
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// companyMetadataRow is a helper struct for scanning partial company metadata
type companyMetadataRow struct {
	Name string `db:"name"`
	CIK  string `db:"cik"`
}

// GetCompanyMetadata retrieves company metadata for a ticker
func GetCompanyMetadata(ticker string) (*models.FinancialsMetadata, error) {
	var row companyMetadataRow
	query := `
		SELECT name, cik
		FROM tickers
		WHERE UPPER(symbol) = UPPER($1)
		AND asset_type = 'stock'
		LIMIT 1
	`

	err := DB.Get(&row, query, ticker)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("ticker not found: %s", ticker)
		}
		return nil, fmt.Errorf("failed to get company metadata: %w", err)
	}

	var cik *string
	if row.CIK != "" {
		cik = &row.CIK
	}

	return &models.FinancialsMetadata{
		CompanyName: row.Name,
		CIK:         cik,
	}, nil
}

// HasFinancialStatements checks if a ticker has any financial statements
func HasFinancialStatements(ticker string) (bool, error) {
	tickerID, err := GetTickerIDBySymbol(ticker)
	if err != nil {
		return false, nil // Ticker doesn't exist
	}

	var count int
	query := `SELECT COUNT(*) FROM financial_statements WHERE ticker_id = $1 LIMIT 1`
	err = DB.Get(&count, query, tickerID)
	if err != nil {
		return false, fmt.Errorf("failed to check financial statements: %w", err)
	}

	return count > 0, nil
}

// GetOldestFinancialStatementDate returns the date of the oldest financial statement for a ticker
func GetOldestFinancialStatementDate(ticker string) (*time.Time, error) {
	tickerID, err := GetTickerIDBySymbol(ticker)
	if err != nil {
		return nil, err
	}

	var oldestDate time.Time
	query := `SELECT MIN(period_end) FROM financial_statements WHERE ticker_id = $1`
	err = DB.Get(&oldestDate, query, tickerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get oldest statement date: %w", err)
	}

	return &oldestDate, nil
}

// DeleteFinancialStatements deletes all financial statements for a ticker
func DeleteFinancialStatements(ticker string) error {
	tickerID, err := GetTickerIDBySymbol(ticker)
	if err != nil {
		return err
	}

	query := `DELETE FROM financial_statements WHERE ticker_id = $1`
	_, err = DB.Exec(query, tickerID)
	if err != nil {
		return fmt.Errorf("failed to delete financial statements: %w", err)
	}

	return nil
}

// GetFinancialStatementsCount returns the count of financial statements for a ticker
func GetFinancialStatementsCount(ticker string, statementType models.StatementType) (int, error) {
	tickerID, err := GetTickerIDBySymbol(ticker)
	if err != nil {
		return 0, err
	}

	var count int
	query := `SELECT COUNT(*) FROM financial_statements WHERE ticker_id = $1 AND statement_type = $2`
	err = DB.Get(&count, query, tickerID, statementType)
	if err != nil {
		return 0, fmt.Errorf("failed to count financial statements: %w", err)
	}

	return count, nil
}

// ICScoreRatioRecord represents a single period's ratio data from IC Score tables
type ICScoreRatioRecord struct {
	Ticker          string   `db:"ticker"`
	CalculationDate string   `db:"calculation_date"`
	StockPrice      *float64 `db:"stock_price"`
	// Valuation ratios
	TTMPERatio   *float64 `db:"ttm_pe_ratio"`
	TTMPBRatio   *float64 `db:"ttm_pb_ratio"`
	TTMPSRatio   *float64 `db:"ttm_ps_ratio"`
	TTMMarketCap *int64   `db:"ttm_market_cap"`
	// Profitability
	GrossMargin     *float64 `db:"gross_margin"`
	OperatingMargin *float64 `db:"operating_margin"`
	NetMargin       *float64 `db:"net_margin"`
	EBITDAMargin    *float64 `db:"ebitda_margin"`
	// Returns
	ROE  *float64 `db:"roe"`
	ROA  *float64 `db:"roa"`
	ROIC *float64 `db:"roic"`
	// Liquidity
	CurrentRatio *float64 `db:"current_ratio"`
	QuickRatio   *float64 `db:"quick_ratio"`
	// Leverage
	DebtToEquity     *float64 `db:"debt_to_equity"`
	DebtToAssets     *float64 `db:"debt_to_assets"`
	InterestCoverage *float64 `db:"interest_coverage"`
	// Valuation extended
	EnterpriseValue *float64 `db:"enterprise_value"`
	EVToRevenue     *float64 `db:"ev_to_revenue"`
	EVToEBITDA      *float64 `db:"ev_to_ebitda"`
	// Growth
	RevenueGrowthYoY *float64 `db:"revenue_growth_yoy"`
	EPSGrowthYoY     *float64 `db:"eps_growth_yoy"`
	// Dividends
	DividendYield *float64 `db:"dividend_yield"`
	PayoutRatio   *float64 `db:"payout_ratio"`
}

// GetICScoreRatios retrieves financial ratios from IC Score tables (valuation_ratios + fundamental_metrics_extended)
func GetICScoreRatios(ticker string, limit int) ([]ICScoreRatioRecord, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not connected")
	}

	if limit <= 0 {
		limit = 8
	}

	// Join valuation_ratios and fundamental_metrics_extended on ticker and calculation_date
	query := `
		SELECT
			v.ticker,
			v.calculation_date::text as calculation_date,
			v.stock_price,
			v.ttm_pe_ratio,
			v.ttm_pb_ratio,
			v.ttm_ps_ratio,
			v.ttm_market_cap,
			m.gross_margin,
			m.operating_margin,
			m.net_margin,
			m.ebitda_margin,
			m.roe,
			m.roa,
			m.roic,
			m.current_ratio,
			m.quick_ratio,
			m.debt_to_equity,
			CASE WHEN m.debt_to_equity IS NOT NULL THEN m.debt_to_equity / (1 + m.debt_to_equity) ELSE NULL END as debt_to_assets,
			m.interest_coverage,
			m.enterprise_value,
			m.ev_to_revenue,
			m.ev_to_ebitda,
			m.revenue_growth_yoy,
			m.eps_growth_yoy,
			m.dividend_yield,
			m.payout_ratio
		FROM valuation_ratios v
		LEFT JOIN fundamental_metrics_extended m
			ON v.ticker = m.ticker AND v.calculation_date = m.calculation_date
		WHERE UPPER(v.ticker) = UPPER($1)
		ORDER BY v.calculation_date DESC
		LIMIT $2
	`

	var records []ICScoreRatioRecord
	err := DB.Select(&records, query, ticker, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get IC Score ratios: %w", err)
	}

	return records, nil
}

// ConvertICScoreRatiosToFinancialPeriods converts IC Score ratio records to the API response format
func ConvertICScoreRatiosToFinancialPeriods(records []ICScoreRatioRecord) []models.FinancialPeriod {
	periods := make([]models.FinancialPeriod, len(records))

	for i, rec := range records {
		data := make(map[string]interface{})

		// Valuation ratios
		if rec.TTMPERatio != nil {
			data["price_to_earnings"] = *rec.TTMPERatio
		}
		if rec.TTMPBRatio != nil {
			data["price_to_book"] = *rec.TTMPBRatio
		}
		if rec.TTMPSRatio != nil {
			data["price_to_sales"] = *rec.TTMPSRatio
		}
		if rec.EVToEBITDA != nil {
			data["enterprise_value_to_ebitda"] = *rec.EVToEBITDA
		}

		// Profitability margins (stored as decimals, convert to percentage)
		if rec.GrossMargin != nil {
			data["gross_margin"] = *rec.GrossMargin * 100
		}
		if rec.OperatingMargin != nil {
			data["operating_margin"] = *rec.OperatingMargin * 100
		}
		if rec.NetMargin != nil {
			data["net_profit_margin"] = *rec.NetMargin * 100
		}

		// Returns (stored as decimals, convert to percentage)
		if rec.ROE != nil {
			data["return_on_equity"] = *rec.ROE * 100
		}
		if rec.ROA != nil {
			data["return_on_assets"] = *rec.ROA * 100
		}
		if rec.ROIC != nil {
			data["return_on_invested_capital"] = *rec.ROIC * 100
		}

		// Liquidity ratios (already as ratios)
		if rec.CurrentRatio != nil {
			data["current_ratio"] = *rec.CurrentRatio
		}
		if rec.QuickRatio != nil {
			data["quick_ratio"] = *rec.QuickRatio
		}

		// Leverage ratios
		if rec.DebtToEquity != nil {
			data["debt_to_equity"] = *rec.DebtToEquity
		}
		if rec.DebtToAssets != nil {
			data["debt_to_assets"] = *rec.DebtToAssets * 100 // Convert to percentage
		}
		if rec.InterestCoverage != nil {
			data["interest_coverage"] = *rec.InterestCoverage
		}

		// Parse the calculation_date to get fiscal year
		// Format is YYYY-MM-DD
		fiscalYear := 0
		if len(rec.CalculationDate) >= 4 {
			_, _ = fmt.Sscanf(rec.CalculationDate, "%d", &fiscalYear)
		}

		periods[i] = models.FinancialPeriod{
			FiscalYear:    fiscalYear,
			FiscalQuarter: nil, // Ratios are typically not quarterly
			PeriodEnd:     rec.CalculationDate,
			FiledDate:     nil,
			Data:          data,
			YoYChange:     nil, // Could calculate if we have previous year data
		}
	}

	return periods
}
