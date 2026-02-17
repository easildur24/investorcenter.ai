package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------------------------------------------------------------------------
// hashToken — pure function
// ---------------------------------------------------------------------------

func TestHashToken_Deterministic(t *testing.T) {
	hash1 := hashToken("my-token-123")
	hash2 := hashToken("my-token-123")
	assert.Equal(t, hash1, hash2, "same input should produce same hash")
}

func TestHashToken_DifferentInputs(t *testing.T) {
	hash1 := hashToken("token-a")
	hash2 := hashToken("token-b")
	assert.NotEqual(t, hash1, hash2, "different inputs should produce different hashes")
}

func TestHashToken_HexEncoded(t *testing.T) {
	hash := hashToken("test")
	// SHA-256 produces 64 hex characters
	assert.Len(t, hash, 64)
}

func TestHashToken_EmptyString(t *testing.T) {
	hash := hashToken("")
	assert.Len(t, hash, 64, "even empty string should produce a valid hash")
}

// ---------------------------------------------------------------------------
// generateRandomToken
// ---------------------------------------------------------------------------

func TestGenerateRandomToken_Length(t *testing.T) {
	token, err := generateRandomToken(32)
	require.NoError(t, err)
	// 32 bytes -> 64 hex chars
	assert.Len(t, token, 64)
}

func TestGenerateRandomToken_Uniqueness(t *testing.T) {
	token1, err := generateRandomToken(32)
	require.NoError(t, err)

	token2, err := generateRandomToken(32)
	require.NoError(t, err)

	assert.NotEqual(t, token1, token2, "tokens should be unique")
}

func TestGenerateRandomToken_DifferentLengths(t *testing.T) {
	tests := []struct {
		length     int
		expectedHx int
	}{
		{16, 32},
		{32, 64},
		{64, 128},
	}
	for _, tt := range tests {
		token, err := generateRandomToken(tt.length)
		require.NoError(t, err)
		assert.Len(t, token, tt.expectedHx)
	}
}

// ---------------------------------------------------------------------------
// ptrString, ptrTime helpers
// ---------------------------------------------------------------------------

func TestPtrString(t *testing.T) {
	p := ptrString("hello")
	require.NotNil(t, p)
	assert.Equal(t, "hello", *p)
}

func TestPtrString_Empty(t *testing.T) {
	p := ptrString("")
	require.NotNil(t, p)
	assert.Equal(t, "", *p)
}

// ---------------------------------------------------------------------------
// Signup — request validation
// ---------------------------------------------------------------------------

func TestSignup_InvalidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewBufferString("not json"))
	c.Request.Header.Set("Content-Type", "application/json")

	Signup(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSignup_MissingEmail(t *testing.T) {
	body := map[string]string{
		"password":  "securepass123",
		"full_name": "John Doe",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	Signup(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSignup_MissingPassword(t *testing.T) {
	body := map[string]string{
		"email":     "test@example.com",
		"full_name": "John Doe",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	Signup(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSignup_InvalidEmail(t *testing.T) {
	body := map[string]string{
		"email":     "not-an-email",
		"password":  "securepass123",
		"full_name": "John Doe",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	Signup(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSignup_PasswordTooShort(t *testing.T) {
	body := map[string]string{
		"email":     "test@example.com",
		"password":  "short",
		"full_name": "John Doe",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	Signup(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSignup_MissingFullName(t *testing.T) {
	body := map[string]string{
		"email":    "test@example.com",
		"password": "securepass123",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	Signup(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// Login — request validation
// ---------------------------------------------------------------------------

func TestLogin_InvalidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString("bad json"))
	c.Request.Header.Set("Content-Type", "application/json")

	Login(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_MissingEmail(t *testing.T) {
	body := map[string]string{"password": "pass123456"}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	Login(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_MissingPassword(t *testing.T) {
	body := map[string]string{"email": "test@example.com"}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	Login(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// RefreshToken — request validation
// ---------------------------------------------------------------------------

func TestRefreshToken_InvalidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString("{}"))
	c.Request.Header.Set("Content-Type", "application/json")

	RefreshToken(c)

	// Missing required field refresh_token -> 400
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// Logout — request validation
// ---------------------------------------------------------------------------

func TestLogout_InvalidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewBufferString("not json"))
	c.Request.Header.Set("Content-Type", "application/json")

	Logout(c)

	// Logout with invalid JSON returns 400 due to ShouldBindJSON
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// ForgotPassword — request validation
// ---------------------------------------------------------------------------

func TestForgotPassword_InvalidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewBufferString("bad"))
	c.Request.Header.Set("Content-Type", "application/json")

	ForgotPassword(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestForgotPassword_MissingEmail(t *testing.T) {
	body := map[string]string{}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	ForgotPassword(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// ResetPassword — request validation
// ---------------------------------------------------------------------------

func TestResetPassword_InvalidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewBufferString("bad"))
	c.Request.Header.Set("Content-Type", "application/json")

	ResetPassword(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResetPassword_MissingToken(t *testing.T) {
	body := map[string]string{
		"new_password": "newpass12345",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	ResetPassword(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResetPassword_MissingNewPassword(t *testing.T) {
	body := map[string]string{
		"token": "valid-reset-token",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	ResetPassword(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResetPassword_PasswordTooShort(t *testing.T) {
	body := map[string]string{
		"token":        "valid-token",
		"new_password": "short",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	ResetPassword(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// VerifyEmail — request validation
// ---------------------------------------------------------------------------

func TestVerifyEmail_MissingToken(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify-email", nil)

	VerifyEmail(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Verification token required", resp["error"])
}
