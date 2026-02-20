'use client';

import { useEffect, useState, useMemo, useCallback, useRef } from 'react';
import { ArrowsPointingOutIcon, ArrowsPointingInIcon } from '@heroicons/react/24/outline';
import { useTheme } from '@/lib/contexts/ThemeContext';
import { getChartColors, themeColors } from '@/lib/theme';

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

interface ChartData {
  dataPoints: ChartDataPoint[];
  period: string;
  count: number;
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
  const { resolvedTheme } = useTheme();
  const chartColors = useMemo(() => getChartColors(resolvedTheme), [resolvedTheme]);

  const [showMA50, setShowMA50] = useState(false);
  const [showMA200, setShowMA200] = useState(false);
  const [showVolume, setShowVolume] = useState(true);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [showSP500, setShowSP500] = useState(false);
  const [sp500Data, setSp500Data] = useState<ChartDataPoint[]>([]);
  const [sp500Loading, setSp500Loading] = useState(false);

  // Client-side chart data state with caching
  const [chartData, setChartData] = useState<ChartData>({
    dataPoints: initialData?.dataPoints || [],
    period: initialData?.period || '1Y',
    count: initialData?.count || 0,
  });
  const [selectedPeriod, setSelectedPeriod] = useState(initialData?.period || '1Y');
  const [isLoading, setIsLoading] = useState(false);
  const chartCache = useRef<Map<string, ChartData>>(new Map());

  // Initialize cache with initial data
  useEffect(() => {
    if (initialData?.dataPoints && initialData.period) {
      chartCache.current.set(initialData.period, {
        dataPoints: initialData.dataPoints,
        period: initialData.period,
        count: initialData.count || initialData.dataPoints.length,
      });
    }
  }, [initialData]);

  // Fetch chart data for a period (with caching)
  const fetchChartData = useCallback(
    async (period: string) => {
      // Check cache first
      const cached = chartCache.current.get(period);
      if (cached) {
        setChartData(cached);
        setSelectedPeriod(period);
        return;
      }

      setIsLoading(true);
      try {
        const response = await fetch(`/api/v1/tickers/${symbol}/chart?period=${period}`);
        const result = await response.json();

        if (result.data?.dataPoints) {
          const newData: ChartData = {
            dataPoints: result.data.dataPoints,
            period: period,
            count: result.data.count || result.data.dataPoints.length,
          };
          // Cache the result
          chartCache.current.set(period, newData);
          setChartData(newData);
          setSelectedPeriod(period);
        }
      } catch (error) {
        console.error('Failed to fetch chart data:', error);
      } finally {
        setIsLoading(false);
      }
    },
    [symbol]
  );

  // Fetch S&P 500 data when comparison is enabled
  useEffect(() => {
    if (showSP500 && sp500Data.length === 0 && !sp500Loading) {
      setSp500Loading(true);
      fetch(`/api/v1/tickers/SPY/chart?period=${selectedPeriod}`)
        .then((res) => res.json())
        .then((data) => {
          if (data.data?.dataPoints) {
            setSp500Data(data.data.dataPoints);
          }
        })
        .catch((err) => console.error('Failed to fetch S&P 500 data:', err))
        .finally(() => setSp500Loading(false));
    }
  }, [showSP500, sp500Data.length, sp500Loading, selectedPeriod]);

  // Reset S&P 500 data when period changes
  useEffect(() => {
    setSp500Data([]);
  }, [selectedPeriod]);

