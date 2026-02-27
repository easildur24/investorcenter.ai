package delivery

import (
	"errors"
	"os"
	"sync"
	"testing"

	"notification-service/config"
	"notification-service/models"
)

// ---------------------------------------------------------------------------
// mockStore implements database.Store with configurable return values.
// ---------------------------------------------------------------------------

type mockStore struct {
	// GetActiveAlertsForSymbols
	activeAlerts    []models.AlertRule
	activeAlertsErr error

	// CreateAlertLog
	createAlertLogID  string
	createAlertLogErr error

	// ClaimAlertTrigger
	claimResult bool
	claimErr    error

	// UpdateAlertLogNotificationSent
	updateLogErr error

	// GetTodayEmailCount
	todayEmailCount    int
	todayEmailCountErr error

	// GetNotificationPreferences
	notifPrefs    *models.NotificationPreferences
	notifPrefsErr error

	// GetUserEmail
	userEmail    *models.UserEmail
	userEmailErr error
}

func (m *mockStore) GetActiveAlertsForSymbols(symbols []string) ([]models.AlertRule, error) {
	return m.activeAlerts, m.activeAlertsErr
}

func (m *mockStore) CreateAlertLog(alertLog *models.AlertLog) (string, error) {
	return m.createAlertLogID, m.createAlertLogErr
}

func (m *mockStore) ClaimAlertTrigger(alertID string, frequency string) (bool, error) {
	return m.claimResult, m.claimErr
}

func (m *mockStore) UpdateAlertLogNotificationSent(logID string, sent bool) error {
	return m.updateLogErr
}

func (m *mockStore) GetTodayEmailCount(userID string) (int, error) {
	return m.todayEmailCount, m.todayEmailCountErr
}

func (m *mockStore) GetNotificationPreferences(userID string) (*models.NotificationPreferences, error) {
	return m.notifPrefs, m.notifPrefsErr
}

func (m *mockStore) GetUserEmail(userID string) (*models.UserEmail, error) {
	return m.userEmail, m.userEmailErr
}

// ---------------------------------------------------------------------------
// sendRecorder tracks calls to sendFunc.
// ---------------------------------------------------------------------------

type sendRecorder struct {
	mu    sync.Mutex
	calls []sendCall
	err   error // error to return from sendFunc
}

type sendCall struct {
	to      string
	subject string
	body    string
}

func (r *sendRecorder) sendFunc(to, subject, body string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, sendCall{to: to, subject: subject, body: body})
	return r.err
}

func (r *sendRecorder) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

func (r *sendRecorder) lastCall() sendCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls[len(r.calls)-1]
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// loadConfigWithSMTP loads a Config via config.Load() after setting the
// SMTP_PASSWORD env var so that cfg.SMTPPassword.Value() is non-empty.
// It then overrides SMTP-related fields for test isolation.
func loadConfigWithSMTP(t *testing.T) *config.Config {
	t.Helper()
	// Set env var so config.Load() picks up a non-empty password.
	os.Setenv("SMTP_PASSWORD", "test-password-123")
	t.Cleanup(func() { os.Unsetenv("SMTP_PASSWORD") })

	cfg := config.Load()
	cfg.SMTPHost = "smtp.example.com"
	cfg.SMTPPort = "587"
	cfg.SMTPFromEmail = "alerts@test.com"
	cfg.SMTPFromName = "Test Alerts"
	cfg.FrontendURL = "https://example.com"
	return cfg
}

// newTestEmailDelivery creates an EmailDelivery wired to the given mockStore
// and sendRecorder with SMTP fully configured.
func newTestEmailDelivery(t *testing.T, store *mockStore, rec *sendRecorder) *EmailDelivery {
	t.Helper()
	cfg := loadConfigWithSMTP(t)
	return &EmailDelivery{
		cfg:      cfg,
		db:       store,
		sendFunc: rec.sendFunc,
	}
}

