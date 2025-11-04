package services

import (
	"errors"
	"investorcenter-api/database"
	"investorcenter-api/models"
	"time"
)

type SubscriptionService struct{}

func NewSubscriptionService() *SubscriptionService {
	return &SubscriptionService{}
}

// GetAllPlans retrieves all available subscription plans
func (s *SubscriptionService) GetAllPlans() ([]models.SubscriptionPlan, error) {
	return database.GetAllSubscriptionPlans()
}

// GetPlanByID retrieves a specific plan by ID
func (s *SubscriptionService) GetPlanByID(planID string) (*models.SubscriptionPlan, error) {
	return database.GetSubscriptionPlanByID(planID)
}

// GetUserSubscription retrieves the user's current subscription
func (s *SubscriptionService) GetUserSubscription(userID string) (*models.UserSubscriptionWithPlan, error) {
	return database.GetUserSubscription(userID)
}

// CreateSubscription creates a new subscription for a user
func (s *SubscriptionService) CreateSubscription(userID string, req *models.CreateSubscriptionRequest) (*models.UserSubscription, error) {
	// Validate plan exists
	plan, err := database.GetSubscriptionPlanByID(req.PlanID)
	if err != nil {
		return nil, errors.New("subscription plan not found")
	}

	// Validate billing period
	if req.BillingPeriod != "monthly" && req.BillingPeriod != "yearly" {
		return nil, errors.New("invalid billing period: must be 'monthly' or 'yearly'")
	}

	// Check if user already has an active subscription
	existing, _ := database.GetUserSubscription(userID)
	if existing != nil && existing.Status == "active" {
		return nil, errors.New("user already has an active subscription")
	}

	// Calculate period end based on billing period
	currentPeriodStart := time.Now()
	var currentPeriodEnd time.Time
	if req.BillingPeriod == "monthly" {
		currentPeriodEnd = currentPeriodStart.AddDate(0, 1, 0)
	} else {
		currentPeriodEnd = currentPeriodStart.AddDate(1, 0, 0)
	}

	// Create subscription
	subscription := &models.UserSubscription{
		UserID:             userID,
		PlanID:             plan.ID,
		Status:             "active",
		BillingPeriod:      req.BillingPeriod,
		CurrentPeriodStart: currentPeriodStart,
		CurrentPeriodEnd:   &currentPeriodEnd,
		NextPaymentDate:    &currentPeriodEnd,
	}

	if err := database.CreateUserSubscription(subscription); err != nil {
		return nil, err
	}

	return subscription, nil
}

// UpdateSubscription updates a user's subscription
func (s *SubscriptionService) UpdateSubscription(userID string, req *models.UpdateSubscriptionRequest) (*models.UserSubscription, error) {
	// Get existing subscription
	existing, err := database.GetUserSubscription(userID)
	if err != nil {
		return nil, errors.New("no active subscription found")
	}

	updates := make(map[string]interface{})

	if req.PlanID != nil {
		// Validate new plan exists
		_, err := database.GetSubscriptionPlanByID(*req.PlanID)
		if err != nil {
			return nil, errors.New("subscription plan not found")
		}
		updates["plan_id"] = *req.PlanID
	}

	if req.BillingPeriod != nil {
		if *req.BillingPeriod != "monthly" && *req.BillingPeriod != "yearly" {
			return nil, errors.New("invalid billing period: must be 'monthly' or 'yearly'")
		}
		updates["billing_period"] = *req.BillingPeriod
	}

	if req.PaymentMethod != nil {
		updates["payment_method"] = *req.PaymentMethod
	}

	if err := database.UpdateUserSubscription(existing.ID, updates); err != nil {
		return nil, err
	}

	// Get updated subscription
	updated, err := database.GetUserSubscription(userID)
	if err != nil {
		return nil, err
	}

	return &updated.UserSubscription, nil
}

// CancelSubscription cancels a user's subscription
func (s *SubscriptionService) CancelSubscription(userID string) error {
	return database.CancelUserSubscription(userID)
}

// GetUserLimits retrieves the subscription limits for a user
func (s *SubscriptionService) GetUserLimits(userID string) (*models.SubscriptionLimits, error) {
	return database.GetUserSubscriptionLimits(userID)
}

// CheckLimit checks if a user can perform an action based on their subscription limits
func (s *SubscriptionService) CheckLimit(userID string, limitType string, currentCount int) (bool, error) {
	limits, err := s.GetUserLimits(userID)
	if err != nil {
		return false, err
	}

	var maxLimit int
	switch limitType {
	case "watch_lists":
		maxLimit = limits.MaxWatchLists
	case "items_per_watch_list":
		maxLimit = limits.MaxItemsPerWatchList
	case "alert_rules":
		maxLimit = limits.MaxAlertRules
	case "heatmap_configs":
		maxLimit = limits.MaxHeatmapConfigs
	default:
		return false, errors.New("invalid limit type")
	}

	// -1 means unlimited
	if maxLimit == -1 {
		return true, nil
	}

	return currentCount < maxLimit, nil
}

// RecordPayment records a payment transaction
func (s *SubscriptionService) RecordPayment(userID string, subscriptionID *string, amount float64, currency string, status string, method *string, stripePaymentIntentID *string) error {
	payment := &models.PaymentHistory{
		UserID:                userID,
		SubscriptionID:        subscriptionID,
		Amount:                amount,
		Currency:              currency,
		Status:                status,
		PaymentMethod:         method,
		StripePaymentIntentID: stripePaymentIntentID,
	}

	return database.CreatePaymentHistory(payment)
}

// GetPaymentHistory retrieves payment history for a user
func (s *SubscriptionService) GetPaymentHistory(userID string, limit int) ([]models.PaymentHistory, error) {
	if limit == 0 {
		limit = 50
	}
	return database.GetPaymentHistory(userID, limit)
}
