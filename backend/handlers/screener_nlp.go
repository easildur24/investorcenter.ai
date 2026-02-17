package handlers

import (
	"net/http"

	"investorcenter-api/services"

	"github.com/gin-gonic/gin"
)

// geminiClient is initialized at startup; nil when GEMINI_API_KEY is not set.
var geminiClient *services.GeminiClient

func init() {
	geminiClient = services.NewGeminiClient()
}

// NLPQueryRequest is the JSON body for POST /api/v1/screener/nlp.
type NLPQueryRequest struct {
	Query string `json:"query" binding:"required"`
}

// PostScreenerNLP translates a natural language query into screener filter params
// using the Gemini LLM.
//
// POST /api/v1/screener/nlp
//
//	Request:  { "query": "show me tech companies with more than 2T market cap" }
//	Response: { "data": { "params": { "sectors": "Technology", "market_cap_min": 2e12 }, "explanation": "..." } }
func PostScreenerNLP(c *gin.Context) {
	// Check Gemini client is available
	if geminiClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "NLP service not available",
			"message": "AI search is not configured. GEMINI_API_KEY is required.",
		})
		return
	}

	// Parse request body
	var req NLPQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": "A non-empty 'query' field is required.",
		})
		return
	}

	// Validate query length
	if len(req.Query) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Query too long",
			"message": "Query must be 500 characters or fewer.",
		})
		return
	}

	// Call Gemini
	result, err := geminiClient.ParseScreenerQuery(req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process query",
			"message": "AI could not interpret the query. Try rephrasing.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}
