/**
 * Comprehensive Financial Metrics Types
 *
 * Type definitions for the /api/v1/stocks/:ticker/metrics endpoint
 * which provides TTM financial ratios, growth metrics, quality scores,
 * and forward estimates from Financial Modeling Prep (FMP) API.
 */

// ============================================================================
// Valuation Metrics
// ============================================================================

export interface ValuationMetrics {
  pe_ratio: number | null;
  forward_pe: number | null;
  pb_ratio: number | null;
  ps_ratio: number | null;
  price_to_fcf: number | null;
  price_to_ocf: number | null;
  peg_ratio: number | null;
  peg_interpretation: string | null;
  enterprise_value: number | null;
  ev_to_sales: number | null;
  ev_to_ebitda: number | null;
  ev_to_ebit: number | null;
  ev_to_fcf: number | null;
  earnings_yield: number | null;
  fcf_yield: number | null;
  market_cap: number | null;
}

// ============================================================================
// Profitability Metrics
// ============================================================================

export interface ProfitabilityMetrics {
  gross_margin: number | null;
  operating_margin: number | null;
  net_margin: number | null;
  ebitda_margin: number | null;
  ebit_margin: number | null;
  fcf_margin: number | null;
  pretax_margin: number | null;
  roe: number | null;
  roa: number | null;
  roic: number | null;
  roce: number | null;
}

// ============================================================================
// Liquidity Metrics
// ============================================================================

export interface LiquidityMetrics {
  current_ratio: number | null;
  quick_ratio: number | null;
  cash_ratio: number | null;
  working_capital: number | null;
}

// ============================================================================
// Leverage Metrics
// ============================================================================

export interface LeverageMetrics {
  debt_to_equity: number | null;
  debt_to_assets: number | null;
  debt_to_ebitda: number | null;
  debt_to_capital: number | null;
  interest_coverage: number | null;
  net_debt_to_ebitda: number | null;
  net_debt: number | null;
  invested_capital: number | null;
}

// ============================================================================
// Efficiency Metrics
// ============================================================================

export interface EfficiencyMetrics {
  asset_turnover: number | null;
  inventory_turnover: number | null;
  receivables_turnover: number | null;
  payables_turnover: number | null;
  fixed_asset_turnover: number | null;
  days_sales_outstanding: number | null;
  days_inventory_outstanding: number | null;
  days_payables_outstanding: number | null;
  cash_conversion_cycle: number | null;
}

// ============================================================================
// Growth Metrics
// ============================================================================

export interface GrowthMetrics {
  revenue_growth_yoy: number | null;
  revenue_growth_3y_cagr: number | null;
  revenue_growth_5y_cagr: number | null;
  gross_profit_growth_yoy: number | null;
  operating_income_growth_yoy: number | null;
  net_income_growth_yoy: number | null;
  eps_growth_yoy: number | null;
  eps_growth_3y_cagr: number | null;
  eps_growth_5y_cagr: number | null;
  fcf_growth_yoy: number | null;
  book_value_growth_yoy: number | null;
  dividend_growth_5y_cagr: number | null;
}

// ============================================================================
// Per Share Metrics
// ============================================================================

export interface PerShareMetrics {
  eps_diluted: number | null;
  book_value_per_share: number | null;
  tangible_book_per_share: number | null;
  revenue_per_share: number | null;
  operating_cf_per_share: number | null;
  fcf_per_share: number | null;
  cash_per_share: number | null;
  dividend_per_share: number | null;
  graham_number: number | null;
  interest_debt_per_share: number | null;
}

// ============================================================================
// Dividend Metrics
// ============================================================================

export interface DividendMetrics {
  dividend_yield: number | null;
  forward_dividend_yield: number | null;
  payout_ratio: number | null;
  payout_interpretation: string | null;
  fcf_payout_ratio: number | null;
  consecutive_dividend_years: number | null;
  ex_dividend_date: string | null;
  payment_date: string | null;
  dividend_frequency: string | null;
}

