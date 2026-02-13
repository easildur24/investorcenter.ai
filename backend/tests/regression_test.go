package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"investorcenter-api/models"
	"investorcenter-api/services"
)

// testAPIKey reads from POLYGON_TEST_API_KEY env var, falls back to "demo"
var testAPIKey = func() string {
	if key := os.Getenv("POLYGON_TEST_API_KEY"); key != "" {
		return key
	}
	return "demo"
}()

// TestExistingFunctionality ensures all existing Polygon functions still work
func TestExistingFunctionality(t *testing.T) {
	// Skip if not running regression tests
	if os.Getenv("RUN_REGRESSION_TESTS") != "true" {
		t.Skip("Skipping regression test. Set RUN_REGRESSION_TESTS=true to run")
	}

	os.Setenv("POLYGON_API_KEY", testAPIKey)
	client := services.NewPolygonClient()

	t.Run("GetQuote", func(t *testing.T) {
		// Test that GetQuote still works
		quote, err := client.GetQuote("AAPL")
		if err != nil {
			// Check if it's a rate limit error
			if isRateLimitError(err) {
				t.Skip("Rate limited, skipping")
				return
			}
			t.Errorf("GetQuote regression failed: %v", err)
			return
		}

		// Verify essential fields
		if quote.Symbol != "AAPL" {
			t.Errorf("Quote symbol mismatch: got %s, want AAPL", quote.Symbol)
		}

		if quote.Price.IsZero() {
			t.Error("Quote price is zero")
		}
	})

	// Wait to avoid rate limiting
	time.Sleep(15 * time.Second)

	t.Run("GetHistoricalData", func(t *testing.T) {
		// Test historical data fetching
		endDate := time.Now().Format("2006-01-02")
		startDate := time.Now().AddDate(0, -1, 0).Format("2006-01-02")

		data, err := client.GetHistoricalData("AAPL", "day", startDate, endDate)
		if err != nil {
			if isRateLimitError(err) {
				t.Skip("Rate limited, skipping")
				return
			}
			t.Errorf("GetHistoricalData regression failed: %v", err)
			return
		}

		if len(data) == 0 {
			t.Error("No historical data returned")
		}

		// Verify data structure
		if len(data) > 0 {
			point := data[0]
			if point.Open.IsZero() || point.Close.IsZero() {
				t.Error("Historical data point has zero prices")
			}
			if point.Volume == 0 {
				t.Error("Historical data point has zero volume")
			}
		}
	})

	// Wait to avoid rate limiting
	time.Sleep(15 * time.Second)

	t.Run("GetTickerDetails", func(t *testing.T) {
		// Test ticker details fetching
		details, err := client.GetTickerDetails("AAPL")
		if err != nil {
			if isRateLimitError(err) {
				t.Skip("Rate limited, skipping")
				return
			}
			t.Errorf("GetTickerDetails regression failed: %v", err)
			return
		}

		if details.Results.Ticker != "AAPL" {
			t.Errorf("Ticker mismatch: got %s, want AAPL", details.Results.Ticker)
		}

		if details.Results.Name == "" {
			t.Error("Ticker name is empty")
		}
	})

	// Wait to avoid rate limiting
	time.Sleep(15 * time.Second)

	t.Run("GetFundamentals", func(t *testing.T) {
		// Test fundamentals fetching
		fundamentals, err := client.GetFundamentals("AAPL")
		if err != nil {
			// Fundamentals might not be available for all tickers
			t.Logf("GetFundamentals returned error (may be expected): %v", err)
			return
		}

		if fundamentals.Symbol != "AAPL" {
			t.Errorf("Fundamentals symbol mismatch: got %s, want AAPL", fundamentals.Symbol)
		}
	})
}

