package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// ===================
// EPS Estimates Models
// ===================

// EPSEstimate represents analyst EPS estimate data for a stock
type EPSEstimate struct {
	ID            string           `json:"id" db:"id"`
	Ticker        string           `json:"ticker" db:"ticker"`
	FiscalYear    int              `json:"fiscal_year" db:"fiscal_year"`
	FiscalQuarter *int             `json:"fiscal_quarter" db:"fiscal_quarter"` // NULL for annual

	// Estimate data
	ConsensusEPS *decimal.Decimal `json:"consensus_eps" db:"consensus_eps"`
	NumAnalysts  *int             `json:"num_analysts" db:"num_analysts"`
	HighEstimate *decimal.Decimal `json:"high_estimate" db:"high_estimate"`
	LowEstimate  *decimal.Decimal `json:"low_estimate" db:"low_estimate"`

	// Historical tracking for revision calculations
	Estimate30dAgo *decimal.Decimal `json:"estimate_30d_ago" db:"estimate_30d_ago"`
	Estimate60dAgo *decimal.Decimal `json:"estimate_60d_ago" db:"estimate_60d_ago"`
	Estimate90dAgo *decimal.Decimal `json:"estimate_90d_ago" db:"estimate_90d_ago"`

	// Revision counts
	Upgrades30d   int `json:"upgrades_30d" db:"upgrades_30d"`
	Downgrades30d int `json:"downgrades_30d" db:"downgrades_30d"`
	Upgrades60d   int `json:"upgrades_60d" db:"upgrades_60d"`
	Downgrades60d int `json:"downgrades_60d" db:"downgrades_60d"`
	Upgrades90d   int `json:"upgrades_90d" db:"upgrades_90d"`
	Downgrades90d int `json:"downgrades_90d" db:"downgrades_90d"`

	// Calculated revision metrics
	RevisionPct30d *decimal.Decimal `json:"revision_pct_30d" db:"revision_pct_30d"`
	RevisionPct60d *decimal.Decimal `json:"revision_pct_60d" db:"revision_pct_60d"`
	RevisionPct90d *decimal.Decimal `json:"revision_pct_90d" db:"revision_pct_90d"`

	FetchedAt time.Time `json:"fetched_at" db:"fetched_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// EPSEstimateResponse represents the API response for EPS estimates
type EPSEstimateResponse struct {
	Ticker        string   `json:"ticker"`
	FiscalYear    int      `json:"fiscal_year"`
	FiscalQuarter *int     `json:"fiscal_quarter,omitempty"`
	ConsensusEPS  *float64 `json:"consensus_eps"`
	NumAnalysts   *int     `json:"num_analysts"`
	HighEstimate  *float64 `json:"high_estimate"`
	LowEstimate   *float64 `json:"low_estimate"`
	RevisionPct   *float64 `json:"revision_pct_90d"`
	Upgrades      int      `json:"upgrades_90d"`
	Downgrades    int      `json:"downgrades_90d"`
}

// ToResponse converts EPSEstimate to API response
func (e *EPSEstimate) ToResponse() EPSEstimateResponse {
	resp := EPSEstimateResponse{
		Ticker:        e.Ticker,
		FiscalYear:    e.FiscalYear,
		FiscalQuarter: e.FiscalQuarter,
		NumAnalysts:   e.NumAnalysts,
		Upgrades:      e.Upgrades90d,
		Downgrades:    e.Downgrades90d,
	}

	if e.ConsensusEPS != nil {
		v := toFloat64(*e.ConsensusEPS)
		resp.ConsensusEPS = &v
	}
	if e.HighEstimate != nil {
		v := toFloat64(*e.HighEstimate)
		resp.HighEstimate = &v
	}
	if e.LowEstimate != nil {
		v := toFloat64(*e.LowEstimate)
		resp.LowEstimate = &v
	}
	if e.RevisionPct90d != nil {
		v := toFloat64(*e.RevisionPct90d)
		resp.RevisionPct = &v
	}

	return resp
}

// ===================
// Valuation History Models
// ===================

// ValuationHistory represents a monthly valuation snapshot for historical comparison
type ValuationHistory struct {
	ID           string    `json:"id" db:"id"`
	Ticker       string    `json:"ticker" db:"ticker"`
	SnapshotDate time.Time `json:"snapshot_date" db:"snapshot_date"`

	// Valuation metrics
	PERatio  *decimal.Decimal `json:"pe_ratio" db:"pe_ratio"`
	PSRatio  *decimal.Decimal `json:"ps_ratio" db:"ps_ratio"`
	PBRatio  *decimal.Decimal `json:"pb_ratio" db:"pb_ratio"`
	EVEbitda *decimal.Decimal `json:"ev_ebitda" db:"ev_ebitda"`
	PEGRatio *decimal.Decimal `json:"peg_ratio" db:"peg_ratio"`

	// Price context
	StockPrice *decimal.Decimal `json:"stock_price" db:"stock_price"`
	MarketCap  *decimal.Decimal `json:"market_cap" db:"market_cap"`

	// Earnings data at snapshot time
	EPSTTM     *decimal.Decimal `json:"eps_ttm" db:"eps_ttm"`
	RevenueTTM *decimal.Decimal `json:"revenue_ttm" db:"revenue_ttm"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ValuationHistoryResponse represents the API response for valuation history
type ValuationHistoryResponse struct {
	Ticker       string   `json:"ticker"`
	SnapshotDate string   `json:"snapshot_date"`
	PERatio      *float64 `json:"pe_ratio"`
	PSRatio      *float64 `json:"ps_ratio"`
	PBRatio      *float64 `json:"pb_ratio"`
	StockPrice   *float64 `json:"stock_price"`
}

// ValuationHistorySummary provides 5-year valuation range summary
type ValuationHistorySummary struct {
	Ticker        string   `json:"ticker"`
	CurrentPE     *float64 `json:"current_pe"`
	PE5YrLow      *float64 `json:"pe_5yr_low"`
	PE5YrHigh     *float64 `json:"pe_5yr_high"`
	PE5YrMedian   *float64 `json:"pe_5yr_median"`
	PEPercentile  *float64 `json:"pe_percentile"`
	CurrentPS     *float64 `json:"current_ps"`
	PS5YrLow      *float64 `json:"ps_5yr_low"`
	PS5YrHigh     *float64 `json:"ps_5yr_high"`
	PS5YrMedian   *float64 `json:"ps_5yr_median"`
	PSPercentile  *float64 `json:"ps_percentile"`
	DataPoints    int      `json:"data_points"`
	IsGrowthStock bool     `json:"is_growth_stock"`
}

// ToResponse converts ValuationHistory to API response
func (v *ValuationHistory) ToResponse() ValuationHistoryResponse {
	resp := ValuationHistoryResponse{
		Ticker:       v.Ticker,
		SnapshotDate: v.SnapshotDate.Format("2006-01-02"),
	}

	if v.PERatio != nil {
		val := toFloat64(*v.PERatio)
		resp.PERatio = &val
	}
	if v.PSRatio != nil {
		val := toFloat64(*v.PSRatio)
		resp.PSRatio = &val
	}
	if v.PBRatio != nil {
		val := toFloat64(*v.PBRatio)
		resp.PBRatio = &val
	}
	if v.StockPrice != nil {
		val := toFloat64(*v.StockPrice)
		resp.StockPrice = &val
	}

	return resp
}

// ===================
// Dividend History Models
// ===================

// DividendTier represents the dividend classification tier
type DividendTier string

const (
	DividendTierKing     DividendTier = "Dividend King"      // 50+ years
	DividendTierAristocr DividendTier = "Dividend Aristocrat" // 25+ years
	DividendTierAchiever DividendTier = "Dividend Achiever"   // 10+ years
	DividendTierPayer    DividendTier = "Dividend Payer"      // 5+ years
	DividendTierNew      DividendTier = "New Dividend Payer"  // <5 years
	DividendTierNone     DividendTier = "Non-Dividend"
)

// DividendHistory represents dividend data for a fiscal year
type DividendHistory struct {
	ID         string `json:"id" db:"id"`
	Ticker     string `json:"ticker" db:"ticker"`
	FiscalYear int    `json:"fiscal_year" db:"fiscal_year"`

	// Dividend metrics
	AnnualDividend    *decimal.Decimal `json:"annual_dividend" db:"annual_dividend"`
	DividendYield     *decimal.Decimal `json:"dividend_yield" db:"dividend_yield"`
	PayoutRatio       *decimal.Decimal `json:"payout_ratio" db:"payout_ratio"`
	DividendGrowthYoY *decimal.Decimal `json:"dividend_growth_yoy" db:"dividend_growth_yoy"`

	// Ex-dividend dates (for catalyst tracking)
	ExDividendDate *time.Time `json:"ex_dividend_date" db:"ex_dividend_date"`
	PaymentDate    *time.Time `json:"payment_date" db:"payment_date"`

	// Dividend streak tracking
	ConsecutiveYearsPaid      int `json:"consecutive_years_paid" db:"consecutive_years_paid"`
	ConsecutiveYearsIncreased int `json:"consecutive_years_increased" db:"consecutive_years_increased"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// DividendHistoryResponse represents the API response for dividend history
type DividendHistoryResponse struct {
	Ticker                    string       `json:"ticker"`
	FiscalYear                int          `json:"fiscal_year"`
	AnnualDividend            *float64     `json:"annual_dividend"`
	DividendYield             *float64     `json:"dividend_yield"`
	PayoutRatio               *float64     `json:"payout_ratio"`
	DividendGrowthYoY         *float64     `json:"dividend_growth_yoy"`
	ConsecutiveYearsPaid      int          `json:"consecutive_years_paid"`
	ConsecutiveYearsIncreased int          `json:"consecutive_years_increased"`
	DividendTier              DividendTier `json:"dividend_tier"`
	ExDividendDate            *string      `json:"ex_dividend_date,omitempty"`
	PaymentDate               *string      `json:"payment_date,omitempty"`
}

// ToResponse converts DividendHistory to API response
func (d *DividendHistory) ToResponse() DividendHistoryResponse {
	resp := DividendHistoryResponse{
		Ticker:                    d.Ticker,
		FiscalYear:                d.FiscalYear,
		ConsecutiveYearsPaid:      d.ConsecutiveYearsPaid,
		ConsecutiveYearsIncreased: d.ConsecutiveYearsIncreased,
		DividendTier:              d.GetDividendTier(),
	}

	if d.AnnualDividend != nil {
		v := toFloat64(*d.AnnualDividend)
		resp.AnnualDividend = &v
	}
	if d.DividendYield != nil {
		v := toFloat64(*d.DividendYield)
		resp.DividendYield = &v
	}
	if d.PayoutRatio != nil {
		v := toFloat64(*d.PayoutRatio)
		resp.PayoutRatio = &v
	}
	if d.DividendGrowthYoY != nil {
		v := toFloat64(*d.DividendGrowthYoY)
		resp.DividendGrowthYoY = &v
	}
	if d.ExDividendDate != nil {
		s := d.ExDividendDate.Format("2006-01-02")
		resp.ExDividendDate = &s
	}
	if d.PaymentDate != nil {
		s := d.PaymentDate.Format("2006-01-02")
		resp.PaymentDate = &s
	}

	return resp
}

// GetDividendTier returns the dividend tier based on consecutive years
func (d *DividendHistory) GetDividendTier() DividendTier {
	years := d.ConsecutiveYearsIncreased

	switch {
	case years >= 50:
		return DividendTierKing
	case years >= 25:
		return DividendTierAristocr
	case years >= 10:
		return DividendTierAchiever
	case years >= 5:
		return DividendTierPayer
	case years > 0:
		return DividendTierNew
	default:
		return DividendTierNone
	}
}

// ===================
// Earnings Revisions Factor Response
// ===================

// EarningsRevisionsFactorResponse represents the detailed Earnings Revisions factor data
type EarningsRevisionsFactorResponse struct {
	Score          float64  `json:"score"`
	MagnitudeScore float64  `json:"magnitude_score"`
	BreadthScore   float64  `json:"breadth_score"`
	RecencyScore   float64  `json:"recency_score"`
	ConsensusEPS   *float64 `json:"consensus_eps"`
	RevisionPct90d *float64 `json:"revision_pct_90d"`
	Upgrades90d    int      `json:"upgrades_90d"`
	Downgrades90d  int      `json:"downgrades_90d"`
	NumAnalysts    *int     `json:"num_analysts"`
	EstimateSpread *float64 `json:"estimate_spread"` // (high - low) / consensus as %
}

// ===================
// Historical Valuation Factor Response
// ===================

// HistoricalValuationFactorResponse represents the detailed Historical Valuation factor data
type HistoricalValuationFactorResponse struct {
	Score           float64  `json:"score"`
	PEPercentile    *float64 `json:"pe_percentile"`
	PSPercentile    *float64 `json:"ps_percentile"`
	CurrentPE       *float64 `json:"current_pe"`
	CurrentPS       *float64 `json:"current_ps"`
	PE5YrLow        *float64 `json:"pe_5yr_low"`
	PE5YrHigh       *float64 `json:"pe_5yr_high"`
	PE5YrMedian     *float64 `json:"pe_5yr_median"`
	PS5YrLow        *float64 `json:"ps_5yr_low"`
	PS5YrHigh       *float64 `json:"ps_5yr_high"`
	PS5YrMedian     *float64 `json:"ps_5yr_median"`
	IsGrowthCompany bool     `json:"is_growth_company"`
	DataPoints      int      `json:"data_points"`
}

// ===================
// Dividend Quality Factor Response
// ===================

// DividendQualityFactorResponse represents the detailed Dividend Quality factor data
type DividendQualityFactorResponse struct {
	Score                     float64      `json:"score"`
	YieldScore                float64      `json:"yield_score"`
	PayoutScore               float64      `json:"payout_score"`
	GrowthScore               float64      `json:"growth_score"`
	StreakScore               float64      `json:"streak_score"`
	DividendYield             *float64     `json:"dividend_yield"`
	PayoutRatio               *float64     `json:"payout_ratio"`
	DividendCAGR5Y            *float64     `json:"dividend_cagr_5y"`
	DividendGrowthYoY         *float64     `json:"dividend_growth_yoy"`
	ConsecutiveYearsPaid      int          `json:"consecutive_years_paid"`
	ConsecutiveYearsIncreased int          `json:"consecutive_years_increased"`
	DividendTier              DividendTier `json:"dividend_tier"`
	IsDividendPayer           bool         `json:"is_dividend_payer"`
}
