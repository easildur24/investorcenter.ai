package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	FMPBaseURL = "https://financialmodelingprep.com/stable"
)

// FMPClient handles Financial Modeling Prep API requests
type FMPClient struct {
	APIKey string
	Client *http.Client
}

// FMPRatiosTTM represents the response from FMP ratios-ttm endpoint
type FMPRatiosTTM struct {
	Symbol                   string   `json:"symbol"`
	GrossProfitMarginTTM     *float64 `json:"grossProfitMarginTTM"`
	NetProfitMarginTTM       *float64 `json:"netProfitMarginTTM"`
	OperatingProfitMarginTTM *float64 `json:"operatingProfitMarginTTM"`
	ReturnOnEquityTTM        *float64 `json:"returnOnEquityTTM"`
	ReturnOnAssetsTTM        *float64 `json:"returnOnAssetsTTM"`
	CurrentRatioTTM          *float64 `json:"currentRatioTTM"`
	QuickRatioTTM            *float64 `json:"quickRatioTTM"`
	DebtEquityRatioTTM       *float64 `json:"debtEquityRatioTTM"`
	DebtToAssetsRatioTTM     *float64 `json:"debtToAssetsRatioTTM"`
	PriceToEarningsRatioTTM  *float64 `json:"priceToEarningsRatioTTM"`
	PriceToBookRatioTTM      *float64 `json:"priceToBookRatioTTM"`
	PriceToSalesRatioTTM     *float64 `json:"priceToSalesRatioTTM"`
	PriceToFreeCashFlowTTM   *float64 `json:"priceToFreeCashFlowRatioTTM"`
	EnterpriseValueTTM       *float64 `json:"enterpriseValueTTM"`
	EVToSalesTTM             *float64 `json:"evToSalesTTM"`
	EVToEBITDATTM            *float64 `json:"evToEbitdaTTM"`
	DividendYieldTTM         *float64 `json:"dividendYieldTTM"`
}

// NewFMPClient creates a new FMP API client
func NewFMPClient() *FMPClient {
	apiKey := os.Getenv("FMP_API_KEY")
	if apiKey == "" {
		log.Println("Warning: FMP_API_KEY not set, FMP features will be disabled")
	}
	return &FMPClient{
		APIKey: apiKey,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetRatiosTTM fetches TTM financial ratios for a ticker from FMP
func (c *FMPClient) GetRatiosTTM(ticker string) (*FMPRatiosTTM, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("FMP API key not configured")
	}

	url := fmt.Sprintf("%s/ratios-ttm?symbol=%s&apikey=%s", FMPBaseURL, ticker, c.APIKey)

	resp, err := c.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("FMP API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FMP API returned status %d", resp.StatusCode)
	}

	var results []FMPRatiosTTM
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode FMP response: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no FMP data found for %s", ticker)
	}

	return &results[0], nil
}

// ConvertToPercentage converts a decimal ratio (0.47) to percentage (47.0)
func ConvertToPercentage(val *float64) *float64 {
	if val == nil {
		return nil
	}
	pct := *val * 100
	return &pct
}

// MergedFinancialMetrics represents the merged data from FMP + our DB
type MergedFinancialMetrics struct {
	// Valuation ratios (prefer FMP)
	PERatio *float64 `json:"pe_ratio"`
	PBRatio *float64 `json:"pb_ratio"`
	PSRatio *float64 `json:"ps_ratio"`

	// Profitability (FMP primary, DB fallback)
	GrossMargin     *float64 `json:"gross_margin"`
	OperatingMargin *float64 `json:"operating_margin"`
	NetMargin       *float64 `json:"net_margin"`
	ROE             *float64 `json:"roe"`
	ROA             *float64 `json:"roa"`

	// Financial health (FMP primary, DB fallback)
	CurrentRatio *float64 `json:"current_ratio"`
	QuickRatio   *float64 `json:"quick_ratio"`
	DebtToEquity *float64 `json:"debt_to_equity"`

	// Source tracking
	FMPAvailable bool `json:"fmp_available"`
}

// MergeWithDBData merges FMP data with database data, preferring FMP when available
// This is a standalone function that safely handles nil fmp data
func MergeWithDBData(
	fmp *FMPRatiosTTM,
	dbGrossMargin, dbOperatingMargin, dbNetMargin *float64,
	dbROE, dbROA *float64,
	dbDebtToEquity, dbCurrentRatio, dbQuickRatio *float64,
	dbPERatio, dbPBRatio, dbPSRatio *float64,
) *MergedFinancialMetrics {
	merged := &MergedFinancialMetrics{
		FMPAvailable: fmp != nil,
	}

	if fmp != nil {
		// Valuation ratios from FMP (already in correct format)
		merged.PERatio = coalesce(fmp.PriceToEarningsRatioTTM, dbPERatio)
		merged.PBRatio = coalesce(fmp.PriceToBookRatioTTM, dbPBRatio)
		merged.PSRatio = coalesce(fmp.PriceToSalesRatioTTM, dbPSRatio)

		// Profitability - FMP stores as decimals (0.47), convert to % (47.0)
		merged.GrossMargin = coalesce(ConvertToPercentage(fmp.GrossProfitMarginTTM), dbGrossMargin)
		merged.OperatingMargin = coalesce(ConvertToPercentage(fmp.OperatingProfitMarginTTM), dbOperatingMargin)
		merged.NetMargin = coalesce(ConvertToPercentage(fmp.NetProfitMarginTTM), dbNetMargin)

		// ROE/ROA - FMP often null, fallback to DB
		merged.ROE = coalesce(ConvertToPercentage(fmp.ReturnOnEquityTTM), dbROE)
		merged.ROA = coalesce(ConvertToPercentage(fmp.ReturnOnAssetsTTM), dbROA)

		// Financial health
		merged.CurrentRatio = coalesce(fmp.CurrentRatioTTM, dbCurrentRatio)
		merged.QuickRatio = coalesce(fmp.QuickRatioTTM, dbQuickRatio)
		merged.DebtToEquity = coalesce(fmp.DebtEquityRatioTTM, dbDebtToEquity)
	} else {
		// No FMP data, use all DB values
		merged.PERatio = dbPERatio
		merged.PBRatio = dbPBRatio
		merged.PSRatio = dbPSRatio
		merged.GrossMargin = dbGrossMargin
		merged.OperatingMargin = dbOperatingMargin
		merged.NetMargin = dbNetMargin
		merged.ROE = dbROE
		merged.ROA = dbROA
		merged.CurrentRatio = dbCurrentRatio
		merged.QuickRatio = dbQuickRatio
		merged.DebtToEquity = dbDebtToEquity
	}

	return merged
}

// coalesce returns the first non-nil value
func coalesce(values ...*float64) *float64 {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}
