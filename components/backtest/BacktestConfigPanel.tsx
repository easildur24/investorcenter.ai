'use client';

import { BacktestConfig } from '@/lib/types/backtest';

interface BacktestConfigPanelProps {
  config: BacktestConfig;
  onChange: (config: BacktestConfig) => void;
  onRun: () => void;
  loading: boolean;
}

export function BacktestConfigPanel({
  config,
  onChange,
  onRun,
  loading,
}: BacktestConfigPanelProps) {
  const updateConfig = (updates: Partial<BacktestConfig>) => {
    onChange({ ...config, ...updates });
  };

  return (
    <div className="bg-gray-50 border rounded-lg p-6">
      <h3 className="text-lg font-semibold mb-4">Backtest Configuration</h3>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {/* Date Range */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Start Date</label>
          <input
            type="date"
            value={config.start_date}
            onChange={(e) => updateConfig({ start_date: e.target.value })}
            className="w-full border rounded px-3 py-2 text-sm"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">End Date</label>
          <input
            type="date"
            value={config.end_date}
            onChange={(e) => updateConfig({ end_date: e.target.value })}
            className="w-full border rounded px-3 py-2 text-sm"
          />
        </div>

        {/* Rebalance Frequency */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Rebalance Frequency
          </label>
          <select
            value={config.rebalance_frequency}
            onChange={(e) =>
              updateConfig({
                rebalance_frequency: e.target.value as BacktestConfig['rebalance_frequency'],
              })
            }
            className="w-full border rounded px-3 py-2 text-sm"
          >
            <option value="daily">Daily</option>
            <option value="weekly">Weekly</option>
            <option value="monthly">Monthly</option>
            <option value="quarterly">Quarterly</option>
          </select>
        </div>

        {/* Universe */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Universe</label>
          <select
            value={config.universe}
            onChange={(e) =>
              updateConfig({ universe: e.target.value as BacktestConfig['universe'] })
            }
            className="w-full border rounded px-3 py-2 text-sm"
          >
            <option value="sp500">S&P 500</option>
            <option value="sp1500">S&P 1500</option>
            <option value="all">All Stocks</option>
          </select>
        </div>

        {/* Benchmark */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Benchmark</label>
          <select
            value={config.benchmark}
            onChange={(e) => updateConfig({ benchmark: e.target.value })}
            className="w-full border rounded px-3 py-2 text-sm"
          >
            <option value="SPY">S&P 500 (SPY)</option>
            <option value="QQQ">NASDAQ 100 (QQQ)</option>
            <option value="IWM">Russell 2000 (IWM)</option>
            <option value="VTI">Total Market (VTI)</option>
          </select>
        </div>

        {/* Transaction Costs */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Transaction Cost (bps)
          </label>
          <input
            type="number"
            value={config.transaction_cost_bps}
            onChange={(e) =>
              updateConfig({ transaction_cost_bps: parseFloat(e.target.value) || 0 })
            }
            min="0"
            max="100"
            step="1"
            className="w-full border rounded px-3 py-2 text-sm"
          />
        </div>

        {/* Slippage */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Slippage (bps)</label>
          <input
            type="number"
            value={config.slippage_bps}
            onChange={(e) => updateConfig({ slippage_bps: parseFloat(e.target.value) || 0 })}
            min="0"
            max="100"
            step="1"
            className="w-full border rounded px-3 py-2 text-sm"
          />
        </div>
      </div>

      {/* Exclusions */}
      <div className="mt-4 flex gap-6">
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={config.exclude_financials}
            onChange={(e) => updateConfig({ exclude_financials: e.target.checked })}
            className="rounded"
          />
          <span>Exclude Financials</span>
        </label>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={config.exclude_utilities}
            onChange={(e) => updateConfig({ exclude_utilities: e.target.checked })}
            className="rounded"
          />
          <span>Exclude Utilities</span>
        </label>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={config.use_smoothed_scores}
            onChange={(e) => updateConfig({ use_smoothed_scores: e.target.checked })}
            className="rounded"
          />
          <span>Use Smoothed Scores</span>
        </label>
      </div>

      {/* Run Button */}
      <div className="mt-6 flex justify-end">
        <button
          onClick={onRun}
          disabled={loading}
          className="px-6 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {loading ? (
            <span className="flex items-center gap-2">
              <span className="animate-spin h-4 w-4 border-2 border-white border-t-transparent rounded-full" />
              Running...
            </span>
          ) : (
            'Run Backtest'
          )}
        </button>
      </div>
    </div>
  );
}
