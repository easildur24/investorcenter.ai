package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"investorcenter-api/database"
	"investorcenter-api/handlers"
	"investorcenter-api/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize database connection
	if err := database.Initialize(); err != nil {
		log.Printf("Database connection failed: %v", err)
		log.Println("Starting in mock mode - database features disabled")
	} else {
		log.Println("Database connected successfully")
		defer database.Close()
	}

	// Set Gin mode
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create Gin router
	r := gin.Default()

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{
		"http://localhost:3000",
		"https://investorcenter.ai",
		"https://www.investorcenter.ai",
	}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	config.AllowCredentials = true
	r.Use(cors.New(config))

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		response := gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"service":   "investorcenter-api",
		}

		// Check database health
		if database.DB != nil {
			if err := database.HealthCheck(); err != nil {
				response["database"] = "unhealthy"
				response["database_error"] = err.Error()
				c.JSON(http.StatusServiceUnavailable, response)
				return
			}
			response["database"] = "healthy"
		} else {
			response["database"] = "not_connected"
		}

		c.JSON(http.StatusOK, response)
	})

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Market data endpoints
		markets := v1.Group("/markets")
		{
			markets.GET("/indices", getMarketIndices)
			markets.GET("/search", searchSecurities)
		}

		// Ticker page endpoints
		tickers := v1.Group("/tickers")
		{
			tickers.GET("/", handlers.GetStocks)                           // List all stocks with pagination
			tickers.POST("/", handlers.CreateStock)                        // Create new stock
			tickers.POST("/import", handlers.ImportTickersFromCSV)         // Import from CSV
			tickers.GET("/:symbol", handlers.GetTicker)                    // Comprehensive ticker data with real-time prices
			tickers.GET("/:symbol/chart", handlers.GetTickerChart)         // Chart data for stocks and crypto
			tickers.GET("/:symbol/price", handlers.GetTickerRealTimePrice) // Real-time price updates only

			// Volume endpoints (hybrid: database + real-time)
			tickers.GET("/:symbol/volume", handlers.GetTickerVolume)                // Get volume data (add ?realtime=true for fresh data)
			tickers.GET("/:symbol/volume/aggregates", handlers.GetVolumeAggregates) // Get volume aggregates

			// Additional ticker endpoints
			tickers.GET("/:symbol/news", handlers.GetTickerNews)
			tickers.GET("/:symbol/earnings", handlers.GetTickerEarnings)
			tickers.GET("/:symbol/analysts", handlers.GetTickerAnalysts)
			// tickers.GET("/:symbol/fundamentals", handlers.GetTickerFundamentals)
			// tickers.GET("/:symbol/dividends", handlers.GetTickerDividends)
			// tickers.GET("/:symbol/insiders", handlers.GetTickerInsiders)
			// tickers.GET("/:symbol/peers", handlers.GetTickerPeers)
		}

		// Crypto endpoints
		crypto := v1.Group("/crypto")
		{
			crypto.GET("/", handlers.GetAllCryptos)                       // All crypto prices with pagination
			crypto.GET("/:symbol/price", handlers.GetCryptoRealTimePrice) // Real-time crypto price from Redis
			crypto.GET("/prices", handlers.GetAllCryptoRealTimePrices)    // All real-time crypto prices
			crypto.GET("/stream", handlers.StreamCryptoPrices)            // SSE endpoint for real-time streaming
		}

		// Volume endpoints for bulk operations
		volume := v1.Group("/volume")
		{
			volume.POST("/bulk", handlers.GetBulkVolume) // Get volume for multiple symbols
			volume.GET("/top", handlers.GetTopVolume)    // Get top stocks by volume
		}

		// Portfolio endpoints - TODO: implement when needed
		// portfolios := v1.Group("/portfolios")
		// {
		// 	portfolios.GET("/", getPortfolios)
		// 	portfolios.POST("/", createPortfolio)
		// 	portfolios.GET("/:id", getPortfolio)
		// 	portfolios.PUT("/:id", updatePortfolio)
		// 	portfolios.DELETE("/:id", deletePortfolio)
		// 	portfolios.GET("/:id/performance", getPortfolioPerformance)
		// }

		// Analytics endpoints - TODO: implement when needed
		// analytics := v1.Group("/analytics")
		// {
		// 	analytics.GET("/sectors", getSectorPerformance)
		// 	analytics.GET("/trends", getMarketTrends)
		// 	analytics.GET("/screener", runStockScreener)
		// }

		// User endpoints - TODO: implement when needed
		// users := v1.Group("/users")
		// {
		// 	users.POST("/register", registerUser)
		// 	users.POST("/login", loginUser)
		// 	users.GET("/profile", getUserProfile)
		// 	users.PUT("/profile", updateUserProfile)
		// }
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting InvestorCenter API server on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// Market data handlers
func getMarketIndices(c *gin.Context) {
	// Mock data - replace with real market data API
	indices := []gin.H{
		{
			"symbol":        "^GSPC",
			"name":          "S&P 500",
			"price":         4567.89,
			"change":        23.45,
			"changePercent": 0.52,
			"lastUpdated":   time.Now().UTC(),
		},
		{
			"symbol":        "^DJI",
			"name":          "Dow Jones",
			"price":         35432.10,
			"change":        -45.67,
			"changePercent": -0.13,
			"lastUpdated":   time.Now().UTC(),
		},
		{
			"symbol":        "^IXIC",
			"name":          "NASDAQ",
			"price":         14123.45,
			"change":        67.89,
			"changePercent": 0.48,
			"lastUpdated":   time.Now().UTC(),
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"data": indices,
		"meta": gin.H{
			"count":     len(indices),
			"timestamp": time.Now().UTC(),
		},
	})
}

func searchSecurities(c *gin.Context) {
	query := c.Query("q")

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	// Use service layer for database operations
	stockService := services.NewStockService()
	stocks, err := stockService.SearchStocks(c.Request.Context(), query, 10)
	if err != nil {
		log.Printf("Database search failed: %v", err)
		// Fall back to mock data
		results := []gin.H{
			{
				"symbol":   "AAPL",
				"name":     "Apple Inc.",
				"type":     "stock",
				"exchange": "NASDAQ",
			},
			{
				"symbol":   "GOOGL",
				"name":     "Alphabet Inc.",
				"type":     "stock",
				"exchange": "NASDAQ",
			},
		}
		c.JSON(http.StatusOK, gin.H{
			"data": results,
			"meta": gin.H{
				"query":     query,
				"count":     len(results),
				"timestamp": time.Now().UTC(),
				"source":    "mock",
			},
		})
		return
	}

	// Convert to API format
	results := make([]gin.H, len(stocks))
	for i, stock := range stocks {
		results[i] = gin.H{
			"symbol":   stock.Symbol,
			"name":     stock.Name,
			"type":     "stock",
			"exchange": stock.Exchange,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": results,
		"meta": gin.H{
			"query":     query,
			"count":     len(results),
			"timestamp": time.Now().UTC(),
			"source":    "database",
		},
	})
}
