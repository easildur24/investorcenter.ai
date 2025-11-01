package services

import (
	"fmt"
	"investorcenter-api/database"
	"investorcenter-api/models"
	"strings"
)

// WatchListService handles business logic for watch lists
type WatchListService struct{}

func NewWatchListService() *WatchListService {
	return &WatchListService{}
}

// GetWatchListWithItems retrieves a watch list with all items and real-time prices
func (s *WatchListService) GetWatchListWithItems(watchListID string, userID string) (*models.WatchListWithItems, error) {
	// Get watch list metadata
	watchList, err := database.GetWatchListByID(watchListID, userID)
	if err != nil {
		return nil, err
	}

	// Get items with ticker data
	items, err := database.GetWatchListItemsWithData(watchListID)
	if err != nil {
		return nil, err
	}

	// Fetch real-time prices for all tickers
	polygonClient := NewPolygonClient()

	for i := range items {
		item := &items[i]

		// Use GetQuote which handles both stocks and crypto with caching
		price, err := polygonClient.GetQuote(item.Symbol)
		if err == nil && price != nil {
			currentPrice := float64(price.Price.InexactFloat64())
			item.CurrentPrice = &currentPrice

			// Calculate change and change percentage
			if price.PrevClose.IsPositive() {
				change := price.Price.Sub(price.PrevClose).InexactFloat64()
				changePercent := price.ChangePercent.InexactFloat64()
				prevClose := price.PrevClose.InexactFloat64()

				item.PriceChange = &change
				item.PriceChangePct = &changePercent
				item.PrevClose = &prevClose
			}

			// Set volume and market cap if available
			if price.Volume > 0 {
				volume := int64(price.Volume)
				item.Volume = &volume
			}

			if price.MarketCap != nil && price.MarketCap.IsPositive() {
				marketCap := price.MarketCap.InexactFloat64()
				item.MarketCap = &marketCap
			}
		}
	}

	result := &models.WatchListWithItems{
		WatchList: *watchList,
		ItemCount: len(items),
		Items:     items,
	}

	return result, nil
}

// ValidateWatchListOwnership checks if user owns the watch list
func (s *WatchListService) ValidateWatchListOwnership(watchListID string, userID string) error {
	watchList, err := database.GetWatchListByID(watchListID, userID)
	if err != nil {
		return err
	}
	if watchList.UserID != userID {
		return fmt.Errorf("unauthorized access to watch list")
	}
	return nil
}

// SearchTickers searches for tickers to add to watch list
func (s *WatchListService) SearchTickers(query string, limit int) ([]models.Stock, error) {
	// Convert query to uppercase for symbol matching
	query = strings.ToUpper(query)

	// Use database search function
	results, err := database.SearchStocks(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search tickers: %w", err)
	}

	return results, nil
}
