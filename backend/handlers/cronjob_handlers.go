package handlers

import (
	"investorcenter-api/models"
	"investorcenter-api/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CronjobHandler struct {
	cronjobService *services.CronjobService
}

func NewCronjobHandler(cronjobService *services.CronjobService) *CronjobHandler {
	return &CronjobHandler{cronjobService: cronjobService}
}

// GetOverview godoc
// @Summary Get cronjob overview
// @Description Get summary and status of all cronjobs
// @Tags cronjobs
// @Produce json
// @Success 200 {object} models.CronjobOverviewResponse
// @Router /api/v1/admin/cronjobs/overview [get]
func (h *CronjobHandler) GetOverview(c *gin.Context) {
	overview, err := h.cronjobService.GetOverview()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cronjob overview", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, overview)
}

// GetJobHistory godoc
// @Summary Get job execution history
// @Description Get execution history for a specific cronjob
// @Tags cronjobs
// @Produce json
// @Param jobName path string true "Job name"
// @Param limit query int false "Number of results" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} models.CronjobHistoryResponse
// @Router /api/v1/admin/cronjobs/{jobName}/history [get]
func (h *CronjobHandler) GetJobHistory(c *gin.Context) {
	jobName := c.Param("jobName")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	history, err := h.cronjobService.GetJobHistory(jobName, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job history", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, history)
}

// GetJobDetails godoc
// @Summary Get job execution details
// @Description Get detailed information about a specific job execution
// @Tags cronjobs
// @Produce json
// @Param executionId path string true "Execution ID"
// @Success 200 {object} models.CronjobExecutionLog
// @Router /api/v1/admin/cronjobs/details/{executionId} [get]
func (h *CronjobHandler) GetJobDetails(c *gin.Context) {
	executionID := c.Param("executionId")

	details, err := h.cronjobService.GetJobDetails(executionID)
	if err != nil {
		if err.Error() == "execution not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Execution not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job details", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, details)
}

// GetMetrics godoc
// @Summary Get cronjob metrics
// @Description Get metrics overview for all cronjobs
// @Tags cronjobs
// @Produce json
// @Param period query int false "Period in days" default(7)
// @Success 200 {object} models.CronjobMetricsResponse
// @Router /api/v1/admin/cronjobs/metrics [get]
func (h *CronjobHandler) GetMetrics(c *gin.Context) {
	period, _ := strconv.Atoi(c.DefaultQuery("period", "7"))

	metrics, err := h.cronjobService.GetMetrics(period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get metrics", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// LogExecution godoc
// @Summary Log cronjob execution
// @Description Log a cronjob execution (for cronjobs to call)
// @Tags cronjobs
// @Accept json
// @Produce json
// @Param execution body models.LogExecutionRequest true "Execution log"
// @Success 200 {object} map[string]bool
// @Router /api/v1/admin/cronjobs/log [post]
func (h *CronjobHandler) LogExecution(c *gin.Context) {
	var req models.LogExecutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.cronjobService.LogExecution(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to log execution", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetAllSchedules godoc
// @Summary Get all cronjob schedules
// @Description Get all configured cronjob schedules
// @Tags cronjobs
// @Produce json
// @Success 200 {array} models.CronjobSchedule
// @Router /api/v1/admin/cronjobs/schedules [get]
func (h *CronjobHandler) GetAllSchedules(c *gin.Context) {
	schedules, err := h.cronjobService.GetAllSchedules()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get schedules", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, schedules)
}
