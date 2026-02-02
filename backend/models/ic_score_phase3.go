package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// ===================
// Event Types
// ===================

// ScoreEventType represents types of events that can affect IC Scores
type ScoreEventType string

const (
	EventTypeEarningsRelease     ScoreEventType = "earnings_release"
	EventTypeAnalystRatingChange ScoreEventType = "analyst_rating_change"
	EventTypeInsiderTradeLarge   ScoreEventType = "insider_trade_large"
	EventTypeDividendAnnouncement ScoreEventType = "dividend_announcement"
	EventTypeAcquisitionNews     ScoreEventType = "acquisition_news"
	EventTypeGuidanceUpdate      ScoreEventType = "guidance_update"
	EventTypeTechnicalBreakout   ScoreEventType = "technical_breakout"
	EventType52WeekHigh          ScoreEventType = "52_week_high"
	EventType52WeekLow           ScoreEventType = "52_week_low"
)

// ImpactDirection represents the direction of impact
type ImpactDirection string

const (
	ImpactDirectionPositive ImpactDirection = "positive"
	ImpactDirectionNegative ImpactDirection = "negative"
	ImpactDirectionNeutral  ImpactDirection = "neutral"
)

// ===================
// IC Score Event Model
// ===================

// ICScoreEvent represents an event that may affect IC Score
type ICScoreEvent struct {
	ID              string           `json:"id" db:"id"`
	Ticker          string           `json:"ticker" db:"ticker"`
	EventType       ScoreEventType   `json:"event_type" db:"event_type"`
	EventDate       time.Time        `json:"event_date" db:"event_date"`
	Description     *string          `json:"description" db:"description"`
	ImpactDirection ImpactDirection  `json:"impact_direction" db:"impact_direction"`
	ImpactMagnitude *decimal.Decimal `json:"impact_magnitude" db:"impact_magnitude"`
	Source          *string          `json:"source" db:"source"`
	Metadata        map[string]any   `json:"metadata" db:"metadata"`
	CreatedAt       time.Time        `json:"created_at" db:"created_at"`
}

// ICScoreEventResponse represents the API response for an IC Score event
type ICScoreEventResponse struct {
	Ticker          string          `json:"ticker"`
	EventType       ScoreEventType  `json:"event_type"`
	EventDate       string          `json:"event_date"`
	Description     *string         `json:"description,omitempty"`
	ImpactDirection ImpactDirection `json:"impact_direction"`
	ImpactMagnitude *float64        `json:"impact_magnitude,omitempty"`
	Source          *string         `json:"source,omitempty"`
}

// ToResponse converts ICScoreEvent to API response
func (e *ICScoreEvent) ToResponse() ICScoreEventResponse {
	resp := ICScoreEventResponse{
		Ticker:          e.Ticker,
		EventType:       e.EventType,
		EventDate:       e.EventDate.Format("2006-01-02"),
		Description:     e.Description,
		ImpactDirection: e.ImpactDirection,
		Source:          e.Source,
	}

	if e.ImpactMagnitude != nil {
		v := toFloat64(*e.ImpactMagnitude)
		resp.ImpactMagnitude = &v
	}

	return resp
}

// ===================
// Stock Peers Model
// ===================

// StockPeer represents a peer relationship between two stocks
type StockPeer struct {
	ID                string           `json:"id" db:"id"`
	Ticker            string           `json:"ticker" db:"ticker"`
	PeerTicker        string           `json:"peer_ticker" db:"peer_ticker"`
	SimilarityScore   decimal.Decimal  `json:"similarity_score" db:"similarity_score"`
	SimilarityFactors map[string]any   `json:"similarity_factors" db:"similarity_factors"`
	CalculatedAt      time.Time        `json:"calculated_at" db:"calculated_at"`
	CreatedAt         time.Time        `json:"created_at" db:"created_at"`
}

// StockPeerResponse represents the API response for a stock peer
type StockPeerResponse struct {
	Ticker            string           `json:"ticker"`
	PeerTicker        string           `json:"peer_ticker"`
	CompanyName       string           `json:"company_name,omitempty"`
	ICScore           *float64         `json:"ic_score,omitempty"`
	SimilarityScore   float64          `json:"similarity_score"`
	SimilarityFactors map[string]any   `json:"similarity_factors,omitempty"`
}

