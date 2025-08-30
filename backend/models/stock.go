package models

import (
	"time"
	"github.com/shopspring/decimal"
)

// Stock represents basic stock information
type Stock struct {
	ID          int       `json:"id" db:"id"`
	Symbol      string    `json:"symbol" db:"symbol"`
	Name        string    `json:"name" db:"name"`
	Exchange    string    `json:"exchange" db:"exchange"`
	Sector      string    `json:"sector" db:"sector"`
	Industry    string    `json:"industry" db:"industry"`
	Country     string    `json:"country" db:"country"`
	Currency    string    `json:"currency" db:"currency"`
	MarketCap   *decimal.Decimal `json:"marketCap" db:"market_cap"`
	Description string    `json:"description" db:"description"`
	Website     string    `json:"website" db:"website"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

// StockPrice represents current and historical price data
type StockPrice struct {
	ID            int       `json:"id" db:"id"`
	Symbol        string    `json:"symbol" db:"symbol"`
	Price         decimal.Decimal `json:"price" db:"price"`
	Open          decimal.Decimal `json:"open" db:"open"`
	High          decimal.Decimal `json:"high" db:"high"`
	Low           decimal.Decimal `json:"low" db:"low"`
	Close         decimal.Decimal `json:"close" db:"close"`
	Volume        int64     `json:"volume" db:"volume"`
	Change        decimal.Decimal `json:"change" db:"change"`
	ChangePercent decimal.Decimal `json:"changePercent" db:"change_percent"`
	Timestamp     time.Time `json:"timestamp" db:"timestamp"`
}

// Fundamentals represents key financial metrics
type Fundamentals struct {
	ID                    int       `json:"id" db:"id"`
	Symbol                string    `json:"symbol" db:"symbol"`
	Period                string    `json:"period" db:"period"` // Q1, Q2, Q3, Q4, FY
	Year                  int       `json:"year" db:"year"`
	
	// Valuation Metrics
	PE                    *decimal.Decimal `json:"pe" db:"pe"`
	PEG                   *decimal.Decimal `json:"peg" db:"peg"`
	PB                    *decimal.Decimal `json:"pb" db:"pb"`
	PS                    *decimal.Decimal `json:"ps" db:"ps"`
	EV                    *decimal.Decimal `json:"ev" db:"ev"`
	EVRevenue             *decimal.Decimal `json:"evRevenue" db:"ev_revenue"`
	EVEBITDA              *decimal.Decimal `json:"evEbitda" db:"ev_ebitda"`
	
	// Per Share Metrics
	EPS                   *decimal.Decimal `json:"eps" db:"eps"`
	EPSDiluted            *decimal.Decimal `json:"epsDiluted" db:"eps_diluted"`
	BookValuePerShare     *decimal.Decimal `json:"bookValuePerShare" db:"book_value_per_share"`
	TangibleBookValue     *decimal.Decimal `json:"tangibleBookValue" db:"tangible_book_value"`
	
	// Income Statement
	Revenue               *decimal.Decimal `json:"revenue" db:"revenue"`
	GrossProfit           *decimal.Decimal `json:"grossProfit" db:"gross_profit"`
	OperatingIncome       *decimal.Decimal `json:"operatingIncome" db:"operating_income"`
	EBITDA                *decimal.Decimal `json:"ebitda" db:"ebitda"`
	NetIncome             *decimal.Decimal `json:"netIncome" db:"net_income"`
	
	// Margins
	GrossMargin           *decimal.Decimal `json:"grossMargin" db:"gross_margin"`
	OperatingMargin       *decimal.Decimal `json:"operatingMargin" db:"operating_margin"`
	NetMargin             *decimal.Decimal `json:"netMargin" db:"net_margin"`
	
	// Returns
	ROE                   *decimal.Decimal `json:"roe" db:"roe"`
	ROA                   *decimal.Decimal `json:"roa" db:"roa"`
	ROIC                  *decimal.Decimal `json:"roic" db:"roic"`
	
	// Balance Sheet
	TotalAssets           *decimal.Decimal `json:"totalAssets" db:"total_assets"`
	TotalLiabilities      *decimal.Decimal `json:"totalLiabilities" db:"total_liabilities"`
	TotalEquity           *decimal.Decimal `json:"totalEquity" db:"total_equity"`
	TotalDebt             *decimal.Decimal `json:"totalDebt" db:"total_debt"`
	Cash                  *decimal.Decimal `json:"cash" db:"cash"`
	
	// Ratios
	DebtToEquity          *decimal.Decimal `json:"debtToEquity" db:"debt_to_equity"`
	CurrentRatio          *decimal.Decimal `json:"currentRatio" db:"current_ratio"`
	QuickRatio            *decimal.Decimal `json:"quickRatio" db:"quick_ratio"`
	
	// Cash Flow
	OperatingCashFlow     *decimal.Decimal `json:"operatingCashFlow" db:"operating_cash_flow"`
	FreeCashFlow          *decimal.Decimal `json:"freeCashFlow" db:"free_cash_flow"`
	CapEx                 *decimal.Decimal `json:"capex" db:"capex"`
	
	// Share Information
	SharesOutstanding     *int64    `json:"sharesOutstanding" db:"shares_outstanding"`
	SharesFloat           *int64    `json:"sharesFloat" db:"shares_float"`
	
	CreatedAt             time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt             time.Time `json:"updatedAt" db:"updated_at"`
}

// Dividend represents dividend information
type Dividend struct {
	ID            int       `json:"id" db:"id"`
	Symbol        string    `json:"symbol" db:"symbol"`
	ExDate        time.Time `json:"exDate" db:"ex_date"`
	PayDate       time.Time `json:"payDate" db:"pay_date"`
	Amount        decimal.Decimal `json:"amount" db:"amount"`
	Frequency     string    `json:"frequency" db:"frequency"` // Monthly, Quarterly, Annual
	Type          string    `json:"type" db:"type"` // Regular, Special
	YieldPercent  *decimal.Decimal `json:"yieldPercent" db:"yield_percent"`
	CreatedAt     time.Time `json:"createdAt" db:"created_at"`
}

// Earnings represents earnings data
type Earnings struct {
	ID                int       `json:"id" db:"id"`
	Symbol            string    `json:"symbol" db:"symbol"`
	Quarter           string    `json:"quarter" db:"quarter"`
	Year              int       `json:"year" db:"year"`
	ReportDate        time.Time `json:"reportDate" db:"report_date"`
	
	// Earnings Data
	EPSActual         *decimal.Decimal `json:"epsActual" db:"eps_actual"`
	EPSEstimate       *decimal.Decimal `json:"epsEstimate" db:"eps_estimate"`
	EPSSurprise       *decimal.Decimal `json:"epsSurprise" db:"eps_surprise"`
	EPSSurprisePercent *decimal.Decimal `json:"epsSurprisePercent" db:"eps_surprise_percent"`
	
	// Revenue Data
	RevenueActual     *decimal.Decimal `json:"revenueActual" db:"revenue_actual"`
	RevenueEstimate   *decimal.Decimal `json:"revenueEstimate" db:"revenue_estimate"`
	RevenueSurprise   *decimal.Decimal `json:"revenueSurprise" db:"revenue_surprise"`
	
	// Guidance
	EPSGuidanceLow    *decimal.Decimal `json:"epsGuidanceLow" db:"eps_guidance_low"`
	EPSGuidanceHigh   *decimal.Decimal `json:"epsGuidanceHigh" db:"eps_guidance_high"`
	RevenueGuidanceLow *decimal.Decimal `json:"revenueGuidanceLow" db:"revenue_guidance_low"`
	RevenueGuidanceHigh *decimal.Decimal `json:"revenueGuidanceHigh" db:"revenue_guidance_high"`
	
	CreatedAt         time.Time `json:"createdAt" db:"created_at"`
}

// AnalystRating represents analyst recommendations
type AnalystRating struct {
	ID            int       `json:"id" db:"id"`
	Symbol        string    `json:"symbol" db:"symbol"`
	Firm          string    `json:"firm" db:"firm"`
	Analyst       string    `json:"analyst" db:"analyst"`
	Rating        string    `json:"rating" db:"rating"` // Strong Buy, Buy, Hold, Sell, Strong Sell
	PriceTarget   *decimal.Decimal `json:"priceTarget" db:"price_target"`
	PreviousRating *string  `json:"previousRating" db:"previous_rating"`
	RatingDate    time.Time `json:"ratingDate" db:"rating_date"`
	CreatedAt     time.Time `json:"createdAt" db:"created_at"`
}

// NewsArticle represents news and analysis
type NewsArticle struct {
	ID          int       `json:"id" db:"id"`
	Symbol      string    `json:"symbol" db:"symbol"`
	Title       string    `json:"title" db:"title"`
	Summary     string    `json:"summary" db:"summary"`
	Content     string    `json:"content" db:"content"`
	Author      string    `json:"author" db:"author"`
	Source      string    `json:"source" db:"source"`
	URL         string    `json:"url" db:"url"`
	Sentiment   string    `json:"sentiment" db:"sentiment"` // Positive, Negative, Neutral
	PublishedAt time.Time `json:"publishedAt" db:"published_at"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
}

