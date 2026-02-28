'use client';

import { Tooltip } from '@/components/ui/Tooltip';
import type { PercentileDistribution } from '@/lib/types/fundamentals';

interface SectorPercentileBarProps {
  /** Percentile 0-100 (direction-adjusted by API: higher = better) */
  percentile: number;
  /** Distribution tick marks from sector data */
  distribution: PercentileDistribution;
  /** The stock's actual metric value */
  value: number;
  /** Whether lower raw values are better (informational, API already adjusts percentile) */
  lowerIsBetter: boolean;
  /** Display name of the metric */
  metricName: string;
  /** Sector name for context */
  sector: string;
  /** Number of companies in the sector sample */
  sampleCount: number;
  /** Size variant: sm for sidebar, md for tab content */
  size?: 'sm' | 'md';
  /** Whether the bar is visible (false = blurred placeholder for free-tier gating) */
  visible?: boolean;
}

/**
 * Returns a Tailwind background color class based on the percentile zone.
 * Higher percentile = greener (API already direction-adjusts).
 *
 * Zones: 0-25 red, 25-40 orange, 40-60 yellow, 60-75 light green, 75-100 green
 */
function getZoneColor(percentile: number): string {
  if (percentile >= 75) return 'bg-green-500';
  if (percentile >= 60) return 'bg-green-400';
  if (percentile >= 40) return 'bg-yellow-400';
  if (percentile >= 25) return 'bg-orange-400';
  return 'bg-red-400';
}

/**
 * Returns a Tailwind text color class for the dot indicator on mobile.
 */
function getDotColor(percentile: number): string {
  if (percentile >= 75) return 'bg-green-500';
  if (percentile >= 60) return 'bg-green-400';
  if (percentile >= 40) return 'bg-yellow-400';
  if (percentile >= 25) return 'bg-orange-400';
  return 'bg-red-400';
}

/**
 * Returns the zone label for the percentile.
 */
function getZoneLabel(percentile: number): string {
  if (percentile >= 75) return 'Excellent';
  if (percentile >= 60) return 'Good';
  if (percentile >= 40) return 'Average';
  if (percentile >= 25) return 'Below Average';
  return 'Poor';
}

/**
 * Formats a metric value for tooltip display.
 */
function formatValue(value: number): string {
  if (Math.abs(value) >= 1e9) return `${(value / 1e9).toFixed(1)}B`;
  if (Math.abs(value) >= 1e6) return `${(value / 1e6).toFixed(1)}M`;
  if (Math.abs(value) >= 1e3) return `${(value / 1e3).toFixed(1)}K`;
  if (Math.abs(value) < 1) return value.toFixed(3);
  if (Math.abs(value) < 100) return value.toFixed(2);
  return value.toFixed(1);
}

/**
 * Skeleton placeholder shown during loading state.
 * Reserves exact bar height to prevent layout shift.
 */
export function SectorPercentileBarSkeleton({ size = 'sm' }: { size?: 'sm' | 'md' }) {
  const height = size === 'sm' ? 'h-[6px]' : 'h-[8px]';
  const width = size === 'sm' ? 'w-[140px]' : 'w-[200px]';

  return <div className={`${width} ${height} bg-ic-border rounded-full animate-pulse mt-1`} />;
}

