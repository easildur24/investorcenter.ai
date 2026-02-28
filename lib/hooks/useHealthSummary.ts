'use client';

import { useState, useEffect } from 'react';
import { API_BASE_URL } from '@/lib/api';
import { stocks } from '@/lib/api/routes';
import type { HealthSummaryResponse } from '@/lib/types/fundamentals';

interface UseHealthSummaryResult {
  data: HealthSummaryResponse | null;
  loading: boolean;
  error: string | null;
}

/**
 * Fetches the fundamental health summary for a given ticker.
 *
 * Uses a 50ms delay so it does not block the initial paint of the page.
 * Handles 404 gracefully (returns null data, no error).
 */
export function useHealthSummary(ticker: string): UseHealthSummaryResult {
  const [data, setData] = useState<HealthSummaryResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!ticker) {
      setLoading(false);
      return;
    }

    let cancelled = false;

    const fetchHealth = async () => {
      try {
        setLoading(true);
        setError(null);

        const response = await fetch(
          `${API_BASE_URL}${stocks.healthSummary(ticker.toUpperCase())}`
        );

        if (cancelled) return;

        if (response.status === 404) {
          // No health data available — not an error
          setData(null);
          return;
        }

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: Failed to fetch health summary`);
        }

        const result = await response.json();
        if (!cancelled) {
          setData(result.data ?? result);
        }
      } catch (err) {
        if (!cancelled) {
          console.error(`Error fetching health summary for ${ticker}:`, err);
          setError(err instanceof Error ? err.message : 'Failed to load health summary');
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    // Slight delay to not block initial paint
    const timer = setTimeout(fetchHealth, 50);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [ticker]);

  return { data, loading, error };
}
