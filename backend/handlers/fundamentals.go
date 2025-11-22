package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"investorcenter-api/database"
)

// CalculateFundamentals calculates and stores fundamental metrics for a symbol
func CalculateFundamentals(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}
	
	log.Printf("🔍 Starting fundamental metrics calculation for %s", symbol)
	
	// TODO: SEC parser temporarily disabled for compilation
	log.Printf("⚠️ SEC parser not available, use Python script for calculation")
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "SEC parser not available",
		"message": "Use Python script to calculate metrics offline",
		"symbol":  symbol,
	})
	return
}

// GetFundamentals retrieves stored fundamental metrics for a symbol
func GetFundamentals(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}
	
	// Retrieve metrics from database
	metrics, err := database.GetFundamentalMetrics(symbol)
	if err != nil {
		log.Printf("❌ Failed to get metrics for %s: %v", symbol, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve fundamental metrics",
			"details": err.Error(),
		})
		return
	}
	
	if metrics == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "No fundamental metrics found",
			"message": "Metrics need to be calculated first",
			"symbol":  symbol,
		})
		return
	}
	
	// Check metrics age
	age, err := database.GetMetricsAge(symbol)
	if err != nil {
		log.Printf("⚠️ Failed to get metrics age for %s: %v", symbol, err)
	}
	
	response := gin.H{
		"symbol":  symbol,
		"metrics": metrics,
	}
	
	if age != nil {
		response["age_hours"] = age.Hours()
		response["updated_at"] = metrics.UpdatedAt
	}
	
	c.JSON(http.StatusOK, response)
}

// ListFundamentals lists all symbols that have fundamental metrics
func ListFundamentals(c *gin.Context) {
	// Get all symbols
	symbols, err := database.ListAllFundamentalMetrics()
	if err != nil {
		log.Printf("❌ Failed to list fundamental metrics: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list fundamental metrics",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"symbols": symbols,
		"count":   len(symbols),
	})
}

// RefreshFundamentals recalculates fundamental metrics for a symbol
func RefreshFundamentals(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}
	
	log.Printf("🔄 Refreshing fundamental metrics for %s", symbol)
	
	// This is the same as CalculateFundamentals but with different messaging
	CalculateFundamentals(c)
}
