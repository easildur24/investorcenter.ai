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
)

// ==================== Helper Function Tests ====================

func TestJoinStrings(t *testing.T) {
	tests := []struct {
		name     string
		parts    []string
		sep      string
		expected string
	}{
		{"empty slice", nil, ", ", ""},
		{"single element", []string{"a"}, ", ", "a"},
		{"multiple elements", []string{"a", "b", "c"}, ", ", "a, b, c"},
		{"different separator", []string{"x", "y"}, " AND ", "x AND y"},
		{"empty strings", []string{"", ""}, ",", ","},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, joinStrings(tt.parts, tt.sep))
		})
	}
}

func TestGetUserID(t *testing.T) {
	t.Run("returns user ID when set", func(t *testing.T) {
		r := setupRouterWithMockAuth("user-123", "user@test.com", false)
		var gotID string
		var gotExists bool
		r.GET("/test", func(c *gin.Context) {
			gotID, gotExists = getUserID(c)
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		assert.True(t, gotExists)
		assert.Equal(t, "user-123", gotID)
	})

	t.Run("returns empty when not set", func(t *testing.T) {
		r := gin.New()
		var gotID string
		var gotExists bool
		r.GET("/test", func(c *gin.Context) {
			gotID, gotExists = getUserID(c)
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		assert.False(t, gotExists)
		assert.Equal(t, "", gotID)
	})
}

func TestScanTask(t *testing.T) {
	t.Run("succeeds with valid row", func(t *testing.T) {
		mockRow := &mockScanner{err: nil, callCount: 0}
		var task Task
		err := scanTask(mockRow, &task)
		assert.NoError(t, err)
		assert.Equal(t, 1, mockRow.callCount)
	})

	t.Run("returns error on scan failure", func(t *testing.T) {
		mockRow := &mockScanner{err: fmt.Errorf("scan failed")}
		var task Task
		err := scanTask(mockRow, &task)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "scan failed")
	})
}

// mockScanner implements the interface{ Scan(dest ...interface{}) error } interface
type mockScanner struct {
	err       error
	callCount int
}

func (m *mockScanner) Scan(dest ...interface{}) error {
	m.callCount++
	return m.err
}

// ==================== ClaimNextTask Tests ====================

func TestClaimNextTask_DatabaseUnavailable(t *testing.T) {
	_, cleanup := setupMockDB(t)
	cleanup() // sets database.DB = nil

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.POST("/tasks/next", ClaimNextTask)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/tasks/next", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestClaimNextTask_NoTasks(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("UPDATE tasks SET").
		WillReturnError(sql.ErrNoRows)

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.POST("/tasks/next", ClaimNextTask)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/tasks/next", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClaimNextTask_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	createdBy := "admin-1"
	claimedBy := "user-1"

	mock.ExpectQuery("UPDATE tasks SET").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "status", "priority", "task_type_id", "params", "retry_count",
			"created_by", "created_at", "updated_at", "started_at", "completed_at", "claimed_by",
		}).AddRow("task-1", "in_progress", "high", 1, `{}`, 0, &createdBy, now, now, &now, nil, &claimedBy))

	// fetchTaskType query
	mock.ExpectQuery("SELECT id, name, skill_path, param_schema FROM task_types").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "skill_path", "param_schema"}).
			AddRow(1, "reddit_crawl", "data-ingestion", nil))

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.POST("/tasks/next", ClaimNextTask)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/tasks/next", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClaimNextTask_WithTaskTypeFilter(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	createdBy := "admin-1"
	claimedBy := "user-1"

	mock.ExpectQuery("UPDATE tasks SET").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "status", "priority", "task_type_id", "params", "retry_count",
			"created_by", "created_at", "updated_at", "started_at", "completed_at", "claimed_by",
		}).AddRow("task-2", "in_progress", "medium", 2, `{}`, 0, &createdBy, now, now, &now, nil, &claimedBy))

	mock.ExpectQuery("SELECT id, name, skill_path, param_schema FROM task_types").
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "skill_path", "param_schema"}).
			AddRow(2, "scrape_ycharts", "scrape-ycharts-keystats", nil))

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.POST("/tasks/next", ClaimNextTask)

	body, _ := json.Marshal(map[string]interface{}{"task_type": "scrape_ycharts"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/tasks/next", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClaimNextTask_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("UPDATE tasks SET").
		WillReturnError(fmt.Errorf("connection refused"))

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.POST("/tasks/next", ClaimNextTask)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/tasks/next", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ==================== CreateTask Tests ====================

func TestCreateTask_DatabaseUnavailable(t *testing.T) {
	_, cleanup := setupMockDB(t)
	cleanup()

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.POST("/tasks", CreateTask)

	body, _ := json.Marshal(map[string]interface{}{"task_type_id": 1})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestCreateTask_MissingTaskTypeID(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.POST("/tasks", CreateTask)

	body, _ := json.Marshal(map[string]interface{}{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateTask_InvalidPriority(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.POST("/tasks", CreateTask)

	body, _ := json.Marshal(map[string]interface{}{
		"task_type_id": 1,
		"priority":     "super_urgent",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "Invalid priority")
}

func TestCreateTask_DefaultPriority(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	createdBy := "user-1"
	mock.ExpectQuery("INSERT INTO tasks").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "status", "priority", "task_type_id", "params", "retry_count",
			"created_by", "created_at", "updated_at", "started_at", "completed_at", "claimed_by",
		}).AddRow("task-new", "pending", "medium", 1, `{}`, 0, &createdBy, now, now, nil, nil, nil))

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.POST("/tasks", CreateTask)

	body, _ := json.Marshal(map[string]interface{}{
		"task_type_id": 1,
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTask_WithParams(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	createdBy := "user-1"
	mock.ExpectQuery("INSERT INTO tasks").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "status", "priority", "task_type_id", "params", "retry_count",
			"created_by", "created_at", "updated_at", "started_at", "completed_at", "claimed_by",
		}).AddRow("task-new", "pending", "high", 1, `{"ticker":"AAPL"}`, 0, &createdBy, now, now, nil, nil, nil))

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.POST("/tasks", CreateTask)

	body, _ := json.Marshal(map[string]interface{}{
		"task_type_id": 1,
		"priority":     "high",
		"params":       map[string]string{"ticker": "AAPL"},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTask_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("INSERT INTO tasks").
		WillReturnError(fmt.Errorf("foreign key violation"))

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.POST("/tasks", CreateTask)

	body, _ := json.Marshal(map[string]interface{}{
		"task_type_id": 999,
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ==================== UpdateTask Tests ====================

func TestUpdateTask_DatabaseUnavailable(t *testing.T) {
	_, cleanup := setupMockDB(t)
	cleanup()

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.PUT("/tasks/:id", UpdateTask)

	body, _ := json.Marshal(map[string]interface{}{"status": "completed"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/tasks/task-1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestUpdateTask_MissingStatus(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.PUT("/tasks/:id", UpdateTask)

	body, _ := json.Marshal(map[string]interface{}{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/tasks/task-1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateTask_InvalidStatus(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.PUT("/tasks/:id", UpdateTask)

	status := "invalid_status"
	body, _ := json.Marshal(map[string]interface{}{"status": status})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/tasks/task-1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "Invalid status")
}

func TestUpdateTask_ToCompleted(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	createdBy := "admin-1"
	mock.ExpectQuery("UPDATE tasks SET").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "status", "priority", "task_type_id", "params", "retry_count",
			"created_by", "created_at", "updated_at", "started_at", "completed_at", "claimed_by",
		}).AddRow("task-1", "completed", "medium", 1, `{}`, 0, &createdBy, now, now, &now, &now, nil))

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.PUT("/tasks/:id", UpdateTask)

	status := "completed"
	body, _ := json.Marshal(map[string]interface{}{"status": status})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/tasks/task-1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTask_ToFailedWithRetry(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	createdBy := "admin-1"
	mock.ExpectQuery("UPDATE tasks SET").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "status", "priority", "task_type_id", "params", "retry_count",
			"created_by", "created_at", "updated_at", "started_at", "completed_at", "claimed_by",
		}).AddRow("task-1", "failed", "medium", 1, `{}`, 1, &createdBy, now, now, &now, &now, nil))

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.PUT("/tasks/:id", UpdateTask)

	status := "failed"
	body, _ := json.Marshal(map[string]interface{}{"status": status, "increment_retry": true})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/tasks/task-1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTask_ToPending(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	createdBy := "admin-1"
	mock.ExpectQuery("UPDATE tasks SET").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "status", "priority", "task_type_id", "params", "retry_count",
			"created_by", "created_at", "updated_at", "started_at", "completed_at", "claimed_by",
		}).AddRow("task-1", "pending", "medium", 1, `{}`, 0, &createdBy, now, now, nil, nil, nil))

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.PUT("/tasks/:id", UpdateTask)

	status := "pending"
	body, _ := json.Marshal(map[string]interface{}{"status": status})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/tasks/task-1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTask_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("UPDATE tasks SET").
		WillReturnError(sql.ErrNoRows)

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.PUT("/tasks/:id", UpdateTask)

	status := "completed"
	body, _ := json.Marshal(map[string]interface{}{"status": status})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/tasks/nonexistent", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ==================== DeleteTask Tests ====================

func TestDeleteTask_DatabaseUnavailable(t *testing.T) {
	_, cleanup := setupMockDB(t)
	cleanup()

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.DELETE("/tasks/:id", DeleteTask)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/tasks/task-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestDeleteTask_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM tasks").
		WithArgs("task-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.DELETE("/tasks/:id", DeleteTask)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/tasks/task-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteTask_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM tasks").
		WithArgs("nonexistent").
		WillReturnResult(sqlmock.NewResult(0, 0))

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.DELETE("/tasks/:id", DeleteTask)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/tasks/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteTask_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM tasks").
		WithArgs("task-1").
		WillReturnError(fmt.Errorf("connection refused"))

	r := setupRouterWithMockAuth("user-1", "user@test.com", false)
	r.DELETE("/tasks/:id", DeleteTask)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/tasks/task-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
