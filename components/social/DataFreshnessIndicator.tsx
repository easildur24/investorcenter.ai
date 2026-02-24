'use client';

import { useState, useEffect, useCallback } from 'react';
import { reddit } from '@/lib/api/routes';
import { API_BASE_URL } from '@/lib/api';

interface PipelineHealth {
  lastHeatmapDate: string | null;
  lastPostAt: string | null;
  totalPosts7d: number;
  status: 'healthy' | 'stale' | 'no_data';
  stalenessMinutes: number;
}

interface DataFreshnessIndicatorProps {
  /** Called when the user clicks the refresh button */
  onRefresh?: () => void;
}

/**
 * BUG-004 fix: Shows data freshness with a status dot, relative time,
 * and a warning when data is stale (>30 min old).
 */
export default function DataFreshnessIndicator({ onRefresh }: DataFreshnessIndicatorProps) {
  const [health, setHealth] = useState<PipelineHealth | null>(null);
  const [displayAge, setDisplayAge] = useState<string>('');

  const fetchHealth = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE_URL}${reddit.health}`);
      if (response.ok) {
        const data: PipelineHealth = await response.json();
        setHealth(data);
      }
    } catch {
      // Silently fail — the indicator just won't update
    }
  }, []);

  // Fetch health on mount and every 60 seconds
  useEffect(() => {
    fetchHealth();
    const interval = setInterval(fetchHealth, 60_000);
    return () => clearInterval(interval);
  }, [fetchHealth]);

  // Update display age every 30 seconds
  useEffect(() => {
    const updateAge = () => {
      if (!health?.lastPostAt) {
        setDisplayAge('');
        return;
      }
      const lastUpdated = new Date(health.lastPostAt);
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
    };

    updateAge();
    const interval = setInterval(updateAge, 30_000);
    return () => clearInterval(interval);
  }, [health?.lastPostAt]);

  if (!health) return null;

  const isStale = health.status === 'stale';
  const isNoData = health.status === 'no_data';

  return (
    <div
      className={`flex items-center gap-2 text-sm ${
        isStale ? 'text-amber-600' : isNoData ? 'text-red-500' : 'text-ic-text-muted'
      }`}
      data-testid="freshness-indicator"
    >
      {/* Status dot */}
      <span
        className={`w-2 h-2 rounded-full flex-shrink-0 ${
          isStale ? 'bg-amber-400' : isNoData ? 'bg-red-400' : 'bg-green-400'
        }`}
      />

      {/* Text */}
      {isNoData ? <span>No data available</span> : <span>Updated {displayAge}</span>}

      {/* Stale warning */}
      {isStale && <span className="font-medium">Data may be stale</span>}

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
