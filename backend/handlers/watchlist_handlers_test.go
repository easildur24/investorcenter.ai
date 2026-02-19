package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"investorcenter-api/database"
	"investorcenter-api/models"
)

// Setup test router with auth middleware mock
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Mock auth middleware for testing
	router.Use(func(c *gin.Context) {
		// Set test user ID in context
		c.Set("user_id", "test-user-id-123")
		c.Next()
	})

	return router
}

// Helper to create a test user in database
func createTestUser(t *testing.T) string {
	userID := "test-user-id-123"

	// Skip if database not available (CI environment)
	if database.DB == nil {
		t.Skip("Skipping test: database connection not available")
	}

	// Check if user already exists
	var exists bool
	err := database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
	if err != nil {
		t.Fatalf("Failed to check user existence: %v", err)
	}

	if !exists {
		_, err = database.DB.Exec(`
			INSERT INTO users (id, email, password_hash, full_name)
			VALUES ($1, 'test@watchlist.com', 'test_hash', 'Test User')
		`, userID)
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}
	}

	return userID
}

// Helper to clean up test data
func cleanupTestData(t *testing.T, userID string) {
	// Delete all watch lists (cascade will delete items)
	_, err := database.DB.Exec("DELETE FROM watch_lists WHERE user_id = $1", userID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup watch lists: %v", err)
	}

	// Delete test user
	_, err = database.DB.Exec("DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup user: %v", err)
	}
}

// Helper to add test tickers to tickers table
func addTestTickers(t *testing.T) {
	testTickers := []struct {
		symbol    string
		name      string
		exchange  string
		assetType string
	}{
		{"AAPL", "Apple Inc.", "NASDAQ", "stock"},
		{"MSFT", "Microsoft Corporation", "NASDAQ", "stock"},
		{"GOOGL", "Alphabet Inc.", "NASDAQ", "stock"},
		{"TSLA", "Tesla Inc.", "NASDAQ", "stock"},
		{"AMZN", "Amazon.com Inc.", "NASDAQ", "stock"},
	}

	for _, ticker := range testTickers {
		// Use INSERT ... ON CONFLICT to avoid duplicates
		_, err := database.DB.Exec(`
			INSERT INTO tickers (symbol, name, exchange, asset_type)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (symbol) DO NOTHING
		`, ticker.symbol, ticker.name, ticker.exchange, ticker.assetType)

		if err != nil {
			t.Logf("Warning: Failed to add ticker %s: %v", ticker.symbol, err)
		}
	}
}

func TestListWatchLists(t *testing.T) {
	router := setupTestRouter()
	router.GET("/watchlists", ListWatchLists)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)

	// Test: List watch lists (should include auto-created default)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/watchlists", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	watchLists := response["watch_lists"].([]interface{})
	assert.GreaterOrEqual(t, len(watchLists), 1, "Should have at least the default watch list")
}

func TestCreateWatchList(t *testing.T) {
	router := setupTestRouter()
	router.POST("/watchlists", CreateWatchList)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)

	// Test: Create valid watch list
	createReq := models.CreateWatchListRequest{
		Name:        "Tech Stocks",
		Description: stringPtr("My technology investments"),
	}

	body, _ := json.Marshal(createReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/watchlists", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var watchList models.WatchList
	err := json.Unmarshal(w.Body.Bytes(), &watchList)
	assert.NoError(t, err)
	assert.Equal(t, "Tech Stocks", watchList.Name)
	assert.NotEmpty(t, watchList.ID)
}

