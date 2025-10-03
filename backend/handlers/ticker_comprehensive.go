package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/shopspring/decimal"
	"investorcenter-api/models"
	"investorcenter-api/services"
)

// Use the redisClient from crypto_realtime_handlers.go

// GetTicker returns comprehensive ticker data with real-time prices
func GetTicker(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	log.Printf("GetTicker called for symbol: %s", symbol)

	// Use service layer for database operations
	stockService := services.NewStockService()
	stock, err := stockService.GetStockBySymbol(c.Request.Context(), symbol)
	if err != nil {
		log.Printf("Stock not found in database: %v", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":  "Stock not found",
			"symbol": symbol,
		})
		return
	}

	// Determine if this is crypto or stock
	isCrypto := isCryptoAsset(stock.AssetType, symbol)

	// Get real-time price data from Polygon
	polygonClient := services.NewPolygonClient()
	priceData, priceErr := polygonClient.GetQuote(symbol)

	// Check if market is open (affects whether we show real-time data)
	var marketStatus string
	var shouldUpdateRealtime bool

	if isCrypto {
		marketStatus = "open" // Crypto markets are always open
		shouldUpdateRealtime = true
	} else {
		isOpen := polygonClient.IsMarketOpen()
		if isOpen {
			marketStatus = "open"
			shouldUpdateRealtime = true
		} else {
			marketStatus = "closed"
			shouldUpdateRealtime = false // Still get last close data, but mark as stale
		}
	}

	// If real-time data fails, generate mock data
	if priceErr != nil {
		log.Printf("Failed to get real-time price data for %s: %v", symbol, priceErr)
		priceData = generateMockPrice(symbol, stock)
	}

	// Get fundamentals (try real data first, fallback to mock)
	var fundamentals *models.Fundamentals
	if !isCrypto { // Only get fundamentals for stocks, not crypto
		fundamentals, err = polygonClient.GetFundamentals(symbol)
		if err != nil {
			log.Printf("Failed to get fundamentals for %s: %v", symbol, err)
			fundamentals = generateMockFundamentals(symbol)
		}
	} else {
		// For crypto, create minimal fundamentals
		fundamentals = &models.Fundamentals{
			Symbol:    symbol,
			Period:    "N/A",
			Year:      time.Now().Year(),
			UpdatedAt: time.Now(),
		}
	}

	// Build comprehensive response
	response := gin.H{
		"success": true,
		"data": gin.H{
			"summary": gin.H{
				"stock": gin.H{
					"symbol":    stock.Symbol,
					"name":      stock.Name,
					"exchange":  stock.Exchange,
					"sector":    stock.Sector,
					"assetType": stock.AssetType,
					"isCrypto":  isCrypto,
				},
				"price": gin.H{
					"price":         priceData.Price.String(),
					"open":          priceData.Open.String(),
					"high":          priceData.High.String(),
					"low":           priceData.Low.String(),
					"close":         priceData.Close.String(),
					"volume":        priceData.Volume,
					"change":        priceData.Change.String(),
					"changePercent": priceData.ChangePercent.String(),
					"timestamp":     priceData.Timestamp.Unix(),
					"lastUpdated":   priceData.Timestamp.Format(time.RFC3339),
				},
				"market": gin.H{
					"status":               marketStatus,
					"shouldUpdateRealtime": shouldUpdateRealtime,
					"updateInterval":       getUpdateInterval(isCrypto, marketStatus),
				},
				"keyMetrics":   buildKeyMetrics(priceData, fundamentals, stock),
				"fundamentals": fundamentals,
			},
		},
		"meta": gin.H{
			"symbol":    symbol,
			"assetType": stock.AssetType,
			"isCrypto":  isCrypto,
			"timestamp": time.Now().UTC(),
			"source":    "polygon",
		},
	}

	c.JSON(http.StatusOK, response)
}

