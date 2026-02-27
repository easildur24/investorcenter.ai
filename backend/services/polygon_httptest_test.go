package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// savePolygonBaseURL saves the current PolygonBaseURL and returns a restore func.
func savePolygonBaseURL() func() {
	orig := PolygonBaseURL
	return func() { PolygonBaseURL = orig }
}

// newPolygonTestClient creates a PolygonClient with a fast timeout.
func newPolygonTestClient() *PolygonClient {
	return &PolygonClient{
		APIKey: "test-key",
		Client: &http.Client{Timeout: 5 * time.Second},
	}
}

// ===========================================================================
// GetIntradayData
// ===========================================================================

func TestPolygon_HTTP_GetIntradayData_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the URL path pattern
		assert.Contains(t, r.URL.Path, "/v2/aggs/ticker/AAPL/range/5/minute")
		assert.Contains(t, r.URL.RawQuery, "apikey=test-key")

		response := AggregatesResponse{
			Status:       "OK",
			ResultsCount: 3,
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
				{Ticker: "AAPL", Open: 150.0, Close: 150.5, High: 151.0, Low: 149.5, Volume: 50000, Timestamp: 1700000000000},
				{Ticker: "AAPL", Open: 150.5, Close: 151.0, High: 151.5, Low: 150.0, Volume: 45000, Timestamp: 1700000300000},
				{Ticker: "AAPL", Open: 151.0, Close: 152.0, High: 152.5, Low: 150.5, Volume: 60000, Timestamp: 1700000600000},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	dataPoints, err := client.GetIntradayData("AAPL")
	require.NoError(t, err)
	assert.Len(t, dataPoints, 3)

	assert.InDelta(t, 150.0, dataPoints[0].Open.InexactFloat64(), 0.01)
	assert.InDelta(t, 150.5, dataPoints[0].Close.InexactFloat64(), 0.01)
	assert.Equal(t, int64(50000), dataPoints[0].Volume)
}

func TestPolygon_HTTP_GetIntradayData_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := AggregatesResponse{Status: "ERROR"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	_, err := client.GetIntradayData("AAPL")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API error")
}

func TestPolygon_HTTP_GetIntradayData_ConnectionError(t *testing.T) {
	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = "http://localhost:99999"

	client := &PolygonClient{
		APIKey: "test-key",
		Client: &http.Client{Timeout: 1 * time.Second},
	}

	_, err := client.GetIntradayData("AAPL")
	assert.Error(t, err)
}

func TestPolygon_HTTP_GetIntradayData_BadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	_, err := client.GetIntradayData("AAPL")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestPolygon_HTTP_GetIntradayData_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := AggregatesResponse{
			Status:       "OK",
			ResultsCount: 0,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	dataPoints, err := client.GetIntradayData("AAPL")
	require.NoError(t, err)
	assert.Empty(t, dataPoints)
}

// ===========================================================================
// GetTickersByType
// ===========================================================================

