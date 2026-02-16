package database

import (
	"fmt"
	"strings"

	"investorcenter-api/models"
)

// RangeFilterDef defines a min/max range filter that maps URL params to a SQL column.
type RangeFilterDef struct {
	Column string                                  // SQL column in screener_data
	GetMin func(p *models.ScreenerParams) *float64 // Accessor for min value
	GetMax func(p *models.ScreenerParams) *float64 // Accessor for max value
}

// RangeFilters is the registry of all supported range filters.
// Adding a new range filter requires only adding an entry here and the
// corresponding fields to ScreenerParams + the handler's param parser.
var RangeFilters = []RangeFilterDef{
	// Market data
	{Column: "market_cap", GetMin: func(p *models.ScreenerParams) *float64 { return p.MarketCapMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.MarketCapMax }},

	// Valuation
	{Column: "pe_ratio", GetMin: func(p *models.ScreenerParams) *float64 { return p.PEMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.PEMax }},
	{Column: "pb_ratio", GetMin: func(p *models.ScreenerParams) *float64 { return p.PBMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.PBMax }},
	{Column: "ps_ratio", GetMin: func(p *models.ScreenerParams) *float64 { return p.PSMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.PSMax }},

	// Profitability
	{Column: "roe", GetMin: func(p *models.ScreenerParams) *float64 { return p.ROEMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.ROEMax }},
	{Column: "roa", GetMin: func(p *models.ScreenerParams) *float64 { return p.ROAMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.ROAMax }},
	{Column: "gross_margin", GetMin: func(p *models.ScreenerParams) *float64 { return p.GrossMarginMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.GrossMarginMax }},
	{Column: "net_margin", GetMin: func(p *models.ScreenerParams) *float64 { return p.NetMarginMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.NetMarginMax }},

	// Financial health
	{Column: "debt_to_equity", GetMin: func(p *models.ScreenerParams) *float64 { return p.DebtToEquityMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.DebtToEquityMax }},
	{Column: "current_ratio", GetMin: func(p *models.ScreenerParams) *float64 { return p.CurrentRatioMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.CurrentRatioMax }},

	// Growth
	{Column: "revenue_growth", GetMin: func(p *models.ScreenerParams) *float64 { return p.RevenueGrowthMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.RevenueGrowthMax }},
	{Column: "eps_growth_yoy", GetMin: func(p *models.ScreenerParams) *float64 { return p.EPSGrowthMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.EPSGrowthMax }},

	// Dividends
	{Column: "dividend_yield", GetMin: func(p *models.ScreenerParams) *float64 { return p.DividendYieldMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.DividendYieldMax }},
	{Column: "payout_ratio", GetMin: func(p *models.ScreenerParams) *float64 { return p.PayoutRatioMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.PayoutRatioMax }},
	{Column: "consecutive_dividend_years", GetMin: func(p *models.ScreenerParams) *float64 { return p.ConsecutiveDivYearsMin }, GetMax: func(p *models.ScreenerParams) *float64 { return nil }},

	// Risk
	{Column: "beta", GetMin: func(p *models.ScreenerParams) *float64 { return p.BetaMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.BetaMax }},

	// Fair value
	{Column: "dcf_upside_percent", GetMin: func(p *models.ScreenerParams) *float64 { return p.DCFUpsideMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.DCFUpsideMax }},

	// IC Score
	{Column: "ic_score", GetMin: func(p *models.ScreenerParams) *float64 { return p.ICScoreMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.ICScoreMax }},

	// IC Score sub-factors
	{Column: "value_score", GetMin: func(p *models.ScreenerParams) *float64 { return p.ValueScoreMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.ValueScoreMax }},
	{Column: "growth_score", GetMin: func(p *models.ScreenerParams) *float64 { return p.GrowthScoreMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.GrowthScoreMax }},
	{Column: "profitability_score", GetMin: func(p *models.ScreenerParams) *float64 { return p.ProfitabilityScoreMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.ProfitabilityScoreMax }},
	{Column: "financial_health_score", GetMin: func(p *models.ScreenerParams) *float64 { return p.FinancialHealthScoreMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.FinancialHealthScoreMax }},
	{Column: "momentum_score", GetMin: func(p *models.ScreenerParams) *float64 { return p.MomentumScoreMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.MomentumScoreMax }},
	{Column: "analyst_consensus_score", GetMin: func(p *models.ScreenerParams) *float64 { return p.AnalystScoreMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.AnalystScoreMax }},
	{Column: "insider_activity_score", GetMin: func(p *models.ScreenerParams) *float64 { return p.InsiderScoreMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.InsiderScoreMax }},
	{Column: "institutional_score", GetMin: func(p *models.ScreenerParams) *float64 { return p.InstitutionalScoreMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.InstitutionalScoreMax }},
	{Column: "news_sentiment_score", GetMin: func(p *models.ScreenerParams) *float64 { return p.SentimentScoreMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.SentimentScoreMax }},
	{Column: "technical_score", GetMin: func(p *models.ScreenerParams) *float64 { return p.TechnicalScoreMin }, GetMax: func(p *models.ScreenerParams) *float64 { return p.TechnicalScoreMax }},
}

// BuildFilterConditions generates parameterized WHERE conditions from the
// filter registry. Returns the conditions slice, args slice, and next arg index.
func BuildFilterConditions(params *models.ScreenerParams, startIndex int) ([]string, []interface{}, int) {
	conditions := []string{}
	args := []interface{}{}
	argIndex := startIndex

	// Categorical: sectors
	if len(params.Sectors) > 0 {
		placeholders := make([]string, len(params.Sectors))
		for i, sector := range params.Sectors {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, sector)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("sector IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Categorical: industries
	if len(params.Industries) > 0 {
		placeholders := make([]string, len(params.Industries))
		for i, industry := range params.Industries {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, industry)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("industry IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Range filters from registry
	for _, f := range RangeFilters {
		minVal := f.GetMin(params)
		maxVal := f.GetMax(params)

		if minVal != nil {
			conditions = append(conditions, fmt.Sprintf("%s >= $%d", f.Column, argIndex))
			args = append(args, *minVal)
			argIndex++
		}
		if maxVal != nil {
			conditions = append(conditions, fmt.Sprintf("%s <= $%d", f.Column, argIndex))
			args = append(args, *maxVal)
			argIndex++
		}
	}

	return conditions, args, argIndex
}
