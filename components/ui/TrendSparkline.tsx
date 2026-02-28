'use client';

import { useMemo } from 'react';
import { Tooltip } from '@/components/ui/Tooltip';

interface TrendSparklineProps {
  values: number[];
  trend: 'up' | 'down' | 'flat';
  higherIsBetter: boolean;
  width?: number;
  height?: number;
  hoverData?: Array<{ label: string; value: number }>;
  onClick?: () => void;
  visible?: boolean;
  metricLabel?: string;
  unit?: 'USD' | 'percent' | 'ratio';
  consecutiveGrowthQuarters?: number;
}

/** Format a value for display in the tooltip based on unit type. */
function formatSparklineValue(value: number, unit: 'USD' | 'percent' | 'ratio'): string {
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

/**
 * TrendSparkline — Inline SVG mini-chart for metric trends.
 *
 * Pure SVG implementation with no library dependencies.
 * Color-coded by trend direction relative to higherIsBetter semantics.
 */
export default function TrendSparkline({
  values,
  trend,
  higherIsBetter,
  width = 80,
  height = 20,
  hoverData,
  onClick,
  visible = true,
  metricLabel,
  unit = 'USD',
  consecutiveGrowthQuarters = 0,
}: TrendSparklineProps) {
  const { pathD, lastPoint, strokeColor } = useMemo(() => {
    if (values.length < 2) {
      return { pathD: '', lastPoint: { x: 0, y: 0 }, strokeColor: '#6b7280' };
    }

    const min = Math.min(...values);
    const max = Math.max(...values);
    const range = max - min || 1;
    const padding = 2;
    const innerHeight = height - padding * 2;
    const innerWidth = width - padding * 2;

    const points = values.map((v, i) => ({
      x: padding + (i / (values.length - 1)) * innerWidth,
      y: padding + innerHeight - ((v - min) / range) * innerHeight,
    }));

    const d = points
      .map((p, i) => `${i === 0 ? 'M' : 'L'}${p.x.toFixed(1)},${p.y.toFixed(1)}`)
      .join(' ');

    const isPositive = (trend === 'up' && higherIsBetter) || (trend === 'down' && !higherIsBetter);
    const isNegative = (trend === 'down' && higherIsBetter) || (trend === 'up' && !higherIsBetter);
    const color = isPositive ? '#10b981' : isNegative ? '#ef4444' : '#6b7280';

    return {
      pathD: d,
      lastPoint: points[points.length - 1],
      strokeColor: color,
    };
  }, [values, trend, higherIsBetter, width, height]);

  // Build tooltip content
  const tooltipContent = useMemo(() => {
    if (!hoverData || hoverData.length === 0) return null;

    const trendArrow = trend === 'up' ? '\u25B2' : trend === 'down' ? '\u25BC' : '\u25C6';
    const trendWord = trend === 'up' ? 'Up' : trend === 'down' ? 'Down' : 'Flat';

    return (
      <div className="text-xs space-y-1 max-w-[200px]">
        {metricLabel && (
          <div className="font-medium text-ic-text-primary mb-1.5">
            {metricLabel} (5Y Quarterly)
          </div>
        )}
        {hoverData.slice(0, 6).map((dp, i) => (
          <div key={i} className="flex justify-between gap-3">
            <span className="text-ic-text-muted">{dp.label}:</span>
            <span className="font-medium text-ic-text-primary">
              {formatSparklineValue(dp.value, unit)}
              {i === hoverData.length - 1 ? ' (latest)' : ''}
            </span>
          </div>
        ))}
        {hoverData.length > 6 && (
          <div className="text-ic-text-dim">...{hoverData.length - 6} more</div>
        )}
        <div className="pt-1 border-t border-ic-border text-ic-text-muted">
          Trend: {trendArrow} {trendWord}
          {consecutiveGrowthQuarters > 0 &&
            `, ${consecutiveGrowthQuarters} consecutive growth quarters`}
        </div>
      </div>
    );
  }, [hoverData, metricLabel, unit, trend, consecutiveGrowthQuarters]);

  // Blurred state for free-tier users
  if (!visible) {
    return (
      <div
        className="blur-sm select-none pointer-events-none"
        style={{ width, height }}
        aria-hidden="true"
      >
        <svg width={width} height={height} role="img" aria-label="Metric trend (premium feature)">
          {pathD && (
            <>
              <path
                d={pathD}
                fill="none"
                stroke="#6b7280"
                strokeWidth={1.5}
                strokeLinecap="round"
                strokeLinejoin="round"
              />
              <circle cx={lastPoint.x} cy={lastPoint.y} r={2} fill="#6b7280" />
            </>
          )}
        </svg>
      </div>
    );
  }

  if (values.length < 2) {
    return (
      <div
        className="bg-ic-bg-secondary rounded"
        style={{ width, height }}
        aria-label="Insufficient data for sparkline"
      />
    );
  }

  const sparklineSvg = (
    <svg
      width={width}
      height={height}
      role="img"
      aria-label={`${metricLabel || 'Metric'} trend: ${trend}`}
      className={onClick ? 'cursor-pointer' : undefined}
      onClick={visible ? onClick : undefined}
    >
      <path
        d={pathD}
        fill="none"
        stroke={strokeColor}
        strokeWidth={1.5}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <circle cx={lastPoint.x} cy={lastPoint.y} r={2} fill={strokeColor} />
    </svg>
  );

  if (tooltipContent) {
    return (
      <Tooltip content={tooltipContent} position="top">
        {sparklineSvg}
      </Tooltip>
    );
  }

  return sparklineSvg;
}
