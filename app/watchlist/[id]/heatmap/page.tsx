'use client';

import { useEffect, useState, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { heatmapAPI, HeatmapData } from '@/lib/api/heatmap';
import { useAuth } from '@/lib/auth/AuthContext';
import { ProtectedRoute } from '@/components/auth/ProtectedRoute';
import WatchListHeatmap from '@/components/watchlist/WatchListHeatmap';
import HeatmapToolbar, { TimePeriod } from '@/components/watchlist/HeatmapConfigPanel';
import {
  HeatmapHeroView,
  HeatmapCardGrid,
  HeatmapBarChart,
  ViewMode,
  getEffectiveView,
} from '@/components/watchlist/HeatmapAdaptiveViews';

// ─── localStorage helpers ─────────────────────────────────────

const STORAGE_KEY_PREFIX = 'heatmap_prefs_';

interface HeatmapPrefs {
  viewMode: ViewMode;
  timePeriod: TimePeriod;
}

function loadPrefs(watchListId: string): HeatmapPrefs {
  try {
    const raw = localStorage.getItem(`${STORAGE_KEY_PREFIX}${watchListId}`);
    if (raw) {
      const parsed = JSON.parse(raw);
      return {
        viewMode: parsed.viewMode || 'auto',
        timePeriod: parsed.timePeriod || '1D',
      };
    }
  } catch {
    // Ignore parse errors
  }
  return { viewMode: 'auto', timePeriod: '1D' };
}

function savePrefs(watchListId: string, prefs: HeatmapPrefs) {
  try {
    localStorage.setItem(`${STORAGE_KEY_PREFIX}${watchListId}`, JSON.stringify(prefs));
  } catch {
    // Ignore quota errors
  }
}

// ─── Hardcoded defaults (removed from UI) ─────────────────────

const SIZE_METRIC = 'market_cap';
const COLOR_METRIC = 'price_change_pct';

// ─── Page component ───────────────────────────────────────────

export default function WatchListHeatmapPage() {
  const params = useParams();
  const router = useRouter();
  const { user, loading: authLoading } = useAuth();
  const watchListId = params.id as string;

  const [heatmapData, setHeatmapData] = useState<HeatmapData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // Restore persisted prefs (SSR-safe: defaults until hydration)
  const [viewMode, setViewMode] = useState<ViewMode>('auto');
  const [timePeriod, setTimePeriod] = useState<TimePeriod>('1D');
  const [prefsLoaded, setPrefsLoaded] = useState(false);

  useEffect(() => {
    const prefs = loadPrefs(watchListId);
    setViewMode(prefs.viewMode);
    setTimePeriod(prefs.timePeriod);
    setPrefsLoaded(true);
  }, [watchListId]);

  // Auto-persist on change
  const handleViewModeChange = useCallback(
    (mode: ViewMode) => {
      setViewMode(mode);
      savePrefs(watchListId, { viewMode: mode, timePeriod });
    },
    [watchListId, timePeriod]
  );

  const handleTimePeriodChange = useCallback(
    (period: TimePeriod) => {
      setTimePeriod(period);
      savePrefs(watchListId, { viewMode, timePeriod: period });
    },
    [watchListId, viewMode]
  );

  // Fetch heatmap data
  const loadHeatmap = useCallback(async () => {
    try {
      setLoading(true);
      const data = await heatmapAPI.getHeatmapData(watchListId, undefined, {
        size_metric: SIZE_METRIC,
        color_metric: COLOR_METRIC,
        time_period: timePeriod,
      });
      setHeatmapData(data);
      setError('');
    } catch (err: any) {
      setError(err.message || 'Failed to load heatmap');
    } finally {
      setLoading(false);
    }
  }, [watchListId, timePeriod]);

  useEffect(() => {
    if (authLoading || !user || !prefsLoaded) return;

    loadHeatmap();
    const interval = setInterval(loadHeatmap, 30000);
    return () => clearInterval(interval);
  }, [loadHeatmap, authLoading, user, prefsLoaded]);

  // Determine effective view
  const tileCount = heatmapData?.tile_count ?? 0;
  const effectiveView = getEffectiveView(viewMode, tileCount);

  if (loading && !heatmapData) {
    return (
      <ProtectedRoute>
        <div className="flex flex-col items-center justify-center min-h-screen">
          <div className="animate-spin rounded-full h-16 w-16 border-b-2 border-ic-blue mb-4"></div>
          <div className="text-xl text-ic-text-secondary font-medium">Loading heatmap...</div>
          <div className="text-sm text-ic-text-dim mt-2">
            Fetching ticker data and calculating metrics
          </div>
        </div>
      </ProtectedRoute>
    );
  }

  const renderVisualization = () => {
    if (!heatmapData || heatmapData.tiles.length === 0) {
      return (
        <div className="text-center py-16 bg-ic-bg-secondary rounded-lg border-2 border-dashed border-ic-border">
          <svg
            className="mx-auto h-16 w-16 text-ic-text-muted mb-4"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
            />
          </svg>
          <h3 className="text-lg font-semibold text-ic-text-primary mb-2">
            No tickers in watchlist
          </h3>
          <p className="text-ic-text-muted mb-4 max-w-md mx-auto">
            Add some tickers to your watch list to visualize them in the heatmap
          </p>
          <button
            onClick={() => router.push(`/watchlist/${watchListId}`)}
            className="inline-flex items-center px-4 py-2 bg-ic-blue text-ic-text-primary font-medium rounded-md hover:bg-ic-blue-hover transition-colors"
          >
            <svg className="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 4v16m8-8H4"
              />
            </svg>
            Add Tickers
          </button>
        </div>
      );
    }

    switch (effectiveView) {
      case 'hero':
        return <HeatmapHeroView data={heatmapData} />;

      case 'cards':
        return <HeatmapCardGrid data={heatmapData} />;

      case 'bars':
        return (
          <div className="bg-ic-surface rounded-lg shadow-lg border border-ic-border-subtle p-4">
            <HeatmapBarChart data={heatmapData} />
          </div>
        );

      case 'hybrid':
        return (
          <div className="space-y-6">
            <div className="bg-ic-surface rounded-lg shadow-lg border border-ic-border-subtle overflow-hidden">
              <WatchListHeatmap
                data={heatmapData}
                width={
                  typeof window !== 'undefined' ? Math.min(window.innerWidth - 200, 1400) : 1200
                }
                height={400}
              />
            </div>
            <div className="bg-ic-surface rounded-lg shadow-lg border border-ic-border-subtle p-4">
              <h3 className="text-sm font-semibold text-ic-text-secondary mb-3 uppercase tracking-wide">
                Ranked by % Change
              </h3>
              <HeatmapBarChart data={heatmapData} maxHeight={300} />
            </div>
          </div>
        );

      case 'treemap':
      default:
        return (
          <div className="bg-ic-surface rounded-lg shadow-lg border border-ic-border-subtle overflow-hidden">
            <WatchListHeatmap
              data={heatmapData}
              width={typeof window !== 'undefined' ? Math.min(window.innerWidth - 200, 1400) : 1200}
              height={700}
            />
          </div>
        );
    }
  };

  return (
    <ProtectedRoute>
      <div className="container mx-auto px-4 py-8 max-w-7xl">
        {/* Header */}
        <div className="mb-6">
          <div className="mb-4">
            <button
              onClick={() => router.push(`/watchlist/${watchListId}`)}
              className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-ic-text-secondary bg-ic-surface border border-ic-border rounded-md hover:bg-ic-surface-hover hover:text-ic-text-primary transition-colors shadow-sm"
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M10 19l-7-7m0 0l7-7m-7 7h18"
                />
              </svg>
              Back to Watch List
            </button>
          </div>

          <div className="flex items-start justify-between flex-wrap gap-4">
            <div>
              <h1 className="text-4xl font-bold text-ic-text-primary mb-2">
                {heatmapData?.watch_list_name || 'Watch List'}
              </h1>
              {heatmapData && (
                <div className="flex items-center gap-4 text-sm text-ic-text-dim">
                  <span>
                    <strong className="text-ic-text-primary">{heatmapData.tile_count}</strong>{' '}
                    tickers
                  </span>
                  <span className="text-ic-text-dim">&bull;</span>
                  <span>Updated {new Date(heatmapData.generated_at).toLocaleTimeString()}</span>
                </div>
              )}
            </div>
          </div>
        </div>

        {error && (
          <div className="mb-6 p-4 bg-red-500/10 border-l-4 border-red-500 rounded-md shadow-sm">
            <div className="flex items-start">
              <svg
                className="w-5 h-5 text-ic-negative mt-0.5 mr-3"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
              <div>
                <h3 className="text-sm font-medium text-ic-negative">Error loading heatmap</h3>
                <p className="text-sm text-ic-text-secondary mt-1">{error}</p>
              </div>
            </div>
          </div>
        )}

        <HeatmapToolbar
          viewMode={viewMode}
          onViewModeChange={handleViewModeChange}
          timePeriod={timePeriod}
          onTimePeriodChange={handleTimePeriodChange}
        />

        {renderVisualization()}
      </div>
    </ProtectedRoute>
  );
}
