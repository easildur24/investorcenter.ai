package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"investorcenter-api/models"
)

var (
	PolygonBaseURL = "https://api.polygon.io"
)

// PolygonClient handles Polygon.io API requests
type PolygonClient struct {
	APIKey string
	Client *http.Client
}

// NewPolygonClient creates a new Polygon.io client
func NewPolygonClient() *PolygonClient {
	apiKey := os.Getenv("POLYGON_API_KEY")
	if apiKey == "" {
		apiKey = "demo" // Use demo key for testing
	}

	return &PolygonClient{
		APIKey: apiKey,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// QuoteResponse represents Polygon.io quote response
type PolygonQuoteResponse struct {
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
	Count     int    `json:"count"`
	Results   []struct {
		Value     float64 `json:"value"`
		LastQuote struct {
			Timestamp int64   `json:"timestamp"`
			Bid       float64 `json:"bid"`
			Ask       float64 `json:"ask"`
			Exchange  int     `json:"exchange"`
			BidSize   int     `json:"bid_size"`
			AskSize   int     `json:"ask_size"`
		} `json:"last_quote"`
		LastTrade struct {
			Timestamp  int64   `json:"timestamp"`
			Price      float64 `json:"price"`
			Size       int     `json:"size"`
			Exchange   int     `json:"exchange"`
			Conditions []int   `json:"conditions"`
		} `json:"last_trade"`
		Min struct {
			Value     float64 `json:"value"`
			Timestamp int64   `json:"timestamp"`
		} `json:"min"`
		Max struct {
			Value     float64 `json:"value"`
			Timestamp int64   `json:"timestamp"`
		} `json:"max"`
	} `json:"results"`
}

// PreviousCloseResponse represents previous close data
type PreviousCloseResponse struct {
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
	Count     int    `json:"count"`
	Results   []struct {
		Ticker       string  `json:"T"`
		Volume       float64 `json:"v"`
		VolumeWeight float64 `json:"vw"`
		Open         float64 `json:"o"`
		Close        float64 `json:"c"`
		High         float64 `json:"h"`
		Low          float64 `json:"l"`
		Timestamp    int64   `json:"t"`
		Transactions int     `json:"n"`
	} `json:"results"`
}

// GetQuote fetches quote for a symbol using Polygon.io
func (p *PolygonClient) GetQuote(symbol string) (*models.StockPrice, error) {
	// For crypto symbols, try cache first
	if strings.HasPrefix(symbol, "X:") {
		cryptoCache := GetCryptoCache()
		if cachedPrice, exists := cryptoCache.GetPrice(symbol); exists {
			log.Printf("🪙 Serving %s from crypto cache", symbol)
			return cachedPrice, nil
		}

		// Fallback to individual API call if not in cache
		log.Printf("⚠️ %s not in crypto cache, using individual API call", symbol)
		return p.GetCryptoRealTimePrice(symbol)
	}

	// For stocks, try cache first
	stockCache := GetStockCache()
	if cachedPrice, exists := stockCache.GetPrice(symbol); exists {
		log.Printf("📊 Serving %s from stock cache", symbol)
		return cachedPrice, nil
	}

	// Fallback to individual API call if not in cache
	log.Printf("⚠️ %s not in stock cache, using individual API call", symbol)
	return p.GetStockRealTimePrice(symbol)
}

// AggregatesResponse represents aggregates/bars data
type AggregatesResponse struct {
	Status       string `json:"status"`
	RequestID    string `json:"request_id"`
	Count        int    `json:"count"`
	ResultsCount int    `json:"resultsCount"`
	Adjusted     bool   `json:"adjusted"`
	Results      []struct {
		Ticker       string  `json:"T"`
		Volume       float64 `json:"v"`
		VolumeWeight float64 `json:"vw"`
		Open         float64 `json:"o"`
		Close        float64 `json:"c"`
		High         float64 `json:"h"`
		Low          float64 `json:"l"`
		Timestamp    int64   `json:"t"`
		Transactions int     `json:"n"`
	} `json:"results"`
	NextURL string `json:"next_url"`
}

// GetHistoricalData fetches historical price data
func (p *PolygonClient) GetHistoricalData(symbol string, timespan string, from string, to string) ([]models.ChartDataPoint, error) {
	url := fmt.Sprintf("%s/v2/aggs/ticker/%s/range/1/%s/%s/%s?adjusted=true&sort=asc&apikey=%s",
		PolygonBaseURL, strings.ToUpper(symbol), timespan, from, to, p.APIKey)

	resp, err := p.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch historical data: %w", err)
	}
	defer resp.Body.Close()

	var aggResp AggregatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&aggResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if aggResp.Status != "OK" && aggResp.Status != "DELAYED" {
		return nil, fmt.Errorf("API error: %s", aggResp.Status)
	}

	var dataPoints []models.ChartDataPoint
	for _, bar := range aggResp.Results {
		timestamp := time.Unix(bar.Timestamp/1000, 0)

		dataPoints = append(dataPoints, models.ChartDataPoint{
			Timestamp: timestamp,
			Open:      decimal.NewFromFloat(bar.Open),
			High:      decimal.NewFromFloat(bar.High),
			Low:       decimal.NewFromFloat(bar.Low),
			Close:     decimal.NewFromFloat(bar.Close),
			Volume:    int64(bar.Volume),
		})
	}

	return dataPoints, nil
}

// GetIntradayData fetches intraday data (1-minute bars)
func (p *PolygonClient) GetIntradayData(symbol string) ([]models.ChartDataPoint, error) {
	// Get the most recent trading day (not weekend)
	now := time.Now()
	var tradingDay time.Time

	// If it's weekend, go back to Friday
	switch now.Weekday() {
	case time.Saturday:
		tradingDay = now.AddDate(0, 0, -1) // Friday
	case time.Sunday:
		tradingDay = now.AddDate(0, 0, -2) // Friday
	default:
		tradingDay = now // Weekday
	}

	tradingDayStr := tradingDay.Format("2006-01-02")

	// Use 5-minute bars for better performance and smoother charts
	url := fmt.Sprintf("%s/v2/aggs/ticker/%s/range/5/minute/%s/%s?adjusted=true&sort=asc&apikey=%s",
		PolygonBaseURL, strings.ToUpper(symbol), tradingDayStr, tradingDayStr, p.APIKey)

	resp, err := p.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch intraday data: %w", err)
	}
	defer resp.Body.Close()

	var aggResp AggregatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&aggResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if aggResp.Status != "OK" && aggResp.Status != "DELAYED" {
		return nil, fmt.Errorf("API error: %s", aggResp.Status)
	}

	var dataPoints []models.ChartDataPoint
	for _, bar := range aggResp.Results {
		timestamp := time.Unix(bar.Timestamp/1000, 0)

		dataPoints = append(dataPoints, models.ChartDataPoint{
			Timestamp: timestamp,
			Open:      decimal.NewFromFloat(bar.Open),
			High:      decimal.NewFromFloat(bar.High),
			Low:       decimal.NewFromFloat(bar.Low),
			Close:     decimal.NewFromFloat(bar.Close),
			Volume:    int64(bar.Volume),
		})
	}

	return dataPoints, nil
}

