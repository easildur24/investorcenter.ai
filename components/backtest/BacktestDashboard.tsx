'use client';

import { useState, useEffect } from 'react';
import {
  BacktestSummary,
  BacktestConfig,
  DecilePerformance,
  DEFAULT_BACKTEST_CONFIG,
  formatPercent,
  getDecileColor,
  getReturnColor,
  getRatingFromSharpe,
} from '@/lib/types/backtest';
import { DecileBarChart } from './DecileBarChart';
import { CumulativeReturnsChart } from './CumulativeReturnsChart';
import { BacktestConfigPanel } from './BacktestConfigPanel';
import { StatisticalSummary } from './StatisticalSummary';
import { backtest } from '@/lib/api/routes';
import { API_BASE_URL } from '@/lib/api';

interface BacktestDashboardProps {
  initialData?: BacktestSummary;
}

export default function BacktestDashboard({ initialData }: BacktestDashboardProps) {
  const [summary, setSummary] = useState<BacktestSummary | null>(initialData || null);
  const [loading, setLoading] = useState(!initialData);
  const [error, setError] = useState<string | null>(null);
  const [config, setConfig] = useState<BacktestConfig>(DEFAULT_BACKTEST_CONFIG);
  const [showConfig, setShowConfig] = useState(false);

  useEffect(() => {
    if (!initialData) {
      fetchLatestBacktest();
    }
  }, [initialData]);

  const fetchLatestBacktest = async () => {
    try {
      setLoading(true);
      const response = await fetch(`${API_BASE_URL}${backtest.latest}`);
      if (!response.ok) {
        throw new Error('Failed to fetch backtest results');
      }
      const data = await response.json();
      setSummary(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  const runBacktest = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await fetch(`${API_BASE_URL}${backtest.run}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config),
      });
      if (!response.ok) {
        throw new Error('Failed to run backtest');
      }
      const data = await response.json();
      setSummary(data);
      setShowConfig(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p className="text-gray-600">Loading backtest results...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-6 text-center">
        <p className="text-red-700 mb-4">{error}</p>
        <button
          onClick={fetchLatestBacktest}
          className="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700"
        >
          Retry
        </button>
      </div>
    );
  }

  if (!summary) {
    return (
      <div className="bg-gray-50 border border-gray-200 rounded-lg p-6 text-center">
        <p className="text-gray-600 mb-4">No backtest results available</p>
        <button
          onClick={() => setShowConfig(true)}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
        >
          Run New Backtest
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">IC Score Backtest Results</h2>
          <p className="text-gray-600">
            {summary.start_date} to {summary.end_date} | {summary.universe.toUpperCase()} Universe |{' '}
            {summary.rebalance_frequency} Rebalancing
          </p>
        </div>
        <button
          onClick={() => setShowConfig(!showConfig)}
          className="px-4 py-2 border border-gray-300 rounded hover:bg-gray-50"
        >
          {showConfig ? 'Hide Config' : 'Configure'}
        </button>
      </div>

      {/* Config Panel */}
      {showConfig && (
        <BacktestConfigPanel
          config={config}
          onChange={setConfig}
          onRun={runBacktest}
          loading={loading}
        />
      )}

      {/* Key Metrics */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <MetricCard
          title="Top Decile CAGR"
          value={formatPercent(summary.top_decile_cagr)}
          color={getReturnColor(summary.top_decile_cagr)}
        />
        <MetricCard
          title="Bottom Decile CAGR"
          value={formatPercent(summary.bottom_decile_cagr)}
          color={getReturnColor(summary.bottom_decile_cagr)}
        />
        <MetricCard
          title="D1-D10 Spread"
          value={formatPercent(summary.spread_cagr)}
          color={summary.spread_cagr > 0 ? '#10b981' : '#ef4444'}
          subtitle="Annual"
        />
        <MetricCard
          title="vs Benchmark"
          value={formatPercent(summary.top_vs_benchmark)}
          color={summary.top_vs_benchmark > 0 ? '#10b981' : '#ef4444'}
          subtitle={summary.benchmark}
        />
      </div>

      {/* Statistical Validity */}
      <div className="bg-white border rounded-lg p-6">
        <h3 className="text-lg font-semibold mb-4">Statistical Validity</h3>
        <div className="grid grid-cols-3 gap-6">
          <div>
            <p className="text-sm text-gray-600">Hit Rate</p>
            <p className="text-2xl font-bold">{formatPercent(summary.hit_rate)}</p>
            <p className="text-xs text-gray-500">D1 beats D10</p>
          </div>
          <div>
            <p className="text-sm text-gray-600">Monotonicity</p>
            <p className="text-2xl font-bold">{formatPercent(summary.monotonicity_score)}</p>
            <p className="text-xs text-gray-500">Rank ordering</p>
          </div>
          <div>
            <p className="text-sm text-gray-600">Information Ratio</p>
            <p className="text-2xl font-bold">{summary.information_ratio.toFixed(2)}</p>
            <p className="text-xs text-gray-500">
              {getRatingFromSharpe(summary.information_ratio)}
            </p>
          </div>
        </div>
      </div>

      {/* Risk Metrics */}
      <div className="bg-white border rounded-lg p-6">
        <h3 className="text-lg font-semibold mb-4">Risk Metrics</h3>
        <div className="grid grid-cols-3 gap-6">
          <div>
            <p className="text-sm text-gray-600">Top Decile Sharpe</p>
            <p className="text-2xl font-bold">{summary.top_decile_sharpe.toFixed(2)}</p>
            <p className="text-xs text-gray-500">
              {getRatingFromSharpe(summary.top_decile_sharpe)}
            </p>
          </div>
          <div>
            <p className="text-sm text-gray-600">Top Decile Max DD</p>
            <p className="text-2xl font-bold text-red-600">
              -{formatPercent(summary.top_decile_max_dd)}
            </p>
          </div>
          <div>
            <p className="text-sm text-gray-600">Bottom Decile Sharpe</p>
            <p className="text-2xl font-bold">{summary.bottom_decile_sharpe.toFixed(2)}</p>
            <p className="text-xs text-gray-500">
              {getRatingFromSharpe(summary.bottom_decile_sharpe)}
            </p>
          </div>
        </div>
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white border rounded-lg p-6">
          <h3 className="text-lg font-semibold mb-4">Decile Returns (CAGR)</h3>
          <DecileBarChart data={summary.decile_performance} />
        </div>
        <div className="bg-white border rounded-lg p-6">
          <h3 className="text-lg font-semibold mb-4">Cumulative Returns</h3>
          <CumulativeReturnsChart summary={summary} />
        </div>
      </div>

      {/* Decile Table */}
      <div className="bg-white border rounded-lg overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 border-b">
            <tr>
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-600">Decile</th>
              <th className="px-4 py-3 text-right text-sm font-medium text-gray-600">CAGR</th>
              <th className="px-4 py-3 text-right text-sm font-medium text-gray-600">Volatility</th>
              <th className="px-4 py-3 text-right text-sm font-medium text-gray-600">Sharpe</th>
              <th className="px-4 py-3 text-right text-sm font-medium text-gray-600">Max DD</th>
              <th className="px-4 py-3 text-right text-sm font-medium text-gray-600">Avg Score</th>
            </tr>
          </thead>
          <tbody>
            {summary.decile_performance.map((dp) => (
              <tr key={dp.decile} className="border-b hover:bg-gray-50">
                <td className="px-4 py-3">
                  <div className="flex items-center gap-2">
                    <span
                      className="w-3 h-3 rounded-full"
                      style={{ backgroundColor: getDecileColor(dp.decile) }}
                    />
                    <span className="font-medium">D{dp.decile}</span>
                  </div>
                </td>
                <td
                  className="px-4 py-3 text-right font-mono"
                  style={{ color: getReturnColor(dp.annualized_return) }}
                >
                  {formatPercent(dp.annualized_return)}
                </td>
                <td className="px-4 py-3 text-right font-mono text-gray-600">
                  {formatPercent(dp.volatility)}
                </td>
                <td className="px-4 py-3 text-right font-mono">{dp.sharpe_ratio.toFixed(2)}</td>
                <td className="px-4 py-3 text-right font-mono text-red-600">
                  -{formatPercent(dp.max_drawdown)}
                </td>
                <td className="px-4 py-3 text-right font-mono text-gray-600">
                  {dp.avg_score.toFixed(1)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Statistical Summary */}
      <StatisticalSummary summary={summary} />

      {/* Footer */}
      <div className="text-sm text-gray-500 text-center">
        <p>
          Backtest period: {summary.num_periods} {summary.rebalance_frequency} periods | Universe:{' '}
          {summary.universe.toUpperCase()} | Benchmark: {summary.benchmark}
        </p>
        <p className="mt-1">
          Past performance does not guarantee future results. Backtest results do not reflect actual
          trading.
        </p>
      </div>
    </div>
  );
}

function MetricCard({
  title,
  value,
  color,
  subtitle,
}: {
  title: string;
  value: string;
  color: string;
  subtitle?: string;
}) {
  return (
    <div className="bg-white border rounded-lg p-4">
      <p className="text-sm text-gray-600">{title}</p>
      <p className="text-2xl font-bold mt-1" style={{ color }}>
        {value}
      </p>
      {subtitle && <p className="text-xs text-gray-500 mt-1">{subtitle}</p>}
    </div>
  );
}
