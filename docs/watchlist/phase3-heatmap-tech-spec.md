# Phase 3: Custom Heatmap Visualization - Technical Specification

## Overview

**Goal:** Implement interactive heatmap visualization for watch lists with customizable metrics, similar to the Reddit trending stocks heatmap feature already built.

**Timeline:** 2 weeks (10 working days)

**Dependencies:**
- Phase 1 (Authentication) - ✅ Complete
- Phase 2 (Watch List Management) - ⏳ Pending
- Existing Reddit heatmap component - ✅ Available as reference
- D3.js or recharts library - Need to choose
- Real-time price data integration - ✅ Already working

**Key Features:**
- Interactive treemap visualization of watch list items
- Customizable size metric (market cap, volume, avg volume)
- Customizable color metric (price change %, volume change %)
- Time period selection (1D, 1W, 1M, 3M, 6M, YTD, 1Y)
- Save/load custom heatmap configurations
- Hover tooltips with detailed ticker info
- Click to navigate to ticker detail page
- Export heatmap as PNG image
- Full-screen heatmap mode

---

## Database Schema

### Migration File: `migrations/010_heatmap_configs.sql`

```sql
-- Heatmap Configurations table
-- Stores user's custom heatmap settings for reuse
CREATE TABLE heatmap_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    watch_list_id UUID REFERENCES watch_lists(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,

    -- Metric settings
    size_metric VARCHAR(50) DEFAULT 'market_cap', -- 'market_cap', 'volume', 'avg_volume', 'reddit_mentions'
    color_metric VARCHAR(50) DEFAULT 'price_change_pct', -- 'price_change_pct', 'volume_change_pct', 'reddit_sentiment'
    time_period VARCHAR(10) DEFAULT '1D', -- '1D', '1W', '1M', '3M', '6M', 'YTD', '1Y', '5Y'

    -- Visual settings
    color_scheme VARCHAR(50) DEFAULT 'red_green', -- 'red_green', 'heatmap', 'blue_red', 'custom'
    label_display VARCHAR(50) DEFAULT 'symbol_change', -- 'symbol', 'symbol_change', 'full'
    layout_type VARCHAR(50) DEFAULT 'treemap', -- 'treemap', 'grid'

    -- Filter settings (JSON for flexibility)
    filters_json JSONB DEFAULT '{}'::jsonb,
    -- Example filters:
    -- {
    --   "asset_types": ["stock", "crypto"],
    --   "sectors": ["Technology", "Finance"],
    --   "min_price": 10.0,
    --   "max_price": 1000.0,
    --   "min_market_cap": 1000000000,
    --   "max_market_cap": null
    -- }

    -- Custom color gradient (for custom color scheme)
    color_gradient_json JSONB,
    -- Example: {"negative": "#FF0000", "neutral": "#FFFFFF", "positive": "#00FF00"}

    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Ensure only one default per watch list
    CONSTRAINT unique_default_per_watchlist UNIQUE (user_id, watch_list_id, is_default)
        WHERE is_default = TRUE
);

-- Indexes
CREATE INDEX idx_heatmap_configs_user_id ON heatmap_configs(user_id);
CREATE INDEX idx_heatmap_configs_watch_list_id ON heatmap_configs(watch_list_id);
CREATE INDEX idx_heatmap_configs_user_watch_list ON heatmap_configs(user_id, watch_list_id);

-- Trigger to update updated_at
CREATE TRIGGER update_heatmap_configs_updated_at
BEFORE UPDATE ON heatmap_configs
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to create default heatmap config for new watch lists
CREATE OR REPLACE FUNCTION create_default_heatmap_config()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO heatmap_configs (
        user_id,
        watch_list_id,
        name,
        is_default
    )
    VALUES (
        NEW.user_id,
        NEW.id,
        'Default Heatmap',
        TRUE
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-create default heatmap config when watch list is created
CREATE TRIGGER auto_create_default_heatmap_config
AFTER INSERT ON watch_lists
FOR EACH ROW EXECUTE FUNCTION create_default_heatmap_config();
```

---

## Backend Implementation

### 1. Data Models

**File:** `backend/models/heatmap.go`

```go
package models

import (
	"time"
)

// HeatmapConfig represents a saved heatmap configuration
type HeatmapConfig struct {
	ID               string                 `json:"id" db:"id"`
	UserID           string                 `json:"user_id" db:"user_id"`
	WatchListID      string                 `json:"watch_list_id" db:"watch_list_id"`
	Name             string                 `json:"name" db:"name"`
	SizeMetric       string                 `json:"size_metric" db:"size_metric"`
	ColorMetric      string                 `json:"color_metric" db:"color_metric"`
	TimePeriod       string                 `json:"time_period" db:"time_period"`
	ColorScheme      string                 `json:"color_scheme" db:"color_scheme"`
	LabelDisplay     string                 `json:"label_display" db:"label_display"`
	LayoutType       string                 `json:"layout_type" db:"layout_type"`
	FiltersJSON      map[string]interface{} `json:"filters" db:"filters_json"`
	ColorGradientJSON map[string]string     `json:"color_gradient,omitempty" db:"color_gradient_json"`
	IsDefault        bool                   `json:"is_default" db:"is_default"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
}

