package evaluator

import (
	"encoding/json"
	"testing"

	"notification-service/models"
)

// ---------------------------------------------------------------------------
// evaluateVolumeAbove
// ---------------------------------------------------------------------------

func TestVolumeAbove_AtThreshold(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 1000000}),
	}
	quote := &models.SymbolQuote{Volume: 1000000}
	triggered, err := evaluateVolumeAbove(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !triggered {
		t.Error("expected trigger when volume equals threshold")
	}
}

func TestVolumeAbove_Above(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 1000000}),
	}
	quote := &models.SymbolQuote{Volume: 2000000}
	triggered, err := evaluateVolumeAbove(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !triggered {
		t.Error("expected trigger when volume > threshold")
	}
}

func TestVolumeAbove_Below(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 1000000}),
	}
	quote := &models.SymbolQuote{Volume: 500000}
	triggered, err := evaluateVolumeAbove(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triggered {
		t.Error("expected no trigger when volume < threshold")
	}
}

func TestVolumeAbove_InvalidJSON(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: json.RawMessage(`{invalid`),
	}
	quote := &models.SymbolQuote{Volume: 2000000}
	_, err := evaluateVolumeAbove(alert, quote)
	if err == nil {
		t.Error("expected error for invalid JSON conditions")
	}
}

func TestVolumeAbove_ZeroThreshold(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 0}),
	}
	quote := &models.SymbolQuote{Volume: 100}
	_, err := evaluateVolumeAbove(alert, quote)
	if err == nil {
		t.Error("expected error for zero threshold")
	}
}

// ---------------------------------------------------------------------------
// evaluateVolumeBelow
// ---------------------------------------------------------------------------

func TestVolumeBelow_AtThreshold(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 500000}),
	}
	quote := &models.SymbolQuote{Volume: 500000}
	triggered, err := evaluateVolumeBelow(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !triggered {
		t.Error("expected trigger when volume equals threshold")
	}
}

func TestVolumeBelow_Below(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 500000}),
	}
	quote := &models.SymbolQuote{Volume: 100000}
	triggered, err := evaluateVolumeBelow(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !triggered {
		t.Error("expected trigger when volume < threshold")
	}
}

func TestVolumeBelow_Above(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 500000}),
	}
	quote := &models.SymbolQuote{Volume: 600000}
	triggered, err := evaluateVolumeBelow(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triggered {
		t.Error("expected no trigger when volume > threshold")
	}
}

func TestVolumeBelow_InvalidJSON(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: json.RawMessage(`{invalid`),
	}
	quote := &models.SymbolQuote{Volume: 100}
	_, err := evaluateVolumeBelow(alert, quote)
	if err == nil {
		t.Error("expected error for invalid JSON conditions")
	}
}

func TestVolumeBelow_ZeroThreshold(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 0}),
	}
	quote := &models.SymbolQuote{Volume: 100}
	_, err := evaluateVolumeBelow(alert, quote)
	if err == nil {
		t.Error("expected error for zero threshold")
	}
}
