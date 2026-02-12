package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

var (
	client *s3.Client
	bucket string
	prefix string // e.g. "worker-data/"
)

// Initialize sets up the S3 client from environment variables.
// Required: S3_BUCKET (or defaults to "investorcenter-sec-filings")
// Optional: S3_WORKER_DATA_PREFIX (defaults to "worker-data/"), AWS_REGION
func Initialize() error {
	bucket = os.Getenv("S3_BUCKET")
	if bucket == "" {
		bucket = "claw-treasure"
	}

	prefix = os.Getenv("S3_WORKER_DATA_PREFIX")
	if prefix == "" {
		prefix = "worker-data/"
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
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

	client = s3.NewFromConfig(cfg)
	log.Printf("S3 storage initialized: bucket=%s prefix=%s region=%s", bucket, prefix, region)
	return nil
}

// IsInitialized returns true if the S3 client is ready.
func IsInitialized() bool {
	return client != nil
}

// TaskDataItem represents a single data item uploaded by a worker.
type TaskDataItem struct {
	Ticker      *string         `json:"ticker"`
	ExternalID  *string         `json:"external_id"`
	CollectedAt *time.Time      `json:"collected_at"`
	Data        json.RawMessage `json:"data"`
}

// TaskDataFile represents metadata about a stored data file in S3.
type TaskDataFile struct {
	Key        string    `json:"key"`
	DataType   string    `json:"data_type"`
	ItemCount  int       `json:"item_count"`
	Size       int64     `json:"size"`
	UploadedAt time.Time `json:"uploaded_at"`
	UploadedBy string    `json:"uploaded_by,omitempty"`
}

// storedBatch is the JSON structure written to S3 for each batch.
type storedBatch struct {
	TaskID     string         `json:"task_id"`
	DataType   string         `json:"data_type"`
	UploadedBy string         `json:"uploaded_by"`
	UploadedAt time.Time      `json:"uploaded_at"`
	ItemCount  int            `json:"item_count"`
	Items      []TaskDataItem `json:"items"`
}

// buildKey creates an S3 object key for a task data batch.
// Format: worker-data/{task_id}/{data_type}/{timestamp}_{uuid}.json
func buildKey(taskID, dataType string) string {
	ts := time.Now().UTC().Format("20060102T150405Z")
	uid := uuid.New().String()[:8]
	return fmt.Sprintf("%s%s/%s/%s_%s.json", prefix, taskID, dataType, ts, uid)
}

// UploadTaskData stores a batch of data items to S3.
// Returns the S3 key and number of items stored.
func UploadTaskData(taskID, dataType, workerID string, items []TaskDataItem) (string, int, error) {
	if client == nil {
		return "", 0, fmt.Errorf("S3 client not initialized")
	}

	batch := storedBatch{
		TaskID:     taskID,
		DataType:   dataType,
		UploadedBy: workerID,
		UploadedAt: time.Now().UTC(),
		ItemCount:  len(items),
		Items:      items,
	}

	data, err := json.Marshal(batch)
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal data: %w", err)
	}

	key := buildKey(taskID, dataType)

	_, err = client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return "", 0, fmt.Errorf("failed to upload to S3: %w", err)
	}

	return key, len(items), nil
}

// ListTaskData lists all data files for a task, optionally filtered by data_type.
func ListTaskData(taskID string, dataType string, limit, offset int) ([]TaskDataFile, int, error) {
	if client == nil {
		return nil, 0, fmt.Errorf("S3 client not initialized")
	}

	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	// Build prefix for listing
	listPrefix := fmt.Sprintf("%s%s/", prefix, taskID)
	if dataType != "" {
		listPrefix = fmt.Sprintf("%s%s/%s/", prefix, taskID, dataType)
	}

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(listPrefix),
	}

	var allFiles []TaskDataFile

	paginator := s3.NewListObjectsV2Paginator(client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			return nil, 0, fmt.Errorf("failed to list S3 objects: %w", err)
		}
		for _, obj := range page.Contents {
			if obj.Key == nil || obj.Size == nil {
				continue
			}
			file := TaskDataFile{
				Key:        *obj.Key,
				Size:       *obj.Size,
				UploadedAt: *obj.LastModified,
			}
			// Extract data_type from key: prefix/task_id/data_type/filename.json
			file.DataType = extractDataType(*obj.Key, taskID)
			allFiles = append(allFiles, file)
		}
	}

	total := len(allFiles)

	// Apply offset and limit
	if offset >= total {
		return []TaskDataFile{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}

	return allFiles[offset:end], total, nil
}

// GetTaskDataFile downloads and returns the contents of a specific data file.
func GetTaskDataFile(key string) (*storedBatch, error) {
	if client == nil {
		return nil, fmt.Errorf("S3 client not initialized")
	}

	out, err := client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get S3 object: %w", err)
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read S3 object: %w", err)
	}

	var batch storedBatch
	if err := json.Unmarshal(data, &batch); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return &batch, nil
}

// extractDataType extracts the data_type component from an S3 key.
// Key format: prefix/task_id/data_type/filename.json
func extractDataType(key, taskID string) string {
	// Remove prefix and task_id portion
	after := strings.TrimPrefix(key, prefix)
	after = strings.TrimPrefix(after, taskID+"/")
	// Now after = "data_type/filename.json"
	parts := strings.SplitN(after, "/", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}