// GetDailyData fetches daily data for a period
func (p *PolygonClient) GetDailyData(symbol string, days int) ([]models.ChartDataPoint, error) {
	to := time.Now()
	from := to.AddDate(0, 0, -days)

	return p.GetHistoricalData(symbol, "day", from.Format("2006-01-02"), to.Format("2006-01-02"))
}

// TickerBranding represents company branding info from Polygon
type TickerBranding struct {
	LogoURL string `json:"logo_url"`
	IconURL string `json:"icon_url"`
}

// TickerDetailsResponse represents ticker details
type TickerDetailsResponse struct {
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
	Results   struct {
		Ticker         string         `json:"ticker"`
		Name           string         `json:"name"`
		Market         string         `json:"market"`
		Locale         string         `json:"locale"`
		PrimaryExch    string         `json:"primary_exchange"`
		Type           string         `json:"type"`
		Active         bool           `json:"active"`
		CurrencyName   string         `json:"currency_name"`
		CIK            string         `json:"cik"`
		Composite      string         `json:"composite_figi"`
		ShareClass     string         `json:"share_class_figi"`
		Description    string         `json:"description"`
		HomepageURL    string         `json:"homepage_url"`
		TotalEmployees int            `json:"total_employees"`
		ListDate       string         `json:"list_date"`
		Branding       TickerBranding `json:"branding"`
	} `json:"results"`
}

// GetTickerDetails fetches detailed information about a ticker
func (p *PolygonClient) GetTickerDetails(symbol string) (*TickerDetailsResponse, error) {
	url := fmt.Sprintf("%s/v3/reference/tickers/%s?apikey=%s",
		PolygonBaseURL, strings.ToUpper(symbol), p.APIKey)

	resp, err := p.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ticker details: %w", err)
	}
	defer resp.Body.Close()

	var detailsResp TickerDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&detailsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if detailsResp.Status != "OK" {
		return nil, fmt.Errorf("API error: %s", detailsResp.Status)
	}

	return &detailsResp, nil
}

// GetMultipleQuotes fetches quotes for multiple symbols efficiently
// This method uses bulk snapshots and filters for the requested symbols
func (p *PolygonClient) GetMultipleQuotes(symbols []string) (map[string]*QuoteData, error) {
	if len(symbols) == 0 {
		return make(map[string]*QuoteData), nil
	}

	// Separate stocks and crypto
	stocks := []string{}
	cryptos := []string{}

	for _, symbol := range symbols {
		if strings.HasPrefix(symbol, "X:") {
			cryptos = append(cryptos, symbol)
		} else {
			stocks = append(stocks, symbol)
		}
	}

	quotes := make(map[string]*QuoteData)

	// Fetch stock quotes
	if len(stocks) > 0 {
		stockQuotes, err := p.getBulkStockQuotes(stocks)
		if err != nil {
			log.Printf("Warning: Failed to fetch bulk stock quotes: %v\n", err)
		} else {
			for symbol, quote := range stockQuotes {
				quotes[symbol] = quote
			}
		}
	}

	// Fetch crypto quotes
	if len(cryptos) > 0 {
		cryptoQuotes, err := p.getBulkCryptoQuotes(cryptos)
		if err != nil {
			log.Printf("Warning: Failed to fetch bulk crypto quotes: %v\n", err)
		} else {
			for symbol, quote := range cryptoQuotes {
				quotes[symbol] = quote
			}
		}
	}

	return quotes, nil
}

// QuoteData represents simplified quote data for alert processing and enrichment
type QuoteData struct {
	Symbol        string
	Price         float64
	ChangePercent float64 // Today's change percent (e.g., 2.5 means +2.5%)
	Volume        int64
	Timestamp     int64
}

// getBulkStockQuotes fetches quotes for multiple stock symbols
func (p *PolygonClient) getBulkStockQuotes(symbols []string) (map[string]*QuoteData, error) {
	// Fetch bulk snapshots
	bulkResp, err := p.GetBulkStockSnapshots()
	if err != nil {
		return nil, err
	}

	// Create map for quick lookup
	symbolMap := make(map[string]bool)
	for _, s := range symbols {
		symbolMap[strings.ToUpper(s)] = true
	}

	// Filter and convert to QuoteData
	quotes := make(map[string]*QuoteData)
	for _, ticker := range bulkResp.Tickers {
		if symbolMap[ticker.Ticker] {
			price := ticker.LastTrade.Price
			if price == 0 {
				price = ticker.PrevDay.Close
			}

			volume := int64(ticker.Day.Volume)
			if volume == 0 {
				volume = int64(ticker.PrevDay.Volume)
			}

			quotes[ticker.Ticker] = &QuoteData{
				Symbol:        ticker.Ticker,
				Price:         price,
				ChangePercent: ticker.TodaysChangePerc,
				Volume:        volume,
				Timestamp:     ticker.LastTrade.Timestamp,
			}
		}
	}

	return quotes, nil
}

// getBulkCryptoQuotes fetches quotes for multiple crypto symbols
func (p *PolygonClient) getBulkCryptoQuotes(symbols []string) (map[string]*QuoteData, error) {
	// Fetch bulk crypto snapshots
	bulkResp, err := p.GetBulkCryptoSnapshots()
	if err != nil {
		return nil, err
	}

	// Create map for quick lookup
	symbolMap := make(map[string]bool)
	for _, s := range symbols {
		symbolMap[strings.ToUpper(s)] = true
	}

	// Filter and convert to QuoteData
	quotes := make(map[string]*QuoteData)
	for _, ticker := range bulkResp.Tickers {
		if symbolMap[ticker.Ticker] {
			price := ticker.LastTrade.Price
			if price == 0 {
				price = ticker.PrevDay.Close
			}

			quotes[ticker.Ticker] = &QuoteData{
				Symbol:        ticker.Ticker,
				Price:         price,
				ChangePercent: ticker.TodaysChangePerc,
				Volume:        int64(ticker.Day.Volume),
				Timestamp:     ticker.LastTrade.Timestamp,
			}
		}
	}

	return quotes, nil
}