  // Handle Escape key to exit fullscreen
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isFullscreen) {
        setIsFullscreen(false);
      }
    };

    if (isFullscreen) {
      document.addEventListener('keydown', handleEscape);
      // Prevent body scroll when fullscreen
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }

    return () => {
      document.removeEventListener('keydown', handleEscape);
      document.body.style.overflow = '';
    };
  }, [isFullscreen]);

  const hasData = chartData?.dataPoints && chartData.dataPoints.length > 0;
  const dataPoints: ChartDataPoint[] = chartData?.dataPoints || [];
  const prices = useMemo(() => dataPoints.map((d) => parseFloat(d.close)), [dataPoints]);
  const volumes = dataPoints.map((d) => d.volume);
  const rawHigh = hasData ? Math.max(...prices) : 0;
  const rawLow = hasData ? Math.min(...prices) : 0;
  const maxVolume = hasData ? Math.max(...volumes) : 0;

  // Calculate nice Y-axis ticks (round numbers like 250, 300, 350...)
  const calculateNiceTicks = (min: number, max: number, tickCount: number = 5) => {
    const range = max - min;
    if (range === 0) return { ticks: [min], min, max: max || 1 };
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
  const firstPrice = prices[0] || 0;
  const priceChange = currentPrice - firstPrice;
  const priceChangePercent = firstPrice > 0 ? (priceChange / firstPrice) * 100 : 0;

  // Calculate moving averages (must be called unconditionally — hooks cannot be after early return)
  const ma50 = useMemo(() => calculateSMA(prices, 50), [prices]);
  const ma200 = useMemo(() => calculateSMA(prices, 200), [prices]);

  // Chart dimensions - adjusted for volume section and axis labels
  // In fullscreen mode, use larger dimensions
  const baseChartHeight = isFullscreen ? 600 : showVolume ? 420 : 340;
  const basePriceChartHeight = isFullscreen ? 480 : showVolume ? 280 : 300;
  const volumeChartHeight = isFullscreen ? 100 : 80;
  const totalChartHeight = showVolume ? baseChartHeight : isFullscreen ? 520 : 340;
  const priceChartHeight = showVolume ? basePriceChartHeight : isFullscreen ? 480 : 300;
  const xAxisHeight = 30;
  const chartWidth = isFullscreen ? 1600 : 900;
  const paddingLeft = 70; // More space for Y-axis labels
  const paddingRight = 30;
  const paddingTop = 20;
  const paddingBottom = 10;
  const plotWidth = chartWidth - paddingLeft - paddingRight;
  const plotHeight = priceChartHeight - paddingTop - paddingBottom;
  const priceScale = plotHeight / priceRange;
  const volumeScale = maxVolume > 0 ? volumeChartHeight / maxVolume : 0;

  // Generate X-axis date labels
  const getDateLabels = () => {
    if (dataPoints.length < 2) return [];
    const period = selectedPeriod;
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
  const pathData = dataPoints
    .map((point, index) => {
      const x = paddingLeft + (index / (dataPoints.length - 1)) * plotWidth;
      const y = paddingTop + plotHeight - (parseFloat(point.close) - low) * priceScale;
      return `${index === 0 ? 'M' : 'L'} ${x} ${y}`;
    })
    .join(' ');

  // Generate SVG path for 50-day MA
  const ma50PathData = ma50
    .map((value, index) => {
      if (value === null) return '';
      const x = paddingLeft + (index / (dataPoints.length - 1)) * plotWidth;
      const y = paddingTop + plotHeight - (value - low) * priceScale;
      // Find the first non-null index to start the path
      const isFirst = ma50.slice(0, index).every((v) => v === null);
      return `${isFirst ? 'M' : 'L'} ${x} ${y}`;
    })
    .filter(Boolean)
    .join(' ');

  // Generate SVG path for 200-day MA
  const ma200PathData = ma200
    .map((value, index) => {
      if (value === null) return '';
      const x = paddingLeft + (index / (dataPoints.length - 1)) * plotWidth;
      const y = paddingTop + plotHeight - (value - low) * priceScale;
      const isFirst = ma200.slice(0, index).every((v) => v === null);
      return `${isFirst ? 'M' : 'L'} ${x} ${y}`;
    })
    .filter(Boolean)
    .join(' ');

  // Generate SVG path for S&P 500 comparison (normalized to start at same point as stock)
  const sp500PathData = useMemo(() => {
    if (!showSP500 || sp500Data.length === 0) return '';

    // Normalize S&P 500 to stock's starting price and scale
    const stockStartPrice = prices[0];
    const sp500StartPrice = parseFloat(sp500Data[0]?.close || '0');
    if (sp500StartPrice === 0) return '';

    // Calculate the scaling factor: sp500 normalized price = stockStartPrice * (sp500Price / sp500StartPrice)
    const sp500Prices = sp500Data.map(
      (d) => stockStartPrice * (parseFloat(d.close) / sp500StartPrice)
    );

    return sp500Prices
      .map((price, index) => {
        // Map S&P 500 index to stock data index (in case they have different lengths)
        const x = paddingLeft + (index / (sp500Prices.length - 1)) * plotWidth;
        // Clamp price to visible range
        const clampedPrice = Math.max(low, Math.min(high, price));
        const y = paddingTop + plotHeight - (clampedPrice - low) * priceScale;
        return `${index === 0 ? 'M' : 'L'} ${x} ${y}`;
      })
      .join(' ');
  }, [
    showSP500,
    sp500Data,
    prices,
    paddingLeft,
    plotWidth,
    paddingTop,
    plotHeight,
    low,
    high,
    priceScale,
  ]);

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
  const timeframes = ['1D', '5D', '1M', '3M', '6M', 'YTD', '1Y', '5Y', 'MAX'];

  // Initialize chart tooltip interactivity
  useEffect(() => {
    const chart = document.getElementById('price-chart');
    const tooltip = document.getElementById('chart-tooltip');

    if (!chart || !tooltip) return;

    // Add hover functionality for tooltip
    const handleMouseMove = (e: MouseEvent) => {
      const rect = chart.getBoundingClientRect();
      const x = e.clientX - rect.left;
      const scaleX = chartWidth / rect.width;

      // Calculate which data point we're hovering over
      const dataIndex = Math.round(
        ((x * scaleX - paddingLeft) / plotWidth) * (dataPoints.length - 1)
      );

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
            'Open: $' +
            parseFloat(point.open).toFixed(2) +
            '<br>' +
            'High: $' +
            parseFloat(point.high).toFixed(2) +
            '<br>' +
            'Low: $' +
            parseFloat(point.low).toFixed(2) +
            '<br>' +
            '<strong>Close: $' +
            parseFloat(point.close).toFixed(2) +
            '</strong><br>' +
            'Volume: ' +
            point.volume.toLocaleString();
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

    // Cleanup event listeners on unmount
    return () => {
      chart.removeEventListener('mousemove', handleMouseMove);
      chart.removeEventListener('mouseleave', handleMouseLeave);
    };
  }, [dataPoints, chartWidth, paddingLeft, plotWidth]);

  // Early return for empty data — placed after ALL hooks to satisfy rules-of-hooks
  if (!hasData) {
    return (
      <div className="p-6">
        <h3 className="text-lg font-semibold text-ic-text-primary mb-6">Price Chart</h3>
        <div className="h-80 bg-ic-bg-secondary rounded-lg flex items-center justify-center">
          <div className="text-ic-text-muted">
            {isLoading ? 'Loading chart data...' : 'No chart data available'}
          </div>
        </div>
      </div>
    );
  }

  const chartContent = (
    <>
      <div className="flex items-center justify-between mb-4">
        <h3
          className={`font-semibold text-ic-text-primary ${isFullscreen ? 'text-xl' : 'text-lg'}`}
        >
          {isFullscreen && <span className="text-primary-600">{symbol}</span>}
          {isFullscreen ? ' - ' : ''}Interactive Price Chart
        </h3>
        <div className="flex items-center gap-3">
          <div className="flex space-x-1 bg-ic-bg-secondary rounded-lg p-1" id="timeframe-buttons">
            {timeframes.map((period) => (
              <button
                key={period}
                onClick={() => fetchChartData(period)}
                disabled={isLoading}
                className={`px-3 py-1 text-sm font-medium rounded-md transition-colors ${
                  period === selectedPeriod
                    ? 'bg-ic-surface text-primary-600 shadow-sm'
                    : 'text-ic-text-secondary hover:text-ic-text-primary hover:bg-ic-surface-hover'
                } ${isLoading ? 'opacity-50 cursor-not-allowed' : ''}`}
              >
                {isLoading && period === selectedPeriod ? '...' : period}
              </button>
            ))}
          </div>
          <button
            onClick={() => setIsFullscreen(!isFullscreen)}
            className="p-2 text-ic-text-muted hover:text-ic-text-secondary hover:bg-ic-surface-hover rounded-lg transition-colors"
            title={isFullscreen ? 'Exit fullscreen (Esc)' : 'Fullscreen'}
          >
            {isFullscreen ? (
              <ArrowsPointingInIcon className="h-5 w-5" />
            ) : (
              <ArrowsPointingOutIcon className="h-5 w-5" />
            )}
          </button>
        </div>
      </div>

      {/* Chart Options */}
      <div className="flex flex-wrap gap-4 mb-4">
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={showVolume}
            onChange={(e) => setShowVolume(e.target.checked)}
            className="w-4 h-4 text-primary-600 rounded border-ic-border focus:ring-primary-500"
          />
          <span className="text-sm text-ic-text-secondary">Volume Bars</span>
        </label>
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={showMA50}
            onChange={(e) => setShowMA50(e.target.checked)}
            className="w-4 h-4 text-blue-600 rounded border-ic-border focus:ring-blue-500"
          />
          <span className="text-sm text-ic-text-secondary">50-Day MA</span>
          {showMA50 && <span className="w-4 h-0.5 bg-blue-500"></span>}
        </label>
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={showMA200}
            onChange={(e) => setShowMA200(e.target.checked)}
            className="w-4 h-4 text-orange-600 rounded border-ic-border focus:ring-orange-500"
          />
          <span className="text-sm text-ic-text-secondary">200-Day MA</span>
          {showMA200 && <span className="w-4 h-0.5 bg-orange-500"></span>}
        </label>
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={showSP500}
            onChange={(e) => setShowSP500(e.target.checked)}
            className="w-4 h-4 text-red-600 rounded border-ic-border focus:ring-red-500"
          />
          <span className="text-sm text-ic-text-secondary">Compare S&P 500</span>
          {showSP500 && <span className="w-4 h-0.5 bg-red-500"></span>}
          {sp500Loading && <span className="text-xs text-ic-text-dim">(loading...)</span>}
        </label>
      </div>

      {/* Enhanced Interactive SVG Chart */}
      <div
        className={`relative bg-ic-surface border rounded-lg overflow-hidden`}
        style={{ height: `${totalChartHeight}px` }}
      >
        {/* Loading overlay */}
        {isLoading && (
          <div className="absolute inset-0 bg-ic-bg-primary/50 flex items-center justify-center z-10">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
          </div>
        )}
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
            <linearGradient id={`priceGradient-${resolvedTheme}`} x1="0%" y1="0%" x2="0%" y2="100%">
              <stop offset="0%" style={{ stopColor: chartColors.positive, stopOpacity: 0.2 }} />
              <stop offset="100%" style={{ stopColor: chartColors.positive, stopOpacity: 0.02 }} />
            </linearGradient>
          </defs>

          {/* Background for plot area */}
          <rect
            x={paddingLeft}
            y={paddingTop}
            width={plotWidth}
            height={plotHeight}
            fill="var(--ic-bg-secondary)"
          />

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
                  stroke="var(--ic-border)"
                  strokeWidth="1"
                />
                <text
                  x={paddingLeft - 8}
                  y={y + 4}
                  textAnchor="end"
                  className="fill-ic-text-muted"
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
              className="fill-ic-text-muted"
              style={{ fontSize: '11px' }}
            >
              {label.label}
            </text>
          ))}

          {/* Price area with gradient */}
          <path
            d={`${pathData} L ${chartWidth - paddingRight} ${paddingTop + plotHeight} L ${paddingLeft} ${paddingTop + plotHeight} Z`}
            fill={`url(#priceGradient-${resolvedTheme})`}
            stroke="none"
          />

          {/* 200-Day Moving Average Line */}
          {showMA200 && ma200PathData && (
            <path
              d={ma200PathData}
              fill="none"
              stroke={themeColors.accent.orange}
              strokeWidth="2"
              strokeDasharray="4,4"
              opacity="0.8"
            />
          )}

          {/* S&P 500 Comparison Line */}
          {showSP500 && sp500PathData && (
            <path
              d={sp500PathData}
              fill="none"
              stroke={chartColors.negative}
              strokeWidth="2"
              strokeDasharray="6,3"
              opacity="0.8"
            />
          )}

          {/* 50-Day Moving Average Line */}
          {showMA50 && ma50PathData && (
            <path
              d={ma50PathData}
              fill="none"
              stroke={chartColors.line}
              strokeWidth="2"
              opacity="0.8"
            />
          )}

          {/* Price line */}
          <path
            d={pathData}
            fill="none"
            stroke={chartColors.positive}
            strokeWidth="2.5"
            strokeLinecap="round"
            strokeLinejoin="round"
          />

          {/* Current price indicator dot */}
          <circle
            cx={chartWidth - paddingRight}
            cy={paddingTop + plotHeight - (currentPrice - low) * priceScale}
            r="6"
            fill={chartColors.positive}
            stroke={resolvedTheme === 'dark' ? '#fff' : '#000'}
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
                stroke="var(--ic-border)"
                strokeWidth="1"
              />

              {/* Volume label */}
              <text
                x={paddingLeft - 8}
                y={priceChartHeight + xAxisHeight + 15}
                textAnchor="end"
                className="fill-ic-text-dim"
                style={{ fontSize: '10px' }}
              >
                Vol
              </text>

              {/* Volume bars */}
              {volumeBars.map((bar, index) => (
                <rect
                  key={index}
                  x={bar.x}
                  y={bar.y}
                  width={bar.width}
                  height={bar.height}
                  fill={bar.isUp ? `${chartColors.positive}80` : `${chartColors.negative}80`}
                />
              ))}
            </g>
          )}
        </svg>

        {/* Hover tooltip placeholder */}
        <div
          id="chart-tooltip"
          className="fixed z-30 bg-ic-bg-tertiary text-ic-text-primary p-2 rounded shadow-xl text-xs pointer-events-none opacity-0 transition-opacity duration-100 border border-ic-border"
          style={{ left: -200, top: -200, minWidth: '140px', fontFamily: 'monospace' }}
        >
          <div
            id="tooltip-date"
            className="font-semibold mb-1 text-ic-text-secondary text-xs"
          ></div>
          <div id="tooltip-ohlc" className="leading-tight"></div>
        </div>
      </div>

      {/* Statistics - Compact in fullscreen, Grid in normal view */}
      {isFullscreen ? (
        <div className="mt-4 flex flex-wrap items-center gap-6 text-sm">
          <div className="flex items-center gap-2">
            <span className="text-ic-text-muted">Current:</span>
            <span className="font-bold text-ic-blue">${currentPrice.toFixed(2)}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-ic-text-muted">Change:</span>
            <span
              className={`font-bold ${priceChange >= 0 ? 'text-ic-positive' : 'text-ic-negative'}`}
            >
              {priceChange >= 0 ? '+' : ''}${priceChange.toFixed(2)} (
              {priceChangePercent.toFixed(2)}%)
            </span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-ic-text-muted">High:</span>
            <span className="font-bold text-ic-positive">${high.toFixed(2)}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-ic-text-muted">Low:</span>
            <span className="font-bold text-ic-negative">${low.toFixed(2)}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-ic-text-muted">Avg Volume:</span>
            <span className="font-bold text-purple-600">
              {(
                dataPoints.reduce((sum, p) => sum + p.volume, 0) /
                dataPoints.length /
                1000000
              ).toFixed(1)}
              M
            </span>
          </div>
        </div>
      ) : (
        <div className="mt-6 grid grid-cols-2 md:grid-cols-5 gap-4">
          <div className="bg-gradient-to-br from-blue-50 to-blue-100 p-4 rounded-xl border border-blue-200">
            <div className="text-blue-600 text-xs font-semibold uppercase tracking-wide">
              Current
            </div>
            <div className="font-bold text-xl text-blue-900">${currentPrice.toFixed(2)}</div>
          </div>
          <div
            className={`bg-gradient-to-br p-4 rounded-xl border ${
              priceChange >= 0
                ? 'from-green-50 to-green-100 border-green-200'
                : 'from-red-50 to-red-100 border-red-200'
            }`}
          >
            <div
              className={`text-xs font-semibold uppercase tracking-wide ${
                priceChange >= 0 ? 'text-green-600' : 'text-red-600'
              }`}
            >
              Change
            </div>
            <div
              className={`font-bold text-lg ${
                priceChange >= 0 ? 'text-green-900' : 'text-red-900'
              }`}
            >
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
            <div className="text-purple-600 text-xs font-semibold uppercase tracking-wide">
              Volume
            </div>
            <div className="font-bold text-lg text-purple-900">
              {(
                dataPoints.reduce((sum, p) => sum + p.volume, 0) /
                dataPoints.length /
                1000000
              ).toFixed(1)}
              M
            </div>
          </div>
        </div>
      )}

      {/* Chart Info */}
      <div className="mt-4 flex justify-between items-center text-sm">
        <div className="text-ic-text-secondary">
          <span className="font-semibold">{selectedPeriod}</span> • {dataPoints.length} data points
          • Local database
        </div>
        <div className="text-ic-text-muted">Last updated: {new Date().toLocaleTimeString()}</div>
      </div>
    </>
  );

  // Fullscreen overlay
  if (isFullscreen) {
    return (
      <div className="fixed inset-0 z-50 bg-ic-bg-primary overflow-auto">
        <div className="min-h-screen p-6">{chartContent}</div>
      </div>
    );
  }

  // Normal view
  return <div className="p-6">{chartContent}</div>;
}
