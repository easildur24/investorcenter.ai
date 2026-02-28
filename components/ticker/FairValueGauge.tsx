'use client';

import Link from 'next/link';
import { cn } from '@/lib/utils';
import { useAuth } from '@/lib/auth/AuthContext';
import { useFairValue } from '@/lib/hooks/useFairValue';
import type { FairValueResponse } from '@/lib/types/fundamentals';

// ── Format helpers ────────────────────────────────────────────────────────────

function formatPrice(n: number): string {
  if (n == null || isNaN(n)) return 'N/A';
  return `$${n.toFixed(2)}`;
}

function formatPercent(n: number): string {
  if (n == null || isNaN(n)) return 'N/A';
  const sign = n >= 0 ? '+' : '';
  return `${sign}${n.toFixed(1)}%`;
}

// ── Confidence badge ──────────────────────────────────────────────────────────

function ConfidenceBadge({ level }: { level: string }) {
  const normalized = level.toLowerCase();
  const colorClass =
    normalized === 'high'
      ? 'bg-green-500/20 text-green-400'
      : normalized === 'medium'
        ? 'bg-yellow-500/20 text-yellow-400'
        : 'bg-red-500/20 text-red-400';

  return (
    <span
      className={cn(
        'inline-flex items-center text-[10px] font-medium px-1.5 py-0.5 rounded',
        colorClass
      )}
    >
      {level}
    </span>
  );
}

// ── Zone label ────────────────────────────────────────────────────────────────

function ZoneBadge({ zone }: { zone: FairValueResponse['margin_of_safety']['zone'] }) {
  const config: Record<string, { label: string; className: string }> = {
    undervalued: {
      label: 'Undervalued',
      className: 'bg-green-500/20 text-green-400 border-green-500/30',
    },
    fairly_valued: {
      label: 'Fairly Valued',
      className: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30',
    },
    overvalued: {
      label: 'Overvalued',
      className: 'bg-red-500/20 text-red-400 border-red-500/30',
    },
  };

  const c = config[zone] ?? config.fairly_valued;

  return (
    <span
      className={cn(
        'inline-flex items-center text-xs font-semibold px-2.5 py-1 rounded-full border',
        c.className
      )}
    >
      {c.label}
    </span>
  );
}

// ── Skeleton ──────────────────────────────────────────────────────────────────

function FairValueSkeleton() {
  return (
    <div className="animate-pulse space-y-4 p-4">
      <div className="h-5 bg-ic-border rounded w-36" />
      <div className="h-8 bg-ic-border rounded w-full" />
      <div className="space-y-2">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="flex justify-between">
            <div className="h-4 bg-ic-border rounded w-24" />
            <div className="h-4 bg-ic-border rounded w-16" />
            <div className="h-4 bg-ic-border rounded w-16" />
          </div>
        ))}
      </div>
      <div className="h-12 bg-ic-border rounded w-full" />
    </div>
  );
}

// ── Gauge bar ─────────────────────────────────────────────────────────────────

interface GaugeMarker {
  label: string;
  value: number;
  color: string;
  isCurrent?: boolean;
}