func TestCreateWatchListInvalidName(t *testing.T) {
	router := setupTestRouter()
	router.POST("/watchlists", CreateWatchList)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)

	// Test: Create watch list with empty name
	createReq := models.CreateWatchListRequest{
		Name: "",
	}

	body, _ := json.Marshal(createReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/watchlists", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetWatchList(t *testing.T) {
	router := setupTestRouter()
	router.GET("/watchlists/:id", GetWatchList)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)
	addTestTickers(t)

	// Create a watch list first
	watchList := &models.WatchList{
		UserID: userID,
		Name:   "Test Portfolio",
	}
	err := database.CreateWatchList(watchList)
	assert.NoError(t, err)

	// Add a ticker
	item := &models.WatchListItem{
		WatchListID: watchList.ID,
		Symbol:      "AAPL",
		Tags:        []string{"tech"},
	}
	err = database.AddTickerToWatchList(item)
	assert.NoError(t, err)

	// Test: Get watch list with items
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/watchlists/%s", watchList.ID), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.WatchListWithItems
	err = json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, "Test Portfolio", result.Name)
	assert.Equal(t, 1, result.ItemCount)
	assert.Equal(t, 1, len(result.Items))
	assert.Equal(t, "AAPL", result.Items[0].Symbol)
}

func TestUpdateWatchList(t *testing.T) {
	router := setupTestRouter()
	router.PUT("/watchlists/:id", UpdateWatchList)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)

	// Create a watch list first
	watchList := &models.WatchList{
		UserID: userID,
		Name:   "Original Name",
	}
	err := database.CreateWatchList(watchList)
	assert.NoError(t, err)

	// Test: Update watch list
	updateReq := models.UpdateWatchListRequest{
		Name:        "Updated Name",
		Description: stringPtr("New description"),
	}

	body, _ := json.Marshal(updateReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/watchlists/%s", watchList.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify update
	updated, err := database.GetWatchListByID(watchList.ID, userID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
}

func TestDeleteWatchList(t *testing.T) {
	router := setupTestRouter()
	router.DELETE("/watchlists/:id", DeleteWatchList)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)

	// Create a watch list first
	watchList := &models.WatchList{
		UserID: userID,
		Name:   "To Be Deleted",
	}
	err := database.CreateWatchList(watchList)
	assert.NoError(t, err)

	// Test: Delete watch list
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/watchlists/%s", watchList.ID), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify deletion
	_, err = database.GetWatchListByID(watchList.ID, userID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestAddTickerToWatchList(t *testing.T) {
	router := setupTestRouter()
	router.POST("/watchlists/:id/items", AddTickerToWatchList)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)
	addTestTickers(t)

	// Create a watch list first
	watchList := &models.WatchList{
		UserID: userID,
		Name:   "My Stocks",
	}
	err := database.CreateWatchList(watchList)
	assert.NoError(t, err)

	// Test: Add valid ticker
	addReq := models.AddTickerRequest{
		Symbol:         "AAPL",
		Notes:          stringPtr("Great company"),
		Tags:           []string{"tech", "growth"},
		TargetBuyPrice: float64Ptr(150.0),
	}

	body, _ := json.Marshal(addReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/watchlists/%s/items", watchList.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var item models.WatchListItem
	err = json.Unmarshal(w.Body.Bytes(), &item)
	assert.NoError(t, err)
	assert.Equal(t, "AAPL", item.Symbol)
	assert.Equal(t, 2, len(item.Tags))
}

func TestAddDuplicateTicker(t *testing.T) {
	router := setupTestRouter()
	router.POST("/watchlists/:id/items", AddTickerToWatchList)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)
	addTestTickers(t)

	// Create a watch list with a ticker
	watchList := &models.WatchList{
		UserID: userID,
		Name:   "My Stocks",
	}
	err := database.CreateWatchList(watchList)
	assert.NoError(t, err)

	item := &models.WatchListItem{
		WatchListID: watchList.ID,
		Symbol:      "AAPL",
		Tags:        []string{},
	}
	err = database.AddTickerToWatchList(item)
	assert.NoError(t, err)

	// Test: Try to add duplicate ticker
	addReq := models.AddTickerRequest{
		Symbol: "AAPL",
	}

	body, _ := json.Marshal(addReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/watchlists/%s/items", watchList.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestAddInvalidTicker(t *testing.T) {
	router := setupTestRouter()
	router.POST("/watchlists/:id/items", AddTickerToWatchList)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)

	// Create a watch list
	watchList := &models.WatchList{
		UserID: userID,
		Name:   "My Stocks",
	}
	err := database.CreateWatchList(watchList)
	assert.NoError(t, err)

	// Test: Add non-existent ticker
	addReq := models.AddTickerRequest{
		Symbol: "INVALID_TICKER",
	}

	body, _ := json.Marshal(addReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/watchlists/%s/items", watchList.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFreeTierLimit(t *testing.T) {
	router := setupTestRouter()
	router.POST("/watchlists/:id/items", AddTickerToWatchList)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)
	addTestTickers(t)

	// Create a watch list
	watchList := &models.WatchList{
		UserID: userID,
		Name:   "Free Tier Test",
	}
	err := database.CreateWatchList(watchList)
	assert.NoError(t, err)

	// Add 10 tickers (free tier limit)
	tickers := []string{"AAPL", "MSFT", "GOOGL", "TSLA", "AMZN"}

	// Add more test tickers to reach 10
	for i := 0; i < 5; i++ {
		symbol := fmt.Sprintf("TEST%d", i)
		_, err := database.DB.Exec(`
			INSERT INTO tickers (symbol, name, exchange, asset_type)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (symbol) DO NOTHING
		`, symbol, fmt.Sprintf("Test Stock %d", i), "NASDAQ", "stock")
		assert.NoError(t, err)
		tickers = append(tickers, symbol)
	}

	// Add 10 tickers successfully
	for i := 0; i < 10; i++ {
		item := &models.WatchListItem{
			WatchListID: watchList.ID,
			Symbol:      tickers[i%len(tickers)],
			Tags:        []string{},
		}
		err = database.AddTickerToWatchList(item)
		if i < 10 {
			assert.NoError(t, err, fmt.Sprintf("Should be able to add ticker %d", i+1))
		}
	}

	// Test: Try to add 11th ticker (should fail)
	addReq := models.AddTickerRequest{
		Symbol: "AAPL",
	}

	// First remove AAPL if it was added, then add a unique ticker for the 11th attempt
	_, _ = database.DB.Exec("DELETE FROM watch_list_items WHERE watch_list_id = $1 AND symbol = $2", watchList.ID, "AAPL")

	// Add AAPL back to ensure we have 10
	item := &models.WatchListItem{
		WatchListID: watchList.ID,
		Symbol:      "AAPL",
		Tags:        []string{},
	}
	err = database.AddTickerToWatchList(item)
	assert.NoError(t, err)

	// Now add an 11th unique ticker
	_, err = database.DB.Exec(`
		INSERT INTO tickers (symbol, name, exchange, asset_type)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (symbol) DO NOTHING
	`, "LIMIT_TEST", "Limit Test Stock", "NASDAQ", "stock")

	addReq.Symbol = "LIMIT_TEST"
	body, _ := json.Marshal(addReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/watchlists/%s/items", watchList.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var errResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &errResponse)
	assert.NoError(t, err)
	assert.Contains(t, errResponse["error"].(string), "limit")
}

func TestRemoveTickerFromWatchList(t *testing.T) {
	router := setupTestRouter()
	router.DELETE("/watchlists/:id/items/:symbol", RemoveTickerFromWatchList)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)
	addTestTickers(t)

	// Create a watch list with a ticker
	watchList := &models.WatchList{
		UserID: userID,
		Name:   "My Stocks",
	}
	err := database.CreateWatchList(watchList)
	assert.NoError(t, err)

	item := &models.WatchListItem{
		WatchListID: watchList.ID,
		Symbol:      "AAPL",
		Tags:        []string{},
	}
	err = database.AddTickerToWatchList(item)
	assert.NoError(t, err)

	// Test: Remove ticker
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/watchlists/%s/items/AAPL", watchList.ID), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify removal
	items, err := database.GetWatchListItems(watchList.ID)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(items))
}

