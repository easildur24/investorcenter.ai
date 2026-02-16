package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Stock represents basic stock information with multi-asset support
type Stock struct {
	ID          int              `json:"id" db:"id"`
	Symbol      string           `json:"symbol" db:"symbol"`
	Name        string           `json:"name" db:"name"`
	Exchange    string           `json:"exchange" db:"exchange"`
	Sector      string           `json:"sector" db:"sector"`
	Industry    string           `json:"industry" db:"industry"`
	Country     string           `json:"country" db:"country"`
	Currency    string           `json:"currency" db:"currency"`
	MarketCap   *decimal.Decimal `json:"marketCap" db:"market_cap"`
	Description string           `json:"description" db:"description"`
	Website     string           `json:"website" db:"website"`
	CreatedAt   time.Time        `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time        `json:"updatedAt" db:"updated_at"`

	// New Polygon fields for multi-asset support
	AssetType           string `json:"assetType,omitempty" db:"asset_type"`
	Locale              string `json:"locale,omitempty" db:"locale"`
	Market              string `json:"market,omitempty" db:"market"`
	Active              bool   `json:"active" db:"active"`
	CurrencyName        string `json:"currencyName,omitempty" db:"currency_name"`
	CIK                 string `json:"cik,omitempty" db:"cik"`
	CompositeFIGI       string `json:"compositeFigi,omitempty" db:"composite_figi"`
	ShareClassFIGI      string `json:"shareClassFigi,omitempty" db:"share_class_figi"`
	PrimaryExchangeCode string `json:"primaryExchangeCode,omitempty" db:"primary_exchange_code"`
	PolygonType         string `json:"polygonType,omitempty" db:"polygon_type"`
	LastUpdatedUTC      string `json:"lastUpdatedUtc,omitempty" db:"last_updated_utc"`
	LogoURL             string `json:"logoUrl,omitempty" db:"logo_url"`
}

// StockPrice represents current and historical price data
type StockPrice struct {
	Symbol        string          `json:"symbol" db:"symbol"`
	Price         decimal.Decimal `json:"price" db:"price"`
	Open          decimal.Decimal `json:"open" db:"open"`
	High          decimal.Decimal `json:"high" db:"high"`
	Low           decimal.Decimal `json:"low" db:"low"`
	Close         decimal.Decimal `json:"close" db:"close"`
	Volume        int64           `json:"volume" db:"volume"`
	Change        decimal.Decimal `json:"change" db:"change"`
	ChangePercent decimal.Decimal `json:"changePercent" db:"change_percent"`
	Timestamp     time.Time       `json:"timestamp" db:"timestamp"`
}

// ChartDataPoint represents a single data point for charts
type ChartDataPoint struct {
	Timestamp time.Time       `json:"timestamp" db:"timestamp"`
	Open      decimal.Decimal `json:"open" db:"open"`
	High      decimal.Decimal `json:"high" db:"high"`
	Low       decimal.Decimal `json:"low" db:"low"`
	Close     decimal.Decimal `json:"close" db:"close"`
	Volume    int64           `json:"volume" db:"volume"`
}

// ChartData represents chart data with multiple data points
type ChartData struct {
	Symbol      string           `json:"symbol"`
	Period      string           `json:"period"`
	DataPoints  []ChartDataPoint `json:"dataPoints"`
	LastUpdated time.Time        `json:"lastUpdated"`
}

// Fundamentals represents financial fundamentals data
type Fundamentals struct {
	Symbol            string           `json:"symbol" db:"symbol"`
	Period            string           `json:"period" db:"period"`
	Year              int              `json:"year" db:"year"`
	PE                *decimal.Decimal `json:"pe" db:"pe"`
	PB                *decimal.Decimal `json:"pb" db:"pb"`
	PS                *decimal.Decimal `json:"ps" db:"ps"`
	Revenue           *decimal.Decimal `json:"revenue" db:"revenue"`
	GrossProfit       *decimal.Decimal `json:"grossProfit" db:"gross_profit"`
	OperatingIncome   *decimal.Decimal `json:"operatingIncome" db:"operating_income"`
	NetIncome         *decimal.Decimal `json:"netIncome" db:"net_income"`
	EPS               *decimal.Decimal `json:"eps" db:"eps"`
	EPSDiluted        *decimal.Decimal `json:"epsDiluted" db:"eps_diluted"`
	GrossMargin       *decimal.Decimal `json:"grossMargin" db:"gross_margin"`
	OperatingMargin   *decimal.Decimal `json:"operatingMargin" db:"operating_margin"`
	NetMargin         *decimal.Decimal `json:"netMargin" db:"net_margin"`
	ROE               *decimal.Decimal `json:"roe" db:"roe"`
	ROA               *decimal.Decimal `json:"roa" db:"roa"`
	TotalAssets       *decimal.Decimal `json:"totalAssets" db:"total_assets"`
	TotalLiabilities  *decimal.Decimal `json:"totalLiabilities" db:"total_liabilities"`
	TotalEquity       *decimal.Decimal `json:"totalEquity" db:"total_equity"`
	TotalDebt         *decimal.Decimal `json:"totalDebt" db:"total_debt"`
	DebtToEquity      *decimal.Decimal `json:"debtToEquity" db:"debt_to_equity"`
	CurrentRatio      *decimal.Decimal `json:"currentRatio" db:"current_ratio"`
	QuickRatio        *decimal.Decimal `json:"quickRatio" db:"quick_ratio"`
	OperatingCashFlow *decimal.Decimal `json:"operatingCashFlow" db:"operating_cash_flow"`
	UpdatedAt         time.Time        `json:"updatedAt" db:"updated_at"`
}

// NewsArticle represents a news article
type NewsArticle struct {
	ID          int       `json:"id" db:"id"`
	Symbol      string    `json:"symbol" db:"symbol"`
	Title       string    `json:"title" db:"title"`
	Summary     string    `json:"summary" db:"summary"`
	Author      string    `json:"author" db:"author"`
	Source      string    `json:"source" db:"source"`
	URL         string    `json:"url" db:"url"`
	Sentiment   string    `json:"sentiment" db:"sentiment"`
	PublishedAt time.Time `json:"publishedAt" db:"published_at"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
}

