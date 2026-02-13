package models

import "time"

// CronjobSchedule represents a configured cronjob schedule
type CronjobSchedule struct {
	ID                      int        `json:"id" db:"id"`
	JobName                 string     `json:"job_name" db:"job_name"`
	JobCategory             string     `json:"job_category" db:"job_category"`
	Description             string     `json:"description" db:"description"`
	ScheduleCron            string     `json:"schedule_cron" db:"schedule_cron"`
	ScheduleDescription     string     `json:"schedule_description" db:"schedule_description"`
	IsActive                bool       `json:"is_active" db:"is_active"`
	ExpectedDurationSeconds *int       `json:"expected_duration_seconds" db:"expected_duration_seconds"`
	TimeoutSeconds          *int       `json:"timeout_seconds" db:"timeout_seconds"`
	LastSuccessAt           *time.Time `json:"last_success_at" db:"last_success_at"`
	LastFailureAt           *time.Time `json:"last_failure_at" db:"last_failure_at"`
	ConsecutiveFailures     int        `json:"consecutive_failures" db:"consecutive_failures"`
	CreatedAt               time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at" db:"updated_at"`
}

// CronjobExecutionLog represents a single cronjob execution
type CronjobExecutionLog struct {
	ID               int        `json:"id" db:"id"`
	JobName          string     `json:"job_name" db:"job_name"`
	JobCategory      string     `json:"job_category" db:"job_category"`
	ExecutionID      string     `json:"execution_id" db:"execution_id"`
	Status           string     `json:"status" db:"status"`
	StartedAt        time.Time  `json:"started_at" db:"started_at"`
	CompletedAt      *time.Time `json:"completed_at" db:"completed_at"`
	DurationSeconds  *int       `json:"duration_seconds" db:"duration_seconds"`
	RecordsProcessed int        `json:"records_processed" db:"records_processed"`
	RecordsUpdated   int        `json:"records_updated" db:"records_updated"`
	RecordsFailed    int        `json:"records_failed" db:"records_failed"`
	ErrorMessage     *string    `json:"error_message" db:"error_message"`
	ErrorStackTrace  *string    `json:"error_stack_trace" db:"error_stack_trace"`
	K8sPodName       *string    `json:"k8s_pod_name" db:"k8s_pod_name"`
	K8sNamespace     *string    `json:"k8s_namespace" db:"k8s_namespace"`
	ExitCode         *int       `json:"exit_code" db:"exit_code"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

// CronjobAlert represents alert configuration for cronjobs
type CronjobAlert struct {
	ID                   int        `json:"id" db:"id"`
	JobName              *string    `json:"job_name" db:"job_name"`
	AlertType            string     `json:"alert_type" db:"alert_type"`
	AlertThreshold       *int       `json:"alert_threshold" db:"alert_threshold"`
	IsActive             bool       `json:"is_active" db:"is_active"`
	NotificationChannels string     `json:"notification_channels" db:"notification_channels"` // JSONB stored as string
	LastTriggeredAt      *time.Time `json:"last_triggered_at" db:"last_triggered_at"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
}

// CronjobStatistics represents statistics for a cronjob
type CronjobStatistics struct {
	JobName              string     `json:"job_name" db:"job_name"`
	TotalExecutions      int64      `json:"total_executions" db:"total_executions"`
	SuccessfulExecutions int64      `json:"successful_executions" db:"successful_executions"`
	FailedExecutions     int64      `json:"failed_executions" db:"failed_executions"`
	SuccessRate          *float64   `json:"success_rate" db:"success_rate"`
	AvgDurationSeconds   *float64   `json:"avg_duration_seconds" db:"avg_duration_seconds"`
	P50DurationSeconds   *float64   `json:"p50_duration_seconds" db:"p50_duration_seconds"`
	P95DurationSeconds   *float64   `json:"p95_duration_seconds" db:"p95_duration_seconds"`
	LastExecutionStatus  *string    `json:"last_execution_status" db:"last_execution_status"`
	LastExecutionAt      *time.Time `json:"last_execution_at" db:"last_execution_at"`
}

// Request/Response models

// CronjobOverviewResponse represents the overview of all cronjobs
type CronjobOverviewResponse struct {
	Summary CronjobSummary          `json:"summary"`
	Jobs    []CronjobStatusWithInfo `json:"jobs"`
}

// CronjobSummary represents summary statistics
type CronjobSummary struct {
	TotalJobs  int            `json:"total_jobs"`
	ActiveJobs int            `json:"active_jobs"`
	Last24h    Last24hSummary `json:"last_24h"`
}

// Last24hSummary represents stats for last 24 hours
type Last24hSummary struct {
	TotalExecutions int     `json:"total_executions"`
	Successful      int     `json:"successful"`
	Failed          int     `json:"failed"`
	SuccessRate     float64 `json:"success_rate"`
}

// CronjobStatusWithInfo combines schedule info with latest execution
type CronjobStatusWithInfo struct {
	JobName             string       `json:"job_name"`
	JobCategory         string       `json:"job_category"`
	Schedule            string       `json:"schedule"`
	ScheduleDescription string       `json:"schedule_description"`
	LastRun             *LastRunInfo `json:"last_run"`
	HealthStatus        string       `json:"health_status"` // 'healthy', 'warning', 'critical'
	ConsecutiveFailures int          `json:"consecutive_failures"`
	AvgDuration7d       *float64     `json:"avg_duration_7d"`
	SuccessRate7d       *float64     `json:"success_rate_7d"`
}

// LastRunInfo represents the last execution info
type LastRunInfo struct {
	Status           string     `json:"status"`
	StartedAt        time.Time  `json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at"`
	DurationSeconds  *int       `json:"duration_seconds"`
	RecordsProcessed int        `json:"records_processed"`
}

// CronjobHistoryResponse represents execution history for a job
type CronjobHistoryResponse struct {
	JobName         string                `json:"job_name"`
	TotalExecutions int64                 `json:"total_executions"`
	Executions      []CronjobExecutionLog `json:"executions"`
	Metrics         CronjobMetrics        `json:"metrics"`
}

// CronjobMetrics represents performance metrics
type CronjobMetrics struct {
	AvgDuration *float64 `json:"avg_duration"`
	P50Duration *float64 `json:"p50_duration"`
	P95Duration *float64 `json:"p95_duration"`
	SuccessRate *float64 `json:"success_rate"`
}

// CronjobDailySummary represents daily success rate data
type CronjobDailySummary struct {
	Date       string  `json:"date" db:"date"`
	Total      int     `json:"total" db:"total"`
	Successful int     `json:"successful" db:"successful"`
	Rate       float64 `json:"rate" db:"rate"`
}

// CronjobPerformance represents job performance data
type CronjobPerformance struct {
	JobName     string  `json:"job_name" db:"job_name"`
	AvgDuration float64 `json:"avg_duration" db:"avg_duration"`
	Trend       string  `json:"trend" db:"trend"`
}

// CronjobMetricsResponse represents metrics overview
type CronjobMetricsResponse struct {
	DailySuccessRate []CronjobDailySummary `json:"daily_success_rate"`
	JobPerformance   []CronjobPerformance  `json:"job_performance"`
	FailureBreakdown map[string]int        `json:"failure_breakdown"`
}
