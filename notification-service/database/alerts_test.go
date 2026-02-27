package database

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"notification-service/models"
)

func newMockDB(t *testing.T) (*DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return &DB{db}, mock
}

// ---------------------------------------------------------------------------
// GetActiveAlertsForSymbols
// ---------------------------------------------------------------------------

func TestGetActiveAlertsForSymbols_EmptySymbols(t *testing.T) {
	db, mock := newMockDB(t)

	alerts, err := db.GetActiveAlertsForSymbols([]string{})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if alerts != nil {
		t.Fatalf("expected nil alerts, got %v", alerts)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected mock expectations: %v", err)
	}
}

func TestGetActiveAlertsForSymbols_Success(t *testing.T) {
	db, mock := newMockDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	lastTriggered := now.Add(-1 * time.Hour)
	conditions := json.RawMessage(`{"threshold":150.0}`)

	columns := []string{
		"id", "user_id", "watch_list_id", "symbol", "alert_type", "conditions",
		"is_active", "frequency", "notify_email", "notify_in_app", "name",
		"last_triggered_at", "trigger_count", "created_at", "updated_at",
	}

	rows := sqlmock.NewRows(columns).AddRow(
		"alert-001",     // id
		"user-123",      // user_id
		"wl-456",        // watch_list_id
		"AAPL",          // symbol
		"price_above",   // alert_type
		conditions,      // conditions
		true,            // is_active
		"daily",         // frequency
		true,            // notify_email
		false,           // notify_in_app
		"AAPL above 150", // name
		lastTriggered,   // last_triggered_at
		3,               // trigger_count
		now,             // created_at
		now,             // updated_at
	)

	mock.ExpectQuery("SELECT id, user_id, watch_list_id, symbol, alert_type, conditions").
		WithArgs("AAPL").
		WillReturnRows(rows)

	alerts, err := db.GetActiveAlertsForSymbols([]string{"AAPL"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}

	a := alerts[0]
	if a.ID != "alert-001" {
		t.Errorf("expected ID 'alert-001', got %q", a.ID)
	}
	if a.UserID != "user-123" {
		t.Errorf("expected UserID 'user-123', got %q", a.UserID)
	}
	if a.WatchListID != "wl-456" {
		t.Errorf("expected WatchListID 'wl-456', got %q", a.WatchListID)
	}
	if a.Symbol != "AAPL" {
		t.Errorf("expected Symbol 'AAPL', got %q", a.Symbol)
	}
	if a.AlertType != "price_above" {
		t.Errorf("expected AlertType 'price_above', got %q", a.AlertType)
	}
	if string(a.Conditions) != string(conditions) {
		t.Errorf("expected Conditions %s, got %s", conditions, a.Conditions)
	}
	if !a.IsActive {
		t.Error("expected IsActive true")
	}
	if a.Frequency != "daily" {
		t.Errorf("expected Frequency 'daily', got %q", a.Frequency)
	}
	if !a.NotifyEmail {
		t.Error("expected NotifyEmail true")
	}
	if a.NotifyInApp {
		t.Error("expected NotifyInApp false")
	}
	if a.Name != "AAPL above 150" {
		t.Errorf("expected Name 'AAPL above 150', got %q", a.Name)
	}
	if a.LastTriggeredAt == nil || !a.LastTriggeredAt.Equal(lastTriggered) {
		t.Errorf("expected LastTriggeredAt %v, got %v", lastTriggered, a.LastTriggeredAt)
	}
	if a.TriggerCount != 3 {
		t.Errorf("expected TriggerCount 3, got %d", a.TriggerCount)
	}
	if !a.CreatedAt.Equal(now) {
		t.Errorf("expected CreatedAt %v, got %v", now, a.CreatedAt)
	}
	if !a.UpdatedAt.Equal(now) {
		t.Errorf("expected UpdatedAt %v, got %v", now, a.UpdatedAt)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected mock expectations: %v", err)
	}
}

func TestGetActiveAlertsForSymbols_QueryError(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectQuery("SELECT id, user_id, watch_list_id, symbol, alert_type, conditions").
		WithArgs("AAPL").
		WillReturnError(fmt.Errorf("connection refused"))

	alerts, err := db.GetActiveAlertsForSymbols([]string{"AAPL"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if alerts != nil {
		t.Fatalf("expected nil alerts on error, got %v", alerts)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected mock expectations: %v", err)
	}
}

func TestGetActiveAlertsForSymbols_ScanError(t *testing.T) {
	db, mock := newMockDB(t)

	// Return fewer columns than expected to trigger a scan error.
	rows := sqlmock.NewRows([]string{"id", "user_id"}).
		AddRow("alert-001", "user-123")

	mock.ExpectQuery("SELECT id, user_id, watch_list_id, symbol, alert_type, conditions").
		WithArgs("AAPL").
		WillReturnRows(rows)

	_, err := db.GetActiveAlertsForSymbols([]string{"AAPL"})
	if err == nil {
		t.Fatal("expected scan error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected mock expectations: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CreateAlertLog
// ---------------------------------------------------------------------------

func TestCreateAlertLog_Success(t *testing.T) {
	db, mock := newMockDB(t)

	alertLog := &models.AlertLog{
		AlertRuleID:      "alert-001",
		UserID:           "user-123",
		Symbol:           "AAPL",
		AlertType:        "price_above",
		ConditionMet:     json.RawMessage(`{"threshold":150.0}`),
		MarketData:       json.RawMessage(`{"price":155.0}`),
		NotificationSent: false,
	}

	rows := sqlmock.NewRows([]string{"id"}).AddRow("log-001")

	mock.ExpectQuery(regexp.QuoteMeta(
		`INSERT INTO alert_logs (alert_rule_id, user_id, symbol, alert_type,
		                        condition_met, market_data, notification_sent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
	)).
		WithArgs(
			"alert-001",
			"user-123",
			"AAPL",
			"price_above",
			json.RawMessage(`{"threshold":150.0}`),
			json.RawMessage(`{"price":155.0}`),
			false,
		).
		WillReturnRows(rows)

	id, err := db.CreateAlertLog(alertLog)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if id != "log-001" {
		t.Errorf("expected id 'log-001', got %q", id)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected mock expectations: %v", err)
	}
}

func TestCreateAlertLog_Error(t *testing.T) {
	db, mock := newMockDB(t)

	alertLog := &models.AlertLog{
		AlertRuleID:      "alert-001",
		UserID:           "user-123",
		Symbol:           "AAPL",
		AlertType:        "price_above",
		ConditionMet:     json.RawMessage(`{"threshold":150.0}`),
		MarketData:       json.RawMessage(`{"price":155.0}`),
		NotificationSent: false,
	}

	mock.ExpectQuery(regexp.QuoteMeta(
		`INSERT INTO alert_logs (alert_rule_id, user_id, symbol, alert_type,
		                        condition_met, market_data, notification_sent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
	)).
		WithArgs(
			"alert-001",
			"user-123",
			"AAPL",
			"price_above",
			json.RawMessage(`{"threshold":150.0}`),
			json.RawMessage(`{"price":155.0}`),
			false,
		).
		WillReturnError(fmt.Errorf("unique violation"))

	id, err := db.CreateAlertLog(alertLog)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if id != "" {
		t.Errorf("expected empty id on error, got %q", id)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected mock expectations: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ClaimAlertTrigger
// ---------------------------------------------------------------------------

func TestClaimAlertTrigger_Once_Claimed(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectExec("UPDATE alert_rules").
		WithArgs("alert-001").
		WillReturnResult(sqlmock.NewResult(0, 1))

	claimed, err := db.ClaimAlertTrigger("alert-001", "once")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !claimed {
		t.Error("expected claimed=true")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected mock expectations: %v", err)
	}
}

func TestClaimAlertTrigger_Once_NotClaimed(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectExec("UPDATE alert_rules").
		WithArgs("alert-001").
		WillReturnResult(sqlmock.NewResult(0, 0))

	claimed, err := db.ClaimAlertTrigger("alert-001", "once")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if claimed {
		t.Error("expected claimed=false when no rows affected")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected mock expectations: %v", err)
	}
}

func TestClaimAlertTrigger_Daily(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectExec("UPDATE alert_rules").
		WithArgs("alert-001").
		WillReturnResult(sqlmock.NewResult(0, 1))

	claimed, err := db.ClaimAlertTrigger("alert-001", "daily")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !claimed {
		t.Error("expected claimed=true for daily frequency")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected mock expectations: %v", err)
	}
}

func TestClaimAlertTrigger_Always(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectExec("UPDATE alert_rules").
		WithArgs("alert-001").
		WillReturnResult(sqlmock.NewResult(0, 1))

	claimed, err := db.ClaimAlertTrigger("alert-001", "always")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !claimed {
		t.Error("expected claimed=true for always frequency")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected mock expectations: %v", err)
	}
}

func TestClaimAlertTrigger_UnknownFrequency(t *testing.T) {
	db, _ := newMockDB(t)

	_, err := db.ClaimAlertTrigger("alert-001", "weekly")
	if err == nil {
		t.Fatal("expected error for unknown frequency, got nil")
	}
	if got := err.Error(); got != "unknown frequency: weekly" {
		t.Errorf("expected 'unknown frequency: weekly', got %q", got)
	}
}

func TestClaimAlertTrigger_ExecError(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectExec("UPDATE alert_rules").
		WithArgs("alert-001").
		WillReturnError(fmt.Errorf("deadlock detected"))

	claimed, err := db.ClaimAlertTrigger("alert-001", "once")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if claimed {
		t.Error("expected claimed=false on error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected mock expectations: %v", err)
	}
}

// ---------------------------------------------------------------------------
// UpdateAlertLogNotificationSent
// ---------------------------------------------------------------------------

func TestUpdateAlertLogNotificationSent_Success(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE alert_logs SET notification_sent = $1 WHERE id = $2`)).
		WithArgs(true, "log-001").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := db.UpdateAlertLogNotificationSent("log-001", true)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected mock expectations: %v", err)
	}
}

func TestUpdateAlertLogNotificationSent_Error(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE alert_logs SET notification_sent = $1 WHERE id = $2`)).
		WithArgs(true, "log-001").
		WillReturnError(fmt.Errorf("connection lost"))

	err := db.UpdateAlertLogNotificationSent("log-001", true)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected mock expectations: %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetTodayEmailCount
// ---------------------------------------------------------------------------

func TestGetTodayEmailCount_Success(t *testing.T) {
	db, mock := newMockDB(t)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(7)

	// The query uses two arguments: userID and todayStart().
	// We use sqlmock.AnyArg() for the time argument since todayStart() is
	// computed inside the function and we cannot predict the exact value.
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT COUNT(*) FROM alert_logs
		WHERE user_id = $1 AND triggered_at >= $2 AND notification_sent = true`,
	)).
		WithArgs("user-123", sqlmock.AnyArg()).
		WillReturnRows(rows)

	count, err := db.GetTodayEmailCount("user-123")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if count != 7 {
		t.Errorf("expected count 7, got %d", count)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected mock expectations: %v", err)
	}
}

func TestGetTodayEmailCount_Error(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT COUNT(*) FROM alert_logs
		WHERE user_id = $1 AND triggered_at >= $2 AND notification_sent = true`,
	)).
		WithArgs("user-123", sqlmock.AnyArg()).
		WillReturnError(fmt.Errorf("timeout"))

	count, err := db.GetTodayEmailCount("user-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if count != 0 {
		t.Errorf("expected count 0 on error, got %d", count)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected mock expectations: %v", err)
	}
}