// GetTickerChart returns chart data for a symbol
func GetTickerChart(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	period := c.DefaultQuery("period", "1Y")

	log.Printf("GetTickerChart called for symbol: %s, period: %s", symbol, period)

	polygonClient := services.NewPolygonClient()

	// Convert period to days for Polygon API
	days := services.GetDaysFromPeriod(period)

	var chartData []models.ChartDataPoint
	var err error

	if period == "1D" {
		// For intraday, get minute-level data
		chartData, err = polygonClient.GetIntradayData(symbol)
	} else {
		// For longer periods, get daily data
		chartData, err = polygonClient.GetDailyData(symbol, days)
	}

	if err != nil {
		log.Printf("Failed to get chart data for %s: %v", symbol, err)
		// Fallback to mock data
		mockChart := generateMockChartData(symbol, period)
		chartData = mockChart.DataPoints
	}

	response := gin.H{
		"success": true,
		"data": gin.H{
			"symbol":      symbol,
			"period":      period,
			"dataPoints":  chartData,
			"count":       len(chartData),
			"lastUpdated": time.Now().UTC(),
		},
		"meta": gin.H{
			"symbol":    symbol,
			"period":    period,
			"count":     len(chartData),
			"timestamp": time.Now().UTC(),
		},
	}

	c.JSON(http.StatusOK, response)
}

// Helper functions

func isCryptoAsset(assetType, symbol string) bool {
	// Check asset type from database
	if assetType == "crypto" {
		return true
	}

	// Check symbol format (Polygon crypto symbols start with X:)
	if strings.HasPrefix(symbol, "X:") {
		return true
	}

	// Check for common crypto symbols
	cryptoSymbols := map[string]bool{
		"BTC": true, "ETH": true, "ADA": true, "DOT": true,
		"LINK": true, "XRP": true, "LTC": true, "BCH": true,
		"BNB": true, "SOL": true, "MATIC": true, "AVAX": true,
	}

	return cryptoSymbols[symbol]
}

func getUpdateInterval(isCrypto bool, marketStatus string) int {
	if isCrypto {
		return 5 // 5 seconds for crypto
	}

	if marketStatus == "open" {
		return 5 // 5 seconds during market hours
	}

	return 300 // 5 minutes when market is closed
}

func generateMockPrice(symbol string, stock *models.Stock) *models.StockPrice {
	// Generate realistic mock price based on symbol
	basePrice := 150.0 + float64(len(symbol)*10)

	open := basePrice * (0.98 + 0.04*0.5) // ±2% from base
	high := open * (1.0 + 0.03*0.8)       // Up to 3% higher
	low := open * (1.0 - 0.03*0.6)        // Up to 3% lower
	close := low + (high-low)*0.7         // 70% of the range

	change := decimal.NewFromFloat(close - open)
	changePercent := decimal.Zero
	if open != 0 {
		changePercent = change.Div(decimal.NewFromFloat(open))
	}

	return &models.StockPrice{
		Symbol:        symbol,
		Price:         decimal.NewFromFloat(close),
		Open:          decimal.NewFromFloat(open),
		High:          decimal.NewFromFloat(high),
		Low:           decimal.NewFromFloat(low),
		Close:         decimal.NewFromFloat(close),
		Volume:        int64(1000000 + len(symbol)*100000),
		Change:        change,
		ChangePercent: changePercent,
		Timestamp:     time.Now(),
	}
}

func generateMockFundamentals(symbol string) *models.Fundamentals {
	// Generate basic mock fundamentals
	return &models.Fundamentals{
		Symbol:    symbol,
		Period:    "TTM",
		Year:      time.Now().Year(),
		PE:        decimalPtrComprehensive(20.5 + float64(len(symbol))),
		PB:        decimalPtrComprehensive(3.2 + float64(len(symbol))*0.1),
		PS:        decimalPtrComprehensive(5.8 + float64(len(symbol))*0.2),
		Revenue:   decimalPtrComprehensive(50000000000.0),
		EPS:       decimalPtrComprehensive(8.50),
		ROE:       decimalPtrComprehensive(0.18),
		ROA:       decimalPtrComprehensive(0.12),
		UpdatedAt: time.Now(),
	}
}

