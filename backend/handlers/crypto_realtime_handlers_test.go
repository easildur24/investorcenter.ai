package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMiniRedis starts a miniredis instance and swaps the package-level redisClient.
func setupMiniRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)

	origClient := redisClient
	redisClient = redis.NewClient(&redis.Options{Addr: mr.Addr()})

	t.Cleanup(func() {
		redisClient = origClient
		mr.Close()
	})

	return mr
}

func setupCryptoRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/crypto/:symbol/price", GetCryptoRealTimePrice)
	r.GET("/api/v1/crypto/prices", GetAllCryptoRealTimePrices)
	r.GET("/api/v1/crypto/stream", StreamCryptoPrices)
	return r
}

func TestGetCryptoRealTimePrice_Success(t *testing.T) {
	mr := setupMiniRedis(t)
	router := setupCryptoRouter()

	priceData := CryptoRealTimePrice{
		Symbol:                   "BTC",
		Name:                     "Bitcoin",
		CurrentPrice:             67500.50,
		MarketCap:                1300000000000,
		TotalVolume:              25000000000,
		PriceChangePercentage24h: 2.5,
		LastUpdated:              "2024-12-15T10:00:00Z",
		Source:                   "coingecko",
	}
	data, _ := json.Marshal(priceData)
	mr.Set("crypto:quote:BTC", string(data))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/crypto/BTC/price", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp CryptoRealTimePrice
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "BTC", resp.Symbol)
	assert.Equal(t, 67500.50, resp.CurrentPrice)
	// Alias fields should be populated
	assert.Equal(t, 67500.50, resp.Price)
	assert.Equal(t, 25000000000.0, resp.Volume24h)
	assert.Equal(t, 2.5, resp.Change24h)
}

func TestGetCryptoRealTimePrice_NotFound(t *testing.T) {
	_ = setupMiniRedis(t) // empty Redis
	router := setupCryptoRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/crypto/UNKNOWN/price", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetCryptoRealTimePrice_InvalidJSON(t *testing.T) {
	mr := setupMiniRedis(t)
	router := setupCryptoRouter()

	mr.Set("crypto:quote:BTC", "not valid json")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/crypto/BTC/price", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetCryptoRealTimePrice_CaseInsensitive(t *testing.T) {
	mr := setupMiniRedis(t)
	router := setupCryptoRouter()

	priceData := CryptoRealTimePrice{Symbol: "BTC", CurrentPrice: 67500.0, Source: "coingecko"}
	data, _ := json.Marshal(priceData)
	mr.Set("crypto:quote:BTC", string(data))

	// Request with lowercase
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/crypto/btc/price", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetAllCryptoRealTimePrices_Success(t *testing.T) {
	mr := setupMiniRedis(t)
	router := setupCryptoRouter()
	ctx := context.Background()

	// Add symbols to sorted set
	redisClient.ZAdd(ctx, "crypto:symbols:ranked", &redis.Z{Score: 1, Member: "BTC"})
	redisClient.ZAdd(ctx, "crypto:symbols:ranked", &redis.Z{Score: 2, Member: "ETH"})

	btcData := CryptoRealTimePrice{Symbol: "BTC", CurrentPrice: 67500.0, Source: "coingecko"}
	ethData := CryptoRealTimePrice{Symbol: "ETH", CurrentPrice: 3800.0, Source: "coingecko"}
	btcJSON, _ := json.Marshal(btcData)
	ethJSON, _ := json.Marshal(ethData)
	mr.Set("crypto:quote:BTC", string(btcJSON))
	mr.Set("crypto:quote:ETH", string(ethJSON))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/crypto/prices", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(2), resp["count"])
}

func TestGetAllCryptoRealTimePrices_Empty(t *testing.T) {
	_ = setupMiniRedis(t) // empty Redis
	router := setupCryptoRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/crypto/prices", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["count"])
}

func TestGetAllCryptoRealTimePrices_PartialData(t *testing.T) {
	mr := setupMiniRedis(t)
	router := setupCryptoRouter()
	ctx := context.Background()

	// 3 symbols in sorted set, only 2 have data
	redisClient.ZAdd(ctx, "crypto:symbols:ranked",
		&redis.Z{Score: 1, Member: "BTC"},
		&redis.Z{Score: 2, Member: "ETH"},
		&redis.Z{Score: 3, Member: "SOL"},
	)

	btcData := CryptoRealTimePrice{Symbol: "BTC", CurrentPrice: 67500.0, Source: "coingecko"}
	ethData := CryptoRealTimePrice{Symbol: "ETH", CurrentPrice: 3800.0, Source: "coingecko"}
	btcJSON, _ := json.Marshal(btcData)
	ethJSON, _ := json.Marshal(ethData)
	mr.Set("crypto:quote:BTC", string(btcJSON))
	mr.Set("crypto:quote:ETH", string(ethJSON))
	// SOL has no data

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/crypto/prices", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(2), resp["count"])
}

func TestStreamCryptoPrices_Headers(t *testing.T) {
	_ = setupMiniRedis(t)
	router := setupCryptoRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/crypto/stream?symbols=BTC,ETH", nil)

	// Use a context with cancel to disconnect immediately
	ctx, cancel := context.WithCancel(req.Context())
	cancel() // Cancel immediately so the SSE loop exits
	req = req.WithContext(ctx)

	router.ServeHTTP(w, req)

	// Verify SSE headers were set
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", w.Header().Get("Connection"))
}
