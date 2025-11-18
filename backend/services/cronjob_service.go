package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"investorcenter-api/database"
	"investorcenter-api/models"
	"time"
)

type CronjobService struct{}

func NewCronjobService() *CronjobService {
	return &CronjobService{}
}

// GetOverview returns summary and status of all cronjobs
func (s *CronjobService) GetOverview() (*models.CronjobOverviewResponse, error) {
	if database.DB == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// Get summary stats for last 24h
	var summary models.CronjobSummary
	err := database.DB.QueryRow(`
		SELECT
			COUNT(DISTINCT cs.job_name) as total_jobs,
			COUNT(DISTINCT cs.job_name) FILTER (WHERE cs.is_active = true) as active_jobs,
			COUNT(cel.id) FILTER (WHERE cel.started_at >= NOW() - INTERVAL '24 hours') as total_executions,
			COUNT(cel.id) FILTER (WHERE cel.started_at >= NOW() - INTERVAL '24 hours' AND cel.status = 'success') as successful,
			COUNT(cel.id) FILTER (WHERE cel.started_at >= NOW() - INTERVAL '24 hours' AND cel.status IN ('failed', 'timeout')) as failed
		FROM cronjob_schedules cs
		LEFT JOIN cronjob_execution_logs cel ON cs.job_name = cel.job_name
	`).Scan(&summary.TotalJobs, &summary.ActiveJobs, &summary.Last24h.TotalExecutions,
		&summary.Last24h.Successful, &summary.Last24h.Failed)

	if err != nil {
		return nil, err
	}

	// Calculate success rate
	if summary.Last24h.TotalExecutions > 0 {
		summary.Last24h.SuccessRate = float64(summary.Last24h.Successful) / float64(summary.Last24h.TotalExecutions) * 100
	}

	// Get all jobs with their status
	rows, err := database.DB.Query(`
		WITH latest_execution AS (
			SELECT DISTINCT ON (job_name)
				job_name,
				status,
				started_at,
				completed_at,
				duration_seconds,
				records_processed
			FROM cronjob_execution_logs
			ORDER BY job_name, started_at DESC
		),
		stats_7d AS (
			SELECT
				job_name,
				AVG(duration_seconds) FILTER (WHERE status = 'success') as avg_duration,
				COUNT(*) FILTER (WHERE status = 'success')::float / NULLIF(COUNT(*), 0) * 100 as success_rate
			FROM cronjob_execution_logs
			WHERE started_at >= NOW() - INTERVAL '7 days'
			GROUP BY job_name
		)
		SELECT
			cs.job_name,
			cs.job_category,
			cs.schedule_cron,
			cs.schedule_description,
			le.status,
			le.started_at,
			le.completed_at,
			le.duration_seconds,
			le.records_processed,
			cs.consecutive_failures,
			s.avg_duration,
			s.success_rate
		FROM cronjob_schedules cs
		LEFT JOIN latest_execution le ON cs.job_name = le.job_name
		LEFT JOIN stats_7d s ON cs.job_name = s.job_name
		WHERE cs.is_active = true
		ORDER BY cs.job_category, cs.job_name
	`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []models.CronjobStatusWithInfo
	for rows.Next() {
		var job models.CronjobStatusWithInfo
		var lastRun models.LastRunInfo
		var hasLastRun bool

		var status sql.NullString
		var startedAt sql.NullTime
		var completedAt sql.NullTime
		var durationSeconds sql.NullInt64
		var recordsProcessed sql.NullInt64

		err := rows.Scan(
			&job.JobName,
			&job.JobCategory,
			&job.Schedule,
			&job.ScheduleDescription,
			&status,
			&startedAt,
			&completedAt,
			&durationSeconds,
			&recordsProcessed,
			&job.ConsecutiveFailures,
			&job.AvgDuration7d,
			&job.SuccessRate7d,
		)

		if err != nil {
			return nil, err
		}

		// Build last run info if exists
		if status.Valid && startedAt.Valid {
			hasLastRun = true
			lastRun.Status = status.String
			lastRun.StartedAt = startedAt.Time

			if completedAt.Valid {
				lastRun.CompletedAt = &completedAt.Time
			}
			if durationSeconds.Valid {
				duration := int(durationSeconds.Int64)
				lastRun.DurationSeconds = &duration
			}
			if recordsProcessed.Valid {
				lastRun.RecordsProcessed = int(recordsProcessed.Int64)
			}
		}

		if hasLastRun {
			job.LastRun = &lastRun
		}

		// Determine health status
		job.HealthStatus = s.calculateHealthStatus(&job)

		jobs = append(jobs, job)
	}

	return &models.CronjobOverviewResponse{
		Summary: summary,
		Jobs:    jobs,
	}, nil
}

// calculateHealthStatus determines health status based on job info
func (s *CronjobService) calculateHealthStatus(job *models.CronjobStatusWithInfo) string {
	// Critical: 3+ consecutive failures
	if job.ConsecutiveFailures >= 3 {
		return "critical"
	}

	// Critical: last run failed
	if job.LastRun != nil && (job.LastRun.Status == "failed" || job.LastRun.Status == "timeout") {
		return "critical"
	}

	// Warning: 1-2 consecutive failures
	if job.ConsecutiveFailures > 0 {
		return "warning"
	}

	// Warning: currently running
	if job.LastRun != nil && job.LastRun.Status == "running" {
		return "warning"
	}

	// Healthy
	if job.LastRun != nil && job.LastRun.Status == "success" {
		return "healthy"
	}

	// Unknown: no executions
	return "unknown"
}

// GetJobHistory returns execution history for a specific job
func (s *CronjobService) GetJobHistory(jobName string, limit, offset int) (*models.CronjobHistoryResponse, error) {
	if database.DB == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// Get total count
	var totalExecutions int64
	err := database.DB.QueryRow(`
		SELECT COUNT(*) FROM cronjob_execution_logs WHERE job_name = $1
	`, jobName).Scan(&totalExecutions)

	if err != nil {
		return nil, err
	}

	// Get execution logs
	rows, err := database.DB.Query(`
		SELECT
			id, job_name, job_category, execution_id, status,
			started_at, completed_at, duration_seconds,
			records_processed, records_updated, records_failed,
			error_message, error_stack_trace,
			k8s_pod_name, k8s_namespace, exit_code, created_at
		FROM cronjob_execution_logs
		WHERE job_name = $1
		ORDER BY started_at DESC
		LIMIT $2 OFFSET $3
	`, jobName, limit, offset)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []models.CronjobExecutionLog
	for rows.Next() {
		var log models.CronjobExecutionLog
		err := rows.Scan(
			&log.ID, &log.JobName, &log.JobCategory, &log.ExecutionID, &log.Status,
			&log.StartedAt, &log.CompletedAt, &log.DurationSeconds,
			&log.RecordsProcessed, &log.RecordsUpdated, &log.RecordsFailed,
			&log.ErrorMessage, &log.ErrorStackTrace,
			&log.K8sPodName, &log.K8sNamespace, &log.ExitCode, &log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		executions = append(executions, log)
	}

	// Get metrics
	var metrics models.CronjobMetrics
	err = database.DB.QueryRow(`
		SELECT
			AVG(duration_seconds) FILTER (WHERE status = 'success') as avg_duration,
			PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY duration_seconds) FILTER (WHERE status = 'success') as p50_duration,
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_seconds) FILTER (WHERE status = 'success') as p95_duration,
			COUNT(*) FILTER (WHERE status = 'success')::float / NULLIF(COUNT(*), 0) * 100 as success_rate
		FROM cronjob_execution_logs
		WHERE job_name = $1 AND started_at >= NOW() - INTERVAL '30 days'
	`, jobName).Scan(&metrics.AvgDuration, &metrics.P50Duration, &metrics.P95Duration, &metrics.SuccessRate)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return &models.CronjobHistoryResponse{
		JobName:         jobName,
		TotalExecutions: totalExecutions,
		Executions:      executions,
		Metrics:         metrics,
	}, nil
}

// GetJobDetails returns details of a specific execution
func (s *CronjobService) GetJobDetails(executionID string) (*models.CronjobExecutionLog, error) {
	if database.DB == nil {
		return nil, fmt.Errorf("database not connected")
	}

	var log models.CronjobExecutionLog
	err := database.DB.QueryRow(`
		SELECT
			id, job_name, job_category, execution_id, status,
			started_at, completed_at, duration_seconds,
			records_processed, records_updated, records_failed,
			error_message, error_stack_trace,
			k8s_pod_name, k8s_namespace, exit_code, created_at
		FROM cronjob_execution_logs
		WHERE execution_id = $1
	`, executionID).Scan(
		&log.ID, &log.JobName, &log.JobCategory, &log.ExecutionID, &log.Status,
		&log.StartedAt, &log.CompletedAt, &log.DurationSeconds,
		&log.RecordsProcessed, &log.RecordsUpdated, &log.RecordsFailed,
		&log.ErrorMessage, &log.ErrorStackTrace,
		&log.K8sPodName, &log.K8sNamespace, &log.ExitCode, &log.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("execution not found")
	}

	if err != nil {
		return nil, err
	}

	return &log, nil
}

// GetMetrics returns metrics overview for all cronjobs
func (s *CronjobService) GetMetrics(period int) (*models.CronjobMetricsResponse, error) {
	if database.DB == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// Get daily success rate
	rows, err := database.DB.Query(`
		SELECT
			DATE(started_at) as date,
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'success') as successful,
			COUNT(*) FILTER (WHERE status = 'success')::float / COUNT(*)::float * 100 as rate
		FROM cronjob_execution_logs
		WHERE started_at >= NOW() - INTERVAL '1 day' * $1
		GROUP BY DATE(started_at)
		ORDER BY date DESC
	`, period)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dailySuccessRate []models.CronjobDailySummary
	for rows.Next() {
		var summary models.CronjobDailySummary
		var date time.Time
		err := rows.Scan(&date, &summary.Total, &summary.Successful, &summary.Rate)
		if err != nil {
			return nil, err
		}
		summary.Date = date.Format("2006-01-02")
		dailySuccessRate = append(dailySuccessRate, summary)
	}

	// Get job performance
	rows, err = database.DB.Query(`
		SELECT
			job_name,
			AVG(duration_seconds) FILTER (WHERE status = 'success') as avg_duration,
			'stable' as trend
		FROM cronjob_execution_logs
		WHERE started_at >= NOW() - INTERVAL '1 day' * $1
		GROUP BY job_name
		ORDER BY avg_duration DESC
	`, period)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobPerformance []models.CronjobPerformance
	for rows.Next() {
		var perf models.CronjobPerformance
		var avgDuration sql.NullFloat64
		err := rows.Scan(&perf.JobName, &avgDuration, &perf.Trend)
		if err != nil {
			return nil, err
		}
		if avgDuration.Valid {
			perf.AvgDuration = avgDuration.Float64
		}
		jobPerformance = append(jobPerformance, perf)
	}

	// Get failure breakdown
	failureBreakdown := make(map[string]int)
	rows, err = database.DB.Query(`
		SELECT
			status,
			COUNT(*)::int as count
		FROM cronjob_execution_logs
		WHERE started_at >= NOW() - INTERVAL '1 day' * $1
			AND status IN ('failed', 'timeout')
		GROUP BY status
	`, period)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		err := rows.Scan(&status, &count)
		if err != nil {
			return nil, err
		}
		failureBreakdown[status] = count
	}

	return &models.CronjobMetricsResponse{
		DailySuccessRate: dailySuccessRate,
		JobPerformance:   jobPerformance,
		FailureBreakdown: failureBreakdown,
	}, nil
}

// LogExecution logs a cronjob execution
func (s *CronjobService) LogExecution(req *models.LogExecutionRequest) error {
	if database.DB == nil {
		return fmt.Errorf("database not connected")
	}

	// Get job category from schedules table
	var jobCategory string
	err := database.DB.QueryRow(`
		SELECT job_category FROM cronjob_schedules WHERE job_name = $1
	`, req.JobName).Scan(&jobCategory)

	if err != nil && err != sql.ErrNoRows {
		return err
	}

	// Default started_at to now if not provided
	startedAt := time.Now()
	if req.StartedAt != nil {
		startedAt = *req.StartedAt
	}

	// Check if execution already exists
	var existingID int
	err = database.DB.QueryRow(`
		SELECT id FROM cronjob_execution_logs WHERE execution_id = $1
	`, req.ExecutionID).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Insert new execution
		_, err = database.DB.Exec(`
			INSERT INTO cronjob_execution_logs (
				job_name, job_category, execution_id, status,
				started_at, completed_at,
				records_processed, records_updated, records_failed,
				error_message, error_stack_trace,
				k8s_pod_name, k8s_namespace, exit_code
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		`, req.JobName, jobCategory, req.ExecutionID, req.Status,
			startedAt, req.CompletedAt,
			req.RecordsProcessed, req.RecordsUpdated, req.RecordsFailed,
			req.ErrorMessage, req.ErrorStackTrace,
			req.K8sPodName, req.K8sNamespace, req.ExitCode)
	} else if err == nil {
		// Update existing execution
		_, err = database.DB.Exec(`
			UPDATE cronjob_execution_logs
			SET status = $1,
				completed_at = $2,
				records_processed = $3,
				records_updated = $4,
				records_failed = $5,
				error_message = $6,
				error_stack_trace = $7,
				exit_code = $8
			WHERE execution_id = $9
		`, req.Status, req.CompletedAt,
			req.RecordsProcessed, req.RecordsUpdated, req.RecordsFailed,
			req.ErrorMessage, req.ErrorStackTrace, req.ExitCode,
			req.ExecutionID)
	}

	return err
}

