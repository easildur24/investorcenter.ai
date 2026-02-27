package delivery

import (
	"errors"
	"os"
	"strings"
	"testing"

	"notification-service/config"
	"notification-service/models"
)

// ---------------------------------------------------------------------------
// NewEmailDelivery constructor
// ---------------------------------------------------------------------------

func TestNewEmailDelivery(t *testing.T) {
	os.Setenv("SMTP_PASSWORD", "test-password")
	t.Cleanup(func() { os.Unsetenv("SMTP_PASSWORD") })

	cfg := config.Load()
	cfg.SMTPHost = "smtp.test.com"
	store := &mockStore{}

	d := NewEmailDelivery(cfg, store)
	if d == nil {
		t.Fatal("expected non-nil EmailDelivery")
	}
	if d.cfg != cfg {
		t.Error("config not set correctly")
	}
	if d.db != store {
		t.Error("store not set correctly")
	}
	if d.sendFunc == nil {
		t.Error("sendFunc should be set to sendEmail method")
	}
}

func TestNewEmailDelivery_NilConfig(t *testing.T) {
	store := &mockStore{}
	d := NewEmailDelivery(nil, store)
	if d == nil {
		t.Fatal("expected non-nil EmailDelivery even with nil config")
	}
	if d.cfg != nil {
		t.Error("config should be nil")
	}
}

// ---------------------------------------------------------------------------
// Deliver error propagation
// ---------------------------------------------------------------------------

func TestDeliver_SendError(t *testing.T) {
	store := &mockStore{
		notifPrefs: nil,
		userEmail: &models.UserEmail{
			Email:    "user@example.com",
			FullName: "Test User",
		},
	}
	rec := &sendRecorder{err: errors.New("SMTP connection refused")}
	emailDelivery := newTestEmailDelivery(t, store, rec)
	router := NewRouter(emailDelivery)

	alert := sampleAlert()
	alert.NotifyEmail = true

	err := router.Deliver(alert, sampleAlertLog(), sampleQuote())
	if err == nil {
		t.Fatal("expected error from Deliver, got nil")
	}
	if !strings.Contains(err.Error(), "email:") {
		t.Errorf("expected error wrapped with 'email:', got: %v", err)
	}
	if !strings.Contains(err.Error(), "SMTP connection refused") {
		t.Errorf("expected original error message, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Send — sendFunc error
// ---------------------------------------------------------------------------

func TestSend_SendFuncError(t *testing.T) {
	store := &mockStore{
		notifPrefs: nil,
		userEmail: &models.UserEmail{
			Email:    "user@example.com",
			FullName: "Test User",
		},
	}
	rec := &sendRecorder{err: errors.New("SMTP timeout")}
	d := newTestEmailDelivery(t, store, rec)

	err := d.Send(sampleAlert(), sampleAlertLog(), sampleQuote())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "SMTP timeout") {
		t.Errorf("expected SMTP timeout error, got: %v", err)
	}
	// sendFunc should still have been called
	if rec.callCount() != 1 {
		t.Errorf("expected sendFunc called once, got %d", rec.callCount())
	}
}

func TestSend_RateLimitCountError(t *testing.T) {
	// When GetTodayEmailCount fails, we log a warning but still send
	store := &mockStore{
		notifPrefs: &models.NotificationPreferences{
			EmailEnabled:    true,
			EmailVerified:   true,
			MaxEmailsPerDay: 10,
		},
		todayEmailCountErr: errors.New("db error"),
		userEmail: &models.UserEmail{
			Email:    "user@example.com",
			FullName: "Test User",
		},
	}
	rec := &sendRecorder{}
	d := newTestEmailDelivery(t, store, rec)

	err := d.Send(sampleAlert(), sampleAlertLog(), sampleQuote())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	// Should still send despite rate limit check error
	if rec.callCount() != 1 {
		t.Errorf("expected sendFunc called once, got %d", rec.callCount())
	}
}

func TestSend_QuietHoursTimezoneError(t *testing.T) {
	// Invalid timezone should log warning but still send
	store := &mockStore{
		notifPrefs: &models.NotificationPreferences{
			EmailEnabled:       true,
			EmailVerified:      true,
			QuietHoursEnabled:  true,
			QuietHoursStart:    "22:00:00",
			QuietHoursEnd:      "06:00:00",
			QuietHoursTimezone: "Invalid/Timezone",
		},
		userEmail: &models.UserEmail{
			Email:    "user@example.com",
			FullName: "Test User",
		},
	}
	rec := &sendRecorder{}
	d := newTestEmailDelivery(t, store, rec)

	err := d.Send(sampleAlert(), sampleAlertLog(), sampleQuote())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	// Should still send despite timezone error (logged warning)
	if rec.callCount() != 1 {
		t.Errorf("expected sendFunc called once, got %d", rec.callCount())
	}
}

func TestSend_EmptyPrefsEmailAddress(t *testing.T) {
	// Prefs email address is empty string — should use user email
	emptyStr := ""
	store := &mockStore{
		notifPrefs: &models.NotificationPreferences{
			EmailEnabled:  true,
			EmailVerified: true,
			EmailAddress:  &emptyStr,
		},
		userEmail: &models.UserEmail{
			Email:    "user@example.com",
			FullName: "Test User",
		},
	}
	rec := &sendRecorder{}
	d := newTestEmailDelivery(t, store, rec)

	err := d.Send(sampleAlert(), sampleAlertLog(), sampleQuote())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if rec.callCount() != 1 {
		t.Fatalf("expected sendFunc called once, got %d", rec.callCount())
	}
	if rec.lastCall().to != "user@example.com" {
		t.Errorf("expected user email, got %s", rec.lastCall().to)
	}
}

// ---------------------------------------------------------------------------
// formatAlertEmailBody
// ---------------------------------------------------------------------------

func TestFormatAlertEmailBody(t *testing.T) {
	alert := sampleAlert()
	quote := sampleQuote()

	body := formatAlertEmailBody(alert, quote, "John Doe", "https://example.com")
	if body == "" {
		t.Fatal("expected non-empty email body")
	}
	if !strings.Contains(body, "AAPL") {
		t.Error("expected body to contain symbol")
	}
	if !strings.Contains(body, "AAPL above 200") {
		t.Error("expected body to contain alert name")
	}
	if !strings.Contains(body, "John Doe") {
		t.Error("expected body to contain user name")
	}
	if !strings.Contains(body, "210.50") {
		t.Error("expected body to contain price")
	}
	if !strings.Contains(body, "2.35") {
		t.Error("expected body to contain change pct")
	}
	if !strings.Contains(body, "https://example.com/watchlist/wl-1") {
		t.Error("expected body to contain watchlist URL")
	}
	if !strings.Contains(body, "https://example.com/settings") {
		t.Error("expected body to contain settings URL")
	}
}

