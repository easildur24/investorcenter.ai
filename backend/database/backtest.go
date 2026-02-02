package database

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"investorcenter/models"
)

// BacktestRepository handles backtest-related database operations
type BacktestRepository struct {
	db *gorm.DB
}

// NewBacktestRepository creates a new backtest repository
func NewBacktestRepository(db *gorm.DB) *BacktestRepository {
	return &BacktestRepository{db: db}
}

// CreateJob creates a new backtest job
func (r *BacktestRepository) CreateJob(config models.BacktestConfig, userID *string) (*models.BacktestJob, error) {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	job := &models.BacktestJob{
		ID:        uuid.New().String(),
		UserID:    userID,
		Config:    string(configJSON),
		Status:    models.BacktestStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := r.db.Create(job).Error; err != nil {
		return nil, err
	}

	return job, nil
}

// GetJob retrieves a backtest job by ID
func (r *BacktestRepository) GetJob(id string) (*models.BacktestJob, error) {
	var job models.BacktestJob
	if err := r.db.First(&job, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// UpdateJobStatus updates the status of a backtest job
func (r *BacktestRepository) UpdateJobStatus(id string, status models.BacktestStatus) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if status == models.BacktestStatusRunning {
		now := time.Now()
		updates["started_at"] = &now
	}

	return r.db.Model(&models.BacktestJob{}).Where("id = ?", id).Updates(updates).Error
}

// CompleteJob marks a job as completed with results
func (r *BacktestRepository) CompleteJob(id string, summary *models.BacktestSummary) error {
	resultJSON, err := json.Marshal(summary)
	if err != nil {
		return err
	}

	result := string(resultJSON)
	now := time.Now()

	return r.db.Model(&models.BacktestJob{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       models.BacktestStatusCompleted,
		"result":       &result,
		"completed_at": &now,
		"updated_at":   now,
	}).Error
}

// FailJob marks a job as failed with an error message
func (r *BacktestRepository) FailJob(id string, errMsg string) error {
	now := time.Now()

	return r.db.Model(&models.BacktestJob{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       models.BacktestStatusFailed,
		"error":        &errMsg,
		"completed_at": &now,
		"updated_at":   now,
	}).Error
}

// GetUserJobs retrieves all backtest jobs for a user
func (r *BacktestRepository) GetUserJobs(userID string, limit int) ([]models.BacktestJob, error) {
	var jobs []models.BacktestJob
	query := r.db.Where("user_id = ?", userID).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&jobs).Error; err != nil {
		return nil, err
	}

	return jobs, nil
}

// GetRecentCompletedBacktest retrieves the most recent completed backtest
func (r *BacktestRepository) GetRecentCompletedBacktest() (*models.BacktestJob, error) {
	var job models.BacktestJob
	if err := r.db.Where("status = ?", models.BacktestStatusCompleted).
		Order("completed_at DESC").
		First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// GetCachedBacktestResult retrieves cached backtest results matching config
func (r *BacktestRepository) GetCachedBacktestResult(config models.BacktestConfig, maxAge time.Duration) (*models.BacktestJob, error) {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	minTime := time.Now().Add(-maxAge)

	var job models.BacktestJob
	if err := r.db.Where("config = ? AND status = ? AND completed_at > ?",
		string(configJSON), models.BacktestStatusCompleted, minTime).
		Order("completed_at DESC").
		First(&job).Error; err != nil {
		return nil, err
	}

	return &job, nil
}

// DeleteOldJobs removes backtest jobs older than the specified duration
func (r *BacktestRepository) DeleteOldJobs(maxAge time.Duration) (int64, error) {
	cutoff := time.Now().Add(-maxAge)
	result := r.db.Where("created_at < ?", cutoff).Delete(&models.BacktestJob{})
	return result.RowsAffected, result.Error
}

// GetJobStats returns statistics about backtest jobs
func (r *BacktestRepository) GetJobStats() (map[string]int64, error) {
	stats := make(map[string]int64)

	// Count by status
	statuses := []models.BacktestStatus{
		models.BacktestStatusPending,
		models.BacktestStatusRunning,
		models.BacktestStatusCompleted,
		models.BacktestStatusFailed,
	}

	for _, status := range statuses {
		var count int64
		if err := r.db.Model(&models.BacktestJob{}).Where("status = ?", status).Count(&count).Error; err != nil {
			return nil, err
		}
		stats[string(status)] = count
	}

	// Total count
	var total int64
	if err := r.db.Model(&models.BacktestJob{}).Count(&total).Error; err != nil {
		return nil, err
	}
	stats["total"] = total

	return stats, nil
}
