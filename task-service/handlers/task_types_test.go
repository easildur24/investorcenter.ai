package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// ==================== ListTaskTypes Tests ====================

func TestListTaskTypes_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "name", "label", "sop", "param_schema", "is_active", "created_at", "updated_at",
	}).
		AddRow(1, "reddit_crawl", "Reddit Crawl", "SOP text", nil, true, now, now).
		AddRow(2, "ai_sentiment", "AI Sentiment", "SOP text 2", nil, true, now, now)

	mock.ExpectQuery("SELECT id, name, label, sop, param_schema").WillReturnRows(rows)

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.GET("/admin/workers/task-types", ListTaskTypes)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/workers/task-types", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	data := resp["data"].([]interface{})
	assert.Len(t, data, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListTaskTypes_Empty(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{
		"id", "name", "label", "sop", "param_schema", "is_active", "created_at", "updated_at",
	})

	mock.ExpectQuery("SELECT id, name, label, sop, param_schema").WillReturnRows(rows)

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.GET("/admin/workers/task-types", ListTaskTypes)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/workers/task-types", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	assert.Len(t, data, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== CreateTaskType Tests ====================

func TestCreateTaskType_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("INSERT INTO task_types").
		WithArgs("new_type", "New Type", "SOP instructions", nil).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "label", "sop", "param_schema", "is_active", "created_at", "updated_at",
		}).AddRow(3, "new_type", "New Type", "SOP instructions", nil, true, now, now))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.POST("/admin/workers/task-types", CreateTaskType)

	body, _ := json.Marshal(map[string]interface{}{
		"name":  "new_type",
		"label": "New Type",
		"sop":   "SOP instructions",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/workers/task-types", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTaskType_InvalidName_Uppercase(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.POST("/admin/workers/task-types", CreateTaskType)

	body, _ := json.Marshal(map[string]interface{}{
		"name":  "InvalidName",
		"label": "Invalid Name",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/workers/task-types", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "lowercase alphanumeric")
	_ = mock
}

func TestCreateTaskType_InvalidName_Spaces(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.POST("/admin/workers/task-types", CreateTaskType)

	body, _ := json.Marshal(map[string]interface{}{
		"name":  "has spaces",
		"label": "Has Spaces",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/workers/task-types", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	_ = mock
}

func TestCreateTaskType_InvalidName_SpecialChars(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.POST("/admin/workers/task-types", CreateTaskType)

	body, _ := json.Marshal(map[string]interface{}{
		"name":  "type-with-dashes",
		"label": "Dashes",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/workers/task-types", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	_ = mock
}

func TestCreateTaskType_ValidName_Underscore(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("INSERT INTO task_types").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "label", "sop", "param_schema", "is_active", "created_at", "updated_at",
		}).AddRow(4, "my_task_type", "My Task Type", "", nil, true, now, now))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.POST("/admin/workers/task-types", CreateTaskType)

	body, _ := json.Marshal(map[string]interface{}{
		"name":  "my_task_type",
		"label": "My Task Type",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/workers/task-types", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTaskType_MissingName(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.POST("/admin/workers/task-types", CreateTaskType)

	body, _ := json.Marshal(map[string]interface{}{
		"label": "No Name",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/workers/task-types", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	_ = mock
}

func TestCreateTaskType_MissingLabel(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.POST("/admin/workers/task-types", CreateTaskType)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "no_label",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/workers/task-types", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	_ = mock
}

// ==================== UpdateTaskType Tests ====================

func TestUpdateTaskType_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("UPDATE task_types SET").
		WithArgs("1", stringPtr("Updated Label"), stringPtr("Updated SOP"), nil).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "label", "sop", "param_schema", "is_active", "created_at", "updated_at",
		}).AddRow(1, "reddit_crawl", "Updated Label", "Updated SOP", nil, true, now, now))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.PUT("/admin/workers/task-types/:id", UpdateTaskType)

	body, _ := json.Marshal(map[string]interface{}{
		"label": "Updated Label",
		"sop":   "Updated SOP",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/admin/workers/task-types/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTaskType_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("UPDATE task_types SET").
		WithArgs("999", nil, stringPtr("SOP"), nil).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "label", "sop", "param_schema", "is_active", "created_at", "updated_at",
		}))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.PUT("/admin/workers/task-types/:id", UpdateTaskType)

	body, _ := json.Marshal(map[string]interface{}{
		"sop": "SOP",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/admin/workers/task-types/999", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== DeleteTaskType Tests ====================

func TestDeleteTaskType_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE task_types SET is_active = FALSE").
		WithArgs("1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.DELETE("/admin/workers/task-types/:id", DeleteTaskType)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/admin/workers/task-types/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteTaskType_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE task_types SET is_active = FALSE").
		WithArgs("999").
		WillReturnResult(sqlmock.NewResult(0, 0))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.DELETE("/admin/workers/task-types/:id", DeleteTaskType)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/admin/workers/task-types/999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== WorkerGetTaskType Tests ====================

func TestWorkerGetTaskType_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT id, name, label, sop, param_schema").
		WithArgs("1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "label", "sop", "param_schema", "is_active", "created_at", "updated_at",
		}).AddRow(1, "reddit_crawl", "Reddit Crawl", "SOP text", nil, true, now, now))

	r := setupRouterWithMockAuth("worker-1", "worker@test.com", false)
	r.GET("/worker/task-types/:id", WorkerGetTaskType)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/worker/task-types/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "reddit_crawl", data["name"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerGetTaskType_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT id, name, label, sop, param_schema").
		WithArgs("999").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "label", "sop", "param_schema", "is_active", "created_at", "updated_at",
		}))

	r := setupRouterWithMockAuth("worker-1", "worker@test.com", false)
	r.GET("/worker/task-types/:id", WorkerGetTaskType)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/worker/task-types/999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== Task Type Name Validation Tests ====================

func TestValidTaskTypeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"simple lowercase", "reddit", true},
		{"with underscore", "reddit_crawl", true},
		{"with numbers", "task123", true},
		{"numbers and underscore", "ai_v2_task", true},
		{"single char", "a", true},
		{"uppercase", "Reddit", false},
		{"with space", "reddit crawl", false},
		{"with dash", "reddit-crawl", false},
		{"with dot", "reddit.crawl", false},
		{"empty", "", false},
		{"special chars", "task@type", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, validTaskTypeName.MatchString(tt.input))
		})
	}
}