// Earnings represents earnings data
type Earnings struct {
	Symbol             string           `json:"symbol" db:"symbol"`
	Quarter            string           `json:"quarter" db:"quarter"`
	Year               int              `json:"year" db:"year"`
	ReportDate         time.Time        `json:"reportDate" db:"report_date"`
	EPSActual          *decimal.Decimal `json:"epsActual" db:"eps_actual"`
	EPSEstimate        *decimal.Decimal `json:"epsEstimate" db:"eps_estimate"`
	EPSSurprise        *decimal.Decimal `json:"epsSurprise" db:"eps_surprise"`
	EPSSurprisePercent *decimal.Decimal `json:"epsSurprisePercent" db:"eps_surprise_percent"`
	RevenueActual      *decimal.Decimal `json:"revenueActual" db:"revenue_actual"`
	RevenueEstimate    *decimal.Decimal `json:"revenueEstimate" db:"revenue_estimate"`
	CreatedAt          time.Time        `json:"createdAt" db:"created_at"`
}

// Dividend represents dividend data
type Dividend struct {
	Symbol       string           `json:"symbol" db:"symbol"`
	ExDate       time.Time        `json:"exDate" db:"ex_date"`
	PayDate      time.Time        `json:"payDate" db:"pay_date"`
	Amount       decimal.Decimal  `json:"amount" db:"amount"`
	Frequency    string           `json:"frequency" db:"frequency"`
	Type         string           `json:"type" db:"type"`
	YieldPercent *decimal.Decimal `json:"yieldPercent" db:"yield_percent"`
	CreatedAt    time.Time        `json:"createdAt" db:"created_at"`
}

// AnalystRating represents analyst rating data
type AnalystRating struct {
	Symbol      string           `json:"symbol" db:"symbol"`
	Firm        string           `json:"firm" db:"firm"`
	Analyst     string           `json:"analyst" db:"analyst"`
	Rating      string           `json:"rating" db:"rating"`
	PriceTarget *decimal.Decimal `json:"priceTarget" db:"price_target"`
	RatingDate  time.Time        `json:"ratingDate" db:"rating_date"`
	CreatedAt   time.Time        `json:"createdAt" db:"created_at"`
}

// InsiderTrading represents insider trading data
type InsiderTrading struct {
	Symbol          string          `json:"symbol" db:"symbol"`
	InsiderName     string          `json:"insiderName" db:"insider_name"`
	Title           string          `json:"title" db:"title"`
	TransactionType string          `json:"transactionType" db:"transaction_type"`
	Shares          int64           `json:"shares" db:"shares"`
	Price           decimal.Decimal `json:"price" db:"price"`
	Value           decimal.Decimal `json:"value" db:"value"`
	SharesOwned     *int64          `json:"sharesOwned" db:"shares_owned"`
	TransactionDate time.Time       `json:"transactionDate" db:"transaction_date"`
	FilingDate      time.Time       `json:"filingDate" db:"filing_date"`
	CreatedAt       time.Time       `json:"createdAt" db:"created_at"`
}

