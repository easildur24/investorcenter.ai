package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"task-service/database"
)

// setupWorkerRouter creates a router with mock auth that simulates an authenticated worker
func setupWorkerRouter() *gin.Engine {
	return setupRouterWithMockAuth("worker-1", "worker@test.com", false)
}

// expectVerifyWorker sets up the sqlmock expectations for the verifyWorker helper
func expectVerifyWorker(mock sqlmock.Sqlmock, userID string, isWorker bool) {
	mock.ExpectQuery("SELECT COALESCE\\(is_worker, FALSE\\) FROM users WHERE id").
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"is_worker"}).AddRow(isWorker))

	if isWorker {
		mock.ExpectExec("UPDATE users SET last_activity_at").
			WithArgs(sqlmock.AnyArg(), userID).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
}

// ==================== WorkerNextTask Tests ====================

func TestWorkerNextTask_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	expectVerifyWorker(mock, "worker-1", true)

	rows := sqlmock.NewRows([]string{
		"id", "title", "description", "assigned_to", "status", "priority",
		"task_type_id", "params", "result", "retry_count",
		"created_by", "created_at", "updated_at", "started_at", "completed_at",
		"tt_id", "tt_name", "tt_label", "tt_sop",
	}).
		AddRow("task-1", "Test Task", "Desc", "worker-1", "in_progress", "high",
			1, nil, nil, 0,
			"admin-1", now, now, &now, nil,
			1, "reddit_crawl", "Reddit Crawl", "SOP text here")

	mock.ExpectQuery("WITH claimed AS").
		WithArgs("worker-1").
		WillReturnRows(rows)

	r := setupWorkerRouter()
	r.POST("/worker/next-task", WorkerNextTask)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/worker/next-task", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "task-1", data["id"])
	assert.Equal(t, "in_progress", data["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerNextTask_NoPendingTasks(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	expectVerifyWorker(mock, "worker-1", true)

	rows := sqlmock.NewRows([]string{
		"id", "title", "description", "assigned_to", "status", "priority",
		"task_type_id", "params", "result", "retry_count",
		"created_by", "created_at", "updated_at", "started_at", "completed_at",
		"tt_id", "tt_name", "tt_label", "tt_sop",
	})

	mock.ExpectQuery("WITH claimed AS").
		WithArgs("worker-1").
		WillReturnRows(rows)

	r := setupWorkerRouter()
	r.POST("/worker/next-task", WorkerNextTask)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/worker/next-task", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
	assert.Equal(t, "No pending tasks available", resp["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerNextTask_NotAWorker(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	expectVerifyWorker(mock, "worker-1", false)

	r := setupWorkerRouter()
	r.POST("/worker/next-task", WorkerNextTask)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/worker/next-task", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerNextTask_DBUnavailable(t *testing.T) {
	database.DB = nil

	r := setupWorkerRouter()
	r.POST("/worker/next-task", WorkerNextTask)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/worker/next-task", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ==================== WorkerGetTask Tests ====================

func TestWorkerGetTask_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	expectVerifyWorker(mock, "worker-1", true)

	mock.ExpectQuery("SELECT t.id, t.title").
		WithArgs("task-1", "worker-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "assigned_to", "status", "priority",
			"task_type_id", "params", "result", "retry_count",
			"created_by", "created_at", "updated_at", "started_at", "completed_at",
			"tt_id", "tt_name", "tt_label", "tt_sop",
		}).AddRow("task-1", "My Task", "Desc", "worker-1", "in_progress", "high",
			1, nil, nil, 0,
			"admin-1", now, now, &now, nil,
			1, "reddit_crawl", "Reddit Crawl", "SOP"))

	r := setupWorkerRouter()
	r.GET("/worker/tasks/:id", WorkerGetTask)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/worker/tasks/task-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerGetTask_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	expectVerifyWorker(mock, "worker-1", true)

	mock.ExpectQuery("SELECT t.id, t.title").
		WithArgs("nonexistent", "worker-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "assigned_to", "status", "priority",
			"task_type_id", "params", "result", "retry_count",
			"created_by", "created_at", "updated_at", "started_at", "completed_at",
			"tt_id", "tt_name", "tt_label", "tt_sop",
		}))

	r := setupWorkerRouter()
	r.GET("/worker/tasks/:id", WorkerGetTask)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/worker/tasks/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== WorkerUpdateTaskStatus Tests ====================

