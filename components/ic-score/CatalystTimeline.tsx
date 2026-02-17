'use client';

import {
  Catalyst,
  CATALYST_TYPE_LABELS,
  CATALYST_ICONS,
  formatDaysUntil,
  getImpactColor,
} from '@/lib/types/ic-score-v2';
import { ImpactBadge } from './Badges';

interface CatalystTimelineProps {
  catalysts: Catalyst[];
  maxItems?: number;
}

/**
 * CatalystTimeline - Displays upcoming catalysts/events for a stock
 *
 * Shows events like:
 * - Earnings reports
 * - Ex-dividend dates
 * - Analyst rating changes
 * - Technical breakouts
 * - 52-week highs/lows
 */
export default function CatalystTimeline({ catalysts, maxItems = 5 }: CatalystTimelineProps) {
  const displayCatalysts = catalysts.slice(0, maxItems);

  if (displayCatalysts.length === 0) {
    return (
      <div className="border-t border-gray-200 p-4">
        <h3 className="font-medium text-gray-900 mb-3">Upcoming Catalysts</h3>
        <p className="text-sm text-gray-500">No upcoming catalysts detected.</p>
      </div>
    );
  }

  return (
    <div className="border-t border-gray-200 p-4">
      <h3 className="font-medium text-gray-900 mb-4">Upcoming Catalysts</h3>

      <div className="space-y-3">
        {displayCatalysts.map((catalyst, index) => (
          <CatalystItem key={`${catalyst.event_type}-${index}`} catalyst={catalyst} />
        ))}
      </div>

      {catalysts.length > maxItems && (
        <div className="mt-3 text-sm text-gray-500 text-center">
          +{catalysts.length - maxItems} more catalysts
        </div>
      )}
    </div>
  );
}

interface CatalystItemProps {
  catalyst: Catalyst;
}

function CatalystItem({ catalyst }: CatalystItemProps) {
  const icon = catalyst.icon || CATALYST_ICONS[catalyst.event_type] || '📌';
  const label = CATALYST_TYPE_LABELS[catalyst.event_type] || catalyst.event_type;
  const impactColor = getImpactColor(catalyst.impact);

  return (
    <div className="flex items-start gap-3 p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors">
      {/* Icon */}
      <span className="text-2xl flex-shrink-0">{icon}</span>

      {/* Content */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="font-medium text-gray-900">{catalyst.title}</span>
          <ImpactBadge impact={catalyst.impact} size="sm" />
        </div>

        <div className="flex items-center gap-3 mt-1 text-sm text-gray-500">
          <span>{label}</span>
          {catalyst.event_date && (
            <>
              <span>•</span>
              <span>{formatEventDate(catalyst.event_date)}</span>
            </>
          )}
        </div>

        {/* Confidence indicator */}
        {catalyst.confidence !== null && catalyst.confidence < 0.8 && (
          <div className="mt-1 text-xs text-gray-400">
            Confidence: {Math.round(catalyst.confidence * 100)}%
          </div>
        )}
      </div>

      {/* Days until */}
      <div className="flex-shrink-0 text-right">
        {catalyst.days_until !== null && <DaysUntilBadge days={catalyst.days_until} />}
      </div>
    </div>
  );
}

interface DaysUntilBadgeProps {
  days: number;
}

function DaysUntilBadge({ days }: DaysUntilBadgeProps) {
  let bgColor = 'bg-gray-100 text-gray-700';
  let text = formatDaysUntil(days);

  if (days === 0) {
    bgColor = 'bg-blue-100 text-blue-700';
    text = 'Today';
  } else if (days === 1) {
    bgColor = 'bg-blue-100 text-blue-700';
    text = 'Tomorrow';
  } else if (days > 0 && days <= 7) {
    bgColor = 'bg-yellow-100 text-yellow-700';
    text = `${days}d`;
  } else if (days > 7 && days <= 30) {
    bgColor = 'bg-gray-100 text-gray-600';
    text = `${days}d`;
  } else if (days < 0) {
    bgColor = 'bg-gray-50 text-gray-400';
    text = `${Math.abs(days)}d ago`;
  }

  return (
    <span className={`inline-flex items-center px-2 py-1 rounded text-xs font-medium ${bgColor}`}>
      {text}
    </span>
  );
}

function formatEventDate(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: date.getFullYear() !== new Date().getFullYear() ? 'numeric' : undefined,
  });
}

/**
 * CatalystCompact - Smaller version showing just upcoming counts
 */
interface CatalystCompactProps {
  catalysts: Catalyst[];
}

export function CatalystCompact({ catalysts }: CatalystCompactProps) {
  const upcoming = catalysts.filter((c) => c.days_until !== null && c.days_until >= 0);
  const nextCatalyst = upcoming[0];

  if (!nextCatalyst) {
    return null;
  }

  const icon = nextCatalyst.icon || CATALYST_ICONS[nextCatalyst.event_type] || '📌';

  return (
    <div className="flex items-center gap-2 text-sm">
      <span>{icon}</span>
      <span className="text-gray-600 truncate max-w-[200px]">{nextCatalyst.title}</span>
      {nextCatalyst.days_until !== null && (
        <span className="text-gray-400">({formatDaysUntil(nextCatalyst.days_until)})</span>
      )}
      {upcoming.length > 1 && <span className="text-gray-400">+{upcoming.length - 1}</span>}
    </div>
  );
}
