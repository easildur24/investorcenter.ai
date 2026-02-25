package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"investorcenter-api/auth"
	"investorcenter-api/database"
	"investorcenter-api/models"
	"investorcenter-api/services"
)

var watchListService = services.NewWatchListService()

// ListWatchLists returns all watch lists for the authenticated user
func ListWatchLists(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	watchLists, err := database.GetWatchListsByUserID(userID)
	if err != nil {
		log.Printf("Error fetching watch lists for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch watch lists"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"watch_lists": watchLists})
}

// CreateWatchList creates a new watch list
func CreateWatchList(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.CreateWatchListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	watchList := &models.WatchList{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		IsDefault:   false,
	}

	// Atomic insert with count check to prevent TOCTOU race
	err := database.CreateWatchListAtomic(watchList, database.MaxWatchListsPerUser)
	if err != nil {
		if isWatchListLimitError(err) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Watch list limit reached. Maximum 3 watch lists allowed",
			})
			return
		}
		log.Printf("Error creating watch list for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create watch list"})
		return
	}

	c.JSON(http.StatusCreated, watchList)
}

// GetWatchList retrieves a single watch list with all items and real-time prices.
// Returns IC Score, fundamentals, valuation ratios, Reddit data,
// alert counts, and summary metrics alongside real-time prices.
func GetWatchList(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	watchListID := c.Param("id")

	// Get watch list with items and summary metrics
	result, err := watchListService.GetWatchListWithItems(watchListID, userID)
	if err != nil {
		if errors.Is(err, database.ErrWatchListNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Watch list not found"})
		} else {
			log.Printf("Error fetching watch list %s for user %s: %v", watchListID, userID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch watch list"})
		}
		return
	}

	// Optionally enrich each item with its full alert object (1:1 per watchlist item).
	// Ownership is already verified: GetWatchListWithItems filters by userID and
	// returns ErrWatchListNotFound (early return above) if the user doesn't own it.
	// Usage: GET /api/v1/watchlists/:id?include_alerts=true
	if c.Query("include_alerts") == "true" {
		alertMap, alertErr := database.GetAlertForWatchListItems(watchListID, userID)
		if alertErr != nil {
			log.Printf("Warning: failed to fetch alerts for watchlist %s: %v", watchListID, alertErr)
			// Non-fatal: items are still returned without alert data.
			// Signal to the frontend so it can show a degraded-state indicator
			// rather than silently representing the state as "no alerts".
			result.AlertsFetchFailed = true
		} else {
			for i := range result.Items {
				result.Items[i].Alert = alertMap[result.Items[i].Symbol]
			}
		}
	}

	c.JSON(http.StatusOK, result)
}

// UpdateWatchList updates watch list metadata
func UpdateWatchList(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	watchListID := c.Param("id")

	var req models.UpdateWatchListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	watchList := &models.WatchList{
		ID:          watchListID,
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
	}

	err := database.UpdateWatchList(watchList)
	if err != nil {
		if errors.Is(err, database.ErrWatchListNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Watch list not found"})
		} else {
			log.Printf("Error updating watch list %s for user %s: %v", watchListID, userID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update watch list"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Watch list updated successfully"})
}

// DeleteWatchList deletes a watch list (protects default watch lists from deletion)
func DeleteWatchList(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	watchListID := c.Param("id")

	// Protect default watchlist from deletion
	watchList, err := database.GetWatchListByID(watchListID, userID)
	if err != nil {
		if errors.Is(err, database.ErrWatchListNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Watch list not found"})
		} else {
			log.Printf("Error fetching watch list %s for deletion: %v", watchListID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete watch list"})
		}
		return
	}

	if watchList.IsDefault {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete the default watch list"})
		return
	}

	err = database.DeleteWatchList(watchListID, userID)
	if err != nil {
		log.Printf("Error deleting watch list %s for user %s: %v", watchListID, userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete watch list"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Watch list deleted successfully"})
}

// AddTickerToWatchList adds a ticker to a watch list
func AddTickerToWatchList(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	watchListID := c.Param("id")

	// Verify ownership
	if err := watchListService.ValidateWatchListOwnership(watchListID, userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized access to watch list"})
		return
	}

	var req models.AddTickerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item := &models.WatchListItem{
		WatchListID:     watchListID,
		Symbol:          req.Symbol,
		Notes:           req.Notes,
		Tags:            req.Tags,
		TargetBuyPrice:  req.TargetBuyPrice,
		TargetSellPrice: req.TargetSellPrice,
	}

	err := database.AddTickerToWatchList(item)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrTickerAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, database.ErrWatchListItemLimitReached):
			c.JSON(http.StatusForbidden, gin.H{"error": "Watch list item limit reached. Maximum 10 tickers per watch list"})
		case errors.Is(err, database.ErrTickerNotFound):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			log.Printf("Error adding ticker %s to watch list %s: %v", req.Symbol, watchListID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add ticker to watch list"})
		}
		return
	}

	c.JSON(http.StatusCreated, item)
}

// RemoveTickerFromWatchList removes a ticker from watch list
func RemoveTickerFromWatchList(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	watchListID := c.Param("id")
	symbol := c.Param("symbol")

	// Verify ownership
	if err := watchListService.ValidateWatchListOwnership(watchListID, userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized access to watch list"})
		return
	}

	err := database.RemoveTickerFromWatchList(watchListID, symbol)
	if err != nil {
		if errors.Is(err, database.ErrWatchListItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Ticker not found in watch list"})
		} else {
			log.Printf("Error removing ticker %s from watch list %s: %v", symbol, watchListID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove ticker"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Ticker removed successfully"})
}

// UpdateWatchListItem updates ticker metadata
func UpdateWatchListItem(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	watchListID := c.Param("id")
	symbol := c.Param("symbol")

	// Verify ownership
	if err := watchListService.ValidateWatchListOwnership(watchListID, userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized access to watch list"})
		return
	}

	var req models.UpdateTickerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing item to update
	items, err := database.GetWatchListItems(watchListID)
	if err != nil {
		log.Printf("Error fetching watch list items for update (list=%s, symbol=%s): %v", watchListID, symbol, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch watch list items"})
		return
	}

	var targetItem *models.WatchListItem
	for _, item := range items {
		if item.Symbol == symbol {
			targetItem = &item
			break
		}
	}

	if targetItem == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticker not found in watch list"})
		return
	}

	// Update fields
	targetItem.Notes = req.Notes
	targetItem.Tags = req.Tags
	targetItem.TargetBuyPrice = req.TargetBuyPrice
	targetItem.TargetSellPrice = req.TargetSellPrice

	err = database.UpdateWatchListItem(targetItem)
	if err != nil {
		log.Printf("Error updating watch list item (list=%s, symbol=%s): %v", watchListID, symbol, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update ticker"})
		return
	}

	c.JSON(http.StatusOK, targetItem)
}

// BulkAddTickers adds multiple tickers from CSV import
func BulkAddTickers(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	watchListID := c.Param("id")

	// Verify ownership
	if err := watchListService.ValidateWatchListOwnership(watchListID, userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized access to watch list"})
		return
	}

	var req models.BulkAddTickersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	added, failed, err := database.BulkAddTickers(watchListID, req.Symbols)
	if err != nil {
		log.Printf("Error bulk adding tickers to watch list %s: %v", watchListID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to bulk add tickers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"added":  added,
		"failed": failed,
		"total":  len(req.Symbols),
	})
}

// ReorderWatchListItems updates display order
func ReorderWatchListItems(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	watchListID := c.Param("id")

	// Verify ownership
	if err := watchListService.ValidateWatchListOwnership(watchListID, userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized access to watch list"})
		return
	}

	var req models.ReorderItemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate all submitted item IDs belong to this watchlist (prevent cross-list manipulation)
	itemIDs := make([]string, len(req.ItemOrders))
	for i, order := range req.ItemOrders {
		itemIDs[i] = order.ItemID
	}
	if err := database.ValidateItemsBelongToWatchList(watchListID, itemIDs); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "One or more items do not belong to this watch list"})
		return
	}

	// Update each item's display order
	for _, itemOrder := range req.ItemOrders {
		err := database.UpdateItemDisplayOrder(itemOrder.ItemID, itemOrder.DisplayOrder)
		if err != nil {
			log.Printf("Error reordering watch list item %s in list %s: %v", itemOrder.ItemID, watchListID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update item order"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Items reordered successfully"})
}

// GetUserTags returns all distinct tags used across the authenticated user's watchlist items.
func GetUserTags(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tags, err := database.GetUserTags(userID)
	if err != nil {
		log.Printf("Error fetching tags for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tags"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

// isWatchListLimitError checks if an error indicates the watchlist count limit was reached.
// This catches the error from CreateWatchListAtomic when the INSERT...WHERE count < limit returns no rows.
func isWatchListLimitError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return len(msg) >= 25 && msg[:25] == "watch list limit reached:"
}
