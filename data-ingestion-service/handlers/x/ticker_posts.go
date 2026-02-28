package x

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

var asOfDateRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// PostTickerPosts handles POST /ingest/x/ticker_posts/:ticker
// Ingests a point-in-time snapshot of X posts about a ticker.
// S3 key: x/ticker_posts/{TICKER}/{YYYY-MM-DD}/{timestamp}.json
func PostTickerPosts(c *gin.Context) {
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

	// Ensure ticker in body matches URL (or set it)
	requestData["ticker"] = ticker

	// Validate required fields
	asOfDate, ok := requestData["as_of_date"].(string)
	if !ok || asOfDate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "as_of_date is required (YYYY-MM-DD format)"})
		return
	}
	if !asOfDateRegex.MatchString(asOfDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "as_of_date must be in YYYY-MM-DD format"})
		return
	}

	sourceURL, ok := requestData["source_url"].(string)
	if !ok || sourceURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source_url is required"})
		return
	}

	// Validate against JSON schema
	schemaPath := "schemas/x/ticker_posts.json"
	schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaPath)
	documentLoader := gojsonschema.NewGoLoader(requestData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		log.Printf("Schema validation error for x/ticker_posts: %v", err)
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

	// Generate S3 key: x/ticker_posts/{TICKER}/{YYYY-MM-DD}/{timestamp}.json
	now := time.Now().UTC()
	timestampPart := now.Format("20060102T150405Z")
	s3Key := fmt.Sprintf("x/ticker_posts/%s/%s/%s.json", ticker, asOfDate, timestampPart)

	// Prepare payload with metadata
	payload := map[string]interface{}{
		"uploaded_by": userID,
		"uploaded_at": now.Format(time.RFC3339),
	}
	for k, v := range requestData {
		payload[k] = v
	}

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

	// Parse as_of_date for ingestion log
	collectedAt, err := time.Parse("2006-01-02", asOfDate)
	if err != nil {
		collectedAt = now
	}

	// Write index record
	tickerStr := ticker
	id, err := database.InsertIngestionLog(
		"x",
		&tickerStr,
		"ticker_posts",
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
				"ticker":  ticker,
				"s3_key":  s3Key,
				"warning": "Data uploaded to S3 but index record failed - contact admin",
			},
		})
		return
	}

	log.Printf("X Ticker Posts ingestion success: id=%d ticker=%s key=%s size=%d",
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
