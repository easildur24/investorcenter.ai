'use client';

import { useState, useEffect } from 'react';
import { safeToFixed, safeParseNumber, formatLargeNumber } from '@/lib/utils';

interface EarningsHistoryProps {
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

export default function EarningsHistory({ symbol }: EarningsHistoryProps) {
  const [earnings, setEarnings] = useState<Earnings[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchEarnings = async () => {
      try {
        setLoading(true);
        const response = await fetch(`/api/v1/tickers/${symbol}/earnings`);
        const result = await response.json();
        setEarnings(result.data || []);
      } catch (error) {
        console.error('Error fetching earnings:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchEarnings();
  }, [symbol]);

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    });
  };

  // Calculate beat rate from last 4 quarters
  const lastFourQuarters = earnings.slice(0, 4);
  const beatsCount = lastFourQuarters.filter(
    (e) => safeParseNumber(e.epsSurprise) > 0
  ).length;
  const beatRate = lastFourQuarters.length > 0
    ? Math.round((beatsCount / lastFourQuarters.length) * 100)
    : 0;

  if (loading) {
    return (
      <div className="bg-ic-surface border border-ic-border rounded-2xl p-6 shadow-[var(--ic-shadow-card)]">
        <div className="flex justify-between items-center mb-5">
          <div className="h-6 bg-ic-border rounded w-40 animate-pulse"></div>
          <div className="h-6 bg-ic-border rounded w-32 animate-pulse"></div>
        </div>
        <div className="space-y-4">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="flex justify-between items-center animate-pulse">
              <div className="h-4 bg-ic-border rounded w-20"></div>
              <div className="h-4 bg-ic-border rounded w-16"></div>
              <div className="h-4 bg-ic-border rounded w-16"></div>
              <div className="h-4 bg-ic-border rounded w-16"></div>
              <div className="h-4 bg-ic-border rounded w-16"></div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (earnings.length === 0) {
    return (
      <div className="bg-ic-surface border border-ic-border rounded-2xl p-6 shadow-[var(--ic-shadow-card)]">
        <div className="flex justify-between items-center mb-5">
          <h2 className="text-xl font-semibold text-ic-text-primary">Earnings History</h2>
        </div>
        <div className="text-center py-8">
          <p className="text-ic-text-muted">No earnings data available</p>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-ic-surface border border-ic-border rounded-2xl p-6 shadow-[var(--ic-shadow-card)]">
      {/* Header */}
      <div className="flex justify-between items-center mb-5">
        <h2 className="text-xl font-semibold text-ic-text-primary">Earnings History</h2>

        {/* Beat Rate with Visual Indicator */}
        <div className="flex items-center gap-3">
          <span className="text-ic-text-muted text-sm">Beat Rate</span>
          <span className="text-ic-positive text-2xl font-bold">{beatRate}%</span>
          <div className="flex gap-1">
            {lastFourQuarters.map((q, i) => (
              <div
                key={i}
                className={`w-3 h-3 rounded-sm ${
                  safeParseNumber(q.epsSurprise) > 0
                    ? 'bg-ic-positive'
                    : 'bg-ic-negative'
                }`}
              />
            ))}
          </div>
        </div>
      </div>

      {/* Table */}
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="border-b border-ic-border">
              <th className="text-left text-ic-text-muted text-sm font-medium pb-3">Period</th>
              <th className="text-right text-ic-text-muted text-sm font-medium pb-3">EPS</th>
              <th className="text-right text-ic-text-muted text-sm font-medium pb-3">Est.</th>
              <th className="text-right text-ic-text-muted text-sm font-medium pb-3">Surprise</th>
              <th className="text-right text-ic-text-muted text-sm font-medium pb-3">Revenue</th>
            </tr>
          </thead>
          <tbody>
            {earnings.slice(0, 8).map((earning) => {
              const isBeat = safeParseNumber(earning.epsSurprise) > 0;
              const surprisePercent = safeParseNumber(earning.epsSurprisePercent);

              return (
                <tr
                  key={`${earning.year}-${earning.quarter}`}
                  className="border-b border-ic-border-subtle last:border-b-0"
                >
                  <td className="py-3.5 pr-4">
                    <div className="text-ic-text-primary font-medium">
                      {earning.quarter} {earning.year}
                    </div>
                    <div className="text-ic-text-dim text-xs mt-0.5">
                      {formatDate(earning.reportDate)}
                    </div>
                  </td>
                  <td className="py-3.5 text-right font-mono text-ic-text-primary font-semibold">
                    ${safeToFixed(earning.epsActual, 2)}
                  </td>
                  <td className="py-3.5 text-right font-mono text-ic-text-muted">
                    ${safeToFixed(earning.epsEstimate, 2)}
                  </td>
                  <td className="py-3.5 text-right">
                    <span
                      className={`font-medium ${
                        isBeat ? 'text-ic-positive' : 'text-ic-negative'
                      }`}
                    >
                      {isBeat ? '✓' : '✗'}{' '}
                      {surprisePercent >= 0 ? '+' : ''}
                      {safeToFixed(surprisePercent, 1)}%
                    </span>
                  </td>
                  <td className="py-3.5 text-right font-mono text-ic-text-secondary">
                    {formatLargeNumber(earning.revenueActual)}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
