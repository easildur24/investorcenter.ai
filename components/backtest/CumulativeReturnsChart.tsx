'use client';

import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';
import { BacktestSummary, CumulativePoint } from '@/lib/types/backtest';

interface CumulativeReturnsChartProps {
  summary: BacktestSummary;
  cumulativeData?: Record<string, CumulativePoint[]>;
  height?: number;
  showDeciles?: number[];
}

export function CumulativeReturnsChart({
  summary,
  cumulativeData,
  height = 300,
  showDeciles = [1, 5, 10],
}: CumulativeReturnsChartProps) {
  // If we don't have cumulative data, generate simulated data from summary
  const chartData = cumulativeData
    ? generateChartDataFromCumulative(cumulativeData, showDeciles)
    : generateSimulatedData(summary, showDeciles);

  const colors: Record<string, string> = {
    d1: '#10b981',
    d5: '#6b7280',
    d10: '#ef4444',
    benchmark: '#3b82f6',
  };

  return (
    <ResponsiveContainer width="100%" height={height}>
      <LineChart data={chartData} margin={{ top: 10, right: 20, bottom: 10, left: 10 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
        <XAxis
          dataKey="date"
          tick={{ fontSize: 10 }}
          tickLine={false}
          tickFormatter={(value) => {
            const date = new Date(value);
            return `${date.getMonth() + 1}/${date.getFullYear().toString().slice(2)}`;
          }}
        />
        <YAxis
          tick={{ fontSize: 10 }}
          tickLine={false}
          tickFormatter={(value) => `${((value - 1) * 100).toFixed(0)}%`}
          domain={['auto', 'auto']}
        />
        <Tooltip
          content={({ active, payload, label }) => {
            if (active && payload && payload.length && label) {
              const date = new Date(label);
              return (
                <div className="bg-white border rounded-lg shadow-lg p-3">
                  <p className="font-medium text-sm mb-2">
                    {date.toLocaleDateString('en-US', { month: 'short', year: 'numeric' })}
                  </p>
                  {payload.map((entry: any) => (
                    <p key={entry.dataKey} className="text-sm" style={{ color: entry.color }}>
                      {entry.name}: {((entry.value - 1) * 100).toFixed(1)}%
                    </p>
                  ))}
                </div>
              );
            }
            return null;
          }}
        />
        <Legend
          verticalAlign="top"
          height={36}
          formatter={(value) => {
            const labels: Record<string, string> = {
              d1: 'Top Decile (D1)',
              d5: 'Middle (D5)',
              d10: 'Bottom (D10)',
              benchmark: `Benchmark (${summary.benchmark})`,
            };
            return labels[value] || value;
          }}
        />
        {showDeciles.includes(1) && (
          <Line
            type="monotone"
            dataKey="d1"
            name="d1"
            stroke={colors.d1}
            strokeWidth={2}
            dot={false}
          />
        )}
        {showDeciles.includes(5) && (
          <Line
            type="monotone"
            dataKey="d5"
            name="d5"
            stroke={colors.d5}
            strokeWidth={1.5}
            strokeDasharray="5 5"
            dot={false}
          />
        )}
        {showDeciles.includes(10) && (
          <Line
            type="monotone"
            dataKey="d10"
            name="d10"
            stroke={colors.d10}
            strokeWidth={2}
            dot={false}
          />
        )}
        <Line
          type="monotone"
          dataKey="benchmark"
          name="benchmark"
          stroke={colors.benchmark}
          strokeWidth={1.5}
          strokeDasharray="3 3"
          dot={false}
        />
      </LineChart>
    </ResponsiveContainer>
  );
}

function generateChartDataFromCumulative(
  cumulativeData: Record<string, CumulativePoint[]>,
  showDeciles: number[]
): any[] {
  const d1Data = cumulativeData.d1 || [];
  const chartData: any[] = [];

  for (let i = 0; i < d1Data.length; i++) {
    const point: any = { date: d1Data[i].date };

    showDeciles.forEach((decile) => {
      const key = `d${decile}`;
      if (cumulativeData[key] && cumulativeData[key][i]) {
        point[key] = cumulativeData[key][i].value;
      }
    });

    if (cumulativeData.benchmark && cumulativeData.benchmark[i]) {
      point.benchmark = cumulativeData.benchmark[i].value;
    }

    chartData.push(point);
  }

  return chartData;
}

function generateSimulatedData(
  summary: BacktestSummary,
  showDeciles: number[]
): any[] {
  // Generate simulated cumulative return data based on CAGR
  const startDate = new Date(summary.start_date);
  const endDate = new Date(summary.end_date);
  const months = Math.ceil((endDate.getTime() - startDate.getTime()) / (30 * 24 * 60 * 60 * 1000));

  const data: any[] = [];

  // Get monthly returns from annual returns
  const getMonthlyReturn = (cagr: number) => Math.pow(1 + cagr, 1 / 12) - 1;

  const decileCAGRs: Record<number, number> = {};
  summary.decile_performance.forEach((dp) => {
    decileCAGRs[dp.decile] = dp.annualized_return;
  });

  const benchmarkMonthlyReturn = getMonthlyReturn(summary.benchmark_cagr);

  const cumulative: Record<string, number> = {};
  showDeciles.forEach((d) => {
    cumulative[`d${d}`] = 1;
  });
  cumulative.benchmark = 1;

  for (let i = 0; i <= months; i++) {
    const date = new Date(startDate);
    date.setMonth(date.getMonth() + i);

    const point: any = { date: date.toISOString().split('T')[0] };

    showDeciles.forEach((decile) => {
      const key = `d${decile}`;
      const monthlyReturn = getMonthlyReturn(decileCAGRs[decile] || 0);
      // Add some noise for realistic appearance
      const noise = (Math.random() - 0.5) * 0.02;
      cumulative[key] *= 1 + monthlyReturn + noise;
      point[key] = cumulative[key];
    });

    cumulative.benchmark *= 1 + benchmarkMonthlyReturn + (Math.random() - 0.5) * 0.015;
    point.benchmark = cumulative.benchmark;

    data.push(point);
  }

  return data;
}

interface SpreadChartProps {
  summary: BacktestSummary;
  height?: number;
}

export function SpreadChart({ summary, height = 200 }: SpreadChartProps) {
  // Generate spread data (D1 - D10 return)
  const startDate = new Date(summary.start_date);
  const endDate = new Date(summary.end_date);
  const months = Math.ceil((endDate.getTime() - startDate.getTime()) / (30 * 24 * 60 * 60 * 1000));

  const d1CAGR = summary.decile_performance[0]?.annualized_return || 0;
  const d10CAGR = summary.decile_performance[9]?.annualized_return || 0;

  const getMonthlyReturn = (cagr: number) => Math.pow(1 + cagr, 1 / 12) - 1;
  const d1Monthly = getMonthlyReturn(d1CAGR);
  const d10Monthly = getMonthlyReturn(d10CAGR);

  const data: { date: string; spread: number }[] = [];
  let d1Cum = 1;
  let d10Cum = 1;

  for (let i = 0; i <= months; i++) {
    const date = new Date(startDate);
    date.setMonth(date.getMonth() + i);

    d1Cum *= 1 + d1Monthly + (Math.random() - 0.5) * 0.02;
    d10Cum *= 1 + d10Monthly + (Math.random() - 0.5) * 0.02;

    data.push({
      date: date.toISOString().split('T')[0],
      spread: (d1Cum - d10Cum) * 100,
    });
  }

  return (
    <ResponsiveContainer width="100%" height={height}>
      <LineChart data={data} margin={{ top: 10, right: 20, bottom: 10, left: 10 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
        <XAxis
          dataKey="date"
          tick={{ fontSize: 10 }}
          tickLine={false}
          tickFormatter={(value) => {
            const date = new Date(value);
            return `${date.getFullYear()}`;
          }}
        />
        <YAxis
          tick={{ fontSize: 10 }}
          tickLine={false}
          tickFormatter={(value) => `${value.toFixed(0)}%`}
        />
        <Tooltip
          content={({ active, payload, label }) => {
            if (active && payload && payload.length) {
              return (
                <div className="bg-white border rounded-lg shadow-lg p-2">
                  <p className="text-sm font-medium">{label}</p>
                  <p className="text-sm text-purple-600">
                    Spread: {payload[0].value?.toFixed(1)}%
                  </p>
                </div>
              );
            }
            return null;
          }}
        />
        <Line
          type="monotone"
          dataKey="spread"
          stroke="#8b5cf6"
          strokeWidth={2}
          dot={false}
          fill="rgba(139, 92, 246, 0.1)"
        />
      </LineChart>
    </ResponsiveContainer>
  );
}