// IsMarketOpen checks if the US market is currently open
func (p *PolygonClient) IsMarketOpen() bool {
	now := time.Now()

	// Convert to Eastern Time (US market timezone)
	est, _ := time.LoadLocation("America/New_York")
	estTime := now.In(est)

	// Check if it's a weekday
	if estTime.Weekday() == time.Saturday || estTime.Weekday() == time.Sunday {
		return false
	}

	// Market hours: 9:30 AM - 4:00 PM EST
	marketOpen := time.Date(estTime.Year(), estTime.Month(), estTime.Day(), 9, 30, 0, 0, est)
	marketClose := time.Date(estTime.Year(), estTime.Month(), estTime.Day(), 16, 0, 0, 0, est)

	return estTime.After(marketOpen) && estTime.Before(marketClose)
}

// FinancialsResponse represents comprehensive financial statements response
type FinancialsResponse struct {
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
	Count     int    `json:"count"`
	Results   []struct {
		StartDate    string `json:"start_date"`
		EndDate      string `json:"end_date"`
		Timeframe    string `json:"timeframe"`
		FiscalPeriod string `json:"fiscal_period"`
		FiscalYear   string `json:"fiscal_year"`
		CIK          string `json:"cik"`
		CompanyName  string `json:"company_name"`
		Financials   struct {
			BalanceSheet struct {
				// Assets
				CurrentAssets struct {
					Value float64 `json:"value"`
					Unit  string  `json:"unit"`
					Label string  `json:"label"`
				} `json:"current_assets"`
				TotalAssets struct {
					Value float64 `json:"value"`
					Unit  string  `json:"unit"`
					Label string  `json:"label"`
				} `json:"assets"`
				Inventory struct {
					Value float64 `json:"value"`
					Unit  string  `json:"unit"`
					Label string  `json:"label"`
				} `json:"inventory"`
				// Liabilities
				CurrentLiabilities struct {
					Value float64 `json:"value"`
					Unit  string  `json:"unit"`
					Label string  `json:"label"`
				} `json:"current_liabilities"`
				TotalLiabilities struct {
					Value float64 `json:"value"`
					Unit  string  `json:"unit"`
					Label string  `json:"label"`
				} `json:"liabilities"`
				LongTermDebt struct {
					Value float64 `json:"value"`
					Unit  string  `json:"unit"`
					Label string  `json:"label"`
				} `json:"long_term_debt"`
				// Equity
				TotalEquity struct {
					Value float64 `json:"value"`
					Unit  string  `json:"unit"`
					Label string  `json:"label"`
				} `json:"equity"`
			} `json:"balance_sheet"`
			IncomeStatement struct {
				// Revenue & Profit
				Revenues struct {
					Value float64 `json:"value"`
					Unit  string  `json:"unit"`
					Label string  `json:"label"`
				} `json:"revenues"`
				CostOfRevenue struct {
					Value float64 `json:"value"`
					Unit  string  `json:"unit"`
					Label string  `json:"label"`
				} `json:"cost_of_revenue"`
				GrossProfit struct {
					Value float64 `json:"value"`
					Unit  string  `json:"unit"`
					Label string  `json:"label"`
				} `json:"gross_profit"`
				OperatingIncome struct {
					Value float64 `json:"value"`
					Unit  string  `json:"unit"`
					Label string  `json:"label"`
				} `json:"operating_income_loss"`
				NetIncome struct {
					Value float64 `json:"value"`
					Unit  string  `json:"unit"`
					Label string  `json:"label"`
				} `json:"net_income_loss"`
				// EPS & Shares
				BasicEPS struct {
					Value float64 `json:"value"`
					Unit  string  `json:"unit"`
					Label string  `json:"label"`
				} `json:"basic_earnings_per_share"`
				DilutedEPS struct {
					Value float64 `json:"value"`
					Unit  string  `json:"unit"`
					Label string  `json:"label"`
				} `json:"diluted_earnings_per_share"`
				BasicShares struct {
					Value float64 `json:"value"`
					Unit  string  `json:"unit"`
					Label string  `json:"label"`
				} `json:"basic_average_shares"`
				DilutedShares struct {
					Value float64 `json:"value"`
					Unit  string  `json:"unit"`
					Label string  `json:"label"`
				} `json:"diluted_average_shares"`
				// Expenses
				OperatingExpenses struct {
					Value float64 `json:"value"`
					Unit  string  `json:"unit"`
					Label string  `json:"label"`
				} `json:"operating_expenses"`
				RnD struct {
					Value float64 `json:"value"`
					Unit  string  `json:"unit"`
					Label string  `json:"label"`
				} `json:"research_and_development"`
			} `json:"income_statement"`
			CashFlowStatement struct {
				OperatingCashFlow struct {
					Value float64 `json:"value"`
					Unit  string  `json:"unit"`
					Label string  `json:"label"`
				} `json:"net_cash_flow_from_operating_activities"`
			} `json:"cash_flow_statement"`
		} `json:"financials"`
	} `json:"results"`
}

