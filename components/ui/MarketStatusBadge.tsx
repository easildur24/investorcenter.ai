'use client';

import type { MarketStatus } from '@/lib/types/market';

interface MarketStatusBadgeProps {
  status: MarketStatus;
  /** Show countdown text alongside label */
  showCountdown?: boolean;
  /** Additional className */
  className?: string;
}

const COLOR_MAP = {
  green: {
    dot: 'bg-green-500',
    text: 'text-green-600 dark:text-green-400',
    bg: 'bg-green-500/10',
  },
  amber: {
    dot: 'bg-amber-500',
    text: 'text-amber-600 dark:text-amber-400',
    bg: 'bg-amber-500/10',
  },
  grey: {
    dot: 'bg-gray-400',
    text: 'text-gray-500 dark:text-gray-400',
    bg: 'bg-gray-500/10',
  },
} as const;

/**
 * Colored dot + label badge indicating current market session.
 * Uses pulse animation when market is open.
 */
export default function MarketStatusBadge({
  status,
  showCountdown = true,
  className = '',
}: MarketStatusBadgeProps) {
  const colors = COLOR_MAP[status.color];
  const isOpen = status.state === 'regular';

  return (
    <div
      className={`inline-flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-medium ${colors.bg} ${colors.text} ${className}`}
      role="status"
      aria-live="polite"
    >
      <span className={`relative flex h-2 w-2`}>
        {isOpen && (
          <span
            className={`animate-ping absolute inline-flex h-full w-full rounded-full opacity-75 ${colors.dot}`}
          />
        )}
        <span className={`relative inline-flex rounded-full h-2 w-2 ${colors.dot}`} />
      </span>
      <span>{status.label}</span>
      {showCountdown && status.nextEvent && (
        <span className="opacity-75">— {status.nextEvent}</span>
      )}
    </div>
  );
}
