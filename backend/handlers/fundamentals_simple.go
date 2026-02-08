package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"investorcenter-api/database"
)

// GetFundamentalsSimple retrieves fundamental metrics directly as JSON
func GetFundamentalsSimple(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	// Get metrics from database
	var metricsJSON []byte
	query := `SELECT metrics_data FROM fundamental_metrics WHERE symbol = $1`
	err := database.DB.QueryRow(query, symbol).Scan(&metricsJSON)

	if err != nil {
		log.Printf("❌ Failed to get metrics for %s: %v", symbol, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "No fundamental metrics found",
			"message": "Metrics need to be calculated first",
			"symbol":  symbol,
		})
		return
	}

	// Parse JSON to map for response
	var metrics map[string]interface{}
	err = json.Unmarshal(metricsJSON, &metrics)
	if err != nil {
		log.Printf("❌ Failed to parse metrics JSON for %s: %v", symbol, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to parse fundamental metrics",
			"details": err.Error(),
		})
		return
	}

	log.Printf("✅ Retrieved fundamental metrics for %s", symbol)

	c.JSON(http.StatusOK, gin.H{
		"symbol":  symbol,
		"metrics": metrics,
	})
}
