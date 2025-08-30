'use client';

import { useState, useEffect } from 'react';
import { safeParseNumber, safeToFixed } from '@/lib/utils';

interface TickerChartProps {
  symbol: string;
}

interface ChartDataPoint {
  timestamp: string;
  open: number | string;
  high: number | string;
  low: number | string;
  close: number | string;
  volume: number | string;
}

interface ChartData {
  symbol: string;
  period: string;
  dataPoints: ChartDataPoint[];
  lastUpdated: string;
}

export default function TickerChart({ symbol }: TickerChartProps) {
  const [chartData, setChartData] = useState<ChartData | null>(null);
  const [selectedPeriod, setSelectedPeriod] = useState('1Y');
  const [loading, setLoading] = useState(true);

  const periods = [
    { label: '1D', value: '1D' },
    { label: '5D', value: '5D' },
    { label: '1M', value: '1M' },
    { label: '3M', value: '3M' },
    { label: '6M', value: '6M' },
    { label: '1Y', value: '1Y' },
    { label: '5Y', value: '5Y' },
  ];

  useEffect(() => {
    const fetchChartData = async () => {
      try {
        setLoading(true);
        const response = await fetch(`/api/v1/tickers/${symbol}/chart?period=${selectedPeriod}`);
        const result = await response.json();
        setChartData(result.data);
      } catch (error) {
        console.error('Error fetching chart data:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchChartData();
  }, [symbol, selectedPeriod]);

  // Simple price trend visualization (you can replace with a proper charting library)
  const renderSimpleChart = () => {
    if (!chartData || chartData.dataPoints.length === 0) return null;

    const prices = chartData.dataPoints.map(point => safeParseNumber(point.close));
    const minPrice = Math.min(...prices);
    const maxPrice = Math.max(...prices);
    const priceRange = maxPrice - minPrice;

    return (
      <div className="relative h-80 bg-gray-50 rounded-lg p-4">
        <svg className="w-full h-full" viewBox="0 0 800 300">
          <defs>
            <linearGradient id="priceGradient" x1="0%" y1="0%" x2="0%" y2="100%">
              <stop offset="0%" stopColor="#3b82f6" stopOpacity="0.3" />
              <stop offset="100%" stopColor="#3b82f6" stopOpacity="0.05" />
            </linearGradient>
          </defs>
          
          {/* Price Line */}
          <path
            d={`M ${chartData.dataPoints.map((point, index) => {
              const x = (index / (chartData.dataPoints.length - 1)) * 780 + 10;
              const y = 280 - ((safeParseNumber(point.close) - minPrice) / priceRange) * 260;
              return `${index === 0 ? 'M' : 'L'} ${x} ${y}`;
            }).join(' ')}`}
            stroke="#3b82f6"
            strokeWidth="2"
            fill="none"
          />
          
          {/* Fill Area */}
          <path
            d={`M ${chartData.dataPoints.map((point, index) => {
              const x = (index / (chartData.dataPoints.length - 1)) * 780 + 10;
              const y = 280 - ((safeParseNumber(point.close) - minPrice) / priceRange) * 260;
              return `${index === 0 ? 'M' : 'L'} ${x} ${y}`;
            }).join(' ')} L 790 280 L 10 280 Z`}
            fill="url(#priceGradient)"
          />
        </svg>
        
        {/* Price Labels */}
        <div className="absolute top-4 left-4 text-sm text-gray-600">
          ${maxPrice.toFixed(2)}
        </div>
        <div className="absolute bottom-4 left-4 text-sm text-gray-600">
          ${minPrice.toFixed(2)}
        </div>
      </div>
    );
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-lg font-semibold text-gray-900">Price Chart</h3>
        
        {/* Period Selector */}
        <div className="flex space-x-1 bg-gray-100 rounded-lg p-1">
          {periods.map((period) => (
            <button
              key={period.value}
              onClick={() => setSelectedPeriod(period.value)}
              className={`px-3 py-1 text-sm font-medium rounded-md transition-colors ${
                selectedPeriod === period.value
                  ? 'bg-white text-primary-600 shadow-sm'
                  : 'text-gray-600 hover:text-gray-900'
              }`}
            >
              {period.label}
            </button>
          ))}
        </div>
      </div>

      {loading ? (
        <div className="h-80 bg-gray-200 rounded-lg animate-pulse"></div>
      ) : (
        <>
          {renderSimpleChart()}
          
          {/* Chart Stats */}
          {chartData && (
            <div className="mt-4 grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
              <div>
                <div className="text-gray-500">Open</div>
                <div className="font-medium">${safeToFixed(chartData.dataPoints[0]?.open, 2)}</div>
              </div>
              <div>
                <div className="text-gray-500">High</div>
                <div className="font-medium text-green-600">
                  ${safeToFixed(Math.max(...chartData.dataPoints.map(p => safeParseNumber(p.high))), 2)}
                </div>
              </div>
              <div>
                <div className="text-gray-500">Low</div>
                <div className="font-medium text-red-600">
                  ${safeToFixed(Math.min(...chartData.dataPoints.map(p => safeParseNumber(p.low))), 2)}
                </div>
              </div>
              <div>
                <div className="text-gray-500">Volume</div>
                <div className="font-medium">
                  {safeToFixed(safeParseNumber(chartData.dataPoints[chartData.dataPoints.length - 1]?.volume) / 1000000, 1)}M
                </div>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
