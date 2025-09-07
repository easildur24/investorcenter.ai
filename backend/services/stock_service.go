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
	var stock models.Stock
	query := `SELECT * FROM tickers WHERE symbol = $1`

	err := s.db.GetContext(ctx, &stock, query, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get stock by symbol %s: %w", symbol, err)
	}

	return &stock, nil
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

// SearchStocks searches for stocks by symbol or name
func (s *StockService) SearchStocks(ctx context.Context, query string, limit int) ([]models.Stock, error) {
	var stocks []models.Stock

	// Search by symbol or name (case insensitive)
	searchQuery := `
		SELECT * FROM tickers
		WHERE UPPER(symbol) LIKE UPPER($1) OR UPPER(name) LIKE UPPER($1)
		ORDER BY
			CASE WHEN UPPER(symbol) LIKE UPPER($1) THEN 1 ELSE 2 END,
			symbol
		LIMIT $2`

	searchTerm := "%" + query + "%"
	err := s.db.SelectContext(ctx, &stocks, searchQuery, searchTerm, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search stocks: %w", err)
	}

	return stocks, nil
}

// UpdateStock updates an existing stock
func (s *StockService) UpdateStock(ctx context.Context, stock *models.Stock) error {
	query := `
		UPDATE tickers
		SET name = :name, exchange = :exchange, sector = :sector, industry = :industry,
		    country = :country, currency = :currency, market_cap = :market_cap,
		    description = :description, website = :website, updated_at = NOW()
		WHERE symbol = :symbol`

	result, err := s.db.NamedExecContext(ctx, query, stock)
	if err != nil {
		return fmt.Errorf("failed to update stock: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("stock with symbol %s not found", stock.Symbol)
	}

	return nil
}

// DeleteStock deletes a stock by symbol
func (s *StockService) DeleteStock(ctx context.Context, symbol string) error {
	query := `DELETE FROM tickers WHERE symbol = $1`

	result, err := s.db.ExecContext(ctx, query, symbol)
	if err != nil {
		return fmt.Errorf("failed to delete stock: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("stock with symbol %s not found", symbol)
	}

	return nil
}

// GetStocksByExchange retrieves stocks by exchange
func (s *StockService) GetStocksByExchange(ctx context.Context, exchange string, limit, offset int) ([]models.Stock, error) {
	var stocks []models.Stock
	query := `
		SELECT * FROM tickers
		WHERE exchange = $1
		ORDER BY symbol
		LIMIT $2 OFFSET $3`

	err := s.db.SelectContext(ctx, &stocks, query, exchange, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get stocks by exchange: %w", err)
	}

	return stocks, nil
}
