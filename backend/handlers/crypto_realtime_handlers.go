package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// Redis client for crypto prices
var redisClient *redis.Client

func init() {
	// Initialize Redis client with configurable options
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"), // optional
		DB:       0,                           // default DB
	})
}

// CryptoRealTimePrice represents real-time price data
type CryptoRealTimePrice struct {
	Symbol           string  `json:"symbol"`
	Name             string  `json:"name,omitempty"`
	Price            float64 `json:"price"`
	PriceBTC         float64 `json:"price_btc,omitempty"`
	Rank             int     `json:"rank,omitempty"`
	Change1h         float64 `json:"change_1h,omitempty"`
	Change24h        float64 `json:"change_24h"`
	Change7d         float64 `json:"change_7d,omitempty"`
	Volume24h        float64 `json:"volume_24h"`
	MarketCap        float64 `json:"market_cap,omitempty"`
	CirculatingSupply float64 `json:"circulating_supply,omitempty"`
	TotalSupply      float64 `json:"total_supply,omitempty"`
	MaxSupply        float64 `json:"max_supply,omitempty"`
	LastUpdated      string  `json:"last_updated"`
	UpdateInterval   int     `json:"update_interval,omitempty"`
	Tier             string  `json:"tier,omitempty"`
	Source           string  `json:"source"`
}

// GetCryptoRealTimePrice handles GET /api/v1/crypto/:symbol/price
func GetCryptoRealTimePrice(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))

	// Get real-time price from Redis
	ctx := context.Background()
	// Try the new key format first (from crypto_complete_service.py)
	priceKey := fmt.Sprintf("crypto:quote:%s", symbol)

	priceData, err := redisClient.Get(ctx, priceKey).Result()
	if err == redis.Nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Real-time price not available for %s", symbol),
		})
		return
	} else if err != nil {
		log.Printf("Redis error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch price",
		})
		return
	}

	// Parse and return price data
	var price CryptoRealTimePrice
	if err := json.Unmarshal([]byte(priceData), &price); err != nil {
		log.Printf("Failed to parse price data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid price data",
		})
		return
	}

	c.JSON(http.StatusOK, price)
}

// GetAllCryptoRealTimePrices handles GET /api/v1/crypto/prices
func GetAllCryptoRealTimePrices(c *gin.Context) {
	ctx := context.Background()

	// Get all crypto symbols from Redis
	symbols, err := redisClient.ZRange(ctx, "crypto:symbols:ranked", 0, -1).Result()
	if err != nil {
		log.Printf("Failed to get symbols: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch symbols",
		})
		return
	}

	// Fetch prices for all symbols
	prices := make(map[string]interface{})

	// Use pipeline for efficiency
	pipe := redisClient.Pipeline()
	for _, symbol := range symbols {
		pipe.Get(ctx, fmt.Sprintf("crypto:quote:%s", symbol))
	}

	results, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		log.Printf("Pipeline error: %v", err)
	}

	// Process results
	for i, symbol := range symbols {
		if i < len(results) {
			if val, err := results[i].(*redis.StringCmd).Result(); err == nil {
				var price CryptoRealTimePrice
				if json.Unmarshal([]byte(val), &price) == nil {
					prices[symbol] = price
				}
			}
		}
	}

	// Return JSON response
	response := gin.H{
		"timestamp": time.Now().UTC(),
		"count":     len(prices),
		"prices":    prices,
	}

	c.JSON(http.StatusOK, response)
}

// StreamCryptoPrices handles SSE endpoint for real-time price streaming
func StreamCryptoPrices(c *gin.Context) {
	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Create channel for client disconnect
	clientGone := c.Request.Context().Done()

	// Get requested symbols from query params
	symbolsParam := c.Query("symbols")
	var symbols []string
	if symbolsParam != "" {
		symbols = strings.Split(symbolsParam, ",")
	} else {
		// Default to top cryptocurrencies
		symbols = []string{"BTC", "ETH", "BNB", "XRP", "SOL", "ADA", "DOGE", "MATIC", "DOT", "SHIB"}
	}

	// Start streaming
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-clientGone:
			log.Printf("Client disconnected from SSE")
			return
		case <-ticker.C:
			// Fetch current prices
			ctx := context.Background()
			prices := make(map[string]interface{})

			for _, symbol := range symbols {
				priceKey := fmt.Sprintf("crypto:quote:%s", strings.ToUpper(symbol))
				if priceData, err := redisClient.Get(ctx, priceKey).Result(); err == nil {
					var price CryptoRealTimePrice
					if json.Unmarshal([]byte(priceData), &price) == nil {
						prices[symbol] = map[string]interface{}{
							"price":  price.Price,
							"change": price.Change24h,
						}
					}
				}
			}

			// Format as SSE
			data, _ := json.Marshal(prices)
			fmt.Fprintf(c.Writer, "data: %s\n\n", string(data))
			c.Writer.Flush()
		}
	}
}