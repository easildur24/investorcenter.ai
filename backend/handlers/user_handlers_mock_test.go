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
	"github.com/stretchr/testify/assert"
	"investorcenter-api/auth"
)

// ---------------------------------------------------------------------------
// GetCurrentUser — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetCurrentUser_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	hash := "hash"

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

	r := setupMockRouter("user-1")
	r.GET("/me", GetCurrentUser)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "user-1", resp["id"])
	assert.Equal(t, "test@example.com", resp["email"])
	assert.Equal(t, "Test User", resp["full_name"])
	// password_hash should NOT appear in response (UserPublic)
	_, hasHash := resp["password_hash"]
	assert.False(t, hasHash, "password_hash should not be in public response")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCurrentUser_UserNotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .+ FROM users WHERE id = \\$1").
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	r := setupMockRouter("nonexistent")
	r.GET("/me", GetCurrentUser)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCurrentUser_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .+ FROM users WHERE id = \\$1").
		WithArgs("user-1").
		WillReturnError(fmt.Errorf("connection refused"))

	r := setupMockRouter("user-1")
	r.GET("/me", GetCurrentUser)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// UpdateProfile — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestUpdateProfile_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	hash := "hash"

	// GetUserByID
	mock.ExpectQuery("SELECT .+ FROM users WHERE id = \\$1").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "password_hash", "full_name", "timezone",
			"created_at", "updated_at", "last_login_at", "email_verified",
			"is_premium", "is_active", "is_admin", "is_worker", "last_activity_at",
		}).AddRow(
			"user-1", "test@example.com", &hash, "Old Name", "UTC",
			now, now, nil, true,
			false, true, false, false, nil,
		))

	// UpdateUser
	mock.ExpectExec("UPDATE users SET full_name").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := setupMockRouter("user-1")
	r.PUT("/me", UpdateProfile)

	body, _ := json.Marshal(map[string]string{
		"full_name": "New Name",
		"timezone":  "US/Eastern",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/me", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "New Name", resp["full_name"])
	assert.Equal(t, "US/Eastern", resp["timezone"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateProfile_UserNotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .+ FROM users WHERE id = \\$1").
		WithArgs("user-gone").
		WillReturnError(sql.ErrNoRows)

	r := setupMockRouter("user-gone")
	r.PUT("/me", UpdateProfile)

	body, _ := json.Marshal(map[string]string{
		"full_name": "New Name",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/me", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateProfile_UpdateFails(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	hash := "hash"

	mock.ExpectQuery("SELECT .+ FROM users WHERE id = \\$1").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "password_hash", "full_name", "timezone",
			"created_at", "updated_at", "last_login_at", "email_verified",
			"is_premium", "is_active", "is_admin", "is_worker", "last_activity_at",
		}).AddRow(
			"user-1", "test@example.com", &hash, "Old Name", "UTC",
			now, now, nil, true,
			false, true, false, false, nil,
		))

	mock.ExpectExec("UPDATE users SET full_name").
		WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouter("user-1")
	r.PUT("/me", UpdateProfile)

	body, _ := json.Marshal(map[string]string{
		"full_name": "New Name",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/me", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Failed to update profile", resp["error"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// ChangePassword — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestChangePassword_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	oldPassword := "oldpass12345"
	oldHash, _ := auth.HashPassword(oldPassword)

	mock.ExpectQuery("SELECT .+ FROM users WHERE id = \\$1").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "password_hash", "full_name", "timezone",
			"created_at", "updated_at", "last_login_at", "email_verified",
			"is_premium", "is_active", "is_admin", "is_worker", "last_activity_at",
		}).AddRow(
			"user-1", "test@example.com", &oldHash, "Test User", "UTC",
			now, now, nil, true,
			false, true, false, false, nil,
		))

	// UpdateUserPassword
	mock.ExpectExec("UPDATE users SET password_hash").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := setupMockRouter("user-1")
	r.POST("/change-password", ChangePassword)

	body, _ := json.Marshal(map[string]string{
		"current_password": oldPassword,
		"new_password":     "newpass12345",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/change-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Password changed successfully", resp["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChangePassword_WrongCurrentPassword(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	correctHash, _ := auth.HashPassword("correct-password")

	mock.ExpectQuery("SELECT .+ FROM users WHERE id = \\$1").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "password_hash", "full_name", "timezone",
			"created_at", "updated_at", "last_login_at", "email_verified",
			"is_premium", "is_active", "is_admin", "is_worker", "last_activity_at",
		}).AddRow(
			"user-1", "test@example.com", &correctHash, "Test User", "UTC",
			now, now, nil, true,
			false, true, false, false, nil,
		))

	r := setupMockRouter("user-1")
	r.POST("/change-password", ChangePassword)

	body, _ := json.Marshal(map[string]string{
		"current_password": "wrong-password",
		"new_password":     "newpass12345",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/change-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Current password is incorrect", resp["error"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChangePassword_NilPasswordHash(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// OAuth user with no password
	mock.ExpectQuery("SELECT .+ FROM users WHERE id = \\$1").
		WithArgs("user-oauth").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "password_hash", "full_name", "timezone",
			"created_at", "updated_at", "last_login_at", "email_verified",
			"is_premium", "is_active", "is_admin", "is_worker", "last_activity_at",
		}).AddRow(
			"user-oauth", "oauth@example.com", nil, "OAuth User", "UTC",
			now, now, nil, true,
			false, true, false, false, nil,
		))

	r := setupMockRouter("user-oauth")
	r.POST("/change-password", ChangePassword)

	body, _ := json.Marshal(map[string]string{
		"current_password": "any-password",
		"new_password":     "newpass12345",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/change-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChangePassword_UserNotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .+ FROM users WHERE id = \\$1").
		WithArgs("user-gone").
		WillReturnError(sql.ErrNoRows)

	r := setupMockRouter("user-gone")
	r.POST("/change-password", ChangePassword)

	body, _ := json.Marshal(map[string]string{
		"current_password": "oldpass12345",
		"new_password":     "newpass12345",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/change-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChangePassword_UpdateFails(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	oldPassword := "oldpass12345"
	oldHash, _ := auth.HashPassword(oldPassword)

	mock.ExpectQuery("SELECT .+ FROM users WHERE id = \\$1").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "password_hash", "full_name", "timezone",
			"created_at", "updated_at", "last_login_at", "email_verified",
			"is_premium", "is_active", "is_admin", "is_worker", "last_activity_at",
		}).AddRow(
			"user-1", "test@example.com", &oldHash, "Test User", "UTC",
			now, now, nil, true,
			false, true, false, false, nil,
		))

	mock.ExpectExec("UPDATE users SET password_hash").
		WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouter("user-1")
	r.POST("/change-password", ChangePassword)

	body, _ := json.Marshal(map[string]string{
		"current_password": oldPassword,
		"new_password":     "newpass12345",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/change-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Failed to update password", resp["error"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// DeleteAccount — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestDeleteAccount_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// SoftDeleteUser
	mock.ExpectExec("UPDATE users SET is_active = FALSE").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// DeleteUserSessions
	mock.ExpectExec("DELETE FROM sessions WHERE user_id = \\$1").
		WillReturnResult(sqlmock.NewResult(0, 3))

	r := setupMockRouter("user-1")
	r.DELETE("/me", DeleteAccount)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/me", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Account deleted successfully", resp["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteAccount_SoftDeleteFails(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE users SET is_active = FALSE").
		WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouter("user-1")
	r.DELETE("/me", DeleteAccount)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/me", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Failed to delete account", resp["error"])
	assert.NoError(t, mock.ExpectationsWereMet())
}
