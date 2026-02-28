'use client';

import { useState } from 'react';
import { cn } from '@/lib/utils';
import type { RedFlagSeverity } from '@/lib/types/fundamentals';

interface RedFlagBadgeProps {
  id: string;
  severity: RedFlagSeverity;
  title: string;
  description: string;
  relatedMetrics: string[];
  defaultExpanded?: boolean;
}

const severityConfig: Record<
  RedFlagSeverity,
  { border: string; bg: string; icon: string; iconColor: string }
> = {
  high: {
    border: 'border-red-500/30',
    bg: 'bg-red-500/5',
    icon: '\u26D4', // no entry icon (red circle)
    iconColor: 'text-red-400',
  },
  medium: {
    border: 'border-orange-500/30',
    bg: 'bg-orange-500/5',
    icon: '\u26A0', // warning triangle
    iconColor: 'text-orange-400',
  },
  low: {
    border: 'border-yellow-500/30',
    bg: 'bg-yellow-500/5',
    icon: '\u24D8', // circled info
    iconColor: 'text-yellow-400',
  },
};

export function RedFlagBadge({
  id,
  severity,
  title,
  description,
  relatedMetrics,
  defaultExpanded = false,
}: RedFlagBadgeProps) {
  const [expanded, setExpanded] = useState(defaultExpanded);
  const config = severityConfig[severity];

  return (
    <div
      className={cn(
        'border rounded-lg overflow-hidden transition-all duration-200',
        config.border,
        config.bg
      )}
      data-flag-id={id}
    >
      {/* Collapsed header — always visible */}
      <button
        type="button"
        onClick={() => setExpanded((prev) => !prev)}
        aria-expanded={expanded}
        aria-controls={`flag-body-${id}`}
        className="w-full flex items-center gap-2 px-3 py-2.5 text-left"
      >
        <span className={cn('text-base flex-shrink-0', config.iconColor)} aria-hidden="true">
          {config.icon}
        </span>
        <span className="flex-1 text-sm font-medium text-ic-text-primary truncate">{title}</span>
        <svg
          className={cn(
            'w-4 h-4 text-ic-text-dim flex-shrink-0 transition-transform duration-200',
            expanded && 'rotate-180'
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

      {/* Expandable body */}
      <div
        id={`flag-body-${id}`}
        className={cn(
          'transition-all duration-200 overflow-hidden',
          expanded ? 'max-h-96 opacity-100' : 'max-h-0 opacity-0'
        )}
      >
        <div className="px-3 pb-3 pt-0 space-y-2">
          <p className="text-sm text-ic-text-muted leading-relaxed">{description}</p>
          {relatedMetrics.length > 0 && (
            <div className="flex flex-wrap gap-1.5">
              {relatedMetrics.map((metric) => (
                <span
                  key={metric}
                  className="text-xs px-1.5 py-0.5 rounded bg-white/5 text-ic-text-dim"
                >
                  {metric}
                </span>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