// InsiderTrading represents insider trading activity
type InsiderTrading struct {
	ID              int       `json:"id" db:"id"`
	Symbol          string    `json:"symbol" db:"symbol"`
	InsiderName     string    `json:"insiderName" db:"insider_name"`
	Title           string    `json:"title" db:"title"`
	TransactionType string    `json:"transactionType" db:"transaction_type"` // Buy, Sell
	Shares          int64     `json:"shares" db:"shares"`
	Price           decimal.Decimal `json:"price" db:"price"`
	Value           decimal.Decimal `json:"value" db:"value"`
	SharesOwned     *int64    `json:"sharesOwned" db:"shares_owned"`
	TransactionDate time.Time `json:"transactionDate" db:"transaction_date"`
	FilingDate      time.Time `json:"filingDate" db:"filing_date"`
	CreatedAt       time.Time `json:"createdAt" db:"created_at"`
}

// PeerComparison represents peer comparison data
type PeerComparison struct {
	Symbol        string           `json:"symbol"`
	Name          string           `json:"name"`
	Price         decimal.Decimal  `json:"price"`
	MarketCap     *decimal.Decimal `json:"marketCap"`
	PE            *decimal.Decimal `json:"pe"`
	PB            *decimal.Decimal `json:"pb"`
	PS            *decimal.Decimal `json:"ps"`
	ROE           *decimal.Decimal `json:"roe"`
	DebtToEquity  *decimal.Decimal `json:"debtToEquity"`
	DividendYield *decimal.Decimal `json:"dividendYield"`
	Revenue       *decimal.Decimal `json:"revenue"`
	NetIncome     *decimal.Decimal `json:"netIncome"`
}

