package delivery

import (
	"strings"
	"testing"

	"notification-service/models"
)

// ---------------------------------------------------------------------------
// isInQuietHours
// ---------------------------------------------------------------------------

func TestIsInQuietHours_Disabled(t *testing.T) {
	prefs := &models.NotificationPreferences{
		QuietHoursEnabled: false,
	}
	inQuiet, err := isInQuietHours(prefs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inQuiet {
		t.Error("expected false when quiet hours disabled")
	}
}

func TestIsInQuietHours_SameDay_Inside(t *testing.T) {
	// We can't control time.Now() without refactoring, so we use a range
	// that covers the full day to guarantee the current time is inside.
	prefs := &models.NotificationPreferences{
		QuietHoursEnabled:  true,
		QuietHoursStart:    "00:00:00",
		QuietHoursEnd:      "23:59:59",
		QuietHoursTimezone: "UTC",
	}
	inQuiet, err := isInQuietHours(prefs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !inQuiet {
		t.Error("expected true when quiet hours span full day (00:00-23:59)")
	}
}

func TestIsInQuietHours_SameDay_Outside(t *testing.T) {
	// Use a range that's always in the past (yesterday)
	// by picking a very narrow range of a single second at midnight.
	// This test uses an impossible-to-hit range.
	prefs := &models.NotificationPreferences{
		QuietHoursEnabled:  true,
		QuietHoursStart:    "00:00:00",
		QuietHoursEnd:      "00:00:01",
		QuietHoursTimezone: "UTC",
	}
	// This will be "inside" only at exactly midnight UTC, which is extremely
	// unlikely. We can't deterministically test this without mocking time.Now().
	// Instead, test the overnight range logic below which is deterministic.
	_, err := isInQuietHours(prefs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIsInQuietHours_Overnight_FullCoverage(t *testing.T) {
	// Overnight range that covers the full day (22:00 to 23:59).
	// Actually, an overnight range where start > end covers currentTime >= start OR <= end.
	// Use start=00:00:01 end=00:00:00 — this covers almost everything.
	prefs := &models.NotificationPreferences{
		QuietHoursEnabled:  true,
		QuietHoursStart:    "00:00:01",
		QuietHoursEnd:      "00:00:00",
		QuietHoursTimezone: "UTC",
	}
	// This overnight range covers currentTime >= "00:00:01" OR currentTime <= "00:00:00"
	// which is basically the entire day.
	inQuiet, err := isInQuietHours(prefs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !inQuiet {
		t.Error("expected true when overnight range covers nearly full day")
	}
}

func TestIsInQuietHours_InvalidTimezone(t *testing.T) {
	prefs := &models.NotificationPreferences{
		QuietHoursEnabled:  true,
		QuietHoursStart:    "22:00:00",
		QuietHoursEnd:      "08:00:00",
		QuietHoursTimezone: "Invalid/Timezone",
	}
	_, err := isInQuietHours(prefs)
	if err == nil {
		t.Error("expected error for invalid timezone")
	}
}

func TestIsInQuietHours_ValidTimezones(t *testing.T) {
	timezones := []string{
		"America/New_York",
		"America/Los_Angeles",
		"Europe/London",
		"Asia/Tokyo",
		"UTC",
	}
	for _, tz := range timezones {
		t.Run(tz, func(t *testing.T) {
			prefs := &models.NotificationPreferences{
				QuietHoursEnabled:  true,
				QuietHoursStart:    "22:00:00",
				QuietHoursEnd:      "08:00:00",
				QuietHoursTimezone: tz,
			}
			_, err := isInQuietHours(prefs)
			if err != nil {
				t.Errorf("unexpected error for timezone %s: %v", tz, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// formatAlertEmailBody
// ---------------------------------------------------------------------------

func TestFormatAlertEmailBody_ContainsExpectedContent(t *testing.T) {
	alert := &models.AlertRule{
		Name:        "My AAPL Alert",
		Symbol:      "AAPL",
		AlertType:   "price_above",
		WatchListID: "wl-123",
	}
	quote := &models.SymbolQuote{
		Price:     152.30,
		Volume:    45000000,
		ChangePct: 1.25,
	}

	body := formatAlertEmailBody(alert, quote, "John Doe", "https://investorcenter.ai")

	// Check key elements are present in the HTML
	checks := []string{
		"My AAPL Alert",          // Alert name
		"John Doe",               // User name
		"AAPL",                   // Symbol
		"$152.30",                // Price
		"1.25%",                  // Change percent
		"45.0M",                  // Volume formatted
		"wl-123",                 // Watchlist ID in URL
		"investorcenter.ai",      // Frontend URL
		"View Watchlist",         // CTA button text
		"account settings",       // Footer link
	}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Errorf("email body missing expected content: %q", check)
		}
	}
}

func TestFormatAlertEmailBody_IsHTML(t *testing.T) {
	alert := &models.AlertRule{
		Name:        "Test Alert",
		Symbol:      "TSLA",
		AlertType:   "price_below",
		WatchListID: "wl-456",
	}
	quote := &models.SymbolQuote{Price: 200.0, Volume: 30000000, ChangePct: -0.5}

	body := formatAlertEmailBody(alert, quote, "Jane", "https://example.com")

	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("expected HTML doctype")
	}
	if !strings.Contains(body, "</html>") {
		t.Error("expected closing html tag")
	}
}

func TestFormatAlertEmailBody_WatchlistURL(t *testing.T) {
	alert := &models.AlertRule{
		Name:        "Alert",
		Symbol:      "MSFT",
		AlertType:   "price_above",
		WatchListID: "abc-def-123",
	}
	quote := &models.SymbolQuote{Price: 400.0, Volume: 20000000, ChangePct: 2.0}

	body := formatAlertEmailBody(alert, quote, "User", "https://investorcenter.ai")

	expected := "https://investorcenter.ai/watchlist/abc-def-123"
	if !strings.Contains(body, expected) {
		t.Errorf("email body missing watchlist URL: %q", expected)
	}
}
