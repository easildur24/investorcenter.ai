'use client';

import { useState, useEffect } from 'react';
import TrendingListItem from './TrendingListItem';
import { tickers } from '@/lib/api/routes';
import { API_BASE_URL } from '@/lib/api';

interface RedditHeatmapData {
  tickerSymbol: string;
  date: string;
  avgRank: number;
  minRank: number;
  maxRank: number;
  totalMentions: number;
  totalUpvotes: number;
  rankVolatility: number;
  trendDirection: string;
  popularityScore: number;
  dataSource: string;
  // BUG-003: Price data from Polygon
  price?: number;
  priceChangePct?: number;
}

interface TrendingListProps {
  items: RedditHeatmapData[];
  timeRange: string;
}

interface TickerMetadata {
  [key: string]: {
    name: string;
  };
}

export default function TrendingList({ items, timeRange }: TrendingListProps) {
  const [tickerNames, setTickerNames] = useState<TickerMetadata>({});

  // Fetch company names for tickers
  useEffect(() => {
    const fetchTickerNames = async () => {
      const symbols = items.map((item) => item.tickerSymbol).slice(0, 20); // Limit to first 20 for performance
      const names: TickerMetadata = {};

      // Fetch names in parallel (with a simple implementation)
      // In production, you might want to batch this into a single API call
      const fetchPromises = symbols.map(async (symbol) => {
        try {
          const response = await fetch(`${API_BASE_URL}${tickers.bySymbol(symbol)}`);
          const data = await response.json();
          if (data.data && data.data.name) {
            names[symbol] = { name: data.data.name };
          }
        } catch (error) {
          // Silently fail for individual ticker names
          console.error(`Failed to fetch name for ${symbol}`, error);
        }
      });

      await Promise.all(fetchPromises);
      setTickerNames(names);
    };

    if (items.length > 0) {
      fetchTickerNames();
    }
  }, [items]);

  // Calculate rank change (simplified - assumes ranks are sequential)
  // BUG-001 fix: always return integer to avoid floating-point display
  const getRankChange = (index: number, avgRank: number) => {
    const expectedRank = index + 1;
    return Math.round(expectedRank - avgRank);
  };

  return (
    <div className="bg-ic-surface rounded-lg shadow-sm overflow-hidden">
      {/* Table Header */}
      <div className="bg-ic-surface border-b border-ic-border-subtle px-4 py-3 hidden sm:block">
        <div className="flex items-center justify-between text-xs font-medium text-ic-text-dim uppercase tracking-wider">
          <div className="flex items-center gap-4 flex-1">
            <span className="w-8">#</span>
            <span>Ticker</span>
          </div>
          <div className="flex items-center gap-6">
            <span className="w-24 text-right hidden lg:block">Price</span>
            <span className="w-20 text-right hidden lg:block">% Change</span>
            <span className="w-20 text-right">Mentions</span>
            <span className="w-20 text-right hidden md:block">Upvotes</span>
            <span className="w-28 text-right">Score</span>
            <span className="w-5"></span> {/* Arrow space */}
          </div>
        </div>
      </div>

      {/* List Items */}
      <div>
        {items.map((item, index) => (
          <TrendingListItem
            key={`${item.tickerSymbol}-${item.date}`}
            rank={index + 1}
            symbol={item.tickerSymbol}
            companyName={tickerNames[item.tickerSymbol]?.name}
            currentRank={Math.round(item.avgRank)}
            rankChange={getRankChange(index, item.avgRank)}
            mentions={item.totalMentions}
            upvotes={item.totalUpvotes}
            popularityScore={item.popularityScore}
            trendDirection={item.trendDirection}
            price={item.price}
            priceChangePct={item.priceChangePct}
          />
        ))}
      </div>

      {/* Footer Info */}
      {items.length > 0 && (
        <div className="bg-ic-surface border-t border-ic-border-subtle px-4 py-3 text-sm text-ic-text-dim">
          Showing {items.length} trending {items.length === 1 ? 'stock' : 'stocks'}
          {timeRange === '1' ? ' today' : ` over the last ${timeRange} days`}
        </div>
      )}
    </div>
  );
}
