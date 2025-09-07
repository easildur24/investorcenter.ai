package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// Test API key - can be used for testing
const testAPIKey = "zapuIgaTVLJoanfEuimZYQ2xRlZmoU1m"

func TestNewPolygonClient(t *testing.T) {
	// Test with environment variable
	os.Setenv("POLYGON_API_KEY", testAPIKey)
	client := NewPolygonClient()
	
	if client.APIKey != testAPIKey {
		t.Errorf("Expected API key %s, got %s", testAPIKey, client.APIKey)
	}
	
	// Test without environment variable (should use demo)
	os.Unsetenv("POLYGON_API_KEY")
	client = NewPolygonClient()
	
	if client.APIKey != "demo" {
		t.Errorf("Expected demo API key when env not set, got %s", client.APIKey)
	}
	
	// Restore for other tests
	os.Setenv("POLYGON_API_KEY", testAPIKey)
}

func TestMapExchangeCode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"XNAS", "NASDAQ"},
		{"XNYS", "NYSE"},
		{"ARCX", "NYSE ARCA"},
		{"BATS", "CBOE BZX"},
		{"UNKNOWN", "UNKNOWN"}, // Should return as-is if not mapped
	}
	
	for _, test := range tests {
		result := MapExchangeCode(test.input)
		if result != test.expected {
			t.Errorf("MapExchangeCode(%s) = %s, expected %s", 
				test.input, result, test.expected)
		}
	}
}

func TestMapAssetType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"CS", "stock"},
		{"ETF", "etf"},
		{"ETN", "etn"},
		{"FUND", "fund"},
		{"PFD", "preferred"},
		{"ADRC", "adr"},
		{"IX", "index"},
		{"X:BTCUSD", "crypto"},
		{"I:SPX", "index"},
		{"UNKNOWN", "other"},
	}
	
	for _, test := range tests {
		result := MapAssetType(test.input)
		if result != test.expected {
			t.Errorf("MapAssetType(%s) = %s, expected %s", 
				test.input, result, test.expected)
		}
	}
}

func TestGetAllTickers_MockServer(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check API key
		if r.URL.Query().Get("apikey") != testAPIKey {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "ERROR",
				"error":  "Invalid API key",
			})
			return
		}
		
		// Return mock response based on asset type
		assetType := ""
		if r.URL.Query().Get("type") == "CS" {
			assetType = "stocks"
		} else if r.URL.Query().Get("type") == "ETF" {
			assetType = "etf"
		}
		
		response := PolygonTickersResponse{
			Status: "OK",
			Count:  2,
			Results: []PolygonTicker{
				{
					Ticker:          "TEST1",
					Name:            "Test Company 1",
					Market:          "stocks",
					Type:            "CS",
					Active:          true,
					PrimaryExchange: "XNAS",
				},
				{
					Ticker:          "TEST2",
					Name:            "Test Company 2",
					Market:          "stocks",
					Type:            "CS",
					Active:          true,
					PrimaryExchange: "XNYS",
				},
			},
		}
		
		if assetType == "etf" {
			response.Results[0].Type = "ETF"
			response.Results[0].Name = "Test ETF 1"
			response.Results[1].Type = "ETF"
			response.Results[1].Name = "Test ETF 2"
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	// Create client with mock server
	client := &PolygonClient{
		APIKey: testAPIKey,
		Client: &http.Client{Timeout: 5 * time.Second},
	}
	
	// Override base URL for testing
	originalURL := PolygonBaseURL
	PolygonBaseURL = server.URL
	defer func() { PolygonBaseURL = originalURL }()
	
	// Test fetching stocks
	tickers, err := client.GetAllTickers("stocks", 10)
	if err != nil {
		t.Fatalf("GetAllTickers failed: %v", err)
	}
	
	if len(tickers) != 2 {
		t.Errorf("Expected 2 tickers, got %d", len(tickers))
	}
	
	if tickers[0].Ticker != "TEST1" {
		t.Errorf("Expected ticker TEST1, got %s", tickers[0].Ticker)
	}
}

