package services

import (
	"log"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"investorcenter-api/models"
)

// StockCache manages real-time stock price cache
type StockCache struct {
	cache      map[string]*models.StockPrice
	mutex      sync.RWMutex
	lastUpdate time.Time
	polygon    *PolygonClient
	ticker     *time.Ticker
	stopChan   chan bool
}

// CryptoCache manages real-time crypto price cache
type CryptoCache struct {
	cache      map[string]*models.StockPrice
	mutex      sync.RWMutex
	lastUpdate time.Time
	polygon    *PolygonClient
	ticker     *time.Ticker
	stopChan   chan bool
}

// NewStockCache creates a new stock cache instance
func NewStockCache() *StockCache {
	cache := &StockCache{
		cache:    make(map[string]*models.StockPrice),
		polygon:  NewPolygonClient(),
		stopChan: make(chan bool),
	}

	// Load initial data immediately
	log.Println("🚀 Loading initial stock cache...")
	cache.updateCache()

	// Start the background updater
	go cache.startUpdater()

	return cache
}

// GetPrice returns cached price for a symbol
func (sc *StockCache) GetPrice(symbol string) (*models.StockPrice, bool) {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	price, exists := sc.cache[symbol]
	return price, exists
}

// IsMarketHours checks if market is currently open (1am-5pm PST, Mon-Fri)
func (sc *StockCache) IsMarketHours() bool {
	now := time.Now()
	pstTime := now.In(time.FixedZone("PST", -8*60*60)) // PST is UTC-8

	dayOfWeek := pstTime.Weekday()
	hour := pstTime.Hour()

	// Market is closed on weekends
	if dayOfWeek == time.Sunday || dayOfWeek == time.Saturday {
		return false
	}

	// Market hours: 1am (01:00) to 5pm (17:00) PST, Monday-Friday
	return hour >= 1 && hour < 17
}

// startUpdater runs the background cache updater
func (sc *StockCache) startUpdater() {
	sc.ticker = time.NewTicker(5 * time.Second)
	defer sc.ticker.Stop()

	log.Println("📊 Stock cache background updater started")
	lastMarketStatus := sc.IsMarketHours()

	for {
		select {
		case <-sc.ticker.C:
			currentMarketStatus := sc.IsMarketHours()

			if currentMarketStatus {
				sc.updateCache()
			} else if currentMarketStatus != lastMarketStatus {
				// Only log once when market closes
				log.Println("💤 Market closed - cache serving previous data")
			}

			lastMarketStatus = currentMarketStatus
		case <-sc.stopChan:
			log.Println("🛑 Stock cache updater stopped")
			return
		}
	}
}

// updateCache fetches bulk snapshot and updates cache
func (sc *StockCache) updateCache() {
	log.Println("🔄 Updating stock cache from bulk snapshot API...")

	bulkData, err := sc.polygon.GetBulkStockSnapshots()
	if err != nil {
		log.Printf("❌ Failed to fetch bulk snapshots: %v", err)
		return
	}

	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	// Clear old cache and populate with new data
	sc.cache = make(map[string]*models.StockPrice)

	for _, ticker := range bulkData.Tickers {
		// Use last trade price for most recent data
		currentPrice := ticker.LastTrade.Price
		if currentPrice == 0 {
			// Fallback to previous close if no last trade
			currentPrice = ticker.PrevDay.Close
		}

		// Calculate change from previous close
		prevClose := ticker.PrevDay.Close
		change := decimal.NewFromFloat(currentPrice - prevClose)
		changePercent := decimal.Zero
		if prevClose != 0 {
			changePercent = change.Div(decimal.NewFromFloat(prevClose)).Mul(decimal.NewFromInt(100))
		}

		// Use today's day data if available, otherwise previous day
		dayData := ticker.Day
		if dayData.Open == 0 {
			dayData = ticker.PrevDay
		}

		// Convert timestamp
		timestamp := time.Unix(ticker.LastTrade.Timestamp/1000000000, 0)
		if ticker.LastTrade.Timestamp == 0 {
			timestamp = time.Now()
		}

		stockPrice := &models.StockPrice{
			Symbol:        ticker.Ticker,
			Price:         decimal.NewFromFloat(currentPrice),
			Open:          decimal.NewFromFloat(dayData.Open),
			High:          decimal.NewFromFloat(dayData.High),
			Low:           decimal.NewFromFloat(dayData.Low),
			Close:         decimal.NewFromFloat(currentPrice),
			Volume:        int64(dayData.Volume),
			Change:        change,
			ChangePercent: changePercent,
			Timestamp:     timestamp,
		}

		sc.cache[ticker.Ticker] = stockPrice
	}

	sc.lastUpdate = time.Now()
	log.Printf("✅ Stock cache updated with %d tickers", len(sc.cache))
}

