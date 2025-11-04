package services

import (
	"encoding/json"
	"fmt"
	"investorcenter-api/models"
	"log"
	"math"
)

type AlertProcessor struct {
	alertService        *AlertService
	notificationService *NotificationService
	polygonService      *PolygonService
}

func NewAlertProcessor(alertService *AlertService, notificationService *NotificationService, polygonService *PolygonService) *AlertProcessor {
	return &AlertProcessor{
		alertService:        alertService,
		notificationService: notificationService,
		polygonService:      polygonService,
	}
}

// ProcessAllAlerts processes all active alert rules
func (ap *AlertProcessor) ProcessAllAlerts() error {
	log.Println("Starting alert processing...")

	// Get all active alert rules
	alerts, err := ap.alertService.GetActiveAlertRules()
	if err != nil {
		return fmt.Errorf("failed to get active alerts: %w", err)
	}

	log.Printf("Found %d active alerts to process\n", len(alerts))

	// Process each alert
	for _, alert := range alerts {
		if err := ap.ProcessAlert(&alert); err != nil {
			log.Printf("Error processing alert %s: %v\n", alert.ID, err)
			continue
		}
	}

	log.Println("Alert processing completed")
	return nil
}

// ProcessAlert evaluates a single alert rule
func (ap *AlertProcessor) ProcessAlert(alert *models.AlertRule) error {
	// Get current market data for the symbol
	quote, err := ap.polygonService.GetLatestQuote(alert.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get quote for %s: %w", alert.Symbol, err)
	}

	// Evaluate alert based on type
	conditionMet, err := ap.evaluateAlert(alert, quote)
	if err != nil {
		return fmt.Errorf("failed to evaluate alert: %w", err)
	}

	if !conditionMet {
		return nil // Alert condition not met, skip
	}

	log.Printf("Alert triggered: %s (%s) for symbol %s\n", alert.Name, alert.AlertType, alert.Symbol)

	// Prepare condition met data
	conditionMetData := map[string]interface{}{
		"alert_type": alert.AlertType,
		"symbol":     alert.Symbol,
		"triggered":  true,
	}

	// Prepare market data snapshot
	marketData := map[string]interface{}{
		"symbol":     quote.Symbol,
		"price":      quote.Price,
		"volume":     quote.Volume,
		"timestamp":  quote.Updated,
	}

	// Trigger the alert (creates log, sends notifications)
	if err := ap.alertService.TriggerAlert(alert, conditionMetData, marketData); err != nil {
		return fmt.Errorf("failed to trigger alert: %w", err)
	}

	// Send notifications if enabled
	if err := ap.sendNotifications(alert, conditionMetData, marketData); err != nil {
		log.Printf("Error sending notifications for alert %s: %v\n", alert.ID, err)
	}

	return nil
}

// evaluateAlert checks if alert conditions are met
func (ap *AlertProcessor) evaluateAlert(alert *models.AlertRule, quote *Quote) (bool, error) {
	switch alert.AlertType {
	case "price_above":
		return ap.evaluatePriceAbove(alert, quote)
	case "price_below":
		return ap.evaluatePriceBelow(alert, quote)
	case "price_change_pct":
		return ap.evaluatePriceChangePct(alert, quote)
	case "volume_above":
		return ap.evaluateVolumeAbove(alert, quote)
	case "volume_below":
		return ap.evaluateVolumeBelow(alert, quote)
	case "volume_spike":
		return ap.evaluateVolumeSpike(alert, quote)
	default:
		return false, fmt.Errorf("unsupported alert type: %s", alert.AlertType)
	}
}

// evaluatePriceAbove checks if price is above threshold
func (ap *AlertProcessor) evaluatePriceAbove(alert *models.AlertRule, quote *Quote) (bool, error) {
	var condition models.PriceAboveCondition
	if err := json.Unmarshal(alert.Conditions, &condition); err != nil {
		return false, err
	}

	return quote.Price >= condition.Threshold, nil
}