// TechnicalIndicators represents technical analysis data
type TechnicalIndicators struct {
	Symbol           string           `json:"symbol"`
	RSI              *decimal.Decimal `json:"rsi"`
	MACD             *decimal.Decimal `json:"macd"`
	MACDSignal       *decimal.Decimal `json:"macdSignal"`
	MACDHistogram    *decimal.Decimal `json:"macdHistogram"`
	SMA20            *decimal.Decimal `json:"sma20"`
	SMA50            *decimal.Decimal `json:"sma50"`
	SMA200           *decimal.Decimal `json:"sma200"`
	EMA12            *decimal.Decimal `json:"ema12"`
	EMA26            *decimal.Decimal `json:"ema26"`
	BollingerUpper   *decimal.Decimal `json:"bollingerUpper"`
	BollingerLower   *decimal.Decimal `json:"bollingerLower"`
	Support          *decimal.Decimal `json:"support"`
	Resistance       *decimal.Decimal `json:"resistance"`
	Volume20DayAvg   *int64           `json:"volume20DayAvg"`
	Beta             *decimal.Decimal `json:"beta"`
	Volatility       *decimal.Decimal `json:"volatility"`
	Timestamp        time.Time        `json:"timestamp"`
}

// StockSummary represents a comprehensive stock overview
type StockSummary struct {
	Stock               Stock                `json:"stock"`
	Price               StockPrice           `json:"price"`
	Fundamentals        *Fundamentals        `json:"fundamentals"`
	TechnicalIndicators *TechnicalIndicators `json:"technicalIndicators"`
	LatestEarnings      *Earnings            `json:"latestEarnings"`
	NextEarnings        *time.Time           `json:"nextEarnings"`
	DividendInfo        *DividendSummary     `json:"dividendInfo"`
	AnalystConsensus    *AnalystConsensus    `json:"analystConsensus"`
	KeyMetrics          *KeyMetrics          `json:"keyMetrics"`
}

// DividendSummary represents dividend summary information
type DividendSummary struct {
	YieldPercent      *decimal.Decimal `json:"yieldPercent"`
	AnnualDividend    *decimal.Decimal `json:"annualDividend"`
	PayoutRatio       *decimal.Decimal `json:"payoutRatio"`
	ExDate            *time.Time       `json:"exDate"`
	PayDate           *time.Time       `json:"payDate"`
	Frequency         string           `json:"frequency"`
	GrowthRate5Y      *decimal.Decimal `json:"growthRate5Y"`
	ConsecutiveYears  *int             `json:"consecutiveYears"`
}