// PeerComparison represents peer comparison data
type PeerComparison struct {
	Symbol        string           `json:"symbol" db:"symbol"`
	Name          string           `json:"name" db:"name"`
	Price         decimal.Decimal  `json:"price" db:"price"`
	MarketCap     *decimal.Decimal `json:"marketCap" db:"market_cap"`
	PE            *decimal.Decimal `json:"pe" db:"pe"`
	PB            *decimal.Decimal `json:"pb" db:"pb"`
	PS            *decimal.Decimal `json:"ps" db:"ps"`
	ROE           *decimal.Decimal `json:"roe" db:"roe"`
	DebtToEquity  *decimal.Decimal `json:"debtToEquity" db:"debt_to_equity"`
	DividendYield *decimal.Decimal `json:"dividendYield" db:"dividend_yield"`
	Revenue       *decimal.Decimal `json:"revenue" db:"revenue"`
	NetIncome     *decimal.Decimal `json:"netIncome" db:"net_income"`
}

// TickerPageData represents the complete data for a ticker page
type TickerPageData struct {
	Summary             StockSummary        `json:"summary"`
	ChartData           ChartData           `json:"chartData"`
	TechnicalIndicators TechnicalIndicators `json:"technicalIndicators"`
	AnalystConsensus    AnalystConsensus    `json:"analystConsensus"`
	KeyMetrics          KeyMetrics          `json:"keyMetrics"`
	News                []NewsArticle       `json:"news"`
	Earnings            []Earnings          `json:"earnings"`
	Dividends           []Dividend          `json:"dividends"`
	AnalystRatings      []AnalystRating     `json:"analystRatings"`
	InsiderActivity     []InsiderTrading    `json:"insiderActivity"`
	PeerComparisons     []PeerComparison    `json:"peerComparisons"`
}

// StockSummary represents summary data for a stock
type StockSummary struct {
	Stock        Stock         `json:"stock"`
	Price        StockPrice    `json:"price"`
	Fundamentals *Fundamentals `json:"fundamentals,omitempty"`
}

// TechnicalIndicators represents technical analysis indicators
type TechnicalIndicators struct {
	Symbol          string           `json:"symbol"`
	RSI             *decimal.Decimal `json:"rsi"`
	MACD            *decimal.Decimal `json:"macd"`
	MACDSignal      *decimal.Decimal `json:"macdSignal"`
	MACDHistogram   *decimal.Decimal `json:"macdHistogram"`
	BollingerUpper  *decimal.Decimal `json:"bollingerUpper"`
	BollingerMiddle *decimal.Decimal `json:"bollingerMiddle"`
	BollingerLower  *decimal.Decimal `json:"bollingerLower"`
	SMA20           *decimal.Decimal `json:"sma20"`
	SMA50           *decimal.Decimal `json:"sma50"`
	SMA200          *decimal.Decimal `json:"sma200"`
	Beta            *decimal.Decimal `json:"beta"`
	Volatility      *decimal.Decimal `json:"volatility"`
	Volume          int64            `json:"volume"`
	Timestamp       time.Time        `json:"timestamp"`
	LastUpdated     time.Time        `json:"lastUpdated"`
}

// AnalystConsensus represents analyst consensus data
type AnalystConsensus struct {
	Symbol          string           `json:"symbol"`
	ConsensusRating string           `json:"consensusRating"`
	PriceTarget     *decimal.Decimal `json:"priceTarget"`
	PriceTargetHigh *decimal.Decimal `json:"priceTargetHigh"`
	PriceTargetLow  *decimal.Decimal `json:"priceTargetLow"`
	PriceTargetMean *decimal.Decimal `json:"priceTargetMean"`
	Recommendations map[string]int   `json:"recommendations"`
	LastUpdated     time.Time        `json:"lastUpdated"`
}

