'use client';

import {
  LifecycleStage,
  LIFECYCLE_LABELS,
  LIFECYCLE_COLORS,
  ConfidenceLevel,
  CatalystImpact,
  getDeltaColor,
  formatDelta,
} from '@/lib/types/ic-score-v2';

// ===================
// Lifecycle Badge
// ===================

interface LifecycleBadgeProps {
  stage: LifecycleStage | null | undefined;
  size?: 'sm' | 'md';
}

export function LifecycleBadge({ stage, size = 'md' }: LifecycleBadgeProps) {
  if (!stage) return null;

  const colorClass = LIFECYCLE_COLORS[stage] || 'text-gray-600 bg-gray-100';
  const label = LIFECYCLE_LABELS[stage] || stage;
  const sizeClass = size === 'sm' ? 'text-xs px-2 py-0.5' : 'text-sm px-3 py-1';

  return (
    <span className={`inline-flex items-center rounded-full font-medium ${colorClass} ${sizeClass}`}>
      {label}
    </span>
  );
}

// ===================
// Confidence Badge
// ===================

interface ConfidenceBadgeProps {
  level: ConfidenceLevel | string;
  percentage?: number;
  size?: 'sm' | 'md';
}

const CONFIDENCE_COLORS: Record<string, string> = {
  High: 'text-green-700 bg-green-100',
  Medium: 'text-yellow-700 bg-yellow-100',
  Low: 'text-red-700 bg-red-100',
};

export function ConfidenceBadge({ level, percentage, size = 'md' }: ConfidenceBadgeProps) {
  const colorClass = CONFIDENCE_COLORS[level] || 'text-gray-600 bg-gray-100';
  const sizeClass = size === 'sm' ? 'text-xs px-2 py-0.5' : 'text-sm px-3 py-1';

  return (
    <span className={`inline-flex items-center gap-1 rounded-full font-medium ${colorClass} ${sizeClass}`}>
      {level === 'High' && '✓'}
      {level === 'Medium' && '~'}
      {level === 'Low' && '!'}
      <span>{level}</span>
      {percentage !== undefined && (
        <span className="opacity-75">({Math.round(percentage)}%)</span>
      )}
    </span>
  );
}

// ===================
// Delta Badge
// ===================

interface DeltaBadgeProps {
  delta: number;
  showSign?: boolean;
  size?: 'sm' | 'md';
}

export function DeltaBadge({ delta, showSign = true, size = 'md' }: DeltaBadgeProps) {
  const colorClass = getDeltaColor(delta);
  const sizeClass = size === 'sm' ? 'text-xs px-2 py-0.5' : 'text-sm px-2 py-1';

  const bgClass =
    delta > 0
      ? 'bg-green-50'
      : delta < 0
      ? 'bg-red-50'
      : 'bg-gray-50';

  return (
    <span className={`inline-flex items-center rounded font-medium ${colorClass} ${bgClass} ${sizeClass}`}>
      {showSign && delta > 0 && '↑'}
      {showSign && delta < 0 && '↓'}
      {showSign && delta === 0 && '→'}
      <span className="ml-1">{formatDelta(delta)}</span>
    </span>
  );
}

// ===================
// Impact Badge
// ===================

interface ImpactBadgeProps {
  impact: CatalystImpact;
  size?: 'sm' | 'md';
}

const IMPACT_STYLES: Record<CatalystImpact, string> = {
  Positive: 'text-green-700 bg-green-100',
  Negative: 'text-red-700 bg-red-100',
  Neutral: 'text-gray-700 bg-gray-100',
  Unknown: 'text-gray-500 bg-gray-50',
};

export function ImpactBadge({ impact, size = 'md' }: ImpactBadgeProps) {
  const styleClass = IMPACT_STYLES[impact] || IMPACT_STYLES.Unknown;
  const sizeClass = size === 'sm' ? 'text-xs px-2 py-0.5' : 'text-sm px-2 py-1';

  return (
    <span className={`inline-flex items-center rounded font-medium ${styleClass} ${sizeClass}`}>
      {impact}
    </span>
  );
}

// ===================
// Sector Rank Badge
// ===================

interface SectorRankBadgeProps {
  rank: number | null | undefined;
  total: number | null | undefined;
  size?: 'sm' | 'md';
}

export function SectorRankBadge({ rank, total, size = 'md' }: SectorRankBadgeProps) {
  if (!rank || !total) return null;

  const percentile = Math.round(((total - rank + 1) / total) * 100);
  const sizeClass = size === 'sm' ? 'text-xs px-2 py-0.5' : 'text-sm px-3 py-1';

  // Color based on percentile
  const colorClass =
    percentile >= 80
      ? 'text-green-700 bg-green-100'
      : percentile >= 60
      ? 'text-green-600 bg-green-50'
      : percentile >= 40
      ? 'text-yellow-700 bg-yellow-100'
      : percentile >= 20
      ? 'text-orange-700 bg-orange-100'
      : 'text-red-700 bg-red-100';

  return (
    <span className={`inline-flex items-center rounded-full font-medium ${colorClass} ${sizeClass}`}>
      #{rank} of {total}
    </span>
  );
}

// ===================
// Rating Badge
// ===================

interface RatingBadgeProps {
  rating: string;
  size?: 'sm' | 'md' | 'lg';
}

const RATING_STYLES: Record<string, string> = {
  'Strong Buy': 'text-green-700 bg-green-100 border-green-200',
  Buy: 'text-green-600 bg-green-50 border-green-100',
  Hold: 'text-yellow-700 bg-yellow-100 border-yellow-200',
  Underperform: 'text-orange-700 bg-orange-100 border-orange-200',
  Sell: 'text-red-700 bg-red-100 border-red-200',
};

export function RatingBadge({ rating, size = 'md' }: RatingBadgeProps) {
  const styleClass = RATING_STYLES[rating] || 'text-gray-700 bg-gray-100 border-gray-200';
  const sizeClass =
    size === 'lg'
      ? 'text-lg px-4 py-2'
      : size === 'sm'
      ? 'text-xs px-2 py-1'
      : 'text-sm px-3 py-1';

  return (
    <span className={`inline-flex items-center rounded-lg font-semibold border ${styleClass} ${sizeClass}`}>
      {rating}
    </span>
  );
}
