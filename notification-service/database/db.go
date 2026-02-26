package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"

	"notification-service/config"
)

// DB wraps *sql.DB for the notification service.
type DB struct {
	*sql.DB
}

// Initialize creates a PostgreSQL connection.
// Called once in Lambda init() and reused across warm invocations.
func Initialize(cfg *config.Config) *DB {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// Conservative pool settings for Lambda (reserved concurrency = 1)
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("✅ Database connection established")
	return &DB{db}
}
