package handlers

import (
	"investorcenter-api/models"
	"investorcenter-api/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	notificationService *services.NotificationService
}

func NewNotificationHandler(notificationService *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{notificationService: notificationService}
}

// GetNotificationPreferences godoc
// @Summary Get notification preferences
// @Tags notifications
// @Produce json
// @Success 200 {object} models.NotificationPreferences
// @Router /api/v1/notifications/preferences [get]
func (h *NotificationHandler) GetNotificationPreferences(c *gin.Context) {
	userID := c.GetString("user_id")

	prefs, err := h.notificationService.GetNotificationPreferences(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get notification preferences"})
		return
	}

	c.JSON(http.StatusOK, prefs)
}

// UpdateNotificationPreferences godoc
// @Summary Update notification preferences
// @Tags notifications
// @Accept json
// @Produce json
// @Param preferences body models.UpdateNotificationPreferencesRequest true "Notification preferences"
// @Success 200 {object} models.NotificationPreferences
// @Router /api/v1/notifications/preferences [put]
func (h *NotificationHandler) UpdateNotificationPreferences(c *gin.Context) {
	userID := c.GetString("user_id")

	var req models.UpdateNotificationPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prefs, err := h.notificationService.UpdateNotificationPreferences(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, prefs)
}

// GetInAppNotifications godoc
// @Summary Get in-app notifications
// @Tags notifications
// @Produce json
// @Param unread_only query bool false "Only return unread notifications"
// @Param limit query int false "Number of results" default(50)
// @Success 200 {array} models.InAppNotification
// @Router /api/v1/notifications [get]
func (h *NotificationHandler) GetInAppNotifications(c *gin.Context) {
	userID := c.GetString("user_id")
	unreadOnly := c.Query("unread_only") == "true"
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	notifications, err := h.notificationService.GetInAppNotifications(userID, unreadOnly, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	c.JSON(http.StatusOK, notifications)
}

// GetUnreadCount godoc
// @Summary Get unread notification count
// @Tags notifications
// @Produce json
// @Success 200 {object} map[string]int
// @Router /api/v1/notifications/unread-count [get]
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID := c.GetString("user_id")

	count, err := h.notificationService.GetUnreadNotificationCount(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get unread count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// MarkNotificationRead godoc
// @Summary Mark notification as read
// @Tags notifications
// @Param id path string true "Notification ID"
// @Success 200
// @Router /api/v1/notifications/:id/read [post]
func (h *NotificationHandler) MarkNotificationRead(c *gin.Context) {
	userID := c.GetString("user_id")
	notificationID := c.Param("id")

	if err := h.notificationService.MarkNotificationAsRead(notificationID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notification as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// MarkAllNotificationsRead godoc
// @Summary Mark all notifications as read
// @Tags notifications
// @Success 200
// @Router /api/v1/notifications/read-all [post]
func (h *NotificationHandler) MarkAllNotificationsRead(c *gin.Context) {
	userID := c.GetString("user_id")

	if err := h.notificationService.MarkAllNotificationsAsRead(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark all notifications as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DismissNotification godoc
// @Summary Dismiss notification
// @Tags notifications
// @Param id path string true "Notification ID"
// @Success 200
// @Router /api/v1/notifications/:id/dismiss [post]
func (h *NotificationHandler) DismissNotification(c *gin.Context) {
	userID := c.GetString("user_id")
	notificationID := c.Param("id")

	if err := h.notificationService.DismissNotification(notificationID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to dismiss notification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
