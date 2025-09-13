import { Suspense } from 'react';
import TickerOverview from '@/components/ticker/TickerOverview';
import TickerChart from '@/components/ticker/TickerChart';
import InteractiveChart from '@/components/ticker/InteractiveChart';
import ServerChart from '@/components/ticker/ServerChart';
import ProperChart from '@/components/ticker/ProperChart';
import HybridChart from '@/components/ticker/HybridChart';
import TickerFundamentals from '@/components/ticker/TickerFundamentals';
import TickerNews from '@/components/ticker/TickerNews';
import TickerEarnings from '@/components/ticker/TickerEarnings';
import TickerAnalysts from '@/components/ticker/TickerAnalysts';
import RealTimePriceHeader from '@/components/ticker/RealTimePriceHeader';

interface PageProps {
  params: {
    symbol: string;
  };
  searchParams: {
    period?: string;
  };
}

// Fetch ticker data server-side to avoid client hydration issues
async function getTickerData(symbol: string) {
  try {
    // Use internal backend service URL for server-side fetching
    const response = await fetch(`http://investorcenter-backend-service.investorcenter.svc.cluster.local:8080/api/v1/tickers/${symbol}`, {
      cache: 'no-store', // Always fetch fresh data
    });
    
    if (!response.ok) {
      console.error(`Failed to fetch ticker data: ${response.status}`);
      return null;
    }
    
    const result = await response.json();
    console.log(`✅ Server-side fetched data for ${symbol}`);
    return result.data;
  } catch (error) {
    console.error('Error fetching ticker data server-side:', error);
    return null;
  }
}

// Fetch chart data server-side
async function getChartData(symbol: string, period: string = '1Y') {
  try {
    const response = await fetch(`http://investorcenter-backend-service.investorcenter.svc.cluster.local:8080/api/v1/tickers/${symbol}/chart?period=${period}`, {
      cache: 'no-store',
    });
    
    if (!response.ok) {
      console.error(`Failed to fetch chart data: ${response.status}`);
      return null;
    }
    
    const result = await response.json();
    console.log(`✅ Server-side fetched chart data for ${symbol}`);
    return result.data;
  } catch (error) {
    console.error('Error fetching chart data server-side:', error);
    return null;
  }
}

export default async function TickerPage({ params, searchParams }: PageProps) {
  const symbol = params.symbol.toUpperCase();
  const period = searchParams.period || '1Y';
  
  // Fetch data server-side
  const [tickerData, chartData] = await Promise.all([
    getTickerData(symbol),
    getChartData(symbol, period)
  ]);
  
  if (!tickerData) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="bg-red-50 border border-red-200 rounded-lg p-6">
          <h2 className="text-red-800 font-semibold">Failed to Load Data</h2>
          <p className="text-red-600 mt-2">Could not fetch data for {symbol}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Ticker Overview Header with Real-time Updates */}
      <div className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <RealTimePriceHeader symbol={symbol} initialData={tickerData.summary} />
        </div>
      </div>

      {/* Main Content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Left Column - Chart and News */}
          <div className="lg:col-span-2 space-y-8">
            {/* HYBRID Interactive Price Chart */}
            <div className="bg-white rounded-lg shadow relative">
              <HybridChart 
                symbol={symbol} 
                initialData={chartData} 
                currentPrice={parseFloat(tickerData.summary.price.price)} 
              />
            </div>

            {/* News & Analysis */}
            <div className="bg-white rounded-lg shadow">
              <TickerNews symbol={symbol} />
            </div>

            {/* Earnings */}
            <div className="bg-white rounded-lg shadow">
              <TickerEarnings symbol={symbol} />
            </div>
          </div>

          {/* Right Column - Fundamentals and Analysis */}
          <div className="space-y-8">
            {/* Key Metrics */}
            <div className="bg-white rounded-lg shadow">
              <TickerFundamentalsServer data={tickerData.summary} />
            </div>

            {/* Analyst Ratings */}
            <div className="bg-white rounded-lg shadow">
              <TickerAnalysts symbol={symbol} />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// Server-side components that display real data
