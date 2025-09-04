package alphavantage

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Mock responses for testing
const mockGlobalQuoteResponse = `{
	"Global Quote": {
		"01. symbol": "AAPL",
		"02. open": "150.00",
		"03. high": "152.00",
		"04. low": "149.50",
		"05. price": "151.25",
		"06. volume": "75000000",
		"07. latest trading day": "2024-01-15",
		"08. previous close": "149.80",
		"09. change": "1.45",
		"10. change percent": "0.97%"
	}
}`

const mockBatchQuotesResponse = `{
	"Meta Data": {
		"1. Information": "Batch Stock Market Quotes",
		"2. Notes": "IEX Real-Time Price",
		"3. Time Zone": "US/Eastern"
	},
	"Stock Quotes": [
		{
			"1. symbol": "AAPL",
			"2. price": "151.25",
			"3. volume": "75000000",
			"4. timestamp": "2024-01-15 16:00:00"
		},
		{
			"1. symbol": "GOOGL",
			"2. price": "2750.50",
			"3. volume": "25000000",
			"4. timestamp": "2024-01-15 16:00:00"
		}
	]
}`

const mockErrorResponse = `{
	"Note": "Thank you for using Alpha Vantage! Our standard API call frequency is 5 calls per minute."
}`

func TestNewClient(t *testing.T) {
	apiKey := "test-api-key"
	client := NewClient(apiKey)

	if client.apiKey != apiKey {
		t.Errorf("Expected API key %s, got %s", apiKey, client.apiKey)
	}

	if client.httpClient == nil {
		t.Error("HTTP client should not be nil")
	}

	if client.rateLimiter == nil {
		t.Error("Rate limiter should not be nil")
	}
}

func TestGetGlobalQuote(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request parameters
		query := r.URL.Query()

		if query.Get("function") != "GLOBAL_QUOTE" {
			t.Errorf("Expected function GLOBAL_QUOTE, got %s", query.Get("function"))
		}

		if query.Get("symbol") != "AAPL" {
			t.Errorf("Expected symbol AAPL, got %s", query.Get("symbol"))
		}

		if query.Get("apikey") == "" {
			t.Error("API key is missing")
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockGlobalQuoteResponse))
	}))
	defer server.Close()

	// Create client with test server URL
	client := NewClient("test-api-key")
	client.baseURL = server.URL

	// Test GetGlobalQuote
	ctx := context.Background()
	quote, err := client.GetGlobalQuote(ctx, "AAPL")

	if err != nil {
		t.Fatalf("GetGlobalQuote failed: %v", err)
	}

	// Verify quote data
	if quote.Symbol != "AAPL" {
		t.Errorf("Expected symbol AAPL, got %s", quote.Symbol)
	}

	if quote.Price != 151.25 {
		t.Errorf("Expected price 151.25, got %f", quote.Price)
	}

	if quote.Volume != 75000000 {
		t.Errorf("Expected volume 75000000, got %d", quote.Volume)
	}

	if quote.ChangePercent != 0.97 {
		t.Errorf("Expected change percent 0.97, got %f", quote.ChangePercent)
	}
}

func TestGetBatchQuotes(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request parameters
		query := r.URL.Query()

		if query.Get("function") != "BATCH_STOCK_QUOTES" {
			t.Errorf("Expected function BATCH_STOCK_QUOTES, got %s", query.Get("function"))
		}

		symbols := query.Get("symbols")
		if !strings.Contains(symbols, "AAPL") || !strings.Contains(symbols, "GOOGL") {
			t.Errorf("Expected symbols to contain AAPL and GOOGL, got %s", symbols)
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockBatchQuotesResponse))
	}))
	defer server.Close()

	// Create client with test server URL
	client := NewClient("test-api-key")
	client.baseURL = server.URL

	// Test GetBatchQuotes
	ctx := context.Background()
	quotes, err := client.GetBatchQuotes(ctx, []string{"AAPL", "GOOGL"})

	if err != nil {
		t.Fatalf("GetBatchQuotes failed: %v", err)
	}

	// Verify quotes
	if len(quotes) != 2 {
		t.Fatalf("Expected 2 quotes, got %d", len(quotes))
	}

	// Check first quote (AAPL)
	if quotes[0].Symbol != "AAPL" {
		t.Errorf("Expected first symbol AAPL, got %s", quotes[0].Symbol)
	}

	if quotes[0].Price != 151.25 {
		t.Errorf("Expected AAPL price 151.25, got %f", quotes[0].Price)
	}

	// Check second quote (GOOGL)
	if quotes[1].Symbol != "GOOGL" {
		t.Errorf("Expected second symbol GOOGL, got %s", quotes[1].Symbol)
	}

	if quotes[1].Price != 2750.50 {
		t.Errorf("Expected GOOGL price 2750.50, got %f", quotes[1].Price)
	}
}

func TestGetBatchQuotesExceedsLimit(t *testing.T) {
	client := NewClient("test-api-key")

	// Create array with 101 symbols
	symbols := make([]string, 101)
	for i := range symbols {
		symbols[i] = "TEST"
	}

	ctx := context.Background()
	_, err := client.GetBatchQuotes(ctx, symbols)

	if err == nil {
		t.Error("Expected error for exceeding batch size limit")
	}

	if !strings.Contains(err.Error(), "batch size exceeds maximum") {
		t.Errorf("Expected batch size error, got: %v", err)
	}
}