// ToResponse converts StockPeer to API response
func (p *StockPeer) ToResponse() StockPeerResponse {
	return StockPeerResponse{
		Ticker:            p.Ticker,
		PeerTicker:        p.PeerTicker,
		SimilarityScore:   toFloat64(p.SimilarityScore),
		SimilarityFactors: p.SimilarityFactors,
	}
}

// PeerComparisonResponse represents peer comparison summary for a stock
type PeerComparisonResponse struct {
	Ticker        string               `json:"ticker"`
	ICScore       float64              `json:"ic_score"`
	Peers         []StockPeerResponse  `json:"peers"`
	AvgPeerScore  *float64             `json:"avg_peer_score"`
	VsPeersDelta  *float64             `json:"vs_peers_delta"`
	SectorRank    *int                 `json:"sector_rank"`
	SectorTotal   *int                 `json:"sector_total"`
}

// ===================
// Catalyst Event Model
// ===================

// CatalystType represents types of upcoming catalysts
type CatalystType string

const (
	CatalystTypeEarnings      CatalystType = "earnings"
	CatalystTypeExDividend    CatalystType = "ex_dividend"
	CatalystTypeAnalystDay    CatalystType = "analyst_day"
	CatalystTypeTechnical     CatalystType = "technical"
	CatalystType52WeekHigh    CatalystType = "52_week_high"
	CatalystType52WeekLow     CatalystType = "52_week_low"
	CatalystTypeInsiderTrade  CatalystType = "insider_trade"
	CatalystTypeAnalystRating CatalystType = "analyst_rating"
)

// CatalystImpact represents the expected impact of a catalyst
type CatalystImpact string

const (
	CatalystImpactPositive CatalystImpact = "Positive"
	CatalystImpactNegative CatalystImpact = "Negative"
	CatalystImpactNeutral  CatalystImpact = "Neutral"
	CatalystImpactUnknown  CatalystImpact = "Unknown"
)

// CatalystEvent represents an upcoming catalyst for a stock
type CatalystEvent struct {
	ID         string           `json:"id" db:"id"`
	Ticker     string           `json:"ticker" db:"ticker"`
	EventType  CatalystType     `json:"event_type" db:"event_type"`
	Title      string           `json:"title" db:"title"`
	EventDate  *time.Time       `json:"event_date" db:"event_date"`
	Icon       *string          `json:"icon" db:"icon"`
	Impact     CatalystImpact   `json:"impact" db:"impact"`
	Confidence *decimal.Decimal `json:"confidence" db:"confidence"`
	DaysUntil  *int             `json:"days_until" db:"days_until"`
	Metadata   map[string]any   `json:"metadata" db:"metadata"`
	CreatedAt  time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at" db:"updated_at"`
	ExpiresAt  *time.Time       `json:"expires_at" db:"expires_at"`
}

// CatalystEventResponse represents the API response for a catalyst event
type CatalystEventResponse struct {
	EventType  CatalystType   `json:"event_type"`
	Title      string         `json:"title"`
	EventDate  *string        `json:"event_date,omitempty"`
	Icon       *string        `json:"icon,omitempty"`
	Impact     CatalystImpact `json:"impact"`
	Confidence *float64       `json:"confidence,omitempty"`
	DaysUntil  *int           `json:"days_until,omitempty"`
}

// ToResponse converts CatalystEvent to API response
func (c *CatalystEvent) ToResponse() CatalystEventResponse {
	resp := CatalystEventResponse{
		EventType:  c.EventType,
		Title:      c.Title,
		Icon:       c.Icon,
		Impact:     c.Impact,
		DaysUntil:  c.DaysUntil,
	}

	if c.EventDate != nil {
		s := c.EventDate.Format("2006-01-02")
		resp.EventDate = &s
	}
	if c.Confidence != nil {
		v := toFloat64(*c.Confidence)
		resp.Confidence = &v
	}

	return resp
}

// ===================
// IC Score Change Model
// ===================

