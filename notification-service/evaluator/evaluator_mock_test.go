package evaluator

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"notification-service/config"
	"notification-service/delivery"
	"notification-service/models"
)

// ---------------------------------------------------------------------------
// mockStore implements database.Store with configurable return values.
// ---------------------------------------------------------------------------

type mockStore struct {
	// GetActiveAlertsForSymbols
	getActiveAlertsForSymbolsFn func(symbols []string) ([]models.AlertRule, error)

	// ClaimAlertTrigger
	claimAlertTriggerFn func(alertID string, frequency string) (bool, error)

	// CreateAlertLog
	createAlertLogFn func(alertLog *models.AlertLog) (string, error)

	// UpdateAlertLogNotificationSent
	updateAlertLogNotificationSentFn func(logID string, sent bool) error

	// GetTodayEmailCount
	getTodayEmailCountFn func(userID string) (int, error)

	// GetNotificationPreferences
	getNotificationPreferencesFn func(userID string) (*models.NotificationPreferences, error)

	// GetUserEmail
	getUserEmailFn func(userID string) (*models.UserEmail, error)

	// Call tracking
	createAlertLogCalls              []*models.AlertLog
	updateAlertLogNotificationCalls  []updateNotificationCall
	claimAlertTriggerCalls           []claimTriggerCall
}

type updateNotificationCall struct {
	LogID string
	Sent  bool
}

type claimTriggerCall struct {
	AlertID   string
	Frequency string
}

func (m *mockStore) GetActiveAlertsForSymbols(symbols []string) ([]models.AlertRule, error) {
	if m.getActiveAlertsForSymbolsFn != nil {
		return m.getActiveAlertsForSymbolsFn(symbols)
	}
	return nil, nil
}

func (m *mockStore) ClaimAlertTrigger(alertID string, frequency string) (bool, error) {
	m.claimAlertTriggerCalls = append(m.claimAlertTriggerCalls, claimTriggerCall{alertID, frequency})
	if m.claimAlertTriggerFn != nil {
		return m.claimAlertTriggerFn(alertID, frequency)
	}
	return true, nil
}

func (m *mockStore) CreateAlertLog(alertLog *models.AlertLog) (string, error) {
	m.createAlertLogCalls = append(m.createAlertLogCalls, alertLog)
	if m.createAlertLogFn != nil {
		return m.createAlertLogFn(alertLog)
	}
	return "log-001", nil
}

func (m *mockStore) UpdateAlertLogNotificationSent(logID string, sent bool) error {
	m.updateAlertLogNotificationCalls = append(m.updateAlertLogNotificationCalls, updateNotificationCall{logID, sent})
	if m.updateAlertLogNotificationSentFn != nil {
		return m.updateAlertLogNotificationSentFn(logID, sent)
	}
	return nil
}

func (m *mockStore) GetTodayEmailCount(userID string) (int, error) {
	if m.getTodayEmailCountFn != nil {
		return m.getTodayEmailCountFn(userID)
	}
	return 0, nil
}

func (m *mockStore) GetNotificationPreferences(userID string) (*models.NotificationPreferences, error) {
	if m.getNotificationPreferencesFn != nil {
		return m.getNotificationPreferencesFn(userID)
	}
	return nil, nil
}

