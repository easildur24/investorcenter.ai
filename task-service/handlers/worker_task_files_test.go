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

// ==================== WorkerRegisterTaskFile Tests ====================

func TestWorkerRegisterTaskFile_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	expectVerifyWorker(mock, "worker-1", true)

	// Expect task status check
	mock.ExpectQuery("SELECT status FROM worker_tasks WHERE id").
		WithArgs("task-1", "worker-1").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("in_progress"))

	now := time.Now()
	// Expect file insert
	mock.ExpectQuery("INSERT INTO worker_task_files").
		WithArgs("task-1", "report.txt", "worker-results/task-1/report.txt", "text/plain", int64(4096), "worker-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "task_id", "filename", "s3_key", "content_type", "size_bytes", "uploaded_by", "created_at"}).
			AddRow(1, "task-1", "report.txt", "worker-results/task-1/report.txt", "text/plain", 4096, "worker-1", now))

	r := setupWorkerRouter()
	r.POST("/worker/tasks/:id/files", WorkerRegisterTaskFile)

	body, _ := json.Marshal(map[string]interface{}{
		"filename":     "report.txt",
		"s3_key":       "worker-results/task-1/report.txt",
		"content_type": "text/plain",
		"size_bytes":   4096,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/worker/tasks/task-1/files", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "report.txt", data["filename"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerRegisterTaskFile_InvalidS3Key(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	expectVerifyWorker(mock, "worker-1", true)

	r := setupWorkerRouter()
	r.POST("/worker/tasks/:id/files", WorkerRegisterTaskFile)

	body, _ := json.Marshal(map[string]interface{}{
		"filename":     "report.txt",
		"s3_key":       "wrong-prefix/task-1/report.txt",
		"content_type": "text/plain",
		"size_bytes":   4096,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/worker/tasks/task-1/files", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "invalid s3_key")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerRegisterTaskFile_WrongTaskInKey(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	expectVerifyWorker(mock, "worker-1", true)

	r := setupWorkerRouter()
	r.POST("/worker/tasks/:id/files", WorkerRegisterTaskFile)

	// s3_key references a different task ID than the URL
	body, _ := json.Marshal(map[string]interface{}{
		"filename": "report.txt",
		"s3_key":   "worker-results/other-task/report.txt",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/worker/tasks/task-1/files", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerRegisterTaskFile_TaskNotInProgress(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	expectVerifyWorker(mock, "worker-1", true)

	// Task exists but is completed
	mock.ExpectQuery("SELECT status FROM worker_tasks WHERE id").
		WithArgs("task-1", "worker-1").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("completed"))

	r := setupWorkerRouter()
	r.POST("/worker/tasks/:id/files", WorkerRegisterTaskFile)

	body, _ := json.Marshal(map[string]interface{}{
		"filename": "report.txt",
		"s3_key":   "worker-results/task-1/report.txt",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/worker/tasks/task-1/files", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "in_progress")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerRegisterTaskFile_NotWorker(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	expectVerifyWorker(mock, "worker-1", false)

	r := setupWorkerRouter()
	r.POST("/worker/tasks/:id/files", WorkerRegisterTaskFile)

	body, _ := json.Marshal(map[string]interface{}{
		"filename": "report.txt",
		"s3_key":   "worker-results/task-1/report.txt",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/worker/tasks/task-1/files", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerRegisterTaskFile_MissingFields(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	expectVerifyWorker(mock, "worker-1", true)

	r := setupWorkerRouter()
	r.POST("/worker/tasks/:id/files", WorkerRegisterTaskFile)

	// Missing required filename
	body, _ := json.Marshal(map[string]interface{}{
		"s3_key": "worker-results/task-1/report.txt",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/worker/tasks/task-1/files", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerRegisterTaskFile_DefaultContentType(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	expectVerifyWorker(mock, "worker-1", true)

	mock.ExpectQuery("SELECT status FROM worker_tasks WHERE id").
		WithArgs("task-1", "worker-1").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("in_progress"))

	now := time.Now()
	// content_type should default to application/octet-stream
	mock.ExpectQuery("INSERT INTO worker_task_files").
		WithArgs("task-1", "data.bin", "worker-results/task-1/data.bin", "application/octet-stream", int64(0), "worker-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "task_id", "filename", "s3_key", "content_type", "size_bytes", "uploaded_by", "created_at"}).
			AddRow(1, "task-1", "data.bin", "worker-results/task-1/data.bin", "application/octet-stream", 0, "worker-1", now))

	r := setupWorkerRouter()
	r.POST("/worker/tasks/:id/files", WorkerRegisterTaskFile)

	body, _ := json.Marshal(map[string]interface{}{
		"filename": "data.bin",
		"s3_key":   "worker-results/task-1/data.bin",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/worker/tasks/task-1/files", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== AdminListTaskFiles Tests ====================

func TestAdminListTaskFiles_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT COUNT").
		WithArgs("task-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	mock.ExpectQuery("SELECT id, task_id, filename, s3_key, content_type, size_bytes, uploaded_by, created_at").
		WithArgs("task-1", 100, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "task_id", "filename", "s3_key", "content_type", "size_bytes", "uploaded_by", "created_at"}).
			AddRow(1, "task-1", "report.txt", "worker-results/task-1/report.txt", "text/plain", 4096, "worker-1", now).
			AddRow(2, "task-1", "data.csv", "worker-results/task-1/data.csv", "text/csv", 8192, "worker-1", now))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.GET("/admin/workers/tasks/:id/files", AdminListTaskFiles)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/workers/tasks/task-1/files", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]interface{})
	files := data["files"].([]interface{})
	assert.Len(t, files, 2)
	assert.Equal(t, float64(2), data["total"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminListTaskFiles_Empty(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT COUNT").
		WithArgs("task-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery("SELECT id, task_id, filename").
		WithArgs("task-1", 100, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "task_id", "filename", "s3_key", "content_type", "size_bytes", "uploaded_by", "created_at"}))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.GET("/admin/workers/tasks/:id/files", AdminListTaskFiles)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/workers/tasks/task-1/files", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	files := data["files"].([]interface{})
	assert.Len(t, files, 0)
	assert.Equal(t, float64(0), data["total"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminListTaskFiles_WithPagination(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT COUNT").
		WithArgs("task-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(50))

	mock.ExpectQuery("SELECT id, task_id, filename").
		WithArgs("task-1", 10, 20).
		WillReturnRows(sqlmock.NewRows([]string{"id", "task_id", "filename", "s3_key", "content_type", "size_bytes", "uploaded_by", "created_at"}).
			AddRow(21, "task-1", "file21.txt", "worker-results/task-1/file21.txt", "text/plain", 100, "worker-1", now))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.GET("/admin/workers/tasks/:id/files", AdminListTaskFiles)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/workers/tasks/task-1/files?limit=10&offset=20", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(10), data["limit"])
	assert.Equal(t, float64(20), data["offset"])
	assert.Equal(t, float64(50), data["total"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== AdminDownloadTaskFile Tests ====================

func TestAdminDownloadTaskFile_InvalidFileID(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.GET("/admin/workers/tasks/:id/files/:fileId/download", AdminDownloadTaskFile)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/workers/tasks/task-1/files/not-a-number/download", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Invalid file ID", resp["error"])
}

func TestAdminDownloadTaskFile_FileNotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT id, task_id, filename").
		WithArgs("task-1", int64(999)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "task_id", "filename", "s3_key", "content_type", "size_bytes", "uploaded_by", "created_at"}))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.GET("/admin/workers/tasks/:id/files/:fileId/download", AdminDownloadTaskFile)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/workers/tasks/task-1/files/999/download", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}