// FactorChange represents a change in a single factor score
type FactorChange struct {
	Factor       string  `json:"factor"`
	Delta        float64 `json:"delta"`
	Contribution float64 `json:"contribution"`
	Explanation  string  `json:"explanation"`
}

// ICScoreChange represents a tracked score change with explanation
type ICScoreChange struct {
	ID               string           `json:"id" db:"id"`
	Ticker           string           `json:"ticker" db:"ticker"`
	CalculatedAt     time.Time        `json:"calculated_at" db:"calculated_at"`
	PreviousScore    *decimal.Decimal `json:"previous_score" db:"previous_score"`
	CurrentScore     decimal.Decimal  `json:"current_score" db:"current_score"`
	Delta            *decimal.Decimal `json:"delta" db:"delta"`
	FactorChanges    []FactorChange   `json:"factor_changes" db:"factor_changes"`
	TriggerEvents    []string         `json:"trigger_events" db:"trigger_events"`
	SmoothingApplied bool             `json:"smoothing_applied" db:"smoothing_applied"`
	Summary          *string          `json:"summary" db:"summary"`
	CreatedAt        time.Time        `json:"created_at" db:"created_at"`
}

// ICScoreChangeResponse represents the API response for a score change
type ICScoreChangeResponse struct {
	Ticker           string          `json:"ticker"`
	CalculatedAt     string          `json:"calculated_at"`
	PreviousScore    *float64        `json:"previous_score,omitempty"`
	CurrentScore     float64         `json:"current_score"`
	Delta            *float64        `json:"delta,omitempty"`
	FactorChanges    []FactorChange  `json:"factor_changes"`
	TriggerEvents    []string        `json:"trigger_events,omitempty"`
	SmoothingApplied bool            `json:"smoothing_applied"`
	Summary          *string         `json:"summary,omitempty"`
}

// ToResponse converts ICScoreChange to API response
func (c *ICScoreChange) ToResponse() ICScoreChangeResponse {
	resp := ICScoreChangeResponse{
		Ticker:           c.Ticker,
		CalculatedAt:     c.CalculatedAt.Format("2006-01-02"),
		CurrentScore:     toFloat64(c.CurrentScore),
		FactorChanges:    c.FactorChanges,
		TriggerEvents:    c.TriggerEvents,
		SmoothingApplied: c.SmoothingApplied,
		Summary:          c.Summary,
	}

	if c.PreviousScore != nil {
		v := toFloat64(*c.PreviousScore)
		resp.PreviousScore = &v
	}
	if c.Delta != nil {
		v := toFloat64(*c.Delta)
		resp.Delta = &v
	}

	return resp
}

// ===================
// Score Explanation Model
// ===================

// ConfidenceLevel represents confidence in a score
type ConfidenceLevel string

const (
	ConfidenceLevelHigh   ConfidenceLevel = "High"
	ConfidenceLevelMedium ConfidenceLevel = "Medium"
	ConfidenceLevelLow    ConfidenceLevel = "Low"
)

// FactorDataStatus represents data availability status for a factor
type FactorDataStatus struct {
	Available     bool    `json:"available"`
	Freshness     string  `json:"freshness"` // 'fresh', 'recent', 'stale', 'missing'
	FreshnessDays *int    `json:"freshness_days,omitempty"`
	Count         *int    `json:"count,omitempty"`
	Warning       *string `json:"warning,omitempty"`
	Reason        *string `json:"reason,omitempty"`
}

// GranularConfidenceResponse represents granular confidence breakdown
type GranularConfidenceResponse struct {
	Level      ConfidenceLevel             `json:"level"`
	Percentage float64                     `json:"percentage"`
	Factors    map[string]FactorDataStatus `json:"factors"`
	Warnings   []string                    `json:"warnings"`
}

// ScoreExplanationResponse represents a complete score explanation
type ScoreExplanationResponse struct {
	Ticker        string                     `json:"ticker"`
	PreviousScore *float64                   `json:"previous_score,omitempty"`
	CurrentScore  float64                    `json:"current_score"`
	Delta         float64                    `json:"delta"`
	Reasons       []FactorChange             `json:"reasons"`
	Summary       string                     `json:"summary"`
	Confidence    GranularConfidenceResponse `json:"confidence"`
}

