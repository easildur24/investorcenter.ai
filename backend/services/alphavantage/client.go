package alphavantage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	baseURL = "https://www.alphavantage.co/query"

	// Endpoints
	functionGlobalQuote = "GLOBAL_QUOTE"
	functionBatchQuotes = "BATCH_STOCK_QUOTES"

	// Rate limits for free tier
	maxRequestsPerMinute = 5
	maxRequestsPerDay    = 500
)

// Client represents the Alpha Vantage API client
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string

	// Rate limiting
	rateLimiter *RateLimiter
}

// NewClient creates a new Alpha Vantage API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:     baseURL,
		rateLimiter: NewRateLimiter(maxRequestsPerMinute, maxRequestsPerDay),
	}
}

// GlobalQuote represents the response from GLOBAL_QUOTE endpoint
type GlobalQuote struct {
	GlobalQuote QuoteData `json:"Global Quote"`
}

// QuoteData represents the quote data
type QuoteData struct {
	Symbol           string `json:"01. symbol"`
	Open             string `json:"02. open"`
	High             string `json:"03. high"`
	Low              string `json:"04. low"`
	Price            string `json:"05. price"`
	Volume           string `json:"06. volume"`
	LatestTradingDay string `json:"07. latest trading day"`
	PreviousClose    string `json:"08. previous close"`
	Change           string `json:"09. change"`
	ChangePercent    string `json:"10. change percent"`
}

// BatchQuotesResponse represents the response from BATCH_STOCK_QUOTES endpoint
type BatchQuotesResponse struct {
	MetaData    BatchMetaData `json:"Meta Data"`
	StockQuotes []StockQuote  `json:"Stock Quotes"`
}

// BatchMetaData represents metadata for batch quotes
type BatchMetaData struct {
	Information string `json:"1. Information"`
	Notes       string `json:"2. Notes"`
	TimeZone    string `json:"3. Time Zone"`
}

// StockQuote represents a single stock quote in batch response
type StockQuote struct {
	Symbol    string `json:"1. symbol"`
	Price     string `json:"2. price"`
	Volume    string `json:"3. volume"`
	Timestamp string `json:"4. timestamp"`
}

// LiveQuote represents normalized quote data
type LiveQuote struct {
	Symbol        string
	Price         float64
	Volume        int64
	Open          float64
	High          float64
	Low           float64
	PreviousClose float64
	Change        float64
	ChangePercent float64
	Timestamp     time.Time
	MarketCap     int64 // Will be calculated separately
}

// GetGlobalQuote fetches real-time quote for a single symbol
func (c *Client) GetGlobalQuote(ctx context.Context, symbol string) (*LiveQuote, error) {
	// Check rate limits
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit error: %w", err)
	}

	// Build request URL
	params := url.Values{}
	params.Set("function", functionGlobalQuote)
	params.Set("symbol", strings.ToUpper(symbol))
	params.Set("apikey", c.apiKey)

	requestURL := fmt.Sprintf("%s?%s", c.baseURL, params.Encode())

	// Make request with retry logic
	resp, err := c.doRequestWithRetry(ctx, requestURL)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var result GlobalQuote
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for API errors
	if result.GlobalQuote.Symbol == "" {
		return nil, fmt.Errorf("no data found for symbol: %s", symbol)
	}

	// Convert to LiveQuote
	quote, err := c.parseGlobalQuote(result.GlobalQuote)
	if err != nil {
		return nil, fmt.Errorf("failed to parse quote: %w", err)
	}

	return quote, nil
}

