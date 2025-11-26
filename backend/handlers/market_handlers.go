package handlers

import (
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"investorcenter-api/services"
)

// MoversCache caches market movers data with TTL
type MoversCache struct {
	mu       sync.RWMutex
	data     *MoversData
	cachedAt time.Time
	cacheTTL time.Duration
}

type MoversData struct {
	Gainers    []MoverStock `json:"gainers"`
	Losers     []MoverStock `json:"losers"`
	MostActive []MoverStock `json:"mostActive"`
}

type MoverStock struct {
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name,omitempty"`
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"changePercent"`
	Volume        float64 `json:"volume"`
}

var moversCache = &MoversCache{
	cacheTTL: 5 * time.Minute,
}

func (c *MoversCache) get() *MoversData {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.data == nil || time.Since(c.cachedAt) > c.cacheTTL {
		return nil
	}
	return c.data
}

func (c *MoversCache) set(data *MoversData) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = data
	c.cachedAt = time.Now()
}

// IndexInfo represents market index information
type IndexInfo struct {
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"changePercent"`
	LastUpdated   string  `json:"lastUpdated"`
}

// GetMarketIndices fetches current market indices from Polygon.io
func GetMarketIndices(c *gin.Context) {
	polygonClient := services.NewPolygonClient()

	// Define major market indices using ETF proxies
	// Note: Using ETFs as proxies since indices require premium Polygon.io plan
	// ETFs track indices very closely and provide real-time data
	indexSymbols := []struct {
		Symbol string
		Name   string
	}{
		{"SPY", "S&P 500"},   // SPDR S&P 500 ETF Trust (tracks S&P 500)
		{"DIA", "Dow Jones"}, // SPDR Dow Jones Industrial Average ETF (tracks DJIA)
		{"QQQ", "NASDAQ"},    // Invesco QQQ Trust (tracks NASDAQ-100)
	}

	indices := []IndexInfo{}
	var fetchErrors []string

	for _, idx := range indexSymbols {
		priceData, err := polygonClient.GetStockRealTimePrice(idx.Symbol)
		if err != nil {
			errMsg := "Failed to fetch " + idx.Name + " (" + idx.Symbol + "): " + err.Error()
			log.Printf("Warning: %s", errMsg)
			fetchErrors = append(fetchErrors, errMsg)
			// Skip this index if we can't fetch it
			continue
		}

		index := IndexInfo{
			Symbol:        idx.Symbol,
			Name:          idx.Name,
			Price:         priceData.Price.InexactFloat64(),
			Change:        priceData.Change.InexactFloat64(),
			ChangePercent: priceData.ChangePercent.InexactFloat64(),
			LastUpdated:   priceData.Timestamp.Format(time.RFC3339),
		}

		indices = append(indices, index)
		log.Printf("Successfully fetched %s: $%.2f (%.2f%%)", idx.Name, index.Price, index.ChangePercent)
	}

	// If we couldn't fetch any indices, return an error with details
	if len(indices) == 0 {
		log.Printf("Error: Failed to fetch any market indices. Errors: %v", fetchErrors)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Failed to fetch market indices from Polygon.io. Please check API key and connectivity.",
			"details": fetchErrors,
			"meta": gin.H{
				"timestamp": time.Now().UTC(),
			},
		})
		return
	}

	// Return successfully fetched indices (even if some failed)
	c.JSON(http.StatusOK, gin.H{
		"data": indices,
		"meta": gin.H{
			"count":     len(indices),
			"timestamp": time.Now().UTC(),
			"source":    "polygon.io",
		},
	})
}

// GetMarketMovers returns top gainers, losers, and most active stocks
func GetMarketMovers(c *gin.Context) {
	// Check cache first
	if cached := moversCache.get(); cached != nil {
		log.Printf("Returning cached market movers data")
		c.JSON(http.StatusOK, gin.H{
			"data": cached,
			"meta": gin.H{
				"timestamp": time.Now().UTC(),
				"source":    "polygon.io",
				"cached":    true,
			},
		})
		return
	}

	// Parse limit parameter (default 5)
	limitStr := c.DefaultQuery("limit", "5")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 20 {
		limit = 5
	}

	polygonClient := services.NewPolygonClient()

	// Fetch bulk stock snapshots
	snapshots, err := polygonClient.GetBulkStockSnapshots()
	if err != nil {
		log.Printf("Error fetching bulk stock snapshots: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Failed to fetch market movers from Polygon.io",
			"meta": gin.H{
				"timestamp": time.Now().UTC(),
			},
		})
		return
	}

	// Convert to MoverStock slice
	var stocks []MoverStock
	for _, ticker := range snapshots.Tickers {
		// Filter out invalid or suspicious data
		if ticker.Day.Close <= 0 || math.IsNaN(ticker.TodaysChangePerc) || math.IsInf(ticker.TodaysChangePerc, 0) {
			continue
		}

		// Filter out stocks with extreme movements (likely data errors or penny stocks)
		if math.Abs(ticker.TodaysChangePerc) > 100 {
			continue
		}

		// Filter out very low volume stocks (less than 100k)
		if ticker.Day.Volume < 100000 {
			continue
		}

		// Filter out penny stocks (price under $1)
		if ticker.Day.Close < 1.0 {
			continue
		}

		stocks = append(stocks, MoverStock{
			Symbol:        ticker.Ticker,
			Price:         ticker.Day.Close,
			Change:        ticker.TodaysChange,
			ChangePercent: ticker.TodaysChangePerc,
			Volume:        ticker.Day.Volume,
		})
	}

	// Sort by change percent (descending) for gainers
	sort.Slice(stocks, func(i, j int) bool {
		return stocks[i].ChangePercent > stocks[j].ChangePercent
	})

	// Get top gainers
	gainers := make([]MoverStock, 0, limit)
	for i := 0; i < len(stocks) && len(gainers) < limit; i++ {
		if stocks[i].ChangePercent > 0 {
			gainers = append(gainers, stocks[i])
		}
	}

	// Get top losers (from the end of sorted list)
	losers := make([]MoverStock, 0, limit)
	for i := len(stocks) - 1; i >= 0 && len(losers) < limit; i-- {
		if stocks[i].ChangePercent < 0 {
			losers = append(losers, stocks[i])
		}
	}

	// Sort by volume for most active
	sort.Slice(stocks, func(i, j int) bool {
		return stocks[i].Volume > stocks[j].Volume
	})

	// Get most active
	mostActive := make([]MoverStock, 0, limit)
	for i := 0; i < len(stocks) && len(mostActive) < limit; i++ {
		mostActive = append(mostActive, stocks[i])
	}

	moversData := &MoversData{
		Gainers:    gainers,
		Losers:     losers,
		MostActive: mostActive,
	}

	// Cache the results
	moversCache.set(moversData)

	log.Printf("Fetched market movers: %d gainers, %d losers, %d most active",
		len(gainers), len(losers), len(mostActive))

	c.JSON(http.StatusOK, gin.H{
		"data": moversData,
		"meta": gin.H{
			"timestamp": time.Now().UTC(),
			"source":    "polygon.io",
			"cached":    false,
		},
	})
}
