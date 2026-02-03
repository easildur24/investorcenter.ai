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

// ManualFundamental represents the manually ingested fundamental data
type ManualFundamental struct {
	Ticker    string          `json:"ticker" db:"ticker"`
	Data      json.RawMessage `json:"data" db:"data"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
}

// PostManualFundamentals handles POST /api/v1/tickers/:symbol/manual-fundamentals
// Accepts a JSON blob of fundamental metrics and stores it
func PostManualFundamentals(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))

	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Ticker symbol is required",
		})
		return
	}

	// Check database connection
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Manual fundamentals service is temporarily unavailable",
		})
		return
	}

	// Parse JSON body
	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid JSON format",
			"message": err.Error(),
		})
		return
	}

	// Validate that we have at least some data
	if len(requestData) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Empty data",
			"message": "Please provide at least one metric",
		})
		return
	}

	// Convert to JSON for storage
	dataJSON, err := json.Marshal(requestData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process data",
			"message": err.Error(),
		})
		return
	}

	// Upsert into database (insert or update if exists)
	query := `
		INSERT INTO manual_fundamentals (ticker, data, created_at, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (ticker) 
		DO UPDATE SET 
			data = EXCLUDED.data,
			updated_at = CURRENT_TIMESTAMP
		RETURNING ticker, data, created_at, updated_at
	`

	var result ManualFundamental
	err = database.DB.QueryRowx(query, symbol, dataJSON).StructScan(&result)
	if err != nil {
		log.Printf("Error upserting manual fundamentals for %s: %v", symbol, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to save data",
			"message": "An error occurred while saving fundamental data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Fundamental data saved successfully",
		"data": gin.H{
			"ticker":     result.Ticker,
			"created_at": result.CreatedAt,
			"updated_at": result.UpdatedAt,
		},
	})
}

// GetManualFundamentals handles GET /api/v1/tickers/:symbol/manual-fundamentals
// Retrieves the manually ingested fundamental data for a ticker
func GetManualFundamentals(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))

	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Ticker symbol is required",
		})
		return
	}

	// Check database connection
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Manual fundamentals service is temporarily unavailable",
		})
		return
	}

	// Query the database
	query := `
		SELECT ticker, data, created_at, updated_at
		FROM manual_fundamentals
		WHERE ticker = $1
	`

	var result ManualFundamental
	err := database.DB.QueryRowx(query, symbol).StructScan(&result)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Data not found",
				"message": "No manual fundamental data available for this ticker",
				"ticker":  symbol,
			})
			return
		}
		log.Printf("Error fetching manual fundamentals for %s: %v", symbol, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch data",
			"message": "An error occurred while retrieving fundamental data",
		})
		return
	}

	// Parse the JSON data
	var parsedData map[string]interface{}
	if err := json.Unmarshal(result.Data, &parsedData); err != nil {
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

// DeleteManualFundamentals handles DELETE /api/v1/tickers/:symbol/manual-fundamentals
// Deletes the manually ingested fundamental data for a ticker
func DeleteManualFundamentals(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))

	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Ticker symbol is required",
		})
		return
	}

	// Check database connection
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Manual fundamentals service is temporarily unavailable",
		})
		return
	}

	// Delete from database
	query := `DELETE FROM manual_fundamentals WHERE ticker = $1`
	result, err := database.DB.Exec(query, symbol)
	if err != nil {
		log.Printf("Error deleting manual fundamentals for %s: %v", symbol, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete data",
			"message": "An error occurred while deleting fundamental data",
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Data not found",
			"message": "No manual fundamental data found for this ticker",
			"ticker":  symbol,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Fundamental data deleted successfully",
		"ticker":  symbol,
	})
}
