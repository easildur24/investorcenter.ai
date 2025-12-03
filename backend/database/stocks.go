package database

import (
	"fmt"
	"investorcenter-api/models"
)

// GetStockBySymbol retrieves stock information by symbol
// When multiple assets have the same symbol (e.g., META stock and META crypto),
// this prioritizes: stock > etf > index > crypto
func GetStockBySymbol(symbol string) (*models.Stock, error) {
	var stock models.Stock

	query := `
		SELECT id, symbol, name, exchange,
		       COALESCE(sector, '') as sector,
		       COALESCE(industry, '') as industry,
		       COALESCE(country, 'US') as country,
		       COALESCE(currency, 'USD') as currency,
		       market_cap,
		       COALESCE(description, '') as description,
		       COALESCE(website, '') as website,
		       COALESCE(asset_type, 'stock') as asset_type,
		       COALESCE(logo_url, '') as logo_url,
		       created_at, updated_at
		FROM tickers
		WHERE UPPER(symbol) = UPPER($1)
		ORDER BY
		  CASE asset_type
		    WHEN 'stock' THEN 0
		    WHEN 'etf' THEN 1
		    WHEN 'index' THEN 2
		    ELSE 3
		  END
		LIMIT 1
	`

	err := DB.Get(&stock, query, symbol)
	if err != nil {
		return nil, fmt.Errorf("stock not found: %w", err)
	}

	return &stock, nil
}

// SearchStocks searches for stocks by symbol or name
// Returns all matching assets, prioritizing:
// 1. Exact symbol match (stocks before crypto)
// 2. Symbol starts with query (stocks before crypto)
// 3. Name contains query (stocks before crypto)
func SearchStocks(query string, limit int) ([]models.Stock, error) {
	var stocks []models.Stock

	searchQuery := `
		SELECT id, symbol, name, exchange,
		       COALESCE(sector, '') as sector,
		       COALESCE(industry, '') as industry,
		       COALESCE(country, 'US') as country,
		       COALESCE(currency, 'USD') as currency,
		       market_cap,
		       COALESCE(description, '') as description,
		       COALESCE(website, '') as website,
		       COALESCE(asset_type, 'stock') as asset_type,
		       created_at, updated_at
		FROM tickers
		WHERE UPPER(symbol) LIKE UPPER($1)
		   OR UPPER(name) LIKE UPPER($2)
		ORDER BY
		  -- First priority: match type (exact > starts with > contains)
		  CASE
		    WHEN UPPER(symbol) = UPPER($3) THEN 1
		    WHEN UPPER(symbol) LIKE UPPER($4) THEN 2
		    WHEN UPPER(name) LIKE UPPER($5) THEN 3
		    ELSE 4
		  END,
		  -- Second priority: asset type (stock > etf > index > crypto)
		  CASE asset_type
		    WHEN 'stock' THEN 0
		    WHEN 'etf' THEN 1
		    WHEN 'index' THEN 2
		    ELSE 3
		  END,
		  -- Third priority: alphabetical by symbol
		  symbol
		LIMIT $6
	`

	searchTerm := "%" + query + "%"

	err := DB.Select(&stocks, searchQuery,
		searchTerm, // symbol LIKE
		searchTerm, // name LIKE
		query,      // exact symbol match
		query+"%",  // symbol starts with
		searchTerm, // name LIKE
		limit)

	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return stocks, nil
}

// GetPopularStocks returns a list of popular/featured stocks
func GetPopularStocks(limit int) ([]models.Stock, error) {
	var stocks []models.Stock

	// Get some popular stocks - you can customize this query
	query := `
		SELECT id, symbol, name, exchange,
		       COALESCE(sector, '') as sector,
		       COALESCE(industry, '') as industry,
		       COALESCE(country, 'US') as country,
		       COALESCE(currency, 'USD') as currency,
		       market_cap,
		       COALESCE(description, '') as description,
		       COALESCE(website, '') as website,
		       created_at, updated_at
		FROM tickers
		WHERE symbol IN ('AAPL', 'GOOGL', 'MSFT', 'TSLA', 'AMZN', 'NVDA', 'META', 'NFLX', 'CRM', 'ORCL')
		ORDER BY symbol
		LIMIT $1
	`

	err := DB.Select(&stocks, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular stocks: %w", err)
	}

	return stocks, nil
}

// GetStockCount returns the total number of stocks in the database
func GetStockCount() (int, error) {
	var count int
	err := DB.Get(&count, "SELECT COUNT(*) FROM tickers")
	if err != nil {
		return 0, fmt.Errorf("failed to get stock count: %w", err)
	}
	return count, nil
}