func sampleAlert() *models.AlertRule {
	return &models.AlertRule{
		ID:          "alert-1",
		UserID:      "user-1",
		WatchListID: "wl-1",
		Symbol:      "AAPL",
		AlertType:   "price_above",
		Name:        "AAPL above 200",
		NotifyEmail: true,
	}
}

func sampleAlertLog() *models.AlertLog {
	return &models.AlertLog{
		ID:          "log-1",
		AlertRuleID: "alert-1",
		UserID:      "user-1",
		Symbol:      "AAPL",
	}
}

func sampleQuote() *models.SymbolQuote {
	return &models.SymbolQuote{
		Price:     210.50,
		Volume:    45000000,
		ChangePct: 2.35,
	}
}

func stringPtr(s string) *string {
	return &s
}

// ---------------------------------------------------------------------------
// Tests for Send
// ---------------------------------------------------------------------------

func TestSend_SMTPNotConfigured(t *testing.T) {
	store := &mockStore{}
	rec := &sendRecorder{}

	// Empty config: SMTPHost="" and SMTPPassword.Value()=""
	d := &EmailDelivery{
		cfg:      &config.Config{},
		db:       store,
		sendFunc: rec.sendFunc,
	}

	err := d.Send(sampleAlert(), sampleAlertLog(), sampleQuote())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if rec.callCount() != 0 {
		t.Fatalf("expected sendFunc not called, got %d calls", rec.callCount())
	}
}

func TestSend_GetNotificationPreferencesError(t *testing.T) {
	store := &mockStore{
		notifPrefsErr: errors.New("db connection failed"),
	}
	rec := &sendRecorder{}
	d := newTestEmailDelivery(t, store, rec)

	err := d.Send(sampleAlert(), sampleAlertLog(), sampleQuote())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if rec.callCount() != 0 {
		t.Fatalf("expected sendFunc not called, got %d calls", rec.callCount())
	}
}

func TestSend_EmailDisabled(t *testing.T) {
	store := &mockStore{
		notifPrefs: &models.NotificationPreferences{
			EmailEnabled:  false,
			EmailVerified: true,
		},
	}
	rec := &sendRecorder{}
	d := newTestEmailDelivery(t, store, rec)

	err := d.Send(sampleAlert(), sampleAlertLog(), sampleQuote())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if rec.callCount() != 0 {
		t.Fatalf("expected sendFunc not called, got %d calls", rec.callCount())
	}
}

func TestSend_EmailNotVerified(t *testing.T) {
	store := &mockStore{
		notifPrefs: &models.NotificationPreferences{
			EmailEnabled:  true,
			EmailVerified: false,
		},
	}
	rec := &sendRecorder{}
	d := newTestEmailDelivery(t, store, rec)

	err := d.Send(sampleAlert(), sampleAlertLog(), sampleQuote())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if rec.callCount() != 0 {
		t.Fatalf("expected sendFunc not called, got %d calls", rec.callCount())
	}
}

func TestSend_QuietHours(t *testing.T) {
	// Use a full-day quiet hours range to guarantee we are always inside.
	store := &mockStore{
		notifPrefs: &models.NotificationPreferences{
			EmailEnabled:       true,
			EmailVerified:      true,
			QuietHoursEnabled:  true,
			QuietHoursStart:    "00:00:00",
			QuietHoursEnd:      "23:59:59",
			QuietHoursTimezone: "UTC",
		},
	}
	rec := &sendRecorder{}
	d := newTestEmailDelivery(t, store, rec)

	err := d.Send(sampleAlert(), sampleAlertLog(), sampleQuote())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if rec.callCount() != 0 {
		t.Fatalf("expected sendFunc not called during quiet hours, got %d calls", rec.callCount())
	}
}

