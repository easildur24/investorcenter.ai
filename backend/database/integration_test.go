package database

import (
	"encoding/json"
	"investorcenter-api/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===================
// Stock Lookup Tests
// ===================

func TestIntegration_GetStockBySymbol(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	// Seed test data
	DB.MustExec(`INSERT INTO tickers (symbol, name, asset_type, sector, exchange)
		VALUES ('AAPL', 'Apple Inc.', 'stock', 'Technology', 'NASDAQ')`)
	DB.MustExec(`INSERT INTO tickers (symbol, name, asset_type)
		VALUES ('SPY', 'SPDR S&P 500 ETF', 'etf')`)

	// Test basic lookup
	stock, err := GetStockBySymbol("AAPL")
	require.NoError(t, err)
	assert.Equal(t, "AAPL", stock.Symbol)
	assert.Equal(t, "Apple Inc.", stock.Name)
	assert.Equal(t, "stock", stock.AssetType)
	assert.Equal(t, "Technology", stock.Sector)

	// Test case-insensitive lookup
	stock2, err := GetStockBySymbol("aapl")
	require.NoError(t, err)
	assert.Equal(t, "AAPL", stock2.Symbol)

	// Test ETF lookup
	etf, err := GetStockBySymbol("SPY")
	require.NoError(t, err)
	assert.Equal(t, "etf", etf.AssetType)

	// Test not found
	_, err = GetStockBySymbol("ZZZZ")
	assert.Error(t, err)
}

func TestIntegration_SearchStocks(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	// Seed test data with different asset types
	DB.MustExec(`INSERT INTO tickers (symbol, name, asset_type) VALUES
		('AAPL', 'Apple Inc.', 'stock'),
		('AMZN', 'Amazon.com Inc.', 'stock'),
		('MSFT', 'Microsoft Corporation', 'stock'),
		('ARKK', 'ARK Innovation ETF', 'etf')`)

	// Search by symbol prefix
	results, err := SearchStocks("AAPL", 10)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(results), 1)
	assert.Equal(t, "AAPL", results[0].Symbol, "Exact match should come first")

	// Search by partial name
	results2, err := SearchStocks("Microsoft", 10)
	require.NoError(t, err)
	assert.Len(t, results2, 1)
	assert.Equal(t, "MSFT", results2[0].Symbol)

	// Search with limit
	results3, err := SearchStocks("A", 2)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(results3), 2)

	// No results
	results4, err := SearchStocks("ZZZZZZ", 10)
	require.NoError(t, err)
	assert.Empty(t, results4)
}

// ===================
// User CRUD Tests
// ===================

func TestIntegration_CreateAndGetUser(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	pwHash := "$2a$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ12345"

	user := &models.User{
		Email:        "test@example.com",
		PasswordHash: &pwHash,
		FullName:     "Test User",
		Timezone:     "America/New_York",
	}

	// Create user
	err := CreateUser(user)
	require.NoError(t, err)
	assert.NotEmpty(t, user.ID, "ID should be generated")
	assert.False(t, user.CreatedAt.IsZero(), "CreatedAt should be set")

	// Get by email
	found, err := GetUserByEmail("test@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, "Test User", found.FullName)
	assert.True(t, found.IsActive)
	assert.False(t, found.EmailVerified)

	// Get by ID
	found2, err := GetUserByID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", found2.Email)

	// Not found
	_, err = GetUserByEmail("nonexistent@example.com")
	assert.Error(t, err)

	// Duplicate email should fail
	dup := &models.User{
		Email:        "test@example.com",
		PasswordHash: &pwHash,
		FullName:     "Duplicate",
		Timezone:     "UTC",
	}
	err = CreateUser(dup)
	assert.Error(t, err, "Duplicate email should fail")
}

func TestIntegration_UpdateUser(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	pwHash := "$2a$10$hash"
	user := &models.User{
		Email:        "update@example.com",
		PasswordHash: &pwHash,
		FullName:     "Original Name",
		Timezone:     "UTC",
	}
	require.NoError(t, CreateUser(user))

	// Update name and timezone
	user.FullName = "Updated Name"
	user.Timezone = "Europe/London"
	err := UpdateUser(user)
	require.NoError(t, err)

	// Verify update
	updated, err := GetUserByID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.FullName)
	assert.Equal(t, "Europe/London", updated.Timezone)
}

// ===================
// Watchlist Tests
// ===================

func TestIntegration_WatchlistCRUD(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	// Create a user
	pwHash := "$2a$10$hash"
	user := &models.User{
		Email:        "wl@test.com",
		PasswordHash: &pwHash,
		FullName:     "WL User",
		Timezone:     "UTC",
	}
	require.NoError(t, CreateUser(user))

	// Insert tickers for FK validation
	DB.MustExec(`INSERT INTO tickers (symbol, name, asset_type) VALUES
		('AAPL', 'Apple Inc.', 'stock'),
		('MSFT', 'Microsoft', 'stock')`)

	// Create watchlist
	wl := &models.WatchList{
		UserID: user.ID,
		Name:   "Tech Stocks",
	}
	err := CreateWatchList(wl)
	require.NoError(t, err)
	assert.NotEmpty(t, wl.ID, "WatchList ID should be generated")

	// Get watchlists for user
	lists, err := GetWatchListsByUserID(user.ID)
	require.NoError(t, err)
	assert.Len(t, lists, 1)
	assert.Equal(t, "Tech Stocks", lists[0].Name)
	assert.Equal(t, 0, lists[0].ItemCount) // No items yet

	// Add ticker to watchlist
	item := &models.WatchListItem{
		WatchListID: wl.ID,
		Symbol:      "AAPL",
	}
	err = AddTickerToWatchList(item)
	require.NoError(t, err)
	assert.NotEmpty(t, item.ID)

	// Add second ticker
	item2 := &models.WatchListItem{
		WatchListID: wl.ID,
		Symbol:      "MSFT",
	}
	err = AddTickerToWatchList(item2)
	require.NoError(t, err)

	// Verify item count
	lists2, err := GetWatchListsByUserID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, lists2[0].ItemCount)

	// Adding duplicate ticker should fail
	dupItem := &models.WatchListItem{
		WatchListID: wl.ID,
		Symbol:      "AAPL",
	}
	err = AddTickerToWatchList(dupItem)
	assert.Error(t, err, "Duplicate ticker should fail")

	// Adding non-existent ticker should fail
	badItem := &models.WatchListItem{
		WatchListID: wl.ID,
		Symbol:      "FAKESYMBOL",
	}
	err = AddTickerToWatchList(badItem)
	assert.Error(t, err, "Non-existent ticker should fail")

	// Remove ticker
	err = RemoveTickerFromWatchList(wl.ID, "AAPL")
	require.NoError(t, err)

	// Verify item count after removal
	lists3, err := GetWatchListsByUserID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, lists3[0].ItemCount)

	// Remove non-existent ticker should fail
	err = RemoveTickerFromWatchList(wl.ID, "ZZZZ")
	assert.Error(t, err)
}

