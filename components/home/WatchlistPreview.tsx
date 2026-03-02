'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { apiClient } from '@/lib/api';
import { useWidgetTracking } from '@/lib/hooks/useWidgetTracking';
import { useAuth } from '@/lib/auth/AuthContext';
import { safeToFixed } from '@/lib/utils';
import { EyeIcon, ArrowTrendingUpIcon, ArrowTrendingDownIcon } from '@heroicons/react/24/outline';

interface WatchlistItem {
  symbol: string;
  name?: string;
  price?: number;
  change?: number;
  changePercent?: number;
}

interface Watchlist {
  id: string;
  name: string;
  items?: WatchlistItem[];
  item_count?: number;
}

/**
 * WatchlistPreview — shows a compact view of the user's default watchlist.
 * Only renders for logged-in users.
 */
export default function WatchlistPreview() {
  const { ref: widgetRef, trackInteraction } = useWidgetTracking('watchlist_preview');
  const { user } = useAuth();
  const isLoggedIn = !!user;
  const [watchlist, setWatchlist] = useState<Watchlist | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!isLoggedIn) {
      setLoading(false);
      return;
    }

    let isMounted = true;

    const fetchWatchlist = async () => {
      try {
        setLoading(true);
        // Fetch all watchlists and show the first one
        const response = await apiClient.request<Watchlist[]>('/watchlists');
        if (!isMounted) return;

        const watchlists: Watchlist[] = response?.data || [];
        if (watchlists.length > 0) {
          // Fetch the first watchlist details
          const detailResponse = await apiClient.request<Watchlist>(
            `/watchlists/${watchlists[0].id}`
          );
          if (!isMounted) return;
          setWatchlist(detailResponse?.data || watchlists[0]);
        }
        setError(null);
      } catch (err) {
        if (!isMounted) return;
        setError(err instanceof Error ? err.message : 'Failed to fetch watchlist');
        console.error('Error fetching watchlist:', err);
      } finally {
        if (isMounted) setLoading(false);
      }
    };

    fetchWatchlist();

    return () => {
      isMounted = false;
    };
  }, [isLoggedIn]);

  // Don't render anything for logged-out users
  if (!isLoggedIn) {
    return null;
  }

  if (loading) {
    return (
      <div
        className="bg-ic-surface rounded-lg border border-ic-border p-6"
        style={{ boxShadow: 'var(--ic-shadow-card)' }}
      >
        <h2 className="text-lg font-semibold text-ic-text-primary mb-4">My Watchlist</h2>
        <div className="animate-pulse space-y-3">
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className="flex justify-between items-center py-2">
              <div className="h-4 bg-ic-bg-tertiary rounded w-16"></div>
              <div className="h-4 bg-ic-bg-tertiary rounded w-20"></div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div
        className="bg-ic-surface rounded-lg border border-ic-border p-6"
        style={{ boxShadow: 'var(--ic-shadow-card)' }}
      >
        <h2 className="text-lg font-semibold text-ic-text-primary mb-4">My Watchlist</h2>
        <p className="text-ic-text-muted text-sm">Unable to load watchlist.</p>
      </div>
    );
  }

  if (!watchlist || !watchlist.items || watchlist.items.length === 0) {
    return (
      <div
        ref={widgetRef}
        className="bg-ic-surface rounded-lg border border-ic-border p-6"
        style={{ boxShadow: 'var(--ic-shadow-card)' }}
      >
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-lg font-semibold text-ic-text-primary">My Watchlist</h2>
          <EyeIcon className="h-5 w-5 text-ic-text-muted" />
        </div>
        <div className="text-center py-6">
          <p className="text-ic-text-muted text-sm mb-3">
            Your watchlist is empty. Add stocks to track them here.
          </p>
          <Link
            href="/watchlist"
            className="text-sm text-ic-blue hover:text-ic-blue-hover font-medium transition-colors"
          >
            Go to Watchlist
          </Link>
        </div>
      </div>
    );
  }

  const items = watchlist.items.slice(0, 8); // Show max 8

  return (
    <div
      ref={widgetRef}
      className="bg-ic-surface rounded-lg border border-ic-border p-6"
      style={{ boxShadow: 'var(--ic-shadow-card)' }}
    >
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-lg font-semibold text-ic-text-primary">
          {watchlist.name || 'My Watchlist'}
        </h2>
        <EyeIcon className="h-5 w-5 text-ic-text-muted" />
      </div>

      <div className="space-y-1">
        {items.map((item) => (
          <Link
            key={item.symbol}
            href={`/ticker/${item.symbol}`}
            className="flex justify-between items-center py-2 px-2 -mx-2 rounded-lg hover:bg-ic-surface-hover transition-colors"
            onClick={() => trackInteraction('watchlist_click', { symbol: item.symbol })}
          >
            <div className="flex flex-col">
              <span className="font-semibold text-ic-text-primary text-sm">{item.symbol}</span>
              {item.name && (
                <span className="text-xs text-ic-text-muted truncate max-w-[100px]">
                  {item.name}
                </span>
              )}
            </div>

            <div className="flex items-center gap-3">
              {item.price != null && (
                <span className="text-sm text-ic-text-muted">${safeToFixed(item.price, 2)}</span>
              )}
              {item.changePercent != null && (
                <div
                  className={`flex items-center text-xs font-medium ${
                    item.changePercent >= 0 ? 'text-ic-positive' : 'text-ic-negative'
                  }`}
                >
                  {item.changePercent >= 0 ? (
                    <ArrowTrendingUpIcon className="h-3 w-3 mr-0.5" />
                  ) : (
                    <ArrowTrendingDownIcon className="h-3 w-3 mr-0.5" />
                  )}
                  <span>
                    {item.changePercent >= 0 ? '+' : ''}
                    {safeToFixed(item.changePercent, 2)}%
                  </span>
                </div>
              )}
            </div>
          </Link>
        ))}
      </div>

      <div className="mt-4 pt-3 border-t border-ic-border-subtle">
        <Link
          href="/watchlist"
          className="text-sm text-ic-blue hover:text-ic-blue-hover font-medium transition-colors"
          onClick={() => trackInteraction('view_full_watchlist')}
        >
          View Full Watchlist →
        </Link>
      </div>
    </div>
  );
}