func TestUpdateWatchListItem(t *testing.T) {
	router := setupTestRouter()
	router.PUT("/watchlists/:id/items/:symbol", UpdateWatchListItem)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)
	addTestTickers(t)

	// Create a watch list with a ticker
	watchList := &models.WatchList{
		UserID: userID,
		Name:   "My Stocks",
	}
	err := database.CreateWatchList(watchList)
	assert.NoError(t, err)

	item := &models.WatchListItem{
		WatchListID: watchList.ID,
		Symbol:      "AAPL",
		Tags:        []string{"tech"},
	}
	err = database.AddTickerToWatchList(item)
	assert.NoError(t, err)

	// Test: Update ticker metadata
	updateReq := models.UpdateTickerRequest{
		Notes:          stringPtr("Updated notes"),
		Tags:           []string{"tech", "growth"},
		TargetBuyPrice: float64Ptr(155.0),
	}

	body, _ := json.Marshal(updateReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/watchlists/%s/items/AAPL", watchList.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var updatedItem models.WatchListItem
	err = json.Unmarshal(w.Body.Bytes(), &updatedItem)
	assert.NoError(t, err)
	assert.Equal(t, "Updated notes", *updatedItem.Notes)
	assert.Equal(t, 2, len(updatedItem.Tags))
	assert.Equal(t, 155.0, *updatedItem.TargetBuyPrice)
}

