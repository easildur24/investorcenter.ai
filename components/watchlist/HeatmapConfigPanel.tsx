'use client';

import { useState } from 'react';
import { useModal } from '@/lib/hooks/useModal';
import { ViewMode } from './HeatmapAdaptiveViews';

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
  viewMode?: ViewMode;
  onViewModeChange?: (mode: ViewMode) => void;
}

function SaveConfigModal({
  onClose,
  onSave,
}: {
  onClose: () => void;
  onSave: (name: string) => void;
}) {
  const modalRef = useModal(onClose);
  const [configName, setConfigName] = useState('');

  const handleSave = () => {
    if (configName) {
      onSave(configName);
    }
  };

  return (
    <div
      className="fixed inset-0 bg-ic-bg-primary bg-opacity-50 flex items-center justify-center z-50"
      onClick={onClose}
    >
      <div
        ref={modalRef}
        role="dialog"
        aria-modal="true"
        aria-label="Save Heatmap Configuration"
        className="bg-ic-surface rounded-lg p-6 w-full max-w-md shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <h3 className="text-xl font-bold mb-4 text-ic-text-primary">Save Heatmap Configuration</h3>
        <p className="text-sm text-ic-text-muted mb-4">Give your configuration a memorable name</p>
        <input
          type="text"
          value={configName}
          onChange={(e) => setConfigName(e.target.value)}
          placeholder="e.g., Reddit Momentum Strategy"
          className="w-full px-3 py-2 border border-ic-border rounded-md focus:outline-none focus:ring-2 focus:ring-ic-blue mb-4 text-ic-text-primary"
        />
        <div className="flex gap-2 justify-end">
          <button
            onClick={onClose}
            className="px-4 py-2 border border-ic-border rounded-md hover:bg-ic-surface-hover text-ic-text-secondary font-medium transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={!configName}
            className="px-4 py-2 bg-ic-blue text-ic-text-primary rounded-md hover:bg-ic-blue-hover disabled:bg-ic-bg-tertiary disabled:cursor-not-allowed font-medium transition-colors"
          >
            Save
          </button>
        </div>
      </div>
    </div>
  );
}

const VIEW_MODES: ViewMode[] = ['auto', 'treemap', 'cards', 'bars'];

const selectClass =
  'px-2 py-1.5 text-xs border border-ic-border rounded-md bg-ic-surface text-ic-text-primary hover:border-zinc-500 focus:outline-none focus:ring-1 focus:ring-ic-blue transition-colors';

export default function HeatmapConfigPanel({
  settings,
  onChange,
  onSave,
  viewMode = 'auto',
  onViewModeChange,
}: HeatmapConfigPanelProps) {
  const [showSaveModal, setShowSaveModal] = useState(false);

  const handleChange = (field: keyof HeatmapSettings, value: string) => {
    onChange({ ...settings, [field]: value });
  };

  const handleSave = (name: string) => {
    if (onSave) {
      onSave(name);
      setShowSaveModal(false);
    }
  };

  return (
    <div className="bg-ic-surface rounded-lg border border-ic-border px-4 py-2.5 mb-4">
      <div className="flex items-center gap-3 flex-wrap">
        {/* View mode toggle */}
        {onViewModeChange && (
          <>
            <div className="flex items-center gap-0.5 bg-ic-bg-secondary rounded-md p-0.5">
              {VIEW_MODES.map((mode) => (
                <button
                  key={mode}
                  onClick={() => onViewModeChange(mode)}
                  className={`px-2.5 py-1 rounded text-xs font-medium transition-colors ${
                    viewMode === mode
                      ? 'bg-ic-blue text-white shadow-sm'
                      : 'text-ic-text-secondary hover:text-ic-text-primary'
                  }`}
                >
                  {mode.charAt(0).toUpperCase() + mode.slice(1)}
                </button>
              ))}
            </div>
            <div className="h-5 w-px bg-ic-border" />
          </>
        )}

        {/* Size metric */}
        <select
          aria-label="Tile size metric"
          value={settings.size_metric}
          onChange={(e) => handleChange('size_metric', e.target.value)}
          className={selectClass}
        >
          <option value="market_cap">Market Cap</option>
          <option value="volume">Volume</option>
          <option value="avg_volume">Avg Volume</option>
          <option value="reddit_mentions">Reddit Mentions</option>
          <option value="reddit_popularity">Reddit Popularity</option>
        </select>

        {/* Color metric */}
        <select
          aria-label="Tile color metric"
          value={settings.color_metric}
          onChange={(e) => handleChange('color_metric', e.target.value)}
          className={selectClass}
        >
          <option value="price_change_pct">Price Change %</option>
          <option value="volume_change_pct">Volume Change %</option>
          <option value="reddit_rank">Reddit Rank</option>
          <option value="reddit_trend">Reddit Trend</option>
        </select>

        {/* Time period */}
        <select
          aria-label="Time period"
          value={settings.time_period}
          onChange={(e) => handleChange('time_period', e.target.value)}
          className={selectClass}
        >
          <option value="1D">1D</option>
          <option value="1W">1W</option>
          <option value="1M">1M</option>
          <option value="3M">3M</option>
          <option value="6M">6M</option>
          <option value="YTD">YTD</option>
          <option value="1Y">1Y</option>
          <option value="5Y">5Y</option>
        </select>

        {/* Color scheme */}
        <select
          aria-label="Color scheme"
          value={settings.color_scheme}
          onChange={(e) => handleChange('color_scheme', e.target.value)}
          className={selectClass}
        >
          <option value="red_green">Red-Green</option>
          <option value="blue_red">Blue-Red</option>
          <option value="heatmap">Heatmap</option>
        </select>

        <div className="flex-1" />

        {/* Save button */}
        {onSave && (
          <button
            onClick={() => setShowSaveModal(true)}
            className="px-3 py-1.5 text-xs font-medium bg-ic-blue text-white rounded-md hover:bg-ic-blue-hover transition-colors"
          >
            Save Config
          </button>
        )}
      </div>

      {showSaveModal && (
        <SaveConfigModal onClose={() => setShowSaveModal(false)} onSave={handleSave} />
      )}
    </div>
  );
}
