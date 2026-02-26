package delivery

import (
	"fmt"
	"log"
	"net/smtp"
	"time"

	"notification-service/config"
	"notification-service/database"
	"notification-service/models"
)

// EmailDelivery sends alert notification emails via SMTP.
type EmailDelivery struct {
	cfg *config.Config
	db  *database.DB
}

// NewEmailDelivery creates a new EmailDelivery.
func NewEmailDelivery(cfg *config.Config, db *database.DB) *EmailDelivery {
	return &EmailDelivery{cfg: cfg, db: db}
}

// Send sends an alert notification email to the user.
// Checks preferences (email enabled, verified) and quiet hours before sending.
func (d *EmailDelivery) Send(alert *models.AlertRule, alertLog *models.AlertLog, quote *models.SymbolQuote) error {
	// Skip if SMTP not configured (local dev)
	if d.cfg.SMTPHost == "" || d.cfg.SMTPPassword == "" {
		log.Printf("SMTP not configured — skipping email for alert %s", alert.ID)
		return nil
	}

	// Check notification preferences
	prefs, err := d.db.GetNotificationPreferences(alert.UserID)
	if err != nil {
		return fmt.Errorf("get notification preferences: %w", err)
	}

	// If preferences exist, check email settings
	if prefs != nil {
		if !prefs.EmailEnabled {
			return nil // Email disabled by user
		}
		if !prefs.EmailVerified {
			return nil // Email not verified
		}

		// Check quiet hours
		if prefs.QuietHoursEnabled {
			inQuietHours, err := isInQuietHours(prefs)
			if err != nil {
				log.Printf("Error checking quiet hours: %v", err)
			} else if inQuietHours {
				log.Printf("Skipping email for alert %s — user in quiet hours", alert.ID)
				return nil
			}
		}
	}

	// Get user email address
	user, err := d.db.GetUserEmail(alert.UserID)
	if err != nil {
		return fmt.Errorf("get user email: %w", err)
	}

	// Use preferences email if set, otherwise fall back to user email
	toEmail := user.Email
	if prefs != nil && prefs.EmailAddress != nil && *prefs.EmailAddress != "" {
		toEmail = *prefs.EmailAddress
	}

	// Build and send email
	subject := fmt.Sprintf("Alert: %s %s", alert.Symbol, alertTypeLabel(alert.AlertType))
	body := formatAlertEmailBody(alert, quote, user.FullName, d.cfg.FrontendURL)

	return d.sendEmail(toEmail, subject, body)
}

// sendEmail sends an HTML email via SMTP.
func (d *EmailDelivery) sendEmail(to, subject, htmlBody string) error {
	from := d.cfg.SMTPFromEmail
	auth := smtp.PlainAuth("", d.cfg.SMTPUsername, d.cfg.SMTPPassword, d.cfg.SMTPHost)

	msg := fmt.Sprintf(
		"From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		d.cfg.SMTPFromName, from, to, subject, htmlBody,
	)

	addr := fmt.Sprintf("%s:%s", d.cfg.SMTPHost, d.cfg.SMTPPort)
	if err := smtp.SendMail(addr, auth, from, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	log.Printf("Email sent to %s for alert subject: %s", to, subject)
	return nil
}

// isInQuietHours checks if the current time falls within the user's quiet hours.
func isInQuietHours(prefs *models.NotificationPreferences) (bool, error) {
	if !prefs.QuietHoursEnabled {
		return false, nil
	}

	loc, err := time.LoadLocation(prefs.QuietHoursTimezone)
	if err != nil {
		return false, fmt.Errorf("load timezone %s: %w", prefs.QuietHoursTimezone, err)
	}

	now := time.Now().In(loc)
	currentTime := now.Format("15:04:05")

	start := prefs.QuietHoursStart
	end := prefs.QuietHoursEnd

	if start <= end {
		// Same-day range (e.g., 08:00 to 22:00)
		return currentTime >= start && currentTime <= end, nil
	}
	// Overnight range (e.g., 22:00 to 08:00)
	return currentTime >= start || currentTime <= end, nil
}

// formatAlertEmailBody generates the HTML email body for an alert notification.
func formatAlertEmailBody(alert *models.AlertRule, quote *models.SymbolQuote, userName, frontendURL string) string {
	watchlistURL := fmt.Sprintf("%s/watchlist/%s", frontendURL, alert.WatchListID)

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <div style="background: #1a1a2e; color: #e0e0e0; padding: 24px; border-radius: 8px;">
    <h2 style="color: #4fc3f7; margin-top: 0;">Alert Triggered: %s</h2>
    <p>Hi %s,</p>
    <p>Your alert <strong>%s</strong> has been triggered:</p>
    <div style="background: #16213e; padding: 16px; border-radius: 6px; margin: 16px 0;">
      <table style="width: 100%%; border-collapse: collapse; color: #e0e0e0;">
        <tr>
          <td style="padding: 8px 0;"><strong>Symbol</strong></td>
          <td style="padding: 8px 0; text-align: right;">%s</td>
        </tr>
        <tr>
          <td style="padding: 8px 0;"><strong>Current Price</strong></td>
          <td style="padding: 8px 0; text-align: right;">$%.2f</td>
        </tr>
        <tr>
          <td style="padding: 8px 0;"><strong>Change</strong></td>
          <td style="padding: 8px 0; text-align: right;">%.2f%%</td>
        </tr>
        <tr>
          <td style="padding: 8px 0;"><strong>Volume</strong></td>
          <td style="padding: 8px 0; text-align: right;">%s</td>
        </tr>
      </table>
    </div>
    <p>
      <a href="%s" style="display: inline-block; background: #4fc3f7; color: #1a1a2e; padding: 10px 24px; border-radius: 6px; text-decoration: none; font-weight: bold;">
        View Watchlist
      </a>
    </p>
    <hr style="border: none; border-top: 1px solid #333; margin: 20px 0;">
    <p style="color: #888; font-size: 12px;">
      You received this email because you have email alerts enabled for this watchlist.
      To manage your notification preferences, visit your
      <a href="%s/settings" style="color: #4fc3f7;">account settings</a>.
    </p>
  </div>
</body>
</html>`,
		alert.Name,
		userName,
		alert.Name,
		alert.Symbol,
		quote.Price,
		quote.ChangePct,
		formatVolume(float64(quote.Volume)),
		watchlistURL,
		frontendURL,
	)
}
