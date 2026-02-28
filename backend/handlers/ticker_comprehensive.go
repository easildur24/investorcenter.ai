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

	var isCrypto bool

	if err != nil {
		// Not in database - check if it's a crypto in Redis
		log.Printf("Stock not found in database: %v, checking Redis for crypto", err)
		cryptoData, cryptoExists := getCryptoFromRedis(symbol)

		if cryptoExists {
			// Build a stock object from Redis crypto data
			stock = &models.Stock{
				Symbol:    symbol,
				Name:      cryptoData.Name,
				AssetType: "crypto",
				Exchange:  "CRYPTO",
			}
			isCrypto = true
			log.Printf("Found crypto %s in Redis: %s", symbol, cryptoData.Name)
		} else {
			log.Printf("Symbol %s not found in database or Redis", symbol)
			c.JSON(http.StatusNotFound, gin.H{
				"error":  "Ticker not found",
				"symbol": symbol,
			})
			return
		}
	} else {
		// Found in database - determine if crypto or stock using all available fields
		isCrypto = isCryptoAssetWithStock(stock)
	}

	// Get real-time price data
	var priceData *models.StockPrice
	var priceErr error
	var marketStatus string
	var shouldUpdateRealtime bool
	polygonClient := services.NewPolygonClient() // Initialize for both crypto and stock

	if isCrypto {
		// For crypto, get price from Redis
		cryptoData, exists := getCryptoFromRedis(symbol)
		if exists {
			priceData = convertCryptoPriceToStockPrice(cryptoData)
			log.Printf("✓ Got crypto price for %s from Redis: $%.2f", symbol, cryptoData.CurrentPrice)
		} else {
			log.Printf("Failed to get crypto price for %s from Redis", symbol)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":  "Price data temporarily unavailable",
				"symbol": symbol,
			})
			return
		}
		marketStatus = "open" // Crypto markets are always open
		shouldUpdateRealtime = true
	} else {
		// For stocks, try unified snapshot first for session-aware data
		snapshot, snapErr := polygonClient.GetUnifiedSnapshot(symbol)
		if snapErr == nil {
			// Use unified snapshot data
			priceData = &models.StockPrice{
				Symbol:        symbol,
				Price:         snapshot.Price,
				Change:        snapshot.Change,
				ChangePercent: snapshot.ChangePercent,
				Volume:        snapshot.Volume,
				Timestamp:     snapshot.Timestamp,
			}
			marketStatus = snapshot.MarketSession
			shouldUpdateRealtime = snapshot.MarketSession != "closed"
		} else {
			// Fallback to v2 snapshot
			log.Printf("Unified snapshot failed for %s: %v, using GetQuote", symbol, snapErr)
			priceData, priceErr = polygonClient.GetQuote(symbol)
			isOpen := polygonClient.IsMarketOpen()
			if isOpen {
				marketStatus = "regular"
				shouldUpdateRealtime = true
			} else {
				marketStatus = "closed"
				shouldUpdateRealtime = false
			}
		}

		if priceErr != nil {
			log.Printf("Failed to get real-time price data for %s: %v", symbol, priceErr)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":  "Price data temporarily unavailable",
				"symbol": symbol,
			})
			return
		}

		// Store snapshot for response building
		c.Set("unifiedSnapshot", snapshot)
	}

	// Get fundamentals from Polygon (nil if unavailable)
	var fundamentals *models.Fundamentals
	if !isCrypto { // Only get fundamentals for stocks, not crypto
		// Use a timeout channel to avoid hanging
		done := make(chan bool, 1)
		var fundamentalsErr error

		go func() {
			fundamentals, fundamentalsErr = polygonClient.GetFundamentals(symbol)
			done <- true
		}()

		// Wait for fundamentals or timeout after 3 seconds
		select {
		case <-done:
			if fundamentalsErr != nil {
				log.Printf("Failed to get fundamentals for %s: %v", symbol, fundamentalsErr)
				fundamentals = &models.Fundamentals{Symbol: symbol, Period: "N/A", Year: time.Now().Year(), UpdatedAt: time.Now()}
			}
		case <-time.After(3 * time.Second):
			log.Printf("Fundamentals request timed out for %s", symbol)
			fundamentals = &models.Fundamentals{Symbol: symbol, Period: "N/A", Year: time.Now().Year(), UpdatedAt: time.Now()}
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
					"logoUrl":   stock.LogoURL,
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
				"market": buildInitialMarketResponse(c, isCrypto, marketStatus, shouldUpdateRealtime),
				"keyMetrics":   buildKeyMetrics(priceData, fundamentals, stock),
				"fundamentals": fundamentals,
			},
		},
		"meta": gin.H{
			"symbol":    symbol,
			"assetType": stock.AssetType,
			"isCrypto":  isCrypto,
			"timestamp": time.Now().UTC(),
			"source":    getDataSource(isCrypto),
		},
	}

	c.JSON(http.StatusOK, response)
}

