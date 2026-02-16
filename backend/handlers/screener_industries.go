package handlers

import (
	"log"
	"net/http"

	"investorcenter-api/database"

	"github.com/gin-gonic/gin"
)

// GetScreenerIndustries returns unique industries, optionally filtered by sector.
// GET /api/v1/screener/industries?sectors=Technology,Healthcare
func GetScreenerIndustries(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Service is temporarily unavailable",
		})
		return
	}

	sectors := c.Query("sectors")
	industries, err := database.GetScreenerIndustries(sectors)
	if err != nil {
		log.Printf("Error fetching screener industries: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch industries",
			"message": "An error occurred while retrieving industry data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": industries,
	})
}