func TestGetAllTickers_RealAPI(t *testing.T) {
	// Skip this test in CI/automated environments
	if os.Getenv("CI") == "true" || os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping real API test in CI environment")
	}
	
	// Use real API key
	os.Setenv("POLYGON_API_KEY", testAPIKey)
	client := NewPolygonClient()
	
	// Test fetching a small number of stocks
	t.Run("FetchStocks", func(t *testing.T) {
		tickers, err := client.GetAllTickers("stocks", 5)
		if err != nil {
			t.Fatalf("Failed to fetch stocks: %v", err)
		}
		
		if len(tickers) == 0 {
			t.Error("No stocks returned")
		}
		
		// Verify first ticker has required fields
		if len(tickers) > 0 {
			ticker := tickers[0]
			if ticker.Ticker == "" {
				t.Error("Ticker symbol is empty")
			}
			if ticker.Name == "" {
				t.Error("Ticker name is empty")
			}
			if ticker.Type != "CS" {
				t.Errorf("Expected type CS for stock, got %s", ticker.Type)
			}
		}
	})
	
	// Add delay to avoid rate limiting
	time.Sleep(15 * time.Second)
	
	// Test fetching ETFs
	t.Run("FetchETFs", func(t *testing.T) {
		tickers, err := client.GetAllTickers("etf", 5)
		if err != nil {
			t.Fatalf("Failed to fetch ETFs: %v", err)
		}
		
		if len(tickers) == 0 {
			t.Error("No ETFs returned")
		}
		
		// Verify ETF type
		if len(tickers) > 0 {
			ticker := tickers[0]
			if ticker.Type != "ETF" {
				t.Errorf("Expected type ETF, got %s", ticker.Type)
			}
		}
	})
}

func TestGetTickerDetails_RealAPI(t *testing.T) {
	// Skip this test in CI/automated environments
	if os.Getenv("CI") == "true" || os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping real API test in CI environment")
	}
	
	os.Setenv("POLYGON_API_KEY", testAPIKey)
	client := NewPolygonClient()
	
	testCases := []struct {
		symbol       string
		expectedType string
		shouldExist  bool
	}{
		{"AAPL", "CS", true},    // Stock
		{"SPY", "ETF", true},    // ETF
		{"INVALID123", "", false}, // Non-existent
	}
	
	for _, tc := range testCases {
		t.Run(tc.symbol, func(t *testing.T) {
			details, err := client.GetTickerDetails(tc.symbol)
			
			if tc.shouldExist {
				if err != nil {
					t.Errorf("Failed to get details for %s: %v", tc.symbol, err)
					return
				}
				
				if details.Results.Ticker != tc.symbol {
					t.Errorf("Expected ticker %s, got %s", tc.symbol, details.Results.Ticker)
				}
				
				if tc.expectedType != "" && details.Results.Type != tc.expectedType {
					t.Errorf("Expected type %s for %s, got %s", 
						tc.expectedType, tc.symbol, details.Results.Type)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for invalid ticker %s, but got none", tc.symbol)
				}
			}
		})
		
		// Add delay between requests to avoid rate limiting
		time.Sleep(15 * time.Second)
	}
}

func TestPolygonTickerSerialization(t *testing.T) {
	// Test that our struct properly serializes/deserializes JSON
	jsonData := `{
		"ticker": "AAPL",
		"name": "Apple Inc.",
		"market": "stocks",
		"locale": "us",
		"type": "CS",
		"active": true,
		"currency_name": "usd",
		"cik": "0000320193",
		"primary_exchange": "XNAS",
		"market_cap": 2950000000000,
		"list_date": "1980-12-12"
	}`
	
	var ticker PolygonTicker
	err := json.Unmarshal([]byte(jsonData), &ticker)
	if err != nil {
		t.Fatalf("Failed to unmarshal ticker: %v", err)
	}
	
	if ticker.Ticker != "AAPL" {
		t.Errorf("Expected ticker AAPL, got %s", ticker.Ticker)
	}
	
	if ticker.MarketCap != 2950000000000 {
		t.Errorf("Expected market cap 2950000000000, got %f", ticker.MarketCap)
	}
	
	if ticker.ListDate != "1980-12-12" {
		t.Errorf("Expected list date 1980-12-12, got %s", ticker.ListDate)
	}
}

// Benchmark tests
func BenchmarkMapExchangeCode(b *testing.B) {
	codes := []string{"XNAS", "XNYS", "ARCX", "UNKNOWN"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MapExchangeCode(codes[i%len(codes)])
	}
}

func BenchmarkMapAssetType(b *testing.B) {
	types := []string{"CS", "ETF", "ETN", "UNKNOWN"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MapAssetType(types[i%len(types)])
	}
}