# Alpha Vantage API Client - API Key Usage Guide

## How the API Key is Used

### 1. **API Key Flow**

```
Environment Variable → Go Application → Client Creation → API Request
```

### 2. **Step-by-Step Process**

#### Step 1: Set Environment Variable
```bash
# In .env file or environment
export ALPHA_VANTAGE_API_KEY="your_api_key_here"
```

#### Step 2: Application Reads the Key
```go
// In your main application or service initialization
package main

import (
    "os"
    "log"
    "investorcenter-api/services/alphavantage"
)

func main() {
    // Read API key from environment variable
    apiKey := os.Getenv("ALPHA_VANTAGE_API_KEY")
    if apiKey == "" {
        log.Fatal("ALPHA_VANTAGE_API_KEY environment variable is required")
    }
    
    // Create client with the API key
    client := alphavantage.NewClient(apiKey)
    
    // Use the client...
}
```

#### Step 3: Client Stores the Key
```go
// In client.go - The client stores the API key privately
type Client struct {
    apiKey         string  // Stored here
    httpClient     *http.Client
    baseURL        string
    logger         Logger
    rateLimiter    *RateLimiter
    circuitBreaker *CircuitBreaker
}

// NewClient creates a client with the API key
func NewClient(apiKey string) *Client {
    return &Client{
        apiKey: apiKey,  // API key is stored in the client
        // ... other fields
    }
}
```

#### Step 4: API Key Added to Requests
```go
// In GetGlobalQuote method - API key is added to the request URL
func (c *Client) GetGlobalQuote(ctx context.Context, symbol string) (*LiveQuote, error) {
    // Build request parameters
    params := url.Values{}
    params.Set("function", "GLOBAL_QUOTE")
    params.Set("symbol", strings.ToUpper(symbol))
    params.Set("apikey", c.apiKey)  // ← API KEY USED HERE
    
    // Create the full URL
    requestURL := fmt.Sprintf("%s?%s", c.baseURL, params.Encode())
    // Result: https://www.alphavantage.co/query?function=GLOBAL_QUOTE&symbol=AAPL&apikey=your_key
    
    // Make the HTTP request
    resp, err := c.doRequestWithRetry(ctx, requestURL)
    // ...
}
```

### 3. **Complete Usage Example**

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "investorcenter-api/services/alphavantage"
)

func main() {
    // 1. Get API key from environment
    apiKey := os.Getenv("ALPHA_VANTAGE_API_KEY")
    if apiKey == "" {
        log.Fatal("Please set ALPHA_VANTAGE_API_KEY environment variable")
    }
    
    // 2. Create client with configuration
    config := &alphavantage.ClientConfig{
        APIKey:               apiKey,
        MaxRequestsPerMinute: 5,
        MaxRequestsPerDay:    500,
        EnableCircuitBreaker: true,
    }
    client := alphavantage.NewClientWithConfig(config)
    
    // 3. Use the client to fetch data
    ctx := context.Background()
    
    // Get single quote
    quote, err := client.GetGlobalQuote(ctx, "AAPL")
    if err != nil {
        log.Printf("Error fetching quote: %v", err)
        return
    }
    
    fmt.Printf("AAPL Price: $%.2f\n", quote.Price)
    
    // Get batch quotes
    symbols := []string{"MSFT", "GOOGL", "AMZN"}
    quotes, err := client.GetBatchQuotes(ctx, symbols)
    if err != nil {
        log.Printf("Error fetching batch quotes: %v", err)
        return
    }
    
    for _, q := range quotes {
        fmt.Printf("%s: $%.2f\n", q.Symbol, q.Price)
    }
}
```

### 4. **Security Features**

The API key is protected through several mechanisms:

1. **Never Logged**: The logger never outputs the API key
2. **Not in Error Messages**: Errors don't include the API key
3. **HTTPS Only**: All requests use HTTPS
4. **URL Encoding**: Properly encoded in query parameters

### 5. **Testing with API Key**

For unit tests, the client can be mocked:
```go
// Tests use a mock server, not real API
func TestGetGlobalQuote(t *testing.T) {
    // Create test server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify API key is present in request
        apiKey := r.URL.Query().Get("apikey")
        if apiKey == "" {
            t.Error("API key missing from request")
        }
        
        // Return mock response
        w.Write([]byte(mockResponse))
    }))
    
    client := alphavantage.NewClient("test-key")
    client.baseURL = server.URL  // Point to test server
    // ...
}
```

### 6. **Production Deployment**

#### Docker
```dockerfile
# Dockerfile
FROM golang:1.21-alpine
# ... build steps ...
CMD ["./app"]
```

```bash
# Run with API key
docker run -e ALPHA_VANTAGE_API_KEY="your_key" myapp
```

#### Kubernetes
```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: backend
        image: investorcenter-api:latest
        env:
        - name: ALPHA_VANTAGE_API_KEY
          valueFrom:
            secretKeyRef:
              name: api-keys
              key: alpha-vantage-api-key
```

### 7. **API Key Validation**

The client doesn't validate the API key format, but Alpha Vantage will return an error if invalid:
- Empty API key → Request will fail
- Invalid API key → API returns error message
- Exceeded rate limit → Client handles with rate limiter

### 8. **Monitoring API Key Usage**

The client tracks usage through:
- **Rate Limiter**: Monitors requests per minute/day
- **Metrics Collector**: Can track API calls
- **Circuit Breaker**: Protects against API failures

```go
// Check remaining quota
minuteRemaining, dayRemaining := client.rateLimiter.GetRemainingQuota()
fmt.Printf("Remaining: %d/min, %d/day\n", minuteRemaining, dayRemaining)
```

## Summary

The API key flow is:
1. **Stored** in environment variable (`ALPHA_VANTAGE_API_KEY`)
2. **Read** by the Go application at startup
3. **Passed** to the Alpha Vantage client during initialization
4. **Included** in every API request as a query parameter
5. **Protected** by HTTPS, rate limiting, and error handling

The key is never hardcoded, logged, or exposed in error messages.