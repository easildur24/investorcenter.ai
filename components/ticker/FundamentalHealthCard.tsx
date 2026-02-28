'use client';

import { useState } from 'react';
import { cn } from '@/lib/utils';
import { useAuth } from '@/lib/auth/AuthContext';
import { HealthBadge } from '@/components/ui/HealthBadge';
import { LifecycleBadge } from '@/components/ui/LifecycleBadge';
import { RedFlagBadge } from '@/components/ui/RedFlagBadge';
import type {
  HealthSummaryResponse,
  StrengthItem,
  ConcernItem,
  RedFlag,
} from '@/lib/types/fundamentals';

// ─── Props ───────────────────────────────────────────────────────────────────

interface FundamentalHealthCardProps {
  ticker: string;
  data?: HealthSummaryResponse | null;
  variant?: 'full' | 'compact';
  isPremium?: boolean;
  loading?: boolean;
}

// ─── Constants ───────────────────────────────────────────────────────────────

const FREE_VISIBLE_COUNT = 2;

// ─── Z-Score zone helpers ────────────────────────────────────────────────────

function getZScoreZoneColor(zone: string): string {
  switch (zone) {
    case 'safe':
      return 'bg-green-400';
    case 'grey':
      return 'bg-yellow-400';
    case 'distress':
      return 'bg-red-400';
    default:
      return 'bg-slate-400';
  }
}

function getZScoreZoneLabel(zone: string): string {
  switch (zone) {
    case 'safe':
      return 'Safe';
    case 'grey':
      return 'Grey';
    case 'distress':
      return 'Distress';
    default:
      return zone;
  }
}

// ─── Mobile collapsed summary ────────────────────────────────────────────────

function mobileCompactSummary(data: HealthSummaryResponse): string {
  const badge = data.health.badge;
  const stage = data.lifecycle.stage.charAt(0).toUpperCase() + data.lifecycle.stage.slice(1);
  const strengths = data.strengths.length;
  const concerns = data.concerns.length;
  const highFlags = data.red_flags.filter((f) => f.severity === 'high').length;

  const parts = [badge, stage];
  if (strengths > 0) parts.push(`${strengths}\u2713`);
  if (concerns > 0) parts.push(`${concerns}\u26A0`);
  if (highFlags > 0) parts.push(`${highFlags}\uD83D\uDD34`);

  return parts.join(' \u00B7 ');
}

// ─── Main Component ──────────────────────────────────────────────────────────