func TestBulkAddTickers(t *testing.T) {
	router := setupTestRouter()
	router.POST("/watchlists/:id/bulk", BulkAddTickers)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)
	addTestTickers(t)

	// Create a watch list
	watchList := &models.WatchList{
		UserID: userID,
		Name:   "Bulk Test",
	}
	err := database.CreateWatchList(watchList)
	assert.NoError(t, err)

	// Test: Bulk add tickers
	bulkReq := models.BulkAddTickersRequest{
		Symbols: []string{"AAPL", "MSFT", "GOOGL"},
	}

	body, _ := json.Marshal(bulkReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/watchlists/%s/bulk", watchList.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, 3.0, result["total"])
	assert.Equal(t, 3, len(result["added"].([]interface{})))
	assert.Equal(t, 0, len(result["failed"].([]interface{})))
}

// ---------------------------------------------------------------------------
// Phase 1: Comprehensive tests for production failure fixes
// ---------------------------------------------------------------------------

// Test: Default watchlist cannot be deleted
func TestDeleteDefaultWatchList(t *testing.T) {
	router := setupTestRouter()
	router.DELETE("/watchlists/:id", DeleteWatchList)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)

	// Find the default watch list (auto-created by trigger on user insert)
	watchLists, err := database.GetWatchListsByUserID(userID)
	assert.NoError(t, err)

	var defaultWL *models.WatchListSummary
	for _, wl := range watchLists {
		if wl.IsDefault {
			defaultWL = &wl
			break
		}
	}

	if defaultWL == nil {
		// If no auto-created default, create one explicitly
		wl := &models.WatchList{
			UserID:    userID,
			Name:      "Default List",
			IsDefault: true,
		}
		err := database.CreateWatchList(wl)
		assert.NoError(t, err)
		defaultWL = &models.WatchListSummary{ID: wl.ID, IsDefault: true}
	}

	// Test: Try to delete default watch list — should be forbidden
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/watchlists/%s", defaultWL.ID), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var errResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &errResponse)
	assert.NoError(t, err)
	assert.Contains(t, errResponse["error"].(string), "default")

	// Verify it still exists
	_, err = database.GetWatchListByID(defaultWL.ID, userID)
	assert.NoError(t, err)
}