// ============================================================================
// Quality Scores
// ============================================================================

export interface QualityScores {
  altman_z_score: number | null;
  altman_z_interpretation: string | null;
  altman_z_description: string | null;
  piotroski_f_score: number | null;
  piotroski_f_interpretation: string | null;
  piotroski_f_description: string | null;
}

// ============================================================================
// Forward Estimates
// ============================================================================

export interface ForwardEstimates {
  forward_eps: number | null;
  forward_eps_high: number | null;
  forward_eps_low: number | null;
  forward_revenue: number | null;
  forward_ebitda: number | null;
  forward_net_income: number | null;
  num_analysts_eps: number | null;
  num_analysts_revenue: number | null;
}

// ============================================================================
// Data Sources (for admin debug mode)
// ============================================================================

export type DataSource = 'fmp' | 'database' | 'calculated' | '';

export interface FieldSources {
  pe_ratio?: DataSource;
  forward_pe?: DataSource;
  pb_ratio?: DataSource;
  ps_ratio?: DataSource;
  price_to_fcf?: DataSource;
  price_to_ocf?: DataSource;
  peg_ratio?: DataSource;
  ev_to_sales?: DataSource;
  ev_to_ebitda?: DataSource;
  ev_to_ebit?: DataSource;
  ev_to_fcf?: DataSource;
  earnings_yield?: DataSource;
  fcf_yield?: DataSource;
  gross_margin?: DataSource;
  operating_margin?: DataSource;
  net_margin?: DataSource;
  ebitda_margin?: DataSource;
  ebit_margin?: DataSource;
  fcf_margin?: DataSource;
  roe?: DataSource;
  roa?: DataSource;
  roic?: DataSource;
  roce?: DataSource;
  current_ratio?: DataSource;
  quick_ratio?: DataSource;
  cash_ratio?: DataSource;
  debt_to_equity?: DataSource;
  debt_to_assets?: DataSource;
  debt_to_ebitda?: DataSource;
  debt_to_capital?: DataSource;
  interest_coverage?: DataSource;
  net_debt?: DataSource;
  asset_turnover?: DataSource;
  inventory_turnover?: DataSource;
  receivables_turnover?: DataSource;
  payables_turnover?: DataSource;
  dso?: DataSource;
  dio?: DataSource;
  dpo?: DataSource;
  cash_conversion_cycle?: DataSource;
  altman_z_score?: DataSource;
  piotroski_f_score?: DataSource;
  revenue_growth_yoy?: DataSource;
  revenue_growth_3y?: DataSource;
  revenue_growth_5y?: DataSource;
  eps_growth_yoy?: DataSource;
  eps_growth_5y?: DataSource;
  dividend_yield?: DataSource;
  payout_ratio?: DataSource;
}

// ============================================================================
// Combined Metrics Response
// ============================================================================

export interface ComprehensiveMetricsData {
  valuation: ValuationMetrics;
  profitability: ProfitabilityMetrics;
  liquidity: LiquidityMetrics;
  leverage: LeverageMetrics;
  efficiency: EfficiencyMetrics;
  growth: GrowthMetrics;
  per_share: PerShareMetrics;
  dividends: DividendMetrics;
  quality_scores: QualityScores;
  forward_estimates: ForwardEstimates;
}

export interface ComprehensiveMetricsDebug {
  sources: FieldSources;
  errors: string[];
}

export interface ComprehensiveMetricsMeta {
  ticker: string;
  fmp_available: boolean;
  current_price: number;
}

export interface ComprehensiveMetricsResponse {
  data: ComprehensiveMetricsData;
  meta: ComprehensiveMetricsMeta;
  debug?: ComprehensiveMetricsDebug;
}

// ============================================================================
// Metric Display Configuration
// ============================================================================