// GetFundamentals fetches comprehensive financial statements and calculates key ratios
func (p *PolygonClient) GetFundamentals(symbol string) (*models.Fundamentals, error) {
	url := fmt.Sprintf("%s/vX/reference/financials?ticker=%s&timeframe=ttm&limit=1&apikey=%s",
		PolygonBaseURL, strings.ToUpper(symbol), p.APIKey)

	resp, err := p.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch fundamentals: %w", err)
	}
	defer resp.Body.Close()

	var finResp FinancialsResponse
	if err := json.NewDecoder(resp.Body).Decode(&finResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if finResp.Status != "OK" || len(finResp.Results) == 0 {
		return nil, fmt.Errorf("no fundamental data for symbol: %s", symbol)
	}

	result := finResp.Results[0]
	fin := result.Financials

	// Get current price for ratio calculations
	priceData, err := p.GetQuote(symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get price for ratio calculations: %w", err)
	}

	price := priceData.Price.InexactFloat64()

	// Extract raw financial data
	revenue := fin.IncomeStatement.Revenues.Value
	netIncome := fin.IncomeStatement.NetIncome.Value
	grossProfit := fin.IncomeStatement.GrossProfit.Value
	operatingIncome := fin.IncomeStatement.OperatingIncome.Value
	basicEPS := fin.IncomeStatement.BasicEPS.Value
	dilutedEPS := fin.IncomeStatement.DilutedEPS.Value
	basicShares := fin.IncomeStatement.BasicShares.Value

	// Balance sheet data
	totalAssets := fin.BalanceSheet.TotalAssets.Value
	currentAssets := fin.BalanceSheet.CurrentAssets.Value
	totalLiabilities := fin.BalanceSheet.TotalLiabilities.Value
	currentLiabilities := fin.BalanceSheet.CurrentLiabilities.Value
	longTermDebt := fin.BalanceSheet.LongTermDebt.Value
	totalEquity := fin.BalanceSheet.TotalEquity.Value
	inventory := fin.BalanceSheet.Inventory.Value

	// Cash flow data
	operatingCashFlow := fin.CashFlowStatement.OperatingCashFlow.Value

	// Calculate key ratios with null safety
	var pe, pb, ps, roe, roa, grossMargin, operatingMargin, netMargin, currentRatio, quickRatio, debtToEquity *decimal.Decimal

	// 1. P/E Ratio (MOST IMPORTANT) - Price / EPS
	if basicEPS > 0 && price > 0 {
		pe = decimalPtr(price / basicEPS)
	}

	// 2. Margins (profitability)
	if revenue > 0 {
		if grossProfit > 0 {
			grossMargin = decimalPtr(grossProfit / revenue)
		}
		if operatingIncome > 0 {
			operatingMargin = decimalPtr(operatingIncome / revenue)
		}
		if netIncome > 0 {
			netMargin = decimalPtr(netIncome / revenue)
		}
		// P/S ratio
		if price > 0 && basicShares > 0 {
			marketCap := price * basicShares
			ps = decimalPtr(marketCap / revenue)
		}
	}

	// 3. Return ratios
	if totalEquity > 0 && netIncome > 0 {
		roe = decimalPtr(netIncome / totalEquity)
	}
	if totalAssets > 0 && netIncome > 0 {
		roa = decimalPtr(netIncome / totalAssets)
	}

	// 4. P/B ratio
	if totalEquity > 0 && price > 0 && basicShares > 0 {
		bookValuePerShare := totalEquity / basicShares
		pb = decimalPtr(price / bookValuePerShare)
	}

	// 5. Liquidity ratios
	if currentLiabilities > 0 {
		if currentAssets > 0 {
			currentRatio = decimalPtr(currentAssets / currentLiabilities)
		}
		// Quick ratio (without inventory)
		if currentAssets > 0 && inventory >= 0 {
			quickAssets := currentAssets - inventory
			if quickAssets > 0 {
				quickRatio = decimalPtr(quickAssets / currentLiabilities)
			}
		}
	}

	// 6. Debt ratio
	if totalEquity > 0 && totalLiabilities > 0 {
		debtToEquity = decimalPtr(totalLiabilities / totalEquity)
	}

	return &models.Fundamentals{
		Symbol:            symbol,
		Period:            "TTM",
		Year:              2024,
		PE:                pe, // NOW CALCULATED!
		PB:                pb,
		PS:                ps,
		Revenue:           decimalPtr(revenue),
		GrossProfit:       decimalPtr(grossProfit),
		OperatingIncome:   decimalPtr(operatingIncome),
		NetIncome:         decimalPtr(netIncome),
		EPS:               decimalPtr(basicEPS), // REAL EPS!
		EPSDiluted:        decimalPtr(dilutedEPS),
		GrossMargin:       grossMargin,
		OperatingMargin:   operatingMargin,
		NetMargin:         netMargin,
		ROE:               roe,
		ROA:               roa,
		TotalAssets:       decimalPtr(totalAssets),
		TotalLiabilities:  decimalPtr(totalLiabilities),
		TotalEquity:       decimalPtr(totalEquity),
		TotalDebt:         decimalPtr(longTermDebt),
		DebtToEquity:      debtToEquity,
		CurrentRatio:      currentRatio,
		QuickRatio:        quickRatio,
		OperatingCashFlow: decimalPtr(operatingCashFlow),
		UpdatedAt:         time.Now(),
	}, nil
}

// Helper function to create decimal pointer
func decimalPtr(f float64) *decimal.Decimal {
	d := decimal.NewFromFloat(f)
	return &d
}

// NewsResponse represents Polygon news API response
type NewsResponse struct {
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
	Count     int    `json:"count"`
	NextURL   string `json:"next_url"`
	Results   []struct {
		ID        string `json:"id"`
		Publisher struct {
			Name        string `json:"name"`
			HomepageURL string `json:"homepage_url"`
			LogoURL     string `json:"logo_url"`
			FaviconURL  string `json:"favicon_url"`
		} `json:"publisher"`
		Title        string   `json:"title"`
		Author       string   `json:"author"`
		PublishedUTC string   `json:"published_utc"`
		ArticleURL   string   `json:"article_url"`
		Tickers      []string `json:"tickers"`
		ImageURL     string   `json:"image_url"`
		Description  string   `json:"description"`
		Keywords     []string `json:"keywords"`
		Insights     []struct {
			Ticker             string `json:"ticker"`
			Sentiment          string `json:"sentiment"`
			SentimentReasoning string `json:"sentiment_reasoning"`
		} `json:"insights"`
	} `json:"results"`
}

// GetNews fetches news articles for a symbol
func (p *PolygonClient) GetNews(symbol string, limit int) ([]models.NewsArticle, error) {
	if limit <= 0 {
		limit = 30 // Default to 30 articles for pagination
	}

	url := fmt.Sprintf("%s/v2/reference/news?ticker=%s&limit=%d&apikey=%s",
		PolygonBaseURL, strings.ToUpper(symbol), limit, p.APIKey)

	resp, err := p.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch news: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("news API request failed with status: %d", resp.StatusCode)
	}

	var newsResp NewsResponse
	if err := json.NewDecoder(resp.Body).Decode(&newsResp); err != nil {
		return nil, fmt.Errorf("failed to decode news response: %w", err)
	}

	if newsResp.Status != "OK" {
		return nil, fmt.Errorf("news API error: %s", newsResp.Status)
	}

	var articles []models.NewsArticle
	for i, article := range newsResp.Results {
		// Parse published date
		publishedAt, err := time.Parse(time.RFC3339, article.PublishedUTC)
		if err != nil {
			publishedAt = time.Now()
		}

		// Get sentiment for this ticker
		sentiment := "Neutral"
		for _, insight := range article.Insights {
			if strings.EqualFold(insight.Ticker, symbol) {
				sentiment = cases.Title(language.English).String(insight.Sentiment)
				break
			}
		}

		articles = append(articles, models.NewsArticle{
			ID:          i + 1,
			Symbol:      symbol,
			Title:       article.Title,
			Summary:     article.Description,
			Author:      article.Author,
			Source:      article.Publisher.Name,
			URL:         article.ArticleURL,
			Sentiment:   sentiment,
			PublishedAt: publishedAt,
			CreatedAt:   time.Now(),
		})
	}

	return articles, nil
}

// PolygonTickersResponse represents the tickers list response
type PolygonTickersResponse struct {
	Status    string          `json:"status"`
	Count     int             `json:"count"`
	NextURL   string          `json:"next_url"`
	RequestID string          `json:"request_id"`
	Results   []PolygonTicker `json:"results"`
}

