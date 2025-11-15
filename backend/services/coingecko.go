package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"investorcenter-api/models"
)

var (
	CoinGeckoBaseURL = "https://api.coingecko.com/api/v3"
)

// CoinGeckoClient handles CoinGecko API requests
type CoinGeckoClient struct {
	APIKey string
	Client *http.Client
}

// NewCoinGeckoClient creates a new CoinGecko API client
func NewCoinGeckoClient() *CoinGeckoClient {
	// API key is optional for free tier
	// For higher rate limits, set COINGECKO_API_KEY environment variable
	return &CoinGeckoClient{
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// MapSymbolToCoinGeckoID maps ticker symbols to CoinGecko IDs
func (c *CoinGeckoClient) MapSymbolToCoinGeckoID(symbol string) string {
	// Map common crypto symbols to their CoinGecko IDs
	symbolMap := map[string]string{
		"BTC":      "bitcoin",
		"ETH":      "ethereum",
		"SOL":      "solana",
		"ADA":      "cardano",
		"XRP":      "ripple",
		"DOT":      "polkadot",
		"DOGE":     "dogecoin",
		"MATIC":    "matic-network",
		"AVAX":     "avalanche-2",
		"LINK":     "chainlink",
		"UNI":      "uniswap",
		"LTC":      "litecoin",
		"BCH":      "bitcoin-cash",
		"ATOM":     "cosmos",
		"ETC":      "ethereum-classic",
		"XLM":      "stellar",
		"ALGO":     "algorand",
		"VET":      "vechain",
		"FIL":      "filecoin",
		"TRX":      "tron",
		"APT":      "aptos",
		"ARB":      "arbitrum",
		"OP":       "optimism",
		"NEAR":     "near",
		"STX":      "blockstack",
		"INJ":      "injective-protocol",
		"SUI":      "sui",
		"SEI":      "sei-network",
		"WIF":      "dogwifcoin",
		"BONK":     "bonk",
		"PEPE":     "pepe",
		"SHIB":     "shiba-inu",
		"FLOKI":    "floki",
		"FARTCOIN": "fartcoin",
		"BNB":      "binancecoin",
		"USDT":     "tether",
		"USDC":     "usd-coin",
		"DAI":      "dai",
	}

	// Check if we have a mapping
	if id, ok := symbolMap[strings.ToUpper(symbol)]; ok {
		return id
	}

	// Default: lowercase symbol (works for many coins)
	return strings.ToLower(symbol)
}

// MarketChartResponse represents CoinGecko market_chart API response
type MarketChartResponse struct {
	Prices       [][]float64 `json:"prices"`        // [[timestamp_ms, price], ...]
	MarketCaps   [][]float64 `json:"market_caps"`   // [[timestamp_ms, market_cap], ...]
	TotalVolumes [][]float64 `json:"total_volumes"` // [[timestamp_ms, volume], ...]
}

// GetMarketChart fetches historical price data from CoinGecko
func (c *CoinGeckoClient) GetMarketChart(symbol string, days int) ([]models.ChartDataPoint, error) {
	coinID := c.MapSymbolToCoinGeckoID(symbol)

	// Build API URL
	url := fmt.Sprintf("%s/coins/%s/market_chart?vs_currency=usd&days=%d",
		CoinGeckoBaseURL, coinID, days)

	log.Printf("Fetching CoinGecko market chart for %s (id: %s, days: %d)", symbol, coinID, days)

	// Make request
	resp, err := c.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("CoinGecko API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("CoinGecko API error: status %d", resp.StatusCode)
	}

	// Parse response
	var result MarketChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse CoinGecko response: %w", err)
	}

	// Convert to ChartDataPoint format
	dataPoints := make([]models.ChartDataPoint, 0, len(result.Prices))

	for i, priceData := range result.Prices {
		if len(priceData) < 2 {
			continue
		}

		// Parse timestamp (milliseconds)
		timestamp := time.Unix(int64(priceData[0])/1000, 0)
		price := decimal.NewFromFloat(priceData[1])

		// For crypto, we don't have OHLC data from this endpoint
		// So we use the price for all OHLC values
		dataPoint := models.ChartDataPoint{
			Timestamp: timestamp,
			Open:      price,
			High:      price,
			Low:       price,
			Close:     price,
		}

		// Add volume if available
		if i < len(result.TotalVolumes) && len(result.TotalVolumes[i]) >= 2 {
			volume := decimal.NewFromFloat(result.TotalVolumes[i][1])
			dataPoint.Volume = volume
		}

		dataPoints = append(dataPoints, dataPoint)
	}

	log.Printf("✓ Got %d data points for %s from CoinGecko", len(dataPoints), symbol)
	return dataPoints, nil
}

// OHLCResponse represents CoinGecko OHLC API response
type OHLCResponse [][]float64 // [[timestamp_ms, open, high, low, close], ...]

// GetOHLC fetches OHLC (candlestick) data from CoinGecko
// Available for 1/7/14/30/90/180/365 days
func (c *CoinGeckoClient) GetOHLC(symbol string, days int) ([]models.ChartDataPoint, error) {
	coinID := c.MapSymbolToCoinGeckoID(symbol)

	// CoinGecko OHLC endpoint only supports specific day values
	// Map our days to supported values
	supportedDays := []int{1, 7, 14, 30, 90, 180, 365}
	validDays := 365 // default

	for _, d := range supportedDays {
		if days <= d {
			validDays = d
			break
		}
	}

	// Build API URL
	url := fmt.Sprintf("%s/coins/%s/ohlc?vs_currency=usd&days=%d",
		CoinGeckoBaseURL, coinID, validDays)

	log.Printf("Fetching CoinGecko OHLC for %s (id: %s, days: %d->%d)",
		symbol, coinID, days, validDays)

	// Make request
	resp, err := c.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("CoinGecko OHLC API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// Fall back to market_chart if OHLC fails
		log.Printf("OHLC failed with status %d, falling back to market_chart", resp.StatusCode)
		return c.GetMarketChart(symbol, days)
	}

	// Parse response
	var result OHLCResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse CoinGecko OHLC response: %w", err)
	}

	// Convert to ChartDataPoint format
	dataPoints := make([]models.ChartDataPoint, 0, len(result))

	for _, ohlc := range result {
		if len(ohlc) < 5 {
			continue
		}

		// Parse data: [timestamp_ms, open, high, low, close]
		timestamp := time.Unix(int64(ohlc[0])/1000, 0)

		dataPoint := models.ChartDataPoint{
			Timestamp: timestamp,
			Open:      decimal.NewFromFloat(ohlc[1]),
			High:      decimal.NewFromFloat(ohlc[2]),
			Low:       decimal.NewFromFloat(ohlc[3]),
			Close:     decimal.NewFromFloat(ohlc[4]),
		}

		dataPoints = append(dataPoints, dataPoint)
	}

	log.Printf("✓ Got %d OHLC data points for %s from CoinGecko", len(dataPoints), symbol)
	return dataPoints, nil
}

// GetChartData is a convenience method that chooses the best endpoint based on period
func (c *CoinGeckoClient) GetChartData(symbol string, period string) ([]models.ChartDataPoint, error) {
	days := GetDaysFromPeriod(period)

	// For 1 day, use market_chart with more granular data
	if days == 1 {
		return c.GetMarketChart(symbol, days)
	}

	// For longer periods, try OHLC first (better for candlestick charts)
	// If it fails, it will automatically fall back to market_chart
	return c.GetOHLC(symbol, days)
}
