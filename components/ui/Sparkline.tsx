'use client';

import { useMemo } from 'react';

interface SparklineProps {
  /** Array of numeric values to plot */
  data: number[];
  /** Width in pixels (default: 80) */
  width?: number;
  /** Height in pixels (default: 24) */
  height?: number;
  /** Stroke color — auto green/red if not provided based on first→last comparison */
  color?: string;
  /** Stroke width (default: 1.5) */
  strokeWidth?: number;
  /** Accessible label */
  ariaLabel?: string;
  /** Optional className for the SVG container */
  className?: string;
}

/**
 * Minimal SVG sparkline chart.
 * Renders a polyline from an array of numbers.
 * Auto-colors green (up) or red (down) by default.
 */
export default function Sparkline({
  data,
  width = 80,
  height = 24,
  color,
  strokeWidth = 1.5,
  ariaLabel,
  className,
}: SparklineProps) {
  const { points, resolvedColor } = useMemo(() => {
    if (!data || data.length < 2) {
      return { points: '', resolvedColor: color || 'var(--ic-text-muted)' };
    }

    const min = Math.min(...data);
    const max = Math.max(...data);
    const range = max - min || 1;

    // Padding to prevent clipping at edges
    const padX = strokeWidth;
    const padY = strokeWidth;
    const innerW = width - padX * 2;
    const innerH = height - padY * 2;

    const pts = data
      .map((val, i) => {
        const x = padX + (i / (data.length - 1)) * innerW;
        const y = padY + innerH - ((val - min) / range) * innerH;
        return `${x.toFixed(1)},${y.toFixed(1)}`;
      })
      .join(' ');

    // Auto-detect color from trend: last value vs first value
    const autoColor =
      data[data.length - 1] >= data[0] ? 'var(--ic-positive)' : 'var(--ic-negative)';

    return { points: pts, resolvedColor: color || autoColor };
  }, [data, width, height, color, strokeWidth]);

  if (!data || data.length < 2) {
    return (
      <svg
        width={width}
        height={height}
        className={className}
        role="img"
        aria-label={ariaLabel || 'No data'}
      />
    );
  }

  return (
    <svg
      width={width}
      height={height}
      className={className}
      role="img"
      aria-label={ariaLabel || 'Sparkline chart'}
      viewBox={`0 0 ${width} ${height}`}
    >
      <polyline
        points={points}
        fill="none"
        stroke={resolvedColor}
        strokeWidth={strokeWidth}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