func TestSend_RateLimitExceeded(t *testing.T) {
	store := &mockStore{
		notifPrefs: &models.NotificationPreferences{
			EmailEnabled:    true,
			EmailVerified:   true,
			MaxEmailsPerDay: 5,
		},
		todayEmailCount: 5, // at the limit (count >= max)
	}
	rec := &sendRecorder{}
	d := newTestEmailDelivery(t, store, rec)

	err := d.Send(sampleAlert(), sampleAlertLog(), sampleQuote())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if rec.callCount() != 0 {
		t.Fatalf("expected sendFunc not called when rate limited, got %d calls", rec.callCount())
	}
}

func TestSend_GetUserEmailError(t *testing.T) {
	store := &mockStore{
		notifPrefs:   nil, // no preferences, skip all checks
		userEmailErr: errors.New("user not found"),
	}
	rec := &sendRecorder{}
	d := newTestEmailDelivery(t, store, rec)

	err := d.Send(sampleAlert(), sampleAlertLog(), sampleQuote())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if rec.callCount() != 0 {
		t.Fatalf("expected sendFunc not called, got %d calls", rec.callCount())
	}
}

func TestSend_HappyPath_NilPrefs(t *testing.T) {
	store := &mockStore{
		notifPrefs: nil, // no prefs, goes straight to GetUserEmail
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
		t.Fatalf("expected sendFunc called once, got %d calls", rec.callCount())
	}
	call := rec.lastCall()
	if call.to != "user@example.com" {
		t.Errorf("expected email to user@example.com, got %s", call.to)
	}
	if call.subject == "" {
		t.Error("expected non-empty subject")
	}
}

func TestSend_HappyPath_PrefsEmailOverride(t *testing.T) {
	overrideEmail := "override@example.com"
	store := &mockStore{
		notifPrefs: &models.NotificationPreferences{
			EmailEnabled:  true,
			EmailVerified: true,
			EmailAddress:  stringPtr(overrideEmail),
		},
		userEmail: &models.UserEmail{
			Email:    "original@example.com",
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
		t.Fatalf("expected sendFunc called once, got %d calls", rec.callCount())
	}
	call := rec.lastCall()
	if call.to != overrideEmail {
		t.Errorf("expected email to %s, got %s", overrideEmail, call.to)
	}
}

func TestSend_HappyPath_RateLimitNotExceeded(t *testing.T) {
	store := &mockStore{
		notifPrefs: &models.NotificationPreferences{
			EmailEnabled:    true,
			EmailVerified:   true,
			MaxEmailsPerDay: 10,
		},
		todayEmailCount: 3, // well under the limit
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
		t.Fatalf("expected sendFunc called once, got %d calls", rec.callCount())
	}
}

// ---------------------------------------------------------------------------
// Tests for Deliver (Router)
// ---------------------------------------------------------------------------

func TestDeliver_NotifyEmailTrue(t *testing.T) {
	store := &mockStore{
		notifPrefs: nil,
		userEmail: &models.UserEmail{
			Email:    "user@example.com",
			FullName: "Test User",
		},
	}
	rec := &sendRecorder{}
	emailDelivery := newTestEmailDelivery(t, store, rec)
	router := NewRouter(emailDelivery)

	alert := sampleAlert()
	alert.NotifyEmail = true

	err := router.Deliver(alert, sampleAlertLog(), sampleQuote())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if rec.callCount() != 1 {
		t.Fatalf("expected sendFunc called once via Deliver, got %d calls", rec.callCount())
	}
}

func TestDeliver_NotifyEmailFalse(t *testing.T) {
	store := &mockStore{}
	rec := &sendRecorder{}
	emailDelivery := newTestEmailDelivery(t, store, rec)
	router := NewRouter(emailDelivery)

	alert := sampleAlert()
	alert.NotifyEmail = false

	err := router.Deliver(alert, sampleAlertLog(), sampleQuote())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if rec.callCount() != 0 {
		t.Fatalf("expected sendFunc not called, got %d calls", rec.callCount())
	}
}
