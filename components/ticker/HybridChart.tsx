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
  const rawHigh = Math.max(...prices);
  const rawLow = Math.min(...prices);
  const maxVolume = Math.max(...volumes);

  // Calculate nice Y-axis ticks (round numbers like 250, 300, 350...)
  const calculateNiceTicks = (min: number, max: number, tickCount: number = 5) => {
    const range = max - min;
    const roughStep = range / (tickCount - 1);

    // Find a nice round step size
    const magnitude = Math.pow(10, Math.floor(Math.log10(roughStep)));
    const residual = roughStep / magnitude;
    let niceStep: number;
    if (residual <= 1.5) niceStep = magnitude;
    else if (residual <= 3) niceStep = 2 * magnitude;
    else if (residual <= 7) niceStep = 5 * magnitude;
    else niceStep = 10 * magnitude;

    // Calculate nice min and max
    const niceMin = Math.floor(min / niceStep) * niceStep;
    const niceMax = Math.ceil(max / niceStep) * niceStep;

    // Generate ticks
    const ticks: number[] = [];
    for (let tick = niceMin; tick <= niceMax; tick += niceStep) {
      ticks.push(tick);
    }
    return { ticks, min: niceMin, max: niceMax };
  };

  const { ticks: yTicks, min: low, max: high } = calculateNiceTicks(rawLow, rawHigh);
  const priceRange = high - low || 1;

  // Use real current price, not chart data
  const firstPrice = prices[0];
  const priceChange = currentPrice - firstPrice;
  const priceChangePercent = firstPrice > 0 ? (priceChange / firstPrice) * 100 : 0;

  // Calculate moving averages
  const ma50 = useMemo(() => calculateSMA(prices, 50), [prices]);
  const ma200 = useMemo(() => calculateSMA(prices, 200), [prices]);

  // Chart dimensions - adjusted for volume section and axis labels
  const totalChartHeight = showVolume ? 420 : 340;
  const priceChartHeight = showVolume ? 280 : 300;
  const volumeChartHeight = 80;
  const xAxisHeight = 30;
  const chartWidth = 900;
  const paddingLeft = 60;  // More space for Y-axis labels
  const paddingRight = 20;
  const paddingTop = 20;
  const paddingBottom = 10;
  const plotWidth = chartWidth - paddingLeft - paddingRight;
  const plotHeight = priceChartHeight - paddingTop - paddingBottom;
  const priceScale = plotHeight / priceRange;
  const volumeScale = volumeChartHeight / maxVolume;

  // Generate X-axis date labels
  const getDateLabels = () => {
    if (dataPoints.length < 2) return [];
    const period = initialData.period || '1Y';
    let labelCount = 6;

    const labels: { date: Date; x: number; label: string }[] = [];
    const step = Math.floor(dataPoints.length / (labelCount - 1));

    for (let i = 0; i < labelCount; i++) {
      const index = Math.min(i * step, dataPoints.length - 1);
      const point = dataPoints[index];
      const date = new Date(point.timestamp);
      const x = paddingLeft + (index / (dataPoints.length - 1)) * plotWidth;

      // Format based on period
      let label: string;
      if (period === '1D' || period === '5D') {
        label = date.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
      } else if (period === '1M' || period === '3M') {
        label = date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
      } else {
        label = date.toLocaleDateString('en-US', { month: 'short', year: 'numeric' });
      }
      labels.push({ date, x, label });
    }
    return labels;
  };

  const dateLabels = getDateLabels();

  // Generate SVG paths for price line
  const pathData = dataPoints.map((point, index) => {
    const x = paddingLeft + (index / (dataPoints.length - 1)) * plotWidth;
    const y = paddingTop + plotHeight - (parseFloat(point.close) - low) * priceScale;
    return `${index === 0 ? 'M' : 'L'} ${x} ${y}`;
  }).join(' ');

  // Generate SVG path for 50-day MA
  const ma50PathData = ma50.map((value, index) => {
    if (value === null) return '';
    const x = paddingLeft + (index / (dataPoints.length - 1)) * plotWidth;
    const y = paddingTop + plotHeight - (value - low) * priceScale;
    // Find the first non-null index to start the path
    const isFirst = ma50.slice(0, index).every(v => v === null);
    return `${isFirst ? 'M' : 'L'} ${x} ${y}`;
  }).filter(Boolean).join(' ');

  // Generate SVG path for 200-day MA
  const ma200PathData = ma200.map((value, index) => {
    if (value === null) return '';
    const x = paddingLeft + (index / (dataPoints.length - 1)) * plotWidth;
    const y = paddingTop + plotHeight - (value - low) * priceScale;
    const isFirst = ma200.slice(0, index).every(v => v === null);
    return `${isFirst ? 'M' : 'L'} ${x} ${y}`;
  }).filter(Boolean).join(' ');

  // Volume bar data
  const volumeBars = dataPoints.map((point, index) => {
    const x = paddingLeft + (index / (dataPoints.length - 1)) * plotWidth;
    const barWidth = Math.max(1, plotWidth / dataPoints.length - 1);
    const barHeight = point.volume * volumeScale;
    const y = priceChartHeight + xAxisHeight + volumeChartHeight - barHeight;
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
      const scaleX = chartWidth / rect.width;

      // Calculate which data point we're hovering over
      const dataIndex = Math.round(((x * scaleX - paddingLeft) / plotWidth) * (dataPoints.length - 1));

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
  }, [dataPoints, chartWidth, symbol, paddingLeft, plotWidth]);

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
      <div className={`relative bg-white border rounded-lg overflow-hidden`} style={{ height: showVolume ? '420px' : '340px' }}>
        <svg
          id="price-chart"
          width={chartWidth}
          height={totalChartHeight}
          className="w-full h-full cursor-crosshair"
          data-symbol={symbol}
          viewBox={`0 0 ${chartWidth} ${totalChartHeight}`}
          preserveAspectRatio="xMidYMid meet"
        >
          {/* Gradient definition */}
          <defs>
            <linearGradient id="priceGradient" x1="0%" y1="0%" x2="0%" y2="100%">
              <stop offset="0%" style={{stopColor: '#10b981', stopOpacity: 0.2}} />
              <stop offset="100%" style={{stopColor: '#10b981', stopOpacity: 0.02}} />
            </linearGradient>
          </defs>

          {/* Background for plot area */}
          <rect x={paddingLeft} y={paddingTop} width={plotWidth} height={plotHeight} fill="#fafafa" />

          {/* Y-axis grid lines and labels */}
          {yTicks.map((tick, i) => {
            const y = paddingTop + plotHeight - (tick - low) * priceScale;
            return (
              <g key={`y-${i}`}>
                <line
                  x1={paddingLeft}
                  y1={y}
                  x2={chartWidth - paddingRight}
                  y2={y}
                  stroke="#e5e7eb"
                  strokeWidth="1"
                />
                <text
                  x={paddingLeft - 8}
                  y={y + 4}
                  textAnchor="end"
                  className="fill-gray-500"
                  style={{ fontSize: '12px' }}
                >
                  {tick >= 1000 ? `$${(tick / 1000).toFixed(0)}K` : `$${tick.toFixed(0)}`}
                </text>
              </g>
            );
          })}

          {/* X-axis date labels */}
          {dateLabels.map((label, i) => (
            <text
              key={`x-${i}`}
              x={label.x}
              y={priceChartHeight + 18}
              textAnchor="middle"
              className="fill-gray-500"
              style={{ fontSize: '11px' }}
            >
              {label.label}
            </text>
          ))}

          {/* Price area with gradient */}
          <path
            d={`${pathData} L ${chartWidth - paddingRight} ${paddingTop + plotHeight} L ${paddingLeft} ${paddingTop + plotHeight} Z`}
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

          {/* Price line */}
          <path
            d={pathData}
            fill="none"
            stroke="#10b981"
            strokeWidth="2.5"
            strokeLinecap="round"
            strokeLinejoin="round"
          />

          {/* Current price indicator dot */}
          <circle
            cx={chartWidth - paddingRight}
            cy={paddingTop + plotHeight - (currentPrice - low) * priceScale}
            r="6"
            fill="#10b981"
            stroke="white"
            strokeWidth="3"
          />

          {/* Volume Bars Section */}
          {showVolume && (
            <g>
              {/* Volume divider line */}
              <line
                x1={paddingLeft}
                y1={priceChartHeight + xAxisHeight}
                x2={chartWidth - paddingRight}
                y2={priceChartHeight + xAxisHeight}
                stroke="#e5e7eb"
                strokeWidth="1"
              />

              {/* Volume label */}
              <text x={paddingLeft - 8} y={priceChartHeight + xAxisHeight + 15} textAnchor="end" className="fill-gray-400" style={{ fontSize: '10px' }}>Vol</text>

              {/* Volume bars */}
              {volumeBars.map((bar, index) => (
                <rect
                  key={index}
                  x={bar.x}
                  y={bar.y}
                  width={bar.width}
                  height={bar.height}
                  fill={bar.isUp ? 'rgba(16, 185, 129, 0.5)' : 'rgba(239, 68, 68, 0.5)'}
                />
              ))}
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
