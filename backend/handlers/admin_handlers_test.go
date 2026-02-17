package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// parseQueryInt — pure function (with gin context)
// ---------------------------------------------------------------------------

func TestParseQueryInt_Default(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := parseQueryInt(c, "limit", 50)
	assert.Equal(t, 50, result)
}

func TestParseQueryInt_ValidValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?limit=25", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := parseQueryInt(c, "limit", 50)
	assert.Equal(t, 25, result)
}

func TestParseQueryInt_InvalidValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?limit=abc", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := parseQueryInt(c, "limit", 50)
	assert.Equal(t, 50, result)
}

func TestParseQueryInt_NegativeValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?limit=-5", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := parseQueryInt(c, "limit", 50)
	assert.Equal(t, 50, result)
}

func TestParseQueryInt_LimitCappedAt200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?limit=500", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := parseQueryInt(c, "limit", 50)
	assert.Equal(t, 200, result)
}

func TestParseQueryInt_LimitExactly200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?limit=200", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := parseQueryInt(c, "limit", 50)
	assert.Equal(t, 200, result)
}

func TestParseQueryInt_NonLimitNotCapped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?offset=500", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// offset key is not capped at 200
	result := parseQueryInt(c, "offset", 0)
	assert.Equal(t, 500, result)
}

func TestParseQueryInt_Zero(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?offset=0", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := parseQueryInt(c, "offset", 10)
	assert.Equal(t, 0, result)
}

// ---------------------------------------------------------------------------
// NewAdminDataHandler — constructor
// ---------------------------------------------------------------------------

func TestNewAdminDataHandler_Nil(t *testing.T) {
	handler := NewAdminDataHandler(nil)
	assert.NotNil(t, handler)
	assert.Nil(t, handler.db)
}

// ---------------------------------------------------------------------------
// NewCronjobHandler — constructor
// ---------------------------------------------------------------------------

func TestNewCronjobHandler(t *testing.T) {
	handler := NewCronjobHandler(nil)
	assert.NotNil(t, handler)
}

// ---------------------------------------------------------------------------
// nullFloatToInterface — pure function
// ---------------------------------------------------------------------------

func TestNullFloatToInterface_Valid(t *testing.T) {
	nf := sql.NullFloat64{Float64: 42.5, Valid: true}
	result := nullFloatToInterface(nf)
	assert.Equal(t, 42.5, result)
}

func TestNullFloatToInterface_Null(t *testing.T) {
	nf := sql.NullFloat64{Valid: false}
	result := nullFloatToInterface(nf)
	assert.Nil(t, result)
}

func TestNullFloatToInterface_ValidZero(t *testing.T) {
	nf := sql.NullFloat64{Float64: 0.0, Valid: true}
	result := nullFloatToInterface(nf)
	assert.Equal(t, 0.0, result)
}

func TestNullFloatToInterface_ValidNegative(t *testing.T) {
	nf := sql.NullFloat64{Float64: -100.5, Valid: true}
	result := nullFloatToInterface(nf)
	assert.Equal(t, -100.5, result)
}

// ---------------------------------------------------------------------------
// nullIntToInterface — pure function
// ---------------------------------------------------------------------------

func TestNullIntToInterface_Valid(t *testing.T) {
	ni := sql.NullInt64{Int64: 42, Valid: true}
	result := nullIntToInterface(ni)
	assert.Equal(t, int64(42), result)
}

func TestNullIntToInterface_Null(t *testing.T) {
	ni := sql.NullInt64{Valid: false}
	result := nullIntToInterface(ni)
	assert.Nil(t, result)
}

func TestNullIntToInterface_ValidZero(t *testing.T) {
	ni := sql.NullInt64{Int64: 0, Valid: true}
	result := nullIntToInterface(ni)
	assert.Equal(t, int64(0), result)
}

func TestNullIntToInterface_ValidNegative(t *testing.T) {
	ni := sql.NullInt64{Int64: -100, Valid: true}
	result := nullIntToInterface(ni)
	assert.Equal(t, int64(-100), result)
}

// ---------------------------------------------------------------------------
// AdminDataHandler with nil DB — all methods should fail gracefully
// ---------------------------------------------------------------------------

func TestAdminDataHandler_GetStocks_NilDB(t *testing.T) {
	handler := NewAdminDataHandler(nil)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stocks", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// With nil DB, the handler will panic trying to call h.db.QueryRow
	// We recover from panic to verify the handler doesn't gracefully handle nil DB
	defer func() {
		r := recover()
		// Expect a panic since AdminDataHandler doesn't check for nil DB
		assert.NotNil(t, r, "Expected panic with nil DB")
	}()

	handler.GetStocks(c)
}

func TestAdminDataHandler_GetDatabaseStats_NilDB(t *testing.T) {
	handler := NewAdminDataHandler(nil)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	defer func() {
		r := recover()
		assert.NotNil(t, r, "Expected panic with nil DB")
	}()

	handler.GetDatabaseStats(c)
}