func TestIntegration_WatchlistMultiple(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	// Create user
	pwHash := "$2a$10$hash"
	user := &models.User{
		Email:        "multi@test.com",
		PasswordHash: &pwHash,
		FullName:     "Multi WL User",
		Timezone:     "UTC",
	}
	require.NoError(t, CreateUser(user))

	// Create multiple watchlists
	wl1 := &models.WatchList{UserID: user.ID, Name: "Favorites"}
	wl2 := &models.WatchList{UserID: user.ID, Name: "Research"}
	require.NoError(t, CreateWatchList(wl1))
	require.NoError(t, CreateWatchList(wl2))

	// Get all watchlists
	lists, err := GetWatchListsByUserID(user.ID)
	require.NoError(t, err)
	assert.Len(t, lists, 2)

	// Verify ordering (display_order ASC, then created_at ASC)
	assert.Equal(t, "Favorites", lists[0].Name)
	assert.Equal(t, "Research", lists[1].Name)
}

// ===================
// Screener Tests
// ===================

func TestIntegration_ScreenerNoFilters(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	// Seed screener data
	DB.MustExec(`INSERT INTO screener_data (symbol, name, sector, market_cap, pe_ratio, ic_score) VALUES
		('AAPL', 'Apple', 'Technology', 3000000000000, 28.5, 85.0),
		('MSFT', 'Microsoft', 'Technology', 2800000000000, 32.0, 82.0),
		('JNJ', 'Johnson & Johnson', 'Healthcare', 400000000000, 15.0, 70.0)`)

	params := models.ScreenerParams{
		Page:  1,
		Limit: 10,
		Sort:  "market_cap",
		Order: "DESC",
	}

	stocks, total, err := GetScreenerStocks(params)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, stocks, 3)
	// Sorted by market_cap DESC: AAPL (3T), MSFT (2.8T), JNJ (400B)
	assert.Equal(t, "AAPL", stocks[0].Symbol)
	assert.Equal(t, "MSFT", stocks[1].Symbol)
	assert.Equal(t, "JNJ", stocks[2].Symbol)
}

func TestIntegration_ScreenerWithFilters(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	DB.MustExec(`INSERT INTO screener_data (symbol, name, sector, market_cap, pe_ratio, ic_score) VALUES
		('AAPL', 'Apple', 'Technology', 3000000000000, 28.5, 85.0),
		('MSFT', 'Microsoft', 'Technology', 2800000000000, 32.0, 82.0),
		('JNJ', 'Johnson & Johnson', 'Healthcare', 400000000000, 15.0, 70.0),
		('PFE', 'Pfizer', 'Healthcare', 150000000000, 10.0, 55.0)`)

	// Filter by sector
	params := models.ScreenerParams{
		Page:    1,
		Limit:   10,
		Sort:    "market_cap",
		Order:   "DESC",
		Sectors: []string{"Technology"},
	}
	stocks, total, err := GetScreenerStocks(params)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	for _, s := range stocks {
		assert.Equal(t, "Technology", *s.Sector)
	}

	// Filter by PE range
	minPE := 20.0
	maxPE := 35.0
	params2 := models.ScreenerParams{
		Page:  1,
		Limit: 10,
		Sort:  "pe_ratio",
		Order: "ASC",
		PEMin: &minPE,
		PEMax: &maxPE,
	}
	stocks2, total2, err := GetScreenerStocks(params2)
	require.NoError(t, err)
	assert.Equal(t, 2, total2) // AAPL (28.5) and MSFT (32.0)
	// Sorted by PE ASC
	assert.Equal(t, "AAPL", stocks2[0].Symbol)
	assert.Equal(t, "MSFT", stocks2[1].Symbol)
}

func TestIntegration_ScreenerPagination(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	DB.MustExec(`INSERT INTO screener_data (symbol, name, sector, market_cap, ic_score) VALUES
		('AAPL', 'Apple', 'Technology', 3000000000000, 85.0),
		('MSFT', 'Microsoft', 'Technology', 2800000000000, 82.0),
		('GOOGL', 'Alphabet', 'Technology', 2000000000000, 80.0),
		('AMZN', 'Amazon', 'Technology', 1800000000000, 78.0),
		('META', 'Meta', 'Technology', 1300000000000, 76.0)`)

	// Page 1, 2 per page
	params := models.ScreenerParams{
		Page:  1,
		Limit: 2,
		Sort:  "market_cap",
		Order: "DESC",
	}
	stocks, total, err := GetScreenerStocks(params)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, stocks, 2)
	assert.Equal(t, "AAPL", stocks[0].Symbol)
	assert.Equal(t, "MSFT", stocks[1].Symbol)

	// Page 2
	params.Page = 2
	stocks2, total2, err := GetScreenerStocks(params)
	require.NoError(t, err)
	assert.Equal(t, 5, total2)
	assert.Len(t, stocks2, 2)
	assert.Equal(t, "GOOGL", stocks2[0].Symbol)
	assert.Equal(t, "AMZN", stocks2[1].Symbol)

	// Page 3 (last page, only 1 item)
	params.Page = 3
	stocks3, _, err := GetScreenerStocks(params)
	require.NoError(t, err)
	assert.Len(t, stocks3, 1)
	assert.Equal(t, "META", stocks3[0].Symbol)
}

