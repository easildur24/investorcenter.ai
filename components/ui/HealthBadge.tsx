'use client';

import { cn } from '@/lib/utils';
import type { HealthBadgeLevel } from '@/lib/types/fundamentals';

interface HealthBadgeProps {
  badge: HealthBadgeLevel;
  score: number;
  size?: 'sm' | 'md' | 'lg';
  /** When true the numeric score is hidden (free-tier). */
  hideScore?: boolean;
}

const badgeColors: Record<HealthBadgeLevel, { bg: string; text: string; bar: string }> = {
  Strong: { bg: 'bg-green-500/15', text: 'text-green-400', bar: 'bg-green-400' },
  Healthy: { bg: 'bg-green-400/15', text: 'text-green-300', bar: 'bg-green-300' },
  Fair: { bg: 'bg-yellow-500/15', text: 'text-yellow-400', bar: 'bg-yellow-400' },
  Weak: { bg: 'bg-orange-500/15', text: 'text-orange-400', bar: 'bg-orange-400' },
  Distressed: { bg: 'bg-red-500/15', text: 'text-red-400', bar: 'bg-red-400' },
};

const sizeClasses: Record<string, { wrapper: string; text: string; bar: string }> = {
  sm: { wrapper: 'px-2 py-1 gap-1.5', text: 'text-xs', bar: 'h-1 w-10' },
  md: { wrapper: 'px-3 py-1.5 gap-2', text: 'text-sm', bar: 'h-1.5 w-14' },
  lg: { wrapper: 'px-4 py-2 gap-2.5', text: 'text-base', bar: 'h-2 w-20' },
};

export function HealthBadge({ badge, score, size = 'md', hideScore = false }: HealthBadgeProps) {
  const colors = badgeColors[badge];
  const sizes = sizeClasses[size];
  const clampedScore = Math.max(0, Math.min(100, score));

  return (
    <div
      role="status"
      aria-label={`Financial health: ${badge}${hideScore ? '' : `, score ${clampedScore} out of 100`}`}
      className={cn(
        'inline-flex items-center rounded-full font-medium',
        colors.bg,
        colors.text,
        sizes.wrapper
      )}
    >
      <span className={sizes.text}>
        {badge}
        {!hideScore && <span className="ml-1 opacity-75">{clampedScore}</span>}
      </span>

      {/* Progress bar */}
      <div className={cn('rounded-full bg-white/10 overflow-hidden', sizes.bar)} aria-hidden="true">
        <div
          className={cn('h-full rounded-full transition-all duration-300', colors.bar)}
          style={{ width: `${clampedScore}%` }}
        />
      </div>
    </div>
  );
}
