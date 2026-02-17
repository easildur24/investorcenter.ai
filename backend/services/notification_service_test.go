package services

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"investorcenter-api/models"
)

// ---------------------------------------------------------------------------
// NewNotificationService
// ---------------------------------------------------------------------------

func TestNewNotificationService(t *testing.T) {
	es := &EmailService{}
	ns := NewNotificationService(es)
	require.NotNil(t, ns)
	assert.Equal(t, es, ns.emailService)
}

// ---------------------------------------------------------------------------
// formatAlertEmailBody — pure function
// ---------------------------------------------------------------------------

func TestFormatAlertEmailBody(t *testing.T) {
	es := &EmailService{frontendURL: "https://investorcenter.ai"}
	ns := NewNotificationService(es)

	alert := &models.AlertRule{
		Name:       "AAPL Price Above $200",
		Symbol:     "AAPL",
		AlertType:  "price_above",
		Conditions: json.RawMessage(`{}`),
	}

	condition := map[string]interface{}{
		"threshold":  200.0,
		"comparison": "above",
	}

	marketData := map[string]interface{}{
		"price":  205.50,
		"volume": 50000000,
	}

	body := ns.formatAlertEmailBody(alert, condition, marketData)

	assert.Contains(t, body, "AAPL Price Above $200")
	assert.Contains(t, body, "AAPL")
	assert.Contains(t, body, "price_above")
	assert.Contains(t, body, "https://investorcenter.ai/alerts")
	assert.Contains(t, body, "https://investorcenter.ai/settings/notifications")
	assert.Contains(t, body, "200")
	assert.Contains(t, body, "205.5")
}

func TestFormatAlertEmailBody_NilData(t *testing.T) {
	es := &EmailService{frontendURL: "https://app.test"}
	ns := NewNotificationService(es)

	alert := &models.AlertRule{
		Name:       "Test Alert",
		Symbol:     "TSLA",
		AlertType:  "volume_above",
		Conditions: json.RawMessage(`{}`),
	}

	body := ns.formatAlertEmailBody(alert, nil, nil)

	assert.Contains(t, body, "Test Alert")
	assert.Contains(t, body, "TSLA")
	assert.Contains(t, body, "null")
}

func TestFormatAlertEmailBody_ContainsHTMLStructure(t *testing.T) {
	es := &EmailService{frontendURL: "https://test.com"}
	ns := NewNotificationService(es)

	alert := &models.AlertRule{
		Name:       "Volume Alert",
		Symbol:     "GOOGL",
		AlertType:  "volume_above",
		Conditions: json.RawMessage(`{}`),
	}

	body := ns.formatAlertEmailBody(alert, nil, nil)

	assert.Contains(t, body, "<html>")
	assert.Contains(t, body, "</html>")
	assert.Contains(t, body, "Alert Triggered")
	assert.Contains(t, body, "Volume Alert")
	assert.Contains(t, body, "GOOGL")
}

// ---------------------------------------------------------------------------
// GetInAppNotifications — default limit
// ---------------------------------------------------------------------------

func TestGetInAppNotifications_DefaultLimit(t *testing.T) {
	// The function sets default limit to 50 if 0 is passed.
	// We can't test the full path without a database, but we can verify
	// the service can be instantiated.
	es := &EmailService{}
	ns := NewNotificationService(es)
	require.NotNil(t, ns)
}