// ========================================
// Batch 1: Financial Data Tests
// ========================================

func TestIntegration_GetTickerIDBySymbol(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	DB.MustExec(`INSERT INTO tickers (symbol, name, asset_type) VALUES ('AAPL', 'Apple Inc.', 'stock')`)

	// Found
	id, err := GetTickerIDBySymbol("AAPL")
	require.NoError(t, err)
	assert.Greater(t, id, 0)

	// Case-insensitive
	id2, err := GetTickerIDBySymbol("aapl")
	require.NoError(t, err)
	assert.Equal(t, id, id2)

	// Not found
	_, err = GetTickerIDBySymbol("ZZZZ")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ticker not found")
}

func TestIntegration_UpsertFinancialStatement(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	DB.MustExec(`INSERT INTO tickers (symbol, name, asset_type) VALUES ('AAPL', 'Apple Inc.', 'stock')`)
	tickerID, err := GetTickerIDBySymbol("AAPL")
	require.NoError(t, err)

	periodEnd := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	q := 4
	cik := "0000320193"

	stmt := &models.FinancialStatement{
		TickerID:      tickerID,
		CIK:           &cik,
		StatementType: models.StatementTypeIncome,
		Timeframe:     models.TimeframeQuarterly,
		FiscalYear:    2024,
		FiscalQuarter: &q,
		PeriodEnd:     periodEnd,
		Data:          models.FinancialData{"revenues": 94000000000.0, "net_income_loss": 24000000000.0},
	}

	// Insert
	err = UpsertFinancialStatement(stmt)
	require.NoError(t, err)
	assert.Greater(t, stmt.ID, 0)

	// Upsert same key — should update, not duplicate
	originalID := stmt.ID
	stmt.Data = models.FinancialData{"revenues": 95000000000.0, "net_income_loss": 25000000000.0}
	err = UpsertFinancialStatement(stmt)
	require.NoError(t, err)
	assert.Equal(t, originalID, stmt.ID, "Upsert should return same ID")
}

func TestIntegration_GetFinancialStatements(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	DB.MustExec(`INSERT INTO tickers (symbol, name, asset_type) VALUES ('AAPL', 'Apple Inc.', 'stock')`)
	tickerID, err := GetTickerIDBySymbol("AAPL")
	require.NoError(t, err)

	// Seed multiple statements
	for i := 1; i <= 4; i++ {
		q := i
		pe := time.Date(2024, time.Month(i*3), 28, 0, 0, 0, 0, time.UTC)
		s := &models.FinancialStatement{
			TickerID:      tickerID,
			StatementType: models.StatementTypeIncome,
			Timeframe:     models.TimeframeQuarterly,
			FiscalYear:    2024,
			FiscalQuarter: &q,
			PeriodEnd:     pe,
			Data:          models.FinancialData{"revenues": float64(i) * 1e9},
		}
		require.NoError(t, UpsertFinancialStatement(s))
	}

	// Default: DESC, limit 8
	params := models.FinancialsParams{Ticker: "AAPL"}
	stmts, err := GetFinancialStatements(params)
	require.NoError(t, err)
	assert.Len(t, stmts, 4)
	assert.Equal(t, 2024, stmts[0].FiscalYear)

	// Timeframe filter
	params2 := models.FinancialsParams{Ticker: "AAPL", Timeframe: models.TimeframeQuarterly}
	stmts2, err := GetFinancialStatements(params2)
	require.NoError(t, err)
	assert.Len(t, stmts2, 4)

	// Fiscal year filter
	fy := 2024
	params3 := models.FinancialsParams{Ticker: "AAPL", FiscalYear: &fy, Limit: 2}
	stmts3, err := GetFinancialStatements(params3)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(stmts3), 2)

	// Sort ASC
	params4 := models.FinancialsParams{Ticker: "AAPL", Sort: "asc"}
	stmts4, err := GetFinancialStatements(params4)
	require.NoError(t, err)
	assert.True(t, stmts4[0].PeriodEnd.Before(stmts4[len(stmts4)-1].PeriodEnd), "ASC should sort oldest first")

	// Ticker not found
	params5 := models.FinancialsParams{Ticker: "ZZZZ"}
	_, err = GetFinancialStatements(params5)
	assert.Error(t, err)
}

func TestIntegration_GetFinancialStatementsByType(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	DB.MustExec(`INSERT INTO tickers (symbol, name, asset_type) VALUES ('MSFT', 'Microsoft', 'stock')`)
	tickerID, err := GetTickerIDBySymbol("MSFT")
	require.NoError(t, err)

	// Seed income + balance sheet
	for _, st := range []models.StatementType{models.StatementTypeIncome, models.StatementTypeBalanceSheet} {
		q := 1
		s := &models.FinancialStatement{
			TickerID:      tickerID,
			StatementType: st,
			Timeframe:     models.TimeframeQuarterly,
			FiscalYear:    2024,
			FiscalQuarter: &q,
			PeriodEnd:     time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC),
			Data:          models.FinancialData{"key": "value"},
		}
		require.NoError(t, UpsertFinancialStatement(s))
	}

	// Filter by income
	results, err := GetFinancialStatementsByType("MSFT", models.StatementTypeIncome, models.TimeframeQuarterly, 10)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, models.StatementTypeIncome, results[0].StatementType)

	// Filter by balance sheet
	results2, err := GetFinancialStatementsByType("MSFT", models.StatementTypeBalanceSheet, models.TimeframeQuarterly, 10)
	require.NoError(t, err)
	assert.Len(t, results2, 1)
}

func TestIntegration_HasFinancialStatements(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	DB.MustExec(`INSERT INTO tickers (symbol, name, asset_type) VALUES ('AAPL', 'Apple', 'stock')`)
	tickerID, _ := GetTickerIDBySymbol("AAPL")

	// No statements yet
	has, err := HasFinancialStatements("AAPL")
	require.NoError(t, err)
	assert.False(t, has)

	// Add one
	q := 1
	s := &models.FinancialStatement{
		TickerID: tickerID, StatementType: models.StatementTypeIncome,
		Timeframe: models.TimeframeAnnual, FiscalYear: 2024, FiscalQuarter: &q,
		PeriodEnd: time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		Data:      models.FinancialData{"revenues": 1e9},
	}
	require.NoError(t, UpsertFinancialStatement(s))

	has2, err := HasFinancialStatements("AAPL")
	require.NoError(t, err)
	assert.True(t, has2)

	// Non-existent ticker should return false (not error)
	has3, err := HasFinancialStatements("ZZZZ")
	require.NoError(t, err)
	assert.False(t, has3)
}

