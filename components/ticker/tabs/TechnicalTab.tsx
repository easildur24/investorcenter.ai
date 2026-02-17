'use client';

import { useState, useEffect } from 'react';
import { cn, safeToFixed, safeParseNumber } from '@/lib/utils';

interface TechnicalTabProps {
  symbol: string;
}

interface TechnicalIndicators {
  ticker?: string;
  calculation_date?: string;
  current_price?: number;
  sma_50?: number;
  sma_200?: number;
  ema_12?: number;
  ema_26?: number;
  rsi_14?: number;
  macd?: number;
  macd_signal?: number;
  macd_histogram?: number;
  bollinger_upper?: number;
  bollinger_middle?: number;
  bollinger_lower?: number;
  volume_ma_20?: number;
  return_1m?: number;
  return_3m?: number;
  return_6m?: number;
  return_12m?: number;
}

// Helper to get signal based on indicator value
function getRSISignal(rsi: number): { label: string; color: string } {
  if (rsi >= 70) return { label: 'Overbought', color: 'text-ic-negative bg-ic-negative-bg' };
  if (rsi <= 30) return { label: 'Oversold', color: 'text-ic-positive bg-ic-positive-bg' };
  return { label: 'Neutral', color: 'text-ic-text-muted bg-ic-bg-secondary' };
}

function getMACDSignal(histogram: number): { label: string; color: string } {
  if (histogram > 0) return { label: 'Bullish', color: 'text-ic-positive bg-ic-positive-bg' };
  if (histogram < 0) return { label: 'Bearish', color: 'text-ic-negative bg-ic-negative-bg' };
  return { label: 'Neutral', color: 'text-ic-text-muted bg-ic-bg-secondary' };
}

function getSMASignal(price: number, sma: number): { label: string; color: string } {
  if (price > sma) return { label: 'Above', color: 'text-ic-positive bg-ic-positive-bg' };
  if (price < sma) return { label: 'Below', color: 'text-ic-negative bg-ic-negative-bg' };
  return { label: 'At', color: 'text-ic-text-muted bg-ic-bg-secondary' };
}

