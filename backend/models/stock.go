package models

import (
	"time"

	"github.com/shopspring/decimal"
)

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

// Helper function to create decimal from float
func DecimalFromFloat(f float64) decimal.Decimal {
	return decimal.NewFromFloat(f)
}
