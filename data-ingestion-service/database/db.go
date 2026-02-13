package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// DB holds the database connection
var DB *sqlx.DB

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// LoadConfigFromEnv loads database configuration from environment variables
func LoadConfigFromEnv() *DatabaseConfig {
	return &DatabaseConfig{
		Host:     getEnvWithDefault("DB_HOST", "localhost"),
		Port:     getEnvWithDefault("DB_PORT", "5432"),
		User:     getEnvWithDefault("DB_USER", "investorcenter"),
		Password: getEnvWithDefault("DB_PASSWORD", ""),
		DBName:   getEnvWithDefault("DB_NAME", "investorcenter_db"),
		SSLMode:  getEnvWithDefault("DB_SSLMODE", "require"),
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Connect establishes database connection
func Connect() (*sqlx.DB, error) {
	config := LoadConfigFromEnv()

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode,
	)

	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Successfully connected to database: %s@%s:%s/%s",
		config.User, config.Host, config.Port, config.DBName)

	return db, nil
}

// Initialize sets up the global database connection
func Initialize() error {
	db, err := Connect()
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	DB = db
	return nil
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// HealthCheck performs a simple health check on the database
func HealthCheck() error {
	if DB == nil {
		return fmt.Errorf("database connection not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return DB.PingContext(ctx)
}
