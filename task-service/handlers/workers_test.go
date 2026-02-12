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
	"task-service/database"
)

// ==================== ListWorkers Tests ====================

func TestListWorkers_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "email", "full_name", "last_login_at", "last_activity_at", "created_at", "task_count"}).
		AddRow("worker-1", "worker1@test.com", "Worker One", now, now, now, 5).
		AddRow("worker-2", "worker2@test.com", "Worker Two", nil, nil, now, 0)

	mock.ExpectQuery("SELECT u.id, u.email, u.full_name").WillReturnRows(rows)

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.GET("/admin/workers", ListWorkers)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/workers", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))

	data := resp["data"].([]interface{})
	assert.Len(t, data, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListWorkers_DBUnavailable(t *testing.T) {
	database.DB = nil

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.GET("/admin/workers", ListWorkers)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/workers", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ==================== RegisterWorker Tests ====================

func TestRegisterWorker_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("INSERT INTO users").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "New Worker").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "full_name", "last_login_at", "last_activity_at", "created_at"}).
			AddRow("worker-new", "new@test.com", "New Worker", nil, nil, now))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.POST("/admin/workers", RegisterWorker)

	body, _ := json.Marshal(map[string]string{
		"email":     "new@test.com",
		"password":  "securePass123",
		"full_name": "New Worker",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/workers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRegisterWorker_MissingFields(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.POST("/admin/workers", RegisterWorker)

	// Missing password
	body, _ := json.Marshal(map[string]string{
		"email":     "new@test.com",
		"full_name": "New Worker",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/workers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	_ = mock
}

func TestRegisterWorker_InvalidJSON(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.POST("/admin/workers", RegisterWorker)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/workers", bytes.NewBufferString("{invalid}"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	_ = mock
}

// ==================== DeleteWorker Tests ====================

func TestDeleteWorker_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE users SET is_worker = FALSE").
		WithArgs("worker-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.DELETE("/admin/workers/:id", DeleteWorker)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/admin/workers/worker-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteWorker_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE users SET is_worker = FALSE").
		WithArgs("nonexistent").
		WillReturnResult(sqlmock.NewResult(0, 0))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.DELETE("/admin/workers/:id", DeleteWorker)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/admin/workers/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== ListTasks Tests ====================

func TestListTasks_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "title", "description", "assigned_to", "status", "priority",
		"task_type_id", "params", "result", "retry_count",
		"created_by", "created_at", "updated_at", "started_at", "completed_at",
		"assigned_to_name", "created_by_name",
		"tt_id", "tt_name", "tt_label",
	}).
		AddRow("task-1", "Test Task", "Description", "worker-1", "pending", "medium",
			1, []byte(`{"key":"val"}`), nil, 0,
			"admin-1", now, now, nil, nil,
			"Worker One", "Admin",
			1, "reddit_crawl", "Reddit Crawl")

	mock.ExpectQuery("SELECT t.id, t.title").WillReturnRows(rows)

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.GET("/admin/workers/tasks", ListTasks)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/workers/tasks", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	data := resp["data"].([]interface{})
	assert.Len(t, data, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListTasks_WithStatusFilter(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{
		"id", "title", "description", "assigned_to", "status", "priority",
		"task_type_id", "params", "result", "retry_count",
		"created_by", "created_at", "updated_at", "started_at", "completed_at",
		"assigned_to_name", "created_by_name",
		"tt_id", "tt_name", "tt_label",
	})

	mock.ExpectQuery("SELECT t.id, t.title").
		WithArgs("pending").
		WillReturnRows(rows)

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.GET("/admin/workers/tasks", ListTasks)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/workers/tasks?status=pending", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== GetTask Tests ====================

func TestGetTask_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "title", "description", "assigned_to", "status", "priority",
		"task_type_id", "params", "result", "retry_count",
		"created_by", "created_at", "updated_at", "started_at", "completed_at",
		"assigned_to_name", "created_by_name",
		"tt_id", "tt_name", "tt_label",
	}).
		AddRow("task-1", "Test Task", "Desc", "worker-1", "pending", "medium",
			1, nil, nil, 0,
			"admin-1", now, now, nil, nil,
			"Worker", "Admin",
			1, "reddit_crawl", "Reddit Crawl")

	mock.ExpectQuery("SELECT t.id, t.title").
		WithArgs("task-1").
		WillReturnRows(rows)

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.GET("/admin/workers/tasks/:id", GetTask)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/workers/tasks/task-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTask_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT t.id, t.title").
		WithArgs("nonexistent").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "assigned_to", "status", "priority",
			"task_type_id", "params", "result", "retry_count",
			"created_by", "created_at", "updated_at", "started_at", "completed_at",
			"assigned_to_name", "created_by_name",
			"tt_id", "tt_name", "tt_label",
		}))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.GET("/admin/workers/tasks/:id", GetTask)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/workers/tasks/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== CreateTask Tests ====================

