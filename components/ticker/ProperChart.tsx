'use client';

import { useState, useEffect } from 'react';

interface ProperChartProps {
  symbol: string;
  initialData?: any;
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

export default function ProperChart({ symbol, initialData }: ProperChartProps) {
  const [chartData, setChartData] = useState<ChartDataPoint[]>(initialData?.dataPoints || []);
  const [loading, setLoading] = useState(false);
  const [selectedPeriod, setSelectedPeriod] = useState<TimeframePeriod>('1Y');
  const [error, setError] = useState<string | null>(null);
  const [hoveredPoint, setHoveredPoint] = useState<ChartDataPoint | null>(null);
  const [mousePosition, setMousePosition] = useState({ x: 0, y: 0 });

  const timeframes: { period: TimeframePeriod; label: string; }[] = [
    { period: '1D', label: '1D' },
    { period: '5D', label: '5D' },
    { period: '1M', label: '1M' },
    { period: '3M', label: '3M' },
    { period: '6M', label: '6M' },
    { period: '1Y', label: '1Y' },
    { period: '5Y', label: '5Y' },
  ];

  const fetchChartData = async (period: TimeframePeriod) => {
    try {
      setLoading(true);
      setError(null);
      console.log(`📈 Fetching ${period} data for ${symbol}...`);
      
      const response = await fetch(`/api/v1/tickers/${symbol}/chart?period=${period}`);
      
      if (!response.ok) {
        throw new Error(`Failed to fetch ${period} data: ${response.status}`);
      }
      
      const result = await response.json();
      console.log(`📊 Received ${period} data:`, result.data?.dataPoints?.length, 'points');
      
      if (result.data?.dataPoints) {
        setChartData(result.data.dataPoints);
      }
    } catch (err) {
      console.error('❌ Chart error:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch chart data');
    } finally {
      setLoading(false);
    }
  };

  const handleTimeframeChange = async (period: TimeframePeriod) => {
    if (period === selectedPeriod) return;
    
    setSelectedPeriod(period);
    await fetchChartData(period);
  };

  // Calculate chart dimensions and statistics
  const chartWidth = 800;
  const chartHeight = 300;
  const padding = 40;
  
  if (!chartData || chartData.length === 0) {
    return (
      <div className="p-6">
        <div className="flex items-center justify-between mb-6">
          <h3 className="text-lg font-semibold text-gray-900">Interactive Price Chart</h3>
          <div className="flex space-x-1 bg-gray-100 rounded-lg p-1">
            {timeframes.map(({ period, label }) => (
              <button
                key={period}
                onClick={() => handleTimeframeChange(period)}
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
        <div className="h-80 bg-gray-100 rounded-lg flex items-center justify-center">
          <div className="text-gray-500">No chart data available</div>
        </div>
      </div>
    );
  }

  const prices = chartData.map(d => d.close);
  const high = Math.max(...prices);
  const low = Math.min(...prices);
  const priceRange = high - low || 1;
  const priceScale = (chartHeight - 2 * padding) / priceRange;

  const currentPrice = prices[prices.length - 1];
  const firstPrice = prices[0];
  const priceChange = currentPrice - firstPrice;
  const priceChangePercent = firstPrice > 0 ? (priceChange / firstPrice) * 100 : 0;

  // Generate SVG path
  const pathData = chartData.map((point, index) => {
    const x = padding + (index / (chartData.length - 1)) * (chartWidth - 2 * padding);
    const y = chartHeight - padding - (point.close - low) * priceScale;
    return `${index === 0 ? 'M' : 'L'} ${x} ${y}`;
  }).join(' ');

  // Handle mouse events
  const handleMouseMove = (event: React.MouseEvent<SVGSVGElement>) => {
    const rect = event.currentTarget.getBoundingClientRect();
    const x = event.clientX - rect.left;
    const y = event.clientY - rect.top;
    
    // Convert mouse position to data index
    const dataIndex = Math.round(((x - padding) / (chartWidth - 2 * padding)) * (chartData.length - 1));
    
    if (dataIndex >= 0 && dataIndex < chartData.length) {
      setHoveredPoint(chartData[dataIndex]);
      setMousePosition({ x: event.clientX, y: event.clientY });
    }
  };

  const handleMouseLeave = () => {
    setHoveredPoint(null);
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-lg font-semibold text-gray-900">Interactive Price Chart</h3>
        <div className="flex space-x-1 bg-gray-100 rounded-lg p-1">
          {timeframes.map(({ period, label }) => (
            <button
              key={period}
              onClick={() => handleTimeframeChange(period)}
              disabled={loading}
              className={`px-3 py-1 text-sm font-medium rounded-md transition-colors ${
                selectedPeriod === period
                  ? 'bg-white text-primary-600 shadow-sm'
                  : loading 
                    ? 'text-gray-400 cursor-not-allowed'
                    : 'text-gray-600 hover:text-gray-900'
              }`}
            >
              {loading && selectedPeriod === period ? '...' : label}
            </button>
          ))}
        </div>
      </div>
      
      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg">
          <p className="text-red-600 text-sm">{error}</p>
        </div>
      )}
      
      {/* Interactive SVG Chart */}
      <div className="relative h-80 bg-white border rounded-lg overflow-hidden">
        <svg 
          width={chartWidth} 
          height={chartHeight} 
          className="w-full h-full cursor-crosshair"
          onMouseMove={handleMouseMove}
          onMouseLeave={handleMouseLeave}
        >
          {/* Grid */}
          <defs>
            <pattern id="chartGrid" width="50" height="30" patternUnits="userSpaceOnUse">
              <path d="M 50 0 L 0 0 0 30" fill="none" stroke="#f3f4f6" strokeWidth="1"/>
            </pattern>
          </defs>
          <rect width="100%" height="100%" fill="url(#chartGrid)" />
          
          {/* Price area fill */}
          <path
            d={`${pathData} L ${chartWidth - padding} ${chartHeight - padding} L ${padding} ${chartHeight - padding} Z`}
            fill="rgba(34, 197, 94, 0.1)"
            stroke="none"
          />
          
          {/* Price line */}
          <path
            d={pathData}
            fill="none"
            stroke="#22c55e"
            strokeWidth="3"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
          
          {/* Y-axis price labels */}
          <text x="10" y="25" className="text-xs fill-gray-600 font-medium">${high.toFixed(2)}</text>
          <text x="10" y={chartHeight - 10} className="text-xs fill-gray-600 font-medium">${low.toFixed(2)}</text>
          
          {/* Current price indicator */}
          <circle
            cx={chartWidth - padding}
            cy={chartHeight - padding - (currentPrice - low) * priceScale}
            r="6"
            fill="#22c55e"
            stroke="white"
            strokeWidth="3"
            className="drop-shadow-sm"
          />
          
          {/* Hover indicator */}
          {hoveredPoint && (
            <>
              <line
                x1={padding + (chartData.indexOf(hoveredPoint) / (chartData.length - 1)) * (chartWidth - 2 * padding)}
                y1={padding}
                x2={padding + (chartData.indexOf(hoveredPoint) / (chartData.length - 1)) * (chartWidth - 2 * padding)}
                y2={chartHeight - padding}
                stroke="#6b7280"
                strokeWidth="1"
                strokeDasharray="4,4"
              />
              <circle
                cx={padding + (chartData.indexOf(hoveredPoint) / (chartData.length - 1)) * (chartWidth - 2 * padding)}
                cy={chartHeight - padding - (hoveredPoint.close - low) * priceScale}
                r="4"
                fill="#2563eb"
                stroke="white"
                strokeWidth="2"
              />
            </>
          )}
        </svg>
        
        {/* Hover Tooltip */}
        {hoveredPoint && (
          <div 
            className="absolute z-10 bg-gray-900 text-white p-3 rounded-lg shadow-lg text-sm pointer-events-none"
            style={{
              left: mousePosition.x - 100,
              top: mousePosition.y - 120,
            }}
          >
            <div className="font-semibold">{new Date(hoveredPoint.timestamp).toLocaleDateString()}</div>
            <div>Open: ${hoveredPoint.open.toFixed(2)}</div>
            <div>High: ${hoveredPoint.high.toFixed(2)}</div>
            <div>Low: ${hoveredPoint.low.toFixed(2)}</div>
            <div className="font-semibold">Close: ${hoveredPoint.close.toFixed(2)}</div>
            <div>Volume: {hoveredPoint.volume.toLocaleString()}</div>
          </div>
        )}
      </div>
      
      {/* Chart Statistics */}
      <div className="mt-6 grid grid-cols-2 md:grid-cols-5 gap-4 text-sm">
        <div className="bg-gray-50 p-3 rounded-lg">
          <div className="text-gray-500 text-xs">Current</div>
          <div className="font-bold text-lg">${currentPrice.toFixed(2)}</div>
        </div>
        <div className="bg-gray-50 p-3 rounded-lg">
          <div className="text-gray-500 text-xs">Change</div>
          <div className={`font-bold ${priceChange >= 0 ? 'text-green-600' : 'text-red-600'}`}>
            {priceChange >= 0 ? '+' : ''}${priceChange.toFixed(2)}
            <div className="text-xs">({priceChangePercent.toFixed(2)}%)</div>
          </div>
        </div>
        <div className="bg-gray-50 p-3 rounded-lg">
          <div className="text-gray-500 text-xs">High</div>
          <div className="font-bold text-green-600">${high.toFixed(2)}</div>
        </div>
        <div className="bg-gray-50 p-3 rounded-lg">
          <div className="text-gray-500 text-xs">Low</div>
          <div className="font-bold text-red-600">${low.toFixed(2)}</div>
        </div>
        <div className="bg-gray-50 p-3 rounded-lg">
          <div className="text-gray-500 text-xs">Avg Volume</div>
          <div className="font-bold">
            {(chartData.reduce((sum, p) => sum + p.volume, 0) / chartData.length / 1000000).toFixed(1)}M
          </div>
        </div>
      </div>
      
      {/* Period Info */}
      <div className="mt-4 flex justify-between items-center text-xs text-gray-500">
        <span>Period: {selectedPeriod} • {chartData.length} data points</span>
        <span>Real-time data from Polygon.io</span>
      </div>
      
      {loading && (
        <div className="absolute inset-0 bg-white bg-opacity-75 flex items-center justify-center">
          <div className="flex items-center space-x-2">
            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary-600"></div>
            <span className="text-gray-600">Loading {selectedPeriod} data...</span>
          </div>
        </div>
      )}
    </div>
  );
}
