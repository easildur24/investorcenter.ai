// Notification service for InvestorCenter.ai.
//
// Consumes stock price updates from an SQS queue (published by the backend
// via SNS), evaluates alert rules in near real-time, and delivers
// notifications (in-app + email).
//
// Designed to run as a single-replica K8s deployment in the investorcenter
// namespace. Exposes only a /health endpoint for liveness/readiness probes.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"notification-service/config"
	"notification-service/consumer"
	"notification-service/database"
	"notification-service/delivery"
	"notification-service/evaluator"
)

func main() {
	log.Println("Starting notification service...")

	// 1. Load config
	cfg := config.Load()

	// 2. Initialize database
	db := database.Initialize(cfg)
	defer db.Close()

	// 3. Initialize SQS consumer
	sqsConsumer, err := consumer.New(cfg.SQSQueueURL, cfg.AWSRegion, cfg.SQSMaxMessages)
	if err != nil {
		log.Fatalf("Failed to create SQS consumer: %v", err)
	}

	// 4. Initialize delivery channels
	emailDelivery := delivery.NewEmailDelivery(cfg, db)
	inAppDelivery := delivery.NewInAppDelivery(db)
	router := delivery.NewRouter(emailDelivery, inAppDelivery)

	// 5. Initialize evaluator
	eval := evaluator.New(db, router)

	// 6. Start SQS consumer in background
	ctx, cancel := context.WithCancel(context.Background())
	go sqsConsumer.Start(ctx, eval.HandlePriceUpdate)

	// 7. Start health server
	healthSrv := startHealthServer(cfg.Port, db, sqsConsumer)

	log.Printf("Notification service running on port %s", cfg.Port)

	// 8. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down notification service...")
	cancel() // Stop SQS consumer

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Health server shutdown error: %v", err)
	}

	log.Println("Notification service stopped")
}

// startHealthServer creates an HTTP server with a /health endpoint
// for Kubernetes liveness and readiness probes.
func startHealthServer(port string, db *database.DB, sqsConsumer *consumer.Consumer) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		status := "ok"
		dbStatus := "connected"
		sqsStatus := "polling"

		// Check DB
		if err := db.Ping(); err != nil {
			status = "degraded"
			dbStatus = "error"
		}

		// Check SQS consumer
		if !sqsConsumer.IsHealthy() {
			status = "degraded"
			sqsStatus = "error"
		}

		w.Header().Set("Content-Type", "application/json")
		if status != "ok" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		json.NewEncoder(w).Encode(map[string]string{
			"status": status,
			"db":     dbStatus,
			"sqs":    sqsStatus,
		})
	})

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Health server error: %v", err)
		}
	}()

	return srv
}
