package database

import (
	"database/sql"
	"errors"
	"fmt"
	"investorcenter-api/models"
)

// Alert Rule Operations

// CreateAlertRule creates a new alert rule
func CreateAlertRule(alert *models.AlertRule) error {
	query := `
		INSERT INTO alert_rules (
			user_id, watch_list_id, watch_list_item_id, symbol, alert_type,
			conditions, is_active, frequency, notify_email, notify_in_app,
			name, description
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at, trigger_count
	`
	err := DB.QueryRow(
		query,
		alert.UserID,
		alert.WatchListID,
		alert.WatchListItemID,
		alert.Symbol,
		alert.AlertType,
		alert.Conditions,
		alert.IsActive,
		alert.Frequency,
		alert.NotifyEmail,
		alert.NotifyInApp,
		alert.Name,
		alert.Description,
	).Scan(&alert.ID, &alert.CreatedAt, &alert.UpdatedAt, &alert.TriggerCount)

	if err != nil {
		return fmt.Errorf("failed to create alert rule: %w", err)
	}
	return nil
}

// GetAlertRuleByID retrieves a single alert rule by ID
func GetAlertRuleByID(alertID string, userID string) (*models.AlertRule, error) {
	query := `
		SELECT
			id, user_id, watch_list_id, watch_list_item_id, symbol, alert_type,
			conditions, is_active, frequency, notify_email, notify_in_app,
			name, description, last_triggered_at, trigger_count, created_at, updated_at
		FROM alert_rules
		WHERE id = $1 AND user_id = $2
	`
	alert := &models.AlertRule{}
	err := DB.QueryRow(query, alertID, userID).Scan(
		&alert.ID,
		&alert.UserID,
		&alert.WatchListID,
		&alert.WatchListItemID,
		&alert.Symbol,
		&alert.AlertType,
		&alert.Conditions,
		&alert.IsActive,
		&alert.Frequency,
		&alert.NotifyEmail,
		&alert.NotifyInApp,
		&alert.Name,
		&alert.Description,
		&alert.LastTriggeredAt,
		&alert.TriggerCount,
		&alert.CreatedAt,
		&alert.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("alert rule not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get alert rule: %w", err)
	}

	return alert, nil
}

