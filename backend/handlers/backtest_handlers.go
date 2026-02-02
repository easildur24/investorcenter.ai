package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"investorcenter/models"
	"investorcenter/services"
)

// BacktestHandler handles backtest-related HTTP requests
type BacktestHandler struct {
	service *services.BacktestService
}

// NewBacktestHandler creates a new backtest handler
func NewBacktestHandler(service *services.BacktestService) *BacktestHandler {
	return &BacktestHandler{service: service}
}

// RunBacktest executes a backtest with the provided configuration
// POST /api/v1/ic-scores/backtest
func (h *BacktestHandler) RunBacktest(w http.ResponseWriter, r *http.Request) {
	var config models.BacktestConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate configuration
	if err := h.service.ValidateConfig(config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Run backtest (synchronous for simple requests)
	summary, err := h.service.RunBacktest(config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// SubmitBacktestJob creates a new backtest job and runs it asynchronously
// POST /api/v1/ic-scores/backtest/jobs
func (h *BacktestHandler) SubmitBacktestJob(w http.ResponseWriter, r *http.Request) {
	var config models.BacktestConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate configuration
	if err := h.service.ValidateConfig(config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get user ID from context if authenticated
	var userID *string
	if user, ok := r.Context().Value("user").(*models.User); ok {
		userID = &user.ID
	}

	// Submit job
	job, err := h.service.SubmitBacktestJob(config, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"job_id":  job.ID,
		"status":  job.Status,
		"message": "Backtest job submitted successfully",
	})
}

// GetBacktestJobStatus retrieves the status of a backtest job
// GET /api/v1/ic-scores/backtest/jobs/{jobId}
func (h *BacktestHandler) GetBacktestJobStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["jobId"]

	if jobID == "" {
		http.Error(w, "Job ID is required", http.StatusBadRequest)
		return
	}

	job, err := h.service.GetJobStatus(jobID)
	if err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"job_id":     job.ID,
		"status":     job.Status,
		"created_at": job.CreatedAt,
	}

	if job.StartedAt != nil {
		response["started_at"] = job.StartedAt
	}

	if job.CompletedAt != nil {
		response["completed_at"] = job.CompletedAt
	}

	if job.Error != nil {
		response["error"] = *job.Error
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetBacktestJobResult retrieves the result of a completed backtest job
// GET /api/v1/ic-scores/backtest/jobs/{jobId}/result
func (h *BacktestHandler) GetBacktestJobResult(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["jobId"]

	if jobID == "" {
		http.Error(w, "Job ID is required", http.StatusBadRequest)
		return
	}

	summary, err := h.service.GetJobResult(jobID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// GetLatestBacktest retrieves the most recent completed backtest
// GET /api/v1/ic-scores/backtest/latest
func (h *BacktestHandler) GetLatestBacktest(w http.ResponseWriter, r *http.Request) {
	summary, err := h.service.GetLatestBacktest()
	if err != nil {
		http.Error(w, "No completed backtests found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// GetDefaultConfig returns the default backtest configuration
// GET /api/v1/ic-scores/backtest/config/default
func (h *BacktestHandler) GetDefaultConfig(w http.ResponseWriter, r *http.Request) {
	config := h.service.GetDefaultBacktestConfig()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// GetBacktestCharts returns chart data for the backtest dashboard
// GET /api/v1/ic-scores/backtest/charts
func (h *BacktestHandler) GetBacktestCharts(w http.ResponseWriter, r *http.Request) {
	// Get the latest backtest results
	summary, err := h.service.GetLatestBacktest()
	if err != nil {
		http.Error(w, "No completed backtests found", http.StatusNotFound)
		return
	}

	charts := h.service.GenerateCharts(summary)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(charts)
}

// GetUserBacktests retrieves backtest history for the authenticated user
// GET /api/v1/ic-scores/backtest/history
func (h *BacktestHandler) GetUserBacktests(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	jobs, err := h.service.GetUserBacktests(user.ID, 10)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Transform to response format
	response := make([]map[string]interface{}, len(jobs))
	for i, job := range jobs {
		response[i] = map[string]interface{}{
			"job_id":     job.ID,
			"status":     job.Status,
			"created_at": job.CreatedAt,
		}

		if job.CompletedAt != nil {
			response[i]["completed_at"] = job.CompletedAt
		}

		// Parse config for display
		var config models.BacktestConfig
		if err := json.Unmarshal([]byte(job.Config), &config); err == nil {
			response[i]["config"] = map[string]interface{}{
				"start_date":          config.StartDate,
				"end_date":            config.EndDate,
				"rebalance_frequency": config.RebalanceFrequency,
				"universe":            config.Universe,
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RunQuickBacktest runs a quick backtest with default settings
// GET /api/v1/ic-scores/backtest/quick
func (h *BacktestHandler) RunQuickBacktest(w http.ResponseWriter, r *http.Request) {
	// Use default config with caching
	config := h.service.GetDefaultBacktestConfig()

	summary, err := h.service.GetCachedOrRunBacktest(config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}
