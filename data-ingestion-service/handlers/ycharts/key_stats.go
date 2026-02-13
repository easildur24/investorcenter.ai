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
			"error":            "Request validation failed",
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

	// Insert into database (ycharts_key_stats table)
	id, err := insertKeyStats(ticker, collectedAt, sourceURL, s3Key, payloadBytes, requestData)
	if err != nil {
		log.Printf("Failed to insert into database (S3 upload succeeded at %s): %v", s3Key, err)
		// S3 upload succeeded but DB write failed — return success with warning
		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"data": gin.H{
				"s3_key":  s3Key,
				"warning": "Data uploaded to S3 but database insert failed — contact admin",
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

// insertKeyStats inserts key stats data into the database
func insertKeyStats(ticker string, collectedAt time.Time, sourceURL string, s3Key string, payloadBytes []byte, data map[string]interface{}) (int64, error) {
	// Extract nested fields from data
	var (
		// Price
		price          *float64
		priceCurrency  *string
		priceExchange  *string
		priceTimestamp *time.Time

		// Income Statement
		revenueTTM         *int64
		revenueQuarterly   *int64
		netIncomeTTM       *int64
		netIncomeQuarterly *int64
		ebitTTM            *int64
		ebitQuarterly      *int64
		ebitdaTTM          *int64
		ebitdaQuarterly    *int64
		revenueGrowthYoY   *float64
		epsGrowthYoY       *float64
		ebitdaGrowthYoY    *float64
		epsDilutedTTM      *float64
		epsBasicTTM        *float64
		sharesOutstanding  *int64

		// ... (many more fields, but for now let's keep it simple and store as JSONB)
	)

	// Extract price section
	if priceData, ok := data["price"].(map[string]interface{}); ok {
		if val, ok := priceData["current"].(float64); ok {
			price = &val
		}
		if val, ok := priceData["currency"].(string); ok {
			priceCurrency = &val
		}
		if val, ok := priceData["exchange"].(string); ok {
			priceExchange = &val
		}
		if val, ok := priceData["timestamp"].(string); ok {
			if ts, err := time.Parse(time.RFC3339, val); err == nil {
				priceTimestamp = &ts
			}
		}
	}

	// Extract income_statement section
	if incomeData, ok := data["income_statement"].(map[string]interface{}); ok {
		if val, ok := incomeData["revenue_ttm"].(float64); ok {
			i := int64(val)
			revenueTTM = &i
		}
		if val, ok := incomeData["revenue_quarterly"].(float64); ok {
			i := int64(val)
			revenueQuarterly = &i
		}
		if val, ok := incomeData["net_income_ttm"].(float64); ok {
			i := int64(val)
			netIncomeTTM = &i
		}
		if val, ok := incomeData["net_income_quarterly"].(float64); ok {
			i := int64(val)
			netIncomeQuarterly = &i
		}
		if val, ok := incomeData["ebit_ttm"].(float64); ok {
			i := int64(val)
			ebitTTM = &i
		}
		if val, ok := incomeData["ebit_quarterly"].(float64); ok {
			i := int64(val)
			ebitQuarterly = &i
		}
		if val, ok := incomeData["ebitda_ttm"].(float64); ok {
			i := int64(val)
			ebitdaTTM = &i
		}
		if val, ok := incomeData["ebitda_quarterly"].(float64); ok {
			i := int64(val)
			ebitdaQuarterly = &i
		}
		if val, ok := incomeData["revenue_growth_yoy"].(float64); ok {
			revenueGrowthYoY = &val
		}
		if val, ok := incomeData["eps_growth_yoy"].(float64); ok {
			epsGrowthYoY = &val
		}
		if val, ok := incomeData["ebitda_growth_yoy"].(float64); ok {
			ebitdaGrowthYoY = &val
		}
		if val, ok := incomeData["eps_diluted_ttm"].(float64); ok {
			epsDilutedTTM = &val
		}
		if val, ok := incomeData["eps_basic_ttm"].(float64); ok {
			epsBasicTTM = &val
		}
		if val, ok := incomeData["shares_outstanding"].(float64); ok {
			i := int64(val)
			sharesOutstanding = &i
		}
	}

	// For now, store the full payload as JSONB
	// This allows us to query specific fields later without losing any data
	query := `
		INSERT INTO ycharts_key_stats (
			ticker, collected_at, source_url, s3_key, s3_bucket, data_size,
			price, price_currency, price_exchange, price_timestamp,
			revenue_ttm, revenue_quarterly, net_income_ttm, net_income_quarterly,
			ebit_ttm, ebit_quarterly, ebitda_ttm, ebitda_quarterly,
			revenue_growth_yoy, eps_growth_yoy, ebitda_growth_yoy,
			eps_diluted_ttm, eps_basic_ttm, shares_outstanding,
			data_json
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14,
			$15, $16, $17, $18,
			$19, $20, $21,
			$22, $23, $24,
			$25
		)
		ON CONFLICT (ticker, collected_at) DO UPDATE SET
			source_url = EXCLUDED.source_url,
			s3_key = EXCLUDED.s3_key,
			data_size = EXCLUDED.data_size,
			price = EXCLUDED.price,
			revenue_ttm = EXCLUDED.revenue_ttm,
			data_json = EXCLUDED.data_json,
			updated_at = NOW()
		RETURNING id
	`

	var id int64
	err := database.DB.QueryRow(
		query,
		ticker, collectedAt, sourceURL, s3Key, storage.GetBucket(), len(payloadBytes),
		price, priceCurrency, priceExchange, priceTimestamp,
		revenueTTM, revenueQuarterly, netIncomeTTM, netIncomeQuarterly,
		ebitTTM, ebitQuarterly, ebitdaTTM, ebitdaQuarterly,
		revenueGrowthYoY, epsGrowthYoY, ebitdaGrowthYoY,
		epsDilutedTTM, epsBasicTTM, sharesOutstanding,
		payloadBytes,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("failed to insert key stats: %w", err)
	}

	return id, nil
}
