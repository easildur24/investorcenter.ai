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
// UpdateNotificationPreferences — request validation
// ---------------------------------------------------------------------------

func TestUpdateNotificationPreferences_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &NotificationHandler{notificationService: nil}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", bytes.NewBufferString("bad json"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", "test-user")

	handler.UpdateNotificationPreferences(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateNotificationPreferences_InvalidEmailAddress(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &NotificationHandler{notificationService: nil}

	body := map[string]interface{}{
		"email_address": "not-an-email",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", "test-user")

	handler.UpdateNotificationPreferences(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateNotificationPreferences_ValidPartialUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &NotificationHandler{notificationService: nil}

	emailEnabled := true
	body := map[string]interface{}{
		"email_enabled": emailEnabled,
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", "test-user")

	// Will fail at the service layer (nil), but JSON binding should succeed
	defer func() {
		if r := recover(); r != nil {
			// Expected: nil pointer on notificationService
		}
	}()

	handler.UpdateNotificationPreferences(c)

	// If we get 400, binding failed; if something else, binding succeeded
	assert.NotEqual(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// GetInAppNotifications — query params
// ---------------------------------------------------------------------------

func TestGetInAppNotifications_QueryParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &NotificationHandler{notificationService: nil}

	tests := []struct {
		name  string
		query string
	}{
		{"default params", ""},
		{"unread only", "unread_only=true"},
		{"custom limit", "limit=10"},
		{"combined", "unread_only=true&limit=25"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			url := "/api/v1/notifications"
			if tt.query != "" {
				url += "?" + tt.query
			}
			c.Request = httptest.NewRequest(http.MethodGet, url, nil)
			c.Set("user_id", "test-user")

			defer func() {
				if r := recover(); r != nil {
					// Expected: nil pointer on notificationService
				}
			}()

			handler.GetInAppNotifications(c)
		})
	}
}
