package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	summaryCacheKey = "market:summary:v1"
	summaryCacheTTL = 15 * time.Minute
)

// MarketSummaryResult is the response returned by GenerateMarketSummary.
type MarketSummaryResult struct {
	Summary   string `json:"summary"`
	Timestamp string `json:"timestamp"`
	Method    string `json:"method"` // "llm" or "template"
}

// marketDataPoint holds a single index/stock data point for prompt building.
type marketDataPoint struct {
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"changePercent"`
}

// summaryPromptData is the structured data sent to Gemini for summarization.
type summaryPromptData struct {
	Date      string            `json:"date"`
	Indices   []marketDataPoint `json:"indices"`
	Gainers   []marketDataPoint `json:"topGainers"`
	Losers    []marketDataPoint `json:"topLosers"`
	Sentiment string            `json:"overallSentiment"`
}

const marketSummarySystemPrompt = `You are a concise financial market reporter. Generate a 2-4 sentence market summary from the structured data provided. Rules:
- Maximum 200 words
- No financial advice or predictions
- No speculation about future moves
- Focus on what happened today: index movements, notable movers, and overall market tone
- Use plain language accessible to retail investors
- Reference specific numbers (percentages, index levels) when relevant
Return ONLY the summary text, no JSON wrapping or markdown.`

// SummaryGenerator creates market summaries using LLM with template fallback.
type SummaryGenerator struct {
	gemini      *GeminiClient
	redisClient *redis.Client
	polygon     *PolygonClient
}

// NewSummaryGenerator creates a SummaryGenerator. gemini may be nil if
// GEMINI_API_KEY is not set (will fall back to template-based summaries).
func NewSummaryGenerator() *SummaryGenerator {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	// Check Redis connectivity at startup (non-blocking — template fallback works without Redis)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis unavailable at startup (%s): %v — summaries will not be cached", redisAddr, err)
	}

	return &SummaryGenerator{
		gemini:      NewGeminiClient(),
		redisClient: rdb,
		polygon:     NewPolygonClient(),
	}
}

// GetCachedSummary returns the cached summary from Redis, or nil if not found.
func (sg *SummaryGenerator) GetCachedSummary(ctx context.Context) (*MarketSummaryResult, error) {
	cached, err := sg.redisClient.Get(ctx, summaryCacheKey).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis GET error: %w", err)
	}

	var result MarketSummaryResult
	if err := json.Unmarshal([]byte(cached), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached summary: %w", err)
	}
	return &result, nil
}

// GenerateMarketSummary builds market data, generates a summary (LLM or
// template fallback), caches it, and returns the result.
func (sg *SummaryGenerator) GenerateMarketSummary(ctx context.Context) (*MarketSummaryResult, error) {
	// Gather market data
	promptData, err := sg.gatherMarketData()
	if err != nil {
		return nil, fmt.Errorf("failed to gather market data: %w", err)
	}

	// Try LLM generation first
	if sg.gemini != nil {
		result, err := sg.generateWithLLM(promptData)
		if err != nil {
			log.Printf("LLM summary generation failed, falling back to template: %v", err)
		} else {
			sg.cacheSummary(ctx, result)
			return result, nil
		}
	}

	// Fallback to template-based summary
	result := sg.generateTemplate(promptData)
	sg.cacheSummary(ctx, result)
	return result, nil
}

// gatherMarketData fetches index snapshots and market movers from Polygon.io.
func (sg *SummaryGenerator) gatherMarketData() (*summaryPromptData, error) {
	polygonClient := sg.polygon

	data := &summaryPromptData{
		Date: time.Now().Format("Monday, January 2, 2006"),
	}

	// Fetch index snapshots
	indexTickers := []string{"I:SPX", "I:DJI", "I:COMP", "I:RUT", "I:VIX"}
	indexNames := map[string]string{
		"I:SPX":  "S&P 500",
		"I:DJI":  "Dow Jones",
		"I:COMP": "NASDAQ",
		"I:RUT":  "Russell 2000",
		"I:VIX":  "VIX",
	}

	indexResults, err := polygonClient.GetIndexSnapshots(indexTickers)
	if err != nil {
		log.Printf("Warning: index snapshots failed: %v", err)
		// Try ETF proxies as fallback
		etfFallbacks := []struct {
			Symbol string
			Name   string
		}{
			{"SPY", "S&P 500"},
			{"DIA", "Dow Jones"},
			{"QQQ", "NASDAQ"},
			{"IWM", "Russell 2000"},
		}

		for _, etf := range etfFallbacks {
			priceData, etfErr := polygonClient.GetStockRealTimePrice(etf.Symbol)
			if etfErr != nil {
				log.Printf("Warning: ETF proxy %s failed: %v", etf.Symbol, etfErr)
				continue
			}
			data.Indices = append(data.Indices, marketDataPoint{
				Name:          etf.Name,
				Price:         priceData.Price.InexactFloat64(),
				Change:        priceData.Change.InexactFloat64(),
				ChangePercent: priceData.ChangePercent.InexactFloat64(),
			})
		}
	} else {
		for _, r := range indexResults {
			name := indexNames[r.Symbol]
			if name == "" {
				name = r.Symbol
			}
			data.Indices = append(data.Indices, marketDataPoint{
				Name:          name,
				Price:         r.Value,
				Change:        r.Change,
				ChangePercent: r.ChangePercent,
			})
		}
	}

	// Fetch bulk snapshots for gainers/losers
	snapshots, err := polygonClient.GetBulkStockSnapshots()
	if err != nil {
		log.Printf("Warning: bulk snapshots failed: %v", err)
	} else {
		type stockMove struct {
			Ticker        string
			Price         float64
			ChangePercent float64
		}

		var validStocks []stockMove
		for _, t := range snapshots.Tickers {
			// Use Day.Close if available (after market close), else LastTrade.Price (during trading)
			price := t.Day.Close
			if price == 0 {
				price = t.LastTrade.Price
			}
			if price <= 0 || math.IsNaN(t.TodaysChangePerc) || math.IsInf(t.TodaysChangePerc, 0) {
				continue
			}
			if math.Abs(t.TodaysChangePerc) > 100 || t.Day.Volume < 100000 || price < 1.0 {
				continue
			}
			validStocks = append(validStocks, stockMove{
				Ticker:        t.Ticker,
				Price:         price,
				ChangePercent: t.TodaysChangePerc,
			})
		}

		// Sort by change percent descending
		sort.Slice(validStocks, func(i, j int) bool {
			return validStocks[i].ChangePercent > validStocks[j].ChangePercent
		})

		// Top 3 gainers
		for i := 0; i < len(validStocks) && i < 3; i++ {
			if validStocks[i].ChangePercent > 0 {
				data.Gainers = append(data.Gainers, marketDataPoint{
					Name:          validStocks[i].Ticker,
					Price:         validStocks[i].Price,
					ChangePercent: validStocks[i].ChangePercent,
				})
			}
		}

		// Top 3 losers (from end)
		for i := len(validStocks) - 1; i >= 0 && len(data.Losers) < 3; i-- {
			if validStocks[i].ChangePercent < 0 {
				data.Losers = append(data.Losers, marketDataPoint{
					Name:          validStocks[i].Ticker,
					Price:         validStocks[i].Price,
					ChangePercent: validStocks[i].ChangePercent,
				})
			}
		}
	}

	// Determine overall sentiment from indices
	if len(data.Indices) > 0 {
		positiveCount := 0
		nonVIXCount := 0
		for _, idx := range data.Indices {
			if idx.Name == "VIX" {
				continue
			}
			nonVIXCount++
			if idx.ChangePercent > 0 {
				positiveCount++
			}
		}
		if nonVIXCount > 0 {
			ratio := float64(positiveCount) / float64(nonVIXCount)
			if ratio >= 0.75 {
				data.Sentiment = "bullish"
			} else if ratio <= 0.25 {
				data.Sentiment = "bearish"
			} else {
				data.Sentiment = "mixed"
			}
		}
	}

	return data, nil
}

// generateWithLLM sends the structured data to Gemini and returns an LLM summary.
func (sg *SummaryGenerator) generateWithLLM(data *summaryPromptData) (*MarketSummaryResult, error) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal prompt data: %w", err)
	}

	reqBody := geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: marketSummarySystemPrompt}},
		},
		Contents: []geminiContent{
			{
				Parts: []geminiPart{{Text: string(dataJSON)}},
				Role:  "user",
			},
		},
		GenerationConfig: &geminiGenerationConfig{
			Temperature:     0.3,
			MaxOutputTokens: 256,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Gemini request: %w", err)
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", geminiBaseURL, geminiModel, sg.gemini.apiKey)
	resp, err := sg.gemini.client.Post(url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("Gemini API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Gemini response: %w", err)
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

	summary := strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text)
	if summary == "" {
		return nil, fmt.Errorf("Gemini returned empty summary")
	}

	return &MarketSummaryResult{
		Summary:   summary,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Method:    "llm",
	}, nil
}

// generateTemplate creates a deterministic summary from the market data
// without using an LLM. Used as a fallback when Gemini is unavailable.
func (sg *SummaryGenerator) generateTemplate(data *summaryPromptData) *MarketSummaryResult {
	var parts []string

	// Lead with major index movements
	if len(data.Indices) > 0 {
		var indexParts []string
		for _, idx := range data.Indices {
			if idx.Name == "VIX" {
				continue
			}
			direction := "up"
			if idx.ChangePercent < 0 {
				direction = "down"
			}
			indexParts = append(indexParts, fmt.Sprintf(
				"the %s %s %.2f%% to %.2f",
				idx.Name, direction, math.Abs(idx.ChangePercent), idx.Price,
			))
		}
		if len(indexParts) > 0 {
			parts = append(parts, "Markets closed with "+strings.Join(indexParts, ", ")+".")
		}
	}

	// Top gainers
	if len(data.Gainers) > 0 {
		var names []string
		for _, g := range data.Gainers {
			names = append(names, fmt.Sprintf("%s (+%.1f%%)", g.Name, g.ChangePercent))
		}
		parts = append(parts, "Top gainers included "+strings.Join(names, ", ")+".")
	}

	// Top losers
	if len(data.Losers) > 0 {
		var names []string
		for _, l := range data.Losers {
			names = append(names, fmt.Sprintf("%s (%.1f%%)", l.Name, l.ChangePercent))
		}
		parts = append(parts, "Notable decliners were "+strings.Join(names, ", ")+".")
	}

	summary := strings.Join(parts, " ")
	if summary == "" {
		summary = "Market data is currently unavailable. Please check back later."
	}

	return &MarketSummaryResult{
		Summary:   summary,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Method:    "template",
	}
}

// cacheSummary stores the summary in Redis with the configured TTL.
// Errors are logged but not propagated — caching failures must not block
// the HTTP response. The summary will simply be regenerated on the next request.
func (sg *SummaryGenerator) cacheSummary(ctx context.Context, result *MarketSummaryResult) {
	data, err := json.Marshal(result)
	if err != nil {
		log.Printf("Failed to marshal summary for caching: %v", err)
		return
	}
	if err := sg.redisClient.Set(ctx, summaryCacheKey, data, summaryCacheTTL).Err(); err != nil {
		log.Printf("Failed to cache market summary: %v", err)
	}
}