func TestIntegration_DeleteFinancialStatements(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	DB.MustExec(`INSERT INTO tickers (symbol, name, asset_type) VALUES ('AAPL', 'Apple', 'stock')`)
	tickerID, _ := GetTickerIDBySymbol("AAPL")

	q := 1
	s := &models.FinancialStatement{
		TickerID: tickerID, StatementType: models.StatementTypeIncome,
		Timeframe: models.TimeframeAnnual, FiscalYear: 2024, FiscalQuarter: &q,
		PeriodEnd: time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		Data:      models.FinancialData{"revenues": 1e9},
	}
	require.NoError(t, UpsertFinancialStatement(s))

	err := DeleteFinancialStatements("AAPL")
	require.NoError(t, err)

	has, _ := HasFinancialStatements("AAPL")
	assert.False(t, has)
}

func TestIntegration_GetEPSEstimates(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	// Seed EPS estimates directly — use current/future years so
	// GetEPSEstimate (which filters fiscal_year >= CURRENT_YEAR) works.
	currentYear := time.Now().Year()
	nextYear := currentYear + 1
	DB.MustExec(`INSERT INTO eps_estimates (ticker, fiscal_year, fiscal_quarter, consensus_eps, num_analysts)
		VALUES ('AAPL', $1, NULL, 7.50, 35)`, nextYear)
	DB.MustExec(`INSERT INTO eps_estimates (ticker, fiscal_year, fiscal_quarter, consensus_eps, num_analysts)
		VALUES ('AAPL', $1, 1, 1.80, 30)`, nextYear)
	DB.MustExec(`INSERT INTO eps_estimates (ticker, fiscal_year, fiscal_quarter, consensus_eps, num_analysts)
		VALUES ('AAPL', $1, NULL, 6.90, 35)`, currentYear)

	// Get all for ticker
	estimates, err := GetEPSEstimates("AAPL", 10)
	require.NoError(t, err)
	assert.Len(t, estimates, 3)
	// Ordered by fiscal_year DESC, fiscal_quarter DESC NULLS FIRST
	assert.Equal(t, nextYear, estimates[0].FiscalYear)

	// Get current annual estimate (fiscal_year >= current year, quarter IS NULL)
	current, err := GetEPSEstimate("AAPL")
	require.NoError(t, err)
	assert.NotNil(t, current)
	assert.Nil(t, current.FiscalQuarter) // Annual

	// Empty result for non-existent ticker
	est2, err := GetEPSEstimates("ZZZZ", 10)
	require.NoError(t, err)
	assert.Empty(t, est2)
}

func TestIntegration_GetICScoreRatios(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	// Seed valuation_ratios and fundamental_metrics_extended
	DB.MustExec(`INSERT INTO valuation_ratios (ticker, calculation_date, stock_price, ttm_pe_ratio, ttm_pb_ratio, ttm_ps_ratio)
		VALUES ('AAPL', '2024-12-15', 195.50, 28.5, 45.2, 7.8)`)
	DB.MustExec(`INSERT INTO fundamental_metrics_extended (ticker, calculation_date, gross_margin, operating_margin, net_margin, roe, roa, current_ratio, debt_to_equity)
		VALUES ('AAPL', '2024-12-15', 0.455, 0.312, 0.256, 1.56, 0.28, 1.07, 1.87)`)

	// JOIN query
	records, err := GetICScoreRatios("AAPL", 10)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "AAPL", records[0].Ticker)
	assert.NotNil(t, records[0].TTMPERatio)
	assert.InDelta(t, 28.5, *records[0].TTMPERatio, 0.1)
	assert.NotNil(t, records[0].GrossMargin)
	assert.InDelta(t, 0.455, *records[0].GrossMargin, 0.001)

	// Default limit
	records2, err := GetICScoreRatios("AAPL", 0)
	require.NoError(t, err)
	assert.Len(t, records2, 1) // default limit = 8

	// No data ticker
	records3, err := GetICScoreRatios("ZZZZ", 10)
	require.NoError(t, err)
	assert.Empty(t, records3)
}

func TestIntegration_GetSectorPercentiles(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	DB.MustExec(`INSERT INTO mv_latest_sector_percentiles
		(sector, metric_name, calculated_at, min_value, p10_value, p25_value, p50_value, p75_value, p90_value, max_value, mean_value, std_dev, sample_count)
		VALUES ('Technology', 'pe_ratio', '2024-12-15', 5.0, 12.0, 18.0, 25.0, 35.0, 50.0, 100.0, 27.5, 15.0, 500)`)

	// Get by sector + metric
	perc, err := GetSectorPercentile("Technology", "pe_ratio")
	require.NoError(t, err)
	assert.NotNil(t, perc)

	// Get all for sector
	allPerc, err := GetSectorPercentiles("Technology")
	require.NoError(t, err)
	assert.Len(t, allPerc, 1)

	// Get all sectors
	sectors, err := GetAllSectors()
	require.NoError(t, err)
	assert.Contains(t, sectors, "Technology")

	// Not found returns nil, nil
	notFoundPerc, err := GetSectorPercentile("NonexistentSector", "pe_ratio")
	require.NoError(t, err)
	assert.Nil(t, notFoundPerc)
}

// ========================================
// Batch 2: User Data Tests
// ========================================

