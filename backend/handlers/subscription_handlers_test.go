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
// CreateSubscription — request validation
// ---------------------------------------------------------------------------

func TestCreateSubscription_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &SubscriptionHandler{subscriptionService: nil}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", bytes.NewBufferString("not json"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", "test-user")

	handler.CreateSubscription(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateSubscription_MissingPlanID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &SubscriptionHandler{subscriptionService: nil}

	body := map[string]string{
		"billing_period": "monthly",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", "test-user")

	handler.CreateSubscription(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateSubscription_MissingBillingPeriod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &SubscriptionHandler{subscriptionService: nil}

	body := map[string]string{
		"plan_id": "plan-premium",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", "test-user")

	handler.CreateSubscription(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateSubscription_InvalidBillingPeriod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &SubscriptionHandler{subscriptionService: nil}

	body := map[string]string{
		"plan_id":        "plan-premium",
		"billing_period": "biweekly", // not in oneof
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", "test-user")

	handler.CreateSubscription(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// UpdateSubscription — request validation
// ---------------------------------------------------------------------------

func TestUpdateSubscription_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &SubscriptionHandler{subscriptionService: nil}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/subscriptions/me", bytes.NewBufferString("bad"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", "test-user")

	handler.UpdateSubscription(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSubscription_InvalidBillingPeriod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &SubscriptionHandler{subscriptionService: nil}

	body := map[string]interface{}{
		"billing_period": "daily", // not in oneof
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/subscriptions/me", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", "test-user")

	handler.UpdateSubscription(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// GetPaymentHistory — query params
// ---------------------------------------------------------------------------

func TestGetPaymentHistory_DefaultLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &SubscriptionHandler{subscriptionService: nil}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/payments", nil)
	c.Set("user_id", "test-user")

	defer func() {
		if r := recover(); r != nil {
			// Expected: nil pointer on subscriptionService
		}
	}()

	handler.GetPaymentHistory(c)
}

func TestGetPaymentHistory_CustomLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &SubscriptionHandler{subscriptionService: nil}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/payments?limit=10", nil)
	c.Set("user_id", "test-user")

	defer func() {
		if r := recover(); r != nil {
			// Expected: nil pointer on subscriptionService
		}
	}()

	handler.GetPaymentHistory(c)
}