export interface MetricDisplayConfig {
  key: string;
  label: string;
  format: 'number' | 'percent' | 'currency' | 'ratio' | 'score';
  decimals?: number;
  suffix?: string;
  tooltip?: string;
  invertColor?: boolean; // For metrics where lower is better
}

// ============================================================================
// Valuation Metrics Display Config
// ============================================================================

export const valuationMetricConfigs: MetricDisplayConfig[] = [
  { key: 'pe_ratio', label: 'P/E Ratio', format: 'ratio', decimals: 2, tooltip: 'Price to Earnings ratio - lower may indicate undervaluation' },
  { key: 'forward_pe', label: 'Forward P/E', format: 'ratio', decimals: 2, tooltip: 'P/E based on estimated future earnings' },
  { key: 'peg_ratio', label: 'PEG Ratio', format: 'ratio', decimals: 2, tooltip: 'P/E to Growth - <1 suggests undervalued relative to growth' },
  { key: 'pb_ratio', label: 'P/B Ratio', format: 'ratio', decimals: 2, tooltip: 'Price to Book value' },
  { key: 'ps_ratio', label: 'P/S Ratio', format: 'ratio', decimals: 2, tooltip: 'Price to Sales' },
  { key: 'price_to_fcf', label: 'P/FCF', format: 'ratio', decimals: 2, tooltip: 'Price to Free Cash Flow' },
  { key: 'ev_to_ebitda', label: 'EV/EBITDA', format: 'ratio', decimals: 2, tooltip: 'Enterprise Value to EBITDA' },
  { key: 'ev_to_sales', label: 'EV/Sales', format: 'ratio', decimals: 2, tooltip: 'Enterprise Value to Sales' },
  { key: 'earnings_yield', label: 'Earnings Yield', format: 'percent', decimals: 2, tooltip: 'Inverse of P/E - higher is better' },
  { key: 'fcf_yield', label: 'FCF Yield', format: 'percent', decimals: 2, tooltip: 'Free Cash Flow Yield - higher is better' },
  { key: 'market_cap', label: 'Market Cap', format: 'currency', decimals: 0, tooltip: 'Total market capitalization' },
  { key: 'enterprise_value', label: 'Enterprise Value', format: 'currency', decimals: 0, tooltip: 'Market Cap + Debt - Cash' },
];

// ============================================================================
// Profitability Metrics Display Config
// ============================================================================

export const profitabilityMetricConfigs: MetricDisplayConfig[] = [
  { key: 'gross_margin', label: 'Gross Margin', format: 'percent', decimals: 1, tooltip: 'Gross Profit / Revenue' },
  { key: 'operating_margin', label: 'Operating Margin', format: 'percent', decimals: 1, tooltip: 'Operating Income / Revenue' },
  { key: 'net_margin', label: 'Net Margin', format: 'percent', decimals: 1, tooltip: 'Net Income / Revenue' },
  { key: 'ebitda_margin', label: 'EBITDA Margin', format: 'percent', decimals: 1, tooltip: 'EBITDA / Revenue' },
  { key: 'fcf_margin', label: 'FCF Margin', format: 'percent', decimals: 1, tooltip: 'Free Cash Flow / Revenue' },
  { key: 'roe', label: 'ROE', format: 'percent', decimals: 1, tooltip: 'Return on Equity - Net Income / Shareholders Equity' },
  { key: 'roa', label: 'ROA', format: 'percent', decimals: 1, tooltip: 'Return on Assets - Net Income / Total Assets' },
  { key: 'roic', label: 'ROIC', format: 'percent', decimals: 1, tooltip: 'Return on Invested Capital' },
  { key: 'roce', label: 'ROCE', format: 'percent', decimals: 1, tooltip: 'Return on Capital Employed' },
];

// ============================================================================
// Financial Health Metrics Display Config
// ============================================================================

