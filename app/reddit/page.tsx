'use client';

import { useState, useEffect } from 'react';
import TimeRangeSelector from '@/components/reddit/TimeRangeSelector';
import TrendingList from '@/components/reddit/TrendingList';
import DataFreshnessIndicator from '@/components/social/DataFreshnessIndicator';
import { reddit } from '@/lib/api/routes';
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

  const fetchData = async (days: string) => {
    try {
      setLoading(true);
      setError(null);

      const response = await fetch(`${API_BASE_URL}${reddit.heatmap}?days=${days}&top=50`);
      const result: ApiResponse = await response.json();

      if (result.data) {
        setData(result.data);
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
    const interval = setInterval(
      () => {
        fetchData(timeRange);
      },
      5 * 60 * 1000
    ); // 5 minutes

    return () => clearInterval(interval);
  }, [timeRange]);

  const handleRefresh = () => {
    fetchData(timeRange);
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

            {/* BUG-004: DataFreshnessIndicator replaces the old static "Last updated" text */}
            <DataFreshnessIndicator onRefresh={handleRefresh} />
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