func TestPolygon_HTTP_GetTickersByType_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/v3/reference/tickers")
		assert.Equal(t, "CS", r.URL.Query().Get("type"))
		assert.Equal(t, "true", r.URL.Query().Get("active"))

		response := PolygonTickersResponse{
			Status: "OK",
			Count:  2,
			Results: []PolygonTicker{
				{Ticker: "AAPL", Name: "Apple Inc.", Type: "CS", Active: true, PrimaryExchange: "XNAS"},
				{Ticker: "MSFT", Name: "Microsoft Corp.", Type: "CS", Active: true, PrimaryExchange: "XNAS"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	tickers, err := client.GetTickersByType("CS")
	require.NoError(t, err)
	assert.Len(t, tickers, 2)
	assert.Equal(t, "AAPL", tickers[0].Ticker)
	assert.Equal(t, "MSFT", tickers[1].Ticker)
}

func TestPolygon_HTTP_GetTickersByType_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := PolygonTickersResponse{Status: "ERROR"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	_, err := client.GetTickersByType("CS")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API error")
}

func TestPolygon_HTTP_GetTickersByType_BadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{broken"))
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	_, err := client.GetTickersByType("CS")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

// ===========================================================================
// GetBulkStockSnapshots
// ===========================================================================

func TestPolygon_HTTP_GetBulkStockSnapshots_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/v2/snapshot/locale/us/markets/stocks/tickers")

		// Build a response that matches BulkStockSnapshotResponse
		response := map[string]interface{}{
			"status": "OK",
			"count":  2,
			"tickers": []map[string]interface{}{
				{
					"ticker":           "AAPL",
					"todaysChange":     2.5,
					"todaysChangePerc": 1.67,
					"day":              map[string]interface{}{"o": 150.0, "h": 153.0, "l": 149.5, "c": 152.5, "v": 80000000.0},
					"lastTrade":        map[string]interface{}{"t": int64(1700000000000000000), "p": 152.5, "s": 100.0, "x": 4},
					"prevDay":          map[string]interface{}{"o": 148.0, "h": 150.5, "l": 147.0, "c": 150.0, "v": 70000000.0},
				},
				{
					"ticker":           "MSFT",
					"todaysChange":     3.0,
					"todaysChangePerc": 0.85,
					"day":              map[string]interface{}{"o": 355.0, "h": 360.0, "l": 354.0, "c": 358.0, "v": 25000000.0},
					"lastTrade":        map[string]interface{}{"t": int64(1700000000000000000), "p": 358.0, "s": 50.0, "x": 4},
					"prevDay":          map[string]interface{}{"o": 352.0, "h": 356.0, "l": 351.0, "c": 355.0, "v": 22000000.0},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	result, err := client.GetBulkStockSnapshots()
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "OK", result.Status)
	assert.Equal(t, 2, result.Count)
	assert.Len(t, result.Tickers, 2)
	assert.Equal(t, "AAPL", result.Tickers[0].Ticker)
	assert.InDelta(t, 152.5, result.Tickers[0].LastTrade.Price, 0.01)
}

func TestPolygon_HTTP_GetBulkStockSnapshots_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	_, err := client.GetBulkStockSnapshots()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status: 500")
}

func TestPolygon_HTTP_GetBulkStockSnapshots_APIStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"status":  "ERROR",
			"tickers": []interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	_, err := client.GetBulkStockSnapshots()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API error")
}

func TestPolygon_HTTP_GetBulkStockSnapshots_ConnectionError(t *testing.T) {
	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = "http://localhost:99999"

	client := &PolygonClient{
		APIKey: "test-key",
		Client: &http.Client{Timeout: 1 * time.Second},
	}

	_, err := client.GetBulkStockSnapshots()
	assert.Error(t, err)
}

// ===========================================================================
// GetBulkCryptoSnapshots
// ===========================================================================

func TestPolygon_HTTP_GetBulkCryptoSnapshots_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/v2/snapshot/locale/global/markets/crypto/tickers")

		response := map[string]interface{}{
			"status": "OK",
			"count":  1,
			"tickers": []map[string]interface{}{
				{
					"ticker":           "X:BTCUSD",
					"todaysChange":     1500.0,
					"todaysChangePerc": 2.5,
					"day":              map[string]interface{}{"o": 60000.0, "h": 62000.0, "l": 59500.0, "c": 61500.0, "v": 50000.0},
					"lastTrade":        map[string]interface{}{"t": int64(1700000000000000000), "p": 61500.0, "s": 0.5, "x": 1},
					"prevDay":          map[string]interface{}{"o": 59000.0, "h": 60500.0, "l": 58500.0, "c": 60000.0, "v": 45000.0},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	result, err := client.GetBulkCryptoSnapshots()
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "OK", result.Status)
	assert.Len(t, result.Tickers, 1)
	assert.Equal(t, "X:BTCUSD", result.Tickers[0].Ticker)
	assert.InDelta(t, 61500.0, result.Tickers[0].LastTrade.Price, 0.01)
}

func TestPolygon_HTTP_GetBulkCryptoSnapshots_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	_, err := client.GetBulkCryptoSnapshots()
	assert.Error(t, err)
}

func TestPolygon_HTTP_GetBulkCryptoSnapshots_APIStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"status":  "ERROR",
			"tickers": []interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	_, err := client.GetBulkCryptoSnapshots()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API error")
}

// ===========================================================================
// GetStockRealTimePrice
// ===========================================================================

func TestPolygon_HTTP_GetStockRealTimePrice_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/v2/snapshot/locale/us/markets/stocks/tickers/AAPL")

		response := StockSnapshotResponse{
			Status: "OK",
		}
		response.Ticker.Ticker = "AAPL"
		response.Ticker.TodaysChange = 2.5
		response.Ticker.TodaysChangePerc = 1.67
		response.Ticker.Day.Open = 150.0
		response.Ticker.Day.High = 153.0
		response.Ticker.Day.Low = 149.5
		response.Ticker.Day.Close = 152.5
		response.Ticker.Day.Volume = 80000000
		response.Ticker.LastTrade.Price = 152.5
		response.Ticker.LastTrade.Timestamp = 1700000000000000000 // nanoseconds
		response.Ticker.PrevDay.Close = 150.0

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	price, err := client.GetStockRealTimePrice("AAPL")
	require.NoError(t, err)
	require.NotNil(t, price)

	assert.Equal(t, "AAPL", price.Symbol)
	assert.InDelta(t, 152.5, price.Price.InexactFloat64(), 0.01)
	assert.InDelta(t, 150.0, price.Open.InexactFloat64(), 0.01)
	assert.InDelta(t, 153.0, price.High.InexactFloat64(), 0.01)
	assert.InDelta(t, 149.5, price.Low.InexactFloat64(), 0.01)
	assert.Equal(t, int64(80000000), price.Volume)
}

