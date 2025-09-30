package services

import (
	"context"
	"fmt"

	"investorcenter-api/database"
	"investorcenter-api/models"

	"github.com/jmoiron/sqlx"
)

// StockService handles stock-related database operations
type StockService struct {
	db *sqlx.DB
}

// NewStockService creates a new stock service
func NewStockService() *StockService {
	return &StockService{
		db: database.DB,
	}
}

// CreateStock creates a new stock record
func (s *StockService) CreateStock(ctx context.Context, stock *models.Stock) error {
	query := `
		INSERT INTO tickers (symbol, name, exchange, sector, industry, country, currency, market_cap, description, website)
		VALUES (:symbol, :name, :exchange, :sector, :industry, :country, :currency, :market_cap, :description, :website)
		RETURNING id, created_at, updated_at`

	rows, err := s.db.NamedQueryContext(ctx, query, stock)
	if err != nil {
		return fmt.Errorf("failed to create stock: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&stock.ID, &stock.CreatedAt, &stock.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to scan created stock: %w", err)
		}
	}

	return nil
}

// GetStockBySymbol retrieves a stock by its symbol
func (s *StockService) GetStockBySymbol(ctx context.Context, symbol string) (*models.Stock, error) {
	// Use the database layer function which has better error handling
	return database.GetStockBySymbol(symbol)
}

// SearchStocks searches for stocks by symbol or name
func (s *StockService) SearchStocks(ctx context.Context, query string, limit int) ([]models.Stock, error) {
	// Use the database layer function
	return database.SearchStocks(query, limit)
}

// ImportStocks imports multiple stocks in a batch transaction
func (s *StockService) ImportStocks(ctx context.Context, stocks []models.Stock) error {
	if len(stocks) == 0 {
		return nil
	}

	// Start transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare batch insert statement
	query := `
		INSERT INTO tickers (symbol, name, exchange, sector, industry, country, currency, market_cap, description, website)
		VALUES (:symbol, :name, :exchange, :sector, :industry, :country, :currency, :market_cap, :description, :website)
		ON CONFLICT (symbol) DO NOTHING`

	// Execute batch insert
	_, err = tx.NamedExecContext(ctx, query, stocks)
	if err != nil {
		return fmt.Errorf("failed to insert stocks: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetAllStocks retrieves all stocks with pagination
func (s *StockService) GetAllStocks(ctx context.Context, limit, offset int) ([]models.Stock, error) {
	var stocks []models.Stock
	query := `
		SELECT * FROM tickers
		ORDER BY symbol
		LIMIT $1 OFFSET $2`

	err := s.db.SelectContext(ctx, &stocks, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get stocks: %w", err)
	}

	return stocks, nil
}

// CountStocks returns the total number of stocks
func (s *StockService) CountStocks(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM tickers`

	err := s.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to count stocks: %w", err)
	}

	return count, nil
}
