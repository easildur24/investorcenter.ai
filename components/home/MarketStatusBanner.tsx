'use client';

import { useState, useEffect } from 'react';
import { getMarketStatus, getCountdown } from '@/lib/market-status';
import type { MarketStatus } from '@/lib/types/market';

const COLOR_MAP = {
  green: {
    dot: 'bg-green-500',
    text: 'text-green-700 dark:text-green-300',
    bg: 'bg-green-500/5',
    border: 'border-green-500/20',
  },
  amber: {
    dot: 'bg-amber-500',
    text: 'text-amber-700 dark:text-amber-300',
    bg: 'bg-amber-500/5',
    border: 'border-amber-500/20',
  },
  grey: {
    dot: 'bg-gray-400',
    text: 'text-gray-600 dark:text-gray-400',
    bg: 'bg-gray-500/5',
    border: 'border-gray-500/20',
  },
} as const;

export default function MarketStatusBanner() {
  const [status, setStatus] = useState<MarketStatus>(() => getMarketStatus());
  const [countdown, setCountdown] = useState<string>(() =>
    getCountdown(getMarketStatus().nextEventTime)
  );

  useEffect(() => {
    // Update market status and countdown every 60 seconds
    // 60s refresh is sufficient — countdown displays hours/minutes format, not seconds
    const interval = setInterval(() => {
      const newStatus = getMarketStatus();
      setStatus(newStatus);
      setCountdown(getCountdown(newStatus.nextEventTime));
    }, 60_000);

    return () => clearInterval(interval);
  }, []);

  const colors = COLOR_MAP[status.color];
  const isOpen = status.state === 'regular';

  return (
    <div
      className={`w-full ${colors.bg} border-b ${colors.border}`}
      role="status"
      aria-live="polite"
    >
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-center gap-2 py-2">
          {/* Colored dot with pulse when market is open */}
          <span className="relative flex h-2 w-2">
            {isOpen && (
              <span
                className={`animate-ping absolute inline-flex h-full w-full rounded-full opacity-75 ${colors.dot}`}
              />
            )}
            <span className={`relative inline-flex rounded-full h-2 w-2 ${colors.dot}`} />
          </span>

          <span className={`text-xs font-medium ${colors.text}`}>{status.label}</span>

          {status.nextEvent && (
            <>
              <span className={`text-xs ${colors.text} opacity-60`}>&mdash;</span>
              <span className={`text-xs ${colors.text} opacity-75`}>{status.nextEvent}</span>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
