import useSWR from 'swr';
import type { ScreenerApiParams, ScreenerResponse } from '@/lib/types/screener';
import { API_BASE_URL } from '@/lib/api';

/** Build a URL with query params, omitting undefined/null/empty values. */
function buildScreenerUrl(params: ScreenerApiParams): string {
  const searchParams = new URLSearchParams();

  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      searchParams.append(key, String(value));
    }
  });

  const qs = searchParams.toString();
  return `${API_BASE_URL}/screener/stocks${qs ? `?${qs}` : ''}`;
}

/** SWR fetcher for screener data. */
async function screenerFetcher(url: string): Promise<ScreenerResponse> {
  const res = await fetch(url);
  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: `HTTP ${res.status}` }));
    throw new Error(error.error || error.message || `HTTP ${res.status}`);
  }
  return res.json();
}

/**
 * useScreener — SWR-powered hook for server-side screener queries.
 *
 * Usage:
 *   const { data, isLoading, error } = useScreener({ page: 1, limit: 25, sort: 'market_cap', order: 'desc' });
 *
 * SWR automatically deduplicates, caches, and revalidates.
 */
export function useScreener(params: ScreenerApiParams) {
  const url = buildScreenerUrl(params);

  const { data, error, isLoading, isValidating, mutate } = useSWR<ScreenerResponse>(
    url,
    screenerFetcher,
    {
      revalidateOnFocus: false, // screener data doesn't need live refresh
      keepPreviousData: true, // show stale data while new filters load
      dedupingInterval: 2000, // dedup rapid filter changes
    }
  );

  return {
    stocks: data?.data ?? [],
    meta: data?.meta ?? null,
    isLoading,
    isValidating,
    error: error as Error | undefined,
    mutate,
  };
}

export { buildScreenerUrl };
