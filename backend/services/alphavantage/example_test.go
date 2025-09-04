package alphavantage

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestIntegrationAlphaVantage tests real API calls (requires API key)
// Run with: go test -v -run TestIntegration ./backend/services/alphavantage/
func TestIntegrationAlphaVantage(t *testing.T) {
	// Skip if not explicitly running integration tests
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run")
	}

	apiKey := os.Getenv("ALPHA_VANTAGE_API_KEY")
	if apiKey == "" {
		apiKey = "6MVGMJ4FCAGF2ONU" // Fallback to provided key for testing
	}

	client := NewClient(apiKey)
	ctx := context.Background()

	t.Run("GetGlobalQuote", func(t *testing.T) {
		quote, err := client.GetGlobalQuote(ctx, "AAPL")
		if err != nil {
			t.Fatalf("Failed to get quote for AAPL: %v", err)
		}

		fmt.Printf("AAPL Quote:\n")
		fmt.Printf("  Symbol: %s\n", quote.Symbol)
		fmt.Printf("  Price: $%.2f\n", quote.Price)
		fmt.Printf("  Volume: %d\n", quote.Volume)
		fmt.Printf("  Change: $%.2f (%.2f%%)\n", quote.Change, quote.ChangePercent)
		fmt.Printf("  Day Range: $%.2f - $%.2f\n", quote.Low, quote.High)
		fmt.Printf("  Previous Close: $%.2f\n", quote.PreviousClose)

		// Basic validations
		if quote.Symbol != "AAPL" {
			t.Errorf("Expected symbol AAPL, got %s", quote.Symbol)
		}

		if quote.Price <= 0 {
			t.Error("Price should be positive")
		}

		if quote.Volume < 0 {
			t.Error("Volume should not be negative")
		}

		// Wait a bit to respect rate limits
		time.Sleep(15 * time.Second)
	})

	t.Run("GetBatchQuotes", func(t *testing.T) {
		symbols := []string{"MSFT", "GOOGL", "AMZN"}
		quotes, err := client.GetBatchQuotes(ctx, symbols)

		if err != nil {
			t.Fatalf("Failed to get batch quotes: %v", err)
		}

		fmt.Printf("\nBatch Quotes:\n")
		for _, quote := range quotes {
			fmt.Printf("  %s: $%.2f (Volume: %d)\n",
				quote.Symbol, quote.Price, quote.Volume)
		}

		if len(quotes) == 0 {
			t.Error("Expected at least one quote")
		}

		// Verify all requested symbols are present
		symbolMap := make(map[string]bool)
		for _, quote := range quotes {
			symbolMap[quote.Symbol] = true
		}

		for _, symbol := range symbols {
			if !symbolMap[symbol] {
				t.Logf("Warning: Symbol %s not found in response", symbol)
			}
		}
	})

	t.Run("RateLimiterStatus", func(t *testing.T) {
		minuteUsed, dayUsed, minuteLimit, dayLimit := client.rateLimiter.GetStatus()

		fmt.Printf("\nRate Limit Status:\n")
		fmt.Printf("  Minute: %d/%d requests used\n", minuteUsed, minuteLimit)
		fmt.Printf("  Day: %d/%d requests used\n", dayUsed, dayLimit)

		if minuteUsed > minuteLimit {
			t.Error("Minute usage should not exceed limit")
		}

		if dayUsed > dayLimit {
			t.Error("Day usage should not exceed limit")
		}
	})
}
