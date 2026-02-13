package database

import (
	"database/sql"
	"fmt"
	"time"
)

// VolumeData represents volume data from database
type VolumeData struct {
	Symbol        string    `json:"symbol"`
	Volume        int64     `json:"volume"`
	AvgVolume30D  int64     `json:"avgVolume30d"`
	AvgVolume90D  int64     `json:"avgVolume90d"`
	VWAP          float64   `json:"vwap"`
	CurrentPrice  float64   `json:"currentPrice"`
	DayOpen       float64   `json:"dayOpen"`
	DayHigh       float64   `json:"dayHigh"`
	DayLow        float64   `json:"dayLow"`
	PreviousClose float64   `json:"previousClose"`
	Week52High    float64   `json:"week52High"`
	Week52Low     float64   `json:"week52Low"`
	LastUpdated   time.Time `json:"lastUpdated"`
}

// GetTickerVolume retrieves volume data for a single ticker
func GetTickerVolume(symbol string) (*VolumeData, error) {
	query := `
		SELECT 
			symbol,
			COALESCE(volume, 0) as volume,
			COALESCE(avg_volume_30d, 0) as avg_volume_30d,
			COALESCE(avg_volume_90d, 0) as avg_volume_90d,
			COALESCE(vwap, 0) as vwap,
			COALESCE(current_price, 0) as current_price,
			COALESCE(day_open, 0) as day_open,
			COALESCE(day_high, 0) as day_high,
			COALESCE(day_low, 0) as day_low,
			COALESCE(previous_close, 0) as previous_close,
			COALESCE(week_52_high, 0) as week_52_high,
			COALESCE(week_52_low, 0) as week_52_low,
			COALESCE(last_trade_timestamp, updated_at) as last_updated
		FROM tickers
		WHERE symbol = $1 AND active = true
	`

	var data VolumeData
	err := DB.QueryRow(query, symbol).Scan(
		&data.Symbol,
		&data.Volume,
		&data.AvgVolume30D,
		&data.AvgVolume90D,
		&data.VWAP,
		&data.CurrentPrice,
		&data.DayOpen,
		&data.DayHigh,
		&data.DayLow,
		&data.PreviousClose,
		&data.Week52High,
		&data.Week52Low,
		&data.LastUpdated,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("ticker not found: %s", symbol)
		}
		return nil, err
	}

	return &data, nil
}

// VolumeAggregates for historical data
type VolumeAggregates struct {
	Symbol       string  `json:"symbol"`
	AvgVolume30D int64   `json:"avgVolume30d"`
	AvgVolume90D int64   `json:"avgVolume90d"`
	Week52High   float64 `json:"week52High"`
	Week52Low    float64 `json:"week52Low"`
	VolumeTrend  string  `json:"volumeTrend"`
}

// GetVolumeAggregates retrieves volume aggregates from database
func GetVolumeAggregates(symbol string) (*VolumeAggregates, error) {
	query := `
		SELECT 
			symbol,
			COALESCE(avg_volume_30d, 0),
			COALESCE(avg_volume_90d, 0),
			COALESCE(week_52_high, 0),
			COALESCE(week_52_low, 0)
		FROM tickers
		WHERE symbol = $1 AND active = true
	`

	var agg VolumeAggregates
	err := DB.QueryRow(query, symbol).Scan(
		&agg.Symbol,
		&agg.AvgVolume30D,
		&agg.AvgVolume90D,
		&agg.Week52High,
		&agg.Week52Low,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("ticker not found: %s", symbol)
		}
		return nil, err
	}

	// Calculate volume trend
	if agg.AvgVolume30D > 0 && agg.AvgVolume90D > 0 {
		ratio := float64(agg.AvgVolume30D) / float64(agg.AvgVolume90D)
		if ratio > 1.2 {
			agg.VolumeTrend = "increasing"
		} else if ratio < 0.8 {
			agg.VolumeTrend = "decreasing"
		} else {
			agg.VolumeTrend = "stable"
		}
	} else {
		agg.VolumeTrend = "unknown"
	}

	return &agg, nil
}