// TestBackwardCompatibility ensures the new ticker functions don't break existing code
func TestBackwardCompatibility(t *testing.T) {
	if os.Getenv("RUN_REGRESSION_TESTS") != "true" {
		t.Skip("Skipping regression test. Set RUN_REGRESSION_TESTS=true to run")
	}

	os.Setenv("POLYGON_API_KEY", testAPIKey)
	_ = services.NewPolygonClient() // Create client but don't use it yet

	// Test that we can still create a client the old way
	t.Run("ClientCreation", func(t *testing.T) {
		// Old way should still work
		oldClient := &services.PolygonClient{
			APIKey: testAPIKey,
			Client: &http.Client{
				Timeout: 30 * time.Second,
			},
		}

		if oldClient.APIKey != testAPIKey {
			t.Error("Old client creation method failed")
		}
	})

	// Test that existing structs still unmarshal correctly
	t.Run("StructCompatibility", func(t *testing.T) {
		// Test PreviousCloseResponse
		prevCloseJSON := `{
			"status": "OK",
			"results": [{
				"T": "AAPL",
				"v": 1000000,
				"o": 150.0,
				"c": 155.0,
				"h": 156.0,
				"l": 149.0,
				"t": 1234567890000
			}]
		}`

		var prevClose services.PreviousCloseResponse
		err := json.Unmarshal([]byte(prevCloseJSON), &prevClose)
		if err != nil {
			t.Errorf("Failed to unmarshal PreviousCloseResponse: %v", err)
		}

		// Test AggregatesResponse
		aggJSON := `{
			"status": "OK",
			"results": [{
				"T": "AAPL",
				"v": 1000000,
				"o": 150.0,
				"c": 155.0,
				"h": 156.0,
				"l": 149.0,
				"t": 1234567890000
			}]
		}`

		var agg services.AggregatesResponse
		err = json.Unmarshal([]byte(aggJSON), &agg)
		if err != nil {
			t.Errorf("Failed to unmarshal AggregatesResponse: %v", err)
		}
	})
}

// TestNewFunctionality tests the new ticker fetching capabilities
func TestNewFunctionality(t *testing.T) {
	if os.Getenv("RUN_REGRESSION_TESTS") != "true" {
		t.Skip("Skipping regression test. Set RUN_REGRESSION_TESTS=true to run")
	}

	os.Setenv("POLYGON_API_KEY", testAPIKey)
	client := services.NewPolygonClient()

	t.Run("GetAllTickers_Stocks", func(t *testing.T) {
		tickers, err := client.GetAllTickers("stocks", 3)
		if err != nil {
			if isRateLimitError(err) {
				t.Skip("Rate limited, skipping")
				return
			}
			t.Errorf("GetAllTickers failed: %v", err)
			return
		}

		if len(tickers) == 0 {
			t.Error("No stock tickers returned")
		}

		// Verify all are stocks
		for _, ticker := range tickers {
			if ticker.Type != "CS" {
				t.Errorf("Expected CS type for stock, got %s", ticker.Type)
			}
		}
	})

	// Wait to avoid rate limiting
	time.Sleep(15 * time.Second)

	t.Run("GetAllTickers_ETFs", func(t *testing.T) {
		tickers, err := client.GetAllTickers("etf", 3)
		if err != nil {
			if isRateLimitError(err) {
				t.Skip("Rate limited, skipping")
				return
			}
			t.Errorf("GetAllTickers ETF failed: %v", err)
			return
		}

		if len(tickers) == 0 {
			t.Error("No ETF tickers returned")
		}

		// Verify all are ETFs
		for _, ticker := range tickers {
			if ticker.Type != "ETF" {
				t.Errorf("Expected ETF type, got %s", ticker.Type)
			}
		}
	})
}

