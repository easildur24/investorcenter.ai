package services

import (
	"encoding/json"
	"investorcenter-api/models"
	"testing"
)

// TestEvaluatePriceAbove tests price above threshold evaluation
func TestEvaluatePriceAbove(t *testing.T) {
	ap := &AlertProcessor{}

	tests := []struct {
		name      string
		threshold float64
		price     float64
		expected  bool
	}{
		{"Price above threshold", 100.00, 105.50, true},
		{"Price exactly at threshold", 100.00, 100.00, true},
		{"Price below threshold", 100.00, 95.50, false},
		{"Price significantly above", 50.00, 150.00, true},
		{"Price significantly below", 200.00, 50.00, false},
		{"Zero threshold with positive price", 0.00, 10.00, true},
		{"Negative price above negative threshold", -10.00, -5.00, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create alert with condition
			condition := models.PriceAboveCondition{
				Threshold:  tt.threshold,
				Comparison: "above",
			}
			conditionJSON, _ := json.Marshal(condition)

			alert := &models.AlertRule{
				AlertType:  "price_above",
				Conditions: conditionJSON,
			}

			quote := &Quote{
				Symbol: "TEST",
				Price:  tt.price,
			}

			result, err := ap.evaluatePriceAbove(alert, quote)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %v, got %v (price: %.2f, threshold: %.2f)",
					tt.expected, result, tt.price, tt.threshold)
			}
		})
	}
}

// TestEvaluatePriceBelow tests price below threshold evaluation
func TestEvaluatePriceBelow(t *testing.T) {
	ap := &AlertProcessor{}

	tests := []struct {
		name      string
		threshold float64
		price     float64
		expected  bool
	}{
		{"Price below threshold", 100.00, 95.50, true},
		{"Price exactly at threshold", 100.00, 100.00, true},
		{"Price above threshold", 100.00, 105.50, false},
		{"Price significantly below", 200.00, 50.00, true},
		{"Price significantly above", 50.00, 150.00, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition := models.PriceAboveCondition{
				Threshold:  tt.threshold,
				Comparison: "below",
			}
			conditionJSON, _ := json.Marshal(condition)

			alert := &models.AlertRule{
				AlertType:  "price_below",
				Conditions: conditionJSON,
			}

			quote := &Quote{
				Symbol: "TEST",
				Price:  tt.price,
			}

			result, err := ap.evaluatePriceBelow(alert, quote)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %v, got %v (price: %.2f, threshold: %.2f)",
					tt.expected, result, tt.price, tt.threshold)
			}
		})
	}
}

// TestEvaluateVolumeAbove tests volume above threshold evaluation
func TestEvaluateVolumeAbove(t *testing.T) {
	ap := &AlertProcessor{}

	tests := []struct {
		name      string
		threshold float64
		volume    int64
		expected  bool
	}{
		{"Volume above threshold", 1000000, 1500000, true},
		{"Volume exactly at threshold", 1000000, 1000000, true},
		{"Volume below threshold", 1000000, 500000, false},
		{"Very high volume", 100000, 50000000, true},
		{"Zero volume", 1000, 0, false},
		{"Zero threshold", 0, 1000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition := models.VolumeCondition{
				Threshold:  tt.threshold,
				Comparison: "above",
			}
			conditionJSON, _ := json.Marshal(condition)

			alert := &models.AlertRule{
				AlertType:  "volume_above",
				Conditions: conditionJSON,
			}

			quote := &Quote{
				Symbol: "TEST",
				Volume: tt.volume,
			}

			result, err := ap.evaluateVolumeAbove(alert, quote)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %v, got %v (volume: %d, threshold: %.0f)",
					tt.expected, result, tt.volume, tt.threshold)
			}
		})
	}
}

