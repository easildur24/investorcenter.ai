package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// LifecycleStage represents the company lifecycle classification
type LifecycleStage string

const (
	LifecycleHypergrowth LifecycleStage = "hypergrowth"
	LifecycleGrowth      LifecycleStage = "growth"
	LifecycleMature      LifecycleStage = "mature"
	LifecycleValue       LifecycleStage = "value"
	LifecycleTurnaround  LifecycleStage = "turnaround"
)

// ICScore represents the InvestorCenter proprietary IC Score
// v2.1: Added lifecycle_stage, sector context, and smoothing fields
// Phase 2: Added earnings_revisions, historical_value, dividend_quality
type ICScore struct {
	ID                    int64            `json:"id" db:"id"`
	Ticker                string           `json:"ticker" db:"ticker"`
	Date                  time.Time        `json:"date" db:"date"`
	OverallScore          decimal.Decimal  `json:"overall_score" db:"overall_score"`
	ValueScore            *decimal.Decimal `json:"value_score" db:"value_score"`
	GrowthScore           *decimal.Decimal `json:"growth_score" db:"growth_score"`
	ProfitabilityScore    *decimal.Decimal `json:"profitability_score" db:"profitability_score"`
	FinancialHealthScore  *decimal.Decimal `json:"financial_health_score" db:"financial_health_score"`
	MomentumScore         *decimal.Decimal `json:"momentum_score" db:"momentum_score"`
	AnalystConsensusScore *decimal.Decimal `json:"analyst_consensus_score" db:"analyst_consensus_score"`
	InsiderActivityScore  *decimal.Decimal `json:"insider_activity_score" db:"insider_activity_score"`
	InstitutionalScore    *decimal.Decimal `json:"institutional_score" db:"institutional_score"`
	NewsSentimentScore    *decimal.Decimal `json:"news_sentiment_score" db:"news_sentiment_score"`
	TechnicalScore        *decimal.Decimal `json:"technical_score" db:"technical_score"`

	// Phase 2: New factor scores
	EarningsRevisionsScore *decimal.Decimal `json:"earnings_revisions_score" db:"earnings_revisions_score"`
	HistoricalValueScore   *decimal.Decimal `json:"historical_value_score" db:"historical_value_score"`
	DividendQualityScore   *decimal.Decimal `json:"dividend_quality_score" db:"dividend_quality_score"`

	Rating           *string          `json:"rating" db:"rating"`
	SectorPercentile *decimal.Decimal `json:"sector_percentile" db:"sector_percentile"`
	ConfidenceLevel  *string          `json:"confidence_level" db:"confidence_level"`
	DataCompleteness *decimal.Decimal `json:"data_completeness" db:"data_completeness"`

	// v2.1: Lifecycle and sector context
	LifecycleStage   *string          `json:"lifecycle_stage" db:"lifecycle_stage"`
	RawScore         *decimal.Decimal `json:"raw_score" db:"raw_score"`
	SmoothingApplied bool             `json:"smoothing_applied" db:"smoothing_applied"`
	WeightsUsed      map[string]any   `json:"weights_used,omitempty" db:"weights_used"`
	SectorRank       *int             `json:"sector_rank" db:"sector_rank"`
	SectorTotal      *int             `json:"sector_total" db:"sector_total"`

	CalculationMetadata map[string]any `json:"calculation_metadata,omitempty" db:"calculation_metadata"`
	CalculatedAt        time.Time      `json:"calculated_at" db:"created_at"`
}

