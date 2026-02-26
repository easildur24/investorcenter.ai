package config

import "os"

// Config holds all configuration for the notification service.
// Values are read from environment variables set via K8s deployment.
type Config struct {
	// Service
	Port string

	// AWS / SQS
	AWSRegion   string
	SQSQueueURL string

	// Database
	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string
	DBSSLMode  string

	// Email (SMTP)
	SMTPHost      string
	SMTPPort      string
	SMTPUsername  string
	SMTPPassword  string
	SMTPFromEmail string
	SMTPFromName  string

	// Frontend URL (for email links)
	FrontendURL string
}

// Load reads configuration from environment variables.
func Load() *Config {
	return &Config{
		Port: getEnv("PORT", "8003"),

		AWSRegion:   getEnv("AWS_REGION", "us-east-1"),
		SQSQueueURL: getEnv("SQS_QUEUE_URL", ""),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBName:     getEnv("DB_NAME", "investorcenter_db"),
		DBUser:     getEnv("DB_USER", "investorcenter"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		SMTPHost:      getEnv("SMTP_HOST", ""),
		SMTPPort:      getEnv("SMTP_PORT", "587"),
		SMTPUsername:  getEnv("SMTP_USERNAME", ""),
		SMTPPassword:  getEnv("SMTP_PASSWORD", ""),
		SMTPFromEmail: getEnv("SMTP_FROM_EMAIL", "alerts@investorcenter.ai"),
		SMTPFromName:  getEnv("SMTP_FROM_NAME", "InvestorCenter Alerts"),

		FrontendURL: getEnv("FRONTEND_URL", "https://investorcenter.ai"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
