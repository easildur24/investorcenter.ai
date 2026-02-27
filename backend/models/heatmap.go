package models

import (
	"time"
)

// HeatmapConfig represents a saved heatmap configuration
type HeatmapConfig struct {
	ID                string                 `json:"id" db:"id"`
	UserID            string                 `json:"user_id" db:"user_id"`
	WatchListID       string                 `json:"watch_list_id" db:"watch_list_id"`
	Name              string                 `json:"name" db:"name"`
	SizeMetric        string                 `json:"size_metric" db:"size_metric"`
	ColorMetric       string                 `json:"color_metric" db:"color_metric"`
	TimePeriod        string                 `json:"time_period" db:"time_period"`
	ColorScheme       string                 `json:"color_scheme" db:"color_scheme"`
	LabelDisplay      string                 `json:"label_display" db:"label_display"`
	LayoutType        string                 `json:"layout_type" db:"layout_type"`
	FiltersJSON       map[string]interface{} `json:"filters" db:"filters_json"`
	ColorGradientJSON map[string]string      `json:"color_gradient,omitempty" db:"color_gradient_json"`
	IsDefault         bool                   `json:"is_default" db:"is_default"`
	CreatedAt         time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at" db:"updated_at"`
}

// HeatmapTile represents a single tile in the heatmap
type HeatmapTile struct {
	Symbol    string `json:"symbol"`
	Name      string `json:"name"`
	AssetType string `json:"asset_type"`
	Sector    string `json:"sector"`

	// Size value (what determines tile size)
	SizeValue float64 `json:"size_value"`
	SizeLabel string  `json:"size_label"` // e.g., "$1.2B", "15.3M shares"

	// Color value (what determines tile color)
	ColorValue float64 `json:"color_value"`
	ColorLabel string  `json:"color_label"` // e.g., "+5.2%", "-2.1%"

	// Current price info
	CurrentPrice   float64 `json:"current_price"`
	PriceChange    float64 `json:"price_change"`
	PriceChangePct float64 `json:"price_change_pct"`

	// Additional metadata for tooltip
	Volume    *int64   `json:"volume,omitempty"`
	MarketCap *float64 `json:"market_cap,omitempty"`
	PrevClose *float64 `json:"prev_close,omitempty"`
	Exchange  string   `json:"exchange"`

	// Reddit data (from reddit_heatmap_daily table)
	RedditRank       *int     `json:"reddit_rank,omitempty"`        // Current Reddit rank (1 = #1 trending)
	RedditMentions   *int     `json:"reddit_mentions,omitempty"`    // Total mentions
	RedditPopularity *float64 `json:"reddit_popularity,omitempty"`  // Popularity score (0-100)
	RedditTrend      *string  `json:"reddit_trend,omitempty"`       // "rising", "falling", "stable"
	RedditRankChange *int     `json:"reddit_rank_change,omitempty"` // Rank change vs 24h ago

	// User's custom data from watch list
	Notes           *string  `json:"notes,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	TargetBuyPrice  *float64 `json:"target_buy_price,omitempty"`
	TargetSellPrice *float64 `json:"target_sell_price,omitempty"`
}

// HeatmapData represents the complete heatmap data
type HeatmapData struct {
	WatchListID   string `json:"watch_list_id"`
	WatchListName string `json:"watch_list_name"`
	ConfigID      string `json:"config_id,omitempty"`
	ConfigName    string `json:"config_name,omitempty"`

	// Configuration used
	SizeMetric  string `json:"size_metric"`
	ColorMetric string `json:"color_metric"`
	TimePeriod  string `json:"time_period"`
	ColorScheme string `json:"color_scheme"`

	// Tiles data
	Tiles     []HeatmapTile `json:"tiles"`
	TileCount int           `json:"tile_count"`

	// Metadata for color scale
	MinColorValue float64 `json:"min_color_value"`
	MaxColorValue float64 `json:"max_color_value"`

	// Timestamp
	GeneratedAt time.Time `json:"generated_at"`
}

// Request/Response DTOs

// CreateHeatmapConfigRequest for saving a new config
type CreateHeatmapConfigRequest struct {
	WatchListID   string                 `json:"watch_list_id" binding:"required"`
	Name          string                 `json:"name" binding:"required,min=1,max=255"`
	SizeMetric    string                 `json:"size_metric" binding:"required,oneof=market_cap volume avg_volume reddit_mentions reddit_popularity"`
	ColorMetric   string                 `json:"color_metric" binding:"required,oneof=price_change_pct volume_change_pct reddit_rank reddit_trend"`
	TimePeriod    string                 `json:"time_period" binding:"required,oneof=1D 1W 1M 3M 6M YTD 1Y 5Y"`
	ColorScheme   string                 `json:"color_scheme" binding:"oneof=red_green heatmap blue_red custom"`
	LabelDisplay  string                 `json:"label_display" binding:"oneof=symbol symbol_change full"`
	LayoutType    string                 `json:"layout_type" binding:"oneof=treemap grid"`
	Filters       map[string]interface{} `json:"filters"`
	ColorGradient map[string]string      `json:"color_gradient,omitempty"`
	IsDefault     bool                   `json:"is_default"`
}

// UpdateHeatmapConfigRequest for updating existing config
type UpdateHeatmapConfigRequest struct {
	Name          string                 `json:"name" binding:"min=1,max=255"`
	SizeMetric    string                 `json:"size_metric" binding:"oneof=market_cap volume avg_volume reddit_mentions reddit_popularity"`
	ColorMetric   string                 `json:"color_metric" binding:"oneof=price_change_pct volume_change_pct reddit_rank reddit_trend"`
	TimePeriod    string                 `json:"time_period" binding:"oneof=1D 1W 1M 3M 6M YTD 1Y 5Y"`
	ColorScheme   string                 `json:"color_scheme" binding:"oneof=red_green heatmap blue_red custom"`
	LabelDisplay  string                 `json:"label_display" binding:"oneof=symbol symbol_change full"`
	LayoutType    string                 `json:"layout_type" binding:"oneof=treemap grid"`
	Filters       map[string]interface{} `json:"filters"`
	ColorGradient map[string]string      `json:"color_gradient,omitempty"`
	IsDefault     bool                   `json:"is_default"`
}

// GetHeatmapDataRequest for generating heatmap data
type GetHeatmapDataRequest struct {
	WatchListID string `json:"watch_list_id"`
	ConfigID    string `json:"config_id,omitempty"` // If empty, use default config

	// Override config settings (optional)
	SizeMetric  string                 `json:"size_metric,omitempty"`
	ColorMetric string                 `json:"color_metric,omitempty"`
	TimePeriod  string                 `json:"time_period,omitempty"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
}
