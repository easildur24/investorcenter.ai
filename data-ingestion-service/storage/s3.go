package storage

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	s3Client *s3.Client
	bucket   string
)

// Initialize sets up the S3 client using IRSA credentials (or default chain)
func Initialize() error {
	bucket = os.Getenv("S3_BUCKET")
	if bucket == "" {
		bucket = "investorcenter-raw-data"
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	s3Client = s3.NewFromConfig(cfg)
	log.Printf("S3 client initialized: bucket=%s, region=%s", bucket, region)
	return nil
}

// Upload uploads raw data to S3 and returns the key
func Upload(key string, data []byte, contentType string) error {
	if s3Client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &bucket,
		Key:         &key,
		Body:        bytes.NewReader(data),
		ContentType: &contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

// GenerateKey creates an S3 key from the ingestion metadata
// Format: raw/{source}/{ticker}/{data_type}/{YYYY-MM-DD}/{timestamp}.json
func GenerateKey(source, ticker, dataType string, collectedAt time.Time) string {
	datePart := collectedAt.UTC().Format("2006-01-02")
	timestampPart := collectedAt.UTC().Format("20060102T150405Z")

	if ticker == "" {
		ticker = "_global"
	}

	return fmt.Sprintf("raw/%s/%s/%s/%s/%s.json", source, ticker, dataType, datePart, timestampPart)
}

// GetBucket returns the configured bucket name
func GetBucket() string {
	return bucket
}

// HealthCheck verifies S3 connectivity by listing bucket (head bucket)
func HealthCheck() error {
	if s3Client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: &bucket,
	})
	return err
}