export default function FundamentalHealthCard({
  ticker,
  data,
  variant = 'full',
  isPremium: isPremiumProp,
  loading = false,
}: FundamentalHealthCardProps) {
  const { user } = useAuth();
  const isPremium = isPremiumProp ?? user?.is_premium ?? false;
  const [mobileExpanded, setMobileExpanded] = useState(false);

  // ── Loading skeleton ────────────────────────────────────────────────────
  if (loading) {
    return <HealthCardSkeleton />;
  }

  // ── No data ─────────────────────────────────────────────────────────────
  if (!data) {
    return null;
  }

  const { health, lifecycle, strengths, concerns, red_flags } = data;

  // ── Freemium gating ─────────────────────────────────────────────────────
  const visibleStrengths = isPremium ? strengths : strengths.slice(0, FREE_VISIBLE_COUNT);
  const hiddenStrengthCount = strengths.length - visibleStrengths.length;

  const visibleConcerns = isPremium ? concerns : concerns.slice(0, FREE_VISIBLE_COUNT);
  const hiddenConcernCount = concerns.length - visibleConcerns.length;

  const visibleFlags = isPremium ? red_flags : red_flags.filter((f) => f.severity === 'high');
  const hiddenFlagCount = red_flags.length - visibleFlags.length;

  return (
    <div className="border border-ic-border rounded-xl bg-ic-bg-secondary overflow-hidden">
      {/* ── Mobile collapsed bar ─────────────────────────────────────────── */}
      <button
        type="button"
        className="w-full flex items-center justify-between px-4 py-3 md:hidden"
        onClick={() => setMobileExpanded((prev) => !prev)}
        aria-expanded={mobileExpanded}
      >
        <span className="text-sm font-medium text-ic-text-primary">
          {mobileCompactSummary(data)}
        </span>
        <svg
          className={cn(
            'w-4 h-4 text-ic-text-dim transition-transform duration-200',
            mobileExpanded && 'rotate-180'
          )}
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          strokeWidth={2}
          aria-hidden="true"
        >
          <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {/* ── Full card body (always visible on desktop, toggle on mobile) ── */}
      <div
        className={cn(
          'transition-all duration-200 overflow-hidden',
          // Desktop: always show. Mobile: toggle.
          'md:max-h-none md:opacity-100',
          mobileExpanded
            ? 'max-h-[2000px] opacity-100'
            : 'max-h-0 opacity-0 md:max-h-none md:opacity-100'
        )}
      >
        <div className="px-4 pb-4 md:pt-4 space-y-5">
          {/* 1. Header */}
          <h3 className="text-base font-semibold text-ic-text-primary">Fundamental Health</h3>

          {/* 2. Badge row */}
          <div className="flex flex-wrap items-center gap-2">
            <HealthBadge badge={health.badge} score={health.score} hideScore={!isPremium} />
            <LifecycleBadge stage={lifecycle.stage} description={lifecycle.description} />
          </div>

          {/* 3. Component bars (F-Score, Z-Score, IC Health) */}
          {variant === 'full' && (
            <div className="space-y-3">
              <ComponentBar
                label="F-Score"
                value={health.components.piotroski_f_score.value}
                max={health.components.piotroski_f_score.max}
                tooltip={health.components.piotroski_f_score.interpretation}
              />
              <ZScoreBar
                value={health.components.altman_z_score.value}
                zone={health.components.altman_z_score.zone}
                tooltip={health.components.altman_z_score.interpretation}
              />
              <ComponentBar
                label="IC Health"
                value={health.components.ic_financial_health.value}
                max={health.components.ic_financial_health.max}
                tooltip={`IC Financial Health score: ${health.components.ic_financial_health.value}/${health.components.ic_financial_health.max}`}
              />
            </div>
          )}

          {/* 4. Strengths */}
          {visibleStrengths.length > 0 && (
            <div>
              <h4 className="text-xs font-medium text-ic-text-secondary uppercase tracking-wide mb-2">
                Strengths
              </h4>
              <ul className="space-y-1.5">
                {visibleStrengths.map((s) => (
                  <StrengthRow key={s.metric} item={s} />
                ))}
              </ul>
              {hiddenStrengthCount > 0 && (
                <UpgradeTeaser count={hiddenStrengthCount} label="strength" />
              )}
            </div>
          )}

          {/* 5. Concerns */}
          {visibleConcerns.length > 0 && (
            <div>
              <h4 className="text-xs font-medium text-ic-text-secondary uppercase tracking-wide mb-2">
                Concerns
              </h4>
              <ul className="space-y-1.5">
                {visibleConcerns.map((c) => (
                  <ConcernRow key={c.metric} item={c} />
                ))}
              </ul>
              {hiddenConcernCount > 0 && (
                <UpgradeTeaser count={hiddenConcernCount} label="concern" />
              )}
            </div>
          )}

          {/* 6. Red flags */}
          {visibleFlags.length > 0 && (
            <div>
              <h4 className="text-xs font-medium text-ic-text-secondary uppercase tracking-wide mb-2">
                Red Flags
              </h4>
              <div className="space-y-2">
                {visibleFlags.map((flag) => (
                  <RedFlagBadge
                    key={flag.id}
                    id={flag.id}
                    severity={flag.severity}
                    title={flag.title}
                    description={flag.description}
                    relatedMetrics={flag.related_metrics}
                  />
                ))}
              </div>
              {hiddenFlagCount > 0 && <UpgradeTeaser count={hiddenFlagCount} label="flag" />}
            </div>
          )}

          {/* 7. Footer */}
          {lifecycle.classified_at && (
            <p className="text-xs text-ic-text-dim pt-1">
              Data as of{' '}
              {new Date(lifecycle.classified_at).toLocaleDateString(undefined, {
                year: 'numeric',
                month: 'short',
                day: 'numeric',
              })}
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

// ─── Sub-components ──────────────────────────────────────────────────────────

/** Horizontal bar for Piotroski F-Score or IC Health (x out of max). */
function ComponentBar({
  label,
  value,
  max,
  tooltip,
}: {
  label: string;
  value: number;
  max: number;
  tooltip?: string;
}) {
  const pct = max > 0 ? Math.min(100, (value / max) * 100) : 0;
  return (
    <div className="flex items-center gap-2" title={tooltip}>
      <span className="text-xs text-ic-text-muted w-16 flex-shrink-0">{label}</span>
      <div className="flex-1 h-1.5 rounded-full bg-white/10 overflow-hidden">
        <div
          className="h-full rounded-full bg-ic-blue transition-all duration-300"
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className="text-xs text-ic-text-primary w-10 text-right">
        {value}/{max}
      </span>
    </div>
  );
}

/** Horizontal bar for Altman Z-Score showing zone coloring. */
function ZScoreBar({ value, zone, tooltip }: { value: number; zone: string; tooltip?: string }) {
  // Z-Score typically ranges from -4 to 8; clamp to 0-100% for display
  const pct = Math.max(0, Math.min(100, ((value + 4) / 12) * 100));
  return (
    <div className="flex items-center gap-2" title={tooltip}>
      <span className="text-xs text-ic-text-muted w-16 flex-shrink-0">Z-Score</span>
      <div className="flex-1 h-1.5 rounded-full bg-white/10 overflow-hidden">
        <div
          className={cn(
            'h-full rounded-full transition-all duration-300',
            getZScoreZoneColor(zone)
          )}
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className="text-xs text-ic-text-primary w-10 text-right">
        {getZScoreZoneLabel(zone)}
      </span>
    </div>
  );
}

/** Single strength row with green check. */
function StrengthRow({ item }: { item: StrengthItem }) {
  return (
    <li className="flex items-start gap-2 text-sm">
      <span className="text-green-400 flex-shrink-0 mt-0.5" aria-hidden="true">
        &#x2713;
      </span>
      <span className="text-ic-text-muted">{item.message}</span>
    </li>
  );
}

/** Single concern row with warning icon. */
function ConcernRow({ item }: { item: ConcernItem }) {
  return (
    <li className="flex items-start gap-2 text-sm">
      <span className="text-yellow-400 flex-shrink-0 mt-0.5" aria-hidden="true">
        &#x26A0;
      </span>
      <span className="text-ic-text-muted">{item.message}</span>
    </li>
  );
}

/** "Upgrade to see N more" teaser. */
function UpgradeTeaser({ count, label }: { count: number; label: string }) {
  const plural = count === 1 ? label : `${label}s`;
  return (
    <p className="text-xs text-ic-blue mt-2 cursor-pointer hover:underline">
      Upgrade to see {count} more {plural}
    </p>
  );
}

// ─── Skeleton ────────────────────────────────────────────────────────────────

export function HealthCardSkeleton() {
  return (
    <div className="border border-ic-border rounded-xl bg-ic-bg-secondary overflow-hidden animate-pulse">
      <div className="px-4 py-4 space-y-4">
        {/* Header skeleton */}
        <div className="h-5 bg-ic-border rounded w-36" />

        {/* Badge row skeleton */}
        <div className="flex gap-2">
          <div className="h-7 bg-ic-border rounded-full w-24" />
          <div className="h-7 bg-ic-border rounded-full w-20" />
        </div>

        {/* Bars skeleton */}
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <div className="h-3 bg-ic-border rounded w-16" />
            <div className="flex-1 h-1.5 bg-ic-border rounded-full" />
            <div className="h-3 bg-ic-border rounded w-8" />
          </div>
          <div className="flex items-center gap-2">
            <div className="h-3 bg-ic-border rounded w-16" />
            <div className="flex-1 h-1.5 bg-ic-border rounded-full" />
            <div className="h-3 bg-ic-border rounded w-8" />
          </div>
          <div className="flex items-center gap-2">
            <div className="h-3 bg-ic-border rounded w-16" />
            <div className="flex-1 h-1.5 bg-ic-border rounded-full" />
            <div className="h-3 bg-ic-border rounded w-8" />
          </div>
        </div>

        {/* Strengths/Concerns skeleton */}
        <div className="space-y-2">
          <div className="h-3 bg-ic-border rounded w-20" />
          <div className="h-4 bg-ic-border rounded w-full" />
          <div className="h-4 bg-ic-border rounded w-5/6" />
        </div>
      </div>
    </div>
  );
}
