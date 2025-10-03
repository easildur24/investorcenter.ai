package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// VolumeData represents real-time volume and price data
type VolumeData struct {
	Symbol         string    `json:"symbol"`
	Volume         int64     `json:"volume"`
	VolumeWeighted float64   `json:"vwap"`
	Open           float64   `json:"open"`
	Close          float64   `json:"close"`
	High           float64   `json:"high"`
	Low            float64   `json:"low"`
	Timestamp      int64     `json:"timestamp"`
	Transactions   int       `json:"transactions"`
	PrevClose      float64   `json:"prevClose"`
	Change         float64   `json:"change"`
	ChangePercent  float64   `json:"changePercent"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// VolumeAggregates represents historical volume aggregates
type VolumeAggregates struct {
	Symbol       string  `json:"symbol"`
	AvgVolume30D int64   `json:"avgVolume30d"`
	AvgVolume90D int64   `json:"avgVolume90d"`
	Week52High   float64 `json:"week52High"`
	Week52Low    float64 `json:"week52Low"`
	VolumeTrend  string  `json:"volumeTrend"` // "increasing", "decreasing", "stable"
}

// VolumeService handles volume data from Polygon API
type VolumeService struct {
	apiKey      string
	baseURL     string
	httpClient  *http.Client
	cache       map[string]*cachedVolume
	cacheMutex  sync.RWMutex
	cacheExpiry time.Duration
}

type cachedVolume struct {
	data      *VolumeData
	timestamp time.Time
}

// NewVolumeService creates a new volume service instance
func NewVolumeService() *VolumeService {
	apiKey := os.Getenv("POLYGON_API_KEY")
	if apiKey == "" {
		apiKey = "demo" // Fallback to demo key
	}

	return &VolumeService{
		apiKey:      apiKey,
		baseURL:     "https://api.polygon.io",
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		cache:       make(map[string]*cachedVolume),
		cacheExpiry: 1 * time.Minute, // Cache for 1 minute
	}
}

// GetRealTimeVolume fetches real-time volume data from Polygon
func (vs *VolumeService) GetRealTimeVolume(symbol string) (*VolumeData, error) {
	// Check cache first
	if cached := vs.getFromCache(symbol); cached != nil {
		return cached, nil
	}

	// Fetch from Polygon API
	url := fmt.Sprintf("%s/v2/snapshot/locale/us/markets/stocks/tickers/%s", vs.baseURL, symbol)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("apiKey", vs.apiKey)
	req.URL.RawQuery = q.Encode()

	resp, err := vs.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limit exceeded")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var result struct {
		Status string `json:"status"`
		Ticker struct {
			Day struct {
				O  float64 `json:"o"`
				H  float64 `json:"h"`
				L  float64 `json:"l"`
				C  float64 `json:"c"`
				V  int64   `json:"v"`
				VW float64 `json:"vw"`
			} `json:"day"`
			PrevDay struct {
				C float64 `json:"c"`
			} `json:"prevDay"`
			Updated int64 `json:"updated"`
		} `json:"ticker"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != "OK" {
		return nil, fmt.Errorf("API returned non-OK status: %s", result.Status)
	}

	// Calculate change and change percent
	change := result.Ticker.Day.C - result.Ticker.PrevDay.C
	changePercent := 0.0
	if result.Ticker.PrevDay.C > 0 {
		changePercent = (change / result.Ticker.PrevDay.C) * 100
	}

	volumeData := &VolumeData{
		Symbol:         symbol,
		Volume:         result.Ticker.Day.V,
		VolumeWeighted: result.Ticker.Day.VW,
		Open:           result.Ticker.Day.O,
		Close:          result.Ticker.Day.C,
		High:           result.Ticker.Day.H,
		Low:            result.Ticker.Day.L,
		Timestamp:      result.Ticker.Updated,
		PrevClose:      result.Ticker.PrevDay.C,
		Change:         change,
		ChangePercent:  changePercent,
		UpdatedAt:      time.Now(),
	}

	// Store in cache
	vs.storeInCache(symbol, volumeData)

	return volumeData, nil
}

// GetVolumeAggregates fetches historical volume aggregates
func (vs *VolumeService) GetVolumeAggregates(symbol string, days int) (*VolumeAggregates, error) {
	to := time.Now().Format("2006-01-02")
	from := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	url := fmt.Sprintf("%s/v2/aggs/ticker/%s/range/1/day/%s/%s", vs.baseURL, symbol, from, to)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("apiKey", vs.apiKey)
	q.Add("adjusted", "true")
	q.Add("sort", "desc")
	q.Add("limit", fmt.Sprintf("%d", days))
	req.URL.RawQuery = q.Encode()

	resp, err := vs.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var result struct {
		Status  string `json:"status"`
		Results []struct {
			V float64 `json:"v"` // Volume
			H float64 `json:"h"` // High
			L float64 `json:"l"` // Low
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Results) == 0 {
		return nil, fmt.Errorf("no data available")
	}

	// Calculate aggregates
	var totalVolume30d, totalVolume90d int64
	var week52High, week52Low float64
	count30d, count90d := 0, 0

	week52High = result.Results[0].H
	week52Low = result.Results[0].L

	for i, bar := range result.Results {
		if i < 30 {
			totalVolume30d += int64(bar.V)
			count30d++
		}
		if i < 90 {
			totalVolume90d += int64(bar.V)
			count90d++
		}

		if bar.H > week52High {
			week52High = bar.H
		}
		if bar.L < week52Low {
			week52Low = bar.L
		}
	}

	avgVolume30d := int64(0)
	if count30d > 0 {
		avgVolume30d = totalVolume30d / int64(count30d)
	}

	avgVolume90d := int64(0)
	if count90d > 0 {
		avgVolume90d = totalVolume90d / int64(count90d)
	}

	// Determine volume trend
	trend := "stable"
	if count30d > 0 && count90d > 0 {
		if float64(avgVolume30d) > float64(avgVolume90d)*1.2 {
			trend = "increasing"
		} else if float64(avgVolume30d) < float64(avgVolume90d)*0.8 {
			trend = "decreasing"
		}
	}

	return &VolumeAggregates{
		Symbol:       symbol,
		AvgVolume30D: avgVolume30d,
		AvgVolume90D: avgVolume90d,
		Week52High:   week52High,
		Week52Low:    week52Low,
		VolumeTrend:  trend,
	}, nil
}

// Cache management functions
func (vs *VolumeService) getFromCache(symbol string) *VolumeData {
	vs.cacheMutex.RLock()
	defer vs.cacheMutex.RUnlock()

	if cached, exists := vs.cache[symbol]; exists {
		if time.Since(cached.timestamp) < vs.cacheExpiry {
			return cached.data
		}
	}
	return nil
}

func (vs *VolumeService) storeInCache(symbol string, data *VolumeData) {
	vs.cacheMutex.Lock()
	defer vs.cacheMutex.Unlock()

	vs.cache[symbol] = &cachedVolume{
		data:      data,
		timestamp: time.Now(),
	}
}

// ClearCache clears expired cache entries
func (vs *VolumeService) ClearCache() {
	vs.cacheMutex.Lock()
	defer vs.cacheMutex.Unlock()

	now := time.Now()
	for symbol, cached := range vs.cache {
		if now.Sub(cached.timestamp) > vs.cacheExpiry {
			delete(vs.cache, symbol)
		}
	}
}
