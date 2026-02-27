package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// GetHeatmapData — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetHeatmapData_Mock_NoAuth(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupMockRouterNoAuth()
	r.GET("/watchlists/:id/heatmap", GetHeatmapData)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/watchlists/wl-1/heatmap", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetHeatmapData_Mock_OwnershipFails(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// ValidateWatchListOwnership fails
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id").
		WillReturnError(fmt.Errorf("not found"))

	r := setupMockRouter("user-1")
	r.GET("/watchlists/:id/heatmap", GetHeatmapData)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/watchlists/wl-other/heatmap", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ---------------------------------------------------------------------------
// ListHeatmapConfigs — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestListHeatmapConfigs_Mock_NoAuth(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupMockRouterNoAuth()
	r.GET("/watchlists/:id/heatmap/configs", ListHeatmapConfigs)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/watchlists/wl-1/heatmap/configs", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestListHeatmapConfigs_Mock_OwnershipFails(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id").
		WillReturnError(fmt.Errorf("not found"))

	r := setupMockRouter("user-1")
	r.GET("/watchlists/:id/heatmap/configs", ListHeatmapConfigs)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/watchlists/wl-other/heatmap/configs", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestListHeatmapConfigs_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Ownership passes
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id").
		WillReturnError(fmt.Errorf("not found"))

	r := setupMockRouter("user-1")
	r.GET("/watchlists/:id/heatmap/configs", ListHeatmapConfigs)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/watchlists/wl-1/heatmap/configs", nil)
	r.ServeHTTP(w, req)

	// Ownership fails → 403
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ---------------------------------------------------------------------------
// CreateHeatmapConfig — validation tests
// ---------------------------------------------------------------------------

func TestCreateHeatmapConfig_Mock_NoAuth(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupMockRouterNoAuth()
	r.POST("/watchlists/:id/heatmap/configs", CreateHeatmapConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/watchlists/wl-1/heatmap/configs", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateHeatmapConfig_Mock_InvalidJSON(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupMockRouter("user-1")
	r.POST("/watchlists/:id/heatmap/configs", CreateHeatmapConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/watchlists/wl-1/heatmap/configs", bytes.NewBufferString("bad"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateHeatmapConfig_Mock_WatchListIDMismatch(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupMockRouter("user-1")
	r.POST("/watchlists/:id/heatmap/configs", CreateHeatmapConfig)

	body, _ := json.Marshal(map[string]interface{}{
		"watch_list_id": "wl-other",
		"name":          "My Config",
		"size_metric":   "market_cap",
		"color_metric":  "price_change_pct",
		"time_period":   "1D",
		"color_scheme":  "red_green",
		"label_display": "symbol",
		"layout_type":   "treemap",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/watchlists/wl-1/heatmap/configs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Watch list ID mismatch")
}

// ---------------------------------------------------------------------------
// UpdateHeatmapConfig — validation tests
// ---------------------------------------------------------------------------

func TestUpdateHeatmapConfig_Mock_NoAuth(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupMockRouterNoAuth()
	r.PUT("/watchlists/:id/heatmap/configs/:configId", UpdateHeatmapConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/watchlists/wl-1/heatmap/configs/cfg-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdateHeatmapConfig_Mock_InvalidJSON(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupMockRouter("user-1")
	r.PUT("/watchlists/:id/heatmap/configs/:configId", UpdateHeatmapConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/watchlists/wl-1/heatmap/configs/cfg-1", bytes.NewBufferString("bad"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateHeatmapConfig_Mock_OwnershipFails(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id").
		WillReturnError(fmt.Errorf("not found"))

	r := setupMockRouter("user-1")
	r.PUT("/watchlists/:id/heatmap/configs/:configId", UpdateHeatmapConfig)

	body, _ := json.Marshal(map[string]interface{}{
		"name":          "Updated Config",
		"size_metric":   "market_cap",
		"color_metric":  "price_change_pct",
		"time_period":   "1D",
		"color_scheme":  "red_green",
		"label_display": "symbol",
		"layout_type":   "treemap",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/watchlists/wl-1/heatmap/configs/cfg-1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ---------------------------------------------------------------------------
// DeleteHeatmapConfig — validation tests
// ---------------------------------------------------------------------------

func TestDeleteHeatmapConfig_Mock_NoAuth(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupMockRouterNoAuth()
	r.DELETE("/watchlists/:id/heatmap/configs/:configId", DeleteHeatmapConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/watchlists/wl-1/heatmap/configs/cfg-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeleteHeatmapConfig_Mock_OwnershipFails(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id").
		WillReturnError(fmt.Errorf("not found"))

	r := setupMockRouter("user-1")
	r.DELETE("/watchlists/:id/heatmap/configs/:configId", DeleteHeatmapConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/watchlists/wl-1/heatmap/configs/cfg-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
