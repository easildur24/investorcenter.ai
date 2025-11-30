package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"investorcenter-api/database"
	"investorcenter-api/models"
)

// FinancialsService handles financial statement operations
type FinancialsService struct {
	polygonClient *PolygonFinancialsClient
}

// NewFinancialsService creates a new financials service
func NewFinancialsService() *FinancialsService {
	return &FinancialsService{
		polygonClient: NewPolygonFinancialsClient(),
	}
}

// IngestFinancials fetches and stores all financial statements for a ticker
func (s *FinancialsService) IngestFinancials(ctx context.Context, ticker string) error {
	log.Printf("Starting financial data ingestion for %s", ticker)

	// Get ticker ID
	tickerID, err := database.GetTickerIDBySymbol(ticker)
	if err != nil {
		return fmt.Errorf("failed to get ticker ID for %s: %w", ticker, err)
	}

	// Fetch from Polygon.io API
	params := FinancialsRequestParams{
		Ticker:    ticker,
		Timeframe: "quarterly",
		Limit:     100, // Get up to 100 quarters (25 years)
	}

	// Fetch all financial data
	financials, err := s.polygonClient.GetAllFinancialsWithPagination(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to fetch financials from Polygon: %w", err)
	}

	log.Printf("Fetched %d financial records for %s", len(financials), ticker)

	// Process and store each statement
	var successCount, errorCount int
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Use a semaphore to limit concurrent database operations
	sem := make(chan struct{}, 10)

	for _, data := range financials {
		wg.Add(1)
		go func(data models.PolygonFinancialsData) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Process all three statement types from the same filing
			statementTypes := []models.StatementType{
				models.StatementTypeIncome,
				models.StatementTypeBalanceSheet,
				models.StatementTypeCashFlow,
			}

			for _, stmtType := range statementTypes {
				stmt, err := ConvertPolygonToFinancialStatement(data, tickerID, stmtType)
				if err != nil {
					log.Printf("Warning: Failed to convert %s statement for %s: %v", stmtType, ticker, err)
					mu.Lock()
					errorCount++
					mu.Unlock()
					continue
				}

				// Skip if no data for this statement type
				if len(stmt.Data) == 0 {
					continue
				}

				if err := database.UpsertFinancialStatement(stmt); err != nil {
					log.Printf("Warning: Failed to store %s statement for %s: %v", stmtType, ticker, err)
					mu.Lock()
					errorCount++
					mu.Unlock()
					continue
				}

				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(data)
	}

	wg.Wait()

	log.Printf("Completed ingestion for %s: %d statements stored, %d errors", ticker, successCount, errorCount)

	if successCount == 0 && errorCount > 0 {
		return fmt.Errorf("failed to store any financial statements for %s", ticker)
	}

	return nil
}

// IngestFinancialsIfNeeded checks if data needs to be refreshed and ingests if necessary
func (s *FinancialsService) IngestFinancialsIfNeeded(ctx context.Context, ticker string) error {
	// Check if we have any data
	hasData, err := database.HasFinancialStatements(ticker)
	if err != nil {
		log.Printf("Warning: Could not check existing data for %s: %v", ticker, err)
	}

	if !hasData {
		// No data, ingest now
		log.Printf("No financial data found for %s, ingesting...", ticker)
		return s.IngestFinancials(ctx, ticker)
	}

	// Check if data is stale (older than 24 hours since last filing check)
	// For now, we'll skip refresh logic and rely on scheduled jobs
	// This can be enhanced later with a last_checked timestamp

	return nil
}

// GetIncomeStatements returns income statements for a ticker
func (s *FinancialsService) GetIncomeStatements(ctx context.Context, ticker string, timeframe models.Timeframe, limit int) (*models.FinancialsResponse, error) {
	return s.getStatements(ctx, ticker, models.StatementTypeIncome, timeframe, limit)
}

// GetBalanceSheets returns balance sheets for a ticker
func (s *FinancialsService) GetBalanceSheets(ctx context.Context, ticker string, timeframe models.Timeframe, limit int) (*models.FinancialsResponse, error) {
	return s.getStatements(ctx, ticker, models.StatementTypeBalanceSheet, timeframe, limit)
}

// GetCashFlowStatements returns cash flow statements for a ticker
func (s *FinancialsService) GetCashFlowStatements(ctx context.Context, ticker string, timeframe models.Timeframe, limit int) (*models.FinancialsResponse, error) {
	return s.getStatements(ctx, ticker, models.StatementTypeCashFlow, timeframe, limit)
}

// GetRatios returns financial ratios for a ticker from IC Score calculated data
func (s *FinancialsService) GetRatios(ctx context.Context, ticker string, timeframe models.Timeframe, limit int) (*models.FinancialsResponse, error) {
	// Get company metadata
	metadata, err := database.GetCompanyMetadata(ticker)
	if err != nil {
		log.Printf("Warning: Failed to get metadata for %s: %v", ticker, err)
		metadata = &models.FinancialsMetadata{
			CompanyName: ticker,
		}
	}

	// Get ratios from IC Score tables (valuation_ratios + fundamental_metrics_extended)
	records, err := database.GetICScoreRatios(ticker, limit)
	if err != nil {
		log.Printf("Warning: Failed to get IC Score ratios for %s: %v", ticker, err)
		return nil, fmt.Errorf("no financial ratios available for %s: %w", ticker, err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("no financial ratios data available for %s", ticker)
	}

	// Convert IC Score records to FinancialPeriod format
	periods := database.ConvertICScoreRatiosToFinancialPeriods(records)

	return &models.FinancialsResponse{
		Ticker:        ticker,
		StatementType: models.StatementTypeRatios,
		Timeframe:     timeframe,
		Periods:       periods,
		Metadata:      *metadata,
	}, nil
}

// getStatements is the generic method for retrieving financial statements
func (s *FinancialsService) getStatements(ctx context.Context, ticker string, statementType models.StatementType, timeframe models.Timeframe, limit int) (*models.FinancialsResponse, error) {
	// Check if we need to ingest data first
	if err := s.IngestFinancialsIfNeeded(ctx, ticker); err != nil {
		log.Printf("Warning: Failed to ingest financials for %s: %v", ticker, err)
		// Continue anyway - we might have cached data
	}

	// Get company metadata
	metadata, err := database.GetCompanyMetadata(ticker)
	if err != nil {
		log.Printf("Warning: Failed to get metadata for %s: %v", ticker, err)
		metadata = &models.FinancialsMetadata{
			CompanyName: ticker,
		}
	}

	// Get statements with YoY changes
	periods, err := database.GetFinancialStatementsWithYoY(ticker, statementType, timeframe, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get financial statements: %w", err)
	}

	// If no data in database, try fetching directly from API
	if len(periods) == 0 {
		log.Printf("No cached data for %s, fetching from API...", ticker)
		if err := s.IngestFinancials(ctx, ticker); err != nil {
			return nil, fmt.Errorf("no financial data available for %s: %w", ticker, err)
		}

		// Try again after ingestion
		periods, err = database.GetFinancialStatementsWithYoY(ticker, statementType, timeframe, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to get financial statements after ingestion: %w", err)
		}
	}

	return &models.FinancialsResponse{
		Ticker:        ticker,
		StatementType: statementType,
		Timeframe:     timeframe,
		Periods:       periods,
		Metadata:      *metadata,
	}, nil
}

// RefreshFinancials forces a refresh of financial data for a ticker
func (s *FinancialsService) RefreshFinancials(ctx context.Context, ticker string) error {
	return s.IngestFinancials(ctx, ticker)
}

// CalculateFreeCashFlow calculates free cash flow from cash flow statements
func CalculateFreeCashFlow(cashFlowData map[string]interface{}) *float64 {
	operatingCF, ok1 := cashFlowData["net_cash_flow_from_operating_activities"].(float64)
	capex, ok2 := cashFlowData["capital_expenditure"].(float64)

	if !ok1 {
		return nil
	}

	// CapEx is typically negative (cash outflow), so we add it
	var fcf float64
	if ok2 {
		fcf = operatingCF + capex // capex is usually negative
	} else {
		// If no capex data, just return operating cash flow
		fcf = operatingCF
	}

	return &fcf
}

// EnrichCashFlowData adds calculated fields like Free Cash Flow
func EnrichCashFlowData(data map[string]interface{}) map[string]interface{} {
	enriched := make(map[string]interface{})
	for k, v := range data {
		enriched[k] = v
	}

	// Calculate Free Cash Flow
	if fcf := CalculateFreeCashFlow(data); fcf != nil {
		enriched["free_cash_flow"] = *fcf
	}

	return enriched
}

// BatchIngestFinancials ingests financial data for multiple tickers
func (s *FinancialsService) BatchIngestFinancials(ctx context.Context, tickers []string) map[string]error {
	results := make(map[string]error)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Use a semaphore to limit concurrent API calls
	sem := make(chan struct{}, 3) // Limit to 3 concurrent API calls

	for _, ticker := range tickers {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			err := s.IngestFinancials(ctx, t)
			mu.Lock()
			results[t] = err
			mu.Unlock()

			// Add delay between API calls to respect rate limits
			time.Sleep(time.Second)
		}(ticker)
	}

	wg.Wait()
	return results
}
