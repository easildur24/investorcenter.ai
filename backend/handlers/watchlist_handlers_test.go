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

// Helper to add test tickers to stocks table
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
			INSERT INTO stocks (symbol, name, exchange, asset_type)
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
			INSERT INTO stocks (symbol, name, exchange, asset_type)
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
	database.DB.Exec("DELETE FROM watch_list_items WHERE watch_list_id = $1 AND symbol = $2", watchList.ID, "AAPL")

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
		INSERT INTO stocks (symbol, name, exchange, asset_type)
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

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}
