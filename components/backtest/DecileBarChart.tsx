'use client';

import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
  ReferenceLine,
} from 'recharts';
import { DecilePerformance, formatPercent, getDecileColor } from '@/lib/types/backtest';

interface DecileBarChartProps {
  data: DecilePerformance[];
  height?: number;
}

export function DecileBarChart({ data, height = 300 }: DecileBarChartProps) {
  const chartData = data.map((dp) => ({
    name: `D${dp.decile}`,
    return: dp.annualized_return * 100,
    decile: dp.decile,
  }));

  return (
    <ResponsiveContainer width="100%" height={height}>
      <BarChart data={chartData} margin={{ top: 20, right: 20, bottom: 20, left: 20 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
        <XAxis dataKey="name" tick={{ fontSize: 12 }} tickLine={false} />
        <YAxis
          tick={{ fontSize: 12 }}
          tickLine={false}
          tickFormatter={(value) => `${value.toFixed(0)}%`}
          domain={['auto', 'auto']}
        />
        <Tooltip
          content={({ active, payload }) => {
            if (active && payload && payload.length) {
              const data = payload[0].payload;
              return (
                <div className="bg-white border rounded-lg shadow-lg p-3">
                  <p className="font-semibold">Decile {data.decile}</p>
                  <p className="text-sm">
                    CAGR:{' '}
                    <span style={{ color: data.return >= 0 ? '#10b981' : '#ef4444' }}>
                      {data.return.toFixed(2)}%
                    </span>
                  </p>
                </div>
              );
            }
            return null;
          }}
        />
        <ReferenceLine y={0} stroke="#9ca3af" strokeWidth={1} />
        <Bar dataKey="return" radius={[4, 4, 0, 0]}>
          {chartData.map((entry, index) => (
            <Cell key={`cell-${index}`} fill={getDecileColor(entry.decile)} />
          ))}
        </Bar>
      </BarChart>
    </ResponsiveContainer>
  );
}

interface DecileComparisonChartProps {
  data: DecilePerformance[];
  metric: 'annualized_return' | 'sharpe_ratio' | 'volatility' | 'max_drawdown';
  height?: number;
}

export function DecileComparisonChart({ data, metric, height = 300 }: DecileComparisonChartProps) {
  const labels: Record<string, string> = {
    annualized_return: 'CAGR',
    sharpe_ratio: 'Sharpe Ratio',
    volatility: 'Volatility',
    max_drawdown: 'Max Drawdown',
  };

  const formatValue = (value: number): string => {
    if (metric === 'sharpe_ratio') return value.toFixed(2);
    return `${(value * 100).toFixed(1)}%`;
  };

  const chartData = data.map((dp) => ({
    name: `D${dp.decile}`,
    value:
      metric === 'max_drawdown'
        ? -dp[metric] * 100
        : dp[metric] * (metric === 'sharpe_ratio' ? 1 : 100),
    decile: dp.decile,
    raw: dp[metric],
  }));

  return (
    <div>
      <h4 className="text-sm font-medium text-gray-600 mb-2">{labels[metric]}</h4>
      <ResponsiveContainer width="100%" height={height}>
        <BarChart data={chartData} margin={{ top: 10, right: 10, bottom: 10, left: 10 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
          <XAxis dataKey="name" tick={{ fontSize: 10 }} tickLine={false} />
          <YAxis tick={{ fontSize: 10 }} tickLine={false} />
          <Tooltip
            content={({ active, payload }) => {
              if (active && payload && payload.length) {
                const data = payload[0].payload;
                return (
                  <div className="bg-white border rounded-lg shadow-lg p-2 text-sm">
                    <p className="font-medium">
                      D{data.decile}: {formatValue(data.raw)}
                    </p>
                  </div>
                );
              }
              return null;
            }}
          />
          <Bar dataKey="value" radius={[2, 2, 0, 0]}>
            {chartData.map((entry, index) => (
              <Cell key={`cell-${index}`} fill={getDecileColor(entry.decile)} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
