package services

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	FMPBaseURL = "https://financialmodelingprep.com/stable"
)

// FMPClient handles Financial Modeling Prep API requests
type FMPClient struct {
	APIKey string
	Client *http.Client
}

// ============================================================================
// FMP API Response Structs
// ============================================================================

// FMPRatiosTTM represents the response from FMP ratios-ttm endpoint (60+ fields)
type FMPRatiosTTM struct {
	Symbol string `json:"symbol"`

	// === PROFITABILITY MARGINS ===
	GrossProfitMarginTTM     *float64 `json:"grossProfitMarginTTM"`
	OperatingProfitMarginTTM *float64 `json:"operatingProfitMarginTTM"`
	NetProfitMarginTTM       *float64 `json:"netProfitMarginTTM"`
	EBITDAMarginTTM          *float64 `json:"ebitdaMarginTTM"`
	EBITMarginTTM            *float64 `json:"ebitMarginTTM"`
	FCFMarginTTM             *float64 `json:"freeCashFlowMarginTTM"`
	PretaxMarginTTM          *float64 `json:"pretaxProfitMarginTTM"`

	// === RETURNS ===
	ReturnOnEquityTTM          *float64 `json:"returnOnEquityTTM"`
	ReturnOnAssetsTTM          *float64 `json:"returnOnAssetsTTM"`
	ReturnOnInvestedCapitalTTM *float64 `json:"returnOnInvestedCapitalTTM"`
	ReturnOnCapitalEmployedTTM *float64 `json:"returnOnCapitalEmployedTTM"`

	// === LIQUIDITY ===
	CurrentRatioTTM *float64 `json:"currentRatioTTM"`
	QuickRatioTTM   *float64 `json:"quickRatioTTM"`
	CashRatioTTM    *float64 `json:"cashRatioTTM"`

	// === LEVERAGE ===
	DebtEquityRatioTTM   *float64 `json:"debtEquityRatioTTM"`
	DebtToAssetsRatioTTM *float64 `json:"debtToAssetsRatioTTM"`
	DebtToEBITDATTM      *float64 `json:"debtToEbitdaTTM"`
	DebtToCapitalTTM     *float64 `json:"debtToCapitalTTM"`
	InterestCoverageTTM  *float64 `json:"interestCoverageTTM"`
	NetDebtToEBITDATTM   *float64 `json:"netDebtToEbitdaTTM"`
	LongTermDebtToCapTTM *float64 `json:"longTermDebtToCapitalizationTTM"`
	TotalDebtToCapTTM    *float64 `json:"totalDebtToCapitalizationTTM"`
	FinancialLeverageTTM *float64 `json:"companyEquityMultiplierTTM"`

	// === VALUATION ===
	PriceToEarningsRatioTTM *float64 `json:"priceToEarningsRatioTTM"`
	PriceToBookRatioTTM     *float64 `json:"priceToBookRatioTTM"`
	PriceToSalesRatioTTM    *float64 `json:"priceToSalesRatioTTM"`
	PriceToFreeCashFlowTTM  *float64 `json:"priceToFreeCashFlowRatioTTM"`
	PriceToOperatingCFTTM   *float64 `json:"priceToOperatingCashFlowRatioTTM"`
	PriceToCashRatioTTM     *float64 `json:"priceCashFlowRatioTTM"`
	PEGRatioTTM             *float64 `json:"pegRatioTTM"`
	EarningsYieldTTM        *float64 `json:"earningsYieldTTM"`
	FCFYieldTTM             *float64 `json:"freeCashFlowYieldTTM"`
	PriceToTangibleBookTTM  *float64 `json:"priceToTangibleAssetsRatioTTM"`

	// === ENTERPRISE VALUE ===
	EnterpriseValueTTM         *float64 `json:"enterpriseValueTTM"`
	EVToSalesTTM               *float64 `json:"evToSalesTTM"`
	EVToEBITDATTM              *float64 `json:"evToEbitdaTTM"`
	EVToEBITTTM                *float64 `json:"evToEbitTTM"`
	EVToFCFTTM                 *float64 `json:"evToFreeCashFlowTTM"`
	EVToOperatingCFTTM         *float64 `json:"evToOperatingCashFlowTTM"`
	EnterpriseValueMultipleTTM *float64 `json:"enterpriseValueMultipleTTM"`

	// === EFFICIENCY ===
	AssetTurnoverTTM              *float64 `json:"assetTurnoverTTM"`
	InventoryTurnoverTTM          *float64 `json:"inventoryTurnoverTTM"`
	ReceivablesTurnoverTTM        *float64 `json:"receivablesTurnoverTTM"`
	PayablesTurnoverTTM           *float64 `json:"payablesTurnoverTTM"`
	FixedAssetTurnoverTTM         *float64 `json:"fixedAssetTurnoverTTM"`
	DaysOfSalesOutstandingTTM     *float64 `json:"daysOfSalesOutstandingTTM"`
	DaysOfInventoryOutstandingTTM *float64 `json:"daysOfInventoryOutstandingTTM"`
	DaysOfPayablesOutstandingTTM  *float64 `json:"daysOfPayablesOutstandingTTM"`
	CashConversionCycleTTM        *float64 `json:"cashConversionCycleTTM"`
	OperatingCycleTTM             *float64 `json:"operatingCycleTTM"`

	// === DIVIDENDS ===
	DividendYieldTTM       *float64 `json:"dividendYieldTTM"`
	PayoutRatioTTM         *float64 `json:"payoutRatioTTM"`
	DividendPerShareTTM    *float64 `json:"dividendPerShareTTM"`
	DividendPayoutRatioTTM *float64 `json:"dividendPayoutRatioTTM"`

	// === CASH FLOW ===
	OperatingCFPerShareTTM  *float64 `json:"operatingCashFlowPerShareTTM"`
	FreeCashFlowPerShareTTM *float64 `json:"freeCashFlowPerShareTTM"`
	CashPerShareTTM         *float64 `json:"cashPerShareTTM"`

	// === PER SHARE ===
	RevenuePerShareTTM            *float64 `json:"revenuePerShareTTM"`
	NetIncomePerShareTTM          *float64 `json:"netIncomePerShareTTM"`
	BookValuePerShareTTM          *float64 `json:"bookValuePerShareTTM"`
	TangibleBookPerShareTTM       *float64 `json:"tangibleBookValuePerShareTTM"`
	ShareholdersEquityPerShareTTM *float64 `json:"shareholdersEquityPerShareTTM"`

	// === OTHER ===
	CapexPerShareTTM *float64 `json:"capexPerShareTTM"`
	GrahamNumberTTM  *float64 `json:"grahamNumberTTM"`
	GrahamNetNetTTM  *float64 `json:"grahamNetNetTTM"`
}

