package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// GetTickerVolume — param validation
// ---------------------------------------------------------------------------

func TestGetTickerVolume_EmptySymbol(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickers//volume", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "symbol", Value: ""}}

	GetTickerVolume(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Symbol is required", resp["error"])
}

// ---------------------------------------------------------------------------
// GetVolumeAggregates — param validation
// ---------------------------------------------------------------------------

func TestGetVolumeAggregates_EmptySymbol(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickers//volume/aggregates", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "symbol", Value: ""}}

	GetVolumeAggregates(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetVolumeAggregates_InvalidDays(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{"non-numeric", "days=abc"},
		{"zero", "days=0"},
		{"negative", "days=-5"},
		{"over 365", "days=400"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test?"+tt.query, nil)
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Params = gin.Params{{Key: "symbol", Value: "AAPL"}}

			GetVolumeAggregates(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var resp map[string]string
			json.Unmarshal(w.Body.Bytes(), &resp)
			assert.Contains(t, resp["error"], "Invalid days parameter")
		})
	}
}