export default function TechnicalTab({ symbol }: TechnicalTabProps) {
  const [data, setData] = useState<TechnicalIndicators | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        const response = await fetch(`/api/v1/stocks/${symbol}/technical`);
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`);
        }
        const result = await response.json();
        setData(result.data || {});
      } catch (err) {
        console.error('Error fetching technical data:', err);
        setError('Failed to load technical indicators');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [symbol]);

  if (loading) {
    return (
      <div className="p-6 animate-pulse">
        <div className="h-6 bg-ic-bg-secondary rounded w-48 mb-6"></div>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {[1, 2, 3, 4, 5, 6, 7, 8].map((i) => (
            <div key={i} className="bg-ic-bg-secondary rounded-lg p-4">
              <div className="h-4 bg-ic-bg-secondary rounded w-20 mb-2"></div>
              <div className="h-6 bg-ic-bg-secondary rounded w-16"></div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="p-6">
        <h3 className="text-lg font-semibold text-ic-text-primary mb-4">Technical Indicators</h3>
        <p className="text-ic-text-muted">{error || 'No technical data available'}</p>
      </div>
    );
  }

  const currentPrice = data.current_price || 0;

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-lg font-semibold text-ic-text-primary">Technical Indicators</h3>
        {data.calculation_date && (
          <span className="text-sm text-ic-text-muted">
            Updated: {new Date(data.calculation_date).toLocaleDateString()}
          </span>
        )}
      </div>

      {/* Momentum Indicators */}
      <div className="mb-8">
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Momentum
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {/* RSI */}
          <div className="bg-ic-bg-secondary rounded-lg p-4">
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm text-ic-text-muted">RSI (14)</span>
              {data.rsi_14 !== undefined && (
                <span
                  className={cn(
                    'text-xs px-2 py-0.5 rounded-full',
                    getRSISignal(data.rsi_14).color
                  )}
                >
                  {getRSISignal(data.rsi_14).label}
                </span>
              )}
            </div>
            <div className="text-xl font-semibold text-ic-text-primary">
              {safeToFixed(data.rsi_14, 1)}
            </div>
            {data.rsi_14 !== undefined && (
              <div className="mt-2 h-2 bg-ic-bg-secondary rounded-full overflow-hidden">
                <div
                  className={cn(
                    'h-full rounded-full transition-all',
                    data.rsi_14 >= 70
                      ? 'bg-ic-negative'
                      : data.rsi_14 <= 30
                        ? 'bg-ic-positive'
                        : 'bg-ic-blue'
                  )}
                  style={{ width: `${Math.min(data.rsi_14, 100)}%` }}
                />
              </div>
            )}
          </div>

          {/* Current Price */}
          <div className="bg-ic-bg-secondary rounded-lg p-4">
            <div className="text-sm text-ic-text-muted mb-2">Current Price</div>
            <div className="text-xl font-semibold text-ic-text-primary">
              ${safeToFixed(data.current_price, 2)}
            </div>
          </div>

          {/* 1M Return */}
          <div className="bg-ic-bg-secondary rounded-lg p-4">
            <div className="text-sm text-ic-text-muted mb-2">1M Return</div>
            <div
              className={cn(
                'text-xl font-semibold',
                safeParseNumber(data.return_1m) >= 0 ? 'text-ic-positive' : 'text-ic-negative'
              )}
            >
              {safeParseNumber(data.return_1m) >= 0 ? '+' : ''}
              {safeToFixed(data.return_1m, 2)}%
            </div>
          </div>

          {/* 3M Return */}
          <div className="bg-ic-bg-secondary rounded-lg p-4">
            <div className="text-sm text-ic-text-muted mb-2">3M Return</div>
            <div
              className={cn(
                'text-xl font-semibold',
                safeParseNumber(data.return_3m) >= 0 ? 'text-ic-positive' : 'text-ic-negative'
              )}
            >
              {safeParseNumber(data.return_3m) >= 0 ? '+' : ''}
              {safeToFixed(data.return_3m, 2)}%
            </div>
          </div>
        </div>
      </div>

      {/* MACD */}
      <div className="mb-8">
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          MACD
        </h4>
        <div className="bg-ic-bg-secondary rounded-lg p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm text-ic-text-muted">MACD Indicator</span>
            {data.macd_histogram !== undefined && (
              <span
                className={cn(
                  'text-xs px-2 py-0.5 rounded-full',
                  getMACDSignal(data.macd_histogram).color
                )}
              >
                {getMACDSignal(data.macd_histogram).label}
              </span>
            )}
          </div>
          <div className="grid grid-cols-3 gap-4 mt-2">
            <div>
              <div className="text-xs text-ic-text-muted">MACD Line</div>
              <div className="text-lg font-semibold text-ic-text-primary">
                {safeToFixed(data.macd, 3)}
              </div>
            </div>
            <div>
              <div className="text-xs text-ic-text-muted">Signal Line</div>
              <div className="text-lg font-semibold text-ic-text-primary">
                {safeToFixed(data.macd_signal, 3)}
              </div>
            </div>
            <div>
              <div className="text-xs text-ic-text-muted">Histogram</div>
              <div
                className={cn(
                  'text-lg font-semibold',
                  safeParseNumber(data.macd_histogram) >= 0
                    ? 'text-ic-positive'
                    : 'text-ic-negative'
                )}
              >
                {safeToFixed(data.macd_histogram, 3)}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Moving Averages */}
      <div className="mb-8">
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Moving Averages
        </h4>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="text-left text-sm text-ic-text-muted border-b">
                <th className="pb-2 font-medium">Indicator</th>
                <th className="pb-2 font-medium text-right">Value</th>
                <th className="pb-2 font-medium text-right">Signal</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-ic-border">
              <tr>
                <td className="py-3 text-ic-text-primary">SMA (50)</td>
                <td className="py-3 text-right font-medium text-ic-text-primary">
                  ${safeToFixed(data.sma_50, 2)}
                </td>
                <td className="py-3 text-right">
                  {data.sma_50 && currentPrice > 0 && (
                    <span
                      className={cn(
                        'text-xs px-2 py-0.5 rounded-full',
                        getSMASignal(currentPrice, data.sma_50).color
                      )}
                    >
                      {getSMASignal(currentPrice, data.sma_50).label}
                    </span>
                  )}
                </td>
              </tr>
              <tr>
                <td className="py-3 text-ic-text-primary">SMA (200)</td>
                <td className="py-3 text-right font-medium text-ic-text-primary">
                  ${safeToFixed(data.sma_200, 2)}
                </td>
                <td className="py-3 text-right">
                  {data.sma_200 && currentPrice > 0 && (
                    <span
                      className={cn(
                        'text-xs px-2 py-0.5 rounded-full',
                        getSMASignal(currentPrice, data.sma_200).color
                      )}
                    >
                      {getSMASignal(currentPrice, data.sma_200).label}
                    </span>
                  )}
                </td>
              </tr>
              <tr>
                <td className="py-3 text-ic-text-primary">EMA (12)</td>
                <td className="py-3 text-right font-medium text-ic-text-primary">
                  ${safeToFixed(data.ema_12, 2)}
                </td>
                <td className="py-3 text-right">
                  {data.ema_12 && currentPrice > 0 && (
                    <span
                      className={cn(
                        'text-xs px-2 py-0.5 rounded-full',
                        getSMASignal(currentPrice, data.ema_12).color
                      )}
                    >
                      {getSMASignal(currentPrice, data.ema_12).label}
                    </span>
                  )}
                </td>
              </tr>
              <tr>
                <td className="py-3 text-ic-text-primary">EMA (26)</td>
                <td className="py-3 text-right font-medium text-ic-text-primary">
                  ${safeToFixed(data.ema_26, 2)}
                </td>
                <td className="py-3 text-right">
                  {data.ema_26 && currentPrice > 0 && (
                    <span
                      className={cn(
                        'text-xs px-2 py-0.5 rounded-full',
                        getSMASignal(currentPrice, data.ema_26).color
                      )}
                    >
                      {getSMASignal(currentPrice, data.ema_26).label}
                    </span>
                  )}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      {/* Bollinger Bands */}
      <div className="mb-8">
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Bollinger Bands (20, 2)
        </h4>
        <div className="grid grid-cols-3 gap-4">
          <div className="bg-ic-bg-secondary rounded-lg p-4 text-center">
            <div className="text-sm text-ic-text-muted mb-1">Upper Band</div>
            <div className="text-lg font-semibold text-ic-text-primary">
              ${safeToFixed(data.bollinger_upper, 2)}
            </div>
          </div>
          <div className="bg-ic-blue-bg rounded-lg p-4 text-center">
            <div className="text-sm text-ic-blue mb-1">Middle (SMA)</div>
            <div className="text-lg font-semibold text-ic-blue">
              ${safeToFixed(data.bollinger_middle, 2)}
            </div>
          </div>
          <div className="bg-ic-bg-secondary rounded-lg p-4 text-center">
            <div className="text-sm text-ic-text-muted mb-1">Lower Band</div>
            <div className="text-lg font-semibold text-ic-text-primary">
              ${safeToFixed(data.bollinger_lower, 2)}
            </div>
          </div>
        </div>
      </div>

      {/* Returns Overview */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Price Performance
        </h4>
        <div className="grid grid-cols-4 gap-4">
          <div className="bg-ic-bg-secondary rounded-lg p-4 text-center">
            <div className="text-sm text-ic-text-muted mb-1">1 Month</div>
            <div
              className={cn(
                'text-lg font-semibold',
                safeParseNumber(data.return_1m) >= 0 ? 'text-ic-positive' : 'text-ic-negative'
              )}
            >
              {safeParseNumber(data.return_1m) >= 0 ? '+' : ''}
              {safeToFixed(data.return_1m, 1)}%
            </div>
          </div>
          <div className="bg-ic-bg-secondary rounded-lg p-4 text-center">
            <div className="text-sm text-ic-text-muted mb-1">3 Months</div>
            <div
              className={cn(
                'text-lg font-semibold',
                safeParseNumber(data.return_3m) >= 0 ? 'text-ic-positive' : 'text-ic-negative'
              )}
            >
              {safeParseNumber(data.return_3m) >= 0 ? '+' : ''}
              {safeToFixed(data.return_3m, 1)}%
            </div>
          </div>
          <div className="bg-ic-bg-secondary rounded-lg p-4 text-center">
            <div className="text-sm text-ic-text-muted mb-1">6 Months</div>
            <div
              className={cn(
                'text-lg font-semibold',
                safeParseNumber(data.return_6m) >= 0 ? 'text-ic-positive' : 'text-ic-negative'
              )}
            >
              {safeParseNumber(data.return_6m) >= 0 ? '+' : ''}
              {safeToFixed(data.return_6m, 1)}%
            </div>
          </div>
          <div className="bg-ic-bg-secondary rounded-lg p-4 text-center">
            <div className="text-sm text-ic-text-muted mb-1">12 Months</div>
            <div
              className={cn(
                'text-lg font-semibold',
                safeParseNumber(data.return_12m) >= 0 ? 'text-ic-positive' : 'text-ic-negative'
              )}
            >
              {safeParseNumber(data.return_12m) >= 0 ? '+' : ''}
              {safeToFixed(data.return_12m, 1)}%
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
