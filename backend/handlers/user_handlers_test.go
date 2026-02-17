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
// GetCurrentUser — no auth returns 401
// ---------------------------------------------------------------------------

func TestGetCurrentUser_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/api/v1/users/me", GetCurrentUser)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------------------------------------------------------------------------
// UpdateProfile — request validation
// ---------------------------------------------------------------------------

func TestUpdateProfile_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.PUT("/api/v1/users/me", UpdateProfile)

	body := map[string]string{"full_name": "New Name"}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdateProfile_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Next()
	})
	router.PUT("/api/v1/users/me", UpdateProfile)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me", bytes.NewBufferString("bad json"))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// ChangePassword — request validation
// ---------------------------------------------------------------------------

func TestChangePassword_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/api/v1/users/me/change-password", ChangePassword)

	body := map[string]string{
		"current_password": "old123456",
		"new_password":     "new123456",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/change-password", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestChangePassword_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Next()
	})
	router.POST("/api/v1/users/me/change-password", ChangePassword)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/change-password", bytes.NewBufferString("bad"))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestChangePassword_MissingCurrentPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Next()
	})
	router.POST("/api/v1/users/me/change-password", ChangePassword)

	body := map[string]string{
		"new_password": "new123456",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/change-password", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestChangePassword_MissingNewPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Next()
	})
	router.POST("/api/v1/users/me/change-password", ChangePassword)

	body := map[string]string{
		"current_password": "old123456",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/change-password", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestChangePassword_NewPasswordTooShort(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Next()
	})
	router.POST("/api/v1/users/me/change-password", ChangePassword)

	body := map[string]string{
		"current_password": "oldpass123",
		"new_password":     "short",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/change-password", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// DeleteAccount — no auth returns 401
// ---------------------------------------------------------------------------

func TestDeleteAccount_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.DELETE("/api/v1/users/me", DeleteAccount)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