func TestPolygon_HTTP_GetStockRealTimePrice_FallbackToPrevDay(t *testing.T) {
	// When lastTrade price is 0 and day data is empty, should fall back to prevDay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := StockSnapshotResponse{
			Status: "OK",
		}
		response.Ticker.Ticker = "AAPL"
		response.Ticker.LastTrade.Price = 0 // No last trade
		response.Ticker.Day.Open = 0        // Market not open yet
		response.Ticker.PrevDay.Open = 148.0
		response.Ticker.PrevDay.High = 150.5
		response.Ticker.PrevDay.Low = 147.0
		response.Ticker.PrevDay.Close = 150.0
		response.Ticker.PrevDay.Volume = 70000000

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	price, err := client.GetStockRealTimePrice("AAPL")
	require.NoError(t, err)
	require.NotNil(t, price)

	// Should use prevDay close as price
	assert.InDelta(t, 150.0, price.Price.InexactFloat64(), 0.01)
}

func TestPolygon_HTTP_GetStockRealTimePrice_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	_, err := client.GetStockRealTimePrice("AAPL")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status: 403")
}

func TestPolygon_HTTP_GetStockRealTimePrice_APIStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := StockSnapshotResponse{Status: "NOT_FOUND"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	_, err := client.GetStockRealTimePrice("INVALID")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NOT_FOUND")
}

// ===========================================================================
// GetCryptoRealTimePrice
// ===========================================================================

func TestPolygon_HTTP_GetCryptoRealTimePrice_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/v2/snapshot/locale/global/markets/crypto/tickers/X:BTCUSD")

		response := CryptoSnapshotResponse{
			Status: "OK",
		}
		response.Ticker.Ticker = "X:BTCUSD"
		response.Ticker.TodaysChange = 1500.0
		response.Ticker.TodaysChangePerc = 2.5
		response.Ticker.Day.Open = 60000.0
		response.Ticker.Day.High = 62000.0
		response.Ticker.Day.Low = 59500.0
		response.Ticker.Day.Close = 61500.0
		response.Ticker.Day.Volume = 50000
		response.Ticker.LastTrade.Price = 61500.0
		response.Ticker.LastTrade.Timestamp = 1700000000000000000

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	price, err := client.GetCryptoRealTimePrice("X:BTCUSD")
	require.NoError(t, err)
	require.NotNil(t, price)

	assert.Equal(t, "X:BTCUSD", price.Symbol)
	assert.InDelta(t, 61500.0, price.Price.InexactFloat64(), 0.01)
	assert.InDelta(t, 60000.0, price.Open.InexactFloat64(), 0.01)
	assert.InDelta(t, 62000.0, price.High.InexactFloat64(), 0.01)
	assert.InDelta(t, 1500.0, price.Change.InexactFloat64(), 0.01)
	assert.InDelta(t, 2.5, price.ChangePercent.InexactFloat64(), 0.01)
}

