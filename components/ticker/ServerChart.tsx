interface ServerChartProps {
  chartData: any;
  symbol: string;
}

interface ChartDataPoint {
  timestamp: string;
  open: string;
  high: string;
  low: string;
  close: string;
  volume: number;
}

export default function ServerChart({ chartData, symbol }: ServerChartProps) {
  if (!chartData?.dataPoints || chartData.dataPoints.length === 0) {
    return (
      <div className="p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-6">Price Chart</h3>
        <div className="h-80 bg-gray-100 rounded-lg flex items-center justify-center">
          <div className="text-gray-500">No chart data available</div>
        </div>
      </div>
    );
  }

  const dataPoints: ChartDataPoint[] = chartData.dataPoints;
  const currentPrice = parseFloat(dataPoints[dataPoints.length - 1]?.close || '0');
  const firstPrice = parseFloat(dataPoints[0]?.close || '0');
  const priceChange = currentPrice - firstPrice;
  const priceChangePercent = firstPrice > 0 ? (priceChange / firstPrice) * 100 : 0;
  
  // Calculate chart statistics
  const high = Math.max(...dataPoints.map(p => parseFloat(p.high)));
  const low = Math.min(...dataPoints.map(p => parseFloat(p.low)));
  const avgVolume = dataPoints.reduce((sum, p) => sum + p.volume, 0) / dataPoints.length;

  // Create a simple SVG chart
  const chartHeight = 200;
  const chartWidth = 600;
  const padding = 20;
  
  const priceRange = high - low;
  const priceScale = (chartHeight - 2 * padding) / priceRange;
  
  // Generate SVG path for price line
  const pathData = dataPoints.map((point, index) => {
    const x = padding + (index / (dataPoints.length - 1)) * (chartWidth - 2 * padding);
    const y = chartHeight - padding - (parseFloat(point.close) - low) * priceScale;
    return `${index === 0 ? 'M' : 'L'} ${x} ${y}`;
  }).join(' ');

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-lg font-semibold text-gray-900">Price Chart ({chartData.period})</h3>
        <div className="flex space-x-1 bg-gray-100 rounded-lg p-1">
          <button className="px-3 py-1 text-sm font-medium rounded-md bg-white text-primary-600 shadow-sm">
            {chartData.period}
          </button>
        </div>
      </div>
      
      {/* SVG Chart */}
      <div className="h-80 bg-gray-50 rounded-lg p-4">
        <svg width="100%" height="100%" viewBox={`0 0 ${chartWidth} ${chartHeight}`} className="w-full h-full">
          {/* Grid lines */}
          <defs>
            <pattern id="grid" width="50" height="25" patternUnits="userSpaceOnUse">
              <path d="M 50 0 L 0 0 0 25" fill="none" stroke="#e5e7eb" strokeWidth="0.5"/>
            </pattern>
          </defs>
          <rect width="100%" height="100%" fill="url(#grid)" />
          
          {/* Price area fill */}
          <path
            d={`${pathData} L ${chartWidth - padding} ${chartHeight - padding} L ${padding} ${chartHeight - padding} Z`}
            fill="rgba(37, 99, 235, 0.1)"
            stroke="none"
          />
          
          {/* Price line */}
          <path
            d={pathData}
            fill="none"
            stroke="#2563eb"
            strokeWidth="2"
          />
          
          {/* Y-axis labels */}
          <text x="5" y="25" className="text-xs fill-gray-600">${high.toFixed(2)}</text>
          <text x="5" y={chartHeight - 10} className="text-xs fill-gray-600">${low.toFixed(2)}</text>
          
          {/* Current price indicator */}
          {dataPoints.length > 0 && (
            <circle
              cx={chartWidth - padding}
              cy={chartHeight - padding - (currentPrice - low) * priceScale}
              r="4"
              fill="#2563eb"
              stroke="white"
              strokeWidth="2"
            />
          )}
        </svg>
      </div>
      
      {/* Chart Statistics */}
      <div className="mt-4 grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
        <div>
          <div className="text-gray-500">Current</div>
          <div className="font-semibold">${currentPrice.toFixed(2)}</div>
        </div>
        <div>
          <div className="text-gray-500">Change</div>
          <div className={`font-semibold ${priceChange >= 0 ? 'text-green-600' : 'text-red-600'}`}>
            {priceChange >= 0 ? '+' : ''}${priceChange.toFixed(2)} ({priceChangePercent.toFixed(2)}%)
          </div>
        </div>
        <div>
          <div className="text-gray-500">High</div>
          <div className="font-semibold text-green-600">${high.toFixed(2)}</div>
        </div>
        <div>
          <div className="text-gray-500">Low</div>
          <div className="font-semibold text-red-600">${low.toFixed(2)}</div>
        </div>
      </div>
      
      {/* Period Summary */}
      <div className="mt-4 p-4 bg-gray-50 rounded-lg">
        <div className="flex justify-between items-center text-sm">
          <span className="text-gray-600">Period: {chartData.period}</span>
          <span className="text-gray-600">Data Points: {dataPoints.length}</span>
          <span className="text-gray-600">Avg Volume: {(avgVolume / 1000000).toFixed(1)}M</span>
        </div>
      </div>
    </div>
  );
}