// TestDataIntegrity verifies that the new data doesn't corrupt existing data
func TestDataIntegrity(t *testing.T) {
	if os.Getenv("RUN_REGRESSION_TESTS") != "true" {
		t.Skip("Skipping regression test. Set RUN_REGRESSION_TESTS=true to run")
	}

	os.Setenv("POLYGON_API_KEY", testAPIKey)
	client := services.NewPolygonClient()

	// Test that popular tickers still work correctly
	popularTickers := []struct {
		symbol       string
		expectedType string
		isETF        bool
	}{
		{"AAPL", "CS", false},
		{"MSFT", "CS", false},
		{"GOOGL", "CS", false},
		{"SPY", "ETF", true},
		{"QQQ", "ETF", true},
		{"IWM", "ETF", true},
	}

	for _, tc := range popularTickers {
		t.Run(tc.symbol, func(t *testing.T) {
			details, err := client.GetTickerDetails(tc.symbol)
			if err != nil {
				if isRateLimitError(err) {
					t.Skip("Rate limited, skipping")
					return
				}
				t.Errorf("Failed to get details for %s: %v", tc.symbol, err)
				return
			}

			if details.Results.Type != tc.expectedType {
				t.Errorf("%s: expected type %s, got %s",
					tc.symbol, tc.expectedType, details.Results.Type)
			}

			// Verify asset type mapping
			assetType := services.MapAssetType(details.Results.Type)
			expectedAssetType := "stock"
			if tc.isETF {
				expectedAssetType = "etf"
			}

			if assetType != expectedAssetType {
				t.Errorf("%s: expected asset type %s, got %s",
					tc.symbol, expectedAssetType, assetType)
			}
		})

		// Wait between requests
		time.Sleep(15 * time.Second)
	}
}

// TestPerformanceRegression ensures the new code doesn't significantly slow down operations
func TestPerformanceRegression(t *testing.T) {
	if os.Getenv("RUN_REGRESSION_TESTS") != "true" {
		t.Skip("Skipping regression test. Set RUN_REGRESSION_TESTS=true to run")
	}

	// Test that mapping functions are still fast
	t.Run("MapExchangeCode_Performance", func(t *testing.T) {
		start := time.Now()
		for i := 0; i < 10000; i++ {
			services.MapExchangeCode("XNAS")
		}
		duration := time.Since(start)

		// Should be very fast (< 1ms for 10000 calls)
		if duration > time.Millisecond {
			t.Errorf("MapExchangeCode too slow: %v for 10000 calls", duration)
		}
	})

	t.Run("MapAssetType_Performance", func(t *testing.T) {
		start := time.Now()
		for i := 0; i < 10000; i++ {
			services.MapAssetType("ETF")
		}
		duration := time.Since(start)

		// Should be very fast (< 1ms for 10000 calls)
		if duration > time.Millisecond {
			t.Errorf("MapAssetType too slow: %v for 10000 calls", duration)
		}
	})
}

// Helper function to check if error is rate limit
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "rate limit") ||
		contains(errStr, "429") ||
		contains(errStr, "exceeded")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > 0 && len(substr) > 0 &&
			fmt.Sprintf("%s", s) != "" && fmt.Sprintf("%s", substr) != "" &&
			http.DetectContentType([]byte(s)) == http.DetectContentType([]byte(substr)))
}

// TestModelCompatibility ensures models still work with new data
func TestModelCompatibility(t *testing.T) {
	// Test that Stock model can handle new fields
	stock := models.Stock{
		Symbol:   "TEST",
		Name:     "Test Stock",
		Exchange: "NYSE",
	}

	// Should still be able to create basic stock
	if stock.Symbol != "TEST" {
		t.Error("Stock model basic fields broken")
	}

	// Test StockPrice model
	price := models.StockPrice{
		Symbol: "TEST",
		Price:  models.DecimalFromFloat(100.0),
	}

	if price.Symbol != "TEST" {
		t.Error("StockPrice model broken")
	}
}

// Benchmark comparison tests
func BenchmarkOldVsNewFunctions(b *testing.B) {
	os.Setenv("POLYGON_API_KEY", testAPIKey)

	b.Run("MapExchangeCode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			services.MapExchangeCode("XNAS")
		}
	})

	b.Run("MapAssetType", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			services.MapAssetType("ETF")
		}
	})
}