// GetTickerChart returns chart data for a symbol
func GetTickerChart(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	period := c.DefaultQuery("period", "1Y")

	// Validate period against allowed values
	validPeriods := map[string]bool{
		"1D": true, "5D": true, "1W": true, "1M": true,
		"3M": true, "6M": true, "YTD": true, "1Y": true,
		"3Y": true, "5Y": true, "MAX": true,
	}
	if !validPeriods[period] {
		period = "1Y"
	}

	log.Printf("GetTickerChart called for symbol: %s, period: %s", symbol, period)

	// Check if this is a crypto asset
	stockService := services.NewStockService()
	stock, err := stockService.GetStockBySymbol(c.Request.Context(), symbol)

	var isCrypto bool
	if err != nil {
		// Not in database - check if it's in Redis (crypto)
		_, cryptoExists := getCryptoFromRedis(symbol)
		isCrypto = cryptoExists
	} else {
		isCrypto = isCryptoAsset(stock.AssetType, symbol)
	}

	var chartData []models.ChartDataPoint
	var chartErr error
	var dataSource string

	if isCrypto {
		// Use CoinGecko for crypto charts
		log.Printf("Fetching crypto chart data for %s from CoinGecko", symbol)
		coinGeckoClient := services.NewCoinGeckoClient()
		chartData, chartErr = coinGeckoClient.GetChartData(symbol, period)
		dataSource = "coingecko"

		if chartErr != nil {
			log.Printf("Failed to get crypto chart data for %s: %v", symbol, chartErr)
			// Return empty chart with error message instead of mock data
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data": gin.H{
					"symbol":      symbol,
					"period":      period,
					"dataPoints":  []models.ChartDataPoint{},
					"count":       0,
					"lastUpdated": time.Now().UTC(),
					"error":       "Chart data temporarily unavailable",
				},
				"meta": gin.H{
					"symbol":    symbol,
					"period":    period,
					"isCrypto":  true,
					"source":    "coingecko",
					"timestamp": time.Now().UTC(),
				},
			})
			return
		}
	} else {
		// For stocks: Try database first (faster, has 3 years of data), fallback to Polygon
		if period == "1D" {
			// For intraday data, must use Polygon
			log.Printf("Fetching intraday chart data for %s from Polygon", symbol)
			polygonClient := services.NewPolygonClient()
			chartData, chartErr = polygonClient.GetIntradayData(symbol)
			dataSource = "polygon"
		} else {
			// For longer periods, try database first
			log.Printf("Fetching chart data for %s from database", symbol)
			priceService := services.NewPriceService()
			chartData, chartErr = priceService.GetHistoricalPrices(c.Request.Context(), symbol, period)

			if chartErr == nil && len(chartData) > 0 {
				dataSource = "database"
				log.Printf("✓ Successfully fetched %d data points from database for %s", len(chartData), symbol)
			} else {
				// Fallback to Polygon if database query fails or returns no data
				log.Printf("Database query failed or returned no data for %s, falling back to Polygon: %v", symbol, chartErr)
				polygonClient := services.NewPolygonClient()
				chartData, chartErr = polygonClient.GetDailyData(symbol, services.GetDaysFromPeriod(period))
				dataSource = "polygon"
			}
		}

		if chartErr != nil {
			log.Printf("Failed to get chart data for %s: %v", symbol, chartErr)
			chartData = []models.ChartDataPoint{}
			dataSource = "none"
		}
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
			"isCrypto":  isCrypto,
			"source":    dataSource,
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

// isCryptoAssetWithStock checks if a stock is crypto based on all available fields
func isCryptoAssetWithStock(stock *models.Stock) bool {
	if stock == nil {
		return false
	}

	// Check asset type
	if stock.AssetType == "crypto" {
		return true
	}

	// Check exchange (crypto tickers have exchange="CRYPTO")
	if stock.Exchange == "CRYPTO" {
		return true
	}

	// Check sector (crypto tickers have sector="Cryptocurrency")
	if stock.Sector == "Cryptocurrency" {
		return true
	}

	// Fall back to symbol-based check
	return isCryptoAsset(stock.AssetType, stock.Symbol)
}

func getUpdateInterval(isCrypto bool, session string) int {
	if isCrypto {
		return 5 // 5 seconds for crypto
	}

	switch session {
	case "regular":
		return 5 // 5 seconds during market hours
	case "pre_market", "after_hours":
		return 15 // 15 seconds during extended hours
	default:
		return 300 // 5 minutes when market is closed
	}
}

// buildInitialMarketResponse creates the market section for the initial GetTicker response
func buildInitialMarketResponse(c *gin.Context, isCrypto bool, marketStatus string, shouldUpdateRealtime bool) gin.H {
	result := gin.H{
		"status":               marketStatus,
		"session":              marketStatus,
		"shouldUpdateRealtime": shouldUpdateRealtime,
		"updateInterval":       getUpdateInterval(isCrypto, marketStatus),
	}

	// Include regularClose data from unified snapshot during extended hours
	if !isCrypto {
		if val, exists := c.Get("unifiedSnapshot"); exists && val != nil {
			snapshot := val.(*services.UnifiedSnapshotResult)
			if snapshot != nil && snapshot.HasRegularClose {
				regularClose := gin.H{
					"price": snapshot.RegularClosePrice.StringFixed(2),
				}
				if snapshot.MarketSession == "after_hours" {
					regularClose["change"] = snapshot.RegularChange.StringFixed(2)
					regularClose["changePercent"] = snapshot.RegularChangePercent.StringFixed(2)
				}
				result["regularClose"] = regularClose
			}
		}
	}

	return result
}

func getDataSource(isCrypto bool) string {
	if isCrypto {
		return "redis" // Crypto data from Redis (populated by coingecko-service)
	}
	return "polygon" // Stock data from Polygon API
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

	// Try unified snapshot first (provides session-aware data)
	snapshot, err := polygonClient.GetUnifiedSnapshot(symbol)
	if err != nil {
		log.Printf("Unified snapshot failed for %s: %v, falling back to GetQuote", symbol, err)
		// Fallback to v2 snapshot
		priceData, fallbackErr := polygonClient.GetQuote(symbol)
		if fallbackErr != nil {
			log.Printf("Polygon API error for %s: %v", symbol, fallbackErr)
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Price not available",
				"symbol":  symbol,
				"message": "This ticker is not currently tracked",
			})
			return
		}
		isOpen := polygonClient.IsMarketOpen()
		session := "closed"
		if isOpen {
			session = "regular"
		}
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
			"market": gin.H{
				"session":        session,
				"isOpen":         isOpen,
				"updateInterval": getUpdateInterval(false, session),
			},
			"meta": gin.H{
				"timestamp": time.Now().UTC(),
			},
		})
		return
	}

	log.Printf("Unified snapshot for %s: session=%s price=%s", symbol, snapshot.MarketSession, snapshot.Price.String())

	// Build market response
	marketData := gin.H{
		"session":        snapshot.MarketSession,
		"isOpen":         snapshot.MarketSession == "regular",
		"updateInterval": getUpdateInterval(false, snapshot.MarketSession),
	}

	// Include regular close data during extended hours
	if snapshot.HasRegularClose {
		regularClose := gin.H{
			"price": snapshot.RegularClosePrice.StringFixed(2),
		}
		// Include change data for after-hours (we have regular session change)
		if snapshot.MarketSession == "after_hours" {
			regularClose["change"] = snapshot.RegularChange.StringFixed(2)
			regularClose["changePercent"] = snapshot.RegularChangePercent.StringFixed(2)
		}
		marketData["regularClose"] = regularClose
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"symbol":        symbol,
			"price":         snapshot.Price.StringFixed(2),
			"change":        snapshot.Change.StringFixed(2),
			"changePercent": snapshot.ChangePercent.StringFixed(2),
			"volume":        snapshot.Volume,
			"timestamp":     snapshot.Timestamp.Unix(),
			"lastUpdated":   snapshot.Timestamp.Format(time.RFC3339),
		},
		"market": marketData,
		"meta": gin.H{
			"timestamp": time.Now().UTC(),
		},
	})
}

