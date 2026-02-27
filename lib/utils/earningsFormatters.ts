/**
 * Shared formatting helpers for earnings data.
 *
 * Used by per-stock EarningsTab components and the Earnings Calendar page
 * to ensure consistent display of EPS, revenue, and surprise values.
 */

/** Format an EPS value as $X.XX, or a fallback string for nullish values. */
export function formatEPS(value: number | null | undefined, fallback: string = '—'): string {
  if (value === null || value === undefined) return fallback;
  return `$${value.toFixed(2)}`;
}

/** Format a revenue value with T/B/M/K suffixes, or a fallback string for nullish values. */
export function formatRevenue(value: number | null | undefined, fallback: string = '—'): string {
  if (value === null || value === undefined) return fallback;
  const abs = Math.abs(value);
  if (abs >= 1e12) return `$${(value / 1e12).toFixed(1)}T`;
  if (abs >= 1e9) return `$${(value / 1e9).toFixed(1)}B`;
  if (abs >= 1e6) return `$${(value / 1e6).toFixed(1)}M`;
  if (abs >= 1e3) return `$${(value / 1e3).toFixed(0)}K`;
  return `$${value.toFixed(0)}`;
}

/** Format a surprise percentage with sign and %, or a fallback string for nullish values. */
export function formatSurprise(value: number | null | undefined, fallback: string = '—'): string {
  if (value === null || value === undefined) return fallback;
  const sign = value > 0 ? '+' : '';
  return `${sign}${value.toFixed(1)}%`;
}

/** Return a Tailwind color class for a surprise percentage value. */
export function surpriseColor(value: number | null | undefined): string {
  if (value === null || value === undefined) return 'text-ic-text-dim';
  if (value > 0.5) return 'text-green-400';
  if (value < -0.5) return 'text-red-400';
  return 'text-ic-text-dim';
}

/**
 * Parse a YYYY-MM-DD date string into a local Date without timezone shift.
 *
 * Appending T12:00:00 avoids the midnight-UTC edge case where
 * `new Date("2026-03-15T00:00:00")` in UTC-X timezones rolls back to the
 * previous calendar day.
 */
export function parseDateLocal(dateStr: string): Date {
  return new Date(dateStr + 'T12:00:00');
}
