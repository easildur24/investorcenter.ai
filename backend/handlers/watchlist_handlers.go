package handlers

import (
	"log"
	"net/http"
	"strings"

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

	// Enforce watchlist count limit before creating
	existingLists, err := database.GetWatchListsByUserID(userID)
	if err != nil {
		log.Printf("Error checking watch list count for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create watch list"})
		return
	}

	// Free tier: max 3 watchlists, Premium: max 20
	maxWatchLists := 3
	isPremium, _ := database.IsUserPremium(userID)
	if isPremium {
		maxWatchLists = 20
	}
	if len(existingLists) >= maxWatchLists {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Watch list limit reached. " + func() string {
				if isPremium {
					return "Premium tier allows maximum 20 watch lists"
				}
				return "Free tier allows maximum 3 watch lists. Upgrade to Premium for up to 20"
			}(),
		})
		return
	}

	watchList := &models.WatchList{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		IsDefault:   false,
	}

	if err := database.CreateWatchList(watchList); err != nil {
		log.Printf("Error creating watch list for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create watch list"})
		return
	}

	c.JSON(http.StatusCreated, watchList)
}

// GetWatchList retrieves a single watch list with all items and real-time prices
func GetWatchList(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	watchListID := c.Param("id")

	// Get watch list with items and prices
	result, err := watchListService.GetWatchListWithItems(watchListID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Watch list not found"})
		} else {
			log.Printf("Error fetching watch list %s for user %s: %v", watchListID, userID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch watch list"})
		}
		return
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		if strings.Contains(err.Error(), "not found") {
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
		if err.Error() == "ticker already exists in this watch list" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else if strings.Contains(err.Error(), "limit reached") {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else if err.Error() == "ticker not found in database" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
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
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
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
