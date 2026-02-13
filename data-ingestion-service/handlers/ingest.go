package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"data-ingestion-service/auth"
	"data-ingestion-service/database"
	"data-ingestion-service/storage"

	"github.com/gin-gonic/gin"
)

// IngestRequest is the request body for POST /ingest
type IngestRequest struct {
	Source      string  `json:"source" binding:"required,max=50"`
	Ticker      *string `json:"ticker"`
	DataType    string  `json:"data_type" binding:"required,max=100"`
	SourceURL   *string `json:"source_url"`
	RawData     string  `json:"raw_data" binding:"required"`
	CollectedAt *string `json:"collected_at"`
}

const maxRawDataSize = 10 * 1024 * 1024 // 10MB

// PostIngest handles POST /ingest — upload raw scraped data to S3
func PostIngest(c *gin.Context) {
	userID, ok := auth.GetUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req IngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %s", err.Error())})
		return
	}

	// Validate raw_data size
	if len(req.RawData) > maxRawDataSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("raw_data exceeds maximum size of %d bytes", maxRawDataSize)})
		return
	}

	// Validate ticker length
	if req.Ticker != nil && len(*req.Ticker) > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticker must be 20 characters or less"})
		return
	}

	// Validate source_url length
	if req.SourceURL != nil && len(*req.SourceURL) > 2000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source_url must be 2000 characters or less"})
		return
	}

	// Parse collected_at or default to now
	collectedAt := time.Now().UTC()
	if req.CollectedAt != nil && *req.CollectedAt != "" {
		parsed, err := time.Parse(time.RFC3339, *req.CollectedAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "collected_at must be in RFC3339 format (e.g. 2026-02-12T15:30:00Z)"})
			return
		}
		collectedAt = parsed.UTC()
	}

	// Generate S3 key
	ticker := ""
	if req.Ticker != nil {
		ticker = *req.Ticker
	}
	s3Key := storage.GenerateKey(req.Source, ticker, req.DataType, collectedAt)

	// Build the payload to store in S3 — wrap raw_data with metadata
	payload := map[string]interface{}{
		"source":       req.Source,
		"ticker":       req.Ticker,
		"data_type":    req.DataType,
		"source_url":   req.SourceURL,
		"raw_data":     req.RawData,
		"collected_at": collectedAt.Format(time.RFC3339),
		"uploaded_by":  userID,
		"uploaded_at":  time.Now().UTC().Format(time.RFC3339),
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

	// Write index record to Postgres
	id, err := database.InsertIngestionLog(
		req.Source,
		req.Ticker,
		req.DataType,
		req.SourceURL,
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
				"s3_key":  s3Key,
				"warning": "Data uploaded to S3 but index record failed — contact admin",
			},
		})
		return
	}

	log.Printf("Ingestion success: id=%d source=%s ticker=%v type=%s key=%s size=%d",
		id, req.Source, req.Ticker, req.DataType, s3Key, len(payloadBytes))

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"id":     id,
			"s3_key": s3Key,
		},
	})
}

// ListIngestionLogs handles GET /ingest — list ingestion records (admin only)
func ListIngestionLogs(c *gin.Context) {
	source := c.Query("source")
	ticker := c.Query("ticker")
	dataType := c.Query("data_type")

	limit := 100
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	records, total, err := database.GetIngestionLogs(source, ticker, dataType, limit, offset)
	if err != nil {
		log.Printf("Failed to list ingestion logs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve records"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    records,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetIngestionLogByID handles GET /ingest/:id — get single record (admin only)
func GetIngestionLogByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	record, err := database.GetIngestionLog(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    record,
	})
}
