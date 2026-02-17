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
// GetTickerRedditHistory — param validation
// ---------------------------------------------------------------------------

func TestGetTickerRedditHistory_EmptySymbol(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reddit/ticker//history", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "symbol", Value: ""}}

	GetTickerRedditHistory(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Symbol is required", resp["error"])
}