// FMPKeyMetricsTTM represents the response from FMP key-metrics-ttm endpoint
type FMPKeyMetricsTTM struct {
	Symbol string `json:"symbol"`

	// === MARKET DATA ===
	MarketCapTTM       *float64 `json:"marketCapTTM"`
	EnterpriseValueTTM *float64 `json:"enterpriseValueTTM"`

	// === PER SHARE ===
	RevenuePerShareTTM            *float64 `json:"revenuePerShareTTM"`
	NetIncomePerShareTTM          *float64 `json:"netIncomePerShareTTM"`
	OperatingCashFlowPerShareTTM  *float64 `json:"operatingCashFlowPerShareTTM"`
	FreeCashFlowPerShareTTM       *float64 `json:"freeCashFlowPerShareTTM"`
	CashPerShareTTM               *float64 `json:"cashPerShareTTM"`
	BookValuePerShareTTM          *float64 `json:"bookValuePerShareTTM"`
	TangibleBookValuePerShareTTM  *float64 `json:"tangibleBookValuePerShareTTM"`
	ShareholdersEquityPerShareTTM *float64 `json:"shareholdersEquityPerShareTTM"`
	InterestDebtPerShareTTM       *float64 `json:"interestDebtPerShareTTM"`

	// === FINANCIAL DATA ===
	WorkingCapitalTTM     *float64 `json:"workingCapitalTTM"`
	NetDebtTTM            *float64 `json:"netDebtTTM"`
	AverageReceivablesTTM *float64 `json:"averageReceivablesTTM"`
	AveragePayablesTTM    *float64 `json:"averagePayablesTTM"`
	AverageInventoryTTM   *float64 `json:"averageInventoryTTM"`
	InvestedCapitalTTM    *float64 `json:"investedCapitalTTM"`
	TangibleAssetValueTTM *float64 `json:"tangibleAssetValueTTM"`

	// === RETURNS ===
	ROETTM                             *float64 `json:"roeTTM"`
	CapexToOperatingCashFlowTTM        *float64 `json:"capexToOperatingCashFlowTTM"`
	CapexToRevenueTTM                  *float64 `json:"capexToRevenueTTM"`
	CapexToDepreciationTTM             *float64 `json:"capexToDepreciationTTM"`
	StockBasedCompensationToRevenueTTM *float64 `json:"stockBasedCompensationToRevenueTTM"`

	// === SPECIAL ===
	GrahamNumberTTM *float64 `json:"grahamNumberTTM"`
	GrahamNetNetTTM *float64 `json:"grahamNetNetTTM"`

	// === VALUATION ===
	PERatioTTM           *float64 `json:"peRatioTTM"`
	PriceToSalesRatioTTM *float64 `json:"priceToSalesRatioTTM"`
	POCFRatioTTM         *float64 `json:"pocfratioTTM"`
	PFCFRatioTTM         *float64 `json:"pfcfRatioTTM"`
	PBRatioTTM           *float64 `json:"pbRatioTTM"`
	PTBRatioTTM          *float64 `json:"ptbRatioTTM"`
	EVToSalesTTM         *float64 `json:"evToSalesTTM"`
	EVToFreeCashFlowTTM  *float64 `json:"evToFreeCashFlowTTM"`
	EarningsYieldTTM     *float64 `json:"earningsYieldTTM"`
	FreeCashFlowYieldTTM *float64 `json:"freeCashFlowYieldTTM"`
	DebtToEquityTTM      *float64 `json:"debtToEquityTTM"`
	DebtToAssetsTTM      *float64 `json:"debtToAssetsTTM"`

	// === DIVIDENDS ===
	DividendYieldTTM           *float64 `json:"dividendYieldTTM"`
	DividendYieldPercentageTTM *float64 `json:"dividendYieldPercentageTTM"`
	PayoutRatioTTM             *float64 `json:"payoutRatioTTM"`
	DividendPerShareTTM        *float64 `json:"dividendPerShareTTM"`

	// === INCOME DATA ===
	IncomeQualityTTM                          *float64 `json:"incomeQualityTTM"`
	SalesGeneralAndAdministrativeToRevenueTTM *float64 `json:"salesGeneralAndAdministrativeToRevenueTTM"`
	ResearchAndDevelopmentToRevenueTTM        *float64 `json:"researchAndDevelopmentToRevenueTTM"`
	IntangiblesToTotalAssetsTTM               *float64 `json:"intangiblesToTotalAssetsTTM"`
	CapexPerShareTTM                          *float64 `json:"capexPerShareTTM"`
}

// FMPFinancialGrowth represents the response from FMP financial-growth endpoint
type FMPFinancialGrowth struct {
	Symbol       string `json:"symbol"`
	Date         string `json:"date"`
	Period       string `json:"period"`
	CalendarYear string `json:"calendarYear"`

	// === REVENUE GROWTH ===
	RevenueGrowth     *float64 `json:"revenueGrowth"`
	GrossProfitGrowth *float64 `json:"grossProfitGrowth"`

	// === INCOME GROWTH ===
	EBITGrowth            *float64 `json:"ebitgrowth"`
	OperatingIncomeGrowth *float64 `json:"operatingIncomeGrowth"`
	NetIncomeGrowth       *float64 `json:"netIncomeGrowth"`

	// === EPS GROWTH ===
	EPSGrowth        *float64 `json:"epsgrowth"`
	EPSDilutedGrowth *float64 `json:"epsdilutedGrowth"`

	// === DIVIDEND GROWTH ===
	DividendPerShareGrowth *float64 `json:"dividendsperShareGrowth"`

	// === CASH FLOW GROWTH ===
	OperatingCashFlowGrowth *float64 `json:"operatingCashFlowGrowth"`
	FreeCashFlowGrowth      *float64 `json:"freeCashFlowGrowth"`

	// === BALANCE SHEET GROWTH ===
	AssetGrowth                            *float64 `json:"assetGrowth"`
	BookValuePerShareGrowth                *float64 `json:"bookValueperShareGrowth"`
	DebtGrowth                             *float64 `json:"debtGrowth"`
	TenYRevenueGrowthPerShare              *float64 `json:"tenYRevenueGrowthPerShare"`
	FiveYRevenueGrowthPerShare             *float64 `json:"fiveYRevenueGrowthPerShare"`
	ThreeYRevenueGrowthPerShare            *float64 `json:"threeYRevenueGrowthPerShare"`
	TenYOperatingCFGrowthPerShare          *float64 `json:"tenYOperatingCFGrowthPerShare"`
	FiveYOperatingCFGrowthPerShare         *float64 `json:"fiveYOperatingCFGrowthPerShare"`
	ThreeYOperatingCFGrowthPerShare        *float64 `json:"threeYOperatingCFGrowthPerShare"`
	TenYNetIncomeGrowthPerShare            *float64 `json:"tenYNetIncomeGrowthPerShare"`
	FiveYNetIncomeGrowthPerShare           *float64 `json:"fiveYNetIncomeGrowthPerShare"`
	ThreeYNetIncomeGrowthPerShare          *float64 `json:"threeYNetIncomeGrowthPerShare"`
	TenYShareholdersEquityGrowthPerShare   *float64 `json:"tenYShareholdersEquityGrowthPerShare"`
	FiveYShareholdersEquityGrowthPerShare  *float64 `json:"fiveYShareholdersEquityGrowthPerShare"`
	ThreeYShareholdersEquityGrowthPerShare *float64 `json:"threeYShareholdersEquityGrowthPerShare"`
	TenYDividendPerShareGrowthPerShare     *float64 `json:"tenYDividendperShareGrowthPerShare"`
	FiveYDividendPerShareGrowthPerShare    *float64 `json:"fiveYDividendperShareGrowthPerShare"`
	ThreeYDividendPerShareGrowthPerShare   *float64 `json:"threeYDividendperShareGrowthPerShare"`

	// === RECEIVABLES/INVENTORY GROWTH ===
	ReceivablesGrowth *float64 `json:"receivablesGrowth"`
	InventoryGrowth   *float64 `json:"inventoryGrowth"`

	// === R&D AND SG&A GROWTH ===
	RDExpenseGrowth   *float64 `json:"rdexpenseGrowth"`
	SGAExpensesGrowth *float64 `json:"sgaexpensesGrowth"`
}

// FMPAnalystEstimate represents the response from FMP analyst-estimates endpoint
type FMPAnalystEstimate struct {
	Symbol string `json:"symbol"`
	Date   string `json:"date"`

	// === REVENUE ESTIMATES ===
	EstimatedRevenueLow  *float64 `json:"estimatedRevenueLow"`
	EstimatedRevenueHigh *float64 `json:"estimatedRevenueHigh"`
	EstimatedRevenueAvg  *float64 `json:"estimatedRevenueAvg"`

	// === EPS ESTIMATES ===
	EstimatedEPSLow  *float64 `json:"estimatedEpsLow"`
	EstimatedEPSHigh *float64 `json:"estimatedEpsHigh"`
	EstimatedEPSAvg  *float64 `json:"estimatedEpsAvg"`

	// === EBITDA ESTIMATES ===
	EstimatedEBITDALow  *float64 `json:"estimatedEbitdaLow"`
	EstimatedEBITDAHigh *float64 `json:"estimatedEbitdaHigh"`
	EstimatedEBITDAAvg  *float64 `json:"estimatedEbitdaAvg"`

	// === NET INCOME ESTIMATES ===
	EstimatedNetIncomeLow  *float64 `json:"estimatedNetIncomeLow"`
	EstimatedNetIncomeHigh *float64 `json:"estimatedNetIncomeHigh"`
	EstimatedNetIncomeAvg  *float64 `json:"estimatedNetIncomeAvg"`

	// === ANALYST COUNT ===
	NumberAnalystsEstimatedRevenue *int `json:"numberAnalystsEstimatedRevenue"`
	NumberAnalystsEstimatedEPS     *int `json:"numberAnalystsEstimatedEps"`
}

