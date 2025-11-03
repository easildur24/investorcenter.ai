'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { heatmapAPI, HeatmapData } from '@/lib/api/heatmap';
import WatchListHeatmap from '@/components/watchlist/WatchListHeatmap';
import HeatmapConfigPanel, { HeatmapSettings } from '@/components/watchlist/HeatmapConfigPanel';

export default function WatchListHeatmapPage() {
  const params = useParams();
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

  useEffect(() => {
    loadHeatmap();
    // Auto-refresh every 30 seconds
    const interval = setInterval(loadHeatmap, 30000);
    return () => clearInterval(interval);
  }, [watchListId, settings.size_metric, settings.color_metric, settings.time_period]);

  const loadHeatmap = async () => {
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
  };

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
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-xl">Loading heatmap...</div>
      </div>
    );
  }

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="mb-6">
        <h1 className="text-3xl font-bold">
          {heatmapData?.watch_list_name || 'Watch List'} - Heatmap
        </h1>
        {heatmapData && (
          <p className="text-gray-600 mt-2">
            {heatmapData.tile_count} tickers | Updated {new Date(heatmapData.generated_at).toLocaleTimeString()}
          </p>
        )}
      </div>

      {error && (
        <div className="mb-4 p-3 bg-red-100 border border-red-400 text-red-700 rounded">
          {error}
        </div>
      )}

      <HeatmapConfigPanel
        settings={settings}
        onChange={setSettings}
        onSave={handleSaveConfig}
      />

      {!heatmapData || heatmapData.tiles.length === 0 ? (
        <div className="text-center py-12 bg-gray-50 rounded-lg">
          <p className="text-gray-600">No tickers to display in heatmap</p>
        </div>
      ) : (
        <WatchListHeatmap
          data={heatmapData}
          width={typeof window !== 'undefined' ? window.innerWidth - 100 : 1200}
          height={600}
        />
      )}
    </div>
  );
}
