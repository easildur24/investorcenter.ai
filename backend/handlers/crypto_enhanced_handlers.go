package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// EnhancedCryptoPrice includes timestamp information
type EnhancedCryptoPrice struct {
	Symbol                    string    `json:"symbol"`
	ID                       string    `json:"id,omitempty"`
	Name                     string    `json:"name"`
	Image                    string    `json:"image,omitempty"`
	CurrentPrice             float64   `json:"current_price"`
	MarketCap                float64   `json:"market_cap"`
	MarketCapRank            int       `json:"market_cap_rank,omitempty"`
	FullyDilutedValuation    float64   `json:"fully_diluted_valuation,omitempty"`
	TotalVolume              float64   `json:"total_volume"`
	High24h                  float64   `json:"high_24h"`
	Low24h                   float64   `json:"low_24h"`
	PriceChange24h           float64   `json:"price_change_24h"`
	PriceChangePercentage24h float64   `json:"price_change_percentage_24h"`
	PriceChangePercentage1h  float64   `json:"price_change_percentage_1h,omitempty"`
	PriceChangePercentage7d  float64   `json:"price_change_percentage_7d,omitempty"`
	PriceChangePercentage30d float64   `json:"price_change_percentage_30d,omitempty"`
	CirculatingSupply        float64   `json:"circulating_supply"`
	TotalSupply              float64   `json:"total_supply"`
	MaxSupply                *float64  `json:"max_supply"`
	ATH                      float64   `json:"ath,omitempty"`
	ATHChangePercentage      float64   `json:"ath_change_percentage,omitempty"`
	ATHDate                  string    `json:"ath_date,omitempty"`
	ATL                      float64   `json:"atl,omitempty"`
	ATLChangePercentage      float64   `json:"atl_change_percentage,omitempty"`
	ATLDate                  string    `json:"atl_date,omitempty"`
	LastUpdated              string    `json:"last_updated"`
	FetchedAt                string    `json:"fetched_at"`     // When we fetched from CoinGecko
	DataAge                  int       `json:"data_age_seconds"` // How old the data is
	Source                   string    `json:"source"`
	FromCache                bool      `json:"from_cache"`
}

// GetEnhancedCryptoPrice handles GET /api/v1/crypto/:symbol/quote
// Implements smart caching with on-demand updates
func GetEnhancedCryptoPrice(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	ctx := context.Background()

	// Check cache first
	priceKey := fmt.Sprintf("crypto:quote:%s", symbol)
	priceData, err := redisClient.Get(ctx, priceKey).Result()

	var response EnhancedCryptoPrice
	fromCache := true

	if err == redis.Nil {
		// No data in cache - try to trigger on-demand update
		log.Printf("No cached data for %s, attempting on-demand fetch", symbol)

		// Check rate limit status
		if canMakeAPICall(ctx) {
			// Trigger async update (in production, this would call the Python service)
			go triggerOnDemandUpdate(symbol)

			// Wait briefly for update (max 2 seconds)
			for i := 0; i < 4; i++ {
				time.Sleep(500 * time.Millisecond)
				priceData, err = redisClient.Get(ctx, priceKey).Result()
				if err == nil {
					fromCache = false
					break
				}
			}
		}

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("No data available for %s", symbol),
				"message": "Coin not found or not yet indexed",
			})
			return
		}
	} else if err != nil {
		log.Printf("Redis error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch price data",
		})
		return
	}

	// Parse the cached data
	if err := json.Unmarshal([]byte(priceData), &response); err != nil {
		log.Printf("Failed to unmarshal price data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid price data format",
		})
		return
	}

	// Calculate data age
	if response.FetchedAt != "" {
		fetchedTime, err := time.Parse(time.RFC3339, response.FetchedAt)
		if err == nil {
			response.DataAge = int(time.Since(fetchedTime).Seconds())
		}
	}

	// Check if data is stale and needs refresh
	if response.DataAge > 120 && canMakeAPICall(ctx) { // Data older than 2 minutes
		// Trigger background update
		go triggerOnDemandUpdate(symbol)

		// Mark that we're serving stale data but update is in progress
		response.FromCache = true
		c.Header("X-Data-Status", "stale-updating")
	} else {
		response.FromCache = fromCache
		if fromCache {
			c.Header("X-Data-Status", "cached")
		} else {
			c.Header("X-Data-Status", "fresh")
		}
	}

	// Add response headers for transparency
	c.Header("X-Data-Age", fmt.Sprintf("%d", response.DataAge))
	c.Header("X-Data-Source", response.Source)

	c.JSON(http.StatusOK, response)
}

