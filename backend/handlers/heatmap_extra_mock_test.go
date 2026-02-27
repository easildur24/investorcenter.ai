package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// heatmapConfigColumns returns the column names for the heatmap_configs table.
func heatmapConfigColumns() []string {
	return []string{
		"id", "user_id", "watch_list_id", "name",
		"size_metric", "color_metric", "time_period",
		"color_scheme", "label_display", "layout_type",
		"filters_json", "color_gradient_json",
		"is_default", "created_at", "updated_at",
	}
}

// watchListOwnershipColumns returns the columns for watch_lists ownership validation.
func watchListOwnershipColumns() []string {
	return []string{
		"id", "user_id", "name", "description", "is_default", "display_order",
		"is_public", "public_slug", "created_at", "updated_at",
	}
}

// expectOwnershipPass sets up mock for ValidateWatchListOwnership to succeed.
func expectOwnershipPass(mock sqlmock.Sqlmock, watchListID, userID string) {
	now := time.Now()
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id").
		WillReturnRows(sqlmock.NewRows(watchListOwnershipColumns()).
			AddRow(watchListID, userID, "Test WL", nil, false, 0, false, nil, now, now))
}

// ---------------------------------------------------------------------------
// ListHeatmapConfigs — success path tests
// ---------------------------------------------------------------------------

func TestListHeatmapConfigs_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// Ownership passes
	expectOwnershipPass(mock, "wl-1", "user-1")

	// GetHeatmapConfigsByWatchListID returns configs
	rows := sqlmock.NewRows(heatmapConfigColumns()).
		AddRow("cfg-1", "user-1", "wl-1", "My Config",
			"market_cap", "price_change_pct", "1D",
			"red_green", "symbol", "treemap",
			[]byte(`{}`), []byte(`{}`),
			true, now, now)
	mock.ExpectQuery("SELECT .+ FROM heatmap_configs").WillReturnRows(rows)

	r := setupMockRouter("user-1")
	r.GET("/watchlists/:id/heatmap/configs", ListHeatmapConfigs)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/watchlists/wl-1/heatmap/configs", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "My Config")
}

// ---------------------------------------------------------------------------
// CreateHeatmapConfig — success path and additional tests
// ---------------------------------------------------------------------------

