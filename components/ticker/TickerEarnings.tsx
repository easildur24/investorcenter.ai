'use client';

import { useState, useEffect } from 'react';
import { ChartBarIcon, CheckCircleIcon, XCircleIcon } from '@heroicons/react/24/outline';
import { safeToFixed, safeParseNumber, formatLargeNumber } from '@/lib/utils';

interface TickerEarningsProps {
  symbol: string;
}

interface Earnings {
  quarter: string;
  year: number;
  reportDate: string;
  epsActual: number | string;
  epsEstimate: number | string;
  epsSurprise: number | string;
  epsSurprisePercent: number | string;
  revenueActual: number | string;
  revenueEstimate: number | string;
  revenueSurprise: number | string;
}

export default function TickerEarnings({ symbol }: TickerEarningsProps) {
  const [earnings, setEarnings] = useState<Earnings[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchEarnings = async () => {
      try {
        setLoading(true);
        const response = await fetch(`/api/v1/tickers/${symbol}/earnings`);
        const result = await response.json();
        setEarnings(result.data);
      } catch (error) {
        console.error('Error fetching earnings:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchEarnings();
  }, [symbol]);

  const formatCurrency = (amount: number) => {
    if (amount >= 1000000000) {
      return `$${(amount / 1000000000).toFixed(1)}B`;
    }
    if (amount >= 1000000) {
      return `$${(amount / 1000000).toFixed(1)}M`;
    }
    return `$${amount.toFixed(2)}`;
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric'
    });
  };

  return (
    <div className="p-6">
      <div className="flex items-center mb-6">
        <ChartBarIcon className="h-6 w-6 text-gray-400 mr-2" />
        <h3 className="text-lg font-semibold text-gray-900">Earnings History</h3>
      </div>

      {loading ? (
        <div className="space-y-4">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="flex justify-between items-center animate-pulse">
              <div className="h-4 bg-gray-200 rounded w-20"></div>
              <div className="h-4 bg-gray-200 rounded w-16"></div>
              <div className="h-4 bg-gray-200 rounded w-16"></div>
              <div className="h-4 bg-gray-200 rounded w-16"></div>
            </div>
          ))}
        </div>
      ) : earnings.length > 0 ? (
        <div className="overflow-hidden">
          {/* Table Header */}
          <div className="grid grid-cols-5 gap-4 pb-3 mb-4 border-b border-gray-200 text-sm font-medium text-gray-500">
            <div>Period</div>
            <div className="text-right">EPS</div>
            <div className="text-right">Est.</div>
            <div className="text-right">Surprise</div>
            <div className="text-right">Revenue</div>
          </div>

          {/* Earnings Data */}
          <div className="space-y-3">
            {earnings.slice(0, 8).map((earning, index) => (
              <div key={`${earning.year}-${earning.quarter}`} className="grid grid-cols-5 gap-4 items-center text-sm">
                <div>
                  <div className="font-medium text-gray-900">
                    {earning.quarter} {earning.year}
                  </div>
                  <div className="text-xs text-gray-500">
                    {formatDate(earning.reportDate)}
                  </div>
                </div>
                
                <div className="text-right font-medium">
                  ${safeToFixed(earning.epsActual, 2)}
                </div>
                
                <div className="text-right text-gray-600">
                  ${safeToFixed(earning.epsEstimate, 2)}
                </div>
                
                <div className="text-right">
                  <div className="flex items-center justify-end">
                    {safeParseNumber(earning.epsSurprise) > 0 ? (
                      <CheckCircleIcon className="h-4 w-4 text-green-500 mr-1" />
                    ) : (
                      <XCircleIcon className="h-4 w-4 text-red-500 mr-1" />
                    )}
                    <span className={safeParseNumber(earning.epsSurprise) > 0 ? 'text-green-600' : 'text-red-600'}>
                      {safeToFixed(earning.epsSurprisePercent, 1)}%
                    </span>
                  </div>
                </div>
                
                <div className="text-right font-medium">
                  {formatLargeNumber(earning.revenueActual)}
                </div>
              </div>
            ))}
          </div>

          {/* Summary Stats */}
          <div className="mt-6 pt-4 border-t border-gray-200">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <div className="text-gray-500">Avg EPS Surprise</div>
                <div className="font-medium">
                  {earnings.length > 0 
                    ? `${(earnings.slice(0, 4).reduce((sum, e) => sum + safeParseNumber(e.epsSurprisePercent), 0) / Math.min(4, earnings.length)).toFixed(1)}%`
                    : 'N/A'
                  }
                </div>
              </div>
              <div>
                <div className="text-gray-500">Beat Rate</div>
                <div className="font-medium">
                  {earnings.length > 0 
                    ? `${Math.round((earnings.slice(0, 8).filter(e => safeParseNumber(e.epsSurprise) > 0).length / Math.min(8, earnings.length)) * 100)}%`
                    : 'N/A'
                  }
                </div>
              </div>
            </div>
          </div>
        </div>
      ) : (
        <div className="text-center py-8">
          <ChartBarIcon className="h-12 w-12 text-gray-300 mx-auto mb-4" />
          <p className="text-gray-500">No earnings data available</p>
        </div>
      )}
    </div>
  );
}
