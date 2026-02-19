package services

import (
	"fmt"
	"investorcenter-api/database"
	"investorcenter-api/models"
	"log"
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
		log.Printf("Error fetching enriched watch list items for list %s: %v", watchListID, err)
		return nil, fmt.Errorf("failed to fetch watch list items: %w", err)
	}

	// Fetch real-time prices for all tickers
	polygonClient := NewPolygonClient()

	for i := range items {
		item := &items[i]

		// Use GetQuote which handles both stocks and crypto with caching
		// Graceful degradation: log Polygon failures but still return items without prices
		price, err := polygonClient.GetQuote(item.Symbol)
		if err != nil {
			log.Printf("Warning: Polygon price fetch failed for %s in watch list %s: %v", item.Symbol, watchListID, err)
		}
		if err == nil && price != nil {
			currentPrice := float64(price.Price.InexactFloat64())
			item.CurrentPrice = &currentPrice

			// Set change and change percentage from cached data
			if price.Change.IsPositive() || price.Change.IsNegative() {
				change := price.Change.InexactFloat64()
				changePercent := price.ChangePercent.InexactFloat64()

				item.PriceChange = &change
				item.PriceChangePct = &changePercent
			}

			// Set volume if available
			if price.Volume > 0 {
				volume := int64(price.Volume)
				item.Volume = &volume
			}

			// Calculate previous close from current price and change
			if price.Change.IsPositive() || price.Change.IsNegative() {
				prevClose := price.Price.Sub(price.Change).InexactFloat64()
				item.PrevClose = &prevClose
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