// GetBatchQuotes fetches quotes for multiple symbols (up to 100)
func (c *Client) GetBatchQuotes(ctx context.Context, symbols []string) ([]*LiveQuote, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("no symbols provided")
	}

	if len(symbols) > 100 {
		return nil, fmt.Errorf("batch size exceeds maximum of 100 symbols")
	}

	// Check rate limits
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit error: %w", err)
	}

	// Build request URL
	params := url.Values{}
	params.Set("function", functionBatchQuotes)
	params.Set("symbols", strings.Join(symbols, ","))
	params.Set("apikey", c.apiKey)

	requestURL := fmt.Sprintf("%s?%s", c.baseURL, params.Encode())

	// Make request with retry logic
	resp, err := c.doRequestWithRetry(ctx, requestURL)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var result BatchQuotesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to LiveQuotes
	quotes := make([]*LiveQuote, 0, len(result.StockQuotes))
	for _, sq := range result.StockQuotes {
		quote, err := c.parseBatchQuote(sq)
		if err != nil {
			// Log error but continue processing other quotes
			fmt.Printf("Warning: failed to parse quote for %s: %v\n", sq.Symbol, err)
			continue
		}
		quotes = append(quotes, quote)
	}

	return quotes, nil
}

// parseGlobalQuote converts API response to LiveQuote
func (c *Client) parseGlobalQuote(data QuoteData) (*LiveQuote, error) {
	price, err := strconv.ParseFloat(data.Price, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid price: %s", data.Price)
	}

	volume, err := strconv.ParseInt(data.Volume, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid volume: %s", data.Volume)
	}

	open, _ := strconv.ParseFloat(data.Open, 64)
	high, _ := strconv.ParseFloat(data.High, 64)
	low, _ := strconv.ParseFloat(data.Low, 64)
	prevClose, _ := strconv.ParseFloat(data.PreviousClose, 64)
	change, _ := strconv.ParseFloat(data.Change, 64)

	// Parse change percent (remove % sign)
	changePercentStr := strings.TrimSuffix(data.ChangePercent, "%")
	changePercent, _ := strconv.ParseFloat(changePercentStr, 64)

	// Parse timestamp
	timestamp, _ := time.Parse("2006-01-02", data.LatestTradingDay)

	return &LiveQuote{
		Symbol:        data.Symbol,
		Price:         price,
		Volume:        volume,
		Open:          open,
		High:          high,
		Low:           low,
		PreviousClose: prevClose,
		Change:        change,
		ChangePercent: changePercent,
		Timestamp:     timestamp,
	}, nil
}

// parseBatchQuote converts batch API response to LiveQuote
func (c *Client) parseBatchQuote(data StockQuote) (*LiveQuote, error) {
	price, err := strconv.ParseFloat(data.Price, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid price: %s", data.Price)
	}

	volume, err := strconv.ParseInt(data.Volume, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid volume: %s", data.Volume)
	}

	// Parse timestamp
	timestamp, err := time.Parse("2006-01-02 15:04:05", data.Timestamp)
	if err != nil {
		timestamp = time.Now()
	}

	return &LiveQuote{
		Symbol:    data.Symbol,
		Price:     price,
		Volume:    volume,
		Timestamp: timestamp,
	}, nil
}

// doRequestWithRetry performs HTTP request with exponential backoff retry
func (c *Client) doRequestWithRetry(ctx context.Context, url string) (*http.Response, error) {
	maxRetries := 3
	baseDelay := time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Create request with context
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}

		// Make request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			// Network error, retry
			if attempt < maxRetries-1 {
				delay := baseDelay * time.Duration(1<<uint(attempt))
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(delay):
					continue
				}
			}
			return nil, err
		}

		// Check status code
		if resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		// Handle rate limiting (429) or server errors (5xx)
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			resp.Body.Close()

			if attempt < maxRetries-1 {
				delay := baseDelay * time.Duration(1<<uint(attempt))
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(delay):
					continue
				}
			}
		}

		// For other error codes, don't retry
		return resp, fmt.Errorf("API returned status code: %d", resp.StatusCode)
	}

	return nil, fmt.Errorf("max retries exceeded")
}

// CalculateMarketCap calculates market cap if shares outstanding is known
func (q *LiveQuote) CalculateMarketCap(sharesOutstanding int64) {
	if sharesOutstanding > 0 {
		q.MarketCap = int64(q.Price * float64(sharesOutstanding))
	}
}
