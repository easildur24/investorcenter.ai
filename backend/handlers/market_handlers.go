package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"investorcenter-api/services"
)

// IndexInfo represents market index information
type IndexInfo struct {
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"changePercent"`
	LastUpdated   string  `json:"lastUpdated"`
}

// GetMarketIndices fetches current market indices from Polygon.io
func GetMarketIndices(c *gin.Context) {
	polygonClient := services.NewPolygonClient()

	// Define major market indices using ETF proxies
	// Note: Using ETFs as proxies since indices require premium Polygon.io plan
	// ETFs track indices very closely and provide real-time data
	indexSymbols := []struct {
		Symbol string
		Name   string
	}{
		{"SPY", "S&P 500"},   // SPDR S&P 500 ETF Trust (tracks S&P 500)
		{"DIA", "Dow Jones"}, // SPDR Dow Jones Industrial Average ETF (tracks DJIA)
		{"QQQ", "NASDAQ"},    // Invesco QQQ Trust (tracks NASDAQ-100)
	}

	indices := []IndexInfo{}
	var fetchErrors []string

	for _, idx := range indexSymbols {
		priceData, err := polygonClient.GetStockRealTimePrice(idx.Symbol)
		if err != nil {
			errMsg := "Failed to fetch " + idx.Name + " (" + idx.Symbol + "): " + err.Error()
			log.Printf("Warning: %s", errMsg)
			fetchErrors = append(fetchErrors, errMsg)
			// Skip this index if we can't fetch it
			continue
		}

		index := IndexInfo{
			Symbol:        idx.Symbol,
			Name:          idx.Name,
			Price:         priceData.Price.InexactFloat64(),
			Change:        priceData.Change.InexactFloat64(),
			ChangePercent: priceData.ChangePercent.InexactFloat64(),
			LastUpdated:   priceData.Timestamp.Format(time.RFC3339),
		}

		indices = append(indices, index)
		log.Printf("Successfully fetched %s: $%.2f (%.2f%%)", idx.Name, index.Price, index.ChangePercent)
	}

	// If we couldn't fetch any indices, return an error with details
	if len(indices) == 0 {
		log.Printf("Error: Failed to fetch any market indices. Errors: %v", fetchErrors)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Failed to fetch market indices from Polygon.io. Please check API key and connectivity.",
			"details": fetchErrors,
			"meta": gin.H{
				"timestamp": time.Now().UTC(),
			},
		})
		return
	}

	// Return successfully fetched indices (even if some failed)
	c.JSON(http.StatusOK, gin.H{
		"data": indices,
		"meta": gin.H{
			"count":     len(indices),
			"timestamp": time.Now().UTC(),
			"source":    "polygon.io",
		},
	})
}
