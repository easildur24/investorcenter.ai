package database

import "notification-service/models"

// Store defines the database operations used by the notification service.
// This interface allows business logic to be tested with mock implementations.
type Store interface {
	GetActiveAlertsForSymbols(symbols []string) ([]models.AlertRule, error)
	CreateAlertLog(alertLog *models.AlertLog) (string, error)
	ClaimAlertTrigger(alertID string, frequency string) (bool, error)
	UpdateAlertLogNotificationSent(logID string, sent bool) error
	GetTodayEmailCount(userID string) (int, error)
	GetNotificationPreferences(userID string) (*models.NotificationPreferences, error)
	GetUserEmail(userID string) (*models.UserEmail, error)
}
