package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

// testAPIKey reads from POLYGON_TEST_API_KEY env var, falls back to "demo"
var testAPIKey = getTestAPIKey()

func getTestAPIKey() string {
	if key := os.Getenv("POLYGON_TEST_API_KEY"); key != "" {
		return key
	}
	return "demo"
}

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
	// Skip this test unless a real API key is available
	if os.Getenv("CI") == "true" || os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping real API test in CI environment")
	}
	if os.Getenv("POLYGON_TEST_API_KEY") == "" {
		t.Skip("Skipping real API test: POLYGON_TEST_API_KEY not set")
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
	// Skip this test unless a real API key is available
	if os.Getenv("CI") == "true" || os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping real API test in CI environment")
	}
	if os.Getenv("POLYGON_TEST_API_KEY") == "" {
		t.Skip("Skipping real API test: POLYGON_TEST_API_KEY not set")
	}

	os.Setenv("POLYGON_API_KEY", testAPIKey)
	client := NewPolygonClient()

	testCases := []struct {
		symbol       string
		expectedType string
		shouldExist  bool
	}{
		{"AAPL", "CS", true},      // Stock
		{"SPY", "ETF", true},      // ETF
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

// ---------------------------------------------------------------------------
// GetDaysFromPeriod — pure function
// ---------------------------------------------------------------------------

func TestGetDaysFromPeriod(t *testing.T) {
	tests := []struct {
		period   string
		expected int
	}{
		{"1D", 1},
		{"5D", 5},
		{"1M", 30},
		{"3M", 90},
		{"6M", 180},
		{"1Y", 365},
		{"5Y", 1825},
		{"MAX", 7300},
		// case insensitive
		{"1d", 1},
		{"5d", 5},
		{"1m", 30},
		{"1y", 365},
		{"max", 7300},
		// unknown defaults to 365
		{"2Y", 365},
		{"unknown", 365},
		{"", 365},
	}

	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			result := GetDaysFromPeriod(tt.period)
			if tt.period == "YTD" || tt.period == "ytd" {
				// YTD is dynamic, just check it's positive
				if result <= 0 {
					t.Errorf("GetDaysFromPeriod(YTD) returned %d, expected positive", result)
				}
			} else if result != tt.expected {
				t.Errorf("GetDaysFromPeriod(%s) = %d, expected %d", tt.period, result, tt.expected)
			}
		})
	}
}

func TestGetDaysFromPeriod_YTD(t *testing.T) {
	result := GetDaysFromPeriod("YTD")
	// YTD should return days from Jan 1 to today + 1
	now := time.Now()
	startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	expectedDays := int(now.Sub(startOfYear).Hours()/24) + 1
	if result != expectedDays {
		t.Errorf("GetDaysFromPeriod(YTD) = %d, expected %d", result, expectedDays)
	}
}

// ---------------------------------------------------------------------------
// decimalPtr — pure function
// ---------------------------------------------------------------------------

func TestDecimalPtr(t *testing.T) {
	result := decimalPtr(42.5)
	if result == nil {
		t.Fatal("expected non-nil pointer")
	}
	if !result.Equal(decimal.NewFromFloat(42.5)) {
		t.Errorf("expected 42.5, got %s", result.String())
	}
}

func TestDecimalPtr_Zero(t *testing.T) {
	result := decimalPtr(0)
	if result == nil {
		t.Fatal("expected non-nil pointer")
	}
	if !result.Equal(decimal.Zero) {
		t.Errorf("expected 0, got %s", result.String())
	}
}

func TestDecimalPtr_Negative(t *testing.T) {
	result := decimalPtr(-123.456)
	if result == nil {
		t.Fatal("expected non-nil pointer")
	}
	if !result.Equal(decimal.NewFromFloat(-123.456)) {
		t.Errorf("expected -123.456, got %s", result.String())
	}
}

// ---------------------------------------------------------------------------
// MapExchangeCode — additional coverage
// ---------------------------------------------------------------------------

