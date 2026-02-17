package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"investorcenter-api/database"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// PostKeyStats — input validation
// ---------------------------------------------------------------------------

func TestPostKeyStats_EmptySymbol(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	body := `{"pe": 25.5}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tickers//keystats", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "symbol", Value: ""}}

	PostKeyStats(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Ticker symbol is required", resp["error"])
}

func TestPostKeyStats_NoDB(t *testing.T) {
	original := database.DB
	database.DB = nil
	defer func() { database.DB = original }()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	body := `{"pe": 25.5}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tickers/AAPL/keystats", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "symbol", Value: "AAPL"}}

	PostKeyStats(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestPostKeyStats_InvalidJSON(t *testing.T) {
	// Need to skip if DB is nil (PostKeyStats checks DB before JSON parse)
	if database.DB == nil {
		t.Skip("database not available")
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tickers/AAPL/keystats", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "symbol", Value: "AAPL"}}

	PostKeyStats(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostKeyStats_EmptyBody(t *testing.T) {
	if database.DB == nil {
		t.Skip("database not available")
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tickers/AAPL/keystats", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "symbol", Value: "AAPL"}}

	PostKeyStats(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Empty data", resp["error"])
}

// ---------------------------------------------------------------------------
// GetKeyStats — input validation
// ---------------------------------------------------------------------------

func TestGetKeyStats_EmptySymbol(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickers//keystats", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "symbol", Value: ""}}

	GetKeyStats(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetKeyStats_NoDB(t *testing.T) {
	original := database.DB
	database.DB = nil
	defer func() { database.DB = original }()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickers/AAPL/keystats", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "symbol", Value: "AAPL"}}

	GetKeyStats(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// DeleteKeyStats — input validation
// ---------------------------------------------------------------------------

func TestDeleteKeyStats_EmptySymbol(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tickers//keystats", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "symbol", Value: ""}}

	DeleteKeyStats(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteKeyStats_NoDB(t *testing.T) {
	original := database.DB
	database.DB = nil
	defer func() { database.DB = original }()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tickers/AAPL/keystats", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "symbol", Value: "AAPL"}}

	DeleteKeyStats(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
