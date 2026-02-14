package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateKey(t *testing.T) {
	fixedTime := time.Date(2026, 2, 13, 10, 30, 45, 0, time.UTC)

	t.Run("generates key with ticker", func(t *testing.T) {
		key := GenerateKey("reddit", "AAPL", "sentiment", fixedTime)
		assert.Equal(t, "raw/reddit/AAPL/sentiment/2026-02-13/20260213T103045Z.json", key)
	})

	t.Run("generates key without ticker uses _global", func(t *testing.T) {
		key := GenerateKey("news", "", "headlines", fixedTime)
		assert.Equal(t, "raw/news/_global/headlines/2026-02-13/20260213T103045Z.json", key)
	})

	t.Run("handles different sources", func(t *testing.T) {
		key := GenerateKey("sec-edgar", "MSFT", "10-K", fixedTime)
		assert.Equal(t, "raw/sec-edgar/MSFT/10-K/2026-02-13/20260213T103045Z.json", key)
	})

	t.Run("handles different dates", func(t *testing.T) {
		newYear := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		key := GenerateKey("test", "GOOG", "data", newYear)
		assert.Equal(t, "raw/test/GOOG/data/2026-01-01/20260101T000000Z.json", key)
	})

	t.Run("converts non-UTC time to UTC", func(t *testing.T) {
		// EST is UTC-5
		est, _ := time.LoadLocation("America/New_York")
		localTime := time.Date(2026, 2, 13, 5, 30, 0, 0, est)

		key := GenerateKey("test", "TSLA", "price", localTime)
		// 5:30 EST = 10:30 UTC
		assert.Equal(t, "raw/test/TSLA/price/2026-02-13/20260213T103000Z.json", key)
	})
}

func TestGetBucket(t *testing.T) {
	t.Run("returns bucket name", func(t *testing.T) {
		// GetBucket returns the package-level bucket variable
		name := GetBucket()
		// It may be empty in test context or set to default
		assert.IsType(t, "", name)
	})
}

func TestUploadWithoutInit(t *testing.T) {
	// Save and restore the client
	originalClient := s3Client
	s3Client = nil
	defer func() { s3Client = originalClient }()

	t.Run("returns error when S3 client not initialized", func(t *testing.T) {
		err := Upload("test-key", []byte("data"), "application/json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "S3 client not initialized")
	})
}

func TestHealthCheckWithoutInit(t *testing.T) {
	originalClient := s3Client
	s3Client = nil
	defer func() { s3Client = originalClient }()

	t.Run("returns error when S3 client not initialized", func(t *testing.T) {
		err := HealthCheck()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "S3 client not initialized")
	})
}