// TestEvaluateVolumeBelow tests volume below threshold evaluation
func TestEvaluateVolumeBelow(t *testing.T) {
	ap := &AlertProcessor{}

	tests := []struct {
		name      string
		threshold float64
		volume    int64
		expected  bool
	}{
		{"Volume below threshold", 1000000, 500000, true},
		{"Volume exactly at threshold", 1000000, 1000000, true},
		{"Volume above threshold", 1000000, 1500000, false},
		{"Very low volume", 1000000, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition := models.VolumeCondition{
				Threshold:  tt.threshold,
				Comparison: "below",
			}
			conditionJSON, _ := json.Marshal(condition)

			alert := &models.AlertRule{
				AlertType:  "volume_below",
				Conditions: conditionJSON,
			}

			quote := &Quote{
				Symbol: "TEST",
				Volume: tt.volume,
			}

			result, err := ap.evaluateVolumeBelow(alert, quote)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %v, got %v (volume: %d, threshold: %.0f)",
					tt.expected, result, tt.volume, tt.threshold)
			}
		})
	}
}

// TestEvaluateAlert_InvalidJSON tests handling of invalid JSON conditions
func TestEvaluateAlert_InvalidJSON(t *testing.T) {
	ap := &AlertProcessor{}

	alert := &models.AlertRule{
		AlertType:  "price_above",
		Conditions: []byte(`{invalid json}`),
	}

	quote := &Quote{
		Symbol: "TEST",
		Price:  100.00,
	}

	_, err := ap.evaluatePriceAbove(alert, quote)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

// TestEvaluateAlert_UnsupportedType tests handling of unsupported alert types
func TestEvaluateAlert_UnsupportedType(t *testing.T) {
	ap := &AlertProcessor{}

	alert := &models.AlertRule{
		AlertType:  "unsupported_type",
		Conditions: []byte(`{}`),
	}

	quote := &Quote{
		Symbol: "TEST",
		Price:  100.00,
	}

	_, err := ap.evaluateAlert(alert, quote)
	if err == nil {
		t.Error("Expected error for unsupported alert type, got nil")
	}
}

// TestQuoteStruct tests the Quote struct
func TestQuoteStruct(t *testing.T) {
	quote := &Quote{
		Symbol:    "AAPL",
		Price:     150.25,
		Volume:    50000000,
		Timestamp: 1699564800,
		Updated:   1699564800,
	}

	if quote.Symbol != "AAPL" {
		t.Errorf("Expected symbol 'AAPL', got '%s'", quote.Symbol)
	}

	if quote.Price != 150.25 {
		t.Errorf("Expected price 150.25, got %.2f", quote.Price)
	}

	if quote.Volume != 50000000 {
		t.Errorf("Expected volume 50000000, got %d", quote.Volume)
	}
}

// Benchmark tests
func BenchmarkEvaluatePriceAbove(b *testing.B) {
	ap := &AlertProcessor{}

	condition := models.PriceAboveCondition{
		Threshold:  100.00,
		Comparison: "above",
	}
	conditionJSON, _ := json.Marshal(condition)

	alert := &models.AlertRule{
		AlertType:  "price_above",
		Conditions: conditionJSON,
	}

	quote := &Quote{
		Symbol: "TEST",
		Price:  105.50,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ap.evaluatePriceAbove(alert, quote)
	}
}

func BenchmarkEvaluateVolumeAbove(b *testing.B) {
	ap := &AlertProcessor{}

	condition := models.VolumeCondition{
		Threshold:  1000000,
		Comparison: "above",
	}
	conditionJSON, _ := json.Marshal(condition)

	alert := &models.AlertRule{
		AlertType:  "volume_above",
		Conditions: conditionJSON,
	}

	quote := &Quote{
		Symbol: "TEST",
		Volume: 1500000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ap.evaluateVolumeAbove(alert, quote)
	}
}

// TestEvaluatePriceChangePct tests the not-yet-implemented price change pct
func TestEvaluatePriceChangePct(t *testing.T) {
	ap := &AlertProcessor{}

	condition := models.PriceChangeCondition{
		PercentChange: 5.0,
		Period:        "1d",
		Direction:     "up",
	}
	conditionJSON, _ := json.Marshal(condition)

	alert := &models.AlertRule{
		AlertType:  "price_change_pct",
		Symbol:     "AAPL",
		Conditions: conditionJSON,
	}

	quote := &Quote{
		Symbol: "AAPL",
		Price:  150.00,
	}

	// This is explicitly not implemented yet, should return error
	result, err := ap.evaluatePriceChangePct(alert, quote)
	if err == nil {
		t.Error("Expected error for unimplemented price_change_pct, got nil")
	}
	if result {
		t.Error("Expected false result for unimplemented alert type")
	}
}

func TestEvaluatePriceChangePct_InvalidJSON(t *testing.T) {
	ap := &AlertProcessor{}

	alert := &models.AlertRule{
		AlertType:  "price_change_pct",
		Conditions: []byte(`{invalid}`),
	}

	quote := &Quote{Symbol: "TEST", Price: 100.0}

	_, err := ap.evaluatePriceChangePct(alert, quote)
	if err == nil {
		t.Error("Expected error for invalid JSON conditions")
	}
}

// TestEvaluateVolumeSpike tests the not-yet-implemented volume spike
func TestEvaluateVolumeSpike(t *testing.T) {
	ap := &AlertProcessor{}

	condition := models.VolumeSpikeCondition{
		VolumeMultiplier: 2.0,
		Baseline:         "avg_30d",
	}
	conditionJSON, _ := json.Marshal(condition)

	alert := &models.AlertRule{
		AlertType:  "volume_spike",
		Symbol:     "AAPL",
		Conditions: conditionJSON,
	}

	quote := &Quote{
		Symbol: "AAPL",
		Volume: 50000000,
	}

	// This is explicitly not implemented yet, should return error
	result, err := ap.evaluateVolumeSpike(alert, quote)
	if err == nil {
		t.Error("Expected error for unimplemented volume_spike, got nil")
	}
	if result {
		t.Error("Expected false result for unimplemented alert type")
	}
}

func TestEvaluateVolumeSpike_InvalidJSON(t *testing.T) {
	ap := &AlertProcessor{}

	alert := &models.AlertRule{
		AlertType:  "volume_spike",
		Conditions: []byte(`{invalid}`),
	}

	quote := &Quote{Symbol: "TEST", Volume: 1000000}

	_, err := ap.evaluateVolumeSpike(alert, quote)
	if err == nil {
		t.Error("Expected error for invalid JSON conditions")
	}
}

// TestEvaluateAlert_ViaSwitch tests evaluateAlert dispatches correctly
func TestEvaluateAlert_PriceChangePctViaSwitch(t *testing.T) {
	ap := &AlertProcessor{}

	condition := models.PriceChangeCondition{PercentChange: 5.0}
	conditionJSON, _ := json.Marshal(condition)

	alert := &models.AlertRule{
		AlertType:  "price_change_pct",
		Conditions: conditionJSON,
	}

	quote := &Quote{Symbol: "TEST", Price: 100.0}

	_, err := ap.evaluateAlert(alert, quote)
	if err == nil {
		t.Error("Expected error from unimplemented price_change_pct via evaluateAlert")
	}
}

func TestEvaluateAlert_VolumeSpikeViaSwitch(t *testing.T) {
	ap := &AlertProcessor{}

	condition := models.VolumeSpikeCondition{VolumeMultiplier: 2.0}
	conditionJSON, _ := json.Marshal(condition)

	alert := &models.AlertRule{
		AlertType:  "volume_spike",
		Conditions: conditionJSON,
	}

	quote := &Quote{Symbol: "TEST", Volume: 5000000}

	_, err := ap.evaluateAlert(alert, quote)
	if err == nil {
		t.Error("Expected error from unimplemented volume_spike via evaluateAlert")
	}
}

// TestNewAlertProcessor tests the constructor
func TestNewAlertProcessor(t *testing.T) {
	ap := NewAlertProcessor(nil, nil, nil)
	if ap == nil {
		t.Error("Expected non-nil AlertProcessor")
	}
}

func TestNewAlertProcessor_WithServices(t *testing.T) {
	alertSvc := NewAlertService()
	ap := NewAlertProcessor(alertSvc, nil, nil)
	if ap == nil {
		t.Error("Expected non-nil AlertProcessor")
	}
	if ap.alertService == nil {
		t.Error("Expected non-nil alertService")
	}
}

// TestEvaluatePriceAbove_InvalidConditions tests error path for invalid JSON
func TestEvaluatePriceAbove_InvalidConditions(t *testing.T) {
	ap := &AlertProcessor{}

	alert := &models.AlertRule{
		AlertType:  "price_above",
		Conditions: []byte(`not json at all`),
	}

	quote := &Quote{Symbol: "TEST", Price: 100.0}

	_, err := ap.evaluatePriceAbove(alert, quote)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// TestEvaluatePriceBelow_InvalidConditions tests error path for invalid JSON
func TestEvaluatePriceBelow_InvalidConditions(t *testing.T) {
	ap := &AlertProcessor{}

	alert := &models.AlertRule{
		AlertType:  "price_below",
		Conditions: []byte(`not json at all`),
	}

	quote := &Quote{Symbol: "TEST", Price: 100.0}

	_, err := ap.evaluatePriceBelow(alert, quote)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// TestEvaluateVolumeAbove_InvalidConditions tests error path for invalid JSON
func TestEvaluateVolumeAbove_InvalidConditions(t *testing.T) {
	ap := &AlertProcessor{}

	alert := &models.AlertRule{
		AlertType:  "volume_above",
		Conditions: []byte(`not json at all`),
	}

	quote := &Quote{Symbol: "TEST", Volume: 1000000}

	_, err := ap.evaluateVolumeAbove(alert, quote)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// TestEvaluateVolumeBelow_InvalidConditions tests error path for invalid JSON
func TestEvaluateVolumeBelow_InvalidConditions(t *testing.T) {
	ap := &AlertProcessor{}

	alert := &models.AlertRule{
		AlertType:  "volume_below",
		Conditions: []byte(`not json at all`),
	}

	quote := &Quote{Symbol: "TEST", Volume: 1000000}

	_, err := ap.evaluateVolumeBelow(alert, quote)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// Table-driven test for multiple alert types
func TestEvaluateAlert_MultipleTypes(t *testing.T) {
	ap := &AlertProcessor{}

	tests := []struct {
		name      string
		alertType string
		condition interface{}
		quote     *Quote
		expected  bool
		wantError bool
	}{
		{
			name:      "Price above - triggers",
			alertType: "price_above",
			condition: models.PriceAboveCondition{Threshold: 100.00},
			quote:     &Quote{Price: 105.00},
			expected:  true,
			wantError: false,
		},
		{
			name:      "Price above - does not trigger",
			alertType: "price_above",
			condition: models.PriceAboveCondition{Threshold: 100.00},
			quote:     &Quote{Price: 95.00},
			expected:  false,
			wantError: false,
		},
		{
			name:      "Volume above - triggers",
			alertType: "volume_above",
			condition: models.VolumeCondition{Threshold: 1000000},
			quote:     &Quote{Volume: 2000000},
			expected:  true,
			wantError: false,
		},
		{
			name:      "Volume below - triggers",
			alertType: "volume_below",
			condition: models.VolumeCondition{Threshold: 1000000},
			quote:     &Quote{Volume: 500000},
			expected:  true,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conditionJSON, _ := json.Marshal(tt.condition)
			alert := &models.AlertRule{
				AlertType:  tt.alertType,
				Conditions: conditionJSON,
			}

			result, err := ap.evaluateAlert(alert, tt.quote)

			if (err != nil) != tt.wantError {
				t.Errorf("wantError = %v, got error = %v", tt.wantError, err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
