package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"investorcenter-api/database"
	"investorcenter-api/services/social"
	"investorcenter-api/services/social/pipeline"
	"investorcenter-api/services/social/reddit"
)

func main() {
	log.Println("Reddit Post Collector starting...")

	// Initialize database connection
	if err := database.Initialize(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Load Reddit credentials from environment
	redditConfig := reddit.Config{
		ClientID:     os.Getenv("REDDIT_CLIENT_ID"),
		ClientSecret: os.Getenv("REDDIT_CLIENT_SECRET"),
	}

	if redditConfig.ClientID == "" || redditConfig.ClientSecret == "" {
		log.Fatal("REDDIT_CLIENT_ID and REDDIT_CLIENT_SECRET environment variables required")
	}

	// Create Reddit data source
	redditDS := reddit.NewDataSource(redditConfig)
	if !redditDS.IsEnabled() {
		log.Fatal("Reddit data source failed to initialize")
	}

	// Create collector
	collector, err := pipeline.NewCollector(
		[]social.SocialDataSource{redditDS},
		pipeline.Config{
			Subreddits:    []string{"wallstreetbets", "stocks", "options", "investing", "Daytrading"},
			MinEngagement: 10,
		},
	)
	if err != nil {
		log.Fatalf("Failed to create collector: %v", err)
	}

	// Log stats
	stats := collector.GetStats()
	log.Printf("Initialized with %d lexicon terms, %d valid tickers",
		stats["lexicon_terms"], stats["valid_tickers"])

	// Setup context with cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutdown signal received")
		cancel()
	}()

	// Run collection
	if err := collector.Run(ctx); err != nil {
		log.Fatalf("Collection failed: %v", err)
	}

	log.Println("Reddit Post Collector completed successfully")
}
