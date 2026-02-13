package database

import (
	"fmt"
	"time"
)

// TaskFileRow represents a row from worker_task_files
type TaskFileRow struct {
	ID          int64     `json:"id"`
	TaskID      string    `json:"task_id"`
	Filename    string    `json:"filename"`
	S3Key       string    `json:"s3_key"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	UploadedBy  *string   `json:"uploaded_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// InsertTaskFile registers a file that was uploaded to S3.
func InsertTaskFile(taskID, filename, s3Key, contentType string, sizeBytes int64, uploadedBy string) (*TaskFileRow, error) {
	var row TaskFileRow
	err := DB.QueryRow(
		`INSERT INTO worker_task_files (task_id, filename, s3_key, content_type, size_bytes, uploaded_by)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, task_id, filename, s3_key, content_type, size_bytes, uploaded_by, created_at`,
		taskID, filename, s3Key, contentType, sizeBytes, uploadedBy,
	).Scan(&row.ID, &row.TaskID, &row.Filename, &row.S3Key, &row.ContentType, &row.SizeBytes, &row.UploadedBy, &row.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert task file: %w", err)
	}
	return &row, nil
}

// ListTaskFiles returns paginated files for a task.
func ListTaskFiles(taskID string, limit, offset int) ([]TaskFileRow, int, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	var total int
	err := DB.QueryRow("SELECT COUNT(*) FROM worker_task_files WHERE task_id = $1", taskID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count task files: %w", err)
	}

	rows, err := DB.Query(
		`SELECT id, task_id, filename, s3_key, content_type, size_bytes, uploaded_by, created_at
		 FROM worker_task_files
		 WHERE task_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		taskID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query task files: %w", err)
	}
	defer rows.Close()

	var results []TaskFileRow
	for rows.Next() {
		var r TaskFileRow
		err := rows.Scan(&r.ID, &r.TaskID, &r.Filename, &r.S3Key, &r.ContentType, &r.SizeBytes, &r.UploadedBy, &r.CreatedAt)
		if err != nil {
			continue
		}
		results = append(results, r)
	}

	if results == nil {
		results = []TaskFileRow{}
	}

	return results, total, nil
}

// GetTaskFile retrieves a single file record by task_id and file id.
func GetTaskFile(taskID string, fileID int64) (*TaskFileRow, error) {
	var row TaskFileRow
	err := DB.QueryRow(
		`SELECT id, task_id, filename, s3_key, content_type, size_bytes, uploaded_by, created_at
		 FROM worker_task_files
		 WHERE task_id = $1 AND id = $2`,
		taskID, fileID,
	).Scan(&row.ID, &row.TaskID, &row.Filename, &row.S3Key, &row.ContentType, &row.SizeBytes, &row.UploadedBy, &row.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get task file: %w", err)
	}
	return &row, nil
}
