package handlers

import (
	"investorcenter-api/models"
	"investorcenter-api/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AlertHandler struct {
	alertService *services.AlertService
}

func NewAlertHandler(alertService *services.AlertService) *AlertHandler {
	return &AlertHandler{alertService: alertService}
}

// ListAlertRules godoc
// @Summary List all alert rules for a user
// @Tags alerts
// @Produce json
// @Param watch_list_id query string false "Filter by watch list ID"
// @Param is_active query bool false "Filter by active status"
// @Success 200 {array} models.AlertRuleWithDetails
// @Router /api/v1/alerts [get]
func (h *AlertHandler) ListAlertRules(c *gin.Context) {
	userID := c.GetString("user_id")
	watchListID := c.Query("watch_list_id")
	isActive := c.Query("is_active")

	alerts, err := h.alertService.GetUserAlerts(userID, watchListID, isActive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch alerts"})
		return
	}

	c.JSON(http.StatusOK, alerts)
}

// CreateAlertRule godoc
// @Summary Create a new alert rule
// @Tags alerts
// @Accept json
// @Produce json
// @Param alert body models.CreateAlertRuleRequest true "Alert rule details"
// @Success 201 {object} models.AlertRule
// @Router /api/v1/alerts [post]
func (h *AlertHandler) CreateAlertRule(c *gin.Context) {
	userID := c.GetString("user_id")

	var req models.CreateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate watch list ownership
	if err := h.alertService.ValidateWatchListOwnership(userID, req.WatchListID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Watch list not found"})
		return
	}

	// Check tier limits
	canCreate, err := h.alertService.CanCreateAlert(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check limits"})
		return
	}
	if !canCreate {
		c.JSON(http.StatusForbidden, gin.H{"error": "Alert limit reached. Upgrade to Premium for more alerts."})
		return
	}

	// Create alert
	alert, err := h.alertService.CreateAlert(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, alert)
}

// GetAlertRule godoc
// @Summary Get alert rule by ID
// @Tags alerts
// @Produce json
// @Param id path string true "Alert ID"
// @Success 200 {object} models.AlertRule
// @Router /api/v1/alerts/:id [get]
func (h *AlertHandler) GetAlertRule(c *gin.Context) {
	userID := c.GetString("user_id")
	alertID := c.Param("id")

	alert, err := h.alertService.GetAlertByID(alertID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}

	c.JSON(http.StatusOK, alert)
}

// UpdateAlertRule godoc
// @Summary Update alert rule
// @Tags alerts
// @Accept json
// @Produce json
// @Param id path string true "Alert ID"
// @Param alert body models.UpdateAlertRuleRequest true "Update details"
// @Success 200 {object} models.AlertRule
// @Router /api/v1/alerts/:id [put]
func (h *AlertHandler) UpdateAlertRule(c *gin.Context) {
	userID := c.GetString("user_id")
	alertID := c.Param("id")

	var req models.UpdateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	alert, err := h.alertService.UpdateAlert(alertID, userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, alert)
}

// DeleteAlertRule godoc
// @Summary Delete alert rule
// @Tags alerts
// @Param id path string true "Alert ID"
// @Success 204
// @Router /api/v1/alerts/:id [delete]
func (h *AlertHandler) DeleteAlertRule(c *gin.Context) {
	userID := c.GetString("user_id")
	alertID := c.Param("id")

	if err := h.alertService.DeleteAlert(alertID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete alert"})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListAlertLogs godoc
// @Summary Get alert trigger history
// @Tags alerts
// @Produce json
// @Param alert_id query string false "Filter by alert rule ID"
// @Param symbol query string false "Filter by symbol"
// @Param limit query int false "Number of results" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {array} models.AlertLogWithRule
// @Router /api/v1/alerts/logs [get]
func (h *AlertHandler) ListAlertLogs(c *gin.Context) {
	userID := c.GetString("user_id")
	alertID := c.Query("alert_id")
	symbol := c.Query("symbol")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	logs, err := h.alertService.GetAlertLogs(userID, alertID, symbol, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch alert logs"})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// MarkAlertLogRead godoc
// @Summary Mark alert log as read
// @Tags alerts
// @Param id path string true "Alert Log ID"
// @Success 200
// @Router /api/v1/alerts/logs/:id/read [post]
func (h *AlertHandler) MarkAlertLogRead(c *gin.Context) {
	userID := c.GetString("user_id")
	logID := c.Param("id")

	if err := h.alertService.MarkAlertLogAsRead(logID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark alert log as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DismissAlertLog godoc
// @Summary Dismiss alert log
// @Tags alerts
// @Param id path string true "Alert Log ID"
// @Success 200
// @Router /api/v1/alerts/logs/:id/dismiss [post]
func (h *AlertHandler) DismissAlertLog(c *gin.Context) {
	userID := c.GetString("user_id")
	logID := c.Param("id")

	if err := h.alertService.MarkAlertLogAsDismissed(logID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to dismiss alert log"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
