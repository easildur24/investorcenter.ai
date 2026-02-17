'use client';

import React, { useMemo } from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  ReferenceLine,
  Area,
  ComposedChart,
} from 'recharts';
import { format, parseISO } from 'date-fns';
import { getICScoreColor, getICScoreRating } from '@/lib/types/ic-score';
import type { ICScoreHistory } from '@/lib/types/ic-score';
import { useTheme } from '@/lib/contexts/ThemeContext';
import { getChartColors, themeColors } from '@/lib/theme';

interface ICScoreTrendChartProps {
  history: ICScoreHistory;
  height?: number;
  showStats?: boolean;
}

/**
 * Line chart showing IC Score historical trend
 *
 * Displays 30-day history with:
 * - Color-coded line based on score bands
 * - Reference lines for rating thresholds
 * - Min/max/average statistics
 */
export default function ICScoreTrendChart({
  history,
  height = 300,
  showStats = true,
}: ICScoreTrendChartProps) {
  const { resolvedTheme } = useTheme();
  const chartColors = useMemo(() => getChartColors(resolvedTheme), [resolvedTheme]);

  // Prepare data for chart
  const chartData = history.history.map((point) => ({
    date: parseISO(point.date),
    score: point.score,
    rating: point.rating,
  }));

  // Calculate statistics
  const currentScore = chartData[chartData.length - 1]?.score || 0;
  const previousScore = chartData[0]?.score || 0;
  const scoreChange = currentScore - previousScore;
  const scoreChangePercent = previousScore > 0 ? (scoreChange / previousScore) * 100 : 0;

  // Format date for tooltip
  const formatDate = (date: Date) => {
    return format(date, 'MMM dd, yyyy');
  };

  // Get line color based on current score
  const lineColor = getICScoreColor(currentScore);

  return (
    <div className="w-full">
      {/* Statistics */}
      {showStats && (
        <div className="mb-4 grid grid-cols-2 md:grid-cols-4 gap-4">
          <StatCard label="Current" value={currentScore.toFixed(1)} trend={scoreChange} />
          <StatCard label="Average" value={history.averageScore.toFixed(1)} />
          <StatCard label="High" value={history.maxScore.toFixed(1)} isPositive />
          <StatCard label="Low" value={history.minScore.toFixed(1)} isNegative />
        </div>
      )}

      {/* Chart */}
      <ResponsiveContainer width="100%" height={height} key={resolvedTheme}>
        <ComposedChart data={chartData} margin={{ top: 10, right: 30, left: 0, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke={chartColors.grid} />
          <XAxis
            dataKey="date"
            tickFormatter={(date) => format(date, 'MMM dd')}
            tick={{ fontSize: 12, fill: chartColors.text }}
            stroke={chartColors.text}
          />
          <YAxis
            domain={[0, 100]}
            tick={{ fontSize: 12, fill: chartColors.text }}
            stroke={chartColors.text}
          />
          <Tooltip content={<CustomTooltip />} />

          {/* Reference lines for rating thresholds */}
          <ReferenceLine
            y={80}
            stroke={themeColors.accent.positive}
            strokeDasharray="3 3"
            label={{ value: 'Strong Buy', fontSize: 10, fill: themeColors.accent.positive }}
          />
          <ReferenceLine
            y={65}
            stroke="#84cc16"
            strokeDasharray="3 3"
            label={{ value: 'Buy', fontSize: 10, fill: '#84cc16' }}
          />
          <ReferenceLine
            y={50}
            stroke={themeColors.accent.warning}
            strokeDasharray="3 3"
            label={{ value: 'Hold', fontSize: 10, fill: themeColors.accent.warning }}
          />
          <ReferenceLine
            y={35}
            stroke={themeColors.accent.orange}
            strokeDasharray="3 3"
            label={{ value: 'Underperform', fontSize: 10, fill: themeColors.accent.orange }}
          />

          {/* Area fill */}
          <defs>
            <linearGradient id="scoreGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={lineColor} stopOpacity={0.3} />
              <stop offset="95%" stopColor={lineColor} stopOpacity={0} />
            </linearGradient>
          </defs>
          <Area type="monotone" dataKey="score" fill="url(#scoreGradient)" stroke="none" />

          {/* Line */}
          <Line
            type="monotone"
            dataKey="score"
            stroke={lineColor}
            strokeWidth={2}
            dot={false}
            activeDot={{ r: 6 }}
          />
        </ComposedChart>
      </ResponsiveContainer>

      {/* Period info */}
      <div className="mt-2 text-xs text-ic-text-dim text-center">
        {formatDate(chartData[0]?.date)} - {formatDate(chartData[chartData.length - 1]?.date)}
      </div>
    </div>
  );
}

/**
 * Statistic card component
 */
interface StatCardProps {
  label: string;
  value: string;
  trend?: number;
  isPositive?: boolean;
  isNegative?: boolean;
}

function StatCard({ label, value, trend, isPositive, isNegative }: StatCardProps) {
  const getTrendColor = () => {
    if (trend === undefined) return 'text-ic-text-muted';
    return trend >= 0 ? 'text-ic-positive' : 'text-ic-negative';
  };

  const getValueColor = () => {
    if (isPositive) return 'text-ic-positive';
    if (isNegative) return 'text-ic-negative';
    return 'text-ic-text-primary';
  };

  return (
    <div className="bg-ic-surface rounded-lg p-3 border border-ic-border-subtle">
      <div className="text-xs text-ic-text-dim mb-1">{label}</div>
      <div className={`text-lg font-bold ${getValueColor()}`}>{value}</div>
      {trend !== undefined && (
        <div className={`text-xs ${getTrendColor()} flex items-center gap-1 mt-1`}>
          {trend >= 0 ? '↑' : '↓'} {Math.abs(trend).toFixed(1)}
        </div>
      )}
    </div>
  );
}

/**
 * Custom tooltip for trend chart
 */
interface TooltipProps {
  active?: boolean;
  payload?: Array<{
    payload: {
      date: Date;
      score: number;
      rating: string;
    };
  }>;
}

function CustomTooltip({ active, payload }: TooltipProps) {
  if (!active || !payload || payload.length === 0) {
    return null;
  }

  const data = payload[0].payload;

  return (
    <div className="bg-ic-surface border border-ic-border-subtle rounded-lg shadow-lg p-3">
      <p className="font-semibold text-ic-text-primary text-sm mb-2">
        {format(data.date, 'MMM dd, yyyy')}
      </p>
      <div className="space-y-1 text-xs">
        <div className="flex justify-between gap-4">
          <span className="text-ic-text-muted">IC Score:</span>
          <span className="font-bold text-ic-text-primary">{data.score.toFixed(1)}</span>
        </div>
        <div className="flex justify-between gap-4">
          <span className="text-ic-text-muted">Rating:</span>
          <span className="font-medium text-ic-text-primary">{data.rating}</span>
        </div>
      </div>
    </div>
  );
}

/**
 * Compact sparkline version (for widgets)
 */
interface ICScoreSparklineProps {
  history: ICScoreHistory;
  height?: number;
  showCurrentScore?: boolean;
}

export function ICScoreSparkline({
  history,
  height = 60,
  showCurrentScore = true,
}: ICScoreSparklineProps) {
  const { resolvedTheme } = useTheme();

  const chartData = history.history.map((point) => ({
    score: point.score,
  }));

  const currentScore = chartData[chartData.length - 1]?.score || 0;
  const lineColor = getICScoreColor(currentScore);

  return (
    <div className="flex items-center gap-2">
      <div className="flex-1">
        <ResponsiveContainer width="100%" height={height} key={resolvedTheme}>
          <LineChart data={chartData}>
            <Line type="monotone" dataKey="score" stroke={lineColor} strokeWidth={2} dot={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>
      {showCurrentScore && (
        <div className="text-right">
          <div className="text-2xl font-bold text-ic-text-primary">{currentScore.toFixed(0)}</div>
          <div className="text-xs text-ic-text-dim">IC Score</div>
        </div>
      )}
    </div>
  );
}
