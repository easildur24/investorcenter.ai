'use client';

import { useEffect, useState, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { heatmapAPI, HeatmapData } from '@/lib/api/heatmap';
import { useAuth } from '@/lib/auth/AuthContext';
import { ProtectedRoute } from '@/components/auth/ProtectedRoute';
import WatchListHeatmap from '@/components/watchlist/WatchListHeatmap';
import HeatmapConfigPanel, { HeatmapSettings } from '@/components/watchlist/HeatmapConfigPanel';

export default function WatchListHeatmapPage() {
  const params = useParams();
  const router = useRouter();
  const { user, loading: authLoading } = useAuth();
  const watchListId = params.id as string;

  const [heatmapData, setHeatmapData] = useState<HeatmapData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [settings, setSettings] = useState<HeatmapSettings>({
    size_metric: 'market_cap',
    color_metric: 'price_change_pct',
    time_period: '1D',
    color_scheme: 'red_green',
    label_display: 'symbol_change',
  });

  const loadHeatmap = useCallback(async () => {
    try {
      setLoading(true);
      const data = await heatmapAPI.getHeatmapData(watchListId, undefined, {
        size_metric: settings.size_metric,
        color_metric: settings.color_metric,
        time_period: settings.time_period,
      });
      setHeatmapData(data);
      setError('');
    } catch (err: any) {
      setError(err.message || 'Failed to load heatmap');
    } finally {
      setLoading(false);
    }
  }, [watchListId, settings.size_metric, settings.color_metric, settings.time_period]);

  useEffect(() => {
    // Wait for auth to be ready before making API calls
    if (authLoading || !user) return;

    loadHeatmap();
    // Auto-refresh every 30 seconds
    const interval = setInterval(loadHeatmap, 30000);
    return () => clearInterval(interval);
  }, [loadHeatmap, authLoading, user]);

  const handleSaveConfig = async (name: string) => {
    try {
      await heatmapAPI.createConfig(watchListId, {
        name,
        ...settings,
      });
      alert('Heatmap configuration saved!');
    } catch (err: any) {
      alert(err.message || 'Failed to save configuration');
    }
  };

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

  return (
    <ProtectedRoute>
      <div className="container mx-auto px-4 py-8 max-w-7xl">
        {/* Header Section */}
        <div className="mb-6">
          {/* Back Button */}
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

          {/* Title & Info */}
          <div className="flex items-start justify-between flex-wrap gap-4">
            <div>
              <h1 className="text-4xl font-bold text-ic-text-primary mb-2 flex items-center gap-3">
                <svg
                  className="w-10 h-10 text-purple-400"
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
                {heatmapData?.watch_list_name || 'Watch List'}
              </h1>
              {heatmapData && (
                <div className="flex items-center gap-4 text-sm text-ic-text-dim">
                  <span className="inline-flex items-center gap-1">
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01"
                      />
                    </svg>
                    <strong className="text-ic-text-primary">{heatmapData.tile_count}</strong>{' '}
                    tickers
                  </span>
                  <span className="text-ic-text-dim">•</span>
                  <span className="inline-flex items-center gap-1">
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
                      />
                    </svg>
                    Updated {new Date(heatmapData.generated_at).toLocaleTimeString()}
                  </span>
                </div>
              )}
            </div>

            {/* Optional: Add export or share buttons here later */}
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

        <HeatmapConfigPanel settings={settings} onChange={setSettings} onSave={handleSaveConfig} />

        {!heatmapData || heatmapData.tiles.length === 0 ? (
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
        ) : (
          <div className="bg-ic-surface rounded-lg shadow-lg border border-ic-border-subtle overflow-hidden">
            <WatchListHeatmap
              data={heatmapData}
              width={typeof window !== 'undefined' ? Math.min(window.innerWidth - 200, 1400) : 1200}
              height={700}
            />
          </div>
        )}
      </div>
    </ProtectedRoute>
  );
}
