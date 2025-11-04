package handlers

import (
	"investorcenter-api/models"
	"investorcenter-api/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SubscriptionHandler struct {
	subscriptionService *services.SubscriptionService
}

func NewSubscriptionHandler(subscriptionService *services.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{subscriptionService: subscriptionService}
}

// ListSubscriptionPlans godoc
// @Summary List all subscription plans
// @Tags subscriptions
// @Produce json
// @Success 200 {array} models.SubscriptionPlan
// @Router /api/v1/subscriptions/plans [get]
func (h *SubscriptionHandler) ListSubscriptionPlans(c *gin.Context) {
	plans, err := h.subscriptionService.GetAllPlans()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch subscription plans"})
		return
	}

	c.JSON(http.StatusOK, plans)
}

// GetSubscriptionPlan godoc
// @Summary Get subscription plan by ID
// @Tags subscriptions
// @Produce json
// @Param id path string true "Plan ID"
// @Success 200 {object} models.SubscriptionPlan
// @Router /api/v1/subscriptions/plans/:id [get]
func (h *SubscriptionHandler) GetSubscriptionPlan(c *gin.Context) {
	planID := c.Param("id")

	plan, err := h.subscriptionService.GetPlanByID(planID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription plan not found"})
		return
	}

	c.JSON(http.StatusOK, plan)
}

// GetUserSubscription godoc
// @Summary Get user's current subscription
// @Tags subscriptions
// @Produce json
// @Success 200 {object} models.UserSubscriptionWithPlan
// @Router /api/v1/subscriptions/me [get]
func (h *SubscriptionHandler) GetUserSubscription(c *gin.Context) {
	userID := c.GetString("user_id")

	subscription, err := h.subscriptionService.GetUserSubscription(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch subscription"})
		return
	}

	c.JSON(http.StatusOK, subscription)
}

// CreateSubscription godoc
// @Summary Create a new subscription
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param subscription body models.CreateSubscriptionRequest true "Subscription details"
// @Success 201 {object} models.UserSubscription
// @Router /api/v1/subscriptions [post]
func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
	userID := c.GetString("user_id")

	var req models.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	subscription, err := h.subscriptionService.CreateSubscription(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, subscription)
}

// UpdateSubscription godoc
// @Summary Update user's subscription
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param subscription body models.UpdateSubscriptionRequest true "Update details"
// @Success 200 {object} models.UserSubscription
// @Router /api/v1/subscriptions/me [put]
func (h *SubscriptionHandler) UpdateSubscription(c *gin.Context) {
	userID := c.GetString("user_id")

	var req models.UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	subscription, err := h.subscriptionService.UpdateSubscription(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, subscription)
}

// CancelSubscription godoc
// @Summary Cancel user's subscription
// @Tags subscriptions
// @Success 200
// @Router /api/v1/subscriptions/me/cancel [post]
func (h *SubscriptionHandler) CancelSubscription(c *gin.Context) {
	userID := c.GetString("user_id")

	if err := h.subscriptionService.CancelSubscription(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Subscription canceled successfully"})
}

// GetSubscriptionLimits godoc
// @Summary Get user's subscription limits
// @Tags subscriptions
// @Produce json
// @Success 200 {object} models.SubscriptionLimits
// @Router /api/v1/subscriptions/limits [get]
func (h *SubscriptionHandler) GetSubscriptionLimits(c *gin.Context) {
	userID := c.GetString("user_id")

	limits, err := h.subscriptionService.GetUserLimits(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch subscription limits"})
		return
	}

	c.JSON(http.StatusOK, limits)
}

// GetPaymentHistory godoc
// @Summary Get user's payment history
// @Tags subscriptions
// @Produce json
// @Param limit query int false "Number of results" default(50)
// @Success 200 {array} models.PaymentHistory
// @Router /api/v1/subscriptions/payments [get]
func (h *SubscriptionHandler) GetPaymentHistory(c *gin.Context) {
	userID := c.GetString("user_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	payments, err := h.subscriptionService.GetPaymentHistory(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payment history"})
		return
	}

	c.JSON(http.StatusOK, payments)
}
