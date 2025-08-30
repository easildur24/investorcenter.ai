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

  useEffect(() => {
    const fetchTickerData = async () => {
      try {
        setLoading(true);
        const response = await fetch(`/api/v1/tickers/${symbol}`);
        if (!response.ok) {
          throw new Error('Failed to fetch ticker data');
        }
        const result = await response.json();
        setData(result.data.summary);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch data');
        console.error('Error fetching ticker data:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchTickerData();
  }, [symbol]);

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

  if (error) {
    return (
      <div className="text-red-600">
        <p>Error loading ticker data: {error}</p>
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
