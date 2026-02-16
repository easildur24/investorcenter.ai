package handlers

import (
	"testing"
)

// TestPtrStr verifies the nil-safe string pointer helper.
func TestPtrStr(t *testing.T) {
	cases := []struct {
		name string
		in   *string
		want string
	}{
		{"nil", nil, ""},
		{"non-nil", strPtr("Technology"), "Technology"},
		{"empty", strPtr(""), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ptrStr(tc.in)
			if got != tc.want {
				t.Errorf("ptrStr(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestFmtFloat verifies the nil-safe float formatting helper.
func TestFmtFloat(t *testing.T) {
	cases := []struct {
		name     string
		in       *float64
		decimals int
		want     string
	}{
		{"nil", nil, 2, ""},
		{"zero", floatPtr(0.0), 2, "0.00"},
		{"positive", floatPtr(123.456), 2, "123.46"},
		{"negative", floatPtr(-5.5), 1, "-5.5"},
		{"large", floatPtr(1e12), 0, "1000000000000"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := fmtFloat(tc.in, tc.decimals)
			if got != tc.want {
				t.Errorf("fmtFloat(%v, %d) = %q, want %q", tc.in, tc.decimals, got, tc.want)
			}
		})
	}
}

// TestFmtInt verifies the nil-safe int formatting helper.
func TestFmtInt(t *testing.T) {
	cases := []struct {
		name string
		in   *int
		want string
	}{
		{"nil", nil, ""},
		{"zero", intPtr(0), "0"},
		{"positive", intPtr(42), "42"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := fmtInt(tc.in)
			if got != tc.want {
				t.Errorf("fmtInt(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func strPtr(s string) *string     { return &s }
func floatPtr(f float64) *float64 { return &f }
func intPtr(i int) *int           { return &i }
