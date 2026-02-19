package models

import (
	"time"
)

// WatchList represents a user's watch list
type WatchList struct {
	ID           string    `json:"id" db:"id"`
	UserID       string    `json:"user_id" db:"user_id"`
	Name         string    `json:"name" db:"name"`
	Description  *string   `json:"description" db:"description"`
	IsDefault    bool      `json:"is_default" db:"is_default"`
	DisplayOrder int       `json:"display_order" db:"display_order"`
	IsPublic     bool      `json:"is_public" db:"is_public"`
	PublicSlug   *string   `json:"public_slug,omitempty" db:"public_slug"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// WatchListItem represents a ticker in a watch list
type WatchListItem struct {
	ID              string    `json:"id" db:"id"`
	WatchListID     string    `json:"watch_list_id" db:"watch_list_id"`
	Symbol          string    `json:"symbol" db:"symbol"`
	Notes           *string   `json:"notes" db:"notes"`
	Tags            []string  `json:"tags" db:"tags"`
	TargetBuyPrice  *float64  `json:"target_buy_price" db:"target_buy_price"`
	TargetSellPrice *float64  `json:"target_sell_price" db:"target_sell_price"`
	AddedAt         time.Time `json:"added_at" db:"added_at"`
	DisplayOrder    int       `json:"display_order" db:"display_order"`
}

// WatchListItemWithData includes ticker data and real-time price
type WatchListItemWithData struct {
	WatchListItem
	// Ticker info from tickers table
	Name      string  `json:"name"`
	Exchange  string  `json:"exchange"`
	AssetType string  `json:"asset_type"`
	LogoURL   *string `json:"logo_url"`
	// Real-time price data
	CurrentPrice   *float64 `json:"current_price"`
	PriceChange    *float64 `json:"price_change"`
	PriceChangePct *float64 `json:"price_change_pct"`
	Volume         *int64   `json:"volume"`
	MarketCap      *float64 `json:"market_cap"`
	PrevClose      *float64 `json:"prev_close"`
	// Reddit data (from reddit_heatmap_daily table)
	RedditRank       *int     `json:"reddit_rank,omitempty"`        // Current Reddit rank (1 = #1 trending)
	RedditMentions   *int     `json:"reddit_mentions,omitempty"`    // Total mentions
	RedditPopularity *float64 `json:"reddit_popularity,omitempty"`  // Popularity score (0-100)
	RedditTrend      *string  `json:"reddit_trend,omitempty"`       // "rising", "falling", "stable"
	RedditRankChange *int     `json:"reddit_rank_change,omitempty"` // Rank change vs 24h ago
}

// WatchListWithItems includes the watch list and all its items with data
type WatchListWithItems struct {
	WatchList
	ItemCount int                     `json:"item_count"`
	Items     []WatchListItemWithData `json:"items"`
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
	Description *string `json:"description" binding:"omitempty,max=5000"`
}

// UpdateWatchListRequest for updating watch list metadata
type UpdateWatchListRequest struct {
	Name        string  `json:"name" binding:"min=1,max=255"`
	Description *string `json:"description" binding:"omitempty,max=5000"`
}

// AddTickerRequest for adding a ticker to watch list
type AddTickerRequest struct {
	Symbol          string   `json:"symbol" binding:"required,min=1,max=20"`
	Notes           *string  `json:"notes" binding:"omitempty,max=10000"`
	Tags            []string `json:"tags" binding:"max=50,dive,max=100"`
	TargetBuyPrice  *float64 `json:"target_buy_price" binding:"omitempty,gte=0"`
	TargetSellPrice *float64 `json:"target_sell_price" binding:"omitempty,gte=0"`
}

// UpdateTickerRequest for updating ticker metadata
type UpdateTickerRequest struct {
	Notes           *string  `json:"notes" binding:"omitempty,max=10000"`
	Tags            []string `json:"tags" binding:"max=50,dive,max=100"`
	TargetBuyPrice  *float64 `json:"target_buy_price" binding:"omitempty,gte=0"`
	TargetSellPrice *float64 `json:"target_sell_price" binding:"omitempty,gte=0"`
}

// BulkAddTickersRequest for CSV import
type BulkAddTickersRequest struct {
	Symbols []string `json:"symbols" binding:"required,min=1,max=500,dive,min=1,max=20"`
}

// ReorderItemsRequest for updating display order
type ReorderItemsRequest struct {
	ItemOrders []ItemOrder `json:"item_orders" binding:"required,min=1,max=500"`
}

type ItemOrder struct {
	ItemID       string `json:"item_id" binding:"required,max=100"`
	DisplayOrder int    `json:"display_order" binding:"min=0,max=10000"`
}