// GetAllSchedules returns all cronjob schedules
func (s *CronjobService) GetAllSchedules() ([]models.CronjobSchedule, error) {
	if database.DB == nil {
		return nil, fmt.Errorf("database not connected")
	}

	rows, err := database.DB.Query(`
		SELECT
			id, job_name, job_category, description,
			schedule_cron, schedule_description, is_active,
			expected_duration_seconds, timeout_seconds,
			last_success_at, last_failure_at, consecutive_failures,
			created_at, updated_at
		FROM cronjob_schedules
		ORDER BY job_category, job_name
	`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []models.CronjobSchedule
	for rows.Next() {
		var schedule models.CronjobSchedule
		err := rows.Scan(
			&schedule.ID, &schedule.JobName, &schedule.JobCategory, &schedule.Description,
			&schedule.ScheduleCron, &schedule.ScheduleDescription, &schedule.IsActive,
			&schedule.ExpectedDurationSeconds, &schedule.TimeoutSeconds,
			&schedule.LastSuccessAt, &schedule.LastFailureAt, &schedule.ConsecutiveFailures,
			&schedule.CreatedAt, &schedule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

// Helper function to convert JSONB to string
func jsonbToString(data interface{}) string {
	bytes, err := json.Marshal(data)
	if err != nil {
		return "[]"
	}
	return string(bytes)
}
