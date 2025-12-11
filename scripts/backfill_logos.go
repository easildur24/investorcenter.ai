package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

const (
	polygonBaseURL = "https://api.polygon.io"
	batchSize      = 100
	rateLimitDelay = 250 * time.Millisecond // 4 requests per second to stay under limits
)

type TickerDetailsResponse struct {
	Status  string `json:"status"`
	Results struct {
		Ticker   string `json:"ticker"`
		Branding struct {
			LogoURL string `json:"logo_url"`
			IconURL string `json:"icon_url"`
		} `json:"branding"`
	} `json:"results"`
}

func main() {
	// Get environment variables
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "investorcenter")
	dbPassword := getEnv("DB_PASSWORD", "password123")
	dbName := getEnv("DB_NAME", "investorcenter_db")
	polygonAPIKey := os.Getenv("POLYGON_API_KEY")

	if polygonAPIKey == "" {
		log.Fatal("POLYGON_API_KEY environment variable is required")
	}

	// Connect to database
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database")

	// Get tickers that need logo URLs (common stocks and preferred stocks)
	rows, err := db.Query(`
		SELECT symbol FROM tickers
		WHERE (logo_url IS NULL OR logo_url = '')
		AND asset_type IN ('CS', 'PFD', 'stock')
		ORDER BY symbol
	`)
	if err != nil {
		log.Fatalf("Failed to query tickers: %v", err)
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		symbols = append(symbols, symbol)
	}

	log.Printf("Found %d tickers without logos", len(symbols))

	// Process each ticker
	client := &http.Client{Timeout: 10 * time.Second}
	updated := 0
	failed := 0

	for i, symbol := range symbols {
		logoURL, err := fetchLogoURL(client, symbol, polygonAPIKey)
		if err != nil {
			log.Printf("[%d/%d] Failed to fetch logo for %s: %v", i+1, len(symbols), symbol, err)
			failed++
			time.Sleep(rateLimitDelay)
			continue
		}

		if logoURL == "" {
			log.Printf("[%d/%d] No logo available for %s", i+1, len(symbols), symbol)
			time.Sleep(rateLimitDelay)
			continue
		}

		// Update database
		_, err = db.Exec("UPDATE tickers SET logo_url = $1 WHERE symbol = $2", logoURL, symbol)
		if err != nil {
			log.Printf("[%d/%d] Failed to update logo for %s: %v", i+1, len(symbols), symbol, err)
			failed++
		} else {
			log.Printf("[%d/%d] Updated logo for %s", i+1, len(symbols), symbol)
			updated++
		}

		time.Sleep(rateLimitDelay)
	}

	log.Printf("Completed: %d updated, %d failed out of %d total", updated, failed, len(symbols))
}

func fetchLogoURL(client *http.Client, symbol, apiKey string) (string, error) {
	url := fmt.Sprintf("%s/v3/reference/tickers/%s?apiKey=%s", polygonBaseURL, symbol, apiKey)

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return "", nil // Ticker not found
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var details TickerDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if details.Status != "OK" {
		return "", fmt.Errorf("API status: %s", details.Status)
	}

	return details.Results.Branding.LogoURL, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
