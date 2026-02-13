package services

import (
	"context"

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
