package services

import (
	"encoding/json"
	"fmt"
	"investorcenter-api/database"
	"investorcenter-api/models"
	"time"
)

type NotificationService struct {
	emailService *EmailService
}

func NewNotificationService(emailService *EmailService) *NotificationService {
	return &NotificationService{
		emailService: emailService,
	}
}

// GetNotificationPreferences retrieves notification preferences for a user
func (s *NotificationService) GetNotificationPreferences(userID string) (*models.NotificationPreferences, error) {
	return database.GetNotificationPreferences(userID)
}

// UpdateNotificationPreferences updates notification preferences
func (s *NotificationService) UpdateNotificationPreferences(userID string, req *models.UpdateNotificationPreferencesRequest) (*models.NotificationPreferences, error) {
	updates := make(map[string]interface{})

	if req.EmailEnabled != nil {
		updates["email_enabled"] = *req.EmailEnabled
	}
	if req.EmailAddress != nil {
		updates["email_address"] = *req.EmailAddress
		// Reset email_verified if address changes
		updates["email_verified"] = false
	}
	if req.PriceAlertsEnabled != nil {
		updates["price_alerts_enabled"] = *req.PriceAlertsEnabled
	}
	if req.VolumeAlertsEnabled != nil {
		updates["volume_alerts_enabled"] = *req.VolumeAlertsEnabled
	}
	if req.NewsAlertsEnabled != nil {
		updates["news_alerts_enabled"] = *req.NewsAlertsEnabled
	}
	if req.EarningsAlertsEnabled != nil {
		updates["earnings_alerts_enabled"] = *req.EarningsAlertsEnabled
	}
	if req.SECFilingAlertsEnabled != nil {
		updates["sec_filing_alerts_enabled"] = *req.SECFilingAlertsEnabled
	}
	if req.DailyDigestEnabled != nil {
		updates["daily_digest_enabled"] = *req.DailyDigestEnabled
	}
	if req.DailyDigestTime != nil {
		updates["daily_digest_time"] = *req.DailyDigestTime
	}
	if req.WeeklyDigestEnabled != nil {
		updates["weekly_digest_enabled"] = *req.WeeklyDigestEnabled
	}
	if req.WeeklyDigestDay != nil {
		updates["weekly_digest_day"] = *req.WeeklyDigestDay
	}
	if req.WeeklyDigestTime != nil {
		updates["weekly_digest_time"] = *req.WeeklyDigestTime
	}
	if req.DigestIncludePortfolioSummary != nil {
		updates["digest_include_portfolio_summary"] = *req.DigestIncludePortfolioSummary
	}
	if req.DigestIncludeTopMovers != nil {
		updates["digest_include_top_movers"] = *req.DigestIncludeTopMovers
	}
	if req.DigestIncludeRecentAlerts != nil {
		updates["digest_include_recent_alerts"] = *req.DigestIncludeRecentAlerts
	}
	if req.DigestIncludeNewsHighlights != nil {
		updates["digest_include_news_highlights"] = *req.DigestIncludeNewsHighlights
	}
	if req.QuietHoursEnabled != nil {
		updates["quiet_hours_enabled"] = *req.QuietHoursEnabled
	}
	if req.QuietHoursStart != nil {
		updates["quiet_hours_start"] = *req.QuietHoursStart
	}
	if req.QuietHoursEnd != nil {
		updates["quiet_hours_end"] = *req.QuietHoursEnd
	}
	if req.QuietHoursTimezone != nil {
		updates["quiet_hours_timezone"] = *req.QuietHoursTimezone
	}

	if err := database.UpdateNotificationPreferences(userID, updates); err != nil {
		return nil, err
	}

	return database.GetNotificationPreferences(userID)
}

// CreateInAppNotification creates a new in-app notification
func (s *NotificationService) CreateInAppNotification(userID string, alertLogID *string, notifType string, title string, message string, metadata interface{}) error {
	var metadataJSON json.RawMessage
	if metadata != nil {
		jsonData, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = jsonData
	}

	notification := &models.InAppNotification{
		UserID:     userID,
		AlertLogID: alertLogID,
		Type:       notifType,
		Title:      title,
		Message:    message,
		Metadata:   metadataJSON,
	}

	return database.CreateInAppNotification(notification)
}

// GetInAppNotifications retrieves in-app notifications for a user
func (s *NotificationService) GetInAppNotifications(userID string, unreadOnly bool, limit int) ([]models.InAppNotification, error) {
	if limit == 0 {
		limit = 50
	}
	return database.GetInAppNotifications(userID, unreadOnly, limit)
}

// GetUnreadNotificationCount gets the count of unread notifications
func (s *NotificationService) GetUnreadNotificationCount(userID string) (int, error) {
	return database.GetUnreadNotificationCount(userID)
}

// MarkNotificationAsRead marks a notification as read
func (s *NotificationService) MarkNotificationAsRead(notificationID string, userID string) error {
	return database.MarkNotificationAsRead(notificationID, userID)
}

