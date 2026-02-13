package database

import (
	"fmt"
	"time"
)

// IngestionLog represents a record in the ingestion_log table
type IngestionLog struct {
	ID          int64      `json:"id" db:"id"`
	Source      string     `json:"source" db:"source"`
	Ticker      *string    `json:"ticker" db:"ticker"`
	DataType    string     `json:"data_type" db:"data_type"`
	SourceURL   *string    `json:"source_url" db:"source_url"`
	S3Key       string     `json:"s3_key" db:"s3_key"`
	S3Bucket    string     `json:"s3_bucket" db:"s3_bucket"`
	FileSize    int64      `json:"file_size" db:"file_size"`
	CollectedAt time.Time  `json:"collected_at" db:"collected_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// InsertIngestionLog inserts a new record and returns the ID
func InsertIngestionLog(source string, ticker *string, dataType string, sourceURL *string, s3Key string, s3Bucket string, fileSize int64, collectedAt time.Time) (int64, error) {
	var id int64
	err := DB.QueryRow(
		`INSERT INTO ingestion_log (source, ticker, data_type, source_url, s3_key, s3_bucket, file_size, collected_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id`,
		source, ticker, dataType, sourceURL, s3Key, s3Bucket, fileSize, collectedAt,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to insert ingestion log: %w", err)
	}
	return id, nil
}

// GetIngestionLogs retrieves ingestion log records with optional filtering
func GetIngestionLogs(source, ticker, dataType string, limit, offset int) ([]IngestionLog, int, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Build WHERE clause dynamically
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if source != "" {
		where += fmt.Sprintf(" AND source = $%d", argIdx)
		args = append(args, source)
		argIdx++
	}
	if ticker != "" {
		where += fmt.Sprintf(" AND ticker = $%d", argIdx)
		args = append(args, ticker)
		argIdx++
	}
	if dataType != "" {
		where += fmt.Sprintf(" AND data_type = $%d", argIdx)
		args = append(args, dataType)
		argIdx++
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM ingestion_log " + where
	err := DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count ingestion logs: %w", err)
	}

	// Get records
	query := fmt.Sprintf(
		"SELECT id, source, ticker, data_type, source_url, s3_key, s3_bucket, file_size, collected_at, created_at FROM ingestion_log %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		where, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows := []IngestionLog{}
	err = DB.Select(&rows, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query ingestion logs: %w", err)
	}

	return rows, total, nil
}

// GetIngestionLog retrieves a single ingestion log by ID
func GetIngestionLog(id int64) (*IngestionLog, error) {
	var log IngestionLog
	err := DB.Get(&log,
		`SELECT id, source, ticker, data_type, source_url, s3_key, s3_bucket, file_size, collected_at, created_at
		 FROM ingestion_log WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("ingestion log not found: %w", err)
	}
	return &log, nil
}
