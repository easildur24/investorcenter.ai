package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"investorcenter-api/auth"
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

	// Start rate limiter cleanup
	auth.StartRateLimiterCleanup(auth.GetLoginLimiter())

	// Auth routes (public, no middleware)
	authRoutes := r.Group("/api/v1/auth")
	{
		// Rate limit on login/signup to prevent brute force
		authRoutes.POST("/signup", auth.RateLimitMiddleware(auth.GetLoginLimiter()), handlers.Signup)
		authRoutes.POST("/login", auth.RateLimitMiddleware(auth.GetLoginLimiter()), handlers.Login)
		authRoutes.POST("/refresh", handlers.RefreshToken)
		authRoutes.POST("/logout", handlers.Logout)
		authRoutes.GET("/verify-email", handlers.VerifyEmail)
		authRoutes.POST("/forgot-password", handlers.ForgotPassword)
		authRoutes.POST("/reset-password", handlers.ResetPassword)
	}

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
			crypto.GET("/", handlers.GetAllCryptos) // All crypto prices with pagination
		}

		// Fundamental metrics endpoints
		// TODO: Implement fundamental handlers
		// fundamentals := v1.Group("/fundamentals")
		// {
		// 	fundamentals.GET("/", handlers.ListFundamentals)                        // List all symbols with metrics
		// 	fundamentals.GET("/:symbol", handlers.GetFundamentalsSimple)            // Get stored metrics for symbol (simple JSON)
		// 	fundamentals.POST("/:symbol/calculate", handlers.CalculateFundamentals) // Calculate and store metrics
		// 	fundamentals.POST("/:symbol/refresh", handlers.RefreshFundamentals)     // Refresh metrics
		// }

		// Volume endpoints for bulk operations
		volume := v1.Group("/volume")
		{
			volume.POST("/bulk", handlers.GetBulkVolume) // Get volume for multiple symbols
			volume.GET("/top", handlers.GetTopVolume)    // Get top stocks by volume
		}

		// Reddit popularity endpoints
		reddit := v1.Group("/reddit")
		{
			reddit.GET("/heatmap", handlers.GetRedditHeatmap)                      // Get trending tickers heatmap (days=7, top=50)
			reddit.GET("/ticker/:symbol/history", handlers.GetTickerRedditHistory) // Get Reddit history for specific ticker (days=30)
		}

		// Portfolio endpoints
		portfolios := v1.Group("/portfolios")
		{
			portfolios.GET("/", getPortfolios)
			portfolios.POST("/", createPortfolio)
			portfolios.GET("/:id", getPortfolio)
			portfolios.PUT("/:id", updatePortfolio)
			portfolios.DELETE("/:id", deletePortfolio)
			portfolios.GET("/:id/performance", getPortfolioPerformance)
		}

		// Analytics endpoints
		analytics := v1.Group("/analytics")
		{
			analytics.GET("/sectors", getSectorPerformance)
			analytics.GET("/trends", getMarketTrends)
			analytics.GET("/screener", runStockScreener)
		}

		// User endpoints (deprecated - use /auth routes instead)
		users := v1.Group("/users")
		{
			users.POST("/register", registerUser)
			users.POST("/login", loginUser)
			users.GET("/profile", getUserProfile)
			users.PUT("/profile", updateUserProfile)
		}
	}

	// Protected user routes (require authentication)
	userRoutes := v1.Group("/user")
	userRoutes.Use(auth.AuthMiddleware())
	{
		userRoutes.GET("/me", handlers.GetCurrentUser)
		userRoutes.PUT("/me", handlers.UpdateProfile)
		userRoutes.PUT("/password", handlers.ChangePassword)
		userRoutes.DELETE("/me", handlers.DeleteAccount)
	}

	// Watch List routes (protected, require authentication)
	watchListRoutes := v1.Group("/watchlists")
	watchListRoutes.Use(auth.AuthMiddleware())
	{
		watchListRoutes.GET("", handlers.ListWatchLists)         // GET /api/v1/watchlists
		watchListRoutes.POST("", handlers.CreateWatchList)       // POST /api/v1/watchlists
		watchListRoutes.GET("/:id", handlers.GetWatchList)       // GET /api/v1/watchlists/:id
		watchListRoutes.PUT("/:id", handlers.UpdateWatchList)    // PUT /api/v1/watchlists/:id
		watchListRoutes.DELETE("/:id", handlers.DeleteWatchList) // DELETE /api/v1/watchlists/:id

		// Watch list items
		watchListRoutes.POST("/:id/items", handlers.AddTickerToWatchList)                // POST /api/v1/watchlists/:id/items
		watchListRoutes.DELETE("/:id/items/:symbol", handlers.RemoveTickerFromWatchList) // DELETE /api/v1/watchlists/:id/items/:symbol
		watchListRoutes.PUT("/:id/items/:symbol", handlers.UpdateWatchListItem)          // PUT /api/v1/watchlists/:id/items/:symbol
		watchListRoutes.POST("/:id/bulk", handlers.BulkAddTickers)                       // POST /api/v1/watchlists/:id/bulk
		watchListRoutes.POST("/:id/reorder", handlers.ReorderWatchListItems)             // POST /api/v1/watchlists/:id/reorder

		// Heatmap routes
		watchListRoutes.GET("/:id/heatmap", handlers.GetHeatmapData)                               // GET /api/v1/watchlists/:id/heatmap
		watchListRoutes.GET("/:id/heatmap/configs", handlers.ListHeatmapConfigs)                   // GET /api/v1/watchlists/:id/heatmap/configs
		watchListRoutes.POST("/:id/heatmap/configs", handlers.CreateHeatmapConfig)                 // POST /api/v1/watchlists/:id/heatmap/configs
		watchListRoutes.PUT("/:id/heatmap/configs/:configId", handlers.UpdateHeatmapConfig)        // PUT /api/v1/watchlists/:id/heatmap/configs/:configId
		watchListRoutes.DELETE("/:id/heatmap/configs/:configId", handlers.DeleteHeatmapConfig)     // DELETE /api/v1/watchlists/:id/heatmap/configs/:configId
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

// Portfolio handlers
func getPortfolios(c *gin.Context) {
	// Mock portfolios - replace with database query
	portfolios := []gin.H{
		{
			"id":            1,
			"name":          "Growth Portfolio",
			"description":   "High growth technology stocks",
			"value":         125000.50,
			"change":        2345.67,
			"changePercent": 1.91,
			"createdAt":     time.Now().Add(-30 * 24 * time.Hour).UTC(),
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"data": portfolios,
		"meta": gin.H{
			"count":     len(portfolios),
			"timestamp": time.Now().UTC(),
		},
	})
}

func createPortfolio(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Mock creation - replace with database insert
	portfolio := gin.H{
		"id":            123,
		"name":          req.Name,
		"description":   req.Description,
		"value":         0,
		"change":        0,
		"changePercent": 0,
		"createdAt":     time.Now().UTC(),
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": portfolio,
		"meta": gin.H{
			"timestamp": time.Now().UTC(),
		},
	})
}