// MarkAllNotificationsAsRead marks all notifications as read
func (s *NotificationService) MarkAllNotificationsAsRead(userID string) error {
	return database.MarkAllNotificationsAsRead(userID)
}

// DismissNotification dismisses a notification
func (s *NotificationService) DismissNotification(notificationID string, userID string) error {
	return database.DismissNotification(notificationID, userID)
}

// SendAlertEmail sends an email notification for an alert
func (s *NotificationService) SendAlertEmail(userID string, alert *models.AlertRule, conditionMet interface{}, marketData interface{}) error {
	// Get user's notification preferences
	prefs, err := database.GetNotificationPreferences(userID)
	if err != nil {
		return err
	}

	// Check if email notifications are enabled
	if !prefs.EmailEnabled || !alert.NotifyEmail {
		return nil
	}

	// Get email address
	emailAddr := prefs.EmailAddress
	if emailAddr == nil || *emailAddr == "" {
		// Use user's primary email
		user, err := database.GetUserByID(userID)
		if err != nil {
			return err
		}
		emailAddr = &user.Email
	}

	// Check if email is verified
	if !prefs.EmailVerified {
		return nil // Skip unverified emails
	}

	// Format alert email
	subject := fmt.Sprintf("Alert Triggered: %s", alert.Name)
	body := s.formatAlertEmailBody(alert, conditionMet, marketData)

	return s.emailService.sendEmail(*emailAddr, subject, body)
}

// formatAlertEmailBody formats the email body for an alert
func (s *NotificationService) formatAlertEmailBody(alert *models.AlertRule, conditionMet interface{}, marketData interface{}) string {
	conditionJSON, _ := json.MarshalIndent(conditionMet, "", "  ")
	marketDataJSON, _ := json.MarshalIndent(marketData, "", "  ")

	return fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="background-color: #f8f9fa; padding: 20px; border-radius: 5px; margin-bottom: 20px;">
				<h2 style="color: #333; margin-top: 0;">🔔 Alert Triggered</h2>
				<h3 style="color: #666; margin-bottom: 20px;">%s</h3>
			</div>

			<div style="background-color: #fff; border: 1px solid #ddd; border-radius: 5px; padding: 20px; margin-bottom: 20px;">
				<h4 style="color: #333; margin-top: 0;">Symbol: %s</h4>
				<p style="color: #666; margin-bottom: 10px;"><strong>Alert Type:</strong> %s</p>
				<p style="color: #666; margin-bottom: 10px;"><strong>Triggered At:</strong> %s</p>
			</div>

			<div style="background-color: #fff; border: 1px solid #ddd; border-radius: 5px; padding: 20px; margin-bottom: 20px;">
				<h4 style="color: #333; margin-top: 0;">Condition Met</h4>
				<pre style="background-color: #f8f9fa; padding: 10px; border-radius: 3px; overflow-x: auto;">%s</pre>
			</div>

			<div style="background-color: #fff; border: 1px solid #ddd; border-radius: 5px; padding: 20px; margin-bottom: 20px;">
				<h4 style="color: #333; margin-top: 0;">Market Data</h4>
				<pre style="background-color: #f8f9fa; padding: 10px; border-radius: 3px; overflow-x: auto;">%s</pre>
			</div>

			<div style="text-align: center; padding: 20px;">
				<a href="%s/alerts" style="background-color: #4CAF50; color: white; padding: 12px 24px; text-decoration: none; border-radius: 5px; display: inline-block;">View Alert Details</a>
			</div>

			<div style="color: #999; font-size: 12px; text-align: center; margin-top: 20px; padding-top: 20px; border-top: 1px solid #ddd;">
				<p>You're receiving this email because you have alert notifications enabled.</p>
				<p><a href="%s/settings/notifications" style="color: #999;">Manage notification preferences</a></p>
			</div>
		</body>
		</html>
	`, alert.Name, alert.Symbol, alert.AlertType, time.Now().Format(time.RFC3339),
		string(conditionJSON), string(marketDataJSON), s.emailService.frontendURL, s.emailService.frontendURL)
}

// IsInQuietHours checks if current time is in user's quiet hours
func (s *NotificationService) IsInQuietHours(userID string) (bool, error) {
	prefs, err := database.GetNotificationPreferences(userID)
	if err != nil {
		return false, err
	}

	if !prefs.QuietHoursEnabled {
		return false, nil
	}

	// Parse times
	loc, err := time.LoadLocation(prefs.QuietHoursTimezone)
	if err != nil {
		loc = time.UTC
	}

	now := time.Now().In(loc)
	currentTime := now.Format("15:04:05")

	// Simple time comparison
	if prefs.QuietHoursStart < prefs.QuietHoursEnd {
		// Same day quiet hours (e.g., 22:00 - 08:00)
		return currentTime >= prefs.QuietHoursStart && currentTime < prefs.QuietHoursEnd, nil
	} else {
		// Overnight quiet hours (e.g., 22:00 - 08:00)
		return currentTime >= prefs.QuietHoursStart || currentTime < prefs.QuietHoursEnd, nil
	}
}
