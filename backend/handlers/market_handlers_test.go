package handlers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// MoversCache — get / set
// ---------------------------------------------------------------------------

func TestMoversCache_GetEmpty(t *testing.T) {
	cache := &MoversCache{cacheTTL: 5 * time.Minute}
	result := cache.get()
	assert.Nil(t, result, "empty cache should return nil")
}

func TestMoversCache_SetAndGet(t *testing.T) {
	cache := &MoversCache{cacheTTL: 5 * time.Minute}

	data := &MoversData{
		Gainers: []MoverStock{
			{Symbol: "AAPL", Price: 150.0, ChangePercent: 5.0},
			{Symbol: "MSFT", Price: 375.0, ChangePercent: 3.5},
		},
		Losers: []MoverStock{
			{Symbol: "TSLA", Price: 250.0, ChangePercent: -4.0},
		},
		MostActive: []MoverStock{
			{Symbol: "NVDA", Price: 800.0, Volume: 50000000},
		},
	}

	cache.set(data)

	result := cache.get()
	require.NotNil(t, result)
	assert.Len(t, result.Gainers, 2)
	assert.Len(t, result.Losers, 1)
	assert.Len(t, result.MostActive, 1)
	assert.Equal(t, "AAPL", result.Gainers[0].Symbol)
}

func TestMoversCache_Expiry(t *testing.T) {
	// Create cache with very short TTL
	cache := &MoversCache{cacheTTL: 1 * time.Millisecond}

	data := &MoversData{
		Gainers: []MoverStock{{Symbol: "AAPL"}},
	}

	cache.set(data)

	// Wait for expiry
	time.Sleep(5 * time.Millisecond)

	result := cache.get()
	assert.Nil(t, result, "expired cache should return nil")
}

func TestMoversCache_NotExpired(t *testing.T) {
	cache := &MoversCache{cacheTTL: 1 * time.Hour}

	data := &MoversData{
		Gainers: []MoverStock{{Symbol: "AAPL"}},
	}

	cache.set(data)

	result := cache.get()
	require.NotNil(t, result)
	assert.Len(t, result.Gainers, 1)
}

func TestMoversCache_Overwrite(t *testing.T) {
	cache := &MoversCache{cacheTTL: 5 * time.Minute}

	data1 := &MoversData{
		Gainers: []MoverStock{{Symbol: "AAPL"}},
	}
	cache.set(data1)

	data2 := &MoversData{
		Gainers: []MoverStock{{Symbol: "MSFT"}, {Symbol: "GOOGL"}},
	}
	cache.set(data2)

	result := cache.get()
	require.NotNil(t, result)
	assert.Len(t, result.Gainers, 2)
	assert.Equal(t, "MSFT", result.Gainers[0].Symbol)
}

// ---------------------------------------------------------------------------
// MoverStock struct
// ---------------------------------------------------------------------------

func TestMoverStock_Fields(t *testing.T) {
	stock := MoverStock{
		Symbol:        "AAPL",
		Name:          "Apple Inc.",
		Price:         150.25,
		Change:        5.50,
		ChangePercent: 3.8,
		Volume:        50000000,
	}

	assert.Equal(t, "AAPL", stock.Symbol)
	assert.Equal(t, "Apple Inc.", stock.Name)
	assert.Equal(t, 150.25, stock.Price)
	assert.Equal(t, 5.50, stock.Change)
	assert.Equal(t, 3.8, stock.ChangePercent)
	assert.Equal(t, float64(50000000), stock.Volume)
}

// ---------------------------------------------------------------------------
// IndexInfo struct
// ---------------------------------------------------------------------------

func TestIndexInfo_Fields(t *testing.T) {
	info := IndexInfo{
		Symbol:        "SPY",
		Name:          "S&P 500",
		Price:         4500.25,
		Change:        25.50,
		ChangePercent: 0.57,
		LastUpdated:   "2024-01-15T16:00:00Z",
	}

	assert.Equal(t, "SPY", info.Symbol)
	assert.Equal(t, "S&P 500", info.Name)
	assert.Equal(t, 4500.25, info.Price)
	assert.Equal(t, 25.50, info.Change)
	assert.Equal(t, 0.57, info.ChangePercent)
}

// ---------------------------------------------------------------------------
// MoversData struct
// ---------------------------------------------------------------------------

func TestMoversData_EmptyLists(t *testing.T) {
	data := MoversData{
		Gainers:    []MoverStock{},
		Losers:     []MoverStock{},
		MostActive: []MoverStock{},
	}

	assert.Empty(t, data.Gainers)
	assert.Empty(t, data.Losers)
	assert.Empty(t, data.MostActive)
}
