package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"investorcenter-api/services"
)

// Command line flags
var (
	assetType  = flag.String("type", "all", "Asset type to import: stocks, etf, crypto, indices, all_equities, or all")
	limit      = flag.Int("limit", 0, "Limit number of tickers to import (0 = ALL tickers)")
	dryRun     = flag.Bool("dry-run", false, "Preview what would be imported without actually importing")
	verbose    = flag.Bool("verbose", false, "Enable verbose logging")
	updateOnly = flag.Bool("update-only", false, "Only update existing tickers, don't insert new ones")
)

func main() {
	flag.Parse()

	// Setup database connection
	db, err := setupDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create Polygon client
	polygonClient := services.NewPolygonClient()
	apiKey := os.Getenv("POLYGON_API_KEY")
	if apiKey == "" || apiKey == "demo" {
		log.Println("Warning: POLYGON_API_KEY not set or using demo key. API calls may fail.")
		log.Println("Set your API key with: export POLYGON_API_KEY=your_key_here")
	}

	// Import tickers based on type
	if *assetType == "all" {
		// Import all asset types (crypto excluded - use CoinGecko)
		importAllTypes(db, polygonClient)
	} else if *assetType == "crypto" {
		// Reject crypto imports - direct to CoinGecko
		log.Println("❌ Crypto import from Polygon is no longer supported")
		log.Println("💡 Please use CoinGecko API for cryptocurrency data")
		log.Println("📚 See docs/CRYPTO_MIGRATION_EXECUTION_PLAN.md for migration guide")
		log.Fatalf("Crypto import aborted - use CoinGecko instead")
	} else {
		// Import specific asset type
		if err := importTickers(db, polygonClient, *assetType); err != nil {
			log.Fatalf("Failed to import %s tickers: %v", *assetType, err)
		}
	}

	// Print summary
	printSummary(db)
}

func setupDatabase() (*sql.DB, error) {
	// Get database connection details from environment
	dbHost := getEnvOrDefault("DB_HOST", "localhost")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "investorcenter")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := getEnvOrDefault("DB_NAME", "investorcenter_db")

	if dbPassword == "" {
		return nil, fmt.Errorf("DB_PASSWORD environment variable is required")
	}

	// Build connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Println("✅ Connected to database successfully")
	return db, nil
}

func importAllTypes(db *sql.DB, client *services.PolygonClient) {
	// Note: crypto removed - use CoinGecko for cryptocurrency data
	types := []string{"stocks", "etf", "indices"}

	for _, assetType := range types {
		log.Printf("\n📦 Importing %s...\n", assetType)
		if err := importTickers(db, client, assetType); err != nil {
			log.Printf("Warning: Failed to import %s: %v", assetType, err)
		}

		// Add delay between different asset types to avoid rate limiting
		if assetType != types[len(types)-1] {
			log.Println("⏳ Waiting 2 seconds before next asset type...")
			time.Sleep(2 * time.Second)
		}
	}
}

func importTickers(db *sql.DB, client *services.PolygonClient, assetType string) error {
	// Reject crypto imports - direct to CoinGecko
	if assetType == "crypto" {
		return fmt.Errorf("❌ Crypto import from Polygon is no longer supported. Please use CoinGecko API for cryptocurrency data")
	}

	log.Printf("🔍 Fetching %s tickers from Polygon API (this will paginate automatically)...", assetType)

	// Fetch ALL tickers from Polygon API (it will paginate automatically)
	// Pass 0 as limit to fetch everything, or *limit to fetch specific amount
	tickers, err := client.GetAllTickers(assetType, *limit)
	if err != nil {
		return fmt.Errorf("failed to fetch tickers: %w", err)
	}

	log.Printf("📊 Successfully fetched %d %s tickers", len(tickers), assetType)

	if *dryRun {
		log.Println("🔍 DRY RUN MODE - Not inserting into database")
		// Just print first 10 for preview
		for i, ticker := range tickers {
			if i >= 10 {
				log.Printf("... and %d more", len(tickers)-10)
				break
			}
			log.Printf("  %s - %s (Type: %s, Exchange: %s)",
				ticker.Ticker, ticker.Name, ticker.Type, services.MapExchangeCode(ticker.PrimaryExchange))
		}
		return nil
	}

	// Process all tickers
	log.Printf("📥 Processing %d tickers...", len(tickers))

	inserted := 0
	updated := 0
	skipped := 0
	errors := 0

	for i, ticker := range tickers {
		if *verbose && i%100 == 0 && i > 0 {
			log.Printf("Progress: %d/%d (inserted: %d, updated: %d, skipped: %d, errors: %d)",
				i, len(tickers), inserted, updated, skipped, errors)
		}

		// Map asset type for the check
		assetType := services.MapAssetType(ticker.Type)

		// Check if ticker exists (with same asset type)
		exists, err := tickerExists(db, ticker.Ticker, assetType)
		if err != nil {
			if *verbose {
				log.Printf("Error checking ticker %s: %v", ticker.Ticker, err)
			}
			errors++
			continue
		}

		if exists {
			if *updateOnly || shouldUpdate(ticker) {
				if err := updateTicker(db, ticker); err != nil {
					if *verbose {
						log.Printf("Error updating ticker %s: %v", ticker.Ticker, err)
					}
					errors++
				} else {
					updated++
				}
			} else {
				skipped++
			}
		} else {
			if !*updateOnly {
				if err := insertTicker(db, ticker); err != nil {
					if *verbose {
						log.Printf("Error inserting ticker %s: %v", ticker.Ticker, err)
					}
					errors++
				} else {
					inserted++
				}
			} else {
				skipped++
			}
		}
	}

	log.Printf("✅ Import complete: %d inserted, %d updated, %d skipped, %d errors",
		inserted, updated, skipped, errors)

	return nil
}