func getPortfolio(c *gin.Context) {
	id := c.Param("id")

	// Mock portfolio - replace with database query
	portfolio := gin.H{
		"id":            id,
		"name":          "Growth Portfolio",
		"description":   "High growth technology stocks",
		"value":         125000.50,
		"change":        2345.67,
		"changePercent": 1.91,
		"holdings": []gin.H{
			{
				"symbol":        "AAPL",
				"name":          "Apple Inc.",
				"shares":        100,
				"avgPrice":      150.00,
				"currentPrice":  175.43,
				"value":         17543.00,
				"change":        2543.00,
				"changePercent": 16.95,
			},
		},
		"createdAt": time.Now().Add(-30 * 24 * time.Hour).UTC(),
	}

	c.JSON(http.StatusOK, gin.H{
		"data": portfolio,
		"meta": gin.H{
			"timestamp": time.Now().UTC(),
		},
	})
}

func updatePortfolio(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Mock update - replace with database update
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"id":          id,
			"name":        req.Name,
			"description": req.Description,
			"updatedAt":   time.Now().UTC(),
		},
		"meta": gin.H{
			"timestamp": time.Now().UTC(),
		},
	})
}

func deletePortfolio(c *gin.Context) {
	id := c.Param("id")

	// Mock deletion - replace with database delete
	c.JSON(http.StatusOK, gin.H{
		"message": "Portfolio deleted successfully",
		"data": gin.H{
			"id": id,
		},
		"meta": gin.H{
			"timestamp": time.Now().UTC(),
		},
	})
}