// GetTickerNews returns news articles for a ticker with AI sentiment analysis
func GetTickerNews(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	log.Printf("GetTickerNews called for symbol: %s", symbol)

	// First, try IC Score service for news with real AI sentiment analysis
	icScoreClient := services.NewICScoreClient()
	icScoreNews, err := icScoreClient.GetNews(symbol, 30, 30)

	if err == nil && icScoreNews != nil && len(icScoreNews.Articles) > 0 {
		log.Printf("Successfully fetched %d news articles with AI sentiment from IC Score for %s", len(icScoreNews.Articles), symbol)

		// Convert to response format compatible with frontend
		articles := make([]map[string]interface{}, len(icScoreNews.Articles))
		for i, article := range icScoreNews.Articles {
			articles[i] = map[string]interface{}{
				"id":              article.ID,
				"title":           article.Title,
				"article_url":     article.URL,
				"published_utc":   article.PublishedAt,
				"description":     article.Summary,
				"author":          article.Author,
				"tickers":         article.Tickers,
				"image_url":       article.ImageURL,
				"sentiment_score": article.SentimentScore, // -100 to +100
				"sentiment_label": article.SentimentLabel, // Positive, Negative, Neutral
				"relevance_score": article.RelevanceScore, // 0 to 100
				"publisher": map[string]interface{}{
					"name": article.Source,
				},
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"data": articles,
			"meta": gin.H{
				"symbol":    symbol,
				"count":     len(articles),
				"timestamp": time.Now().UTC(),
				"source":    "ic-score",
			},
		})
		return
	}

	log.Printf("IC Score news not available for %s: %v, falling back to Polygon", symbol, err)

	// Fallback to Polygon API for news
	polygonClient := services.NewPolygonClient()
	url := "https://api.polygon.io/v2/reference/news?ticker=" + symbol + "&limit=30&apikey=" + polygonClient.APIKey
	resp, err := polygonClient.Client.Get(url)

	if err != nil || resp.StatusCode != 200 {
		log.Printf("Failed to get news from Polygon for %s: %v", symbol, err)
		c.JSON(http.StatusOK, gin.H{
			"data": []interface{}{},
			"meta": gin.H{
				"symbol":    symbol,
				"count":     0,
				"timestamp": time.Now().UTC(),
				"source":    "none",
			},
		})
		return
	}
	defer resp.Body.Close()

	var polygonResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&polygonResp); err != nil {
		log.Printf("Failed to decode Polygon news response: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"data": []interface{}{},
			"meta": gin.H{
				"symbol":    symbol,
				"count":     0,
				"timestamp": time.Now().UTC(),
				"source":    "none",
			},
		})
		return
	}

	results := polygonResp["results"]
	log.Printf("Successfully fetched news from Polygon for %s", symbol)

	c.JSON(http.StatusOK, gin.H{
		"data": results,
		"meta": gin.H{
			"symbol":    symbol,
			"timestamp": time.Now().UTC(),
			"source":    "polygon",
		},
	})
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

