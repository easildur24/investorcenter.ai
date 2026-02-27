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
	"investorcenter-api/services"
)

// newTestAlertHandler creates an AlertHandler with a real AlertService that
// will hit the mocked database.DB via sqlmock.
func newTestAlertHandler() *AlertHandler {
	return NewAlertHandler(services.NewAlertService())
}

// ---------------------------------------------------------------------------
// ListAlertRules — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestListAlertRules_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// GetAlertRulesByUserID: SELECT ... FROM alert_rules ar JOIN watch_lists wl ...
	mock.ExpectQuery("SELECT .+ FROM alert_rules ar").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "watch_list_id", "watch_list_item_id", "symbol",
			"alert_type", "conditions", "is_active", "frequency", "notify_email",
			"notify_in_app", "name", "description", "last_triggered_at",
			"trigger_count", "created_at", "updated_at",
			"watch_list_name", "company_name",
		}).AddRow(
			"alert-1", "user-1", "wl-1", nil, "AAPL",
			"price_above", []byte(`{"threshold":150}`), true, "once", true,
			true, "AAPL Alert", nil, nil,
			0, now, now,
			"My Watchlist", "Apple Inc.",
		))

	handler := newTestAlertHandler()
	r := setupMockRouter("user-1")
	r.GET("/alerts", handler.ListAlertRules)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/alerts", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Len(t, resp, 1)
	assert.Equal(t, "AAPL", resp[0]["symbol"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListAlertRules_Mock_Empty(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .+ FROM alert_rules ar").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "watch_list_id", "watch_list_item_id", "symbol",
			"alert_type", "conditions", "is_active", "frequency", "notify_email",
			"notify_in_app", "name", "description", "last_triggered_at",
			"trigger_count", "created_at", "updated_at",
			"watch_list_name", "company_name",
		}))

	handler := newTestAlertHandler()
	r := setupMockRouter("user-1")
	r.GET("/alerts", handler.ListAlertRules)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/alerts", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListAlertRules_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .+ FROM alert_rules ar").
		WillReturnError(fmt.Errorf("connection refused"))

	handler := newTestAlertHandler()
	r := setupMockRouter("user-1")
	r.GET("/alerts", handler.ListAlertRules)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/alerts", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListAlertRules_Mock_WithFilters(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// With watchListID and isActive filters
	mock.ExpectQuery("SELECT .+ FROM alert_rules ar").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "watch_list_id", "watch_list_item_id", "symbol",
			"alert_type", "conditions", "is_active", "frequency", "notify_email",
			"notify_in_app", "name", "description", "last_triggered_at",
			"trigger_count", "created_at", "updated_at",
			"watch_list_name", "company_name",
		}))

	handler := newTestAlertHandler()
	r := setupMockRouter("user-1")
	r.GET("/alerts", handler.ListAlertRules)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/alerts?watch_list_id=wl-1&is_active=true", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// GetAlertRule — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetAlertRule_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	mock.ExpectQuery("SELECT .+ FROM alert_rules WHERE id = \\$1 AND user_id = \\$2").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "watch_list_id", "watch_list_item_id", "symbol",
			"alert_type", "conditions", "is_active", "frequency", "notify_email",
			"notify_in_app", "name", "description", "last_triggered_at",
			"trigger_count", "created_at", "updated_at",
		}).AddRow(
			"alert-1", "user-1", "wl-1", nil, "AAPL",
			"price_above", []byte(`{"threshold":150}`), true, "once", true,
			true, "AAPL Alert", nil, nil,
			0, now, now,
		))

	handler := newTestAlertHandler()
	r := setupMockRouter("user-1")
	r.GET("/alerts/:id", handler.GetAlertRule)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/alerts/alert-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "alert-1", resp["id"])
	assert.Equal(t, "AAPL", resp["symbol"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAlertRule_Mock_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .+ FROM alert_rules WHERE id = \\$1 AND user_id = \\$2").
		WillReturnError(sql.ErrNoRows)

	handler := newTestAlertHandler()
	r := setupMockRouter("user-1")
	r.GET("/alerts/:id", handler.GetAlertRule)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/alerts/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Alert not found", resp["error"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// DeleteAlertRule — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestDeleteAlertRule_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// DeleteAlertRule does DELETE FROM alert_rules WHERE id = $1 AND user_id = $2
	mock.ExpectExec("DELETE FROM alert_rules WHERE id").
		WillReturnResult(sqlmock.NewResult(0, 1))

	handler := newTestAlertHandler()
	r := setupMockRouter("user-1")
	r.DELETE("/alerts/:id", handler.DeleteAlertRule)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/alerts/alert-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteAlertRule_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM alert_rules WHERE id").
		WillReturnError(fmt.Errorf("db error"))

	handler := newTestAlertHandler()
	r := setupMockRouter("user-1")
	r.DELETE("/alerts/:id", handler.DeleteAlertRule)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/alerts/alert-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Failed to delete alert", resp["error"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// MarkAlertLogRead — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestMarkAlertLogRead_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// MarkAlertLogAsRead does UPDATE alert_logs SET is_read = true, read_at = ... WHERE id = $1 AND user_id = $2
	mock.ExpectExec("UPDATE alert_logs SET is_read").
		WillReturnResult(sqlmock.NewResult(0, 1))

	handler := newTestAlertHandler()
	r := setupMockRouter("user-1")
	r.POST("/alerts/logs/:id/read", handler.MarkAlertLogRead)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/alerts/logs/log-1/read", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMarkAlertLogRead_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE alert_logs SET is_read").
		WillReturnError(fmt.Errorf("db error"))

	handler := newTestAlertHandler()
	r := setupMockRouter("user-1")
	r.POST("/alerts/logs/:id/read", handler.MarkAlertLogRead)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/alerts/logs/log-1/read", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// DismissAlertLog — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestDismissAlertLog_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE alert_logs SET is_dismissed").
		WillReturnResult(sqlmock.NewResult(0, 1))

	handler := newTestAlertHandler()
	r := setupMockRouter("user-1")
	r.POST("/alerts/logs/:id/dismiss", handler.DismissAlertLog)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/alerts/logs/log-1/dismiss", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDismissAlertLog_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE alert_logs SET is_dismissed").
		WillReturnError(fmt.Errorf("db error"))

	handler := newTestAlertHandler()
	r := setupMockRouter("user-1")
	r.POST("/alerts/logs/:id/dismiss", handler.DismissAlertLog)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/alerts/logs/log-1/dismiss", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// ListAlertLogs — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestListAlertLogs_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// GetAlertLogs: SELECT ... FROM alert_logs al JOIN alert_rules ar ...
	mock.ExpectQuery("SELECT .+ FROM alert_logs").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "alert_rule_id", "user_id", "symbol", "triggered_at",
			"alert_type", "condition_met", "market_data", "notification_sent",
			"notification_sent_at", "notification_error", "is_read", "read_at",
			"is_dismissed", "dismissed_at", "rule_name",
		}).AddRow(
			"log-1", "alert-1", "user-1", "AAPL", now,
			"price_above", []byte(`{}`), []byte(`{}`), true,
			nil, nil, false, nil,
			false, nil, "My Alert",
		))

	handler := newTestAlertHandler()
	r := setupMockRouter("user-1")
	r.GET("/alerts/logs", handler.ListAlertLogs)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/alerts/logs", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Len(t, resp, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListAlertLogs_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .+ FROM alert_logs").
		WillReturnError(fmt.Errorf("db error"))

	handler := newTestAlertHandler()
	r := setupMockRouter("user-1")
	r.GET("/alerts/logs", handler.ListAlertLogs)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/alerts/logs", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListAlertLogs_Mock_WithParams(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// With alert_id filter, symbol filter, custom limit/offset
	mock.ExpectQuery("SELECT .+ FROM alert_logs").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "alert_rule_id", "user_id", "symbol", "triggered_at",
			"alert_type", "condition_met", "market_data", "notification_sent",
			"notification_sent_at", "notification_error", "is_read", "read_at",
			"is_dismissed", "dismissed_at", "rule_name",
		}))

	handler := newTestAlertHandler()
	r := setupMockRouter("user-1")
	r.GET("/alerts/logs", handler.ListAlertLogs)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/alerts/logs?alert_id=alert-1&symbol=AAPL&limit=10&offset=5", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// UpdateAlertRule — mock-based tests for DB path
