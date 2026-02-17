package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"investorcenter-api/services"

	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// PostScreenerNLP handler tests
// ---------------------------------------------------------------------------

func TestPostScreenerNLP_NoGeminiClient(t *testing.T) {
	original := geminiClient
	geminiClient = nil
	defer func() { geminiClient = original }()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	body := `{"query": "tech stocks"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/screener/nlp", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	PostScreenerNLP(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if _, ok := resp["error"]; !ok {
		t.Error("expected error field in response")
	}
}

func TestPostScreenerNLP_EmptyBody(t *testing.T) {
	original := geminiClient
	geminiClient = &services.GeminiClient{APIKey: "test-key"}
	defer func() { geminiClient = original }()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/screener/nlp", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	PostScreenerNLP(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestPostScreenerNLP_EmptyQuery(t *testing.T) {
	original := geminiClient
	geminiClient = &services.GeminiClient{APIKey: "test-key"}
	defer func() { geminiClient = original }()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	body := `{"query": ""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/screener/nlp", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	PostScreenerNLP(c)

	// Gin's "required" binding rejects empty strings
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestPostScreenerNLP_QueryTooLong(t *testing.T) {
	original := geminiClient
	geminiClient = &services.GeminiClient{APIKey: "test-key"}
	defer func() { geminiClient = original }()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	longQuery := strings.Repeat("a", 501)
	body, _ := json.Marshal(map[string]string{"query": longQuery})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/screener/nlp", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	PostScreenerNLP(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if msg, ok := resp["message"].(string); !ok || !strings.Contains(msg, "500") {
		t.Errorf("expected message mentioning 500-char limit, got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// ExtractNLPResult unit tests
// ---------------------------------------------------------------------------

func TestExtractNLPResult_DirectJSON(t *testing.T) {
	input := `{"params": {"sectors": "Technology", "market_cap_min": 2000000000000}, "explanation": "Tech stocks over $2T"}`
	result, err := services.ExtractNLPResult(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Explanation != "Tech stocks over $2T" {
		t.Errorf("expected explanation 'Tech stocks over $2T', got %q", result.Explanation)
	}
	if sectors, ok := result.Params["sectors"].(string); !ok || sectors != "Technology" {
		t.Errorf("expected sectors=Technology, got %v", result.Params["sectors"])
	}
}

func TestExtractNLPResult_WrappedInCodeFence(t *testing.T) {
	input := "```json\n{\"params\": {\"pe_max\": 15}, \"explanation\": \"Low P/E stocks\"}\n```"
	result, err := services.ExtractNLPResult(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Explanation != "Low P/E stocks" {
		t.Errorf("expected explanation 'Low P/E stocks', got %q", result.Explanation)
	}
}

func TestExtractNLPResult_WithExtraText(t *testing.T) {
	input := "Here are the filters:\n{\"params\": {\"roe_min\": 15}, \"explanation\": \"High ROE\"}\nHope this helps!"
	result, err := services.ExtractNLPResult(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Explanation != "High ROE" {
		t.Errorf("expected explanation 'High ROE', got %q", result.Explanation)
	}
}

func TestExtractNLPResult_NoJSON(t *testing.T) {
	input := "I cannot process this query."
	_, err := services.ExtractNLPResult(input)
	if err == nil {
		t.Error("expected error for input with no JSON")
	}
}

func TestExtractNLPResult_NoParamsField(t *testing.T) {
	input := `{"explanation": "no params here"}`
	_, err := services.ExtractNLPResult(input)
	if err == nil {
		t.Error("expected error when params field is missing")
	}
}
