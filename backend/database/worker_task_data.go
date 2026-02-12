package database

import (
	"encoding/json"
	"fmt"
	"time"
)

// TaskDataItem represents a single data item to insert
type TaskDataItem struct {
	Ticker      *string         `json:"ticker"`
	ExternalID  *string         `json:"external_id"`
	CollectedAt *time.Time      `json:"collected_at"`
	Data        json.RawMessage `json:"data"`
}

// TaskDataRow represents a row from worker_task_data
type TaskDataRow struct {
	ID          int64           `json:"id"`
	TaskID      string          `json:"task_id"`
	DataType    string          `json:"data_type"`
	Ticker      *string         `json:"ticker"`
	ExternalID  *string         `json:"external_id"`
	Data        json.RawMessage `json:"data"`
	CollectedAt time.Time       `json:"collected_at"`
	CreatedAt   time.Time       `json:"created_at"`
}

// BulkInsertTaskData inserts multiple data items for a worker task.
// Returns (inserted, skipped, error). Duplicates (by data_type+external_id) are silently skipped.
func BulkInsertTaskData(taskID string, dataType string, items []TaskDataItem) (int, int, error) {
	if len(items) == 0 {
		return 0, 0, nil
	}

	tx, err := DB.Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO worker_task_data (task_id, data_type, ticker, external_id, data, collected_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (data_type, external_id) WHERE external_id IS NOT NULL
		DO NOTHING
	`)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, item := range items {
		collectedAt := time.Now()
		if item.CollectedAt != nil {
			collectedAt = *item.CollectedAt
		}

		result, err := stmt.Exec(taskID, dataType, item.Ticker, item.ExternalID, item.Data, collectedAt)
		if err != nil {
			continue
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			inserted++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	skipped := len(items) - inserted
	return inserted, skipped, nil
}

// GetTaskData retrieves collected data for a task with optional filters and pagination.
func GetTaskData(taskID string, dataType string, ticker string, limit, offset int) ([]TaskDataRow, int, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	// Build query with optional filters
	where := "WHERE task_id = $1"
	args := []interface{}{taskID}
	argIdx := 2

	if dataType != "" {
		where += fmt.Sprintf(" AND data_type = $%d", argIdx)
		args = append(args, dataType)
		argIdx++
	}
	if ticker != "" {
		where += fmt.Sprintf(" AND ticker = $%d", argIdx)
		args = append(args, ticker)
		argIdx++
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM worker_task_data " + where
	err := DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count task data: %w", err)
	}

	// Get rows
	query := fmt.Sprintf(`
		SELECT id, task_id, data_type, ticker, external_id, data, collected_at, created_at
		FROM worker_task_data
		%s
		ORDER BY collected_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query task data: %w", err)
	}
	defer rows.Close()

	var results []TaskDataRow
	for rows.Next() {
		var r TaskDataRow
		err := rows.Scan(&r.ID, &r.TaskID, &r.DataType, &r.Ticker, &r.ExternalID,
			&r.Data, &r.CollectedAt, &r.CreatedAt)
		if err != nil {
			continue
		}
		results = append(results, r)
	}

	if results == nil {
		results = []TaskDataRow{}
	}

	return results, total, nil
}
