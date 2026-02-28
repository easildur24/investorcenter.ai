package models

// ============================================================================
// Sector Percentiles Response
// ============================================================================

// PercentileDistribution represents the percentile distribution for a metric
type PercentileDistribution struct {
	Min float64 `json:"min"`
	P10 float64 `json:"p10"`
	P25 float64 `json:"p25"`
	P50 float64 `json:"p50"`
	P75 float64 `json:"p75"`
	P90 float64 `json:"p90"`
	Max float64 `json:"max"`
}

// MetricPercentileData represents a single metric's percentile info for a stock
type MetricPercentileData struct {
	Value         *float64                `json:"value"`
	Percentile    *float64                `json:"percentile"`
	LowerIsBetter bool                   `json:"lower_is_better"`
	Distribution  *PercentileDistribution `json:"distribution"`
	SampleCount   *int                   `json:"sample_count"`
}

// SectorPercentilesResponse is the full response for GET /stocks/:ticker/sector-percentiles
type SectorPercentilesResponse struct {
	Ticker       string                           `json:"ticker"`
	Sector       string                           `json:"sector"`
	CalculatedAt string                           `json:"calculated_at"`
	SampleCount  *int                             `json:"sample_count"`
	Metrics      map[string]*MetricPercentileData `json:"metrics"`
}

// ============================================================================
// Peers Response
// ============================================================================

// PeerMetrics holds the comparison metrics for a single stock
type PeerMetrics struct {
	PERatio          *float64 `json:"pe_ratio"`
	ROE              *float64 `json:"roe"`
	RevenueGrowthYoY *float64 `json:"revenue_growth_yoy"`
	NetMargin        *float64 `json:"net_margin"`
	DebtToEquity     *float64 `json:"debt_to_equity"`
	MarketCap        *float64 `json:"market_cap"`
}

// PeerData represents a single peer stock with its metrics
type PeerData struct {
	Ticker      string       `json:"ticker"`
	CompanyName string       `json:"company_name"`
	ICScore     *float64     `json:"ic_score,omitempty"`
	Industry    string       `json:"industry"`
	Metrics     *PeerMetrics `json:"metrics"`
}

// PeersResponse is the full response for GET /stocks/:ticker/peers
type PeersResponse struct {
	Ticker       string       `json:"ticker"`
	ICScore      *float64     `json:"ic_score,omitempty"`
	Industry     string       `json:"industry"`
	Peers        []PeerData   `json:"peers"`
	StockMetrics *PeerMetrics `json:"stock_metrics"`
	AvgPeerScore *float64     `json:"avg_peer_score,omitempty"`
	VsPeersDelta *float64     `json:"vs_peers_delta,omitempty"`
}

// ============================================================================
// Fair Value Response
// ============================================================================

// FairValueModel represents a single valuation model's estimate
type FairValueModel struct {
	FairValue     *float64               `json:"fair_value"`
	UpsidePercent *float64               `json:"upside_percent"`
	Confidence    string                 `json:"confidence"`
	Inputs        map[string]interface{} `json:"inputs,omitempty"`
}

// FVAnalystConsensus represents analyst target price consensus
type FVAnalystConsensus struct {
	TargetPrice   *float64 `json:"target_price"`
	UpsidePercent *float64 `json:"upside_percent"`
	NumAnalysts   *int     `json:"num_analysts,omitempty"`
}

// MarginOfSafety represents the overall valuation zone
type MarginOfSafety struct {
	AvgFairValue *float64 `json:"avg_fair_value"`
	Zone         string   `json:"zone"`
	Description  string   `json:"description"`
}

// FairValueResponse is the full response for GET /stocks/:ticker/fair-value
type FairValueResponse struct {
	Ticker            string                     `json:"ticker"`
	CurrentPrice      *float64                   `json:"current_price"`
	Models            map[string]*FairValueModel `json:"models"`
	FVAnalystConsensus  *FVAnalystConsensus           `json:"analyst_consensus,omitempty"`
	MarginOfSafety    *MarginOfSafety            `json:"margin_of_safety"`
	Suppressed        bool                       `json:"suppressed"`
	SuppressionReason *string                    `json:"suppression_reason,omitempty"`
}

// ============================================================================
// Health Summary Response
// ============================================================================

// HealthComponent represents a single health signal.
// Value and Max are float64 for consistent JSON serialization — int-valued signals
// (e.g. Piotroski F-Score 7/9) are represented as 7.0/9.0 on the wire.
type HealthComponent struct {
	Value          float64  `json:"value"`
	Max            *float64 `json:"max,omitempty"`
	Zone           string   `json:"zone,omitempty"`
	Interpretation string   `json:"interpretation"`
}

