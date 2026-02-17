package services

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"investorcenter-api/models"
)

// newTestStockCache creates a StockCache without background updater.
func newTestStockCache() *StockCache {
	return &StockCache{
		cache:    make(map[string]*models.StockPrice),
		stopChan: make(chan bool),
	}
}

// newTestCryptoCache creates a CryptoCache without background updater.
func newTestCryptoCache() *CryptoCache {
	return &CryptoCache{
		cache:    make(map[string]*models.StockPrice),
		stopChan: make(chan bool),
	}
}

// ---------------------------------------------------------------------------
// StockCache — GetPrice
// ---------------------------------------------------------------------------

func TestStockCache_GetPrice_Empty(t *testing.T) {
	sc := newTestStockCache()

	price, exists := sc.GetPrice("AAPL")
	assert.False(t, exists)
	assert.Nil(t, price)
}

func TestStockCache_GetPrice_Exists(t *testing.T) {
	sc := newTestStockCache()

	sc.cache["AAPL"] = &models.StockPrice{
		Symbol: "AAPL",
		Price:  decimal.NewFromFloat(150.25),
		Volume: 50000000,
	}

	price, exists := sc.GetPrice("AAPL")
	require.True(t, exists)
	require.NotNil(t, price)
	assert.Equal(t, "AAPL", price.Symbol)
	assert.True(t, price.Price.Equal(decimal.NewFromFloat(150.25)))
	assert.Equal(t, int64(50000000), price.Volume)
}

func TestStockCache_GetPrice_CaseSensitive(t *testing.T) {
	sc := newTestStockCache()

	sc.cache["AAPL"] = &models.StockPrice{Symbol: "AAPL"}

	_, exists := sc.GetPrice("aapl")
	assert.False(t, exists, "cache keys are case-sensitive")

	_, exists = sc.GetPrice("AAPL")
	assert.True(t, exists)
}

