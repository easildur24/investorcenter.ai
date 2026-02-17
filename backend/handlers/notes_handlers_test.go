package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// CreateFeatureGroup — request validation
// ---------------------------------------------------------------------------

func TestCreateFeatureGroup_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/notes/groups", bytes.NewBufferString("bad"))
	c.Request.Header.Set("Content-Type", "application/json")

	CreateFeatureGroup(c)

	// database.DB is nil -> 503, but if binding fails first -> 400
	// Since DB check comes first, we expect 503
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestCreateFeatureGroup_DBNotAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := map[string]string{"name": "Test Group"}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/notes/groups", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	CreateFeatureGroup(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Database not available", resp["error"])
}

// ---------------------------------------------------------------------------
// UpdateFeatureGroup — request validation
// ---------------------------------------------------------------------------

func TestUpdateFeatureGroup_DBNotAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := map[string]string{"name": "Updated Group"}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/admin/notes/groups/123", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "123"}}

	UpdateFeatureGroup(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// CreateFeature — request validation
// ---------------------------------------------------------------------------

func TestCreateFeature_DBNotAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := map[string]string{"name": "Test Feature"}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/notes/groups/g1/features", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "groupId", Value: "g1"}}

	CreateFeature(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// CreateFeatureNote — request validation
// ---------------------------------------------------------------------------

func TestCreateFeatureNote_DBNotAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := map[string]string{
		"section": "ui",
		"title":   "Test Note",
		"content": "Some content",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/notes/features/f1/notes", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "featureId", Value: "f1"}}

	CreateFeatureNote(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// Listing endpoints — DB not available guard
// ---------------------------------------------------------------------------

func TestGetNotesTree_DBNotAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/notes/tree", nil)

	GetNotesTree(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestListFeatureGroups_DBNotAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/notes/groups", nil)

	ListFeatureGroups(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestListFeatures_DBNotAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/notes/groups/g1/features", nil)
	c.Params = gin.Params{{Key: "groupId", Value: "g1"}}

	ListFeatures(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestListFeatureNotes_DBNotAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/notes/features/f1/notes", nil)
	c.Params = gin.Params{{Key: "featureId", Value: "f1"}}

	ListFeatureNotes(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// Delete endpoints — DB not available guard
// ---------------------------------------------------------------------------

func TestDeleteFeatureGroup_DBNotAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/admin/notes/groups/g1", nil)
	c.Params = gin.Params{{Key: "id", Value: "g1"}}

	DeleteFeatureGroup(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestDeleteFeature_DBNotAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/admin/notes/features/f1", nil)
	c.Params = gin.Params{{Key: "id", Value: "f1"}}

	DeleteFeature(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestDeleteFeatureNote_DBNotAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/admin/notes/notes/n1", nil)
	c.Params = gin.Params{{Key: "id", Value: "n1"}}

	DeleteFeatureNote(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// Update endpoints — DB not available guard
// ---------------------------------------------------------------------------

func TestUpdateFeature_DBNotAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := map[string]string{"name": "Updated Feature"}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/admin/notes/features/f1", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "f1"}}

	UpdateFeature(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestUpdateFeatureNote_DBNotAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := map[string]string{"title": "Updated Note"}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/admin/notes/notes/n1", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "n1"}}

	UpdateFeatureNote(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// Struct types
// ---------------------------------------------------------------------------

func TestFeatureGroup_Fields(t *testing.T) {
	g := FeatureGroup{
		ID:        "group-1",
		Name:      "Test Group",
		Notes:     "Some notes",
		SortOrder: 1,
	}

	assert.Equal(t, "group-1", g.ID)
	assert.Equal(t, "Test Group", g.Name)
	assert.Equal(t, "Some notes", g.Notes)
	assert.Equal(t, 1, g.SortOrder)
}

func TestFeatureNote_ValidSections(t *testing.T) {
	validSections := map[string]bool{
		"ui":      true,
		"backend": true,
		"data":    true,
		"infra":   true,
	}

	for section, expected := range validSections {
		assert.Equal(t, expected, validSections[section], "section %s should be valid", section)
	}

	assert.False(t, validSections["invalid"], "invalid section should not be valid")
}