function TickerOverviewServer({ data }: { data: any }) {
  const { stock, price, keyMetrics } = data;
  
  return (
    <div className="flex items-center justify-between">
      <div>
        <h1 className="text-3xl font-bold text-gray-900">
          {stock.name} ({stock.symbol})
        </h1>
        <p className="text-gray-600">{stock.exchange} • {stock.sector}</p>
      </div>
      <div className="text-right">
        <div className="text-3xl font-bold text-gray-900">
          ${parseFloat(price.price).toFixed(2)}
        </div>
        <div className={`text-sm ${parseFloat(price.change) >= 0 ? 'text-green-600' : 'text-red-600'}`}>
          {parseFloat(price.change) >= 0 ? '+' : ''}{parseFloat(price.change).toFixed(2)} 
          ({(parseFloat(price.changePercent) * 100).toFixed(2)}%)
        </div>
      </div>
    </div>
  );
}

function TickerFundamentalsServer({ data }: { data: any }) {
  const { fundamentals, keyMetrics } = data;
  
  const formatLargeNumber = (value: string | number | null | undefined) => {
    if (value === null || value === undefined) return 'N/A';
    const num = parseFloat(value.toString());
    if (isNaN(num)) return 'N/A';
    if (num >= 1e12) return `$${(num / 1e12).toFixed(1)}T`;
    if (num >= 1e9) return `$${(num / 1e9).toFixed(1)}B`;
    if (num >= 1e6) return `$${(num / 1e6).toFixed(1)}M`;
    return `$${num.toFixed(2)}`;
  };
  
  const formatPercent = (value: string | number | null | undefined) => {
    if (value === null || value === undefined) return 'N/A';
    const num = parseFloat(value.toString());
    if (isNaN(num)) return 'N/A';
    return `${(num * 100).toFixed(1)}%`;
  };
  
  const formatRatio = (value: string | number | null | undefined) => {
    if (value === null || value === undefined) return 'N/A';
    const num = parseFloat(value.toString());
    if (isNaN(num)) return 'N/A';
    return num.toFixed(2);
  };
  
  return (
    <div className="p-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-6">Key Fundamentals</h3>
      
      {/* 🥇 TOP PRIORITY METRICS */}
      <div className="mb-6">
        <h4 className="text-sm font-semibold text-gray-800 mb-3 uppercase tracking-wide">Valuation & Profitability</h4>
        <div className="space-y-3">
          <div className="flex justify-between">
            <span className="text-gray-600">P/E Ratio</span>
            <span className="font-semibold text-blue-600">{formatRatio(fundamentals.pe)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">EPS (Basic)</span>
            <span className="font-semibold">${formatRatio(fundamentals.eps)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Revenue (TTM)</span>
            <span className="font-semibold">{formatLargeNumber(fundamentals.revenue)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Net Income</span>
            <span className="font-semibold">{formatLargeNumber(fundamentals.netIncome)}</span>
          </div>
        </div>
      </div>

      {/* 🥈 PROFITABILITY MARGINS */}
      <div className="mb-6">
        <h4 className="text-sm font-semibold text-gray-800 mb-3 uppercase tracking-wide">Margins</h4>
        <div className="space-y-3">
          <div className="flex justify-between">
            <span className="text-gray-600">Gross Margin</span>
            <span className="font-semibold">{formatPercent(fundamentals.grossMargin)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Operating Margin</span>
            <span className="font-semibold">{formatPercent(fundamentals.operatingMargin)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Net Margin</span>
            <span className="font-semibold">{formatPercent(fundamentals.netMargin)}</span>
          </div>
        </div>
      </div>

      {/* 🥉 FINANCIAL HEALTH */}
      <div className="mb-6">
        <h4 className="text-sm font-semibold text-gray-800 mb-3 uppercase tracking-wide">Financial Health</h4>
        <div className="space-y-3">
          <div className="flex justify-between">
            <span className="text-gray-600">ROE</span>
            <span className="font-semibold">{formatPercent(fundamentals.roe)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">ROA</span>
            <span className="font-semibold">{formatPercent(fundamentals.roa)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Current Ratio</span>
            <span className="font-semibold">{formatRatio(fundamentals.currentRatio)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Debt-to-Equity</span>
            <span className="font-semibold">{formatRatio(fundamentals.debtToEquity)}</span>
          </div>
        </div>
      </div>

      {/* 📊 OTHER RATIOS */}
      <div>
        <h4 className="text-sm font-semibold text-gray-800 mb-3 uppercase tracking-wide">Valuation Ratios</h4>
        <div className="space-y-3">
          <div className="flex justify-between">
            <span className="text-gray-600">P/B Ratio</span>
            <span className="font-semibold">{formatRatio(fundamentals.pb)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">P/S Ratio</span>
            <span className="font-semibold">{formatRatio(fundamentals.ps)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Quick Ratio</span>
            <span className="font-semibold">{formatRatio(fundamentals.quickRatio)}</span>
          </div>
        </div>
      </div>
    </div>
  );
}

// Loading Skeletons
function TickerHeaderSkeleton() {
  return (
    <div className="animate-pulse">
      <div className="flex items-center justify-between">
        <div>
          <div className="h-8 bg-gray-200 rounded w-48 mb-2"></div>
          <div className="h-4 bg-gray-200 rounded w-32"></div>
        </div>
        <div className="text-right">
          <div className="h-8 bg-gray-200 rounded w-24 mb-2"></div>
          <div className="h-4 bg-gray-200 rounded w-20"></div>
        </div>
      </div>
    </div>
  );
}

function ChartSkeleton() {
  return (
    <div className="p-6">
      <div className="h-6 bg-gray-200 rounded w-32 mb-4"></div>
      <div className="h-80 bg-gray-200 rounded"></div>
    </div>
  );
}

function NewsSkeleton() {
  return (
    <div className="p-6">
      <div className="h-6 bg-gray-200 rounded w-40 mb-4"></div>
      <div className="space-y-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="border-b border-gray-100 pb-4">
            <div className="h-4 bg-gray-200 rounded w-3/4 mb-2"></div>
            <div className="h-3 bg-gray-200 rounded w-full mb-1"></div>
            <div className="h-3 bg-gray-200 rounded w-2/3"></div>
          </div>
        ))}
      </div>
    </div>
  );
}

function EarningsSkeleton() {
  return (
    <div className="p-6">
      <div className="h-6 bg-gray-200 rounded w-32 mb-4"></div>
      <div className="space-y-3">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="flex justify-between items-center">
            <div className="h-4 bg-gray-200 rounded w-20"></div>
            <div className="h-4 bg-gray-200 rounded w-16"></div>
            <div className="h-4 bg-gray-200 rounded w-16"></div>
          </div>
        ))}
      </div>
    </div>
  );
}

function FundamentalsSkeleton() {
  return (
    <div className="p-6">
      <div className="h-6 bg-gray-200 rounded w-32 mb-4"></div>
      <div className="space-y-3">
        {[1, 2, 3, 4, 5, 6].map((i) => (
          <div key={i} className="flex justify-between">
            <div className="h-4 bg-gray-200 rounded w-24"></div>
            <div className="h-4 bg-gray-200 rounded w-16"></div>
          </div>
        ))}
      </div>
    </div>
  );
}

function AnalystsSkeleton() {
  return (
    <div className="p-6">
      <div className="h-6 bg-gray-200 rounded w-32 mb-4"></div>
      <div className="space-y-3">
        <div className="h-8 bg-gray-200 rounded w-full"></div>
        <div className="h-4 bg-gray-200 rounded w-3/4"></div>
        <div className="h-4 bg-gray-200 rounded w-2/3"></div>
      </div>
    </div>
  );
}