// evaluatePriceBelow checks if price is below threshold
func (ap *AlertProcessor) evaluatePriceBelow(alert *models.AlertRule, quote *Quote) (bool, error) {
	var condition models.PriceAboveCondition
	if err := json.Unmarshal(alert.Conditions, &condition); err != nil {
		return false, err
	}

	return quote.Price <= condition.Threshold, nil
}

// evaluatePriceChangePct checks if price changed by percentage
func (ap *AlertProcessor) evaluatePriceChangePct(alert *models.AlertRule, quote *Quote) (bool, error) {
	var condition models.PriceChangeCondition
	if err := json.Unmarshal(alert.Conditions, &condition); err != nil {
		return false, err
	}

	// Get previous day's close price
	prevClose, err := ap.polygonService.GetPreviousClose(alert.Symbol)
	if err != nil {
		return false, err
	}

	// Calculate percentage change
	changePct := ((quote.Price - prevClose) / prevClose) * 100

	switch condition.Direction {
	case "up":
		return changePct >= condition.PercentChange, nil
	case "down":
		return changePct <= -condition.PercentChange, nil
	case "either":
		return math.Abs(changePct) >= condition.PercentChange, nil
	default:
		return false, fmt.Errorf("invalid direction: %s", condition.Direction)
	}
}

// evaluateVolumeAbove checks if volume is above threshold
func (ap *AlertProcessor) evaluateVolumeAbove(alert *models.AlertRule, quote *Quote) (bool, error) {
	var condition models.VolumeCondition
	if err := json.Unmarshal(alert.Conditions, &condition); err != nil {
		return false, err
	}

	return float64(quote.Volume) >= condition.Threshold, nil
}

// evaluateVolumeBelow checks if volume is below threshold
func (ap *AlertProcessor) evaluateVolumeBelow(alert *models.AlertRule, quote *Quote) (bool, error) {
	var condition models.VolumeCondition
	if err := json.Unmarshal(alert.Conditions, &condition); err != nil {
		return false, err
	}

	return float64(quote.Volume) <= condition.Threshold, nil
}

// evaluateVolumeSpike checks if volume is significantly higher than average
func (ap *AlertProcessor) evaluateVolumeSpike(alert *models.AlertRule, quote *Quote) (bool, error) {
	var condition models.VolumeSpikeCondition
	if err := json.Unmarshal(alert.Conditions, &condition); err != nil {
		return false, err
	}

	// Get average volume (this is simplified - in production, calculate actual average)
	avgVolume, err := ap.polygonService.GetAverageVolume(alert.Symbol, 30)
	if err != nil {
		return false, err
	}

	threshold := avgVolume * condition.VolumeMultiplier
	return float64(quote.Volume) >= threshold, nil
}

// sendNotifications sends email and in-app notifications
func (ap *AlertProcessor) sendNotifications(alert *models.AlertRule, conditionMet interface{}, marketData interface{}) error {
	// Send in-app notification
	if alert.NotifyInApp {
		title := fmt.Sprintf("Alert: %s", alert.Name)
		message := fmt.Sprintf("Your alert for %s has been triggered", alert.Symbol)

		if err := ap.notificationService.CreateInAppNotification(
			alert.UserID,
			nil,
			"alert_triggered",
			title,
			message,
			marketData,
		); err != nil {
			log.Printf("Failed to create in-app notification: %v\n", err)
		}
	}

	// Send email notification
	if alert.NotifyEmail {
		// Check quiet hours
		inQuietHours, err := ap.notificationService.IsInQuietHours(alert.UserID)
		if err != nil {
			log.Printf("Failed to check quiet hours: %v\n", err)
		} else if !inQuietHours {
			if err := ap.notificationService.SendAlertEmail(alert.UserID, alert, conditionMet, marketData); err != nil {
				log.Printf("Failed to send alert email: %v\n", err)
			}
		}
	}

	return nil
}
