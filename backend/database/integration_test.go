package database

import (
	"investorcenter-api/models"
	"testing"

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
