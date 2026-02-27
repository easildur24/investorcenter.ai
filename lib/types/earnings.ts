// Earnings data types matching backend response shapes

/**
 * Generic wrapper for the standard `{ data: T, meta: {...} }` envelope
 * returned by the Go backend. Keeps callers type-safe without hand-casting
 * `result.data` after every fetch.
 */
export interface ApiEnvelope<T> {
  data: T;
  meta: Record<string, unknown>;
}

export interface EarningsResult {
  symbol: string;
  date: string;
  fiscalQuarter: string;
  epsEstimated: number | null;
  epsActual: number | null;
  epsSurprisePercent: number | null;
  epsBeat: boolean | null;
  revenueEstimated: number | null;
  revenueActual: number | null;
  revenueSurprisePercent: number | null;
  revenueBeat: boolean | null;
  isUpcoming: boolean;
}

export interface BeatRate {
  epsBeats: number;
  revenueBeats: number;
  totalQuarters: number;
  totalRevenueQuarters: number;
}

export interface EarningsResponse {
  earnings: EarningsResult[];
  nextEarnings: EarningsResult | null;
  mostRecentEarnings: EarningsResult | null;
  beatRate: BeatRate | null;
}

export interface EarningsCalendarResponse {
  earnings: EarningsResult[];
  earningsCounts: Record<string, number>;
}
