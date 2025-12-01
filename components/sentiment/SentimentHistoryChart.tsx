'use client';

import { useState, useEffect } from 'react';
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
      <div className="flex items-center justify-center bg-gray-50 rounded-lg" style={{ height }}>
        <p className="text-gray-500 text-sm">
          {error || 'No sentiment history available'}
        </p>
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
        <h4 className="text-sm font-medium text-gray-700">Sentiment History</h4>
        <div className="flex gap-1">
          {[7, 30, 90].map((d) => (
            <button
              key={d}
              onClick={() => setDays(d)}
              className={`px-3 py-1 text-xs rounded-md transition-colors ${
                days === d
                  ? 'bg-ic-blue text-white'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
              }`}
            >
              {d}D
            </button>
          ))}
        </div>
      </div>

      {/* Chart */}
      <ResponsiveContainer width="100%" height={height}>
        <ComposedChart
          data={chartData}
          margin={{ top: 10, right: 10, left: -10, bottom: 0 }}
        >
          <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
          <XAxis
            dataKey="date"
            tickFormatter={(date) => format(date, 'MMM dd')}
            tick={{ fontSize: 11 }}
            stroke="#9ca3af"
          />
          <YAxis
            yAxisId="left"
            domain={[-1, 1]}
            tick={{ fontSize: 11 }}
            stroke="#9ca3af"
            tickFormatter={(value) => value.toFixed(1)}
          />
          {showPostCount && (
            <YAxis
              yAxisId="right"
              orientation="right"
              tick={{ fontSize: 11 }}
              stroke="#9ca3af"
            />
          )}
          <Tooltip content={<CustomTooltip />} />

          {/* Reference line at 0 */}
          <ReferenceLine
            yAxisId="left"
            y={0}
            stroke="#9ca3af"
            strokeDasharray="3 3"
          />

          {/* Reference lines for thresholds */}
          <ReferenceLine
            yAxisId="left"
            y={0.2}
            stroke="#22c55e"
            strokeDasharray="2 2"
            strokeOpacity={0.5}
          />
          <ReferenceLine
            yAxisId="left"
            y={-0.2}
            stroke="#ef4444"
            strokeDasharray="2 2"
            strokeOpacity={0.5}
          />

          {/* Post count bars */}
          {showPostCount && (
            <Bar
              yAxisId="right"
              dataKey="post_count"
              fill="#e5e7eb"
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
    <div className="bg-white border border-gray-200 rounded-lg shadow-lg p-3">
      <p className="font-medium text-gray-900 text-sm mb-2">
        {format(data.date, 'MMM dd, yyyy')}
      </p>
      <div className="space-y-1 text-xs">
        <div className="flex justify-between gap-4">
          <span className="text-gray-600">Sentiment:</span>
          <span className="font-bold" style={{ color: scoreColor }}>
            {data.score >= 0 ? '+' : ''}{data.score.toFixed(2)}
          </span>
        </div>
        <div className="flex justify-between gap-4">
          <span className="text-gray-600">Posts:</span>
          <span className="font-medium text-gray-900">{data.post_count}</span>
        </div>
        <div className="flex justify-between gap-4 pt-1 border-t border-gray-100">
          <span className="text-green-600">Bullish: {data.bullish}</span>
          <span className="text-red-600">Bearish: {data.bearish}</span>
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
        <div className="h-4 w-32 bg-gray-200 rounded" />
        <div className="flex gap-1">
          <div className="h-6 w-10 bg-gray-200 rounded" />
          <div className="h-6 w-10 bg-gray-200 rounded" />
          <div className="h-6 w-10 bg-gray-200 rounded" />
        </div>
      </div>
      <div className="bg-gray-200 rounded-lg" style={{ height }} />
    </div>
  );
}
