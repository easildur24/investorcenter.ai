package main

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"investorcenter-api/services"
)

const testAPIKey = "zapuIgaTVLJoanfEuimZYQ2xRlZmoU1m"

// Test database connection for integration tests
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// Skip if not in integration test mode
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run")
	}

	// Use test database
	dbHost := getEnvOrDefault("TEST_DB_HOST", "localhost")
	dbPort := getEnvOrDefault("TEST_DB_PORT", "5432")
	dbUser := getEnvOrDefault("TEST_DB_USER", "investorcenter")
	dbPassword := os.Getenv("TEST_DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "test_password"
	}
	dbName := getEnvOrDefault("TEST_DB_NAME", "investorcenter_test")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Create test schema
	if err := createTestSchema(db); err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		// Clean up test data
		db.Exec("DELETE FROM tickers WHERE symbol LIKE 'TEST%'")
		db.Close()
	}

	return db, cleanup
}

func createTestSchema(db *sql.DB) error {
	// Create tickers table if not exists (simplified version for testing)
	schema := `
	CREATE TABLE IF NOT EXISTS tickers (
		id SERIAL PRIMARY KEY,
		symbol VARCHAR(20) UNIQUE NOT NULL,
		name VARCHAR(255) NOT NULL,
		exchange VARCHAR(50),
		sector VARCHAR(100),
		industry VARCHAR(100),
		country VARCHAR(50) DEFAULT 'US',
		currency VARCHAR(3) DEFAULT 'USD',
		market_cap DECIMAL(20,2),
		description TEXT,
		website VARCHAR(255),
		asset_type VARCHAR(20) DEFAULT 'stock',
		cik VARCHAR(20),
		ipo_date DATE,
		logo_url TEXT,
		icon_url TEXT,
		primary_exchange_code VARCHAR(10),
		composite_figi VARCHAR(20),
		share_class_figi VARCHAR(20),
		sic_code VARCHAR(10),
		sic_description VARCHAR(255),
		employees INTEGER,
		phone_number VARCHAR(50),
		address_city VARCHAR(100),
		address_state VARCHAR(50),
		address_postal VARCHAR(20),
		weighted_shares_outstanding BIGINT,
		base_currency_symbol VARCHAR(20),
		base_currency_name VARCHAR(100),
		currency_symbol VARCHAR(20),
		source_feed VARCHAR(100),
		active BOOLEAN DEFAULT true,
		delisted_date DATE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`

	_, err := db.Exec(schema)
	return err
}

