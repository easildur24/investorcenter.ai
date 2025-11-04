package database

import (
	"database/sql"
	"errors"
	"fmt"
	"investorcenter-api/models"
)

// Notification Preferences Operations

// GetNotificationPreferences retrieves notification preferences for a user
func GetNotificationPreferences(userID string) (*models.NotificationPreferences, error) {
	query := `
		SELECT
			id, user_id, email_enabled, email_address, email_verified,
			price_alerts_enabled, volume_alerts_enabled, news_alerts_enabled,
			earnings_alerts_enabled, sec_filing_alerts_enabled,
			daily_digest_enabled, daily_digest_time, weekly_digest_enabled,
			weekly_digest_day, weekly_digest_time,
			digest_include_portfolio_summary, digest_include_top_movers,
			digest_include_recent_alerts, digest_include_news_highlights,
			quiet_hours_enabled, quiet_hours_start, quiet_hours_end,
			quiet_hours_timezone, max_alerts_per_day, max_emails_per_day,
			created_at, updated_at
		FROM notification_preferences
		WHERE user_id = $1
	`
	prefs := &models.NotificationPreferences{}
	err := DB.QueryRow(query, userID).Scan(
		&prefs.ID,
		&prefs.UserID,
		&prefs.EmailEnabled,
		&prefs.EmailAddress,
		&prefs.EmailVerified,
		&prefs.PriceAlertsEnabled,
		&prefs.VolumeAlertsEnabled,
		&prefs.NewsAlertsEnabled,
		&prefs.EarningsAlertsEnabled,
		&prefs.SECFilingAlertsEnabled,
		&prefs.DailyDigestEnabled,
		&prefs.DailyDigestTime,
		&prefs.WeeklyDigestEnabled,
		&prefs.WeeklyDigestDay,
		&prefs.WeeklyDigestTime,
		&prefs.DigestIncludePortfolioSummary,
		&prefs.DigestIncludeTopMovers,
		&prefs.DigestIncludeRecentAlerts,
		&prefs.DigestIncludeNewsHighlights,
		&prefs.QuietHoursEnabled,
		&prefs.QuietHoursStart,
		&prefs.QuietHoursEnd,
		&prefs.QuietHoursTimezone,
		&prefs.MaxAlertsPerDay,
		&prefs.MaxEmailsPerDay,
		&prefs.CreatedAt,
		&prefs.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("notification preferences not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get notification preferences: %w", err)
	}

	return prefs, nil
}

// UpdateNotificationPreferences updates notification preferences
func UpdateNotificationPreferences(userID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return errors.New("no fields to update")
	}

	query := "UPDATE notification_preferences SET "
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
	query += fmt.Sprintf(" WHERE user_id = $%d", argCount)
	args = append(args, userID)

	result, err := DB.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update notification preferences: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("notification preferences not found")
	}

	return nil
}

// In-App Notification Operations

// CreateInAppNotification creates a new in-app notification
func CreateInAppNotification(notification *models.InAppNotification) error {
	query := `
		INSERT INTO notification_queue (
			user_id, alert_log_id, type, title, message, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, expires_at, is_read, is_dismissed
	`
	err := DB.QueryRow(
		query,
		notification.UserID,
		notification.AlertLogID,
		notification.Type,
		notification.Title,
		notification.Message,
		notification.Metadata,
	).Scan(
		&notification.ID,
		&notification.CreatedAt,
		&notification.ExpiresAt,
		&notification.IsRead,
		&notification.IsDismissed,
	)

	if err != nil {
		return fmt.Errorf("failed to create in-app notification: %w", err)
	}
	return nil
}

// GetInAppNotifications retrieves in-app notifications for a user
func GetInAppNotifications(userID string, unreadOnly bool, limit int) ([]models.InAppNotification, error) {
	query := `
		SELECT
			id, user_id, alert_log_id, type, title, message, metadata,
			is_read, read_at, is_dismissed, dismissed_at, created_at, expires_at
		FROM notification_queue
		WHERE user_id = $1
	`
	args := []interface{}{userID}

	if unreadOnly {
		query += " AND is_read = false AND is_dismissed = false"
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $2")
		args = append(args, limit)
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get in-app notifications: %w", err)
	}
	defer rows.Close()

	notifications := []models.InAppNotification{}
	for rows.Next() {
		var notif models.InAppNotification
		err := rows.Scan(
			&notif.ID,
			&notif.UserID,
			&notif.AlertLogID,
			&notif.Type,
			&notif.Title,
			&notif.Message,
			&notif.Metadata,
			&notif.IsRead,
			&notif.ReadAt,
			&notif.IsDismissed,
			&notif.DismissedAt,
			&notif.CreatedAt,
			&notif.ExpiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan in-app notification: %w", err)
		}
		notifications = append(notifications, notif)
	}

	return notifications, nil
}

// MarkNotificationAsRead marks a notification as read
func MarkNotificationAsRead(notificationID string, userID string) error {
	query := `
		UPDATE notification_queue
		SET is_read = true, read_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $2
	`
	result, err := DB.Exec(query, notificationID, userID)
	if err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("notification not found")
	}

	return nil
}

// MarkAllNotificationsAsRead marks all notifications as read for a user
func MarkAllNotificationsAsRead(userID string) error {
	query := `
		UPDATE notification_queue
		SET is_read = true, read_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND is_read = false
	`
	_, err := DB.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}
	return nil
}

// DismissNotification dismisses a notification
func DismissNotification(notificationID string, userID string) error {
	query := `
		UPDATE notification_queue
		SET is_dismissed = true, dismissed_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $2
	`
	result, err := DB.Exec(query, notificationID, userID)
	if err != nil {
		return fmt.Errorf("failed to dismiss notification: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("notification not found")
	}

	return nil
}

// GetUnreadNotificationCount gets count of unread notifications
func GetUnreadNotificationCount(userID string) (int, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM notification_queue
		WHERE user_id = $1 AND is_read = false AND is_dismissed = false
	`
	err := DB.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get unread notification count: %w", err)
	}
	return count, nil
}

// Digest Log Operations

// CreateDigestLog creates a new digest log entry
func CreateDigestLog(log *models.DigestLog) error {
	query := `
		INSERT INTO digest_logs (
			user_id, digest_type, period_start, period_end,
			email_sent, content_snapshot
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, sent_at, email_opened, email_clicked
	`
	err := DB.QueryRow(
		query,
		log.UserID,
		log.DigestType,
		log.PeriodStart,
		log.PeriodEnd,
		log.EmailSent,
		log.ContentSnapshot,
	).Scan(&log.ID, &log.SentAt, &log.EmailOpened, &log.EmailClicked)

	if err != nil {
		return fmt.Errorf("failed to create digest log: %w", err)
	}
	return nil
}

// CheckDigestSent checks if a digest was already sent for a period
func CheckDigestSent(userID string, digestType string, periodStart string) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM digest_logs
		WHERE user_id = $1 AND digest_type = $2 AND period_start = $3
	`
	err := DB.QueryRow(query, userID, digestType, periodStart).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check digest sent: %w", err)
	}
	return count > 0, nil
}
