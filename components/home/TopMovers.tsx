'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { apiClient } from '@/lib/api';
import { ArrowTrendingUpIcon, ArrowTrendingDownIcon, FireIcon } from '@heroicons/react/24/outline';
import { safeToFixed, formatRelativeTime } from '@/lib/utils';

interface MoverStock {
  symbol: string;
  name?: string;
  price: number;
  change: number;
  changePercent: number;
  volume: number;
}

interface MoversData {
  gainers: MoverStock[];
  losers: MoverStock[];
  mostActive: MoverStock[];
}

type TabType = 'gainers' | 'losers' | 'active';

export default function TopMovers() {
  const [movers, setMovers] = useState<MoversData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<TabType>('gainers');
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  useEffect(() => {
    const fetchMovers = async () => {
      try {
        setLoading(true);
        const response = await apiClient.getMarketMovers(5);
        setMovers(response.data);
        setLastUpdated(new Date());
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch market movers');
        console.error('Error fetching market movers:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchMovers();

    // Refresh every 5 minutes
    const interval = setInterval(fetchMovers, 5 * 60 * 1000);

    return () => clearInterval(interval);
  }, []);

  const formatVolume = (volume: number) => {
    if (volume >= 1e9) return `${(volume / 1e9).toFixed(1)}B`;
    if (volume >= 1e6) return `${(volume / 1e6).toFixed(1)}M`;
    if (volume >= 1e3) return `${(volume / 1e3).toFixed(1)}K`;
    return volume.toString();
  };

  const tabs: { id: TabType; label: string; icon: React.ReactNode }[] = [
    {
      id: 'gainers',
      label: 'Top Gainers',
      icon: <ArrowTrendingUpIcon className="h-4 w-4" />
    },
    {
      id: 'losers',
      label: 'Top Losers',
      icon: <ArrowTrendingDownIcon className="h-4 w-4" />
    },
    {
      id: 'active',
      label: 'Most Active',
      icon: <FireIcon className="h-4 w-4" />
    },
  ];

  const getCurrentData = (): MoverStock[] => {
    if (!movers) return [];
    switch (activeTab) {
      case 'gainers': return movers.gainers;
      case 'losers': return movers.losers;
      case 'active': return movers.mostActive;
      default: return [];
    }
  };

  if (loading) {
    return (
      <div className="bg-ic-surface rounded-lg border border-ic-border p-6">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-lg font-semibold text-ic-text-primary">Top Movers</h2>
        </div>
        <div className="animate-pulse space-y-4">
          <div className="flex space-x-4 border-b border-ic-border">
            {[1, 2, 3].map((i) => (
              <div key={i} className="h-8 bg-ic-bg-tertiary rounded w-24 mb-2"></div>
            ))}
          </div>
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
      <div className="bg-ic-surface rounded-lg border border-ic-border p-6">
        <h2 className="text-lg font-semibold text-ic-text-primary mb-4">Top Movers</h2>
        <div className="text-ic-negative text-sm">
          <p>Error loading market movers: {error}</p>
          <p className="text-ic-text-muted mt-2">
            This will work once the backend is running with market data access.
          </p>
        </div>
      </div>
    );
  }

  const currentData = getCurrentData();

  return (
    <div className="bg-ic-surface rounded-lg border border-ic-border p-6">
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-lg font-semibold text-ic-text-primary">Top Movers</h2>
        {lastUpdated && (
          <span className="text-xs text-ic-text-muted">
            Updated {formatRelativeTime(lastUpdated)}
          </span>
        )}
      </div>

      {/* Tabs */}
      <div className="flex space-x-1 border-b border-ic-border mb-4">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`flex items-center gap-1.5 px-3 py-2 text-sm font-medium rounded-t-lg transition-colors ${
              activeTab === tab.id
                ? 'text-ic-blue bg-ic-surface-hover border-b-2 border-ic-blue -mb-px'
                : 'text-ic-text-muted hover:text-ic-text-primary hover:bg-ic-surface-hover'
            }`}
          >
            {tab.icon}
            <span className="hidden sm:inline">{tab.label}</span>
          </button>
        ))}
      </div>

      {/* Stock List */}
      <div className="space-y-1">
        {currentData.length === 0 ? (
          <p className="text-ic-text-muted text-sm py-4 text-center">No data available</p>
        ) : (
          currentData.map((stock) => (
            <Link
              key={stock.symbol}
              href={`/ticker/${stock.symbol}`}
              className="flex justify-between items-center py-2.5 px-2 -mx-2 rounded-lg hover:bg-ic-surface-hover transition-colors"
            >
              <div className="flex items-center gap-3">
                <span className="font-semibold text-ic-text-primary w-14">{stock.symbol}</span>
                <span className="text-sm text-ic-text-muted">
                  ${safeToFixed(stock.price, 2)}
                </span>
              </div>

              <div className="flex items-center gap-3">
                {activeTab === 'active' && (
                  <span className="text-xs text-ic-text-dim">
                    Vol: {formatVolume(stock.volume)}
                  </span>
                )}
                <div
                  className={`flex items-center text-sm font-medium ${
                    stock.changePercent >= 0 ? 'text-ic-positive' : 'text-ic-negative'
                  }`}
                >
                  {stock.changePercent >= 0 ? (
                    <ArrowTrendingUpIcon className="h-4 w-4 mr-1" />
                  ) : (
                    <ArrowTrendingDownIcon className="h-4 w-4 mr-1" />
                  )}
                  <span>
                    {stock.changePercent >= 0 ? '+' : ''}
                    {safeToFixed(stock.changePercent, 2)}%
                  </span>
                </div>
              </div>
            </Link>
          ))
        )}
      </div>

      {/* Link to screener */}
      <div className="mt-4 pt-3 border-t border-ic-border-subtle">
        <Link
          href="/screener"
          className="text-sm text-ic-blue hover:text-ic-blue-hover font-medium transition-colors"
        >
          View all stocks in Screener →
        </Link>
      </div>
    </div>
  );
}
