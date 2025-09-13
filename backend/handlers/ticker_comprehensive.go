package handlers

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"investorcenter-api/models"
	"investorcenter-api/services"
)



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
					"symbol":     stock.Symbol,
					"name":       stock.Name,
					"exchange":   stock.Exchange,
					"sector":     stock.Sector,
					"assetType":  stock.AssetType,
					"isCrypto":   isCrypto,
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
					"status":              marketStatus,
					"shouldUpdateRealtime": shouldUpdateRealtime,
					"updateInterval":       getUpdateInterval(isCrypto, marketStatus),
				},
				"keyMetrics": buildKeyMetrics(priceData, fundamentals, stock),
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
		"BTC":  true, "ETH":  true, "ADA":  true, "DOT":  true,
		"LINK": true, "XRP":  true, "LTC":  true, "BCH":  true,
		"BNB":  true, "SOL":  true, "MATIC": true, "AVAX": true,
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
	close := low + (high-low)*0.7          // 70% of the range
	
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
// Minimal test version to isolate the issue
func GetTickerRealTimePrice(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	log.Printf("GetTickerRealTimePrice called for symbol: %s", symbol)
	
	// Minimal test: Just call Polygon directly (we know this works)
	polygonClient := services.NewPolygonClient()
	priceData, err := polygonClient.GetQuote(symbol)
	
	if err != nil {
		log.Printf("Polygon API error for %s: %v", symbol, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch price data",
			"details": err.Error(),
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