func buildKeyMetrics(price *models.StockPrice, fundamentals *models.Fundamentals, stock *models.Stock) gin.H {
	metrics := gin.H{
		"volume":    price.Volume,
		"timestamp": price.Timestamp.Unix(),
	}

	// Add fundamental metrics if available
	if fundamentals != nil {
		if fundamentals.PE != nil {
			metrics["pe"] = fundamentals.PE.String()
		}
		if fundamentals.EPS != nil {
			metrics["eps"] = fundamentals.EPS.String()
		}
		if fundamentals.Revenue != nil {
			metrics["revenue"] = fundamentals.Revenue.String()
		}
	}

	return metrics
}

// decimalPtr helper function (avoiding redeclaration with mock_data.go)
func decimalPtrComprehensive(f float64) *decimal.Decimal {
	d := decimal.NewFromFloat(f)
	return &d
}

// GetTickerRealTimePrice returns just the current price for real-time updates
// Handles both stocks and crypto
func GetTickerRealTimePrice(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	log.Printf("GetTickerRealTimePrice called for symbol: %s", symbol)

	// First, try to get from crypto (Redis) - fast check
	ctx := context.Background()
	priceKey := fmt.Sprintf("crypto:quote:%s", symbol)
	cryptoData, redisErr := redisClient.Get(ctx, priceKey).Result()

	if redisErr == nil {
		// Found in crypto cache, parse and return
		log.Printf("Symbol %s found in crypto cache, returning crypto price", symbol)
		var price CryptoRealTimePrice
		if err := json.Unmarshal([]byte(cryptoData), &price); err == nil {
			c.JSON(http.StatusOK, gin.H{
				"data": gin.H{
					"symbol":        symbol,
					"price":         fmt.Sprintf("%.2f", price.Price),
					"change":        fmt.Sprintf("%.2f", price.Price*price.Change24h/100),
					"changePercent": fmt.Sprintf("%.2f", price.Change24h),
					"volume":        price.Volume24h,
					"timestamp":     time.Now().Unix(),
					"lastUpdated":   price.LastUpdated,
					"marketStatus":  "open", // Crypto is always open
					"assetType":     "crypto",
				},
				"meta": gin.H{
					"timestamp": time.Now().UTC(),
					"source":    "redis",
				},
			})
			return
		}
	}

	// Not crypto, use Polygon for stocks/ETFs
	polygonClient := services.NewPolygonClient()
	priceData, err := polygonClient.GetQuote(symbol)

	if err != nil {
		log.Printf("Polygon API error for %s: %v", symbol, err)
		// Return a more graceful error response
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Price not available",
			"symbol":  symbol,
			"message": "This ticker is not currently tracked",
		})
		return
	}

	log.Printf("Success! Got price for %s: %s", symbol, priceData.Price.String())

	// Return price data directly in ApiClient format
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"symbol":        symbol,
			"price":         priceData.Price.String(),
			"change":        priceData.Change.String(),
			"changePercent": priceData.ChangePercent.String(),
			"volume":        priceData.Volume,
			"timestamp":     priceData.Timestamp.Unix(),
			"lastUpdated":   priceData.Timestamp.Format(time.RFC3339),
		},
		"meta": gin.H{
			"timestamp": time.Now().UTC(),
		},
	})
}

// GetTickerNews returns news articles for a ticker
func GetTickerNews(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	log.Printf("GetTickerNews called for symbol: %s", symbol)

	// Try to get real news from Polygon first
	polygonClient := services.NewPolygonClient()

	// Get raw Polygon news data with all fields
	url := "https://api.polygon.io/v2/reference/news?ticker=" + symbol + "&limit=30&apikey=" + polygonClient.APIKey
	resp, err := polygonClient.Client.Get(url)

	if err != nil || resp.StatusCode != 200 {
		log.Printf("Failed to get real news for %s: %v, using mock data", symbol, err)
		// Fallback to mock news data
		mockArticles := generateMockNews(symbol)
		c.JSON(http.StatusOK, gin.H{
			"data": mockArticles,
			"meta": gin.H{
				"symbol":    symbol,
				"count":     len(mockArticles),
				"timestamp": time.Now().UTC(),
				"source":    "mock",
			},
		})
		return
	}
	defer resp.Body.Close()

	// Parse and return raw Polygon response with all fields
	var polygonResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&polygonResp); err != nil {
		log.Printf("Failed to decode news response: %v", err)
		mockArticles := generateMockNews(symbol)
		c.JSON(http.StatusOK, gin.H{
			"data": mockArticles,
			"meta": gin.H{
				"symbol":    symbol,
				"count":     len(mockArticles),
				"timestamp": time.Now().UTC(),
				"source":    "mock",
			},
		})
		return
	}

	// Return the raw Polygon results with all fields intact
	results := polygonResp["results"]
	log.Printf("Successfully fetched real news for %s", symbol)

	c.JSON(http.StatusOK, gin.H{
		"data": results,
		"meta": gin.H{
			"symbol":    symbol,
			"timestamp": time.Now().UTC(),
			"source":    "polygon",
		},
	})
}