func TestWorkerUpdateTaskStatus_Completed(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	expectVerifyWorker(mock, "worker-1", true)

	mock.ExpectQuery("UPDATE worker_tasks SET status").
		WithArgs("task-1", "worker-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "assigned_to", "status", "priority",
			"task_type_id", "params", "result", "retry_count",
			"created_by", "created_at", "updated_at", "started_at", "completed_at",
		}).AddRow("task-1", "Task", "Desc", "worker-1", "completed", "high",
			1, nil, nil, 0,
			"admin-1", now, now, &now, &now))

	r := setupWorkerRouter()
	r.PUT("/worker/tasks/:id/status", WorkerUpdateTaskStatus)

	body, _ := json.Marshal(map[string]interface{}{
		"status": "completed",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/worker/tasks/task-1/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerUpdateTaskStatus_BackToPending(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	expectVerifyWorker(mock, "worker-1", true)

	mock.ExpectQuery("UPDATE worker_tasks SET status").
		WithArgs("task-1", "worker-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "assigned_to", "status", "priority",
			"task_type_id", "params", "result", "retry_count",
			"created_by", "created_at", "updated_at", "started_at", "completed_at",
		}).AddRow("task-1", "Task", "Desc", "worker-1", "pending", "high",
			1, nil, nil, 1,
			"admin-1", now, now, nil, nil))

	r := setupWorkerRouter()
	r.PUT("/worker/tasks/:id/status", WorkerUpdateTaskStatus)

	body, _ := json.Marshal(map[string]interface{}{
		"status": "pending",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/worker/tasks/task-1/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerUpdateTaskStatus_InvalidStatus(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	expectVerifyWorker(mock, "worker-1", true)

	r := setupWorkerRouter()
	r.PUT("/worker/tasks/:id/status", WorkerUpdateTaskStatus)

	body, _ := json.Marshal(map[string]interface{}{
		"status": "in_progress",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/worker/tasks/task-1/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "pending, completed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerUpdateTaskStatus_MissingStatus(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	expectVerifyWorker(mock, "worker-1", true)

	r := setupWorkerRouter()
	r.PUT("/worker/tasks/:id/status", WorkerUpdateTaskStatus)

	body, _ := json.Marshal(map[string]interface{}{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/worker/tasks/task-1/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== WorkerPostUpdate Tests ====================

func TestWorkerPostUpdate_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	expectVerifyWorker(mock, "worker-1", true)

	// Verify task ownership
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("task-1", "worker-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	// Insert update
	mock.ExpectQuery("INSERT INTO worker_task_updates").
		WithArgs("task-1", "Working on it", "worker-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "task_id", "content", "created_by", "created_at"}).
			AddRow("update-1", "task-1", "Working on it", "worker-1", now))

	r := setupWorkerRouter()
	r.POST("/worker/tasks/:id/updates", WorkerPostUpdate)

	body, _ := json.Marshal(map[string]string{"content": "Working on it"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/worker/tasks/task-1/updates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerPostUpdate_TaskNotAssigned(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	expectVerifyWorker(mock, "worker-1", true)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("task-other", "worker-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	r := setupWorkerRouter()
	r.POST("/worker/tasks/:id/updates", WorkerPostUpdate)

	body, _ := json.Marshal(map[string]string{"content": "Update"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/worker/tasks/task-other/updates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== WorkerPostResult Tests ====================

func TestWorkerPostResult_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	expectVerifyWorker(mock, "worker-1", true)

	// Check task status
	mock.ExpectQuery("SELECT status FROM worker_tasks").
		WithArgs("task-1", "worker-1").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("in_progress"))

	// Update result
	mock.ExpectQuery("UPDATE worker_tasks SET result").
		WithArgs(sqlmock.AnyArg(), "task-1", "worker-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "assigned_to", "status", "priority",
			"task_type_id", "params", "result", "retry_count",
			"created_by", "created_at", "updated_at", "started_at", "completed_at",
		}).AddRow("task-1", "Task", "Desc", "worker-1", "in_progress", "high",
			1, nil, []byte(`{"output":"done"}`), 0,
			"admin-1", now, now, &now, nil))

	r := setupWorkerRouter()
	r.POST("/worker/tasks/:id/result", WorkerPostResult)

	body, _ := json.Marshal(map[string]interface{}{
		"result": map[string]string{"output": "done"},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/worker/tasks/task-1/result", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerPostResult_NotInProgress(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	expectVerifyWorker(mock, "worker-1", true)

	mock.ExpectQuery("SELECT status FROM worker_tasks").
		WithArgs("task-1", "worker-1").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("pending"))

	r := setupWorkerRouter()
	r.POST("/worker/tasks/:id/result", WorkerPostResult)

	body, _ := json.Marshal(map[string]interface{}{
		"result": map[string]string{"output": "done"},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/worker/tasks/task-1/result", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "in_progress")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== WorkerPostTaskData Tests ====================

func TestWorkerPostTaskData_EmptyItems(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	expectVerifyWorker(mock, "worker-1", true)

	r := setupWorkerRouter()
	r.POST("/worker/tasks/:id/data", WorkerPostTaskData)

	body, _ := json.Marshal(map[string]interface{}{
		"data_type": "reddit_posts",
		"items":     []interface{}{},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/worker/tasks/task-1/data", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "non-empty")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerPostTaskData_TooManyItems(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	expectVerifyWorker(mock, "worker-1", true)

	// Create 501 items
	items := make([]map[string]interface{}, 501)
	for i := range items {
		items[i] = map[string]interface{}{
			"ticker": "AAPL",
			"data":   map[string]string{"text": "test"},
		}
	}

	r := setupWorkerRouter()
	r.POST("/worker/tasks/:id/data", WorkerPostTaskData)

	body, _ := json.Marshal(map[string]interface{}{
		"data_type": "reddit_posts",
		"items":     items,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/worker/tasks/task-1/data", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "500")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerPostTaskData_NotInProgress(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	expectVerifyWorker(mock, "worker-1", true)

	mock.ExpectQuery("SELECT status FROM worker_tasks").
		WithArgs("task-1", "worker-1").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("completed"))

	r := setupWorkerRouter()
	r.POST("/worker/tasks/:id/data", WorkerPostTaskData)

	body, _ := json.Marshal(map[string]interface{}{
		"data_type": "reddit_posts",
		"items": []map[string]interface{}{
			{"ticker": "AAPL", "data": map[string]string{"text": "test"}},
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/worker/tasks/task-1/data", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== WorkerHeartbeat Tests ====================

func TestWorkerHeartbeat_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	expectVerifyWorker(mock, "worker-1", true)

	r := setupWorkerRouter()
	r.POST("/worker/heartbeat", WorkerHeartbeat)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/worker/heartbeat", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	assert.Equal(t, "Heartbeat received", resp["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerHeartbeat_NotAWorker(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	expectVerifyWorker(mock, "worker-1", false)

	r := setupWorkerRouter()
	r.POST("/worker/heartbeat", WorkerHeartbeat)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/worker/heartbeat", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}
