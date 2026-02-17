/**
 * TypeScript types for IC Score Backtest functionality
 */

export interface BacktestConfig {
  start_date: string;
  end_date: string;
  rebalance_frequency: 'daily' | 'weekly' | 'monthly' | 'quarterly';
  universe: 'sp500' | 'sp1500' | 'all';
  min_market_cap?: number;
  max_market_cap?: number;
  sectors?: string[];
  exclude_financials: boolean;
  exclude_utilities: boolean;
  transaction_cost_bps: number;
  slippage_bps: number;
  use_smoothed_scores: boolean;
  benchmark: string;
}

export interface DecilePerformance {
  decile: number;
  total_return: number;
  annualized_return: number;
  volatility: number;
  sharpe_ratio: number;
  max_drawdown: number;
  avg_score: number;
  num_periods: number;
}

export interface BacktestSummary {
  // Configuration
  start_date: string;
  end_date: string;
  rebalance_frequency: string;
  universe: string;
  benchmark: string;
  num_periods: number;

  // Key findings
  top_decile_cagr: number;
  bottom_decile_cagr: number;
  spread_cagr: number;
  benchmark_cagr: number;
  top_vs_benchmark: number;

  // Statistical validity
  hit_rate: number;
  monotonicity_score: number;
  information_ratio: number;

  // Risk metrics
  top_decile_sharpe: number;
  top_decile_max_dd: number;
  bottom_decile_sharpe: number;

  // Decile breakdown
  decile_performance: DecilePerformance[];
}

export interface BacktestJob {
  job_id: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  created_at: string;
  started_at?: string;
  completed_at?: string;
  error?: string;
  config?: Partial<BacktestConfig>;
}

export interface CumulativePoint {
  date: string;
  value: number;
  return: number;
}

export interface RollingMetricPoint {
  date: string;
  rolling_return: number;
  rolling_sharpe: number;
  rolling_volatility: number;
}

export interface BacktestDetailedReport {
  summary: BacktestSummary;
  period_data: Record<string, number | string>[];
  cumulative_returns: Record<string, CumulativePoint[]>;
  sector_analysis?: Record<string, Record<string, number>>;
  rolling_metrics?: Record<string, RollingMetricPoint[]>;
  statistical_tests?: {
    t_test?: {
      t_statistic: number;
      p_value: number;
      significant_5pct: boolean;
      significant_1pct: boolean;
    };
    wilcoxon_test?: {
      w_statistic: number;
      p_value: number;
      significant_5pct: boolean;
    };
    binomial_test?: {
      n_periods: number;
      n_hits: number;
      hit_rate: number;
      expected_if_random: number;
      excess_vs_random: number;
    };
  };
  generated_at: string;
}

export interface ChartDataset {
  label: string;
  data: number[];
  backgroundColor?: string | string[];
  borderColor?: string;
  borderDash?: number[];
  fill?: boolean;
}

export interface ChartData {
  labels: string[];
  datasets: ChartDataset[];
}

export interface BacktestCharts {
  decile_bar_chart: ChartData;
  cumulative_line_chart: ChartData;
  spread_chart: ChartData;
}

// Utility functions
export function formatPercent(value: number, decimals: number = 2): string {
  return `${(value * 100).toFixed(decimals)}%`;
}

export function formatNumber(value: number, decimals: number = 2): string {
  return value.toFixed(decimals);
}

export function getDecileColor(decile: number): string {
  // Green for top deciles, red for bottom
  const colors: Record<number, string> = {
    1: '#10b981', // Emerald
    2: '#34d399',
    3: '#6ee7b7',
    4: '#a7f3d0',
    5: '#d1fae5', // Light green
    6: '#fef3c7', // Light yellow
    7: '#fde68a',
    8: '#fca5a5',
    9: '#f87171',
    10: '#ef4444', // Red
  };
  return colors[decile] || '#6b7280';
}

export function getReturnColor(value: number): string {
  if (value > 0.15) return '#10b981'; // Strong positive
  if (value > 0.05) return '#34d399'; // Positive
  if (value > 0) return '#6ee7b7'; // Slight positive
  if (value > -0.05) return '#fca5a5'; // Slight negative
  if (value > -0.15) return '#f87171'; // Negative
  return '#ef4444'; // Strong negative
}

export function getRatingFromSharpe(sharpe: number): string {
  if (sharpe >= 2) return 'Excellent';
  if (sharpe >= 1.5) return 'Very Good';
  if (sharpe >= 1) return 'Good';
  if (sharpe >= 0.5) return 'Average';
  if (sharpe >= 0) return 'Below Average';
  return 'Poor';
}

export const DEFAULT_BACKTEST_CONFIG: BacktestConfig = {
  start_date: new Date(Date.now() - 5 * 365 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
  end_date: new Date().toISOString().split('T')[0],
  rebalance_frequency: 'monthly',
  universe: 'sp500',
  transaction_cost_bps: 10,
  slippage_bps: 5,
  use_smoothed_scores: true,
  benchmark: 'SPY',
  exclude_financials: false,
  exclude_utilities: false,
};
