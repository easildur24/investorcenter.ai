'use client';

import { useState } from 'react';
import { GranularConfidence, FactorDataStatus, FACTOR_CONFIGS } from '@/lib/types/ic-score-v2';
import { ConfidenceBadge } from './Badges';

interface GranularConfidenceDisplayProps {
  confidence: GranularConfidence | undefined;
  expanded?: boolean;
}

/**
 * GranularConfidenceDisplay - Shows detailed data availability per factor
 *
 * Displays:
 * - Overall confidence level and percentage
 * - Per-factor data status (available, freshness)
 * - Warnings about missing or stale data
 */
export default function GranularConfidenceDisplay({
  confidence,
  expanded = false,
}: GranularConfidenceDisplayProps) {
  const [isExpanded, setIsExpanded] = useState(expanded);

  if (!confidence) {
    return null;
  }

  const { level, percentage, factors, warnings } = confidence;

  return (
    <div className="border-t border-gray-200 p-4">
      <div
        className="flex items-center justify-between cursor-pointer"
        onClick={() => setIsExpanded(!isExpanded)}
      >
        <div className="flex items-center gap-3">
          <h3 className="font-medium text-gray-900">Data Confidence</h3>
          <ConfidenceBadge level={level} percentage={percentage} />
        </div>
        <button className="text-gray-400 hover:text-gray-600">{isExpanded ? '▲' : '▼'}</button>
      </div>

      {isExpanded && (
        <div className="mt-4 space-y-4">
          {/* Progress bar */}
          <div>
            <div className="flex justify-between text-sm mb-1">
              <span className="text-gray-500">Data Completeness</span>
              <span className="font-medium">{Math.round(percentage)}%</span>
            </div>
            <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
              <div
                className={`h-full rounded-full transition-all ${
                  percentage >= 90
                    ? 'bg-green-500'
                    : percentage >= 70
                      ? 'bg-yellow-500'
                      : 'bg-red-500'
                }`}
                style={{ width: `${percentage}%` }}
              />
            </div>
          </div>

          {/* Factor status grid */}
          <div className="grid grid-cols-2 gap-2">
            {Object.entries(factors).map(([factorName, status]) => (
              <FactorStatusRow key={factorName} factorName={factorName} status={status} />
            ))}
          </div>

          {/* Warnings */}
          {warnings.length > 0 && (
            <div className="mt-3 p-3 bg-yellow-50 border border-yellow-100 rounded-lg">
              <h4 className="text-sm font-medium text-yellow-800 mb-2">Notes</h4>
              <ul className="space-y-1">
                {warnings.map((warning, index) => (
                  <li key={index} className="flex items-start gap-2 text-sm text-yellow-700">
                    <span className="text-yellow-500 flex-shrink-0">⚠</span>
                    <span>{warning}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

interface FactorStatusRowProps {
  factorName: string;
  status: FactorDataStatus;
}

function FactorStatusRow({ factorName, status }: FactorStatusRowProps) {
  const factorConfig = FACTOR_CONFIGS.find((f) => f.name === factorName);
  const displayName = factorConfig?.display_name || formatFactorName(factorName);

  const freshnessConfig = getFreshnessConfig(status.freshness);

  return (
    <div className="flex items-center justify-between p-2 bg-gray-50 rounded text-sm">
      <span className="text-gray-700">{displayName}</span>
      <div className="flex items-center gap-2">
        {status.available ? (
          <>
            <span className={`px-1.5 py-0.5 rounded text-xs ${freshnessConfig.className}`}>
              {freshnessConfig.label}
            </span>
            {status.freshness_days !== undefined && status.freshness_days > 0 && (
              <span className="text-gray-400 text-xs">{status.freshness_days}d</span>
            )}
          </>
        ) : (
          <span className="text-xs text-gray-400">{status.reason || 'Missing'}</span>
        )}
      </div>
    </div>
  );
}

interface FreshnessConfig {
  label: string;
  className: string;
}

function getFreshnessConfig(freshness: string): FreshnessConfig {
  switch (freshness) {
    case 'fresh':
      return { label: '✓ Fresh', className: 'bg-green-100 text-green-700' };
    case 'recent':
      return { label: 'Recent', className: 'bg-blue-100 text-blue-700' };
    case 'stale':
      return { label: 'Stale', className: 'bg-yellow-100 text-yellow-700' };
    case 'missing':
      return { label: 'Missing', className: 'bg-gray-100 text-gray-500' };
    default:
      return { label: freshness, className: 'bg-gray-100 text-gray-600' };
  }
}

function formatFactorName(name: string): string {
  return name.replace(/_/g, ' ').replace(/\b\w/g, (l) => l.toUpperCase());
}

/**
 * ConfidenceIndicator - Very compact confidence display
 */
interface ConfidenceIndicatorProps {
  level: string;
  percentage?: number;
}

export function ConfidenceIndicator({ level, percentage }: ConfidenceIndicatorProps) {
  const dotColor =
    level === 'High' ? 'bg-green-500' : level === 'Medium' ? 'bg-yellow-500' : 'bg-red-500';

  return (
    <div className="flex items-center gap-1.5">
      <span className={`w-2 h-2 rounded-full ${dotColor}`} />
      <span className="text-xs text-gray-500">
        {level}
        {percentage !== undefined && ` (${Math.round(percentage)}%)`}
      </span>
    </div>
  );
}

/**
 * DataFreshnessBar - Visual indicator of data freshness
 */
interface DataFreshnessBarProps {
  factors: Record<string, FactorDataStatus>;
}

export function DataFreshnessBar({ factors }: DataFreshnessBarProps) {
  const total = Object.keys(factors).length;
  const fresh = Object.values(factors).filter((f) => f.available && f.freshness === 'fresh').length;
  const recent = Object.values(factors).filter(
    (f) => f.available && f.freshness === 'recent'
  ).length;
  const stale = Object.values(factors).filter((f) => f.available && f.freshness === 'stale').length;
  const missing = Object.values(factors).filter((f) => !f.available).length;

  return (
    <div className="flex h-2 rounded-full overflow-hidden bg-gray-100">
      {fresh > 0 && (
        <div
          className="bg-green-500"
          style={{ width: `${(fresh / total) * 100}%` }}
          title={`${fresh} fresh`}
        />
      )}
      {recent > 0 && (
        <div
          className="bg-blue-400"
          style={{ width: `${(recent / total) * 100}%` }}
          title={`${recent} recent`}
        />
      )}
      {stale > 0 && (
        <div
          className="bg-yellow-400"
          style={{ width: `${(stale / total) * 100}%` }}
          title={`${stale} stale`}
        />
      )}
      {missing > 0 && (
        <div
          className="bg-gray-300"
          style={{ width: `${(missing / total) * 100}%` }}
          title={`${missing} missing`}
        />
      )}
    </div>
  );
}
