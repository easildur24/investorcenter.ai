'use client';

import { useState, useEffect } from 'react';
import { apiClient } from '@/lib/api';
import { safeToFixed, safeParseNumber } from '@/lib/utils';

interface IndexData {
  symbol: string;
  name: string;
  price: number | string;
  change: number | string;
  changePercent: number | string;
  lastUpdated: string;
  displayFormat?: 'points' | 'usd';
  dataType?: 'index' | 'etf_proxy';
}

function formatPrice(index: IndexData): string {
  const price = safeParseNumber(index.price);
  if (index.displayFormat === 'points' || index.dataType === 'index') {
    return price.toLocaleString('en-US', {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    });
  }
  return `$${price.toLocaleString('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })}`;
}

/** Skeleton tile shown during loading */
function SkeletonTile() {
  return (
    <div
      className="flex-shrink-0 w-36 sm:w-40 rounded-lg border border-ic-border bg-ic-surface p-3 animate-pulse"
      style={{ boxShadow: 'var(--ic-shadow-card)' }}
    >
      <div className="h-3 bg-ic-bg-tertiary rounded w-16 mb-2" />
      <div className="h-5 bg-ic-bg-tertiary rounded w-20 mb-1.5" />
      <div className="h-3 bg-ic-bg-tertiary rounded w-14" />
    </div>
  );
}

export default function HeroMiniDashboard() {
  const [indices, setIndices] = useState<IndexData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let isMounted = true;

    const fetchIndices = async () => {
      try {
        const response = await apiClient.getMarketIndices();
        if (isMounted) {
          setIndices(response.data);
          setError(null);
          setLoading(false);
        }
      } catch (err) {
        if (isMounted) {
          setError(err instanceof Error ? err.message : 'Failed to load indices');
          setLoading(false);
        }
      }
    };

    fetchIndices();

    // Poll every 30 seconds
    // TODO: Migrate from 30s polling to WebSocket for real-time price updates during market hours
    const interval = setInterval(fetchIndices, 30_000);

    return () => {
      isMounted = false;
      clearInterval(interval);
    };
  }, []);

  // Loading state: show skeleton tiles
  if (loading) {
    return (
      <div className="flex gap-3 overflow-x-auto pb-1 scrollbar-hide">
        {[1, 2, 3, 4].map((i) => (
          <SkeletonTile key={i} />
        ))}
      </div>
    );
  }

  // Error state: graceful fallback
  if (error || indices.length === 0) {
    return (
      <div className="rounded-lg border border-ic-border bg-ic-surface p-4 text-center">
        <p className="text-sm text-ic-text-muted">
          {error ? 'Unable to load market data' : 'No market data available'}
        </p>
      </div>
    );
  }

  // Show up to 5 tiles
  const displayIndices = indices.slice(0, 5);

  return (
    <div className="flex gap-3 overflow-x-auto pb-1 scrollbar-hide">
      {displayIndices.map((index) => {
        const change = safeParseNumber(index.change);
        const changePct = safeParseNumber(index.changePercent);
        const isPositive = change >= 0;

        return (
          <div
            key={index.symbol}
            className="flex-shrink-0 w-36 sm:w-40 rounded-lg border border-ic-border bg-ic-surface p-3 transition-colors hover:bg-ic-surface-hover"
            style={{ boxShadow: 'var(--ic-shadow-card)' }}
          >
            {/* Index name */}
            <p className="text-xs text-ic-text-muted truncate font-medium">{index.name}</p>

            {/* Price */}
            <p className="text-base font-semibold text-ic-text-primary mt-0.5 tabular-nums">
              {formatPrice(index)}
            </p>

            {/* Change */}
            <p
              className={`text-xs font-medium mt-0.5 tabular-nums ${
                isPositive ? 'text-ic-positive' : 'text-ic-negative'
              }`}
            >
              {isPositive ? '+' : ''}
              {safeToFixed(change, 2)} ({isPositive ? '+' : ''}
              {safeToFixed(changePct, 2)}%)
            </p>
          </div>
        );
      })}
    </div>
  );
}
