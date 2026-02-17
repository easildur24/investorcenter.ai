'use client';

import { useState, useEffect } from 'react';
import { cn, safeToFixed, safeParseNumber, formatPercent } from '@/lib/utils';

interface RiskTabProps {
  symbol: string;
}

interface RiskMetrics {
  ticker?: string;
  period?: string;
  calculation_date?: string;
  data_points?: number;
  beta?: number;
  volatility?: number;
  annualized_return?: number;
  sharpe_ratio?: number;
  sortino_ratio?: number;
  max_drawdown?: number;
  var_95?: number;
  alpha?: number;
  downside_deviation?: number;
}

function getBetaInterpretation(beta: number): {
  label: string;
  description: string;
  color: string;
} {
  if (beta > 1.5)
    return {
      label: 'High Risk',
      description: 'Much more volatile than market',
      color: 'text-ic-negative bg-ic-negative-bg',
    };
  if (beta > 1.1)
    return {
      label: 'Above Market',
      description: 'More volatile than market',
      color: 'text-orange-600 bg-orange-50',
    };
  if (beta >= 0.9)
    return {
      label: 'Market-Like',
      description: 'Similar to market',
      color: 'text-ic-text-muted bg-ic-bg-secondary',
    };
  if (beta >= 0.5)
    return {
      label: 'Defensive',
      description: 'Less volatile than market',
      color: 'text-ic-positive bg-ic-positive-bg',
    };
  return {
    label: 'Low Beta',
    description: 'Much less volatile',
    color: 'text-blue-600 bg-blue-50',
  };
}

function getSharpeInterpretation(sharpe: number): { label: string; color: string } {
  if (sharpe >= 2) return { label: 'Excellent', color: 'text-ic-positive' };
  if (sharpe >= 1) return { label: 'Good', color: 'text-ic-positive' };
  if (sharpe >= 0.5) return { label: 'Acceptable', color: 'text-ic-warning' };
  if (sharpe >= 0) return { label: 'Poor', color: 'text-orange-600' };
  return { label: 'Negative', color: 'text-ic-negative' };
}