func TestTickerExists(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert test ticker
	_, err := db.Exec(`
		INSERT INTO tickers (symbol, name, exchange, asset_type)
		VALUES ('TEST1', 'Test Company 1', 'NYSE', 'stock')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Test existing ticker with matching asset type
	exists, err := tickerExists(db, "TEST1", "stock")
	if err != nil {
		t.Errorf("tickerExists failed: %v", err)
	}
	if !exists {
		t.Error("Expected TEST1 stock to exist")
	}

	// Test existing symbol but different asset type
	exists, err = tickerExists(db, "TEST1", "crypto")
	if err != nil {
		t.Errorf("tickerExists failed: %v", err)
	}
	if exists {
		t.Error("Expected TEST1 crypto to not exist")
	}

	// Test non-existing ticker
	exists, err = tickerExists(db, "NOTEXIST", "stock")
	if err != nil {
		t.Errorf("tickerExists failed: %v", err)
	}
	if exists {
		t.Error("Expected NOTEXIST to not exist")
	}
}

func TestInsertTicker(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create test ticker
	ticker := services.PolygonTicker{
		Ticker:          "TEST2",
		Name:            "Test Company 2",
		Market:          "stocks",
		Locale:          "us",
		Type:            "CS",
		Active:          true,
		CurrencyName:    "usd",
		PrimaryExchange: "XNAS",
		CIK:             "0001234567",
		ListDate:        "2020-01-15",
		MarketCap:       1000000000,
		TotalEmployees:  500,
		SICCode:         "3571",
		SICDescription:  "Electronic Computers",
		HomepageURL:     "https://test.com",
	}

	// Insert ticker
	err := insertTicker(db, ticker)
	if err != nil {
		t.Fatalf("Failed to insert ticker: %v", err)
	}

	// Verify insertion
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM tickers WHERE symbol = $1", "TEST2").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query ticker: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 ticker, found %d", count)
	}

	// Verify data
	var name, assetType string
	var marketCap sql.NullFloat64
	err = db.QueryRow(`
		SELECT name, asset_type, market_cap 
		FROM tickers WHERE symbol = $1
	`, "TEST2").Scan(&name, &assetType, &marketCap)

	if err != nil {
		t.Fatalf("Failed to query ticker details: %v", err)
	}

	if name != "Test Company 2" {
		t.Errorf("Expected name 'Test Company 2', got '%s'", name)
	}

	if assetType != "stock" {
		t.Errorf("Expected asset_type 'stock', got '%s'", assetType)
	}

	if !marketCap.Valid || marketCap.Float64 != 1000000000 {
		t.Errorf("Expected market_cap 1000000000, got %v", marketCap)
	}
}

func TestInsertETF(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create test ETF
	etf := services.PolygonTicker{
		Ticker:          "TESTETF",
		Name:            "Test ETF Fund",
		Market:          "stocks",
		Locale:          "us",
		Type:            "ETF",
		Active:          true,
		CurrencyName:    "usd",
		PrimaryExchange: "ARCX",
		CIK:             "0009876543",
	}

	// Insert ETF
	err := insertTicker(db, etf)
	if err != nil {
		t.Fatalf("Failed to insert ETF: %v", err)
	}

	// Verify ETF type
	var assetType string
	err = db.QueryRow(`
		SELECT asset_type FROM tickers WHERE symbol = $1
	`, "TESTETF").Scan(&assetType)

	if err != nil {
		t.Fatalf("Failed to query ETF: %v", err)
	}

	if assetType != "etf" {
		t.Errorf("Expected asset_type 'etf', got '%s'", assetType)
	}
}

func TestUpdateTicker(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert initial ticker
	_, err := db.Exec(`
		INSERT INTO tickers (symbol, name, exchange, market_cap) 
		VALUES ('TEST3', 'Test Company 3', 'NYSE', 500000000)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Create updated ticker data
	ticker := services.PolygonTicker{
		Ticker:         "TEST3",
		Name:           "Test Company 3 Updated",
		MarketCap:      2000000000,
		TotalEmployees: 1000,
		HomepageURL:    "https://updated.com",
		Active:         true,
	}

	// Update ticker
	err = updateTicker(db, ticker)
	if err != nil {
		t.Fatalf("Failed to update ticker: %v", err)
	}

	// Verify update
	var name string
	var marketCap sql.NullFloat64
	var employees sql.NullInt64
	err = db.QueryRow(`
		SELECT name, market_cap, employees 
		FROM tickers WHERE symbol = $1
	`, "TEST3").Scan(&name, &marketCap, &employees)

	if err != nil {
		t.Fatalf("Failed to query updated ticker: %v", err)
	}

	if name != "Test Company 3 Updated" {
		t.Errorf("Expected updated name, got '%s'", name)
	}

	if !marketCap.Valid || marketCap.Float64 != 2000000000 {
		t.Errorf("Expected market_cap 2000000000, got %v", marketCap)
	}

	if !employees.Valid || employees.Int64 != 1000 {
		t.Errorf("Expected employees 1000, got %v", employees)
	}
}

func TestShouldUpdate(t *testing.T) {
	tests := []struct {
		ticker   services.PolygonTicker
		expected bool
	}{
		{
			ticker: services.PolygonTicker{
				MarketCap: 1000000,
			},
			expected: true,
		},
		{
			ticker: services.PolygonTicker{
				TotalEmployees: 100,
			},
			expected: true,
		},
		{
			ticker: services.PolygonTicker{
				HomepageURL: "https://example.com",
			},
			expected: true,
		},
		{
			ticker: services.PolygonTicker{
				// No significant data
			},
			expected: false,
		},
	}

	for i, test := range tests {
		result := shouldUpdate(test.ticker)
		if result != test.expected {
			t.Errorf("Test %d: shouldUpdate() = %v, expected %v",
				i, result, test.expected)
		}
	}
}

func TestMapSICToSector(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TestMapSICToSector in short mode")
	}
	tests := []struct {
		sicCode  string
		sicDesc  string
		expected string
	}{
		{"0100", "Agricultural Production", "Agriculture"},
		{"1000", "Metal Mining", "Mining"},
		{"1500", "Construction", "Construction"},
		{"2000", "Food Products", "Manufacturing"},
		{"3571", "Electronic Computers", "Manufacturing"},
		{"4000", "Railroad Transportation", "Transportation"},
		{"5000", "Wholesale Trade", "Wholesale Trade"},
		{"5200", "Retail Trade", "Retail Trade"},
		{"6000", "Banking", "Finance"},
		{"7000", "Hotels", "Services"},
		{"8000", "Health Services", "Healthcare"},
		{"9100", "Government", "Public Administration"},
		{"", "Software Technology", ""},
		{"", "Pharmaceutical", ""},
		{"9999", "Unknown", "Public Administration"},
	}

	for _, test := range tests {
		result := mapSICToSector(test.sicCode, test.sicDesc)
		if result != test.expected {
			t.Errorf("mapSICToSector(%s, %s) = %s, expected %s",
				test.sicCode, test.sicDesc, result, test.expected)
		}
	}
}

func TestNullIfEmpty(t *testing.T) {
	// Test empty string
	result := nullIfEmpty("")
	if result != nil {
		t.Error("Expected nil for empty string")
	}

	// Test non-empty string
	result = nullIfEmpty("test")
	if result == nil || *result != "test" {
		t.Error("Expected 'test' for non-empty string")
	}
}
