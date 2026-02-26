package evaluator

import (
	"encoding/json"
	"testing"
	"time"

	"notification-service/models"
)

// ---------------------------------------------------------------------------
// shouldTriggerBasedOnFrequency
// ---------------------------------------------------------------------------

func TestShouldTrigger_Once_NeverTriggered(t *testing.T) {
	alert := &models.AlertRule{Frequency: "once", LastTriggeredAt: nil}
	if !shouldTriggerBasedOnFrequency(alert) {
		t.Error("expected once-alert with no prior trigger to fire")
	}
}

func TestShouldTrigger_Once_AlreadyTriggered(t *testing.T) {
	now := time.Now()
	alert := &models.AlertRule{Frequency: "once", LastTriggeredAt: &now}
	if shouldTriggerBasedOnFrequency(alert) {
		t.Error("expected once-alert with prior trigger NOT to fire")
	}
}

func TestShouldTrigger_Daily_NeverTriggered(t *testing.T) {
	alert := &models.AlertRule{Frequency: "daily", LastTriggeredAt: nil}
	if !shouldTriggerBasedOnFrequency(alert) {
		t.Error("expected daily-alert with no prior trigger to fire")
	}
}

func TestShouldTrigger_Daily_TriggeredWithin24h(t *testing.T) {
	recent := time.Now().Add(-12 * time.Hour)
	alert := &models.AlertRule{Frequency: "daily", LastTriggeredAt: &recent}
	if shouldTriggerBasedOnFrequency(alert) {
		t.Error("expected daily-alert triggered 12h ago NOT to fire")
	}
}

func TestShouldTrigger_Daily_TriggeredOver24hAgo(t *testing.T) {
	old := time.Now().Add(-25 * time.Hour)
	alert := &models.AlertRule{Frequency: "daily", LastTriggeredAt: &old}
	if !shouldTriggerBasedOnFrequency(alert) {
		t.Error("expected daily-alert triggered 25h ago to fire")
	}
}

func TestShouldTrigger_Always_NeverTriggered(t *testing.T) {
	alert := &models.AlertRule{Frequency: "always", LastTriggeredAt: nil}
	if !shouldTriggerBasedOnFrequency(alert) {
		t.Error("expected always-alert with no prior trigger to fire")
	}
}

func TestShouldTrigger_Always_TriggeredRecently(t *testing.T) {
	recent := time.Now().Add(-2 * time.Minute)
	alert := &models.AlertRule{Frequency: "always", LastTriggeredAt: &recent}
	if shouldTriggerBasedOnFrequency(alert) {
		t.Error("expected always-alert triggered 2min ago NOT to fire (5min cooldown)")
	}
}

func TestShouldTrigger_Always_TriggeredOver5minAgo(t *testing.T) {
	old := time.Now().Add(-6 * time.Minute)
	alert := &models.AlertRule{Frequency: "always", LastTriggeredAt: &old}
	if !shouldTriggerBasedOnFrequency(alert) {
		t.Error("expected always-alert triggered 6min ago to fire")
	}
}

func TestShouldTrigger_UnknownFrequency(t *testing.T) {
	alert := &models.AlertRule{Frequency: "unknown"}
	if shouldTriggerBasedOnFrequency(alert) {
		t.Error("expected unknown frequency to NOT fire")
	}
}

// ---------------------------------------------------------------------------
// evaluate (dispatcher)
// ---------------------------------------------------------------------------

func mustJSON(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func TestEvaluate_PriceAbove_Triggered(t *testing.T) {
	alert := &models.AlertRule{
		AlertType:  "price_above",
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 150.0}),
	}
	quote := &models.SymbolQuote{Price: 155.0}
	triggered, err := evaluate(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !triggered {
		t.Error("expected price_above to trigger when price >= threshold")
	}
}

func TestEvaluate_PriceAbove_NotTriggered(t *testing.T) {
	alert := &models.AlertRule{
		AlertType:  "price_above",
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 150.0}),
	}
	quote := &models.SymbolQuote{Price: 149.0}
	triggered, err := evaluate(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triggered {
		t.Error("expected price_above NOT to trigger when price < threshold")
	}
}

func TestEvaluate_PriceBelow_Triggered(t *testing.T) {
	alert := &models.AlertRule{
		AlertType:  "price_below",
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 100.0}),
	}
	quote := &models.SymbolQuote{Price: 95.0}
	triggered, err := evaluate(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !triggered {
		t.Error("expected price_below to trigger when price <= threshold")
	}
}

func TestEvaluate_PriceBelow_NotTriggered(t *testing.T) {
	alert := &models.AlertRule{
		AlertType:  "price_below",
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 100.0}),
	}
	quote := &models.SymbolQuote{Price: 105.0}
	triggered, err := evaluate(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triggered {
		t.Error("expected price_below NOT to trigger when price > threshold")
	}
}

func TestEvaluate_UnknownType(t *testing.T) {
	alert := &models.AlertRule{
		AlertType:  "news",
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 1.0}),
	}
	quote := &models.SymbolQuote{Price: 100.0}
	triggered, err := evaluate(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triggered {
		t.Error("expected unknown alert type to NOT trigger")
	}
}

// ---------------------------------------------------------------------------
// getThreshold
// ---------------------------------------------------------------------------

func TestGetThreshold_ThresholdCondition(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 150.0}),
	}
	got := getThreshold(alert)
	if got != 150.0 {
		t.Errorf("expected 150.0, got %f", got)
	}
}

func TestGetThreshold_VolumeSpikeCondition(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.VolumeSpikeCondition{VolumeMultiplier: 2.5, Baseline: "avg_30d"}),
	}
	got := getThreshold(alert)
	if got != 2.5 {
		t.Errorf("expected 2.5, got %f", got)
	}
}

func TestGetThreshold_InvalidJSON(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: json.RawMessage(`{invalid`),
	}
	got := getThreshold(alert)
	if got != 0 {
		t.Errorf("expected 0 for invalid JSON, got %f", got)
	}
}
