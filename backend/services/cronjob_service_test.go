package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"investorcenter-api/models"
)

// ---------------------------------------------------------------------------
// NewCronjobService — constructor
// ---------------------------------------------------------------------------

func TestNewCronjobService(t *testing.T) {
	svc := NewCronjobService()
	require.NotNil(t, svc)
}

// ---------------------------------------------------------------------------
// calculateHealthStatus — pure function
// ---------------------------------------------------------------------------

func TestCalculateHealthStatus_Healthy(t *testing.T) {
	svc := NewCronjobService()
	job := &models.CronjobStatusWithInfo{
		ConsecutiveFailures: 0,
		LastRun: &models.LastRunInfo{
			Status:    "success",
			StartedAt: time.Now().Add(-1 * time.Hour),
		},
	}

	status := svc.calculateHealthStatus(job)
	assert.Equal(t, "healthy", status)
}

func TestCalculateHealthStatus_CriticalLastFailed(t *testing.T) {
	svc := NewCronjobService()
	job := &models.CronjobStatusWithInfo{
		ConsecutiveFailures: 0,
		LastRun: &models.LastRunInfo{
			Status:    "failed",
			StartedAt: time.Now().Add(-30 * time.Minute),
		},
	}

	status := svc.calculateHealthStatus(job)
	assert.Equal(t, "critical", status)
}

func TestCalculateHealthStatus_CriticalTimeout(t *testing.T) {
	svc := NewCronjobService()
	job := &models.CronjobStatusWithInfo{
		ConsecutiveFailures: 0,
		LastRun: &models.LastRunInfo{
			Status:    "timeout",
			StartedAt: time.Now().Add(-30 * time.Minute),
		},
	}

	status := svc.calculateHealthStatus(job)
	assert.Equal(t, "critical", status)
}

func TestCalculateHealthStatus_CriticalConsecutiveFailures(t *testing.T) {
	svc := NewCronjobService()
	job := &models.CronjobStatusWithInfo{
		ConsecutiveFailures: 3,
		LastRun: &models.LastRunInfo{
			Status:    "success", // Even success, but 3 failures is critical
			StartedAt: time.Now().Add(-1 * time.Hour),
		},
	}

	status := svc.calculateHealthStatus(job)
	assert.Equal(t, "critical", status)
}

func TestCalculateHealthStatus_CriticalHighConsecutiveFailures(t *testing.T) {
	svc := NewCronjobService()
	job := &models.CronjobStatusWithInfo{
		ConsecutiveFailures: 10,
	}

	status := svc.calculateHealthStatus(job)
	assert.Equal(t, "critical", status)
}

func TestCalculateHealthStatus_Warning1Failure(t *testing.T) {
	svc := NewCronjobService()
	job := &models.CronjobStatusWithInfo{
		ConsecutiveFailures: 1,
		LastRun: &models.LastRunInfo{
			Status:    "success",
			StartedAt: time.Now().Add(-1 * time.Hour),
		},
	}

	status := svc.calculateHealthStatus(job)
	assert.Equal(t, "warning", status)
}

func TestCalculateHealthStatus_Warning2Failures(t *testing.T) {
	svc := NewCronjobService()
	job := &models.CronjobStatusWithInfo{
		ConsecutiveFailures: 2,
		LastRun: &models.LastRunInfo{
			Status:    "success",
			StartedAt: time.Now().Add(-1 * time.Hour),
		},
	}

	status := svc.calculateHealthStatus(job)
	assert.Equal(t, "warning", status)
}

func TestCalculateHealthStatus_WarningRunning(t *testing.T) {
	svc := NewCronjobService()
	job := &models.CronjobStatusWithInfo{
		ConsecutiveFailures: 0,
		LastRun: &models.LastRunInfo{
			Status:    "running",
			StartedAt: time.Now().Add(-5 * time.Minute),
		},
	}

	status := svc.calculateHealthStatus(job)
	assert.Equal(t, "warning", status)
}

func TestCalculateHealthStatus_UnknownNoExecution(t *testing.T) {
	svc := NewCronjobService()
	job := &models.CronjobStatusWithInfo{
		ConsecutiveFailures: 0,
		LastRun:             nil,
	}

	status := svc.calculateHealthStatus(job)
	assert.Equal(t, "unknown", status)
}

// ---------------------------------------------------------------------------
// jsonbToString — pure function
// ---------------------------------------------------------------------------

func TestJsonbToString_Map(t *testing.T) {
	data := map[string]interface{}{
		"key": "value",
	}
	result := jsonbToString(data)
	assert.Contains(t, result, "key")
	assert.Contains(t, result, "value")
}

func TestJsonbToString_Slice(t *testing.T) {
	data := []string{"a", "b", "c"}
	result := jsonbToString(data)
	assert.Contains(t, result, "a")
	assert.Contains(t, result, "b")
	assert.Contains(t, result, "c")
}

func TestJsonbToString_Nil(t *testing.T) {
	result := jsonbToString(nil)
	assert.Equal(t, "null", result)
}

func TestJsonbToString_EmptySlice(t *testing.T) {
	result := jsonbToString([]string{})
	assert.Equal(t, "[]", result)
}

func TestJsonbToString_Number(t *testing.T) {
	result := jsonbToString(42)
	assert.Equal(t, "42", result)
}

func TestJsonbToString_String(t *testing.T) {
	result := jsonbToString("hello")
	assert.Equal(t, `"hello"`, result)
}
