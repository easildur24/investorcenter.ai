package services

import (
	"investorcenter-api/models"
	"testing"
	"time"
)

// TestShouldTriggerBasedOnFrequency_Once tests that "once" frequency never re-triggers
func TestShouldTriggerBasedOnFrequency_Once(t *testing.T) {
	service := NewAlertService()

	// Test 1: No last trigger - should trigger
	alert := &models.AlertRule{
		Frequency:       "once",
		LastTriggeredAt: nil,
	}
	if !service.ShouldTriggerBasedOnFrequency(alert) {
		t.Error("Expected alert to trigger when LastTriggeredAt is nil")
	}

	// Test 2: Has last trigger - should NOT trigger
	now := time.Now()
	alert.LastTriggeredAt = &now
	if service.ShouldTriggerBasedOnFrequency(alert) {
		t.Error("Expected 'once' alert to NOT trigger after being triggered once")
	}

	// Test 3: Last triggered 30 days ago - still should NOT trigger
	oldTime := now.AddDate(0, 0, -30)
	alert.LastTriggeredAt = &oldTime
	if service.ShouldTriggerBasedOnFrequency(alert) {
		t.Error("Expected 'once' alert to never re-trigger, even after 30 days")
	}
}

// TestShouldTriggerBasedOnFrequency_Daily tests daily frequency enforcement
func TestShouldTriggerBasedOnFrequency_Daily(t *testing.T) {
	service := NewAlertService()

	// Test 1: No last trigger - should trigger
	alert := &models.AlertRule{
		Frequency:       "daily",
		LastTriggeredAt: nil,
	}
	if !service.ShouldTriggerBasedOnFrequency(alert) {
		t.Error("Expected alert to trigger when LastTriggeredAt is nil")
	}

	// Test 2: Triggered 1 hour ago - should NOT trigger
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	alert.LastTriggeredAt = &oneHourAgo
	if service.ShouldTriggerBasedOnFrequency(alert) {
		t.Error("Expected 'daily' alert to NOT trigger within 24 hours")
	}

	// Test 3: Triggered 23 hours ago - should NOT trigger
	almostOneDayAgo := time.Now().Add(-23 * time.Hour)
	alert.LastTriggeredAt = &almostOneDayAgo
	if service.ShouldTriggerBasedOnFrequency(alert) {
		t.Error("Expected 'daily' alert to NOT trigger before 24 hours")
	}

	// Test 4: Triggered exactly 24 hours ago - should trigger
	exactlyOneDayAgo := time.Now().Add(-24 * time.Hour)
	alert.LastTriggeredAt = &exactlyOneDayAgo
	if !service.ShouldTriggerBasedOnFrequency(alert) {
		t.Error("Expected 'daily' alert to trigger after 24 hours")
	}

	// Test 5: Triggered 25 hours ago - should trigger
	moreThanOneDayAgo := time.Now().Add(-25 * time.Hour)
	alert.LastTriggeredAt = &moreThanOneDayAgo
	if !service.ShouldTriggerBasedOnFrequency(alert) {
		t.Error("Expected 'daily' alert to trigger after more than 24 hours")
	}

	// Test 6: Triggered 7 days ago - should trigger
	weekAgo := time.Now().AddDate(0, 0, -7)
	alert.LastTriggeredAt = &weekAgo
	if !service.ShouldTriggerBasedOnFrequency(alert) {
		t.Error("Expected 'daily' alert to trigger after 7 days")
	}
}