export default function RiskTab({ symbol }: RiskTabProps) {
  const [data, setData] = useState<RiskMetrics | null>(null);
  const [period, setPeriod] = useState('1Y');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        const response = await fetch(`/api/v1/stocks/${symbol}/risk?period=${period}`);
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`);
        }
        const result = await response.json();
        setData(result.data || {});
      } catch (err) {
        console.error('Error fetching risk data:', err);
        setError('Failed to load risk metrics');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [symbol, period]);

  if (loading) {
    return (
      <div className="p-6 animate-pulse">
        <div className="h-6 bg-ic-bg-tertiary rounded w-48 mb-6"></div>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {[1, 2, 3, 4, 5, 6, 7, 8].map((i) => (
            <div key={i} className="bg-ic-bg-secondary rounded-lg p-4">
              <div className="h-4 bg-ic-bg-tertiary rounded w-20 mb-2"></div>
              <div className="h-6 bg-ic-bg-tertiary rounded w-16"></div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="p-6">
        <h3 className="text-lg font-semibold text-ic-text-primary mb-4">Risk Metrics</h3>
        <p className="text-ic-text-muted">{error || 'No risk data available'}</p>
      </div>
    );
  }

  const betaInfo = data.beta !== undefined ? getBetaInterpretation(data.beta) : null;
  const sharpeInfo =
    data.sharpe_ratio !== undefined ? getSharpeInterpretation(data.sharpe_ratio) : null;

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-lg font-semibold text-ic-text-primary">Risk Metrics</h3>
        <div className="flex gap-2">
          {['3M', '6M', '1Y', '3Y', '5Y'].map((p) => (
            <button
              key={p}
              onClick={() => setPeriod(p)}
              className={cn(
                'px-3 py-1 text-sm rounded-md transition-colors',
                period === p
                  ? 'bg-ic-blue text-ic-text-primary'
                  : 'bg-ic-bg-secondary text-ic-text-muted hover:bg-ic-surface-hover'
              )}
            >
              {p}
            </button>
          ))}
        </div>
      </div>

      {/* Data Info */}
      {data.calculation_date && (
        <div className="text-sm text-ic-text-muted mb-4">
          Calculated: {new Date(data.calculation_date).toLocaleDateString()} •{' '}
          {data.data_points || 0} data points
        </div>
      )}

      {/* Key Risk Metrics */}
      <div className="mb-8">
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Key Metrics
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {/* Beta */}
          <div className="bg-ic-bg-secondary rounded-lg p-4">
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm text-ic-text-muted">Beta</span>
              {betaInfo && (
                <span className={cn('text-xs px-2 py-0.5 rounded-full', betaInfo.color)}>
                  {betaInfo.label}
                </span>
              )}
            </div>
            <div className="text-2xl font-semibold text-ic-text-primary">
              {safeToFixed(data.beta, 2)}
            </div>
            {betaInfo && (
              <div className="text-xs text-ic-text-muted mt-1">{betaInfo.description}</div>
            )}
          </div>

          {/* Volatility */}
          <div className="bg-ic-bg-secondary rounded-lg p-4">
            <div className="text-sm text-ic-text-muted mb-2">Annualized Volatility</div>
            <div className="text-2xl font-semibold text-ic-text-primary">
              {safeToFixed(data.volatility, 1)}%
            </div>
            <div className="text-xs text-ic-text-muted mt-1">Standard deviation of returns</div>
          </div>

          {/* Sharpe Ratio */}
          <div className="bg-ic-bg-secondary rounded-lg p-4">
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm text-ic-text-muted">Sharpe Ratio</span>
              {sharpeInfo && (
                <span className={cn('text-xs', sharpeInfo.color)}>{sharpeInfo.label}</span>
              )}
            </div>
            <div className="text-2xl font-semibold text-ic-text-primary">
              {safeToFixed(data.sharpe_ratio, 2)}
            </div>
            <div className="text-xs text-ic-text-muted mt-1">Risk-adjusted return</div>
          </div>

          {/* Max Drawdown */}
          <div className="bg-ic-bg-secondary rounded-lg p-4">
            <div className="text-sm text-ic-text-muted mb-2">Max Drawdown</div>
            <div className="text-2xl font-semibold text-ic-negative">
              {safeToFixed(data.max_drawdown, 1)}%
            </div>
            <div className="text-xs text-ic-text-muted mt-1">Largest peak-to-trough decline</div>
          </div>
        </div>
      </div>

      {/* Risk-Adjusted Performance */}
      <div className="mb-8">
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Risk-Adjusted Performance
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {/* Sortino Ratio */}
          <div className="bg-ic-bg-secondary rounded-lg p-4">
            <div className="text-sm text-ic-text-muted mb-2">Sortino Ratio</div>
            <div className="text-xl font-semibold text-ic-text-primary">
              {safeToFixed(data.sortino_ratio, 2)}
            </div>
            <div className="text-xs text-ic-text-muted mt-1">Downside risk-adjusted</div>
          </div>

          {/* Alpha */}
          <div className="bg-ic-bg-secondary rounded-lg p-4">
            <div className="text-sm text-ic-text-muted mb-2">Alpha</div>
            <div
              className={cn(
                'text-xl font-semibold',
                safeParseNumber(data.alpha) >= 0 ? 'text-ic-positive' : 'text-ic-negative'
              )}
            >
              {safeParseNumber(data.alpha) >= 0 ? '+' : ''}
              {safeToFixed(data.alpha, 2)}%
            </div>
            <div className="text-xs text-ic-text-muted mt-1">Excess return vs benchmark</div>
          </div>

          {/* Annualized Return */}
          <div className="bg-ic-bg-secondary rounded-lg p-4">
            <div className="text-sm text-ic-text-muted mb-2">Annualized Return</div>
            <div
              className={cn(
                'text-xl font-semibold',
                safeParseNumber(data.annualized_return) >= 0
                  ? 'text-ic-positive'
                  : 'text-ic-negative'
              )}
            >
              {safeParseNumber(data.annualized_return) >= 0 ? '+' : ''}
              {safeToFixed(data.annualized_return, 1)}%
            </div>
            <div className="text-xs text-ic-text-muted mt-1">Yearly return rate</div>
          </div>

          {/* Downside Deviation */}
          <div className="bg-ic-bg-secondary rounded-lg p-4">
            <div className="text-sm text-ic-text-muted mb-2">Downside Deviation</div>
            <div className="text-xl font-semibold text-ic-text-primary">
              {safeToFixed(data.downside_deviation, 1)}%
            </div>
            <div className="text-xs text-ic-text-muted mt-1">Negative return volatility</div>
          </div>
        </div>
      </div>

      {/* Value at Risk */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Value at Risk (VaR)
        </h4>
        <div className="bg-ic-bg-secondary rounded-lg p-4">
          <div className="text-sm text-ic-text-muted mb-2">VaR (95% Confidence)</div>
          <div className="text-2xl font-semibold text-ic-negative">
            {safeToFixed(data.var_95, 2)}%
          </div>
          <div className="text-sm text-ic-text-muted mt-2">
            There is a 5% chance of losing more than{' '}
            {safeToFixed(Math.abs(safeParseNumber(data.var_95)), 1)}% in a single day.
          </div>
        </div>
      </div>

      {/* Interpretation */}
      <div className="mt-8 p-4 bg-blue-50 rounded-lg">
        <h4 className="text-sm font-medium text-blue-800 mb-2">Understanding These Metrics</h4>
        <ul className="text-sm text-blue-700 space-y-1">
          <li>
            • <strong>Beta &gt; 1:</strong> Stock is more volatile than the market (S&P 500)
          </li>
          <li>
            • <strong>Sharpe Ratio &gt; 1:</strong> Good risk-adjusted returns
          </li>
          <li>
            • <strong>Positive Alpha:</strong> Outperforming the benchmark
          </li>
          <li>
            • <strong>Sortino &gt; Sharpe:</strong> More upside than downside volatility
          </li>
        </ul>
      </div>
    </div>
  );
}