// ---------------------------------------------------------------------------

func TestUpdateAlertRule_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// UpdateAlertRule: UPDATE alert_rules SET ... WHERE id = $N AND user_id = $N
	mock.ExpectExec("UPDATE alert_rules SET").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// GetAlertRuleByID after update
	mock.ExpectQuery("SELECT .+ FROM alert_rules WHERE id = \\$1 AND user_id = \\$2").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "watch_list_id", "watch_list_item_id", "symbol",
			"alert_type", "conditions", "is_active", "frequency", "notify_email",
			"notify_in_app", "name", "description", "last_triggered_at",
			"trigger_count", "created_at", "updated_at",
		}).AddRow(
			"alert-1", "user-1", "wl-1", nil, "AAPL",
			"price_above", []byte(`{"threshold":200}`), true, "daily", true,
			true, "Updated Alert", nil, nil,
			0, now, now,
		))

	handler := newTestAlertHandler()
	r := setupMockRouter("user-1")
	r.PUT("/alerts/:id", handler.UpdateAlertRule)

	name := "Updated Alert"
	body, _ := json.Marshal(map[string]interface{}{
		"name": name,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/alerts/alert-1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "alert-1", resp["id"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// CreateAlertRule — mock-based tests for DB path
// ---------------------------------------------------------------------------

func TestCreateAlertRule_Mock_OwnershipFails(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// ValidateWatchListOwnership: GetWatchListByID returns not found
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnError(sql.ErrNoRows)

	handler := newTestAlertHandler()
	r := setupMockRouter("user-1")
	r.POST("/alerts", handler.CreateAlertRule)

	body, _ := json.Marshal(map[string]interface{}{
		"watch_list_id": "wl-other-user",
		"symbol":        "AAPL",
		"alert_type":    "price_above",
		"conditions":    map[string]interface{}{"threshold": 150},
		"name":          "Test Alert",
		"frequency":     "once",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/alerts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Watch list not found", resp["error"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// BulkCreateAlertRules — mock-based tests for DB path
// ---------------------------------------------------------------------------

func TestBulkCreateAlertRules_Mock_OwnershipFails(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// ValidateWatchListOwnership fails
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnError(sql.ErrNoRows)

	handler := newTestAlertHandler()
	r := setupMockRouter("user-1")
	r.POST("/alerts/bulk", handler.BulkCreateAlertRules)

	body, _ := json.Marshal(map[string]interface{}{
		"watch_list_id": "wl-other",
		"alert_type":    "price_above",
		"conditions":    map[string]interface{}{"threshold": 100},
		"frequency":     "daily",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/alerts/bulk", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}
