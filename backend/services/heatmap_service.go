package services

import (
	"fmt"
	"investorcenter-api/database"
	"investorcenter-api/models"
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

	// Get items with data (ticker info, Reddit, screener, alerts)
	items, err := database.GetWatchListItemsWithData(watchListID)
	if err != nil {
		return nil, err
	}

	// Get configuration
	var config *models.HeatmapConfig
	if configID != "" {
		config, err = database.GetHeatmapConfigByID(configID, userID)
		if err != nil {
			return nil, err
		}
	} else {
		config, err = database.GetDefaultHeatmapConfig(watchListID, userID)
		if err != nil {
			// Fall back to sensible in-memory defaults if DB query fails
			// (e.g. heatmap_configs table not yet created)
			config = &models.HeatmapConfig{
				UserID:       userID,
				WatchListID:  watchListID,
				Name:         "Default Heatmap",
				SizeMetric:   "market_cap",
				ColorMetric:  "price_change_pct",
				TimePeriod:   "1D",
				ColorScheme:  "red_green",
				LabelDisplay: "symbol_change",
				LayoutType:   "treemap",
				IsDefault:    true,
			}
		}
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

	// Fetch real-time prices for all tickers
	fetchRealTimePrices(items, fmt.Sprintf("heatmap %s", watchListID))

	// Generate tiles from items
	tiles := make([]models.HeatmapTile, 0, len(items))
	var minColorValue, maxColorValue float64 = math.MaxFloat64, -math.MaxFloat64

	for i := range items {
		item := &items[i]
		if !s.passesFilters(item, config.FiltersJSON) {
			continue
		}

		tile := models.HeatmapTile{
			Symbol:           item.Symbol,
			Name:             item.Name,
			AssetType:        item.AssetType,
			Sector:           item.Sector,
			Exchange:         item.Exchange,
			Notes:            item.Notes,
			Tags:             item.Tags,
			TargetBuyPrice:   item.TargetBuyPrice,
			TargetSellPrice:  item.TargetSellPrice,
			RedditRank:       item.RedditRank,
			RedditMentions:   item.RedditMentions,
			RedditPopularity: item.RedditPopularity,
			RedditTrend:      item.RedditTrend,
			RedditRankChange: item.RedditRankChange,
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
		sizeValue, sizeLabel := s.calculateSizeValue(item, config.SizeMetric)
		tile.SizeValue = sizeValue
		tile.SizeLabel = sizeLabel

		// Calculate color value based on color metric
		colorValue, colorLabel := s.calculateColorValue(item, config.ColorMetric, config.TimePeriod)
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

	// Handle edge case where all values are the same
	if minColorValue == maxColorValue {
		minColorValue = minColorValue - 1
		maxColorValue = maxColorValue + 1
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
	item *models.WatchListItemDetail,
	metric string,
) (float64, string) {
	switch metric {
	case "market_cap":
		if item.MarketCap != nil {
			return *item.MarketCap, s.formatMarketCap(*item.MarketCap)
		}
		// Default market cap if not available
		return 1000000000, "N/A"

	case "volume":
		if item.Volume != nil {
			return float64(*item.Volume), s.formatVolume(*item.Volume)
		}
		return 1000000, "N/A"

	case "avg_volume":
		// Would need historical data - for now use current volume
		if item.Volume != nil {
			return float64(*item.Volume), s.formatVolume(*item.Volume)
		}
		return 1000000, "N/A"

	case "reddit_mentions":
		if item.RedditMentions != nil {
			return float64(*item.RedditMentions), fmt.Sprintf("%d mentions", *item.RedditMentions)
		}
		return 10, "N/A"

	case "reddit_popularity":
		if item.RedditPopularity != nil {
			return *item.RedditPopularity, fmt.Sprintf("%.0f score", *item.RedditPopularity)
		}
		return 10, "N/A"

	default:
		// Default to market cap
		return 1000000000, "N/A"
	}
}

// calculateColorValue determines tile color based on metric
func (s *HeatmapService) calculateColorValue(
	item *models.WatchListItemDetail,
	metric string,
	timePeriod string,
) (float64, string) {
	switch metric {
	case "price_change_pct":
		if item.PriceChangePct != nil {
			return *item.PriceChangePct, fmt.Sprintf("%+.2f%%", *item.PriceChangePct)
		}
		return 0, "N/A"

	case "volume_change_pct":
		// Would need historical volume data
		return 0, "N/A"

	case "reddit_rank":
		// Lower rank = better (1 = #1 trending)
		// Invert for color scale: display as (101 - rank) so higher is greener
		if item.RedditRank != nil {
			invertedRank := 101 - *item.RedditRank
			return float64(invertedRank), fmt.Sprintf("#%d", *item.RedditRank)
		}
		return 0, "N/A"

	case "reddit_trend":
		// Map trend to numeric value: rising = +10, stable = 0, falling = -10
		if item.RedditTrend != nil {
			switch *item.RedditTrend {
			case "rising":
				return 10.0, "↑ Rising"
			case "falling":
				return -10.0, "↓ Falling"
			case "stable":
				return 0.0, "→ Stable"
			}
		}
		return 0, "N/A"

	default:
		return 0, "N/A"
	}
}

// passesFilters checks if item passes filter criteria
func (s *HeatmapService) passesFilters(
	item *models.WatchListItemDetail,
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

// ValidateWatchListOwnership checks if user owns the watch list
func (s *HeatmapService) ValidateWatchListOwnership(watchListID string, userID string) error {
	_, err := database.GetWatchListByID(watchListID, userID)
	return err
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
