package ycharts

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"data-ingestion-service/auth"
	"data-ingestion-service/database"
	"data-ingestion-service/storage"

	"github.com/gin-gonic/gin"
	"github.com/xeipuuv/gojsonschema"
)

// KeyStatsRequest is the request body for POST /ingest/ycharts/key_stats/:ticker
type KeyStatsRequest struct {
	CollectedAt string                 `json:"collected_at" binding:"required"`
	SourceURL   string                 `json:"source_url" binding:"required"`
	Data        map[string]interface{} `json:",inline"` // All other fields
}

// PostKeyStats handles POST /ingest/ycharts/key_stats/:ticker
func PostKeyStats(c *gin.Context) {
	userID, ok := auth.GetUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get ticker from URL path
	ticker := strings.ToUpper(c.Param("ticker"))
	if ticker == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticker is required in URL path"})
		return
	}

	// Validate ticker format
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

	// Validate required fields
	collectedAtStr, ok := requestData["collected_at"].(string)
	if !ok || collectedAtStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "collected_at is required"})
		return
	}

	sourceURL, ok := requestData["source_url"].(string)
	if !ok || sourceURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source_url is required"})
		return
	}

	// Parse collected_at timestamp
	collectedAt, err := time.Parse(time.RFC3339, collectedAtStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "collected_at must be in RFC3339 format (e.g. 2026-02-12T20:30:00Z)"})
		return
	}

	// Validate against JSON schema
	// Use relative path or environment variable for schema location
	schemaPath := "schemas/ycharts/key_stats.json"
	schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaPath)
	documentLoader := gojsonschema.NewGoLoader(requestData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		log.Printf("Schema validation error: %v", err)
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

	// Generate S3 key: ycharts/key_stats/{TICKER}/{YYYY-MM-DD}/{timestamp}.json
	datePart := collectedAt.UTC().Format("2006-01-02")
	timestampPart := collectedAt.UTC().Format("20060102T150405Z")
	s3Key := fmt.Sprintf("ycharts/key_stats/%s/%s/%s.json", ticker, datePart, timestampPart)

	// Prepare payload for S3 (add metadata)
	payload := map[string]interface{}{
		"ticker":       ticker,
		"collected_at": collectedAt.Format(time.RFC3339),
		"source_url":   sourceURL,
		"uploaded_by":  userID,
		"uploaded_at":  time.Now().UTC().Format(time.RFC3339),
	}

	// Merge request data into payload
	for k, v := range requestData {
		if k != "collected_at" && k != "source_url" {
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

	// Write index record to ingestion_log table
	tickerStr := ticker
	id, err := database.InsertIngestionLog(
		"ycharts",
		&tickerStr,
		"key_stats",
		&sourceURL,
		s3Key,
		storage.GetBucket(),
		int64(len(payloadBytes)),
		collectedAt,
	)
	if err != nil {
		log.Printf("Failed to insert ingestion log (S3 upload succeeded at %s): %v", s3Key, err)
		// S3 upload succeeded but DB write failed — return success with warning
		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"data": gin.H{
				"ticker":  ticker,
				"s3_key":  s3Key,
				"warning": "Data uploaded to S3 but index record failed — contact admin",
			},
		})
		return
	}

	log.Printf("YCharts Key Stats ingestion success: id=%d ticker=%s key=%s size=%d",
		id, ticker, s3Key, len(payloadBytes))

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"id":     id,
			"ticker": ticker,
			"s3_key": s3Key,
		},
	})
}
