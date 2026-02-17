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

// ─── NewVolumeService ─────────────────────────────────────────────

func TestVolumeServiceCreation(t *testing.T) {
	t.Run("creates service with default values", func(t *testing.T) {
		vs := NewVolumeService()
		assert.NotNil(t, vs)
		assert.NotEmpty(t, vs.apiKey)
		assert.Equal(t, "https://api.polygon.io", vs.baseURL)
		assert.NotNil(t, vs.httpClient)
		assert.NotNil(t, vs.cache)
		assert.Equal(t, 1*time.Minute, vs.cacheExpiry)
	})
}

// ─── Cache behavior ───────────────────────────────────────────────

func TestVolumeServiceCache(t *testing.T) {
	t.Run("stores and retrieves from cache", func(t *testing.T) {
		vs := NewVolumeService()

		data := &VolumeData{
			Symbol: "AAPL",
			Volume: 50000000,
			Close:  150.25,
		}
		vs.storeInCache("AAPL", data)

		cached := vs.getFromCache("AAPL")
		require.NotNil(t, cached)
		assert.Equal(t, "AAPL", cached.Symbol)
		assert.Equal(t, int64(50000000), cached.Volume)
		assert.Equal(t, 150.25, cached.Close)
	})

	t.Run("returns nil for cache miss", func(t *testing.T) {
		vs := NewVolumeService()
		cached := vs.getFromCache("MISSING")
		assert.Nil(t, cached)
	})

	t.Run("returns nil for expired cache entry", func(t *testing.T) {
		vs := &VolumeService{
			cache:       make(map[string]*cachedVolume),
			cacheExpiry: 1 * time.Millisecond, // Very short expiry
		}

		data := &VolumeData{Symbol: "TSLA", Volume: 100}
		vs.storeInCache("TSLA", data)

		// Wait for cache to expire
		time.Sleep(5 * time.Millisecond)

		cached := vs.getFromCache("TSLA")
		assert.Nil(t, cached)
	})

	t.Run("ClearCache removes expired entries only", func(t *testing.T) {
		vs := &VolumeService{
			cache:       make(map[string]*cachedVolume),
			cacheExpiry: 50 * time.Millisecond,
		}

		// Add two entries
		vs.storeInCache("OLD", &VolumeData{Symbol: "OLD"})
		time.Sleep(60 * time.Millisecond)
		vs.storeInCache("NEW", &VolumeData{Symbol: "NEW"})

		vs.ClearCache()

		// OLD should be removed (expired), NEW should remain
		vs.cacheMutex.RLock()
		_, oldExists := vs.cache["OLD"]
		_, newExists := vs.cache["NEW"]
		vs.cacheMutex.RUnlock()

		assert.False(t, oldExists, "expired entry should be removed")
		assert.True(t, newExists, "fresh entry should remain")
	})

	t.Run("cache overwrites existing entry", func(t *testing.T) {
		vs := NewVolumeService()

		vs.storeInCache("GOOG", &VolumeData{Symbol: "GOOG", Volume: 100})
		vs.storeInCache("GOOG", &VolumeData{Symbol: "GOOG", Volume: 200})

		cached := vs.getFromCache("GOOG")
		require.NotNil(t, cached)
		assert.Equal(t, int64(200), cached.Volume)
	})
}

// ─── GetRealTimeVolume with mock server ──────────────────────────

