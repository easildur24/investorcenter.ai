package database

import (
	"testing"
	"time"
)

func TestTodayStart(t *testing.T) {
	now := time.Now().UTC()
	got := todayStart()

	if got.Year() != now.Year() {
		t.Errorf("Year = %d, want %d", got.Year(), now.Year())
	}
	if got.Month() != now.Month() {
		t.Errorf("Month = %s, want %s", got.Month(), now.Month())
	}
	if got.Day() != now.Day() {
		t.Errorf("Day = %d, want %d", got.Day(), now.Day())
	}
	if got.Hour() != 0 {
		t.Errorf("Hour = %d, want 0", got.Hour())
	}
	if got.Minute() != 0 {
		t.Errorf("Minute = %d, want 0", got.Minute())
	}
	if got.Second() != 0 {
		t.Errorf("Second = %d, want 0", got.Second())
	}
	if got.Location() != time.UTC {
		t.Errorf("Location = %v, want UTC", got.Location())
	}
}