// Test: Non-default watchlist can still be deleted
func TestDeleteNonDefaultWatchList(t *testing.T) {
	router := setupTestRouter()
	router.DELETE("/watchlists/:id", DeleteWatchList)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)

	// Create a non-default watch list
	watchList := &models.WatchList{
		UserID:    userID,
		Name:      "Deletable List",
		IsDefault: false,
	}
	err := database.CreateWatchList(watchList)
	assert.NoError(t, err)

	// Test: Delete non-default watch list — should succeed
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/watchlists/%s", watchList.ID), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify it's gone
	_, err = database.GetWatchListByID(watchList.ID, userID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// Test: Delete non-existent watchlist returns 404
func TestDeleteNonExistentWatchList(t *testing.T) {
	router := setupTestRouter()
	router.DELETE("/watchlists/:id", DeleteWatchList)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/watchlists/non-existent-id-123", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Test: Watch list count limit enforcement for free tier
func TestCreateWatchListCountLimit(t *testing.T) {
	router := setupTestRouter()
	router.POST("/watchlists", CreateWatchList)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)

	// User may already have a default watch list from trigger, count existing ones
	existing, err := database.GetWatchListsByUserID(userID)
	assert.NoError(t, err)

	// Create up to 3 watch lists total (free tier limit)
	for i := len(existing); i < 3; i++ {
		createReq := models.CreateWatchListRequest{
			Name: fmt.Sprintf("Watch List %d", i+1),
		}
		body, _ := json.Marshal(createReq)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/watchlists", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code, "Should be able to create watch list %d", i+1)
	}

	// Verify we now have 3
	lists, err := database.GetWatchListsByUserID(userID)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(lists))

	// Test: 4th watch list should fail for free tier
	createReq := models.CreateWatchListRequest{
		Name: "Fourth List - Should Fail",
	}
	body, _ := json.Marshal(createReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/watchlists", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var errResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &errResponse)
	assert.NoError(t, err)
	assert.Contains(t, errResponse["error"].(string), "limit")
	assert.Contains(t, errResponse["error"].(string), "Maximum 3")
}

// Test: GetWatchList returns proper error for not found
func TestGetWatchListNotFound(t *testing.T) {
	router := setupTestRouter()
	router.GET("/watchlists/:id", GetWatchList)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/watchlists/non-existent-uuid-123", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errResponse)
	assert.NoError(t, err)
	assert.Contains(t, errResponse["error"].(string), "not found")
}

// Test: GetWatchList with items works correctly (enriched data query)
func TestGetWatchListWithEnrichedData(t *testing.T) {
	router := setupTestRouter()
	router.GET("/watchlists/:id", GetWatchList)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)
	addTestTickers(t)

	// Create a watch list and add items
	watchList := &models.WatchList{
		UserID: userID,
		Name:   "Enriched Test",
	}
	err := database.CreateWatchList(watchList)
	assert.NoError(t, err)

	for _, symbol := range []string{"AAPL", "MSFT"} {
		item := &models.WatchListItem{
			WatchListID: watchList.ID,
			Symbol:      symbol,
			Tags:        []string{},
		}
		err = database.AddTickerToWatchList(item)
		assert.NoError(t, err)
	}

	// Test: Fetch enriched data — should succeed even without reddit data
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/watchlists/%s", watchList.ID), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.WatchListWithItems
	err = json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, 2, result.ItemCount)
	assert.Equal(t, 2, len(result.Items))

	// Items should have ticker data from tickers table
	for _, item := range result.Items {
		assert.NotEmpty(t, item.Name, "Item should have name from tickers table")
		assert.NotEmpty(t, item.Exchange, "Item should have exchange from tickers table")
		assert.NotEmpty(t, item.AssetType, "Item should have asset_type from tickers table")
	}

	// Reddit fields should be nil when no reddit data exists
	for _, item := range result.Items {
		assert.Nil(t, item.RedditRank, "Reddit rank should be nil when no data")
		assert.Nil(t, item.RedditMentions, "Reddit mentions should be nil when no data")
		assert.Nil(t, item.RedditPopularity, "Reddit popularity should be nil when no data")
		assert.Nil(t, item.RedditTrend, "Reddit trend should be nil when no data")
		assert.Nil(t, item.RedditRankChange, "Reddit rank change should be nil when no data")
	}
}

