package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"investorcenter-api/services"
)

// GetTickerSimple returns basic ticker data for testing
func GetTickerSimple(c *gin.Context) {
	symbol := c.Param("symbol")
	log.Printf("GetTickerSimple called for symbol: %s", symbol)
	
	// Use service layer for database operations
	stockService := services.NewStockService()
	stock, err := stockService.GetStockBySymbol(c.Request.Context(), symbol)
	if err != nil {
		log.Printf("Failed to get stock from database: %v", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Stock not found",
			"symbol": symbol,
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"stock": stock,
		},
	})
}