func TestIntegration_AlertsCRUD(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	// Create user + watchlist
	pwHash := "$2a$10$hash"
	user := &models.User{Email: "alert@test.com", PasswordHash: &pwHash, FullName: "Alert User", Timezone: "UTC"}
	require.NoError(t, CreateUser(user))

	wl := &models.WatchList{UserID: user.ID, Name: "Alerts WL"}
	require.NoError(t, CreateWatchList(wl))

	DB.MustExec(`INSERT INTO tickers (symbol, name, asset_type) VALUES ('AAPL', 'Apple', 'stock')`)

	conditions := json.RawMessage(`{"threshold": 200.0, "comparison": "above"}`)
	alert := &models.AlertRule{
		UserID:      user.ID,
		WatchListID: wl.ID,
		Symbol:      "AAPL",
		AlertType:   "price_above",
		Conditions:  conditions,
		IsActive:    true,
		Frequency:   "once",
		NotifyEmail: true,
		NotifyInApp: true,
		Name:        "AAPL above $200",
	}

	// Create
	err := CreateAlertRule(alert)
	require.NoError(t, err)
	assert.NotEmpty(t, alert.ID)
	assert.Equal(t, 0, alert.TriggerCount)

	// Get by ID
	found, err := GetAlertRuleByID(alert.ID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "AAPL", found.Symbol)
	assert.Equal(t, "price_above", found.AlertType)

	// Get by user
	alerts, err := GetAlertRulesByUserID(user.ID, "", "")
	require.NoError(t, err)
	assert.Len(t, alerts, 1)

	// Filter by active status
	alertsActive, err := GetAlertRulesByUserID(user.ID, "", "true")
	require.NoError(t, err)
	assert.Len(t, alertsActive, 1)

	// Update
	err = UpdateAlertRule(alert.ID, user.ID, map[string]interface{}{"name": "Updated Name"})
	require.NoError(t, err)
	updated, _ := GetAlertRuleByID(alert.ID, user.ID)
	assert.Equal(t, "Updated Name", updated.Name)

	// Trigger count increment
	err = UpdateAlertRuleTrigger(alert.ID)
	require.NoError(t, err)
	triggered, _ := GetAlertRuleByID(alert.ID, user.ID)
	assert.Equal(t, 1, triggered.TriggerCount)
	assert.NotNil(t, triggered.LastTriggeredAt)

	// Count
	count, err := CountAlertRulesByUserID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Delete
	err = DeleteAlertRule(alert.ID, user.ID)
	require.NoError(t, err)
	_, err = GetAlertRuleByID(alert.ID, user.ID)
	assert.Error(t, err)
}

func TestIntegration_AlertLogs(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	// Setup user, watchlist, alert rule
	pwHash := "$2a$10$hash"
	user := &models.User{Email: "log@test.com", PasswordHash: &pwHash, FullName: "Log User", Timezone: "UTC"}
	require.NoError(t, CreateUser(user))
	wl := &models.WatchList{UserID: user.ID, Name: "WL"}
	require.NoError(t, CreateWatchList(wl))
	DB.MustExec(`INSERT INTO tickers (symbol, name, asset_type) VALUES ('AAPL', 'Apple', 'stock')`)

	alert := &models.AlertRule{
		UserID: user.ID, WatchListID: wl.ID, Symbol: "AAPL",
		AlertType: "price_above", Conditions: json.RawMessage(`{}`),
		IsActive: true, Frequency: "once", NotifyEmail: true, NotifyInApp: true,
		Name: "Test Alert",
	}
	require.NoError(t, CreateAlertRule(alert))

	// Create log
	log := &models.AlertLog{
		AlertRuleID:      alert.ID,
		UserID:           user.ID,
		Symbol:           "AAPL",
		AlertType:        "price_above",
		ConditionMet:     json.RawMessage(`{"price": 205.0}`),
		MarketData:       json.RawMessage(`{"current_price": 205.0}`),
		NotificationSent: true,
	}
	err := CreateAlertLog(log)
	require.NoError(t, err)
	assert.NotEmpty(t, log.ID)
	assert.False(t, log.IsRead)

	// Get logs by user
	logs, err := GetAlertLogsByUserID(user.ID, "", "", 10, 0)
	require.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, "Test Alert", logs[0].RuleName)

	// Mark as read
	err = MarkAlertLogAsRead(log.ID, user.ID)
	require.NoError(t, err)

	// Mark as dismissed
	err = MarkAlertLogAsDismissed(log.ID, user.ID)
	require.NoError(t, err)

	// Not found
	err = MarkAlertLogAsRead("00000000-0000-0000-0000-000000000000", user.ID)
	assert.Error(t, err)
}

func TestIntegration_NotificationPreferences(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	pwHash := "$2a$10$hash"
	user := &models.User{Email: "notif@test.com", PasswordHash: &pwHash, FullName: "Notif User", Timezone: "UTC"}
	require.NoError(t, CreateUser(user))

	// Seed notification preferences
	DB.MustExec(`INSERT INTO notification_preferences (user_id) VALUES ($1)`, user.ID)

	// Get
	prefs, err := GetNotificationPreferences(user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, prefs.UserID)
	assert.True(t, prefs.EmailEnabled, "Default email_enabled should be true")
	assert.True(t, prefs.PriceAlertsEnabled)
	assert.Equal(t, 50, prefs.MaxAlertsPerDay)

	// Update
	err = UpdateNotificationPreferences(user.ID, map[string]interface{}{
		"email_enabled":      false,
		"max_alerts_per_day": 100,
	})
	require.NoError(t, err)

	// Verify update
	prefs2, err := GetNotificationPreferences(user.ID)
	require.NoError(t, err)
	assert.False(t, prefs2.EmailEnabled)
	assert.Equal(t, 100, prefs2.MaxAlertsPerDay)

	// Not found
	_, err = GetNotificationPreferences("00000000-0000-0000-0000-000000000000")
	assert.Error(t, err)
}

