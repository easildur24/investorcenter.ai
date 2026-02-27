'use client';

import { useMemo, useState } from 'react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
  ReferenceLine,
} from 'recharts';
import { useTheme } from '@/lib/contexts/ThemeContext';
import { getChartColors } from '@/lib/theme';
import type { EarningsResult } from '@/lib/types/earnings';
import { formatEPS, formatRevenue, formatSurprise } from '@/lib/utils/earningsFormatters';

interface EarningsBarChartProps {
  data: EarningsResult[];
  type: 'eps' | 'revenue';
}

type ZoomOption = '2Y' | '5Y' | 'All';

const QUARTERS_MAP: Record<ZoomOption, number> = {
  '2Y': 8,
  '5Y': 20,
  All: 999,
};

export default function EarningsBarChart({ data, type }: EarningsBarChartProps) {
  const [zoom, setZoom] = useState<ZoomOption>('2Y');
  const { resolvedTheme } = useTheme();
  const chartColors = useMemo(
    () => getChartColors(resolvedTheme as 'light' | 'dark'),
    [resolvedTheme]
  );

  const chartData = useMemo(() => {
    const maxQuarters = QUARTERS_MAP[zoom];
    return data
      .filter((e) => !e.isUpcoming)
      .slice(0, maxQuarters)
      .reverse() // oldest first for chart
      .map((e) => ({
        quarter: e.fiscalQuarter,
        estimated: type === 'eps' ? e.epsEstimated : e.revenueEstimated,
        actual: type === 'eps' ? e.epsActual : e.revenueActual,
        beat: type === 'eps' ? e.epsBeat : e.revenueBeat,
        surprise: type === 'eps' ? e.epsSurprisePercent : e.revenueSurprisePercent,
      }));
  }, [data, type, zoom]);

  if (chartData.length === 0) return null;

  const isEPS = type === 'eps';
  const title = isEPS ? 'EPS History' : 'Revenue History';

  const yTickFormatter = isEPS
    ? (value: number) => formatEPS(value)
    : (value: number) => formatRevenue(value);

  return (
    <div className="bg-ic-bg-secondary rounded-lg p-4">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-base font-semibold text-ic-text-primary">{title}</h3>
        <div className="flex gap-1">
          {(['2Y', '5Y', 'All'] as ZoomOption[]).map((opt) => (
            <button
              key={opt}
              onClick={() => setZoom(opt)}
              className={`px-2.5 py-1 text-xs rounded-md font-medium transition-colors ${
                zoom === opt
                  ? 'bg-ic-blue text-white'
                  : 'text-ic-text-muted hover:text-ic-text-primary hover:bg-ic-bg-tertiary'
              }`}
            >
              {opt}
            </button>
          ))}
        </div>
      </div>

      <ResponsiveContainer width="100%" height={300} key={resolvedTheme}>
        <BarChart
          data={chartData}
          margin={{ top: 10, right: 10, bottom: 10, left: isEPS ? 10 : 20 }}
        >
          <CartesianGrid strokeDasharray="3 3" stroke={chartColors.grid} vertical={false} />
          <XAxis
            dataKey="quarter"
            tick={{ fill: chartColors.text, fontSize: 11 }}
            tickLine={false}
            axisLine={{ stroke: chartColors.grid }}
          />
          <YAxis
            tick={{ fill: chartColors.text, fontSize: 11 }}
            tickLine={false}
            axisLine={false}
            tickFormatter={yTickFormatter}
          />
          {isEPS && <ReferenceLine y={0} stroke={chartColors.grid} />}
          <Tooltip
            content={({ active, payload }) => {
              if (!active || !payload || payload.length === 0) return null;
              const item = payload[0].payload;
              return (
                <div className="bg-ic-surface border border-ic-border rounded-lg shadow-lg p-3 text-sm">
                  <p className="font-medium text-ic-text-primary mb-1">{item.quarter}</p>
                  <p className="text-ic-text-muted">
                    Estimate: {isEPS ? formatEPS(item.estimated) : formatRevenue(item.estimated)}
                  </p>
                  <p className="text-ic-text-primary">
                    Actual: {isEPS ? formatEPS(item.actual) : formatRevenue(item.actual)}
                  </p>
                  {item.surprise != null && (
                    <p className={item.beat ? 'text-green-400' : 'text-red-400'}>
                      Surprise: {formatSurprise(item.surprise)} {item.beat ? '(Beat)' : '(Miss)'}
                    </p>
                  )}
                </div>
              );
            }}
          />
          {/* Estimate bar — ghost/outline style */}
          <Bar
            dataKey="estimated"
            name="Estimate"
            fill="transparent"
            stroke="#6B7280"
            strokeWidth={1}
            radius={[2, 2, 0, 0]}
          />
          {/* Actual bar — colored by beat/miss */}
          <Bar dataKey="actual" name="Actual" radius={[2, 2, 0, 0]}>
            {chartData.map((entry) => (
              <Cell
                key={entry.quarter}
                fill={
                  entry.beat === true
                    ? chartColors.positive
                    : entry.beat === false
                      ? chartColors.negative
                      : chartColors.text
                }
              />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
