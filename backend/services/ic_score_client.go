package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"investorcenter-api/models"
)

// ICScoreClient is an HTTP client for the IC Score service API
type ICScoreClient struct {
	baseURL    string
	httpClient *http.Client
}

// ICScoreAPIResponse represents the response from IC Score API financial endpoints
type ICScoreAPIResponse struct {
	Ticker        string                   `json:"ticker"`
	StatementType string                   `json:"statement_type"`
	Timeframe     string                   `json:"timeframe"`
	Periods       []ICScoreFinancialPeriod `json:"periods"`
	Metadata      ICScoreFinancialMetadata `json:"metadata"`
}

// ICScoreFinancialPeriod represents a single period from IC Score API
type ICScoreFinancialPeriod struct {
	FiscalYear         int                 `json:"fiscal_year"`
	FiscalQuarter      *int                `json:"fiscal_quarter"`
	PeriodEndDate      string              `json:"period_end_date"`
	FilingDate         *string             `json:"filing_date"`
	Revenue            *int64              `json:"revenue"`
	CostOfRevenue      *int64              `json:"cost_of_revenue"`
	GrossProfit        *int64              `json:"gross_profit"`
	OperatingExpenses  *int64              `json:"operating_expenses"`
	OperatingIncome    *int64              `json:"operating_income"`
	NetIncome          *int64              `json:"net_income"`
	EPSBasic           *float64            `json:"eps_basic"`
	EPSDiluted         *float64            `json:"eps_diluted"`
	SharesOutstanding  *int64              `json:"shares_outstanding"`
	TotalAssets        *int64              `json:"total_assets"`
	TotalLiabilities   *int64              `json:"total_liabilities"`
	ShareholdersEquity *int64              `json:"shareholders_equity"`
	CashAndEquivalents *int64              `json:"cash_and_equivalents"`
	ShortTermDebt      *int64              `json:"short_term_debt"`
	LongTermDebt       *int64              `json:"long_term_debt"`
	OperatingCashFlow  *int64              `json:"operating_cash_flow"`
	InvestingCashFlow  *int64              `json:"investing_cash_flow"`
	FinancingCashFlow  *int64              `json:"financing_cash_flow"`
	FreeCashFlow       *int64              `json:"free_cash_flow"`
	Capex              *int64              `json:"capex"`
	GrossMargin        *float64            `json:"gross_margin"`
	OperatingMargin    *float64            `json:"operating_margin"`
	NetMargin          *float64            `json:"net_margin"`
	ROE                *float64            `json:"roe"`
	ROA                *float64            `json:"roa"`
	CurrentRatio       *float64            `json:"current_ratio"`
	DebtToEquity       *float64            `json:"debt_to_equity"`
	YoYChange          map[string]*float64 `json:"yoy_change"`
}

// ICScoreFinancialMetadata contains company metadata
type ICScoreFinancialMetadata struct {
	CompanyName string `json:"company_name"`
}

// NewICScoreClient creates a new IC Score API client
func NewICScoreClient() *ICScoreClient {
	baseURL := os.Getenv("IC_SCORE_API_URL")
	if baseURL == "" {
		// Default to Kubernetes service URL in production
		baseURL = "http://ic-score-service:8000"
	}

	return &ICScoreClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetAnnualFinancials fetches annual financial statements from IC Score API
func (c *ICScoreClient) GetAnnualFinancials(ticker string, statementType string, limit int) (*ICScoreAPIResponse, error) {
	url := fmt.Sprintf("%s/api/financials/%s/annual?statement_type=%s&limit=%d", c.baseURL, ticker, statementType, limit)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call IC Score API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no annual financial data found for %s", ticker)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("IC Score API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result ICScoreAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode IC Score API response: %w", err)
	}

	return &result, nil
}

// GetTTMFinancials fetches TTM financial statements from IC Score API
func (c *ICScoreClient) GetTTMFinancials(ticker string, statementType string, limit int) (*ICScoreAPIResponse, error) {
	url := fmt.Sprintf("%s/api/financials/%s/ttm?statement_type=%s&limit=%d", c.baseURL, ticker, statementType, limit)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call IC Score API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no TTM financial data found for %s", ticker)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("IC Score API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result ICScoreAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode IC Score API response: %w", err)
	}

	return &result, nil
}

// ICScoreNewsArticle represents a news article with AI sentiment analysis from IC Score service
type ICScoreNewsArticle struct {
	ID             int64    `json:"id"`
	Title          string   `json:"title"`
	URL            string   `json:"url"`
	Source         string   `json:"source"`
	PublishedAt    string   `json:"published_at"`
	Summary        *string  `json:"summary"`
	Author         *string  `json:"author"`
	Tickers        []string `json:"tickers"`
	SentimentScore *float64 `json:"sentiment_score"` // -100 to +100
	SentimentLabel *string  `json:"sentiment_label"` // Positive, Negative, Neutral
	RelevanceScore *float64 `json:"relevance_score"` // 0 to 100
	ImageURL       *string  `json:"image_url"`
}

