'use client';

import type { EarningsResult } from '@/lib/types/earnings';
import { formatEPS, formatRevenue, parseDateLocal } from '@/lib/utils/earningsFormatters';

interface NextEarningsCardProps {
  nextEarnings: EarningsResult | null;
  mostRecentEarnings: EarningsResult | null;
}

export default function NextEarningsCard({
  nextEarnings,
  mostRecentEarnings,
}: NextEarningsCardProps) {
  // Prefer upcoming earnings; fall back to most recent past record
  const data = nextEarnings ?? mostRecentEarnings;
  if (!data) {
    return (
      <div className="bg-ic-bg-secondary rounded-lg p-6">
        <p className="text-ic-text-muted text-sm">Earnings date not available</p>
      </div>
    );
  }

  const earningsDate = parseDateLocal(data.date);
  const now = new Date();
  const daysUntil = Math.ceil((earningsDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));

  const dateLabel = earningsDate.toLocaleDateString('en-US', {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });

  return (
    <div className="bg-ic-bg-secondary rounded-lg p-6">
      <div className="flex items-start justify-between flex-wrap gap-4">
        <div>
          <div className="flex items-center gap-3 mb-1">
            <h3 className="text-base font-semibold text-ic-text-primary">
              {data.isUpcoming ? 'Next Earnings' : 'Last Reported'}
            </h3>
            {data.isUpcoming ? (
              <span className="bg-yellow-500/20 text-yellow-400 text-xs px-2 py-0.5 rounded font-medium">
                Estimated
              </span>
            ) : (
              <span className="bg-green-500/20 text-green-400 text-xs px-2 py-0.5 rounded font-medium">
                Reported
              </span>
            )}
          </div>
          <p className="text-lg font-medium text-ic-text-primary">{dateLabel}</p>
          {data.isUpcoming && daysUntil > 0 && (
            <p className="text-sm text-ic-blue font-medium mt-1">
              in {daysUntil} {daysUntil === 1 ? 'day' : 'days'}
            </p>
          )}
        </div>

        <div className="flex gap-8">
          <div>
            <p className="text-xs text-ic-text-dim uppercase tracking-wide mb-1">
              EPS {data.isUpcoming ? 'Estimate' : 'Actual'}
            </p>
            <p className="text-lg font-semibold text-ic-text-primary">
              {data.isUpcoming
                ? formatEPS(data.epsEstimated, 'N/A')
                : formatEPS(data.epsActual, 'N/A')}
            </p>
          </div>
          <div>
            <p className="text-xs text-ic-text-dim uppercase tracking-wide mb-1">
              Revenue {data.isUpcoming ? 'Estimate' : 'Actual'}
            </p>
            <p className="text-lg font-semibold text-ic-text-primary">
              {data.isUpcoming
                ? formatRevenue(data.revenueEstimated, 'N/A')
                : formatRevenue(data.revenueActual, 'N/A')}
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