// TestShouldTriggerBasedOnFrequency_Always tests that "always" frequency always triggers
func TestShouldTriggerBasedOnFrequency_Always(t *testing.T) {
	service := NewAlertService()

	// Test 1: No last trigger - should trigger
	alert := &models.AlertRule{
		Frequency:       "always",
		LastTriggeredAt: nil,
	}
	if !service.ShouldTriggerBasedOnFrequency(alert) {
		t.Error("Expected alert to trigger when LastTriggeredAt is nil")
	}

	// Test 2: Triggered 1 second ago - should trigger
	oneSecondAgo := time.Now().Add(-1 * time.Second)
	alert.LastTriggeredAt = &oneSecondAgo
	if !service.ShouldTriggerBasedOnFrequency(alert) {
		t.Error("Expected 'always' alert to trigger immediately")
	}

	// Test 3: Triggered just now - should trigger
	now := time.Now()
	alert.LastTriggeredAt = &now
	if !service.ShouldTriggerBasedOnFrequency(alert) {
		t.Error("Expected 'always' alert to always trigger")
	}

	// Test 4: Triggered 1 hour ago - should trigger
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	alert.LastTriggeredAt = &oneHourAgo
	if !service.ShouldTriggerBasedOnFrequency(alert) {
		t.Error("Expected 'always' alert to trigger after 1 hour")
	}
}

// TestShouldTriggerBasedOnFrequency_Invalid tests invalid frequency handling
func TestShouldTriggerBasedOnFrequency_Invalid(t *testing.T) {
	service := NewAlertService()

	// Test 1: First trigger with invalid frequency - should allow (nil LastTriggeredAt takes precedence)
	alert := &models.AlertRule{
		Frequency:       "invalid_frequency",
		LastTriggeredAt: nil,
	}
	if !service.ShouldTriggerBasedOnFrequency(alert) {
		t.Error("Expected alert to trigger on first time even with invalid frequency")
	}

	// Test 2: After first trigger with invalid frequency - should NOT allow
	now := time.Now()
	alert.LastTriggeredAt = &now
	if service.ShouldTriggerBasedOnFrequency(alert) {
		t.Error("Expected invalid frequency to return false after first trigger")
	}
}

// TestShouldTriggerBasedOnFrequency_EdgeCases tests edge cases
func TestShouldTriggerBasedOnFrequency_EdgeCases(t *testing.T) {
	service := NewAlertService()

	// Test 1: Empty frequency string with nil LastTriggeredAt - should allow first trigger
	alert := &models.AlertRule{
		Frequency:       "",
		LastTriggeredAt: nil,
	}
	if !service.ShouldTriggerBasedOnFrequency(alert) {
		t.Error("Expected alert to trigger on first time even with empty frequency")
	}

	// Test 1b: Empty frequency string after first trigger - should NOT allow
	now := time.Now()
	alert.LastTriggeredAt = &now
	if service.ShouldTriggerBasedOnFrequency(alert) {
		t.Error("Expected empty frequency to return false after first trigger")
	}

	// Test 2: Future timestamp (clock skew)
	future := time.Now().Add(1 * time.Hour)
	alert = &models.AlertRule{
		Frequency:       "daily",
		LastTriggeredAt: &future,
	}
	// Should handle gracefully (likely won't trigger due to negative duration)
	// This is a defensive test for clock skew scenarios
	result := service.ShouldTriggerBasedOnFrequency(alert)
	t.Logf("Future timestamp with 'daily' frequency returned: %v", result)
}

// Benchmark tests
func BenchmarkShouldTriggerBasedOnFrequency_Once(b *testing.B) {
	service := NewAlertService()
	now := time.Now()
	alert := &models.AlertRule{
		Frequency:       "once",
		LastTriggeredAt: &now,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ShouldTriggerBasedOnFrequency(alert)
	}
}

func BenchmarkShouldTriggerBasedOnFrequency_Daily(b *testing.B) {
	service := NewAlertService()
	yesterday := time.Now().AddDate(0, 0, -1)
	alert := &models.AlertRule{
		Frequency:       "daily",
		LastTriggeredAt: &yesterday,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ShouldTriggerBasedOnFrequency(alert)
	}
}

func BenchmarkShouldTriggerBasedOnFrequency_Always(b *testing.B) {
	service := NewAlertService()
	now := time.Now()
	alert := &models.AlertRule{
		Frequency:       "always",
		LastTriggeredAt: &now,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ShouldTriggerBasedOnFrequency(alert)
	}
}