// getCryptoFromRedis checks if a symbol exists in Redis crypto data
func getCryptoFromRedis(symbol string) (*CryptoRealTimePrice, bool) {
	ctx := context.Background()
	priceKey := fmt.Sprintf("crypto:quote:%s", symbol)

	priceData, err := redisClient.Get(ctx, priceKey).Result()
	if err == redis.Nil {
		// Not found in Redis
		return nil, false
	} else if err != nil {
		// Redis error
		log.Printf("Redis error checking crypto %s: %v", symbol, err)
		return nil, false
	}

	// Parse the crypto data
	var crypto CryptoRealTimePrice
	if err := json.Unmarshal([]byte(priceData), &crypto); err != nil {
		log.Printf("Failed to parse crypto data for %s: %v", symbol, err)
		return nil, false
	}

	// Ensure aliases are set for compatibility
	crypto.Price = crypto.CurrentPrice
	crypto.Volume24h = crypto.TotalVolume
	crypto.Change24h = crypto.PriceChange24h

	return &crypto, true
}

// convertCryptoPriceToStockPrice converts CryptoRealTimePrice to StockPrice format
func convertCryptoPriceToStockPrice(crypto *CryptoRealTimePrice) *models.StockPrice {
	// Parse timestamp
	timestamp := time.Now()
	if crypto.LastUpdated != "" {
		if parsedTime, err := time.Parse(time.RFC3339, crypto.LastUpdated); err == nil {
			timestamp = parsedTime
		}
	}

	// Current price
	price := decimal.NewFromFloat(crypto.CurrentPrice)

	// Calculate change and change percent
	change := decimal.NewFromFloat(crypto.PriceChange24h)
	changePercent := decimal.NewFromFloat(crypto.PriceChangePercentage24h)

	return &models.StockPrice{
		Symbol:        crypto.Symbol,
		Price:         price,
		Open:          price.Sub(change), // Approximate open = current - change
		High:          decimal.NewFromFloat(crypto.High24h),
		Low:           decimal.NewFromFloat(crypto.Low24h),
		Close:         price,
		Volume:        int64(crypto.TotalVolume),
		Change:        change,
		ChangePercent: changePercent,
		Timestamp:     timestamp,
	}
}
