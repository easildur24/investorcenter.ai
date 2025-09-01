package handlers

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"investorcenter-api/models"
	"investorcenter-api/services"

	"github.com/gin-gonic/gin"
)

// ImportTickersFromCSV handles importing tickers from CSV data
func ImportTickersFromCSV(c *gin.Context) {
	// Parse multipart form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	// Validate file type
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".csv") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File must be a CSV"})
		return
	}

	// Parse CSV
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse CSV: " + err.Error()})
		return
	}

	if len(records) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV must contain header and at least one data row"})
		return
	}

	// Process the CSV data
	stocks, err := processTickerCSV(records)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to process CSV: " + err.Error()})
		return
	}

	// Import to database
	stockService := services.NewStockService()
	ctx := context.WithValue(c.Request.Context(), "request_id", c.GetHeader("X-Request-ID"))

	err = stockService.ImportStocks(ctx, stocks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import stocks: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tickers imported successfully",
		"data": gin.H{
			"imported_count": len(stocks),
			"filename":       header.Filename,
		},
		"meta": gin.H{
			"timestamp": time.Now().UTC(),
		},
	})
}

// processTickerCSV processes raw CSV records into Stock models
func processTickerCSV(records [][]string) ([]models.Stock, error) {
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV must contain header and data rows")
	}

	header := records[0]
	dataRows := records[1:]

	// Find column indices
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.TrimSpace(col)] = i
	}

	// Validate required columns
	requiredCols := []string{"Ticker", "Security Name", "Exchange"}
	for _, col := range requiredCols {
		if _, exists := colMap[col]; !exists {
			return nil, fmt.Errorf("missing required column: %s", col)
		}
	}

	var stocks []models.Stock

	for _, row := range dataRows {
		if len(row) < len(header) {
			continue // Skip incomplete rows
		}

		// Extract data from row
		symbol := strings.TrimSpace(row[colMap["Ticker"]])
		securityName := strings.TrimSpace(row[colMap["Security Name"]])
		exchangeCode := strings.TrimSpace(row[colMap["Exchange"]])

		// Apply filtering logic
		if !shouldIncludeTickerGo(symbol, securityName) {
			continue
		}

		// Clean security name
		cleanName := cleanSecurityNameGo(securityName)

		// Convert exchange code
		exchangeName := convertExchangeCode(exchangeCode)

		// Create Stock model
		stock := models.Stock{
			Symbol:   strings.ToUpper(symbol),
			Name:     cleanName,
			Exchange: exchangeName,
			Country:  "US",
			Currency: "USD",
			// Other fields will be NULL/empty and populated later
		}

		stocks = append(stocks, stock)
	}

	return stocks, nil
}

// shouldIncludeTickerGo determines if a ticker should be included (Go version)
func shouldIncludeTickerGo(symbol, securityName string) bool {
	if symbol == "" {
		return false
	}

	// Skip tickers with special characters
	if strings.ContainsAny(symbol, ".$^+") {
		return false
	}

	// Skip warrant/rights/unit indicators
	warrantIndicators := []string{"W", "R", "U", "WS", "RT", "WT"}
	for _, indicator := range warrantIndicators {
		if strings.HasSuffix(symbol, indicator) {
			return false
		}
	}

	// Skip derivatives by security name
	derivativeIndicators := []string{
		"WARRANT", "RIGHTS", "UNITS", "PREFERRED", "NOTES", "TRUST",
		"DEPOSITARY SHARES", "SUBORDINATED", "CUMULATIVE",
	}
	securityUpper := strings.ToUpper(securityName)
	for _, indicator := range derivativeIndicators {
		if strings.Contains(securityUpper, indicator) {
			return false
		}
	}

	return true
}

// cleanSecurityNameGo cleans security name (Go version)
func cleanSecurityNameGo(name string) string {
	if name == "" {
		return ""
	}

	// Remove quotes
	name = strings.Trim(name, `"`)

	// Remove common suffixes (order matters - longer first)
	suffixes := []string{
		" - CLASS A COMMON STOCK",
		" CLASS A COMMON STOCK",
		" - CLASS B COMMON STOCK",
		" CLASS B COMMON STOCK",
		" - COMMON STOCK",
		" COMMON STOCK",
		" - ORDINARY SHARES",
		" ORDINARY SHARES",
		" - AMERICAN DEPOSITARY SHARES",
		" AMERICAN DEPOSITARY SHARES",
		" - ADS",
		" ADS",
	}

	nameUpper := strings.ToUpper(name)
	for _, suffix := range suffixes {
		if strings.HasSuffix(nameUpper, suffix) {
			name = name[:len(name)-len(suffix)]
			break
		}
	}

	// Clean up whitespace and trailing punctuation
	name = strings.TrimSpace(name)
	name = strings.TrimRight(name, " .,")

	// Handle corporate endings
	if strings.HasSuffix(name, " INC") {
		name = name + "."
	} else if strings.HasSuffix(name, " CORP") {
		name = name + "."
	} else if strings.HasSuffix(name, " LTD") {
		name = name + "."
	}

	return name
}

// convertExchangeCode converts exchange code to full name
func convertExchangeCode(code string) string {
	exchangeMap := map[string]string{
		"Q": "Nasdaq",
		"N": "NYSE",
		"A": "NYSE American",
		"P": "NYSE Arca",
		"Z": "Cboe",
	}

	if fullName, exists := exchangeMap[code]; exists {
		return fullName
	}
	return code // Return original if not found
}

// GetStocks handles retrieving stocks with pagination and search
func GetStocks(c *gin.Context) {
	stockService := services.NewStockService()
	ctx := c.Request.Context()

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "100")
	offsetStr := c.DefaultQuery("offset", "0")
	search := c.Query("search")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 1000 {
		limit = 100
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	var stocks []models.Stock

	if search != "" {
		// Search stocks
		stocks, err = stockService.SearchStocks(ctx, search, limit)
	} else {
		// Get all stocks with pagination
		stocks, err = stockService.GetAllStocks(ctx, limit, offset)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get total count for pagination
	totalCount := 0
	if search == "" {
		totalCount, _ = stockService.CountStocks(ctx)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": stocks,
		"meta": gin.H{
			"count":     len(stocks),
			"limit":     limit,
			"offset":    offset,
			"total":     totalCount,
			"search":    search,
			"timestamp": time.Now().UTC(),
		},
	})
}

// CreateStock handles creating a new stock
func CreateStock(c *gin.Context) {
	var stock models.Stock

	if err := c.ShouldBindJSON(&stock); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate required fields
	if stock.Symbol == "" || stock.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol and name are required"})
		return
	}

	stock.Symbol = strings.ToUpper(stock.Symbol)

	stockService := services.NewStockService()
	ctx := c.Request.Context()

	err := stockService.CreateStock(ctx, &stock)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			c.JSON(http.StatusConflict, gin.H{"error": "Stock with this symbol already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": stock,
		"meta": gin.H{
			"timestamp": time.Now().UTC(),
		},
	})
}
