package database

import (
	"database/sql"
	"fmt"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// ---------------------------------------------------------------------------
// GetNotificationPreferences
// ---------------------------------------------------------------------------

func TestGetNotificationPreferences_Success(t *testing.T) {
	db, mock := newMockDB(t)

	emailAddr := "user@example.com"

	columns := []string{
		"user_id", "email_enabled", "email_address", "email_verified",
		"quiet_hours_enabled", "quiet_hours_start", "quiet_hours_end", "quiet_hours_timezone",
		"max_alerts_per_day", "max_emails_per_day",
	}
	rows := sqlmock.NewRows(columns).AddRow(
		"user-123",         // user_id
		true,               // email_enabled
		emailAddr,          // email_address
		true,               // email_verified
		true,               // quiet_hours_enabled
		"22:00:00",         // quiet_hours_start
		"08:00:00",         // quiet_hours_end
		"America/New_York", // quiet_hours_timezone
		50,                 // max_alerts_per_day
		10,                 // max_emails_per_day
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT user_id, email_enabled, email_address, email_verified,
		       quiet_hours_enabled, quiet_hours_start, quiet_hours_end, quiet_hours_timezone,
		       max_alerts_per_day, max_emails_per_day
		FROM notification_preferences
		WHERE user_id = $1`,
	)).
		WithArgs("user-123").
		WillReturnRows(rows)

	prefs, err := db.GetNotificationPreferences("user-123")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if prefs == nil {
		t.Fatal("expected non-nil prefs")
	}

	if prefs.UserID != "user-123" {
		t.Errorf("expected UserID 'user-123', got %q", prefs.UserID)
	}
	if !prefs.EmailEnabled {
		t.Error("expected EmailEnabled true")
	}
	if prefs.EmailAddress == nil || *prefs.EmailAddress != emailAddr {
		t.Errorf("expected EmailAddress %q, got %v", emailAddr, prefs.EmailAddress)
	}
	if !prefs.EmailVerified {
		t.Error("expected EmailVerified true")
	}
	if !prefs.QuietHoursEnabled {
		t.Error("expected QuietHoursEnabled true")
	}
	if prefs.QuietHoursStart != "22:00:00" {
		t.Errorf("expected QuietHoursStart '22:00:00', got %q", prefs.QuietHoursStart)
	}
	if prefs.QuietHoursEnd != "08:00:00" {
		t.Errorf("expected QuietHoursEnd '08:00:00', got %q", prefs.QuietHoursEnd)
	}
	if prefs.QuietHoursTimezone != "America/New_York" {
		t.Errorf("expected QuietHoursTimezone 'America/New_York', got %q", prefs.QuietHoursTimezone)
	}
	if prefs.MaxAlertsPerDay != 50 {
		t.Errorf("expected MaxAlertsPerDay 50, got %d", prefs.MaxAlertsPerDay)
	}
	if prefs.MaxEmailsPerDay != 10 {
		t.Errorf("expected MaxEmailsPerDay 10, got %d", prefs.MaxEmailsPerDay)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected mock expectations: %v", err)
	}
}

func TestGetNotificationPreferences_NoRows(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT user_id, email_enabled, email_address, email_verified,
		       quiet_hours_enabled, quiet_hours_start, quiet_hours_end, quiet_hours_timezone,
		       max_alerts_per_day, max_emails_per_day
		FROM notification_preferences
		WHERE user_id = $1`,
	)).
		WithArgs("user-999").
		WillReturnError(sql.ErrNoRows)

	prefs, err := db.GetNotificationPreferences("user-999")
	if err != nil {
		t.Fatalf("expected nil error for no rows, got %v", err)
	}
	if prefs != nil {
		t.Fatalf("expected nil prefs for no rows, got %+v", prefs)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected mock expectations: %v", err)
	}
}

func TestGetNotificationPreferences_Error(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT user_id, email_enabled, email_address, email_verified,
		       quiet_hours_enabled, quiet_hours_start, quiet_hours_end, quiet_hours_timezone,
		       max_alerts_per_day, max_emails_per_day
		FROM notification_preferences
		WHERE user_id = $1`,
	)).
		WithArgs("user-123").
		WillReturnError(fmt.Errorf("connection refused"))

	prefs, err := db.GetNotificationPreferences("user-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if prefs != nil {
		t.Fatalf("expected nil prefs on error, got %+v", prefs)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected mock expectations: %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetUserEmail
// ---------------------------------------------------------------------------

func TestGetUserEmail_Success(t *testing.T) {
	db, mock := newMockDB(t)

	rows := sqlmock.NewRows([]string{"email", "full_name"}).
		AddRow("john@example.com", "John Doe")

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT email, full_name FROM users WHERE id = $1`,
	)).
		WithArgs("user-123").
		WillReturnRows(rows)

	user, err := db.GetUserEmail("user-123")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if user == nil {
		t.Fatal("expected non-nil user")
	}
	if user.Email != "john@example.com" {
		t.Errorf("expected Email 'john@example.com', got %q", user.Email)
	}
	if user.FullName != "John Doe" {
		t.Errorf("expected FullName 'John Doe', got %q", user.FullName)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected mock expectations: %v", err)
	}
}

func TestGetUserEmail_Error(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT email, full_name FROM users WHERE id = $1`,
	)).
		WithArgs("user-123").
		WillReturnError(fmt.Errorf("user not found"))

	user, err := db.GetUserEmail("user-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if user != nil {
		t.Fatalf("expected nil user on error, got %+v", user)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected mock expectations: %v", err)
	}
}
