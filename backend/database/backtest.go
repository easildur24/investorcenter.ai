package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"investorcenter-api/models"
)

// CreateBacktestJob creates a new backtest job
func CreateBacktestJob(config models.BacktestConfig, userID *string) (*models.BacktestJob, error) {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	job := &models.BacktestJob{
		ID:        uuid.New().String(),
		UserID:    userID,
		Config:    string(configJSON),
		Status:    models.BacktestStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	query := `
		INSERT INTO backtest_jobs (id, user_id, config, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = DB.Exec(query, job.ID, job.UserID, job.Config, job.Status, job.CreatedAt, job.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create backtest job: %w", err)
	}

	return job, nil
}

// GetBacktestJob retrieves a backtest job by ID
func GetBacktestJob(id string) (*models.BacktestJob, error) {
	query := `
		SELECT id, user_id, config, status, error, result, started_at, completed_at, created_at, updated_at
		FROM backtest_jobs
		WHERE id = $1
	`
	job := &models.BacktestJob{}
	err := DB.QueryRow(query, id).Scan(
		&job.ID,
		&job.UserID,
		&job.Config,
		&job.Status,
		&job.Error,
		&job.Result,
		&job.StartedAt,
		&job.CompletedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("backtest job not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get backtest job: %w", err)
	}
	return job, nil
}

// UpdateBacktestJobStatus updates the status of a backtest job
func UpdateBacktestJobStatus(id string, status models.BacktestStatus) error {
	now := time.Now()
	var query string
	var err error

	if status == models.BacktestStatusRunning {
		query = `
			UPDATE backtest_jobs
			SET status = $1, started_at = $2, updated_at = $3
			WHERE id = $4
		`
		_, err = DB.Exec(query, status, now, now, id)
	} else {
		query = `
			UPDATE backtest_jobs
			SET status = $1, updated_at = $2
			WHERE id = $3
		`
		_, err = DB.Exec(query, status, now, id)
	}

	if err != nil {
		return fmt.Errorf("failed to update backtest job status: %w", err)
	}
	return nil
}

// CompleteBacktestJob marks a job as completed with results
func CompleteBacktestJob(id string, summary *models.BacktestSummary) error {
	resultJSON, err := json.Marshal(summary)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	now := time.Now()
	result := string(resultJSON)

	query := `
		UPDATE backtest_jobs
		SET status = $1, result = $2, completed_at = $3, updated_at = $4
		WHERE id = $5
	`
	_, err = DB.Exec(query, models.BacktestStatusCompleted, result, now, now, id)
	if err != nil {
		return fmt.Errorf("failed to complete backtest job: %w", err)
	}
	return nil
}

// FailBacktestJob marks a job as failed with an error message
func FailBacktestJob(id string, errMsg string) error {
	now := time.Now()

	query := `
		UPDATE backtest_jobs
		SET status = $1, error = $2, completed_at = $3, updated_at = $4
		WHERE id = $5
	`
	_, err := DB.Exec(query, models.BacktestStatusFailed, errMsg, now, now, id)
	if err != nil {
		return fmt.Errorf("failed to fail backtest job: %w", err)
	}
	return nil
}

// GetUserBacktestJobs retrieves all backtest jobs for a user
func GetUserBacktestJobs(userID string, limit int) ([]models.BacktestJob, error) {
	query := `
		SELECT id, user_id, config, status, error, result, started_at, completed_at, created_at, updated_at
		FROM backtest_jobs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := DB.Query(query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get user backtest jobs: %w", err)
	}
	defer rows.Close()

	var jobs []models.BacktestJob
	for rows.Next() {
		var job models.BacktestJob
		err := rows.Scan(
			&job.ID,
			&job.UserID,
			&job.Config,
			&job.Status,
			&job.Error,
			&job.Result,
			&job.StartedAt,
			&job.CompletedAt,
			&job.CreatedAt,
			&job.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan backtest job: %w", err)
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// GetRecentCompletedBacktest retrieves the most recent completed backtest
func GetRecentCompletedBacktest() (*models.BacktestJob, error) {
	query := `
		SELECT id, user_id, config, status, error, result, started_at, completed_at, created_at, updated_at
		FROM backtest_jobs
		WHERE status = $1
		ORDER BY completed_at DESC
		LIMIT 1
	`
	job := &models.BacktestJob{}
	err := DB.QueryRow(query, models.BacktestStatusCompleted).Scan(
		&job.ID,
		&job.UserID,
		&job.Config,
		&job.Status,
		&job.Error,
		&job.Result,
		&job.StartedAt,
		&job.CompletedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no completed backtests found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get recent completed backtest: %w", err)
	}
	return job, nil
}

// GetCachedBacktestResult retrieves cached backtest results matching config
func GetCachedBacktestResult(config models.BacktestConfig, maxAge time.Duration) (*models.BacktestJob, error) {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	minTime := time.Now().Add(-maxAge)

	query := `
		SELECT id, user_id, config, status, error, result, started_at, completed_at, created_at, updated_at
		FROM backtest_jobs
		WHERE config = $1 AND status = $2 AND completed_at > $3
		ORDER BY completed_at DESC
		LIMIT 1
	`
	job := &models.BacktestJob{}
	err = DB.QueryRow(query, string(configJSON), models.BacktestStatusCompleted, minTime).Scan(
		&job.ID,
		&job.UserID,
		&job.Config,
		&job.Status,
		&job.Error,
		&job.Result,
		&job.StartedAt,
		&job.CompletedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no cached backtest result found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cached backtest result: %w", err)
	}
	return job, nil
}

// DeleteOldBacktestJobs removes backtest jobs older than the specified duration
func DeleteOldBacktestJobs(maxAge time.Duration) (int64, error) {
	cutoff := time.Now().Add(-maxAge)

	query := `DELETE FROM backtest_jobs WHERE created_at < $1`
	result, err := DB.Exec(query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old backtest jobs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return rowsAffected, nil
}

// GetBacktestJobStats returns statistics about backtest jobs
func GetBacktestJobStats() (map[string]int64, error) {
	stats := make(map[string]int64)

	statuses := []models.BacktestStatus{
		models.BacktestStatusPending,
		models.BacktestStatusRunning,
		models.BacktestStatusCompleted,
		models.BacktestStatusFailed,
	}

	for _, status := range statuses {
		var count int64
		query := `SELECT COUNT(*) FROM backtest_jobs WHERE status = $1`
		err := DB.QueryRow(query, status).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("failed to count backtest jobs: %w", err)
		}
		stats[string(status)] = count
	}

	var total int64
	query := `SELECT COUNT(*) FROM backtest_jobs`
	err := DB.QueryRow(query).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count total backtest jobs: %w", err)
	}
	stats["total"] = total

	return stats, nil
}
