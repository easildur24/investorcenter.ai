package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"investorcenter-api/auth"
)

// ---------------------------------------------------------------------------
// Login — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestLogin_UserNotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// database.GetUserByEmail does a QueryRow with SELECT ... FROM users WHERE email = $1
	mock.ExpectQuery("SELECT .+ FROM users WHERE email = \\$1").
		WithArgs("nobody@example.com").
		WillReturnError(sql.ErrNoRows)

	body, _ := json.Marshal(map[string]string{
		"email":    "nobody@example.com",
		"password": "password123",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	Login(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Invalid email or password", resp["error"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogin_WrongPassword(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()
	ensureJWTSecret(t)

	// We need a real bcrypt hash for "correct-password"
	correctHash, _ := auth.HashPassword("correct-password")

	now := time.Now()

	mock.ExpectQuery("SELECT .+ FROM users WHERE email = \\$1").
		WithArgs("test@example.com").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "password_hash", "full_name", "timezone",
			"created_at", "updated_at", "last_login_at", "email_verified",
			"is_premium", "is_active", "is_admin", "is_worker", "last_activity_at",
		}).AddRow(
			"user-1", "test@example.com", &correctHash, "Test User", "UTC",
			now, now, nil, true,
			false, true, false, false, nil,
		))

	body, _ := json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "wrong-password",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	Login(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Invalid email or password", resp["error"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogin_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()
	ensureJWTSecret(t)

	password := "securepass123"
	hash, _ := auth.HashPassword(password)
	now := time.Now()

	// GetUserByEmail
	mock.ExpectQuery("SELECT .+ FROM users WHERE email = \\$1").
		WithArgs("test@example.com").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "password_hash", "full_name", "timezone",
			"created_at", "updated_at", "last_login_at", "email_verified",
			"is_premium", "is_active", "is_admin", "is_worker", "last_activity_at",
		}).AddRow(
			"user-1", "test@example.com", &hash, "Test User", "UTC",
			now, now, nil, true,
			false, true, false, false, nil,
		))

	// UpdateLastLogin
	mock.ExpectExec("UPDATE users SET last_login_at").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// CreateSession (INSERT ... RETURNING)
	mock.ExpectQuery("INSERT INTO sessions").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "last_used_at"}).
			AddRow("session-1", now, now))

	body, _ := json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": password,
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	Login(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NotEmpty(t, resp["access_token"])
	assert.NotEmpty(t, resp["refresh_token"])
	assert.NotNil(t, resp["expires_in"])
	assert.NotNil(t, resp["user"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogin_NilPasswordHash(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// Simulate an OAuth-only user (no password hash)
	mock.ExpectQuery("SELECT .+ FROM users WHERE email = \\$1").
		WithArgs("oauth@example.com").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "password_hash", "full_name", "timezone",
			"created_at", "updated_at", "last_login_at", "email_verified",
			"is_premium", "is_active", "is_admin", "is_worker", "last_activity_at",
		}).AddRow(
			"user-oauth", "oauth@example.com", nil, "OAuth User", "UTC",
			now, now, nil, true,
			false, true, false, false, nil,
		))

	body, _ := json.Marshal(map[string]string{
		"email":    "oauth@example.com",
		"password": "any-password",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	Login(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogin_SessionCreationFailure(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()
	ensureJWTSecret(t)

	password := "securepass123"
	hash, _ := auth.HashPassword(password)
	now := time.Now()

	// GetUserByEmail
	mock.ExpectQuery("SELECT .+ FROM users WHERE email = \\$1").
		WithArgs("test@example.com").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "password_hash", "full_name", "timezone",
			"created_at", "updated_at", "last_login_at", "email_verified",
			"is_premium", "is_active", "is_admin", "is_worker", "last_activity_at",
		}).AddRow(
			"user-1", "test@example.com", &hash, "Test User", "UTC",
			now, now, nil, true,
			false, true, false, false, nil,
		))

	// UpdateLastLogin
	mock.ExpectExec("UPDATE users SET last_login_at").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// CreateSession fails
	mock.ExpectQuery("INSERT INTO sessions").
		WillReturnError(fmt.Errorf("connection refused"))

	body, _ := json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": password,
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	Login(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Failed to create session", resp["error"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Signup — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestSignup_EmailAlreadyExists(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()
	ensureJWTSecret(t)

	now := time.Now()
	hash := "existing-hash"

	// GetUserByEmail returns an existing user
	mock.ExpectQuery("SELECT .+ FROM users WHERE email = \\$1").
		WithArgs("existing@example.com").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "password_hash", "full_name", "timezone",
			"created_at", "updated_at", "last_login_at", "email_verified",
			"is_premium", "is_active", "is_admin", "is_worker", "last_activity_at",
		}).AddRow(
			"user-existing", "existing@example.com", &hash, "Existing User", "UTC",
			now, now, nil, true,
			false, true, false, false, nil,
		))

	body, _ := json.Marshal(map[string]string{
		"email":     "existing@example.com",
		"password":  "securepass123",
		"full_name": "John Doe",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	Signup(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Email already registered", resp["error"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSignup_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()
	ensureJWTSecret(t)

	now := time.Now()

	// GetUserByEmail returns no rows (user doesn't exist)
	mock.ExpectQuery("SELECT .+ FROM users WHERE email = \\$1").
		WithArgs("new@example.com").
		WillReturnError(sql.ErrNoRows)

	// CreateUser INSERT ... RETURNING
	mock.ExpectQuery("INSERT INTO users").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow("user-new", now, now))

	// CreateSession INSERT ... RETURNING
	mock.ExpectQuery("INSERT INTO sessions").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "last_used_at"}).
			AddRow("session-new", now, now))

	body, _ := json.Marshal(map[string]string{
		"email":     "new@example.com",
		"password":  "securepass123",
		"full_name": "New User",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	Signup(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NotEmpty(t, resp["access_token"])
	assert.NotEmpty(t, resp["refresh_token"])
	assert.NotNil(t, resp["user"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSignup_CreateUserDBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()
	ensureJWTSecret(t)

	// GetUserByEmail returns no rows
	mock.ExpectQuery("SELECT .+ FROM users WHERE email = \\$1").
		WithArgs("new@example.com").
		WillReturnError(sql.ErrNoRows)

	// CreateUser fails
	mock.ExpectQuery("INSERT INTO users").
		WillReturnError(fmt.Errorf("unique constraint violation"))

	body, _ := json.Marshal(map[string]string{
		"email":     "new@example.com",
		"password":  "securepass123",
		"full_name": "New User",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	Signup(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Failed to create user", resp["error"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSignup_DefaultTimezone(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()
	ensureJWTSecret(t)

	now := time.Now()

	mock.ExpectQuery("SELECT .+ FROM users WHERE email = \\$1").
		WithArgs("tz@example.com").
		WillReturnError(sql.ErrNoRows)

	mock.ExpectQuery("INSERT INTO users").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow("user-tz", now, now))

	mock.ExpectQuery("INSERT INTO sessions").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "last_used_at"}).
			AddRow("session-tz", now, now))

	// No timezone field -> handler should default to UTC
	body, _ := json.Marshal(map[string]string{
		"email":     "tz@example.com",
		"password":  "securepass123",
		"full_name": "TZ User",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	Signup(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// RefreshToken — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestRefreshToken_MissingRefreshToken(t *testing.T) {
	body, _ := json.Marshal(map[string]string{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	RefreshToken(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// GetSessionByRefreshTokenHash returns no rows
	mock.ExpectQuery("SELECT .+ FROM sessions WHERE refresh_token_hash = \\$1").
		WillReturnError(sql.ErrNoRows)

	body, _ := json.Marshal(map[string]string{
		"refresh_token": "invalid-refresh-token",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	RefreshToken(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Invalid or expired refresh token", resp["error"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshToken_UserNotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()
	ensureJWTSecret(t)

	now := time.Now()

	// GetSessionByRefreshTokenHash returns a valid session
	mock.ExpectQuery("SELECT .+ FROM sessions WHERE refresh_token_hash = \\$1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "refresh_token_hash", "expires_at",
			"created_at", "last_used_at", "user_agent", "ip_address",
		}).AddRow(
			"session-1", "user-deleted", "hash", now.Add(24*time.Hour),
			now, now, nil, nil,
		))

	// GetUserByID returns no rows
	mock.ExpectQuery("SELECT .+ FROM users WHERE id = \\$1").
		WithArgs("user-deleted").
		WillReturnError(sql.ErrNoRows)

	body, _ := json.Marshal(map[string]string{
		"refresh_token": "some-refresh-token",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	RefreshToken(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "User not found", resp["error"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshToken_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()
	ensureJWTSecret(t)

	now := time.Now()
	hash := "some-hash"

	// GetSessionByRefreshTokenHash
	mock.ExpectQuery("SELECT .+ FROM sessions WHERE refresh_token_hash = \\$1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "refresh_token_hash", "expires_at",
			"created_at", "last_used_at", "user_agent", "ip_address",
		}).AddRow(
			"session-1", "user-1", "hash", now.Add(24*time.Hour),
			now, now, nil, nil,
		))

	// GetUserByID
	mock.ExpectQuery("SELECT .+ FROM users WHERE id = \\$1").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "password_hash", "full_name", "timezone",
			"created_at", "updated_at", "last_login_at", "email_verified",
			"is_premium", "is_active", "is_admin", "is_worker", "last_activity_at",
		}).AddRow(
			"user-1", "test@example.com", &hash, "Test User", "UTC",
			now, now, nil, true,
			false, true, false, false, nil,
		))

	// UpdateSessionLastUsed
	mock.ExpectExec("UPDATE sessions SET last_used_at").
		WillReturnResult(sqlmock.NewResult(0, 1))

	body, _ := json.Marshal(map[string]string{
		"refresh_token": "some-refresh-token",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	RefreshToken(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NotEmpty(t, resp["access_token"])
	assert.NotNil(t, resp["expires_in"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Logout — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestLogout_ValidToken(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// GetSessionByRefreshTokenHash
	mock.ExpectQuery("SELECT .+ FROM sessions WHERE refresh_token_hash = \\$1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "refresh_token_hash", "expires_at",
			"created_at", "last_used_at", "user_agent", "ip_address",
		}).AddRow(
			"session-1", "user-1", "hash", now.Add(24*time.Hour),
			now, now, nil, nil,
		))

	// DeleteSession
	mock.ExpectExec("DELETE FROM sessions WHERE id = \\$1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	body, _ := json.Marshal(map[string]string{
		"refresh_token": "some-token",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	Logout(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Logged out successfully", resp["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogout_InvalidToken_StillSucceeds(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Session not found - logout still succeeds
	mock.ExpectQuery("SELECT .+ FROM sessions WHERE refresh_token_hash = \\$1").
		WillReturnError(sql.ErrNoRows)

	body, _ := json.Marshal(map[string]string{
		"refresh_token": "nonexistent-token",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	Logout(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// VerifyEmail — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestVerifyEmail_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// VerifyEmail: UPDATE users SET email_verified = TRUE WHERE ...
	mock.ExpectExec("UPDATE users SET email_verified = TRUE").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify-email?token=valid-token", nil)

	VerifyEmail(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Email verified successfully", resp["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestVerifyEmail_InvalidToken(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// VerifyEmail returns 0 rows affected (invalid/expired token)
	mock.ExpectExec("UPDATE users SET email_verified = TRUE").
		WillReturnResult(sqlmock.NewResult(0, 0))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify-email?token=expired-token", nil)

	VerifyEmail(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestVerifyEmail_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE users SET email_verified = TRUE").
		WillReturnError(fmt.Errorf("connection error"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify-email?token=some-token", nil)

	VerifyEmail(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// ForgotPassword — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestForgotPassword_UserNotFound_StillSucceeds(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// GetUserByEmail returns no rows - user doesn't exist
	mock.ExpectQuery("SELECT .+ FROM users WHERE email = \\$1").
		WithArgs("nonexistent@example.com").
		WillReturnError(sql.ErrNoRows)

	body, _ := json.Marshal(map[string]string{
		"email": "nonexistent@example.com",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	ForgotPassword(c)

	// Should still return 200 to prevent email enumeration
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["message"], "If the email exists")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestForgotPassword_UserExists(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	hash := "some-hash"

	// GetUserByEmail
	mock.ExpectQuery("SELECT .+ FROM users WHERE email = \\$1").
		WithArgs("user@example.com").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "password_hash", "full_name", "timezone",
			"created_at", "updated_at", "last_login_at", "email_verified",
			"is_premium", "is_active", "is_admin", "is_worker", "last_activity_at",
		}).AddRow(
			"user-1", "user@example.com", &hash, "Test User", "UTC",
			now, now, nil, true,
			false, true, false, false, nil,
		))

	// SetPasswordResetToken
	mock.ExpectExec("UPDATE users SET password_reset_token").
		WillReturnResult(sqlmock.NewResult(0, 1))

	body, _ := json.Marshal(map[string]string{
		"email": "user@example.com",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	ForgotPassword(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// ResetPassword — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestResetPassword_InvalidToken(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// GetUserByPasswordResetToken returns no rows
	mock.ExpectQuery("SELECT .+ FROM users WHERE password_reset_token = \\$1").
		WillReturnError(sql.ErrNoRows)

	body, _ := json.Marshal(map[string]string{
		"token":        "invalid-token",
		"new_password": "newpass12345",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	ResetPassword(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Invalid or expired reset token", resp["error"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestResetPassword_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	hash := "old-hash"

	// GetUserByPasswordResetToken
	mock.ExpectQuery("SELECT .+ FROM users WHERE password_reset_token = \\$1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "password_hash", "full_name", "timezone",
			"created_at", "updated_at", "last_login_at", "email_verified",
			"is_premium", "is_active", "is_admin", "is_worker", "last_activity_at",
		}).AddRow(
			"user-1", "test@example.com", &hash, "Test User", "UTC",
			now, now, nil, true,
			false, true, false, false, nil,
		))

	// UpdateUserPassword
	mock.ExpectExec("UPDATE users SET password_hash").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// DeleteUserSessions
	mock.ExpectExec("DELETE FROM sessions WHERE user_id = \\$1").
		WillReturnResult(sqlmock.NewResult(0, 2))

	body, _ := json.Marshal(map[string]string{
		"token":        "valid-reset-token",
		"new_password": "newpass12345",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	ResetPassword(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Password reset successfully", resp["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestResetPassword_UpdatePasswordDBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	hash := "old-hash"

	mock.ExpectQuery("SELECT .+ FROM users WHERE password_reset_token = \\$1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "password_hash", "full_name", "timezone",
			"created_at", "updated_at", "last_login_at", "email_verified",
			"is_premium", "is_active", "is_admin", "is_worker", "last_activity_at",
		}).AddRow(
			"user-1", "test@example.com", &hash, "Test User", "UTC",
			now, now, nil, true,
			false, true, false, false, nil,
		))

	// UpdateUserPassword fails
	mock.ExpectExec("UPDATE users SET password_hash").
		WillReturnError(fmt.Errorf("db error"))

	body, _ := json.Marshal(map[string]string{
		"token":        "valid-reset-token",
		"new_password": "newpass12345",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	ResetPassword(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Failed to update password", resp["error"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// hashToken and ptrTime — additional coverage
// ---------------------------------------------------------------------------

func TestPtrTime(t *testing.T) {
	now := time.Now()
	p := ptrTime(now)
	require.NotNil(t, p)
	assert.Equal(t, now, *p)
}
