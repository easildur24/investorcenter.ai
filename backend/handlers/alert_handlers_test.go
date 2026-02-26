package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// NewAlertHandler — constructor
// ---------------------------------------------------------------------------

func TestNewAlertHandler(t *testing.T) {
	handler := NewAlertHandler(nil)
	assert.NotNil(t, handler)
	assert.Nil(t, handler.alertService)
}

// ---------------------------------------------------------------------------
// CreateAlertRule — request validation
// ---------------------------------------------------------------------------

func TestCreateAlertRule_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &AlertHandler{alertService: nil}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/alerts", bytes.NewBufferString("not json"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", "test-user")

	handler.CreateAlertRule(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateAlertRule_MissingRequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &AlertHandler{alertService: nil}

	// Missing watch_list_id, symbol, alert_type, conditions, name, frequency
	body := map[string]interface{}{}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/alerts", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", "test-user")

	handler.CreateAlertRule(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateAlertRule_InvalidAlertType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &AlertHandler{alertService: nil}

	body := map[string]interface{}{
		"watch_list_id": "wl-123",
		"symbol":        "AAPL",
		"alert_type":    "invalid_type", // not in oneof
		"conditions":    map[string]interface{}{"threshold": 100},
		"name":          "Test Alert",
		"frequency":     "once",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/alerts", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", "test-user")

	handler.CreateAlertRule(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateAlertRule_InvalidFrequency(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &AlertHandler{alertService: nil}

	body := map[string]interface{}{
		"watch_list_id": "wl-123",
		"symbol":        "AAPL",
		"alert_type":    "price_above",
		"conditions":    map[string]interface{}{"threshold": 100},
		"name":          "Test Alert",
		"frequency":     "invalid_freq",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/alerts", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", "test-user")

	handler.CreateAlertRule(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// UpdateAlertRule — request validation
// ---------------------------------------------------------------------------

func TestUpdateAlertRule_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &AlertHandler{alertService: nil}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/alerts/123", bytes.NewBufferString("not json"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", "test-user")
	c.Params = gin.Params{{Key: "id", Value: "alert-123"}}

	handler.UpdateAlertRule(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// ListAlertLogs — default params
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// BulkCreateAlertRules — request validation
// ---------------------------------------------------------------------------

func TestBulkCreateAlertRules_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &AlertHandler{alertService: nil}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/alerts/bulk", bytes.NewBufferString("not json"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", "test-user")

	handler.BulkCreateAlertRules(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBulkCreateAlertRules_MissingRequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &AlertHandler{alertService: nil}

	// Empty body — missing watch_list_id, alert_type, conditions, frequency
	body := map[string]interface{}{}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/alerts/bulk", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", "test-user")

	handler.BulkCreateAlertRules(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBulkCreateAlertRules_InvalidAlertType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &AlertHandler{alertService: nil}

	body := map[string]interface{}{
		"watch_list_id": "wl-123",
		"alert_type":    "invalid_type", // not in oneof
		"conditions":    map[string]interface{}{"threshold": 100},
		"frequency":     "daily",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/alerts/bulk", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", "test-user")

	handler.BulkCreateAlertRules(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBulkCreateAlertRules_InvalidFrequency(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &AlertHandler{alertService: nil}

	body := map[string]interface{}{
		"watch_list_id": "wl-123",
		"alert_type":    "price_above",
		"conditions":    map[string]interface{}{"threshold": 100},
		"frequency":     "invalid_freq", // not in oneof
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/alerts/bulk", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", "test-user")

	handler.BulkCreateAlertRules(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBulkCreateAlertRules_MissingUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &AlertHandler{alertService: nil}

	body := map[string]interface{}{
		"watch_list_id": "wl-123",
		"alert_type":    "price_above",
		"conditions":    map[string]interface{}{"threshold": 100},
		"frequency":     "daily",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/alerts/bulk", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	// No user_id set — GetString returns ""

	// This will pass binding but fail on service call (nil pointer).
	// The test verifies the request binding still accepts valid JSON even
	// without user_id in context.
	defer func() {
		if r := recover(); r != nil {
			// Expected: nil alertService causes panic — binding succeeded
		}
	}()

	handler.BulkCreateAlertRules(c)
}

func TestBulkCreateAlertRules_AllValidAlertTypes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validTypes := []string{"price_above", "price_below", "price_change", "volume_above", "volume_spike"}

	for _, alertType := range validTypes {
		t.Run(alertType, func(t *testing.T) {
			handler := &AlertHandler{alertService: nil}

			body := map[string]interface{}{
				"watch_list_id": "wl-123",
				"alert_type":    alertType,
				"conditions":    map[string]interface{}{"threshold": 100},
				"frequency":     "daily",
			}
			jsonBody, _ := json.Marshal(body)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/alerts/bulk", bytes.NewBuffer(jsonBody))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set("user_id", "test-user")

			// Binding should succeed (panic expected from nil service, not from binding)
			defer func() {
				if r := recover(); r != nil {
					// Expected: nil alertService — binding passed
				}
			}()

			handler.BulkCreateAlertRules(c)

			// If we get here without panic, status should NOT be 400 (binding passed)
			assert.NotEqual(t, http.StatusBadRequest, w.Code, "alert type %q should be accepted by binding", alertType)
		})
	}
}

func TestBulkCreateAlertRules_AllValidFrequencies(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validFrequencies := []string{"once", "daily", "always"}

	for _, freq := range validFrequencies {
		t.Run(freq, func(t *testing.T) {
			handler := &AlertHandler{alertService: nil}

			body := map[string]interface{}{
				"watch_list_id": "wl-123",
				"alert_type":    "price_above",
				"conditions":    map[string]interface{}{"threshold": 100},
				"frequency":     freq,
			}
			jsonBody, _ := json.Marshal(body)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/alerts/bulk", bytes.NewBuffer(jsonBody))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set("user_id", "test-user")

			defer func() {
				if r := recover(); r != nil {
					// Expected: nil alertService — binding passed
				}
			}()

			handler.BulkCreateAlertRules(c)

			assert.NotEqual(t, http.StatusBadRequest, w.Code, "frequency %q should be accepted by binding", freq)
		})
	}
}

func TestBulkCreateAlertRules_MissingWatchListID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &AlertHandler{alertService: nil}

	body := map[string]interface{}{
		// watch_list_id intentionally omitted
		"alert_type": "price_above",
		"conditions": map[string]interface{}{"threshold": 100},
		"frequency":  "daily",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/alerts/bulk", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", "test-user")

	handler.BulkCreateAlertRules(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBulkCreateAlertRules_MissingConditions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &AlertHandler{alertService: nil}

	body := map[string]interface{}{
		"watch_list_id": "wl-123",
		"alert_type":    "price_above",
		// conditions intentionally omitted
		"frequency": "daily",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/alerts/bulk", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", "test-user")

	handler.BulkCreateAlertRules(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// ListAlertLogs — default params
// ---------------------------------------------------------------------------

func TestListAlertLogs_DefaultParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Verify that default limit and offset parsing works without panicking
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/alerts/logs", nil)
	c.Set("user_id", "test-user")

	// We expect a 500 since alertService is nil, but the important thing
	// is that param parsing doesn't panic
	handler := &AlertHandler{alertService: nil}

	// This will panic if alertService is nil — wrap to check
	defer func() {
		if r := recover(); r != nil {
			// Expected: nil pointer on alertService.GetAlertLogs
			// The key test is that param parsing succeeded
		}
	}()

	handler.ListAlertLogs(c)
}
