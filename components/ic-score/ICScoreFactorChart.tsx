'use client';

import React, { useMemo } from 'react';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, Cell } from 'recharts';
import { formatFactorName } from '@/lib/types/ic-score';
import type { ICScore, ICScoreFactor } from '@/lib/types/ic-score';
import { useTheme } from '@/lib/contexts/ThemeContext';
import { getChartColors, themeColors } from '@/lib/theme';

interface ICScoreFactorChartProps {
  factors: ICScore['factors'];
  height?: number;
}

/**
 * Horizontal bar chart showing IC Score factor breakdown
 *
 * Displays all 10 factors with:
 * - Factor name
 * - Weighted contribution to total score
 * - Color-coded bars based on performance
 */
export default function ICScoreFactorChart({ factors, height = 500 }: ICScoreFactorChartProps) {
  const { resolvedTheme } = useTheme();
  const chartColors = useMemo(() => getChartColors(resolvedTheme), [resolvedTheme]);

  // Convert factors object to array for charting
  const factorData = Object.entries(factors).map(([key, factor]) => ({
    name: formatFactorName(key),
    value: factor.value,
    weight: factor.weight,
    contribution: factor.contribution,
    fullName: key,
  }));

  // Sort by contribution (descending)
  factorData.sort((a, b) => b.contribution - a.contribution);

  // Get color based on factor value
  const getBarColor = (value: number): string => {
    if (value >= 80) return themeColors.accent.positive; // green
    if (value >= 65) return '#84cc16'; // lime-500
    if (value >= 50) return themeColors.accent.warning; // yellow
    if (value >= 35) return themeColors.accent.orange; // orange
    return themeColors.accent.negative; // red
  };

  return (
    <div className="w-full">
      <ResponsiveContainer width="100%" height={height} key={resolvedTheme}>
        <BarChart
          data={factorData}
          layout="vertical"
          margin={{ top: 10, right: 30, left: 20, bottom: 10 }}
        >
          <XAxis
            type="number"
            domain={[0, 100]}
            tick={{ fill: chartColors.text }}
            stroke={chartColors.text}
          />
          <YAxis
            type="category"
            dataKey="name"
            width={150}
            tick={{ fontSize: 13, fill: chartColors.text }}
            stroke={chartColors.text}
          />
          <Tooltip content={<CustomTooltip />} />
          <Bar dataKey="value" radius={[0, 4, 4, 0]}>
            {factorData.map((entry, index) => (
              <Cell key={`cell-${index}`} fill={getBarColor(entry.value)} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>

      {/* Legend */}
      <div className="mt-4 px-6">
        <div className="flex flex-wrap gap-4 text-xs text-ic-text-muted">
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded bg-green-500"></div>
            <span>Strong Buy (80-100)</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded bg-lime-500"></div>
            <span>Buy (65-79)</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded bg-yellow-500"></div>
            <span>Hold (50-64)</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded bg-orange-500"></div>
            <span>Underperform (35-49)</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded bg-red-500"></div>
            <span>Sell (0-34)</span>
          </div>
        </div>
      </div>
    </div>
  );
}

/**
 * Custom tooltip for factor chart
 */
interface TooltipProps {
  active?: boolean;
  payload?: Array<{
    payload: {
      name: string;
      value: number;
      weight: number;
      contribution: number;
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
      <p className="font-semibold text-ic-text-primary text-sm mb-2">{data.name}</p>
      <div className="space-y-1 text-xs text-ic-text-muted">
        <div className="flex justify-between gap-4">
          <span>Score:</span>
          <span className="font-medium text-ic-text-primary">{data.value.toFixed(1)}/100</span>
        </div>
        <div className="flex justify-between gap-4">
          <span>Weight:</span>
          <span className="font-medium text-ic-text-primary">{(data.weight * 100).toFixed(0)}%</span>
        </div>
        <div className="flex justify-between gap-4">
          <span>Contribution:</span>
          <span className="font-medium text-ic-text-primary">{data.contribution.toFixed(1)}</span>
        </div>
      </div>
    </div>
  );
}

/**
 * Compact factor list view (alternative to chart)
 */
interface ICScoreFactorListProps {
  factors: ICScore['factors'];
}

export function ICScoreFactorList({ factors }: ICScoreFactorListProps) {
  const factorData = Object.entries(factors).map(([key, factor]) => ({
    name: formatFactorName(key),
    value: factor.value,
    weight: factor.weight,
    contribution: factor.contribution,
  }));

  // Sort by contribution (descending)
  factorData.sort((a, b) => b.contribution - a.contribution);

  const getBarColor = (value: number): string => {
    if (value >= 80) return 'bg-ic-positive';
    if (value >= 65) return 'bg-lime-500';
    if (value >= 50) return 'bg-ic-warning';
    if (value >= 35) return 'bg-orange-500';
    return 'bg-ic-negative';
  };

  return (
    <div className="space-y-3">
      {factorData.map((factor, index) => (
        <div key={index} className="space-y-1">
          <div className="flex justify-between items-center text-sm">
            <span className="font-medium text-ic-text-secondary">{factor.name}</span>
            <span className="text-ic-text-muted">
              {factor.value.toFixed(1)} <span className="text-xs text-ic-text-muted">/ 100</span>
            </span>
          </div>
          <div className="w-full bg-ic-bg-tertiary rounded-full h-2">
            <div
              className={`h-2 rounded-full transition-all duration-500 ${getBarColor(
                factor.value
              )}`}
              style={{ width: `${factor.value}%` }}
            />
          </div>
          <div className="flex justify-between text-xs text-ic-text-dim">
            <span>Weight: {(factor.weight * 100).toFixed(0)}%</span>
            <span>Contributes: {factor.contribution.toFixed(1)}</span>
          </div>
        </div>
      ))}
    </div>
  );
}
