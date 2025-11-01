# Phase 2: Watch List Management - Technical Specification

## Overview

**Goal:** Implement a complete watch list management system that allows authenticated users to create multiple watch lists, add/remove tickers, organize them, and view real-time prices.

**Timeline:** 2 weeks (10 working days)

**Dependencies:**
- Phase 1 (Authentication & User Management) - ✅ Complete
- PostgreSQL database (already running)
- Backend Go/Gin server (already set up)
- Frontend Next.js 14 app (already set up)
- Polygon.io API for real-time stock prices (already integrated)
- Redis for crypto price caching (already integrated)
- Authentication middleware from Phase 1

**Key Features:**
- Multiple watch lists per user
- Add/remove tickers (stocks & crypto)
- Organize watch lists (rename, reorder, delete)
- Ticker metadata (notes, tags, target prices)
- Table view with real-time price updates
- Bulk import from CSV
- Free tier limits (max 10 tickers per watch list)

---

## Database Schema

### Migration File: `migrations/007_watchlist_tables.sql`

```sql
-- Watch Lists table
CREATE TABLE watch_lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_default BOOLEAN DEFAULT FALSE,
    display_order INTEGER,
    is_public BOOLEAN DEFAULT FALSE, -- For future sharing feature
    public_slug VARCHAR(100) UNIQUE, -- For future public URLs
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for watch_lists
CREATE INDEX idx_watch_lists_user_id ON watch_lists(user_id);
CREATE INDEX idx_watch_lists_user_id_display_order ON watch_lists(user_id, display_order);

-- Watch List Items table
CREATE TABLE watch_list_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    watch_list_id UUID REFERENCES watch_lists(id) ON DELETE CASCADE,
    symbol VARCHAR(20) NOT NULL, -- References stocks.symbol
    notes TEXT,
    tags TEXT[], -- Array of custom tags
    target_buy_price DECIMAL(20, 4),
    target_sell_price DECIMAL(20, 4),
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    display_order INTEGER,
    UNIQUE(watch_list_id, symbol) -- Prevent duplicate tickers in same watch list
);

-- Indexes for watch_list_items
CREATE INDEX idx_watch_list_items_watch_list_id ON watch_list_items(watch_list_id);
CREATE INDEX idx_watch_list_items_symbol ON watch_list_items(symbol);
CREATE INDEX idx_watch_list_items_watch_list_id_display_order ON watch_list_items(watch_list_id, display_order);

-- Trigger to update updated_at timestamp for watch_lists
CREATE TRIGGER update_watch_lists_updated_at BEFORE UPDATE ON watch_lists
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to enforce watch list limits for free tier
CREATE OR REPLACE FUNCTION check_watch_list_item_limit()
RETURNS TRIGGER AS $$
DECLARE
    user_is_premium BOOLEAN;
    current_item_count INTEGER;
    max_items INTEGER := 10; -- Free tier limit
BEGIN
    -- Get user's premium status
    SELECT u.is_premium INTO user_is_premium
    FROM watch_lists wl
    JOIN users u ON wl.user_id = u.id
    WHERE wl.id = NEW.watch_list_id;

    -- Skip check for premium users
    IF user_is_premium THEN
        RETURN NEW;
    END IF;

    -- Count existing items in watch list
    SELECT COUNT(*) INTO current_item_count
    FROM watch_list_items
    WHERE watch_list_id = NEW.watch_list_id;

    -- Check if limit exceeded
    IF current_item_count >= max_items THEN
        RAISE EXCEPTION 'Watch list limit reached. Free tier allows maximum % tickers per watch list.', max_items;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to enforce watch list item limit
CREATE TRIGGER enforce_watch_list_item_limit
BEFORE INSERT ON watch_list_items
FOR EACH ROW EXECUTE FUNCTION check_watch_list_item_limit();

-- Function to create default watch list for new users
CREATE OR REPLACE FUNCTION create_default_watch_list()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO watch_lists (user_id, name, description, is_default, display_order)
    VALUES (NEW.id, 'My Watch List', 'Default watch list', TRUE, 0);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-create default watch list when user signs up
CREATE TRIGGER auto_create_default_watch_list
AFTER INSERT ON users
FOR EACH ROW EXECUTE FUNCTION create_default_watch_list();
```

---

## Backend Implementation

### 1. Data Models

**File:** `backend/models/watchlist.go`

```go
package models

import (
	"time"
)

// WatchList represents a user's watch list
type WatchList struct {
	ID          string     `json:"id" db:"id"`
	UserID      string     `json:"user_id" db:"user_id"`
	Name        string     `json:"name" db:"name"`
	Description *string    `json:"description" db:"description"`
	IsDefault   bool       `json:"is_default" db:"is_default"`
	DisplayOrder int       `json:"display_order" db:"display_order"`
	IsPublic    bool       `json:"is_public" db:"is_public"`
	PublicSlug  *string    `json:"public_slug,omitempty" db:"public_slug"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// WatchListItem represents a ticker in a watch list
type WatchListItem struct {
	ID              string     `json:"id" db:"id"`
	WatchListID     string     `json:"watch_list_id" db:"watch_list_id"`
	Symbol          string     `json:"symbol" db:"symbol"`
	Notes           *string    `json:"notes" db:"notes"`
	Tags            []string   `json:"tags" db:"tags"`
	TargetBuyPrice  *float64   `json:"target_buy_price" db:"target_buy_price"`
	TargetSellPrice *float64   `json:"target_sell_price" db:"target_sell_price"`
	AddedAt         time.Time  `json:"added_at" db:"added_at"`
	DisplayOrder    int        `json:"display_order" db:"display_order"`
}

// WatchListItemWithData includes ticker data and real-time price
type WatchListItemWithData struct {
	WatchListItem
	// Ticker info from stocks table
	Name       string  `json:"name"`
	Exchange   string  `json:"exchange"`
	AssetType  string  `json:"asset_type"`
	LogoURL    *string `json:"logo_url"`
	// Real-time price data
	CurrentPrice    *float64 `json:"current_price"`
	PriceChange     *float64 `json:"price_change"`
	PriceChangePct  *float64 `json:"price_change_pct"`
	Volume          *int64   `json:"volume"`
	MarketCap       *float64 `json:"market_cap"`
	PrevClose       *float64 `json:"prev_close"`
}

