package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"data-ingestion-service/auth"
	"data-ingestion-service/database"
	"data-ingestion-service/handlers"
	"data-ingestion-service/handlers/ycharts"
	"data-ingestion-service/storage"
)

func main() {
	// Load .env for local dev (ignore error in production)
	godotenv.Load()

	// Initialize database
	if err := database.Initialize(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize S3 client
	if err := storage.Initialize(); err != nil {
		log.Fatalf("Failed to initialize S3 storage: %v", err)
	}

	r := gin.Default()

	// Increase max request body size to 12MB (raw_data can be up to 10MB + metadata)
	r.MaxMultipartMemory = 12 << 20

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:8080"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		dbErr := database.HealthCheck()
		s3Err := storage.HealthCheck()

		if dbErr != nil || s3Err != nil {
			errors := gin.H{}
			if dbErr != nil {
				errors["database"] = dbErr.Error()
			}
			if s3Err != nil {
				errors["s3"] = s3Err.Error()
			}
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"errors": errors,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Worker routes — any authenticated user can ingest data
	ingestRoutes := r.Group("/ingest")
	ingestRoutes.Use(auth.AuthMiddleware())
	{
		ingestRoutes.POST("", handlers.PostIngest)
		
		// YCharts endpoints
		ingestRoutes.POST("/ycharts/key_stats/:ticker", ycharts.PostKeyStats)
	}

	// Admin routes — list and view ingestion records
	adminRoutes := r.Group("/ingest")
	adminRoutes.Use(auth.AuthMiddleware(), auth.AdminMiddleware())
	{
		adminRoutes.GET("", handlers.ListIngestionLogs)
		adminRoutes.GET("/:id", handlers.GetIngestionLogByID)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8002"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Data ingestion service starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down data ingestion service...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Data ingestion service exited")
}