// ===================
// Extended IC Score Response with Phase 3 Data
// ===================

// ICScoreV3Response extends ICScoreResponse with Phase 3 features
type ICScoreV3Response struct {
	// Base IC Score fields
	Ticker           string   `json:"ticker"`
	Date             string   `json:"date"`
	OverallScore     float64  `json:"overall_score"`
	PreviousScore    *float64 `json:"previous_score,omitempty"`
	RawScore         *float64 `json:"raw_score,omitempty"`
	SmoothingApplied bool     `json:"smoothing_applied"`
	Rating           string   `json:"rating"`
	ConfidenceLevel  string   `json:"confidence_level"`
	DataCompleteness float64  `json:"data_completeness"`

	// Factor scores
	ValueScore            *float64 `json:"value_score,omitempty"`
	GrowthScore           *float64 `json:"growth_score,omitempty"`
	ProfitabilityScore    *float64 `json:"profitability_score,omitempty"`
	FinancialHealthScore  *float64 `json:"financial_health_score,omitempty"`
	MomentumScore         *float64 `json:"momentum_score,omitempty"`
	TechnicalScore        *float64 `json:"technical_score,omitempty"`
	AnalystConsensusScore *float64 `json:"analyst_consensus_score,omitempty"`
	InsiderActivityScore  *float64 `json:"insider_activity_score,omitempty"`
	InstitutionalScore    *float64 `json:"institutional_score,omitempty"`
	NewsSentimentScore    *float64 `json:"news_sentiment_score,omitempty"`

	// Phase 2 factor scores
	EarningsRevisionsScore *float64 `json:"earnings_revisions_score,omitempty"`
	HistoricalValueScore   *float64 `json:"historical_value_score,omitempty"`
	DividendQualityScore   *float64 `json:"dividend_quality_score,omitempty"`

	// v2.1 fields
	LifecycleStage   *string  `json:"lifecycle_stage,omitempty"`
	SectorRank       *int     `json:"sector_rank,omitempty"`
	SectorTotal      *int     `json:"sector_total,omitempty"`
	SectorPercentile *float64 `json:"sector_percentile,omitempty"`

	// Phase 3: Peer comparison
	Peers          []StockPeerResponse `json:"peers,omitempty"`
	PeerComparison *struct {
		AvgPeerScore *float64 `json:"avg_peer_score,omitempty"`
		VsPeersDelta *float64 `json:"vs_peers_delta,omitempty"`
		SectorRank   *int     `json:"sector_rank,omitempty"`
		SectorTotal  *int     `json:"sector_total,omitempty"`
	} `json:"peer_comparison,omitempty"`

	// Phase 3: Catalysts
	Catalysts []CatalystEventResponse `json:"catalysts,omitempty"`

	// Phase 3: Explanation
	Explanation *struct {
		Summary string         `json:"summary"`
		Delta   float64        `json:"delta"`
		Reasons []FactorChange `json:"reasons"`
		Confidence struct {
			Level      string   `json:"level"`
			Percentage float64  `json:"percentage"`
			Warnings   []string `json:"warnings"`
		} `json:"confidence"`
	} `json:"explanation,omitempty"`
}

// ===================
// Score Settings Model
// ===================

// ICScoreSetting represents a configurable score setting
type ICScoreSetting struct {
	ID           string         `json:"id" db:"id"`
	SettingKey   string         `json:"setting_key" db:"setting_key"`
	SettingValue map[string]any `json:"setting_value" db:"setting_value"`
	Description  *string        `json:"description" db:"description"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
}

// GetFloat64Value extracts a float64 value from setting_value
func (s *ICScoreSetting) GetFloat64Value() *float64 {
	if v, ok := s.SettingValue["value"].(float64); ok {
		return &v
	}
	return nil
}

// GetStringSlice extracts a string slice from setting_value
func (s *ICScoreSetting) GetStringSlice(key string) []string {
	if v, ok := s.SettingValue[key].([]any); ok {
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return nil
}
