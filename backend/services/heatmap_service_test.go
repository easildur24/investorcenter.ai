package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"investorcenter-api/models"
)

// ---------------------------------------------------------------------------
// NewHeatmapService
// ---------------------------------------------------------------------------

func TestNewHeatmapService(t *testing.T) {
	svc := NewHeatmapService()
	require.NotNil(t, svc)
}

// ---------------------------------------------------------------------------
// formatMarketCap — pure function
// ---------------------------------------------------------------------------

func TestFormatMarketCap(t *testing.T) {
	svc := NewHeatmapService()

	tests := []struct {
		name     string
		value    float64
		expected string
	}{
		{"Trillions", 2.5e12, "$2.5T"},
		{"Billions", 150e9, "$150.0B"},
		{"Millions", 500e6, "$500.0M"},
		{"Thousands", 50000, "$50000"},
		{"One trillion", 1e12, "$1.0T"},
		{"One billion", 1e9, "$1.0B"},
		{"One million", 1e6, "$1.0M"},
		{"Small value", 999, "$999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.formatMarketCap(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// formatVolume — pure function
// ---------------------------------------------------------------------------

func TestFormatVolume(t *testing.T) {
	svc := NewHeatmapService()

	tests := []struct {
		name     string
		value    int64
		expected string
	}{
		{"Billions", 2500000000, "2.5B"},
		{"Millions", 150000000, "150.0M"},
		{"Thousands", 500000, "500.0K"},
		{"Small", 999, "999"},
		{"One billion", 1000000000, "1.0B"},
		{"One million", 1000000, "1.0M"},
		{"One thousand", 1000, "1.0K"},
		{"Zero", 0, "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.formatVolume(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// passesFilters — pure function
// ---------------------------------------------------------------------------

func TestPassesFilters_NilFilters(t *testing.T) {
	svc := NewHeatmapService()

	item := &models.WatchListItemDetail{
		WatchListItemWithData: models.WatchListItemWithData{
			WatchListItem: models.WatchListItem{Symbol: "AAPL"},
		},
	}
	assert.True(t, svc.passesFilters(item, nil), "nil filters should pass everything")
}

func TestPassesFilters_EmptyFilters(t *testing.T) {
	svc := NewHeatmapService()

	item := &models.WatchListItemDetail{
		WatchListItemWithData: models.WatchListItemWithData{
			WatchListItem: models.WatchListItem{Symbol: "AAPL"},
		},
	}
	filters := map[string]interface{}{}
	assert.True(t, svc.passesFilters(item, filters), "empty filters should pass everything")
}

func TestPassesFilters_AssetTypeFilter_Pass(t *testing.T) {
	svc := NewHeatmapService()

	item := &models.WatchListItemDetail{
		WatchListItemWithData: models.WatchListItemWithData{
			WatchListItem: models.WatchListItem{Symbol: "AAPL"},
			AssetType:     "CS",
		},
	}
	filters := map[string]interface{}{
		"asset_types": []interface{}{"CS", "ETF"},
	}

	assert.True(t, svc.passesFilters(item, filters))
}

func TestPassesFilters_AssetTypeFilter_Fail(t *testing.T) {
	svc := NewHeatmapService()

	item := &models.WatchListItemDetail{
		WatchListItemWithData: models.WatchListItemWithData{
			WatchListItem: models.WatchListItem{Symbol: "BTC"},
			AssetType:     "crypto",
		},
	}
	filters := map[string]interface{}{
		"asset_types": []interface{}{"CS", "ETF"},
	}

	assert.False(t, svc.passesFilters(item, filters))
}

func TestPassesFilters_PriceRange_Pass(t *testing.T) {
	svc := NewHeatmapService()

	price := 150.0
	item := &models.WatchListItemDetail{
		WatchListItemWithData: models.WatchListItemWithData{
			WatchListItem: models.WatchListItem{Symbol: "AAPL"},
			CurrentPrice:  &price,
		},
	}
	filters := map[string]interface{}{
		"min_price": 100.0,
		"max_price": 200.0,
	}

	assert.True(t, svc.passesFilters(item, filters))
}

func TestPassesFilters_PriceRange_BelowMin(t *testing.T) {
	svc := NewHeatmapService()

	price := 50.0
	item := &models.WatchListItemDetail{
		WatchListItemWithData: models.WatchListItemWithData{
			WatchListItem: models.WatchListItem{Symbol: "PENNY"},
			CurrentPrice:  &price,
		},
	}
	filters := map[string]interface{}{
		"min_price": 100.0,
	}

	assert.False(t, svc.passesFilters(item, filters))
}

func TestPassesFilters_PriceRange_AboveMax(t *testing.T) {
	svc := NewHeatmapService()

	price := 500.0
	item := &models.WatchListItemDetail{
		WatchListItemWithData: models.WatchListItemWithData{
			WatchListItem: models.WatchListItem{Symbol: "EXPENSIVE"},
			CurrentPrice:  &price,
		},
	}
	filters := map[string]interface{}{
		"max_price": 200.0,
	}

	assert.False(t, svc.passesFilters(item, filters))
}

func TestPassesFilters_MarketCapRange_Pass(t *testing.T) {
	svc := NewHeatmapService()

	marketCap := 500e9
	item := &models.WatchListItemDetail{
		WatchListItemWithData: models.WatchListItemWithData{
			WatchListItem: models.WatchListItem{Symbol: "AAPL"},
			MarketCap:     &marketCap,
		},
	}
	filters := map[string]interface{}{
		"min_market_cap": 100e9,
		"max_market_cap": 1e12,
	}

	assert.True(t, svc.passesFilters(item, filters))
}

func TestPassesFilters_MarketCapRange_BelowMin(t *testing.T) {
	svc := NewHeatmapService()

	marketCap := 50e6
	item := &models.WatchListItemDetail{
		WatchListItemWithData: models.WatchListItemWithData{
			WatchListItem: models.WatchListItem{Symbol: "SMALL"},
			MarketCap:     &marketCap,
		},
	}
	filters := map[string]interface{}{
		"min_market_cap": 1e9,
	}

	assert.False(t, svc.passesFilters(item, filters))
}

func TestPassesFilters_NilPrice_SkipsFilter(t *testing.T) {
	svc := NewHeatmapService()

	item := &models.WatchListItemDetail{
		WatchListItemWithData: models.WatchListItemWithData{
			WatchListItem: models.WatchListItem{Symbol: "NODATA"},
			// CurrentPrice is nil
		},
	}
	filters := map[string]interface{}{
		"min_price": 100.0,
		"max_price": 200.0,
	}

	// With nil price, the price filter is simply not applied
	assert.True(t, svc.passesFilters(item, filters))
}

// ---------------------------------------------------------------------------
// calculateSizeValue — pure function
// ---------------------------------------------------------------------------

func TestCalculateSizeValue_MarketCap(t *testing.T) {
	svc := NewHeatmapService()

	marketCap := 2.5e12
	item := &models.WatchListItemDetail{
		WatchListItemWithData: models.WatchListItemWithData{MarketCap: &marketCap},
	}

	value, label := svc.calculateSizeValue(item, "market_cap")
	assert.Equal(t, 2.5e12, value)
	assert.Equal(t, "$2.5T", label)
}

func TestCalculateSizeValue_MarketCapNil(t *testing.T) {
	svc := NewHeatmapService()

	item := &models.WatchListItemDetail{}
	value, label := svc.calculateSizeValue(item, "market_cap")
	assert.Equal(t, float64(1000000000), value) // default
	assert.Equal(t, "N/A", label)
}

func TestCalculateSizeValue_Volume(t *testing.T) {
	svc := NewHeatmapService()

	volume := int64(50000000)
	item := &models.WatchListItemDetail{
		WatchListItemWithData: models.WatchListItemWithData{Volume: &volume},
	}

	value, label := svc.calculateSizeValue(item, "volume")
	assert.Equal(t, float64(50000000), value)
	assert.Equal(t, "50.0M", label)
}

func TestCalculateSizeValue_RedditMentions(t *testing.T) {
	svc := NewHeatmapService()

	mentions := 42
	item := &models.WatchListItemDetail{
		WatchListItemWithData: models.WatchListItemWithData{RedditMentions: &mentions},
	}

	value, label := svc.calculateSizeValue(item, "reddit_mentions")
	assert.Equal(t, float64(42), value)
	assert.Equal(t, "42 mentions", label)
}

func TestCalculateSizeValue_UnknownMetric(t *testing.T) {
	svc := NewHeatmapService()

	item := &models.WatchListItemDetail{}
	value, label := svc.calculateSizeValue(item, "unknown_metric")
	assert.Equal(t, float64(1000000000), value) // defaults to market cap
	assert.Equal(t, "N/A", label)
}

// ---------------------------------------------------------------------------
// calculateColorValue — pure function
// ---------------------------------------------------------------------------

func TestCalculateColorValue_PriceChangePct(t *testing.T) {
	svc := NewHeatmapService()

	changePct := 5.25
	item := &models.WatchListItemDetail{
		WatchListItemWithData: models.WatchListItemWithData{PriceChangePct: &changePct},
	}

	value, label := svc.calculateColorValue(item, "price_change_pct", "1D")
	assert.Equal(t, 5.25, value)
	assert.Equal(t, "+5.25%", label)
}

func TestCalculateColorValue_PriceChangePctNil(t *testing.T) {
	svc := NewHeatmapService()

	item := &models.WatchListItemDetail{}
	value, label := svc.calculateColorValue(item, "price_change_pct", "1D")
	assert.Equal(t, float64(0), value)
	assert.Equal(t, "N/A", label)
}

func TestCalculateColorValue_RedditRank(t *testing.T) {
	svc := NewHeatmapService()

	rank := 5
	item := &models.WatchListItemDetail{
		WatchListItemWithData: models.WatchListItemWithData{RedditRank: &rank},
	}

	value, label := svc.calculateColorValue(item, "reddit_rank", "1D")
	assert.Equal(t, float64(96), value) // 101 - 5 = 96
	assert.Equal(t, "#5", label)
}

func TestCalculateColorValue_RedditTrend(t *testing.T) {
	svc := NewHeatmapService()

	tests := []struct {
		trend     string
		wantValue float64
		wantLabel string
	}{
		{"rising", 10.0, "↑ Rising"},
		{"falling", -10.0, "↓ Falling"},
		{"stable", 0.0, "→ Stable"},
	}

	for _, tt := range tests {
		t.Run(tt.trend, func(t *testing.T) {
			trend := tt.trend
			item := &models.WatchListItemDetail{
				WatchListItemWithData: models.WatchListItemWithData{RedditTrend: &trend},
			}
			value, label := svc.calculateColorValue(item, "reddit_trend", "1D")
			assert.Equal(t, tt.wantValue, value)
			assert.Equal(t, tt.wantLabel, label)
		})
	}
}

func TestCalculateColorValue_UnknownMetric(t *testing.T) {
	svc := NewHeatmapService()

	item := &models.WatchListItemDetail{}
	value, label := svc.calculateColorValue(item, "unknown", "1D")
	assert.Equal(t, float64(0), value)
	assert.Equal(t, "N/A", label)
}
