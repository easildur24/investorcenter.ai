'use client';

import { useState, useCallback, useRef } from 'react';
import { API_BASE_URL } from '@/lib/api';
import { stocks } from '@/lib/api/routes';
import type { PeersResponse } from '@/lib/types/fundamentals';

interface UsePeerComparisonOptions {
  ticker: string;
  enabled?: boolean;
}

interface UsePeerComparisonResult {
  data: PeersResponse | null;
  loading: boolean;
  error: string | null;
  fetch: () => void;
}

/**
 * Lazy-loaded hook for fetching peer comparison data.
 * Only fetches when `enabled` is true (triggered on panel expand)
 * or when `fetch()` is called manually.
 */
export function usePeerComparison({
  ticker,
  enabled = false,
}: UsePeerComparisonOptions): UsePeerComparisonResult {
  const [data, setData] = useState<PeersResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const hasFetched = useRef(false);

  const fetchPeers = useCallback(async () => {
    if (hasFetched.current && data) return;
    hasFetched.current = true;

    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE_URL}${stocks.peers(ticker)}`);

      if (!response.ok) {
        throw new Error(`Failed to fetch peers: ${response.status}`);
      }

      const result = await response.json();
      setData(result.data ?? result);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load peer data';
      setError(message);
      hasFetched.current = false; // allow retry
    } finally {
      setLoading(false);
    }
  }, [ticker, data]);

  // Auto-fetch when enabled becomes true
  if (enabled && !hasFetched.current && !loading) {
    fetchPeers();
  }

  return { data, loading, error, fetch: fetchPeers };
}