// Test: GetWatchList enriched data with Reddit data present
func TestGetWatchListWithRedditData(t *testing.T) {
	router := setupTestRouter()
	router.GET("/watchlists/:id", GetWatchList)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)
	addTestTickers(t)

	// Create watch list and add AAPL
	watchList := &models.WatchList{
		UserID: userID,
		Name:   "Reddit Data Test",
	}
	err := database.CreateWatchList(watchList)
	assert.NoError(t, err)

	item := &models.WatchListItem{
		WatchListID: watchList.ID,
		Symbol:      "AAPL",
		Tags:        []string{},
	}
	err = database.AddTickerToWatchList(item)
	assert.NoError(t, err)

	// Insert Reddit heatmap data for AAPL
	_, err = database.DB.Exec(`
		INSERT INTO reddit_heatmap_daily (ticker_symbol, date, avg_rank, total_mentions, popularity_score, trend_direction)
		VALUES ('AAPL', CURRENT_DATE, 5.0, 120, 85.5, 'rising')
	`)
	if err != nil {
		t.Logf("Warning: Failed to insert reddit heatmap data: %v (may lack FK constraints in test DB)", err)
	}

	// Insert Reddit ticker ranking data for AAPL
	_, err = database.DB.Exec(`
		INSERT INTO reddit_ticker_rankings (ticker_symbol, rank, mentions, rank_24h_ago, snapshot_date, snapshot_time)
		VALUES ('AAPL', 5, 120, 8, CURRENT_DATE, NOW())
	`)
	if err != nil {
		t.Logf("Warning: Failed to insert reddit ranking data: %v", err)
	}

	// Cleanup reddit data after test
	defer func() {
		database.DB.Exec("DELETE FROM reddit_ticker_rankings WHERE ticker_symbol = 'AAPL'")
		database.DB.Exec("DELETE FROM reddit_heatmap_daily WHERE ticker_symbol = 'AAPL'")
	}()

	// Test: Fetch enriched data — should include reddit data
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/watchlists/%s", watchList.ID), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.WatchListWithItems
	err = json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.ItemCount)

	aaplItem := result.Items[0]
	assert.Equal(t, "AAPL", aaplItem.Symbol)

	// Reddit heatmap data should be present
	if aaplItem.RedditRank != nil {
		assert.Equal(t, 5, *aaplItem.RedditRank)
	}
	if aaplItem.RedditMentions != nil {
		assert.Equal(t, 120, *aaplItem.RedditMentions)
	}
	if aaplItem.RedditTrend != nil {
		assert.Equal(t, "rising", *aaplItem.RedditTrend)
	}
	// Rank change: rank(5) - rank_24h_ago(8) = -3
	if aaplItem.RedditRankChange != nil {
		assert.Equal(t, -3, *aaplItem.RedditRankChange)
	}
}

