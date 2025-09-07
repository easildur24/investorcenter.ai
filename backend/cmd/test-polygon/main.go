package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"investorcenter-api/services"
)

func main() {
	// Create Polygon client
	client := services.NewPolygonClient()
	
	// Check API key
	apiKey := os.Getenv("POLYGON_API_KEY")
	if apiKey == "" {
		apiKey = "zapuIgaTVLJoanfEuimZYQ2xRlZmoU1m" // Use provided key for testing
		os.Setenv("POLYGON_API_KEY", apiKey)
		client = services.NewPolygonClient() // Recreate with key
	}
	
	fmt.Println("🧪 Testing Polygon Ticker API Integration")
	fmt.Println("=========================================")
	fmt.Printf("API Key: %s...%s\n\n", apiKey[:10], apiKey[len(apiKey)-4:])
	
	// Test 1: Fetch a few stocks
	fmt.Println("1️⃣ Fetching US Stocks (limit 5)...")
	stocks, err := client.GetAllTickers("stocks", 5)
	if err != nil {
		log.Printf("❌ Error fetching stocks: %v", err)
	} else {
		fmt.Printf("✅ Found %d stocks:\n", len(stocks))
		for _, ticker := range stocks {
			fmt.Printf("   %s - %s (Exchange: %s, Type: %s)\n", 
				ticker.Ticker, ticker.Name, 
				services.MapExchangeCode(ticker.PrimaryExchange),
				services.MapAssetType(ticker.Type))
		}
	}
	
	fmt.Println()
	
	// Test 2: Fetch ETFs
	fmt.Println("2️⃣ Fetching ETFs (limit 5)...")
	etfs, err := client.GetAllTickers("etf", 5)
	if err != nil {
		log.Printf("❌ Error fetching ETFs: %v", err)
	} else {
		fmt.Printf("✅ Found %d ETFs:\n", len(etfs))
		for _, ticker := range etfs {
			fmt.Printf("   %s - %s (Type: %s -> %s)\n", 
				ticker.Ticker, ticker.Name, ticker.Type,
				services.MapAssetType(ticker.Type))
		}
	}
	
	fmt.Println()
	
	// Test 3: Check specific ticker details
	fmt.Println("3️⃣ Checking specific tickers...")
	testTickers := []string{"AAPL", "SPY", "QQQ"}
	
	for _, symbol := range testTickers {
		details, err := client.GetTickerDetails(symbol)
		if err != nil {
			log.Printf("❌ Error fetching %s: %v", symbol, err)
		} else {
			fmt.Printf("✅ %s:\n", symbol)
			fmt.Printf("   Name: %s\n", details.Results.Name)
			fmt.Printf("   Type: %s -> %s\n", details.Results.Type, services.MapAssetType(details.Results.Type))
			fmt.Printf("   Market: %s\n", details.Results.Market)
			fmt.Printf("   Exchange: %s -> %s\n", details.Results.PrimaryExch, services.MapExchangeCode(details.Results.PrimaryExch))
			if details.Results.CIK != "" {
				fmt.Printf("   CIK: %s\n", details.Results.CIK)
			}
			if details.Results.HomepageURL != "" {
				fmt.Printf("   Website: %s\n", details.Results.HomepageURL)
			}
		}
		fmt.Println()
	}
	
	// Test 4: Show ticker counts
	fmt.Println("4️⃣ Fetching ticker counts...")
	types := map[string]string{
		"stocks":      "US Stocks",
		"etf":         "ETFs",
		"crypto":      "Crypto",
		"indices":     "Indices",
		"all_equities": "All Equities (Stocks + ETFs)",
	}
	
	for key, label := range types {
		// Just fetch 1 to get the count
		tickers, err := client.GetAllTickers(key, 1)
		if err != nil {
			fmt.Printf("   %s: Error\n", label)
		} else {
			// The actual count would be in the response, but we can't get it without pagination
			// This is just to test the connection works
			if len(tickers) > 0 {
				fmt.Printf("   %s: ✅ Working (got %s)\n", label, tickers[0].Ticker)
			} else {
				fmt.Printf("   %s: No data\n", label)
			}
		}
	}
	
	fmt.Println("\n✅ Test complete!")
	
	// Show sample JSON for documentation
	if len(stocks) > 0 {
		fmt.Println("\n📋 Sample Stock JSON:")
		data, _ := json.MarshalIndent(stocks[0], "", "  ")
		fmt.Println(string(data))
	}
}