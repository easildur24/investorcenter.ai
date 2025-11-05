package services

import (
	"encoding/json"
	"fmt"
	"investorcenter-api/models"
	"log"
)

// Quote represents market data for alert evaluation
type Quote struct {
	Symbol    string
	Price     float64
	Volume    int64
	Timestamp int64
	Updated   int64
}

type AlertProcessor struct {
	alertService        *AlertService
	notificationService *NotificationService
	polygonService      *PolygonClient
}

func NewAlertProcessor(alertService *AlertService, notificationService *NotificationService, polygonService *PolygonClient) *AlertProcessor {
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

	if len(alerts) == 0 {
		log.Println("No active alerts to process")
		return nil
	}

	// Batch fetch all symbols (fixes N+1 query problem)
	symbols := make([]string, 0, len(alerts))
	symbolToAlerts := make(map[string][]*models.AlertRule)

	for i := range alerts {
		symbol := alerts[i].Symbol
		// Skip if already added
		if _, exists := symbolToAlerts[symbol]; !exists {
			symbols = append(symbols, symbol)
		}
		symbolToAlerts[symbol] = append(symbolToAlerts[symbol], &alerts[i])
	}

	log.Printf("Fetching quotes for %d unique symbols...\n", len(symbols))

	// Fetch all quotes in one batch call
	quotes, err := ap.polygonService.GetMultipleQuotes(symbols)
	if err != nil {
		return fmt.Errorf("failed to get bulk quotes: %w", err)
	}

	log.Printf("Received %d quotes, processing alerts...\n", len(quotes))

	// Process alerts with their quotes
	processedCount := 0
	triggeredCount := 0

	for symbol, alertList := range symbolToAlerts {
		quote, exists := quotes[symbol]
		if !exists {
			log.Printf("Warning: No quote found for symbol %s, skipping %d alerts\n", symbol, len(alertList))
			continue
		}

		// Convert QuoteData to Quote for compatibility
		alertQuote := &Quote{
			Symbol:    quote.Symbol,
			Price:     quote.Price,
			Volume:    quote.Volume,
			Timestamp: quote.Timestamp,
			Updated:   quote.Timestamp,
		}

		for _, alert := range alertList {
			processedCount++
			triggered, err := ap.ProcessAlertWithQuote(alert, alertQuote)
			if err != nil {
				log.Printf("Error processing alert %s: %v\n", alert.ID, err)
				continue
			}
			if triggered {
				triggeredCount++
			}
		}
	}

	log.Printf("Alert processing completed: %d processed, %d triggered\n", processedCount, triggeredCount)
	return nil
}

// ProcessAlert evaluates a single alert rule (legacy method for backward compatibility)
// For batch processing, use ProcessAllAlerts() which is more efficient
// Note: This method is less efficient as it makes individual API calls
func (ap *AlertProcessor) ProcessAlert(alert *models.AlertRule) error {
	// Use batch method with single symbol for compatibility
	quotes, err := ap.polygonService.GetMultipleQuotes([]string{alert.Symbol})
	if err != nil {
		return fmt.Errorf("failed to get quote for %s: %w", alert.Symbol, err)
	}

	quote, exists := quotes[alert.Symbol]
	if !exists {
		return fmt.Errorf("no quote found for symbol %s", alert.Symbol)
	}

	// Convert to Quote type
	alertQuote := &Quote{
		Symbol:    quote.Symbol,
		Price:     quote.Price,
		Volume:    quote.Volume,
		Timestamp: quote.Timestamp,
		Updated:   quote.Timestamp,
	}

	_, err = ap.ProcessAlertWithQuote(alert, alertQuote)
	return err
}

// ProcessAlertWithQuote evaluates a single alert rule with a pre-fetched quote
// Returns (triggered bool, error)
func (ap *AlertProcessor) ProcessAlertWithQuote(alert *models.AlertRule, quote *Quote) (bool, error) {
	// Check if alert should trigger based on frequency
	if !ap.alertService.ShouldTriggerBasedOnFrequency(alert) {
		log.Printf("Skipping alert %s - frequency restriction (last triggered: %v)\n", alert.ID, alert.LastTriggeredAt)
		return false, nil
	}

	// Evaluate alert based on type
	conditionMet, err := ap.evaluateAlert(alert, quote)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate alert: %w", err)
	}

	if !conditionMet {
		return false, nil // Alert condition not met, skip
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
		"symbol":    quote.Symbol,
		"price":     quote.Price,
		"volume":    quote.Volume,
		"timestamp": quote.Updated,
	}

	// Trigger the alert (creates log, sends notifications)
	if err := ap.alertService.TriggerAlert(alert, conditionMetData, marketData); err != nil {
		return false, fmt.Errorf("failed to trigger alert: %w", err)
	}

	// Send notifications if enabled
	if err := ap.sendNotifications(alert, conditionMetData, marketData); err != nil {
		log.Printf("Error sending notifications for alert %s: %v\n", alert.ID, err)
	}

	return true, nil
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

	// TODO: Implement historical price fetching for percentage change calculation
	// For now, this alert type is not fully implemented
	log.Printf("Warning: price_change_pct alert type not yet fully implemented for %s\n", alert.Symbol)
	return false, fmt.Errorf("price_change_pct alert type not yet implemented")
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

	// TODO: Implement historical volume fetching for spike detection
	// For now, this alert type is not fully implemented
	log.Printf("Warning: volume_spike alert type not yet fully implemented for %s\n", alert.Symbol)
	return false, fmt.Errorf("volume_spike alert type not yet implemented")
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