// Test: Add ticker to watch list owned by another user fails
func TestAddTickerUnauthorizedWatchList(t *testing.T) {
	router := setupTestRouter()
	router.POST("/watchlists/:id/items", AddTickerToWatchList)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)

	// Create a watch list with a different user (not the mock "test-user-id-123")
	otherUserID := "other-user-id-456"
	var exists bool
	_ = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", otherUserID).Scan(&exists)
	if !exists {
		_, err := database.DB.Exec(`
			INSERT INTO users (id, email, password_hash, full_name)
			VALUES ($1, 'other@test.com', 'hash', 'Other User')
		`, otherUserID)
		if err != nil {
			t.Fatalf("Failed to create other user: %v", err)
		}
	}
	defer database.DB.Exec("DELETE FROM watch_lists WHERE user_id = $1", otherUserID)
	defer database.DB.Exec("DELETE FROM users WHERE id = $1", otherUserID)

	watchList := &models.WatchList{
		UserID: otherUserID,
		Name:   "Other User's List",
	}
	err := database.CreateWatchList(watchList)
	assert.NoError(t, err)

	// Test: Try to add ticker to another user's watch list
	addReq := models.AddTickerRequest{Symbol: "AAPL"}
	body, _ := json.Marshal(addReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/watchlists/%s/items", watchList.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// Test: Remove ticker that doesn't exist returns 404
func TestRemoveNonExistentTicker(t *testing.T) {
	router := setupTestRouter()
	router.DELETE("/watchlists/:id/items/:symbol", RemoveTickerFromWatchList)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)

	watchList := &models.WatchList{
		UserID: userID,
		Name:   "Test List",
	}
	err := database.CreateWatchList(watchList)
	assert.NoError(t, err)

	// Test: Remove ticker that was never added
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/watchlists/%s/items/INVALID", watchList.ID), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Test: Reorder items with valid data
func TestReorderWatchListItems(t *testing.T) {
	router := setupTestRouter()
	router.PUT("/watchlists/:id/reorder", ReorderWatchListItems)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)
	addTestTickers(t)

	// Create watch list and add items
	watchList := &models.WatchList{
		UserID: userID,
		Name:   "Reorder Test",
	}
	err := database.CreateWatchList(watchList)
	assert.NoError(t, err)

	var itemIDs []string
	for _, symbol := range []string{"AAPL", "MSFT", "GOOGL"} {
		item := &models.WatchListItem{
			WatchListID: watchList.ID,
			Symbol:      symbol,
			Tags:        []string{},
		}
		err = database.AddTickerToWatchList(item)
		assert.NoError(t, err)
		itemIDs = append(itemIDs, item.ID)
	}

	// Reorder: reverse the order
	reorderReq := models.ReorderItemsRequest{
		ItemOrders: []models.ItemOrder{
			{ItemID: itemIDs[0], DisplayOrder: 2},
			{ItemID: itemIDs[1], DisplayOrder: 1},
			{ItemID: itemIDs[2], DisplayOrder: 0},
		},
	}

	body, _ := json.Marshal(reorderReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/watchlists/%s/reorder", watchList.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify new order
	items, err := database.GetWatchListItems(watchList.ID)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(items))
	// Items ordered by display_order ASC, so GOOGL (order=0) should be first
	assert.Equal(t, "GOOGL", items[0].Symbol)
	assert.Equal(t, "MSFT", items[1].Symbol)
	assert.Equal(t, "AAPL", items[2].Symbol)
}

// Test: Empty watchlist returns empty items array (not null)
func TestGetEmptyWatchList(t *testing.T) {
	router := setupTestRouter()
	router.GET("/watchlists/:id", GetWatchList)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)

	watchList := &models.WatchList{
		UserID: userID,
		Name:   "Empty List",
	}
	err := database.CreateWatchList(watchList)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/watchlists/%s", watchList.ID), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.WatchListWithItems
	err = json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ItemCount)
	assert.NotNil(t, result.Items, "Items should be empty array, not null")
	assert.Equal(t, 0, len(result.Items))
}

// Test: Update watch list item that doesn't exist returns 404
func TestUpdateNonExistentWatchListItem(t *testing.T) {
	router := setupTestRouter()
	router.PUT("/watchlists/:id/items/:symbol", UpdateWatchListItem)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)

	watchList := &models.WatchList{
		UserID: userID,
		Name:   "Test List",
	}
	err := database.CreateWatchList(watchList)
	assert.NoError(t, err)

	// Test: Update a non-existent symbol
	updateReq := models.UpdateTickerRequest{
		Notes: stringPtr("Some notes"),
	}
	body, _ := json.Marshal(updateReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/watchlists/%s/items/NONEXIST", watchList.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Test: CreateWatchList with invalid request body
func TestCreateWatchListInvalidJSON(t *testing.T) {
	router := setupTestRouter()
	router.POST("/watchlists", CreateWatchList)

	userID := createTestUser(t)
	defer cleanupTestData(t, userID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/watchlists", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}
