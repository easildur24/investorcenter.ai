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
	"investorcenter-api/database"
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
	DisplayFormat string  `json:"displayFormat"` // "points" for indices, "usd" for ETF proxies
	DataType      string  `json:"dataType"`      // "index" or "etf_proxy"
}

// GetMarketIndices fetches current market indices from Polygon.io
// Attempts to use real index values (I:SPX, I:DJI, etc.), falls back to ETF proxies
func GetMarketIndices(c *gin.Context) {
	polygonClient := services.NewPolygonClient()

	// Try real index snapshots first
	indexSymbols := []struct {
		Symbol   string
		Name     string
		ETFProxy string // Fallback ETF symbol
	}{
		{"I:SPX", "S&P 500", "SPY"},
		{"I:DJI", "Dow Jones", "DIA"},
		{"I:COMP", "NASDAQ", "QQQ"},
		{"I:RUT", "Russell 2000", "IWM"},
		{"I:VIX", "VIX", "VIXY"},
	}

	// Collect all index tickers for a batch call
	indexTickers := make([]string, len(indexSymbols))
	for i, idx := range indexSymbols {
		indexTickers[i] = idx.Symbol
	}

	indexResults, indexErr := polygonClient.GetIndexSnapshots(indexTickers)
	if indexErr != nil {
		log.Printf("Warning: Failed to fetch index snapshots, falling back to ETF proxies: %v", indexErr)
	}

	// Build a map for quick lookup of index results
	indexMap := make(map[string]services.IndexSnapshotResult)
	for _, r := range indexResults {
		indexMap[r.Symbol] = r
	}

	indices := []IndexInfo{}
	var fetchErrors []string

	for _, idx := range indexSymbols {
		// Try index data first — skip if value is 0 (weekends/holidays return all zeros)
		if result, ok := indexMap[idx.Symbol]; ok && result.Value != 0 {
			indices = append(indices, IndexInfo{
				Symbol:        idx.Symbol,
				Name:          idx.Name,
				Price:         result.Value,
				Change:        result.Change,
				ChangePercent: result.ChangePercent,
				LastUpdated:   result.Timestamp.Format(time.RFC3339),
				DisplayFormat: "points",
				DataType:      "index",
			})
			log.Printf("Successfully fetched %s (index): %.2f (%.2f%%)", idx.Name, result.Value, result.ChangePercent)
			continue
		}

		// Fallback to ETF proxy (also used when index returns zero on weekends/holidays)
		if _, exists := indexMap[idx.Symbol]; exists {
			log.Printf("Index %s returned zero value, falling back to ETF proxy %s", idx.Symbol, idx.ETFProxy)
		}
		if idx.ETFProxy != "" {
			priceData, err := polygonClient.GetStockRealTimePrice(idx.ETFProxy)
			if err != nil {
				errMsg := "Failed to fetch " + idx.Name + " (" + idx.ETFProxy + "): " + err.Error()
				log.Printf("Warning: %s", errMsg)
				fetchErrors = append(fetchErrors, errMsg)
				continue
			}

			indices = append(indices, IndexInfo{
				Symbol:        idx.ETFProxy,
				Name:          idx.Name,
				Price:         priceData.Price.InexactFloat64(),
				Change:        priceData.Change.InexactFloat64(),
				ChangePercent: priceData.ChangePercent.InexactFloat64(),
				LastUpdated:   priceData.Timestamp.Format(time.RFC3339),
				DisplayFormat: "usd",
				DataType:      "etf_proxy",
			})
			log.Printf("Successfully fetched %s (ETF proxy %s): $%.2f (%.2f%%)", idx.Name, idx.ETFProxy, priceData.Price.InexactFloat64(), priceData.ChangePercent.InexactFloat64())
		}
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
		// Use Day.Close if available (after market close), else LastTrade.Price (during trading)
		price := ticker.Day.Close
		if price == 0 {
			price = ticker.LastTrade.Price
		}

		// Filter out invalid or suspicious data
		if price <= 0 || math.IsNaN(ticker.TodaysChangePerc) || math.IsInf(ticker.TodaysChangePerc, 0) {
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
		if price < 1.0 {
			continue
		}

		stocks = append(stocks, MoverStock{
			Symbol:        ticker.Ticker,
			Price:         price,
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

	// Look up company names from the database for all mover stocks
	allMovers := make([]MoverStock, 0, len(gainers)+len(losers)+len(mostActive))
	allMovers = append(allMovers, gainers...)
	allMovers = append(allMovers, losers...)
	allMovers = append(allMovers, mostActive...)

	symbolSet := make(map[string]bool)
	var symbols []string
	for _, m := range allMovers {
		if !symbolSet[m.Symbol] {
			symbolSet[m.Symbol] = true
			symbols = append(symbols, m.Symbol)
		}
	}

	if len(symbols) > 0 {
		names, err := database.GetCompanyNames(symbols)
		if err != nil {
			log.Printf("Warning: Failed to look up company names: %v", err)
		} else {
			for i := range gainers {
				if name, ok := names[gainers[i].Symbol]; ok {
					gainers[i].Name = name
				}
			}
			for i := range losers {
				if name, ok := names[losers[i].Symbol]; ok {
					losers[i].Name = name
				}
			}
			for i := range mostActive {
				if name, ok := names[mostActive[i].Symbol]; ok {
					mostActive[i].Name = name
				}
			}
		}
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