// Stop stops the cache updater
func (sc *StockCache) Stop() {
	close(sc.stopChan)
	if sc.ticker != nil {
		sc.ticker.Stop()
	}
}

// NewCryptoCache creates a new crypto cache instance
func NewCryptoCache() *CryptoCache {
	cache := &CryptoCache{
		cache:    make(map[string]*models.StockPrice),
		polygon:  NewPolygonClient(),
		stopChan: make(chan bool),
	}

	// Load initial data immediately
	log.Println("🚀 Loading initial crypto cache...")
	cache.updateCache()

	// Start the background updater (crypto is 24/7)
	go cache.startUpdater()

	return cache
}

// GetPrice returns cached price for a crypto symbol
func (cc *CryptoCache) GetPrice(symbol string) (*models.StockPrice, bool) {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()

	price, exists := cc.cache[symbol]
	return price, exists
}

// GetAllPrices returns all cached crypto prices
func (cc *CryptoCache) GetAllPrices() []*models.StockPrice {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()

	prices := make([]*models.StockPrice, 0, len(cc.cache))
	for _, price := range cc.cache {
		prices = append(prices, price)
	}
	return prices
}

// startUpdater runs the background crypto cache updater (24/7)
func (cc *CryptoCache) startUpdater() {
	cc.ticker = time.NewTicker(5 * time.Second)
	defer cc.ticker.Stop()

	log.Println("📊 Crypto cache background updater started (24/7)")

	for {
		select {
		case <-cc.ticker.C:
			cc.updateCache()
		case <-cc.stopChan:
			log.Println("🛑 Crypto cache updater stopped")
			return
		}
	}
}

// updateCache fetches bulk crypto snapshot and updates cache
func (cc *CryptoCache) updateCache() {
	log.Println("🔄 Updating crypto cache from bulk snapshot API...")

	bulkData, err := cc.polygon.GetBulkCryptoSnapshots()
	if err != nil {
		log.Printf("❌ Failed to fetch bulk crypto snapshots: %v", err)
		return
	}

	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	// Clear old cache and populate with new data
	cc.cache = make(map[string]*models.StockPrice)

	for _, ticker := range bulkData.Tickers {
		// Use last trade price for most recent data
		currentPrice := ticker.LastTrade.Price
		if currentPrice == 0 {
			currentPrice = ticker.Day.Close
		}

		// Calculate change from previous close
		change := decimal.NewFromFloat(ticker.TodaysChange)
		changePercent := decimal.NewFromFloat(ticker.TodaysChangePerc)

		// Convert timestamp
		timestamp := time.Unix(ticker.LastTrade.Timestamp/1000000000, 0)
		if ticker.LastTrade.Timestamp == 0 {
			timestamp = time.Now()
		}

		stockPrice := &models.StockPrice{
			Symbol:        ticker.Ticker,
			Price:         decimal.NewFromFloat(currentPrice),
			Open:          decimal.NewFromFloat(ticker.Day.Open),
			High:          decimal.NewFromFloat(ticker.Day.High),
			Low:           decimal.NewFromFloat(ticker.Day.Low),
			Close:         decimal.NewFromFloat(currentPrice),
			Volume:        int64(ticker.Day.Volume),
			Change:        change,
			ChangePercent: changePercent,
			Timestamp:     timestamp,
		}

		cc.cache[ticker.Ticker] = stockPrice
	}

	cc.lastUpdate = time.Now()
	log.Printf("✅ Crypto cache updated with %d tickers", len(cc.cache))
}

// Stop stops the crypto cache updater
func (cc *CryptoCache) Stop() {
	close(cc.stopChan)
	if cc.ticker != nil {
		cc.ticker.Stop()
	}
}

// Global cache instances
var globalStockCache *StockCache
var globalCryptoCache *CryptoCache
var stockCacheOnce sync.Once
var cryptoCacheOnce sync.Once

// GetStockCache returns the global stock cache instance
func GetStockCache() *StockCache {
	stockCacheOnce.Do(func() {
		globalStockCache = NewStockCache()
	})
	return globalStockCache
}

// GetCryptoCache returns the global crypto cache instance
func GetCryptoCache() *CryptoCache {
	cryptoCacheOnce.Do(func() {
		globalCryptoCache = NewCryptoCache()
	})
	return globalCryptoCache
}
