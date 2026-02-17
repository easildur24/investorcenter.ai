'use client';

import { useState, useEffect } from 'react';
import { apiClient } from '@/lib/api';
import { ArrowTrendingUpIcon, ArrowTrendingDownIcon } from '@heroicons/react/24/outline';
import { safeToFixed, safeParseNumber } from '@/lib/utils';

interface MarketIndex {
  symbol: string;
  name: string;
  price: number | string;
  change: number | string;
  changePercent: number | string;
  lastUpdated: string;
}

export default function MarketOverview() {
  const [indices, setIndices] = useState<MarketIndex[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchMarketData = async () => {
      try {
        setLoading(true);
        const response = await apiClient.getMarketIndices();
        setIndices(response.data);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch market data');
        console.error('Error fetching market data:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchMarketData();

    // Refresh data every 30 seconds
    const interval = setInterval(fetchMarketData, 30000);

    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <div
        className="bg-ic-surface rounded-lg border border-ic-border p-6"
        style={{ boxShadow: 'var(--ic-shadow-card)' }}
      >
        <h2 className="text-lg font-semibold text-ic-text-primary mb-4">Market Overview</h2>
        <div className="animate-pulse space-y-4">
          {[1, 2, 3].map((i) => (
            <div key={i} className="flex justify-between items-center">
              <div className="h-4 bg-ic-bg-tertiary rounded w-1/3"></div>
              <div className="h-4 bg-ic-bg-tertiary rounded w-1/4"></div>
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
        <h2 className="text-lg font-semibold text-ic-text-primary mb-4">Market Overview</h2>
        <div className="text-ic-negative text-sm">
          <p>Error loading market data: {error}</p>
          <p className="text-ic-text-muted mt-2">
            Note: This will work once the Go backend is deployed and running.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div
      className="bg-ic-surface rounded-lg border border-ic-border p-6"
      style={{ boxShadow: 'var(--ic-shadow-card)' }}
    >
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-lg font-semibold text-ic-text-primary">Market Overview</h2>
        <span className="text-xs text-ic-text-muted">
          Last updated: {new Date().toLocaleTimeString()}
        </span>
      </div>

      <div className="space-y-4">
        {indices.map((index) => (
          <div
            key={index.symbol}
            className="flex justify-between items-center py-2 border-b border-ic-border-subtle last:border-b-0"
          >
            <div>
              <div className="font-medium text-ic-text-primary">{index.name}</div>
              <div className="text-sm text-ic-text-muted">{index.symbol}</div>
            </div>

            <div className="text-right">
              <div className="font-semibold text-ic-text-primary">
                $
                {safeParseNumber(index.price).toLocaleString('en-US', {
                  minimumFractionDigits: 2,
                  maximumFractionDigits: 2,
                })}
              </div>

              <div
                className={`flex items-center text-sm ${
                  safeParseNumber(index.change) >= 0 ? 'text-ic-positive' : 'text-ic-negative'
                }`}
              >
                {safeParseNumber(index.change) >= 0 ? (
                  <ArrowTrendingUpIcon className="h-4 w-4 mr-1" />
                ) : (
                  <ArrowTrendingDownIcon className="h-4 w-4 mr-1" />
                )}
                <span>
                  {safeParseNumber(index.change) >= 0 ? '+' : ''}
                  {safeToFixed(index.change, 2)}(
                  {safeParseNumber(index.changePercent) >= 0 ? '+' : ''}
                  {safeToFixed(index.changePercent, 2)}%)
                </span>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
