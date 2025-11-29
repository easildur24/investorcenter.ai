package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"investorcenter-api/models"
)

// FinancialsEndpoint represents a financials API endpoint type
type FinancialsEndpoint string

const (
	EndpointIncomeStatements FinancialsEndpoint = "income-statements"
	EndpointBalanceSheets    FinancialsEndpoint = "balance-sheets"
	EndpointCashFlowStatements FinancialsEndpoint = "cash-flow-statements"
	EndpointRatios           FinancialsEndpoint = "ratios"
)

// PolygonFinancialsClient handles Polygon.io Financials API requests
type PolygonFinancialsClient struct {
	*PolygonClient
	rateLimiter *RateLimiter
}

// RateLimiter provides simple rate limiting for API requests
type RateLimiter struct {
	mu           sync.Mutex
	requestCount int
	resetTime    time.Time
	maxRequests  int
	window       time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		maxRequests: maxRequests,
		window:      window,
		resetTime:   time.Now().Add(window),
	}
}

// Wait blocks until a request can be made
func (r *RateLimiter) Wait() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if now.After(r.resetTime) {
		r.requestCount = 0
		r.resetTime = now.Add(r.window)
	}

	if r.requestCount >= r.maxRequests {
		sleepDuration := r.resetTime.Sub(now)
		if sleepDuration > 0 {
			log.Printf("Rate limit reached, waiting %v", sleepDuration)
			time.Sleep(sleepDuration)
			r.requestCount = 0
			r.resetTime = time.Now().Add(r.window)
		}
	}

	r.requestCount++
}

// NewPolygonFinancialsClient creates a new Polygon Financials client
func NewPolygonFinancialsClient() *PolygonFinancialsClient {
	return &PolygonFinancialsClient{
		PolygonClient: NewPolygonClient(),
		// Polygon's free tier: 5 requests per minute for financials
		// Paid tier: Higher limits
		rateLimiter: NewRateLimiter(5, time.Minute),
	}
}

// FinancialsRequestParams contains parameters for financials API requests
type FinancialsRequestParams struct {
	Ticker       string
	Timeframe    string // "quarterly", "annual", "trailing_twelve_months"
	FiscalYear   *int
	FiscalQuarter *int
	Limit        int
	Sort         string // e.g., "period_end.desc"
}

// GetIncomeStatements fetches income statements from Polygon.io
func (c *PolygonFinancialsClient) GetIncomeStatements(ctx context.Context, params FinancialsRequestParams) (*models.PolygonFinancialsResponse, error) {
	return c.fetchFinancials(ctx, EndpointIncomeStatements, params)
}

// GetBalanceSheets fetches balance sheets from Polygon.io
func (c *PolygonFinancialsClient) GetBalanceSheets(ctx context.Context, params FinancialsRequestParams) (*models.PolygonFinancialsResponse, error) {
	return c.fetchFinancials(ctx, EndpointBalanceSheets, params)
}

// GetCashFlowStatements fetches cash flow statements from Polygon.io
func (c *PolygonFinancialsClient) GetCashFlowStatements(ctx context.Context, params FinancialsRequestParams) (*models.PolygonFinancialsResponse, error) {
	return c.fetchFinancials(ctx, EndpointCashFlowStatements, params)
}

