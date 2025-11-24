package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"investorcenter-api/database"
	"investorcenter-api/models"
)

// PriceService handles stock price data from TimescaleDB
type PriceService struct {
	db *sqlx.DB
}

// NewPriceService creates a new price service
func NewPriceService() *PriceService {
	return &PriceService{
		db: database.DB,
	}
}

// GetHistoricalPrices fetches historical price data from stock_prices table
func (s *PriceService) GetHistoricalPrices(ctx context.Context, symbol string, period string) ([]models.ChartDataPoint, error) {
	days := GetDaysFromPeriod(period)

	query := `
		SELECT
			time,
			open,
			high,
			low,
			close,
			volume
		FROM stock_prices
		WHERE ticker = $1
		  AND interval = '1day'
		  AND time >= NOW() - INTERVAL '1 day' * $2
		ORDER BY time ASC
	`

	rows, err := s.db.QueryContext(ctx, query, symbol, days+30) // Add buffer for trading days
	if err != nil {
		return nil, fmt.Errorf("failed to query prices: %w", err)
	}
	defer rows.Close()

	var dataPoints []models.ChartDataPoint

	for rows.Next() {
		var timestamp time.Time
		var open, high, low, close sql.NullFloat64
		var volume sql.NullInt64

		err := rows.Scan(&timestamp, &open, &high, &low, &close, &volume)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		// Only include if we have close price
		if !close.Valid {
			continue
		}

		dataPoint := models.ChartDataPoint{
			Timestamp: timestamp,
			Close:     decimal.NewFromFloat(close.Float64),
			Open:      decimal.NewFromFloat(getFloat64(open)),
			High:      decimal.NewFromFloat(getFloat64(high)),
			Low:       decimal.NewFromFloat(getFloat64(low)),
			Volume:    getInt64(volume),
		}

		dataPoints = append(dataPoints, dataPoint)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return dataPoints, nil
}

// Get52WeekHighLow calculates 52-week high and low from historical data
func (s *PriceService) Get52WeekHighLow(ctx context.Context, symbol string) (high float64, low float64, err error) {
	query := `
		SELECT
			MAX(high) as week_52_high,
			MIN(low) as week_52_low
		FROM stock_prices
		WHERE ticker = $1
		  AND interval = '1day'
		  AND time >= NOW() - INTERVAL '365 days'
	`

	err = s.db.QueryRowContext(ctx, query, symbol).Scan(&high, &low)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, 0, fmt.Errorf("no price data found for %s", symbol)
		}
		return 0, 0, fmt.Errorf("failed to get 52-week high/low: %w", err)
	}

	return high, low, nil
}

// GetLatestPrice gets the latest close price for a symbol
func (s *PriceService) GetLatestPrice(ctx context.Context, symbol string) (float64, error) {
	query := `
		SELECT close
		FROM stock_prices
		WHERE ticker = $1
		  AND interval = '1day'
		ORDER BY time DESC
		LIMIT 1
	`

	var close sql.NullFloat64

	err := s.db.QueryRowContext(ctx, query, symbol).Scan(&close)

	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("no price data found for %s", symbol)
		}
		return 0, fmt.Errorf("failed to get latest price: %w", err)
	}

	if !close.Valid {
		return 0, fmt.Errorf("invalid close price for %s", symbol)
	}

	return close.Float64, nil
}

// Helper functions
func getFloat64(n sql.NullFloat64) float64 {
	if n.Valid {
		return n.Float64
	}
	return 0
}

func getInt64(n sql.NullInt64) int64 {
	if n.Valid {
		return n.Int64
	}
	return 0
}
