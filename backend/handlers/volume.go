package handlers

import (
	"net/http"
	"strconv"

	"investorcenter-api/database"
	"investorcenter-api/services"

	"github.com/gin-gonic/gin"
)

var volumeService = services.NewVolumeService()

// GetTickerVolume returns hybrid volume data
// First checks database for recent data, then fetches real-time if needed
func GetTickerVolume(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	// Check if we want real-time data (query param)
	realtime := c.Query("realtime") == "true"

	// For real-time requests or if database data is stale
	if realtime {
		// Fetch real-time data from Polygon
		volumeData, err := volumeService.GetRealTimeVolume(symbol)
		if err != nil {
			// Fallback to database if API fails
			dbVolume, dbErr := database.GetTickerVolume(symbol)
			if dbErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Failed to fetch volume data",
					"details": err.Error(),
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"data":     dbVolume,
				"source":   "database",
				"realtime": false,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":     volumeData,
			"source":   "polygon",
			"realtime": true,
		})
		return
	}

	// Default: Get from database (fast, cached data)
	dbVolume, err := database.GetTickerVolume(symbol)
	if err != nil {
		// If not in database, try to fetch real-time
		volumeData, apiErr := volumeService.GetRealTimeVolume(symbol)
		if apiErr != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":  "Volume data not found",
				"symbol": symbol,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"data":     volumeData,
			"source":   "polygon",
			"realtime": true,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     dbVolume,
		"source":   "database",
		"realtime": false,
	})
}

// GetVolumeAggregates returns historical volume aggregates
func GetVolumeAggregates(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	// Get days parameter (default 90)
	daysStr := c.DefaultQuery("days", "90")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 || days > 365 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid days parameter (1-365)"})
		return
	}

	// Fetch aggregates from Polygon
	aggregates, err := volumeService.GetVolumeAggregates(symbol, days)
	if err != nil {
		// Try to get from database as fallback
		dbAggregates, dbErr := database.GetVolumeAggregates(symbol)
		if dbErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to fetch volume aggregates",
				"details": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"data":   dbAggregates,
			"source": "database",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   aggregates,
		"source": "polygon",
	})
}
