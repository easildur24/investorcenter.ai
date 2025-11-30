'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { getTrendingSentiment } from '@/lib/api/sentiment';
import {
  getSentimentScoreColor,
  formatPercentageChange,
  formatCompactNumber,
} from '@/lib/types/sentiment';
import type { TrendingResponse, TrendingTicker, TrendingPeriod } from '@/lib/types/sentiment';
import { SentimentIndicator } from './SentimentGauge';
import { CompactBreakdownBar } from './SentimentBreakdownBar';

interface TrendingTickersListProps {
  initialPeriod?: TrendingPeriod;
  limit?: number;
  onTickerSelect?: (ticker: string) => void;
}

/**
 * Table of trending tickers by social media activity
 */
export default function TrendingTickersList({
  initialPeriod = '24h',
  limit = 20,
  onTickerSelect,
}: TrendingTickersListProps) {
  const [data, setData] = useState<TrendingResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [period, setPeriod] = useState<TrendingPeriod>(initialPeriod);

  useEffect(() => {
    async function fetchTrending() {
      try {
        setLoading(true);
        setError(null);
        const result = await getTrendingSentiment(period, limit);
        setData(result);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load trending');
      } finally {
        setLoading(false);
      }
    }

    fetchTrending();
  }, [period, limit]);

  const handleTickerClick = (ticker: string) => {
    if (onTickerSelect) {
      onTickerSelect(ticker);
    }
  };

  return (
    <div className="bg-white rounded-lg shadow-sm overflow-hidden">
      {/* Header with period toggle */}
      <div className="px-4 py-4 border-b border-gray-200">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-semibold text-gray-900">
            Trending by Sentiment
          </h3>
          <div className="flex gap-1">
            <button
              onClick={() => setPeriod('24h')}
              className={`px-3 py-1.5 text-sm rounded-md transition-colors ${
                period === '24h'
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
              }`}
            >
              24h
            </button>
            <button
              onClick={() => setPeriod('7d')}
              className={`px-3 py-1.5 text-sm rounded-md transition-colors ${
                period === '7d'
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
              }`}
            >
              7d
            </button>
          </div>
        </div>
      </div>

      {/* Loading state */}
      {loading && (
        <div className="p-12 text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto" />
          <p className="mt-4 text-gray-500 text-sm">Loading trending tickers...</p>
        </div>
      )}

      {/* Error state */}
      {error && !loading && (
        <div className="p-12 text-center">
          <p className="text-red-600">{error}</p>
          <button
            onClick={() => setPeriod(period)}
            className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
          >
            Try Again
          </button>
        </div>
      )}

      {/* Empty state */}
      {!loading && !error && (!data || data.tickers.length === 0) && (
        <div className="p-12 text-center">
          <p className="text-gray-500">No trending data available</p>
        </div>
      )}

      {/* Table */}
      {!loading && !error && data && data.tickers.length > 0 && (
        <>
          {/* Table header */}
          <div className="bg-gray-50 border-b border-gray-200 px-4 py-3 hidden sm:block">
            <div className="flex items-center text-xs font-medium text-gray-500 uppercase tracking-wider">
              <span className="w-12 text-center">#</span>
              <span className="flex-1">Ticker</span>
              <span className="w-28 text-center">Sentiment</span>
              <span className="w-20 text-right">Posts</span>
              <span className="w-20 text-right hidden md:block">Change</span>
            </div>
          </div>

          {/* Table body */}
          <div className="divide-y divide-gray-100">
            {data.tickers.map((ticker) => (
              <TrendingTickerRow
                key={ticker.ticker}
                ticker={ticker}
                onClick={() => handleTickerClick(ticker.ticker)}
              />
            ))}
          </div>
        </>
      )}

      {/* Footer */}
      {data && data.tickers.length > 0 && (
        <div className="bg-gray-50 border-t border-gray-200 px-4 py-3 text-sm text-gray-500">
          Showing {data.tickers.length} trending tickers ({period})
        </div>
      )}
    </div>
  );
}

/**
 * Individual row for a trending ticker
 */
interface TrendingTickerRowProps {
  ticker: TrendingTicker;
  onClick: () => void;
}

function TrendingTickerRow({ ticker, onClick }: TrendingTickerRowProps) {
  const mentionDeltaColor = ticker.mention_delta >= 0 ? 'text-green-600' : 'text-red-600';

  return (
    <Link
      href={`/ticker/${ticker.ticker}`}
      className="flex items-center px-4 py-3 hover:bg-gray-50 transition-colors cursor-pointer"
      onClick={(e) => {
        if (onClick) {
          e.preventDefault();
          onClick();
        }
      }}
    >
      {/* Rank */}
      <span className="w-12 text-center text-sm font-medium text-gray-500">
        {ticker.rank}
      </span>

      {/* Ticker info */}
      <div className="flex-1 min-w-0">
        <div className="font-semibold text-gray-900">{ticker.ticker}</div>
        <div className="sm:hidden mt-1">
          <SentimentIndicator score={ticker.score} label={ticker.label} />
        </div>
      </div>

      {/* Sentiment (desktop) */}
      <div className="w-28 hidden sm:flex flex-col items-center gap-1">
        <SentimentIndicator score={ticker.score} label={ticker.label} />
        <span
          className="text-xs font-medium"
          style={{ color: getSentimentScoreColor(ticker.score) }}
        >
          {ticker.score >= 0 ? '+' : ''}{(ticker.score * 100).toFixed(0)}%
        </span>
      </div>

      {/* Post count */}
      <span className="w-20 text-right text-sm font-medium text-gray-900">
        {formatCompactNumber(ticker.post_count)}
      </span>

      {/* Mention delta (desktop) */}
      <span className={`w-20 text-right text-sm font-medium hidden md:block ${mentionDeltaColor}`}>
        {formatPercentageChange(ticker.mention_delta)}
      </span>
    </Link>
  );
}

/**
 * Compact version for smaller spaces
 */
interface CompactTrendingListProps {
  tickers: TrendingTicker[];
  limit?: number;
}

export function CompactTrendingList({ tickers, limit = 5 }: CompactTrendingListProps) {
  const displayTickers = tickers.slice(0, limit);

  return (
    <div className="space-y-2">
      {displayTickers.map((ticker) => (
        <Link
          key={ticker.ticker}
          href={`/ticker/${ticker.ticker}`}
          className="flex items-center justify-between p-2 rounded-lg hover:bg-gray-50 transition-colors"
        >
          <div className="flex items-center gap-3">
            <span className="text-xs text-gray-400 w-4">{ticker.rank}</span>
            <span className="font-medium text-gray-900">{ticker.ticker}</span>
          </div>
          <div className="flex items-center gap-2">
            <SentimentIndicator score={ticker.score} label={ticker.label} />
            <span className="text-xs text-gray-500">
              {formatCompactNumber(ticker.post_count)}
            </span>
          </div>
        </Link>
      ))}
    </div>
  );
}