export const liquidityMetricConfigs: MetricDisplayConfig[] = [
  { key: 'current_ratio', label: 'Current Ratio', format: 'ratio', decimals: 2, tooltip: 'Current Assets / Current Liabilities - >1 is healthy' },
  { key: 'quick_ratio', label: 'Quick Ratio', format: 'ratio', decimals: 2, tooltip: '(Current Assets - Inventory) / Current Liabilities' },
  { key: 'cash_ratio', label: 'Cash Ratio', format: 'ratio', decimals: 2, tooltip: 'Cash / Current Liabilities' },
  { key: 'working_capital', label: 'Working Capital', format: 'currency', decimals: 0, tooltip: 'Current Assets - Current Liabilities' },
];

export const leverageMetricConfigs: MetricDisplayConfig[] = [
  { key: 'debt_to_equity', label: 'Debt/Equity', format: 'ratio', decimals: 2, tooltip: 'Total Debt / Shareholders Equity - lower is less risky', invertColor: true },
  { key: 'debt_to_assets', label: 'Debt/Assets', format: 'percent', decimals: 1, tooltip: 'Total Debt / Total Assets', invertColor: true },
  { key: 'debt_to_ebitda', label: 'Debt/EBITDA', format: 'ratio', decimals: 2, tooltip: 'Net Debt / EBITDA - <3 is generally healthy', invertColor: true },
  { key: 'interest_coverage', label: 'Interest Coverage', format: 'ratio', decimals: 2, tooltip: 'EBIT / Interest Expense - higher means more ability to pay interest' },
  { key: 'net_debt', label: 'Net Debt', format: 'currency', decimals: 0, tooltip: 'Total Debt - Cash & Equivalents' },
];

// ============================================================================
// Efficiency Metrics Display Config
// ============================================================================

export const efficiencyMetricConfigs: MetricDisplayConfig[] = [
  { key: 'asset_turnover', label: 'Asset Turnover', format: 'ratio', decimals: 2, tooltip: 'Revenue / Total Assets - higher is more efficient' },
  { key: 'inventory_turnover', label: 'Inventory Turnover', format: 'ratio', decimals: 2, tooltip: 'COGS / Average Inventory - higher is better' },
  { key: 'receivables_turnover', label: 'Receivables Turnover', format: 'ratio', decimals: 2, tooltip: 'Revenue / Average Receivables' },
  { key: 'payables_turnover', label: 'Payables Turnover', format: 'ratio', decimals: 2, tooltip: 'COGS / Average Payables' },
  { key: 'days_sales_outstanding', label: 'DSO', format: 'number', decimals: 0, suffix: ' days', tooltip: 'Days Sales Outstanding - lower is better', invertColor: true },
  { key: 'days_inventory_outstanding', label: 'DIO', format: 'number', decimals: 0, suffix: ' days', tooltip: 'Days Inventory Outstanding - lower is better', invertColor: true },
  { key: 'days_payables_outstanding', label: 'DPO', format: 'number', decimals: 0, suffix: ' days', tooltip: 'Days Payables Outstanding' },
  { key: 'cash_conversion_cycle', label: 'Cash Conversion Cycle', format: 'number', decimals: 0, suffix: ' days', tooltip: 'DSO + DIO - DPO - lower is better', invertColor: true },
];

// ============================================================================
// Growth Metrics Display Config
// ============================================================================

