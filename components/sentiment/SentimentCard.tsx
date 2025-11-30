'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { getSentiment } from '@/lib/api/sentiment';
import { formatRelativeTime, getSentimentScoreColor } from '@/lib/types/sentiment';
import type { SentimentResponse } from '@/lib/types/sentiment';
import SentimentGauge from './SentimentGauge';
import SentimentBreakdownBar from './SentimentBreakdownBar';

interface SentimentCardProps {
  ticker: string;
  variant?: 'full' | 'compact';
}

/**
 * Summary card displaying sentiment analysis for a ticker
 */
export default function SentimentCard({
  ticker,
  variant = 'full',
}: SentimentCardProps) {
  const [data, setData] = useState<SentimentResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchSentiment() {
      try {
        setLoading(true);
        setError(null);
        const result = await getSentiment(ticker);
        setData(result);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load sentiment');
      } finally {
        setLoading(false);
      }
    }

    fetchSentiment();
  }, [ticker]);

  if (loading) {
    return <LoadingSkeleton variant={variant} />;
  }

  if (error || !data) {
    return <ErrorState ticker={ticker} message={error} />;
  }

  // Check if there's no activity
  if (data.post_count_7d === 0) {
    return <NoActivityState ticker={ticker} />;
  }

  if (variant === 'compact') {
    return (
      <div className="bg-white rounded-lg shadow border border-gray-200 p-6">
        <div className="flex items-start justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-gray-900">Social Sentiment</h3>
            <p className="text-sm text-gray-600">
              {data.post_count_7d} posts in 7 days
            </p>
          </div>
          <Link
            href={`/sentiment?ticker=${ticker}`}
            className="text-sm text-blue-600 hover:text-blue-700 hover:underline"
          >
            View Details
          </Link>
        </div>

        {/* Score display */}
        <div className="text-center mb-4">
          <div
            className="text-4xl font-bold mb-1"
            style={{ color: getSentimentScoreColor(data.score) }}
          >
            {data.score >= 0 ? '+' : ''}{(data.score * 100).toFixed(0)}%
          </div>
          <div className="text-sm font-medium text-gray-600 capitalize">
            {data.label}
          </div>
        </div>

        {/* Breakdown bar */}
        <SentimentBreakdownBar
          breakdown={data.breakdown}
          showLabels={true}
          height="sm"
        />

        {/* Quick stats */}
        <div className="mt-4 grid grid-cols-2 gap-4 text-center">
          <div>
            <div className="text-lg font-semibold text-gray-900">
              {data.post_count_24h}
            </div>
            <div className="text-xs text-gray-500">Posts (24h)</div>
          </div>
          <div>
            <div className="text-lg font-semibold text-gray-900">
              {data.top_subreddits.length}
            </div>
            <div className="text-xs text-gray-500">Subreddits</div>
          </div>
        </div>

        {/* Top subreddits */}
        {data.top_subreddits.length > 0 && (
          <div className="mt-4 flex flex-wrap gap-2">
            {data.top_subreddits.slice(0, 3).map((sub) => (
              <span
                key={sub.subreddit}
                className="px-2 py-1 text-xs bg-gray-100 text-gray-600 rounded-full"
              >
                r/{sub.subreddit}
              </span>
            ))}
          </div>
        )}

        {/* Last updated */}
        <p className="text-xs text-gray-400 mt-4 text-center">
          Updated {formatRelativeTime(data.last_updated)}
        </p>
      </div>
    );
  }

  // Full variant
  return (
    <div className="bg-white rounded-lg shadow border border-gray-200 overflow-hidden">
      <div className="p-6 space-y-6">
        {/* Header */}
        <div className="flex items-start justify-between">
          <div>
            <h3 className="text-xl font-semibold text-gray-900">
              Social Sentiment: {ticker}
            </h3>
            <p className="text-sm text-gray-600 mt-1">
              Based on {data.post_count_7d} posts from the past 7 days
            </p>
          </div>
          <Link
            href={`/sentiment?ticker=${ticker}`}
            className="text-sm text-blue-600 hover:text-blue-700 hover:underline"
          >
            Full Analysis
          </Link>
        </div>

        {/* Gauge */}
        <SentimentGauge
          score={data.score}
          label={data.label}
          size="lg"
        />

        {/* Breakdown */}
        <div>
          <h4 className="text-sm font-medium text-gray-700 mb-2">Breakdown</h4>
          <SentimentBreakdownBar
            breakdown={data.breakdown}
            showLabels={true}
            showPercentages={true}
            height="md"
          />
        </div>

        {/* Stats grid */}
        <div className="grid grid-cols-2 gap-4">
          <StatBox
            label="Posts (24h)"
            value={data.post_count_24h.toString()}
          />
          <StatBox
            label="Posts (7d)"
            value={data.post_count_7d.toString()}
          />
        </div>

        {/* Top subreddits */}
        {data.top_subreddits.length > 0 && (
          <div>
            <h4 className="text-sm font-medium text-gray-700 mb-2">
              Top Subreddits
            </h4>
            <div className="space-y-2">
              {data.top_subreddits.slice(0, 5).map((sub) => (
                <div
                  key={sub.subreddit}
                  className="flex items-center justify-between"
                >
                  <span className="text-sm text-gray-600">
                    r/{sub.subreddit}
                  </span>
                  <span className="text-sm font-medium text-gray-900">
                    {sub.count} posts
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Footer */}
        <div className="text-xs text-gray-400 text-center pt-4 border-t border-gray-100">
          Last updated {formatRelativeTime(data.last_updated)}
        </div>
      </div>
    </div>
  );
}

/**
 * Stat box helper component
 */
interface StatBoxProps {
  label: string;
  value: string;
  trend?: 'up' | 'down' | 'neutral';
}

function StatBox({ label, value, trend }: StatBoxProps) {
  return (
    <div className="bg-gray-50 rounded-lg p-3 border border-gray-100">
      <div className="text-xs text-gray-500 mb-1">{label}</div>
      <div className="flex items-center gap-2">
        <span className="text-lg font-semibold text-gray-900">{value}</span>
        {trend === 'up' && <span className="text-green-500">+</span>}
        {trend === 'down' && <span className="text-red-500">-</span>}
      </div>
    </div>
  );
}

/**
 * Loading skeleton
 */
function LoadingSkeleton({ variant }: { variant: 'full' | 'compact' }) {
  if (variant === 'compact') {
    return (
      <div className="bg-white rounded-lg shadow border border-gray-200 p-6 animate-pulse">
        <div className="h-6 w-32 bg-gray-200 rounded mb-4" />
        <div className="h-20 bg-gray-200 rounded mb-4" />
        <div className="h-3 bg-gray-200 rounded mb-4" />
        <div className="grid grid-cols-2 gap-4">
          <div className="h-12 bg-gray-200 rounded" />
          <div className="h-12 bg-gray-200 rounded" />
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg shadow border border-gray-200 p-6 animate-pulse">
      <div className="h-8 w-48 bg-gray-200 rounded mb-6" />
      <div className="h-32 bg-gray-200 rounded mb-6" />
      <div className="h-4 bg-gray-200 rounded mb-6" />
      <div className="grid grid-cols-2 gap-4 mb-6">
        <div className="h-16 bg-gray-200 rounded" />
        <div className="h-16 bg-gray-200 rounded" />
      </div>
      <div className="space-y-2">
        <div className="h-4 bg-gray-200 rounded w-3/4" />
        <div className="h-4 bg-gray-200 rounded w-1/2" />
      </div>
    </div>
  );
}

/**
 * Error state
 */
function ErrorState({ ticker, message }: { ticker: string; message: string | null }) {
  return (
    <div className="bg-white rounded-lg shadow border border-gray-200 p-8 text-center">
      <div className="text-gray-400 text-4xl mb-4">📊</div>
      <h3 className="text-lg font-semibold text-gray-900 mb-2">
        Sentiment Not Available
      </h3>
      <p className="text-gray-600 text-sm">
        {message || `Unable to load sentiment data for ${ticker}`}
      </p>
    </div>
  );
}

/**
 * No activity state
 */
function NoActivityState({ ticker }: { ticker: string }) {
  return (
    <div className="bg-white rounded-lg shadow border border-gray-200 p-8 text-center">
      <div className="text-gray-400 text-4xl mb-4">💬</div>
      <h3 className="text-lg font-semibold text-gray-900 mb-2">
        No Recent Activity
      </h3>
      <p className="text-gray-600 text-sm">
        No social media posts found for {ticker} in the last 7 days
      </p>
    </div>
  );
}