func TestCreateHeatmapConfig_Mock_OwnershipFails(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Ownership check fails
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id").
		WillReturnError(fmt.Errorf("not found"))

	r := setupMockRouter("user-1")
	r.POST("/watchlists/:id/heatmap/configs", CreateHeatmapConfig)

	body, _ := json.Marshal(map[string]interface{}{
		"watch_list_id": "wl-1",
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

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCreateHeatmapConfig_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Ownership passes
	expectOwnershipPass(mock, "wl-1", "user-1")

	// CreateHeatmapConfig INSERT RETURNING fails
	mock.ExpectQuery("INSERT INTO heatmap_configs").
		WillReturnError(fmt.Errorf("insert failed"))

	r := setupMockRouter("user-1")
	r.POST("/watchlists/:id/heatmap/configs", CreateHeatmapConfig)

	body, _ := json.Marshal(map[string]interface{}{
		"watch_list_id": "wl-1",
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

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to create heatmap config")
}

func TestCreateHeatmapConfig_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Ownership passes
	expectOwnershipPass(mock, "wl-1", "user-1")

	// CreateHeatmapConfig INSERT RETURNING succeeds
	now := time.Now()
	mock.ExpectQuery("INSERT INTO heatmap_configs").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow("cfg-1", now, now))

	r := setupMockRouter("user-1")
	r.POST("/watchlists/:id/heatmap/configs", CreateHeatmapConfig)

	body, _ := json.Marshal(map[string]interface{}{
		"watch_list_id": "wl-1",
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

	assert.Equal(t, http.StatusCreated, w.Code)
}

// ---------------------------------------------------------------------------
// UpdateHeatmapConfig — success path and additional tests
// ---------------------------------------------------------------------------

func TestUpdateHeatmapConfig_Mock_ConfigNotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Ownership passes
	expectOwnershipPass(mock, "wl-1", "user-1")

	// GetHeatmapConfigByID returns not found
	mock.ExpectQuery("SELECT .+ FROM heatmap_configs").
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

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateHeatmapConfig_Mock_WatchListMismatch(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// Ownership passes for wl-1
	expectOwnershipPass(mock, "wl-1", "user-1")

	// GetHeatmapConfigByID returns a config that belongs to wl-other
	mock.ExpectQuery("SELECT .+ FROM heatmap_configs").
		WillReturnRows(sqlmock.NewRows(heatmapConfigColumns()).
			AddRow("cfg-1", "user-1", "wl-other", "Config",
				"market_cap", "price_change_pct", "1D",
				"red_green", "symbol", "treemap",
				[]byte(`{}`), []byte(`{}`),
				false, now, now))

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

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Config does not belong to this watch list")
}

func TestUpdateHeatmapConfig_Mock_UpdateDBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// Ownership passes
	expectOwnershipPass(mock, "wl-1", "user-1")

	// GetHeatmapConfigByID succeeds
	mock.ExpectQuery("SELECT .+ FROM heatmap_configs").
		WillReturnRows(sqlmock.NewRows(heatmapConfigColumns()).
			AddRow("cfg-1", "user-1", "wl-1", "Config",
				"market_cap", "price_change_pct", "1D",
				"red_green", "symbol", "treemap",
				[]byte(`{}`), []byte(`{}`),
				false, now, now))

	// UpdateHeatmapConfig fails
	mock.ExpectExec("UPDATE heatmap_configs").
		WillReturnError(fmt.Errorf("update failed"))

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

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to update heatmap config")
}

func TestUpdateHeatmapConfig_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// Ownership passes
	expectOwnershipPass(mock, "wl-1", "user-1")

	// GetHeatmapConfigByID succeeds
	mock.ExpectQuery("SELECT .+ FROM heatmap_configs").
		WillReturnRows(sqlmock.NewRows(heatmapConfigColumns()).
			AddRow("cfg-1", "user-1", "wl-1", "Config",
				"market_cap", "price_change_pct", "1D",
				"red_green", "symbol", "treemap",
				[]byte(`{}`), []byte(`{}`),
				false, now, now))

	// UpdateHeatmapConfig succeeds
	mock.ExpectExec("UPDATE heatmap_configs").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := setupMockRouter("user-1")
	r.PUT("/watchlists/:id/heatmap/configs/:configId", UpdateHeatmapConfig)

	body, _ := json.Marshal(map[string]interface{}{
		"name":          "Updated Config",
		"size_metric":   "volume",
		"color_metric":  "volume_change_pct",
		"time_period":   "1W",
		"color_scheme":  "blue_red",
		"label_display": "full",
		"layout_type":   "grid",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/watchlists/wl-1/heatmap/configs/cfg-1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// DeleteHeatmapConfig — success path and additional tests
// ---------------------------------------------------------------------------

func TestDeleteHeatmapConfig_Mock_ConfigNotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Ownership passes
	expectOwnershipPass(mock, "wl-1", "user-1")

	// GetHeatmapConfigByID returns not found
	mock.ExpectQuery("SELECT .+ FROM heatmap_configs").
		WillReturnError(fmt.Errorf("not found"))

	r := setupMockRouter("user-1")
	r.DELETE("/watchlists/:id/heatmap/configs/:configId", DeleteHeatmapConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/watchlists/wl-1/heatmap/configs/cfg-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteHeatmapConfig_Mock_WatchListMismatch(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// Ownership passes
	expectOwnershipPass(mock, "wl-1", "user-1")

	// GetHeatmapConfigByID returns a config that belongs to wl-other
	mock.ExpectQuery("SELECT .+ FROM heatmap_configs").
		WillReturnRows(sqlmock.NewRows(heatmapConfigColumns()).
			AddRow("cfg-1", "user-1", "wl-other", "Config",
				"market_cap", "price_change_pct", "1D",
				"red_green", "symbol", "treemap",
				[]byte(`{}`), []byte(`{}`),
				false, now, now))

	r := setupMockRouter("user-1")
	r.DELETE("/watchlists/:id/heatmap/configs/:configId", DeleteHeatmapConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/watchlists/wl-1/heatmap/configs/cfg-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Config does not belong to this watch list")
}

func TestDeleteHeatmapConfig_Mock_DeleteDBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// Ownership passes
	expectOwnershipPass(mock, "wl-1", "user-1")

	// GetHeatmapConfigByID succeeds
	mock.ExpectQuery("SELECT .+ FROM heatmap_configs").
		WillReturnRows(sqlmock.NewRows(heatmapConfigColumns()).
			AddRow("cfg-1", "user-1", "wl-1", "Config",
				"market_cap", "price_change_pct", "1D",
				"red_green", "symbol", "treemap",
				[]byte(`{}`), []byte(`{}`),
				false, now, now))

	// Delete fails
	mock.ExpectExec("DELETE FROM heatmap_configs").
		WillReturnError(fmt.Errorf("delete failed"))

	r := setupMockRouter("user-1")
	r.DELETE("/watchlists/:id/heatmap/configs/:configId", DeleteHeatmapConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/watchlists/wl-1/heatmap/configs/cfg-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeleteHeatmapConfig_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// Ownership passes
	expectOwnershipPass(mock, "wl-1", "user-1")

	// GetHeatmapConfigByID succeeds
	mock.ExpectQuery("SELECT .+ FROM heatmap_configs").
		WillReturnRows(sqlmock.NewRows(heatmapConfigColumns()).
			AddRow("cfg-1", "user-1", "wl-1", "Config",
				"market_cap", "price_change_pct", "1D",
				"red_green", "symbol", "treemap",
				[]byte(`{}`), []byte(`{}`),
				false, now, now))

	// Delete succeeds
	mock.ExpectExec("DELETE FROM heatmap_configs").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := setupMockRouter("user-1")
	r.DELETE("/watchlists/:id/heatmap/configs/:configId", DeleteHeatmapConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/watchlists/wl-1/heatmap/configs/cfg-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "deleted successfully")
}

// ---------------------------------------------------------------------------
// GetHeatmapData — success path tests
// ---------------------------------------------------------------------------

func TestGetHeatmapData_Mock_GenerateError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Ownership passes
	expectOwnershipPass(mock, "wl-1", "user-1")

	// GenerateHeatmapData queries watch list items, then errors
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("generation error"))

	r := setupMockRouter("user-1")
	r.GET("/watchlists/:id/heatmap", GetHeatmapData)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/watchlists/wl-1/heatmap", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetHeatmapData_Mock_WithOverrides(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Ownership passes
	expectOwnershipPass(mock, "wl-1", "user-1")

	// GenerateHeatmapData with overrides
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("error"))

	r := setupMockRouter("user-1")
	r.GET("/watchlists/:id/heatmap", GetHeatmapData)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/watchlists/wl-1/heatmap?size_metric=revenue&color_metric=volume_change_pct&time_period=1W", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
