'use client';

import { useEffect, useState, useMemo } from 'react';

interface HybridChartProps {
  symbol: string;
  initialData: any;
  currentPrice: number;
}

interface ChartDataPoint {
  timestamp: string;
  open: string;
  high: string;
  low: string;
  close: string;
  volume: number;
}

// Calculate Simple Moving Average
function calculateSMA(data: number[], period: number): (number | null)[] {
  const sma: (number | null)[] = [];
  for (let i = 0; i < data.length; i++) {
    if (i < period - 1) {
      sma.push(null);
    } else {
      const sum = data.slice(i - period + 1, i + 1).reduce((a, b) => a + b, 0);
      sma.push(sum / period);
    }
  }
  return sma;
}

export default function HybridChart({ symbol, initialData, currentPrice }: HybridChartProps) {
  const [showMA50, setShowMA50] = useState(false);
  const [showMA200, setShowMA200] = useState(false);
  const [showVolume, setShowVolume] = useState(true);
  if (!initialData?.dataPoints || initialData.dataPoints.length === 0) {
    return (
      <div className="p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-6">Price Chart</h3>
        <div className="h-80 bg-gray-100 rounded-lg flex items-center justify-center">
          <div className="text-gray-500">No chart data available</div>
        </div>
      </div>
    );
  }

  const dataPoints: ChartDataPoint[] = initialData.dataPoints;
  const prices = dataPoints.map(d => parseFloat(d.close));
  const volumes = dataPoints.map(d => d.volume);
  const high = Math.max(...prices);
  const low = Math.min(...prices);
  const priceRange = high - low || 1;
  const maxVolume = Math.max(...volumes);

  // Use real current price, not chart data
  const firstPrice = prices[0];
  const priceChange = currentPrice - firstPrice;
  const priceChangePercent = firstPrice > 0 ? (priceChange / firstPrice) * 100 : 0;

  // Calculate moving averages
  const ma50 = useMemo(() => calculateSMA(prices, 50), [prices]);
  const ma200 = useMemo(() => calculateSMA(prices, 200), [prices]);

  // Chart dimensions - adjusted for volume section
  const totalChartHeight = showVolume ? 380 : 300;
  const priceChartHeight = showVolume ? 260 : 300;
  const volumeChartHeight = 80;
  const chartWidth = 900;
  const padding = 40;
  const priceScale = (priceChartHeight - 2 * padding) / priceRange;
  const volumeScale = volumeChartHeight / maxVolume;

  // Generate SVG paths for price line
  const pathData = dataPoints.map((point, index) => {
    const x = padding + (index / (dataPoints.length - 1)) * (chartWidth - 2 * padding);
    const y = priceChartHeight - padding - (parseFloat(point.close) - low) * priceScale;
    return `${index === 0 ? 'M' : 'L'} ${x} ${y}`;
  }).join(' ');

  // Generate SVG path for 50-day MA
  const ma50PathData = ma50.map((value, index) => {
    if (value === null) return '';
    const x = padding + (index / (dataPoints.length - 1)) * (chartWidth - 2 * padding);
    const y = priceChartHeight - padding - (value - low) * priceScale;
    // Find the first non-null index to start the path
    const isFirst = ma50.slice(0, index).every(v => v === null);
    return `${isFirst ? 'M' : 'L'} ${x} ${y}`;
  }).filter(Boolean).join(' ');

  // Generate SVG path for 200-day MA
  const ma200PathData = ma200.map((value, index) => {
    if (value === null) return '';
    const x = padding + (index / (dataPoints.length - 1)) * (chartWidth - 2 * padding);
    const y = priceChartHeight - padding - (value - low) * priceScale;
    const isFirst = ma200.slice(0, index).every(v => v === null);
    return `${isFirst ? 'M' : 'L'} ${x} ${y}`;
  }).filter(Boolean).join(' ');

  // Volume bar data
  const volumeBars = dataPoints.map((point, index) => {
    const x = padding + (index / (dataPoints.length - 1)) * (chartWidth - 2 * padding);
    const barWidth = Math.max(1, (chartWidth - 2 * padding) / dataPoints.length - 1);
    const barHeight = point.volume * volumeScale;
    const y = priceChartHeight + volumeChartHeight - barHeight;
    const isUp = parseFloat(point.close) >= parseFloat(point.open);
    return { x: x - barWidth / 2, y, width: barWidth, height: barHeight, isUp };
  });

  // Timeframe buttons with JavaScript for interactivity
  const timeframes = ['1D', '5D', '1M', '3M', '6M', '1Y', '5Y'];

  // Initialize chart interactivity
  useEffect(() => {
    const chart = document.getElementById('price-chart');
    const tooltip = document.getElementById('chart-tooltip');
    const buttons = document.querySelectorAll('.timeframe-btn');

    if (!chart || !tooltip) return;

    // Add hover functionality for tooltip
    const handleMouseMove = (e: MouseEvent) => {
      const rect = chart.getBoundingClientRect();
      const x = e.clientX - rect.left;

      // Calculate which data point we're hovering over
      const dataIndex = Math.round(((x - 40) / (chartWidth - 80)) * (dataPoints.length - 1));

      if (dataIndex >= 0 && dataIndex < dataPoints.length) {
        const point = dataPoints[dataIndex];
        const date = new Date(point.timestamp);

        // Update tooltip content
        const tooltipDate = document.getElementById('tooltip-date');
        const tooltipOhlc = document.getElementById('tooltip-ohlc');

        if (tooltipDate) {
          tooltipDate.textContent = date.toLocaleDateString();
        }

        if (tooltipOhlc) {
          tooltipOhlc.innerHTML =
            'Open: $' + parseFloat(point.open).toFixed(2) + '<br>' +
            'High: $' + parseFloat(point.high).toFixed(2) + '<br>' +
            'Low: $' + parseFloat(point.low).toFixed(2) + '<br>' +
            '<strong>Close: $' + parseFloat(point.close).toFixed(2) + '</strong><br>' +
            'Volume: ' + point.volume.toLocaleString();
        }

        // Position tooltip near cursor
        const tooltipWidth = 160;
        const tooltipHeight = 100;
        const offset = 10;

        let left = e.clientX + offset;
        let top = e.clientY - offset;

        // Keep tooltip within viewport
        if (left + tooltipWidth > window.innerWidth - 10) {
          left = e.clientX - tooltipWidth - offset;
        }

        if (top < 10) {
          top = e.clientY + offset;
        }

        if (top + tooltipHeight > window.innerHeight - 10) {
          top = e.clientY - tooltipHeight - offset;
        }

        tooltip.style.left = left + 'px';
        tooltip.style.top = top + 'px';
        tooltip.style.opacity = '1';
      }
    };

    const handleMouseLeave = () => {
      tooltip.style.opacity = '0';
    };

    chart.addEventListener('mousemove', handleMouseMove);
    chart.addEventListener('mouseleave', handleMouseLeave);

    // Add timeframe button functionality
    buttons.forEach(button => {
      const handleClick = async function(this: HTMLElement) {
        const period = this.dataset.period;
        const btnSymbol = this.dataset.symbol;

        // Update button states
        buttons.forEach(btn => {
          btn.classList.remove('bg-white', 'text-primary-600', 'shadow-sm');
          btn.classList.add('text-gray-600');
        });
        this.classList.add('bg-white', 'text-primary-600', 'shadow-sm');
        this.classList.remove('text-gray-600');

        // Show loading
        const originalText = this.textContent;
        this.textContent = '...';

        try {
          // Fetch new data
          const response = await fetch(`/api/v1/tickers/${btnSymbol}/chart?period=${period}`);
          const result = await response.json();

          if (result.data?.dataPoints) {
            // Reload the page with new period
            window.location.search = `?period=${period}`;
          }
        } catch (error) {
          console.error('Failed to fetch chart data:', error);
          this.textContent = originalText || period || '';
        }
      };

      button.addEventListener('click', handleClick);
    });

    // Cleanup event listeners on unmount
    return () => {
      chart.removeEventListener('mousemove', handleMouseMove);
      chart.removeEventListener('mouseleave', handleMouseLeave);
    };
  }, [dataPoints, chartWidth, symbol]);

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-gray-900">Interactive Price Chart</h3>
        <div className="flex space-x-1 bg-gray-100 rounded-lg p-1" id="timeframe-buttons">
          {timeframes.map((period) => (
            <button
              key={period}
              className={`timeframe-btn px-3 py-1 text-sm font-medium rounded-md transition-colors ${
                period === initialData.period
                  ? 'bg-white text-primary-600 shadow-sm'
                  : 'text-gray-600 hover:text-gray-900 hover:bg-white'
              }`}
              data-period={period}
              data-symbol={symbol}
            >
              {period}
            </button>
          ))}
        </div>
      </div>

      {/* Chart Options */}
      <div className="flex flex-wrap gap-4 mb-4">
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={showVolume}
            onChange={(e) => setShowVolume(e.target.checked)}
            className="w-4 h-4 text-primary-600 rounded border-gray-300 focus:ring-primary-500"
          />
          <span className="text-sm text-gray-600">Volume Bars</span>
        </label>
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={showMA50}
            onChange={(e) => setShowMA50(e.target.checked)}
            className="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
          />
          <span className="text-sm text-gray-600">50-Day MA</span>
          {showMA50 && <span className="w-4 h-0.5 bg-blue-500"></span>}
        </label>
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={showMA200}
            onChange={(e) => setShowMA200(e.target.checked)}
            className="w-4 h-4 text-orange-600 rounded border-gray-300 focus:ring-orange-500"
          />
          <span className="text-sm text-gray-600">200-Day MA</span>
          {showMA200 && <span className="w-4 h-0.5 bg-orange-500"></span>}
        </label>
      </div>

      {/* Enhanced Interactive SVG Chart */}
      <div className={`relative bg-white border rounded-lg overflow-hidden`} style={{ height: showVolume ? '380px' : '300px' }}>
        <svg
          id="price-chart"
          width={chartWidth}
          height={totalChartHeight}
          className="w-full h-full cursor-crosshair"
          data-symbol={symbol}
          data-chart-data={JSON.stringify(dataPoints)}
        >
          {/* Enhanced Grid */}
          <defs>
            <pattern id="chartGrid" width="60" height="40" patternUnits="userSpaceOnUse">
              <path d="M 60 0 L 0 0 0 40" fill="none" stroke="#f1f5f9" strokeWidth="1"/>
            </pattern>
            <linearGradient id="priceGradient" x1="0%" y1="0%" x2="0%" y2="100%">
              <stop offset="0%" style={{stopColor: '#10b981', stopOpacity: 0.3}} />
              <stop offset="100%" style={{stopColor: '#10b981', stopOpacity: 0.05}} />
            </linearGradient>
          </defs>
          <rect width="100%" height={priceChartHeight} fill="url(#chartGrid)" />

          {/* Price area with gradient */}
          <path
            d={`${pathData} L ${chartWidth - padding} ${priceChartHeight - padding} L ${padding} ${priceChartHeight - padding} Z`}
            fill="url(#priceGradient)"
            stroke="none"
          />

          {/* 200-Day Moving Average Line */}
          {showMA200 && ma200PathData && (
            <path
              d={ma200PathData}
              fill="none"
              stroke="#f97316"
              strokeWidth="2"
              strokeDasharray="4,4"
              opacity="0.8"
            />
          )}

          {/* 50-Day Moving Average Line */}
          {showMA50 && ma50PathData && (
            <path
              d={ma50PathData}
              fill="none"
              stroke="#3b82f6"
              strokeWidth="2"
              opacity="0.8"
            />
          )}

          {/* Enhanced price line */}
          <path
            d={pathData}
            fill="none"
            stroke="#10b981"
            strokeWidth="3"
            strokeLinecap="round"
            strokeLinejoin="round"
            className="drop-shadow-sm"
          />

          {/* Y-axis labels */}
          <text x="10" y="30" className="text-sm fill-gray-700 font-semibold">${high.toFixed(2)}</text>
          <text x="10" y={priceChartHeight - 15} className="text-sm fill-gray-700 font-semibold">${low.toFixed(2)}</text>

          {/* Enhanced current price indicator */}
          <circle
            cx={chartWidth - padding}
            cy={priceChartHeight - padding - (currentPrice - low) * priceScale}
            r="8"
            fill="#10b981"
            stroke="white"
            strokeWidth="4"
            className="drop-shadow-lg"
          />

          {/* Current price label */}
          <text
            x={chartWidth - padding - 60}
            y={priceChartHeight - padding - (currentPrice - low) * priceScale + 5}
            className="text-sm fill-white font-bold"
            textAnchor="end"
          >
            ${currentPrice.toFixed(2)}
          </text>

          {/* Volume Bars Section */}
          {showVolume && (
            <g>
              {/* Volume divider line */}
              <line
                x1={padding}
                y1={priceChartHeight}
                x2={chartWidth - padding}
                y2={priceChartHeight}
                stroke="#e5e7eb"
                strokeWidth="1"
              />

              {/* Volume bars */}
              {volumeBars.map((bar, index) => (
                <rect
                  key={index}
                  x={bar.x}
                  y={bar.y}
                  width={bar.width}
                  height={bar.height}
                  fill={bar.isUp ? 'rgba(16, 185, 129, 0.4)' : 'rgba(239, 68, 68, 0.4)'}
                  stroke={bar.isUp ? '#10b981' : '#ef4444'}
                  strokeWidth="0.5"
                />
              ))}

              {/* Volume label */}
              <text x="10" y={priceChartHeight + 15} className="text-xs fill-gray-400">Volume</text>
            </g>
          )}
        </svg>

        {/* Hover tooltip placeholder */}
        <div
          id="chart-tooltip"
          className="fixed z-30 bg-gray-900 text-white p-2 rounded shadow-xl text-xs pointer-events-none opacity-0 transition-opacity duration-100 border border-gray-700"
          style={{ left: -200, top: -200, minWidth: '140px', fontFamily: 'monospace' }}
        >
          <div id="tooltip-date" className="font-semibold mb-1 text-gray-300 text-xs"></div>
          <div id="tooltip-ohlc" className="leading-tight"></div>
        </div>
      </div>
      
      {/* Enhanced Statistics Grid */}
      <div className="mt-6 grid grid-cols-2 md:grid-cols-5 gap-4">
        <div className="bg-gradient-to-br from-blue-50 to-blue-100 p-4 rounded-xl border border-blue-200">
          <div className="text-blue-600 text-xs font-semibold uppercase tracking-wide">Current</div>
          <div className="font-bold text-xl text-blue-900">${currentPrice.toFixed(2)}</div>
        </div>
        <div className={`bg-gradient-to-br p-4 rounded-xl border ${
          priceChange >= 0 
            ? 'from-green-50 to-green-100 border-green-200' 
            : 'from-red-50 to-red-100 border-red-200'
        }`}>
          <div className={`text-xs font-semibold uppercase tracking-wide ${
            priceChange >= 0 ? 'text-green-600' : 'text-red-600'
          }`}>
            Change
          </div>
          <div className={`font-bold text-lg ${
            priceChange >= 0 ? 'text-green-900' : 'text-red-900'
          }`}>
            {priceChange >= 0 ? '+' : ''}${priceChange.toFixed(2)}
            <div className="text-sm">({priceChangePercent.toFixed(2)}%)</div>
          </div>
        </div>
        <div className="bg-gradient-to-br from-green-50 to-green-100 p-4 rounded-xl border border-green-200">
          <div className="text-green-600 text-xs font-semibold uppercase tracking-wide">High</div>
          <div className="font-bold text-lg text-green-900">${high.toFixed(2)}</div>
        </div>
        <div className="bg-gradient-to-br from-red-50 to-red-100 p-4 rounded-xl border border-red-200">
          <div className="text-red-600 text-xs font-semibold uppercase tracking-wide">Low</div>
          <div className="font-bold text-lg text-red-900">${low.toFixed(2)}</div>
        </div>
        <div className="bg-gradient-to-br from-purple-50 to-purple-100 p-4 rounded-xl border border-purple-200">
          <div className="text-purple-600 text-xs font-semibold uppercase tracking-wide">Volume</div>
          <div className="font-bold text-lg text-purple-900">
            {(dataPoints.reduce((sum, p) => sum + p.volume, 0) / dataPoints.length / 1000000).toFixed(1)}M
          </div>
        </div>
      </div>
      
      {/* Chart Info */}
      <div className="mt-4 flex justify-between items-center text-sm">
        <div className="text-gray-600">
          <span className="font-semibold">{initialData.period}</span> • {dataPoints.length} data points • Real-time from Polygon.io
        </div>
        <div className="text-gray-500">
          Last updated: {new Date().toLocaleTimeString()}
        </div>
      </div>

    </div>
  );
}
