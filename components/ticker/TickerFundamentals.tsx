'use client';

import { useState, useEffect } from 'react';
import { safeToFixed, safeParseNumber, formatLargeNumber, formatPercent } from '@/lib/utils';

interface TickerFundamentalsProps {
  symbol: string;
}

interface Fundamentals {
  pe: number | string;
  pb: number | string;
  ps: number | string;
  roe: number | string;
  roa: number | string;
  revenue: number | string;
  netIncome: number | string;
  eps: number | string;
  debtToEquity: number | string;
  currentRatio: number | string;
  grossMargin: number | string;
  operatingMargin: number | string;
  netMargin: number | string;
}

interface KeyMetrics {
  week52High: number | string;
  week52Low: number | string;
  ytdChange: number | string;
  beta: number | string;
  averageVolume: number | string;
  sharesOutstanding: number | string;
  revenueGrowth1Y: number | string;
  earningsGrowth1Y: number | string;
}

export default function TickerFundamentals({ symbol }: TickerFundamentalsProps) {
  const [fundamentals, setFundamentals] = useState<Fundamentals | null>(null);
  const [keyMetrics, setKeyMetrics] = useState<KeyMetrics | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        console.log(`🔥 Fetching REAL fundamentals for ${symbol}...`);
        
        const response = await fetch(`/api/v1/tickers/${symbol}`);
        console.log(`📡 Fundamentals response status: ${response.status}`);
        
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: Failed to fetch fundamentals`);
        }
        
        const result = await response.json();
        console.log('📊 Full API Response for fundamentals:', result);
        
        // Extract real fundamentals data from Polygon.io
        const realFundamentals = result.data.summary.fundamentals;
        const realKeyMetrics = result.data.summary.keyMetrics;
        
        console.log('💰 REAL Fundamentals:', realFundamentals);
        console.log('📈 REAL Key Metrics:', realKeyMetrics);
        
        // Map to our component interface with REAL data
        const mappedFundamentals: Fundamentals = {
          pe: realFundamentals?.pe || 'N/A',
          pb: realFundamentals?.pb || 'N/A', 
          ps: realFundamentals?.ps || 'N/A',
          roe: realFundamentals?.roe || 'N/A',
          roa: realFundamentals?.roa || 'N/A',
          revenue: realFundamentals?.revenue || '0',
          netIncome: realFundamentals?.netIncome || '0',
          eps: realFundamentals?.eps || 'N/A',
          debtToEquity: realKeyMetrics?.debtToEquity || 'N/A',
          currentRatio: realKeyMetrics?.currentRatio || 'N/A',
          grossMargin: realFundamentals?.grossMargin || 'N/A',
          operatingMargin: realFundamentals?.operatingMargin || 'N/A',
          netMargin: realFundamentals?.netMargin || 'N/A'
        };
        
        const mappedKeyMetrics: KeyMetrics = {
          week52High: realKeyMetrics?.week52High || '0',
          week52Low: realKeyMetrics?.week52Low || '0',
          ytdChange: realKeyMetrics?.ytdChange || '0',
          beta: realKeyMetrics?.beta || '1.0',
          averageVolume: realKeyMetrics?.averageVolume || '0',
          sharesOutstanding: realKeyMetrics?.sharesOutstanding || '0',
          revenueGrowth1Y: realKeyMetrics?.revenueGrowth1Y || '0',
          earningsGrowth1Y: realKeyMetrics?.earningsGrowth1Y || '0'
        };
        
        console.log('✅ MAPPED Real Fundamentals:', mappedFundamentals);
        setFundamentals(mappedFundamentals);
        setKeyMetrics(mappedKeyMetrics);
      } catch (error) {
        console.error('❌ Error fetching fundamentals:', error);
      } finally {
        setLoading(false);
        console.log(`🏁 Fundamentals loading complete for ${symbol}`);
      }
    };

    // Add delay for proper mounting
    const timer = setTimeout(fetchData, 100);
    return () => clearTimeout(timer);
  }, [symbol]);

  if (loading) {
    return (
      <div className="p-6">
        <div className="h-6 bg-gray-200 rounded w-32 mb-4 animate-pulse"></div>
        <div className="space-y-3">
          {[1, 2, 3, 4, 5, 6].map((i) => (
            <div key={i} className="flex justify-between animate-pulse">
              <div className="h-4 bg-gray-200 rounded w-24"></div>
              <div className="h-4 bg-gray-200 rounded w-16"></div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (!fundamentals || !keyMetrics) {
    return (
      <div className="p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Key Metrics</h3>
        <p className="text-gray-500">No fundamental data available</p>
      </div>
    );
  }



  return (
    <div className="p-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-6">Key Metrics</h3>
      
      {/* Valuation Metrics */}
      <div className="mb-6">
        <h4 className="text-sm font-medium text-gray-700 mb-3 uppercase tracking-wide">Valuation</h4>
        <div className="space-y-3">
          <div className="flex justify-between">
            <span className="text-gray-600">P/E Ratio</span>
            <span className="font-medium">{safeToFixed(fundamentals.pe, 1)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Price/Book</span>
            <span className="font-medium">{safeToFixed(fundamentals.pb, 1)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Price/Sales</span>
            <span className="font-medium">{safeToFixed(fundamentals.ps, 1)}</span>
          </div>
        </div>
      </div>

      {/* Profitability */}
      <div className="mb-6">
        <h4 className="text-sm font-medium text-gray-700 mb-3 uppercase tracking-wide">Profitability</h4>
        <div className="space-y-3">
          <div className="flex justify-between">
            <span className="text-gray-600">ROE</span>
            <span className="font-medium">{formatPercent(fundamentals.roe)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">ROA</span>
            <span className="font-medium">{formatPercent(fundamentals.roa)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Gross Margin</span>
            <span className="font-medium">{formatPercent(fundamentals.grossMargin)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Net Margin</span>
            <span className="font-medium">{formatPercent(fundamentals.netMargin)}</span>
          </div>
        </div>
      </div>

      {/* Financial Health */}
      <div className="mb-6">
        <h4 className="text-sm font-medium text-gray-700 mb-3 uppercase tracking-wide">Financial Health</h4>
        <div className="space-y-3">
          <div className="flex justify-between">
            <span className="text-gray-600">Debt/Equity</span>
            <span className="font-medium">{safeToFixed(fundamentals.debtToEquity, 1)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Current Ratio</span>
            <span className="font-medium">{safeToFixed(fundamentals.currentRatio, 1)}</span>
          </div>
        </div>
      </div>

      {/* Performance */}
      <div className="mb-6">
        <h4 className="text-sm font-medium text-gray-700 mb-3 uppercase tracking-wide">Performance</h4>
        <div className="space-y-3">
          <div className="flex justify-between">
            <span className="text-gray-600">YTD Change</span>
            <span className={`font-medium ${safeParseNumber(keyMetrics.ytdChange) >= 0 ? 'text-green-600' : 'text-red-600'}`}>
              {formatPercent(keyMetrics.ytdChange)}
            </span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Revenue Growth</span>
            <span className={`font-medium ${safeParseNumber(keyMetrics.revenueGrowth1Y) >= 0 ? 'text-green-600' : 'text-red-600'}`}>
              {formatPercent(keyMetrics.revenueGrowth1Y)}
            </span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Earnings Growth</span>
            <span className={`font-medium ${safeParseNumber(keyMetrics.earningsGrowth1Y) >= 0 ? 'text-green-600' : 'text-red-600'}`}>
              {formatPercent(keyMetrics.earningsGrowth1Y)}
            </span>
          </div>
        </div>
      </div>

      {/* Market Data */}
      <div>
        <h4 className="text-sm font-medium text-gray-700 mb-3 uppercase tracking-wide">Market Data</h4>
        <div className="space-y-3">
          <div className="flex justify-between">
            <span className="text-gray-600">Beta</span>
            <span className="font-medium">{safeToFixed(keyMetrics.beta, 2)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Avg Volume</span>
            <span className="font-medium">{safeToFixed(safeParseNumber(keyMetrics.averageVolume) / 1000000, 1)}M</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Shares Out</span>
            <span className="font-medium">{safeToFixed(safeParseNumber(keyMetrics.sharesOutstanding) / 1000000000, 1)}B</span>
          </div>
        </div>
      </div>
    </div>
  );
}
