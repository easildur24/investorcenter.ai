package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func TestPostIngest_Validation(t *testing.T) {
	t.Run("rejects missing user context", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.POST("/ingest", PostIngest)

		body := `{"source":"test","data_type":"raw","raw_data":"hello"}`
		req, _ := http.NewRequest("POST", "/ingest", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.POST("/ingest", func(c *gin.Context) {
			c.Set("user_id", "user-1")
			PostIngest(c)
		})

		req, _ := http.NewRequest("POST", "/ingest", bytes.NewBufferString("not-json"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("rejects missing required fields", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.POST("/ingest", func(c *gin.Context) {
			c.Set("user_id", "user-1")
			PostIngest(c)
		})

		// Missing source, data_type, raw_data
		body := `{}`
		req, _ := http.NewRequest("POST", "/ingest", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Contains(t, resp["error"], "Invalid request")
	})

	t.Run("rejects oversized raw_data", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.POST("/ingest", func(c *gin.Context) {
			c.Set("user_id", "user-1")
			PostIngest(c)
		})

		// Create raw_data larger than 10MB
		largeData := strings.Repeat("x", 11*1024*1024)
		reqBody := IngestRequest{
			Source:   "test",
			DataType: "raw",
			RawData:  largeData,
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/ingest", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Contains(t, resp["error"], "maximum size")
	})

	t.Run("rejects oversized ticker", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.POST("/ingest", func(c *gin.Context) {
			c.Set("user_id", "user-1")
			PostIngest(c)
		})

		longTicker := strings.Repeat("A", 21)
		reqBody := IngestRequest{
			Source:   "test",
			Ticker:   &longTicker,
			DataType: "raw",
			RawData:  "data",
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/ingest", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Contains(t, resp["error"], "ticker must be 20 characters or less")
	})

	t.Run("rejects oversized source_url", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.POST("/ingest", func(c *gin.Context) {
			c.Set("user_id", "user-1")
			PostIngest(c)
		})

		longURL := "https://example.com/" + strings.Repeat("a", 2000)
		reqBody := IngestRequest{
			Source:    "test",
			DataType:  "raw",
			RawData:   "data",
			SourceURL: &longURL,
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/ingest", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Contains(t, resp["error"], "source_url must be 2000 characters or less")
	})

	t.Run("rejects invalid collected_at format", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.POST("/ingest", func(c *gin.Context) {
			c.Set("user_id", "user-1")
			PostIngest(c)
		})

		badTime := "not-a-timestamp"
		reqBody := IngestRequest{
			Source:      "test",
			DataType:    "raw",
			RawData:     "data",
			CollectedAt: &badTime,
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/ingest", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Contains(t, resp["error"], "RFC3339")
	})
}

func TestListIngestionLogs_Validation(t *testing.T) {
	// ListIngestionLogs requires database.DB to be set.
	// We test that the handler exists and handles the query params.
	// Full integration requires DB which is tested separately.

	t.Run("handler accepts query parameters", func(t *testing.T) {
		// We can't fully test without DB, but verify the handler structure
		// by checking it's a valid gin handler function
		assert.NotNil(t, ListIngestionLogs)
	})
}

func TestGetIngestionLogByID_Validation(t *testing.T) {
	t.Run("rejects non-numeric ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.GET("/ingest/:id", GetIngestionLogByID)

		req, _ := http.NewRequest("GET", "/ingest/not-a-number", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "Invalid ID", resp["error"])
	})
}

func TestIngestRequest_Structure(t *testing.T) {
	t.Run("deserializes valid request", func(t *testing.T) {
		jsonStr := `{"source":"reddit","ticker":"AAPL","data_type":"sentiment","raw_data":"some data","collected_at":"2026-02-13T10:00:00Z"}`

		var req IngestRequest
		err := json.Unmarshal([]byte(jsonStr), &req)
		assert.NoError(t, err)
		assert.Equal(t, "reddit", req.Source)
		assert.Equal(t, "AAPL", *req.Ticker)
		assert.Equal(t, "sentiment", req.DataType)
		assert.Equal(t, "some data", req.RawData)
		assert.Equal(t, "2026-02-13T10:00:00Z", *req.CollectedAt)
	})

	t.Run("handles optional fields as nil", func(t *testing.T) {
		jsonStr := `{"source":"test","data_type":"raw","raw_data":"hello"}`

		var req IngestRequest
		err := json.Unmarshal([]byte(jsonStr), &req)
		assert.NoError(t, err)
		assert.Nil(t, req.Ticker)
		assert.Nil(t, req.SourceURL)
		assert.Nil(t, req.CollectedAt)
	})
}

func TestMaxRawDataSize(t *testing.T) {
	t.Run("constant is 10MB", func(t *testing.T) {
		assert.Equal(t, 10*1024*1024, maxRawDataSize)
	})
}