// KeyMetrics represents key financial metrics
type KeyMetrics struct {
	Symbol          string           `json:"symbol"`
	MarketCap       *decimal.Decimal `json:"marketCap"`
	EnterpriseValue *decimal.Decimal `json:"enterpriseValue"`
	TrailingPE      *decimal.Decimal `json:"trailingPE"`
	ForwardPE       *decimal.Decimal `json:"forwardPE"`
	PEGRatio        *decimal.Decimal `json:"pegRatio"`
	PriceToSales    *decimal.Decimal `json:"priceToSales"`
	PriceToBook     *decimal.Decimal `json:"priceToBook"`
	EVToRevenue     *decimal.Decimal `json:"evToRevenue"`
	EVToEBITDA      *decimal.Decimal `json:"evToEbitda"`
	Beta            *decimal.Decimal `json:"beta"`
	Week52High      *decimal.Decimal `json:"week52High"`
	Week52Low       *decimal.Decimal `json:"week52Low"`
	DividendYield   *decimal.Decimal `json:"dividendYield"`
	DividendRate    *decimal.Decimal `json:"dividendRate"`
	PayoutRatio     *decimal.Decimal `json:"payoutRatio"`
	LastUpdated     time.Time        `json:"lastUpdated"`
}

// ScreenerStock represents a stock with screening metrics
type ScreenerStock struct {
	// Core identity
	Symbol   string  `json:"symbol" db:"symbol"`
	Name     string  `json:"name" db:"name"`
	Sector   *string `json:"sector" db:"sector"`
	Industry *string `json:"industry" db:"industry"`

	// Market data
	MarketCap *float64 `json:"market_cap" db:"market_cap"`
	Price     *float64 `json:"price" db:"price"`

	// Valuation
	PERatio *float64 `json:"pe_ratio" db:"pe_ratio"`
	PBRatio *float64 `json:"pb_ratio" db:"pb_ratio"`
	PSRatio *float64 `json:"ps_ratio" db:"ps_ratio"`

	// Profitability
	ROE             *float64 `json:"roe" db:"roe"`
	ROA             *float64 `json:"roa" db:"roa"`
	GrossMargin     *float64 `json:"gross_margin" db:"gross_margin"`
	OperatingMargin *float64 `json:"operating_margin" db:"operating_margin"`
	NetMargin       *float64 `json:"net_margin" db:"net_margin"`

	// Financial health
	DebtToEquity *float64 `json:"debt_to_equity" db:"debt_to_equity"`
	CurrentRatio *float64 `json:"current_ratio" db:"current_ratio"`

	// Growth
	RevenueGrowth *float64 `json:"revenue_growth" db:"revenue_growth"`
	EPSGrowthYoY  *float64 `json:"eps_growth_yoy" db:"eps_growth_yoy"`

	// Dividends
	DividendYield            *float64 `json:"dividend_yield" db:"dividend_yield"`
	PayoutRatio              *float64 `json:"payout_ratio" db:"payout_ratio"`
	ConsecutiveDividendYears *int     `json:"consecutive_dividend_years" db:"consecutive_dividend_years"`

	// Risk
	Beta *float64 `json:"beta" db:"beta"`

	// Fair value
	DCFUpsidePercent *float64 `json:"dcf_upside_percent" db:"dcf_upside_percent"`

	// IC Score: overall
	ICScore  *float64 `json:"ic_score" db:"ic_score"`
	ICRating *string  `json:"ic_rating" db:"ic_rating"`

	// IC Score: sub-factor scores (0-100)
	ValueScore            *float64 `json:"value_score" db:"value_score"`
	GrowthScore           *float64 `json:"growth_score" db:"growth_score"`
	ProfitabilityScore    *float64 `json:"profitability_score" db:"profitability_score"`
	FinancialHealthScore  *float64 `json:"financial_health_score" db:"financial_health_score"`
	MomentumScore         *float64 `json:"momentum_score" db:"momentum_score"`
	AnalystConsensusScore *float64 `json:"analyst_consensus_score" db:"analyst_consensus_score"`
	InsiderActivityScore  *float64 `json:"insider_activity_score" db:"insider_activity_score"`
	InstitutionalScore    *float64 `json:"institutional_score" db:"institutional_score"`
	NewsSentimentScore    *float64 `json:"news_sentiment_score" db:"news_sentiment_score"`
	TechnicalScore        *float64 `json:"technical_score" db:"technical_score"`

	// IC Score: metadata
	ICSectorPercentile *float64 `json:"ic_sector_percentile" db:"ic_sector_percentile"`
	LifecycleStage     *string  `json:"lifecycle_stage" db:"lifecycle_stage"`
}