// ICScoreNewsResponse represents the response from the IC Score news endpoint
type ICScoreNewsResponse struct {
	Ticker   string               `json:"ticker"`
	Articles []ICScoreNewsArticle `json:"articles"`
	Count    int                  `json:"count"`
}

// GetNews fetches news articles with AI sentiment from IC Score service
func (c *ICScoreClient) GetNews(ticker string, limit int, days int) (*ICScoreNewsResponse, error) {
	url := fmt.Sprintf("%s/api/news/%s?limit=%d&days=%d", c.baseURL, ticker, limit, days)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call IC Score API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no news data found for %s", ticker)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("IC Score API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result ICScoreNewsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode IC Score API response: %w", err)
	}

	return &result, nil
}

// ConvertToFinancialPeriods converts IC Score API response to models.FinancialPeriod
func ConvertToFinancialPeriods(apiResponse *ICScoreAPIResponse, statementType models.StatementType) []models.FinancialPeriod {
	periods := make([]models.FinancialPeriod, len(apiResponse.Periods))

	for i, p := range apiResponse.Periods {
		// Build data map based on statement type
		data := make(map[string]interface{})

		switch statementType {
		case models.StatementTypeIncome:
			if p.Revenue != nil {
				data["revenue"] = *p.Revenue
			}
			if p.CostOfRevenue != nil {
				data["cost_of_revenue"] = *p.CostOfRevenue
			}
			if p.GrossProfit != nil {
				data["gross_profit"] = *p.GrossProfit
			}
			if p.OperatingExpenses != nil {
				data["operating_expenses"] = *p.OperatingExpenses
			}
			if p.OperatingIncome != nil {
				data["operating_income"] = *p.OperatingIncome
			}
			if p.NetIncome != nil {
				data["net_income"] = *p.NetIncome
			}
			if p.EPSBasic != nil {
				data["basic_earnings_per_share"] = *p.EPSBasic
			}
			if p.EPSDiluted != nil {
				data["diluted_earnings_per_share"] = *p.EPSDiluted
			}
			if p.GrossMargin != nil {
				data["gross_margin"] = *p.GrossMargin * 100
			}
			if p.OperatingMargin != nil {
				data["operating_margin"] = *p.OperatingMargin * 100
			}
			if p.NetMargin != nil {
				data["net_margin"] = *p.NetMargin * 100
			}

		case models.StatementTypeBalanceSheet:
			if p.TotalAssets != nil {
				data["total_assets"] = *p.TotalAssets
			}
			if p.TotalLiabilities != nil {
				data["total_liabilities"] = *p.TotalLiabilities
			}
			if p.ShareholdersEquity != nil {
				data["stockholders_equity"] = *p.ShareholdersEquity
			}
			if p.CashAndEquivalents != nil {
				data["cash_and_cash_equivalents"] = *p.CashAndEquivalents
			}
			if p.ShortTermDebt != nil {
				data["short_term_debt"] = *p.ShortTermDebt
			}
			if p.LongTermDebt != nil {
				data["long_term_debt"] = *p.LongTermDebt
			}
			if p.CurrentRatio != nil {
				data["current_ratio"] = *p.CurrentRatio
			}
			if p.DebtToEquity != nil {
				data["debt_to_equity"] = *p.DebtToEquity
			}
			if p.ROE != nil {
				data["return_on_equity"] = *p.ROE * 100
			}
			if p.ROA != nil {
				data["return_on_assets"] = *p.ROA * 100
			}

		case models.StatementTypeCashFlow:
			if p.OperatingCashFlow != nil {
				data["net_cash_flow_from_operating_activities"] = *p.OperatingCashFlow
			}
			if p.InvestingCashFlow != nil {
				data["net_cash_flow_from_investing_activities"] = *p.InvestingCashFlow
			}
			if p.FinancingCashFlow != nil {
				data["net_cash_flow_from_financing_activities"] = *p.FinancingCashFlow
			}
			if p.FreeCashFlow != nil {
				data["free_cash_flow"] = *p.FreeCashFlow
			}
			if p.Capex != nil {
				data["capital_expenditure"] = *p.Capex
			}
		}

		// Add shares outstanding to all statement types
		if p.SharesOutstanding != nil {
			data["shares_outstanding"] = *p.SharesOutstanding
		}

		periods[i] = models.FinancialPeriod{
			FiscalYear:    p.FiscalYear,
			FiscalQuarter: p.FiscalQuarter,
			PeriodEnd:     p.PeriodEndDate,
			FiledDate:     p.FilingDate,
			Data:          data,
			YoYChange:     p.YoYChange,
		}
	}

	return periods
}