// GetTickerEarnings returns earnings data for a ticker
func GetTickerEarnings(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	log.Printf("GetTickerEarnings called for symbol: %s", symbol)

	// Generate mock earnings data (same as in mock_data.go)
	earnings := generateMockEarnings(symbol)

	c.JSON(http.StatusOK, gin.H{
		"data": earnings,
		"meta": gin.H{
			"symbol":    symbol,
			"count":     len(earnings),
			"timestamp": time.Now().UTC(),
		},
	})
}

// GetTickerAnalysts returns analyst ratings for a ticker
func GetTickerAnalysts(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	log.Printf("GetTickerAnalysts called for symbol: %s", symbol)

	// Generate mock analyst data (same as in mock_data.go)
	analysts := generateMockAnalystRatings(symbol)

	c.JSON(http.StatusOK, gin.H{
		"data": analysts,
		"meta": gin.H{
			"symbol":    symbol,
			"count":     len(analysts),
			"timestamp": time.Now().UTC(),
		},
	})
}

// calculateMarketCap estimates market cap for a crypto symbol
func calculateMarketCap(symbol string, price decimal.Decimal) decimal.Decimal {
	cleanSymbol := strings.Replace(symbol, "X:", "", 1)

	// Extract base crypto (BTC, ETH, etc.) from pairs like BTCUSD, BTCJPY, etc.
	var supply int64
	var usdPrice decimal.Decimal = price

	// Identify the base cryptocurrency
	if strings.HasPrefix(cleanSymbol, "BTC") {
		supply = 19_800_000 // ~19.8M BTC
		// Convert to USD if needed (rough conversion for JPY, EUR, etc.)
		if strings.Contains(cleanSymbol, "JPY") {
			usdPrice = price.Div(decimal.NewFromInt(150)) // ~150 JPY per USD
		} else if strings.Contains(cleanSymbol, "EUR") {
			usdPrice = price.Mul(decimal.NewFromFloat(1.1)) // ~1.1 USD per EUR
		}
	} else if strings.HasPrefix(cleanSymbol, "ETH") {
		supply = 120_000_000 // ~120M ETH
		if strings.Contains(cleanSymbol, "JPY") {
			usdPrice = price.Div(decimal.NewFromInt(150))
		} else if strings.Contains(cleanSymbol, "EUR") {
			usdPrice = price.Mul(decimal.NewFromFloat(1.1))
		}
	} else {
		// For other cryptos, extract the base
		switch {
		case strings.HasPrefix(cleanSymbol, "SOL"):
			supply = 470_000_000 // ~470M SOL
		case strings.HasPrefix(cleanSymbol, "XRP"):
			supply = 56_000_000_000 // ~56B XRP
		case strings.HasPrefix(cleanSymbol, "DOGE"):
			supply = 147_000_000_000 // ~147B DOGE
		case strings.HasPrefix(cleanSymbol, "ADA"):
			supply = 35_000_000_000 // ~35B ADA
		case strings.HasPrefix(cleanSymbol, "LTC"):
			supply = 75_000_000 // ~75M LTC
		case strings.HasPrefix(cleanSymbol, "LINK"):
			supply = 600_000_000 // ~600M LINK
		case strings.HasPrefix(cleanSymbol, "AVAX"):
			supply = 400_000_000 // ~400M AVAX
		case strings.HasPrefix(cleanSymbol, "MATIC"):
			supply = 10_000_000_000 // ~10B MATIC
		case strings.Contains(cleanSymbol, "USDT") || strings.Contains(cleanSymbol, "USDC"):
			supply = 100_000_000_000 // ~100B for stablecoins
		default:
			supply = 1_000_000 // 1M default
		}
	}

	return usdPrice.Mul(decimal.NewFromInt(supply))
}

