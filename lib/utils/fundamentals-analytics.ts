/**
 * Fundamentals event tracking system
 *
 * Queues analytics events for batch sending with SSR safety.
 * Events are stored in window.__fundamentalsEvents for collection
 * by the analytics pipeline.
 */

import type { FundamentalsEvent } from '@/lib/types/fundamentals';

declare global {
  interface Window {
    __fundamentalsEvents?: FundamentalsEvent[];
  }
}

/**
 * All trackable fundamentals event actions
 */
export const FUNDAMENTALS_EVENTS = {
  HEALTH_CARD_VIEWED: 'health_card_viewed',
  HEALTH_CARD_EXPANDED: 'health_card_expanded',
  PERCENTILE_BAR_HOVERED: 'percentile_bar_hovered',
  RED_FLAG_EXPANDED: 'red_flag_expanded',
  PEER_PANEL_EXPANDED: 'peer_panel_expanded',
  SPARKLINE_CLICKED: 'sparkline_clicked',
  METRIC_HISTORY_VIEWED: 'metric_history_viewed',
  FAIR_VALUE_GAUGE_VIEWED: 'fair_value_gauge_viewed',
  PEER_TICKER_CLICKED: 'peer_ticker_clicked',
  PAYWALL_IMPRESSION: 'paywall_impression',
  PAYWALL_CTA_CLICKED: 'paywall_cta_clicked',
  PAYWALL_DISMISSED: 'paywall_dismissed',
  FUNDAMENTALS_TO_WATCHLIST: 'fundamentals_to_watchlist',
  FUNDAMENTALS_TO_SCREENER: 'fundamentals_to_screener',
} as const;

export type FundamentalsEventAction =
  (typeof FUNDAMENTALS_EVENTS)[keyof typeof FUNDAMENTALS_EVENTS];

/**
 * Track a fundamentals analytics event.
 *
 * Events are queued in window.__fundamentalsEvents for batch collection.
 * Safe to call during SSR (no-ops when window is unavailable).
 *
 * @param action - Event action name from FUNDAMENTALS_EVENTS
 * @param ticker - Ticker symbol for context
 * @param metadata - Optional additional event data
 */
export function trackFundamentalsEvent(
  action: string,
  ticker: string,
  metadata?: Record<string, unknown>
): void {
  // SSR safety: skip if window is not available
  if (typeof window === 'undefined') {
    return;
  }

  // Initialize the events array if it doesn't exist
  if (!window.__fundamentalsEvents) {
    window.__fundamentalsEvents = [];
  }

  const event: FundamentalsEvent = {
    action,
    ticker,
    timestamp: new Date().toISOString(),
    metadata,
  };

  window.__fundamentalsEvents.push(event);
}

/**
 * Retrieve and flush all queued fundamentals events.
 * Returns the events and clears the queue.
 *
 * @returns Array of queued events, or empty array if none
 */
export function flushFundamentalsEvents(): FundamentalsEvent[] {
  if (typeof window === 'undefined' || !window.__fundamentalsEvents) {
    return [];
  }

  const events = [...window.__fundamentalsEvents];
  window.__fundamentalsEvents = [];
  return events;
}

/**
 * Get the count of queued events without flushing.
 *
 * @returns Number of queued events
 */
export function getFundamentalsEventCount(): number {
  if (typeof window === 'undefined' || !window.__fundamentalsEvents) {
    return 0;
  }

  return window.__fundamentalsEvents.length;
}