func TestPolygon_HTTP_GetCryptoRealTimePrice_FallbackToDayClose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := CryptoSnapshotResponse{
			Status: "OK",
		}
		response.Ticker.Ticker = "X:ETHUSD"
		response.Ticker.LastTrade.Price = 0 // No last trade
		response.Ticker.Day.Close = 3500.0  // Fallback
		response.Ticker.Day.Open = 3400.0
		response.Ticker.TodaysChange = 100.0
		response.Ticker.TodaysChangePerc = 2.94

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	price, err := client.GetCryptoRealTimePrice("X:ETHUSD")
	require.NoError(t, err)
	require.NotNil(t, price)

	assert.InDelta(t, 3500.0, price.Price.InexactFloat64(), 0.01)
}

func TestPolygon_HTTP_GetCryptoRealTimePrice_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	_, err := client.GetCryptoRealTimePrice("X:BTCUSD")
	assert.Error(t, err)
}

func TestPolygon_HTTP_GetCryptoRealTimePrice_APIStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := CryptoSnapshotResponse{Status: "ERROR"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	_, err := client.GetCryptoRealTimePrice("X:BTCUSD")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API error")
}

// ===========================================================================
// GetMultipleQuotes — full mock with stocks and crypto
// ===========================================================================

func TestPolygon_HTTP_GetMultipleQuotes_StocksOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This should hit the bulk stock snapshot endpoint
		if strings.Contains(r.URL.Path, "/v2/snapshot/locale/us/markets/stocks") {
			response := map[string]interface{}{
				"status": "OK",
				"count":  2,
				"tickers": []map[string]interface{}{
					{
						"ticker":           "AAPL",
						"todaysChangePerc": 1.5,
						"day":              map[string]interface{}{"v": 80000000.0},
						"lastTrade":        map[string]interface{}{"t": int64(1700000000000000000), "p": 152.5},
						"prevDay":          map[string]interface{}{"c": 150.0, "v": 70000000.0},
					},
					{
						"ticker":           "MSFT",
						"todaysChangePerc": 0.8,
						"day":              map[string]interface{}{"v": 25000000.0},
						"lastTrade":        map[string]interface{}{"t": int64(1700000000000000000), "p": 358.0},
						"prevDay":          map[string]interface{}{"c": 355.0, "v": 22000000.0},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	quotes, err := client.GetMultipleQuotes([]string{"AAPL", "MSFT"})
	require.NoError(t, err)
	assert.Len(t, quotes, 2)

	aapl, ok := quotes["AAPL"]
	require.True(t, ok)
	assert.InDelta(t, 152.5, aapl.Price, 0.01)
	assert.InDelta(t, 1.5, aapl.ChangePercent, 0.01)

	msft, ok := quotes["MSFT"]
	require.True(t, ok)
	assert.InDelta(t, 358.0, msft.Price, 0.01)
}

func TestPolygon_HTTP_GetMultipleQuotes_CryptoOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/v2/snapshot/locale/global/markets/crypto") {
			response := map[string]interface{}{
				"status": "OK",
				"count":  1,
				"tickers": []map[string]interface{}{
					{
						"ticker":           "X:BTCUSD",
						"todaysChangePerc": 3.0,
						"day":              map[string]interface{}{"v": 50000.0},
						"lastTrade":        map[string]interface{}{"t": int64(1700000000000000000), "p": 61500.0},
						"prevDay":          map[string]interface{}{"c": 60000.0, "v": 45000.0},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	quotes, err := client.GetMultipleQuotes([]string{"X:BTCUSD"})
	require.NoError(t, err)
	assert.Len(t, quotes, 1)

	btc, ok := quotes["X:BTCUSD"]
	require.True(t, ok)
	assert.InDelta(t, 61500.0, btc.Price, 0.01)
}

func TestPolygon_HTTP_GetMultipleQuotes_Mixed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/locale/us/markets/stocks") {
			response := map[string]interface{}{
				"status": "OK",
				"count":  1,
				"tickers": []map[string]interface{}{
					{
						"ticker":           "AAPL",
						"todaysChangePerc": 1.0,
						"day":              map[string]interface{}{"v": 80000000.0},
						"lastTrade":        map[string]interface{}{"t": int64(1700000000000000000), "p": 152.5},
						"prevDay":          map[string]interface{}{"c": 150.0, "v": 70000000.0},
					},
				},
			}
			json.NewEncoder(w).Encode(response)
			return
		}
		if strings.Contains(r.URL.Path, "/locale/global/markets/crypto") {
			response := map[string]interface{}{
				"status": "OK",
				"count":  1,
				"tickers": []map[string]interface{}{
					{
						"ticker":           "X:BTCUSD",
						"todaysChangePerc": 2.0,
						"day":              map[string]interface{}{"v": 50000.0},
						"lastTrade":        map[string]interface{}{"t": int64(1700000000000000000), "p": 61500.0},
						"prevDay":          map[string]interface{}{"c": 60000.0, "v": 45000.0},
					},
				},
			}
			json.NewEncoder(w).Encode(response)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	quotes, err := client.GetMultipleQuotes([]string{"AAPL", "X:BTCUSD"})
	require.NoError(t, err)
	assert.Len(t, quotes, 2)
	assert.NotNil(t, quotes["AAPL"])
	assert.NotNil(t, quotes["X:BTCUSD"])
}

// ===========================================================================
// GetDailyData
// ===========================================================================

func TestPolygon_HTTP_GetDailyData_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/v2/aggs/ticker/AAPL/range/1/day")

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

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	dataPoints, err := client.GetDailyData("AAPL", 30)
	require.NoError(t, err)
	assert.Len(t, dataPoints, 2)
	assert.InDelta(t, 155.0, dataPoints[0].Close.InexactFloat64(), 0.01)
}

// ===========================================================================
// GetHistoricalData — additional edge cases
// ===========================================================================

func TestPolygon_HTTP_GetHistoricalData_BadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{{not json}}"))
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	_, err := client.GetHistoricalData("AAPL", "day", "2023-11-01", "2023-11-02")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestPolygon_HTTP_GetHistoricalData_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := AggregatesResponse{
			Status:       "OK",
			ResultsCount: 0,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	dataPoints, err := client.GetHistoricalData("AAPL", "day", "2023-11-01", "2023-11-02")
	require.NoError(t, err)
	assert.Empty(t, dataPoints)
}

// ===========================================================================
// GetTickerDetails — additional edge cases
// ===========================================================================

func TestPolygon_HTTP_GetTickerDetails_BadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not-valid-json"))
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	_, err := client.GetTickerDetails("AAPL")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestPolygon_HTTP_GetTickerDetails_ConnectionError(t *testing.T) {
	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = "http://localhost:99999"

	client := &PolygonClient{
		APIKey: "test-key",
		Client: &http.Client{Timeout: 1 * time.Second},
	}

	_, err := client.GetTickerDetails("AAPL")
	assert.Error(t, err)
}

// ===========================================================================
// GetNews — additional edge cases
// ===========================================================================

func TestPolygon_HTTP_GetNews_BadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{broken"))
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	_, err := client.GetNews("AAPL", 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestPolygon_HTTP_GetNews_ConnectionError(t *testing.T) {
	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = "http://localhost:99999"

	client := &PolygonClient{
		APIKey: "test-key",
		Client: &http.Client{Timeout: 1 * time.Second},
	}

	_, err := client.GetNews("AAPL", 10)
	assert.Error(t, err)
}

func TestPolygon_HTTP_GetNews_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := NewsResponse{
			Status: "OK",
			Count:  0,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	articles, err := client.GetNews("AAPL", 10)
	require.NoError(t, err)
	assert.Empty(t, articles)
}

func TestPolygon_HTTP_GetNews_StatusNotOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := NewsResponse{Status: "ERROR"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	restore := savePolygonBaseURL()
	defer restore()
	PolygonBaseURL = server.URL

	client := newPolygonTestClient()

	_, err := client.GetNews("AAPL", 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API error")
}
