'use client';

import { useState, useEffect } from 'react';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Title,
  Tooltip,
  Legend,
  TimeScale,
  Filler,
} from 'chart.js';
import { Line } from 'react-chartjs-2';
import 'chartjs-adapter-date-fns';

// Register Chart.js components
ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Title,
  Tooltip,
  Legend,
  TimeScale,
  Filler
);

interface InteractiveChartProps {
  symbol: string;
}

interface ChartDataPoint {
  timestamp: string;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
}

type TimeframePeriod = '1D' | '5D' | '1M' | '3M' | '6M' | '1Y' | '5Y';

export default function InteractiveChart({ symbol }: InteractiveChartProps) {
  const [chartData, setChartData] = useState<ChartDataPoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedPeriod, setSelectedPeriod] = useState<TimeframePeriod>('1Y');
  const [error, setError] = useState<string | null>(null);

  const timeframes: { period: TimeframePeriod; label: string; }[] = [
    { period: '1D', label: '1D' },
    { period: '5D', label: '5D' },
    { period: '1M', label: '1M' },
    { period: '3M', label: '3M' },
    { period: '6M', label: '6M' },
    { period: '1Y', label: '1Y' },
    { period: '5Y', label: '5Y' },
  ];

  useEffect(() => {
    const fetchChartData = async () => {
      try {
        setLoading(true);
        console.log(`📈 Fetching ${selectedPeriod} chart data for ${symbol}...`);
        
        const response = await fetch(`/api/v1/tickers/${symbol}/chart?period=${selectedPeriod}`);
        
        if (!response.ok) {
          throw new Error(`Failed to fetch chart data: ${response.status}`);
        }
        
        const result = await response.json();
        console.log(`📊 Chart data received:`, result);
        
        setChartData(result.data || []);
        setError(null);
      } catch (err) {
        console.error('❌ Error fetching chart data:', err);
        setError(err instanceof Error ? err.message : 'Failed to fetch chart data');
      } finally {
        setLoading(false);
      }
    };

    fetchChartData();
  }, [symbol, selectedPeriod]);

  // Prepare Chart.js data
  const chartJsData = {
    labels: chartData.map(point => new Date(point.timestamp)),
    datasets: [
      {
        label: `${symbol} Price`,
        data: chartData.map(point => point.close),
        borderColor: '#2563eb',
        backgroundColor: 'rgba(37, 99, 235, 0.1)',
        borderWidth: 2,
        fill: true,
        tension: 0.1,
        pointRadius: 0,
        pointHoverRadius: 4,
      },
    ],
  };

  // Chart.js options
  const chartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    interaction: {
      mode: 'index' as const,
      intersect: false,
    },
    plugins: {
      legend: {
        display: false,
      },
      tooltip: {
        mode: 'index' as const,
        intersect: false,
        backgroundColor: 'rgba(0, 0, 0, 0.8)',
        titleColor: 'white',
        bodyColor: 'white',
        borderColor: '#2563eb',
        borderWidth: 1,
        callbacks: {
          title: (context: any) => {
            const date = new Date(context[0].parsed.x);
            return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
          },
          label: (context: any) => {
            const dataPoint = chartData[context.dataIndex];
            if (!dataPoint) return '';
            
            return [
              `Close: $${context.parsed.y.toFixed(2)}`,
              `Open: $${dataPoint.open.toFixed(2)}`,
              `High: $${dataPoint.high.toFixed(2)}`,
              `Low: $${dataPoint.low.toFixed(2)}`,
              `Volume: ${dataPoint.volume.toLocaleString()}`,
            ];
          },
        },
      },
    },
    scales: {
      x: {
        type: 'time' as const,
        time: {
          displayFormats: {
            minute: 'HH:mm',
            hour: 'MMM dd HH:mm',
            day: 'MMM dd',
            week: 'MMM dd',
            month: 'MMM yyyy',
            year: 'yyyy',
          },
        },
        grid: {
          display: false,
        },
        ticks: {
          color: '#6b7280',
        },
      },
      y: {
        position: 'right' as const,
        grid: {
          color: 'rgba(0, 0, 0, 0.1)',
        },
        ticks: {
          color: '#6b7280',
          callback: function(value: any) {
            return '$' + value.toFixed(2);
          },
        },
      },
    },
    elements: {
      point: {
        hoverRadius: 8,
      },
    },
  };

  if (loading) {
    return (
      <div className="p-6">
        <div className="flex items-center justify-between mb-6">
          <h3 className="text-lg font-semibold text-gray-900">Interactive Price Chart</h3>
          <div className="flex space-x-1 bg-gray-100 rounded-lg p-1">
            {timeframes.map(({ period, label }) => (
              <button
                key={period}
                className="px-3 py-1 text-sm font-medium rounded-md transition-colors text-gray-600"
                disabled
              >
                {label}
              </button>
            ))}
          </div>
        </div>
        <div className="h-80 bg-gray-200 rounded-lg animate-pulse flex items-center justify-center">
          <div className="text-gray-500">Loading chart data...</div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <h3 className="text-red-800 font-semibold">Chart Error</h3>
          <p className="text-red-600 text-sm mt-1">{error}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-lg font-semibold text-gray-900">Interactive Price Chart</h3>
        <div className="flex space-x-1 bg-gray-100 rounded-lg p-1">
          {timeframes.map(({ period, label }) => (
            <button
              key={period}
              onClick={() => setSelectedPeriod(period)}
              className={`px-3 py-1 text-sm font-medium rounded-md transition-colors ${
                selectedPeriod === period
                  ? 'bg-white text-primary-600 shadow-sm'
                  : 'text-gray-600 hover:text-gray-900'
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      </div>
      
      <div className="h-80 relative">
        <Line data={chartJsData} options={chartOptions} />
      </div>
      
      {/* Chart Stats */}
      {chartData.length > 0 && (
        <div className="mt-4 grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
          <div>
            <div className="text-gray-500">Current</div>
            <div className="font-semibold">${chartData[chartData.length - 1]?.close.toFixed(2)}</div>
          </div>
          <div>
            <div className="text-gray-500">High</div>
            <div className="font-semibold text-green-600">
              ${Math.max(...chartData.map(p => p.high)).toFixed(2)}
            </div>
          </div>
          <div>
            <div className="text-gray-500">Low</div>
            <div className="font-semibold text-red-600">
              ${Math.min(...chartData.map(p => p.low)).toFixed(2)}
            </div>
          </div>
          <div>
            <div className="text-gray-500">Avg Volume</div>
            <div className="font-semibold">
              {(chartData.reduce((sum, p) => sum + p.volume, 0) / chartData.length / 1000000).toFixed(1)}M
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
