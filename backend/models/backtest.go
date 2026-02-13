package models

import (
	"time"
)

// BacktestConfig represents configuration for a backtest run
type BacktestConfig struct {
	StartDate          string   `json:"start_date" binding:"required,datetime=2006-01-02"`
	EndDate            string   `json:"end_date" binding:"required,datetime=2006-01-02"`
	RebalanceFrequency string   `json:"rebalance_frequency" binding:"required,oneof=daily weekly monthly quarterly"`
	Universe           string   `json:"universe" binding:"required,oneof=sp500 sp1500 all"`
	MinMarketCap       *float64 `json:"min_market_cap,omitempty" binding:"omitempty,gte=0"`
	MaxMarketCap       *float64 `json:"max_market_cap,omitempty" binding:"omitempty,gte=0"`
	Sectors            []string `json:"sectors,omitempty" binding:"max=50,dive,max=100"`
	ExcludeFinancials  bool     `json:"exclude_financials"`
	ExcludeUtilities   bool     `json:"exclude_utilities"`
	TransactionCostBps float64  `json:"transaction_cost_bps" binding:"gte=0,lte=1000"`
	SlippageBps        float64  `json:"slippage_bps" binding:"gte=0,lte=1000"`
	UseSmoothedScores  bool     `json:"use_smoothed_scores"`
	Benchmark          string   `json:"benchmark" binding:"required,max=50"`
}

// DecilePerformance represents performance metrics for a single decile
type DecilePerformance struct {
	Decile           int     `json:"decile"`
	TotalReturn      float64 `json:"total_return"`
	AnnualizedReturn float64 `json:"annualized_return"`
	Volatility       float64 `json:"volatility"`
	SharpeRatio      float64 `json:"sharpe_ratio"`
	MaxDrawdown      float64 `json:"max_drawdown"`
	AvgScore         float64 `json:"avg_score"`
	NumPeriods       int     `json:"num_periods"`
}

// BacktestSummary represents high-level backtest results
type BacktestSummary struct {
	// Configuration
	StartDate          string `json:"start_date"`
	EndDate            string `json:"end_date"`
	RebalanceFrequency string `json:"rebalance_frequency"`
	Universe           string `json:"universe"`
	Benchmark          string `json:"benchmark"`
	NumPeriods         int    `json:"num_periods"`

	// Key findings
	TopDecileCAGR    float64 `json:"top_decile_cagr"`
	BottomDecileCAGR float64 `json:"bottom_decile_cagr"`
	SpreadCAGR       float64 `json:"spread_cagr"`
	BenchmarkCAGR    float64 `json:"benchmark_cagr"`
	TopVsBenchmark   float64 `json:"top_vs_benchmark"`

	// Statistical validity
	HitRate           float64 `json:"hit_rate"`
	MonotonicityScore float64 `json:"monotonicity_score"`
	InformationRatio  float64 `json:"information_ratio"`

	// Risk metrics
	TopDecileSharpe    float64 `json:"top_decile_sharpe"`
	TopDecileMaxDD     float64 `json:"top_decile_max_dd"`
	BottomDecileSharpe float64 `json:"bottom_decile_sharpe"`

	// Decile breakdown
	DecilePerformance []DecilePerformance `json:"decile_performance"`
}

// BacktestPeriodResult represents results for a single period
type BacktestPeriodResult struct {
	PeriodStart     string   `json:"period_start"`
	PeriodEnd       string   `json:"period_end"`
	Decile          int      `json:"decile"`
	Holdings        []string `json:"holdings"`
	NumHoldings     int      `json:"num_holdings"`
	PeriodReturn    float64  `json:"period_return"`
	BenchmarkReturn float64  `json:"benchmark_return"`
	ExcessReturn    float64  `json:"excess_return"`
	AvgScore        float64  `json:"avg_score"`
	Turnover        float64  `json:"turnover"`
}

// BacktestDetailedReport represents full backtest results with period data
type BacktestDetailedReport struct {
	Summary           BacktestSummary                 `json:"summary"`
	PeriodData        []map[string]interface{}        `json:"period_data"`
	CumulativeReturns map[string][]CumulativePoint    `json:"cumulative_returns"`
	SectorAnalysis    map[string]map[string]float64   `json:"sector_analysis,omitempty"`
	RollingMetrics    map[string][]RollingMetricPoint `json:"rolling_metrics,omitempty"`
	StatisticalTests  map[string]interface{}          `json:"statistical_tests,omitempty"`
	GeneratedAt       time.Time                       `json:"generated_at"`
}

// CumulativePoint represents a point on the cumulative returns chart
type CumulativePoint struct {
	Date   string  `json:"date"`
	Value  float64 `json:"value"`
	Return float64 `json:"return"`
}

// RollingMetricPoint represents rolling performance metrics at a point in time
type RollingMetricPoint struct {
	Date              string  `json:"date"`
	RollingReturn     float64 `json:"rolling_return"`
	RollingSharpe     float64 `json:"rolling_sharpe"`
	RollingVolatility float64 `json:"rolling_volatility"`
}

// BacktestStatus represents the status of a backtest job
type BacktestStatus string

const (
	BacktestStatusPending   BacktestStatus = "pending"
	BacktestStatusRunning   BacktestStatus = "running"
	BacktestStatusCompleted BacktestStatus = "completed"
	BacktestStatusFailed    BacktestStatus = "failed"
)

// BacktestJob represents a backtest job in the database
type BacktestJob struct {
	ID          string         `json:"id" gorm:"primaryKey"`
	UserID      *string        `json:"user_id,omitempty"`
	Config      string         `json:"config" gorm:"type:jsonb"` // JSON-encoded BacktestConfig
	Status      BacktestStatus `json:"status"`
	Result      *string        `json:"result,omitempty" gorm:"type:jsonb"` // JSON-encoded BacktestSummary
	Error       *string        `json:"error,omitempty"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// BacktestChartData represents data formatted for frontend backtest charts
type BacktestChartData struct {
	Labels   []string                 `json:"labels"`
	Datasets []map[string]interface{} `json:"datasets"`
}

// BacktestCharts represents all chart data for the backtest dashboard
type BacktestCharts struct {
	DecileBarChart      BacktestChartData `json:"decile_bar_chart"`
	CumulativeLineChart BacktestChartData `json:"cumulative_line_chart"`
	SpreadChart         BacktestChartData `json:"spread_chart"`
}