// GetAllCryptoQuotes handles GET /api/v1/crypto/quotes
// Returns all available crypto quotes with pagination
func GetAllCryptoQuotes(c *gin.Context) {
	ctx := context.Background()

	// Get pagination parameters
	page := 1
	limit := 100
	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
		if limit > 500 {
			limit = 500 // Max limit
		}
	}

	// Get sorting parameters (for future implementation)
	// sortBy := c.DefaultQuery("sort", "market_cap_rank")
	// order := c.DefaultQuery("order", "asc")

	// Use SCAN to get all crypto keys
	var cursor uint64
	var allKeys []string
	pattern := "crypto:quote:*"

	for {
		var keys []string
		var err error
		keys, cursor, err = redisClient.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			log.Printf("Redis scan error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch crypto data",
			})
			return
		}

		allKeys = append(allKeys, keys...)

		if cursor == 0 {
			break
		}
	}

	// Get data for all keys
	var quotes []EnhancedCryptoPrice
	for _, key := range allKeys {
		data, err := redisClient.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var quote EnhancedCryptoPrice
		if err := json.Unmarshal([]byte(data), &quote); err != nil {
			continue
		}

		// Calculate data age
		if quote.FetchedAt != "" {
			fetchedTime, err := time.Parse(time.RFC3339, quote.FetchedAt)
			if err == nil {
				quote.DataAge = int(time.Since(fetchedTime).Seconds())
			}
		}
		quote.FromCache = true

		quotes = append(quotes, quote)
	}

	// Sort quotes based on parameters
	// (In production, implement proper sorting logic here)

	// Apply pagination
	start := (page - 1) * limit
	end := start + limit
	if end > len(quotes) {
		end = len(quotes)
	}
	if start > len(quotes) {
		start = len(quotes)
	}

	paginatedQuotes := quotes[start:end]

	// Get statistics
	stats, _ := redisClient.Get(ctx, "crypto:stats:last_update").Result()
	var statsData map[string]interface{}
	json.Unmarshal([]byte(stats), &statsData)

	c.JSON(http.StatusOK, gin.H{
		"data": paginatedQuotes,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       len(quotes),
			"total_pages": (len(quotes) + limit - 1) / limit,
		},
		"stats": statsData,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// GetCryptoStats returns statistics about the crypto data
func GetCryptoStats(c *gin.Context) {
	ctx := context.Background()

	// Get statistics from Redis
	stats, err := redisClient.Get(ctx, "crypto:stats:last_update").Result()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "No statistics available",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	var statsData map[string]interface{}
	if err := json.Unmarshal([]byte(stats), &statsData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse statistics",
		})
		return
	}

	// Add current timestamp
	statsData["current_time"] = time.Now().UTC().Format(time.RFC3339)

	// Calculate time since last update
	if lastUpdate, ok := statsData["last_full_update"].(string); ok {
		if updateTime, err := time.Parse(time.RFC3339, lastUpdate); err == nil {
			statsData["seconds_since_update"] = int(time.Since(updateTime).Seconds())
		}
	}

	c.JSON(http.StatusOK, statsData)
}

// Helper function to check if we can make an API call (rate limiting)
func canMakeAPICall(ctx context.Context) bool {
	// Check rate limit counter in Redis
	key := "crypto:rate_limit:counter"
	count, _ := redisClient.Get(ctx, key).Int()

	// Free tier: 10 calls per minute
	if count >= 10 {
		return false
	}

	// Increment counter with 60 second expiry
	pipe := redisClient.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 60*time.Second)
	pipe.Exec(ctx)

	return true
}

// Helper function to trigger on-demand update
func triggerOnDemandUpdate(symbol string) {
	// In production, this would send a message to the Python service
	// to fetch fresh data for this specific symbol
	// For now, we'll just log it
	log.Printf("Triggering on-demand update for %s", symbol)

	// You could implement this via:
	// 1. Redis pub/sub to notify the Python service
	// 2. HTTP call to a Python service endpoint
	// 3. Message queue (RabbitMQ, etc.)

	ctx := context.Background()
	updateKey := fmt.Sprintf("crypto:update_request:%s", symbol)
	redisClient.SetEX(ctx, updateKey, "pending", 60*time.Second)
}