/**
 * Shared TypeScript interfaces for fundamental metric history data.
 * Used by TrendSparkline, MetricHistoryChart, and useSparklineData.
 */

export interface MetricHistoryResponse {
  ticker: string;
  metric: string;
  timeframe: 'quarterly' | 'annual';
  unit: 'USD' | 'percent' | 'ratio';
  data_points: Array<{
    period_end: string;
    fiscal_year: number;
    fiscal_quarter: number;
    value: number;
    yoy_change: number | null;
  }>;
  trend: {
    direction: 'up' | 'down' | 'flat';
    slope: number;
    consecutive_growth_quarters: number;
  };
}

export interface SparklineMetricConfig {
  key: string;
  label: string;
  unit: 'USD' | 'percent' | 'ratio';
  higherIsBetter: boolean;
  surfaces: ('sidebar' | 'metricsTab')[];
}

export const SPARKLINE_METRICS: SparklineMetricConfig[] = [
  {
    key: 'revenue',
    label: 'Revenue',
    unit: 'USD',
    higherIsBetter: true,
    surfaces: ['sidebar', 'metricsTab'],
  },
  {
    key: 'net_income',
    label: 'Net Income',
    unit: 'USD',
    higherIsBetter: true,
    surfaces: ['metricsTab'],
  },
  {
    key: 'free_cash_flow',
    label: 'Free Cash Flow',
    unit: 'USD',
    higherIsBetter: true,
    surfaces: ['metricsTab'],
  },
  {
    key: 'gross_margin',
    label: 'Gross Margin',
    unit: 'percent',
    higherIsBetter: true,
    surfaces: ['sidebar', 'metricsTab'],
  },
  {
    key: 'operating_margin',
    label: 'Operating Margin',
    unit: 'percent',
    higherIsBetter: true,
    surfaces: ['metricsTab'],
  },
  {
    key: 'net_margin',
    label: 'Net Margin',
    unit: 'percent',
    higherIsBetter: true,
    surfaces: ['sidebar', 'metricsTab'],
  },
  {
    key: 'roe',
    label: 'ROE',
    unit: 'percent',
    higherIsBetter: true,
    surfaces: ['sidebar', 'metricsTab'],
  },
  { key: 'roa', label: 'ROA', unit: 'percent', higherIsBetter: true, surfaces: ['metricsTab'] },
  {
    key: 'debt_to_equity',
    label: 'Debt/Equity',
    unit: 'ratio',
    higherIsBetter: false,
    surfaces: ['sidebar', 'metricsTab'],
  },
  {
    key: 'eps',
    label: 'EPS',
    unit: 'USD',
    higherIsBetter: true,
    surfaces: ['sidebar', 'metricsTab'],
  },
  {
    key: 'current_ratio',
    label: 'Current Ratio',
    unit: 'ratio',
    higherIsBetter: true,
    surfaces: ['metricsTab'],
  },
];

export interface SparklineDataEntry {
  values: number[];
  trend: 'up' | 'down' | 'flat';
  latestValue: number;
  yoyChange: number | null;
  hoverData: Array<{ label: string; value: number }>;
  unit: 'USD' | 'percent' | 'ratio';
  consecutiveGrowthQuarters: number;
}

export type SparklineDataMap = Record<string, SparklineDataEntry>;