// PolygonTicker represents a single ticker from the API
type PolygonTicker struct {
	Ticker                    string  `json:"ticker"`
	Name                      string  `json:"name"`
	Market                    string  `json:"market"`
	Locale                    string  `json:"locale"`
	Type                      string  `json:"type"`
	Active                    bool    `json:"active"`
	CurrencyName              string  `json:"currency_name"`
	CIK                       string  `json:"cik,omitempty"`
	CompositeFigi             string  `json:"composite_figi,omitempty"`
	ShareClassFigi            string  `json:"share_class_figi,omitempty"`
	PrimaryExchange           string  `json:"primary_exchange,omitempty"`
	LastUpdatedUTC            string  `json:"last_updated_utc,omitempty"`
	DelistedUTC               *string `json:"delisted_utc,omitempty"`
	ListDate                  string  `json:"list_date,omitempty"`
	HomepageURL               string  `json:"homepage_url,omitempty"`
	MarketCap                 float64 `json:"market_cap,omitempty"`
	TotalEmployees            int     `json:"total_employees,omitempty"`
	PhoneNumber               string  `json:"phone_number,omitempty"`
	WeightedSharesOutstanding float64 `json:"weighted_shares_outstanding,omitempty"`
	SICCode                   string  `json:"sic_code,omitempty"`
	SICDescription            string  `json:"sic_description,omitempty"`
	// Crypto specific fields
	BaseCurrencySymbol string `json:"base_currency_symbol,omitempty"`
	BaseCurrencyName   string `json:"base_currency_name,omitempty"`
	CurrencySymbol     string `json:"currency_symbol,omitempty"`
	// Index specific fields
	SourceFeed string `json:"source_feed,omitempty"`
}

// GetAllTickers fetches all tickers with optional filters
func (p *PolygonClient) GetAllTickers(assetType string, limit int) ([]PolygonTicker, error) {
	var allTickers []PolygonTicker
	baseURL := fmt.Sprintf("%s/v3/reference/tickers", PolygonBaseURL)

	// API has a max of 1000 per request
	const maxPerRequest = 1000
	requestLimit := maxPerRequest
	if limit > 0 && limit < maxPerRequest {
		requestLimit = limit
	}

	// Build initial URL with filters
	params := fmt.Sprintf("?active=true&limit=%d&apikey=%s", requestLimit, p.APIKey)

	// Add asset type filters
	switch assetType {
	case "stocks":
		params += "&market=stocks&locale=us&type=CS"
	case "etf":
		params += "&market=stocks&type=ETF"
	case "crypto":
		params += "&market=crypto"
	case "indices":
		params += "&market=indices"
	case "all_equities":
		params += "&market=stocks&locale=us"
	default:
		// No additional filters - get everything
	}

	url := baseURL + params
	pageCount := 0

	for {
		pageCount++
		log.Printf("Fetching page %d (already have %d tickers)...", pageCount, len(allTickers))

		resp, err := p.Client.Get(url)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch tickers on page %d: %w", pageCount, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("API request failed with status: %d on page %d", resp.StatusCode, pageCount)
		}

		var tickersResp PolygonTickersResponse
		if err := json.NewDecoder(resp.Body).Decode(&tickersResp); err != nil {
			return nil, fmt.Errorf("failed to decode response on page %d: %w", pageCount, err)
		}

		if tickersResp.Status != "OK" {
			return nil, fmt.Errorf("API error on page %d: %s", pageCount, tickersResp.Status)
		}

		allTickers = append(allTickers, tickersResp.Results...)
		log.Printf("Page %d: fetched %d tickers (total: %d)", pageCount, len(tickersResp.Results), len(allTickers))

		// Check if there's more data to fetch
		if tickersResp.NextURL == "" {
			log.Printf("No more pages available. Total tickers fetched: %d", len(allTickers))
			break
		}

		// Check if we've reached the user-specified limit
		if limit > 0 && len(allTickers) >= limit {
			log.Printf("Reached user-specified limit of %d tickers", limit)
			break
		}

		// Use the next URL for pagination (add API key if not present)
		url = tickersResp.NextURL
		if !strings.Contains(url, "apikey=") {
			if strings.Contains(url, "?") {
				url = url + "&apikey=" + p.APIKey
			} else {
				url = url + "?apikey=" + p.APIKey
			}
		}

		// Add a small delay to avoid rate limiting
		time.Sleep(500 * time.Millisecond)
	}

	// Trim to requested limit if specified
	if limit > 0 && len(allTickers) > limit {
		allTickers = allTickers[:limit]
	}

	log.Printf("Finished fetching tickers. Total returned: %d", len(allTickers))
	return allTickers, nil
}

// GetTickersByType fetches tickers of a specific type
func (p *PolygonClient) GetTickersByType(tickerType string) ([]PolygonTicker, error) {
	url := fmt.Sprintf("%s/v3/reference/tickers?type=%s&active=true&limit=1000&apikey=%s",
		PolygonBaseURL, tickerType, p.APIKey)

	resp, err := p.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tickers by type: %w", err)
	}
	defer resp.Body.Close()

	var tickersResp PolygonTickersResponse
	if err := json.NewDecoder(resp.Body).Decode(&tickersResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if tickersResp.Status != "OK" {
		return nil, fmt.Errorf("API error: %s", tickersResp.Status)
	}

	return tickersResp.Results, nil
}

// MapExchangeCode maps Polygon exchange codes to readable names
func MapExchangeCode(code string) string {
	exchangeMap := map[string]string{
		"XNAS": "NASDAQ",
		"XNYS": "NYSE",
		"ARCX": "NYSE ARCA",
		"XASE": "NYSE MKT",
		"BATS": "CBOE BZX",
		"XOTC": "OTC",
		"XCBO": "CBOE",
		"XPHL": "PHLX",
		"XISX": "ISE",
	}

	if mapped, ok := exchangeMap[code]; ok {
		return mapped
	}
	return code
}

// MapAssetType converts Polygon type codes to our asset types
func MapAssetType(typeCode string) string {
	switch typeCode {
	case "CS":
		return "stock"
	case "ETF":
		return "etf"
	case "ETN":
		return "etn"
	case "FUND":
		return "fund"
	case "PFD":
		return "preferred"
	case "WARRANT":
		return "warrant"
	case "RIGHT":
		return "right"
	case "BOND":
		return "bond"
	case "ADRC", "ADRP", "ADRW", "ADRR":
		return "adr"
	case "IX":
		return "index"
	default:
		if strings.HasPrefix(typeCode, "X:") {
			return "crypto"
		}
		if strings.HasPrefix(typeCode, "I:") {
			return "index"
		}
		return "other"
	}
}

// Helper function to convert period to days
func GetDaysFromPeriod(period string) int {
	switch strings.ToUpper(period) {
	case "1D":
		return 1
	case "5D":
		return 5
	case "1W":
		return 7
	case "1M":
		return 30
	case "3M":
		return 90
	case "6M":
		return 180
	case "YTD":
		// Calculate days from January 1st of current year to today
		now := time.Now()
		startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		return int(now.Sub(startOfYear).Hours()/24) + 1
	case "1Y":
		return 365
	case "5Y":
		return 1825
	case "MAX":
		// Return 20 years of data (maximum available history)
		return 7300
	default:
		return 365
	}
}

