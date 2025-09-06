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

export default function HybridChart({ symbol, initialData, currentPrice }: HybridChartProps) {
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
  const high = Math.max(...prices);
  const low = Math.min(...prices);
  const priceRange = high - low || 1;
  
  // Use real current price, not chart data
  const firstPrice = prices[0];
  const priceChange = currentPrice - firstPrice;
  const priceChangePercent = firstPrice > 0 ? (priceChange / firstPrice) * 100 : 0;

  // Chart dimensions
  const chartHeight = 300;
  const chartWidth = 900;
  const padding = 40;
  const priceScale = (chartHeight - 2 * padding) / priceRange;

  // Generate SVG paths
  const pathData = dataPoints.map((point, index) => {
    const x = padding + (index / (dataPoints.length - 1)) * (chartWidth - 2 * padding);
    const y = chartHeight - padding - (parseFloat(point.close) - low) * priceScale;
    return `${index === 0 ? 'M' : 'L'} ${x} ${y}`;
  }).join(' ');

  // Timeframe buttons with JavaScript for interactivity
  const timeframes = ['1D', '5D', '1M', '3M', '6M', '1Y', '5Y'];

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
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
      
      {/* Enhanced Interactive SVG Chart */}
      <div className="relative h-80 bg-white border rounded-lg overflow-hidden">
        <svg 
          id="price-chart"
          width={chartWidth} 
          height={chartHeight} 
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
          <rect width="100%" height="100%" fill="url(#chartGrid)" />
          
          {/* Price area with gradient */}
          <path
            d={`${pathData} L ${chartWidth - padding} ${chartHeight - padding} L ${padding} ${chartHeight - padding} Z`}
            fill="url(#priceGradient)"
            stroke="none"
          />
          
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
          <text x="10" y={chartHeight - 15} className="text-sm fill-gray-700 font-semibold">${low.toFixed(2)}</text>
          
          {/* Enhanced current price indicator */}
          <circle
            cx={chartWidth - padding}
            cy={chartHeight - padding - (currentPrice - low) * priceScale}
            r="8"
            fill="#10b981"
            stroke="white"
            strokeWidth="4"
            className="drop-shadow-lg"
          />
          
          {/* Current price label */}
          <text
            x={chartWidth - padding - 60}
            y={chartHeight - padding - (currentPrice - low) * priceScale + 5}
            className="text-sm fill-white font-bold"
            textAnchor="end"
          >
            ${currentPrice.toFixed(2)}
          </text>
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

      {/* Client-side JavaScript for interactivity */}
      <script 
        dangerouslySetInnerHTML={{
          __html: `
            (function() {
              // Wait for DOM to be ready
              if (document.readyState === 'loading') {
                document.addEventListener('DOMContentLoaded', initChart);
              } else {
                initChart();
              }
              
              function initChart() {
                const chart = document.getElementById('price-chart');
                const tooltip = document.getElementById('chart-tooltip');
                const buttons = document.querySelectorAll('.timeframe-btn');
                
                if (!chart || !tooltip) return;
                
                const chartData = JSON.parse(chart.dataset.chartData || '[]');
                const symbol = chart.dataset.symbol;
                
                // Add hover functionality
                chart.addEventListener('mousemove', function(e) {
                  const rect = chart.getBoundingClientRect();
                  const x = e.clientX - rect.left;
                  const y = e.clientY - rect.top;
                  
                  // Calculate which data point we're hovering over
                  const dataIndex = Math.round(((x - 40) / (${chartWidth} - 80)) * (chartData.length - 1));
                  
                  if (dataIndex >= 0 && dataIndex < chartData.length) {
                    const point = chartData[dataIndex];
                    const date = new Date(point.timestamp);
                    
                    // Update tooltip content
                    document.getElementById('tooltip-date').textContent = date.toLocaleDateString();
                    document.getElementById('tooltip-ohlc').innerHTML = 
                      'Open: $' + parseFloat(point.open).toFixed(2) + '<br>' +
                      'High: $' + parseFloat(point.high).toFixed(2) + '<br>' +
                      'Low: $' + parseFloat(point.low).toFixed(2) + '<br>' +
                      '<strong>Close: $' + parseFloat(point.close).toFixed(2) + '</strong><br>' +
                      'Volume: ' + point.volume.toLocaleString();
                    
                    // Position tooltip EXACTLY next to cursor
                    const tooltipWidth = 160;
                    const tooltipHeight = 100;
                    const offset = 10; // Very small offset from cursor
                    
                    // Position to the right and slightly above cursor
                    let left = e.clientX + offset;
                    let top = e.clientY - offset;
                    
                    // If tooltip would go off right edge, show on left side
                    if (left + tooltipWidth > window.innerWidth - 10) {
                      left = e.clientX - tooltipWidth - offset;
                    }
                    
                    // If tooltip would go off top edge, show below cursor
                    if (top < 10) {
                      top = e.clientY + offset;
                    }
                    
                    // If tooltip would go off bottom edge, show above cursor
                    if (top + tooltipHeight > window.innerHeight - 10) {
                      top = e.clientY - tooltipHeight - offset;
                    }
                    
                    tooltip.style.left = left + 'px';
                    tooltip.style.top = top + 'px';
                    tooltip.style.opacity = '1';
                  }
                });
                
                chart.addEventListener('mouseleave', function() {
                  tooltip.style.opacity = '0';
                });
                
                // Add timeframe button functionality
                buttons.forEach(button => {
                  button.addEventListener('click', async function() {
                    const period = this.dataset.period;
                    const symbol = this.dataset.symbol;
                    
                    // Update button states
                    buttons.forEach(btn => {
                      btn.classList.remove('bg-white', 'text-primary-600', 'shadow-sm');
                      btn.classList.add('text-gray-600');
                    });
                    this.classList.add('bg-white', 'text-primary-600', 'shadow-sm');
                    this.classList.remove('text-gray-600');
                    
                    // Show loading
                    this.textContent = '...';
                    
                    try {
                      // Fetch new data
                      const response = await fetch('/api/v1/tickers/' + symbol + '/chart?period=' + period);
                      const result = await response.json();
                      
                      if (result.data?.dataPoints) {
                        // Reload the page with new period (simple but effective)
                        window.location.search = '?period=' + period;
                      }
                    } catch (error) {
                      console.error('Failed to fetch chart data:', error);
                      this.textContent = period;
                    }
                  });
                });
              }
            })();
          `
        }}
      />
    </div>
  );
}
