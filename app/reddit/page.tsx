'use client';

import { useState, useEffect } from 'react';
import TimeRangeSelector from '@/components/reddit/TimeRangeSelector';
import TrendingList from '@/components/reddit/TrendingList';

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
}

interface ApiResponse {
  data: RedditHeatmapData[];
  meta: {
    days: number;
    limit: number;
    count: number;
    latestDate: string;
  };
}

export default function RedditTrendingPage() {
  const [timeRange, setTimeRange] = useState<'1' | '7' | '14' | '30'>('7');
  const [data, setData] = useState<RedditHeatmapData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const fetchData = async (days: string) => {
    try {
      setLoading(true);
      setError(null);

      const response = await fetch(`/api/v1/reddit/heatmap?days=${days}&top=50`);
      const result: ApiResponse = await response.json();

      if (result.data) {
        setData(result.data);
        setLastUpdated(new Date());
      } else {
        setError('Failed to load Reddit trending data');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData(timeRange);
  }, [timeRange]);

  // Auto-refresh every 5 minutes
  useEffect(() => {
    const interval = setInterval(() => {
      fetchData(timeRange);
    }, 5 * 60 * 1000); // 5 minutes

    return () => clearInterval(interval);
  }, [timeRange]);

  const handleRefresh = () => {
    fetchData(timeRange);
  };

  const getLastUpdatedText = () => {
    if (!lastUpdated) return '';
    const now = new Date();
    const diff = Math.floor((now.getTime() - lastUpdated.getTime()) / 1000 / 60);

    if (diff < 1) return 'Just now';
    if (diff === 1) return '1 minute ago';
    return `${diff} minutes ago`;
  };

  return (
    <div className="min-h-screen bg-ic-bg-primary">
      {/* Header Section */}
      <div className="bg-ic-surface border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <div className="flex items-center gap-3 mb-2">
            <span className="text-3xl">🔥</span>
            <h1 className="text-3xl font-bold text-ic-text-primary">Trending on Reddit</h1>
          </div>
          <p className="text-ic-text-muted mt-2">
            Most mentioned stocks across r/wallstreetbets, r/stocks, and r/investing
          </p>
        </div>
      </div>

      {/* Controls Section */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <div className="bg-ic-surface rounded-lg shadow-sm p-4 mb-6">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <TimeRangeSelector value={timeRange} onChange={setTimeRange} />

            <div className="flex items-center gap-2 text-sm text-ic-text-muted">
              <span>Last updated: {getLastUpdatedText()}</span>
              <button
                onClick={handleRefresh}
                className="ml-2 p-1 hover:bg-ic-surface-hover rounded"
                title="Refresh data"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
              </button>
            </div>
          </div>
        </div>

        {/* Main Content */}
        {loading && !data.length ? (
          <div className="bg-ic-surface rounded-lg shadow-sm p-12 text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mx-auto"></div>
            <p className="mt-4 text-ic-text-muted">Loading trending stocks...</p>
          </div>
        ) : error ? (
          <div className="bg-ic-surface rounded-lg shadow-sm p-12 text-center">
            <p className="text-ic-negative">{error}</p>
            <button
              onClick={handleRefresh}
              className="mt-4 px-4 py-2 bg-ic-blue text-ic-text-primary rounded-md hover:bg-ic-blue-hover"
            >
              Try Again
            </button>
          </div>
        ) : data.length === 0 ? (
          <div className="bg-ic-surface rounded-lg shadow-sm p-12 text-center">
            <p className="text-ic-text-muted">No trending data available for this time range</p>
          </div>
        ) : (
          <TrendingList items={data} timeRange={timeRange} />
        )}
      </div>
    </div>
  );
}
