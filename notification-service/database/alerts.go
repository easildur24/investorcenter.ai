package database

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"notification-service/models"
)

// GetActiveAlertsForSymbols fetches all active alert rules whose symbol
// is in the given set. Returns an empty slice if no matches.
func (db *DB) GetActiveAlertsForSymbols(symbols []string) ([]models.AlertRule, error) {
	if len(symbols) == 0 {
		return nil, nil
	}

	// Build placeholders: $1, $2, $3, ...
	placeholders := make([]string, len(symbols))
	args := make([]interface{}, len(symbols))
	for i, s := range symbols {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = s
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, watch_list_id, symbol, alert_type, conditions,
		       is_active, frequency, notify_email, notify_in_app, name,
		       last_triggered_at, trigger_count, created_at, updated_at
		FROM alert_rules
		WHERE is_active = true AND symbol IN (%s)
		ORDER BY created_at ASC
	`, strings.Join(placeholders, ", "))

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query active alerts: %w", err)
	}
	defer rows.Close()

	var alerts []models.AlertRule
	for rows.Next() {
		var a models.AlertRule
		if err := rows.Scan(
			&a.ID, &a.UserID, &a.WatchListID, &a.Symbol, &a.AlertType, &a.Conditions,
			&a.IsActive, &a.Frequency, &a.NotifyEmail, &a.NotifyInApp, &a.Name,
			&a.LastTriggeredAt, &a.TriggerCount, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		alerts = append(alerts, a)
	}

	return alerts, rows.Err()
}

// CreateAlertLog inserts a new alert trigger log and returns the generated ID.
func (db *DB) CreateAlertLog(alertLog *models.AlertLog) (string, error) {
	conditionMet, err := json.Marshal(alertLog.ConditionMet)
	if err != nil {
		log.Printf("Warning: failed to marshal condition_met for alert %s: %v", alertLog.AlertRuleID, err)
		conditionMet = []byte("{}")
	}
	marketData, err := json.Marshal(alertLog.MarketData)
	if err != nil {
		log.Printf("Warning: failed to marshal market_data for alert %s: %v", alertLog.AlertRuleID, err)
		marketData = []byte("{}")
	}

	var id string
	err = db.QueryRow(`
		INSERT INTO alert_logs (alert_rule_id, user_id, symbol, alert_type,
		                        condition_met, market_data, notification_sent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`,
		alertLog.AlertRuleID, alertLog.UserID, alertLog.Symbol, alertLog.AlertType,
		conditionMet, marketData, alertLog.NotificationSent,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("create alert log: %w", err)
	}
	return id, nil
}

// ClaimAlertTrigger atomically claims an alert for triggering using a conditional
// UPDATE. This prevents race conditions where multiple consumers could trigger
// the same alert simultaneously.
//
// For "once" alerts: only claims if last_triggered_at IS NULL (never triggered).
// For "daily" alerts: only claims if last_triggered_at is >24h ago or NULL.
// For "always" alerts: only claims if last_triggered_at is >5min ago or NULL.
//
// Returns true if the alert was successfully claimed (row was updated).
func (db *DB) ClaimAlertTrigger(alertID string, frequency string) (bool, error) {
	var query string

	switch frequency {
	case "once":
		// Atomically claim: only if never triggered, also deactivate
		query = `
			UPDATE alert_rules
			SET last_triggered_at = NOW(),
			    trigger_count = trigger_count + 1,
			    updated_at = NOW(),
			    is_active = false
			WHERE id = $1 AND last_triggered_at IS NULL
		`
	case "daily":
		// Atomically claim: only if >24h since last trigger or never triggered
		query = `
			UPDATE alert_rules
			SET last_triggered_at = NOW(),
			    trigger_count = trigger_count + 1,
			    updated_at = NOW()
			WHERE id = $1 AND (last_triggered_at IS NULL OR last_triggered_at < NOW() - INTERVAL '24 hours')
		`
	case "always":
		// Atomically claim: only if >5min since last trigger or never triggered
		query = `
			UPDATE alert_rules
			SET last_triggered_at = NOW(),
			    trigger_count = trigger_count + 1,
			    updated_at = NOW()
			WHERE id = $1 AND (last_triggered_at IS NULL OR last_triggered_at < NOW() - INTERVAL '5 minutes')
		`
	default:
		return false, fmt.Errorf("unknown frequency: %s", frequency)
	}

	result, err := db.Exec(query, alertID)
	if err != nil {
		return false, fmt.Errorf("claim alert trigger: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("check rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

// UpdateAlertLogNotificationSent updates the notification_sent flag on an alert log.
func (db *DB) UpdateAlertLogNotificationSent(logID string, sent bool) error {
	_, err := db.Exec(`UPDATE alert_logs SET notification_sent = $1 WHERE id = $2`, sent, logID)
	if err != nil {
		return fmt.Errorf("update alert log notification_sent: %w", err)
	}
	return nil
}

// GetTodayAlertCount returns the number of alerts triggered today for a user.
func (db *DB) GetTodayAlertCount(userID string) (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM alert_logs
		WHERE user_id = $1 AND triggered_at >= $2
	`, userID, todayStart()).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get today alert count: %w", err)
	}
	return count, nil
}

// GetTodayEmailCount returns the number of alert emails sent today for a user.
func (db *DB) GetTodayEmailCount(userID string) (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM alert_logs
		WHERE user_id = $1 AND triggered_at >= $2 AND notification_sent = true
	`, userID, todayStart()).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get today email count: %w", err)
	}
	return count, nil
}

// todayStart returns the start of the current UTC day.
func todayStart() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}