// FMPScore represents the response from FMP score endpoint (Altman Z, Piotroski F)
type FMPScore struct {
	Symbol         string   `json:"symbol"`
	AltmanZScore   *float64 `json:"altmanZScore"`
	PiotroskiScore *int     `json:"piotroskiScore"`
}

// FMPDividendHistorical represents historical dividend data
type FMPDividendHistorical struct {
	Symbol          string  `json:"symbol"`
	Date            string  `json:"date"`
	Label           string  `json:"label"`
	AdjDividend     float64 `json:"adjDividend"`
	Dividend        float64 `json:"dividend"`
	RecordDate      string  `json:"recordDate"`
	PaymentDate     string  `json:"paymentDate"`
	DeclarationDate string  `json:"declarationDate"`
}

// FMPGradesSummary represents the response from FMP grades-summary endpoint
type FMPGradesSummary struct {
	Symbol     string `json:"symbol"`
	StrongBuy  int    `json:"strongBuy"`
	Buy        int    `json:"buy"`
	Hold       int    `json:"hold"`
	Sell       int    `json:"sell"`
	StrongSell int    `json:"strongSell"`
	Consensus  string `json:"consensus"`
}

// FMPPriceTargetConsensus represents the response from FMP price-target-consensus endpoint
type FMPPriceTargetConsensus struct {
	Symbol          string   `json:"symbol"`
	TargetHigh      *float64 `json:"targetHigh"`
	TargetLow       *float64 `json:"targetLow"`
	TargetConsensus *float64 `json:"targetConsensus"`
	TargetMedian    *float64 `json:"targetMedian"`
}

// FMPPriceTargetSummary represents the response from FMP price-target-summary endpoint
type FMPPriceTargetSummary struct {
	Symbol                    string   `json:"symbol"`
	LastMonth                 *float64 `json:"lastMonth"`
	LastMonthAvgPriceTarget   *float64 `json:"lastMonthAvgPriceTarget"`
	LastQuarter               *float64 `json:"lastQuarter"`
	LastQuarterAvgPriceTarget *float64 `json:"lastQuarterAvgPriceTarget"`
	LastYear                  *float64 `json:"lastYear"`
	LastYearAvgPriceTarget    *float64 `json:"lastYearAvgPriceTarget"`
	AllTime                   *float64 `json:"allTime"`
	AllTimeAvgPriceTarget     *float64 `json:"allTimeAvgPriceTarget"`
	Publishers                string   `json:"publishers"`
}

// ============================================================================
// Client Constructor
// ============================================================================

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

// ============================================================================
// API Fetch Functions
// ============================================================================

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

// GetKeyMetricsTTM fetches TTM key metrics for a ticker from FMP
func (c *FMPClient) GetKeyMetricsTTM(ticker string) (*FMPKeyMetricsTTM, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("FMP API key not configured")
	}

	url := fmt.Sprintf("%s/key-metrics-ttm?symbol=%s&apikey=%s", FMPBaseURL, ticker, c.APIKey)

	resp, err := c.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("FMP key-metrics-ttm request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FMP key-metrics-ttm returned status %d", resp.StatusCode)
	}

	var results []FMPKeyMetricsTTM
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode FMP key-metrics-ttm response: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no FMP key-metrics-ttm data found for %s", ticker)
	}

	return &results[0], nil
}

// GetFinancialGrowth fetches historical financial growth data for a ticker
func (c *FMPClient) GetFinancialGrowth(ticker string, limit int) ([]FMPFinancialGrowth, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("FMP API key not configured")
	}

	url := fmt.Sprintf("%s/financial-growth?symbol=%s&period=annual&limit=%d&apikey=%s",
		FMPBaseURL, ticker, limit, c.APIKey)

	resp, err := c.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("FMP financial-growth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FMP financial-growth returned status %d", resp.StatusCode)
	}

	var results []FMPFinancialGrowth
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode FMP financial-growth response: %w", err)
	}

	return results, nil
}

// GetAnalystEstimates fetches forward analyst estimates for a ticker
func (c *FMPClient) GetAnalystEstimates(ticker string, limit int) ([]FMPAnalystEstimate, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("FMP API key not configured")
	}

	url := fmt.Sprintf("%s/analyst-estimates?symbol=%s&limit=%d&apikey=%s",
		FMPBaseURL, ticker, limit, c.APIKey)

	resp, err := c.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("FMP analyst-estimates request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FMP analyst-estimates returned status %d", resp.StatusCode)
	}

	var results []FMPAnalystEstimate
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode FMP analyst-estimates response: %w", err)
	}

	return results, nil
}

// GetScore fetches Altman Z-Score and Piotroski F-Score for a ticker
func (c *FMPClient) GetScore(ticker string) (*FMPScore, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("FMP API key not configured")
	}

	url := fmt.Sprintf("%s/score?symbol=%s&apikey=%s", FMPBaseURL, ticker, c.APIKey)

	resp, err := c.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("FMP score request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FMP score returned status %d", resp.StatusCode)
	}

	var results []FMPScore
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode FMP score response: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no FMP score data found for %s", ticker)
	}

	return &results[0], nil
}