export const growthMetricConfigs: MetricDisplayConfig[] = [
  { key: 'revenue_growth_yoy', label: 'Revenue Growth (YoY)', format: 'percent', decimals: 1, tooltip: 'Year-over-year revenue growth' },
  { key: 'revenue_growth_3y_cagr', label: 'Revenue CAGR (3Y)', format: 'percent', decimals: 1, tooltip: '3-year compound annual revenue growth rate' },
  { key: 'revenue_growth_5y_cagr', label: 'Revenue CAGR (5Y)', format: 'percent', decimals: 1, tooltip: '5-year compound annual revenue growth rate' },
  { key: 'eps_growth_yoy', label: 'EPS Growth (YoY)', format: 'percent', decimals: 1, tooltip: 'Year-over-year earnings per share growth' },
  { key: 'eps_growth_5y_cagr', label: 'EPS CAGR (5Y)', format: 'percent', decimals: 1, tooltip: '5-year compound annual EPS growth rate' },
  { key: 'net_income_growth_yoy', label: 'Net Income Growth (YoY)', format: 'percent', decimals: 1, tooltip: 'Year-over-year net income growth' },
  { key: 'fcf_growth_yoy', label: 'FCF Growth (YoY)', format: 'percent', decimals: 1, tooltip: 'Year-over-year free cash flow growth' },
  { key: 'dividend_growth_5y_cagr', label: 'Dividend CAGR (5Y)', format: 'percent', decimals: 1, tooltip: '5-year compound annual dividend growth rate' },
];

// ============================================================================
// Per Share Metrics Display Config
// ============================================================================

export const perShareMetricConfigs: MetricDisplayConfig[] = [
  { key: 'eps_diluted', label: 'EPS (Diluted)', format: 'currency', decimals: 2, tooltip: 'Diluted Earnings Per Share' },
  { key: 'book_value_per_share', label: 'Book Value/Share', format: 'currency', decimals: 2, tooltip: 'Shareholders Equity / Shares Outstanding' },
  { key: 'tangible_book_per_share', label: 'Tangible Book/Share', format: 'currency', decimals: 2, tooltip: 'Book Value excluding intangibles' },
  { key: 'revenue_per_share', label: 'Revenue/Share', format: 'currency', decimals: 2, tooltip: 'Revenue Per Share' },
  { key: 'fcf_per_share', label: 'FCF/Share', format: 'currency', decimals: 2, tooltip: 'Free Cash Flow Per Share' },
  { key: 'cash_per_share', label: 'Cash/Share', format: 'currency', decimals: 2, tooltip: 'Cash Per Share' },
  { key: 'graham_number', label: 'Graham Number', format: 'currency', decimals: 2, tooltip: 'Fair value estimate based on Ben Graham formula' },
];

// ============================================================================
// Dividend Metrics Display Config
// ============================================================================

export const dividendMetricConfigs: MetricDisplayConfig[] = [
  { key: 'dividend_yield', label: 'Dividend Yield', format: 'percent', decimals: 2, tooltip: 'Annual Dividend / Share Price' },
  { key: 'payout_ratio', label: 'Payout Ratio', format: 'percent', decimals: 1, tooltip: 'Dividends / Net Income - lower is more sustainable' },
  { key: 'dividend_per_share', label: 'Dividend/Share', format: 'currency', decimals: 2, tooltip: 'Annual Dividend Per Share' },
  { key: 'consecutive_dividend_years', label: 'Dividend Streak', format: 'number', decimals: 0, suffix: ' years', tooltip: 'Years of consecutive dividend payments' },
  { key: 'dividend_frequency', label: 'Frequency', format: 'number', decimals: 0, tooltip: 'How often dividends are paid' },
];

// ============================================================================
// Quality Scores Display Config
// ============================================================================

export const qualityScoreConfigs: MetricDisplayConfig[] = [
  { key: 'altman_z_score', label: 'Altman Z-Score', format: 'score', decimals: 2, tooltip: 'Bankruptcy risk indicator: >2.99 Safe, 1.81-2.99 Grey Zone, <1.81 Distress' },
  { key: 'piotroski_f_score', label: 'Piotroski F-Score', format: 'score', decimals: 0, tooltip: 'Financial health score (0-9): >=8 Strong, 5-7 Average, <5 Weak' },
];

// ============================================================================
// Forward Estimates Display Config
// ============================================================================