func TestCreateTask_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("INSERT INTO worker_tasks").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "assigned_to", "status", "priority",
			"task_type_id", "params", "result", "retry_count",
			"created_by", "created_at", "updated_at", "started_at", "completed_at",
		}).AddRow("task-new", "New Task", "Description", "worker-1", "pending", "high",
			1, nil, nil, 0,
			"admin-1", now, now, nil, nil))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.POST("/admin/workers/tasks", CreateTask)

	body, _ := json.Marshal(map[string]interface{}{
		"title":       "New Task",
		"description": "Description",
		"assigned_to": "worker-1",
		"priority":    "high",
		"task_type_id": 1,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/workers/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTask_MissingTitle(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.POST("/admin/workers/tasks", CreateTask)

	body, _ := json.Marshal(map[string]interface{}{
		"description": "No title",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/workers/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	_ = mock
}

func TestCreateTask_DefaultPriority(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("INSERT INTO worker_tasks").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "assigned_to", "status", "priority",
			"task_type_id", "params", "result", "retry_count",
			"created_by", "created_at", "updated_at", "started_at", "completed_at",
		}).AddRow("task-new", "Task", "", nil, "pending", "medium",
			nil, nil, nil, 0,
			"admin-1", now, now, nil, nil))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.POST("/admin/workers/tasks", CreateTask)

	body, _ := json.Marshal(map[string]interface{}{
		"title": "Task",
		// No priority — should default to "medium"
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/workers/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== DeleteTask Tests ====================

func TestDeleteTask_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM worker_tasks").
		WithArgs("task-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.DELETE("/admin/workers/tasks/:id", DeleteTask)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/admin/workers/tasks/task-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteTask_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM worker_tasks").
		WithArgs("nonexistent").
		WillReturnResult(sqlmock.NewResult(0, 0))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.DELETE("/admin/workers/tasks/:id", DeleteTask)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/admin/workers/tasks/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== ListTaskUpdates Tests ====================

func TestListTaskUpdates_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "task_id", "content", "created_by", "created_at", "created_by_name"}).
		AddRow("update-1", "task-1", "Started work", "worker-1", now, "Worker One").
		AddRow("update-2", "task-1", "Completed step 1", "worker-1", now, "Worker One")

	mock.ExpectQuery("SELECT u.id, u.task_id, u.content").
		WithArgs("task-1").
		WillReturnRows(rows)

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.GET("/admin/workers/tasks/:id/updates", ListTaskUpdates)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/workers/tasks/task-1/updates", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	assert.Len(t, data, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== CreateTaskUpdate Tests ====================

func TestCreateTaskUpdate_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("INSERT INTO worker_task_updates").
		WithArgs("task-1", "Progress update", "admin-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "task_id", "content", "created_by", "created_at"}).
			AddRow("update-new", "task-1", "Progress update", "admin-1", now))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.POST("/admin/workers/tasks/:id/updates", CreateTaskUpdate)

	body, _ := json.Marshal(map[string]string{"content": "Progress update"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/workers/tasks/task-1/updates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTaskUpdate_MissingContent(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.POST("/admin/workers/tasks/:id/updates", CreateTaskUpdate)

	body, _ := json.Marshal(map[string]string{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/workers/tasks/task-1/updates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	_ = mock
}
