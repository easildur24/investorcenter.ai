'use client';

import { useState, useEffect, useMemo } from 'react';
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';
import { API_BASE_URL } from '@/lib/api';
import { stocks } from '@/lib/api/routes';
import type { MetricHistoryResponse } from '@/lib/types/fundamentals';

interface MetricHistoryChartProps {
  ticker: string;
  metric: string;
  metricLabel: string;
  unit: 'USD' | 'percent' | 'ratio';
  initialData?: MetricHistoryResponse;
  onClose: () => void;
}

interface ChartDataPoint {
  label: string;
  value: number;
  yoyChange: number | null;
}

/** Format a quarter label like Q1'22 from fiscal year/quarter. */
function formatQuarterLabel(fy: number, fq: number): string {
  return `Q${fq}'${String(fy).slice(-2)}`;
}

/** Format a metric value for Y-axis and tooltip display. */
function formatMetricDisplay(value: number, unit: 'USD' | 'percent' | 'ratio'): string {
  if (unit === 'USD') {
    const abs = Math.abs(value);
    if (abs >= 1e12) return `$${(value / 1e12).toFixed(1)}T`;
    if (abs >= 1e9) return `$${(value / 1e9).toFixed(1)}B`;
    if (abs >= 1e6) return `$${(value / 1e6).toFixed(1)}M`;
    return `$${value.toFixed(2)}`;
  }
  if (unit === 'percent') return `${value.toFixed(2)}%`;
  return value.toFixed(2);
}

/** Calculate 5-year CAGR from first and last values. */
function calculateCAGR(first: number, last: number, years: number): number | null {
  if (first <= 0 || last <= 0 || years <= 0) return null;
  return (Math.pow(last / first, 1 / years) - 1) * 100;
}

/**
 * MetricHistoryChart — Full-width Recharts area chart for metric history.
 *
 * Displays quarterly data points with formatted axes, custom tooltip,
 * and summary statistics (latest value, YoY change, 5Y CAGR).
 */
