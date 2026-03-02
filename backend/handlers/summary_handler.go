package handlers

import (
	"log"
	"net/http"
	"time"

	"investorcenter-api/services"

	"github.com/gin-gonic/gin"
)

// summaryGenerator is initialized at startup.
var summaryGenerator *services.SummaryGenerator

func init() {
	summaryGenerator = services.NewSummaryGenerator()
}

// GetMarketSummary handles GET /api/v1/markets/summary
// Returns an LLM-generated (or template-based fallback) market summary.
// Results are cached in Redis for 15 minutes.
func GetMarketSummary(c *gin.Context) {
	ctx := c.Request.Context()

	// Try cached summary first
	cached, err := summaryGenerator.GetCachedSummary(ctx)
	if err != nil {
		log.Printf("Warning: failed to read cached summary: %v", err)
	}
	if cached != nil {
		c.JSON(http.StatusOK, gin.H{
			"data": cached,
			"meta": gin.H{
				"timestamp": time.Now().UTC(),
				"cached":    true,
			},
		})
		return
	}

	// Generate on-demand
	result, err := summaryGenerator.GenerateMarketSummary(ctx)
	if err != nil {
		log.Printf("Error generating market summary: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Failed to generate market summary",
			"message": "Market summary is temporarily unavailable. Please try again later.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
		"meta": gin.H{
			"timestamp": time.Now().UTC(),
			"cached":    false,
		},
	})
}