// AnalystConsensus represents analyst rating consensus
type AnalystConsensus struct {
	Rating               string           `json:"rating"` // Strong Buy, Buy, Hold, Sell, Strong Sell
	RatingScore          decimal.Decimal  `json:"ratingScore"` // 1-5 scale
	PriceTarget          *decimal.Decimal `json:"priceTarget"`
	PriceTargetHigh      *decimal.Decimal `json:"priceTargetHigh"`
	PriceTargetLow       *decimal.Decimal `json:"priceTargetLow"`
	PriceTargetMedian    *decimal.Decimal `json:"priceTargetMedian"`
	Upside               *decimal.Decimal `json:"upside"`
	NumberOfAnalysts     int              `json:"numberOfAnalysts"`
	StrongBuy            int              `json:"strongBuy"`
	Buy                  int              `json:"buy"`
	Hold                 int              `json:"hold"`
	Sell                 int              `json:"sell"`
	StrongSell           int              `json:"strongSell"`
	LastUpdated          time.Time        `json:"lastUpdated"`
}

// KeyMetrics represents key financial metrics and ratios
type KeyMetrics struct {
	Symbol            string           `json:"symbol"`
	
	// Price Metrics
	Week52High        *decimal.Decimal `json:"week52High"`
	Week52Low         *decimal.Decimal `json:"week52Low"`
	Week52Change      *decimal.Decimal `json:"week52Change"`
	YTDChange         *decimal.Decimal `json:"ytdChange"`
	
	// Valuation
	MarketCap         *decimal.Decimal `json:"marketCap"`
	EnterpriseValue   *decimal.Decimal `json:"enterpriseValue"`
	TrailingPE        *decimal.Decimal `json:"trailingPE"`
	ForwardPE         *decimal.Decimal `json:"forwardPE"`
	PriceToBook       *decimal.Decimal `json:"priceToBook"`
	PriceToSales      *decimal.Decimal `json:"priceToSales"`
	
	// Growth Rates
	RevenueGrowth1Y   *decimal.Decimal `json:"revenueGrowth1Y"`
	RevenueGrowth3Y   *decimal.Decimal `json:"revenueGrowth3Y"`
	RevenueGrowth5Y   *decimal.Decimal `json:"revenueGrowth5Y"`
	EarningsGrowth1Y  *decimal.Decimal `json:"earningsGrowth1Y"`
	EarningsGrowth3Y  *decimal.Decimal `json:"earningsGrowth3Y"`
	EarningsGrowth5Y  *decimal.Decimal `json:"earningsGrowth5Y"`
	
	// Financial Health
	DebtToEquity      *decimal.Decimal `json:"debtToEquity"`
	CurrentRatio      *decimal.Decimal `json:"currentRatio"`
	QuickRatio        *decimal.Decimal `json:"quickRatio"`
	InterestCoverage  *decimal.Decimal `json:"interestCoverage"`
	
	// Efficiency
	AssetTurnover     *decimal.Decimal `json:"assetTurnover"`
	InventoryTurnover *decimal.Decimal `json:"inventoryTurnover"`
	ReceivablesTurnover *decimal.Decimal `json:"receivablesTurnover"`
	
	// Market Data
	Beta              *decimal.Decimal `json:"beta"`
	AverageVolume     *int64           `json:"averageVolume"`
	SharesOutstanding *int64           `json:"sharesOutstanding"`
	SharesFloat       *int64           `json:"sharesFloat"`
	ShortInterest     *decimal.Decimal `json:"shortInterest"`
	ShortRatio        *decimal.Decimal `json:"shortRatio"`
	
	LastUpdated       time.Time        `json:"lastUpdated"`
}

// ChartDataPoint represents a single point in price/volume charts
type ChartDataPoint struct {
	Timestamp time.Time       `json:"timestamp"`
	Open      decimal.Decimal `json:"open"`
	High      decimal.Decimal `json:"high"`
	Low       decimal.Decimal `json:"low"`
	Close     decimal.Decimal `json:"close"`
	Volume    int64           `json:"volume"`
	VWAP      *decimal.Decimal `json:"vwap,omitempty"`
}

// ChartData represents chart data with technical indicators
type ChartData struct {
	Symbol     string            `json:"symbol"`
	Period     string            `json:"period"`
	DataPoints []ChartDataPoint  `json:"dataPoints"`
	Indicators map[string][]decimal.Decimal `json:"indicators,omitempty"`
	LastUpdated time.Time        `json:"lastUpdated"`
}

// TickerPageData represents all data for a ticker page
type TickerPageData struct {
	Summary             StockSummary      `json:"summary"`
	ChartData           ChartData         `json:"chartData"`
	RecentNews          []NewsArticle     `json:"recentNews"`
	EarningsHistory     []Earnings        `json:"earningsHistory"`
	DividendHistory     []Dividend        `json:"dividendHistory"`
	AnalystRatings      []AnalystRating   `json:"analystRatings"`
	InsiderActivity     []InsiderTrading  `json:"insiderActivity"`
	PeerComparisons     []PeerComparison  `json:"peerComparisons"`
	FundamentalsHistory []Fundamentals    `json:"fundamentalsHistory"`
}