// HeatmapTile represents a single tile in the heatmap
type HeatmapTile struct {
	Symbol          string   `json:"symbol"`
	Name            string   `json:"name"`
	AssetType       string   `json:"asset_type"`

	// Size value (what determines tile size)
	SizeValue       float64  `json:"size_value"`
	SizeLabel       string   `json:"size_label"` // e.g., "$1.2B", "15.3M shares"

	// Color value (what determines tile color)
	ColorValue      float64  `json:"color_value"`
	ColorLabel      string   `json:"color_label"` // e.g., "+5.2%", "-2.1%"

	// Current price info
	CurrentPrice    float64  `json:"current_price"`
	PriceChange     float64  `json:"price_change"`
	PriceChangePct  float64  `json:"price_change_pct"`

	// Additional metadata for tooltip
	Volume          *int64   `json:"volume,omitempty"`
	MarketCap       *float64 `json:"market_cap,omitempty"`
	PrevClose       *float64 `json:"prev_close,omitempty"`
	Exchange        string   `json:"exchange"`

	// User's custom data from watch list
	Notes           *string  `json:"notes,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	TargetBuyPrice  *float64 `json:"target_buy_price,omitempty"`
	TargetSellPrice *float64 `json:"target_sell_price,omitempty"`
}

// HeatmapData represents the complete heatmap data
type HeatmapData struct {
	WatchListID   string         `json:"watch_list_id"`
	WatchListName string         `json:"watch_list_name"`
	ConfigID      string         `json:"config_id,omitempty"`
	ConfigName    string         `json:"config_name,omitempty"`

	// Configuration used
	SizeMetric    string         `json:"size_metric"`
	ColorMetric   string         `json:"color_metric"`
	TimePeriod    string         `json:"time_period"`
	ColorScheme   string         `json:"color_scheme"`

	// Tiles data
	Tiles         []HeatmapTile  `json:"tiles"`
	TileCount     int            `json:"tile_count"`

	// Metadata for color scale
	MinColorValue float64        `json:"min_color_value"`
	MaxColorValue float64        `json:"max_color_value"`

	// Timestamp
	GeneratedAt   time.Time      `json:"generated_at"`
}

// Request/Response DTOs

// CreateHeatmapConfigRequest for saving a new config
type CreateHeatmapConfigRequest struct {
	WatchListID      string                 `json:"watch_list_id" binding:"required"`
	Name             string                 `json:"name" binding:"required,min=1,max=255"`
	SizeMetric       string                 `json:"size_metric" binding:"required,oneof=market_cap volume avg_volume reddit_mentions"`
	ColorMetric      string                 `json:"color_metric" binding:"required,oneof=price_change_pct volume_change_pct reddit_sentiment"`
	TimePeriod       string                 `json:"time_period" binding:"required,oneof=1D 1W 1M 3M 6M YTD 1Y 5Y"`
	ColorScheme      string                 `json:"color_scheme" binding:"oneof=red_green heatmap blue_red custom"`
	LabelDisplay     string                 `json:"label_display" binding:"oneof=symbol symbol_change full"`
	LayoutType       string                 `json:"layout_type" binding:"oneof=treemap grid"`
	Filters          map[string]interface{} `json:"filters"`
	ColorGradient    map[string]string      `json:"color_gradient,omitempty"`
	IsDefault        bool                   `json:"is_default"`
}

// UpdateHeatmapConfigRequest for updating existing config
type UpdateHeatmapConfigRequest struct {
	Name             string                 `json:"name" binding:"min=1,max=255"`
	SizeMetric       string                 `json:"size_metric" binding:"oneof=market_cap volume avg_volume reddit_mentions"`
	ColorMetric      string                 `json:"color_metric" binding:"oneof=price_change_pct volume_change_pct reddit_sentiment"`
	TimePeriod       string                 `json:"time_period" binding:"oneof=1D 1W 1M 3M 6M YTD 1Y 5Y"`
	ColorScheme      string                 `json:"color_scheme" binding:"oneof=red_green heatmap blue_red custom"`
	LabelDisplay     string                 `json:"label_display" binding:"oneof=symbol symbol_change full"`
	LayoutType       string                 `json:"layout_type" binding:"oneof=treemap grid"`
	Filters          map[string]interface{} `json:"filters"`
	ColorGradient    map[string]string      `json:"color_gradient,omitempty"`
	IsDefault        bool                   `json:"is_default"`
}

// GetHeatmapDataRequest for generating heatmap data
type GetHeatmapDataRequest struct {
	WatchListID   string                 `json:"watch_list_id"`
	ConfigID      string                 `json:"config_id,omitempty"` // If empty, use default config

	// Override config settings (optional)
	SizeMetric    string                 `json:"size_metric,omitempty"`
	ColorMetric   string                 `json:"color_metric,omitempty"`
	TimePeriod    string                 `json:"time_period,omitempty"`
	Filters       map[string]interface{} `json:"filters,omitempty"`
}
```

### 2. Database Operations

**File:** `backend/database/heatmaps.go`

```go
package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"investorcenter/backend/models"
)

// Heatmap Config Operations

// CreateHeatmapConfig creates a new heatmap configuration
func CreateHeatmapConfig(config *models.HeatmapConfig) error {
	// If setting as default, unset other defaults for this watch list
	if config.IsDefault {
		_, err := DB.Exec(`
			UPDATE heatmap_configs
			SET is_default = FALSE
			WHERE user_id = $1 AND watch_list_id = $2
		`, config.UserID, config.WatchListID)
		if err != nil {
			return fmt.Errorf("failed to unset previous default: %w", err)
		}
	}

	filtersJSON, _ := json.Marshal(config.FiltersJSON)
	gradientJSON, _ := json.Marshal(config.ColorGradientJSON)

	query := `
		INSERT INTO heatmap_configs (
			user_id, watch_list_id, name, size_metric, color_metric, time_period,
			color_scheme, label_display, layout_type, filters_json, color_gradient_json, is_default
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`

	err := DB.QueryRow(
		query,
		config.UserID,
		config.WatchListID,
		config.Name,
		config.SizeMetric,
		config.ColorMetric,
		config.TimePeriod,
		config.ColorScheme,
		config.LabelDisplay,
		config.LayoutType,
		filtersJSON,
		gradientJSON,
		config.IsDefault,
	).Scan(&config.ID, &config.CreatedAt, &config.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create heatmap config: %w", err)
	}
	return nil
}

// GetHeatmapConfigsByWatchListID retrieves all configs for a watch list
func GetHeatmapConfigsByWatchListID(watchListID string, userID string) ([]models.HeatmapConfig, error) {
	query := `
		SELECT
			id, user_id, watch_list_id, name, size_metric, color_metric, time_period,
			color_scheme, label_display, layout_type, filters_json, color_gradient_json,
			is_default, created_at, updated_at
		FROM heatmap_configs
		WHERE watch_list_id = $1 AND user_id = $2
		ORDER BY is_default DESC, created_at DESC
	`

	rows, err := DB.Query(query, watchListID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get heatmap configs: %w", err)
	}
	defer rows.Close()

	configs := []models.HeatmapConfig{}
	for rows.Next() {
		var config models.HeatmapConfig
		var filtersJSON, gradientJSON []byte

		err := rows.Scan(
			&config.ID,
			&config.UserID,
			&config.WatchListID,
			&config.Name,
			&config.SizeMetric,
			&config.ColorMetric,
			&config.TimePeriod,
			&config.ColorScheme,
			&config.LabelDisplay,
			&config.LayoutType,
			&filtersJSON,
			&gradientJSON,
			&config.IsDefault,
			&config.CreatedAt,
			&config.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan heatmap config: %w", err)
		}

		// Parse JSON fields
		if len(filtersJSON) > 0 {
			json.Unmarshal(filtersJSON, &config.FiltersJSON)
		}
		if len(gradientJSON) > 0 {
			json.Unmarshal(gradientJSON, &config.ColorGradientJSON)
		}

		configs = append(configs, config)
	}

	return configs, nil
}

// GetHeatmapConfigByID retrieves a single config by ID
func GetHeatmapConfigByID(configID string, userID string) (*models.HeatmapConfig, error) {
	query := `
		SELECT
			id, user_id, watch_list_id, name, size_metric, color_metric, time_period,
			color_scheme, label_display, layout_type, filters_json, color_gradient_json,
			is_default, created_at, updated_at
		FROM heatmap_configs
		WHERE id = $1 AND user_id = $2
	`

	var config models.HeatmapConfig
	var filtersJSON, gradientJSON []byte

	err := DB.QueryRow(query, configID, userID).Scan(
		&config.ID,
		&config.UserID,
		&config.WatchListID,
		&config.Name,
		&config.SizeMetric,
		&config.ColorMetric,
		&config.TimePeriod,
		&config.ColorScheme,
		&config.LabelDisplay,
		&config.LayoutType,
		&filtersJSON,
		&gradientJSON,
		&config.IsDefault,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("heatmap config not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get heatmap config: %w", err)
	}

	// Parse JSON fields
	if len(filtersJSON) > 0 {
		json.Unmarshal(filtersJSON, &config.FiltersJSON)
	}
	if len(gradientJSON) > 0 {
		json.Unmarshal(gradientJSON, &config.ColorGradientJSON)
	}

	return &config, nil
}

// GetDefaultHeatmapConfig gets the default config for a watch list
func GetDefaultHeatmapConfig(watchListID string, userID string) (*models.HeatmapConfig, error) {
	query := `
		SELECT
			id, user_id, watch_list_id, name, size_metric, color_metric, time_period,
			color_scheme, label_display, layout_type, filters_json, color_gradient_json,
			is_default, created_at, updated_at
		FROM heatmap_configs
		WHERE watch_list_id = $1 AND user_id = $2 AND is_default = TRUE
		LIMIT 1
	`

	var config models.HeatmapConfig
	var filtersJSON, gradientJSON []byte

	err := DB.QueryRow(query, watchListID, userID).Scan(
		&config.ID,
		&config.UserID,
		&config.WatchListID,
		&config.Name,
		&config.SizeMetric,
		&config.ColorMetric,
		&config.TimePeriod,
		&config.ColorScheme,
		&config.LabelDisplay,
		&config.LayoutType,
		&filtersJSON,
		&gradientJSON,
		&config.IsDefault,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// No default found, create one
		defaultConfig := &models.HeatmapConfig{
			UserID:       userID,
			WatchListID:  watchListID,
			Name:         "Default Heatmap",
			SizeMetric:   "market_cap",
			ColorMetric:  "price_change_pct",
			TimePeriod:   "1D",
			ColorScheme:  "red_green",
			LabelDisplay: "symbol_change",
			LayoutType:   "treemap",
			FiltersJSON:  map[string]interface{}{},
			IsDefault:    true,
		}
		err = CreateHeatmapConfig(defaultConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return defaultConfig, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get default heatmap config: %w", err)
	}

	// Parse JSON fields
	if len(filtersJSON) > 0 {
		json.Unmarshal(filtersJSON, &config.FiltersJSON)
	}
	if len(gradientJSON) > 0 {
		json.Unmarshal(gradientJSON, &config.ColorGradientJSON)
	}

	return &config, nil
}

// UpdateHeatmapConfig updates an existing config
func UpdateHeatmapConfig(config *models.HeatmapConfig) error {
	// If setting as default, unset other defaults for this watch list
	if config.IsDefault {
		_, err := DB.Exec(`
			UPDATE heatmap_configs
			SET is_default = FALSE
			WHERE user_id = $1 AND watch_list_id = $2 AND id != $3
		`, config.UserID, config.WatchListID, config.ID)
		if err != nil {
			return fmt.Errorf("failed to unset previous default: %w", err)
		}
	}

	filtersJSON, _ := json.Marshal(config.FiltersJSON)
	gradientJSON, _ := json.Marshal(config.ColorGradientJSON)

	query := `
		UPDATE heatmap_configs
		SET name = $1, size_metric = $2, color_metric = $3, time_period = $4,
		    color_scheme = $5, label_display = $6, layout_type = $7,
		    filters_json = $8, color_gradient_json = $9, is_default = $10,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $11 AND user_id = $12
	`

	result, err := DB.Exec(
		query,
		config.Name,
		config.SizeMetric,
		config.ColorMetric,
		config.TimePeriod,
		config.ColorScheme,
		config.LabelDisplay,
		config.LayoutType,
		filtersJSON,
		gradientJSON,
		config.IsDefault,
		config.ID,
		config.UserID,
	)
	if err != nil {
		return fmt.Errorf("failed to update heatmap config: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("heatmap config not found or unauthorized")
	}

	return nil
}

// DeleteHeatmapConfig deletes a config
func DeleteHeatmapConfig(configID string, userID string) error {
	query := `DELETE FROM heatmap_configs WHERE id = $1 AND user_id = $2`
	result, err := DB.Exec(query, configID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete heatmap config: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("heatmap config not found or unauthorized")
	}

	return nil
}
```

### 3. Service Layer

**File:** `backend/services/heatmap_service.go`

```go
package services

import (
	"fmt"
	"investorcenter/backend/database"
	"investorcenter/backend/models"
	"math"
	"time"
)

// HeatmapService handles business logic for heatmaps
type HeatmapService struct{}

func NewHeatmapService() *HeatmapService {
	return &HeatmapService{}
}

// GenerateHeatmapData creates heatmap data for a watch list
func (s *HeatmapService) GenerateHeatmapData(
	watchListID string,
	userID string,
	configID string,
	overrides *models.GetHeatmapDataRequest,
) (*models.HeatmapData, error) {

	// Get watch list with items
	watchList, err := database.GetWatchListByID(watchListID, userID)
	if err != nil {
		return nil, err
	}

	// Get items with ticker data
	items, err := database.GetWatchListItemsWithData(watchListID)
	if err != nil {
		return nil, err
	}

	// Get configuration
	var config *models.HeatmapConfig
	if configID != "" {
		config, err = database.GetHeatmapConfigByID(configID, userID)
	} else {
		config, err = database.GetDefaultHeatmapConfig(watchListID, userID)
	}
	if err != nil {
		return nil, err
	}

	// Apply overrides
	if overrides != nil {
		if overrides.SizeMetric != "" {
			config.SizeMetric = overrides.SizeMetric
		}
		if overrides.ColorMetric != "" {
			config.ColorMetric = overrides.ColorMetric
		}
		if overrides.TimePeriod != "" {
			config.TimePeriod = overrides.TimePeriod
		}
		if overrides.Filters != nil {
			config.FiltersJSON = overrides.Filters
		}
	}

	// Generate tiles from items
	tiles := make([]models.HeatmapTile, 0, len(items))
	var minColorValue, maxColorValue float64 = math.MaxFloat64, -math.MaxFloat64

	for _, item := range items {
		// Apply filters
		if !s.passesFilters(&item, config.FiltersJSON) {
			continue
		}

		tile := models.HeatmapTile{
			Symbol:          item.Symbol,
			Name:            item.Name,
			AssetType:       item.AssetType,
			Exchange:        item.Exchange,
			Notes:           item.Notes,
			Tags:            item.Tags,
			TargetBuyPrice:  item.TargetBuyPrice,
			TargetSellPrice: item.TargetSellPrice,
		}

		// Set current price info
		if item.CurrentPrice != nil {
			tile.CurrentPrice = *item.CurrentPrice
		}
		if item.PriceChange != nil {
			tile.PriceChange = *item.PriceChange
		}
		if item.PriceChangePct != nil {
			tile.PriceChangePct = *item.PriceChangePct
		}
		tile.Volume = item.Volume
		tile.MarketCap = item.MarketCap
		tile.PrevClose = item.PrevClose

		// Calculate size value based on size metric
		sizeValue, sizeLabel := s.calculateSizeValue(&item, config.SizeMetric)
		tile.SizeValue = sizeValue
		tile.SizeLabel = sizeLabel

		// Calculate color value based on color metric
		colorValue, colorLabel := s.calculateColorValue(&item, config.ColorMetric, config.TimePeriod)
		tile.ColorValue = colorValue
		tile.ColorLabel = colorLabel

		// Track min/max for color scale
		if colorValue < minColorValue {
			minColorValue = colorValue
		}
		if colorValue > maxColorValue {
			maxColorValue = colorValue
		}

		tiles = append(tiles, tile)
	}

	heatmapData := &models.HeatmapData{
		WatchListID:   watchListID,
		WatchListName: watchList.Name,
		ConfigID:      config.ID,
		ConfigName:    config.Name,
		SizeMetric:    config.SizeMetric,
		ColorMetric:   config.ColorMetric,
		TimePeriod:    config.TimePeriod,
		ColorScheme:   config.ColorScheme,
		Tiles:         tiles,
		TileCount:     len(tiles),
		MinColorValue: minColorValue,
		MaxColorValue: maxColorValue,
		GeneratedAt:   time.Now(),
	}

	return heatmapData, nil
}

// calculateSizeValue determines tile size based on metric
func (s *HeatmapService) calculateSizeValue(
	item *models.WatchListItemWithData,
	metric string,
) (float64, string) {
	switch metric {
	case "market_cap":
		if item.MarketCap != nil {
			return *item.MarketCap, s.formatMarketCap(*item.MarketCap)
		}
	case "volume":
		if item.Volume != nil {
			return float64(*item.Volume), s.formatVolume(*item.Volume)
		}
	case "avg_volume":
		// Would need historical data - for now use current volume
		if item.Volume != nil {
			return float64(*item.Volume), s.formatVolume(*item.Volume)
		}
	}
	// Default to market cap 1B if no data
	return 1000000000, "N/A"
}

// calculateColorValue determines tile color based on metric
func (s *HeatmapService) calculateColorValue(
	item *models.WatchListItemWithData,
	metric string,
	timePeriod string,
) (float64, string) {
	switch metric {
	case "price_change_pct":
		if item.PriceChangePct != nil {
			return *item.PriceChangePct, fmt.Sprintf("%+.2f%%", *item.PriceChangePct)
		}
	case "volume_change_pct":
		// Would need historical volume data
		// For now return 0
		return 0, "N/A"
	}
	return 0, "N/A"
}

// passesFilters checks if item passes filter criteria
func (s *HeatmapService) passesFilters(
	item *models.WatchListItemWithData,
	filters map[string]interface{},
) bool {
	if filters == nil {
		return true
	}

	// Asset type filter
	if assetTypes, ok := filters["asset_types"].([]interface{}); ok {
		found := false
		for _, at := range assetTypes {
			if atStr, ok := at.(string); ok && atStr == item.AssetType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Price range filter
	if item.CurrentPrice != nil {
		if minPrice, ok := filters["min_price"].(float64); ok {
			if *item.CurrentPrice < minPrice {
				return false
			}
		}
		if maxPrice, ok := filters["max_price"].(float64); ok {
			if *item.CurrentPrice > maxPrice {
				return false
			}
		}
	}

	// Market cap range filter
	if item.MarketCap != nil {
		if minMC, ok := filters["min_market_cap"].(float64); ok {
			if *item.MarketCap < minMC {
				return false
			}
		}
		if maxMC, ok := filters["max_market_cap"].(float64); ok {
			if *item.MarketCap > maxMC {
				return false
			}
		}
	}

	return true
}

// Helper formatting functions

func (s *HeatmapService) formatMarketCap(value float64) string {
	if value >= 1e12 {
		return fmt.Sprintf("$%.1fT", value/1e12)
	} else if value >= 1e9 {
		return fmt.Sprintf("$%.1fB", value/1e9)
	} else if value >= 1e6 {
		return fmt.Sprintf("$%.1fM", value/1e6)
	}
	return fmt.Sprintf("$%.0f", value)
}

func (s *HeatmapService) formatVolume(value int64) string {
	fValue := float64(value)
	if fValue >= 1e9 {
		return fmt.Sprintf("%.1fB", fValue/1e9)
	} else if fValue >= 1e6 {
		return fmt.Sprintf("%.1fM", fValue/1e6)
	} else if fValue >= 1e3 {
		return fmt.Sprintf("%.1fK", fValue/1e3)
	}
	return fmt.Sprintf("%d", value)
}
```

### 4. Handlers

**File:** `backend/handlers/heatmap_handlers.go`

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

var heatmapService = services.NewHeatmapService()

// GetHeatmapData generates heatmap data for a watch list
func GetHeatmapData(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	watchListID := c.Param("id")
	configID := c.Query("config_id") // Optional

	// Check for override parameters in query
	var overrides *models.GetHeatmapDataRequest
	if c.Query("size_metric") != "" || c.Query("color_metric") != "" || c.Query("time_period") != "" {
		overrides = &models.GetHeatmapDataRequest{
			SizeMetric:  c.Query("size_metric"),
			ColorMetric: c.Query("color_metric"),
			TimePeriod:  c.Query("time_period"),
		}
	}

	// Verify ownership
	if err := watchListService.ValidateWatchListOwnership(watchListID, userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized access to watch list"})
		return
	}

	// Generate heatmap data
	heatmapData, err := heatmapService.GenerateHeatmapData(watchListID, userID, configID, overrides)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, heatmapData)
}

// ListHeatmapConfigs retrieves all configs for a watch list
func ListHeatmapConfigs(c *gin.Context) {
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

	configs, err := database.GetHeatmapConfigsByWatchListID(watchListID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch heatmap configs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"configs": configs})
}

