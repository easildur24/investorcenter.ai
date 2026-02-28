package cache

import (
	"context"
	"log"
	"os"

	"github.com/go-redis/redis/v8"
)

var client *redis.Client

// Initialize sets up the Redis client. Non-fatal — if Redis is unavailable,
// ingestion still works (S3 upload succeeds, Redis write is best-effort).
func Initialize() {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	client = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("WARNING: Redis not reachable at %s: %v (ingestion will still work, Redis writes will be skipped)", addr, err)
	} else {
		log.Printf("Redis connected at %s", addr)
	}
}

// Client returns the Redis client. May be nil if Initialize was never called.
func Client() *redis.Client {
	return client
}

// Close shuts down the Redis connection.
func Close() {
	if client != nil {
		client.Close()
	}
}
