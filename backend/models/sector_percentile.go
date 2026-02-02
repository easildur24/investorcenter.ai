package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// SectorPercentile represents distribution statistics for a metric within a sector
// Used for IC Score v2.1 sector-relative scoring
type SectorPercentile struct {
	ID           string    `json:"id" db:"id"`
	Sector       string    `json:"sector" db:"sector"`
	MetricName   string    `json:"metric_name" db:"metric_name"`
	CalculatedAt time.Time `json:"calculated_at" db:"calculated_at"`

	// Distribution statistics
	MinValue    *decimal.Decimal `json:"min_value" db:"min_value"`
	P10Value    *decimal.Decimal `json:"p10_value" db:"p10_value"`
	P25Value    *decimal.Decimal `json:"p25_value" db:"p25_value"`
	P50Value    *decimal.Decimal `json:"p50_value" db:"p50_value"` // median
	P75Value    *decimal.Decimal `json:"p75_value" db:"p75_value"`
	P90Value    *decimal.Decimal `json:"p90_value" db:"p90_value"`
	MaxValue    *decimal.Decimal `json:"max_value" db:"max_value"`
	MeanValue   *decimal.Decimal `json:"mean_value" db:"mean_value"`
	StdDev      *decimal.Decimal `json:"std_dev" db:"std_dev"`
	SampleCount *int             `json:"sample_count" db:"sample_count"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// SectorPercentileResponse represents the API response for sector percentile data
type SectorPercentileResponse struct {
	Sector       string   `json:"sector"`
	MetricName   string   `json:"metric_name"`
	CalculatedAt string   `json:"calculated_at"`
	MinValue     *float64 `json:"min_value"`
	P10Value     *float64 `json:"p10_value"`
	P25Value     *float64 `json:"p25_value"`
	P50Value     *float64 `json:"p50_value"`
	P75Value     *float64 `json:"p75_value"`
	P90Value     *float64 `json:"p90_value"`
	MaxValue     *float64 `json:"max_value"`
	MeanValue    *float64 `json:"mean_value"`
	StdDev       *float64 `json:"std_dev"`
	SampleCount  *int     `json:"sample_count"`
}

// ToResponse converts SectorPercentile model to API response format
func (sp *SectorPercentile) ToResponse() SectorPercentileResponse {
	return SectorPercentileResponse{
		Sector:       sp.Sector,
		MetricName:   sp.MetricName,
		CalculatedAt: sp.CalculatedAt.Format("2006-01-02"),
		MinValue:     decimalPtrToFloat64Ptr(sp.MinValue),
		P10Value:     decimalPtrToFloat64Ptr(sp.P10Value),
		P25Value:     decimalPtrToFloat64Ptr(sp.P25Value),
		P50Value:     decimalPtrToFloat64Ptr(sp.P50Value),
		P75Value:     decimalPtrToFloat64Ptr(sp.P75Value),
		P90Value:     decimalPtrToFloat64Ptr(sp.P90Value),
		MaxValue:     decimalPtrToFloat64Ptr(sp.MaxValue),
		MeanValue:    decimalPtrToFloat64Ptr(sp.MeanValue),
		StdDev:       decimalPtrToFloat64Ptr(sp.StdDev),
		SampleCount:  sp.SampleCount,
	}
}

// LifecycleClassification represents a company's lifecycle stage classification
type LifecycleClassification struct {
	ID             string    `json:"id" db:"id"`
	Ticker         string    `json:"ticker" db:"ticker"`
	ClassifiedAt   time.Time `json:"classified_at" db:"classified_at"`
	LifecycleStage string    `json:"lifecycle_stage" db:"lifecycle_stage"`

	// Input metrics used for classification
	RevenueGrowthYoY *decimal.Decimal `json:"revenue_growth_yoy" db:"revenue_growth_yoy"`
	NetMargin        *decimal.Decimal `json:"net_margin" db:"net_margin"`
	PERatio          *decimal.Decimal `json:"pe_ratio" db:"pe_ratio"`
	MarketCap        *int64           `json:"market_cap" db:"market_cap"`

	// Adjusted weights applied
	WeightsApplied map[string]any `json:"weights_applied,omitempty" db:"weights_applied"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// LifecycleClassificationResponse represents the API response
type LifecycleClassificationResponse struct {
	Ticker           string             `json:"ticker"`
	LifecycleStage   string             `json:"lifecycle_stage"`
	ClassifiedAt     string             `json:"classified_at"`
	RevenueGrowthYoY *float64           `json:"revenue_growth_yoy,omitempty"`
	NetMargin        *float64           `json:"net_margin,omitempty"`
	PERatio          *float64           `json:"pe_ratio,omitempty"`
	WeightsApplied   map[string]float64 `json:"weights_applied,omitempty"`
	Description      string             `json:"description"`
}

// ToResponse converts LifecycleClassification model to API response
func (lc *LifecycleClassification) ToResponse() LifecycleClassificationResponse {
	response := LifecycleClassificationResponse{
		Ticker:           lc.Ticker,
		LifecycleStage:   lc.LifecycleStage,
		ClassifiedAt:     lc.ClassifiedAt.Format("2006-01-02"),
		RevenueGrowthYoY: decimalPtrToFloat64Ptr(lc.RevenueGrowthYoY),
		NetMargin:        decimalPtrToFloat64Ptr(lc.NetMargin),
		PERatio:          decimalPtrToFloat64Ptr(lc.PERatio),
		Description:      GetLifecycleDescription(lc.LifecycleStage),
	}

	// Convert weights to float64 map
	if lc.WeightsApplied != nil {
		response.WeightsApplied = make(map[string]float64)
		for k, v := range lc.WeightsApplied {
			if f, ok := v.(float64); ok {
				response.WeightsApplied[k] = f
			}
		}
	}

	return response
}

// GetLifecycleDescription returns a human-readable description for a lifecycle stage
func GetLifecycleDescription(stage string) string {
	descriptions := map[string]string{
		"hypergrowth": "Hypergrowth company with >50% revenue growth. Focus on growth trajectory over current profitability.",
		"growth":      "Growth company with 20-50% revenue growth. Balancing expansion with emerging profitability.",
		"mature":      "Mature company with stable operations. Focus on profitability, cash flow, and capital efficiency.",
		"value":       "Value opportunity with low valuation and solid margins. Focus on intrinsic value and dividend potential.",
		"turnaround":  "Turnaround situation with declining revenue. Focus on financial health and recovery signals.",
	}

	if desc, ok := descriptions[stage]; ok {
		return desc
	}
	return "Unknown lifecycle stage"
}

// Metrics tracked for sector percentiles
var TrackedMetrics = []string{
	// Valuation (lower is better)
	"pe_ratio", "ps_ratio", "pb_ratio", "ev_ebitda", "peg_ratio",
	// Profitability (higher is better)
	"roe", "roa", "roic", "gross_margin", "operating_margin", "net_margin",
	// Growth (higher is better)
	"revenue_growth_yoy", "earnings_growth_yoy", "eps_growth_yoy",
	// Financial Health
	"current_ratio", "quick_ratio", "debt_to_equity", "interest_coverage",
	// Efficiency
	"asset_turnover", "inventory_turnover", "receivables_turnover",
	// Market
	"dividend_yield", "free_cash_flow_yield", "earnings_yield",
}

// LowerIsBetterMetrics are metrics where lower values indicate better performance
var LowerIsBetterMetrics = map[string]bool{
	"pe_ratio":           true,
	"ps_ratio":           true,
	"pb_ratio":           true,
	"ev_ebitda":          true,
	"peg_ratio":          true,
	"debt_to_equity":     true,
	"net_debt_to_ebitda": true,
}

// Helper function for decimal pointer to float64 pointer conversion
func decimalPtrToFloat64Ptr(d *decimal.Decimal) *float64 {
	if d == nil {
		return nil
	}
	f, _ := d.Float64()
	return &f
}
