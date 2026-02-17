package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"
)

const (
	geminiBaseURL   = "https://generativelanguage.googleapis.com/v1beta/models"
	geminiModel     = "gemini-2.0-flash-lite"
	maxOutputTokens = 1024
)

// GeminiClient handles Google Gemini API requests for NLP screener queries.
type GeminiClient struct {
	APIKey string
	Client *http.Client
}

// NewGeminiClient creates a new Gemini client. Returns nil if API key is not set.
func NewGeminiClient() *GeminiClient {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil
	}
	return &GeminiClient{
		APIKey: apiKey,
		Client: &http.Client{Timeout: 30 * time.Second},
	}
}

// NLPResult holds the structured output from Gemini's screener query parsing.
type NLPResult struct {
	Params      map[string]interface{} `json:"params"`
	Explanation string                 `json:"explanation"`
}

// screenerSystemPrompt tells Gemini how to translate natural language into filter params.
const screenerSystemPrompt = `You are a stock screener filter translator. Convert the user's natural language query into structured JSON filter parameters.

Available filters (all optional):
- sectors: comma-separated string. Valid values: Technology, Healthcare, Financial Services, Consumer Cyclical, Consumer Defensive, Industrials, Energy, Basic Materials, Real Estate, Communication Services, Utilities
- market_cap_min / market_cap_max: market capitalization in dollars (e.g. 1000000000 for $1B, 1000000000000 for $1T, 2000000000000 for $2T)
- pe_min / pe_max: P/E ratio (typical range 0-100)
- pb_min / pb_max: P/B ratio (typical range 0-20)
- ps_min / ps_max: P/S ratio (typical range 0-30)
- roe_min / roe_max: Return on Equity in percent (e.g. 15 means 15%)
- roa_min / roa_max: Return on Assets in percent
- gross_margin_min / gross_margin_max: Gross Margin in percent (0-100)
- net_margin_min / net_margin_max: Net Margin in percent (-50 to 80)
- de_min / de_max: Debt-to-Equity ratio (e.g. 1.5)
- current_ratio_min / current_ratio_max: Current Ratio (e.g. 1.5)
- revenue_growth_min / revenue_growth_max: Revenue Growth YoY in percent
- eps_growth_min / eps_growth_max: EPS Growth YoY in percent
- dividend_yield_min / dividend_yield_max: Dividend Yield in percent
- payout_ratio_min / payout_ratio_max: Payout Ratio in percent
- consec_div_years_min: minimum Consecutive Dividend Years (integer)
- beta_min / beta_max: Beta coefficient (e.g. 0.5-2.0)
- dcf_upside_min / dcf_upside_max: DCF Fair Value Upside in percent
- ic_score_min / ic_score_max: IC Score rating 0-100
- sort: column to sort by (market_cap, pe_ratio, pb_ratio, ps_ratio, roe, roa, gross_margin, net_margin, debt_to_equity, current_ratio, revenue_growth, eps_growth_yoy, dividend_yield, payout_ratio, beta, dcf_upside_percent, ic_score)
- order: "asc" or "desc"

Important rules:
1. Use numeric values, not strings, for all numeric filters
2. For market cap: "billion" = multiply by 1e9, "trillion" = multiply by 1e12
3. For percentages (ROE, margins, yields, growth): use the number directly (e.g. "15%" = 15)
4. Only include filters the user explicitly or implicitly requests
5. If the user says "high" for a metric, use a reasonable minimum threshold
6. If the user says "low" for a metric, use a reasonable maximum threshold

Return ONLY valid JSON with no other text:
{"params": {...}, "explanation": "one-sentence human-readable summary of applied filters"}`

// geminiRequest is the REST API request body for Gemini generateContent.
type geminiRequest struct {
	Contents          []geminiContent         `json:"contents"`
	SystemInstruction *geminiContent          `json:"systemInstruction,omitempty"`
	GenerationConfig  *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

// geminiResponse is the REST API response from Gemini generateContent.
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// ParseScreenerQuery sends a natural language query to Gemini and returns structured filter params.
func (c *GeminiClient) ParseScreenerQuery(query string) (*NLPResult, error) {
	reqBody := geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: screenerSystemPrompt}},
		},
		Contents: []geminiContent{
			{
				Parts: []geminiPart{{Text: query}},
				Role:  "user",
			},
		},
		GenerationConfig: &geminiGenerationConfig{
			Temperature:     0.0,
			MaxOutputTokens: maxOutputTokens,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", geminiBaseURL, geminiModel, c.APIKey)
	resp, err := c.Client.Post(url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("Gemini API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gemini API returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	if geminiResp.Error != nil {
		return nil, fmt.Errorf("Gemini API error %d: %s", geminiResp.Error.Code, geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("Gemini returned no content")
	}

	text := geminiResp.Candidates[0].Content.Parts[0].Text
	return ExtractNLPResult(text)
}

// ExtractNLPResult parses a JSON object from text, handling cases where the LLM
// wraps JSON in markdown code fences or adds explanation text.
func ExtractNLPResult(text string) (*NLPResult, error) {
	// Try direct parse first
	var result NLPResult
	if err := json.Unmarshal([]byte(text), &result); err == nil && result.Params != nil {
		return &result, nil
	}

	// Regex fallback: find JSON object in response text
	re := regexp.MustCompile(`\{[\s\S]*\}`)
	match := re.FindString(text)
	if match == "" {
		return nil, fmt.Errorf("no JSON object found in Gemini response: %s", truncate(text, 200))
	}

	if err := json.Unmarshal([]byte(match), &result); err != nil {
		return nil, fmt.Errorf("failed to parse extracted JSON: %w", err)
	}

	if result.Params == nil {
		return nil, fmt.Errorf("parsed JSON has no 'params' field")
	}

	return &result, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