// ScreenerParams represents query parameters for the screener endpoint
type ScreenerParams struct {
	// Pagination & sorting
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
	Sort  string `json:"sort"`
	Order string `json:"order"`

	// Categorical filters
	Sectors    []string `json:"sectors"`
	Industries []string `json:"industries"`
	AssetType  string   `json:"asset_type"`

	// Market data
	MarketCapMin *float64 `json:"market_cap_min"`
	MarketCapMax *float64 `json:"market_cap_max"`

	// Valuation
	PEMin *float64 `json:"pe_min"`
	PEMax *float64 `json:"pe_max"`
	PBMin *float64 `json:"pb_min"`
	PBMax *float64 `json:"pb_max"`
	PSMin *float64 `json:"ps_min"`
	PSMax *float64 `json:"ps_max"`

	// Profitability
	ROEMin         *float64 `json:"roe_min"`
	ROEMax         *float64 `json:"roe_max"`
	ROAMin         *float64 `json:"roa_min"`
	ROAMax         *float64 `json:"roa_max"`
	GrossMarginMin *float64 `json:"gross_margin_min"`
	GrossMarginMax *float64 `json:"gross_margin_max"`
	NetMarginMin   *float64 `json:"net_margin_min"`
	NetMarginMax   *float64 `json:"net_margin_max"`

	// Financial health
	DebtToEquityMin *float64 `json:"de_min"`
	DebtToEquityMax *float64 `json:"de_max"`
	CurrentRatioMin *float64 `json:"current_ratio_min"`
	CurrentRatioMax *float64 `json:"current_ratio_max"`

	// Growth
	RevenueGrowthMin *float64 `json:"revenue_growth_min"`
	RevenueGrowthMax *float64 `json:"revenue_growth_max"`
	EPSGrowthMin     *float64 `json:"eps_growth_min"`
	EPSGrowthMax     *float64 `json:"eps_growth_max"`

	// Dividends
	DividendYieldMin       *float64 `json:"dividend_yield_min"`
	DividendYieldMax       *float64 `json:"dividend_yield_max"`
	PayoutRatioMin         *float64 `json:"payout_ratio_min"`
	PayoutRatioMax         *float64 `json:"payout_ratio_max"`
	ConsecutiveDivYearsMin *float64 `json:"consec_div_years_min"`

	// Risk
	BetaMin *float64 `json:"beta_min"`
	BetaMax *float64 `json:"beta_max"`

	// Fair value
	DCFUpsideMin *float64 `json:"dcf_upside_min"`
	DCFUpsideMax *float64 `json:"dcf_upside_max"`

	// IC Score
	ICScoreMin *float64 `json:"ic_score_min"`
	ICScoreMax *float64 `json:"ic_score_max"`

	// IC Score sub-factors
	ValueScoreMin           *float64 `json:"value_score_min"`
	ValueScoreMax           *float64 `json:"value_score_max"`
	GrowthScoreMin          *float64 `json:"growth_score_min"`
	GrowthScoreMax          *float64 `json:"growth_score_max"`
	ProfitabilityScoreMin   *float64 `json:"profitability_score_min"`
	ProfitabilityScoreMax   *float64 `json:"profitability_score_max"`
	FinancialHealthScoreMin *float64 `json:"financial_health_score_min"`
	FinancialHealthScoreMax *float64 `json:"financial_health_score_max"`
	MomentumScoreMin        *float64 `json:"momentum_score_min"`
	MomentumScoreMax        *float64 `json:"momentum_score_max"`
	AnalystScoreMin         *float64 `json:"analyst_score_min"`
	AnalystScoreMax         *float64 `json:"analyst_score_max"`
	InsiderScoreMin         *float64 `json:"insider_score_min"`
	InsiderScoreMax         *float64 `json:"insider_score_max"`
	InstitutionalScoreMin   *float64 `json:"institutional_score_min"`
	InstitutionalScoreMax   *float64 `json:"institutional_score_max"`
	SentimentScoreMin       *float64 `json:"sentiment_score_min"`
	SentimentScoreMax       *float64 `json:"sentiment_score_max"`
	TechnicalScoreMin       *float64 `json:"technical_score_min"`
	TechnicalScoreMax       *float64 `json:"technical_score_max"`
}

// ScreenerResponse represents the paginated response for the screener endpoint
type ScreenerResponse struct {
	Data []ScreenerStock `json:"data"`
	Meta ScreenerMeta    `json:"meta"`
}

// ScreenerMeta represents pagination metadata
type ScreenerMeta struct {
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
	TotalPages int    `json:"total_pages"`
	Timestamp  string `json:"timestamp"`
}

// Helper functions
func DecimalFromFloat(f float64) decimal.Decimal {
	return decimal.NewFromFloat(f)
}

func DecimalPtr(f float64) *decimal.Decimal {
	d := decimal.NewFromFloat(f)
	return &d
}

func Int64Ptr(i int64) *int64 {
	return &i
}
