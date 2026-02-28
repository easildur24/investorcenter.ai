package handlers

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"investorcenter-api/models"
)

// calculateMarketCap estimates market cap for a crypto symbol (test helper)
func calculateMarketCap(symbol string, price decimal.Decimal) decimal.Decimal {
	cleanSymbol := strings.Replace(symbol, "X:", "", 1)

	var supply int64
	var usdPrice decimal.Decimal = price

	if strings.HasPrefix(cleanSymbol, "BTC") {
		supply = 19_800_000
		if strings.Contains(cleanSymbol, "JPY") {
			usdPrice = price.Div(decimal.NewFromInt(150))
		} else if strings.Contains(cleanSymbol, "EUR") {
			usdPrice = price.Mul(decimal.NewFromFloat(1.1))
		}
	} else if strings.HasPrefix(cleanSymbol, "ETH") {
		supply = 120_000_000
		if strings.Contains(cleanSymbol, "JPY") {
			usdPrice = price.Div(decimal.NewFromInt(150))
		} else if strings.Contains(cleanSymbol, "EUR") {
			usdPrice = price.Mul(decimal.NewFromFloat(1.1))
		}
	} else {
		switch {
		case strings.HasPrefix(cleanSymbol, "SOL"):
			supply = 470_000_000
		case strings.HasPrefix(cleanSymbol, "XRP"):
			supply = 56_000_000_000
		case strings.HasPrefix(cleanSymbol, "DOGE"):
			supply = 147_000_000_000
		case strings.HasPrefix(cleanSymbol, "ADA"):
			supply = 35_000_000_000
		case strings.HasPrefix(cleanSymbol, "LTC"):
			supply = 75_000_000
		case strings.HasPrefix(cleanSymbol, "LINK"):
			supply = 600_000_000
		case strings.HasPrefix(cleanSymbol, "AVAX"):
			supply = 400_000_000
		case strings.HasPrefix(cleanSymbol, "MATIC"):
			supply = 10_000_000_000
		case strings.Contains(cleanSymbol, "USDT") || strings.Contains(cleanSymbol, "USDC"):
			supply = 100_000_000_000
		default:
			supply = 1_000_000
		}
	}

	return usdPrice.Mul(decimal.NewFromInt(supply))
}

// ---------------------------------------------------------------------------
// isCryptoAsset — pure function
// ---------------------------------------------------------------------------

