package evaluator

import (
	"encoding/json"
	"testing"

	"notification-service/models"
)

// ---------------------------------------------------------------------------
// evaluatePriceAbove
// ---------------------------------------------------------------------------

func TestPriceAbove_AtThreshold(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 150.0}),
	}
	quote := &models.SymbolQuote{Price: 150.0}
	triggered, err := evaluatePriceAbove(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !triggered {
		t.Error("expected trigger when price equals threshold")
	}
}

func TestPriceAbove_AboveThreshold(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 150.0}),
	}
	quote := &models.SymbolQuote{Price: 200.0}
	triggered, err := evaluatePriceAbove(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !triggered {
		t.Error("expected trigger when price > threshold")
	}
}

func TestPriceAbove_BelowThreshold(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 150.0}),
	}
	quote := &models.SymbolQuote{Price: 149.99}
	triggered, err := evaluatePriceAbove(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triggered {
		t.Error("expected no trigger when price < threshold")
	}
}

func TestPriceAbove_InvalidJSON(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: json.RawMessage(`{invalid`),
	}
	quote := &models.SymbolQuote{Price: 200.0}
	_, err := evaluatePriceAbove(alert, quote)
	if err == nil {
		t.Error("expected error for invalid JSON conditions")
	}
}

func TestPriceAbove_ZeroThreshold(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 0}),
	}
	quote := &models.SymbolQuote{Price: 200.0}
	_, err := evaluatePriceAbove(alert, quote)
	if err == nil {
		t.Error("expected error for zero threshold")
	}
}

func TestPriceAbove_NegativeThreshold(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.ThresholdCondition{Threshold: -10}),
	}
	quote := &models.SymbolQuote{Price: 200.0}
	_, err := evaluatePriceAbove(alert, quote)
	if err == nil {
		t.Error("expected error for negative threshold")
	}
}

// ---------------------------------------------------------------------------
// evaluatePriceBelow
// ---------------------------------------------------------------------------

func TestPriceBelow_AtThreshold(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 100.0}),
	}
	quote := &models.SymbolQuote{Price: 100.0}
	triggered, err := evaluatePriceBelow(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !triggered {
		t.Error("expected trigger when price equals threshold")
	}
}

func TestPriceBelow_BelowThreshold(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 100.0}),
	}
	quote := &models.SymbolQuote{Price: 50.0}
	triggered, err := evaluatePriceBelow(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !triggered {
		t.Error("expected trigger when price < threshold")
	}
}

func TestPriceBelow_AboveThreshold(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 100.0}),
	}
	quote := &models.SymbolQuote{Price: 100.01}
	triggered, err := evaluatePriceBelow(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triggered {
		t.Error("expected no trigger when price > threshold")
	}
}

func TestPriceBelow_InvalidJSON(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: json.RawMessage(`{invalid`),
	}
	quote := &models.SymbolQuote{Price: 50.0}
	_, err := evaluatePriceBelow(alert, quote)
	if err == nil {
		t.Error("expected error for invalid JSON conditions")
	}
}

func TestPriceBelow_ZeroThreshold(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 0}),
	}
	quote := &models.SymbolQuote{Price: 50.0}
	_, err := evaluatePriceBelow(alert, quote)
	if err == nil {
		t.Error("expected error for zero threshold")
	}
}

// ---------------------------------------------------------------------------
// evaluatePriceChangePct
// ---------------------------------------------------------------------------

func TestPriceChangePct_Up_Triggered(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.PriceChangeCondition{PercentChange: 5.0, Direction: "up"}),
	}
	quote := &models.SymbolQuote{ChangePct: 6.0}
	triggered, err := evaluatePriceChangePct(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !triggered {
		t.Error("expected trigger for upward move >= threshold")
	}
}

func TestPriceChangePct_Up_NotTriggered_BelowThreshold(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.PriceChangeCondition{PercentChange: 5.0, Direction: "up"}),
	}
	quote := &models.SymbolQuote{ChangePct: 3.0}
	triggered, err := evaluatePriceChangePct(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triggered {
		t.Error("expected no trigger for upward move below threshold")
	}
}

func TestPriceChangePct_Up_NotTriggered_NegativeChange(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.PriceChangeCondition{PercentChange: 5.0, Direction: "up"}),
	}
	quote := &models.SymbolQuote{ChangePct: -6.0}
	triggered, err := evaluatePriceChangePct(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triggered {
		t.Error("expected no trigger for negative change with direction=up")
	}
}

func TestPriceChangePct_Down_Triggered(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.PriceChangeCondition{PercentChange: 5.0, Direction: "down"}),
	}
	quote := &models.SymbolQuote{ChangePct: -6.0}
	triggered, err := evaluatePriceChangePct(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !triggered {
		t.Error("expected trigger for downward move >= threshold")
	}
}

func TestPriceChangePct_Down_NotTriggered_PositiveChange(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.PriceChangeCondition{PercentChange: 5.0, Direction: "down"}),
	}
	quote := &models.SymbolQuote{ChangePct: 6.0}
	triggered, err := evaluatePriceChangePct(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triggered {
		t.Error("expected no trigger for positive change with direction=down")
	}
}

func TestPriceChangePct_Either_TriggeredPositive(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.PriceChangeCondition{PercentChange: 5.0, Direction: "either"}),
	}
	quote := &models.SymbolQuote{ChangePct: 7.0}
	triggered, err := evaluatePriceChangePct(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !triggered {
		t.Error("expected trigger for either direction with positive move >= threshold")
	}
}

func TestPriceChangePct_Either_TriggeredNegative(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.PriceChangeCondition{PercentChange: 5.0, Direction: "either"}),
	}
	quote := &models.SymbolQuote{ChangePct: -7.0}
	triggered, err := evaluatePriceChangePct(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !triggered {
		t.Error("expected trigger for either direction with negative move >= threshold")
	}
}

func TestPriceChangePct_Either_NotTriggered(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.PriceChangeCondition{PercentChange: 5.0, Direction: "either"}),
	}
	quote := &models.SymbolQuote{ChangePct: 2.0}
	triggered, err := evaluatePriceChangePct(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triggered {
		t.Error("expected no trigger when abs(change) < threshold")
	}
}

func TestPriceChangePct_EmptyDirection_DefaultsToEither(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.PriceChangeCondition{PercentChange: 5.0, Direction: ""}),
	}
	quote := &models.SymbolQuote{ChangePct: -6.0}
	triggered, err := evaluatePriceChangePct(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !triggered {
		t.Error("expected empty direction to default to 'either' behavior")
	}
}

func TestPriceChangePct_InvalidJSON(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: json.RawMessage(`{invalid`),
	}
	quote := &models.SymbolQuote{ChangePct: 10.0}
	_, err := evaluatePriceChangePct(alert, quote)
	if err == nil {
		t.Error("expected error for invalid JSON conditions")
	}
}

func TestPriceChangePct_ZeroPercentChange(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.PriceChangeCondition{PercentChange: 0, Direction: "up"}),
	}
	quote := &models.SymbolQuote{ChangePct: 1.0}
	_, err := evaluatePriceChangePct(alert, quote)
	if err == nil {
		t.Error("expected error for zero percent_change threshold")
	}
}

func TestPriceChangePct_AtExactThreshold(t *testing.T) {
	alert := &models.AlertRule{
		Conditions: mustJSON(models.PriceChangeCondition{PercentChange: 5.0, Direction: "up"}),
	}
	quote := &models.SymbolQuote{ChangePct: 5.0}
	triggered, err := evaluatePriceChangePct(alert, quote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !triggered {
		t.Error("expected trigger when change exactly equals threshold")
	}
}
