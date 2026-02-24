package ycharts

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"data-ingestion-service/auth"
	"data-ingestion-service/database"
	"data-ingestion-service/storage"

	"github.com/gin-gonic/gin"
	"github.com/xeipuuv/gojsonschema"
)

// Valid statement types and their schema paths
var financialStatementSchemas = map[string]string{
	"income_statement": "schemas/ycharts/financials/income_statement.json",
	"balance_sheet":    "schemas/ycharts/financials/balance_sheet.json",
	"cash_flow":        "schemas/ycharts/financials/cash_flow.json",
}

var periodRegex = regexp.MustCompile(`^\d{4}-\d{2}$`)

// PostFinancials handles POST /ingest/ycharts/financials/:statement/:ticker
// Ingests a single period of financial statement data.
// S3 key: ycharts/financials/{statement}/{TICKER}/{period_type}/{period}.json
func PostFinancials(c *gin.Context) {
	userID, ok := auth.GetUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get statement type from URL path
	statement := strings.ToLower(c.Param("statement"))
	schemaPath, validStatement := financialStatementSchemas[statement]
	if !validStatement {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid statement type '%s'. Must be one of: income_statement, balance_sheet, cash_flow", statement),
		})
		return
	}

	// Get ticker from URL path
	ticker := strings.ToUpper(c.Param("ticker"))
	if ticker == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticker is required in URL path"})
		return
	}
	if len(ticker) > 20 || len(ticker) < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticker must be 1-20 characters"})
		return
	}

	// Parse request body
	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid JSON: %s", err.Error())})
		return
	}

	// Validate required fields before schema validation for better error messages
	period, ok := requestData["period"].(string)
	if !ok || period == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "period is required (YYYY-MM format)"})
		return
	}
	if !periodRegex.MatchString(period) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "period must be in YYYY-MM format (e.g. 2025-09)"})
		return
	}

	periodType, ok := requestData["period_type"].(string)
	if !ok || periodType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "period_type is required (quarterly, annual, or ttm)"})
		return
	}
	validPeriodTypes := map[string]bool{"quarterly": true, "annual": true, "ttm": true}
	if !validPeriodTypes[periodType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "period_type must be one of: quarterly, annual, ttm"})
		return
	}

	sourceURL, ok := requestData["source_url"].(string)
	if !ok || sourceURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source_url is required"})
		return
	}

	// Validate against JSON schema
	schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaPath)
	documentLoader := gojsonschema.NewGoLoader(requestData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		log.Printf("Schema validation error for %s: %v", statement, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Schema validation failed"})
		return
	}

	if !result.Valid() {
		errors := []string{}
		for _, desc := range result.Errors() {
			errors = append(errors, desc.String())
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "Request validation failed",
			"validation_errors": errors,
		})
		return
	}

	// Generate S3 key: ycharts/financials/{statement}/{TICKER}/{period_type}/{period}.json
	// Overwrites on duplicate (idempotent)
	s3Key := fmt.Sprintf("ycharts/financials/%s/%s/%s/%s.json", statement, ticker, periodType, period)

	// Prepare payload for S3 (add metadata)
	payload := map[string]interface{}{
		"ticker":       ticker,
		"statement":    statement,
		"period":       period,
		"period_type":  periodType,
		"source_url":   sourceURL,
		"uploaded_by":  userID,
		"uploaded_at":  time.Now().UTC().Format(time.RFC3339),
	}

	// Merge all data fields into payload
	for k, v := range requestData {
		if k != "source_url" {
			payload[k] = v
		}
	}

	// Serialize to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal payload: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare data"})
		return
	}

	// Upload to S3
	if err := storage.Upload(s3Key, payloadBytes, "application/json"); err != nil {
		log.Printf("Failed to upload to S3: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload data to storage"})
		return
	}

	// For ingestion_log, use period as a synthetic timestamp (first of month)
	collectedAt, err := time.Parse("2006-01", period)
	if err != nil {
		collectedAt = time.Now().UTC()
	}

	// Write index record to ingestion_log table
	dataType := fmt.Sprintf("financials_%s", statement)
	tickerStr := ticker
	id, err := database.InsertIngestionLog(
		"ycharts",
		&tickerStr,
		dataType,
		&sourceURL,
		s3Key,
		storage.GetBucket(),
		int64(len(payloadBytes)),
		collectedAt,
	)
	if err != nil {
		log.Printf("Failed to insert ingestion log (S3 upload succeeded at %s): %v", s3Key, err)
		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"data": gin.H{
				"ticker":    ticker,
				"statement": statement,
				"period":    period,
				"s3_key":    s3Key,
				"warning":   "Data uploaded to S3 but index record failed - contact admin",
			},
		})
		return
	}

	log.Printf("YCharts Financials ingestion success: id=%d ticker=%s statement=%s period=%s key=%s size=%d",
		id, ticker, statement, period, s3Key, len(payloadBytes))

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"id":        id,
			"ticker":    ticker,
			"statement": statement,
			"period":    period,
			"s3_key":    s3Key,
		},
	})
}