export const forwardEstimateConfigs: MetricDisplayConfig[] = [
  { key: 'forward_eps', label: 'Forward EPS (Avg)', format: 'currency', decimals: 2, tooltip: 'Average analyst EPS estimate' },
  { key: 'forward_eps_high', label: 'Forward EPS (High)', format: 'currency', decimals: 2, tooltip: 'Highest analyst EPS estimate' },
  { key: 'forward_eps_low', label: 'Forward EPS (Low)', format: 'currency', decimals: 2, tooltip: 'Lowest analyst EPS estimate' },
  { key: 'forward_revenue', label: 'Forward Revenue', format: 'currency', decimals: 0, tooltip: 'Analyst revenue estimate' },
  { key: 'num_analysts_eps', label: 'Analysts (EPS)', format: 'number', decimals: 0, tooltip: 'Number of analysts providing EPS estimates' },
  { key: 'num_analysts_revenue', label: 'Analysts (Revenue)', format: 'number', decimals: 0, tooltip: 'Number of analysts providing revenue estimates' },
];

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Get interpretation color for Altman Z-Score
 */
export function getZScoreColor(interpretation: string | null): string {
  switch (interpretation) {
    case 'safe':
      return 'text-green-600 bg-green-50';
    case 'grey':
      return 'text-yellow-600 bg-yellow-50';
    case 'distress':
      return 'text-red-600 bg-red-50';
    default:
      return 'text-gray-600 bg-gray-50';
  }
}

/**
 * Get interpretation color for Piotroski F-Score
 */
export function getFScoreColor(interpretation: string | null): string {
  switch (interpretation) {
    case 'strong':
      return 'text-green-600 bg-green-50';
    case 'average':
      return 'text-yellow-600 bg-yellow-50';
    case 'weak':
      return 'text-red-600 bg-red-50';
    default:
      return 'text-gray-600 bg-gray-50';
  }
}

/**
 * Get interpretation color for PEG ratio
 */
export function getPEGColor(interpretation: string | null): string {
  switch (interpretation) {
    case 'undervalued':
      return 'text-green-600';
    case 'fair':
      return 'text-blue-600';
    case 'high':
      return 'text-yellow-600';
    case 'overvalued':
      return 'text-red-600';
    default:
      return 'text-gray-600';
  }
}

/**
 * Get interpretation color for payout ratio
 */
export function getPayoutColor(interpretation: string | null): string {
  switch (interpretation) {
    case 'very_safe':
      return 'text-green-600';
    case 'safe':
      return 'text-green-500';
    case 'moderate':
      return 'text-yellow-600';
    case 'at_risk':
      return 'text-red-600';
    default:
      return 'text-gray-600';
  }
}

/**
 * Get color for a numeric value based on whether higher or lower is better
 */
export function getValueColor(value: number | null, invertColor: boolean = false): string {
  if (value === null) return 'text-gray-500';

  if (invertColor) {
    // For metrics where lower is better (debt ratios, DSO, etc.)
    return value < 0 ? 'text-green-600' : 'text-red-600';
  }

  // For metrics where higher is better (margins, returns, etc.)
  return value >= 0 ? 'text-green-600' : 'text-red-600';
}

/**
 * Format a metric value based on its configuration
 */
export function formatMetricValue(
  value: number | null | undefined,
  config: MetricDisplayConfig
): string {
  if (value === null || value === undefined) return '—';

  const decimals = config.decimals ?? 2;
  const suffix = config.suffix ?? '';

  switch (config.format) {
    case 'percent':
      return `${value.toFixed(decimals)}%${suffix}`;
    case 'currency':
      if (Math.abs(value) >= 1e12) {
        return `$${(value / 1e12).toFixed(decimals)}T${suffix}`;
      }
      if (Math.abs(value) >= 1e9) {
        return `$${(value / 1e9).toFixed(decimals)}B${suffix}`;
      }
      if (Math.abs(value) >= 1e6) {
        return `$${(value / 1e6).toFixed(decimals)}M${suffix}`;
      }
      return `$${value.toFixed(decimals)}${suffix}`;
    case 'ratio':
    case 'number':
    case 'score':
    default:
      return `${value.toFixed(decimals)}${suffix}`;
  }
}