func TestRetryLogic(t *testing.T) {
	attemptCount := 0

	// Create test server that fails twice then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++

		if attemptCount <= 2 {
			// Return 500 error for first two attempts
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Return success on third attempt
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockGlobalQuoteResponse))
	}))
	defer server.Close()

	// Create client with test server URL
	client := NewClient("test-api-key")
	client.baseURL = server.URL

	// Test retry logic
	ctx := context.Background()
	quote, err := client.GetGlobalQuote(ctx, "AAPL")

	if err != nil {
		t.Fatalf("GetGlobalQuote failed after retries: %v", err)
	}

	if quote.Symbol != "AAPL" {
		t.Errorf("Expected symbol AAPL, got %s", quote.Symbol)
	}

	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(2, 10) // 2 per minute, 10 per day
	ctx := context.Background()

	// First two requests should succeed immediately
	err := limiter.Wait(ctx)
	if err != nil {
		t.Errorf("First request failed: %v", err)
	}

	err = limiter.Wait(ctx)
	if err != nil {
		t.Errorf("Second request failed: %v", err)
	}

	// Check status
	minuteUsed, dayUsed, _, _ := limiter.GetStatus()
	if minuteUsed != 2 {
		t.Errorf("Expected 2 requests per minute used, got %d", minuteUsed)
	}
	if dayUsed != 2 {
		t.Errorf("Expected 2 requests per day used, got %d", dayUsed)
	}

	// Third request should be rate limited (would wait)
	// We'll use a context with timeout to avoid actually waiting
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	err = limiter.Wait(ctxWithTimeout)
	if err == nil {
		t.Error("Expected rate limit to trigger wait")
	}
}

func TestParseGlobalQuote(t *testing.T) {
	client := NewClient("test-api-key")

	quoteData := QuoteData{
		Symbol:           "AAPL",
		Open:             "150.00",
		High:             "152.00",
		Low:              "149.50",
		Price:            "151.25",
		Volume:           "75000000",
		LatestTradingDay: "2024-01-15",
		PreviousClose:    "149.80",
		Change:           "1.45",
		ChangePercent:    "0.97%",
	}

	quote, err := client.parseGlobalQuote(quoteData)
	if err != nil {
		t.Fatalf("Failed to parse quote: %v", err)
	}

	if quote.Symbol != "AAPL" {
		t.Errorf("Expected symbol AAPL, got %s", quote.Symbol)
	}

	if quote.Price != 151.25 {
		t.Errorf("Expected price 151.25, got %f", quote.Price)
	}

	if quote.Volume != 75000000 {
		t.Errorf("Expected volume 75000000, got %d", quote.Volume)
	}

	if quote.ChangePercent != 0.97 {
		t.Errorf("Expected change percent 0.97, got %f", quote.ChangePercent)
	}
}

func TestCalculateMarketCap(t *testing.T) {
	quote := &LiveQuote{
		Symbol: "AAPL",
		Price:  150.00,
		Volume: 75000000,
	}

	// Calculate market cap with 1 billion shares
	sharesOutstanding := int64(1000000000)
	quote.CalculateMarketCap(sharesOutstanding)

	expectedMarketCap := int64(150.00 * 1000000000)
	if quote.MarketCap != expectedMarketCap {
		t.Errorf("Expected market cap %d, got %d", expectedMarketCap, quote.MarketCap)
	}
}

func TestContextCancellation(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockGlobalQuoteResponse))
	}))
	defer server.Close()

	client := NewClient("test-api-key")
	client.baseURL = server.URL

	// Create context with immediate cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.GetGlobalQuote(ctx, "AAPL")

	if err == nil {
		t.Error("Expected context cancellation error")
	}

	// Check if the error contains context.Canceled (it's wrapped)
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Expected error containing 'context canceled', got: %v", err)
	}
}

func TestEmptySymbolResponse(t *testing.T) {
	emptyResponse := `{"Global Quote": {}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(emptyResponse))
	}))
	defer server.Close()

	client := NewClient("test-api-key")
	client.baseURL = server.URL

	ctx := context.Background()
	_, err := client.GetGlobalQuote(ctx, "INVALID")

	if err == nil {
		t.Error("Expected error for empty response")
	}

	if !strings.Contains(err.Error(), "no data found") {
		t.Errorf("Expected 'no data found' error, got: %v", err)
	}
}

// Benchmark tests
func BenchmarkParseGlobalQuote(b *testing.B) {
	client := NewClient("test-api-key")

	quoteData := QuoteData{
		Symbol:           "AAPL",
		Open:             "150.00",
		High:             "152.00",
		Low:              "149.50",
		Price:            "151.25",
		Volume:           "75000000",
		LatestTradingDay: "2024-01-15",
		PreviousClose:    "149.80",
		Change:           "1.45",
		ChangePercent:    "0.97%",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.parseGlobalQuote(quoteData)
	}
}

func BenchmarkJSONDecode(b *testing.B) {
	data := []byte(mockGlobalQuoteResponse)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result GlobalQuote
		json.Unmarshal(data, &result)
	}
}
