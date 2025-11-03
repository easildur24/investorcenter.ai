'use client';

import { useState } from 'react';

export interface HeatmapSettings {
  size_metric: string;
  color_metric: string;
  time_period: string;
  color_scheme: string;
  label_display: string;
}

interface HeatmapConfigPanelProps {
  settings: HeatmapSettings;
  onChange: (settings: HeatmapSettings) => void;
  onSave?: (name: string) => void;
}

export default function HeatmapConfigPanel({ settings, onChange, onSave }: HeatmapConfigPanelProps) {
  const [showSaveModal, setShowSaveModal] = useState(false);
  const [configName, setConfigName] = useState('');

  const handleChange = (field: keyof HeatmapSettings, value: string) => {
    onChange({ ...settings, [field]: value });
  };

  const handleSave = () => {
    if (onSave && configName) {
      onSave(configName);
      setConfigName('');
      setShowSaveModal(false);
    }
  };

  return (
    <div className="bg-white p-4 rounded-lg shadow mb-4">
      <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
        {/* Size Metric */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Size
          </label>
          <select
            value={settings.size_metric}
            onChange={(e) => handleChange('size_metric', e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="market_cap">Market Cap</option>
            <option value="volume">Volume</option>
            <option value="avg_volume">Avg Volume</option>
            <option value="reddit_mentions">Reddit Mentions</option>
            <option value="reddit_popularity">Reddit Popularity</option>
          </select>
        </div>

        {/* Color Metric */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Color
          </label>
          <select
            value={settings.color_metric}
            onChange={(e) => handleChange('color_metric', e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="price_change_pct">Price Change %</option>
            <option value="volume_change_pct">Volume Change %</option>
            <option value="reddit_rank">Reddit Rank</option>
            <option value="reddit_trend">Reddit Trend</option>
          </select>
        </div>

        {/* Time Period */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Time Period
          </label>
          <select
            value={settings.time_period}
            onChange={(e) => handleChange('time_period', e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="1D">1 Day</option>
            <option value="1W">1 Week</option>
            <option value="1M">1 Month</option>
            <option value="3M">3 Months</option>
            <option value="6M">6 Months</option>
            <option value="YTD">YTD</option>
            <option value="1Y">1 Year</option>
            <option value="5Y">5 Years</option>
          </select>
        </div>

        {/* Color Scheme */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Colors
          </label>
          <select
            value={settings.color_scheme}
            onChange={(e) => handleChange('color_scheme', e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="red_green">Red-Green</option>
            <option value="blue_red">Blue-Red</option>
            <option value="heatmap">Heatmap</option>
          </select>
        </div>

        {/* Save Button */}
        {onSave && (
          <div className="flex items-end">
            <button
              onClick={() => setShowSaveModal(true)}
              className="w-full px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
            >
              Save Config
            </button>
          </div>
        )}
      </div>

      {/* Save Config Modal */}
      {showSaveModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h3 className="text-xl font-bold mb-4">Save Heatmap Configuration</h3>
            <input
              type="text"
              value={configName}
              onChange={(e) => setConfigName(e.target.value)}
              placeholder="Configuration name"
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 mb-4"
            />
            <div className="flex gap-2 justify-end">
              <button
                onClick={() => setShowSaveModal(false)}
                className="px-4 py-2 border border-gray-300 rounded hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={handleSave}
                disabled={!configName}
                className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:bg-gray-400"
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