func TestStockCache_GetPrice_MultipleSymbols(t *testing.T) {
	sc := newTestStockCache()

	sc.cache["AAPL"] = &models.StockPrice{Symbol: "AAPL", Price: decimal.NewFromFloat(150.0)}
	sc.cache["GOOGL"] = &models.StockPrice{Symbol: "GOOGL", Price: decimal.NewFromFloat(2800.0)}
	sc.cache["TSLA"] = &models.StockPrice{Symbol: "TSLA", Price: decimal.NewFromFloat(250.0)}

	tests := []struct {
		symbol    string
		exists    bool
		wantPrice float64
	}{
		{"AAPL", true, 150.0},
		{"GOOGL", true, 2800.0},
		{"TSLA", true, 250.0},
		{"MSFT", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.symbol, func(t *testing.T) {
			price, exists := sc.GetPrice(tt.symbol)
			assert.Equal(t, tt.exists, exists)
			if tt.exists {
				require.NotNil(t, price)
				assert.True(t, price.Price.Equal(decimal.NewFromFloat(tt.wantPrice)))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// StockCache — IsMarketHours
// ---------------------------------------------------------------------------

func TestStockCache_IsMarketHours(t *testing.T) {
	sc := newTestStockCache()

	// IsMarketHours depends on wall-clock time, so we just verify it
	// returns a boolean without panicking.
	result := sc.IsMarketHours()
	assert.IsType(t, true, result)
}

// ---------------------------------------------------------------------------
// StockCache — Stop
// ---------------------------------------------------------------------------

func TestStockCache_Stop(t *testing.T) {
	sc := newTestStockCache()

	// Stop should not panic even without a ticker
	assert.NotPanics(t, func() {
		sc.Stop()
	})
}

func TestStockCache_StopWithTicker(t *testing.T) {
	sc := newTestStockCache()
	sc.ticker = time.NewTicker(1 * time.Second)

	assert.NotPanics(t, func() {
		sc.Stop()
	})
}

// ---------------------------------------------------------------------------
// CryptoCache — GetPrice
// ---------------------------------------------------------------------------

func TestCryptoCache_GetPrice_Empty(t *testing.T) {
	cc := newTestCryptoCache()

	price, exists := cc.GetPrice("X:BTCUSD")
	assert.False(t, exists)
	assert.Nil(t, price)
}

func TestCryptoCache_GetPrice_Exists(t *testing.T) {
	cc := newTestCryptoCache()

	cc.cache["X:BTCUSD"] = &models.StockPrice{
		Symbol: "X:BTCUSD",
		Price:  decimal.NewFromFloat(50000.0),
		Volume: 1000000000,
	}

	price, exists := cc.GetPrice("X:BTCUSD")
	require.True(t, exists)
	require.NotNil(t, price)
	assert.Equal(t, "X:BTCUSD", price.Symbol)
	assert.True(t, price.Price.Equal(decimal.NewFromFloat(50000.0)))
}

// ---------------------------------------------------------------------------
// CryptoCache — GetAllPrices
// ---------------------------------------------------------------------------

func TestCryptoCache_GetAllPrices_Empty(t *testing.T) {
	cc := newTestCryptoCache()

	prices := cc.GetAllPrices()
	assert.Empty(t, prices)
}

func TestCryptoCache_GetAllPrices_Multiple(t *testing.T) {
	cc := newTestCryptoCache()

	cc.cache["X:BTCUSD"] = &models.StockPrice{Symbol: "X:BTCUSD", Price: decimal.NewFromFloat(50000.0)}
	cc.cache["X:ETHUSD"] = &models.StockPrice{Symbol: "X:ETHUSD", Price: decimal.NewFromFloat(3000.0)}

	prices := cc.GetAllPrices()
	assert.Len(t, prices, 2)
}

// ---------------------------------------------------------------------------
// CryptoCache — Stop
// ---------------------------------------------------------------------------

func TestCryptoCache_Stop(t *testing.T) {
	cc := newTestCryptoCache()

	assert.NotPanics(t, func() {
		cc.Stop()
	})
}

// ---------------------------------------------------------------------------
// StockPrice model fields via cache
// ---------------------------------------------------------------------------

func TestStockCache_PriceFields(t *testing.T) {
	sc := newTestStockCache()

	now := time.Now()
	sc.cache["MSFT"] = &models.StockPrice{
		Symbol:        "MSFT",
		Price:         decimal.NewFromFloat(375.50),
		Open:          decimal.NewFromFloat(370.00),
		High:          decimal.NewFromFloat(378.00),
		Low:           decimal.NewFromFloat(369.50),
		Close:         decimal.NewFromFloat(375.50),
		Volume:        30000000,
		Change:        decimal.NewFromFloat(5.50),
		ChangePercent: decimal.NewFromFloat(1.49),
		Timestamp:     now,
	}

	price, exists := sc.GetPrice("MSFT")
	require.True(t, exists)
	require.NotNil(t, price)

	assert.True(t, price.Open.Equal(decimal.NewFromFloat(370.00)))
	assert.True(t, price.High.Equal(decimal.NewFromFloat(378.00)))
	assert.True(t, price.Low.Equal(decimal.NewFromFloat(369.50)))
	assert.True(t, price.Change.Equal(decimal.NewFromFloat(5.50)))
	assert.True(t, price.ChangePercent.Equal(decimal.NewFromFloat(1.49)))
	assert.Equal(t, int64(30000000), price.Volume)
}

// ---------------------------------------------------------------------------
// Concurrent access safety (basic test)
// ---------------------------------------------------------------------------

func TestStockCache_ConcurrentAccess(t *testing.T) {
	sc := newTestStockCache()

	sc.cache["AAPL"] = &models.StockPrice{
		Symbol: "AAPL",
		Price:  decimal.NewFromFloat(150.0),
	}

	done := make(chan bool, 10)

	// Multiple concurrent readers
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			_, _ = sc.GetPrice("AAPL")
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