func TestVolumeServiceGetRealTimeVolume(t *testing.T) {
	t.Run("returns data on successful API response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.Path, "/v2/snapshot/locale/us/markets/stocks/tickers/AAPL")
			assert.NotEmpty(t, r.URL.Query().Get("apiKey"))

			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "OK",
				"ticker": map[string]interface{}{
					"day": map[string]interface{}{
						"o":  148.50,
						"h":  152.00,
						"l":  147.80,
						"c":  151.25,
						"v":  55000000,
						"vw": 149.75,
					},
					"prevDay": map[string]interface{}{
						"c": 149.00,
					},
					"updated": 1707840000000,
				},
			})
		}))
		defer server.Close()

		vs := &VolumeService{
			apiKey:      "test-key",
			baseURL:     server.URL,
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			cache:       make(map[string]*cachedVolume),
			cacheExpiry: 1 * time.Minute,
		}

		data, err := vs.GetRealTimeVolume("AAPL")
		require.NoError(t, err)
		require.NotNil(t, data)

		assert.Equal(t, "AAPL", data.Symbol)
		assert.Equal(t, int64(55000000), data.Volume)
		assert.Equal(t, 149.75, data.VolumeWeighted)
		assert.Equal(t, 148.50, data.Open)
		assert.Equal(t, 151.25, data.Close)
		assert.Equal(t, 152.00, data.High)
		assert.Equal(t, 147.80, data.Low)
		assert.Equal(t, 149.00, data.PrevClose)
	})

	t.Run("calculates change and changePercent correctly", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "OK",
				"ticker": map[string]interface{}{
					"day": map[string]interface{}{
						"o": 100.0, "h": 110.0, "l": 95.0, "c": 105.0, "v": 1000, "vw": 102.5,
					},
					"prevDay": map[string]interface{}{"c": 100.0},
					"updated": 1707840000000,
				},
			})
		}))
		defer server.Close()

		vs := &VolumeService{
			apiKey:      "test-key",
			baseURL:     server.URL,
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			cache:       make(map[string]*cachedVolume),
			cacheExpiry: 1 * time.Minute,
		}

		data, err := vs.GetRealTimeVolume("TEST")
		require.NoError(t, err)

		assert.InDelta(t, 5.0, data.Change, 0.001)
		assert.InDelta(t, 5.0, data.ChangePercent, 0.001)
	})

	t.Run("handles zero previous close without panic", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "OK",
				"ticker": map[string]interface{}{
					"day": map[string]interface{}{
						"o": 10.0, "h": 10.0, "l": 10.0, "c": 10.0, "v": 100, "vw": 10.0,
					},
					"prevDay": map[string]interface{}{"c": 0.0},
					"updated": 1707840000000,
				},
			})
		}))
		defer server.Close()

		vs := &VolumeService{
			apiKey:      "test-key",
			baseURL:     server.URL,
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			cache:       make(map[string]*cachedVolume),
			cacheExpiry: 1 * time.Minute,
		}

		data, err := vs.GetRealTimeVolume("IPO")
		require.NoError(t, err)

		// When prevClose is 0, changePercent should be 0 (no divide by zero)
		assert.Equal(t, 0.0, data.ChangePercent)
		assert.Equal(t, 10.0, data.Change)
	})

	t.Run("returns cached data when available", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "OK",
				"ticker": map[string]interface{}{
					"day":     map[string]interface{}{"o": 1.0, "h": 1.0, "l": 1.0, "c": 1.0, "v": 1, "vw": 1.0},
					"prevDay": map[string]interface{}{"c": 1.0},
					"updated": 1707840000000,
				},
			})
		}))
		defer server.Close()

		vs := &VolumeService{
			apiKey:      "test-key",
			baseURL:     server.URL,
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			cache:       make(map[string]*cachedVolume),
			cacheExpiry: 1 * time.Minute,
		}

		// First call hits server
		_, err := vs.GetRealTimeVolume("MSFT")
		require.NoError(t, err)
		assert.Equal(t, 1, callCount)

		// Second call should use cache
		_, err = vs.GetRealTimeVolume("MSFT")
		require.NoError(t, err)
		assert.Equal(t, 1, callCount, "second call should use cache")
	})

	t.Run("returns error on rate limit (429)", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer server.Close()

		vs := &VolumeService{
			apiKey:      "test-key",
			baseURL:     server.URL,
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			cache:       make(map[string]*cachedVolume),
			cacheExpiry: 1 * time.Minute,
		}

		_, err := vs.GetRealTimeVolume("AAPL")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rate limit")
	})

	t.Run("returns error on non-200 status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		vs := &VolumeService{
			apiKey:      "test-key",
			baseURL:     server.URL,
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			cache:       make(map[string]*cachedVolume),
			cacheExpiry: 1 * time.Minute,
		}

		_, err := vs.GetRealTimeVolume("AAPL")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})

	t.Run("returns error on non-OK API status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "ERROR",
			})
		}))
		defer server.Close()

		vs := &VolumeService{
			apiKey:      "test-key",
			baseURL:     server.URL,
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			cache:       make(map[string]*cachedVolume),
			cacheExpiry: 1 * time.Minute,
		}

		_, err := vs.GetRealTimeVolume("BAD")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "non-OK")
	})
}