func TestIntegration_InAppNotifications(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	pwHash := "$2a$10$hash"
	user := &models.User{Email: "inapp@test.com", PasswordHash: &pwHash, FullName: "InApp User", Timezone: "UTC"}
	require.NoError(t, CreateUser(user))

	// Create notifications (Metadata must be valid JSON for JSONB column)
	n1 := &models.InAppNotification{
		UserID:   user.ID,
		Type:     "alert",
		Title:    "Price Alert",
		Message:  "AAPL crossed $200",
		Metadata: json.RawMessage(`{}`),
	}
	err := CreateInAppNotification(n1)
	require.NoError(t, err)
	assert.NotEmpty(t, n1.ID)
	assert.False(t, n1.IsRead)

	n2 := &models.InAppNotification{
		UserID:   user.ID,
		Type:     "system",
		Title:    "Welcome",
		Message:  "Welcome to InvestorCenter",
		Metadata: json.RawMessage(`{}`),
	}
	require.NoError(t, CreateInAppNotification(n2))

	// Get all
	all, err := GetInAppNotifications(user.ID, false, 0)
	require.NoError(t, err)
	assert.Len(t, all, 2)

	// Unread only
	unread, err := GetInAppNotifications(user.ID, true, 0)
	require.NoError(t, err)
	assert.Len(t, unread, 2)

	// Unread count
	count, err := GetUnreadNotificationCount(user.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Mark one as read
	err = MarkNotificationAsRead(n1.ID, user.ID)
	require.NoError(t, err)

	count2, _ := GetUnreadNotificationCount(user.ID)
	assert.Equal(t, 1, count2)

	// Mark all as read
	err = MarkAllNotificationsAsRead(user.ID)
	require.NoError(t, err)

	count3, _ := GetUnreadNotificationCount(user.ID)
	assert.Equal(t, 0, count3)

	// Dismiss
	err = DismissNotification(n2.ID, user.ID)
	require.NoError(t, err)

	// Not found
	err = MarkNotificationAsRead("00000000-0000-0000-0000-000000000000", user.ID)
	assert.Error(t, err)
}

func TestIntegration_SessionManagement(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	pwHash := "$2a$10$hash"
	user := &models.User{Email: "session@test.com", PasswordHash: &pwHash, FullName: "Session User", Timezone: "UTC"}
	require.NoError(t, CreateUser(user))

	// Create session
	session := &models.Session{
		UserID:           user.ID,
		RefreshTokenHash: "hash_abc123",
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
		UserAgent:        strPtr("Mozilla/5.0"),
		IPAddress:        strPtr("192.168.1.1"),
	}
	err := CreateSession(session)
	require.NoError(t, err)
	assert.NotEmpty(t, session.ID)

	// Get by token hash
	found, err := GetSessionByRefreshTokenHash("hash_abc123")
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.UserID)

	// Expired session should not be found
	expiredSession := &models.Session{
		UserID:           user.ID,
		RefreshTokenHash: "expired_hash",
		ExpiresAt:        time.Now().Add(-1 * time.Hour),
	}
	require.NoError(t, CreateSession(expiredSession))
	_, err = GetSessionByRefreshTokenHash("expired_hash")
	assert.Error(t, err)

	// Update last used
	err = UpdateSessionLastUsed(session.ID)
	require.NoError(t, err)

	// Delete session
	err = DeleteSession(session.ID)
	require.NoError(t, err)
	_, err = GetSessionByRefreshTokenHash("hash_abc123")
	assert.Error(t, err)

	// Delete all user sessions
	s2 := &models.Session{UserID: user.ID, RefreshTokenHash: "hash2", ExpiresAt: time.Now().Add(24 * time.Hour)}
	s3 := &models.Session{UserID: user.ID, RefreshTokenHash: "hash3", ExpiresAt: time.Now().Add(24 * time.Hour)}
	require.NoError(t, CreateSession(s2))
	require.NoError(t, CreateSession(s3))
	err = DeleteUserSessions(user.ID)
	require.NoError(t, err)

	// Cleanup expired
	err = CleanupExpiredSessions()
	require.NoError(t, err)
}

func TestIntegration_PasswordResetFlow(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	pwHash := "$2a$10$hash"
	user := &models.User{Email: "reset@test.com", PasswordHash: &pwHash, FullName: "Reset User", Timezone: "UTC"}
	require.NoError(t, CreateUser(user))

	// Generate token
	token, err := GenerateResetToken()
	require.NoError(t, err)
	assert.Len(t, token, 64) // 32 bytes hex-encoded

	// Create reset token
	resetToken, err := CreatePasswordResetToken(user.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, resetToken.ID)
	assert.NotEmpty(t, resetToken.Token)
	assert.Equal(t, user.ID, resetToken.UserID)

	// Get valid token
	found, err := GetPasswordResetToken(resetToken.Token)
	require.NoError(t, err)
	assert.Equal(t, resetToken.ID, found.ID)
	assert.False(t, found.Used)

	// Mark as used
	err = MarkResetTokenAsUsed(resetToken.ID)
	require.NoError(t, err)

	// Used token should error
	_, err = GetPasswordResetToken(resetToken.Token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already been used")

	// Invalid token
	_, err = GetPasswordResetToken("nonexistent_token")
	assert.Error(t, err)

	// Delete expired
	err = DeleteExpiredResetTokens()
	require.NoError(t, err)
}

// ========================================
// Batch 3: Social/Sentiment Tests
// ========================================

func TestIntegration_SentimentLexicon(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	cat := "market_action"

	// Add terms
	term1 := &models.SentimentLexiconTerm{Term: "bullish", Sentiment: "bullish", Weight: 1.0, Category: &cat}
	term2 := &models.SentimentLexiconTerm{Term: "bearish", Sentiment: "bearish", Weight: 1.0, Category: &cat}
	term3 := &models.SentimentLexiconTerm{Term: "very", Sentiment: "modifier", Weight: 1.5}
	require.NoError(t, AddSentimentTerm(term1))
	require.NoError(t, AddSentimentTerm(term2))
	require.NoError(t, AddSentimentTerm(term3))
	assert.Greater(t, term1.ID, 0)

	// Get all
	all, err := GetSentimentLexicon()
	require.NoError(t, err)
	assert.Len(t, all, 3)

	// Get by sentiment
	bullish, err := GetSentimentTermsBySentiment("bullish")
	require.NoError(t, err)
	assert.Len(t, bullish, 1)

	// Get by category
	byCat, err := GetSentimentTermsByCategory("market_action")
	require.NoError(t, err)
	assert.Len(t, byCat, 2)

	// Lookup term (case-insensitive)
	found, err := LookupTerm("BULLISH")
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "bullish", found.Term) // stored lowercase

	// Lookup not found
	notFound, err := LookupTerm("nonexistent")
	require.NoError(t, err)
	assert.Nil(t, notFound)

	// Lexicon as map
	lexMap, err := GetLexiconAsMap()
	require.NoError(t, err)
	assert.Len(t, lexMap, 3)
	_, exists := lexMap["bullish"]
	assert.True(t, exists)

	// Stats
	stats, err := GetLexiconStats()
	require.NoError(t, err)
	assert.Equal(t, 3, stats["total"])
	assert.Equal(t, 1, stats["bullish"])
	assert.Equal(t, 1, stats["bearish"])
	assert.Equal(t, 1, stats["modifiers"])

	// Delete term
	err = DeleteSentimentTerm(term3.ID)
	require.NoError(t, err)

	all2, _ := GetSentimentLexicon()
	assert.Len(t, all2, 2)

	// Delete not found
	err = DeleteSentimentTerm(99999)
	assert.Error(t, err)
}

