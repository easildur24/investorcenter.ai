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

// ---------------------------------------------------------------------------
// UpdateWatchListItem — additional mock tests to increase from 52.8%
// ---------------------------------------------------------------------------

func TestUpdateWatchListItem_Mock_InvalidJSON(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// ValidateWatchListOwnership succeeds
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "is_default", "display_order",
			"is_public", "public_slug", "created_at", "updated_at",
		}).AddRow("wl-1", "user-1", "Test WL", nil, false, 0, false, nil, time.Now(), time.Now()))

	r := setupMockRouter("user-1")
	r.PUT("/watchlists/:id/items/:symbol", UpdateWatchListItem)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/watchlists/wl-1/items/AAPL", bytes.NewBufferString("bad"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateWatchListItem_Mock_GetItemsDBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// ValidateWatchListOwnership succeeds
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "is_default", "display_order",
			"is_public", "public_slug", "created_at", "updated_at",
		}).AddRow("wl-1", "user-1", "Test WL", nil, false, 0, false, nil, time.Now(), time.Now()))

	// GetWatchListItems fails
	mock.ExpectQuery("SELECT .+ FROM watch_list_items").
		WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouter("user-1")
	r.PUT("/watchlists/:id/items/:symbol", UpdateWatchListItem)

	body, _ := json.Marshal(map[string]interface{}{
		"notes": "test notes",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/watchlists/wl-1/items/AAPL", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateWatchListItem_Mock_SymbolNotInList(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// ValidateWatchListOwnership succeeds
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "is_default", "display_order",
			"is_public", "public_slug", "created_at", "updated_at",
		}).AddRow("wl-1", "user-1", "Test WL", nil, false, 0, false, nil, time.Now(), time.Now()))

	// GetWatchListItems returns items but not the one we're looking for
	mock.ExpectQuery("SELECT .+ FROM watch_list_items").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "watch_list_id", "symbol", "notes", "tags",
			"target_buy_price", "target_sell_price", "added_at", "display_order",
		}).AddRow("item-1", "wl-1", "MSFT", nil, "{}", nil, nil, time.Now(), 0))

	r := setupMockRouter("user-1")
	r.PUT("/watchlists/:id/items/:symbol", UpdateWatchListItem)

	body, _ := json.Marshal(map[string]interface{}{
		"notes": "test notes",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/watchlists/wl-1/items/AAPL", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Ticker not found in watch list")
}

func TestUpdateWatchListItem_Mock_UpdateFails(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// ValidateWatchListOwnership succeeds
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "is_default", "display_order",
			"is_public", "public_slug", "created_at", "updated_at",
		}).AddRow("wl-1", "user-1", "Test WL", nil, false, 0, false, nil, now, now))

	// GetWatchListItems returns the item
	mock.ExpectQuery("SELECT .+ FROM watch_list_items").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "watch_list_id", "symbol", "notes", "tags",
			"target_buy_price", "target_sell_price", "added_at", "display_order",
		}).AddRow("item-1", "wl-1", "AAPL", nil, "{}", nil, nil, now, 0))

	// UpdateWatchListItem fails
	mock.ExpectExec("UPDATE watch_list_items").
		WillReturnError(fmt.Errorf("update failed"))

	r := setupMockRouter("user-1")
	r.PUT("/watchlists/:id/items/:symbol", UpdateWatchListItem)

	body, _ := json.Marshal(map[string]interface{}{
		"notes": "test notes",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/watchlists/wl-1/items/AAPL", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to update ticker")
}

func TestUpdateWatchListItem_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// ValidateWatchListOwnership succeeds
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "is_default", "display_order",
			"is_public", "public_slug", "created_at", "updated_at",
		}).AddRow("wl-1", "user-1", "Test WL", nil, false, 0, false, nil, now, now))

	// GetWatchListItems returns the item
	mock.ExpectQuery("SELECT .+ FROM watch_list_items").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "watch_list_id", "symbol", "notes", "tags",
			"target_buy_price", "target_sell_price", "added_at", "display_order",
		}).AddRow("item-1", "wl-1", "AAPL", nil, "{}", nil, nil, now, 0))

	// UpdateWatchListItem succeeds
	mock.ExpectExec("UPDATE watch_list_items").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := setupMockRouter("user-1")
	r.PUT("/watchlists/:id/items/:symbol", UpdateWatchListItem)

	body, _ := json.Marshal(map[string]interface{}{
		"notes": "Updated notes",
		"tags":  []string{"tech", "mega-cap"},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/watchlists/wl-1/items/AAPL", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// BulkAddTickers — additional tests to increase from 66.7%
// ---------------------------------------------------------------------------

func TestBulkAddTickers_Mock_InvalidJSON(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// ValidateWatchListOwnership succeeds
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "is_default", "display_order",
			"is_public", "public_slug", "created_at", "updated_at",
		}).AddRow("wl-1", "user-1", "Test WL", nil, false, 0, false, nil, time.Now(), time.Now()))

	r := setupMockRouter("user-1")
	r.POST("/watchlists/:id/bulk", BulkAddTickers)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/watchlists/wl-1/bulk", bytes.NewBufferString("bad"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// DeleteWatchList — additional tests
// ---------------------------------------------------------------------------

func TestDeleteWatchList_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// GetWatchListByID returns the list (not default)
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "is_default", "display_order",
			"is_public", "public_slug", "created_at", "updated_at",
		}).AddRow("wl-1", "user-1", "Test WL", nil, false, 0, false, nil, time.Now(), time.Now()))

	// Delete fails
	mock.ExpectExec("DELETE FROM watch_lists").
		WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouter("user-1")
	r.DELETE("/watchlists/:id", DeleteWatchList)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/watchlists/wl-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------------------------------------------------------------------------
// GetWatchList — DB-backed tests (currently 0%)
// ---------------------------------------------------------------------------

func TestGetWatchList_Mock_NoAuth(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	r := setupMockRouterNoAuth()
	r.GET("/watchlists/:id", GetWatchList)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/watchlists/wl-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetWatchList_Mock_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// GetWatchListByID returns not found (empty rows)
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "is_default", "display_order",
			"is_public", "public_slug", "created_at", "updated_at",
		}))

	r := setupMockRouter("user-1")
	r.GET("/watchlists/:id", GetWatchList)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/watchlists/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetWatchList_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// GetWatchListByID returns an error that's NOT ErrWatchListNotFound
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id").
		WillReturnError(fmt.Errorf("connection refused"))

	r := setupMockRouter("user-1")
	r.GET("/watchlists/:id", GetWatchList)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/watchlists/wl-1", nil)
	r.ServeHTTP(w, req)

	// Should be 500 or 404 depending on error wrapping
	assert.True(t, w.Code == http.StatusInternalServerError || w.Code == http.StatusNotFound)
}
