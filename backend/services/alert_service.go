package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"investorcenter-api/database"
	"investorcenter-api/models"
)

type AlertService struct{}

func NewAlertService() *AlertService {
	return &AlertService{}
}

// CreateAlert creates a new alert rule
func (s *AlertService) CreateAlert(userID string, req *models.CreateAlertRuleRequest) (*models.AlertRule, error) {
	// Validate alert type
	validTypes := map[string]bool{
		"price_above":        true,
		"price_below":        true,
		"price_change_pct":   true,
		"price_change_amount": true,
		"volume_spike":       true,
		"unusual_volume":     true,
		"volume_above":       true,
		"volume_below":       true,
		"news":               true,
		"earnings":           true,
		"dividend":           true,
		"sec_filing":         true,
		"analyst_rating":     true,
	}

	if !validTypes[req.AlertType] {
		return nil, errors.New("invalid alert type")
	}

	// Validate frequency
	if req.Frequency != "once" && req.Frequency != "daily" && req.Frequency != "always" {
		return nil, errors.New("invalid frequency: must be 'once', 'daily', or 'always'")
	}

	// Validate conditions JSON
	var conditionsMap map[string]interface{}
	if err := json.Unmarshal(req.Conditions, &conditionsMap); err != nil {
		return nil, errors.New("invalid conditions format")
	}

	// Create alert rule
	alert := &models.AlertRule{
		UserID:          userID,
		WatchListID:     req.WatchListID,
		Symbol:          req.Symbol,
		AlertType:       req.AlertType,
		Conditions:      req.Conditions,
		Name:            req.Name,
		Description:     req.Description,
		Frequency:       req.Frequency,
		NotifyEmail:     req.NotifyEmail,
		NotifyInApp:     req.NotifyInApp,
		IsActive:        true,
	}

	if err := database.CreateAlertRule(alert); err != nil {
		return nil, err
	}

	return alert, nil
}

// GetUserAlerts retrieves all alert rules for a user
func (s *AlertService) GetUserAlerts(userID string, watchListID string, isActive string) ([]models.AlertRuleWithDetails, error) {
	return database.GetAlertRulesByUserID(userID, watchListID, isActive)
}

// GetAlertByID retrieves a single alert rule
func (s *AlertService) GetAlertByID(alertID string, userID string) (*models.AlertRule, error) {
	return database.GetAlertRuleByID(alertID, userID)
}

// UpdateAlert updates an existing alert rule
func (s *AlertService) UpdateAlert(alertID string, userID string, req *models.UpdateAlertRuleRequest) (*models.AlertRule, error) {
	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Conditions != nil {
		// Validate conditions JSON
		var conditionsMap map[string]interface{}
		if err := json.Unmarshal(req.Conditions, &conditionsMap); err != nil {
			return nil, errors.New("invalid conditions format")
		}
		updates["conditions"] = req.Conditions
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.Frequency != nil {
		if *req.Frequency != "once" && *req.Frequency != "daily" && *req.Frequency != "always" {
			return nil, errors.New("invalid frequency: must be 'once', 'daily', or 'always'")
		}
		updates["frequency"] = *req.Frequency
	}
	if req.NotifyEmail != nil {
		updates["notify_email"] = *req.NotifyEmail
	}
	if req.NotifyInApp != nil {
		updates["notify_in_app"] = *req.NotifyInApp
	}

	if err := database.UpdateAlertRule(alertID, userID, updates); err != nil {
		return nil, err
	}

	return database.GetAlertRuleByID(alertID, userID)
}

// DeleteAlert deletes an alert rule
func (s *AlertService) DeleteAlert(alertID string, userID string) error {
	return database.DeleteAlertRule(alertID, userID)
}

// ValidateWatchListOwnership checks if user owns the watch list
func (s *AlertService) ValidateWatchListOwnership(userID string, watchListID string) error {
	watchList, err := database.GetWatchListByID(watchListID, userID)
	if err != nil {
		return errors.New("watch list not found")
	}
	if watchList.UserID != userID {
		return errors.New("unauthorized")
	}
	return nil
}

// CanCreateAlert checks if user can create more alerts based on their subscription
func (s *AlertService) CanCreateAlert(userID string) (bool, error) {
	// Get user's subscription limits
	limits, err := database.GetUserSubscriptionLimits(userID)
	if err != nil {
		// If no subscription found, use free tier limits
		limits = &models.SubscriptionLimits{
			MaxAlertRules: 10,
		}
	}

	// Count existing alerts
	count, err := database.CountAlertRulesByUserID(userID)
	if err != nil {
		return false, err
	}

	// Check if limit is unlimited (-1) or if count is below limit
	if limits.MaxAlertRules == -1 {
		return true, nil
	}

	return count < limits.MaxAlertRules, nil
}

// GetAlertLogs retrieves alert history
func (s *AlertService) GetAlertLogs(userID string, alertID string, symbol string, limit int, offset int) ([]models.AlertLogWithRule, error) {
	if limit == 0 {
		limit = 50
	}
	return database.GetAlertLogsByUserID(userID, alertID, symbol, limit, offset)
}

// MarkAlertLogAsRead marks an alert log as read
func (s *AlertService) MarkAlertLogAsRead(logID string, userID string) error {
	return database.MarkAlertLogAsRead(logID, userID)
}

// MarkAlertLogAsDismissed marks an alert log as dismissed
func (s *AlertService) MarkAlertLogAsDismissed(logID string, userID string) error {
	return database.MarkAlertLogAsDismissed(logID, userID)
}

// GetActiveAlertRules retrieves all active alerts for processing
func (s *AlertService) GetActiveAlertRules() ([]models.AlertRule, error) {
	return database.GetActiveAlertRules()
}

// TriggerAlert creates an alert log and sends notifications
func (s *AlertService) TriggerAlert(alert *models.AlertRule, conditionMet interface{}, marketData interface{}) error {
	// Marshal condition and market data to JSON
	conditionJSON, err := json.Marshal(conditionMet)
	if err != nil {
		return fmt.Errorf("failed to marshal condition: %w", err)
	}

	marketDataJSON, err := json.Marshal(marketData)
	if err != nil {
		return fmt.Errorf("failed to marshal market data: %w", err)
	}

	// Create alert log
	log := &models.AlertLog{
		AlertRuleID:      alert.ID,
		UserID:           alert.UserID,
		Symbol:           alert.Symbol,
		AlertType:        alert.AlertType,
		ConditionMet:     conditionJSON,
		MarketData:       marketDataJSON,
		NotificationSent: false,
	}

	if err := database.CreateAlertLog(log); err != nil {
		return err
	}

	// Update alert rule trigger count
	if err := database.UpdateAlertRuleTrigger(alert.ID); err != nil {
		return err
	}

	// Check if alert should be disabled after triggering (frequency = "once")
	if alert.Frequency == "once" {
		updates := map[string]interface{}{
			"is_active": false,
		}
		if err := database.UpdateAlertRule(alert.ID, alert.UserID, updates); err != nil {
			return err
		}
	}

	return nil
}
