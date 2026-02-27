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
)

// ---------------------------------------------------------------------------
// GetNotesTree — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetNotesTree_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// Select groups
	mock.ExpectQuery("SELECT \\* FROM feature_groups ORDER BY").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "notes", "sort_order", "created_at", "updated_at",
		}).AddRow("g1", "Group 1", "notes", 1, now, now))

	// Select features
	mock.ExpectQuery("SELECT \\* FROM features ORDER BY").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "group_id", "name", "notes", "sort_order", "created_at", "updated_at",
		}).AddRow("f1", "g1", "Feature 1", "notes", 1, now, now))

	// Select note counts
	mock.ExpectQuery("SELECT feature_id, section, COUNT").
		WillReturnRows(sqlmock.NewRows([]string{
			"feature_id", "section", "count",
		}).AddRow("f1", "ui", 3).AddRow("f1", "backend", 2))

	r := setupMockRouterNoAuth()
	r.GET("/tree", GetNotesTree)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tree", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	data := resp["data"].([]interface{})
	assert.Len(t, data, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNotesTree_GroupsQueryError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT \\* FROM feature_groups ORDER BY").
		WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/tree", GetNotesTree)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tree", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Failed to fetch groups", resp["error"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNotesTree_FeaturesQueryError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	mock.ExpectQuery("SELECT \\* FROM feature_groups ORDER BY").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "notes", "sort_order", "created_at", "updated_at",
		}).AddRow("g1", "Group 1", "notes", 1, now, now))

	mock.ExpectQuery("SELECT \\* FROM features ORDER BY").
		WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/tree", GetNotesTree)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tree", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNotesTree_NoteCountsQueryError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	mock.ExpectQuery("SELECT \\* FROM feature_groups ORDER BY").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "notes", "sort_order", "created_at", "updated_at",
		}).AddRow("g1", "Group 1", "notes", 1, now, now))

	mock.ExpectQuery("SELECT \\* FROM features ORDER BY").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "group_id", "name", "notes", "sort_order", "created_at", "updated_at",
		}))

	mock.ExpectQuery("SELECT feature_id, section, COUNT").
		WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/tree", GetNotesTree)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tree", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNotesTree_EmptyGroups(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT \\* FROM feature_groups ORDER BY").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "notes", "sort_order", "created_at", "updated_at",
		}))

	mock.ExpectQuery("SELECT \\* FROM features ORDER BY").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "group_id", "name", "notes", "sort_order", "created_at", "updated_at",
		}))

	mock.ExpectQuery("SELECT feature_id, section, COUNT").
		WillReturnRows(sqlmock.NewRows([]string{
			"feature_id", "section", "count",
		}))

	r := setupMockRouterNoAuth()
	r.GET("/tree", GetNotesTree)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tree", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	assert.Len(t, data, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// ListFeatureGroups — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestListFeatureGroups_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	mock.ExpectQuery("SELECT \\* FROM feature_groups ORDER BY").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "notes", "sort_order", "created_at", "updated_at",
		}).AddRow("g1", "Group 1", "notes1", 1, now, now).
			AddRow("g2", "Group 2", "notes2", 2, now, now))

	r := setupMockRouterNoAuth()
	r.GET("/groups", ListFeatureGroups)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/groups", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	data := resp["data"].([]interface{})
	assert.Len(t, data, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListFeatureGroups_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT \\* FROM feature_groups ORDER BY").
		WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/groups", ListFeatureGroups)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/groups", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// CreateFeatureGroup — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestCreateFeatureGroup_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	mock.ExpectQuery("INSERT INTO feature_groups").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "notes", "sort_order", "created_at", "updated_at",
		}).AddRow("g-new", "New Group", "some notes", 0, now, now))

	r := setupMockRouterNoAuth()
	r.POST("/groups", CreateFeatureGroup)

	body, _ := json.Marshal(map[string]string{
		"name":  "New Group",
		"notes": "some notes",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/groups", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateFeatureGroup_MissingName(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupMockRouterNoAuth()
	r.POST("/groups", CreateFeatureGroup)

	body, _ := json.Marshal(map[string]string{
		"notes": "notes without name",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/groups", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateFeatureGroup_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("INSERT INTO feature_groups").
		WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.POST("/groups", CreateFeatureGroup)

	body, _ := json.Marshal(map[string]string{
		"name": "New Group",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/groups", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// UpdateFeatureGroup — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestUpdateFeatureGroup_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	mock.ExpectQuery("UPDATE feature_groups SET").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "notes", "sort_order", "created_at", "updated_at",
		}).AddRow("g1", "Updated Group", "updated notes", 1, now, now))

	r := setupMockRouterNoAuth()
	r.PUT("/groups/:id", UpdateFeatureGroup)

	name := "Updated Group"
	body, _ := json.Marshal(map[string]*string{
		"name": &name,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/groups/g1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateFeatureGroup_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("UPDATE feature_groups SET").
		WillReturnError(sql.ErrNoRows)

	r := setupMockRouterNoAuth()
	r.PUT("/groups/:id", UpdateFeatureGroup)

	name := "Updated"
	body, _ := json.Marshal(map[string]*string{
		"name": &name,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/groups/nonexistent", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// DeleteFeatureGroup — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestDeleteFeatureGroup_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM feature_groups WHERE id = \\$1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := setupMockRouterNoAuth()
	r.DELETE("/groups/:id", DeleteFeatureGroup)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/groups/g1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteFeatureGroup_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM feature_groups WHERE id = \\$1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	r := setupMockRouterNoAuth()
	r.DELETE("/groups/:id", DeleteFeatureGroup)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/groups/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteFeatureGroup_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM feature_groups WHERE id = \\$1").
		WillReturnError(fmt.Errorf("constraint violation"))

	r := setupMockRouterNoAuth()
	r.DELETE("/groups/:id", DeleteFeatureGroup)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/groups/g1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// ListFeatures — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestListFeatures_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	mock.ExpectQuery("SELECT \\* FROM features WHERE group_id = \\$1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "group_id", "name", "notes", "sort_order", "created_at", "updated_at",
		}).AddRow("f1", "g1", "Feature 1", "", 1, now, now))

	r := setupMockRouterNoAuth()
	r.GET("/groups/:groupId/features", ListFeatures)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/groups/g1/features", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListFeatures_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT \\* FROM features WHERE group_id = \\$1").
		WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/groups/:groupId/features", ListFeatures)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/groups/g1/features", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// CreateFeature — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestCreateFeature_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	mock.ExpectQuery("INSERT INTO features").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "group_id", "name", "notes", "sort_order", "created_at", "updated_at",
		}).AddRow("f-new", "g1", "New Feature", "", 0, now, now))

	r := setupMockRouterNoAuth()
	r.POST("/groups/:groupId/features", CreateFeature)

	body, _ := json.Marshal(map[string]string{
		"name": "New Feature",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/groups/g1/features", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateFeature_MissingName(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupMockRouterNoAuth()
	r.POST("/groups/:groupId/features", CreateFeature)

	body, _ := json.Marshal(map[string]string{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/groups/g1/features", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateFeature_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("INSERT INTO features").
		WillReturnError(fmt.Errorf("foreign key violation"))

	r := setupMockRouterNoAuth()
	r.POST("/groups/:groupId/features", CreateFeature)

	body, _ := json.Marshal(map[string]string{
		"name": "New Feature",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/groups/g1/features", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// UpdateFeature — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestUpdateFeature_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	mock.ExpectQuery("UPDATE features SET").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "group_id", "name", "notes", "sort_order", "created_at", "updated_at",
		}).AddRow("f1", "g1", "Updated Feature", "", 1, now, now))

	r := setupMockRouterNoAuth()
	r.PUT("/features/:id", UpdateFeature)

	name := "Updated Feature"
	body, _ := json.Marshal(map[string]*string{
		"name": &name,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/features/f1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateFeature_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("UPDATE features SET").
		WillReturnError(sql.ErrNoRows)

	r := setupMockRouterNoAuth()
	r.PUT("/features/:id", UpdateFeature)

	name := "Updated"
	body, _ := json.Marshal(map[string]*string{"name": &name})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/features/nonexistent", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// DeleteFeature — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestDeleteFeature_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM features WHERE id = \\$1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := setupMockRouterNoAuth()
	r.DELETE("/features/:id", DeleteFeature)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/features/f1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteFeature_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM features WHERE id = \\$1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	r := setupMockRouterNoAuth()
	r.DELETE("/features/:id", DeleteFeature)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/features/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// ListFeatureNotes — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestListFeatureNotes_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	mock.ExpectQuery("SELECT \\* FROM feature_notes WHERE feature_id = \\$1 ORDER BY section").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "feature_id", "section", "title", "content", "sort_order", "created_at", "updated_at",
		}).AddRow("n1", "f1", "ui", "Note 1", "content", 1, now, now))

	r := setupMockRouterNoAuth()
	r.GET("/features/:featureId/notes", ListFeatureNotes)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/features/f1/notes", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListFeatureNotes_WithSectionFilter(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	mock.ExpectQuery("SELECT \\* FROM feature_notes WHERE feature_id = \\$1 AND section = \\$2").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "feature_id", "section", "title", "content", "sort_order", "created_at", "updated_at",
		}).AddRow("n1", "f1", "backend", "Note 1", "content", 1, now, now))

	r := setupMockRouterNoAuth()
	r.GET("/features/:featureId/notes", ListFeatureNotes)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/features/f1/notes?section=backend", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListFeatureNotes_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT \\* FROM feature_notes WHERE feature_id = \\$1").
		WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/features/:featureId/notes", ListFeatureNotes)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/features/f1/notes", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// CreateFeatureNote — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestCreateFeatureNote_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	mock.ExpectQuery("INSERT INTO feature_notes").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "feature_id", "section", "title", "content", "sort_order", "created_at", "updated_at",
		}).AddRow("n-new", "f1", "ui", "My Note", "some content", 0, now, now))

	r := setupMockRouterNoAuth()
	r.POST("/features/:featureId/notes", CreateFeatureNote)

	body, _ := json.Marshal(map[string]string{
		"section": "ui",
		"title":   "My Note",
		"content": "some content",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/features/f1/notes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateFeatureNote_InvalidSection(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupMockRouterNoAuth()
	r.POST("/features/:featureId/notes", CreateFeatureNote)

	body, _ := json.Marshal(map[string]string{
		"section": "invalid_section",
		"title":   "My Note",
		"content": "some content",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/features/f1/notes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "Invalid section")
}

func TestCreateFeatureNote_MissingSection(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupMockRouterNoAuth()
	r.POST("/features/:featureId/notes", CreateFeatureNote)

	body, _ := json.Marshal(map[string]string{
		"title":   "My Note",
		"content": "some content",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/features/f1/notes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateFeatureNote_DefaultTitle(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// When title is empty, handler sets it to "Untitled"
	mock.ExpectQuery("INSERT INTO feature_notes").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "feature_id", "section", "title", "content", "sort_order", "created_at", "updated_at",
		}).AddRow("n-new", "f1", "backend", "Untitled", "", 0, now, now))

	r := setupMockRouterNoAuth()
	r.POST("/features/:featureId/notes", CreateFeatureNote)

	body, _ := json.Marshal(map[string]string{
		"section": "backend",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/features/f1/notes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateFeatureNote_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("INSERT INTO feature_notes").
		WillReturnError(fmt.Errorf("foreign key violation"))

	r := setupMockRouterNoAuth()
	r.POST("/features/:featureId/notes", CreateFeatureNote)

	body, _ := json.Marshal(map[string]string{
		"section": "infra",
		"title":   "Note",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/features/f1/notes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// UpdateFeatureNote — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestUpdateFeatureNote_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	mock.ExpectQuery("UPDATE feature_notes SET").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "feature_id", "section", "title", "content", "sort_order", "created_at", "updated_at",
		}).AddRow("n1", "f1", "ui", "Updated Title", "updated content", 1, now, now))

	r := setupMockRouterNoAuth()
	r.PUT("/notes/:id", UpdateFeatureNote)

	title := "Updated Title"
	body, _ := json.Marshal(map[string]*string{"title": &title})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/notes/n1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateFeatureNote_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("UPDATE feature_notes SET").
		WillReturnError(sql.ErrNoRows)

	r := setupMockRouterNoAuth()
	r.PUT("/notes/:id", UpdateFeatureNote)

	title := "Updated"
	body, _ := json.Marshal(map[string]*string{"title": &title})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/notes/nonexistent", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// DeleteFeatureNote — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestDeleteFeatureNote_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM feature_notes WHERE id = \\$1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := setupMockRouterNoAuth()
	r.DELETE("/notes/:id", DeleteFeatureNote)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/notes/n1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteFeatureNote_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM feature_notes WHERE id = \\$1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	r := setupMockRouterNoAuth()
	r.DELETE("/notes/:id", DeleteFeatureNote)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/notes/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteFeatureNote_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM feature_notes WHERE id = \\$1").
		WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.DELETE("/notes/:id", DeleteFeatureNote)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/notes/n1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}
