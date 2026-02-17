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
// CreateHeatmapConfig — request validation
// ---------------------------------------------------------------------------

func TestCreateHeatmapConfig_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/api/v1/watchlists/:id/heatmap/configs", CreateHeatmapConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/watchlists/wl-1/heatmap/configs", bytes.NewBufferString("bad"))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	// Without auth context, will return 401 first
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateHeatmapConfig_MissingRequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set up router with a middleware that sets user_id
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Next()
	})
	router.POST("/api/v1/watchlists/:id/heatmap/configs", CreateHeatmapConfig)

	body := map[string]interface{}{} // All required fields missing
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/watchlists/wl-1/heatmap/configs", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateHeatmapConfig_InvalidSizeMetric(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Next()
	})
	router.POST("/api/v1/watchlists/:id/heatmap/configs", CreateHeatmapConfig)

	body := map[string]interface{}{
		"watch_list_id": "wl-1",
		"name":          "My Config",
		"size_metric":   "invalid_metric",
		"color_metric":  "price_change_pct",
		"time_period":   "1D",
		"color_scheme":  "red_green",
		"label_display": "symbol",
		"layout_type":   "treemap",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/watchlists/wl-1/heatmap/configs", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateHeatmapConfig_InvalidColorMetric(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Next()
	})
	router.POST("/api/v1/watchlists/:id/heatmap/configs", CreateHeatmapConfig)

	body := map[string]interface{}{
		"watch_list_id": "wl-1",
		"name":          "My Config",
		"size_metric":   "market_cap",
		"color_metric":  "invalid_metric",
		"time_period":   "1D",
		"color_scheme":  "red_green",
		"label_display": "symbol",
		"layout_type":   "treemap",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/watchlists/wl-1/heatmap/configs", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateHeatmapConfig_InvalidTimePeriod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Next()
	})
	router.POST("/api/v1/watchlists/:id/heatmap/configs", CreateHeatmapConfig)

	body := map[string]interface{}{
		"watch_list_id": "wl-1",
		"name":          "My Config",
		"size_metric":   "market_cap",
		"color_metric":  "price_change_pct",
		"time_period":   "10Y", // not valid
		"color_scheme":  "red_green",
		"label_display": "symbol",
		"layout_type":   "treemap",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/watchlists/wl-1/heatmap/configs", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// UpdateHeatmapConfig — request validation
// ---------------------------------------------------------------------------

func TestUpdateHeatmapConfig_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Next()
	})
	router.PUT("/api/v1/watchlists/:id/heatmap/configs/:configId", UpdateHeatmapConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/watchlists/wl-1/heatmap/configs/cfg-1", bytes.NewBufferString("bad"))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// GetHeatmapData — no auth returns 401
// ---------------------------------------------------------------------------

func TestGetHeatmapData_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/api/v1/watchlists/:id/heatmap", GetHeatmapData)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/watchlists/wl-1/heatmap", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestListHeatmapConfigs_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/api/v1/watchlists/:id/heatmap/configs", ListHeatmapConfigs)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/watchlists/wl-1/heatmap/configs", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeleteHeatmapConfig_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.DELETE("/api/v1/watchlists/:id/heatmap/configs/:configId", DeleteHeatmapConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/watchlists/wl-1/heatmap/configs/cfg-1", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
