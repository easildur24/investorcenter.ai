package database

import (
	"database/sql"
	"errors"
	"fmt"

	"notification-service/models"
)

// GetNotificationPreferences retrieves the notification preferences for a user.
// Returns nil if no preferences row exists.
func (db *DB) GetNotificationPreferences(userID string) (*models.NotificationPreferences, error) {
	var prefs models.NotificationPreferences
	err := db.QueryRow(`
		SELECT user_id, email_enabled, email_address, email_verified,
		       quiet_hours_enabled, quiet_hours_start, quiet_hours_end, quiet_hours_timezone,
		       max_alerts_per_day, max_emails_per_day
		FROM notification_preferences
		WHERE user_id = $1
	`, userID).Scan(
		&prefs.UserID, &prefs.EmailEnabled, &prefs.EmailAddress, &prefs.EmailVerified,
		&prefs.QuietHoursEnabled, &prefs.QuietHoursStart, &prefs.QuietHoursEnd, &prefs.QuietHoursTimezone,
		&prefs.MaxAlertsPerDay, &prefs.MaxEmailsPerDay,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No preferences set — use defaults
		}
		return nil, fmt.Errorf("get notification preferences: %w", err)
	}
	return &prefs, nil
}

// GetUserEmail retrieves the email address and name for a user.
func (db *DB) GetUserEmail(userID string) (*models.UserEmail, error) {
	var user models.UserEmail
	err := db.QueryRow(`
		SELECT email, full_name FROM users WHERE id = $1
	`, userID).Scan(&user.Email, &user.FullName)

	if err != nil {
		return nil, fmt.Errorf("get user email: %w", err)
	}
	return &user, nil
}