// WatchListWithItems includes the watch list and all its items with data
type WatchListWithItems struct {
	WatchList
	ItemCount int                      `json:"item_count"`
	Items     []WatchListItemWithData  `json:"items"`
}

// WatchListSummary is a lightweight version for listing watch lists
type WatchListSummary struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	IsDefault   bool      `json:"is_default"`
	ItemCount   int       `json:"item_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Request/Response DTOs

// CreateWatchListRequest for creating a new watch list
type CreateWatchListRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=255"`
	Description *string `json:"description"`
}

// UpdateWatchListRequest for updating watch list metadata
type UpdateWatchListRequest struct {
	Name        string  `json:"name" binding:"min=1,max=255"`
	Description *string `json:"description"`
}

// AddTickerRequest for adding a ticker to watch list
type AddTickerRequest struct {
	Symbol          string   `json:"symbol" binding:"required"`
	Notes           *string  `json:"notes"`
	Tags            []string `json:"tags"`
	TargetBuyPrice  *float64 `json:"target_buy_price"`
	TargetSellPrice *float64 `json:"target_sell_price"`
}

// UpdateTickerRequest for updating ticker metadata
type UpdateTickerRequest struct {
	Notes           *string  `json:"notes"`
	Tags            []string `json:"tags"`
	TargetBuyPrice  *float64 `json:"target_buy_price"`
	TargetSellPrice *float64 `json:"target_sell_price"`
}

// BulkAddTickersRequest for CSV import
type BulkAddTickersRequest struct {
	Symbols []string `json:"symbols" binding:"required,min=1"`
}

// ReorderItemsRequest for updating display order
type ReorderItemsRequest struct {
	ItemOrders []ItemOrder `json:"item_orders" binding:"required"`
}

type ItemOrder struct {
	ItemID       string `json:"item_id" binding:"required"`
	DisplayOrder int    `json:"display_order" binding:"required"`
}
```

### 2. Database Operations

**File:** `backend/database/watchlists.go`

```go
package database

import (
	"database/sql"
	"errors"
	"fmt"
	"investorcenter/backend/models"
	"strings"

	"github.com/lib/pq"
)

// Watch List Operations

// CreateWatchList creates a new watch list
func CreateWatchList(watchList *models.WatchList) error {
	query := `
		INSERT INTO watch_lists (user_id, name, description, is_default, display_order)
		VALUES ($1, $2, $3, $4, COALESCE((SELECT MAX(display_order) + 1 FROM watch_lists WHERE user_id = $1), 0))
		RETURNING id, created_at, updated_at, display_order
	`
	err := DB.QueryRow(
		query,
		watchList.UserID,
		watchList.Name,
		watchList.Description,
		watchList.IsDefault,
	).Scan(&watchList.ID, &watchList.CreatedAt, &watchList.UpdatedAt, &watchList.DisplayOrder)

	if err != nil {
		return fmt.Errorf("failed to create watch list: %w", err)
	}
	return nil
}

// GetWatchListsByUserID retrieves all watch lists for a user
func GetWatchListsByUserID(userID string) ([]models.WatchListSummary, error) {
	query := `
		SELECT
			wl.id, wl.name, wl.description, wl.is_default, wl.created_at, wl.updated_at,
			COUNT(wli.id) as item_count
		FROM watch_lists wl
		LEFT JOIN watch_list_items wli ON wl.id = wli.watch_list_id
		WHERE wl.user_id = $1
		GROUP BY wl.id, wl.name, wl.description, wl.is_default, wl.created_at, wl.updated_at, wl.display_order
		ORDER BY wl.display_order ASC, wl.created_at ASC
	`
	rows, err := DB.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get watch lists: %w", err)
	}
	defer rows.Close()

	watchLists := []models.WatchListSummary{}
	for rows.Next() {
		var wl models.WatchListSummary
		err := rows.Scan(
			&wl.ID,
			&wl.Name,
			&wl.Description,
			&wl.IsDefault,
			&wl.CreatedAt,
			&wl.UpdatedAt,
			&wl.ItemCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan watch list: %w", err)
		}
		watchLists = append(watchLists, wl)
	}

	return watchLists, nil
}

// GetWatchListByID retrieves a single watch list by ID
func GetWatchListByID(watchListID string, userID string) (*models.WatchList, error) {
	query := `
		SELECT id, user_id, name, description, is_default, display_order, is_public, public_slug, created_at, updated_at
		FROM watch_lists
		WHERE id = $1 AND user_id = $2
	`
	watchList := &models.WatchList{}
	err := DB.QueryRow(query, watchListID, userID).Scan(
		&watchList.ID,
		&watchList.UserID,
		&watchList.Name,
		&watchList.Description,
		&watchList.IsDefault,
		&watchList.DisplayOrder,
		&watchList.IsPublic,
		&watchList.PublicSlug,
		&watchList.CreatedAt,
		&watchList.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("watch list not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get watch list: %w", err)
	}
	return watchList, nil
}