// GetDividendHistory fetches historical dividend data for a ticker
func (c *FMPClient) GetDividendHistory(ticker string) ([]FMPDividendHistorical, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("FMP API key not configured")
	}

	url := fmt.Sprintf("%s/historical-price-eod/dividend/%s?apikey=%s",
		FMPBaseURL, ticker, c.APIKey)

	resp, err := c.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("FMP dividend history request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FMP dividend history returned status %d", resp.StatusCode)
	}

	var wrapper struct {
		Historical []FMPDividendHistorical `json:"historical"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("failed to decode FMP dividend history response: %w", err)
	}

	return wrapper.Historical, nil
}

// GetGradesSummary fetches analyst grades summary (strongBuy, buy, hold, sell, strongSell counts)
func (c *FMPClient) GetGradesSummary(ticker string) (*FMPGradesSummary, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("FMP API key not configured")
	}

	url := fmt.Sprintf("%s/grades-summary?symbol=%s&apikey=%s", FMPBaseURL, ticker, c.APIKey)

	resp, err := c.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("FMP grades-summary request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FMP grades-summary returned status %d", resp.StatusCode)
	}

	var results []FMPGradesSummary
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode FMP grades-summary response: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no FMP grades-summary data found for %s", ticker)
	}

	return &results[0], nil
}

// GetPriceTargetConsensus fetches analyst price target consensus data
func (c *FMPClient) GetPriceTargetConsensus(ticker string) (*FMPPriceTargetConsensus, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("FMP API key not configured")
	}

	url := fmt.Sprintf("%s/price-target-consensus?symbol=%s&apikey=%s", FMPBaseURL, ticker, c.APIKey)

	resp, err := c.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("FMP price-target-consensus request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FMP price-target-consensus returned status %d", resp.StatusCode)
	}

	var results []FMPPriceTargetConsensus
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode FMP price-target-consensus response: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no FMP price-target-consensus data found for %s", ticker)
	}

	return &results[0], nil
}

// ============================================================================
// Aggregated Data Fetch
// ============================================================================

// FMPAllMetrics contains all FMP data for a ticker
type FMPAllMetrics struct {
	RatiosTTM            *FMPRatiosTTM
	KeyMetricsTTM        *FMPKeyMetricsTTM
	Growth               []FMPFinancialGrowth
	Estimates            []FMPAnalystEstimate
	Score                *FMPScore
	Dividends            []FMPDividendHistorical
	GradesSummary        *FMPGradesSummary
	PriceTargetConsensus *FMPPriceTargetConsensus
	Errors               map[string]error
}

// GetAllMetrics fetches all FMP data for a ticker in parallel
func (c *FMPClient) GetAllMetrics(ticker string) *FMPAllMetrics {
	result := &FMPAllMetrics{
		Errors: make(map[string]error),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Fetch ratios-ttm
	wg.Add(1)
	go func() {
		defer wg.Done()
		data, err := c.GetRatiosTTM(ticker)
		mu.Lock()
		if err != nil {
			result.Errors["ratios-ttm"] = err
		} else {
			result.RatiosTTM = data
		}
		mu.Unlock()
	}()

	// Fetch key-metrics-ttm
	wg.Add(1)
	go func() {
		defer wg.Done()
		data, err := c.GetKeyMetricsTTM(ticker)
		mu.Lock()
		if err != nil {
			result.Errors["key-metrics-ttm"] = err
		} else {
			result.KeyMetricsTTM = data
		}
		mu.Unlock()
	}()

	// Fetch financial-growth (5 years)
	wg.Add(1)
	go func() {
		defer wg.Done()
		data, err := c.GetFinancialGrowth(ticker, 5)
		mu.Lock()
		if err != nil {
			result.Errors["financial-growth"] = err
		} else {
			result.Growth = data
		}
		mu.Unlock()
	}()

	// Fetch analyst-estimates (4 periods)
	wg.Add(1)
	go func() {
		defer wg.Done()
		data, err := c.GetAnalystEstimates(ticker, 4)
		mu.Lock()
		if err != nil {
			result.Errors["analyst-estimates"] = err
		} else {
			result.Estimates = data
		}
		mu.Unlock()
	}()

	// Fetch score
	wg.Add(1)
	go func() {
		defer wg.Done()
		data, err := c.GetScore(ticker)
		mu.Lock()
		if err != nil {
			result.Errors["score"] = err
		} else {
			result.Score = data
		}
		mu.Unlock()
	}()

	// Fetch dividend history
	wg.Add(1)
	go func() {
		defer wg.Done()
		data, err := c.GetDividendHistory(ticker)
		mu.Lock()
		if err != nil {
			result.Errors["dividends"] = err
		} else {
			result.Dividends = data
		}
		mu.Unlock()
	}()

	// Fetch grades summary (analyst ratings)
	wg.Add(1)
	go func() {
		defer wg.Done()
		data, err := c.GetGradesSummary(ticker)
		mu.Lock()
		if err != nil {
			result.Errors["grades-summary"] = err
		} else {
			result.GradesSummary = data
		}
		mu.Unlock()
	}()

	// Fetch price target consensus
	wg.Add(1)
	go func() {
		defer wg.Done()
		data, err := c.GetPriceTargetConsensus(ticker)
		mu.Lock()
		if err != nil {
			result.Errors["price-target-consensus"] = err
		} else {
			result.PriceTargetConsensus = data
		}
		mu.Unlock()
	}()

	wg.Wait()
	return result
}

// ============================================================================
// Utility Functions
// ============================================================================

// ConvertToPercentage converts a decimal ratio (0.47) to percentage (47.0)
func ConvertToPercentage(val *float64) *float64 {
	if val == nil {
		return nil
	}
	pct := *val * 100
	return &pct
}

// CalculateCAGR calculates Compound Annual Growth Rate
func CalculateCAGR(startValue, endValue float64, years int) *float64 {
	if startValue <= 0 || years <= 0 {
		return nil
	}
	cagr := math.Pow(endValue/startValue, 1.0/float64(years)) - 1
	result := cagr * 100 // Convert to percentage
	return &result
}

// GetZScoreInterpretation returns interpretation of Altman Z-Score
func GetZScoreInterpretation(score float64) (string, string) {
	if score > 2.99 {
		return "safe", "Low bankruptcy risk"
	}
	if score > 1.81 {
		return "grey", "Moderate risk"
	}
	return "distress", "High bankruptcy risk"
}

// GetFScoreInterpretation returns interpretation of Piotroski F-Score
func GetFScoreInterpretation(score int) (string, string) {
	if score >= 8 {
		return "strong", "Excellent financial health"
	}
	if score >= 5 {
		return "average", "Moderate financial health"
	}
	return "weak", "Poor financial health"
}

// GetPEGInterpretation returns interpretation of PEG ratio
func GetPEGInterpretation(peg float64) (string, string) {
	if peg < 1 {
		return "undervalued", "Growing faster than P/E suggests"
	}
	if peg <= 1.5 {
		return "fair", "Reasonably valued"
	}
	if peg <= 2 {
		return "high", "May be overvalued"
	}
	return "overvalued", "Price exceeds growth rate"
}

// GetPayoutRatioInterpretation returns interpretation of dividend payout ratio
func GetPayoutRatioInterpretation(ratio float64) (string, string) {
	if ratio < 30 {
		return "very_safe", "Room for dividend growth"
	}
	if ratio < 50 {
		return "safe", "Sustainable payout"
	}
	if ratio < 75 {
		return "moderate", "Limited room for growth"
	}
	return "at_risk", "May not be sustainable"
}

// ============================================================================
// Data Source Tracking
// ============================================================================

// DataSource indicates where a metric value came from
type DataSource string

const (
	SourceFMP        DataSource = "fmp"
	SourceDatabase   DataSource = "database"
	SourceCalculated DataSource = "calculated"
	SourceNone       DataSource = ""
)

// FieldSources tracks the data source for each field (for admin debug mode)
type FieldSources struct {
	// Valuation
	PERatio       DataSource `json:"pe_ratio,omitempty"`
	ForwardPE     DataSource `json:"forward_pe,omitempty"`
	PBRatio       DataSource `json:"pb_ratio,omitempty"`
	PSRatio       DataSource `json:"ps_ratio,omitempty"`
	PriceToFCF    DataSource `json:"price_to_fcf,omitempty"`
	PriceToOCF    DataSource `json:"price_to_ocf,omitempty"`
	PEGRatio      DataSource `json:"peg_ratio,omitempty"`
	EVToSales     DataSource `json:"ev_to_sales,omitempty"`
	EVToEBITDA    DataSource `json:"ev_to_ebitda,omitempty"`
	EVToEBIT      DataSource `json:"ev_to_ebit,omitempty"`
	EVToFCF       DataSource `json:"ev_to_fcf,omitempty"`
	EarningsYield DataSource `json:"earnings_yield,omitempty"`
	FCFYield      DataSource `json:"fcf_yield,omitempty"`
	MarketCap     DataSource `json:"market_cap,omitempty"`

	// Profitability
	GrossMargin     DataSource `json:"gross_margin,omitempty"`
	OperatingMargin DataSource `json:"operating_margin,omitempty"`
	NetMargin       DataSource `json:"net_margin,omitempty"`
	EBITDAMargin    DataSource `json:"ebitda_margin,omitempty"`
	EBITMargin      DataSource `json:"ebit_margin,omitempty"`
	FCFMargin       DataSource `json:"fcf_margin,omitempty"`
	ROE             DataSource `json:"roe,omitempty"`
	ROA             DataSource `json:"roa,omitempty"`
	ROIC            DataSource `json:"roic,omitempty"`
	ROCE            DataSource `json:"roce,omitempty"`

	// Liquidity
	CurrentRatio DataSource `json:"current_ratio,omitempty"`
	QuickRatio   DataSource `json:"quick_ratio,omitempty"`
	CashRatio    DataSource `json:"cash_ratio,omitempty"`

	// Leverage
	DebtToEquity     DataSource `json:"debt_to_equity,omitempty"`
	DebtToAssets     DataSource `json:"debt_to_assets,omitempty"`
	DebtToEBITDA     DataSource `json:"debt_to_ebitda,omitempty"`
	DebtToCapital    DataSource `json:"debt_to_capital,omitempty"`
	InterestCoverage DataSource `json:"interest_coverage,omitempty"`
	NetDebt          DataSource `json:"net_debt,omitempty"`

	// Efficiency
	AssetTurnover       DataSource `json:"asset_turnover,omitempty"`
	InventoryTurnover   DataSource `json:"inventory_turnover,omitempty"`
	ReceivablesTurnover DataSource `json:"receivables_turnover,omitempty"`
	PayablesTurnover    DataSource `json:"payables_turnover,omitempty"`
	DSO                 DataSource `json:"dso,omitempty"`
	DIO                 DataSource `json:"dio,omitempty"`
	DPO                 DataSource `json:"dpo,omitempty"`
	CashConversionCycle DataSource `json:"cash_conversion_cycle,omitempty"`

	// Quality
	AltmanZScore    DataSource `json:"altman_z_score,omitempty"`
	PiotroskiFScore DataSource `json:"piotroski_f_score,omitempty"`

	// Growth
	RevenueGrowthYoY DataSource `json:"revenue_growth_yoy,omitempty"`
	RevenueGrowth3Y  DataSource `json:"revenue_growth_3y,omitempty"`
	RevenueGrowth5Y  DataSource `json:"revenue_growth_5y,omitempty"`
	EPSGrowthYoY     DataSource `json:"eps_growth_yoy,omitempty"`
	EPSGrowth5Y      DataSource `json:"eps_growth_5y,omitempty"`

	// Dividends
	DividendYield        DataSource `json:"dividend_yield,omitempty"`
	PayoutRatio          DataSource `json:"payout_ratio,omitempty"`
	ForwardDividendYield DataSource `json:"forward_dividend_yield,omitempty"`
	FCFPayoutRatio       DataSource `json:"fcf_payout_ratio,omitempty"`

	// Per Share
	EPSDiluted DataSource `json:"eps_diluted,omitempty"`
}

// ============================================================================
// Merged Financial Metrics
// ============================================================================

// MergedFinancialMetrics represents the complete merged data from FMP + DB
type MergedFinancialMetrics struct {
	// === VALUATION (15 metrics) ===
	PERatio         *float64 `json:"pe_ratio"`
	ForwardPE       *float64 `json:"forward_pe"`
	PBRatio         *float64 `json:"pb_ratio"`
	PSRatio         *float64 `json:"ps_ratio"`
	PriceToFCF      *float64 `json:"price_to_fcf"`
	PriceToOCF      *float64 `json:"price_to_ocf"`
	PEGRatio        *float64 `json:"peg_ratio"`
	EnterpriseValue *float64 `json:"enterprise_value"`
	EVToSales       *float64 `json:"ev_to_sales"`
	EVToEBITDA      *float64 `json:"ev_to_ebitda"`
	EVToEBIT        *float64 `json:"ev_to_ebit"`
	EVToFCF         *float64 `json:"ev_to_fcf"`
	EarningsYield   *float64 `json:"earnings_yield"`
	FCFYield        *float64 `json:"fcf_yield"`
	MarketCap       *float64 `json:"market_cap"`

	// === PROFITABILITY (12 metrics) ===
	GrossMargin     *float64 `json:"gross_margin"`
	OperatingMargin *float64 `json:"operating_margin"`
	NetMargin       *float64 `json:"net_margin"`
	EBITDAMargin    *float64 `json:"ebitda_margin"`
	EBITMargin      *float64 `json:"ebit_margin"`
	FCFMargin       *float64 `json:"fcf_margin"`
	PretaxMargin    *float64 `json:"pretax_margin"`
	ROE             *float64 `json:"roe"`
	ROA             *float64 `json:"roa"`
	ROIC            *float64 `json:"roic"`
	ROCE            *float64 `json:"roce"`

	// === LIQUIDITY (4 metrics) ===
	CurrentRatio   *float64 `json:"current_ratio"`
	QuickRatio     *float64 `json:"quick_ratio"`
	CashRatio      *float64 `json:"cash_ratio"`
	WorkingCapital *float64 `json:"working_capital"`

	// === LEVERAGE (8 metrics) ===
	DebtToEquity     *float64 `json:"debt_to_equity"`
	DebtToAssets     *float64 `json:"debt_to_assets"`
	DebtToEBITDA     *float64 `json:"debt_to_ebitda"`
	DebtToCapital    *float64 `json:"debt_to_capital"`
	InterestCoverage *float64 `json:"interest_coverage"`
	NetDebtToEBITDA  *float64 `json:"net_debt_to_ebitda"`
	NetDebt          *float64 `json:"net_debt"`
	InvestedCapital  *float64 `json:"invested_capital"`

	// === EFFICIENCY (9 metrics) ===
	AssetTurnover              *float64 `json:"asset_turnover"`
	InventoryTurnover          *float64 `json:"inventory_turnover"`
	ReceivablesTurnover        *float64 `json:"receivables_turnover"`
	PayablesTurnover           *float64 `json:"payables_turnover"`
	FixedAssetTurnover         *float64 `json:"fixed_asset_turnover"`
	DaysOfSalesOutstanding     *float64 `json:"days_sales_outstanding"`
	DaysOfInventoryOutstanding *float64 `json:"days_inventory_outstanding"`
	DaysOfPayablesOutstanding  *float64 `json:"days_payables_outstanding"`
	CashConversionCycle        *float64 `json:"cash_conversion_cycle"`

	// === GROWTH (12 metrics) ===
	RevenueGrowthYoY         *float64 `json:"revenue_growth_yoy"`
	RevenueGrowth3YCAGR      *float64 `json:"revenue_growth_3y_cagr"`
	RevenueGrowth5YCAGR      *float64 `json:"revenue_growth_5y_cagr"`
	GrossProfitGrowthYoY     *float64 `json:"gross_profit_growth_yoy"`
	OperatingIncomeGrowthYoY *float64 `json:"operating_income_growth_yoy"`
	NetIncomeGrowthYoY       *float64 `json:"net_income_growth_yoy"`
	EPSGrowthYoY             *float64 `json:"eps_growth_yoy"`
	EPSGrowth3YCAGR          *float64 `json:"eps_growth_3y_cagr"`
	EPSGrowth5YCAGR          *float64 `json:"eps_growth_5y_cagr"`
	FCFGrowthYoY             *float64 `json:"fcf_growth_yoy"`
	BookValueGrowthYoY       *float64 `json:"book_value_growth_yoy"`
	DividendGrowth5YCAGR     *float64 `json:"dividend_growth_5y_cagr"`

	// === PER SHARE (11 metrics) ===
	EPSDiluted           *float64 `json:"eps_diluted"`
	BookValuePerShare    *float64 `json:"book_value_per_share"`
	TangibleBookPerShare *float64 `json:"tangible_book_per_share"`
	RevenuePerShare      *float64 `json:"revenue_per_share"`
	OperatingCFPerShare  *float64 `json:"operating_cf_per_share"`
	FCFPerShare          *float64 `json:"fcf_per_share"`
	CashPerShare         *float64 `json:"cash_per_share"`
	DividendPerShare     *float64 `json:"dividend_per_share"`
	GrahamNumber         *float64 `json:"graham_number"`
	InterestDebtPerShare *float64 `json:"interest_debt_per_share"`

	// === DIVIDENDS (9 metrics) ===
	DividendYield            *float64 `json:"dividend_yield"`
	ForwardDividendYield     *float64 `json:"forward_dividend_yield"`
	PayoutRatio              *float64 `json:"payout_ratio"`
	FCFPayoutRatio           *float64 `json:"fcf_payout_ratio"`
	ConsecutiveDividendYears *int     `json:"consecutive_dividend_years"`
	ExDividendDate           *string  `json:"ex_dividend_date"`
	PaymentDate              *string  `json:"payment_date"`
	DividendFrequency        *string  `json:"dividend_frequency"`

	// === QUALITY SCORES (6 metrics) ===
	AltmanZScore             *float64 `json:"altman_z_score"`
	AltmanZInterpretation    *string  `json:"altman_z_interpretation"`
	AltmanZDescription       *string  `json:"altman_z_description"`
	PiotroskiFScore          *int     `json:"piotroski_f_score"`
	PiotroskiFInterpretation *string  `json:"piotroski_f_interpretation"`
	PiotroskiFDescription    *string  `json:"piotroski_f_description"`

	// === FORWARD ESTIMATES (8 metrics) ===
	ForwardEPS         *float64 `json:"forward_eps"`
	ForwardEPSHigh     *float64 `json:"forward_eps_high"`
	ForwardEPSLow      *float64 `json:"forward_eps_low"`
	ForwardRevenue     *float64 `json:"forward_revenue"`
	ForwardEBITDA      *float64 `json:"forward_ebitda"`
	ForwardNetIncome   *float64 `json:"forward_net_income"`
	NumAnalystsEPS     *int     `json:"num_analysts_eps"`
	NumAnalystsRevenue *int     `json:"num_analysts_revenue"`

	// === ANALYST RATINGS (10 metrics) ===
	AnalystRatingStrongBuy  *int     `json:"analyst_rating_strong_buy"`
	AnalystRatingBuy        *int     `json:"analyst_rating_buy"`
	AnalystRatingHold       *int     `json:"analyst_rating_hold"`
	AnalystRatingSell       *int     `json:"analyst_rating_sell"`
	AnalystRatingStrongSell *int     `json:"analyst_rating_strong_sell"`
	AnalystConsensus        *string  `json:"analyst_consensus"`
	TargetHigh              *float64 `json:"target_high"`
	TargetLow               *float64 `json:"target_low"`
	TargetConsensus         *float64 `json:"target_consensus"`
	TargetMedian            *float64 `json:"target_median"`

	// === INTERPRETATIONS ===
	PEGInterpretation    *string `json:"peg_interpretation,omitempty"`
	PayoutInterpretation *string `json:"payout_interpretation,omitempty"`

	// === METADATA ===
	FMPAvailable bool          `json:"fmp_available"`
	Sources      *FieldSources `json:"sources,omitempty"`
}

// ============================================================================
// Merge Functions
// ============================================================================

// MergeAllData merges all FMP data with database data
func MergeAllData(fmp *FMPAllMetrics, currentPrice float64) *MergedFinancialMetrics {
	merged := &MergedFinancialMetrics{
		FMPAvailable: fmp != nil && fmp.RatiosTTM != nil,
		Sources:      &FieldSources{},
	}

	if fmp == nil {
		return merged
	}

	// Merge ratios-ttm data
	if fmp.RatiosTTM != nil {
		r := fmp.RatiosTTM

		// Valuation
		merged.PERatio = r.PriceToEarningsRatioTTM
		merged.PBRatio = r.PriceToBookRatioTTM
		merged.PSRatio = r.PriceToSalesRatioTTM
		merged.PriceToFCF = r.PriceToFreeCashFlowTTM
		merged.PriceToOCF = r.PriceToOperatingCFTTM
		merged.PEGRatio = r.PEGRatioTTM
		merged.EnterpriseValue = r.EnterpriseValueTTM
		merged.EVToSales = r.EVToSalesTTM
		merged.EVToEBITDA = r.EVToEBITDATTM
		merged.EVToEBIT = r.EVToEBITTTM
		merged.EVToFCF = r.EVToFCFTTM
		merged.EarningsYield = ConvertToPercentage(r.EarningsYieldTTM)
		merged.FCFYield = ConvertToPercentage(r.FCFYieldTTM)

		// Profitability (convert decimals to percentages)
		merged.GrossMargin = ConvertToPercentage(r.GrossProfitMarginTTM)
		merged.OperatingMargin = ConvertToPercentage(r.OperatingProfitMarginTTM)
		merged.NetMargin = ConvertToPercentage(r.NetProfitMarginTTM)
		merged.EBITDAMargin = ConvertToPercentage(r.EBITDAMarginTTM)
		merged.EBITMargin = ConvertToPercentage(r.EBITMarginTTM)
		merged.FCFMargin = ConvertToPercentage(r.FCFMarginTTM)
		merged.PretaxMargin = ConvertToPercentage(r.PretaxMarginTTM)
		merged.ROE = ConvertToPercentage(r.ReturnOnEquityTTM)
		merged.ROA = ConvertToPercentage(r.ReturnOnAssetsTTM)
		merged.ROIC = ConvertToPercentage(r.ReturnOnInvestedCapitalTTM)
		merged.ROCE = ConvertToPercentage(r.ReturnOnCapitalEmployedTTM)

		// Liquidity
		merged.CurrentRatio = r.CurrentRatioTTM
		merged.QuickRatio = r.QuickRatioTTM
		merged.CashRatio = r.CashRatioTTM

		// Leverage
		merged.DebtToEquity = r.DebtEquityRatioTTM
		merged.DebtToAssets = r.DebtToAssetsRatioTTM
		merged.DebtToEBITDA = r.DebtToEBITDATTM
		merged.DebtToCapital = r.DebtToCapitalTTM
		merged.InterestCoverage = r.InterestCoverageTTM
		merged.NetDebtToEBITDA = r.NetDebtToEBITDATTM

		// Efficiency
		merged.AssetTurnover = r.AssetTurnoverTTM
		merged.InventoryTurnover = r.InventoryTurnoverTTM
		merged.ReceivablesTurnover = r.ReceivablesTurnoverTTM
		merged.PayablesTurnover = r.PayablesTurnoverTTM
		merged.FixedAssetTurnover = r.FixedAssetTurnoverTTM
		merged.DaysOfSalesOutstanding = r.DaysOfSalesOutstandingTTM
		merged.DaysOfInventoryOutstanding = r.DaysOfInventoryOutstandingTTM
		merged.DaysOfPayablesOutstanding = r.DaysOfPayablesOutstandingTTM
		merged.CashConversionCycle = r.CashConversionCycleTTM

		// Dividends
		merged.DividendYield = ConvertToPercentage(r.DividendYieldTTM)
		merged.PayoutRatio = ConvertToPercentage(r.PayoutRatioTTM)
		merged.DividendPerShare = r.DividendPerShareTTM

		// Per Share
		merged.RevenuePerShare = r.RevenuePerShareTTM
		merged.BookValuePerShare = r.BookValuePerShareTTM
		merged.TangibleBookPerShare = r.TangibleBookPerShareTTM
		merged.OperatingCFPerShare = r.OperatingCFPerShareTTM
		merged.FCFPerShare = r.FreeCashFlowPerShareTTM
		merged.CashPerShare = r.CashPerShareTTM
		merged.GrahamNumber = r.GrahamNumberTTM
		merged.EPSDiluted = r.NetIncomePerShareTTM

		// PEG Interpretation
		if r.PEGRatioTTM != nil {
			interp, _ := GetPEGInterpretation(*r.PEGRatioTTM)
			merged.PEGInterpretation = &interp
		}

		// Payout Interpretation
		if r.PayoutRatioTTM != nil {
			interp, _ := GetPayoutRatioInterpretation(*r.PayoutRatioTTM * 100)
			merged.PayoutInterpretation = &interp
		}

		// Set sources for ratios-ttm fields
		merged.Sources.PERatio = SourceFMP
		merged.Sources.PBRatio = SourceFMP
		merged.Sources.PSRatio = SourceFMP
		merged.Sources.PriceToFCF = SourceFMP
		merged.Sources.PriceToOCF = SourceFMP
		merged.Sources.PEGRatio = SourceFMP
		merged.Sources.EVToSales = SourceFMP
		merged.Sources.EVToEBITDA = SourceFMP
		merged.Sources.EVToEBIT = SourceFMP
		merged.Sources.EVToFCF = SourceFMP
		merged.Sources.EarningsYield = SourceFMP
		merged.Sources.FCFYield = SourceFMP
		merged.Sources.GrossMargin = SourceFMP
		merged.Sources.OperatingMargin = SourceFMP
		merged.Sources.NetMargin = SourceFMP
		merged.Sources.EBITDAMargin = SourceFMP
		merged.Sources.EBITMargin = SourceFMP
		merged.Sources.FCFMargin = SourceFMP
		merged.Sources.ROE = SourceFMP
		merged.Sources.ROA = SourceFMP
		merged.Sources.ROIC = SourceFMP
		merged.Sources.ROCE = SourceFMP
		merged.Sources.CurrentRatio = SourceFMP
		merged.Sources.QuickRatio = SourceFMP
		merged.Sources.CashRatio = SourceFMP
		merged.Sources.DebtToEquity = SourceFMP
		merged.Sources.DebtToAssets = SourceFMP
		merged.Sources.DebtToEBITDA = SourceFMP
		merged.Sources.DebtToCapital = SourceFMP
		merged.Sources.InterestCoverage = SourceFMP
		merged.Sources.AssetTurnover = SourceFMP
		merged.Sources.InventoryTurnover = SourceFMP
		merged.Sources.ReceivablesTurnover = SourceFMP
		merged.Sources.PayablesTurnover = SourceFMP
		merged.Sources.DSO = SourceFMP
		merged.Sources.DIO = SourceFMP
		merged.Sources.DPO = SourceFMP
		merged.Sources.CashConversionCycle = SourceFMP
		merged.Sources.DividendYield = SourceFMP
		merged.Sources.PayoutRatio = SourceFMP
		merged.Sources.EPSDiluted = SourceFMP
	}

	// Merge key-metrics-ttm data
	if fmp.KeyMetricsTTM != nil {
		k := fmp.KeyMetricsTTM

		merged.MarketCap = k.MarketCapTTM
		if k.MarketCapTTM != nil {
			merged.Sources.MarketCap = SourceFMP
		}
		merged.WorkingCapital = k.WorkingCapitalTTM
		merged.NetDebt = k.NetDebtTTM
		merged.InvestedCapital = k.InvestedCapitalTTM

		// Fill in per-share metrics if not from ratios-ttm
		if merged.RevenuePerShare == nil {
			merged.RevenuePerShare = k.RevenuePerShareTTM
		}
		if merged.OperatingCFPerShare == nil {
			merged.OperatingCFPerShare = k.OperatingCashFlowPerShareTTM
		}
		if merged.FCFPerShare == nil {
			merged.FCFPerShare = k.FreeCashFlowPerShareTTM
		}
		if merged.CashPerShare == nil {
			merged.CashPerShare = k.CashPerShareTTM
		}
		if merged.BookValuePerShare == nil {
			merged.BookValuePerShare = k.BookValuePerShareTTM
		}
		if merged.TangibleBookPerShare == nil {
			merged.TangibleBookPerShare = k.TangibleBookValuePerShareTTM
		}
		if merged.GrahamNumber == nil {
			merged.GrahamNumber = k.GrahamNumberTTM
		}
		if merged.InterestDebtPerShare == nil {
			merged.InterestDebtPerShare = k.InterestDebtPerShareTTM
		}

		// Fallback for ROE from key-metrics-ttm (uses different field name: roeTTM)
		if merged.ROE == nil && k.ROETTM != nil {
			merged.ROE = ConvertToPercentage(k.ROETTM)
			merged.Sources.ROE = SourceFMP
		}
	}

	// Merge financial-growth data
	if len(fmp.Growth) > 0 {
		g := fmp.Growth[0] // Most recent year

		merged.RevenueGrowthYoY = ConvertToPercentage(g.RevenueGrowth)
		merged.GrossProfitGrowthYoY = ConvertToPercentage(g.GrossProfitGrowth)
		merged.OperatingIncomeGrowthYoY = ConvertToPercentage(g.OperatingIncomeGrowth)
		merged.NetIncomeGrowthYoY = ConvertToPercentage(g.NetIncomeGrowth)
		merged.EPSGrowthYoY = ConvertToPercentage(g.EPSGrowth)
		merged.FCFGrowthYoY = ConvertToPercentage(g.FreeCashFlowGrowth)
		merged.BookValueGrowthYoY = ConvertToPercentage(g.BookValuePerShareGrowth)

		// FMP provides pre-calculated CAGRs
		merged.RevenueGrowth3YCAGR = ConvertToPercentage(g.ThreeYRevenueGrowthPerShare)
		merged.RevenueGrowth5YCAGR = ConvertToPercentage(g.FiveYRevenueGrowthPerShare)
		merged.EPSGrowth3YCAGR = ConvertToPercentage(g.ThreeYNetIncomeGrowthPerShare)
		merged.EPSGrowth5YCAGR = ConvertToPercentage(g.FiveYNetIncomeGrowthPerShare)
		merged.DividendGrowth5YCAGR = ConvertToPercentage(g.FiveYDividendPerShareGrowthPerShare)

		merged.Sources.RevenueGrowthYoY = SourceFMP
		merged.Sources.EPSGrowthYoY = SourceFMP
		merged.Sources.RevenueGrowth3Y = SourceFMP
		merged.Sources.RevenueGrowth5Y = SourceFMP
		merged.Sources.EPSGrowth5Y = SourceFMP
	}

	// Merge analyst-estimates data
	if len(fmp.Estimates) > 0 {
		e := fmp.Estimates[0] // Next period estimate

		merged.ForwardEPS = e.EstimatedEPSAvg
		merged.ForwardEPSHigh = e.EstimatedEPSHigh
		merged.ForwardEPSLow = e.EstimatedEPSLow
		merged.ForwardRevenue = e.EstimatedRevenueAvg
		merged.ForwardEBITDA = e.EstimatedEBITDAAvg
		merged.ForwardNetIncome = e.EstimatedNetIncomeAvg
		merged.NumAnalystsEPS = e.NumberAnalystsEstimatedEPS
		merged.NumAnalystsRevenue = e.NumberAnalystsEstimatedRevenue

		// Calculate Forward P/E
		if e.EstimatedEPSAvg != nil && *e.EstimatedEPSAvg > 0 && currentPrice > 0 {
			forwardPE := currentPrice / *e.EstimatedEPSAvg
			merged.ForwardPE = &forwardPE
			merged.Sources.ForwardPE = SourceCalculated
		}
	}

	// Merge score data
	if fmp.Score != nil {
		s := fmp.Score

		merged.AltmanZScore = s.AltmanZScore
		merged.PiotroskiFScore = s.PiotroskiScore

		if s.AltmanZScore != nil {
			interp, desc := GetZScoreInterpretation(*s.AltmanZScore)
			merged.AltmanZInterpretation = &interp
			merged.AltmanZDescription = &desc
		}

		if s.PiotroskiScore != nil {
			interp, desc := GetFScoreInterpretation(*s.PiotroskiScore)
			merged.PiotroskiFInterpretation = &interp
			merged.PiotroskiFDescription = &desc
		}

		merged.Sources.AltmanZScore = SourceFMP
		merged.Sources.PiotroskiFScore = SourceFMP
	}

	// Merge dividend history data
	if len(fmp.Dividends) > 0 {
		d := fmp.Dividends[0] // Most recent dividend

		merged.ExDividendDate = &d.Date
		if d.PaymentDate != "" {
			merged.PaymentDate = &d.PaymentDate
		}

		// Calculate consecutive dividend years
		if len(fmp.Dividends) >= 4 {
			years := countConsecutiveDividendYears(fmp.Dividends)
			merged.ConsecutiveDividendYears = &years
		}

		// Estimate dividend frequency
		freq := estimateDividendFrequency(fmp.Dividends)
		merged.DividendFrequency = &freq
	}

	// Calculate ForwardDividendYield: (Dividend Per Share / Current Price) * 100
	if merged.DividendPerShare != nil && currentPrice > 0 {
		forwardYield := (*merged.DividendPerShare / currentPrice) * 100
		merged.ForwardDividendYield = &forwardYield
		merged.Sources.ForwardDividendYield = SourceCalculated
	}

	// Calculate FCF Payout Ratio: (Dividend Per Share / FCF Per Share) * 100
	if merged.DividendPerShare != nil && merged.FCFPerShare != nil && *merged.FCFPerShare > 0 {
		fcfPayout := (*merged.DividendPerShare / *merged.FCFPerShare) * 100
		merged.FCFPayoutRatio = &fcfPayout
		merged.Sources.FCFPayoutRatio = SourceCalculated
	}

	// Calculate FCF Margin fallback: (FCF Per Share / Revenue Per Share) * 100
	// This is equivalent to FCF/Revenue since per-share values cancel out shares outstanding
	if merged.FCFMargin == nil && merged.FCFPerShare != nil && merged.RevenuePerShare != nil && *merged.RevenuePerShare > 0 {
		fcfMargin := (*merged.FCFPerShare / *merged.RevenuePerShare) * 100
		merged.FCFMargin = &fcfMargin
		merged.Sources.FCFMargin = SourceCalculated
	}

	// Merge analyst grades summary data
	if fmp.GradesSummary != nil {
		g := fmp.GradesSummary

		merged.AnalystRatingStrongBuy = &g.StrongBuy
		merged.AnalystRatingBuy = &g.Buy
		merged.AnalystRatingHold = &g.Hold
		merged.AnalystRatingSell = &g.Sell
		merged.AnalystRatingStrongSell = &g.StrongSell
		if g.Consensus != "" {
			merged.AnalystConsensus = &g.Consensus
		}
	}

	// Merge price target consensus data
	if fmp.PriceTargetConsensus != nil {
		p := fmp.PriceTargetConsensus

		merged.TargetHigh = p.TargetHigh
		merged.TargetLow = p.TargetLow
		merged.TargetConsensus = p.TargetConsensus
		merged.TargetMedian = p.TargetMedian
	}

	return merged
}

// countConsecutiveDividendYears counts years of consecutive dividend payments
func countConsecutiveDividendYears(dividends []FMPDividendHistorical) int {
	if len(dividends) == 0 {
		return 0
	}

	years := make(map[string]bool)
	for _, d := range dividends {
		if len(d.Date) >= 4 {
			years[d.Date[:4]] = true
		}
	}
	return len(years)
}

// estimateDividendFrequency estimates dividend payment frequency
func estimateDividendFrequency(dividends []FMPDividendHistorical) string {
	if len(dividends) < 2 {
		return "unknown"
	}

	// Count dividends in the last year
	count := 0
	now := time.Now()
	oneYearAgo := now.AddDate(-1, 0, 0)

	for _, d := range dividends {
		t, err := time.Parse("2006-01-02", d.Date)
		if err != nil {
			continue
		}
		if t.After(oneYearAgo) {
			count++
		}
	}

	switch {
	case count >= 12:
		return "monthly"
	case count >= 4:
		return "quarterly"
	case count >= 2:
		return "semi-annual"
	case count >= 1:
		return "annual"
	default:
		return "irregular"
	}
}

// ============================================================================
// Legacy Merge Function (for backward compatibility)
// ============================================================================

// MergeWithDBData merges FMP data with database data, preferring FMP when available
// This is maintained for backward compatibility
func MergeWithDBData(
	fmp *FMPRatiosTTM,
	dbGrossMargin, dbOperatingMargin, dbNetMargin *float64,
	dbROE, dbROA *float64,
	dbDebtToEquity, dbCurrentRatio, dbQuickRatio *float64,
	dbPERatio, dbPBRatio, dbPSRatio *float64,
) *MergedFinancialMetrics {
	merged := &MergedFinancialMetrics{
		FMPAvailable: fmp != nil,
		Sources:      &FieldSources{},
	}

	if fmp != nil {
		// Valuation ratios from FMP (already in correct format)
		merged.PERatio, merged.Sources.PERatio = coalesceWithSource(fmp.PriceToEarningsRatioTTM, dbPERatio)
		merged.PBRatio, merged.Sources.PBRatio = coalesceWithSource(fmp.PriceToBookRatioTTM, dbPBRatio)
		merged.PSRatio, merged.Sources.PSRatio = coalesceWithSource(fmp.PriceToSalesRatioTTM, dbPSRatio)

		// Profitability - FMP stores as decimals (0.47), convert to % (47.0)
		merged.GrossMargin, merged.Sources.GrossMargin = coalesceWithSource(ConvertToPercentage(fmp.GrossProfitMarginTTM), dbGrossMargin)
		merged.OperatingMargin, merged.Sources.OperatingMargin = coalesceWithSource(ConvertToPercentage(fmp.OperatingProfitMarginTTM), dbOperatingMargin)
		merged.NetMargin, merged.Sources.NetMargin = coalesceWithSource(ConvertToPercentage(fmp.NetProfitMarginTTM), dbNetMargin)

		// ROE/ROA - FMP often null, fallback to DB
		merged.ROE, merged.Sources.ROE = coalesceWithSource(ConvertToPercentage(fmp.ReturnOnEquityTTM), dbROE)
		merged.ROA, merged.Sources.ROA = coalesceWithSource(ConvertToPercentage(fmp.ReturnOnAssetsTTM), dbROA)

		// Financial health
		merged.CurrentRatio, merged.Sources.CurrentRatio = coalesceWithSource(fmp.CurrentRatioTTM, dbCurrentRatio)
		merged.QuickRatio, merged.Sources.QuickRatio = coalesceWithSource(fmp.QuickRatioTTM, dbQuickRatio)
		merged.DebtToEquity, merged.Sources.DebtToEquity = coalesceWithSource(fmp.DebtEquityRatioTTM, dbDebtToEquity)

		// NEW: Additional fields from expanded ratios-ttm
		merged.EBITDAMargin = ConvertToPercentage(fmp.EBITDAMarginTTM)
		merged.EBITMargin = ConvertToPercentage(fmp.EBITMarginTTM)
		merged.FCFMargin = ConvertToPercentage(fmp.FCFMarginTTM)
		merged.PretaxMargin = ConvertToPercentage(fmp.PretaxMarginTTM)
		merged.ROIC = ConvertToPercentage(fmp.ReturnOnInvestedCapitalTTM)
		merged.ROCE = ConvertToPercentage(fmp.ReturnOnCapitalEmployedTTM)
		merged.CashRatio = fmp.CashRatioTTM
		merged.DebtToAssets = fmp.DebtToAssetsRatioTTM
		merged.DebtToEBITDA = fmp.DebtToEBITDATTM
		merged.DebtToCapital = fmp.DebtToCapitalTTM
		merged.InterestCoverage = fmp.InterestCoverageTTM
		merged.PriceToFCF = fmp.PriceToFreeCashFlowTTM
		merged.PriceToOCF = fmp.PriceToOperatingCFTTM
		merged.PEGRatio = fmp.PEGRatioTTM
		merged.EnterpriseValue = fmp.EnterpriseValueTTM
		merged.EVToSales = fmp.EVToSalesTTM
		merged.EVToEBITDA = fmp.EVToEBITDATTM
		merged.EVToEBIT = fmp.EVToEBITTTM
		merged.EVToFCF = fmp.EVToFCFTTM
		merged.EarningsYield = ConvertToPercentage(fmp.EarningsYieldTTM)
		merged.FCFYield = ConvertToPercentage(fmp.FCFYieldTTM)
		merged.AssetTurnover = fmp.AssetTurnoverTTM
		merged.InventoryTurnover = fmp.InventoryTurnoverTTM
		merged.ReceivablesTurnover = fmp.ReceivablesTurnoverTTM
		merged.PayablesTurnover = fmp.PayablesTurnoverTTM
		merged.FixedAssetTurnover = fmp.FixedAssetTurnoverTTM
		merged.DaysOfSalesOutstanding = fmp.DaysOfSalesOutstandingTTM
		merged.DaysOfInventoryOutstanding = fmp.DaysOfInventoryOutstandingTTM
		merged.DaysOfPayablesOutstanding = fmp.DaysOfPayablesOutstandingTTM
		merged.CashConversionCycle = fmp.CashConversionCycleTTM
		merged.DividendYield = ConvertToPercentage(fmp.DividendYieldTTM)
		merged.PayoutRatio = ConvertToPercentage(fmp.PayoutRatioTTM)
		merged.DividendPerShare = fmp.DividendPerShareTTM
		merged.RevenuePerShare = fmp.RevenuePerShareTTM
		merged.BookValuePerShare = fmp.BookValuePerShareTTM
		merged.TangibleBookPerShare = fmp.TangibleBookPerShareTTM
		merged.OperatingCFPerShare = fmp.OperatingCFPerShareTTM
		merged.FCFPerShare = fmp.FreeCashFlowPerShareTTM
		merged.CashPerShare = fmp.CashPerShareTTM
		merged.GrahamNumber = fmp.GrahamNumberTTM

		// Set sources for new fields
		if fmp.EBITDAMarginTTM != nil {
			merged.Sources.EBITDAMargin = SourceFMP
		}
		if fmp.EBITMarginTTM != nil {
			merged.Sources.EBITMargin = SourceFMP
		}
		if fmp.FCFMarginTTM != nil {
			merged.Sources.FCFMargin = SourceFMP
		}
		if fmp.ReturnOnInvestedCapitalTTM != nil {
			merged.Sources.ROIC = SourceFMP
		}
		if fmp.ReturnOnCapitalEmployedTTM != nil {
			merged.Sources.ROCE = SourceFMP
		}

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

		// All sources are database when FMP unavailable
		merged.Sources.PERatio = sourceFor(dbPERatio, SourceDatabase)
		merged.Sources.PBRatio = sourceFor(dbPBRatio, SourceDatabase)
		merged.Sources.PSRatio = sourceFor(dbPSRatio, SourceDatabase)
		merged.Sources.GrossMargin = sourceFor(dbGrossMargin, SourceDatabase)
		merged.Sources.OperatingMargin = sourceFor(dbOperatingMargin, SourceDatabase)
		merged.Sources.NetMargin = sourceFor(dbNetMargin, SourceDatabase)
		merged.Sources.ROE = sourceFor(dbROE, SourceDatabase)
		merged.Sources.ROA = sourceFor(dbROA, SourceDatabase)
		merged.Sources.CurrentRatio = sourceFor(dbCurrentRatio, SourceDatabase)
		merged.Sources.QuickRatio = sourceFor(dbQuickRatio, SourceDatabase)
		merged.Sources.DebtToEquity = sourceFor(dbDebtToEquity, SourceDatabase)
	}

	return merged
}

// sourceFor returns the source if the value is non-nil, otherwise empty
func sourceFor(val *float64, source DataSource) DataSource {
	if val != nil {
		return source
	}
	return SourceNone
}

// coalesceWithSource returns the first non-nil value and its source (FMP or database)
func coalesceWithSource(fmpVal, dbVal *float64) (*float64, DataSource) {
	if fmpVal != nil {
		return fmpVal, SourceFMP
	}
	if dbVal != nil {
		return dbVal, SourceDatabase
	}
	return nil, SourceNone
}
