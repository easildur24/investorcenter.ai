'use client';

import { useState, useEffect, useMemo } from 'react';
import {
  LineChart,
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

interface SentimentHistoryChartProps {
  ticker: string;
  initialDays?: number;
  height?: number;
  showPostCount?: boolean;
}

/**
 * Line chart showing sentiment over time with optional post count overlay
 */
export default function SentimentHistoryChart({
  ticker,
  initialDays = 7,
  height = 300,
  showPostCount = true,
}: SentimentHistoryChartProps) {
  const { resolvedTheme } = useTheme();
  const chartColors = useMemo(() => getChartColors(resolvedTheme), [resolvedTheme]);

  const [data, setData] = useState<SentimentHistoryResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [days, setDays] = useState(initialDays);

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

  // Prepare chart data
  const chartData = data.history.map((point) => ({
    ...point,
    date: parseISO(point.date),
  }));

  // Get current score for color
  const currentScore = chartData[chartData.length - 1]?.score || 0;
  const lineColor = getSentimentScoreColor(currentScore);

  return (
    <div className="w-full">
      {/* Period selector */}
      <div className="flex items-center justify-between mb-4">
        <h4 className="text-sm font-medium text-ic-text-secondary">Sentiment History</h4>
        <div className="flex gap-1">
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
      <ResponsiveContainer width="100%" height={height} key={resolvedTheme}>
        <ComposedChart data={chartData} margin={{ top: 10, right: 10, left: -10, bottom: 0 }}>
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
          {showPostCount && (
            <YAxis
              yAxisId="right"
              orientation="right"
              tick={{ fontSize: 11, fill: chartColors.text }}
              stroke={chartColors.text}
            />
          )}
          <Tooltip content={<CustomTooltip />} />

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

          {/* Post count bars */}
          {showPostCount && (
            <Bar
              yAxisId="right"
              dataKey="post_count"
              fill={resolvedTheme === 'dark' ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.1)'}
              opacity={0.6}
              barSize={20}
            />
          )}

          {/* Area fill under line */}
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
        </ComposedChart>
      </ResponsiveContainer>
    </div>
  );
}

/**
 * Custom tooltip for the chart
 */
interface TooltipProps {
  active?: boolean;
  payload?: Array<{
    payload: SentimentHistoryPoint & { date: Date };
  }>;
}

function CustomTooltip({ active, payload }: TooltipProps) {
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