// GetRatios fetches financial ratios from Polygon.io
func (c *PolygonFinancialsClient) GetRatios(ctx context.Context, params FinancialsRequestParams) (*models.PolygonRatiosResponse, error) {
	c.rateLimiter.Wait()

	reqURL := c.buildFinancialsURL(EndpointRatios, params)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ratios: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var ratiosResp models.PolygonRatiosResponse
	if err := json.NewDecoder(resp.Body).Decode(&ratiosResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &ratiosResp, nil
}

// fetchFinancials is the generic method for fetching financial statements
func (c *PolygonFinancialsClient) fetchFinancials(ctx context.Context, endpoint FinancialsEndpoint, params FinancialsRequestParams) (*models.PolygonFinancialsResponse, error) {
	c.rateLimiter.Wait()

	reqURL := c.buildFinancialsURL(endpoint, params)
	log.Printf("Fetching financials from: %s", strings.Replace(reqURL, c.APIKey, "***", 1))

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch financials: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var financialsResp models.PolygonFinancialsResponse
	if err := json.NewDecoder(resp.Body).Decode(&financialsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &financialsResp, nil
}

// buildFinancialsURL builds the URL for a financials API request
func (c *PolygonFinancialsClient) buildFinancialsURL(endpoint FinancialsEndpoint, params FinancialsRequestParams) string {
	baseURL := fmt.Sprintf("%s/vX/reference/financials", PolygonBaseURL)

	queryParams := url.Values{}
	queryParams.Set("apiKey", c.APIKey)

	if params.Ticker != "" {
		queryParams.Set("ticker", strings.ToUpper(params.Ticker))
	}

	if params.Timeframe != "" {
		queryParams.Set("timeframe", params.Timeframe)
	}

	if params.FiscalYear != nil {
		queryParams.Set("fiscal_year", strconv.Itoa(*params.FiscalYear))
	}

	if params.FiscalQuarter != nil {
		queryParams.Set("fiscal_period", fmt.Sprintf("Q%d", *params.FiscalQuarter))
	}

	if params.Limit > 0 {
		queryParams.Set("limit", strconv.Itoa(params.Limit))
	} else {
		queryParams.Set("limit", "100")
	}

	if params.Sort != "" {
		queryParams.Set("sort", params.Sort)
	} else {
		queryParams.Set("sort", "period_of_report_date")
		queryParams.Set("order", "desc")
	}

	return fmt.Sprintf("%s?%s", baseURL, queryParams.Encode())
}

// GetAllFinancialsWithPagination fetches all financials for a ticker, handling pagination
func (c *PolygonFinancialsClient) GetAllFinancialsWithPagination(ctx context.Context, params FinancialsRequestParams) ([]models.PolygonFinancialsData, error) {
	var allResults []models.PolygonFinancialsData
	nextURL := ""

	for {
		c.rateLimiter.Wait()

		var reqURL string
		if nextURL != "" {
			reqURL = nextURL + "&apiKey=" + c.APIKey
		} else {
			reqURL = c.buildFinancialsURL(EndpointIncomeStatements, params)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := c.Client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch financials: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
		}

		var financialsResp models.PolygonFinancialsResponse
		if err := json.NewDecoder(resp.Body).Decode(&financialsResp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		allResults = append(allResults, financialsResp.Results...)

		if financialsResp.NextURL == nil || *financialsResp.NextURL == "" {
			break
		}
		nextURL = *financialsResp.NextURL
	}

	return allResults, nil
}

// ParseFiscalPeriod converts Polygon's fiscal period string to quarter number
func ParseFiscalPeriod(fiscalPeriod string) *int {
	switch fiscalPeriod {
	case "Q1":
		q := 1
		return &q
	case "Q2":
		q := 2
		return &q
	case "Q3":
		q := 3
		return &q
	case "Q4":
		q := 4
		return &q
	case "FY", "TTM":
		return nil // Annual or TTM doesn't have a quarter
	default:
		return nil
	}
}

// ParseTimeframe converts Polygon's fiscal period to our timeframe enum
func ParseTimeframe(fiscalPeriod string) models.Timeframe {
	switch fiscalPeriod {
	case "Q1", "Q2", "Q3", "Q4":
		return models.TimeframeQuarterly
	case "FY":
		return models.TimeframeAnnual
	case "TTM":
		return models.TimeframeTTM
	default:
		return models.TimeframeQuarterly
	}
}

// ConvertPolygonToFinancialStatement converts Polygon API response to our model
func ConvertPolygonToFinancialStatement(data models.PolygonFinancialsData, tickerID int, statementType models.StatementType) (*models.FinancialStatement, error) {
	// Parse dates
	periodEnd, err := time.Parse("2006-01-02", data.EndDate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse end date: %w", err)
	}

	var periodStart *time.Time
	if data.StartDate != "" {
		ps, err := time.Parse("2006-01-02", data.StartDate)
		if err == nil {
			periodStart = &ps
		}
	}

	var filedDate *time.Time
	if data.FilingDate != "" {
		fd, err := time.Parse("2006-01-02", data.FilingDate)
		if err == nil {
			filedDate = &fd
		}
	}

	// Parse fiscal year
	fiscalYear, err := strconv.Atoi(data.FiscalYear)
	if err != nil {
		return nil, fmt.Errorf("failed to parse fiscal year: %w", err)
	}

	// Extract the appropriate statement data
	var financialData models.FinancialData
	switch statementType {
	case models.StatementTypeIncome:
		financialData = convertToFinancialData(data.Financials.IncomeStatement)
	case models.StatementTypeBalanceSheet:
		financialData = convertToFinancialData(data.Financials.BalanceSheet)
	case models.StatementTypeCashFlow:
		financialData = convertToFinancialData(data.Financials.CashFlowStatement)
	}

	// Get CIK
	cik := data.CIK
	var cikPtr *string
	if cik != "" {
		cikPtr = &cik
	}

	// Get source URL
	sourceURL := data.SourceFilingURL
	var sourceURLPtr *string
	if sourceURL != "" {
		sourceURLPtr = &sourceURL
	}

	// Get source filing type
	sourceType := data.SourceFilingType
	var sourceTypePtr *string
	if sourceType != "" {
		sourceTypePtr = &sourceType
	}

	return &models.FinancialStatement{
		TickerID:         tickerID,
		CIK:              cikPtr,
		StatementType:    statementType,
		Timeframe:        ParseTimeframe(data.FiscalPeriod),
		FiscalYear:       fiscalYear,
		FiscalQuarter:    ParseFiscalPeriod(data.FiscalPeriod),
		PeriodStart:      periodStart,
		PeriodEnd:        periodEnd,
		FiledDate:        filedDate,
		SourceFilingURL:  sourceURLPtr,
		SourceFilingType: sourceTypePtr,
		Data:             financialData,
	}, nil
}

// convertToFinancialData converts Polygon's financial values map to our generic map
func convertToFinancialData(values map[string]models.PolygonFinancialValue) models.FinancialData {
	data := make(models.FinancialData)
	for key, val := range values {
		// Store the raw value
		data[key] = val.Value
		// Also store metadata if needed
		data[key+"_label"] = val.Label
		data[key+"_unit"] = val.Unit
	}
	return data
}

// CalculateYoYChange calculates year-over-year percentage change
func CalculateYoYChange(current, previous float64) *float64 {
	if previous == 0 {
		return nil
	}
	change := (current - previous) / absFloat64(previous)
	return &change
}

// absFloat64 returns the absolute value of a float64
func absFloat64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// CalculateYoYChanges calculates YoY changes for all metrics between two periods
func CalculateYoYChanges(current, previous models.FinancialData) map[string]*float64 {
	changes := make(map[string]*float64)

	for key, currentVal := range current {
		// Skip metadata fields
		if strings.HasSuffix(key, "_label") || strings.HasSuffix(key, "_unit") {
			continue
		}

		currentFloat, okCurrent := currentVal.(float64)
		previousVal, exists := previous[key]
		previousFloat, okPrevious := previousVal.(float64)

		if okCurrent && exists && okPrevious && previousFloat != 0 {
			change := CalculateYoYChange(currentFloat, previousFloat)
			changes[key] = change
		}
	}

	return changes
}
