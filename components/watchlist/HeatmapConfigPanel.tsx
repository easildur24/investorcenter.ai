'use client';

import { ViewMode } from './HeatmapAdaptiveViews';

const TIME_PERIODS = ['1D', '1W', '1M', '3M', '6M', 'YTD', '1Y', '5Y'] as const;
export type TimePeriod = (typeof TIME_PERIODS)[number];

/** Compact subset shown on mobile (< 640px) */
const MOBILE_PERIODS: TimePeriod[] = ['1D', '1W', '1M', '1Y'];

const VIEW_MODES: { mode: ViewMode; label: string }[] = [
  { mode: 'auto', label: 'Auto' },
  { mode: 'treemap', label: 'Heatmap' },
  { mode: 'cards', label: 'Cards' },
  { mode: 'bars', label: 'Bar' },
];

interface HeatmapToolbarProps {
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
  timePeriod: TimePeriod;
  onTimePeriodChange: (period: TimePeriod) => void;
}

export default function HeatmapToolbar({
  viewMode,
  onViewModeChange,
  timePeriod,
  onTimePeriodChange,
}: HeatmapToolbarProps) {
  return (
    <div className="flex items-center justify-between gap-4 mb-4 h-[48px]">
      {/* View mode toggle — left side */}
      <div
        className="flex items-center gap-0.5 bg-ic-bg-secondary rounded-lg p-0.5 shrink-0"
        role="group"
        aria-label="View mode"
      >
        {VIEW_MODES.map(({ mode, label }) => (
          <button
            key={mode}
            onClick={() => onViewModeChange(mode)}
            className={`px-3 py-1.5 rounded-md text-[13px] font-medium transition-colors ${
              viewMode === mode
                ? 'bg-ic-blue text-white shadow-sm'
                : 'text-ic-text-dim hover:text-ic-text-primary'
            }`}
          >
            {label}
          </button>
        ))}
      </div>

      {/* Time period pills — right side */}
      <div className="flex items-center gap-0.5 shrink-0" role="group" aria-label="Time period">
        {TIME_PERIODS.map((period) => (
          <button
            key={period}
            onClick={() => onTimePeriodChange(period)}
            className={`px-2.5 py-1.5 rounded-md text-[13px] font-medium transition-colors ${
              timePeriod === period
                ? 'bg-ic-blue text-white shadow-sm'
                : 'text-ic-text-dim hover:text-ic-text-primary'
            } ${!MOBILE_PERIODS.includes(period) ? 'hidden sm:inline-flex' : 'inline-flex'}`}
          >
            {period}
          </button>
        ))}
      </div>
    </div>
  );
}
