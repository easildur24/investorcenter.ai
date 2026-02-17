package handlers

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"investorcenter-api/database"
)

// ProxyLogo proxies logo requests to Polygon.io with the API key
func ProxyLogo(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))

	// Get stock from database to find logo URL
	stock, err := database.GetStockBySymbol(symbol)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Stock not found"})
		return
	}

	if stock.LogoURL == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "No logo available"})
		return
	}

	// Add API key to the Polygon URL
	apiKey := os.Getenv("POLYGON_API_KEY")
	if apiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "API key not configured"})
		return
	}

	// Append API key to URL
	logoURL := stock.LogoURL
	if strings.Contains(logoURL, "?") {
		logoURL += "&apiKey=" + apiKey
	} else {
		logoURL += "?apiKey=" + apiKey
	}

	// Fetch the logo from Polygon
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(logoURL)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch logo"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": "Logo not available"})
		return
	}

	// Set cache headers (cache for 24 hours)
	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("Content-Type", resp.Header.Get("Content-Type"))

	// Stream the response
	if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		log.Printf("Failed to stream logo for %s: %v", symbol, err)
	}
}