// HealthBadge represents the composite health assessment
type HealthBadge struct {
	Badge      string                      `json:"badge"`
	Score      float64                     `json:"score"`
	Components map[string]*HealthComponent `json:"components"`
}

// LifecycleInfo represents a stock's lifecycle stage
type LifecycleInfo struct {
	Stage        string `json:"stage"`
	Description  string `json:"description"`
	ClassifiedAt string `json:"classified_at"`
}

// StrengthConcern represents a strength or concern identified from percentile rankings
type StrengthConcern struct {
	Metric     string   `json:"metric"`
	Value      *float64 `json:"value"`
	Percentile *float64 `json:"percentile"`
	Message    string   `json:"message"`
}

// RedFlag represents a detected red flag
type RedFlag struct {
	ID             string   `json:"id"`
	Severity       string   `json:"severity"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	RelatedMetrics []string `json:"related_metrics"`
}

// HealthSummaryResponse is the full response for GET /stocks/:ticker/health-summary
type HealthSummaryResponse struct {
	Ticker    string            `json:"ticker"`
	Health    *HealthBadge      `json:"health"`
	Lifecycle *LifecycleInfo    `json:"lifecycle,omitempty"`
	Strengths []StrengthConcern `json:"strengths"`
	Concerns  []StrengthConcern `json:"concerns"`
	RedFlags  []RedFlag         `json:"red_flags"`
}

// ============================================================================
// Metric History Response
// ============================================================================

// MetricDataPoint represents a single period's metric value
type MetricDataPoint struct {
	PeriodEnd     string   `json:"period_end"`
	FiscalYear    int      `json:"fiscal_year"`
	FiscalQuarter *int     `json:"fiscal_quarter,omitempty"`
	Value         *float64 `json:"value"`
	YoYChange     *float64 `json:"yoy_change,omitempty"`
}

// MetricTrend summarizes the direction of a metric over time
type MetricTrend struct {
	Direction                 string   `json:"direction"`
	Slope                     *float64 `json:"slope,omitempty"`
	ConsecutiveGrowthQuarters int      `json:"consecutive_growth_quarters"`
}

// MetricHistoryResponse is the full response for GET /stocks/:ticker/metric-history/:metric
type MetricHistoryResponse struct {
	Ticker     string            `json:"ticker"`
	Metric     string            `json:"metric"`
	Timeframe  string            `json:"timeframe"`
	Unit       string            `json:"unit"`
	DataPoints []MetricDataPoint `json:"data_points"`
	Trend      *MetricTrend      `json:"trend,omitempty"`
}

// ============================================================================
// DB Row Types (used by database layer, not API responses)
// ============================================================================

// FairValueMetrics holds fair value data from fundamental_metrics_extended + valuation_ratios
type FairValueMetrics struct {
	Ticker           string   `db:"ticker"`
	DCFFairValue     *float64 `db:"dcf_fair_value"`
	EPVFairValue     *float64 `db:"epv_fair_value"`
	GrahamNumber     *float64 `db:"graham_number"`
	DCFUpsidePercent *float64 `db:"dcf_upside_percent"`
	WACC             *float64 `db:"wacc"`
	StockPrice       *float64 `db:"stock_price"`
}

// EnrichedPeer holds peer data enriched with IC Score and financial metrics
type EnrichedPeer struct {
	Symbol           string   `db:"symbol"`
	Name             string   `db:"name"`
	Industry         string   `db:"industry"`
	MarketCap        *float64 `db:"market_cap"`
	ICScore          *float64 `db:"ic_score"`
	PERatio          *float64 `db:"pe_ratio"`
	ROE              *float64 `db:"roe"`
	RevenueGrowthYoY *float64 `db:"revenue_growth_yoy"`
	NetMargin        *float64 `db:"net_margin"`
	DebtToEquity     *float64 `db:"debt_to_equity"`
}

// StockMetricsRow holds all available metrics for a stock from FME + valuation_ratios
type StockMetricsRow struct {
	GrossMargin      *float64 `db:"gross_margin"`
	OperatingMargin  *float64 `db:"operating_margin"`
	NetMargin        *float64 `db:"net_margin"`
	EBITDAMargin     *float64 `db:"ebitda_margin"`
	ROE              *float64 `db:"roe"`
	ROA              *float64 `db:"roa"`
	ROIC             *float64 `db:"roic"`
	RevenueGrowthYoY *float64 `db:"revenue_growth_yoy"`
	EPSGrowthYoY     *float64 `db:"eps_growth_yoy"`
	FCFGrowthYoY     *float64 `db:"fcf_growth_yoy"`
	EVToRevenue      *float64 `db:"ev_to_revenue"`
	EVToEBITDA       *float64 `db:"ev_to_ebitda"`
	EVToFCF          *float64 `db:"ev_to_fcf"`
	CurrentRatio     *float64 `db:"current_ratio"`
	QuickRatio       *float64 `db:"quick_ratio"`
	DebtToEquity     *float64 `db:"debt_to_equity"`
	InterestCoverage *float64 `db:"interest_coverage"`
	NetDebtToEBITDA  *float64 `db:"net_debt_to_ebitda"`
	DividendYield    *float64 `db:"dividend_yield"`
	PayoutRatio      *float64 `db:"payout_ratio"`
	PERatio          *float64 `db:"pe_ratio"`
	PBRatio          *float64 `db:"pb_ratio"`
	PSRatio          *float64 `db:"ps_ratio"`
	StockPrice       *float64 `db:"stock_price"`
}

// ToMap converts StockMetricsRow to a map keyed by metric name.
// StockPrice is intentionally excluded — it's not a comparable metric for percentile
// ranking and is returned separately via dedicated fields (e.g. FairValueResponse.CurrentPrice).
func (sm *StockMetricsRow) ToMap() map[string]*float64 {
	return map[string]*float64{
		"gross_margin":       sm.GrossMargin,
		"operating_margin":   sm.OperatingMargin,
		"net_margin":         sm.NetMargin,
		"ebitda_margin":      sm.EBITDAMargin,
		"roe":                sm.ROE,
		"roa":                sm.ROA,
		"roic":               sm.ROIC,
		"revenue_growth_yoy": sm.RevenueGrowthYoY,
		"eps_growth_yoy":     sm.EPSGrowthYoY,
		"fcf_growth_yoy":     sm.FCFGrowthYoY,
		"ev_to_revenue":      sm.EVToRevenue,
		"ev_to_ebitda":       sm.EVToEBITDA,
		"ev_to_fcf":          sm.EVToFCF,
		"current_ratio":      sm.CurrentRatio,
		"quick_ratio":        sm.QuickRatio,
		"debt_to_equity":     sm.DebtToEquity,
		"interest_coverage":  sm.InterestCoverage,
		"net_debt_to_ebitda": sm.NetDebtToEBITDA,
		"dividend_yield":     sm.DividendYield,
		"payout_ratio":       sm.PayoutRatio,
		"pe_ratio":           sm.PERatio,
		"pb_ratio":           sm.PBRatio,
		"ps_ratio":           sm.PSRatio,
	}
}

// MetricHistoryRow represents a single row from the metric history query
type MetricHistoryRow struct {
	PeriodEnd     string   `db:"period_end"`
	FiscalYear    int      `db:"fiscal_year"`
	FiscalQuarter *int     `db:"fiscal_quarter"`
	Value         *float64 `db:"value"`
}

// ============================================================================
// Metric-to-Statement Mapping (for metric history endpoint)
// ============================================================================

// MetricMapping maps a user-facing metric name to its financial statement source
type MetricMapping struct {
	StatementType string
	FieldName     string
	Unit          string
}

// MetricStatementMap maps metric names to their financial statement sources
var MetricStatementMap = map[string]MetricMapping{
	"revenue":          {StatementType: "income", FieldName: "revenues", Unit: "USD"},
	"net_income":       {StatementType: "income", FieldName: "net_income_loss", Unit: "USD"},
	"gross_profit":     {StatementType: "income", FieldName: "gross_profit", Unit: "USD"},
	"operating_income": {StatementType: "income", FieldName: "operating_income_loss", Unit: "USD"},
	"eps":              {StatementType: "income", FieldName: "diluted_earnings_per_share", Unit: "USD"},
	"gross_margin":     {StatementType: "ratios", FieldName: "gross_margin", Unit: "percent"},
	"operating_margin": {StatementType: "ratios", FieldName: "operating_margin", Unit: "percent"},
	"net_margin":       {StatementType: "ratios", FieldName: "net_profit_margin", Unit: "percent"},
	"roe":              {StatementType: "ratios", FieldName: "return_on_equity", Unit: "percent"},
	"roa":              {StatementType: "ratios", FieldName: "return_on_assets", Unit: "percent"},
	"debt_to_equity":   {StatementType: "ratios", FieldName: "debt_to_equity", Unit: "ratio"},
	"current_ratio":    {StatementType: "ratios", FieldName: "current_ratio", Unit: "ratio"},
}