// CryptoTradesResponse represents the crypto trades API response
type CryptoTradesResponse struct {
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
	Count     int    `json:"count"`
	Results   []struct {
		Conditions           []int   `json:"conditions"`
		Exchange             int     `json:"exchange"`
		Price                float64 `json:"price"`
		Size                 float64 `json:"size"`
		ParticipantTimestamp int64   `json:"participant_timestamp"`
	} `json:"results"`
}

// CryptoSnapshotResponse represents the crypto snapshot API response
type CryptoSnapshotResponse struct {
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
	Ticker    struct {
		Ticker           string  `json:"ticker"`
		TodaysChange     float64 `json:"todaysChange"`
		TodaysChangePerc float64 `json:"todaysChangePerc"`
		Day              struct {
			Open   float64 `json:"o"`
			High   float64 `json:"h"`
			Low    float64 `json:"l"`
			Close  float64 `json:"c"`
			Volume float64 `json:"v"`
		} `json:"day"`
		LastQuote struct {
			Timestamp int64   `json:"t"`
			Bid       float64 `json:"b"`
			Ask       float64 `json:"a"`
			Exchange  int     `json:"x"`
		} `json:"lastQuote"`
		LastTrade struct {
			Timestamp  int64   `json:"t"`
			Price      float64 `json:"p"`
			Size       float64 `json:"s"`
			Exchange   int     `json:"x"`
			Conditions []int   `json:"c"`
		} `json:"lastTrade"`
		Min struct {
			Timestamp int64   `json:"t"`
			Price     float64 `json:"av"`
		} `json:"min"`
		PrevDay struct {
			Open   float64 `json:"o"`
			High   float64 `json:"h"`
			Low    float64 `json:"l"`
			Close  float64 `json:"c"`
			Volume float64 `json:"v"`
		} `json:"prevDay"`
	} `json:"ticker"`
}

// StockSnapshotResponse represents the stock snapshot API response
type StockSnapshotResponse struct {
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
	Ticker    struct {
		Ticker           string  `json:"ticker"`
		TodaysChange     float64 `json:"todaysChange"`
		TodaysChangePerc float64 `json:"todaysChangePerc"`
		Day              struct {
			Open   float64 `json:"o"`
			High   float64 `json:"h"`
			Low    float64 `json:"l"`
			Close  float64 `json:"c"`
			Volume float64 `json:"v"`
		} `json:"day"`
		LastQuote struct {
			Timestamp int64   `json:"t"`
			Bid       float64 `json:"b"`
			Ask       float64 `json:"a"`
			Exchange  int     `json:"x"`
		} `json:"lastQuote"`
		LastTrade struct {
			Timestamp  int64   `json:"t"`
			Price      float64 `json:"p"`
			Size       float64 `json:"s"`
			Exchange   int     `json:"x"`
			Conditions []int   `json:"c"`
		} `json:"lastTrade"`
		Min struct {
			Timestamp int64   `json:"t"`
			Price     float64 `json:"av"`
		} `json:"min"`
		PrevDay struct {
			Open   float64 `json:"o"`
			High   float64 `json:"h"`
			Low    float64 `json:"l"`
			Close  float64 `json:"c"`
			Volume float64 `json:"v"`
		} `json:"prevDay"`
	} `json:"ticker"`
}

// UnifiedSnapshotResponse represents the Polygon v3 unified snapshot API response
type UnifiedSnapshotResponse struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	Results   []struct {
		Ticker       string `json:"ticker"`
		Name         string `json:"name"`
		Type         string `json:"type"`
		MarketStatus string `json:"market_status"` // "open", "closed", "early_trading", "late_trading"
		Session      struct {
			Change                    float64 `json:"change"`
			ChangePercent             float64 `json:"change_percent"`
			Close                     float64 `json:"close"`
			High                      float64 `json:"high"`
			Low                       float64 `json:"low"`
			Open                      float64 `json:"open"`
			Volume                    float64 `json:"volume"`
			PreviousClose             float64 `json:"previous_close"`
			EarlyTradingChange        float64 `json:"early_trading_change"`
			EarlyTradingChangePercent float64 `json:"early_trading_change_percent"`
			RegularTradingChange      float64 `json:"regular_trading_change"`
			RegularTradingChangePercent float64 `json:"regular_trading_change_percent"`
			LateTradingChange         float64 `json:"late_trading_change"`
			LateTradingChangePercent  float64 `json:"late_trading_change_percent"`
		} `json:"session"`
		LastTrade struct {
			Timestamp  int64   `json:"sip_timestamp"`
			Price      float64 `json:"price"`
			Size       float64 `json:"size"`
			Exchange   int     `json:"exchange"`
			Conditions []int   `json:"conditions"`
		} `json:"last_trade"`
		LastQuote struct {
			Timestamp int64   `json:"sip_timestamp"`
			Bid       float64 `json:"bid"`
			Ask       float64 `json:"ask"`
			BidSize   float64 `json:"bid_size"`
			AskSize   float64 `json:"ask_size"`
		} `json:"last_quote"`
	} `json:"results"`
}

// UnifiedSnapshotResult contains parsed session-aware price data
type UnifiedSnapshotResult struct {
	Symbol              string
	MarketSession       string // "regular", "pre_market", "after_hours", "closed"
	Price               decimal.Decimal
	Change              decimal.Decimal
	ChangePercent       decimal.Decimal
	Volume              int64
	Timestamp           time.Time
	// Regular session close data (populated during extended hours)
	RegularClosePrice   decimal.Decimal
	RegularChange       decimal.Decimal
	RegularChangePercent decimal.Decimal
	HasRegularClose     bool
	PreviousClose       decimal.Decimal
}

