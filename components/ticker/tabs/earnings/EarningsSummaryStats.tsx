'use client';

import type { BeatRate, EarningsResult } from '@/lib/types/earnings';
import { formatSurprise, surpriseColor } from '@/lib/utils/earningsFormatters';

interface EarningsSummaryStatsProps {
  beatRate: BeatRate | null;
  earnings: EarningsResult[];
}

function computeAvgSurprise(
  earnings: EarningsResult[],
  field: 'epsSurprisePercent' | 'revenueSurprisePercent'
): number | null {
  const values = earnings.map((e) => e[field]).filter((v): v is number => v !== null);
  if (values.length === 0) return null;
  return values.reduce((sum, v) => sum + v, 0) / values.length;
}

function percentColor(value: number | null): string {
  if (value === null) return 'text-ic-text-primary';
  if (value > 0) return 'text-ic-positive';
  if (value < 0) return 'text-ic-negative';
  return 'text-ic-text-primary';
}

export default function EarningsSummaryStats({ beatRate, earnings }: EarningsSummaryStatsProps) {
  if (!beatRate) return null;

  const avgEpsSurprise = computeAvgSurprise(earnings, 'epsSurprisePercent');
  const avgRevSurprise = computeAvgSurprise(earnings, 'revenueSurprisePercent');

  const stats = [
    {
      label: 'EPS Beat Rate',
      value: `${beatRate.epsBeats} / ${beatRate.totalQuarters}`,
      sub:
        beatRate.totalQuarters > 0
          ? `${Math.round((beatRate.epsBeats / beatRate.totalQuarters) * 100)}%`
          : 'N/A',
    },
    {
      label: 'Revenue Beat Rate',
      value: `${beatRate.revenueBeats} / ${beatRate.totalRevenueQuarters}`,
      sub:
        beatRate.totalRevenueQuarters > 0
          ? `${Math.round((beatRate.revenueBeats / beatRate.totalRevenueQuarters) * 100)}%`
          : 'N/A',
    },
    {
      label: 'Avg EPS Surprise',
      value: formatSurprise(avgEpsSurprise, 'N/A'),
      colorClass: percentColor(avgEpsSurprise),
    },
    {
      label: 'Avg Revenue Surprise',
      value: formatSurprise(avgRevSurprise, 'N/A'),
      colorClass: percentColor(avgRevSurprise),
    },
  ];

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
      {stats.map((stat) => (
        <div key={stat.label} className="bg-ic-bg-secondary rounded-lg p-4">
          <p className="text-xs text-ic-text-muted mb-1">{stat.label}</p>
          <p className={`text-lg font-semibold ${stat.colorClass || 'text-ic-text-primary'}`}>
            {stat.value}
          </p>
          {stat.sub && <p className="text-xs text-ic-text-dim mt-0.5">{stat.sub}</p>}
        </div>
      ))}
    </div>
  );
}
