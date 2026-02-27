'use client';

import { useState } from 'react';
import type { EarningsResult } from '@/lib/types/earnings';

interface EarningsTableProps {
  earnings: EarningsResult[];
}

function formatEPS(value: number | null | undefined): string {
  if (value === null || value === undefined) return '-';
  return `$${value.toFixed(2)}`;
}

function formatRevenue(value: number | null | undefined): string {
  if (value === null || value === undefined) return '-';
  const abs = Math.abs(value);
  if (abs >= 1e12) return `$${(value / 1e12).toFixed(1)}T`;
  if (abs >= 1e9) return `$${(value / 1e9).toFixed(1)}B`;
  if (abs >= 1e6) return `$${(value / 1e6).toFixed(0)}M`;
  return `$${value.toLocaleString()}`;
}

function formatSurprise(value: number | null | undefined): string {
  if (value === null || value === undefined) return '-';
  const sign = value > 0 ? '+' : '';
  return `${sign}${value.toFixed(1)}%`;
}

function surpriseColor(value: number | null | undefined): string {
  if (value === null || value === undefined) return 'text-ic-text-dim';
  if (value > 0.5) return 'text-green-400';
  if (value < -0.5) return 'text-red-400';
  return 'text-ic-text-dim';
}

function beatIcon(beat: boolean | null): string {
  if (beat === null) return '';
  return beat ? ' ✓' : ' ✗';
}

function beatIconColor(beat: boolean | null): string {
  if (beat === null) return '';
  return beat ? 'text-green-400' : 'text-red-400';
}

const DEFAULT_VISIBLE = 8;
const MAX_VISIBLE = 40;

export default function EarningsTable({ earnings }: EarningsTableProps) {
  const [showAll, setShowAll] = useState(false);

  // Filter to past quarters only (not upcoming)
  const pastEarnings = earnings.filter((e) => !e.isUpcoming);
  const visibleEarnings = showAll
    ? pastEarnings.slice(0, MAX_VISIBLE)
    : pastEarnings.slice(0, DEFAULT_VISIBLE);

  if (pastEarnings.length === 0) {
    return (
      <div className="bg-ic-bg-secondary rounded-lg p-6 text-center">
        <p className="text-ic-text-muted">No earnings history available</p>
      </div>
    );
  }

  return (
    <div className="bg-ic-bg-secondary rounded-lg overflow-hidden">
      <div className="px-4 py-3 border-b border-ic-border/30">
        <h3 className="text-base font-semibold text-ic-text-primary">Earnings History</h3>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-ic-border/30">
              <th className="text-left text-ic-text-muted font-medium px-3 py-2">Quarter</th>
              <th className="text-left text-ic-text-muted font-medium px-3 py-2">Date</th>
              <th className="text-right text-ic-text-muted font-medium px-3 py-2">EPS Est</th>
              <th className="text-right text-ic-text-muted font-medium px-3 py-2">EPS Actual</th>
              <th className="text-right text-ic-text-muted font-medium px-3 py-2">EPS Surprise</th>
              <th className="text-right text-ic-text-muted font-medium px-3 py-2">Rev Est</th>
              <th className="text-right text-ic-text-muted font-medium px-3 py-2">Rev Actual</th>
              <th className="text-right text-ic-text-muted font-medium px-3 py-2">Rev Surprise</th>
            </tr>
          </thead>
          <tbody>
            {visibleEarnings.map((e) => (
              <tr key={e.date} className="border-t border-ic-border/20 hover:bg-ic-bg-tertiary/30">
                <td className="px-3 py-2.5 text-ic-text-primary font-medium whitespace-nowrap">
                  {e.fiscalQuarter}
                </td>
                <td className="px-3 py-2.5 text-ic-text-muted whitespace-nowrap">{e.date}</td>
                <td className="px-3 py-2.5 text-right text-ic-text-muted">
                  {formatEPS(e.epsEstimated)}
                </td>
                <td className="px-3 py-2.5 text-right text-ic-text-primary">
                  {formatEPS(e.epsActual)}
                </td>
                <td
                  className={`px-3 py-2.5 text-right font-medium ${surpriseColor(e.epsSurprisePercent)}`}
                >
                  {formatSurprise(e.epsSurprisePercent)}
                  <span className={beatIconColor(e.epsBeat)}>{beatIcon(e.epsBeat)}</span>
                </td>
                <td className="px-3 py-2.5 text-right text-ic-text-muted">
                  {e.revenueEstimated !== null ? formatRevenue(e.revenueEstimated) : 'N/A'}
                </td>
                <td className="px-3 py-2.5 text-right text-ic-text-primary">
                  {formatRevenue(e.revenueActual)}
                </td>
                <td
                  className={`px-3 py-2.5 text-right font-medium ${surpriseColor(e.revenueSurprisePercent)}`}
                >
                  {formatSurprise(e.revenueSurprisePercent)}
                  <span className={beatIconColor(e.revenueBeat)}>{beatIcon(e.revenueBeat)}</span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {pastEarnings.length > DEFAULT_VISIBLE && !showAll && (
        <div className="px-4 py-3 border-t border-ic-border/30">
          <button
            onClick={() => setShowAll(true)}
            className="text-sm text-ic-blue hover:text-ic-blue/80 font-medium"
          >
            Show more ({pastEarnings.length - DEFAULT_VISIBLE} more quarters)
          </button>
        </div>
      )}
    </div>
  );
}