export default function MetricHistoryChart({
  ticker,
  metric,
  metricLabel,
  unit,
  initialData,
  onClose,
}: MetricHistoryChartProps) {
  const [data, setData] = useState<MetricHistoryResponse | null>(initialData || null);
  const [loading, setLoading] = useState(!initialData);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (initialData) return;

    async function fetchHistory() {
      try {
        setLoading(true);
        const res = await fetch(`${API_BASE_URL}${stocks.metricHistory(ticker, metric)}?limit=20`);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const json = await res.json();
        setData(json.data || json);
      } catch (err) {
        console.error(`Error fetching ${metric} history:`, err);
        setError('Failed to load metric history');
      } finally {
        setLoading(false);
      }
    }

    fetchHistory();
  }, [ticker, metric, initialData]);

  const chartData: ChartDataPoint[] = useMemo(() => {
    if (!data?.data_points) return [];
    return data.data_points.map((dp) => ({
      label: formatQuarterLabel(dp.fiscal_year, dp.fiscal_quarter),
      value: dp.value,
      yoyChange: dp.yoy_change,
    }));
  }, [data]);

  const summaryStats = useMemo(() => {
    if (!data?.data_points || data.data_points.length === 0) {
      return { latest: null, yoyChange: null, cagr5y: null };
    }
    const points = data.data_points;
    const latest = points[points.length - 1];
    const first = points[0];
    const yearSpan = (points.length - 1) / 4; // approximate years from quarterly data
    const cagr5y = calculateCAGR(first.value, latest.value, yearSpan);

    return {
      latest: latest.value,
      yoyChange: latest.yoy_change,
      cagr5y,
    };
  }, [data]);

  // Determine area color based on trend
  const areaColor = useMemo(() => {
    if (!data?.trend) return '#6b7280';
    const dir = data.trend.direction;
    if (dir === 'up') return '#10b981';
    if (dir === 'down') return '#ef4444';
    return '#6b7280';
  }, [data]);

  if (loading) {
    return (
      <div className="mt-4 bg-ic-bg-secondary rounded-lg p-4 animate-pulse">
        <div className="h-48 bg-ic-bg-tertiary rounded" />
      </div>
    );
  }

  if (error || !data || chartData.length === 0) {
    return (
      <div className="mt-4 bg-ic-bg-secondary rounded-lg p-4">
        <div className="flex justify-between items-center mb-2">
          <span className="text-sm text-ic-text-muted">
            {error || 'No historical data available'}
          </span>
          <button
            onClick={onClose}
            className="text-ic-text-dim hover:text-ic-text-primary transition-colors p-1"
            aria-label="Close chart"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
              <path
                d="M4 4l8 8M12 4l-8 8"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
              />
            </svg>
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="mt-4 bg-ic-bg-secondary rounded-lg p-4">
      {/* Header with close button */}
      <div className="flex justify-between items-center mb-4">
        <h4 className="text-sm font-medium text-ic-text-primary">{metricLabel} History</h4>
        <button
          onClick={onClose}
          className="text-ic-text-dim hover:text-ic-text-primary transition-colors p-1"
          aria-label="Close chart"
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
            <path
              d="M4 4l8 8M12 4l-8 8"
              stroke="currentColor"
              strokeWidth="1.5"
              strokeLinecap="round"
            />
          </svg>
        </button>
      </div>

      {/* Chart */}
      <ResponsiveContainer width="100%" height={200}>
        <AreaChart data={chartData} margin={{ top: 5, right: 10, bottom: 5, left: 10 }}>
          <defs>
            <linearGradient id={`gradient-${metric}`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={areaColor} stopOpacity={0.2} />
              <stop offset="95%" stopColor={areaColor} stopOpacity={0} />
            </linearGradient>
          </defs>
          <XAxis
            dataKey="label"
            axisLine={false}
            tickLine={false}
            tick={{ fontSize: 11, fill: '#9ca3af' }}
            interval="preserveStartEnd"
          />
          <YAxis
            axisLine={false}
            tickLine={false}
            tick={{ fontSize: 11, fill: '#9ca3af' }}
            tickFormatter={(v: number) => formatMetricDisplay(v, unit)}
            width={60}
          />
          <Tooltip content={<MetricTooltip unit={unit} />} />
          <Area
            type="monotone"
            dataKey="value"
            stroke={areaColor}
            strokeWidth={2}
            fill={`url(#gradient-${metric})`}
            dot={false}
            activeDot={{ r: 4, fill: areaColor }}
          />
        </AreaChart>
      </ResponsiveContainer>

      {/* Summary stats */}
      <div className="flex gap-6 mt-4 pt-3 border-t border-ic-border">
        {summaryStats.latest !== null && (
          <div className="text-center">
            <div className="text-xs text-ic-text-muted">Latest</div>
            <div className="text-sm font-medium text-ic-text-primary">
              {formatMetricDisplay(summaryStats.latest, unit)}
            </div>
          </div>
        )}
        {summaryStats.yoyChange !== null && (
          <div className="text-center">
            <div className="text-xs text-ic-text-muted">YoY Change</div>
            <div
              className={`text-sm font-medium ${
                summaryStats.yoyChange >= 0 ? 'text-ic-positive' : 'text-ic-negative'
              }`}
            >
              {summaryStats.yoyChange >= 0 ? '+' : ''}
              {summaryStats.yoyChange.toFixed(1)}%
            </div>
          </div>
        )}
        {summaryStats.cagr5y !== null && (
          <div className="text-center">
            <div className="text-xs text-ic-text-muted">5Y CAGR</div>
            <div
              className={`text-sm font-medium ${
                summaryStats.cagr5y >= 0 ? 'text-ic-positive' : 'text-ic-negative'
              }`}
            >
              {summaryStats.cagr5y >= 0 ? '+' : ''}
              {summaryStats.cagr5y.toFixed(1)}%
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

/** Custom tooltip component for the area chart. */
function MetricTooltip({ active, payload, label, unit }: any) {
  if (!active || !payload || !payload.length) return null;

  const dataPoint = payload[0].payload as ChartDataPoint;

  return (
    <div className="bg-ic-bg-tertiary border border-ic-border rounded-lg shadow-lg p-3 text-sm">
      <div className="font-medium text-ic-text-primary mb-1">{label}</div>
      <div className="flex items-center justify-between gap-4">
        <span className="text-ic-text-muted">Value</span>
        <span className="font-semibold text-ic-text-primary">
          {formatMetricDisplay(dataPoint.value, unit)}
        </span>
      </div>
      {dataPoint.yoyChange !== null && (
        <div className="flex items-center justify-between gap-4">
          <span className="text-ic-text-muted">YoY</span>
          <span
            className={`font-medium ${
              dataPoint.yoyChange >= 0 ? 'text-ic-positive' : 'text-ic-negative'
            }`}
          >
            {dataPoint.yoyChange >= 0 ? '+' : ''}
            {dataPoint.yoyChange.toFixed(1)}%
          </span>
        </div>
      )}
    </div>
  );
}
