'use client';

import { useState, useEffect } from 'react';
import { API_BASE_URL } from '@/lib/api';
import { stocks } from '@/lib/api/routes';
import type { FairValueResponse } from '@/lib/types/fundamentals';

interface UseFairValueResult {
  data: FairValueResponse | null;
  loading: boolean;
  error: string | null;
}

/**
 * Fetches fair value data on mount (eager, not lazy).
 * Handles 404 (no data available) and suppression gracefully.
 */
export function useFairValue(ticker: string): UseFairValueResult {
  const [data, setData] = useState<FairValueResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function fetchFairValue() {
      setLoading(true);
      setError(null);

      try {
        const response = await fetch(`${API_BASE_URL}${stocks.fairValue(ticker)}`);

        if (response.status === 404) {
          // No fair value data — not an error, just no data
          setData(null);
          return;
        }

        if (!response.ok) {
          throw new Error(`Failed to fetch fair value: ${response.status}`);
        }

        const result = await response.json();
        if (!cancelled) {
          setData(result.data ?? result);
        }
      } catch (err) {
        if (!cancelled) {
          const message = err instanceof Error ? err.message : 'Failed to load fair value data';
          setError(message);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    fetchFairValue();

    return () => {
      cancelled = true;
    };
  }, [ticker]);

  return { data, loading, error };
}
