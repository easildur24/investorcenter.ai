'use client';

import type { SentimentLabel } from '@/lib/types/sentiment';

const BADGE_STYLES: Record<SentimentLabel, string> = {
  bullish: 'bg-green-100 text-green-800 border-green-200',
  neutral: 'bg-gray-100 text-gray-700 border-gray-200',
  bearish: 'bg-red-100 text-red-800 border-red-200',
};

const BADGE_ICONS: Record<SentimentLabel, string> = {
  bullish: '\u25B2', // ▲
  neutral: '\u25CF', // ●
  bearish: '\u25BC', // ▼
};

interface SentimentBadgeProps {
  label: SentimentLabel;
  /** Bullish percentage (0 to 1 scale). If omitted, no percentage is shown. */
  bullishPct?: number;
  /** Size variant */
  size?: 'sm' | 'md';
}

/**
 * Consistent sentiment badge that renders identically regardless of rank.
 * BUG-002 fix: Replaces the old emoji-based score icons that varied by rank.
 */
export default function SentimentBadge({ label, bullishPct, size = 'sm' }: SentimentBadgeProps) {
  const sizeClasses = size === 'sm' ? 'px-2 py-0.5 text-xs' : 'px-2.5 py-1 text-sm';

  return (
    <span
      className={`inline-flex items-center rounded font-medium border ${sizeClasses} ${BADGE_STYLES[label]}`}
      data-testid="sentiment-badge"
    >
      <span className="mr-1">{BADGE_ICONS[label]}</span>
      {label.charAt(0).toUpperCase() + label.slice(1)}
      {bullishPct !== undefined && (
        <span className="ml-1 opacity-70">{Math.round(bullishPct * 100)}%</span>
      )}
    </span>
  );
}