// GetAlertRulesByUserID retrieves all alert rules for a user
func GetAlertRulesByUserID(userID string, watchListID string, isActive string) ([]models.AlertRuleWithDetails, error) {
	query := `
		SELECT
			ar.id, ar.user_id, ar.watch_list_id, ar.watch_list_item_id, ar.symbol,
			ar.alert_type, ar.conditions, ar.is_active, ar.frequency, ar.notify_email,
			ar.notify_in_app, ar.name, ar.description, ar.last_triggered_at,
			ar.trigger_count, ar.created_at, ar.updated_at,
			wl.name as watch_list_name,
			COALESCE(t.name, '') as company_name
		FROM alert_rules ar
		JOIN watch_lists wl ON ar.watch_list_id = wl.id
		LEFT JOIN tickers t ON ar.symbol = t.symbol
		WHERE ar.user_id = $1
	`
	args := []interface{}{userID}
	argCount := 1

	if watchListID != "" {
		argCount++
		query += fmt.Sprintf(" AND ar.watch_list_id = $%d", argCount)
		args = append(args, watchListID)
	}

	if isActive != "" {
		argCount++
		query += fmt.Sprintf(" AND ar.is_active = $%d", argCount)
		args = append(args, isActive == "true")
	}

	query += " ORDER BY ar.created_at DESC"

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert rules: %w", err)
	}
	defer rows.Close()

	alerts := []models.AlertRuleWithDetails{}
	for rows.Next() {
		var alert models.AlertRuleWithDetails
		err := rows.Scan(
			&alert.ID,
			&alert.UserID,
			&alert.WatchListID,
			&alert.WatchListItemID,
			&alert.Symbol,
			&alert.AlertType,
			&alert.Conditions,
			&alert.IsActive,
			&alert.Frequency,
			&alert.NotifyEmail,
			&alert.NotifyInApp,
			&alert.Name,
			&alert.Description,
			&alert.LastTriggeredAt,
			&alert.TriggerCount,
			&alert.CreatedAt,
			&alert.UpdatedAt,
			&alert.WatchListName,
			&alert.CompanyName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert rule: %w", err)
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// GetActiveAlertRules retrieves all active alert rules for processing
func GetActiveAlertRules() ([]models.AlertRule, error) {
	query := `
		SELECT
			id, user_id, watch_list_id, watch_list_item_id, symbol, alert_type,
			conditions, is_active, frequency, notify_email, notify_in_app,
			name, description, last_triggered_at, trigger_count, created_at, updated_at
		FROM alert_rules
		WHERE is_active = true
		ORDER BY created_at ASC
	`
	rows, err := DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active alert rules: %w", err)
	}
	defer rows.Close()

	alerts := []models.AlertRule{}
	for rows.Next() {
		var alert models.AlertRule
		err := rows.Scan(
			&alert.ID,
			&alert.UserID,
			&alert.WatchListID,
			&alert.WatchListItemID,
			&alert.Symbol,
			&alert.AlertType,
			&alert.Conditions,
			&alert.IsActive,
			&alert.Frequency,
			&alert.NotifyEmail,
			&alert.NotifyInApp,
			&alert.Name,
			&alert.Description,
			&alert.LastTriggeredAt,
			&alert.TriggerCount,
			&alert.CreatedAt,
			&alert.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert rule: %w", err)
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// UpdateAlertRule updates an existing alert rule
func UpdateAlertRule(alertID string, userID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return errors.New("no fields to update")
	}

	query := "UPDATE alert_rules SET "
	args := []interface{}{}
	argCount := 0

	for field, value := range updates {
		argCount++
		if argCount > 1 {
			query += ", "
		}
		query += fmt.Sprintf("%s = $%d", field, argCount)
		args = append(args, value)
	}

	argCount++
	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, alertID)

	argCount++
	query += fmt.Sprintf(" AND user_id = $%d", argCount)
	args = append(args, userID)

	result, err := DB.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update alert rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("alert rule not found")
	}

	return nil
}

// DeleteAlertRule deletes an alert rule
func DeleteAlertRule(alertID string, userID string) error {
	query := "DELETE FROM alert_rules WHERE id = $1 AND user_id = $2"
	result, err := DB.Exec(query, alertID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete alert rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("alert rule not found")
	}

	return nil
}

// UpdateAlertRuleTrigger updates the last triggered timestamp and count
func UpdateAlertRuleTrigger(alertID string) error {
	query := `
		UPDATE alert_rules
		SET last_triggered_at = CURRENT_TIMESTAMP,
		    trigger_count = trigger_count + 1
		WHERE id = $1
	`
	_, err := DB.Exec(query, alertID)
	if err != nil {
		return fmt.Errorf("failed to update alert rule trigger: %w", err)
	}
	return nil
}

// GetAlertForWatchListItems retrieves alerts for all tickers in a watchlist,
// returning a map keyed by symbol. Since alerts are 1:1 with watchlist items,
// each symbol has at most one alert. If legacy duplicates exist, the most
// recent (by created_at) wins.
func GetAlertForWatchListItems(watchListID string, userID string) (map[string]*models.AlertRule, error) {
	query := `
		SELECT
			id, user_id, watch_list_id, watch_list_item_id, symbol, alert_type,
			conditions, is_active, frequency, notify_email, notify_in_app,
			name, description, last_triggered_at, trigger_count, created_at, updated_at
		FROM alert_rules
		WHERE watch_list_id = $1 AND user_id = $2
		ORDER BY created_at DESC
	`
	rows, err := DB.Query(query, watchListID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get alerts for watchlist items: %w", err)
	}
	defer rows.Close()

	alertMap := make(map[string]*models.AlertRule)
	for rows.Next() {
		alert := &models.AlertRule{}
		err := rows.Scan(
			&alert.ID,
			&alert.UserID,
			&alert.WatchListID,
			&alert.WatchListItemID,
			&alert.Symbol,
			&alert.AlertType,
			&alert.Conditions,
			&alert.IsActive,
			&alert.Frequency,
			&alert.NotifyEmail,
			&alert.NotifyInApp,
			&alert.Name,
			&alert.Description,
			&alert.LastTriggeredAt,
			&alert.TriggerCount,
			&alert.CreatedAt,
			&alert.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert rule: %w", err)
		}
		// First-write-wins: ORDER BY created_at DESC means most recent first.
		// Only set if not already present (enforces 1:1).
		if _, exists := alertMap[alert.Symbol]; !exists {
			alertMap[alert.Symbol] = alert
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating alert rules: %w", err)
	}

	return alertMap, nil
}

// AlertExistsForSymbol checks if an alert already exists for a given symbol
// in a watchlist. Used to enforce the 1:1 constraint (one alert per watchlist item).
func AlertExistsForSymbol(watchListID string, symbol string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM alert_rules WHERE watch_list_id = $1 AND symbol = $2)`
	err := DB.QueryRow(query, watchListID, symbol).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check alert existence: %w", err)
	}
	return exists, nil
}

// CountAlertRulesByUserID counts alert rules for a user
func CountAlertRulesByUserID(userID string) (int, error) {
	var count int
	query := "SELECT COUNT(*) FROM alert_rules WHERE user_id = $1"
	err := DB.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count alert rules: %w", err)
	}
	return count, nil
}

// Alert Log Operations

// CreateAlertLog creates a new alert log entry
func CreateAlertLog(log *models.AlertLog) error {
	query := `
		INSERT INTO alert_logs (
			alert_rule_id, user_id, symbol, alert_type, condition_met,
			market_data, notification_sent
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, triggered_at, is_read, is_dismissed
	`
	err := DB.QueryRow(
		query,
		log.AlertRuleID,
		log.UserID,
		log.Symbol,
		log.AlertType,
		log.ConditionMet,
		log.MarketData,
		log.NotificationSent,
	).Scan(&log.ID, &log.TriggeredAt, &log.IsRead, &log.IsDismissed)

	if err != nil {
		return fmt.Errorf("failed to create alert log: %w", err)
	}
	return nil
}

// GetAlertLogsByUserID retrieves alert logs for a user
func GetAlertLogsByUserID(userID string, alertID string, symbol string, limit int, offset int) ([]models.AlertLogWithRule, error) {
	query := `
		SELECT
			al.id, al.alert_rule_id, al.user_id, al.symbol, al.triggered_at,
			al.alert_type, al.condition_met, al.market_data, al.notification_sent,
			al.notification_sent_at, al.notification_error, al.is_read, al.read_at,
			al.is_dismissed, al.dismissed_at,
			ar.name as rule_name
		FROM alert_logs al
		JOIN alert_rules ar ON al.alert_rule_id = ar.id
		WHERE al.user_id = $1
	`
	args := []interface{}{userID}
	argCount := 1

	if alertID != "" {
		argCount++
		query += fmt.Sprintf(" AND al.alert_rule_id = $%d", argCount)
		args = append(args, alertID)
	}

	if symbol != "" {
		argCount++
		query += fmt.Sprintf(" AND al.symbol = $%d", argCount)
		args = append(args, symbol)
	}

	query += " ORDER BY al.triggered_at DESC"

	if limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
	}

	if offset > 0 {
		argCount++
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert logs: %w", err)
	}
	defer rows.Close()

	logs := []models.AlertLogWithRule{}
	for rows.Next() {
		var log models.AlertLogWithRule
		err := rows.Scan(
			&log.ID,
			&log.AlertRuleID,
			&log.UserID,
			&log.Symbol,
			&log.TriggeredAt,
			&log.AlertType,
			&log.ConditionMet,
			&log.MarketData,
			&log.NotificationSent,
			&log.NotificationSentAt,
			&log.NotificationError,
			&log.IsRead,
			&log.ReadAt,
			&log.IsDismissed,
			&log.DismissedAt,
			&log.RuleName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// MarkAlertLogAsRead marks an alert log as read
func MarkAlertLogAsRead(logID string, userID string) error {
	query := `
		UPDATE alert_logs
		SET is_read = true, read_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $2
	`
	result, err := DB.Exec(query, logID, userID)
	if err != nil {
		return fmt.Errorf("failed to mark alert log as read: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("alert log not found")
	}

	return nil
}

// MarkAlertLogAsDismissed marks an alert log as dismissed
func MarkAlertLogAsDismissed(logID string, userID string) error {
	query := `
		UPDATE alert_logs
		SET is_dismissed = true, dismissed_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $2
	`
	result, err := DB.Exec(query, logID, userID)
	if err != nil {
		return fmt.Errorf("failed to mark alert log as dismissed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("alert log not found")
	}

	return nil
}
