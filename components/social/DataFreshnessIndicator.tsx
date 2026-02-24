'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { reddit } from '@/lib/api/routes';
import { API_BASE_URL } from '@/lib/api';

interface PipelineHealth {
  lastPostAt: string | null;
  status: 'healthy' | 'stale' | 'no_data';
}

interface DataFreshnessIndicatorProps {
  /** Called when the user clicks the refresh button */
  onRefresh?: () => void;
  /** Bump this counter to trigger a health re-fetch from outside (e.g. after manual data refresh) */
  refreshTrigger?: number;
}

/**
 * BUG-004 fix: Shows data freshness with a status dot, relative time,
 * and a warning when data is stale (>30 min old).
 */
export default function DataFreshnessIndicator({
  onRefresh,
  refreshTrigger = 0,
}: DataFreshnessIndicatorProps) {
  const [health, setHealth] = useState<PipelineHealth | null>(null);
  const [fetchFailed, setFetchFailed] = useState(false);
  const [displayAge, setDisplayAge] = useState<string>('');

  // Keep a ref so the age-update timer always reads the latest health
  const healthRef = useRef<PipelineHealth | null>(null);

  const computeDisplayAge = useCallback((lastPostAt: string | null) => {
    if (!lastPostAt) {
      setDisplayAge('');
      return;
    }
    const lastUpdated = new Date(lastPostAt);
    const diffMs = Date.now() - lastUpdated.getTime();
    const diffMin = Math.floor(diffMs / 60_000);

    if (diffMin < 1) {
      setDisplayAge('Just now');
    } else if (diffMin < 60) {
      setDisplayAge(`${diffMin}m ago`);
    } else {
      const diffHours = Math.floor(diffMin / 60);
      if (diffHours < 24) {
        setDisplayAge(`${diffHours}h ago`);
      } else {
        setDisplayAge(`${Math.floor(diffHours / 24)}d ago`);
      }
    }
  }, []);

  const fetchHealth = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE_URL}${reddit.health}`);
      if (response.ok) {
        const data: PipelineHealth = await response.json();
        setHealth(data);
        healthRef.current = data;
        setFetchFailed(false);
        // Immediately recompute age so the display doesn't drift
        computeDisplayAge(data.lastPostAt);
      } else {
        setFetchFailed(true);
      }
    } catch {
      setFetchFailed(true);
    }
  }, [computeDisplayAge]);

  // Fetch health on mount and every 60 seconds
  useEffect(() => {
    fetchHealth();
    const interval = setInterval(fetchHealth, 60_000);
    return () => clearInterval(interval);
  }, [fetchHealth]);

  // Re-fetch when the parent bumps refreshTrigger (e.g. after manual data refresh)
  useEffect(() => {
    if (refreshTrigger > 0) {
      fetchHealth();
    }
  }, [refreshTrigger, fetchHealth]);

  // Update display age every 30 seconds (uses ref to avoid stale closure)
  useEffect(() => {
    const tick = () => computeDisplayAge(healthRef.current?.lastPostAt ?? null);
    const interval = setInterval(tick, 30_000);
    return () => clearInterval(interval);
  }, [computeDisplayAge]);

  if (!health && !fetchFailed) return null;

  const isStale = health?.status === 'stale';
  const isNoData = health?.status === 'no_data';

  // Determine visual state: fetch failure shows grey dot to indicate degraded state
  const dotColor = fetchFailed
    ? 'bg-gray-400'
    : isStale
      ? 'bg-amber-400'
      : isNoData
        ? 'bg-red-400'
        : 'bg-green-400';

  const textColor = fetchFailed
    ? 'text-gray-500'
    : isStale
      ? 'text-amber-600'
      : isNoData
        ? 'text-red-500'
        : 'text-ic-text-muted';

  return (
    <div
      className={`flex items-center gap-2 text-sm ${textColor}`}
      data-testid="freshness-indicator"
    >
      {/* Status dot */}
      <span className={`w-2 h-2 rounded-full flex-shrink-0 ${dotColor}`} />

      {/* Text */}
      {fetchFailed ? (
        <span>Unable to check freshness</span>
      ) : isNoData ? (
        <span>No data available</span>
      ) : (
        <span>Updated {displayAge}</span>
      )}

      {/* Stale warning */}
      {!fetchFailed && isStale && <span className="font-medium">Data may be stale</span>}

      {/* Refresh button */}
      {onRefresh && (
        <button
          onClick={onRefresh}
          className="ml-1 p-1 hover:bg-ic-surface-hover rounded transition-colors"
          title="Refresh data"
        >
          <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
            />
          </svg>
        </button>
      )}
    </div>
  );
}
