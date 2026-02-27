package config

import (
	"fmt"
	"os"
)

// RedactedString wraps a sensitive string value to prevent accidental logging.
// Its String() method returns "[REDACTED]" instead of the actual value.
// Use .Value() to access the underlying string when needed for authentication.
type RedactedString struct {
	val string
}

// String implements fmt.Stringer — returns "[REDACTED]" to prevent accidental
// exposure in log.Printf("%+v", cfg) or similar.
func (r RedactedString) String() string {
	return "[REDACTED]"
}

// GoString implements fmt.GoStringer — returns "[REDACTED]" for %#v formatting.
func (r RedactedString) GoString() string {
	return "[REDACTED]"
}

// Value returns the actual string value. Use this only when passing
// the value to authentication functions (e.g., smtp.PlainAuth).
func (r RedactedString) Value() string {
	return r.val
}

// MarshalJSON returns "[REDACTED]" to prevent JSON serialization of the value.
func (r RedactedString) MarshalJSON() ([]byte, error) {
	return []byte(`"[REDACTED]"`), nil
}

// Config holds all configuration for the notification service.
// Values are read from environment variables set via K8s deployment.
type Config struct {
	// Service
	Port string

	// AWS / SQS
	AWSRegion   string
	SQSQueueURL string

	// SQS Consumer settings
	SQSMaxMessages int32 // Max messages per poll (1-10, default 1)

	// Database
	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string
	DBSSLMode  string

	// Email (SMTP)
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	// SMTPPassword is wrapped in RedactedString to prevent accidental logging.
	// Use cfg.SMTPPassword.Value() when passing to smtp.PlainAuth.
	SMTPPassword  RedactedString
	SMTPFromEmail string
	SMTPFromName  string

	// Frontend URL (for email links)
	FrontendURL string

	// Canary token for authenticated test endpoints
	CanaryToken string
}

// Load reads configuration from environment variables.
func Load() *Config {
	maxMessages := int32(1)
	if v := os.Getenv("SQS_MAX_MESSAGES"); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n >= 1 && n <= 10 {
			maxMessages = int32(n)
		}
	}

	return &Config{
		Port: getEnv("PORT", "8003"),

		AWSRegion:      getEnv("AWS_REGION", "us-east-1"),
		SQSQueueURL:    getEnv("SQS_QUEUE_URL", ""),
		SQSMaxMessages: maxMessages,

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBName:     getEnv("DB_NAME", "investorcenter_db"),
		DBUser:     getEnv("DB_USER", "investorcenter"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		// Default to "require" for encrypted connections in production.
		// Override to "disable" for local development via DB_SSLMODE env var.
		DBSSLMode: getEnv("DB_SSLMODE", "require"),

		SMTPHost:      getEnv("SMTP_HOST", ""),
		SMTPPort:      getEnv("SMTP_PORT", "587"),
		SMTPUsername:  getEnv("SMTP_USERNAME", ""),
		SMTPPassword:  RedactedString{val: getEnv("SMTP_PASSWORD", "")},
		SMTPFromEmail: getEnv("SMTP_FROM_EMAIL", "alerts@investorcenter.ai"),
		SMTPFromName:  getEnv("SMTP_FROM_NAME", "InvestorCenter Alerts"),

		FrontendURL: getEnv("FRONTEND_URL", "https://investorcenter.ai"),

		CanaryToken: getEnv("CANARY_TOKEN", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