// ICScoreResponse represents the API response for IC Score
// v2.1: Added lifecycle_stage, sector_rank, peer_comparison
// Phase 2: Added earnings_revisions, historical_value, dividend_quality
type ICScoreResponse struct {
	Ticker                string   `json:"ticker"`
	Date                  string   `json:"date"`
	OverallScore          float64  `json:"overall_score"`
	ValueScore            *float64 `json:"value_score"`
	GrowthScore           *float64 `json:"growth_score"`
	ProfitabilityScore    *float64 `json:"profitability_score"`
	FinancialHealthScore  *float64 `json:"financial_health_score"`
	MomentumScore         *float64 `json:"momentum_score"`
	AnalystConsensusScore *float64 `json:"analyst_consensus_score"`
	InsiderActivityScore  *float64 `json:"insider_activity_score"`
	InstitutionalScore    *float64 `json:"institutional_score"`
	NewsSentimentScore    *float64 `json:"news_sentiment_score"`
	TechnicalScore        *float64 `json:"technical_score"`

	// Phase 2: New factor scores
	EarningsRevisionsScore *float64 `json:"earnings_revisions_score,omitempty"`
	HistoricalValueScore   *float64 `json:"historical_value_score,omitempty"`
	DividendQualityScore   *float64 `json:"dividend_quality_score,omitempty"`

	Rating           string   `json:"rating"`
	SectorPercentile *float64 `json:"sector_percentile"`
	ConfidenceLevel  string   `json:"confidence_level"`
	DataCompleteness float64  `json:"data_completeness"`
	CalculatedAt     string   `json:"calculated_at"`
	FactorCount      int      `json:"factor_count"`
	AvailableFactors []string `json:"available_factors"`
	MissingFactors   []string `json:"missing_factors"`

	// v2.1: New fields
	LifecycleStage *string  `json:"lifecycle_stage,omitempty"`
	SectorRank     *int     `json:"sector_rank,omitempty"`
	SectorTotal    *int     `json:"sector_total,omitempty"`
	ScoringVersion string   `json:"scoring_version"`
	RawScore       *float64 `json:"raw_score,omitempty"`
	IncomeMode     bool     `json:"income_mode"`
}

// ICScoreListItem represents a summary for the admin list view
type ICScoreListItem struct {
	Ticker           string    `json:"ticker" db:"ticker"`
	OverallScore     float64   `json:"overall_score" db:"overall_score"`
	Rating           string    `json:"rating" db:"rating"`
	DataCompleteness float64   `json:"data_completeness" db:"data_completeness"`
	CalculatedAt     time.Time `json:"calculated_at" db:"created_at"`
}

