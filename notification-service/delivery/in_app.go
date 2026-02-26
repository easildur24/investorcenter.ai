package delivery

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"notification-service/database"
	"notification-service/models"
)

// InAppDelivery writes notifications to the notification_queue table,
// which the frontend polls via the NotificationDropdown component.
type InAppDelivery struct {
	db *database.DB
}

// NewInAppDelivery creates a new InAppDelivery.
func NewInAppDelivery(db *database.DB) *InAppDelivery {
	return &InAppDelivery{db: db}
}

// Send creates an in-app notification for the alert trigger.
func (d *InAppDelivery) Send(alert *models.AlertRule, alertLog *models.AlertLog, quote *models.SymbolQuote) error {
	// Check daily alert rate limit
	prefs, err := d.db.GetNotificationPreferences(alert.UserID)
	if err != nil {
		log.Printf("Warning: failed to get preferences for rate limit check: %v", err)
		// Continue — don't block delivery on preference lookup failure
	}
	if prefs != nil && prefs.MaxAlertsPerDay > 0 {
		count, err := d.db.GetTodayAlertCount(alert.UserID)
		if err != nil {
			log.Printf("Warning: failed to get today's alert count: %v", err)
		} else if count >= prefs.MaxAlertsPerDay {
			log.Printf("Skipping in-app notification for alert %s — user %s exceeded daily limit (%d/%d)",
				alert.ID, alert.UserID, count, prefs.MaxAlertsPerDay)
			return nil
		}
	}

	title := buildTitle(alert, quote)
	message := buildMessage(alert, quote)
	data := buildData(alert, quote)

	dataJSON, err := json.Marshal(data)
	if err != nil {
		log.Printf("Warning: failed to marshal notification data for alert %s: %v", alert.ID, err)
		dataJSON = []byte("{}")
	}

	notification := &models.InAppNotification{
		UserID:     alert.UserID,
		AlertLogID: &alertLog.ID,
		Type:       "alert_triggered",
		Title:      title,
		Message:    message,
		Data:       dataJSON,
	}

	if err := d.db.CreateInAppNotification(notification); err != nil {
		return fmt.Errorf("create in-app notification: %w", err)
	}

	log.Printf("In-app notification created for alert %s (%s %s)", alert.ID, alert.Symbol, alert.AlertType)
	return nil
}

// buildTitle generates a human-readable notification title.
func buildTitle(alert *models.AlertRule, quote *models.SymbolQuote) string {
	typeLabel := alertTypeLabel(alert.AlertType)
	return fmt.Sprintf("%s %s", alert.Symbol, typeLabel)
}

// buildMessage generates a descriptive notification message.
func buildMessage(alert *models.AlertRule, quote *models.SymbolQuote) string {
	switch alert.AlertType {
	case "price_above":
		var cond models.ThresholdCondition
		if err := json.Unmarshal(alert.Conditions, &cond); err != nil {
			log.Printf("Warning: failed to parse price_above conditions: %v", err)
			return fmt.Sprintf("Alert triggered for %s", alert.Symbol)
		}
		return fmt.Sprintf("%s crossed above $%.2f (current: $%.2f)", alert.Symbol, cond.Threshold, quote.Price)
	case "price_below":
		var cond models.ThresholdCondition
		if err := json.Unmarshal(alert.Conditions, &cond); err != nil {
			log.Printf("Warning: failed to parse price_below conditions: %v", err)
			return fmt.Sprintf("Alert triggered for %s", alert.Symbol)
		}
		return fmt.Sprintf("%s dropped below $%.2f (current: $%.2f)", alert.Symbol, cond.Threshold, quote.Price)
	case "volume_above":
		var cond models.ThresholdCondition
		if err := json.Unmarshal(alert.Conditions, &cond); err != nil {
			log.Printf("Warning: failed to parse volume_above conditions: %v", err)
			return fmt.Sprintf("Alert triggered for %s", alert.Symbol)
		}
		return fmt.Sprintf("%s volume exceeded %s (current: %s)", alert.Symbol, formatVolume(cond.Threshold), formatVolume(float64(quote.Volume)))
	case "volume_below":
		var cond models.ThresholdCondition
		if err := json.Unmarshal(alert.Conditions, &cond); err != nil {
			log.Printf("Warning: failed to parse volume_below conditions: %v", err)
			return fmt.Sprintf("Alert triggered for %s", alert.Symbol)
		}
		return fmt.Sprintf("%s volume dropped below %s (current: %s)", alert.Symbol, formatVolume(cond.Threshold), formatVolume(float64(quote.Volume)))
	case "price_change_pct":
		return fmt.Sprintf("%s moved %.2f%% today", alert.Symbol, quote.ChangePct)
	default:
		return fmt.Sprintf("Alert triggered for %s", alert.Symbol)
	}
}

// buildData creates the metadata JSON stored with the notification.
// Includes watch_list_id for navigation in the frontend dropdown.
func buildData(alert *models.AlertRule, quote *models.SymbolQuote) map[string]interface{} {
	return map[string]interface{}{
		"watch_list_id": alert.WatchListID,
		"symbol":        alert.Symbol,
		"price":         quote.Price,
		"volume":        quote.Volume,
		"alert_type":    alert.AlertType,
	}
}

// alertTypeLabel returns a human-readable label for an alert type.
func alertTypeLabel(alertType string) string {
	labels := map[string]string{
		"price_above":      "Price Above",
		"price_below":      "Price Below",
		"price_change_pct": "Price Change %",
		"volume_above":     "Volume Above",
		"volume_below":     "Volume Below",
		"volume_spike":     "Volume Spike",
		"news":             "News Alert",
		"earnings":         "Earnings Report",
	}
	if label, ok := labels[alertType]; ok {
		return label
	}
	return strings.ReplaceAll(alertType, "_", " ")
}

// formatVolume formats a volume number with K/M/B suffixes.
func formatVolume(vol float64) string {
	switch {
	case vol >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", vol/1_000_000_000)
	case vol >= 1_000_000:
		return fmt.Sprintf("%.1fM", vol/1_000_000)
	case vol >= 1_000:
		return fmt.Sprintf("%.1fK", vol/1_000)
	default:
		return fmt.Sprintf("%.0f", vol)
	}
}
