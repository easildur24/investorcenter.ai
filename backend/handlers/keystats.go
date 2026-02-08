package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"investorcenter-api/database"
)

// KeyStats represents the ingested key stats data
type KeyStats struct {
	Ticker    string          `json:"ticker" db:"ticker"`
	KeyStats  json.RawMessage `json:"key_stats" db:"key_stats"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
}

// PostKeyStats handles POST /api/v1/tickers/:symbol/keystats
// Accepts a JSON blob of key stats and stores it
func PostKeyStats(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))

	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Ticker symbol is required",
		})
		return
	}

	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Key stats service is temporarily unavailable",
		})
		return
	}

	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid JSON format",
			"message": err.Error(),
		})
		return
	}

	if len(requestData) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Empty data",
			"message": "Please provide at least one metric",
		})
		return
	}

	dataJSON, err := json.Marshal(requestData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process data",
			"message": err.Error(),
		})
		return
	}

	query := `
		INSERT INTO keystats (ticker, key_stats, created_at, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (ticker)
		DO UPDATE SET
			key_stats = EXCLUDED.key_stats,
			updated_at = CURRENT_TIMESTAMP
		RETURNING ticker, key_stats, created_at, updated_at
	`

	var result KeyStats
	err = database.DB.QueryRowx(query, symbol, dataJSON).StructScan(&result)
	if err != nil {
		log.Printf("Error upserting key stats for %s: %v", symbol, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to save data",
			"message": "An error occurred while saving key stats data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Key stats data saved successfully",
		"data": gin.H{
			"ticker":     result.Ticker,
			"created_at": result.CreatedAt,
			"updated_at": result.UpdatedAt,
		},
	})
}

// GetKeyStats handles GET /api/v1/tickers/:symbol/keystats
// Retrieves the key stats data for a ticker
func GetKeyStats(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))

	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Ticker symbol is required",
		})
		return
	}

	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Key stats service is temporarily unavailable",
		})
		return
	}

	query := `
		SELECT ticker, key_stats, created_at, updated_at
		FROM keystats
		WHERE ticker = $1
	`

	var result KeyStats
	err := database.DB.QueryRowx(query, symbol).StructScan(&result)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Data not found",
				"message": "No key stats data available for this ticker",
				"ticker":  symbol,
			})
			return
		}
		log.Printf("Error fetching key stats for %s: %v", symbol, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch data",
			"message": "An error occurred while retrieving key stats data",
		})
		return
	}

	var parsedData map[string]interface{}
	if err := json.Unmarshal(result.KeyStats, &parsedData); err != nil {
		log.Printf("Error parsing JSON data for %s: %v", symbol, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Data corruption",
			"message": "Failed to parse stored data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    parsedData,
		"meta": gin.H{
			"ticker":     result.Ticker,
			"created_at": result.CreatedAt,
			"updated_at": result.UpdatedAt,
		},
	})
}

// DeleteKeyStats handles DELETE /api/v1/tickers/:symbol/keystats
// Deletes the key stats data for a ticker
func DeleteKeyStats(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))

	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Ticker symbol is required",
		})
		return
	}

	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Key stats service is temporarily unavailable",
		})
		return
	}

	query := `DELETE FROM keystats WHERE ticker = $1`
	result, err := database.DB.Exec(query, symbol)
	if err != nil {
		log.Printf("Error deleting key stats for %s: %v", symbol, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete data",
			"message": "An error occurred while deleting key stats data",
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Data not found",
			"message": "No key stats data found for this ticker",
			"ticker":  symbol,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Key stats data deleted successfully",
		"ticker":  symbol,
	})
}