export default function SectorPercentileBar({
  percentile,
  distribution,
  value,
  lowerIsBetter,
  metricName,
  sector,
  sampleCount,
  size = 'sm',
  visible = true,
}: SectorPercentileBarProps) {
  const barHeight = size === 'sm' ? 'h-[6px]' : 'h-[8px]';
  const barWidth = size === 'sm' ? 'w-[140px]' : 'w-[200px]';
  const dotSize = size === 'sm' ? 'w-2.5 h-2.5' : 'w-3 h-3';

  // Clamp percentile to 0-100
  const clampedPercentile = Math.max(0, Math.min(100, percentile));

  // Calculate distribution tick positions as percentages
  const p25Pos = 25;
  const p50Pos = 50;
  const p75Pos = 75;

  // Blurred placeholder for non-premium metrics
  if (!visible) {
    return (
      <div className={`${barWidth} mt-1`}>
        {/* Mobile: blurred dot */}
        <div className="md:hidden flex items-center gap-1.5">
          <div className="w-2.5 h-2.5 rounded-full bg-ic-border blur-[2px]" />
          <span className="text-[10px] text-ic-text-dim">Premium</span>
        </div>
        {/* Desktop: blurred bar */}
        <div className="hidden md:block">
          <div
            className={`${barHeight} rounded-full bg-ic-border blur-[2px] relative`}
            aria-hidden="true"
          />
          <span className="text-[10px] text-ic-text-dim">Premium</span>
        </div>
      </div>
    );
  }

  const tooltipContent = (
    <div className="space-y-1.5 min-w-[180px]">
      <div className="font-medium text-ic-text-primary text-sm">{metricName}</div>
      <div className="flex justify-between text-xs">
        <span className="text-ic-text-muted">Value</span>
        <span className="text-ic-text-primary font-medium">{formatValue(value)}</span>
      </div>
      <div className="flex justify-between text-xs">
        <span className="text-ic-text-muted">Percentile</span>
        <span className="text-ic-text-primary font-medium">
          {clampedPercentile.toFixed(0)}th ({getZoneLabel(clampedPercentile)})
        </span>
      </div>
      <div className="flex justify-between text-xs">
        <span className="text-ic-text-muted">Sector</span>
        <span className="text-ic-text-primary">{sector}</span>
      </div>
      <div className="flex justify-between text-xs">
        <span className="text-ic-text-muted">Sample</span>
        <span className="text-ic-text-primary">{sampleCount} companies</span>
      </div>
      <div className="flex justify-between text-xs">
        <span className="text-ic-text-muted">Sector Median</span>
        <span className="text-ic-text-primary">{formatValue(distribution.p50)}</span>
      </div>
      {lowerIsBetter && (
        <div className="text-[10px] text-ic-text-dim italic">
          Lower values are better for this metric
        </div>
      )}
    </div>
  );

  const ariaLabel = `${metricName}: ${formatValue(value)}, ${clampedPercentile.toFixed(0)}th percentile in ${sector} sector (${sampleCount} companies). ${getZoneLabel(clampedPercentile)}.`;

  return (
    <Tooltip content={tooltipContent} position="top">
      <div
        className={`${barWidth} mt-1`}
        role="meter"
        aria-label={ariaLabel}
        aria-valuenow={clampedPercentile}
        aria-valuemin={0}
        aria-valuemax={100}
      >
        {/* Mobile: dot-only indicator */}
        <div className="md:hidden flex items-center gap-1.5">
          <div
            className={`${dotSize} rounded-full ${getDotColor(clampedPercentile)} flex-shrink-0`}
          />
          <span className="text-[10px] text-ic-text-dim">P{clampedPercentile.toFixed(0)}</span>
        </div>

        {/* Desktop: full bar with zones and marker */}
        <div className="hidden md:block relative">
          {/* Background bar with 5 color zones */}
          <div className={`${barHeight} rounded-full overflow-hidden flex`}>
            <div className="w-[25%] bg-red-400/30 h-full" />
            <div className="w-[15%] bg-orange-400/30 h-full" />
            <div className="w-[20%] bg-yellow-400/30 h-full" />
            <div className="w-[15%] bg-green-400/30 h-full" />
            <div className="w-[25%] bg-green-500/30 h-full" />
          </div>

          {/* Distribution tick marks at p25, p50, p75 */}
          <div
            className="absolute top-0 w-px bg-ic-text-dim/40 h-full"
            style={{ left: `${p25Pos}%` }}
          />
          <div
            className="absolute top-0 w-px bg-ic-text-dim/60 h-full"
            style={{ left: `${p50Pos}%` }}
          />
          <div
            className="absolute top-0 w-px bg-ic-text-dim/40 h-full"
            style={{ left: `${p75Pos}%` }}
          />

          {/* Percentile position marker (solid dot) */}
          <div
            className={`absolute top-1/2 -translate-y-1/2 ${dotSize} rounded-full ${getZoneColor(clampedPercentile)} border border-white/80 shadow-sm`}
            style={{
              left: `calc(${clampedPercentile}% - ${size === 'sm' ? '5px' : '6px'})`,
            }}
          />
        </div>
      </div>
    </Tooltip>
  );
}
