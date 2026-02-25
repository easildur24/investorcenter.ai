'use client';

import { useState, useEffect, useMemo } from 'react';
import {
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  ReferenceLine,
  Bar,
  ComposedChart,
  Area,
} from 'recharts';
import { format, parseISO } from 'date-fns';
import { getSentimentHistory } from '@/lib/api/sentiment';
import { getSentimentScoreColor } from '@/lib/types/sentiment';
import type { SentimentHistoryResponse, SentimentHistoryPoint } from '@/lib/types/sentiment';
import { useTheme } from '@/lib/contexts/ThemeContext';
import { getChartColors, themeColors } from '@/lib/theme';
import { apiClient } from '@/lib/api/client';
import { tickers } from '@/lib/api/routes';

interface SentimentHistoryChartProps {
  ticker: string;
  initialDays?: number;
  height?: number;
  showPostCount?: boolean;
  showPriceOverlay?: boolean;
}

/** Map sentiment days to chart API period */
const DAYS_TO_CHART_PERIOD: Record<number, string> = {
  7: '1M',
  30: '3M',
  90: '6M',
};

interface ChartDataPoint {
  timestamp: string;
  close: string;
}

/**
 * Line chart showing sentiment over time with optional post count overlay
 * and optional stock price overlay.
 */
export default function SentimentHistoryChart({
  ticker,
  initialDays = 7,
  height = 300,
  showPostCount = true,
  showPriceOverlay = false,
}: SentimentHistoryChartProps) {
  const { resolvedTheme } = useTheme();
  const chartColors = useMemo(() => getChartColors(resolvedTheme), [resolvedTheme]);

  const [data, setData] = useState<SentimentHistoryResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [days, setDays] = useState(initialDays);
  const [priceEnabled, setPriceEnabled] = useState(showPriceOverlay);
  const [priceData, setPriceData] = useState<Map<string, number> | null>(null);

  // Fetch sentiment history
  useEffect(() => {
    async function fetchHistory() {
      try {
        setLoading(true);
        setError(null);
        const result = await getSentimentHistory(ticker, days);
        setData(result);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load history');
      } finally {
        setLoading(false);
      }
    }

    fetchHistory();
  }, [ticker, days]);

  // Fetch price data when price overlay is enabled
  useEffect(() => {
    if (!priceEnabled) {
      setPriceData(null);
      return;
    }

    async function fetchPriceData() {
      try {
        const period = DAYS_TO_CHART_PERIOD[days] || '3M';
        const result = await apiClient.get<{ dataPoints: ChartDataPoint[] }>(
          `${tickers.chart(ticker)}?period=${period}`
        );

        if (result?.dataPoints) {
          // Build a date→price map keyed by YYYY-MM-DD
          const priceMap = new Map<string, number>();
          for (const point of result.dataPoints) {
            const dateKey = point.timestamp.slice(0, 10); // YYYY-MM-DD
            priceMap.set(dateKey, parseFloat(point.close));
          }
          setPriceData(priceMap);
        }
      } catch {
        // Price overlay is optional; silently fail
        setPriceData(null);
      }
    }

    fetchPriceData();
  }, [ticker, days, priceEnabled]);

  if (loading) {
    return <LoadingSkeleton height={height} />;
  }

  if (error || !data || data.history.length === 0) {
    return (
      <div className="flex items-center justify-center bg-ic-surface rounded-lg" style={{ height }}>
        <p className="text-ic-text-dim text-sm">{error || 'No sentiment history available'}</p>
      </div>
    );
  }

  // Prepare chart data — merge sentiment with optional price
  const chartData = data.history.map((point) => {
    const price = priceEnabled && priceData ? (priceData.get(point.date) ?? null) : null;
    return {
      ...point,
      date: parseISO(point.date),
      price,
    };
  });

  // Get current score for color
  const currentScore = chartData[chartData.length - 1]?.score || 0;
  const lineColor = getSentimentScoreColor(currentScore);

  // Check if we have any price data to show
  const hasPriceData = priceEnabled && chartData.some((d) => d.price !== null);

  return (
    <div className="w-full">
      {/* Period selector + price toggle */}
      <div className="flex items-center justify-between mb-4">
        <h4 className="text-sm font-medium text-ic-text-secondary">Sentiment History</h4>
        <div className="flex gap-1">
          {/* Price overlay toggle */}
          <button
            onClick={() => setPriceEnabled(!priceEnabled)}
            className={`px-3 py-1 text-xs rounded-md transition-colors mr-2 ${
              priceEnabled
                ? 'bg-ic-blue text-ic-text-primary'
                : 'bg-ic-surface text-ic-text-muted hover:bg-ic-surface-hover'
            }`}
          >
            Price
          </button>
          {/* Period buttons */}
          {[7, 30, 90].map((d) => (
            <button
              key={d}
              onClick={() => setDays(d)}
              className={`px-3 py-1 text-xs rounded-md transition-colors ${
                days === d
                  ? 'bg-ic-blue text-ic-text-primary'
                  : 'bg-ic-surface text-ic-text-muted hover:bg-ic-surface-hover'
              }`}
            >
              {d}D
            </button>
          ))}
        </div>
      </div>

      {/* Chart */}
      <ResponsiveContainer width="100%" height={height} key={`${resolvedTheme}-${priceEnabled}`}>
        <ComposedChart
          data={chartData}
          margin={{ top: 10, right: hasPriceData ? 50 : 10, left: -10, bottom: 0 }}
        >
          <CartesianGrid strokeDasharray="3 3" stroke={chartColors.grid} />
          <XAxis
            dataKey="date"
            tickFormatter={(date) => format(date, 'MMM dd')}
            tick={{ fontSize: 11, fill: chartColors.text }}
            stroke={chartColors.text}
          />
          <YAxis
            yAxisId="left"
            domain={[-1, 1]}
            tick={{ fontSize: 11, fill: chartColors.text }}
            stroke={chartColors.text}
            tickFormatter={(value) => value.toFixed(1)}
          />
          {showPostCount && !hasPriceData && (
            <YAxis
              yAxisId="right"
              orientation="right"
              tick={{ fontSize: 11, fill: chartColors.text }}
              stroke={chartColors.text}
            />
          )}
          {hasPriceData && (
            <YAxis
              yAxisId="price"
              orientation="right"
              tick={{ fontSize: 11, fill: chartColors.text }}
              stroke={chartColors.line}
              tickFormatter={(value) => `$${value.toFixed(0)}`}
              domain={['auto', 'auto']}
            />
          )}
          <Tooltip content={<CustomTooltip showPrice={hasPriceData} />} />

          {/* Reference line at 0 */}
          <ReferenceLine yAxisId="left" y={0} stroke={chartColors.text} strokeDasharray="3 3" />

          {/* Reference lines for thresholds */}
          <ReferenceLine
            yAxisId="left"
            y={0.2}
            stroke={chartColors.positive}
            strokeDasharray="2 2"
            strokeOpacity={0.5}
          />
          <ReferenceLine
            yAxisId="left"
            y={-0.2}
            stroke={chartColors.negative}
            strokeDasharray="2 2"
            strokeOpacity={0.5}
          />

          {/* Post count bars (hidden when price overlay is active) */}
          {showPostCount && !hasPriceData && (
            <Bar
              yAxisId="right"
              dataKey="post_count"
              fill={resolvedTheme === 'dark' ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.1)'}
              opacity={0.6}
              barSize={20}
            />
          )}

          {/* Area fill under sentiment line */}
          <defs>
            <linearGradient id="sentimentGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={lineColor} stopOpacity={0.2} />
              <stop offset="95%" stopColor={lineColor} stopOpacity={0} />
            </linearGradient>
          </defs>
          <Area
            yAxisId="left"
            type="monotone"
            dataKey="score"
            fill="url(#sentimentGradient)"
            stroke="none"
          />

          {/* Sentiment line */}
          <Line
            yAxisId="left"
            type="monotone"
            dataKey="score"
            stroke={lineColor}
            strokeWidth={2}
            dot={false}
            activeDot={{ r: 5 }}
          />

          {/* Price overlay line */}
          {hasPriceData && (
            <Line
              yAxisId="price"
              type="monotone"
              dataKey="price"
              stroke={chartColors.line}
              strokeWidth={1.5}
              strokeDasharray="4 4"
              dot={false}
              activeDot={{ r: 4, fill: chartColors.line }}
              connectNulls={true}
            />
          )}
        </ComposedChart>
      </ResponsiveContainer>

      {/* Legend */}
      {hasPriceData && (
        <div className="flex items-center justify-center gap-6 mt-3 text-xs text-ic-text-muted">
          <div className="flex items-center gap-1.5">
            <div className="w-4 h-0.5 rounded" style={{ backgroundColor: lineColor }} />
            <span>Sentiment</span>
          </div>
          <div className="flex items-center gap-1.5">
            <div
              className="w-4 h-0.5 rounded"
              style={{
                backgroundColor: chartColors.line,
                backgroundImage: `repeating-linear-gradient(90deg, ${chartColors.line} 0px, ${chartColors.line} 4px, transparent 4px, transparent 8px)`,
              }}
            />
            <span>Price</span>
          </div>
        </div>
      )}
    </div>
  );
}