// ToResponse converts ICScore model to API response format
func (ic *ICScore) ToResponse() ICScoreResponse {
	response := ICScoreResponse{
		Ticker:           ic.Ticker,
		Date:             ic.Date.Format("2006-01-02"),
		OverallScore:     toFloat64(ic.OverallScore),
		Rating:           stringOrDefault(ic.Rating, "N/A"),
		ConfidenceLevel:  stringOrDefault(ic.ConfidenceLevel, "Low"),
		DataCompleteness: decimalToFloat64(ic.DataCompleteness, 0),
		CalculatedAt:     ic.CalculatedAt.Format(time.RFC3339),
		AvailableFactors: []string{},
		MissingFactors:   []string{},
	}

	// Track available and missing factors
	factorCount := 0

	if ic.ValueScore != nil {
		v := toFloat64(*ic.ValueScore)
		response.ValueScore = &v
		factorCount++
		response.AvailableFactors = append(response.AvailableFactors, "value")
	} else {
		response.MissingFactors = append(response.MissingFactors, "value")
	}

	if ic.GrowthScore != nil {
		v := toFloat64(*ic.GrowthScore)
		response.GrowthScore = &v
		factorCount++
		response.AvailableFactors = append(response.AvailableFactors, "growth")
	} else {
		response.MissingFactors = append(response.MissingFactors, "growth")
	}

	if ic.ProfitabilityScore != nil {
		v := toFloat64(*ic.ProfitabilityScore)
		response.ProfitabilityScore = &v
		factorCount++
		response.AvailableFactors = append(response.AvailableFactors, "profitability")
	} else {
		response.MissingFactors = append(response.MissingFactors, "profitability")
	}

	if ic.FinancialHealthScore != nil {
		v := toFloat64(*ic.FinancialHealthScore)
		response.FinancialHealthScore = &v
		factorCount++
		response.AvailableFactors = append(response.AvailableFactors, "financial_health")
	} else {
		response.MissingFactors = append(response.MissingFactors, "financial_health")
	}

	if ic.MomentumScore != nil {
		v := toFloat64(*ic.MomentumScore)
		response.MomentumScore = &v
		factorCount++
		response.AvailableFactors = append(response.AvailableFactors, "momentum")
	} else {
		response.MissingFactors = append(response.MissingFactors, "momentum")
	}

	if ic.AnalystConsensusScore != nil {
		v := toFloat64(*ic.AnalystConsensusScore)
		response.AnalystConsensusScore = &v
		factorCount++
		response.AvailableFactors = append(response.AvailableFactors, "analyst_consensus")
	} else {
		response.MissingFactors = append(response.MissingFactors, "analyst_consensus")
	}

	if ic.InsiderActivityScore != nil {
		v := toFloat64(*ic.InsiderActivityScore)
		response.InsiderActivityScore = &v
		factorCount++
		response.AvailableFactors = append(response.AvailableFactors, "insider_activity")
	} else {
		response.MissingFactors = append(response.MissingFactors, "insider_activity")
	}

	if ic.InstitutionalScore != nil {
		v := toFloat64(*ic.InstitutionalScore)
		response.InstitutionalScore = &v
		factorCount++
		response.AvailableFactors = append(response.AvailableFactors, "institutional")
	} else {
		response.MissingFactors = append(response.MissingFactors, "institutional")
	}

	if ic.NewsSentimentScore != nil {
		v := toFloat64(*ic.NewsSentimentScore)
		response.NewsSentimentScore = &v
		factorCount++
		response.AvailableFactors = append(response.AvailableFactors, "news_sentiment")
	} else {
		response.MissingFactors = append(response.MissingFactors, "news_sentiment")
	}

	if ic.TechnicalScore != nil {
		v := toFloat64(*ic.TechnicalScore)
		response.TechnicalScore = &v
		factorCount++
		response.AvailableFactors = append(response.AvailableFactors, "technical")
	} else {
		response.MissingFactors = append(response.MissingFactors, "technical")
	}

	// Phase 2: New factor scores
	if ic.EarningsRevisionsScore != nil {
		v := toFloat64(*ic.EarningsRevisionsScore)
		response.EarningsRevisionsScore = &v
		factorCount++
		response.AvailableFactors = append(response.AvailableFactors, "earnings_revisions")
	} else {
		response.MissingFactors = append(response.MissingFactors, "earnings_revisions")
	}

	if ic.HistoricalValueScore != nil {
		v := toFloat64(*ic.HistoricalValueScore)
		response.HistoricalValueScore = &v
		factorCount++
		response.AvailableFactors = append(response.AvailableFactors, "historical_value")
	} else {
		response.MissingFactors = append(response.MissingFactors, "historical_value")
	}

	if ic.DividendQualityScore != nil {
		v := toFloat64(*ic.DividendQualityScore)
		response.DividendQualityScore = &v
		factorCount++
		response.AvailableFactors = append(response.AvailableFactors, "dividend_quality")
	}
	// Note: dividend_quality is optional, don't add to missing factors

	response.FactorCount = factorCount

	if ic.SectorPercentile != nil {
		v := toFloat64(*ic.SectorPercentile)
		response.SectorPercentile = &v
	}

	// v2.1: Add lifecycle and sector context
	response.LifecycleStage = ic.LifecycleStage
	response.SectorRank = ic.SectorRank
	response.SectorTotal = ic.SectorTotal

	if ic.RawScore != nil {
		v := toFloat64(*ic.RawScore)
		response.RawScore = &v
	}

	// Determine scoring version
	if ic.LifecycleStage != nil {
		response.ScoringVersion = "2.1"
	} else {
		response.ScoringVersion = "2.0"
	}

	return response
}

// Helper functions
func toFloat64(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return f
}

func decimalToFloat64(d *decimal.Decimal, defaultVal float64) float64 {
	if d == nil {
		return defaultVal
	}
	f, _ := d.Float64()
	return f
}

func stringOrDefault(s *string, defaultVal string) string {
	if s == nil {
		return defaultVal
	}
	return *s
}