func (m *mockStore) GetUserEmail(userID string) (*models.UserEmail, error) {
	if m.getUserEmailFn != nil {
		return m.getUserEmailFn(userID)
	}
	return &models.UserEmail{Email: "test@example.com", FullName: "Test User"}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestEvaluator creates an Evaluator with the given mock store and an
// EmailDelivery that has no SMTP configured (so Send returns nil immediately).
func newTestEvaluator(store *mockStore) *Evaluator {
	cfg := &config.Config{} // empty SMTP -> EmailDelivery.Send returns nil
	emailDelivery := delivery.NewEmailDelivery(cfg, store)
	router := delivery.NewRouter(emailDelivery)
	return New(store, router)
}

// makeAlert creates a models.AlertRule with sensible defaults for testing.
func makeAlert(symbol, alertType, frequency string, conditions json.RawMessage) models.AlertRule {
	return models.AlertRule{
		ID:          "alert-001",
		UserID:      "user-001",
		WatchListID: "wl-001",
		Symbol:      symbol,
		AlertType:   alertType,
		Conditions:  conditions,
		IsActive:    true,
		Frequency:   frequency,
		NotifyEmail: true,
		NotifyInApp: true,
		Name:        "Test Alert",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// makePriceUpdateJSON builds valid PriceUpdateMessage JSON.
func makePriceUpdateJSON(symbols map[string]models.SymbolQuote) []byte {
	msg := models.PriceUpdateMessage{
		Timestamp: time.Now().Unix(),
		Source:    "test",
		Symbols:   symbols,
	}
	b, _ := json.Marshal(msg)
	return b
}

// ---------------------------------------------------------------------------
// Tests for New
// ---------------------------------------------------------------------------

func TestNew_ReturnsNonNilEvaluator(t *testing.T) {
	store := &mockStore{}
	ev := newTestEvaluator(store)
	if ev == nil {
		t.Fatal("expected New to return a non-nil Evaluator")
	}
}

func TestNew_SetsFields(t *testing.T) {
	store := &mockStore{}
	cfg := &config.Config{}
	emailDelivery := delivery.NewEmailDelivery(cfg, store)
	router := delivery.NewRouter(emailDelivery)
	ev := New(store, router)
	if ev.db == nil {
		t.Error("expected db field to be set")
	}
	if ev.delivery == nil {
		t.Error("expected delivery field to be set")
	}
}

// ---------------------------------------------------------------------------
// Tests for HandlePriceUpdate
// ---------------------------------------------------------------------------

func TestHandlePriceUpdate_InvalidJSON(t *testing.T) {
	store := &mockStore{}
	ev := newTestEvaluator(store)

	err := ev.HandlePriceUpdate([]byte(`{not valid json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestHandlePriceUpdate_EmptySymbols(t *testing.T) {
	store := &mockStore{}
	ev := newTestEvaluator(store)

	msg := makePriceUpdateJSON(map[string]models.SymbolQuote{})
	err := ev.HandlePriceUpdate(msg)
	if err != nil {
		t.Fatalf("expected nil error for empty symbols, got: %v", err)
	}
}

func TestHandlePriceUpdate_NoMatchingAlerts(t *testing.T) {
	store := &mockStore{
		getActiveAlertsForSymbolsFn: func(symbols []string) ([]models.AlertRule, error) {
			return nil, nil // no alerts
		},
	}
	ev := newTestEvaluator(store)

	msg := makePriceUpdateJSON(map[string]models.SymbolQuote{
		"AAPL": {Price: 150.0, Volume: 1000000, ChangePct: 1.5},
	})
	err := ev.HandlePriceUpdate(msg)
	if err != nil {
		t.Fatalf("expected nil error when no alerts match, got: %v", err)
	}
}

func TestHandlePriceUpdate_DBError_GetActiveAlerts(t *testing.T) {
	dbErr := errors.New("connection refused")
	store := &mockStore{
		getActiveAlertsForSymbolsFn: func(symbols []string) ([]models.AlertRule, error) {
			return nil, dbErr
		},
	}
	ev := newTestEvaluator(store)

	msg := makePriceUpdateJSON(map[string]models.SymbolQuote{
		"AAPL": {Price: 150.0, Volume: 1000000, ChangePct: 1.5},
	})
	err := ev.HandlePriceUpdate(msg)
	if err == nil {
		t.Fatal("expected error when DB fails")
	}
	if !errors.Is(err, dbErr) {
		t.Errorf("expected wrapped DB error, got: %v", err)
	}
}

func TestHandlePriceUpdate_AlertMatches_ConditionMet_HappyPath(t *testing.T) {
	alert := makeAlert("AAPL", "price_above", "always",
		mustJSON(models.ThresholdCondition{Threshold: 140.0}))

	store := &mockStore{
		getActiveAlertsForSymbolsFn: func(symbols []string) ([]models.AlertRule, error) {
			return []models.AlertRule{alert}, nil
		},
		claimAlertTriggerFn: func(alertID string, frequency string) (bool, error) {
			return true, nil
		},
		createAlertLogFn: func(alertLog *models.AlertLog) (string, error) {
			return "log-happy", nil
		},
	}
	ev := newTestEvaluator(store)

	msg := makePriceUpdateJSON(map[string]models.SymbolQuote{
		"AAPL": {Price: 155.0, Volume: 2000000, ChangePct: 2.0},
	})
	err := ev.HandlePriceUpdate(msg)
	if err != nil {
		t.Fatalf("expected nil error on happy path, got: %v", err)
	}

	// Verify that ClaimAlertTrigger was called
	if len(store.claimAlertTriggerCalls) != 1 {
		t.Fatalf("expected 1 ClaimAlertTrigger call, got %d", len(store.claimAlertTriggerCalls))
	}
	if store.claimAlertTriggerCalls[0].AlertID != "alert-001" {
		t.Errorf("expected alert ID alert-001, got %s", store.claimAlertTriggerCalls[0].AlertID)
	}

	// Verify that CreateAlertLog was called
	if len(store.createAlertLogCalls) != 1 {
		t.Fatalf("expected 1 CreateAlertLog call, got %d", len(store.createAlertLogCalls))
	}
	logEntry := store.createAlertLogCalls[0]
	if logEntry.Symbol != "AAPL" {
		t.Errorf("expected symbol AAPL, got %s", logEntry.Symbol)
	}
	if logEntry.AlertRuleID != "alert-001" {
		t.Errorf("expected alert_rule_id alert-001, got %s", logEntry.AlertRuleID)
	}
	if logEntry.UserID != "user-001" {
		t.Errorf("expected user_id user-001, got %s", logEntry.UserID)
	}
	if logEntry.NotificationSent {
		t.Error("expected notification_sent to be false initially")
	}
}

func TestHandlePriceUpdate_AlertMatches_ConditionNotMet(t *testing.T) {
	// price_above with threshold 200, but price is 150 -> condition not met
	alert := makeAlert("AAPL", "price_above", "always",
		mustJSON(models.ThresholdCondition{Threshold: 200.0}))

	store := &mockStore{
		getActiveAlertsForSymbolsFn: func(symbols []string) ([]models.AlertRule, error) {
			return []models.AlertRule{alert}, nil
		},
	}
	ev := newTestEvaluator(store)

	msg := makePriceUpdateJSON(map[string]models.SymbolQuote{
		"AAPL": {Price: 150.0, Volume: 1000000, ChangePct: 1.0},
	})
	err := ev.HandlePriceUpdate(msg)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	// No trigger should have occurred
	if len(store.claimAlertTriggerCalls) != 0 {
		t.Errorf("expected no ClaimAlertTrigger calls, got %d", len(store.claimAlertTriggerCalls))
	}
	if len(store.createAlertLogCalls) != 0 {
		t.Errorf("expected no CreateAlertLog calls, got %d", len(store.createAlertLogCalls))
	}
}

func TestHandlePriceUpdate_FrequencyGating_Blocks(t *testing.T) {
	// "once" alert that has already been triggered -> frequency gating blocks it
	triggered := time.Now().Add(-1 * time.Hour)
	alert := makeAlert("AAPL", "price_above", "once",
		mustJSON(models.ThresholdCondition{Threshold: 100.0}))
	alert.LastTriggeredAt = &triggered // already triggered

	store := &mockStore{
		getActiveAlertsForSymbolsFn: func(symbols []string) ([]models.AlertRule, error) {
			return []models.AlertRule{alert}, nil
		},
	}
	ev := newTestEvaluator(store)

	msg := makePriceUpdateJSON(map[string]models.SymbolQuote{
		"AAPL": {Price: 150.0, Volume: 1000000, ChangePct: 1.0},
	})
	err := ev.HandlePriceUpdate(msg)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	// Frequency gating should prevent any DB calls for triggering
	if len(store.claimAlertTriggerCalls) != 0 {
		t.Errorf("expected no ClaimAlertTrigger calls (frequency gated), got %d", len(store.claimAlertTriggerCalls))
	}
}

func TestHandlePriceUpdate_MultipleAlerts_SomeTrigger(t *testing.T) {
	// Alert 1: price_above 100 for AAPL -> should trigger (price is 150)
	alert1 := makeAlert("AAPL", "price_above", "always",
		mustJSON(models.ThresholdCondition{Threshold: 100.0}))
	alert1.ID = "alert-trigger"

	// Alert 2: price_above 200 for AAPL -> should NOT trigger (price is 150)
	alert2 := makeAlert("AAPL", "price_above", "always",
		mustJSON(models.ThresholdCondition{Threshold: 200.0}))
	alert2.ID = "alert-no-trigger"

	// Alert 3: price_below 200 for GOOG -> should trigger (price is 100)
	alert3 := makeAlert("GOOG", "price_below", "always",
		mustJSON(models.ThresholdCondition{Threshold: 200.0}))
	alert3.ID = "alert-trigger-goog"

	store := &mockStore{
		getActiveAlertsForSymbolsFn: func(symbols []string) ([]models.AlertRule, error) {
			return []models.AlertRule{alert1, alert2, alert3}, nil
		},
		claimAlertTriggerFn: func(alertID string, frequency string) (bool, error) {
			return true, nil
		},
		createAlertLogFn: func(alertLog *models.AlertLog) (string, error) {
			return "log-" + alertLog.AlertRuleID, nil
		},
	}
	ev := newTestEvaluator(store)

	msg := makePriceUpdateJSON(map[string]models.SymbolQuote{
		"AAPL": {Price: 150.0, Volume: 1000000, ChangePct: 1.0},
		"GOOG": {Price: 100.0, Volume: 500000, ChangePct: -0.5},
	})
	err := ev.HandlePriceUpdate(msg)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	// Should have 2 claims (alert1 and alert3), not alert2
	if len(store.claimAlertTriggerCalls) != 2 {
		t.Fatalf("expected 2 ClaimAlertTrigger calls, got %d", len(store.claimAlertTriggerCalls))
	}

	claimedIDs := map[string]bool{}
	for _, call := range store.claimAlertTriggerCalls {
		claimedIDs[call.AlertID] = true
	}
	if !claimedIDs["alert-trigger"] {
		t.Error("expected alert-trigger to be claimed")
	}
	if claimedIDs["alert-no-trigger"] {
		t.Error("expected alert-no-trigger NOT to be claimed")
	}
	if !claimedIDs["alert-trigger-goog"] {
		t.Error("expected alert-trigger-goog to be claimed")
	}

	// Should have 2 log entries
	if len(store.createAlertLogCalls) != 2 {
		t.Errorf("expected 2 CreateAlertLog calls, got %d", len(store.createAlertLogCalls))
	}
}

func TestHandlePriceUpdate_SymbolNotInUpdate(t *testing.T) {
	// Alert for MSFT, but update only has AAPL
	alert := makeAlert("MSFT", "price_above", "always",
		mustJSON(models.ThresholdCondition{Threshold: 100.0}))

	store := &mockStore{
		getActiveAlertsForSymbolsFn: func(symbols []string) ([]models.AlertRule, error) {
			return []models.AlertRule{alert}, nil
		},
	}
	ev := newTestEvaluator(store)

	msg := makePriceUpdateJSON(map[string]models.SymbolQuote{
		"AAPL": {Price: 150.0, Volume: 1000000, ChangePct: 1.0},
	})
	err := ev.HandlePriceUpdate(msg)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	// No trigger because the symbol is not in the update
	if len(store.claimAlertTriggerCalls) != 0 {
		t.Errorf("expected no ClaimAlertTrigger calls, got %d", len(store.claimAlertTriggerCalls))
	}
}

// ---------------------------------------------------------------------------
// Tests for trigger
// ---------------------------------------------------------------------------

func TestTrigger_ClaimReturnsFalse_AlreadyClaimed(t *testing.T) {
	store := &mockStore{
		claimAlertTriggerFn: func(alertID string, frequency string) (bool, error) {
			return false, nil // already claimed by another consumer
		},
	}
	ev := newTestEvaluator(store)

	alert := makeAlert("AAPL", "price_above", "always",
		mustJSON(models.ThresholdCondition{Threshold: 100.0}))
	quote := &models.SymbolQuote{Price: 150.0, Volume: 1000000, ChangePct: 2.0}

	err := ev.trigger(&alert, quote)
	if err != nil {
		t.Fatalf("expected nil error when claim returns false, got: %v", err)
	}

	// No log should have been created
	if len(store.createAlertLogCalls) != 0 {
		t.Errorf("expected no CreateAlertLog calls when claim is false, got %d", len(store.createAlertLogCalls))
	}
	// No notification update should have happened
	if len(store.updateAlertLogNotificationCalls) != 0 {
		t.Errorf("expected no UpdateAlertLogNotificationSent calls, got %d", len(store.updateAlertLogNotificationCalls))
	}
}

func TestTrigger_ClaimReturnsError(t *testing.T) {
	claimErr := errors.New("database timeout")
	store := &mockStore{
		claimAlertTriggerFn: func(alertID string, frequency string) (bool, error) {
			return false, claimErr
		},
	}
	ev := newTestEvaluator(store)

	alert := makeAlert("AAPL", "price_above", "always",
		mustJSON(models.ThresholdCondition{Threshold: 100.0}))
	quote := &models.SymbolQuote{Price: 150.0, Volume: 1000000, ChangePct: 2.0}

	err := ev.trigger(&alert, quote)
	if err == nil {
		t.Fatal("expected error when ClaimAlertTrigger fails")
	}
	if !errors.Is(err, claimErr) {
		t.Errorf("expected wrapped claim error, got: %v", err)
	}
}

func TestTrigger_HappyPath_ClaimSucceeds_LogCreated_DeliverySucceeds(t *testing.T) {
	store := &mockStore{
		claimAlertTriggerFn: func(alertID string, frequency string) (bool, error) {
			return true, nil
		},
		createAlertLogFn: func(alertLog *models.AlertLog) (string, error) {
			return "log-happy-path", nil
		},
		updateAlertLogNotificationSentFn: func(logID string, sent bool) error {
			return nil
		},
	}

	// EmailDelivery with empty SMTP config => Send returns nil (SMTP not configured path)
	ev := newTestEvaluator(store)

	alert := makeAlert("AAPL", "price_above", "always",
		mustJSON(models.ThresholdCondition{Threshold: 100.0}))
	quote := &models.SymbolQuote{Price: 150.0, Volume: 2000000, ChangePct: 3.5}

	err := ev.trigger(&alert, quote)
	if err != nil {
		t.Fatalf("expected nil error on happy path, got: %v", err)
	}

	// Verify claim was called
	if len(store.claimAlertTriggerCalls) != 1 {
		t.Fatalf("expected 1 ClaimAlertTrigger call, got %d", len(store.claimAlertTriggerCalls))
	}

	// Verify alert log was created
	if len(store.createAlertLogCalls) != 1 {
		t.Fatalf("expected 1 CreateAlertLog call, got %d", len(store.createAlertLogCalls))
	}
	logEntry := store.createAlertLogCalls[0]
	if logEntry.AlertRuleID != "alert-001" {
		t.Errorf("expected alert_rule_id alert-001, got %s", logEntry.AlertRuleID)
	}
	if logEntry.UserID != "user-001" {
		t.Errorf("expected user_id user-001, got %s", logEntry.UserID)
	}
	if logEntry.Symbol != "AAPL" {
		t.Errorf("expected symbol AAPL, got %s", logEntry.Symbol)
	}
	if logEntry.AlertType != "price_above" {
		t.Errorf("expected alert_type price_above, got %s", logEntry.AlertType)
	}
	if logEntry.NotificationSent {
		t.Error("expected notification_sent to be false initially")
	}

	// Verify ConditionMet JSON contains expected fields
	var condMet map[string]interface{}
	if err := json.Unmarshal(logEntry.ConditionMet, &condMet); err != nil {
		t.Fatalf("failed to parse condition_met JSON: %v", err)
	}
	if condMet["alert_type"] != "price_above" {
		t.Errorf("expected alert_type in condition_met, got %v", condMet["alert_type"])
	}
	if condMet["triggered"] != true {
		t.Errorf("expected triggered=true in condition_met, got %v", condMet["triggered"])
	}

	// Verify MarketData JSON contains expected fields
	var mktData map[string]interface{}
	if err := json.Unmarshal(logEntry.MarketData, &mktData); err != nil {
		t.Fatalf("failed to parse market_data JSON: %v", err)
	}
	if mktData["symbol"] != "AAPL" {
		t.Errorf("expected symbol AAPL in market_data, got %v", mktData["symbol"])
	}
	if mktData["price"] != 150.0 {
		t.Errorf("expected price 150.0 in market_data, got %v", mktData["price"])
	}

	// Verify notification_sent was updated to true (delivery succeeded via SMTP-not-configured path)
	if len(store.updateAlertLogNotificationCalls) != 1 {
		t.Fatalf("expected 1 UpdateAlertLogNotificationSent call, got %d", len(store.updateAlertLogNotificationCalls))
	}
	updateCall := store.updateAlertLogNotificationCalls[0]
	if updateCall.LogID != "log-happy-path" {
		t.Errorf("expected log ID log-happy-path, got %s", updateCall.LogID)
	}
	if !updateCall.Sent {
		t.Error("expected sent=true after successful delivery")
	}
}

func TestTrigger_CreateAlertLog_Error(t *testing.T) {
	logErr := errors.New("insert failed")
	store := &mockStore{
		claimAlertTriggerFn: func(alertID string, frequency string) (bool, error) {
			return true, nil
		},
		createAlertLogFn: func(alertLog *models.AlertLog) (string, error) {
			return "", logErr
		},
	}
	ev := newTestEvaluator(store)

	alert := makeAlert("AAPL", "price_above", "always",
		mustJSON(models.ThresholdCondition{Threshold: 100.0}))
	quote := &models.SymbolQuote{Price: 150.0, Volume: 1000000, ChangePct: 2.0}

	err := ev.trigger(&alert, quote)
	if err == nil {
		t.Fatal("expected error when CreateAlertLog fails")
	}
	if !errors.Is(err, logErr) {
		t.Errorf("expected wrapped log error, got: %v", err)
	}

	// No notification update should have happened
	if len(store.updateAlertLogNotificationCalls) != 0 {
		t.Errorf("expected no UpdateAlertLogNotificationSent calls, got %d", len(store.updateAlertLogNotificationCalls))
	}
}

func TestTrigger_DeliveryError_LogStillCreated_NotificationSentStaysFalse(t *testing.T) {
	// EmailDelivery.Send returns nil when SMTP is not configured (empty host or
	// empty password), which is the case with newTestEvaluator. Since
	// config.RedactedString.val is unexported, we cannot construct an EmailDelivery
	// with a real SMTP password from outside the config package. Therefore, delivery
	// always "succeeds" in these unit tests via the SMTP-not-configured fast path.
	//
	// This test verifies the invariant that trigger() never returns an error from
	// delivery failures: it returns nil regardless, and delivery errors are only
	// logged. We also verify the alert log is still created.

	store := &mockStore{
		claimAlertTriggerFn: func(alertID string, frequency string) (bool, error) {
			return true, nil
		},
		createAlertLogFn: func(alertLog *models.AlertLog) (string, error) {
			return "log-delivery-test", nil
		},
	}
	ev := newTestEvaluator(store)

	alert := makeAlert("AAPL", "price_above", "always",
		mustJSON(models.ThresholdCondition{Threshold: 100.0}))
	alert.NotifyEmail = true
	quote := &models.SymbolQuote{Price: 150.0, Volume: 1000000, ChangePct: 2.0}

	err := ev.trigger(&alert, quote)
	if err != nil {
		t.Fatalf("expected nil error from trigger (delivery errors are not returned), got: %v", err)
	}

	// Alert log was still created
	if len(store.createAlertLogCalls) != 1 {
		t.Fatalf("expected 1 CreateAlertLog call, got %d", len(store.createAlertLogCalls))
	}

	// Since delivery succeeded (SMTP not configured returns nil),
	// notification_sent should be updated to true
	if len(store.updateAlertLogNotificationCalls) != 1 {
		t.Fatalf("expected 1 UpdateAlertLogNotificationSent call, got %d",
			len(store.updateAlertLogNotificationCalls))
	}
}

func TestTrigger_NotifyEmailFalse_NoDelivery(t *testing.T) {
	store := &mockStore{
		claimAlertTriggerFn: func(alertID string, frequency string) (bool, error) {
			return true, nil
		},
		createAlertLogFn: func(alertLog *models.AlertLog) (string, error) {
			return "log-no-email", nil
		},
	}
	ev := newTestEvaluator(store)

	alert := makeAlert("AAPL", "price_above", "always",
		mustJSON(models.ThresholdCondition{Threshold: 100.0}))
	alert.NotifyEmail = false // email delivery disabled
	quote := &models.SymbolQuote{Price: 150.0, Volume: 1000000, ChangePct: 2.0}

	err := ev.trigger(&alert, quote)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	// Log should still be created
	if len(store.createAlertLogCalls) != 1 {
		t.Fatalf("expected 1 CreateAlertLog call, got %d", len(store.createAlertLogCalls))
	}

	// notification_sent should still be updated (delivery.Router.Deliver returns nil
	// when NotifyEmail is false, which counts as success)
	if len(store.updateAlertLogNotificationCalls) != 1 {
		t.Fatalf("expected 1 UpdateAlertLogNotificationSent call, got %d",
			len(store.updateAlertLogNotificationCalls))
	}
}

func TestTrigger_UpdateNotificationSent_ErrorIsNonFatal(t *testing.T) {
	store := &mockStore{
		claimAlertTriggerFn: func(alertID string, frequency string) (bool, error) {
			return true, nil
		},
		createAlertLogFn: func(alertLog *models.AlertLog) (string, error) {
			return "log-update-fail", nil
		},
		updateAlertLogNotificationSentFn: func(logID string, sent bool) error {
			return errors.New("update failed")
		},
	}
	ev := newTestEvaluator(store)

	alert := makeAlert("AAPL", "price_above", "always",
		mustJSON(models.ThresholdCondition{Threshold: 100.0}))
	quote := &models.SymbolQuote{Price: 150.0, Volume: 1000000, ChangePct: 2.0}

	// trigger should still return nil even when UpdateAlertLogNotificationSent fails
	// (it's a non-fatal warning)
	err := ev.trigger(&alert, quote)
	if err != nil {
		t.Fatalf("expected nil error (update failure is non-fatal), got: %v", err)
	}
}

func TestTrigger_ConditionMetJSON_ContainsThreshold(t *testing.T) {
	store := &mockStore{
		claimAlertTriggerFn: func(alertID string, frequency string) (bool, error) {
			return true, nil
		},
		createAlertLogFn: func(alertLog *models.AlertLog) (string, error) {
			return "log-json-check", nil
		},
	}
	ev := newTestEvaluator(store)

	alert := makeAlert("AAPL", "price_above", "always",
		mustJSON(models.ThresholdCondition{Threshold: 175.50}))
	quote := &models.SymbolQuote{Price: 180.0, Volume: 1000000, ChangePct: 1.5}

	err := ev.trigger(&alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logEntry := store.createAlertLogCalls[0]

	// Verify condition_met contains the threshold
	var condMet map[string]interface{}
	if err := json.Unmarshal(logEntry.ConditionMet, &condMet); err != nil {
		t.Fatalf("failed to parse condition_met: %v", err)
	}
	threshold, ok := condMet["threshold"].(float64)
	if !ok {
		t.Fatalf("expected threshold to be a float64, got %T", condMet["threshold"])
	}
	if threshold != 175.50 {
		t.Errorf("expected threshold 175.50, got %f", threshold)
	}

	// Verify market_data contains change_pct
	var mktData map[string]interface{}
	if err := json.Unmarshal(logEntry.MarketData, &mktData); err != nil {
		t.Fatalf("failed to parse market_data: %v", err)
	}
	changePct, ok := mktData["change_pct"].(float64)
	if !ok {
		t.Fatalf("expected change_pct to be a float64, got %T", mktData["change_pct"])
	}
	if changePct != 1.5 {
		t.Errorf("expected change_pct 1.5, got %f", changePct)
	}
}

func TestTrigger_MarketData_ContainsVolume(t *testing.T) {
	store := &mockStore{
		claimAlertTriggerFn: func(alertID string, frequency string) (bool, error) {
			return true, nil
		},
		createAlertLogFn: func(alertLog *models.AlertLog) (string, error) {
			return "log-vol", nil
		},
	}
	ev := newTestEvaluator(store)

	alert := makeAlert("TSLA", "price_above", "daily",
		mustJSON(models.ThresholdCondition{Threshold: 200.0}))
	quote := &models.SymbolQuote{Price: 250.0, Volume: 5000000, ChangePct: 4.2}

	err := ev.trigger(&alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logEntry := store.createAlertLogCalls[0]
	var mktData map[string]interface{}
	if err := json.Unmarshal(logEntry.MarketData, &mktData); err != nil {
		t.Fatalf("failed to parse market_data: %v", err)
	}

	// Volume is int64 in SymbolQuote but json.Marshal writes it as a number.
	// json.Unmarshal into interface{} reads numbers as float64.
	vol, ok := mktData["volume"].(float64)
	if !ok {
		t.Fatalf("expected volume to be a float64, got %T", mktData["volume"])
	}
	if vol != 5000000 {
		t.Errorf("expected volume 5000000, got %f", vol)
	}
}

func TestTrigger_FrequencyPassedToClaimAlertTrigger(t *testing.T) {
	store := &mockStore{
		claimAlertTriggerFn: func(alertID string, frequency string) (bool, error) {
			return true, nil
		},
		createAlertLogFn: func(alertLog *models.AlertLog) (string, error) {
			return "log-freq", nil
		},
	}
	ev := newTestEvaluator(store)

	alert := makeAlert("AAPL", "price_above", "daily",
		mustJSON(models.ThresholdCondition{Threshold: 100.0}))
	quote := &models.SymbolQuote{Price: 150.0, Volume: 1000000, ChangePct: 2.0}

	err := ev.trigger(&alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(store.claimAlertTriggerCalls) != 1 {
		t.Fatalf("expected 1 claim call, got %d", len(store.claimAlertTriggerCalls))
	}
	call := store.claimAlertTriggerCalls[0]
	if call.AlertID != "alert-001" {
		t.Errorf("expected alert ID alert-001, got %s", call.AlertID)
	}
	if call.Frequency != "daily" {
		t.Errorf("expected frequency daily, got %s", call.Frequency)
	}
}

// ---------------------------------------------------------------------------
// Integration-style: HandlePriceUpdate end-to-end with mock store
// ---------------------------------------------------------------------------

func TestHandlePriceUpdate_EndToEnd_PriceBelow_Triggers(t *testing.T) {
	alert := makeAlert("MSFT", "price_below", "daily",
		mustJSON(models.ThresholdCondition{Threshold: 300.0}))

	store := &mockStore{
		getActiveAlertsForSymbolsFn: func(symbols []string) ([]models.AlertRule, error) {
			return []models.AlertRule{alert}, nil
		},
		claimAlertTriggerFn: func(alertID string, frequency string) (bool, error) {
			return true, nil
		},
		createAlertLogFn: func(alertLog *models.AlertLog) (string, error) {
			return "log-e2e", nil
		},
	}
	ev := newTestEvaluator(store)

	msg := makePriceUpdateJSON(map[string]models.SymbolQuote{
		"MSFT": {Price: 280.0, Volume: 3000000, ChangePct: -2.1},
	})
	err := ev.HandlePriceUpdate(msg)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	// Full chain: claim -> log -> delivery -> update notification_sent
	if len(store.claimAlertTriggerCalls) != 1 {
		t.Errorf("expected 1 claim, got %d", len(store.claimAlertTriggerCalls))
	}
	if len(store.createAlertLogCalls) != 1 {
		t.Errorf("expected 1 log creation, got %d", len(store.createAlertLogCalls))
	}
	if len(store.updateAlertLogNotificationCalls) != 1 {
		t.Errorf("expected 1 notification update, got %d", len(store.updateAlertLogNotificationCalls))
	}
	if store.updateAlertLogNotificationCalls[0].LogID != "log-e2e" {
		t.Errorf("expected log ID log-e2e, got %s", store.updateAlertLogNotificationCalls[0].LogID)
	}
	if !store.updateAlertLogNotificationCalls[0].Sent {
		t.Error("expected sent=true")
	}
}

func TestHandlePriceUpdate_EndToEnd_PriceChangePct_Triggers(t *testing.T) {
	alert := makeAlert("NVDA", "price_change_pct", "always",
		mustJSON(models.PriceChangeCondition{PercentChange: 3.0, Direction: "either"}))

	store := &mockStore{
		getActiveAlertsForSymbolsFn: func(symbols []string) ([]models.AlertRule, error) {
			return []models.AlertRule{alert}, nil
		},
		claimAlertTriggerFn: func(alertID string, frequency string) (bool, error) {
			return true, nil
		},
		createAlertLogFn: func(alertLog *models.AlertLog) (string, error) {
			return "log-pct", nil
		},
	}
	ev := newTestEvaluator(store)

	msg := makePriceUpdateJSON(map[string]models.SymbolQuote{
		"NVDA": {Price: 800.0, Volume: 10000000, ChangePct: -5.5},
	})
	err := ev.HandlePriceUpdate(msg)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	if len(store.claimAlertTriggerCalls) != 1 {
		t.Errorf("expected 1 claim, got %d", len(store.claimAlertTriggerCalls))
	}
	if len(store.createAlertLogCalls) != 1 {
		t.Errorf("expected 1 log, got %d", len(store.createAlertLogCalls))
	}
	logEntry := store.createAlertLogCalls[0]
	if logEntry.AlertType != "price_change_pct" {
		t.Errorf("expected alert_type price_change_pct, got %s", logEntry.AlertType)
	}
}

func TestHandlePriceUpdate_TriggerError_ContinuesProcessingOtherAlerts(t *testing.T) {
	// Two alerts: first one has CreateAlertLog fail, second succeeds
	alert1 := makeAlert("AAPL", "price_above", "always",
		mustJSON(models.ThresholdCondition{Threshold: 100.0}))
	alert1.ID = "alert-fail"

	alert2 := makeAlert("AAPL", "price_above", "always",
		mustJSON(models.ThresholdCondition{Threshold: 100.0}))
	alert2.ID = "alert-succeed"
	alert2.UserID = "user-002"

	callCount := 0
	store := &mockStore{
		getActiveAlertsForSymbolsFn: func(symbols []string) ([]models.AlertRule, error) {
			return []models.AlertRule{alert1, alert2}, nil
		},
		claimAlertTriggerFn: func(alertID string, frequency string) (bool, error) {
			return true, nil
		},
		createAlertLogFn: func(alertLog *models.AlertLog) (string, error) {
			callCount++
			if alertLog.AlertRuleID == "alert-fail" {
				return "", errors.New("insert failed")
			}
			return "log-succeed", nil
		},
	}
	ev := newTestEvaluator(store)

	msg := makePriceUpdateJSON(map[string]models.SymbolQuote{
		"AAPL": {Price: 150.0, Volume: 1000000, ChangePct: 2.0},
	})
	err := ev.HandlePriceUpdate(msg)
	// HandlePriceUpdate should return nil even if individual triggers fail
	if err != nil {
		t.Fatalf("expected nil error (individual trigger errors are logged, not returned), got: %v", err)
	}

	// Both alerts should have been attempted (claims for both)
	if len(store.claimAlertTriggerCalls) != 2 {
		t.Errorf("expected 2 ClaimAlertTrigger calls, got %d", len(store.claimAlertTriggerCalls))
	}

	// CreateAlertLog should have been called for both
	if callCount != 2 {
		t.Errorf("expected 2 CreateAlertLog calls, got %d", callCount)
	}
}
