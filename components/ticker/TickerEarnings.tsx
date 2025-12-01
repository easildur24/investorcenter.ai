'use client';

import { useState, useEffect } from 'react';
import { ChartBarIcon } from '@heroicons/react/24/outline';
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

  const formatCurrency = (amount: number | string) => {
    const num = safeParseNumber(amount);
    return formatLargeNumber(num);
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
        <ChartBarIcon className="h-6 w-6 text-ic-text-dim mr-2" />
        <h3 className="text-lg font-semibold text-ic-text-primary">Earnings History</h3>
      </div>

      {loading ? (
        <div className="space-y-4">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="flex justify-between items-center animate-pulse">
              <div className="h-4 bg-ic-border rounded w-20"></div>
              <div className="h-4 bg-ic-border rounded w-16"></div>
              <div className="h-4 bg-ic-border rounded w-16"></div>
              <div className="h-4 bg-ic-border rounded w-16"></div>
            </div>
          ))}
        </div>
      ) : earnings.length > 0 ? (
        <div className="overflow-x-auto -mx-6 px-6">
          {/* Table Header */}
          <div className="grid grid-cols-5 gap-2 pb-2 mb-3 border-b border-ic-border text-xs font-medium text-ic-text-muted min-w-[320px]">
            <div>Period</div>
            <div className="text-right">EPS</div>
            <div className="text-right">Est.</div>
            <div className="text-right">Surp.</div>
            <div className="text-right">Rev.</div>
          </div>

          {/* Earnings Data */}
          <div className="space-y-2 min-w-[320px]">
            {earnings.slice(0, 8).map((earning) => (
              <div key={`${earning.year}-${earning.quarter}`} className="grid grid-cols-5 gap-2 items-center text-xs">
                <div>
                  <div className="font-medium text-ic-text-primary">
                    {earning.quarter} {earning.year}
                  </div>
                  <div className="text-[10px] text-ic-text-muted">
                    {formatDate(earning.reportDate)}
                  </div>
                </div>

                <div className="text-right font-medium">
                  ${safeToFixed(earning.epsActual, 2)}
                </div>

                <div className="text-right text-ic-text-muted">
                  ${safeToFixed(earning.epsEstimate, 2)}
                </div>

                <div className="text-right">
                  <span className={safeParseNumber(earning.epsSurprise) > 0 ? 'text-ic-positive' : 'text-ic-negative'}>
                    {safeParseNumber(earning.epsSurprise) > 0 ? '✓' : '✗'} {safeToFixed(earning.epsSurprisePercent, 1)}%
                  </span>
                </div>

                <div className="text-right font-medium">
                  {formatLargeNumber(earning.revenueActual)}
                </div>
              </div>
            ))}
          </div>

          {/* Summary Stats */}
          <div className="mt-6 pt-4 border-t border-ic-border">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <div className="text-ic-text-muted">Avg EPS Surprise</div>
                <div className="font-medium">
                  {earnings.length > 0
                    ? `${safeToFixed((earnings.slice(0, 4).reduce((sum, e) => sum + safeParseNumber(e.epsSurprisePercent), 0) / Math.min(4, earnings.length)), 1)}%`
                    : 'N/A'
                  }
                </div>
              </div>
              <div>
                <div className="text-ic-text-muted">Beat Rate</div>
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
          <ChartBarIcon className="h-12 w-12 text-ic-text-dim mx-auto mb-4" />
          <p className="text-ic-text-muted">No earnings data available</p>
        </div>
      )}
    </div>
  );
}