// ─── GetVolumeAggregates with mock server ────────────────────────

func TestVolumeServiceGetVolumeAggregates(t *testing.T) {
	t.Run("calculates averages and trend correctly", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.Path, "/v2/aggs/ticker/AAPL/range/1/day")
			assert.Equal(t, "true", r.URL.Query().Get("adjusted"))
			assert.Equal(t, "desc", r.URL.Query().Get("sort"))

			// Generate 90 bars of data
			results := make([]map[string]interface{}, 90)
			for i := 0; i < 90; i++ {
				results[i] = map[string]interface{}{
					"v": 1000000.0, // stable volume
					"h": 155.0 + float64(i)*0.1,
					"l": 145.0 - float64(i)*0.1,
				}
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "OK",
				"results": results,
			})
		}))
		defer server.Close()

		vs := &VolumeService{
			apiKey:      "test-key",
			baseURL:     server.URL,
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			cache:       make(map[string]*cachedVolume),
			cacheExpiry: 1 * time.Minute,
		}

		agg, err := vs.GetVolumeAggregates("AAPL", 90)
		require.NoError(t, err)
		require.NotNil(t, agg)

		assert.Equal(t, "AAPL", agg.Symbol)
		assert.Equal(t, int64(1000000), agg.AvgVolume30D)
		assert.Equal(t, int64(1000000), agg.AvgVolume90D)
		assert.Equal(t, "stable", agg.VolumeTrend) // 30d avg == 90d avg
	})

	t.Run("detects increasing volume trend", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			results := make([]map[string]interface{}, 90)
			for i := 0; i < 90; i++ {
				vol := 500000.0
				if i < 30 {
					vol = 2000000.0 // Recent 30 days have much higher volume
				}
				results[i] = map[string]interface{}{
					"v": vol,
					"h": 150.0,
					"l": 140.0,
				}
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "OK",
				"results": results,
			})
		}))
		defer server.Close()

		vs := &VolumeService{
			apiKey:      "test-key",
			baseURL:     server.URL,
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			cache:       make(map[string]*cachedVolume),
			cacheExpiry: 1 * time.Minute,
		}

		agg, err := vs.GetVolumeAggregates("TSLA", 90)
		require.NoError(t, err)
		assert.Equal(t, "increasing", agg.VolumeTrend)
	})

	t.Run("detects decreasing volume trend", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			results := make([]map[string]interface{}, 90)
			for i := 0; i < 90; i++ {
				vol := 2000000.0
				if i < 30 {
					vol = 500000.0 // Recent 30 days have much lower volume
				}
				results[i] = map[string]interface{}{
					"v": vol,
					"h": 150.0,
					"l": 140.0,
				}
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "OK",
				"results": results,
			})
		}))
		defer server.Close()

		vs := &VolumeService{
			apiKey:      "test-key",
			baseURL:     server.URL,
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			cache:       make(map[string]*cachedVolume),
			cacheExpiry: 1 * time.Minute,
		}

		agg, err := vs.GetVolumeAggregates("INTC", 90)
		require.NoError(t, err)
		assert.Equal(t, "decreasing", agg.VolumeTrend)
	})

	t.Run("calculates week52 high and low across all data", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			results := []map[string]interface{}{
				{"v": 100.0, "h": 200.0, "l": 150.0},
				{"v": 100.0, "h": 250.0, "l": 120.0}, // highest
				{"v": 100.0, "h": 180.0, "l": 100.0}, // lowest
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "OK",
				"results": results,
			})
		}))
		defer server.Close()

		vs := &VolumeService{
			apiKey:      "test-key",
			baseURL:     server.URL,
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			cache:       make(map[string]*cachedVolume),
			cacheExpiry: 1 * time.Minute,
		}

		agg, err := vs.GetVolumeAggregates("TEST", 365)
		require.NoError(t, err)
		assert.Equal(t, 250.0, agg.Week52High)
		assert.Equal(t, 100.0, agg.Week52Low)
	})

	t.Run("returns error when no data available", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "OK",
				"results": []interface{}{},
			})
		}))
		defer server.Close()

		vs := &VolumeService{
			apiKey:      "test-key",
			baseURL:     server.URL,
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			cache:       make(map[string]*cachedVolume),
			cacheExpiry: 1 * time.Minute,
		}

		_, err := vs.GetVolumeAggregates("EMPTY", 90)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no data available")
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()

		vs := &VolumeService{
			apiKey:      "test-key",
			baseURL:     server.URL,
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			cache:       make(map[string]*cachedVolume),
			cacheExpiry: 1 * time.Minute,
		}

		_, err := vs.GetVolumeAggregates("AAPL", 90)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "403")
	})

	t.Run("handles single data point", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			results := []map[string]interface{}{
				{"v": 5000.0, "h": 50.0, "l": 45.0},
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "OK",
				"results": results,
			})
		}))
		defer server.Close()

		vs := &VolumeService{
			apiKey:      "test-key",
			baseURL:     server.URL,
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			cache:       make(map[string]*cachedVolume),
			cacheExpiry: 1 * time.Minute,
		}

		agg, err := vs.GetVolumeAggregates("PENNY", 30)
		require.NoError(t, err)
		assert.Equal(t, int64(5000), agg.AvgVolume30D)
		assert.Equal(t, int64(5000), agg.AvgVolume90D)
		assert.Equal(t, 50.0, agg.Week52High)
		assert.Equal(t, 45.0, agg.Week52Low)
	})

	t.Run("handles zero volume data", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			results := []map[string]interface{}{
				{"v": 0.0, "h": 10.0, "l": 9.0},
				{"v": 0.0, "h": 11.0, "l": 8.0},
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "OK",
				"results": results,
			})
		}))
		defer server.Close()

		vs := &VolumeService{
			apiKey:      "test-key",
			baseURL:     server.URL,
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			cache:       make(map[string]*cachedVolume),
			cacheExpiry: 1 * time.Minute,
		}

		agg, err := vs.GetVolumeAggregates("DEAD", 30)
		require.NoError(t, err)
		assert.Equal(t, int64(0), agg.AvgVolume30D)
		assert.Equal(t, int64(0), agg.AvgVolume90D)
	})
}