func getPortfolioPerformance(c *gin.Context) {
	id := c.Param("id")
	period := c.DefaultQuery("period", "1m")

	// Mock performance data
	var performance []gin.H
	baseValue := 100000.0

	for i := 0; i < 30; i++ {
		value := baseValue + float64(i)*1000 + float64(i%7-3)*500
		performance = append(performance, gin.H{
			"date":  time.Now().Add(-time.Duration(30-i) * 24 * time.Hour).UTC(),
			"value": value,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"portfolioId": id,
			"period":      period,
			"performance": performance,
		},
		"meta": gin.H{
			"count":     len(performance),
			"timestamp": time.Now().UTC(),
		},
	})
}

// Analytics handlers
func getSectorPerformance(c *gin.Context) {
	sectors := []gin.H{
		{"name": "Technology", "change": 2.34, "changePercent": 1.45},
		{"name": "Healthcare", "change": -1.23, "changePercent": -0.67},
		{"name": "Financial", "change": 0.89, "changePercent": 0.34},
		{"name": "Energy", "change": 3.45, "changePercent": 2.78},
		{"name": "Consumer", "change": 1.67, "changePercent": 0.89},
	}

	c.JSON(http.StatusOK, gin.H{
		"data": sectors,
		"meta": gin.H{
			"count":     len(sectors),
			"timestamp": time.Now().UTC(),
		},
	})
}

func getMarketTrends(c *gin.Context) {
	trends := gin.H{
		"bullishSentiment": 65.4,
		"bearishSentiment": 34.6,
		"volatilityIndex":  18.7,
		"topGainers": []gin.H{
			{"symbol": "NVDA", "change": 5.67, "changePercent": 3.45},
			{"symbol": "TSLA", "change": 8.90, "changePercent": 4.23},
		},
		"topLosers": []gin.H{
			{"symbol": "META", "change": -4.32, "changePercent": -2.10},
			{"symbol": "NFLX", "change": -6.78, "changePercent": -1.89},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"data": trends,
		"meta": gin.H{
			"timestamp": time.Now().UTC(),
		},
	})
}

func runStockScreener(c *gin.Context) {
	// Mock screener results
	results := []gin.H{
		{
			"symbol":        "AAPL",
			"name":          "Apple Inc.",
			"price":         175.43,
			"marketCap":     2800000000000,
			"pe":            28.5,
			"dividendYield": 0.55,
			"score":         8.5,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"data": results,
		"meta": gin.H{
			"count":     len(results),
			"timestamp": time.Now().UTC(),
		},
	})
}

// User handlers
func registerUser(c *gin.Context) {
	var req struct {
		Email     string `json:"email" binding:"required,email"`
		Password  string `json:"password" binding:"required,min=8"`
		FirstName string `json:"firstName" binding:"required"`
		LastName  string `json:"lastName" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Mock user creation - replace with database insert and password hashing
	user := gin.H{
		"id":        123,
		"email":     req.Email,
		"firstName": req.FirstName,
		"lastName":  req.LastName,
		"createdAt": time.Now().UTC(),
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": user,
		"meta": gin.H{
			"timestamp": time.Now().UTC(),
		},
	})
}

func loginUser(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Mock login - replace with database query and password verification
	token := "mock-jwt-token-here"

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"token": token,
			"user": gin.H{
				"id":        123,
				"email":     req.Email,
				"firstName": "John",
				"lastName":  "Doe",
			},
		},
		"meta": gin.H{
			"timestamp": time.Now().UTC(),
		},
	})
}

func getUserProfile(c *gin.Context) {
	// Mock user profile - replace with database query and JWT verification
	user := gin.H{
		"id":        123,
		"email":     "john.doe@example.com",
		"firstName": "John",
		"lastName":  "Doe",
		"createdAt": time.Now().Add(-90 * 24 * time.Hour).UTC(),
	}

	c.JSON(http.StatusOK, gin.H{
		"data": user,
		"meta": gin.H{
			"timestamp": time.Now().UTC(),
		},
	})
}

func updateUserProfile(c *gin.Context) {
	var req struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Mock update - replace with database update
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"id":        123,
			"firstName": req.FirstName,
			"lastName":  req.LastName,
			"updatedAt": time.Now().UTC(),
		},
		"meta": gin.H{
			"timestamp": time.Now().UTC(),
		},
	})
}