// GetUnifiedSnapshot fetches session-aware price data from Polygon v3 snapshot
func (p *PolygonClient) GetUnifiedSnapshot(symbol string) (*UnifiedSnapshotResult, error) {
	url := fmt.Sprintf("%s/v3/snapshot?ticker.any_of=%s&apikey=%s",
		PolygonBaseURL, strings.ToUpper(symbol), p.APIKey)

	resp, err := p.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch unified snapshot: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unified snapshot API returned status %d", resp.StatusCode)
	}

	var snapshotResp UnifiedSnapshotResponse
	if err := json.NewDecoder(resp.Body).Decode(&snapshotResp); err != nil {
		return nil, fmt.Errorf("failed to decode unified snapshot: %w", err)
	}

	if len(snapshotResp.Results) == 0 {
		return nil, fmt.Errorf("no snapshot data for symbol %s", symbol)
	}

	ticker := snapshotResp.Results[0]
	session := ticker.Session

	// Map Polygon market_status to our session names
	marketSession := mapMarketStatus(ticker.MarketStatus)

	// Use last trade price as current price
	currentPrice := ticker.LastTrade.Price
	if currentPrice == 0 {
		currentPrice = session.Close
	}
	if currentPrice == 0 {
		currentPrice = session.PreviousClose
	}

	// Parse timestamp
	var timestamp time.Time
	if ticker.LastTrade.Timestamp > 0 {
		timestamp = time.Unix(0, ticker.LastTrade.Timestamp) // SIP timestamp is nanoseconds
	} else {
		timestamp = time.Now()
	}

	result := &UnifiedSnapshotResult{
		Symbol:        strings.ToUpper(symbol),
		MarketSession: marketSession,
		Volume:        int64(session.Volume),
		Timestamp:     timestamp,
		PreviousClose: decimal.NewFromFloat(session.PreviousClose),
	}

	switch marketSession {
	case "after_hours":
		// Regular close = previous_close + regular_trading_change
		regularClose := session.PreviousClose + session.RegularTradingChange
		result.RegularClosePrice = decimal.NewFromFloat(regularClose)
		result.RegularChange = decimal.NewFromFloat(session.RegularTradingChange)
		result.RegularChangePercent = decimal.NewFromFloat(session.RegularTradingChangePercent)
		result.HasRegularClose = true
		// Extended price and change from regular close
		result.Price = decimal.NewFromFloat(currentPrice)
		result.Change = decimal.NewFromFloat(session.LateTradingChange)
		result.ChangePercent = decimal.NewFromFloat(session.LateTradingChangePercent)

	case "pre_market":
		// Previous close is the reference
		result.RegularClosePrice = decimal.NewFromFloat(session.PreviousClose)
		result.HasRegularClose = true
		// Extended price and change from previous close
		result.Price = decimal.NewFromFloat(currentPrice)
		result.Change = decimal.NewFromFloat(session.EarlyTradingChange)
		result.ChangePercent = decimal.NewFromFloat(session.EarlyTradingChangePercent)

	case "closed":
		// When market is closed, check if after-hours trading happened
		// (late_trading_change != 0 means AH session had trades)
		if session.LateTradingChange != 0 {
			// Show as after_hours with the AH data
			result.MarketSession = "after_hours"
			regularClose := session.PreviousClose + session.RegularTradingChange
			result.RegularClosePrice = decimal.NewFromFloat(regularClose)
			result.RegularChange = decimal.NewFromFloat(session.RegularTradingChange)
			result.RegularChangePercent = decimal.NewFromFloat(session.RegularTradingChangePercent)
			result.HasRegularClose = true
			result.Price = decimal.NewFromFloat(currentPrice)
			result.Change = decimal.NewFromFloat(session.LateTradingChange)
			result.ChangePercent = decimal.NewFromFloat(session.LateTradingChangePercent)
		} else {
			result.Price = decimal.NewFromFloat(currentPrice)
			result.Change = decimal.NewFromFloat(session.Change)
			result.ChangePercent = decimal.NewFromFloat(session.ChangePercent)
		}

	default: // "regular"
		result.Price = decimal.NewFromFloat(currentPrice)
		result.Change = decimal.NewFromFloat(session.Change)
		result.ChangePercent = decimal.NewFromFloat(session.ChangePercent)
	}

	return result, nil
}

// mapMarketStatus converts Polygon's market_status to our session names
func mapMarketStatus(status string) string {
	switch status {
	case "open":
		return "regular"
	case "early_trading":
		return "pre_market"
	case "late_trading":
		return "after_hours"
	default:
		return "closed"
	}
}

// GetCryptoRealTimePrice fetches real-time crypto price using snapshot API
// BulkStockSnapshotResponse represents the bulk stocks snapshot API response
type BulkStockSnapshotResponse struct {
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
	Count     int    `json:"count"`
	Tickers   []struct {
		Ticker           string  `json:"ticker"`
		TodaysChange     float64 `json:"todaysChange"`
		TodaysChangePerc float64 `json:"todaysChangePerc"`
		Day              struct {
			Open   float64 `json:"o"`
			High   float64 `json:"h"`
			Low    float64 `json:"l"`
			Close  float64 `json:"c"`
			Volume float64 `json:"v"`
		} `json:"day"`
		LastQuote struct {
			Timestamp int64   `json:"t"`
			Bid       float64 `json:"b"`
			Ask       float64 `json:"a"`
			Exchange  int     `json:"x"`
		} `json:"lastQuote"`
		LastTrade struct {
			Timestamp  int64   `json:"t"`
			Price      float64 `json:"p"`
			Size       float64 `json:"s"`
			Exchange   int     `json:"x"`
			Conditions []int   `json:"c"`
		} `json:"lastTrade"`
		Min struct {
			Timestamp int64   `json:"t"`
			Price     float64 `json:"av"`
		} `json:"min"`
		PrevDay struct {
			Open   float64 `json:"o"`
			High   float64 `json:"h"`
			Low    float64 `json:"l"`
			Close  float64 `json:"c"`
			Volume float64 `json:"v"`
		} `json:"prevDay"`
	} `json:"tickers"`
}

// GetBulkStockSnapshots fetches all US stock snapshots
func (p *PolygonClient) GetBulkStockSnapshots() (*BulkStockSnapshotResponse, error) {
	snapshotURL := fmt.Sprintf("%s/v2/snapshot/locale/us/markets/stocks/tickers?apikey=%s",
		PolygonBaseURL, p.APIKey)

	resp, err := p.Client.Get(snapshotURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bulk stock snapshots: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bulk snapshot API request failed with status: %d", resp.StatusCode)
	}

	var snapshotResp BulkStockSnapshotResponse
	if err := json.NewDecoder(resp.Body).Decode(&snapshotResp); err != nil {
		return nil, fmt.Errorf("failed to decode bulk snapshot response: %w", err)
	}

	if snapshotResp.Status != "OK" {
		return nil, fmt.Errorf("bulk snapshot API error: %s", snapshotResp.Status)
	}

	return &snapshotResp, nil
}

// BulkCryptoSnapshotResponse represents the bulk crypto snapshot API response
type BulkCryptoSnapshotResponse struct {
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
	Count     int    `json:"count"`
	Tickers   []struct {
		Ticker           string  `json:"ticker"`
		TodaysChange     float64 `json:"todaysChange"`
		TodaysChangePerc float64 `json:"todaysChangePerc"`
		Day              struct {
			Open   float64 `json:"o"`
			High   float64 `json:"h"`
			Low    float64 `json:"l"`
			Close  float64 `json:"c"`
			Volume float64 `json:"v"`
		} `json:"day"`
		LastQuote struct {
			Timestamp int64   `json:"t"`
			Bid       float64 `json:"b"`
			Ask       float64 `json:"a"`
			Exchange  int     `json:"x"`
		} `json:"lastQuote"`
		LastTrade struct {
			Timestamp  int64   `json:"t"`
			Price      float64 `json:"p"`
			Size       float64 `json:"s"`
			Exchange   int     `json:"x"`
			Conditions []int   `json:"c"`
		} `json:"lastTrade"`
		Min struct {
			Timestamp int64   `json:"t"`
			Price     float64 `json:"av"`
		} `json:"min"`
		PrevDay struct {
			Open   float64 `json:"o"`
			High   float64 `json:"h"`
			Low    float64 `json:"l"`
			Close  float64 `json:"c"`
			Volume float64 `json:"v"`
		} `json:"prevDay"`
	} `json:"tickers"`
}