func TestIntegration_SocialPostsCRUD(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	now := time.Now()
	sentiment := "bullish"
	conf := 0.85
	body := "Great earnings!"
	flair := "DD"

	post := &models.SocialPost{
		ExternalPostID:      "reddit_abc123",
		Source:              "reddit",
		Ticker:              "AAPL",
		Subreddit:           "wallstreetbets",
		Title:               "AAPL to the moon",
		BodyPreview:         &body,
		URL:                 "https://reddit.com/r/wsb/abc",
		Upvotes:             1500,
		CommentCount:        200,
		AwardCount:          5,
		Sentiment:           &sentiment,
		SentimentConfidence: &conf,
		Flair:               &flair,
		PostedAt:            now,
	}

	// Upsert (insert)
	err := UpsertSocialPost(post)
	require.NoError(t, err)
	assert.Greater(t, post.ID, int64(0))

	// Upsert same post (update)
	post.Upvotes = 2000
	err = UpsertSocialPost(post)
	require.NoError(t, err)

	// Update sentiment
	err = UpdatePostSentiment("reddit_abc123", "bearish", 0.90)
	require.NoError(t, err)

	// Post count
	count, err := GetPostCountByTicker("AAPL", 7)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Bulk upsert
	posts := []models.SocialPost{
		{
			ExternalPostID: "reddit_def456", Source: "reddit", Ticker: "AAPL",
			Subreddit: "stocks", Title: "AAPL analysis", URL: "https://reddit.com/r/stocks/def",
			Upvotes: 500, PostedAt: now,
		},
		{
			ExternalPostID: "reddit_ghi789", Source: "reddit", Ticker: "TSLA",
			Subreddit: "wallstreetbets", Title: "TSLA calls", URL: "https://reddit.com/r/wsb/ghi",
			Upvotes: 3000, PostedAt: now,
		},
	}
	inserted, err := BulkUpsertPosts(posts)
	require.NoError(t, err)
	assert.Equal(t, 2, inserted)

	// Exists check
	cnt, err := GetPostCountByExternalID("reddit_abc123")
	require.NoError(t, err)
	assert.Equal(t, 1, cnt)
}

func TestIntegration_RedditHeatmapData(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	today := time.Now().Truncate(24 * time.Hour)

	// Seed multi-day data
	for i := 0; i < 5; i++ {
		date := today.AddDate(0, 0, -i)
		DB.MustExec(`INSERT INTO reddit_heatmap_daily
			(ticker_symbol, date, avg_rank, min_rank, max_rank, total_mentions, total_upvotes,
			 rank_volatility, trend_direction, popularity_score, data_source)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			"AAPL", date, 5.5, 3, 8, 100+i*10, 5000+i*200,
			1.5, "up", float64(200+i*20), "apewisdom")
	}
	// Seed another ticker with fewer days
	for i := 0; i < 3; i++ {
		date := today.AddDate(0, 0, -i)
		DB.MustExec(`INSERT INTO reddit_heatmap_daily
			(ticker_symbol, date, avg_rank, min_rank, max_rank, total_mentions, total_upvotes,
			 rank_volatility, trend_direction, popularity_score, data_source)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			"TSLA", date, 2.0, 1, 3, 200+i*10, 10000+i*500,
			0.8, "up", float64(350+i*30), "apewisdom")
	}

	// Get heatmap
	heatmap, err := GetRedditHeatmap(7, 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(heatmap), 1)

	// Get ticker history
	history, err := GetTickerRedditHistory("AAPL", 30)
	require.NoError(t, err)
	assert.NotNil(t, history)
	assert.Equal(t, "AAPL", history.TickerSymbol)
	assert.Len(t, history.History, 5)
	assert.Equal(t, 5, history.Summary.DaysAppeared)

	// Get latest date
	latest, err := GetLatestRedditDate()
	require.NoError(t, err)
	assert.False(t, latest.IsZero())

	// Not found ticker
	_, err = GetTickerRedditHistory("ZZZZ", 30)
	assert.Error(t, err)
}

// ========================================
// Batch 4: Admin / Config Tests
// ========================================

