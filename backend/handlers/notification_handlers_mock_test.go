package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"investorcenter-api/services"
)

// newTestNotificationHandler creates a NotificationHandler with a real
// NotificationService that hits the mocked database.DB via sqlmock.
func newTestNotificationHandler() *NotificationHandler {
	return NewNotificationHandler(services.NewNotificationService(nil))
}

// ---------------------------------------------------------------------------
// GetNotificationPreferences — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetNotificationPreferences_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// GetNotificationPreferences: SELECT ... FROM notification_preferences
	mock.ExpectQuery("SELECT .+ FROM notification_preferences").
		WillReturnError(fmt.Errorf("db error"))

	handler := newTestNotificationHandler()
	r := setupMockRouter("user-1")
	r.GET("/notifications/preferences", handler.GetNotificationPreferences)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications/preferences", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// GetUnreadCount — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetUnreadCount_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT COUNT").
		WillReturnError(fmt.Errorf("db error"))

	handler := newTestNotificationHandler()
	r := setupMockRouter("user-1")
	r.GET("/notifications/unread-count", handler.GetUnreadCount)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications/unread-count", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// MarkNotificationRead — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestMarkNotificationRead_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// MarkNotificationAsRead: UPDATE notification_queue SET is_read = true ...
	mock.ExpectExec("UPDATE notification_queue SET is_read").
		WillReturnResult(sqlmock.NewResult(0, 1))

	handler := newTestNotificationHandler()
	r := setupMockRouter("user-1")
	r.POST("/notifications/:id/read", handler.MarkNotificationRead)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/notifications/notif-1/read", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMarkNotificationRead_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE notification_queue SET is_read").
		WillReturnError(fmt.Errorf("db error"))

	handler := newTestNotificationHandler()
	r := setupMockRouter("user-1")
	r.POST("/notifications/:id/read", handler.MarkNotificationRead)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/notifications/notif-1/read", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// MarkAllNotificationsRead — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestMarkAllNotificationsRead_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE notification_queue SET is_read").
		WillReturnResult(sqlmock.NewResult(0, 5))

	handler := newTestNotificationHandler()
	r := setupMockRouter("user-1")
	r.POST("/notifications/read-all", handler.MarkAllNotificationsRead)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/notifications/read-all", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMarkAllNotificationsRead_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE notification_queue SET is_read").
		WillReturnError(fmt.Errorf("db error"))

	handler := newTestNotificationHandler()
	r := setupMockRouter("user-1")
	r.POST("/notifications/read-all", handler.MarkAllNotificationsRead)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/notifications/read-all", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// DismissNotification — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestDismissNotification_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE notification_queue SET is_dismissed").
		WillReturnResult(sqlmock.NewResult(0, 1))

	handler := newTestNotificationHandler()
	r := setupMockRouter("user-1")
	r.POST("/notifications/:id/dismiss", handler.DismissNotification)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/notifications/notif-1/dismiss", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDismissNotification_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE notification_queue SET is_dismissed").
		WillReturnError(fmt.Errorf("db error"))

	handler := newTestNotificationHandler()
	r := setupMockRouter("user-1")
	r.POST("/notifications/:id/dismiss", handler.DismissNotification)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/notifications/notif-1/dismiss", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}
