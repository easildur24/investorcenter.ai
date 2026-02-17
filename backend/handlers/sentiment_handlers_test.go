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
// GetTickerSentiment — param validation
// ---------------------------------------------------------------------------

func TestGetTickerSentiment_EmptyTicker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sentiment/", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: ""}}

	GetTickerSentiment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Ticker symbol is required", resp["error"])
}

// ---------------------------------------------------------------------------
// GetTickerSentimentHistory — param validation
// ---------------------------------------------------------------------------

func TestGetTickerSentimentHistory_EmptyTicker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sentiment//history", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: ""}}

	GetTickerSentimentHistory(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Ticker symbol is required", resp["error"])
}

// ---------------------------------------------------------------------------
// GetTickerPosts — param validation
// ---------------------------------------------------------------------------

func TestGetTickerPosts_EmptyTicker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sentiment//posts", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ticker", Value: ""}}

	GetTickerPosts(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Ticker symbol is required", resp["error"])
}
