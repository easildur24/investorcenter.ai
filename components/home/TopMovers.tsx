'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { apiClient } from '@/lib/api';
import {
  ArrowTrendingUpIcon,
  ArrowTrendingDownIcon,
  FireIcon,
  ChevronDownIcon,
  ChevronUpIcon,
} from '@heroicons/react/24/outline';
import { safeToFixed, formatRelativeTime } from '@/lib/utils';
import { useWidgetTracking } from '@/lib/hooks/useWidgetTracking';
import SectorTag from '@/components/ui/SectorTag';

interface MoverStock {
  symbol: string;
  name?: string;
  price: number;
  change: number;
  changePercent: number;
  volume: number;
  sector?: string;
}

interface MoversData {
  gainers: MoverStock[];
  losers: MoverStock[];
  mostActive: MoverStock[];
}

type TabType = 'gainers' | 'losers' | 'active';

export default function TopMovers() {
  const { ref: widgetRef, trackInteraction } = useWidgetTracking('top_movers');
  const [movers, setMovers] = useState<MoversData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<TabType>('gainers');
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [expanded, setExpanded] = useState(false);

  useEffect(() => {
    const fetchMovers = async () => {
      try {
        setLoading(true);
        const response = await apiClient.getMarketMovers(10);
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
      icon: <ArrowTrendingUpIcon className="h-4 w-4" />,
    },
    {
      id: 'losers',
      label: 'Top Losers',
      icon: <ArrowTrendingDownIcon className="h-4 w-4" />,
    },
    {
      id: 'active',
      label: 'Most Active',
      icon: <FireIcon className="h-4 w-4" />,
    },
  ];

  const getAllData = (): MoverStock[] => {
    if (!movers) return [];
    switch (activeTab) {
      case 'gainers':
        return movers.gainers;
      case 'losers':
        return movers.losers;
      case 'active':
        return movers.mostActive;
      default:
        return [];
    }
  };

  const getCurrentData = (): MoverStock[] => {
    const all = getAllData();
    return expanded ? all : all.slice(0, 5);
  };

  const getMaxChangePercent = (): number => {
    const all = getAllData();
    if (all.length === 0) return 1;
    return Math.max(...all.map((s) => Math.abs(s.changePercent)), 1);
  };

  if (loading) {
    return (
      <div
        className="bg-ic-surface rounded-lg border border-ic-border p-6"
        style={{ boxShadow: 'var(--ic-shadow-card)' }}
      >
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
      <div
        className="bg-ic-surface rounded-lg border border-ic-border p-6"
        style={{ boxShadow: 'var(--ic-shadow-card)' }}
      >
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
  const allData = getAllData();
  const maxChange = getMaxChangePercent();
  const hasMore = allData.length > 5;

  return (
    <div
      ref={widgetRef}
      className="bg-ic-surface rounded-lg border border-ic-border p-6"
      style={{ boxShadow: 'var(--ic-shadow-card)' }}
    >
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
            onClick={() => {
              setActiveTab(tab.id);
              trackInteraction('tab_click', { tab: tab.id });
            }}
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
          currentData.map((stock) => {
            const barWidth = Math.min((Math.abs(stock.changePercent) / maxChange) * 100, 100);
            const barColor = stock.changePercent >= 0 ? 'rgb(34, 197, 94)' : 'rgb(239, 68, 68)';

            return (
              <Link
                key={stock.symbol}
                href={`/ticker/${stock.symbol}`}
                className="relative flex justify-between items-center py-2.5 px-2 -mx-2 rounded-lg hover:bg-ic-surface-hover transition-colors overflow-hidden"
              >
                {/* Change % color bar background */}
                <div
                  className="absolute inset-y-0 left-0 rounded-lg pointer-events-none"
                  style={{
                    width: `${barWidth}%`,
                    backgroundColor: barColor,
                    opacity: 0.07,
                  }}
                />

                <div className="relative flex items-center gap-3 min-w-0">
                  <div className="flex flex-col">
                    <div className="flex items-center gap-2">
                      <span className="font-semibold text-ic-text-primary">{stock.symbol}</span>
                      {stock.sector && <SectorTag sector={stock.sector} size="sm" />}
                    </div>
                    {stock.name && (
                      <span className="text-xs text-ic-text-muted truncate max-w-[120px]">
                        {stock.name}
                      </span>
                    )}
                  </div>
                  <span className="text-sm text-ic-text-muted">${safeToFixed(stock.price, 2)}</span>
                </div>

                <div className="relative flex items-center gap-3">
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
            );
          })
        )}
      </div>

      {/* Show more / Show less toggle */}
      {hasMore && (
        <button
          onClick={() => {
            setExpanded(!expanded);
            trackInteraction('toggle_expand', { expanded: !expanded });
          }}
          className="flex items-center justify-center gap-1 w-full mt-2 py-2 text-sm text-ic-text-muted hover:text-ic-text-primary transition-colors rounded-lg hover:bg-ic-surface-hover"
        >
          {expanded ? (
            <>
              <ChevronUpIcon className="h-4 w-4" />
              Show less
            </>
          ) : (
            <>
              <ChevronDownIcon className="h-4 w-4" />
              Show 5 more
            </>
          )}
        </button>
      )}

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