// UpdateWatchList updates watch list metadata
func UpdateWatchList(watchList *models.WatchList) error {
	query := `
		UPDATE watch_lists
		SET name = $1, description = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3 AND user_id = $4
	`
	result, err := DB.Exec(query, watchList.Name, watchList.Description, watchList.ID, watchList.UserID)
	if err != nil {
		return fmt.Errorf("failed to update watch list: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("watch list not found or unauthorized")
	}
	return nil
}

// DeleteWatchList deletes a watch list
func DeleteWatchList(watchListID string, userID string) error {
	query := `DELETE FROM watch_lists WHERE id = $1 AND user_id = $2`
	result, err := DB.Exec(query, watchListID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete watch list: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("watch list not found or unauthorized")
	}
	return nil
}

// Watch List Item Operations

// AddTickerToWatchList adds a ticker to a watch list
func AddTickerToWatchList(item *models.WatchListItem) error {
	// Verify ticker exists in stocks table
	var exists bool
	err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM stocks WHERE symbol = $1)", item.Symbol).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to verify ticker: %w", err)
	}
	if !exists {
		return errors.New("ticker not found in database")
	}

	query := `
		INSERT INTO watch_list_items (watch_list_id, symbol, notes, tags, target_buy_price, target_sell_price, display_order)
		VALUES ($1, $2, $3, $4, $5, $6, COALESCE((SELECT MAX(display_order) + 1 FROM watch_list_items WHERE watch_list_id = $1), 0))
		RETURNING id, added_at, display_order
	`
	err = DB.QueryRow(
		query,
		item.WatchListID,
		item.Symbol,
		item.Notes,
		pq.Array(item.Tags),
		item.TargetBuyPrice,
		item.TargetSellPrice,
	).Scan(&item.ID, &item.AddedAt, &item.DisplayOrder)

	if err != nil {
		// Check for unique constraint violation
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return errors.New("ticker already exists in this watch list")
		}
		// Check for limit trigger
		if strings.Contains(err.Error(), "Watch list limit reached") {
			return errors.New("watch list limit reached. Free tier allows maximum 10 tickers per watch list")
		}
		return fmt.Errorf("failed to add ticker to watch list: %w", err)
	}
	return nil
}

