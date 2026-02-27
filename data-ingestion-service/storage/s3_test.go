package storage

import (
	"strings"
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

	t.Run("key always starts with raw/ prefix", func(t *testing.T) {
		key := GenerateKey("anything", "XYZ", "type", fixedTime)
		assert.True(t, strings.HasPrefix(key, "raw/"))
	})

	t.Run("key always ends with .json extension", func(t *testing.T) {
		key := GenerateKey("src", "TICK", "dt", fixedTime)
		assert.True(t, strings.HasSuffix(key, ".json"))
	})

	t.Run("handles special characters in source", func(t *testing.T) {
		key := GenerateKey("my-source_v2", "AAPL", "data", fixedTime)
		assert.Equal(t, "raw/my-source_v2/AAPL/data/2026-02-13/20260213T103045Z.json", key)
	})

	t.Run("handles single character ticker", func(t *testing.T) {
		key := GenerateKey("test", "X", "price", fixedTime)
		assert.Equal(t, "raw/test/X/price/2026-02-13/20260213T103045Z.json", key)
	})

	t.Run("date changes when time crosses midnight UTC", func(t *testing.T) {
		// 11:59 PM UTC on Feb 13
		beforeMidnight := time.Date(2026, 2, 13, 23, 59, 59, 0, time.UTC)
		key1 := GenerateKey("test", "AAPL", "data", beforeMidnight)
		assert.Contains(t, key1, "2026-02-13")

		// 12:00 AM UTC on Feb 14
		afterMidnight := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
		key2 := GenerateKey("test", "AAPL", "data", afterMidnight)
		assert.Contains(t, key2, "2026-02-14")
	})

	t.Run("non-UTC timezone crossing date boundary", func(t *testing.T) {
		// 11 PM EST = 4 AM next day UTC
		est, _ := time.LoadLocation("America/New_York")
		lateEST := time.Date(2026, 2, 13, 23, 0, 0, 0, est)

		key := GenerateKey("test", "AAPL", "data", lateEST)
		// Should use UTC date (Feb 14), not local date (Feb 13)
		assert.Contains(t, key, "2026-02-14")
	})

	t.Run("generates consistent keys for same inputs", func(t *testing.T) {
		key1 := GenerateKey("src", "TICK", "type", fixedTime)
		key2 := GenerateKey("src", "TICK", "type", fixedTime)
		assert.Equal(t, key1, key2)
	})

	t.Run("generates different keys for different tickers", func(t *testing.T) {
		key1 := GenerateKey("src", "AAPL", "type", fixedTime)
		key2 := GenerateKey("src", "GOOG", "type", fixedTime)
		assert.NotEqual(t, key1, key2)
	})

	t.Run("generates different keys for different timestamps", func(t *testing.T) {
		t1 := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)
		t2 := time.Date(2026, 2, 13, 10, 0, 1, 0, time.UTC)
		key1 := GenerateKey("src", "TICK", "type", t1)
		key2 := GenerateKey("src", "TICK", "type", t2)
		assert.NotEqual(t, key1, key2)
	})

	t.Run("key has exactly 5 path segments", func(t *testing.T) {
		key := GenerateKey("src", "TICK", "type", fixedTime)
		// raw/{source}/{ticker}/{data_type}/{date}/{timestamp}.json
		parts := strings.Split(key, "/")
		assert.Equal(t, 6, len(parts))
		assert.Equal(t, "raw", parts[0])
	})
}

func TestGetBucket(t *testing.T) {
	t.Run("returns bucket name", func(t *testing.T) {
		// GetBucket returns the package-level bucket variable
		name := GetBucket()
		// It may be empty in test context or set to default
		assert.IsType(t, "", name)
	})

	t.Run("returns string type", func(t *testing.T) {
		name := GetBucket()
		assert.IsType(t, "", name)
	})

	t.Run("bucket value is settable and retrievable", func(t *testing.T) {
		originalBucket := bucket
		defer func() { bucket = originalBucket }()

		bucket = "test-bucket-name"
		assert.Equal(t, "test-bucket-name", GetBucket())
	})

	t.Run("returns empty string when not initialized", func(t *testing.T) {
		originalBucket := bucket
		defer func() { bucket = originalBucket }()

		bucket = ""
		assert.Equal(t, "", GetBucket())
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

	t.Run("returns error with empty key when client nil", func(t *testing.T) {
		err := Upload("", []byte("data"), "application/json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "S3 client not initialized")
	})

	t.Run("returns error with empty data when client nil", func(t *testing.T) {
		err := Upload("key", []byte{}, "application/json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "S3 client not initialized")
	})

	t.Run("returns error with nil data when client nil", func(t *testing.T) {
		err := Upload("key", nil, "application/json")
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
