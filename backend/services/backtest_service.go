package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"investorcenter-api/database"
	"investorcenter-api/models"
)

// BacktestService handles backtest operations
type BacktestService struct {
	icScoreAPIURL string
	httpClient    *http.Client
}

// NewBacktestService creates a new backtest service
func NewBacktestService() *BacktestService {
	apiURL := os.Getenv("IC_SCORE_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8001"
	}

	return &BacktestService{
		icScoreAPIURL: apiURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Minute, // Backtests can take a while
		},
	}
}

// RunBacktest executes a backtest with the given configuration
func (s *BacktestService) RunBacktest(config models.BacktestConfig) (*models.BacktestSummary, error) {
	// Call the IC Score service backtest endpoint
	url := fmt.Sprintf("%s/api/v1/backtest", s.icScoreAPIURL)

	body, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call backtest API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("backtest API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var summary models.BacktestSummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &summary, nil
}

// SubmitBacktestJob creates a new backtest job and runs it asynchronously
func (s *BacktestService) SubmitBacktestJob(config models.BacktestConfig, userID *string) (*models.BacktestJob, error) {
	// Create job in pending state
	job, err := database.CreateBacktestJob(config, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	// Run backtest asynchronously
	go s.runBacktestJob(job.ID, config)

	return job, nil
}

// runBacktestJob runs the backtest and updates job status
func (s *BacktestService) runBacktestJob(jobID string, config models.BacktestConfig) {
	// Update status to running
	if err := database.UpdateBacktestJobStatus(jobID, models.BacktestStatusRunning); err != nil {
		database.FailBacktestJob(jobID, fmt.Sprintf("failed to update status: %v", err))
		return
	}

	// Run backtest
	summary, err := s.RunBacktest(config)
	if err != nil {
		database.FailBacktestJob(jobID, err.Error())
		return
	}

	// Mark as completed
	if err := database.CompleteBacktestJob(jobID, summary); err != nil {
		database.FailBacktestJob(jobID, fmt.Sprintf("failed to save results: %v", err))
		return
	}
}

// GetJobStatus retrieves the status of a backtest job
func (s *BacktestService) GetJobStatus(jobID string) (*models.BacktestJob, error) {
	return database.GetBacktestJob(jobID)
}

// GetJobResult retrieves the result of a completed backtest job
func (s *BacktestService) GetJobResult(jobID string) (*models.BacktestSummary, error) {
	job, err := database.GetBacktestJob(jobID)
	if err != nil {
		return nil, err
	}

	if job.Status != models.BacktestStatusCompleted {
		return nil, fmt.Errorf("job not completed, current status: %s", job.Status)
	}

	if job.Result == nil {
		return nil, fmt.Errorf("job has no result")
	}

	var summary models.BacktestSummary
	if err := json.Unmarshal([]byte(*job.Result), &summary); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &summary, nil
}

// GetLatestBacktest retrieves the most recent completed backtest
func (s *BacktestService) GetLatestBacktest() (*models.BacktestSummary, error) {
	job, err := database.GetRecentCompletedBacktest()
	if err != nil {
		return nil, err
	}

	if job.Result == nil {
		return nil, fmt.Errorf("no result available")
	}

	var summary models.BacktestSummary
	if err := json.Unmarshal([]byte(*job.Result), &summary); err != nil {
		return nil, err
	}

	return &summary, nil
}

// GetCachedOrRunBacktest returns cached results if available, otherwise runs new backtest
func (s *BacktestService) GetCachedOrRunBacktest(config models.BacktestConfig) (*models.BacktestSummary, error) {
	// Try to get cached result (max 24 hours old)
	job, err := database.GetCachedBacktestResult(config, 24*time.Hour)
	if err == nil && job.Result != nil {
		var summary models.BacktestSummary
		if err := json.Unmarshal([]byte(*job.Result), &summary); err == nil {
			return &summary, nil
		}
	}

	// Run new backtest
	return s.RunBacktest(config)
}

// GetUserBacktests retrieves backtest history for a user
func (s *BacktestService) GetUserBacktests(userID string, limit int) ([]models.BacktestJob, error) {
	return database.GetUserBacktestJobs(userID, limit)
}

// GetDefaultBacktestConfig returns the default backtest configuration
func (s *BacktestService) GetDefaultBacktestConfig() models.BacktestConfig {
	return models.BacktestConfig{
		StartDate:          time.Now().AddDate(-5, 0, 0).Format("2006-01-02"),
		EndDate:            time.Now().Format("2006-01-02"),
		RebalanceFrequency: "monthly",
		Universe:           "sp500",
		TransactionCostBps: 10,
		SlippageBps:        5,
		UseSmoothedScores:  true,
		Benchmark:          "SPY",
		ExcludeFinancials:  false,
		ExcludeUtilities:   false,
	}
}

// GenerateCharts generates chart data for the backtest dashboard
func (s *BacktestService) GenerateCharts(summary *models.BacktestSummary) *models.BacktestCharts {
	charts := &models.BacktestCharts{}

	// Decile bar chart
	labels := make([]string, len(summary.DecilePerformance))
	returns := make([]float64, len(summary.DecilePerformance))
	colors := make([]string, len(summary.DecilePerformance))

	for i, dp := range summary.DecilePerformance {
		labels[i] = fmt.Sprintf("D%d", dp.Decile)
		returns[i] = dp.AnnualizedReturn * 100
		if dp.AnnualizedReturn > 0 {
			colors[i] = "#10b981" // Green
		} else {
			colors[i] = "#ef4444" // Red
		}
	}

	charts.DecileBarChart = models.BacktestChartData{
		Labels: labels,
		Datasets: []map[string]interface{}{
			{
				"label":           "Annualized Return (%)",
				"data":            returns,
				"backgroundColor": colors,
			},
		},
	}

	return charts
}

// ValidateConfig validates backtest configuration
func (s *BacktestService) ValidateConfig(config models.BacktestConfig) error {
	// Parse dates
	start, err := time.Parse("2006-01-02", config.StartDate)
	if err != nil {
		return fmt.Errorf("invalid start_date format: %w", err)
	}

	end, err := time.Parse("2006-01-02", config.EndDate)
	if err != nil {
		return fmt.Errorf("invalid end_date format: %w", err)
	}

	// Validate date range
	if !end.After(start) {
		return fmt.Errorf("end_date must be after start_date")
	}

	if end.After(time.Now()) {
		return fmt.Errorf("end_date cannot be in the future")
	}

	// Minimum 1 year of data
	if end.Sub(start) < 365*24*time.Hour {
		return fmt.Errorf("backtest period must be at least 1 year")
	}

	// Maximum 10 years
	if end.Sub(start) > 10*365*24*time.Hour {
		return fmt.Errorf("backtest period cannot exceed 10 years")
	}

	// Validate rebalance frequency
	validFrequencies := map[string]bool{
		"daily":     true,
		"weekly":    true,
		"monthly":   true,
		"quarterly": true,
	}
	if !validFrequencies[config.RebalanceFrequency] {
		return fmt.Errorf("invalid rebalance_frequency: must be daily, weekly, monthly, or quarterly")
	}

	// Validate universe
	validUniverses := map[string]bool{
		"sp500":  true,
		"sp1500": true,
		"all":    true,
	}
	if !validUniverses[config.Universe] {
		return fmt.Errorf("invalid universe: must be sp500, sp1500, or all")
	}

	// Validate cost parameters
	if config.TransactionCostBps < 0 || config.TransactionCostBps > 100 {
		return fmt.Errorf("transaction_cost_bps must be between 0 and 100")
	}

	if config.SlippageBps < 0 || config.SlippageBps > 100 {
		return fmt.Errorf("slippage_bps must be between 0 and 100")
	}

	return nil
}
