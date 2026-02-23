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
		"id", "name", "skill_path", "param_schema", "created_at", "updated_at",
	}).
		AddRow(1, "reddit_crawl", "data-ingestion", nil, now, now).
		AddRow(2, "scrape_ycharts", "scrape-ycharts-keystats", nil, now, now)

	mock.ExpectQuery("SELECT id, name, skill_path, param_schema").WillReturnRows(rows)

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.GET("/task-types", ListTaskTypes)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/task-types", nil)
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
		"id", "name", "skill_path", "param_schema", "created_at", "updated_at",
	})

	mock.ExpectQuery("SELECT id, name, skill_path, param_schema").WillReturnRows(rows)

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.GET("/task-types", ListTaskTypes)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/task-types", nil)
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
	skillPath := "scrape-ycharts-keystats"
	mock.ExpectQuery("INSERT INTO task_types").
		WithArgs("new_type", &skillPath, nil).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "skill_path", "param_schema", "created_at", "updated_at",
		}).AddRow(3, "new_type", "scrape-ycharts-keystats", nil, now, now))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.POST("/task-types", CreateTaskType)

	body, _ := json.Marshal(map[string]interface{}{
		"name":       "new_type",
		"skill_path": "scrape-ycharts-keystats",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/task-types", bytes.NewBuffer(body))
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
	r.POST("/task-types", CreateTaskType)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "InvalidName",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/task-types", bytes.NewBuffer(body))
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
	r.POST("/task-types", CreateTaskType)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "has spaces",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/task-types", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	_ = mock
}

func TestCreateTaskType_InvalidName_SpecialChars(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.POST("/task-types", CreateTaskType)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "type-with-dashes",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/task-types", bytes.NewBuffer(body))
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
			"id", "name", "skill_path", "param_schema", "created_at", "updated_at",
		}).AddRow(4, "my_task_type", nil, nil, now, now))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.POST("/task-types", CreateTaskType)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "my_task_type",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/task-types", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTaskType_MissingName(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.POST("/task-types", CreateTaskType)

	body, _ := json.Marshal(map[string]interface{}{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/task-types", bytes.NewBuffer(body))
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
		WithArgs("1", stringPtr("scrape-ycharts-keystats"), nil).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "skill_path", "param_schema", "created_at", "updated_at",
		}).AddRow(1, "reddit_crawl", "scrape-ycharts-keystats", nil, now, now))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.PUT("/task-types/:id", UpdateTaskType)

	body, _ := json.Marshal(map[string]interface{}{
		"skill_path": "scrape-ycharts-keystats",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/task-types/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTaskType_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("UPDATE task_types SET").
		WithArgs("999", stringPtr("data-ingestion"), nil).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "skill_path", "param_schema", "created_at", "updated_at",
		}))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.PUT("/task-types/:id", UpdateTaskType)

	body, _ := json.Marshal(map[string]interface{}{
		"skill_path": "data-ingestion",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/task-types/999", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== DeleteTaskType Tests ====================

func TestDeleteTaskType_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM task_types").
		WithArgs("1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.DELETE("/task-types/:id", DeleteTaskType)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/task-types/1", nil)
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

	mock.ExpectExec("DELETE FROM task_types").
		WithArgs("999").
		WillReturnResult(sqlmock.NewResult(0, 0))

	r := setupRouterWithMockAuth("admin-1", "admin@test.com", true)
	r.DELETE("/task-types/:id", DeleteTaskType)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/task-types/999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== Validation Tests ====================

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

func TestValidSkillPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"with hyphens", "scrape-ycharts-keystats", true},
		{"simple", "dataingestion", true},
		{"with numbers", "scrape2", true},
		{"single char", "a", true},
		{"uppercase", "Scrape", false},
		{"with underscore", "scrape_data", false},
		{"with space", "scrape data", false},
		{"empty", "", false},
		{"starts with hyphen", "-scrape", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, validSkillPath.MatchString(tt.input))
		})
	}
}
