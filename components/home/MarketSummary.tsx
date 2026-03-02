'use client';

import { useState, useEffect, useCallback } from 'react';
import { apiClient } from '@/lib/api';
import { useWidgetTracking } from '@/lib/hooks/useWidgetTracking';
import { formatRelativeTime } from '@/lib/utils';
import { SparklesIcon } from '@heroicons/react/24/outline';

interface MarketSummaryData {
  summary: string;
  timestamp: string;
  method: 'llm' | 'template';
}

const REFRESH_INTERVAL_MS = 5 * 60 * 1000; // 5 minutes

export default function MarketSummary() {
  const { ref: widgetRef, trackInteraction } = useWidgetTracking('market_summary');
  const [data, setData] = useState<MarketSummaryData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchSummary = useCallback(async () => {
    try {
      setLoading(true);
      const response = await apiClient.request<MarketSummaryData>('/markets/summary');
      setData(response.data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch market summary');
      console.error('Error fetching market summary:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSummary();

    const interval = setInterval(fetchSummary, REFRESH_INTERVAL_MS);
    return () => clearInterval(interval);
  }, [fetchSummary]);

  // Skeleton loading state
  if (loading && !data) {
    return (
      <div
        className="bg-ic-surface rounded-lg border border-ic-border p-6"
        style={{ boxShadow: 'var(--ic-shadow-card)' }}
      >
        <div className="flex items-start gap-4">
          <div className="w-1 self-stretch rounded-full bg-blue-500/30 shrink-0" />
          <div className="flex-1 animate-pulse space-y-3">
            <div className="flex justify-between items-center">
              <div className="h-5 bg-ic-bg-tertiary rounded w-36" />
              <div className="h-4 bg-ic-bg-tertiary rounded w-20" />
            </div>
            <div className="space-y-2">
              <div className="h-4 bg-ic-bg-tertiary rounded w-full" />
              <div className="h-4 bg-ic-bg-tertiary rounded w-5/6" />
              <div className="h-4 bg-ic-bg-tertiary rounded w-3/4" />
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Error state
  if (error && !data) {
    return (
      <div
        className="bg-ic-surface rounded-lg border border-ic-border p-6"
        style={{ boxShadow: 'var(--ic-shadow-card)' }}
      >
        <div className="flex items-start gap-4">
          <div className="w-1 self-stretch rounded-full bg-ic-negative/30 shrink-0" />
          <div className="flex-1">
            <h2 className="text-lg font-semibold text-ic-text-primary mb-2">Market Summary</h2>
            <p className="text-ic-negative text-sm">{error}</p>
            <button
              onClick={() => {
                trackInteraction('retry');
                fetchSummary();
              }}
              className="mt-3 text-sm text-ic-blue hover:text-ic-blue-hover font-medium transition-colors"
            >
              Retry
            </button>
          </div>
        </div>
      </div>
    );
  }

  if (!data) return null;

  return (
    <div
      ref={widgetRef}
      className="bg-ic-surface rounded-lg border border-ic-border p-6"
      style={{ boxShadow: 'var(--ic-shadow-card)' }}
    >
      <div className="flex items-start gap-4">
        {/* Blue accent bar */}
        <div className="w-1 self-stretch rounded-full bg-blue-500 shrink-0" />

        <div className="flex-1 min-w-0">
          {/* Header */}
          <div className="flex justify-between items-center mb-3">
            <div className="flex items-center gap-2">
              <h2 className="text-lg font-semibold text-ic-text-primary">Market Summary</h2>
              {data.method === 'llm' && (
                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-blue-500/10 text-blue-500">
                  <SparklesIcon className="h-3 w-3" />
                  AI-generated
                </span>
              )}
            </div>
            <span className="text-xs text-ic-text-muted shrink-0">
              {formatRelativeTime(new Date(data.timestamp))}
            </span>
          </div>

          {/* Summary text */}
          <p className="text-sm leading-relaxed text-ic-text-secondary">{data.summary}</p>
        </div>
      </div>
    </div>
  );
}
