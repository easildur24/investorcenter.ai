'use client';

import { useMemo } from 'react';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, Cell } from 'recharts';
import { useTheme } from '@/lib/contexts/ThemeContext';
import { getChartColors, themeColors } from '@/lib/theme';
import type { SubredditCount } from '@/lib/types/sentiment';

interface SubredditDistributionChartProps {
  subreddits: SubredditCount[];
  height?: number;
}

/**
 * Horizontal bar chart showing post distribution across subreddits.
 * Follows the ICScoreFactorChart pattern with theme-aware colors.
 */
export default function SubredditDistributionChart({
  subreddits,
  height = 200,
}: SubredditDistributionChartProps) {
  const { resolvedTheme } = useTheme();
  const chartColors = useMemo(() => getChartColors(resolvedTheme), [resolvedTheme]);

  if (!subreddits || subreddits.length === 0) {
    return null;
  }

  const totalPosts = subreddits.reduce((sum, s) => sum + s.count, 0);

  // Prepare chart data — prefix subreddit names with r/
  const chartData = subreddits.map((sub) => ({
    name: `r/${sub.subreddit}`,
    count: sub.count,
    percentage: totalPosts > 0 ? (sub.count / totalPosts) * 100 : 0,
  }));

  // Compute dynamic height: at least the default, but scale with number of subreddits
  const dynamicHeight = Math.max(height, chartData.length * 36 + 20);

  // Bar colors: top subreddit gets full orange, rest fade slightly
  const getBarColor = (index: number): string => {
    const baseOpacity = 1.0 - index * 0.12;
    const opacity = Math.max(baseOpacity, 0.4);
    return `rgba(249, 115, 22, ${opacity})`; // orange-500 with fading opacity
  };

  return (
    <div className="w-full">
      <ResponsiveContainer width="100%" height={dynamicHeight} key={resolvedTheme}>
        <BarChart
          data={chartData}
          layout="vertical"
          margin={{ top: 5, right: 40, left: 10, bottom: 5 }}
        >
          <XAxis
            type="number"
            tick={{ fontSize: 11, fill: chartColors.text }}
            stroke={chartColors.text}
            allowDecimals={false}
          />
          <YAxis
            type="category"
            dataKey="name"
            width={120}
            tick={{ fontSize: 12, fill: chartColors.text }}
            stroke={chartColors.text}
          />
          <Tooltip content={<SubredditTooltip totalPosts={totalPosts} />} />
          <Bar dataKey="count" radius={[0, 4, 4, 0]} barSize={18}>
            {chartData.map((_entry, index) => (
              <Cell key={`cell-${index}`} fill={getBarColor(index)} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}

/**
 * Custom tooltip for the subreddit chart
 */
interface SubredditTooltipProps {
  totalPosts: number;
  active?: boolean;
  payload?: Array<{
    payload: {
      name: string;
      count: number;
      percentage: number;
    };
  }>;
}

function SubredditTooltip({ totalPosts, active, payload }: SubredditTooltipProps) {
  if (!active || !payload || payload.length === 0) {
    return null;
  }

  const data = payload[0].payload;

  return (
    <div className="bg-ic-surface border border-ic-border-subtle rounded-lg shadow-lg p-3">
      <p className="font-semibold text-ic-text-primary text-sm mb-2">{data.name}</p>
      <div className="space-y-1 text-xs text-ic-text-muted">
        <div className="flex justify-between gap-4">
          <span>Posts:</span>
          <span className="font-medium text-ic-text-primary">{data.count}</span>
        </div>
        <div className="flex justify-between gap-4">
          <span>Share:</span>
          <span className="font-medium text-ic-text-primary">{data.percentage.toFixed(1)}%</span>
        </div>
        <div className="flex justify-between gap-4">
          <span>Total:</span>
          <span className="font-medium text-ic-text-primary">{totalPosts}</span>
        </div>
      </div>
    </div>
  );
}
