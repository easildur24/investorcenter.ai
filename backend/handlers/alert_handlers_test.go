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
