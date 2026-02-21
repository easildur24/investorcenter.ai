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
	"task-service/auth"
	"task-service/database"
	"task-service/handlers"
	"task-service/storage"
)

func main() {
	// Load .env for local dev (ignore error in production)
	godotenv.Load()

	// Validate JWT secret before starting — fail fast if missing or too short
	auth.ValidateJWTSecret()

	// Initialize database
	database.Initialize()
	defer database.Close()

	// Initialize S3 storage (for admin file downloads)
	if err := storage.Initialize(); err != nil {
		log.Printf("S3 storage initialization failed: %v", err)
		log.Println("File download features disabled")
	}

	r := gin.Default()

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
		if err := database.HealthCheck(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Admin worker management routes (proxy strips /api/v1 prefix)
	adminRoutes := r.Group("/admin/workers")
	adminRoutes.Use(auth.AuthMiddleware(), auth.AdminMiddleware())
	{
		adminRoutes.GET("", handlers.ListWorkers)
		adminRoutes.POST("", handlers.RegisterWorker)
		adminRoutes.DELETE("/:id", handlers.DeleteWorker)
		// Task type management
		adminRoutes.GET("/task-types", handlers.ListTaskTypes)
		adminRoutes.POST("/task-types", handlers.CreateTaskType)
		adminRoutes.PUT("/task-types/:id", handlers.UpdateTaskType)
		adminRoutes.DELETE("/task-types/:id", handlers.DeleteTaskType)
		// Task management
		adminRoutes.GET("/tasks", handlers.ListTasks)
		adminRoutes.POST("/tasks", handlers.CreateTask)
		adminRoutes.GET("/tasks/:id", handlers.GetTask)
		adminRoutes.PUT("/tasks/:id", handlers.UpdateTask)
		adminRoutes.DELETE("/tasks/:id", handlers.DeleteTask)
		adminRoutes.GET("/tasks/:id/updates", handlers.ListTaskUpdates)
		adminRoutes.POST("/tasks/:id/updates", handlers.CreateTaskUpdate)
		adminRoutes.GET("/tasks/:id/data", handlers.AdminGetTaskData)
		adminRoutes.GET("/tasks/:id/files", handlers.AdminListTaskFiles)
		adminRoutes.GET("/tasks/:id/files/:fileId/download", handlers.AdminDownloadTaskFile)
	}

	// Worker API routes
	workerRoutes := r.Group("/worker")
	workerRoutes.Use(auth.AuthMiddleware())
	{
		workerRoutes.GET("/task-types", handlers.ListTaskTypes)
		workerRoutes.GET("/task-types/:id", handlers.WorkerGetTaskType)
		workerRoutes.POST("/next-task", handlers.WorkerNextTask)
		workerRoutes.GET("/tasks/:id", handlers.WorkerGetTask)
		workerRoutes.PUT("/tasks/:id/status", handlers.WorkerUpdateTaskStatus)
		workerRoutes.POST("/tasks/:id/result", handlers.WorkerPostResult)
		workerRoutes.GET("/tasks/:id/updates", handlers.WorkerGetTaskUpdates)
		workerRoutes.POST("/tasks/:id/updates", handlers.WorkerPostUpdate)
		workerRoutes.POST("/tasks/:id/data", handlers.WorkerPostTaskData)
		workerRoutes.POST("/tasks/:id/files", handlers.WorkerRegisterTaskFile)
		workerRoutes.POST("/heartbeat", handlers.WorkerHeartbeat)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Task service starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down task service...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Task service exited")
}