// GetWatchListItems retrieves all items in a watch list
func GetWatchListItems(watchListID string) ([]models.WatchListItem, error) {
	query := `
		SELECT id, watch_list_id, symbol, notes, tags, target_buy_price, target_sell_price, added_at, display_order
		FROM watch_list_items
		WHERE watch_list_id = $1
		ORDER BY display_order ASC, added_at DESC
	`
	rows, err := DB.Query(query, watchListID)
	if err != nil {
		return nil, fmt.Errorf("failed to get watch list items: %w", err)
	}
	defer rows.Close()

	items := []models.WatchListItem{}
	for rows.Next() {
		var item models.WatchListItem
		err := rows.Scan(
			&item.ID,
			&item.WatchListID,
			&item.Symbol,
			&item.Notes,
			pq.Array(&item.Tags),
			&item.TargetBuyPrice,
			&item.TargetSellPrice,
			&item.AddedAt,
			&item.DisplayOrder,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan watch list item: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

// GetWatchListItemsWithData retrieves items with ticker data and real-time prices
func GetWatchListItemsWithData(watchListID string) ([]models.WatchListItemWithData, error) {
	query := `
		SELECT
			wli.id, wli.watch_list_id, wli.symbol, wli.notes, wli.tags,
			wli.target_buy_price, wli.target_sell_price, wli.added_at, wli.display_order,
			s.name, s.exchange, s.asset_type, s.logo_url
		FROM watch_list_items wli
		JOIN stocks s ON wli.symbol = s.symbol
		WHERE wli.watch_list_id = $1
		ORDER BY wli.display_order ASC, wli.added_at DESC
	`
	rows, err := DB.Query(query, watchListID)
	if err != nil {
		return nil, fmt.Errorf("failed to get watch list items: %w", err)
	}
	defer rows.Close()

	items := []models.WatchListItemWithData{}
	for rows.Next() {
		var item models.WatchListItemWithData
		err := rows.Scan(
			&item.ID,
			&item.WatchListID,
			&item.Symbol,
			&item.Notes,
			pq.Array(&item.Tags),
			&item.TargetBuyPrice,
			&item.TargetSellPrice,
			&item.AddedAt,
			&item.DisplayOrder,
			&item.Name,
			&item.Exchange,
			&item.AssetType,
			&item.LogoURL,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan watch list item: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

// UpdateWatchListItem updates ticker metadata
func UpdateWatchListItem(item *models.WatchListItem) error {
	query := `
		UPDATE watch_list_items
		SET notes = $1, tags = $2, target_buy_price = $3, target_sell_price = $4
		WHERE id = $5
	`
	result, err := DB.Exec(
		query,
		item.Notes,
		pq.Array(item.Tags),
		item.TargetBuyPrice,
		item.TargetSellPrice,
		item.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update watch list item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("watch list item not found")
	}
	return nil
}

// RemoveTickerFromWatchList removes a ticker from watch list
func RemoveTickerFromWatchList(watchListID string, symbol string) error {
	query := `DELETE FROM watch_list_items WHERE watch_list_id = $1 AND symbol = $2`
	result, err := DB.Exec(query, watchListID, symbol)
	if err != nil {
		return fmt.Errorf("failed to remove ticker from watch list: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("ticker not found in watch list")
	}
	return nil
}

// BulkAddTickers adds multiple tickers to a watch list
func BulkAddTickers(watchListID string, symbols []string) ([]string, []string, error) {
	added := []string{}
	failed := []string{}

	for _, symbol := range symbols {
		item := &models.WatchListItem{
			WatchListID: watchListID,
			Symbol:      symbol,
			Tags:        []string{},
		}
		err := AddTickerToWatchList(item)
		if err == nil {
			added = append(added, symbol)
		} else {
			failed = append(failed, symbol)
		}
	}

	return added, failed, nil
}

// GetWatchListItemByID retrieves a single watch list item
func GetWatchListItemByID(itemID string) (*models.WatchListItem, error) {
	query := `
		SELECT id, watch_list_id, symbol, notes, tags, target_buy_price, target_sell_price, added_at, display_order
		FROM watch_list_items
		WHERE id = $1
	`
	item := &models.WatchListItem{}
	err := DB.QueryRow(query, itemID).Scan(
		&item.ID,
		&item.WatchListID,
		&item.Symbol,
		&item.Notes,
		pq.Array(&item.Tags),
		&item.TargetBuyPrice,
		&item.TargetSellPrice,
		&item.AddedAt,
		&item.DisplayOrder,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("watch list item not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get watch list item: %w", err)
	}
	return item, nil
}

// UpdateItemDisplayOrder updates display order for items
func UpdateItemDisplayOrder(itemID string, displayOrder int) error {
	query := `UPDATE watch_list_items SET display_order = $1 WHERE id = $2`
	_, err := DB.Exec(query, displayOrder, itemID)
	return err
}
```

### 3. Service Layer

**File:** `backend/services/watchlist_service.go`

```go
package services

import (
	"fmt"
	"investorcenter/backend/database"
	"investorcenter/backend/models"
)

// WatchListService handles business logic for watch lists
type WatchListService struct{}

func NewWatchListService() *WatchListService {
	return &WatchListService{}
}

// GetWatchListWithItems retrieves a watch list with all items and real-time prices
func (s *WatchListService) GetWatchListWithItems(watchListID string, userID string) (*models.WatchListWithItems, error) {
	// Get watch list metadata
	watchList, err := database.GetWatchListByID(watchListID, userID)
	if err != nil {
		return nil, err
	}

	// Get items with ticker data
	items, err := database.GetWatchListItemsWithData(watchListID)
	if err != nil {
		return nil, err
	}

	// Fetch real-time prices for all tickers
	for i := range items {
		item := &items[i]

		// Use existing services to get real-time price
		if item.AssetType == "crypto" || item.AssetType == "cryptocurrency" {
			// Get crypto price from Redis cache
			price, err := GetCachedCryptoPrice(item.Symbol)
			if err == nil && price != nil {
				item.CurrentPrice = &price.Price
				item.PriceChange = &price.Change
				item.PriceChangePct = &price.ChangePercent
			}
		} else {
			// Get stock price from Polygon.io
			polygonClient := NewPolygonClient()
			quote, err := polygonClient.GetRealTimeQuote(item.Symbol)
			if err == nil && quote != nil {
				item.CurrentPrice = &quote.Price
				if quote.PrevClose != nil && *quote.PrevClose > 0 {
					change := quote.Price - *quote.PrevClose
					changePct := (change / *quote.PrevClose) * 100
					item.PriceChange = &change
					item.PriceChangePct = &changePct
					item.PrevClose = quote.PrevClose
				}
				item.Volume = quote.Volume
				item.MarketCap = quote.MarketCap
			}
		}
	}

	result := &models.WatchListWithItems{
		WatchList: *watchList,
		ItemCount: len(items),
		Items:     items,
	}

	return result, nil
}

// ValidateWatchListOwnership checks if user owns the watch list
func (s *WatchListService) ValidateWatchListOwnership(watchListID string, userID string) error {
	watchList, err := database.GetWatchListByID(watchListID, userID)
	if err != nil {
		return err
	}
	if watchList.UserID != userID {
		return fmt.Errorf("unauthorized access to watch list")
	}
	return nil
}
```

### 4. Handlers

**File:** `backend/handlers/watchlist_handlers.go`

```go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"investorcenter/backend/auth"
	"investorcenter/backend/database"
	"investorcenter/backend/models"
	"investorcenter/backend/services"
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

	err := database.CreateWatchList(watchList)
	if err != nil {
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
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
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

// DeleteWatchList deletes a watch list
func DeleteWatchList(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	watchListID := c.Param("id")

	err := database.DeleteWatchList(watchListID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		} else if err.Error() == "watch list limit reached. Free tier allows maximum 10 tickers per watch list" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add ticker"})
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
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update item order"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Items reordered successfully"})
}
```

### 5. Update main.go with Watch List Routes

**File:** `backend/main.go` (add these routes)

```go
// Watch List routes (protected, require authentication)
watchListRoutes := v1.Group("/watchlists")
watchListRoutes.Use(auth.AuthMiddleware())
{
	watchListRoutes.GET("", handlers.ListWatchLists)                           // GET /api/v1/watchlists
	watchListRoutes.POST("", handlers.CreateWatchList)                         // POST /api/v1/watchlists
	watchListRoutes.GET("/:id", handlers.GetWatchList)                         // GET /api/v1/watchlists/:id
	watchListRoutes.PUT("/:id", handlers.UpdateWatchList)                      // PUT /api/v1/watchlists/:id
	watchListRoutes.DELETE("/:id", handlers.DeleteWatchList)                   // DELETE /api/v1/watchlists/:id

	// Watch list items
	watchListRoutes.POST("/:id/items", handlers.AddTickerToWatchList)          // POST /api/v1/watchlists/:id/items
	watchListRoutes.DELETE("/:id/items/:symbol", handlers.RemoveTickerFromWatchList) // DELETE /api/v1/watchlists/:id/items/:symbol
	watchListRoutes.PUT("/:id/items/:symbol", handlers.UpdateWatchListItem)    // PUT /api/v1/watchlists/:id/items/:symbol
	watchListRoutes.POST("/:id/bulk", handlers.BulkAddTickers)                 // POST /api/v1/watchlists/:id/bulk
	watchListRoutes.POST("/:id/reorder", handlers.ReorderWatchListItems)       // POST /api/v1/watchlists/:id/reorder
}
```

---

## Frontend Implementation

### 1. API Client

**File:** `lib/api/watchlist.ts`

```typescript
import { apiClient } from './client';

export interface WatchList {
  id: string;
  user_id: string;
  name: string;
  description?: string;
  is_default: boolean;
  item_count: number;
  created_at: string;
  updated_at: string;
}

export interface WatchListItem {
  id: string;
  watch_list_id: string;
  symbol: string;
  notes?: string;
  tags: string[];
  target_buy_price?: number;
  target_sell_price?: number;
  added_at: string;
  display_order: number;
  // Ticker data
  name: string;
  exchange: string;
  asset_type: string;
  logo_url?: string;
  // Real-time price data
  current_price?: number;
  price_change?: number;
  price_change_pct?: number;
  volume?: number;
  market_cap?: number;
  prev_close?: number;
}

export interface WatchListWithItems {
  id: string;
  name: string;
  description?: string;
  is_default: boolean;
  item_count: number;
  items: WatchListItem[];
  created_at: string;
  updated_at: string;
}

export const watchListAPI = {
  // Get all watch lists for user
  async getWatchLists(): Promise<{ watch_lists: WatchList[] }> {
    return apiClient.get('/watchlists');
  },

  // Create new watch list
  async createWatchList(data: { name: string; description?: string }): Promise<WatchList> {
    return apiClient.post('/watchlists', data);
  },

  // Get single watch list with items
  async getWatchList(id: string): Promise<WatchListWithItems> {
    return apiClient.get(`/watchlists/${id}`);
  },

  // Update watch list metadata
  async updateWatchList(id: string, data: { name: string; description?: string }): Promise<void> {
    return apiClient.put(`/watchlists/${id}`, data);
  },

  // Delete watch list
  async deleteWatchList(id: string): Promise<void> {
    return apiClient.delete(`/watchlists/${id}`);
  },

  // Add ticker to watch list
  async addTicker(watchListId: string, data: {
    symbol: string;
    notes?: string;
    tags?: string[];
    target_buy_price?: number;
    target_sell_price?: number;
  }): Promise<WatchListItem> {
    return apiClient.post(`/watchlists/${watchListId}/items`, data);
  },

  // Remove ticker from watch list
  async removeTicker(watchListId: string, symbol: string): Promise<void> {
    return apiClient.delete(`/watchlists/${watchListId}/items/${symbol}`);
  },

  // Update ticker metadata
  async updateTicker(watchListId: string, symbol: string, data: {
    notes?: string;
    tags?: string[];
    target_buy_price?: number;
    target_sell_price?: number;
  }): Promise<WatchListItem> {
    return apiClient.put(`/watchlists/${watchListId}/items/${symbol}`, data);
  },

  // Bulk add tickers
  async bulkAddTickers(watchListId: string, symbols: string[]): Promise<{
    added: string[];
    failed: string[];
    total: number;
  }> {
    return apiClient.post(`/watchlists/${watchListId}/bulk`, { symbols });
  },

  // Reorder items
  async reorderItems(watchListId: string, itemOrders: Array<{ item_id: string; display_order: number }>): Promise<void> {
    return apiClient.post(`/watchlists/${watchListId}/reorder`, { item_orders: itemOrders });
  },
};
```

**File:** `lib/api/client.ts` (update with auth headers)

```typescript
const getAuthToken = (): string | null => {
  if (typeof window !== 'undefined') {
    return localStorage.getItem('access_token');
  }
  return null;
};

export const apiClient = {
  async get<T>(endpoint: string): Promise<T> {
    const token = getAuthToken();
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    };
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}${endpoint}`, {
      method: 'GET',
      headers,
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Request failed');
    }

    return response.json();
  },

  async post<T>(endpoint: string, data: any): Promise<T> {
    const token = getAuthToken();
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    };
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}${endpoint}`, {
      method: 'POST',
      headers,
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Request failed');
    }

    return response.json();
  },

  async put<T>(endpoint: string, data: any): Promise<T> {
    const token = getAuthToken();
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    };
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}${endpoint}`, {
      method: 'PUT',
      headers,
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Request failed');
    }

    return response.json();
  },

  async delete<T>(endpoint: string): Promise<T> {
    const token = getAuthToken();
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    };
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}${endpoint}`, {
      method: 'DELETE',
      headers,
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Request failed');
    }

    return response.json();
  },
};
```

### 2. Watch List Dashboard Page

**File:** `app/watchlist/page.tsx`

```typescript
'use client';

import { useEffect, useState } from 'react';
import { useAuth } from '@/lib/auth/AuthContext';
import { watchListAPI, WatchList } from '@/lib/api/watchlist';
import { ProtectedRoute } from '@/components/auth/ProtectedRoute';
import Link from 'next/link';
import CreateWatchListModal from '@/components/watchlist/CreateWatchListModal';

export default function WatchListDashboard() {
  const { user } = useAuth();
  const [watchLists, setWatchLists] = useState<WatchList[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    loadWatchLists();
  }, []);

  const loadWatchLists = async () => {
    try {
      const data = await watchListAPI.getWatchLists();
      setWatchLists(data.watch_lists);
    } catch (err: any) {
      setError(err.message || 'Failed to load watch lists');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateWatchList = async (name: string, description?: string) => {
    try {
      await watchListAPI.createWatchList({ name, description });
      await loadWatchLists();
      setShowCreateModal(false);
    } catch (err: any) {
      setError(err.message || 'Failed to create watch list');
    }
  };

  const handleDeleteWatchList = async (id: string) => {
    if (!confirm('Are you sure you want to delete this watch list?')) return;

    try {
      await watchListAPI.deleteWatchList(id);
      await loadWatchLists();
    } catch (err: any) {
      setError(err.message || 'Failed to delete watch list');
    }
  };

  return (
    <ProtectedRoute>
      <div className="container mx-auto px-4 py-8">
        <div className="flex justify-between items-center mb-8">
          <h1 className="text-3xl font-bold">My Watch Lists</h1>
          <button
            onClick={() => setShowCreateModal(true)}
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
          >
            + Create Watch List
          </button>
        </div>

        {error && (
          <div className="mb-4 p-3 bg-red-100 border border-red-400 text-red-700 rounded">
            {error}
          </div>
        )}

        {loading ? (
          <div className="text-center py-12">Loading...</div>
        ) : watchLists.length === 0 ? (
          <div className="text-center py-12">
            <p className="text-gray-600 mb-4">You don't have any watch lists yet.</p>
            <button
              onClick={() => setShowCreateModal(true)}
              className="px-6 py-3 bg-blue-600 text-white rounded hover:bg-blue-700"
            >
              Create Your First Watch List
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {watchLists.map((watchList) => (
              <div key={watchList.id} className="bg-white p-6 rounded-lg shadow hover:shadow-lg transition">
                <div className="flex justify-between items-start mb-3">
                  <h3 className="text-xl font-semibold">{watchList.name}</h3>
                  {watchList.is_default && (
                    <span className="text-xs bg-blue-100 text-blue-800 px-2 py-1 rounded">Default</span>
                  )}
                </div>

                {watchList.description && (
                  <p className="text-gray-600 text-sm mb-4">{watchList.description}</p>
                )}

                <div className="flex items-center justify-between mb-4">
                  <span className="text-gray-500">{watchList.item_count} tickers</span>
                  <span className="text-xs text-gray-400">
                    {new Date(watchList.updated_at).toLocaleDateString()}
                  </span>
                </div>

                <div className="flex gap-2">
                  <Link
                    href={`/watchlist/${watchList.id}`}
                    className="flex-1 text-center px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
                  >
                    View
                  </Link>
                  <button
                    onClick={() => handleDeleteWatchList(watchList.id)}
                    className="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700"
                  >
                    Delete
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}

        {showCreateModal && (
          <CreateWatchListModal
            onClose={() => setShowCreateModal(false)}
            onCreate={handleCreateWatchList}
          />
        )}
      </div>
    </ProtectedRoute>
  );
}
```

### 3. Watch List Detail Page

**File:** `app/watchlist/[id]/page.tsx`

```typescript
'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { watchListAPI, WatchListWithItems } from '@/lib/api/watchlist';
import { ProtectedRoute } from '@/components/auth/ProtectedRoute';
import WatchListTable from '@/components/watchlist/WatchListTable';
import AddTickerModal from '@/components/watchlist/AddTickerModal';
import EditTickerModal from '@/components/watchlist/EditTickerModal';

export default function WatchListDetailPage() {
  const params = useParams();
  const watchListId = params.id as string;

  const [watchList, setWatchList] = useState<WatchListWithItems | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showAddModal, setShowAddModal] = useState(false);
  const [editingSymbol, setEditingSymbol] = useState<string | null>(null);

  useEffect(() => {
    loadWatchList();
    // Set up auto-refresh for real-time prices
    const interval = setInterval(loadWatchList, 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, [watchListId]);

  const loadWatchList = async () => {
    try {
      const data = await watchListAPI.getWatchList(watchListId);
      setWatchList(data);
    } catch (err: any) {
      setError(err.message || 'Failed to load watch list');
    } finally {
      setLoading(false);
    }
  };

  const handleAddTicker = async (symbol: string, notes?: string, tags?: string[], targetBuy?: number, targetSell?: number) => {
    try {
      await watchListAPI.addTicker(watchListId, {
        symbol,
        notes,
        tags,
        target_buy_price: targetBuy,
        target_sell_price: targetSell,
      });
      await loadWatchList();
      setShowAddModal(false);
    } catch (err: any) {
      alert(err.message || 'Failed to add ticker');
    }
  };

  const handleRemoveTicker = async (symbol: string) => {
    if (!confirm(`Remove ${symbol} from watch list?`)) return;

    try {
      await watchListAPI.removeTicker(watchListId, symbol);
      await loadWatchList();
    } catch (err: any) {
      alert(err.message || 'Failed to remove ticker');
    }
  };

  const handleUpdateTicker = async (symbol: string, data: any) => {
    try {
      await watchListAPI.updateTicker(watchListId, symbol, data);
      await loadWatchList();
      setEditingSymbol(null);
    } catch (err: any) {
      alert(err.message || 'Failed to update ticker');
    }
  };

  if (loading) {
    return (
      <ProtectedRoute>
        <div className="flex items-center justify-center min-h-screen">
          <div className="text-xl">Loading...</div>
        </div>
      </ProtectedRoute>
    );
  }

  if (!watchList) {
    return (
      <ProtectedRoute>
        <div className="container mx-auto px-4 py-8">
          <div className="text-center">
            <p className="text-red-600">{error || 'Watch list not found'}</p>
          </div>
        </div>
      </ProtectedRoute>
    );
  }

  return (
    <ProtectedRoute>
      <div className="container mx-auto px-4 py-8">
        <div className="flex justify-between items-center mb-6">
          <div>
            <h1 className="text-3xl font-bold">{watchList.name}</h1>
            {watchList.description && (
              <p className="text-gray-600 mt-2">{watchList.description}</p>
            )}
            <p className="text-sm text-gray-500 mt-1">{watchList.item_count} tickers</p>
          </div>
          <button
            onClick={() => setShowAddModal(true)}
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
          >
            + Add Ticker
          </button>
        </div>

        {error && (
          <div className="mb-4 p-3 bg-red-100 border border-red-400 text-red-700 rounded">
            {error}
          </div>
        )}

        {watchList.items.length === 0 ? (
          <div className="text-center py-12 bg-gray-50 rounded-lg">
            <p className="text-gray-600 mb-4">No tickers in this watch list yet.</p>
            <button
              onClick={() => setShowAddModal(true)}
              className="px-6 py-3 bg-blue-600 text-white rounded hover:bg-blue-700"
            >
              Add Your First Ticker
            </button>
          </div>
        ) : (
          <WatchListTable
            items={watchList.items}
            onRemove={handleRemoveTicker}
            onEdit={setEditingSymbol}
          />
        )}

        {showAddModal && (
          <AddTickerModal
            onClose={() => setShowAddModal(false)}
            onAdd={handleAddTicker}
          />
        )}

        {editingSymbol && (
          <EditTickerModal
            symbol={editingSymbol}
            item={watchList.items.find(i => i.symbol === editingSymbol)!}
            onClose={() => setEditingSymbol(null)}
            onUpdate={handleUpdateTicker}
          />
        )}
      </div>
    </ProtectedRoute>
  );
}
```

### 4. Components

**File:** `components/watchlist/CreateWatchListModal.tsx`

```typescript
'use client';

import { useState } from 'react';

interface CreateWatchListModalProps {
  onClose: () => void;
  onCreate: (name: string, description?: string) => Promise<void>;
}

export default function CreateWatchListModal({ onClose, onCreate }: CreateWatchListModalProps) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await onCreate(name, description || undefined);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 w-full max-w-md">
        <h2 className="text-2xl font-bold mb-4">Create Watch List</h2>

        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label className="block text-sm font-medium mb-2" htmlFor="name">
              Name *
            </label>
            <input
              id="name"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
              maxLength={255}
            />
          </div>

          <div className="mb-6">
            <label className="block text-sm font-medium mb-2" htmlFor="description">
              Description (optional)
            </label>
            <textarea
              id="description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              rows={3}
            />
          </div>

          <div className="flex gap-2 justify-end">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border border-gray-300 rounded hover:bg-gray-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading || !name}
              className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed"
            >
              {loading ? 'Creating...' : 'Create'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
```

**File:** `components/watchlist/WatchListTable.tsx`

```typescript
'use client';

import Link from 'next/link';
import { WatchListItem } from '@/lib/api/watchlist';

interface WatchListTableProps {
  items: WatchListItem[];
  onRemove: (symbol: string) => void;
  onEdit: (symbol: string) => void;
}

export default function WatchListTable({ items, onRemove, onEdit }: WatchListTableProps) {
  const formatPrice = (price?: number) => {
    if (price === undefined || price === null) return '-';
    return `$${price.toFixed(2)}`;
  };

  const formatChange = (change?: number, changePct?: number) => {
    if (change === undefined || changePct === undefined) return '-';
    const color = change >= 0 ? 'text-green-600' : 'text-red-600';
    const sign = change >= 0 ? '+' : '';
    return (
      <span className={color}>
        {sign}{change.toFixed(2)} ({sign}{changePct.toFixed(2)}%)
      </span>
    );
  };

  return (
    <div className="overflow-x-auto">
      <table className="w-full bg-white rounded-lg shadow">
        <thead className="bg-gray-50">
          <tr>
            <th className="px-4 py-3 text-left text-sm font-semibold">Symbol</th>
            <th className="px-4 py-3 text-left text-sm font-semibold">Name</th>
            <th className="px-4 py-3 text-right text-sm font-semibold">Price</th>
            <th className="px-4 py-3 text-right text-sm font-semibold">Change</th>
            <th className="px-4 py-3 text-right text-sm font-semibold">Target Buy</th>
            <th className="px-4 py-3 text-right text-sm font-semibold">Target Sell</th>
            <th className="px-4 py-3 text-center text-sm font-semibold">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y">
          {items.map((item) => (
            <tr key={item.symbol} className="hover:bg-gray-50">
              <td className="px-4 py-3">
                <Link href={`/ticker/${item.symbol}`} className="text-blue-600 hover:underline font-medium">
                  {item.symbol}
                </Link>
              </td>
              <td className="px-4 py-3 text-sm">{item.name}</td>
              <td className="px-4 py-3 text-right font-medium">{formatPrice(item.current_price)}</td>
              <td className="px-4 py-3 text-right">{formatChange(item.price_change, item.price_change_pct)}</td>
              <td className="px-4 py-3 text-right text-sm">{formatPrice(item.target_buy_price)}</td>
              <td className="px-4 py-3 text-right text-sm">{formatPrice(item.target_sell_price)}</td>
              <td className="px-4 py-3 text-center">
                <button
                  onClick={() => onEdit(item.symbol)}
                  className="text-blue-600 hover:underline text-sm mr-3"
                >
                  Edit
                </button>
                <button
                  onClick={() => onRemove(item.symbol)}
                  className="text-red-600 hover:underline text-sm"
                >
                  Remove
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
```

**File:** `components/watchlist/AddTickerModal.tsx`

```typescript
'use client';

import { useState } from 'react';

interface AddTickerModalProps {
  onClose: () => void;
  onAdd: (symbol: string, notes?: string, tags?: string[], targetBuy?: number, targetSell?: number) => Promise<void>;
}

export default function AddTickerModal({ onClose, onAdd }: AddTickerModalProps) {
  const [symbol, setSymbol] = useState('');
  const [notes, setNotes] = useState('');
  const [tags, setTags] = useState('');
  const [targetBuy, setTargetBuy] = useState('');
  const [targetSell, setTargetSell] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      const tagArray = tags.split(',').map(t => t.trim()).filter(t => t);
      await onAdd(
        symbol.toUpperCase(),
        notes || undefined,
        tagArray.length > 0 ? tagArray : undefined,
        targetBuy ? parseFloat(targetBuy) : undefined,
        targetSell ? parseFloat(targetSell) : undefined
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 w-full max-w-md">
        <h2 className="text-2xl font-bold mb-4">Add Ticker</h2>

        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label className="block text-sm font-medium mb-2" htmlFor="symbol">
              Symbol *
            </label>
            <input
              id="symbol"
              type="text"
              value={symbol}
              onChange={(e) => setSymbol(e.target.value)}
              placeholder="e.g., AAPL, X:BTCUSD"
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-2" htmlFor="notes">
              Notes
            </label>
            <textarea
              id="notes"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              rows={2}
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-2" htmlFor="tags">
              Tags (comma-separated)
            </label>
            <input
              id="tags"
              type="text"
              value={tags}
              onChange={(e) => setTags(e.target.value)}
              placeholder="e.g., tech, growth"
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div className="grid grid-cols-2 gap-4 mb-6">
            <div>
              <label className="block text-sm font-medium mb-2" htmlFor="targetBuy">
                Target Buy Price
              </label>
              <input
                id="targetBuy"
                type="number"
                step="0.01"
                value={targetBuy}
                onChange={(e) => setTargetBuy(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-2" htmlFor="targetSell">
                Target Sell Price
              </label>
              <input
                id="targetSell"
                type="number"
                step="0.01"
                value={targetSell}
                onChange={(e) => setTargetSell(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
          </div>

          <div className="flex gap-2 justify-end">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border border-gray-300 rounded hover:bg-gray-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading || !symbol}
              className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed"
            >
              {loading ? 'Adding...' : 'Add'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
```

**File:** `components/watchlist/EditTickerModal.tsx`

```typescript
'use client';

import { useState } from 'react';
import { WatchListItem } from '@/lib/api/watchlist';

interface EditTickerModalProps {
  symbol: string;
  item: WatchListItem;
  onClose: () => void;
  onUpdate: (symbol: string, data: any) => Promise<void>;
}

export default function EditTickerModal({ symbol, item, onClose, onUpdate }: EditTickerModalProps) {
  const [notes, setNotes] = useState(item.notes || '');
  const [tags, setTags] = useState(item.tags.join(', '));
  const [targetBuy, setTargetBuy] = useState(item.target_buy_price?.toString() || '');
  const [targetSell, setTargetSell] = useState(item.target_sell_price?.toString() || '');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      const tagArray = tags.split(',').map(t => t.trim()).filter(t => t);
      await onUpdate(symbol, {
        notes: notes || undefined,
        tags: tagArray,
        target_buy_price: targetBuy ? parseFloat(targetBuy) : undefined,
        target_sell_price: targetSell ? parseFloat(targetSell) : undefined,
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 w-full max-w-md">
        <h2 className="text-2xl font-bold mb-4">Edit {symbol}</h2>

        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label className="block text-sm font-medium mb-2" htmlFor="notes">
              Notes
            </label>
            <textarea
              id="notes"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              rows={3}
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-2" htmlFor="tags">
              Tags (comma-separated)
            </label>
            <input
              id="tags"
              type="text"
              value={tags}
              onChange={(e) => setTags(e.target.value)}
              placeholder="e.g., tech, growth"
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div className="grid grid-cols-2 gap-4 mb-6">
            <div>
              <label className="block text-sm font-medium mb-2" htmlFor="targetBuy">
                Target Buy Price
              </label>
              <input
                id="targetBuy"
                type="number"
                step="0.01"
                value={targetBuy}
                onChange={(e) => setTargetBuy(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-2" htmlFor="targetSell">
                Target Sell Price
              </label>
              <input
                id="targetSell"
                type="number"
                step="0.01"
                value={targetSell}
                onChange={(e) => setTargetSell(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
          </div>

          <div className="flex gap-2 justify-end">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border border-gray-300 rounded hover:bg-gray-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed"
            >
              {loading ? 'Saving...' : 'Save'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
```

---

## Testing Plan

### Backend Testing

**Manual Testing Checklist:**

- [ ] Create watch list with valid data
- [ ] Create watch list with empty name (should fail)
- [ ] Get all watch lists for user
- [ ] Get single watch list by ID
- [ ] Get watch list for different user (should fail - unauthorized)
- [ ] Update watch list name and description
- [ ] Delete watch list
- [ ] Add ticker to watch list (valid symbol)
- [ ] Add ticker with invalid symbol (should fail)
- [ ] Add duplicate ticker to same watch list (should fail)
- [ ] Add 11th ticker to free tier watch list (should fail with limit error)
- [ ] Remove ticker from watch list
- [ ] Update ticker metadata (notes, tags, targets)
- [ ] Bulk add multiple tickers (mix of valid and invalid)
- [ ] Get watch list with items - verify real-time prices populated
- [ ] Verify default watch list auto-created for new user
- [ ] Test with crypto symbols (X:BTCUSD) - verify price from Redis
- [ ] Test with stock symbols (AAPL) - verify price from Polygon
- [ ] Reorder watch list items

### Frontend Testing

**Manual Testing Checklist:**

- [ ] Watch list dashboard loads and displays all watch lists
- [ ] Create watch list modal opens and closes
- [ ] Create watch list with valid data
- [ ] Create watch list form validation
- [ ] Watch list cards display correctly with item count
- [ ] Click "View" navigates to watch list detail page
- [ ] Delete watch list works and updates list
- [ ] Watch list detail page loads with items
- [ ] Real-time prices update every 30 seconds
- [ ] Add ticker modal opens and closes
- [ ] Add ticker form validation
- [ ] Add ticker with autocomplete search (future enhancement)
- [ ] Edit ticker modal opens with pre-filled data
- [ ] Update ticker metadata and verify changes
- [ ] Remove ticker from watch list
- [ ] Table sorts by columns (future enhancement)
- [ ] Price change colors (green/red) display correctly
- [ ] Target price alerts highlighted when price crosses target (future)
- [ ] Empty state displays when watch list has no tickers
- [ ] Error handling for API failures
- [ ] Loading states display correctly

---

## Deployment Checklist

### Database

- [ ] Run migration `migrations/007_watchlist_tables.sql` on production
- [ ] Verify tables created: watch_lists, watch_list_items
- [ ] Verify indexes created
- [ ] Verify triggers created (limit enforcement, auto default watch list)
- [ ] Test trigger: create new user, verify default watch list auto-created

### Backend

- [ ] Build Go binary with watch list packages
- [ ] Update Kubernetes deployment (no new env vars needed)
- [ ] Deploy to EKS cluster
- [ ] Test health check endpoint
- [ ] Test watch list endpoints with Postman/curl
- [ ] Verify real-time price integration works (Polygon + Redis)

### Frontend

- [ ] Build Next.js app (`npm run build`)
- [ ] Deploy to production
- [ ] Test watch list pages
- [ ] Verify real-time price updates
- [ ] Test on mobile devices

### Monitoring

- [ ] Log watch list operations (create, delete, add/remove tickers)
- [ ] Track watch list creation rate
- [ ] Monitor free tier limit enforcement
- [ ] Track API error rates for watch list endpoints

---

## Success Criteria

Phase 2 is considered complete when:

- [x] Database schema migrated with watch_lists and watch_list_items tables
- [x] Users can create multiple watch lists
- [x] Users can add tickers to watch lists (stocks and crypto)
- [x] Users can remove tickers from watch lists
- [x] Users can update ticker metadata (notes, tags, target prices)
- [x] Users can delete watch lists
- [x] Watch list table view displays real-time prices
- [x] Real-time prices refresh automatically
- [x] Bulk import from CSV works
- [x] Free tier limit enforced (max 10 tickers per watch list)
- [x] Default watch list auto-created for new users
- [x] All manual tests pass
- [x] Deployed to production and tested

---

## Next Steps (Phase 3)

After Phase 2 is complete and tested, we'll move to Phase 3:

- Heatmap visualization component (D3.js or recharts)
- Heatmap configuration options (size metric, color metric, time period)
- Save/load custom heatmap configurations
- Interactive heatmap (hover tooltips, click to ticker detail)
- Export heatmap as PNG
- Full-screen heatmap view

**Estimated Timeline:** Phase 2 completion by Day 20, ready to start Phase 3.
