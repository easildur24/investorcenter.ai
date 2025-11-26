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

// Helper to check if value is valid (not null, undefined, 0, or 'N/A')
const isValidValue = (value: any): boolean => {
  if (value === null || value === undefined || value === 'N/A') return false;
  if (typeof value === 'number' && value === 0) return false;
  if (typeof value === 'string' && (value === '0' || value === '0.00')) return false;
  return true;
};

export default function TickerFundamentals({ symbol }: TickerFundamentalsProps) {
  const [fundamentals, setFundamentals] = useState<Fundamentals | null>(null);
  const [keyMetrics, setKeyMetrics] = useState<KeyMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [isCrypto, setIsCrypto] = useState(false);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        console.log(`🔥 Fetching fundamentals for ${symbol}...`);

        // Fetch all data sources in parallel
        const [tickerResponse, financialsResponse, riskResponse] = await Promise.all([
          fetch(`/api/v1/tickers/${symbol}`),
          fetch(`/api/v1/stocks/${symbol}/financials`).catch(() => null),
          fetch(`/api/v1/stocks/${symbol}/risk?period=1Y`).catch(() => null)
        ]);

        if (!tickerResponse.ok) {
          throw new Error(`HTTP ${tickerResponse.status}: Failed to fetch ticker data`);
        }

        const tickerResult = await tickerResponse.json();

        // Check if this is a crypto asset
        if (tickerResult.data.summary.stock.isCrypto) {
          console.log('🪙 This is a crypto asset, skipping stock fundamentals');
          setIsCrypto(true);
          setLoading(false);
          return;
        }

        // Extract data from main ticker endpoint (Polygon.io)
        const polygonFundamentals = tickerResult.data.summary.fundamentals || {};
        const polygonKeyMetrics = tickerResult.data.summary.keyMetrics || {};

        // Extract IC Score financial metrics if available
        let icScoreFinancials: any = {};
        if (financialsResponse?.ok) {
          const financialsResult = await financialsResponse.json();
          icScoreFinancials = financialsResult.data || {};
          console.log('📊 IC Score Financials:', icScoreFinancials);
        }

        // Extract IC Score risk metrics if available
        let icScoreRisk: any = {};
        if (riskResponse?.ok) {
          const riskResult = await riskResponse.json();
          icScoreRisk = riskResult.data || {};
          console.log('📊 IC Score Risk:', icScoreRisk);
        }

        // Merge data - prefer IC Score data when available, fallback to Polygon
        const mappedFundamentals: Fundamentals = {
          pe: polygonFundamentals?.pe || icScoreFinancials?.pe_ratio || 'N/A',
          pb: polygonFundamentals?.pb || icScoreFinancials?.pb_ratio || 'N/A',
          ps: polygonFundamentals?.ps || icScoreFinancials?.ps_ratio || 'N/A',
          // Prefer IC Score margins (from SEC filings) as they're more accurate
          roe: isValidValue(icScoreFinancials?.roe) ? icScoreFinancials.roe : (polygonFundamentals?.roe || 'N/A'),
          roa: isValidValue(icScoreFinancials?.roa) ? icScoreFinancials.roa : (polygonFundamentals?.roa || 'N/A'),
          revenue: polygonFundamentals?.revenue || '0',
          netIncome: polygonFundamentals?.netIncome || '0',
          eps: polygonFundamentals?.eps || 'N/A',
          debtToEquity: isValidValue(icScoreFinancials?.debt_to_equity) ? icScoreFinancials.debt_to_equity : (polygonKeyMetrics?.debtToEquity || 'N/A'),
          currentRatio: isValidValue(icScoreFinancials?.current_ratio) ? icScoreFinancials.current_ratio : (polygonKeyMetrics?.currentRatio || 'N/A'),
          grossMargin: isValidValue(icScoreFinancials?.gross_margin) ? icScoreFinancials.gross_margin : (polygonFundamentals?.grossMargin || 'N/A'),
          operatingMargin: isValidValue(icScoreFinancials?.operating_margin) ? icScoreFinancials.operating_margin : (polygonFundamentals?.operatingMargin || 'N/A'),
          netMargin: isValidValue(icScoreFinancials?.net_margin) ? icScoreFinancials.net_margin : (polygonFundamentals?.netMargin || 'N/A')
        };

        const mappedKeyMetrics: KeyMetrics = {
          week52High: polygonKeyMetrics?.week52High || '0',
          week52Low: polygonKeyMetrics?.week52Low || '0',
          ytdChange: polygonKeyMetrics?.ytdChange || '0',
          // Prefer IC Score beta from risk_metrics table
          beta: isValidValue(icScoreRisk?.beta) ? icScoreRisk.beta : (polygonKeyMetrics?.beta || '1.0'),
          averageVolume: polygonKeyMetrics?.averageVolume || '0',
          // Prefer IC Score shares outstanding from SEC filings
          sharesOutstanding: isValidValue(icScoreFinancials?.shares_outstanding) ? icScoreFinancials.shares_outstanding : (polygonKeyMetrics?.sharesOutstanding || '0'),
          // Prefer IC Score growth metrics
          revenueGrowth1Y: isValidValue(icScoreFinancials?.revenue_growth_yoy) ? icScoreFinancials.revenue_growth_yoy : (polygonKeyMetrics?.revenueGrowth1Y || '0'),
          earningsGrowth1Y: isValidValue(icScoreFinancials?.earnings_growth_yoy) ? icScoreFinancials.earnings_growth_yoy : (polygonKeyMetrics?.earningsGrowth1Y || '0')
        };

        console.log('✅ Merged Fundamentals:', mappedFundamentals);
        console.log('✅ Merged Key Metrics:', mappedKeyMetrics);
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

  // Don't render stock fundamentals for crypto assets
  if (isCrypto) {
    return null;
  }

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
            <span className="font-medium text-gray-900">{safeToFixed(fundamentals.pe, 1)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Price/Book</span>
            <span className="font-medium text-gray-900">{safeToFixed(fundamentals.pb, 1)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Price/Sales</span>
            <span className="font-medium text-gray-900">{safeToFixed(fundamentals.ps, 1)}</span>
          </div>
        </div>
      </div>

      {/* Profitability */}
      <div className="mb-6">
        <h4 className="text-sm font-medium text-gray-700 mb-3 uppercase tracking-wide">Profitability</h4>
        <div className="space-y-3">
          <div className="flex justify-between">
            <span className="text-gray-600">ROE</span>
            <span className="font-medium text-gray-900">{formatPercent(fundamentals.roe)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">ROA</span>
            <span className="font-medium text-gray-900">{formatPercent(fundamentals.roa)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Gross Margin</span>
            <span className="font-medium text-gray-900">{formatPercent(fundamentals.grossMargin)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Net Margin</span>
            <span className="font-medium text-gray-900">{formatPercent(fundamentals.netMargin)}</span>
          </div>
        </div>
      </div>

      {/* Financial Health */}
      <div className="mb-6">
        <h4 className="text-sm font-medium text-gray-700 mb-3 uppercase tracking-wide">Financial Health</h4>
        <div className="space-y-3">
          <div className="flex justify-between">
            <span className="text-gray-600">Debt/Equity</span>
            <span className="font-medium text-gray-900">{safeToFixed(fundamentals.debtToEquity, 1)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Current Ratio</span>
            <span className="font-medium text-gray-900">{safeToFixed(fundamentals.currentRatio, 1)}</span>
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
            <span className="font-medium text-gray-900">{safeToFixed(keyMetrics.beta, 2)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Avg Volume</span>
            <span className="font-medium text-gray-900">{safeToFixed(safeParseNumber(keyMetrics.averageVolume) / 1000000, 1)}M</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Shares Out</span>
            <span className="font-medium text-gray-900">{safeToFixed(safeParseNumber(keyMetrics.sharesOutstanding) / 1000000000, 1)}B</span>
          </div>
        </div>
      </div>
    </div>
  );
}