// GetAllCryptos returns all cached crypto prices sorted by market cap
func GetAllCryptos(c *gin.Context) {
	log.Printf("GetAllCryptos called")

	// Get page parameter (default to 1)
	page := 1
	if pageParam := c.Query("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}

	// Get crypto symbols from Redis (ranked by market cap from CoinGecko)
	ctx := context.Background()
	symbols, err := redisClient.ZRange(ctx, "crypto:symbols:ranked", 0, -1).Result()
	if err != nil {
		log.Printf("Failed to get crypto symbols from Redis: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch crypto symbols",
		})
		return
	}

	if len(symbols) == 0 {
		log.Printf("No crypto symbols found in Redis")
		c.JSON(http.StatusOK, gin.H{
			"data": []interface{}{},
			"meta": gin.H{
				"page":       page,
				"perPage":    100,
				"total":      0,
				"totalPages": 0,
				"timestamp":  time.Now().UTC(),
				"source":     "redis",
			},
		})
		return
	}

	// Fetch all crypto data from Redis using pipeline
	pipe := redisClient.Pipeline()
	for _, symbol := range symbols {
		pipe.Get(ctx, fmt.Sprintf("crypto:quote:%s", symbol))
	}

	results, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		log.Printf("Pipeline error fetching crypto data: %v", err)
	}

	// Parse all crypto data
	type CryptoData struct {
		Symbol        string  `json:"symbol"`
		Price         string  `json:"price"`
		Change        string  `json:"change"`
		ChangePercent string  `json:"changePercent"`
		Volume        float64 `json:"volume"`
		High          string  `json:"high"`
		Low           string  `json:"low"`
		Timestamp     int64   `json:"timestamp"`
		MarketCapRank int     `json:"market_cap_rank"`
	}

	var allCryptos []CryptoData

	for i, symbol := range symbols {
		if i < len(results) {
			if val, err := results[i].(*redis.StringCmd).Result(); err == nil {
				var cryptoPrice CryptoRealTimePrice
				if json.Unmarshal([]byte(val), &cryptoPrice) == nil {
					// Populate alias fields
					cryptoPrice.Price = cryptoPrice.CurrentPrice
					cryptoPrice.Volume24h = cryptoPrice.TotalVolume
					cryptoPrice.Change24h = cryptoPrice.PriceChangePercentage24h

					allCryptos = append(allCryptos, CryptoData{
						Symbol:        symbol,
						Price:         fmt.Sprintf("%.8f", cryptoPrice.Price),
						Change:        fmt.Sprintf("%.2f", cryptoPrice.PriceChange24h),
						ChangePercent: fmt.Sprintf("%.2f", cryptoPrice.Change24h),
						Volume:        cryptoPrice.Volume24h,
						High:          fmt.Sprintf("%.8f", cryptoPrice.High24h),
						Low:           fmt.Sprintf("%.8f", cryptoPrice.Low24h),
						Timestamp:     time.Now().Unix(),
						MarketCapRank: cryptoPrice.MarketCapRank,
					})
				}
			}
		}
	}

	// Cryptos are already sorted by market cap rank in Redis (crypto:symbols:ranked)
	// No need to sort again

	// Pagination
	perPage := 100
	totalCryptos := len(allCryptos)
	totalPages := (totalCryptos + perPage - 1) / perPage

	startIdx := (page - 1) * perPage
	endIdx := startIdx + perPage
	if endIdx > totalCryptos {
		endIdx = totalCryptos
	}

	var pageData []CryptoData
	if startIdx < totalCryptos {
		pageData = allCryptos[startIdx:endIdx]
	}

	c.JSON(http.StatusOK, gin.H{
		"data": pageData,
		"meta": gin.H{
			"page":       page,
			"perPage":    perPage,
			"total":      totalCryptos,
			"totalPages": totalPages,
			"timestamp":  time.Now().UTC(),
			"source":     "redis",
		},
	})
}
