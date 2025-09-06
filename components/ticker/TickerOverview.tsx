'use client';

import { useState, useEffect } from 'react';
import { ArrowTrendingUpIcon, ArrowTrendingDownIcon } from '@heroicons/react/24/outline';
import { safeToFixed, safeParseNumber, formatLargeNumber } from '@/lib/utils';

interface TickerOverviewProps {
  symbol: string;
}

interface StockData {
  stock: {
    symbol: string;
    name: string;
    exchange: string;
    sector: string;
    marketCap: number | string;
  };
  price: {
    price: number | string;
    change: number | string;
    changePercent: number | string;
    volume: number | string;
    timestamp: string;
  };
  keyMetrics: {
    week52High: number | string;
    week52Low: number | string;
    trailingPE: number | string;
    marketCap: number | string;
  };
}

export default function TickerOverview({ symbol }: TickerOverviewProps) {
  const [data, setData] = useState<StockData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // TEMP: Test with static real data to verify display works
  const testData: StockData = {
    stock: {
      symbol: symbol,
      name: symbol === 'AAPL' ? 'APPLE INC.' : 'Company Name',
      exchange: 'NASDAQ',
      sector: 'Technology',
      marketCap: symbol === 'AAPL' ? '3700000000000' : '1000000000000'
    },
    price: {
      price: symbol === 'AAPL' ? '239.69' : '100.00',
      change: '-0.30',
      changePercent: '-0.13',
      volume: '54870397',
      timestamp: new Date().toISOString()
    },
    keyMetrics: {
      week52High: symbol === 'AAPL' ? '198.23' : '150.00',
      week52Low: symbol === 'AAPL' ? '124.17' : '80.00',
      trailingPE: symbol === 'AAPL' ? '28.5' : '25.0',
      marketCap: symbol === 'AAPL' ? '3700000000000' : '1000000000000'
    }
  };

  useEffect(() => {
    const fetchTickerData = async () => {
      try {
        setLoading(true);
        console.log(`🚀 Fetching data for ${symbol}...`);
        
        // TEMP: Use test data to verify component display works
        console.log('🧪 Using test data temporarily');
        setData(testData);
        setError(null);
        
        // TODO: Uncomment this when client-side rendering is fixed
        /*
        const response = await fetch(`/api/v1/tickers/${symbol}`);
        console.log(`📡 Response status: ${response.status}`);
        
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: Failed to fetch ticker data`);
        }
        
        const result = await response.json();
        console.log('📊 API Result:', result);
        
        // Validate the response structure
        if (!result?.data?.summary) {
          throw new Error('Invalid API response structure');
        }
        
        // Map the API response to match our component interface
        const mappedData = {
          stock: result.data.summary.stock,
          price: result.data.summary.price,
          keyMetrics: result.data.summary.keyMetrics
        };
        
        console.log('✅ Mapped data:', mappedData);
        setData(mappedData);
        setError(null);
        */
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : 'Failed to fetch data';
        console.error('❌ Error fetching ticker data:', err);
        setError(errorMsg);
      } finally {
        setLoading(false);
        console.log(`🏁 Loading complete for ${symbol}`);
      }
    };

    // Add a small delay to ensure the component is fully mounted
    const timer = setTimeout(fetchTickerData, 100);
    return () => clearTimeout(timer);
  }, [symbol]);

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4">
        <h3 className="text-red-800 font-semibold">Error Loading Data</h3>
        <p className="text-red-600 text-sm mt-1">{error}</p>
        <p className="text-red-500 text-xs mt-2">Check browser console for details</p>
      </div>
    );
  }

  if (loading) {
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


  if (!data) {
    return null;
  }

  const { stock, price, keyMetrics } = data;
  const changeValue = safeParseNumber(price.change);
  const isPositive = changeValue >= 0;

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">
            {stock.symbol}
          </h1>
          <p className="text-lg text-gray-600">{stock.name}</p>
          <div className="flex items-center space-x-4 text-sm text-gray-500 mt-1">
            <span>{stock.exchange}</span>
            <span>•</span>
            <span>{stock.sector}</span>
            <span>•</span>
            <span>Market Cap: {formatLargeNumber(keyMetrics.marketCap)}</span>
          </div>
        </div>

        <div className="text-right">
          <div className="text-3xl font-bold text-gray-900">
            ${safeToFixed(price.price, 2)}
          </div>
          <div className={`flex items-center justify-end text-lg font-medium ${
            isPositive ? 'text-green-600' : 'text-red-600'
          }`}>
            {isPositive ? (
              <ArrowTrendingUpIcon className="h-5 w-5 mr-1" />
            ) : (
              <ArrowTrendingDownIcon className="h-5 w-5 mr-1" />
            )}
            <span>
              {isPositive ? '+' : ''}{safeToFixed(price.change, 2)} 
              ({isPositive ? '+' : ''}{safeToFixed(price.changePercent, 2)}%)
            </span>
          </div>
          <div className="text-sm text-gray-500 mt-1">
            Volume: {safeToFixed(safeParseNumber(price.volume) / 1000000, 1)}M
          </div>
        </div>
      </div>

      {/* Quick Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 pt-4 border-t border-gray-200">
        <div className="text-center">
          <div className="text-sm text-gray-500">52W High</div>
          <div className="font-semibold">${safeToFixed(keyMetrics.week52High, 2)}</div>
        </div>
        <div className="text-center">
          <div className="text-sm text-gray-500">52W Low</div>
          <div className="font-semibold">${safeToFixed(keyMetrics.week52Low, 2)}</div>
        </div>
        <div className="text-center">
          <div className="text-sm text-gray-500">P/E Ratio</div>
          <div className="font-semibold">{safeToFixed(keyMetrics.trailingPE, 1)}</div>
        </div>
        <div className="text-center">
          <div className="text-sm text-gray-500">Market Cap</div>
          <div className="font-semibold">{formatLargeNumber(keyMetrics.marketCap)}</div>
        </div>
      </div>
    </div>
  );
}
