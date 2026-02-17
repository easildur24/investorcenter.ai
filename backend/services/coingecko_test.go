package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// MapSymbolToCoinGeckoID
// ---------------------------------------------------------------------------

func TestMapSymbolToCoinGeckoID(t *testing.T) {
	client := NewCoinGeckoClient()

	tests := []struct {
		symbol   string
		expected string
	}{
		{"BTC", "bitcoin"},
		{"ETH", "ethereum"},
		{"SOL", "solana"},
		{"ADA", "cardano"},
		{"XRP", "ripple"},
		{"DOGE", "dogecoin"},
		{"LINK", "chainlink"},
		{"UNI", "uniswap"},
		{"LTC", "litecoin"},
		{"AVAX", "avalanche-2"},
		{"MATIC", "matic-network"},
		{"SHIB", "shiba-inu"},
		{"BNB", "binancecoin"},
		{"USDT", "tether"},
		{"USDC", "usd-coin"},
		{"DAI", "dai"},
		{"PEPE", "pepe"},
		{"SUI", "sui"},
	}

	for _, tt := range tests {
		t.Run(tt.symbol, func(t *testing.T) {
			result := client.MapSymbolToCoinGeckoID(tt.symbol)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapSymbolToCoinGeckoID_CaseInsensitive(t *testing.T) {
	client := NewCoinGeckoClient()

	assert.Equal(t, "bitcoin", client.MapSymbolToCoinGeckoID("btc"))
	assert.Equal(t, "bitcoin", client.MapSymbolToCoinGeckoID("BTC"))
	assert.Equal(t, "bitcoin", client.MapSymbolToCoinGeckoID("Btc"))
}

func TestMapSymbolToCoinGeckoID_UnknownSymbol(t *testing.T) {
	client := NewCoinGeckoClient()

	// Unknown symbols default to lowercase
	result := client.MapSymbolToCoinGeckoID("NEWCOIN")
	assert.Equal(t, "newcoin", result)
}

// ---------------------------------------------------------------------------
// NewCoinGeckoClient
// ---------------------------------------------------------------------------

func TestNewCoinGeckoClient(t *testing.T) {
	client := NewCoinGeckoClient()
	require.NotNil(t, client)
	require.NotNil(t, client.Client)
}

// ---------------------------------------------------------------------------
// GetMarketChart — mock server
// ---------------------------------------------------------------------------

func TestGetMarketChart_Success(t *testing.T) {
	// Create mock CoinGecko server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := MarketChartResponse{
			Prices: [][]float64{
				{1700000000000, 50000.0},
				{1700086400000, 51000.0},
				{1700172800000, 49500.0},
			},
			TotalVolumes: [][]float64{
				{1700000000000, 1000000000},
				{1700086400000, 1200000000},
				{1700172800000, 900000000},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Override base URL
	originalURL := CoinGeckoBaseURL
	CoinGeckoBaseURL = server.URL
	defer func() { CoinGeckoBaseURL = originalURL }()

	client := NewCoinGeckoClient()
	dataPoints, err := client.GetMarketChart("BTC", 3)

	require.NoError(t, err)
	assert.Len(t, dataPoints, 3)

	// First data point
	assert.True(t, dataPoints[0].Close.InexactFloat64() == 50000.0)
	assert.Equal(t, int64(1000000000), dataPoints[0].Volume)

	// Second data point
	assert.True(t, dataPoints[1].Close.InexactFloat64() == 51000.0)
}

func TestGetMarketChart_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	originalURL := CoinGeckoBaseURL
	CoinGeckoBaseURL = server.URL
	defer func() { CoinGeckoBaseURL = originalURL }()

	client := NewCoinGeckoClient()
	_, err := client.GetMarketChart("BTC", 7)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 429")
}

func TestGetMarketChart_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	originalURL := CoinGeckoBaseURL
	CoinGeckoBaseURL = server.URL
	defer func() { CoinGeckoBaseURL = originalURL }()

	client := NewCoinGeckoClient()
	_, err := client.GetMarketChart("BTC", 7)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestGetMarketChart_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := MarketChartResponse{
			Prices:       [][]float64{},
			TotalVolumes: [][]float64{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := CoinGeckoBaseURL
	CoinGeckoBaseURL = server.URL
	defer func() { CoinGeckoBaseURL = originalURL }()

	client := NewCoinGeckoClient()
	dataPoints, err := client.GetMarketChart("BTC", 7)

	require.NoError(t, err)
	assert.Empty(t, dataPoints)
}

func TestGetMarketChart_ConnectionError(t *testing.T) {
	originalURL := CoinGeckoBaseURL
	CoinGeckoBaseURL = "http://localhost:99999" // invalid port
	defer func() { CoinGeckoBaseURL = originalURL }()

	client := &CoinGeckoClient{
		Client: &http.Client{Timeout: 1 * time.Second},
	}
	_, err := client.GetMarketChart("BTC", 7)

	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// GetOHLC — mock server
// ---------------------------------------------------------------------------

func TestGetOHLC_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := OHLCResponse{
			{1700000000000, 50000, 51000, 49500, 50500},
			{1700086400000, 50500, 52000, 50000, 51500},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := CoinGeckoBaseURL
	CoinGeckoBaseURL = server.URL
	defer func() { CoinGeckoBaseURL = originalURL }()

	client := NewCoinGeckoClient()
	dataPoints, err := client.GetOHLC("BTC", 7)

	require.NoError(t, err)
	assert.Len(t, dataPoints, 2)

	// OHLC data should have distinct values
	dp := dataPoints[0]
	assert.True(t, dp.Open.InexactFloat64() == 50000.0)
	assert.True(t, dp.High.InexactFloat64() == 51000.0)
	assert.True(t, dp.Low.InexactFloat64() == 49500.0)
	assert.True(t, dp.Close.InexactFloat64() == 50500.0)
}

func TestGetOHLC_FallsBackToMarketChart(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call (OHLC) fails
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Second call (market_chart fallback) succeeds
		response := MarketChartResponse{
			Prices: [][]float64{
				{1700000000000, 50000.0},
			},
			TotalVolumes: [][]float64{
				{1700000000000, 1000000000},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := CoinGeckoBaseURL
	CoinGeckoBaseURL = server.URL
	defer func() { CoinGeckoBaseURL = originalURL }()

	client := NewCoinGeckoClient()
	dataPoints, err := client.GetOHLC("BTC", 7)

	require.NoError(t, err)
	assert.Len(t, dataPoints, 1)
}

func TestGetOHLC_IncompleteDataPoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := OHLCResponse{
			{1700000000000, 50000, 51000, 49500, 50500},
			{1700086400000},        // Incomplete — fewer than 5 elements
			{1700172800000, 52000}, // Incomplete
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := CoinGeckoBaseURL
	CoinGeckoBaseURL = server.URL
	defer func() { CoinGeckoBaseURL = originalURL }()

	client := NewCoinGeckoClient()
	dataPoints, err := client.GetOHLC("BTC", 7)

	require.NoError(t, err)
	assert.Len(t, dataPoints, 1, "should skip incomplete data points")
}

// ---------------------------------------------------------------------------
// GetChartData — routing
// ---------------------------------------------------------------------------

func TestGetChartData_OneDayUsesMarketChart(t *testing.T) {
	var requestPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		response := MarketChartResponse{
			Prices: [][]float64{{1700000000000, 50000.0}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := CoinGeckoBaseURL
	CoinGeckoBaseURL = server.URL
	defer func() { CoinGeckoBaseURL = originalURL }()

	client := NewCoinGeckoClient()
	_, err := client.GetChartData("BTC", "1D")

	require.NoError(t, err)
	assert.Contains(t, requestPath, "market_chart")
}

func TestGetChartData_LongerPeriodUsesOHLC(t *testing.T) {
	var requestPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		response := OHLCResponse{
			{1700000000000, 50000, 51000, 49500, 50500},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := CoinGeckoBaseURL
	CoinGeckoBaseURL = server.URL
	defer func() { CoinGeckoBaseURL = originalURL }()

	client := NewCoinGeckoClient()
	_, err := client.GetChartData("BTC", "1M")

	require.NoError(t, err)
	assert.Contains(t, requestPath, "ohlc")
}

// ---------------------------------------------------------------------------
// MarketChartResponse struct
// ---------------------------------------------------------------------------

func TestMarketChartResponse_PartialPriceData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := MarketChartResponse{
			Prices: [][]float64{
				{1700000000000, 50000.0},
				{1700086400000}, // Only one element — should be skipped
			},
			TotalVolumes: [][]float64{
				{1700000000000, 1000000000},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := CoinGeckoBaseURL
	CoinGeckoBaseURL = server.URL
	defer func() { CoinGeckoBaseURL = originalURL }()

	client := NewCoinGeckoClient()
	dataPoints, err := client.GetMarketChart("BTC", 1)

	require.NoError(t, err)
	assert.Len(t, dataPoints, 1, "should skip price data with fewer than 2 elements")
}
