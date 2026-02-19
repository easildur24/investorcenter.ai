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

// GetWatchListWithItems retrieves a watch list with enriched items
// (IC Score, fundamentals, Reddit data, alert counts) and real-time prices.
func (s *WatchListService) GetWatchListWithItems(watchListID string, userID string) (*models.WatchListWithItemsEnriched, error) {
	// Get watch list metadata
	watchList, err := database.GetWatchListByID(watchListID, userID)
	if err != nil {
		return nil, err
	}

	// Get items with enriched data (screener_data + reddit + alerts)
	items, err := database.GetWatchListItemsWithEnrichedData(watchListID)
	if err != nil {
		log.Printf("Error fetching enriched watch list items for list %s: %v", watchListID, err)
		return nil, fmt.Errorf("failed to fetch watch list items: %w", err)
	}

	// Fetch real-time prices for all tickers
	polygonClient := NewPolygonClient()

	for i := range items {
		item := &items[i]

		// Graceful degradation: log Polygon failures but still return items without prices
		price, err := polygonClient.GetQuote(item.Symbol)
		if err != nil {
			log.Printf("Warning: Polygon price fetch failed for %s in watch list %s: %v", item.Symbol, watchListID, err)
		}
		if err == nil && price != nil {
			currentPrice := price.Price.InexactFloat64()
			item.CurrentPrice = &currentPrice

			if price.Change.IsPositive() || price.Change.IsNegative() {
				change := price.Change.InexactFloat64()
				changePercent := price.ChangePercent.InexactFloat64()
				item.PriceChange = &change
				item.PriceChangePct = &changePercent

				prevClose := price.Price.Sub(price.Change).InexactFloat64()
				item.PrevClose = &prevClose
			}

			if price.Volume > 0 {
				volume := int64(price.Volume)
				item.Volume = &volume
			}
		}
	}

	// Compute summary metrics from enriched items
	summary := computeSummaryMetrics(items)

	return &models.WatchListWithItemsEnriched{
		WatchList: *watchList,
		ItemCount: len(items),
		Items:     items,
		Summary:   summary,
	}, nil
}

// computeSummaryMetrics calculates aggregate statistics from a set of enriched watchlist items.
func computeSummaryMetrics(items []models.WatchListItemEnriched) *models.WatchListSummaryMetrics {
	summary := &models.WatchListSummaryMetrics{
		TotalTickers: len(items),
	}
	if len(items) == 0 {
		return summary
	}

	// Collect non-nil values for averaging
	var icScores, dayChangePcts, dividendYields []float64

	for i := range items {
		if items[i].ICScore != nil {
			icScores = append(icScores, *items[i].ICScore)
		}
		if items[i].PriceChangePct != nil {
			dayChangePcts = append(dayChangePcts, *items[i].PriceChangePct)
		}
		if items[i].DividendYield != nil {
			dividendYields = append(dividendYields, *items[i].DividendYield)
		}
		if items[i].RedditTrend != nil && *items[i].RedditTrend == "rising" {
			summary.RedditTrendingCount++
		}
	}

	if len(icScores) > 0 {
		avg := average(icScores)
		summary.AvgICScore = &avg
	}
	if len(dayChangePcts) > 0 {
		avg := average(dayChangePcts)
		summary.AvgDayChangePct = &avg
	}
	if len(dividendYields) > 0 {
		avg := average(dividendYields)
		summary.AvgDividendYield = &avg
	}

	return summary
}

// average returns the mean of a float64 slice.
func average(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
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