// GetBulkCryptoSnapshots fetches all crypto snapshots
func (p *PolygonClient) GetBulkCryptoSnapshots() (*BulkCryptoSnapshotResponse, error) {
	snapshotURL := fmt.Sprintf("%s/v2/snapshot/locale/global/markets/crypto/tickers?apikey=%s",
		PolygonBaseURL, p.APIKey)

	resp, err := p.Client.Get(snapshotURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bulk crypto snapshots: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bulk crypto snapshot API request failed with status: %d", resp.StatusCode)
	}

	var snapshotResp BulkCryptoSnapshotResponse
	if err := json.NewDecoder(resp.Body).Decode(&snapshotResp); err != nil {
		return nil, fmt.Errorf("failed to decode bulk crypto snapshot response: %w", err)
	}

	if snapshotResp.Status != "OK" {
		return nil, fmt.Errorf("bulk crypto snapshot API error: %s", snapshotResp.Status)
	}

	return &snapshotResp, nil
}

func (p *PolygonClient) GetStockRealTimePrice(symbol string) (*models.StockPrice, error) {
	// Use the real-time stock snapshot endpoint
	snapshotURL := fmt.Sprintf("%s/v2/snapshot/locale/us/markets/stocks/tickers/%s?apikey=%s",
		PolygonBaseURL, strings.ToUpper(symbol), p.APIKey)

	resp, err := p.Client.Get(snapshotURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stock snapshot: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stock snapshot API request failed with status: %d", resp.StatusCode)
	}

	var snapshotResp StockSnapshotResponse
	if err := json.NewDecoder(resp.Body).Decode(&snapshotResp); err != nil {
		return nil, fmt.Errorf("failed to decode stock snapshot response: %w", err)
	}

	if snapshotResp.Status != "OK" {
		return nil, fmt.Errorf("stock snapshot API error for symbol %s: %s", symbol, snapshotResp.Status)
	}

	ticker := snapshotResp.Ticker

	// Use last trade price for most recent data during market hours
	currentPrice := ticker.LastTrade.Price
	if currentPrice == 0 {
		// Fallback to previous close if no last trade
		currentPrice = ticker.PrevDay.Close
	}

	// Use today's day data if available, otherwise previous day
	dayData := ticker.Day
	useTodayData := dayData.Open != 0
	if !useTodayData {
		dayData = ticker.PrevDay
	}

	// Calculate change - use today's open if market is open, otherwise previous close
	var basePrice float64
	if useTodayData {
		// Market is open - compare to today's open
		basePrice = dayData.Open
	} else {
		// Market is closed - compare to previous close
		basePrice = ticker.PrevDay.Close
	}

	change := decimal.NewFromFloat(currentPrice - basePrice)
	changePercent := decimal.Zero
	if basePrice != 0 {
		changePercent = change.Div(decimal.NewFromFloat(basePrice)).Mul(decimal.NewFromInt(100))
	}

	// Convert timestamp
	timestamp := time.Unix(ticker.LastTrade.Timestamp/1000000000, 0) // Nanoseconds to seconds
	if ticker.LastTrade.Timestamp == 0 {
		timestamp = time.Now() // Fallback to current time
	}

	return &models.StockPrice{
		Symbol:        symbol,
		Price:         decimal.NewFromFloat(currentPrice),
		Open:          decimal.NewFromFloat(dayData.Open),
		High:          decimal.NewFromFloat(dayData.High),
		Low:           decimal.NewFromFloat(dayData.Low),
		Close:         decimal.NewFromFloat(currentPrice), // Current price as close
		Volume:        int64(dayData.Volume),
		Change:        change,
		ChangePercent: changePercent,
		Timestamp:     timestamp,
	}, nil
}

func (p *PolygonClient) GetCryptoRealTimePrice(symbol string) (*models.StockPrice, error) {
	// Use the real-time crypto snapshot endpoint
	snapshotURL := fmt.Sprintf("%s/v2/snapshot/locale/global/markets/crypto/tickers/%s?apikey=%s",
		PolygonBaseURL, strings.ToUpper(symbol), p.APIKey)

	resp, err := p.Client.Get(snapshotURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch crypto snapshot: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("crypto snapshot API request failed with status: %d", resp.StatusCode)
	}

	var snapshotResp CryptoSnapshotResponse
	if err := json.NewDecoder(resp.Body).Decode(&snapshotResp); err != nil {
		return nil, fmt.Errorf("failed to decode crypto snapshot response: %w", err)
	}

	if snapshotResp.Status != "OK" {
		return nil, fmt.Errorf("crypto snapshot API error for symbol %s: %s", symbol, snapshotResp.Status)
	}

	ticker := snapshotResp.Ticker

	// Use last trade price for most recent data
	currentPrice := ticker.LastTrade.Price
	if currentPrice == 0 {
		// Fallback to day close if no last trade
		currentPrice = ticker.Day.Close
	}

	// Use today's change data from the snapshot
	change := decimal.NewFromFloat(ticker.TodaysChange)
	changePercent := decimal.NewFromFloat(ticker.TodaysChangePerc) // Already in percentage format

	// Use last trade timestamp for real-time timestamp
	var timestamp time.Time
	if ticker.LastTrade.Timestamp > 0 {
		timestamp = time.Unix(ticker.LastTrade.Timestamp/1000000000, 0) // Convert nanoseconds
	} else {
		timestamp = time.Now() // Fallback to current time
	}

	return &models.StockPrice{
		Symbol:        symbol,
		Price:         decimal.NewFromFloat(currentPrice),    // Real-time last trade price
		Open:          decimal.NewFromFloat(ticker.Day.Open), // Today's open
		High:          decimal.NewFromFloat(ticker.Day.High), // Today's high
		Low:           decimal.NewFromFloat(ticker.Day.Low),  // Today's low
		Close:         decimal.NewFromFloat(currentPrice),    // Current price
		Volume:        int64(ticker.Day.Volume),              // Today's volume
		Change:        change,                                // Today's change
		ChangePercent: changePercent,                         // Today's change percent
		Timestamp:     timestamp,                             // Real-time timestamp
	}, nil
}
