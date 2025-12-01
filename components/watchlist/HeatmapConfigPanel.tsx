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
    <div className="bg-ic-surface p-6 rounded-lg border border-ic-border mb-6">
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
        {/* Size Metric */}
        <div>
          <label className="block text-sm font-semibold text-ic-text-primary mb-2 flex items-center gap-1">
            <svg className="w-4 h-4 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
            </svg>
            Tile Size
          </label>
          <select
            value={settings.size_metric}
            onChange={(e) => handleChange('size_metric', e.target.value)}
            className="w-full px-3 py-2 border border-ic-border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-ic-text-primary bg-ic-surface hover:border-gray-400 transition-colors"
          >
            <option value="market_cap">📊 Market Cap</option>
            <option value="volume">📈 Volume</option>
            <option value="avg_volume">📉 Avg Volume</option>
            <option value="reddit_mentions">💬 Reddit Mentions</option>
            <option value="reddit_popularity">🔥 Reddit Popularity</option>
          </select>
        </div>

        {/* Color Metric */}
        <div>
          <label className="block text-sm font-semibold text-ic-text-primary mb-2 flex items-center gap-1">
            <svg className="w-4 h-4 text-purple-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01" />
            </svg>
            Tile Color
          </label>
          <select
            value={settings.color_metric}
            onChange={(e) => handleChange('color_metric', e.target.value)}
            className="w-full px-3 py-2 border border-ic-border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-ic-text-primary bg-ic-surface hover:border-gray-400 transition-colors"
          >
            <option value="price_change_pct">💲 Price Change %</option>
            <option value="volume_change_pct">📊 Volume Change %</option>
            <option value="reddit_rank">🏆 Reddit Rank</option>
            <option value="reddit_trend">📈 Reddit Trend</option>
          </select>
        </div>

        {/* Time Period */}
        <div>
          <label className="block text-sm font-semibold text-ic-text-primary mb-2 flex items-center gap-1">
            <svg className="w-4 h-4 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            Time Period
          </label>
          <select
            value={settings.time_period}
            onChange={(e) => handleChange('time_period', e.target.value)}
            className="w-full px-3 py-2 border border-ic-border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-ic-text-primary bg-ic-surface hover:border-gray-400 transition-colors"
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
          <label className="block text-sm font-semibold text-ic-text-primary mb-2 flex items-center gap-1">
            <svg className="w-4 h-4 text-pink-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01" />
            </svg>
            Color Scheme
          </label>
          <select
            value={settings.color_scheme}
            onChange={(e) => handleChange('color_scheme', e.target.value)}
            className="w-full px-3 py-2 border border-ic-border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-ic-text-primary bg-ic-surface hover:border-gray-400 transition-colors"
          >
            <option value="red_green">🔴 Red-Green</option>
            <option value="blue_red">🔵 Blue-Red</option>
            <option value="heatmap">🌈 Heatmap</option>
          </select>
        </div>

        {/* Save Button */}
        {onSave && (
          <div className="flex items-end">
            <button
              onClick={() => setShowSaveModal(true)}
              className="w-full px-4 py-2 bg-blue-600 text-white font-medium rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 transition-all shadow-sm hover:shadow-md"
            >
              💾 Save Config
            </button>
          </div>
        )}
      </div>

      {/* Save Config Modal */}
      {showSaveModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" onClick={() => setShowSaveModal(false)}>
          <div className="bg-ic-surface rounded-lg p-6 w-full max-w-md shadow-xl" onClick={(e) => e.stopPropagation()}>
            <h3 className="text-xl font-bold mb-4 text-ic-text-primary">💾 Save Heatmap Configuration</h3>
            <p className="text-sm text-ic-text-muted mb-4">Give your configuration a memorable name</p>
            <input
              type="text"
              value={configName}
              onChange={(e) => setConfigName(e.target.value)}
              placeholder="e.g., Reddit Momentum Strategy"
              className="w-full px-3 py-2 border border-ic-border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 mb-4 text-ic-text-primary"
              autoFocus
            />
            <div className="flex gap-2 justify-end">
              <button
                onClick={() => setShowSaveModal(false)}
                className="px-4 py-2 border border-ic-border rounded-md hover:bg-ic-surface-hover text-ic-text-secondary font-medium transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleSave}
                disabled={!configName}
                className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed font-medium transition-colors"
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