func tickerExists(db *sql.DB, symbol string, assetType string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM tickers WHERE symbol = $1 AND asset_type = $2", symbol, assetType).Scan(&count)
	return count > 0, err
}

func shouldUpdate(ticker services.PolygonTicker) bool {
	// Update if we have new data like market cap, employees, etc.
	return ticker.MarketCap > 0 || ticker.TotalEmployees > 0 || ticker.HomepageURL != ""
}

func insertTicker(db *sql.DB, ticker services.PolygonTicker) error {
	query := `
		INSERT INTO tickers (
			symbol, name, exchange, sector, industry, country, currency,
			market_cap, description, website, asset_type, cik, ipo_date,
			logo_url, primary_exchange_code, composite_figi, share_class_figi,
			sic_code, sic_description, employees, phone_number,
			weighted_shares_outstanding, base_currency_symbol, base_currency_name,
			currency_symbol, source_feed, active
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27
		) ON CONFLICT (symbol, asset_type) DO UPDATE SET
			name = EXCLUDED.name,
			exchange = COALESCE(EXCLUDED.exchange, tickers.exchange),
			market_cap = COALESCE(EXCLUDED.market_cap, tickers.market_cap),
			website = COALESCE(EXCLUDED.website, tickers.website),
			updated_at = NOW()`

	// Map values
	exchange := services.MapExchangeCode(ticker.PrimaryExchange)
	assetType := services.MapAssetType(ticker.Type)
	country := "US"
	if ticker.Locale == "global" || ticker.Market == "crypto" {
		country = "GLOBAL"
	}

	// Parse IPO date if available
	var ipoDate *time.Time
	if ticker.ListDate != "" {
		if parsed, err := time.Parse("2006-01-02", ticker.ListDate); err == nil {
			ipoDate = &parsed
		}
	}

	// Handle nullable fields
	var marketCap *float64
	if ticker.MarketCap > 0 {
		marketCap = &ticker.MarketCap
	}

	var employees *int
	if ticker.TotalEmployees > 0 {
		employees = &ticker.TotalEmployees
	}

	var sharesOutstanding *int64
	if ticker.WeightedSharesOutstanding > 0 {
		so := int64(ticker.WeightedSharesOutstanding)
		sharesOutstanding = &so
	}

	// Determine sector/industry from SIC description
	sector := mapSICToSector(ticker.SICCode, ticker.SICDescription)
	industry := ticker.SICDescription
	if assetType == "etf" {
		sector = "ETF"
		industry = "Exchange Traded Fund"
	} else if assetType == "crypto" {
		sector = "Cryptocurrency"
		industry = "Digital Currency"
	} else if assetType == "index" {
		sector = "Index"
		industry = "Market Index"
	}

	_, err := db.Exec(query,
		ticker.Ticker,
		ticker.Name,
		exchange,
		sector,
		industry,
		country,
		strings.ToUpper(ticker.CurrencyName),
		marketCap,
		"", // description - not provided by tickers endpoint
		ticker.HomepageURL,
		assetType,
		nullIfEmpty(ticker.CIK),
		ipoDate,
		"", // logo_url - would need separate call
		nullIfEmpty(ticker.PrimaryExchange),
		nullIfEmpty(ticker.CompositeFigi),
		nullIfEmpty(ticker.ShareClassFigi),
		nullIfEmpty(ticker.SICCode),
		nullIfEmpty(ticker.SICDescription),
		employees,
		nullIfEmpty(ticker.PhoneNumber),
		sharesOutstanding,
		nullIfEmpty(ticker.BaseCurrencySymbol),
		nullIfEmpty(ticker.BaseCurrencyName),
		nullIfEmpty(ticker.CurrencySymbol),
		nullIfEmpty(ticker.SourceFeed),
		ticker.Active,
	)

	return err
}