function GaugeBar({ data }: { data: FairValueResponse }) {
  const avgFV = data.margin_of_safety.avg_fair_value;
  const current = data.current_price;

  // Build markers
  const markers: GaugeMarker[] = [
    {
      label: 'DCF',
      value: data.models.dcf.fair_value,
      color: 'bg-blue-400',
    },
    {
      label: 'Graham',
      value: data.models.graham_number.fair_value,
      color: 'bg-purple-400',
    },
    {
      label: 'EPV',
      value: data.models.epv.fair_value,
      color: 'bg-cyan-400',
    },
  ];

  if (data.analyst_consensus) {
    markers.push({
      label: 'Analyst',
      value: data.analyst_consensus.target_price,
      color: 'bg-orange-400',
    });
  }

  markers.push({
    label: 'Price',
    value: current,
    color: 'bg-white',
    isCurrent: true,
  });

  // Determine price range for the gauge
  const allValues = markers.map((m) => m.value).filter((v) => v > 0);
  const min = Math.min(...allValues) * 0.7;
  const max = Math.max(...allValues) * 1.3;
  const range = max - min;

  function getPosition(value: number): number {
    if (range === 0) return 50;
    return Math.max(2, Math.min(98, ((value - min) / range) * 100));
  }

  // Zone boundaries relative to avg fair value
  const undervaluedEnd = getPosition(avgFV * 0.85);
  const fairEnd = getPosition(avgFV * 1.15);

  return (
    <div className="relative mb-8">
      {/* Gauge track */}
      <div className="relative h-3 rounded-full overflow-hidden bg-ic-bg-secondary">
        {/* Green (undervalued) zone */}
        <div
          className="absolute inset-y-0 left-0 bg-gradient-to-r from-green-500 to-green-400 opacity-60"
          style={{ width: `${undervaluedEnd}%` }}
        />
        {/* Yellow (fair) zone */}
        <div
          className="absolute inset-y-0 bg-gradient-to-r from-yellow-400 to-yellow-500 opacity-60"
          style={{ left: `${undervaluedEnd}%`, width: `${fairEnd - undervaluedEnd}%` }}
        />
        {/* Red (overvalued) zone */}
        <div
          className="absolute inset-y-0 bg-gradient-to-r from-red-400 to-red-500 opacity-60"
          style={{ left: `${fairEnd}%`, right: '0' }}
        />
      </div>

      {/* Markers */}
      <div className="relative h-16 mt-1">
        {markers.map((marker) => {
          const pos = getPosition(marker.value);
          return (
            <div
              key={marker.label}
              className="absolute flex flex-col items-center"
              style={{ left: `${pos}%`, transform: 'translateX(-50%)' }}
            >
              {/* Tick */}
              <div
                className={cn(
                  'rounded-full',
                  marker.isCurrent
                    ? 'w-1 h-5 -mt-4 bg-white shadow-lg shadow-white/30'
                    : 'w-0.5 h-3 -mt-2',
                  !marker.isCurrent && marker.color
                )}
              />
              {/* Label */}
              <span
                className={cn(
                  'text-[9px] mt-0.5 whitespace-nowrap',
                  marker.isCurrent ? 'text-white font-bold text-[10px]' : 'text-ic-text-dim'
                )}
              >
                {marker.label}
              </span>
              <span
                className={cn(
                  'text-[9px]',
                  marker.isCurrent ? 'text-white font-medium' : 'text-ic-text-dim'
                )}
              >
                {formatPrice(marker.value)}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

// ── Model estimates table ─────────────────────────────────────────────────────

function ModelEstimatesTable({ data }: { data: FairValueResponse }) {
  const rows = [
    {
      model: 'DCF',
      fairValue: data.models.dcf.fair_value,
      upside: data.models.dcf.upside_percent,
      confidence: data.models.dcf.confidence,
    },
    {
      model: 'Graham Number',
      fairValue: data.models.graham_number.fair_value,
      upside: data.models.graham_number.upside_percent,
      confidence: data.models.graham_number.confidence,
    },
    {
      model: 'Earnings Power',
      fairValue: data.models.epv.fair_value,
      upside: data.models.epv.upside_percent,
      confidence: data.models.epv.confidence,
    },
  ];

  if (data.analyst_consensus) {
    rows.push({
      model: 'Analyst Target',
      fairValue: data.analyst_consensus.target_price,
      upside: data.analyst_consensus.upside_percent,
      confidence: `${data.analyst_consensus.num_analysts} analysts`,
    });
  }

  return (
    <table className="w-full text-sm mb-4">
      <thead>
        <tr className="border-b border-ic-border/50">
          <th className="text-left text-xs font-medium text-ic-text-dim py-2 pr-2">Model</th>
          <th className="text-right text-xs font-medium text-ic-text-dim py-2 px-2">Fair Value</th>
          <th className="text-right text-xs font-medium text-ic-text-dim py-2 pl-2">vs. Price</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((row) => (
          <tr key={row.model} className="border-b border-ic-border/30 hover:bg-ic-bg-secondary/30">
            <td className="py-2 pr-2">
              <div className="flex items-center gap-2">
                <span className="text-ic-text-primary text-xs">{row.model}</span>
                <ConfidenceBadge level={row.confidence} />
              </div>
            </td>
            <td className="text-right py-2 px-2 font-medium text-ic-text-primary text-xs">
              {formatPrice(row.fairValue)}
            </td>
            <td
              className={cn(
                'text-right py-2 pl-2 font-medium text-xs',
                row.upside >= 0 ? 'text-green-400' : 'text-red-400'
              )}
            >
              {formatPercent(row.upside)}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

// ── Main component ────────────────────────────────────────────────────────────

interface FairValueGaugeProps {
  ticker: string;
  isPremium?: boolean;
}

export default function FairValueGauge({ ticker, isPremium: isPremiumProp }: FairValueGaugeProps) {
  const { user } = useAuth();
  const isPremium = isPremiumProp ?? user?.is_premium ?? false;
  const { data, loading, error } = useFairValue(ticker);

  if (loading) return <FairValueSkeleton />;

  if (error) {
    return <div className="p-4 text-sm text-ic-negative">Failed to load fair value: {error}</div>;
  }

  if (!data) return null;

  // Suppressed (e.g., pre-revenue companies)
  if (data.meta.suppressed) {
    return (
      <div className="p-4">
        <h3 className="text-sm font-semibold text-ic-text-primary mb-2">Fair Value Estimate</h3>
        <div className="flex items-start gap-2 p-3 rounded-lg bg-yellow-500/10 border border-yellow-500/30">
          <svg
            className="w-5 h-5 text-yellow-400 flex-shrink-0 mt-0.5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z"
            />
          </svg>
          <div>
            <p className="text-sm font-medium text-yellow-400">Fair value unavailable</p>
            <p className="text-xs text-ic-text-dim mt-0.5">
              {data.meta.suppression_reason ??
                'This stock does not have sufficient data for fair value modeling.'}
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="p-4">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-semibold text-ic-text-primary">Fair Value Estimate</h3>
        <span className="text-xs text-ic-text-dim">Current: {formatPrice(data.current_price)}</span>
      </div>

      <div className="relative">
        {/* Free tier overlay */}
        {!isPremium && (
          <div className="absolute inset-0 z-10 flex items-center justify-center bg-ic-bg-primary/60 backdrop-blur-sm rounded-lg">
            <div className="text-center p-6">
              <svg
                className="w-8 h-8 mx-auto mb-2 text-ic-text-dim"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
                />
              </svg>
              <p className="text-sm font-medium text-ic-text-primary mb-1">Upgrade to unlock</p>
              <p className="text-xs text-ic-text-dim mb-3">
                See fair value models and margin of safety
              </p>
              <Link
                href="/pricing"
                className="inline-block text-sm font-medium px-4 py-2 bg-ic-blue text-white rounded-lg hover:bg-ic-blue/90 transition-colors"
              >
                See Plans
              </Link>
            </div>
          </div>
        )}

        <div className={cn(!isPremium && 'blur-[6px] select-none pointer-events-none')}>
          {/* Gauge visualization */}
          <GaugeBar data={data} />

          {/* Model estimates table */}
          <ModelEstimatesTable data={data} />

          {/* Margin of safety assessment */}
          <div className="flex items-center justify-between p-3 rounded-lg bg-ic-bg-secondary border border-ic-border/50">
            <div>
              <p className="text-xs text-ic-text-dim mb-1">Margin of Safety</p>
              <ZoneBadge zone={data.margin_of_safety.zone} />
            </div>
            <div className="text-right max-w-[200px]">
              <p className="text-xs text-ic-text-dim">
                Avg Fair Value: {formatPrice(data.margin_of_safety.avg_fair_value)}
              </p>
              <p className="text-[11px] text-ic-text-dim mt-0.5">
                {data.margin_of_safety.description}
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
