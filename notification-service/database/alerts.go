package database

import (
	"encoding/json"
	"fmt"
	"strings"

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
func (db *DB) CreateAlertLog(log *models.AlertLog) (string, error) {
	conditionMet, _ := json.Marshal(log.ConditionMet)
	marketData, _ := json.Marshal(log.MarketData)

	var id string
	err := db.QueryRow(`
		INSERT INTO alert_logs (alert_rule_id, user_id, symbol, alert_type,
		                        condition_met, market_data, notification_sent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`,
		log.AlertRuleID, log.UserID, log.Symbol, log.AlertType,
		conditionMet, marketData, log.NotificationSent,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("create alert log: %w", err)
	}
	return id, nil
}

// UpdateAlertTrigger atomically increments the trigger count and sets
// last_triggered_at. If frequency is "once", also deactivates the rule.
func (db *DB) UpdateAlertTrigger(alertID string, frequency string) error {
	query := `
		UPDATE alert_rules
		SET last_triggered_at = NOW(),
		    trigger_count = trigger_count + 1,
		    updated_at = NOW()
	`
	if frequency == "once" {
		query += `, is_active = false`
	}
	query += ` WHERE id = $1`

	_, err := db.Exec(query, alertID)
	if err != nil {
		return fmt.Errorf("update alert trigger: %w", err)
	}
	return nil
}
