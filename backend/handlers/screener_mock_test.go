package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// GetScreenerStocks — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetScreenerStocks_Mock_NilDB(t *testing.T) {
	// Save and nil out database.DB
	mock, cleanup := setupMockDB(t)
	defer cleanup()
	_ = mock

	// Temporarily set DB to nil to test the nil guard
	origDB := getDatabaseDB()
	setDatabaseDBNil()
	defer restoreDatabaseDB(origDB)

	r := setupMockRouterNoAuth()
	r.GET("/screener/stocks", GetScreenerStocks)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/screener/stocks", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "Database not available")
}

func TestGetScreenerStocks_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// GetScreenerStocks calls database.GetScreenerStocks which runs a complex query
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/screener/stocks", GetScreenerStocks)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/screener/stocks", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to fetch stocks")
}