// ─── VolumeData struct ───────────────────────────────────────────

func TestVolumeDataSerialization(t *testing.T) {
	data := VolumeData{
		Symbol:         "AAPL",
		Volume:         50000000,
		VolumeWeighted: 149.75,
		Open:           148.50,
		Close:          151.25,
		High:           152.00,
		Low:            147.80,
		Timestamp:      1707840000000,
		Transactions:   1500,
		PrevClose:      149.00,
		Change:         2.25,
		ChangePercent:  1.51,
		UpdatedAt:      time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC),
	}

	jsonBytes, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded VolumeData
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, data.Symbol, decoded.Symbol)
	assert.Equal(t, data.Volume, decoded.Volume)
	assert.Equal(t, data.Change, decoded.Change)
	assert.Equal(t, data.ChangePercent, decoded.ChangePercent)
}

func TestVolumeAggregatesSerialization(t *testing.T) {
	agg := VolumeAggregates{
		Symbol:       "GOOG",
		AvgVolume30D: 25000000,
		AvgVolume90D: 22000000,
		Week52High:   191.75,
		Week52Low:    120.21,
		VolumeTrend:  "increasing",
	}

	jsonBytes, err := json.Marshal(agg)
	require.NoError(t, err)

	var decoded VolumeAggregates
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, agg.Symbol, decoded.Symbol)
	assert.Equal(t, agg.AvgVolume30D, decoded.AvgVolume30D)
	assert.Equal(t, agg.VolumeTrend, decoded.VolumeTrend)
}
