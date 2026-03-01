'use client';

import { useState, useEffect, useMemo } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { API_BASE_URL } from '@/lib/api';
import { earningsCalendar, logos } from '@/lib/api/routes';
import type { ApiEnvelope, EarningsCalendarResponse, EarningsResult } from '@/lib/types/earnings';
import { formatEPS, parseDateLocal } from '@/lib/utils/earningsFormatters';
import { useWidgetTracking } from '@/lib/hooks/useWidgetTracking';
import { CalendarDaysIcon } from '@heroicons/react/24/outline';

// ============================================================================
// Date Helpers
// ============================================================================

/** Format YYYY-MM-DD for API queries. */
function formatDateISO(date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

/**
 * Get the start date for the earnings query.
 * If current ET time is before 4:00 PM (market close), start from today.
 * If >= 4:00 PM ET, start from tomorrow.
 */
function getStartDate(): Date {
  const now = new Date();
  // Convert to ET by using toLocaleString with America/New_York timezone
  const etString = now.toLocaleString('en-US', { timeZone: 'America/New_York' });
  const etDate = new Date(etString);
  const etHour = etDate.getHours();

  const startDate = new Date(now);
  if (etHour >= 16) {
    startDate.setDate(startDate.getDate() + 1);
  }
  startDate.setHours(0, 0, 0, 0);
  return startDate;
}

/**
 * Add `tradingDays` weekdays to a start date and return the end date.
 * Skips Saturdays and Sundays.
 */
function addTradingDays(start: Date, tradingDays: number): Date {
  const d = new Date(start);
  let added = 0;
  while (added < tradingDays) {
    d.setDate(d.getDate() + 1);
    const dow = d.getDay();
    if (dow !== 0 && dow !== 6) {
      added++;
    }
  }
  return d;
}

/** Format a date like "Monday, Mar 2". */
function formatDateHeader(date: Date): string {
  return date.toLocaleDateString('en-US', {
    weekday: 'long',
    month: 'short',
    day: 'numeric',
  });
}

/** Map raw time strings to display-friendly badges. */
function getTimeBadge(time: string | undefined): { label: string; className: string } | null {
  if (!time || time === '--') return null;
  const lower = time.toLowerCase();
  if (lower === 'bmo') {
    return {
      label: 'BMO',
      className: 'bg-amber-500/15 text-amber-400',
    };
  }
  if (lower === 'amc') {
    return {
      label: 'AMC',
      className: 'bg-indigo-500/15 text-indigo-400',
    };
  }
  if (lower === 'dmh') {
    return {
      label: 'DMH',
      className: 'bg-ic-text-dim/15 text-ic-text-dim',
    };
  }
  return null;
}

// ============================================================================
// Types
// ============================================================================

interface EarningsGroup {
  date: string;
  label: string;
  items: EarningsResult[];
}

// ============================================================================
// Component
// ============================================================================

const MAX_ITEMS = 10;

export default function UpcomingEarnings() {
  const { ref: widgetRef, trackInteraction } = useWidgetTracking('upcoming_earnings');
  const [earnings, setEarnings] = useState<EarningsResult[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [failedLogos, setFailedLogos] = useState<Set<string>>(new Set());

  useEffect(() => {
    const controller = new AbortController();

    const fetchEarnings = async () => {
      try {
        setLoading(true);
        setError(null);

        const startDate = getStartDate();
        const endDate = addTradingDays(startDate, 5);

        const from = formatDateISO(startDate);
        const to = formatDateISO(endDate);

        const headers: Record<string, string> = {};
        if (typeof window !== 'undefined') {
          const token = localStorage.getItem('auth_token');
          if (token) {
            headers['Authorization'] = `Bearer ${token}`;
          }
        }

        const response = await fetch(
          `${API_BASE_URL}${earningsCalendar.list}?from=${from}&to=${to}`,
          {
            cache: 'no-store',
            signal: controller.signal,
            headers,
          }
        );

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`);
        }

        const result: ApiEnvelope<EarningsCalendarResponse> = await response.json();
        setEarnings(result.data.earnings ?? []);
      } catch (err) {
        if (err instanceof DOMException && err.name === 'AbortError') return;
        console.error('Error fetching upcoming earnings:', err);
        setError('Failed to load upcoming earnings');
      } finally {
        setLoading(false);
      }
    };

    fetchEarnings();
    return () => controller.abort();
  }, []);

  // Group the top N earnings by date
  const groups: EarningsGroup[] = useMemo(() => {
    // Take top MAX_ITEMS items (API already sorts by date)
    const topItems = earnings.slice(0, MAX_ITEMS);

    // Group by date
    const dateMap = new Map<string, EarningsResult[]>();
    for (const item of topItems) {
      const existing = dateMap.get(item.date);
      if (existing) {
        existing.push(item);
      } else {
        dateMap.set(item.date, [item]);
      }
    }

    // Convert to array sorted by date
    return Array.from(dateMap.entries())
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([date, items]) => ({
        date,
        label: formatDateHeader(parseDateLocal(date)),
        items,
      }));
  }, [earnings]);

  const handleLogoError = (symbol: string) => {
    setFailedLogos((prev) => new Set(prev).add(symbol));
  };

  // ── Loading State ──────────────────────────────────────────────────────────
  if (loading) {
    return (
      <div
        className="bg-ic-surface rounded-lg border border-ic-border p-6"
        style={{ boxShadow: 'var(--ic-shadow-card)' }}
      >
        <div className="flex items-center gap-2 mb-4">
          <CalendarDaysIcon className="h-5 w-5 text-ic-text-muted" />
          <h2 className="text-lg font-semibold text-ic-text-primary">Upcoming Earnings</h2>
        </div>
        <div className="animate-pulse space-y-3">
          <div className="h-4 bg-ic-bg-tertiary rounded w-32"></div>
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className="flex items-center gap-3 py-2">
              <div className="h-8 w-8 bg-ic-bg-tertiary rounded-full flex-shrink-0"></div>
              <div className="flex-1 space-y-1.5">
                <div className="h-3.5 bg-ic-bg-tertiary rounded w-20"></div>
                <div className="h-3 bg-ic-bg-tertiary rounded w-32"></div>
              </div>
              <div className="h-3.5 bg-ic-bg-tertiary rounded w-14"></div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  // ── Error State ────────────────────────────────────────────────────────────
  if (error) {
    return (
      <div
        className="bg-ic-surface rounded-lg border border-ic-border p-6"
        style={{ boxShadow: 'var(--ic-shadow-card)' }}
      >
        <div className="flex items-center gap-2 mb-4">
          <CalendarDaysIcon className="h-5 w-5 text-ic-text-muted" />
          <h2 className="text-lg font-semibold text-ic-text-primary">Upcoming Earnings</h2>
        </div>
        <div className="text-ic-negative text-sm">
          <p>{error}</p>
          <p className="text-ic-text-muted mt-2">
            This will work once the backend is running with earnings data.
          </p>
        </div>
      </div>
    );
  }

  // ── Empty State ────────────────────────────────────────────────────────────
  if (groups.length === 0) {
    return (
      <div
        ref={widgetRef}
        className="bg-ic-surface rounded-lg border border-ic-border p-6"
        style={{ boxShadow: 'var(--ic-shadow-card)' }}
      >
        <div className="flex items-center gap-2 mb-4">
          <CalendarDaysIcon className="h-5 w-5 text-ic-text-muted" />
          <h2 className="text-lg font-semibold text-ic-text-primary">Upcoming Earnings</h2>
        </div>
        <p className="text-ic-text-muted text-sm text-center py-6">
          No upcoming earnings scheduled
        </p>
        <div className="pt-3 border-t border-ic-border-subtle">
          <Link
            href="/earnings-calendar"
            className="text-sm text-ic-blue hover:text-ic-blue-hover font-medium transition-colors"
          >
            View Full Calendar →
          </Link>
        </div>
      </div>
    );
  }

  // ── Main Render ────────────────────────────────────────────────────────────
  return (
    <div
      ref={widgetRef}
      className="bg-ic-surface rounded-lg border border-ic-border p-6"
      style={{ boxShadow: 'var(--ic-shadow-card)' }}
    >
      {/* Header */}
      <div className="flex items-center gap-2 mb-4">
        <CalendarDaysIcon className="h-5 w-5 text-ic-text-muted" />
        <h2 className="text-lg font-semibold text-ic-text-primary">Upcoming Earnings</h2>
      </div>

      {/* Grouped Earnings List */}
      <div className="space-y-3">
        {groups.map((group) => (
          <div key={group.date}>
            {/* Date Header */}
            <h3 className="text-xs font-medium text-ic-text-muted uppercase tracking-wider mb-1.5">
              {group.label}
            </h3>

            {/* Earnings rows */}
            <div className="space-y-0.5">
              {group.items.map((item) => {
                const badge = getTimeBadge(item.fiscalQuarter);
                const showFallbackLogo = failedLogos.has(item.symbol);
                // Generate a consistent color from the symbol
                const fallbackColor = symbolToColor(item.symbol);

                return (
                  <Link
                    key={`${item.symbol}-${item.date}`}
                    href={`/ticker/${item.symbol}?tab=earnings`}
                    onClick={() =>
                      trackInteraction('earnings_click', { symbol: item.symbol, date: item.date })
                    }
                    className="flex items-center gap-3 py-2 px-2 -mx-2 rounded-lg hover:bg-ic-surface-hover transition-colors"
                  >
                    {/* Logo */}
                    {showFallbackLogo ? (
                      <div
                        className="w-8 h-8 flex-shrink-0 rounded-full flex items-center justify-center"
                        style={{ backgroundColor: fallbackColor }}
                      >
                        <span className="text-xs font-bold text-white">
                          {item.symbol.charAt(0)}
                        </span>
                      </div>
                    ) : (
                      <div className="w-8 h-8 flex-shrink-0 relative bg-white dark:bg-gray-800 rounded-full overflow-hidden">
                        <Image
                          src={`${API_BASE_URL}${logos.bySymbol(item.symbol)}`}
                          alt={`${item.symbol} logo`}
                          fill
                          className="object-contain p-0.5"
                          unoptimized
                          onError={() => handleLogoError(item.symbol)}
                        />
                      </div>
                    )}

                    {/* Ticker + Company Name */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="font-semibold text-sm text-ic-text-primary">
                          {item.symbol}
                        </span>
                        {badge && (
                          <span
                            className={`text-[10px] font-semibold px-1.5 py-0.5 rounded ${badge.className}`}
                          >
                            {badge.label}
                          </span>
                        )}
                      </div>
                      {item.fiscalQuarter && (
                        <span className="text-xs text-ic-text-muted truncate block">
                          {item.fiscalQuarter}
                        </span>
                      )}
                    </div>

                    {/* EPS Estimate */}
                    <div className="text-right flex-shrink-0">
                      {item.epsEstimated != null ? (
                        <div>
                          <span className="text-xs text-ic-text-dim">Est.</span>{' '}
                          <span className="text-sm text-ic-text-secondary tabular-nums">
                            {formatEPS(item.epsEstimated)}
                          </span>
                        </div>
                      ) : (
                        <span className="text-xs text-ic-text-dim">--</span>
                      )}
                    </div>
                  </Link>
                );
              })}
            </div>
          </div>
        ))}
      </div>

      {/* Footer Link */}
      <div className="mt-4 pt-3 border-t border-ic-border-subtle">
        <Link
          href="/earnings-calendar"
          onClick={() => trackInteraction('view_full_calendar')}
          className="text-sm text-ic-blue hover:text-ic-blue-hover font-medium transition-colors"
        >
          View Full Calendar →
        </Link>
      </div>
    </div>
  );
}

// ============================================================================
// Helpers
// ============================================================================

/** Generate a deterministic color from a symbol string for fallback logos. */
function symbolToColor(symbol: string): string {
  let hash = 0;
  for (let i = 0; i < symbol.length; i++) {
    hash = symbol.charCodeAt(i) + ((hash << 5) - hash);
  }
  const hue = Math.abs(hash % 360);
  return `hsl(${hue}, 55%, 45%)`;
}
