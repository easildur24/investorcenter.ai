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
	config.ExposeHeaders = []string{"Content-Length"}
	config.MaxAge = 12 * time.Hour
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
			markets.GET("/indices", handlers.GetMarketIndices)
			markets.GET("/movers", handlers.GetMarketMovers)
			markets.GET("/search", searchSecurities)
		}

		// Ticker page endpoints
		tickers := v1.Group("/tickers")
		{
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

			// Key stats endpoints (user-ingested data)
			tickers.GET("/:symbol/keystats", handlers.GetKeyStats)       // Get key stats data
			tickers.POST("/:symbol/keystats", handlers.PostKeyStats)     // Upload key stats data
			tickers.DELETE("/:symbol/keystats", handlers.DeleteKeyStats) // Delete key stats data
		}

		// IC Score endpoints
		stocks := v1.Group("/stocks")
		{
			stocks.GET("/:ticker/ic-score", handlers.GetICScore)                      // Get IC Score for a ticker
			stocks.GET("/:ticker/ic-score/history", handlers.GetICScoreHistory)       // Get IC Score history
			stocks.GET("/:ticker/financials", handlers.GetFinancialMetrics)           // Get financial metrics from SEC filings (legacy)
			stocks.GET("/:ticker/metrics", handlers.GetComprehensiveFinancialMetrics) // Get comprehensive financial metrics (FMP)
			stocks.GET("/:ticker/risk", handlers.GetRiskMetrics)                      // Get risk metrics (Beta, Alpha, Sharpe)
			stocks.GET("/:ticker/technical", handlers.GetTechnicalIndicators)         // Get technical indicators

			// Financial Statements endpoints (SEC EDGAR data)
			financialsHandler := handlers.NewFinancialsHandler()
			stocks.GET("/:ticker/financials/all", financialsHandler.GetAllFinancials)           // Get all financial statements summary
			stocks.GET("/:ticker/financials/income", financialsHandler.GetIncomeStatements)     // Get income statements
			stocks.GET("/:ticker/financials/balance", financialsHandler.GetBalanceSheets)       // Get balance sheets
			stocks.GET("/:ticker/financials/cashflow", financialsHandler.GetCashFlowStatements) // Get cash flow statements
			stocks.GET("/:ticker/financials/ratios", financialsHandler.GetRatios)               // Get financial ratios
			stocks.POST("/:ticker/financials/refresh", financialsHandler.RefreshFinancials)     // Refresh financial data
		}

		// IC Score Backtest endpoints
		backtestService := services.NewBacktestService()
		backtestHandler := handlers.NewBacktestHandler(backtestService)

		backtestRoutes := v1.Group("/ic-scores/backtest")
		{
			backtestRoutes.GET("/latest", backtestHandler.GetLatestBacktest)                // GET /api/v1/ic-scores/backtest/latest
			backtestRoutes.GET("/config/default", backtestHandler.GetDefaultConfig)         // GET /api/v1/ic-scores/backtest/config/default
			backtestRoutes.GET("/quick", backtestHandler.RunQuickBacktest)                  // GET /api/v1/ic-scores/backtest/quick
			backtestRoutes.POST("", backtestHandler.RunBacktest)                            // POST /api/v1/ic-scores/backtest
			backtestRoutes.GET("/charts", backtestHandler.GetBacktestCharts)                // GET /api/v1/ic-scores/backtest/charts
			backtestRoutes.POST("/jobs", backtestHandler.SubmitBacktestJob)                 // POST /api/v1/ic-scores/backtest/jobs
			backtestRoutes.GET("/jobs/:jobId", backtestHandler.GetBacktestJobStatus)        // GET /api/v1/ic-scores/backtest/jobs/:jobId
			backtestRoutes.GET("/jobs/:jobId/result", backtestHandler.GetBacktestJobResult) // GET /api/v1/ic-scores/backtest/jobs/:jobId/result
		}

		// Crypto endpoints
		crypto := v1.Group("/crypto")
		{
			crypto.GET("/", handlers.GetAllCryptos) // All crypto prices with pagination
		}

		// Reddit popularity endpoints
		reddit := v1.Group("/reddit")
		{
			reddit.GET("/heatmap", handlers.GetRedditHeatmap)                      // Get trending tickers heatmap (days=7, top=50)
			reddit.GET("/ticker/:symbol/history", handlers.GetTickerRedditHistory) // Get Reddit history for specific ticker (days=30)
		}

		// Social sentiment endpoints
		sentiment := v1.Group("/sentiment")
		{
			// IMPORTANT: /trending must come before /:ticker to avoid matching "trending" as a ticker
			sentiment.GET("/trending", handlers.GetTrendingSentiment)             // GET /api/v1/sentiment/trending?period=24h&limit=20
			sentiment.GET("/:ticker", handlers.GetTickerSentiment)                // GET /api/v1/sentiment/AAPL
			sentiment.GET("/:ticker/history", handlers.GetTickerSentimentHistory) // GET /api/v1/sentiment/AAPL/history?days=30
			sentiment.GET("/:ticker/posts", handlers.GetTickerPosts)              // GET /api/v1/sentiment/AAPL/posts?limit=10
		}

		// Screener endpoint (real implementation in handlers)
		// Note: Portfolio and Analytics endpoints were removed (mock-only, not implemented)

		// Screener endpoints
		screener := v1.Group("/screener")
		{
			screener.GET("/stocks", handlers.GetScreenerStocks)
		}

		// Logo proxy endpoint (proxies logos from Polygon.io with API key)
		v1.GET("/logos/:symbol", handlers.ProxyLogo)

		// Note: User endpoints removed (deprecated mock handlers - use /api/v1/auth routes instead)
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
		watchListRoutes.GET("/:id/heatmap", handlers.GetHeatmapData)                           // GET /api/v1/watchlists/:id/heatmap
		watchListRoutes.GET("/:id/heatmap/configs", handlers.ListHeatmapConfigs)               // GET /api/v1/watchlists/:id/heatmap/configs
		watchListRoutes.POST("/:id/heatmap/configs", handlers.CreateHeatmapConfig)             // POST /api/v1/watchlists/:id/heatmap/configs
		watchListRoutes.PUT("/:id/heatmap/configs/:configId", handlers.UpdateHeatmapConfig)    // PUT /api/v1/watchlists/:id/heatmap/configs/:configId
		watchListRoutes.DELETE("/:id/heatmap/configs/:configId", handlers.DeleteHeatmapConfig) // DELETE /api/v1/watchlists/:id/heatmap/configs/:configId
	}

	// Initialize services for alert/notification/subscription features
	emailService := services.NewEmailService()
	alertService := services.NewAlertService()
	notificationService := services.NewNotificationService(emailService)
	subscriptionService := services.NewSubscriptionService()
	cronjobService := services.NewCronjobService()

	// Initialize handlers
	alertHandler := handlers.NewAlertHandler(alertService)
	notificationHandler := handlers.NewNotificationHandler(notificationService)
	subscriptionHandler := handlers.NewSubscriptionHandler(subscriptionService)
	cronjobHandler := handlers.NewCronjobHandler(cronjobService)

	// Alert routes (protected, require authentication)
	alertRoutes := v1.Group("/alerts")
	alertRoutes.Use(auth.AuthMiddleware())
	{
		alertRoutes.GET("", alertHandler.ListAlertRules)         // GET /api/v1/alerts
		alertRoutes.POST("", alertHandler.CreateAlertRule)       // POST /api/v1/alerts
		alertRoutes.GET("/:id", alertHandler.GetAlertRule)       // GET /api/v1/alerts/:id
		alertRoutes.PUT("/:id", alertHandler.UpdateAlertRule)    // PUT /api/v1/alerts/:id
		alertRoutes.DELETE("/:id", alertHandler.DeleteAlertRule) // DELETE /api/v1/alerts/:id

		// Alert logs
		alertRoutes.GET("/logs", alertHandler.ListAlertLogs)                // GET /api/v1/alerts/logs
		alertRoutes.POST("/logs/:id/read", alertHandler.MarkAlertLogRead)   // POST /api/v1/alerts/logs/:id/read
		alertRoutes.POST("/logs/:id/dismiss", alertHandler.DismissAlertLog) // POST /api/v1/alerts/logs/:id/dismiss
	}

	// Notification routes (protected, require authentication)
	notificationRoutes := v1.Group("/notifications")
	notificationRoutes.Use(auth.AuthMiddleware())
	{
		notificationRoutes.GET("", notificationHandler.GetInAppNotifications)              // GET /api/v1/notifications
		notificationRoutes.GET("/unread-count", notificationHandler.GetUnreadCount)        // GET /api/v1/notifications/unread-count
		notificationRoutes.POST("/:id/read", notificationHandler.MarkNotificationRead)     // POST /api/v1/notifications/:id/read
		notificationRoutes.POST("/read-all", notificationHandler.MarkAllNotificationsRead) // POST /api/v1/notifications/read-all
		notificationRoutes.POST("/:id/dismiss", notificationHandler.DismissNotification)   // POST /api/v1/notifications/:id/dismiss

		// Notification preferences
		notificationRoutes.GET("/preferences", notificationHandler.GetNotificationPreferences)    // GET /api/v1/notifications/preferences
		notificationRoutes.PUT("/preferences", notificationHandler.UpdateNotificationPreferences) // PUT /api/v1/notifications/preferences
	}

	// Subscription routes (protected, require authentication)
	subscriptionRoutes := v1.Group("/subscriptions")
	subscriptionRoutes.Use(auth.AuthMiddleware())
	{
		subscriptionRoutes.GET("/plans", subscriptionHandler.ListSubscriptionPlans)   // GET /api/v1/subscriptions/plans
		subscriptionRoutes.GET("/plans/:id", subscriptionHandler.GetSubscriptionPlan) // GET /api/v1/subscriptions/plans/:id
		subscriptionRoutes.GET("/me", subscriptionHandler.GetUserSubscription)        // GET /api/v1/subscriptions/me
		subscriptionRoutes.POST("", subscriptionHandler.CreateSubscription)           // POST /api/v1/subscriptions
		subscriptionRoutes.PUT("/me", subscriptionHandler.UpdateSubscription)         // PUT /api/v1/subscriptions/me
		subscriptionRoutes.POST("/me/cancel", subscriptionHandler.CancelSubscription) // POST /api/v1/subscriptions/me/cancel
		subscriptionRoutes.GET("/limits", subscriptionHandler.GetSubscriptionLimits)  // GET /api/v1/subscriptions/limits
		subscriptionRoutes.GET("/payments", subscriptionHandler.GetPaymentHistory)    // GET /api/v1/subscriptions/payments
	}

	// Admin cronjob monitoring routes (protected, require authentication + admin role)
	cronjobRoutes := v1.Group("/admin/cronjobs")
	cronjobRoutes.Use(auth.AuthMiddleware())
	cronjobRoutes.Use(auth.AdminMiddleware())
	{
		cronjobRoutes.GET("/overview", cronjobHandler.GetOverview)               // GET /api/v1/admin/cronjobs/overview
		cronjobRoutes.GET("/schedules", cronjobHandler.GetAllSchedules)          // GET /api/v1/admin/cronjobs/schedules
		cronjobRoutes.GET("/metrics", cronjobHandler.GetMetrics)                 // GET /api/v1/admin/cronjobs/metrics
		cronjobRoutes.GET("/:jobName/history", cronjobHandler.GetJobHistory)     // GET /api/v1/admin/cronjobs/:jobName/history
		cronjobRoutes.GET("/details/:executionId", cronjobHandler.GetJobDetails) // GET /api/v1/admin/cronjobs/details/:executionId
	}

	// Admin data query routes (protected, require authentication + admin role)
	adminDataHandler := handlers.NewAdminDataHandler(database.DB)
	adminRoutes := v1.Group("/admin")
	adminRoutes.Use(auth.AuthMiddleware())
	adminRoutes.Use(auth.AdminMiddleware())
	{
		adminRoutes.GET("/stocks", adminDataHandler.GetStocks)                    // GET /api/v1/admin/stocks
		adminRoutes.GET("/users", adminDataHandler.GetUsers)                      // GET /api/v1/admin/users
		adminRoutes.GET("/news", adminDataHandler.GetNewsArticles)                // GET /api/v1/admin/news
		adminRoutes.GET("/fundamentals", adminDataHandler.GetFundamentals)        // GET /api/v1/admin/fundamentals
		adminRoutes.GET("/sec-financials", adminDataHandler.GetSECFinancials)     // GET /api/v1/admin/sec-financials
		adminRoutes.GET("/ttm-financials", adminDataHandler.GetTTMFinancials)     // GET /api/v1/admin/ttm-financials
		adminRoutes.GET("/valuation-ratios", adminDataHandler.GetValuationRatios) // GET /api/v1/admin/valuation-ratios
		adminRoutes.GET("/alerts", adminDataHandler.GetAlerts)                    // GET /api/v1/admin/alerts
		adminRoutes.GET("/watchlists", adminDataHandler.GetWatchLists)            // GET /api/v1/admin/watchlists
		adminRoutes.GET("/stats", adminDataHandler.GetDatabaseStats)              // GET /api/v1/admin/stats
		// IC Score pipeline data (from IC Score service database)
		adminRoutes.GET("/analyst-ratings", adminDataHandler.GetAnalystRatings)               // GET /api/v1/admin/analyst-ratings
		adminRoutes.GET("/insider-trades", adminDataHandler.GetInsiderTrades)                 // GET /api/v1/admin/insider-trades
		adminRoutes.GET("/institutional-holdings", adminDataHandler.GetInstitutionalHoldings) // GET /api/v1/admin/institutional-holdings
		adminRoutes.GET("/technical-indicators", adminDataHandler.GetTechnicalIndicators)     // GET /api/v1/admin/technical-indicators
		adminRoutes.GET("/companies", adminDataHandler.GetCompanies)                          // GET /api/v1/admin/companies
		adminRoutes.GET("/risk-metrics", adminDataHandler.GetRiskMetrics)                     // GET /api/v1/admin/risk-metrics
		adminRoutes.GET("/ic-scores", handlers.GetICScores)                                   // GET /api/v1/admin/ic-scores

		// Notes/brainstorming endpoints
		notes := adminRoutes.Group("/notes")
		{
			notes.GET("/tree", handlers.GetNotesTree) // GET /api/v1/admin/notes/tree
			// Groups
			notes.GET("/groups", handlers.ListFeatureGroups)         // GET /api/v1/admin/notes/groups
			notes.POST("/groups", handlers.CreateFeatureGroup)       // POST /api/v1/admin/notes/groups
			notes.PUT("/groups/:id", handlers.UpdateFeatureGroup)    // PUT /api/v1/admin/notes/groups/:id
			notes.DELETE("/groups/:id", handlers.DeleteFeatureGroup) // DELETE /api/v1/admin/notes/groups/:id
			// Features
			notes.GET("/groups/:groupId/features", handlers.ListFeatures)   // GET /api/v1/admin/notes/groups/:groupId/features
			notes.POST("/groups/:groupId/features", handlers.CreateFeature) // POST /api/v1/admin/notes/groups/:groupId/features
			notes.PUT("/features/:id", handlers.UpdateFeature)              // PUT /api/v1/admin/notes/features/:id
			notes.DELETE("/features/:id", handlers.DeleteFeature)           // DELETE /api/v1/admin/notes/features/:id
			// Notes
			notes.GET("/features/:featureId/notes", handlers.ListFeatureNotes)   // GET /api/v1/admin/notes/features/:featureId/notes
			notes.POST("/features/:featureId/notes", handlers.CreateFeatureNote) // POST /api/v1/admin/notes/features/:featureId/notes
			notes.PUT("/notes/:id", handlers.UpdateFeatureNote)                  // PUT /api/v1/admin/notes/notes/:id
			notes.DELETE("/notes/:id", handlers.DeleteFeatureNote)               // DELETE /api/v1/admin/notes/notes/:id
		}

	}

	// Worker/task management routes — proxied to task-service (protected, require authentication + admin role)
	taskProxy := services.TaskServiceProxy()
	workerRoutes := v1.Group("")
	workerRoutes.Use(auth.AuthMiddleware())
	workerRoutes.Use(auth.AdminMiddleware())
	{
		workerRoutes.Any("/admin/workers", taskProxy)
		workerRoutes.Any("/admin/workers/*path", taskProxy)
		workerRoutes.Any("/worker/*path", taskProxy)
	}

	// Data ingestion routes — proxied to data-ingestion-service (protected, require authentication + admin role)
	ingestProxy := services.DataIngestionProxy()
	ingestRoutes := v1.Group("")
	ingestRoutes.Use(auth.AuthMiddleware())
	ingestRoutes.Use(auth.AdminMiddleware())
	{
		ingestRoutes.Any("/ingest", ingestProxy)
		ingestRoutes.Any("/ingest/*path", ingestProxy)
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
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Search temporarily unavailable",
			"details": "Database connection failed",
		})
		return
	}

	// Convert to API format
	results := make([]gin.H, len(stocks))
	for i, stock := range stocks {
		results[i] = gin.H{
			"symbol":   stock.Symbol,
			"name":     stock.Name,
			"type":     stock.AssetType,
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
