package storage

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	s3Client   *s3.S3
	bucketName string
)

const defaultBucket = "claw-treasure"
const keyPrefix = "worker-results/"

// Initialize sets up the S3 client for read-only access (admin downloads).
func Initialize() error {
	bucketName = os.Getenv("S3_BUCKET")
	if bucketName == "" {
		bucketName = defaultBucket
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return fmt.Errorf("failed to create AWS session: %w", err)
	}

	s3Client = s3.New(sess)
	log.Printf("S3 storage initialized: bucket=%s, region=%s", bucketName, region)
	return nil
}

// ValidateS3Key ensures the key starts with the expected prefix for the given task.
func ValidateS3Key(s3Key string, taskID string) error {
	expectedPrefix := keyPrefix + taskID + "/"
	if !strings.HasPrefix(s3Key, expectedPrefix) {
		return fmt.Errorf("invalid s3_key: must start with %s", expectedPrefix)
	}
	return nil
}

// DownloadFile streams a file from S3. Caller must close the returned ReadCloser.
func DownloadFile(s3Key string) (io.ReadCloser, string, int64, error) {
	if s3Client == nil {
		return nil, "", 0, fmt.Errorf("S3 client not initialized")
	}

	resp, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to download from S3: %w", err)
	}

	contentType := "application/octet-stream"
	if resp.ContentType != nil {
		contentType = *resp.ContentType
	}

	var size int64
	if resp.ContentLength != nil {
		size = *resp.ContentLength
	}

	return resp.Body, contentType, size, nil
}