func TestIntegration_HeatmapConfigCRUD(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	pwHash := "$2a$10$hash"
	user := &models.User{Email: "heatmap@test.com", PasswordHash: &pwHash, FullName: "HM User", Timezone: "UTC"}
	require.NoError(t, CreateUser(user))
	wl := &models.WatchList{UserID: user.ID, Name: "HM WL"}
	require.NoError(t, CreateWatchList(wl))

	// Create config
	config := &models.HeatmapConfig{
		UserID:       user.ID,
		WatchListID:  wl.ID,
		Name:         "My Heatmap",
		SizeMetric:   "market_cap",
		ColorMetric:  "price_change_pct",
		TimePeriod:   "1D",
		ColorScheme:  "red_green",
		LabelDisplay: "symbol_change",
		LayoutType:   "treemap",
		FiltersJSON:  map[string]interface{}{},
		IsDefault:    true,
	}
	err := CreateHeatmapConfig(config)
	require.NoError(t, err)
	assert.NotEmpty(t, config.ID)

	// Get by ID
	found, err := GetHeatmapConfigByID(config.ID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "My Heatmap", found.Name)
	assert.True(t, found.IsDefault)

	// Get by watch list
	configs, err := GetHeatmapConfigsByWatchListID(wl.ID, user.ID)
	require.NoError(t, err)
	assert.Len(t, configs, 1)

	// Get default
	defaultConfig, err := GetDefaultHeatmapConfig(wl.ID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, config.ID, defaultConfig.ID)

	// Create second config (non-default)
	config2 := &models.HeatmapConfig{
		UserID: user.ID, WatchListID: wl.ID, Name: "Second",
		SizeMetric: "market_cap", ColorMetric: "ic_score", TimePeriod: "1W",
		ColorScheme: "blue_red", LabelDisplay: "symbol", LayoutType: "treemap",
		FiltersJSON: map[string]interface{}{}, IsDefault: false,
	}
	require.NoError(t, CreateHeatmapConfig(config2))

	configs2, _ := GetHeatmapConfigsByWatchListID(wl.ID, user.ID)
	assert.Len(t, configs2, 2)

	// Update
	config.Name = "Updated Heatmap"
	err = UpdateHeatmapConfig(config)
	require.NoError(t, err)
	upd, _ := GetHeatmapConfigByID(config.ID, user.ID)
	assert.Equal(t, "Updated Heatmap", upd.Name)

	// Delete
	err = DeleteHeatmapConfig(config2.ID, user.ID)
	require.NoError(t, err)
	configs3, _ := GetHeatmapConfigsByWatchListID(wl.ID, user.ID)
	assert.Len(t, configs3, 1)

	// Not found
	_, err = GetHeatmapConfigByID("00000000-0000-0000-0000-000000000000", user.ID)
	assert.Error(t, err)
}

func TestIntegration_SubscriptionPlans(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	// Seed plans
	DB.MustExec(`INSERT INTO subscription_plans (name, display_name, description, price_monthly, price_yearly,
		max_watch_lists, max_items_per_watch_list, max_alert_rules, max_heatmap_configs, features, is_active)
		VALUES ('free', 'Free', 'Free tier', 0, 0, 3, 10, 5, 1, '{"screener": true}'::jsonb, true)`)
	DB.MustExec(`INSERT INTO subscription_plans (name, display_name, description, price_monthly, price_yearly,
		max_watch_lists, max_items_per_watch_list, max_alert_rules, max_heatmap_configs, features, is_active)
		VALUES ('pro', 'Pro', 'Pro tier', 9.99, 99.99, 10, 50, 50, 10, '{"screener": true, "alerts": true}'::jsonb, true)`)
	DB.MustExec(`INSERT INTO subscription_plans (name, display_name, description, price_monthly, price_yearly,
		max_watch_lists, max_items_per_watch_list, max_alert_rules, max_heatmap_configs, features, is_active)
		VALUES ('legacy', 'Legacy', 'Deprecated', 4.99, 49.99, 5, 20, 20, 5, '{}'::jsonb, false)`)

	// Get all active
	plans, err := GetAllSubscriptionPlans()
	require.NoError(t, err)
	assert.Len(t, plans, 2, "Inactive plans should be excluded")
	assert.Equal(t, "free", plans[0].Name, "Should be sorted by price_monthly ASC")

	// Get by name
	free, err := GetSubscriptionPlanByName("free")
	require.NoError(t, err)
	assert.Equal(t, "Free", free.DisplayName)
	assert.Equal(t, 3, free.MaxWatchLists)

	// Get by ID
	pro, _ := GetSubscriptionPlanByName("pro")
	byID, err := GetSubscriptionPlanByID(pro.ID)
	require.NoError(t, err)
	assert.Equal(t, "pro", byID.Name)

	// Not found
	_, err = GetSubscriptionPlanByName("nonexistent")
	assert.Error(t, err)
	_, err = GetSubscriptionPlanByID("00000000-0000-0000-0000-000000000000")
	assert.Error(t, err)
}

func TestIntegration_UserSubscriptions(t *testing.T) {
	setupTestDB(t)
	cleanTables(t)

	pwHash := "$2a$10$hash"
	user := &models.User{Email: "sub@test.com", PasswordHash: &pwHash, FullName: "Sub User", Timezone: "UTC"}
	require.NoError(t, CreateUser(user))

	// Seed free plan
	DB.MustExec(`INSERT INTO subscription_plans (name, display_name, description, price_monthly, price_yearly,
		max_watch_lists, max_items_per_watch_list, max_alert_rules, max_heatmap_configs, features, is_active)
		VALUES ('free', 'Free', 'Free tier', 0, 0, 3, 10, 5, 1, '{"screener": true}'::jsonb, true)`)
	DB.MustExec(`INSERT INTO subscription_plans (name, display_name, description, price_monthly, price_yearly,
		max_watch_lists, max_items_per_watch_list, max_alert_rules, max_heatmap_configs, features, is_active)
		VALUES ('pro', 'Pro', 'Pro tier', 9.99, 99.99, 10, 50, 50, 10, '{"screener": true, "alerts": true}'::jsonb, true)`)

	// User with no subscription gets free tier defaults
	sub, err := GetUserSubscription(user.ID)
	require.NoError(t, err)
	assert.Equal(t, "free", sub.PlanName)
	assert.Equal(t, 3, sub.MaxWatchLists)

	// Create subscription
	pro, _ := GetSubscriptionPlanByName("pro")
	now := time.Now()
	periodEnd := now.AddDate(0, 1, 0)
	userSub := &models.UserSubscription{
		UserID:             user.ID,
		PlanID:             pro.ID,
		Status:             "active",
		BillingPeriod:      "monthly",
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   &periodEnd,
	}
	err = CreateUserSubscription(userSub)
	require.NoError(t, err)
	assert.NotEmpty(t, userSub.ID)

	// Get with plan details
	withPlan, err := GetUserSubscription(user.ID)
	require.NoError(t, err)
	assert.Equal(t, "pro", withPlan.PlanName)
	assert.Equal(t, 10, withPlan.MaxWatchLists)

	// Cancel
	err = CancelUserSubscription(user.ID)
	require.NoError(t, err)
}

// ========================================
// Helpers
// ========================================

func strPtr(s string) *string {
	return &s
}