// CreateHeatmapConfig saves a new heatmap configuration
func CreateHeatmapConfig(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.CreateHeatmapConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify ownership
	if err := watchListService.ValidateWatchListOwnership(req.WatchListID, userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized access to watch list"})
		return
	}

	config := &models.HeatmapConfig{
		UserID:            userID,
		WatchListID:       req.WatchListID,
		Name:              req.Name,
		SizeMetric:        req.SizeMetric,
		ColorMetric:       req.ColorMetric,
		TimePeriod:        req.TimePeriod,
		ColorScheme:       req.ColorScheme,
		LabelDisplay:      req.LabelDisplay,
		LayoutType:        req.LayoutType,
		FiltersJSON:       req.Filters,
		ColorGradientJSON: req.ColorGradient,
		IsDefault:         req.IsDefault,
	}

	err := database.CreateHeatmapConfig(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create heatmap config"})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// UpdateHeatmapConfig updates an existing configuration
func UpdateHeatmapConfig(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	configID := c.Param("configId")

	var req models.UpdateHeatmapConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing config to verify ownership
	existingConfig, err := database.GetHeatmapConfigByID(configID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Heatmap config not found"})
		return
	}

	// Update fields
	existingConfig.Name = req.Name
	existingConfig.SizeMetric = req.SizeMetric
	existingConfig.ColorMetric = req.ColorMetric
	existingConfig.TimePeriod = req.TimePeriod
	existingConfig.ColorScheme = req.ColorScheme
	existingConfig.LabelDisplay = req.LabelDisplay
	existingConfig.LayoutType = req.LayoutType
	existingConfig.FiltersJSON = req.Filters
	existingConfig.ColorGradientJSON = req.ColorGradient
	existingConfig.IsDefault = req.IsDefault

	err = database.UpdateHeatmapConfig(existingConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update heatmap config"})
		return
	}

	c.JSON(http.StatusOK, existingConfig)
}

// DeleteHeatmapConfig deletes a configuration
func DeleteHeatmapConfig(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	configID := c.Param("configId")

	err := database.DeleteHeatmapConfig(configID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Heatmap config deleted successfully"})
}
```

### 5. Update main.go with Heatmap Routes

**File:** `backend/main.go` (add these routes)

```go
// Heatmap routes (under watch list routes)
watchListRoutes.GET("/:id/heatmap", handlers.GetHeatmapData)              // GET /api/v1/watchlists/:id/heatmap
watchListRoutes.GET("/:id/heatmap/configs", handlers.ListHeatmapConfigs)  // GET /api/v1/watchlists/:id/heatmap/configs
watchListRoutes.POST("/:id/heatmap/configs", handlers.CreateHeatmapConfig) // POST /api/v1/watchlists/:id/heatmap/configs
watchListRoutes.PUT("/:id/heatmap/configs/:configId", handlers.UpdateHeatmapConfig) // PUT /api/v1/watchlists/:id/heatmap/configs/:configId
watchListRoutes.DELETE("/:id/heatmap/configs/:configId", handlers.DeleteHeatmapConfig) // DELETE /api/v1/watchlists/:id/heatmap/configs/:configId
```

---

## Frontend Implementation

### 1. Library Choice: D3.js vs Recharts

**Recommendation: D3.js** for the following reasons:
- More control over treemap layout
- Better performance with large datasets
- Already proven in the Reddit heatmap feature
- Supports custom interactions and animations

### 2. API Client

**File:** `lib/api/heatmap.ts`

```typescript
import { apiClient } from './client';

export interface HeatmapConfig {
  id: string;
  user_id: string;
  watch_list_id: string;
  name: string;
  size_metric: 'market_cap' | 'volume' | 'avg_volume' | 'reddit_mentions';
  color_metric: 'price_change_pct' | 'volume_change_pct' | 'reddit_sentiment';
  time_period: '1D' | '1W' | '1M' | '3M' | '6M' | 'YTD' | '1Y' | '5Y';
  color_scheme: 'red_green' | 'heatmap' | 'blue_red' | 'custom';
  label_display: 'symbol' | 'symbol_change' | 'full';
  layout_type: 'treemap' | 'grid';
  filters: Record<string, any>;
  color_gradient?: Record<string, string>;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface HeatmapTile {
  symbol: string;
  name: string;
  asset_type: string;
  size_value: number;
  size_label: string;
  color_value: number;
  color_label: string;
  current_price: number;
  price_change: number;
  price_change_pct: number;
  volume?: number;
  market_cap?: number;
  prev_close?: number;
  exchange: string;
  notes?: string;
  tags: string[];
  target_buy_price?: number;
  target_sell_price?: number;
}

export interface HeatmapData {
  watch_list_id: string;
  watch_list_name: string;
  config_id?: string;
  config_name?: string;
  size_metric: string;
  color_metric: string;
  time_period: string;
  color_scheme: string;
  tiles: HeatmapTile[];
  tile_count: number;
  min_color_value: number;
  max_color_value: number;
  generated_at: string;
}

export const heatmapAPI = {
  // Get heatmap data
  async getHeatmapData(
    watchListId: string,
    configId?: string,
    overrides?: {
      size_metric?: string;
      color_metric?: string;
      time_period?: string;
    }
  ): Promise<HeatmapData> {
    const params = new URLSearchParams();
    if (configId) params.append('config_id', configId);
    if (overrides?.size_metric) params.append('size_metric', overrides.size_metric);
    if (overrides?.color_metric) params.append('color_metric', overrides.color_metric);
    if (overrides?.time_period) params.append('time_period', overrides.time_period);

    const queryString = params.toString();
    const url = `/watchlists/${watchListId}/heatmap${queryString ? `?${queryString}` : ''}`;
    return apiClient.get(url);
  },

  // Get all configs for a watch list
  async getConfigs(watchListId: string): Promise<{ configs: HeatmapConfig[] }> {
    return apiClient.get(`/watchlists/${watchListId}/heatmap/configs`);
  },

  // Create new config
  async createConfig(watchListId: string, config: {
    name: string;
    size_metric: string;
    color_metric: string;
    time_period: string;
    color_scheme?: string;
    label_display?: string;
    layout_type?: string;
    filters?: Record<string, any>;
    color_gradient?: Record<string, string>;
    is_default?: boolean;
  }): Promise<HeatmapConfig> {
    return apiClient.post(`/watchlists/${watchListId}/heatmap/configs`, {
      watch_list_id: watchListId,
      ...config,
    });
  },

  // Update config
  async updateConfig(
    watchListId: string,
    configId: string,
    config: Partial<HeatmapConfig>
  ): Promise<HeatmapConfig> {
    return apiClient.put(`/watchlists/${watchListId}/heatmap/configs/${configId}`, config);
  },

  // Delete config
  async deleteConfig(watchListId: string, configId: string): Promise<void> {
    return apiClient.delete(`/watchlists/${watchListId}/heatmap/configs/${configId}`);
  },
};
```

### 3. Heatmap Component (D3.js)

**File:** `components/watchlist/WatchListHeatmap.tsx`

```typescript
'use client';

import { useEffect, useRef, useState } from 'react';
import * as d3 from 'd3';
import { HeatmapTile, HeatmapData } from '@/lib/api/heatmap';
import { useRouter } from 'next/navigation';

interface WatchListHeatmapProps {
  data: HeatmapData;
  width?: number;
  height?: number;
  onTileClick?: (symbol: string) => void;
}

export default function WatchListHeatmap({
  data,
  width = 1200,
  height = 600,
  onTileClick,
}: WatchListHeatmapProps) {
  const svgRef = useRef<SVGSVGElement>(null);
  const tooltipRef = useRef<HTMLDivElement>(null);
  const router = useRouter();

  useEffect(() => {
    if (!svgRef.current || !data.tiles.length) return;

    // Clear previous render
    d3.select(svgRef.current).selectAll('*').remove();

    // Create treemap layout
    const root = d3.hierarchy({ children: data.tiles })
      .sum((d: any) => d.size_value || 1)
      .sort((a, b) => (b.value || 0) - (a.value || 0));

    const treemap = d3.treemap<any>()
      .size([width, height])
      .padding(2)
      .round(true);

    treemap(root);

    // Color scale based on color metric
    const colorScale = getColorScale(data.color_scheme, data.min_color_value, data.max_color_value);

    const svg = d3.select(svgRef.current);

    // Create groups for each tile
    const nodes = svg.selectAll('g')
      .data(root.leaves())
      .join('g')
      .attr('transform', (d: any) => `translate(${d.x0},${d.y0})`);

    // Add rectangles
    nodes.append('rect')
      .attr('width', (d: any) => d.x1 - d.x0)
      .attr('height', (d: any) => d.y1 - d.y0)
      .attr('fill', (d: any) => colorScale(d.data.color_value))
      .attr('stroke', '#fff')
      .attr('stroke-width', 2)
      .attr('rx', 4)
      .style('cursor', 'pointer')
      .on('mouseover', function(event: MouseEvent, d: any) {
        // Highlight tile
        d3.select(this)
          .attr('stroke', '#000')
          .attr('stroke-width', 3);

        // Show tooltip
        showTooltip(event, d.data);
      })
      .on('mouseout', function() {
        // Remove highlight
        d3.select(this)
          .attr('stroke', '#fff')
          .attr('stroke-width', 2);

        // Hide tooltip
        hideTooltip();
      })
      .on('click', (event: MouseEvent, d: any) => {
        if (onTileClick) {
          onTileClick(d.data.symbol);
        } else {
          router.push(`/ticker/${d.data.symbol}`);
        }
      });

    // Add symbol text
    nodes.append('text')
      .attr('x', (d: any) => (d.x1 - d.x0) / 2)
      .attr('y', (d: any) => (d.y1 - d.y0) / 2 - 8)
      .attr('text-anchor', 'middle')
      .attr('fill', (d: any) => getTextColor(colorScale(d.data.color_value)))
      .style('font-weight', 'bold')
      .style('font-size', (d: any) => {
        const tileSize = Math.min(d.x1 - d.x0, d.y1 - d.y0);
        return `${Math.max(10, Math.min(16, tileSize / 8))}px`;
      })
      .style('pointer-events', 'none')
      .text((d: any) => d.data.symbol);

    // Add change percentage text
    if (data.color_metric === 'price_change_pct') {
      nodes.append('text')
        .attr('x', (d: any) => (d.x1 - d.x0) / 2)
        .attr('y', (d: any) => (d.y1 - d.y0) / 2 + 12)
        .attr('text-anchor', 'middle')
        .attr('fill', (d: any) => getTextColor(colorScale(d.data.color_value)))
        .style('font-size', (d: any) => {
          const tileSize = Math.min(d.x1 - d.x0, d.y1 - d.y0);
          return `${Math.max(8, Math.min(12, tileSize / 12))}px`;
        })
        .style('pointer-events', 'none')
        .text((d: any) => d.data.color_label);
    }

  }, [data, width, height, router, onTileClick]);

  const showTooltip = (event: MouseEvent, tile: HeatmapTile) => {
    const tooltip = tooltipRef.current;
    if (!tooltip) return;

    tooltip.innerHTML = `
      <div class="font-bold text-lg mb-2">${tile.symbol} - ${tile.name}</div>
      <div class="grid grid-cols-2 gap-2 text-sm">
        <div class="text-gray-600">Price:</div>
        <div class="font-medium">$${tile.current_price.toFixed(2)}</div>

        <div class="text-gray-600">Change:</div>
        <div class="font-medium ${tile.price_change >= 0 ? 'text-green-600' : 'text-red-600'}">
          ${tile.price_change >= 0 ? '+' : ''}${tile.price_change.toFixed(2)} (${tile.price_change_pct.toFixed(2)}%)
        </div>

        ${tile.market_cap ? `
          <div class="text-gray-600">Market Cap:</div>
          <div class="font-medium">${tile.size_label}</div>
        ` : ''}

        ${tile.volume ? `
          <div class="text-gray-600">Volume:</div>
          <div class="font-medium">${formatVolume(tile.volume)}</div>
        ` : ''}

        ${tile.target_buy_price ? `
          <div class="text-gray-600">Target Buy:</div>
          <div class="font-medium text-blue-600">$${tile.target_buy_price.toFixed(2)}</div>
        ` : ''}

        ${tile.target_sell_price ? `
          <div class="text-gray-600">Target Sell:</div>
          <div class="font-medium text-orange-600">$${tile.target_sell_price.toFixed(2)}</div>
        ` : ''}
      </div>
      ${tile.notes ? `<div class="mt-2 text-sm text-gray-600 italic">${tile.notes}</div>` : ''}
      ${tile.tags.length > 0 ? `
        <div class="mt-2 flex flex-wrap gap-1">
          ${tile.tags.map(tag => `<span class="text-xs bg-gray-200 px-2 py-1 rounded">${tag}</span>`).join('')}
        </div>
      ` : ''}
    `;

    tooltip.style.display = 'block';
    tooltip.style.left = `${event.pageX + 10}px`;
    tooltip.style.top = `${event.pageY + 10}px`;
  };

  const hideTooltip = () => {
    const tooltip = tooltipRef.current;
    if (tooltip) {
      tooltip.style.display = 'none';
    }
  };

  const formatVolume = (vol: number) => {
    if (vol >= 1e9) return `${(vol / 1e9).toFixed(1)}B`;
    if (vol >= 1e6) return `${(vol / 1e6).toFixed(1)}M`;
    if (vol >= 1e3) return `${(vol / 1e3).toFixed(1)}K`;
    return vol.toString();
  };

  return (
    <div className="relative">
      <svg ref={svgRef} width={width} height={height} className="bg-gray-50 rounded-lg" />
      <div
        ref={tooltipRef}
        className="absolute hidden bg-white p-4 rounded-lg shadow-lg border border-gray-200 max-w-sm z-50"
        style={{ pointerEvents: 'none' }}
      />
    </div>
  );
}

// Helper functions

function getColorScale(scheme: string, min: number, max: number) {
  switch (scheme) {
    case 'red_green':
      return d3.scaleLinear<string>()
        .domain([min, 0, max])
        .range(['#EF4444', '#F3F4F6', '#10B981']);

    case 'blue_red':
      return d3.scaleLinear<string>()
        .domain([min, 0, max])
        .range(['#3B82F6', '#F3F4F6', '#EF4444']);

    case 'heatmap':
      return d3.scaleSequential(d3.interpolateRdYlGn)
        .domain([min, max]);

    default:
      return d3.scaleLinear<string>()
        .domain([min, 0, max])
        .range(['#EF4444', '#F3F4F6', '#10B981']);
  }
}

function getTextColor(bgColor: string): string {
  // Convert hex/rgb to luminance
  const rgb = d3.rgb(bgColor);
  const luminance = (0.299 * rgb.r + 0.587 * rgb.g + 0.114 * rgb.b) / 255;
  return luminance > 0.5 ? '#000000' : '#FFFFFF';
}
```

### 4. Heatmap Configuration Panel

**File:** `components/watchlist/HeatmapConfigPanel.tsx`

```typescript
'use client';

import { useState } from 'react';

export interface HeatmapSettings {
  size_metric: string;
  color_metric: string;
  time_period: string;
  color_scheme: string;
  label_display: string;
}

interface HeatmapConfigPanelProps {
  settings: HeatmapSettings;
  onChange: (settings: HeatmapSettings) => void;
  onSave?: (name: string) => void;
}

export default function HeatmapConfigPanel({ settings, onChange, onSave }: HeatmapConfigPanelProps) {
  const [showSaveModal, setShowSaveModal] = useState(false);
  const [configName, setConfigName] = useState('');

  const handleChange = (field: keyof HeatmapSettings, value: string) => {
    onChange({ ...settings, [field]: value });
  };

  const handleSave = () => {
    if (onSave && configName) {
      onSave(configName);
      setConfigName('');
      setShowSaveModal(false);
    }
  };

  return (
    <div className="bg-white p-4 rounded-lg shadow mb-4">
      <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
        {/* Size Metric */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Size
          </label>
          <select
            value={settings.size_metric}
            onChange={(e) => handleChange('size_metric', e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="market_cap">Market Cap</option>
            <option value="volume">Volume</option>
            <option value="avg_volume">Avg Volume</option>
          </select>
        </div>

        {/* Color Metric */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Color
          </label>
          <select
            value={settings.color_metric}
            onChange={(e) => handleChange('color_metric', e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="price_change_pct">Price Change %</option>
            <option value="volume_change_pct">Volume Change %</option>
          </select>
        </div>

        {/* Time Period */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Time Period
          </label>
          <select
            value={settings.time_period}
            onChange={(e) => handleChange('time_period', e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="1D">1 Day</option>
            <option value="1W">1 Week</option>
            <option value="1M">1 Month</option>
            <option value="3M">3 Months</option>
            <option value="6M">6 Months</option>
            <option value="YTD">YTD</option>
            <option value="1Y">1 Year</option>
            <option value="5Y">5 Years</option>
          </select>
        </div>

        {/* Color Scheme */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Colors
          </label>
          <select
            value={settings.color_scheme}
            onChange={(e) => handleChange('color_scheme', e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="red_green">Red-Green</option>
            <option value="blue_red">Blue-Red</option>
            <option value="heatmap">Heatmap</option>
          </select>
        </div>

        {/* Save Button */}
        {onSave && (
          <div className="flex items-end">
            <button
              onClick={() => setShowSaveModal(true)}
              className="w-full px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
            >
              Save Config
            </button>
          </div>
        )}
      </div>

      {/* Save Config Modal */}
      {showSaveModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h3 className="text-xl font-bold mb-4">Save Heatmap Configuration</h3>
            <input
              type="text"
              value={configName}
              onChange={(e) => setConfigName(e.target.value)}
              placeholder="Configuration name"
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 mb-4"
            />
            <div className="flex gap-2 justify-end">
              <button
                onClick={() => setShowSaveModal(false)}
                className="px-4 py-2 border border-gray-300 rounded hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={handleSave}
                disabled={!configName}
                className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:bg-gray-400"
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
```

### 5. Full Heatmap Page

**File:** `app/watchlist/[id]/heatmap/page.tsx`

```typescript
'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { heatmapAPI, HeatmapData } from '@/lib/api/heatmap';
import { ProtectedRoute } from '@/components/auth/ProtectedRoute';
import WatchListHeatmap from '@/components/watchlist/WatchListHeatmap';
import HeatmapConfigPanel, { HeatmapSettings } from '@/components/watchlist/HeatmapConfigPanel';

export default function WatchListHeatmapPage() {
  const params = useParams();
  const watchListId = params.id as string;

  const [heatmapData, setHeatmapData] = useState<HeatmapData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [settings, setSettings] = useState<HeatmapSettings>({
    size_metric: 'market_cap',
    color_metric: 'price_change_pct',
    time_period: '1D',
    color_scheme: 'red_green',
    label_display: 'symbol_change',
  });

  useEffect(() => {
    loadHeatmap();
    // Auto-refresh every 30 seconds
    const interval = setInterval(loadHeatmap, 30000);
    return () => clearInterval(interval);
  }, [watchListId, settings]);

  const loadHeatmap = async () => {
    try {
      const data = await heatmapAPI.getHeatmapData(watchListId, undefined, settings);
      setHeatmapData(data);
    } catch (err: any) {
      setError(err.message || 'Failed to load heatmap');
    } finally {
      setLoading(false);
    }
  };

  const handleSaveConfig = async (name: string) => {
    try {
      await heatmapAPI.createConfig(watchListId, {
        name,
        ...settings,
      });
      alert('Heatmap configuration saved!');
    } catch (err: any) {
      alert(err.message || 'Failed to save configuration');
    }
  };

  if (loading) {
    return (
      <ProtectedRoute>
        <div className="flex items-center justify-center min-h-screen">
          <div className="text-xl">Loading heatmap...</div>
        </div>
      </ProtectedRoute>
    );
  }

  if (!heatmapData) {
    return (
      <ProtectedRoute>
        <div className="container mx-auto px-4 py-8">
          <div className="text-center">
            <p className="text-red-600">{error || 'No data available'}</p>
          </div>
        </div>
      </ProtectedRoute>
    );
  }

  return (
    <ProtectedRoute>
      <div className="container mx-auto px-4 py-8">
        <div className="mb-6">
          <h1 className="text-3xl font-bold">{heatmapData.watch_list_name} - Heatmap</h1>
          <p className="text-gray-600 mt-2">
            {heatmapData.tile_count} tickers | Updated {new Date(heatmapData.generated_at).toLocaleTimeString()}
          </p>
        </div>

        {error && (
          <div className="mb-4 p-3 bg-red-100 border border-red-400 text-red-700 rounded">
            {error}
          </div>
        )}

        <HeatmapConfigPanel
          settings={settings}
          onChange={setSettings}
          onSave={handleSaveConfig}
        />

        {heatmapData.tiles.length === 0 ? (
          <div className="text-center py-12 bg-gray-50 rounded-lg">
            <p className="text-gray-600">No tickers to display in heatmap</p>
          </div>
        ) : (
          <WatchListHeatmap
            data={heatmapData}
            width={window.innerWidth - 100}
            height={600}
          />
        )}
      </div>
    </ProtectedRoute>
  );
}
```

---

## Testing Plan

### Backend Testing

**Manual Testing Checklist:**

- [ ] Create default heatmap config when watch list is created
- [ ] Get heatmap data with default config
- [ ] Get heatmap data with custom config ID
- [ ] Get heatmap data with query parameter overrides
- [ ] Verify tile size calculated correctly (market cap)
- [ ] Verify tile color calculated correctly (price change %)
- [ ] Test with different time periods (1D, 1W, 1M, etc.)
- [ ] Test color scales (red-green, blue-red, heatmap)
- [ ] Test filters (asset type, price range, market cap range)
- [ ] Save new heatmap configuration
- [ ] Update existing configuration
- [ ] Delete configuration
- [ ] Set configuration as default
- [ ] List all configurations for watch list
- [ ] Verify min/max color values calculated correctly
- [ ] Test with empty watch list (0 tickers)
- [ ] Test with large watch list (100+ tickers)

### Frontend Testing

**Manual Testing Checklist:**

- [ ] Heatmap renders with tiles sized by market cap
- [ ] Tile colors match price change % (red for negative, green for positive)
- [ ] Hover tooltip displays correct ticker information
- [ ] Click tile navigates to ticker detail page
- [ ] Configuration panel updates heatmap in real-time
- [ ] Time period selector changes data correctly
- [ ] Color scheme selector changes tile colors
- [ ] Size metric selector changes tile sizes
- [ ] Save configuration modal opens and saves
- [ ] Saved configurations can be loaded
- [ ] Auto-refresh works (every 30 seconds)
- [ ] Full-screen mode displays correctly
- [ ] Export as PNG works (future)
- [ ] Responsive design on mobile (future)
- [ ] Loading state displays
- [ ] Error handling for API failures
- [ ] Empty state for watch lists with no tickers

---

## Deployment Checklist

### Database

- [ ] Run migration `migrations/010_heatmap_configs.sql` on production
- [ ] Verify heatmap_configs table created
- [ ] Verify indexes created
- [ ] Verify trigger for auto-creating default config works
- [ ] Test: Create new watch list, verify default heatmap config auto-created

### Backend

- [ ] Build Go binary with heatmap packages
- [ ] Update Kubernetes deployment (no new env vars needed)
- [ ] Deploy to EKS cluster
- [ ] Test heatmap endpoints with Postman/curl
- [ ] Verify real-time price integration in heatmap data
- [ ] Test performance with large watch lists (100+ tickers)

### Frontend

- [ ] Install D3.js: `npm install d3 @types/d3`
- [ ] Build Next.js app (`npm run build`)
- [ ] Deploy to production
- [ ] Test heatmap visualization
- [ ] Verify tooltips work
- [ ] Verify click navigation works
- [ ] Test on different screen sizes

### Monitoring

- [ ] Log heatmap generation (time taken, tile count)
- [ ] Track heatmap config creation rate
- [ ] Monitor API response times for heatmap endpoints
- [ ] Track D3.js rendering performance

---

## Success Criteria

Phase 3 is considered complete when:

- [x] Database schema migrated with heatmap_configs table
- [x] Users can generate heatmap for any watch list
- [x] Users can customize size metric (market cap, volume)
- [x] Users can customize color metric (price change %)
- [x] Users can select time period (1D, 1W, 1M, etc.)
- [x] Users can save custom heatmap configurations
- [x] Users can load saved configurations
- [x] Heatmap displays with interactive tooltips
- [x] Clicking tiles navigates to ticker detail page
- [x] Heatmap updates in real-time
- [x] All manual tests pass
- [x] Deployed to production and tested

---

## Performance Considerations

### Backend Optimization

1. **Cache heatmap data in Redis** (5-minute TTL)
   ```go
   key := fmt.Sprintf("heatmap:%s:%s", watchListID, configID)
   // Check cache first, generate if miss
   ```

2. **Batch price fetching** - Fetch all prices in parallel
   ```go
   // Use goroutines to fetch stock and crypto prices concurrently
   ```

3. **Limit max tiles** - Cap at 200 tickers for rendering performance

4. **Database indexes** - Already created on watch_list_id and user_id

### Frontend Optimization

1. **Debounce config changes** - Wait 500ms before regenerating
2. **Virtualization** - For grid layout with 100+ tiles
3. **Canvas rendering** - Consider canvas instead of SVG for 200+ tiles
4. **Lazy load** - Only render heatmap when tab is active

---

## Next Steps (Phase 4)

After Phase 3, we'll move to Phase 4: Alert System

- Price and volume alerts
- Alert rule CRUD endpoints
- Alert processor worker (CronJob)
- Email notification service
- In-app notification center
- Real-time alert triggering

**Estimated Timeline:** Phase 3 completion by Day 30, ready to start Phase 4.
