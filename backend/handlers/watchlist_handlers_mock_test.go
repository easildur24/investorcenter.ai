package handlers

import (
	"bytes"
	"database/sql"
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
// ListWatchLists — DB-backed mock tests
// ---------------------------------------------------------------------------

func TestListWatchLists_NoAuth(t *testing.T) {
	r := setupMockRouterNoAuth()
	r.GET("/watchlists", ListWatchLists)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/watchlists", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestListWatchLists_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// GetWatchListsByUserID does a JOIN query
	mock.ExpectQuery("SELECT .+ FROM watch_lists wl LEFT JOIN watch_list_items").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "description", "is_default", "created_at", "updated_at", "item_count",
		}).AddRow("wl-1", "My Watchlist", nil, true, now, now, 3).
			AddRow("wl-2", "Tech Stocks", nil, false, now, now, 5))

	r := setupMockRouter("user-1")
	r.GET("/watchlists", ListWatchLists)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/watchlists", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	watchLists := resp["watch_lists"].([]interface{})
	assert.Len(t, watchLists, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListWatchLists_Mock_Empty(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .+ FROM watch_lists wl LEFT JOIN watch_list_items").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "description", "is_default", "created_at", "updated_at", "item_count",
		}))

	r := setupMockRouter("user-1")
	r.GET("/watchlists", ListWatchLists)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/watchlists", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	watchLists := resp["watch_lists"].([]interface{})
	assert.Len(t, watchLists, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListWatchLists_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .+ FROM watch_lists wl LEFT JOIN watch_list_items").
		WillReturnError(fmt.Errorf("connection refused"))

	r := setupMockRouter("user-1")
	r.GET("/watchlists", ListWatchLists)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/watchlists", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Failed to fetch watch lists", resp["error"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// CreateWatchList — DB-backed mock tests
// ---------------------------------------------------------------------------

func TestCreateWatchList_NoAuth(t *testing.T) {
	r := setupMockRouterNoAuth()
	r.POST("/watchlists", CreateWatchList)

	body, _ := json.Marshal(map[string]string{"name": "Test"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/watchlists", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateWatchList_Mock_InvalidJSON(t *testing.T) {
	r := setupMockRouter("user-1")
	r.POST("/watchlists", CreateWatchList)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/watchlists", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateWatchList_Mock_MissingName(t *testing.T) {
	r := setupMockRouter("user-1")
	r.POST("/watchlists", CreateWatchList)

	body, _ := json.Marshal(map[string]string{"name": ""})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/watchlists", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateWatchList_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// CreateWatchListAtomic INSERT ... SELECT ... WHERE count < $5
	mock.ExpectQuery("INSERT INTO watch_lists").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "display_order"}).
			AddRow("wl-new", now, now, 0))

	r := setupMockRouter("user-1")
	r.POST("/watchlists", CreateWatchList)

	body, _ := json.Marshal(map[string]string{
		"name": "My New List",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/watchlists", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "My New List", resp["name"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateWatchList_Mock_LimitReached(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// CreateWatchListAtomic: INSERT ... WHERE count < $5 returns 0 rows
	mock.ExpectQuery("INSERT INTO watch_lists").
		WillReturnError(sql.ErrNoRows)

	r := setupMockRouter("user-1")
	r.POST("/watchlists", CreateWatchList)

	body, _ := json.Marshal(map[string]string{
		"name": "Fourth List",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/watchlists", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "Watch list limit reached")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateWatchList_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("INSERT INTO watch_lists").
		WillReturnError(fmt.Errorf("some other db error"))

	r := setupMockRouter("user-1")
	r.POST("/watchlists", CreateWatchList)

	body, _ := json.Marshal(map[string]string{
		"name": "New List",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/watchlists", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// UpdateWatchList — DB-backed mock tests
// ---------------------------------------------------------------------------

func TestUpdateWatchList_NoAuth(t *testing.T) {
	r := setupMockRouterNoAuth()
	r.PUT("/watchlists/:id", UpdateWatchList)

	body, _ := json.Marshal(map[string]string{"name": "Updated"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/watchlists/wl-1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdateWatchList_Mock_InvalidJSON(t *testing.T) {
	r := setupMockRouter("user-1")
	r.PUT("/watchlists/:id", UpdateWatchList)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/watchlists/wl-1", bytes.NewBufferString("bad"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateWatchList_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// database.UpdateWatchList does UPDATE ... WHERE id = $3 AND user_id = $4
	mock.ExpectExec("UPDATE watch_lists SET name").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := setupMockRouter("user-1")
	r.PUT("/watchlists/:id", UpdateWatchList)

	body, _ := json.Marshal(map[string]string{
		"name": "Updated Name",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/watchlists/wl-1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Watch list updated successfully", resp["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateWatchList_Mock_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// 0 rows affected = not found
	mock.ExpectExec("UPDATE watch_lists SET name").
		WillReturnResult(sqlmock.NewResult(0, 0))

	r := setupMockRouter("user-1")
	r.PUT("/watchlists/:id", UpdateWatchList)

	body, _ := json.Marshal(map[string]string{
		"name": "Updated Name",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/watchlists/nonexistent", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateWatchList_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE watch_lists SET name").
		WillReturnError(fmt.Errorf("connection error"))

	r := setupMockRouter("user-1")
	r.PUT("/watchlists/:id", UpdateWatchList)

	body, _ := json.Marshal(map[string]string{
		"name": "Updated",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/watchlists/wl-1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// DeleteWatchList — DB-backed mock tests
// ---------------------------------------------------------------------------

func TestDeleteWatchList_NoAuth(t *testing.T) {
	r := setupMockRouterNoAuth()
	r.DELETE("/watchlists/:id", DeleteWatchList)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/watchlists/wl-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeleteWatchList_Mock_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// GetWatchListByID returns no rows
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnError(sql.ErrNoRows)

	r := setupMockRouter("user-1")
	r.DELETE("/watchlists/:id", DeleteWatchList)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/watchlists/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteWatchList_Mock_DefaultProtected(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// GetWatchListByID returns default watchlist
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "is_default", "display_order",
			"is_public", "public_slug", "created_at", "updated_at",
		}).AddRow("wl-default", "user-1", "Default", nil, true, 0, false, nil, now, now))

	r := setupMockRouter("user-1")
	r.DELETE("/watchlists/:id", DeleteWatchList)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/watchlists/wl-default", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Cannot delete the default watch list", resp["error"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteWatchList_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// GetWatchListByID returns non-default watchlist
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "is_default", "display_order",
			"is_public", "public_slug", "created_at", "updated_at",
		}).AddRow("wl-1", "user-1", "To Delete", nil, false, 1, false, nil, now, now))

	// DeleteWatchList
	mock.ExpectExec("DELETE FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := setupMockRouter("user-1")
	r.DELETE("/watchlists/:id", DeleteWatchList)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/watchlists/wl-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Watch list deleted successfully", resp["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteWatchList_Mock_DeleteFails(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "is_default", "display_order",
			"is_public", "public_slug", "created_at", "updated_at",
		}).AddRow("wl-1", "user-1", "To Delete", nil, false, 1, false, nil, now, now))

	mock.ExpectExec("DELETE FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnError(fmt.Errorf("constraint violation"))

	r := setupMockRouter("user-1")
	r.DELETE("/watchlists/:id", DeleteWatchList)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/watchlists/wl-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// GetUserTags — DB-backed mock tests
// ---------------------------------------------------------------------------

func TestGetUserTags_NoAuth(t *testing.T) {
	r := setupMockRouterNoAuth()
	r.GET("/tags", GetUserTags)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tags", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetUserTags_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// GetUserTags uses DB.Select
	mock.ExpectQuery("SELECT unnest").
		WillReturnRows(sqlmock.NewRows([]string{"name", "count"}).
			AddRow("tech", 5).AddRow("finance", 3))

	r := setupMockRouter("user-1")
	r.GET("/tags", GetUserTags)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tags", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	tags := resp["tags"].([]interface{})
	assert.Len(t, tags, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserTags_Mock_Empty(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT unnest").
		WillReturnRows(sqlmock.NewRows([]string{"name", "count"}))

	r := setupMockRouter("user-1")
	r.GET("/tags", GetUserTags)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tags", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserTags_Mock_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT unnest").
		WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouter("user-1")
	r.GET("/tags", GetUserTags)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tags", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// isWatchListLimitError — unit tests
// ---------------------------------------------------------------------------

func TestIsWatchListLimitError_Nil(t *testing.T) {
	assert.False(t, isWatchListLimitError(nil))
}

func TestIsWatchListLimitError_True(t *testing.T) {
	err := fmt.Errorf("watch list limit reached: maximum 3 allowed")
	assert.True(t, isWatchListLimitError(err))
}

func TestIsWatchListLimitError_False(t *testing.T) {
	err := fmt.Errorf("some other error")
	assert.False(t, isWatchListLimitError(err))
}

func TestIsWatchListLimitError_ShortString(t *testing.T) {
	err := fmt.Errorf("short")
	assert.False(t, isWatchListLimitError(err))
}

// ---------------------------------------------------------------------------
// AddTickerToWatchList — DB-backed mock tests
// ---------------------------------------------------------------------------

func TestAddTickerToWatchList_NoAuth(t *testing.T) {
	r := setupMockRouterNoAuth()
	r.POST("/watchlists/:id/items", AddTickerToWatchList)

	body, _ := json.Marshal(map[string]string{"symbol": "AAPL"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/watchlists/wl-1/items", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAddTickerToWatchList_Mock_OwnershipFails(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// ValidateWatchListOwnership -> GetWatchListByID returns not found
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnError(sql.ErrNoRows)

	r := setupMockRouter("user-1")
	r.POST("/watchlists/:id/items", AddTickerToWatchList)

	body, _ := json.Marshal(map[string]string{"symbol": "AAPL"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/watchlists/wl-other/items", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddTickerToWatchList_Mock_InvalidJSON(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// ValidateWatchListOwnership succeeds
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "is_default", "display_order",
			"is_public", "public_slug", "created_at", "updated_at",
		}).AddRow("wl-1", "user-1", "My List", nil, false, 0, false, nil, now, now))

	r := setupMockRouter("user-1")
	r.POST("/watchlists/:id/items", AddTickerToWatchList)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/watchlists/wl-1/items", bytes.NewBufferString("bad json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddTickerToWatchList_Mock_MissingSymbol(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "is_default", "display_order",
			"is_public", "public_slug", "created_at", "updated_at",
		}).AddRow("wl-1", "user-1", "My List", nil, false, 0, false, nil, now, now))

	r := setupMockRouter("user-1")
	r.POST("/watchlists/:id/items", AddTickerToWatchList)

	body, _ := json.Marshal(map[string]string{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/watchlists/wl-1/items", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddTickerToWatchList_Mock_TickerNotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// ValidateWatchListOwnership succeeds
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "is_default", "display_order",
			"is_public", "public_slug", "created_at", "updated_at",
		}).AddRow("wl-1", "user-1", "My List", nil, false, 0, false, nil, now, now))

	// AddTickerToWatchList: verify ticker exists - returns false
	mock.ExpectQuery("SELECT EXISTS").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	r := setupMockRouter("user-1")
	r.POST("/watchlists/:id/items", AddTickerToWatchList)

	body, _ := json.Marshal(map[string]string{"symbol": "ZZZZ"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/watchlists/wl-1/items", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddTickerToWatchList_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// ValidateWatchListOwnership
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "is_default", "display_order",
			"is_public", "public_slug", "created_at", "updated_at",
		}).AddRow("wl-1", "user-1", "My List", nil, false, 0, false, nil, now, now))

	// Verify ticker exists
	mock.ExpectQuery("SELECT EXISTS").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	// INSERT INTO watch_list_items
	mock.ExpectQuery("INSERT INTO watch_list_items").
		WillReturnRows(sqlmock.NewRows([]string{"id", "added_at", "display_order"}).
			AddRow("item-new", now, 0))

	r := setupMockRouter("user-1")
	r.POST("/watchlists/:id/items", AddTickerToWatchList)

	body, _ := json.Marshal(map[string]string{"symbol": "AAPL"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/watchlists/wl-1/items", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// RemoveTickerFromWatchList — DB-backed mock tests
// ---------------------------------------------------------------------------

func TestRemoveTickerFromWatchList_NoAuth(t *testing.T) {
	r := setupMockRouterNoAuth()
	r.DELETE("/watchlists/:id/items/:symbol", RemoveTickerFromWatchList)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/watchlists/wl-1/items/AAPL", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRemoveTickerFromWatchList_Mock_OwnershipFails(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnError(sql.ErrNoRows)

	r := setupMockRouter("user-1")
	r.DELETE("/watchlists/:id/items/:symbol", RemoveTickerFromWatchList)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/watchlists/wl-other/items/AAPL", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRemoveTickerFromWatchList_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// ValidateWatchListOwnership
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "is_default", "display_order",
			"is_public", "public_slug", "created_at", "updated_at",
		}).AddRow("wl-1", "user-1", "My List", nil, false, 0, false, nil, now, now))

	// RemoveTickerFromWatchList
	mock.ExpectExec("DELETE FROM watch_list_items WHERE watch_list_id = \\$1 AND symbol = \\$2").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := setupMockRouter("user-1")
	r.DELETE("/watchlists/:id/items/:symbol", RemoveTickerFromWatchList)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/watchlists/wl-1/items/AAPL", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Ticker removed successfully", resp["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRemoveTickerFromWatchList_Mock_NotFound(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "is_default", "display_order",
			"is_public", "public_slug", "created_at", "updated_at",
		}).AddRow("wl-1", "user-1", "My List", nil, false, 0, false, nil, now, now))

	// 0 rows affected = not found
	mock.ExpectExec("DELETE FROM watch_list_items WHERE watch_list_id = \\$1 AND symbol = \\$2").
		WillReturnResult(sqlmock.NewResult(0, 0))

	r := setupMockRouter("user-1")
	r.DELETE("/watchlists/:id/items/:symbol", RemoveTickerFromWatchList)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/watchlists/wl-1/items/ZZZZ", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// UpdateWatchListItem — DB-backed mock tests
// ---------------------------------------------------------------------------

func TestUpdateWatchListItem_NoAuth(t *testing.T) {
	r := setupMockRouterNoAuth()
	r.PUT("/watchlists/:id/items/:symbol", UpdateWatchListItem)

	body, _ := json.Marshal(map[string]string{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/watchlists/wl-1/items/AAPL", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdateWatchListItem_Mock_OwnershipFails(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnError(sql.ErrNoRows)

	r := setupMockRouter("user-1")
	r.PUT("/watchlists/:id/items/:symbol", UpdateWatchListItem)

	body, _ := json.Marshal(map[string]interface{}{
		"notes": "updated notes",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/watchlists/wl-other/items/AAPL", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateWatchListItem_Mock_TickerNotInList(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// ValidateWatchListOwnership
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "is_default", "display_order",
			"is_public", "public_slug", "created_at", "updated_at",
		}).AddRow("wl-1", "user-1", "My List", nil, false, 0, false, nil, now, now))

	// GetWatchListItems returns items but not ZZZZ
	mock.ExpectQuery("SELECT .+ FROM watch_list_items WHERE watch_list_id = \\$1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "watch_list_id", "symbol", "notes", "tags",
			"target_buy_price", "target_sell_price", "added_at", "display_order",
		}).AddRow("item-1", "wl-1", "AAPL", nil, "{}", nil, nil, now, 0))

	r := setupMockRouter("user-1")
	r.PUT("/watchlists/:id/items/:symbol", UpdateWatchListItem)

	body, _ := json.Marshal(map[string]interface{}{
		"notes": "updated notes",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/watchlists/wl-1/items/ZZZZ", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// BulkAddTickers — DB-backed mock tests
// ---------------------------------------------------------------------------

func TestBulkAddTickers_NoAuth(t *testing.T) {
	r := setupMockRouterNoAuth()
	r.POST("/watchlists/:id/bulk", BulkAddTickers)

	body, _ := json.Marshal(map[string]interface{}{
		"symbols": []string{"AAPL", "MSFT"},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/watchlists/wl-1/bulk", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBulkAddTickers_Mock_OwnershipFails(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnError(sql.ErrNoRows)

	r := setupMockRouter("user-1")
	r.POST("/watchlists/:id/bulk", BulkAddTickers)

	body, _ := json.Marshal(map[string]interface{}{
		"symbols": []string{"AAPL"},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/watchlists/wl-other/bulk", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBulkAddTickers_Mock_MissingSymbols(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "is_default", "display_order",
			"is_public", "public_slug", "created_at", "updated_at",
		}).AddRow("wl-1", "user-1", "My List", nil, false, 0, false, nil, now, now))

	r := setupMockRouter("user-1")
	r.POST("/watchlists/:id/bulk", BulkAddTickers)

	body, _ := json.Marshal(map[string]interface{}{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/watchlists/wl-1/bulk", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// ReorderWatchListItems — DB-backed mock tests
// ---------------------------------------------------------------------------

func TestReorderWatchListItems_NoAuth(t *testing.T) {
	r := setupMockRouterNoAuth()
	r.PUT("/watchlists/:id/reorder", ReorderWatchListItems)

	body, _ := json.Marshal(map[string]interface{}{
		"item_orders": []map[string]interface{}{
			{"item_id": "item-1", "display_order": 0},
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/watchlists/wl-1/reorder", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestReorderWatchListItems_Mock_OwnershipFails(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnError(sql.ErrNoRows)

	r := setupMockRouter("user-1")
	r.PUT("/watchlists/:id/reorder", ReorderWatchListItems)

	body, _ := json.Marshal(map[string]interface{}{
		"item_orders": []map[string]interface{}{
			{"item_id": "item-1", "display_order": 0},
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/watchlists/wl-other/reorder", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReorderWatchListItems_Mock_Success(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// ValidateWatchListOwnership
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "is_default", "display_order",
			"is_public", "public_slug", "created_at", "updated_at",
		}).AddRow("wl-1", "user-1", "My List", nil, false, 0, false, nil, now, now))

	// ValidateItemsBelongToWatchList: SELECT COUNT(*) FROM watch_list_items WHERE ...
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// UpdateItemDisplayOrder for each item
	mock.ExpectExec("UPDATE watch_list_items SET display_order").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE watch_list_items SET display_order").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := setupMockRouter("user-1")
	r.PUT("/watchlists/:id/reorder", ReorderWatchListItems)

	body, _ := json.Marshal(map[string]interface{}{
		"item_orders": []map[string]interface{}{
			{"item_id": "item-1", "display_order": 0},
			{"item_id": "item-2", "display_order": 1},
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/watchlists/wl-1/reorder", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Items reordered successfully", resp["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReorderWatchListItems_Mock_ItemsDontBelong(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	now := time.Now()

	// ValidateWatchListOwnership
	mock.ExpectQuery("SELECT .+ FROM watch_lists WHERE id = \\$1 AND user_id = \\$2").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "is_default", "display_order",
			"is_public", "public_slug", "created_at", "updated_at",
		}).AddRow("wl-1", "user-1", "My List", nil, false, 0, false, nil, now, now))

	// ValidateItemsBelongToWatchList: count doesn't match
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	r := setupMockRouter("user-1")
	r.PUT("/watchlists/:id/reorder", ReorderWatchListItems)

	body, _ := json.Marshal(map[string]interface{}{
		"item_orders": []map[string]interface{}{
			{"item_id": "item-1", "display_order": 0},
			{"item_id": "item-other", "display_order": 1},
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/watchlists/wl-1/reorder", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}
