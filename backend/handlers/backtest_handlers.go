package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"investorcenter-api/models"
	"investorcenter-api/services"
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
func (h *BacktestHandler) RunBacktest(c *gin.Context) {
	var config models.BacktestConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate configuration
	if err := h.service.ValidateConfig(config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Run backtest (synchronous for simple requests)
	summary, err := h.service.RunBacktest(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// SubmitBacktestJob creates a new backtest job and runs it asynchronously
// POST /api/v1/ic-scores/backtest/jobs
func (h *BacktestHandler) SubmitBacktestJob(c *gin.Context) {
	var config models.BacktestConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate configuration
	if err := h.service.ValidateConfig(config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context if authenticated
	var userID *string
	if user, exists := c.Get("user"); exists {
		if u, ok := user.(*models.User); ok {
			userID = &u.ID
		}
	}

	// Submit job
	job, err := h.service.SubmitBacktestJob(config, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"job_id":  job.ID,
		"status":  job.Status,
		"message": "Backtest job submitted successfully",
	})
}

// GetBacktestJobStatus retrieves the status of a backtest job
// GET /api/v1/ic-scores/backtest/jobs/:jobId
func (h *BacktestHandler) GetBacktestJobStatus(c *gin.Context) {
	jobID := c.Param("jobId")

	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	job, err := h.service.GetJobStatus(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	response := gin.H{
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

	c.JSON(http.StatusOK, response)
}

// GetBacktestJobResult retrieves the result of a completed backtest job
// GET /api/v1/ic-scores/backtest/jobs/:jobId/result
func (h *BacktestHandler) GetBacktestJobResult(c *gin.Context) {
	jobID := c.Param("jobId")

	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	summary, err := h.service.GetJobResult(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetLatestBacktest retrieves the most recent completed backtest
// GET /api/v1/ic-scores/backtest/latest
func (h *BacktestHandler) GetLatestBacktest(c *gin.Context) {
	summary, err := h.service.GetLatestBacktest()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No completed backtests found"})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetDefaultConfig returns the default backtest configuration
// GET /api/v1/ic-scores/backtest/config/default
func (h *BacktestHandler) GetDefaultConfig(c *gin.Context) {
	config := h.service.GetDefaultBacktestConfig()
	c.JSON(http.StatusOK, config)
}

// GetBacktestCharts returns chart data for the backtest dashboard
// GET /api/v1/ic-scores/backtest/charts
func (h *BacktestHandler) GetBacktestCharts(c *gin.Context) {
	// Get the latest backtest results
	summary, err := h.service.GetLatestBacktest()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No completed backtests found"})
		return
	}

	charts := h.service.GenerateCharts(summary)
	c.JSON(http.StatusOK, charts)
}

// GetUserBacktests retrieves backtest history for the authenticated user
// GET /api/v1/ic-scores/backtest/history
func (h *BacktestHandler) GetUserBacktests(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	u, ok := user.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user context"})
		return
	}

	jobs, err := h.service.GetUserBacktests(u.ID, 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Transform to response format
	response := make([]gin.H, len(jobs))
	for i, job := range jobs {
		response[i] = gin.H{
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
			response[i]["config"] = gin.H{
				"start_date":          config.StartDate,
				"end_date":            config.EndDate,
				"rebalance_frequency": config.RebalanceFrequency,
				"universe":            config.Universe,
			}
		}
	}

	c.JSON(http.StatusOK, response)
}

// RunQuickBacktest runs a quick backtest with default settings
// GET /api/v1/ic-scores/backtest/quick
func (h *BacktestHandler) RunQuickBacktest(c *gin.Context) {
	// Use default config with caching
	config := h.service.GetDefaultBacktestConfig()

	summary, err := h.service.GetCachedOrRunBacktest(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}
