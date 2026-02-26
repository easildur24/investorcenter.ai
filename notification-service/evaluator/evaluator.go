package evaluator

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"notification-service/database"
	"notification-service/delivery"
	"notification-service/models"
)

// Evaluator processes price update messages and triggers matching alert rules.
type Evaluator struct {
	db       *database.DB
	delivery *delivery.Router
}

// New creates a new Evaluator.
func New(db *database.DB, delivery *delivery.Router) *Evaluator {
	return &Evaluator{db: db, delivery: delivery}
}

// HandlePriceUpdate processes a single SNS price update message.
// It parses the message, queries matching alerts, evaluates conditions,
// and delivers notifications for triggered alerts.
func (e *Evaluator) HandlePriceUpdate(msg []byte) error {
	var update models.PriceUpdateMessage
	if err := json.Unmarshal(msg, &update); err != nil {
		return fmt.Errorf("parse price update: %w", err)
	}

	if len(update.Symbols) == 0 {
		return nil
	}

	// Extract symbol list for DB query
	symbols := make([]string, 0, len(update.Symbols))
	for symbol := range update.Symbols {
		symbols = append(symbols, symbol)
	}

	// Query only alerts for symbols in this update
	alerts, err := e.db.GetActiveAlertsForSymbols(symbols)
	if err != nil {
		return fmt.Errorf("fetch alerts: %w", err)
	}

	if len(alerts) == 0 {
		return nil // No alerts to evaluate — fast path
	}

	var triggered int
	for i := range alerts {
		alert := &alerts[i]
		quote, exists := update.Symbols[alert.Symbol]
		if !exists {
			continue
		}

		// Pre-check frequency gating (fast path to avoid DB round-trip for
		// alerts that clearly shouldn't fire). The actual atomic claim happens
		// in trigger() via ClaimAlertTrigger.
		if !shouldTriggerBasedOnFrequency(alert) {
			continue
		}

		// Evaluate the alert condition
		conditionMet, err := evaluate(alert, &quote)
		if err != nil {
			log.Printf("Error evaluating alert %s: %v", alert.ID, err)
			continue
		}
		if !conditionMet {
			continue
		}

		// Alert triggered — log, update, and deliver
		if err := e.trigger(alert, &quote); err != nil {
			log.Printf("Error triggering alert %s: %v", alert.ID, err)
			// Continue processing other alerts
		} else {
			triggered++
		}
	}

	if triggered > 0 {
		log.Printf("Evaluated %d alerts, triggered %d", len(alerts), triggered)
	}

	return nil
}

// trigger handles a single triggered alert: atomically claims the trigger slot,
// creates a log entry, and delivers notifications.
func (e *Evaluator) trigger(alert *models.AlertRule, quote *models.SymbolQuote) error {
	// Atomically claim the trigger slot in the DB. This prevents race conditions
	// where multiple consumers (or a future multi-replica setup) could trigger
	// the same alert simultaneously. The UPDATE uses a WHERE clause that checks
	// frequency constraints, so only one caller wins the claim.
	claimed, err := e.db.ClaimAlertTrigger(alert.ID, alert.Frequency)
	if err != nil {
		return fmt.Errorf("claim alert trigger: %w", err)
	}
	if !claimed {
		// Another consumer already triggered this alert, or the frequency
		// constraint was not met at the DB level. Skip silently.
		return nil
	}

	// Build condition_met and market_data JSON
	conditionMet, err := json.Marshal(map[string]interface{}{
		"alert_type": alert.AlertType,
		"threshold":  getThreshold(alert),
		"triggered":  true,
	})
	if err != nil {
		log.Printf("Warning: failed to marshal condition_met for alert %s: %v", alert.ID, err)
		conditionMet = []byte(`{"triggered":true}`)
	}

	marketData, err := json.Marshal(map[string]interface{}{
		"symbol":     alert.Symbol,
		"price":      quote.Price,
		"volume":     quote.Volume,
		"change_pct": quote.ChangePct,
		"timestamp":  time.Now().Unix(),
	})
	if err != nil {
		log.Printf("Warning: failed to marshal market_data for alert %s: %v", alert.ID, err)
		marketData = []byte(`{}`)
	}

	// 1. Create alert log with notification_sent=false initially.
	//    It will be set to true only after successful delivery.
	alertLog := &models.AlertLog{
		AlertRuleID:      alert.ID,
		UserID:           alert.UserID,
		Symbol:           alert.Symbol,
		AlertType:        alert.AlertType,
		ConditionMet:     conditionMet,
		MarketData:       marketData,
		NotificationSent: false,
	}
	logID, err := e.db.CreateAlertLog(alertLog)
	if err != nil {
		return fmt.Errorf("create alert log: %w", err)
	}
	alertLog.ID = logID

	// 2. Deliver notifications (in-app + email)
	deliveryErr := e.delivery.Deliver(alert, alertLog, quote)
	if deliveryErr != nil {
		log.Printf("Delivery error for alert %s: %v", alert.ID, deliveryErr)
		// Don't return error — the alert was still triggered successfully.
		// notification_sent remains false to reflect failed delivery.
	} else {
		// Mark notification as successfully sent
		if err := e.db.UpdateAlertLogNotificationSent(logID, true); err != nil {
			log.Printf("Warning: failed to update notification_sent for log %s: %v", logID, err)
		}
	}

	return nil
}

// getThreshold extracts the numeric threshold from an alert's conditions JSON.
func getThreshold(alert *models.AlertRule) float64 {
	var cond models.ThresholdCondition
	if err := json.Unmarshal(alert.Conditions, &cond); err == nil && cond.Threshold > 0 {
		return cond.Threshold
	}
	var spike models.VolumeSpikeCondition
	if err := json.Unmarshal(alert.Conditions, &spike); err == nil && spike.VolumeMultiplier > 0 {
		return spike.VolumeMultiplier
	}
	return 0
}

// shouldTriggerBasedOnFrequency performs a fast in-memory check of whether an
// alert should fire based on its frequency setting and when it last triggered.
//
// NOTE: This is a pre-filter only. The actual atomic claim happens in
// ClaimAlertTrigger at the DB level, which prevents race conditions.
func shouldTriggerBasedOnFrequency(alert *models.AlertRule) bool {
	switch alert.Frequency {
	case "once":
		// Only trigger if never triggered before
		return alert.LastTriggeredAt == nil
	case "daily":
		// Trigger if no previous trigger or >24h since last trigger
		if alert.LastTriggeredAt == nil {
			return true
		}
		return time.Since(*alert.LastTriggeredAt) >= 24*time.Hour
	case "always":
		// Trigger on every evaluation cycle, but with a 5-minute cooldown
		// to prevent notification spam. Users selecting "always" will receive
		// at most one notification per 5 minutes per alert rule.
		if alert.LastTriggeredAt == nil {
			return true
		}
		return time.Since(*alert.LastTriggeredAt) >= 5*time.Minute
	default:
		return false
	}
}

// evaluate dispatches to the appropriate evaluator based on alert type.
func evaluate(alert *models.AlertRule, quote *models.SymbolQuote) (bool, error) {
	switch alert.AlertType {
	case "price_above":
		return evaluatePriceAbove(alert, quote)
	case "price_below":
		return evaluatePriceBelow(alert, quote)
	case "price_change_pct":
		return evaluatePriceChangePct(alert, quote)
	case "volume_above":
		return evaluateVolumeAbove(alert, quote)
	case "volume_below":
		return evaluateVolumeBelow(alert, quote)
	// volume_spike, news, earnings — not yet implemented
	default:
		return false, nil
	}
}