/**
 * Custom tooltip for the chart
 */
interface CustomTooltipProps {
  showPrice: boolean;
  active?: boolean;
  payload?: Array<{
    payload: SentimentHistoryPoint & { date: Date; price: number | null };
  }>;
}

function CustomTooltip({ showPrice, active, payload }: CustomTooltipProps) {
  if (!active || !payload || payload.length === 0) {
    return null;
  }

  const data = payload[0].payload;
  const scoreColor = getSentimentScoreColor(data.score);

  return (
    <div className="bg-ic-surface border border-ic-border-subtle rounded-lg shadow-lg p-3">
      <p className="font-medium text-ic-text-primary text-sm mb-2">
        {format(data.date, 'MMM dd, yyyy')}
      </p>
      <div className="space-y-1 text-xs">
        <div className="flex justify-between gap-4">
          <span className="text-ic-text-muted">Sentiment:</span>
          <span className="font-bold" style={{ color: scoreColor }}>
            {data.score >= 0 ? '+' : ''}
            {data.score.toFixed(2)}
          </span>
        </div>
        {showPrice && data.price !== null && (
          <div className="flex justify-between gap-4">
            <span className="text-ic-text-muted">Price:</span>
            <span className="font-medium" style={{ color: themeColors.accent.blue }}>
              ${data.price.toFixed(2)}
            </span>
          </div>
        )}
        <div className="flex justify-between gap-4">
          <span className="text-ic-text-muted">Posts:</span>
          <span className="font-medium text-ic-text-primary">{data.post_count}</span>
        </div>
        <div className="flex justify-between gap-4 pt-1 border-t border-ic-border-subtle">
          <span className="text-ic-positive">Bullish: {data.bullish}</span>
          <span className="text-ic-negative">Bearish: {data.bearish}</span>
        </div>
      </div>
    </div>
  );
}

/**
 * Loading skeleton
 */
function LoadingSkeleton({ height }: { height: number }) {
  return (
    <div className="animate-pulse">
      <div className="flex justify-between mb-4">
        <div className="h-4 w-32 bg-ic-bg-tertiary rounded" />
        <div className="flex gap-1">
          <div className="h-6 w-10 bg-ic-bg-tertiary rounded" />
          <div className="h-6 w-10 bg-ic-bg-tertiary rounded" />
          <div className="h-6 w-10 bg-ic-bg-tertiary rounded" />
        </div>
      </div>
      <div className="bg-ic-bg-tertiary rounded-lg" style={{ height }} />
    </div>
  );
}
