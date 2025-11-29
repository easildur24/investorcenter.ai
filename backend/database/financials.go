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
		AND asset_type = 'stock'
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

// GetCompanyMetadata retrieves company metadata for a ticker
func GetCompanyMetadata(ticker string) (*models.FinancialsMetadata, error) {
	var stock models.Stock
	query := `
		SELECT name, cik
		FROM tickers
		WHERE UPPER(symbol) = UPPER($1)
		AND asset_type = 'stock'
		LIMIT 1
	`

	err := DB.Get(&stock, query, ticker)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("ticker not found: %s", ticker)
		}
		return nil, fmt.Errorf("failed to get company metadata: %w", err)
	}

	var cik *string
	if stock.CIK != "" {
		cik = &stock.CIK
	}

	return &models.FinancialsMetadata{
		CompanyName: stock.Name,
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
