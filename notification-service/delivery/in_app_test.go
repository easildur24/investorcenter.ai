package delivery

import (
	"encoding/json"
	"testing"

	"notification-service/models"
)

func mustJSON(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// ---------------------------------------------------------------------------
// alertTypeLabel
// ---------------------------------------------------------------------------

func TestAlertTypeLabel_KnownTypes(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"price_above", "Price Above"},
		{"price_below", "Price Below"},
		{"price_change_pct", "Price Change %"},
		{"volume_above", "Volume Above"},
		{"volume_below", "Volume Below"},
		{"volume_spike", "Volume Spike"},
		{"news", "News Alert"},
		{"earnings", "Earnings Report"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := alertTypeLabel(tc.input)
			if got != tc.expected {
				t.Errorf("alertTypeLabel(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestAlertTypeLabel_UnknownType(t *testing.T) {
	got := alertTypeLabel("my_custom_alert")
	if got != "my custom alert" {
		t.Errorf("expected underscores replaced with spaces, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// formatVolume
// ---------------------------------------------------------------------------

func TestFormatVolume(t *testing.T) {
	cases := []struct {
		input    float64
		expected string
	}{
		{500, "500"},
		{1_000, "1.0K"},
		{1_500, "1.5K"},
		{10_000, "10.0K"},
		{1_000_000, "1.0M"},
		{1_500_000, "1.5M"},
		{45_000_000, "45.0M"},
		{1_000_000_000, "1.0B"},
		{2_500_000_000, "2.5B"},
		{0, "0"},
		{999, "999"},
	}
	for _, tc := range cases {
		t.Run(tc.expected, func(t *testing.T) {
			got := formatVolume(tc.input)
			if got != tc.expected {
				t.Errorf("formatVolume(%f) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// buildTitle
// ---------------------------------------------------------------------------

func TestBuildTitle(t *testing.T) {
	alert := &models.AlertRule{Symbol: "AAPL", AlertType: "price_above"}
	quote := &models.SymbolQuote{Price: 150.0}
	title := buildTitle(alert, quote)
	expected := "AAPL Price Above"
	if title != expected {
		t.Errorf("buildTitle = %q, want %q", title, expected)
	}
}

func TestBuildTitle_VolumeBelow(t *testing.T) {
	alert := &models.AlertRule{Symbol: "TSLA", AlertType: "volume_below"}
	quote := &models.SymbolQuote{Volume: 100000}
	title := buildTitle(alert, quote)
	expected := "TSLA Volume Below"
	if title != expected {
		t.Errorf("buildTitle = %q, want %q", title, expected)
	}
}

// ---------------------------------------------------------------------------
// buildMessage
// ---------------------------------------------------------------------------

func TestBuildMessage_PriceAbove(t *testing.T) {
	alert := &models.AlertRule{
		Symbol:     "AAPL",
		AlertType:  "price_above",
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 150.0}),
	}
	quote := &models.SymbolQuote{Price: 152.30}
	msg := buildMessage(alert, quote)
	expected := "AAPL crossed above $150.00 (current: $152.30)"
	if msg != expected {
		t.Errorf("buildMessage = %q, want %q", msg, expected)
	}
}

func TestBuildMessage_PriceBelow(t *testing.T) {
	alert := &models.AlertRule{
		Symbol:     "TSLA",
		AlertType:  "price_below",
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 200.0}),
	}
	quote := &models.SymbolQuote{Price: 195.50}
	msg := buildMessage(alert, quote)
	expected := "TSLA dropped below $200.00 (current: $195.50)"
	if msg != expected {
		t.Errorf("buildMessage = %q, want %q", msg, expected)
	}
}

func TestBuildMessage_VolumeAbove(t *testing.T) {
	alert := &models.AlertRule{
		Symbol:     "AAPL",
		AlertType:  "volume_above",
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 50000000}),
	}
	quote := &models.SymbolQuote{Volume: 55000000}
	msg := buildMessage(alert, quote)
	expected := "AAPL volume exceeded 50.0M (current: 55.0M)"
	if msg != expected {
		t.Errorf("buildMessage = %q, want %q", msg, expected)
	}
}

func TestBuildMessage_VolumeBelow(t *testing.T) {
	alert := &models.AlertRule{
		Symbol:     "AAPL",
		AlertType:  "volume_below",
		Conditions: mustJSON(models.ThresholdCondition{Threshold: 1000000}),
	}
	quote := &models.SymbolQuote{Volume: 500000}
	msg := buildMessage(alert, quote)
	expected := "AAPL volume dropped below 1.0M (current: 500.0K)"
	if msg != expected {
		t.Errorf("buildMessage = %q, want %q", msg, expected)
	}
}

func TestBuildMessage_PriceChangePct(t *testing.T) {
	alert := &models.AlertRule{
		Symbol:    "AAPL",
		AlertType: "price_change_pct",
	}
	quote := &models.SymbolQuote{ChangePct: 3.45}
	msg := buildMessage(alert, quote)
	expected := "AAPL moved 3.45% today"
	if msg != expected {
		t.Errorf("buildMessage = %q, want %q", msg, expected)
	}
}

func TestBuildMessage_UnknownType(t *testing.T) {
	alert := &models.AlertRule{
		Symbol:    "AAPL",
		AlertType: "custom_type",
	}
	quote := &models.SymbolQuote{Price: 150.0}
	msg := buildMessage(alert, quote)
	expected := "Alert triggered for AAPL"
	if msg != expected {
		t.Errorf("buildMessage = %q, want %q", msg, expected)
	}
}

// ---------------------------------------------------------------------------
// buildData
// ---------------------------------------------------------------------------

func TestBuildData(t *testing.T) {
	alert := &models.AlertRule{
		WatchListID: "wl-123",
		Symbol:      "AAPL",
		AlertType:   "price_above",
	}
	quote := &models.SymbolQuote{Price: 152.30, Volume: 45000000}
	data := buildData(alert, quote)

	if data["watch_list_id"] != "wl-123" {
		t.Errorf("watch_list_id = %v, want wl-123", data["watch_list_id"])
	}
	if data["symbol"] != "AAPL" {
		t.Errorf("symbol = %v, want AAPL", data["symbol"])
	}
	if data["price"] != 152.30 {
		t.Errorf("price = %v, want 152.30", data["price"])
	}
	if data["volume"] != int64(45000000) {
		t.Errorf("volume = %v, want 45000000", data["volume"])
	}
	if data["alert_type"] != "price_above" {
		t.Errorf("alert_type = %v, want price_above", data["alert_type"])
	}
}