func TestMapExchangeCode_AllMappings(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"XNAS", "NASDAQ"},
		{"XNYS", "NYSE"},
		{"ARCX", "NYSE ARCA"},
		{"XASE", "NYSE MKT"},
		{"BATS", "CBOE BZX"},
		{"XOTC", "OTC"},
		{"XCBO", "CBOE"},
		{"XPHL", "PHLX"},
		{"XISX", "ISE"},
		// Unmapped returns as-is
		{"XXXX", "XXXX"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := MapExchangeCode(tt.code)
			if result != tt.expected {
				t.Errorf("MapExchangeCode(%s) = %s, expected %s", tt.code, result, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// MapAssetType — additional coverage
// ---------------------------------------------------------------------------

func TestMapAssetType_AllMappings(t *testing.T) {
	tests := []struct {
		typeCode string
		expected string
	}{
		{"CS", "stock"},
		{"ETF", "etf"},
		{"ETN", "etn"},
		{"FUND", "fund"},
		{"PFD", "preferred"},
		{"WARRANT", "warrant"},
		{"RIGHT", "right"},
		{"BOND", "bond"},
		{"ADRC", "adr"},
		{"ADRP", "adr"},
		{"ADRW", "adr"},
		{"ADRR", "adr"},
		{"IX", "index"},
		{"X:BTCUSD", "crypto"},
		{"X:ETHUSD", "crypto"},
		{"I:SPX", "index"},
		{"I:DJI", "index"},
		{"UNKNOWN", "other"},
		{"", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.typeCode, func(t *testing.T) {
			result := MapAssetType(tt.typeCode)
			if result != tt.expected {
				t.Errorf("MapAssetType(%s) = %s, expected %s", tt.typeCode, result, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetHistoricalData — mock server
// ---------------------------------------------------------------------------

func TestGetHistoricalData_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := AggregatesResponse{
			Status:       "OK",
			ResultsCount: 2,
			Results: []struct {
				Ticker       string  `json:"T"`
				Volume       float64 `json:"v"`
				VolumeWeight float64 `json:"vw"`
				Open         float64 `json:"o"`
				Close        float64 `json:"c"`
				High         float64 `json:"h"`
				Low          float64 `json:"l"`
				Timestamp    int64   `json:"t"`
				Transactions int     `json:"n"`
			}{
				{Ticker: "AAPL", Open: 150.0, Close: 155.0, High: 156.0, Low: 149.0, Volume: 1000000, Timestamp: 1700000000000},
				{Ticker: "AAPL", Open: 155.0, Close: 157.0, High: 158.0, Low: 154.0, Volume: 900000, Timestamp: 1700086400000},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := PolygonBaseURL
	PolygonBaseURL = server.URL
	defer func() { PolygonBaseURL = originalURL }()

	client := &PolygonClient{
		APIKey: "test-key",
		Client: &http.Client{Timeout: 5 * time.Second},
	}

	dataPoints, err := client.GetHistoricalData("AAPL", "day", "2023-11-01", "2023-11-02")
	if err != nil {
		t.Fatalf("GetHistoricalData failed: %v", err)
	}

	if len(dataPoints) != 2 {
		t.Errorf("Expected 2 data points, got %d", len(dataPoints))
	}

	if dataPoints[0].Close.InexactFloat64() != 155.0 {
		t.Errorf("Expected close 155.0, got %v", dataPoints[0].Close)
	}

	if dataPoints[0].Volume != 1000000 {
		t.Errorf("Expected volume 1000000, got %d", dataPoints[0].Volume)
	}
}

func TestGetHistoricalData_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := AggregatesResponse{
			Status: "ERROR",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := PolygonBaseURL
	PolygonBaseURL = server.URL
	defer func() { PolygonBaseURL = originalURL }()

	client := &PolygonClient{
		APIKey: "test-key",
		Client: &http.Client{Timeout: 5 * time.Second},
	}

	_, err := client.GetHistoricalData("AAPL", "day", "2023-11-01", "2023-11-02")
	if err == nil {
		t.Error("Expected error for API error status")
	}
}

func TestGetHistoricalData_DelayedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := AggregatesResponse{
			Status:       "DELAYED",
			ResultsCount: 1,
			Results: []struct {
				Ticker       string  `json:"T"`
				Volume       float64 `json:"v"`
				VolumeWeight float64 `json:"vw"`
				Open         float64 `json:"o"`
				Close        float64 `json:"c"`
				High         float64 `json:"h"`
				Low          float64 `json:"l"`
				Timestamp    int64   `json:"t"`
				Transactions int     `json:"n"`
			}{
				{Ticker: "AAPL", Open: 150.0, Close: 155.0, High: 156.0, Low: 149.0, Volume: 1000000, Timestamp: 1700000000000},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := PolygonBaseURL
	PolygonBaseURL = server.URL
	defer func() { PolygonBaseURL = originalURL }()

	client := &PolygonClient{
		APIKey: "test-key",
		Client: &http.Client{Timeout: 5 * time.Second},
	}

	dataPoints, err := client.GetHistoricalData("AAPL", "day", "2023-11-01", "2023-11-02")
	if err != nil {
		t.Fatalf("DELAYED status should not error: %v", err)
	}

	if len(dataPoints) != 1 {
		t.Errorf("Expected 1 data point, got %d", len(dataPoints))
	}
}

func TestGetHistoricalData_ConnectionError(t *testing.T) {
	originalURL := PolygonBaseURL
	PolygonBaseURL = "http://localhost:99999"
	defer func() { PolygonBaseURL = originalURL }()

	client := &PolygonClient{
		APIKey: "test-key",
		Client: &http.Client{Timeout: 1 * time.Second},
	}

	_, err := client.GetHistoricalData("AAPL", "day", "2023-11-01", "2023-11-02")
	if err == nil {
		t.Error("Expected connection error")
	}
}

// ---------------------------------------------------------------------------
// GetTickerDetails — mock server
// ---------------------------------------------------------------------------

func TestGetTickerDetails_MockSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := TickerDetailsResponse{
			Status: "OK",
		}
		response.Results.Ticker = "AAPL"
		response.Results.Name = "Apple Inc."
		response.Results.Type = "CS"
		response.Results.PrimaryExch = "XNAS"
		response.Results.Active = true
		response.Results.CIK = "0000320193"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := PolygonBaseURL
	PolygonBaseURL = server.URL
	defer func() { PolygonBaseURL = originalURL }()

	client := &PolygonClient{
		APIKey: "test-key",
		Client: &http.Client{Timeout: 5 * time.Second},
	}

	details, err := client.GetTickerDetails("AAPL")
	if err != nil {
		t.Fatalf("GetTickerDetails failed: %v", err)
	}

	if details.Results.Ticker != "AAPL" {
		t.Errorf("Expected AAPL, got %s", details.Results.Ticker)
	}
	if details.Results.Name != "Apple Inc." {
		t.Errorf("Expected Apple Inc., got %s", details.Results.Name)
	}
	if details.Results.CIK != "0000320193" {
		t.Errorf("Expected CIK 0000320193, got %s", details.Results.CIK)
	}
}

func TestGetTickerDetails_MockError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := TickerDetailsResponse{
			Status: "NOT_FOUND",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := PolygonBaseURL
	PolygonBaseURL = server.URL
	defer func() { PolygonBaseURL = originalURL }()

	client := &PolygonClient{
		APIKey: "test-key",
		Client: &http.Client{Timeout: 5 * time.Second},
	}

	_, err := client.GetTickerDetails("INVALID")
	if err == nil {
		t.Error("Expected error for NOT_FOUND status")
	}
}

// ---------------------------------------------------------------------------
// GetNews — mock server
// ---------------------------------------------------------------------------

func TestPolygonGetNews_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := NewsResponse{
			Status: "OK",
			Count:  1,
			Results: []struct {
				ID        string `json:"id"`
				Publisher struct {
					Name        string `json:"name"`
					HomepageURL string `json:"homepage_url"`
					LogoURL     string `json:"logo_url"`
					FaviconURL  string `json:"favicon_url"`
				} `json:"publisher"`
				Title        string   `json:"title"`
				Author       string   `json:"author"`
				PublishedUTC string   `json:"published_utc"`
				ArticleURL   string   `json:"article_url"`
				Tickers      []string `json:"tickers"`
				ImageURL     string   `json:"image_url"`
				Description  string   `json:"description"`
				Keywords     []string `json:"keywords"`
				Insights     []struct {
					Ticker             string `json:"ticker"`
					Sentiment          string `json:"sentiment"`
					SentimentReasoning string `json:"sentiment_reasoning"`
				} `json:"insights"`
			}{
				{
					ID:           "1",
					Title:        "Apple News",
					Author:       "Test Author",
					PublishedUTC: "2024-01-15T12:00:00Z",
					ArticleURL:   "https://example.com/news",
					Description:  "Test description",
					Tickers:      []string{"AAPL"},
				},
			},
		}
		response.Results[0].Publisher.Name = "Test Publisher"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := PolygonBaseURL
	PolygonBaseURL = server.URL
	defer func() { PolygonBaseURL = originalURL }()

	client := &PolygonClient{
		APIKey: "test-key",
		Client: &http.Client{Timeout: 5 * time.Second},
	}

	articles, err := client.GetNews("AAPL", 10)
	if err != nil {
		t.Fatalf("GetNews failed: %v", err)
	}

	if len(articles) != 1 {
		t.Errorf("Expected 1 article, got %d", len(articles))
	}

	if articles[0].Title != "Apple News" {
		t.Errorf("Expected title 'Apple News', got %s", articles[0].Title)
	}

	if articles[0].Source != "Test Publisher" {
		t.Errorf("Expected source 'Test Publisher', got %s", articles[0].Source)
	}
}

func TestPolygonGetNews_DefaultLimit(t *testing.T) {
	var requestURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURL = r.URL.String()
		response := NewsResponse{
			Status: "OK",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := PolygonBaseURL
	PolygonBaseURL = server.URL
	defer func() { PolygonBaseURL = originalURL }()

	client := &PolygonClient{
		APIKey: "test-key",
		Client: &http.Client{Timeout: 5 * time.Second},
	}

	// Passing 0 should use default 30
	_, err := client.GetNews("AAPL", 0)
	if err != nil {
		t.Fatalf("GetNews failed: %v", err)
	}

	if !containsSubstring(requestURL, "limit=30") {
		t.Errorf("Expected default limit=30 in URL, got %s", requestURL)
	}
}

func TestPolygonGetNews_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	originalURL := PolygonBaseURL
	PolygonBaseURL = server.URL
	defer func() { PolygonBaseURL = originalURL }()

	client := &PolygonClient{
		APIKey: "test-key",
		Client: &http.Client{Timeout: 5 * time.Second},
	}

	_, err := client.GetNews("AAPL", 10)
	if err == nil {
		t.Error("Expected error for API failure")
	}
}

// ---------------------------------------------------------------------------
// GetMultipleQuotes — empty symbols
// ---------------------------------------------------------------------------

func TestGetMultipleQuotes_EmptySymbols(t *testing.T) {
	client := &PolygonClient{
		APIKey: "test-key",
		Client: &http.Client{Timeout: 5 * time.Second},
	}

	quotes, err := client.GetMultipleQuotes([]string{})
	if err != nil {
		t.Fatalf("Expected no error for empty symbols: %v", err)
	}
	if len(quotes) != 0 {
		t.Errorf("Expected 0 quotes, got %d", len(quotes))
	}
}

// helper
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && strings.Contains(s, substr))
}

// ---------------------------------------------------------------------------
// IsMarketOpen — basic validation
// ---------------------------------------------------------------------------

func TestIsMarketOpen_ReturnsBoolean(t *testing.T) {
	client := NewPolygonClient()
	// Just verify it doesn't panic and returns a bool
	_ = client.IsMarketOpen()
}

// ---------------------------------------------------------------------------
// Struct serialization
// ---------------------------------------------------------------------------

func TestQuoteData_Fields(t *testing.T) {
	q := &QuoteData{
		Symbol:    "AAPL",
		Price:     150.25,
		Volume:    1000000,
		Timestamp: 1700000000,
	}

	if q.Symbol != "AAPL" {
		t.Errorf("Expected AAPL, got %s", q.Symbol)
	}
	if q.Price != 150.25 {
		t.Errorf("Expected 150.25, got %f", q.Price)
	}
	if q.Volume != 1000000 {
		t.Errorf("Expected 1000000, got %d", q.Volume)
	}
}