func updateTicker(db *sql.DB, ticker services.PolygonTicker) error {
	assetType := services.MapAssetType(ticker.Type)

	query := `
		UPDATE tickers SET
			name = $2,
			market_cap = COALESCE($3, market_cap),
			website = COALESCE($4, website),
			cik = COALESCE($5, cik),
			sic_code = COALESCE($6, sic_code),
			sic_description = COALESCE($7, sic_description),
			employees = COALESCE($8, employees),
			phone_number = COALESCE($9, phone_number),
			weighted_shares_outstanding = COALESCE($10, weighted_shares_outstanding),
			active = $11,
			updated_at = NOW()
		WHERE symbol = $1 AND asset_type = $12`

	var marketCap *float64
	if ticker.MarketCap > 0 {
		marketCap = &ticker.MarketCap
	}

	var employees *int
	if ticker.TotalEmployees > 0 {
		employees = &ticker.TotalEmployees
	}

	var sharesOutstanding *int64
	if ticker.WeightedSharesOutstanding > 0 {
		so := int64(ticker.WeightedSharesOutstanding)
		sharesOutstanding = &so
	}

	_, err := db.Exec(query,
		ticker.Ticker,
		ticker.Name,
		marketCap,
		nullIfEmpty(ticker.HomepageURL),
		nullIfEmpty(ticker.CIK),
		nullIfEmpty(ticker.SICCode),
		nullIfEmpty(ticker.SICDescription),
		employees,
		nullIfEmpty(ticker.PhoneNumber),
		sharesOutstanding,
		ticker.Active,
		assetType,
	)

	return err
}

func printSummary(db *sql.DB) {
	log.Println("\n📊 Database Summary:")

	// Query counts by asset type
	query := `
		SELECT asset_type, COUNT(*) as count
		FROM tickers
		WHERE asset_type IS NOT NULL
		GROUP BY asset_type
		ORDER BY count DESC`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error querying summary: %v", err)
		return
	}
	defer rows.Close()

	total := 0
	for rows.Next() {
		var assetType string
		var count int
		if err := rows.Scan(&assetType, &count); err != nil {
			continue
		}
		log.Printf("  %s: %d", assetType, count)
		total += count
	}

	log.Printf("  Total: %d", total)
}

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func mapSICToSector(sicCode, sicDesc string) string {
	if sicCode == "" {
		return ""
	}

	// Simple mapping based on SIC code ranges
	code := sicCode[:2] // First 2 digits determine major group
	switch code {
	case "01", "02", "07", "08", "09":
		return "Agriculture"
	case "10", "11", "12", "13", "14":
		return "Mining"
	case "15", "16", "17":
		return "Construction"
	case "20", "21", "22", "23", "24", "25", "26", "27", "28", "29", "30", "31", "32", "33", "34", "35", "36", "37", "38", "39":
		return "Manufacturing"
	case "40", "41", "42", "43", "44", "45", "46", "47", "48", "49":
		return "Transportation"
	case "50", "51":
		return "Wholesale Trade"
	case "52", "53", "54", "55", "56", "57", "58", "59":
		return "Retail Trade"
	case "60", "61", "62", "63", "64", "65", "66", "67":
		return "Finance"
	case "70", "72", "73", "75", "76", "78", "79":
		return "Services"
	case "80", "82", "83", "86", "87", "89":
		return "Healthcare"
	case "91", "92", "93", "94", "95", "96", "97", "99":
		return "Public Administration"
	default:
		if strings.Contains(strings.ToLower(sicDesc), "technology") || strings.Contains(strings.ToLower(sicDesc), "software") {
			return "Technology"
		}
		if strings.Contains(strings.ToLower(sicDesc), "pharmaceutical") || strings.Contains(strings.ToLower(sicDesc), "biotech") {
			return "Healthcare"
		}
		return "Other"
	}
}
