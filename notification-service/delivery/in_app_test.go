package delivery

import (
	"testing"
)

// ---------------------------------------------------------------------------
// alertTypeLabel (lives in email.go, tested here for coverage)
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
// formatVolume (lives in email.go, tested here for coverage)
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