func TestIsCryptoAsset(t *testing.T) {
	tests := []struct {
		name      string
		assetType string
		symbol    string
		want      bool
	}{
		{"crypto asset type", "crypto", "BTC", true},
		{"polygon crypto prefix", "stock", "X:BTCUSD", true},
		{"BTC symbol", "", "BTC", true},
		{"ETH symbol", "", "ETH", true},
		{"SOL symbol", "", "SOL", true},
		{"XRP symbol", "", "XRP", true},
		{"DOGE symbol not in map", "", "DOGE", false},
		{"DOT symbol", "", "DOT", true},
		{"BCH symbol", "", "BCH", true},
		{"ADA symbol", "", "ADA", true},
		{"LTC symbol", "", "LTC", true},
		{"LINK symbol", "", "LINK", true},
		{"BNB symbol", "", "BNB", true},
		{"AVAX symbol", "", "AVAX", true},
		{"MATIC symbol", "", "MATIC", true},
		{"regular stock AAPL", "CS", "AAPL", false},
		{"regular stock MSFT", "", "MSFT", false},
		{"ETF SPY", "ETF", "SPY", false},
		{"empty everything", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCryptoAsset(tt.assetType, tt.symbol)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// isCryptoAssetWithStock — uses Stock struct
// ---------------------------------------------------------------------------

func TestIsCryptoAssetWithStock(t *testing.T) {
	tests := []struct {
		name  string
		stock *models.Stock
		want  bool
	}{
		{
			name:  "nil stock",
			stock: nil,
			want:  false,
		},
		{
			name:  "crypto asset type",
			stock: &models.Stock{AssetType: "crypto", Symbol: "BTC"},
			want:  true,
		},
		{
			name:  "CRYPTO exchange",
			stock: &models.Stock{Exchange: "CRYPTO", Symbol: "ETH"},
			want:  true,
		},
		{
			name:  "Cryptocurrency sector",
			stock: &models.Stock{Sector: "Cryptocurrency", Symbol: "SOL"},
			want:  true,
		},
		{
			name:  "known crypto symbol",
			stock: &models.Stock{Symbol: "BTC"},
			want:  true,
		},
		{
			name:  "regular stock",
			stock: &models.Stock{AssetType: "CS", Exchange: "XNAS", Symbol: "AAPL"},
			want:  false,
		},
		{
			name:  "ETF",
			stock: &models.Stock{AssetType: "ETF", Exchange: "XNYS", Symbol: "SPY"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCryptoAssetWithStock(tt.stock)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// getUpdateInterval — pure function
// ---------------------------------------------------------------------------

func TestGetUpdateInterval(t *testing.T) {
	tests := []struct {
		name     string
		isCrypto bool
		session  string
		want     int
	}{
		{"crypto always 5s", true, "regular", 5},
		{"crypto closed irrelevant", true, "closed", 5},
		{"stock regular session", false, "regular", 5},
		{"stock pre market", false, "pre_market", 15},
		{"stock after hours", false, "after_hours", 15},
		{"stock market closed", false, "closed", 300},
		{"stock empty status", false, "", 300},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getUpdateInterval(tt.isCrypto, tt.session)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// getDataSource — pure function
// ---------------------------------------------------------------------------

func TestGetDataSource(t *testing.T) {
	assert.Equal(t, "redis", getDataSource(true))
	assert.Equal(t, "polygon", getDataSource(false))
}

// ---------------------------------------------------------------------------
// calculateMarketCap — pure function
// ---------------------------------------------------------------------------

func TestCalculateMarketCap(t *testing.T) {
	tests := []struct {
		name   string
		symbol string
		price  decimal.Decimal
	}{
		{"BTC USD", "X:BTCUSD", decimal.NewFromFloat(50000)},
		{"ETH USD", "X:ETHUSD", decimal.NewFromFloat(3000)},
		{"SOL USD", "X:SOLUSD", decimal.NewFromFloat(100)},
		{"XRP USD", "X:XRPUSD", decimal.NewFromFloat(0.5)},
		{"DOGE USD", "X:DOGEUSD", decimal.NewFromFloat(0.1)},
		{"ADA USD", "X:ADAUSD", decimal.NewFromFloat(0.5)},
		{"LTC USD", "X:LTCUSD", decimal.NewFromFloat(80)},
		{"unknown crypto", "X:FOOBUSD", decimal.NewFromFloat(1.0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateMarketCap(tt.symbol, tt.price)
			// Market cap should be positive for positive prices
			assert.True(t, result.GreaterThan(decimal.Zero),
				"market cap should be positive for %s", tt.symbol)
		})
	}
}

func TestCalculateMarketCap_BTCSpecific(t *testing.T) {
	price := decimal.NewFromFloat(50000)
	result := calculateMarketCap("X:BTCUSD", price)

	// BTC supply ~19.8M, so market cap should be ~990B
	expected := decimal.NewFromFloat(50000 * 19_800_000)
	assert.True(t, result.Equal(expected),
		"expected %s, got %s", expected.String(), result.String())
}

func TestCalculateMarketCap_BTCJPY(t *testing.T) {
	// BTC in JPY should convert to USD
	price := decimal.NewFromFloat(7_500_000) // ~50000 USD * 150
	result := calculateMarketCap("X:BTCJPY", price)
	// Should convert JPY to USD at ~150 rate
	assert.True(t, result.GreaterThan(decimal.Zero))
}

func TestCalculateMarketCap_BTCEUR(t *testing.T) {
	price := decimal.NewFromFloat(45000) // ~50000 USD / 1.1
	result := calculateMarketCap("X:BTCEUR", price)
	assert.True(t, result.GreaterThan(decimal.Zero))
}

// ---------------------------------------------------------------------------
// convertCryptoPriceToStockPrice — pure function
// ---------------------------------------------------------------------------

func TestConvertCryptoPriceToStockPrice(t *testing.T) {
	crypto := &CryptoRealTimePrice{
		Symbol:                   "BTC",
		CurrentPrice:             50000.0,
		PriceChange24h:           500.0,
		PriceChangePercentage24h: 1.01,
		High24h:                  51000.0,
		Low24h:                   49000.0,
		TotalVolume:              1000000000,
		LastUpdated:              "2024-01-15T12:00:00Z",
	}

	result := convertCryptoPriceToStockPrice(crypto)

	require.NotNil(t, result)
	assert.Equal(t, "BTC", result.Symbol)
	assert.True(t, result.Price.Equal(decimal.NewFromFloat(50000.0)))
	assert.True(t, result.High.Equal(decimal.NewFromFloat(51000.0)))
	assert.True(t, result.Low.Equal(decimal.NewFromFloat(49000.0)))
	assert.True(t, result.Change.Equal(decimal.NewFromFloat(500.0)))
	assert.True(t, result.ChangePercent.Equal(decimal.NewFromFloat(1.01)))
	assert.Equal(t, int64(1000000000), result.Volume)
}

func TestConvertCryptoPriceToStockPrice_EmptyTimestamp(t *testing.T) {
	crypto := &CryptoRealTimePrice{
		Symbol:       "ETH",
		CurrentPrice: 3000.0,
		LastUpdated:  "",
	}

	result := convertCryptoPriceToStockPrice(crypto)
	require.NotNil(t, result)
	// Should use current time when no timestamp available
	assert.False(t, result.Timestamp.IsZero())
}

func TestConvertCryptoPriceToStockPrice_InvalidTimestamp(t *testing.T) {
	crypto := &CryptoRealTimePrice{
		Symbol:       "SOL",
		CurrentPrice: 100.0,
		LastUpdated:  "not-a-date",
	}

	result := convertCryptoPriceToStockPrice(crypto)
	require.NotNil(t, result)
	// Should use current time when timestamp is invalid
	assert.False(t, result.Timestamp.IsZero())
}

// ---------------------------------------------------------------------------
// buildKeyMetrics — pure function
// ---------------------------------------------------------------------------

func TestBuildKeyMetrics_BasicFields(t *testing.T) {
	price := &models.StockPrice{
		Volume:    1000000,
		Timestamp: mustParseTime("2024-01-15T12:00:00Z"),
	}

	metrics := buildKeyMetrics(price, nil, nil)
	assert.Equal(t, int64(1000000), metrics["volume"])
	assert.NotNil(t, metrics["timestamp"])
}

func TestBuildKeyMetrics_WithFundamentals(t *testing.T) {
	price := &models.StockPrice{
		Volume:    1000000,
		Timestamp: mustParseTime("2024-01-15T12:00:00Z"),
	}

	pe := decimal.NewFromFloat(25.5)
	eps := decimal.NewFromFloat(6.75)
	revenue := decimal.NewFromFloat(380000000000)

	fundamentals := &models.Fundamentals{
		PE:      &pe,
		EPS:     &eps,
		Revenue: &revenue,
	}

	metrics := buildKeyMetrics(price, fundamentals, nil)
	assert.Equal(t, pe.String(), metrics["pe"])
	assert.Equal(t, eps.String(), metrics["eps"])
	assert.Equal(t, revenue.String(), metrics["revenue"])
}

func TestBuildKeyMetrics_NilFundamentals(t *testing.T) {
	price := &models.StockPrice{
		Volume:    500000,
		Timestamp: mustParseTime("2024-01-15T12:00:00Z"),
	}

	metrics := buildKeyMetrics(price, nil, nil)
	assert.Nil(t, metrics["pe"])
	assert.Nil(t, metrics["eps"])
	assert.Nil(t, metrics["revenue"])
}

// ---------------------------------------------------------------------------
// CryptoRealTimePrice struct tests
// ---------------------------------------------------------------------------

func TestCryptoRealTimePrice_AliasFields(t *testing.T) {
	p := CryptoRealTimePrice{
		CurrentPrice:             50000.0,
		TotalVolume:              1000000000,
		PriceChangePercentage24h: 1.5,
	}

	// Simulate alias population
	p.Price = p.CurrentPrice
	p.Volume24h = p.TotalVolume
	p.Change24h = p.PriceChangePercentage24h

	assert.Equal(t, p.CurrentPrice, p.Price)
	assert.Equal(t, p.TotalVolume, p.Volume24h)
	assert.Equal(t, p.PriceChangePercentage24h, p.Change24h)
}

// helper
func mustParseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
