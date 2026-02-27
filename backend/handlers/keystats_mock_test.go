package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// PostKeyStats — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestPostKeyStats_Mock_MissingSymbol(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupMockRouterNoAuth()
	r.POST("/keystats/:symbol", PostKeyStats)

	w := httptest.NewRecorder()
	// The route has :symbol param so even with empty it would match.
	// Test with a valid route but empty body first.
	req := httptest.NewRequest(http.MethodPost, "/keystats/AAPL", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostKeyStats_Mock_NilDB(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	origDB := getDatabaseDB()
	setDatabaseDBNil()
	defer restoreDatabaseDB(origDB)

	r := setupMockRouterNoAuth()
	r.POST("/keystats/:symbol", PostKeyStats)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/keystats/AAPL", bytes.NewBufferString(`{"pe":25}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestPostKeyStats_Mock_InvalidJSON(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupMockRouterNoAuth()
	r.POST("/keystats/:symbol", PostKeyStats)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/keystats/AAPL", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostKeyStats_Mock_EmptyData(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupMockRouterNoAuth()
	r.POST("/keystats/:symbol", PostKeyStats)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/keystats/AAPL", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Empty data")
}

func TestPostKeyStats_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("INSERT INTO keystats").
		WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.POST("/keystats/:symbol", PostKeyStats)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/keystats/AAPL", bytes.NewBufferString(`{"pe_ratio":25.5}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPostKeyStats_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("INSERT INTO keystats").
		WillReturnRows(sqlmock.NewRows([]string{"ticker", "key_stats", "created_at", "updated_at"}).
			AddRow("AAPL", []byte(`{"pe_ratio":25.5}`), now, now))

	r := setupMockRouterNoAuth()
	r.POST("/keystats/:symbol", PostKeyStats)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/keystats/AAPL", bytes.NewBufferString(`{"pe_ratio":25.5}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}

// ---------------------------------------------------------------------------
// GetKeyStats — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetKeyStats_Mock_NilDB(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	origDB := getDatabaseDB()
	setDatabaseDBNil()
	defer restoreDatabaseDB(origDB)

	r := setupMockRouterNoAuth()
	r.GET("/keystats/:symbol", GetKeyStats)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/keystats/AAPL", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestGetKeyStats_Mock_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .+ FROM keystats").
		WillReturnRows(sqlmock.NewRows([]string{"ticker", "key_stats", "created_at", "updated_at"}))

	r := setupMockRouterNoAuth()
	r.GET("/keystats/:symbol", GetKeyStats)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/keystats/AAPL", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetKeyStats_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .+ FROM keystats").
		WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/keystats/:symbol", GetKeyStats)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/keystats/AAPL", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetKeyStats_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT .+ FROM keystats").
		WillReturnRows(sqlmock.NewRows([]string{"ticker", "key_stats", "created_at", "updated_at"}).
			AddRow("AAPL", []byte(`{"pe_ratio":25.5,"market_cap":3000000000000}`), now, now))

	r := setupMockRouterNoAuth()
	r.GET("/keystats/:symbol", GetKeyStats)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/keystats/AAPL", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "pe_ratio")
}

// ---------------------------------------------------------------------------
// DeleteKeyStats — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestDeleteKeyStats_Mock_NilDB(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	origDB := getDatabaseDB()
	setDatabaseDBNil()
	defer restoreDatabaseDB(origDB)

	r := setupMockRouterNoAuth()
	r.DELETE("/keystats/:symbol", DeleteKeyStats)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/keystats/AAPL", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestDeleteKeyStats_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM keystats").
		WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.DELETE("/keystats/:symbol", DeleteKeyStats)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/keystats/AAPL", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeleteKeyStats_Mock_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM keystats").
		WillReturnResult(sqlmock.NewResult(0, 0))

	r := setupMockRouterNoAuth()
	r.DELETE("/keystats/:symbol", DeleteKeyStats)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/keystats/AAPL", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteKeyStats_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM keystats").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := setupMockRouterNoAuth()
	r.DELETE("/keystats/:symbol", DeleteKeyStats)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/keystats/AAPL", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}
