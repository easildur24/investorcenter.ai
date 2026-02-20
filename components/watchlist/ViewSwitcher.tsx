'use client';

import { cn } from '@/lib/utils';
import { VIEW_PRESETS, ViewPresetId } from '@/lib/watchlist/columns';

interface ViewSwitcherProps {
  activeView: ViewPresetId;
  onViewChange: (view: ViewPresetId) => void;
}

export default function ViewSwitcher({ activeView, onViewChange }: ViewSwitcherProps) {
  return (
    <div className="flex gap-2 overflow-x-auto pb-1" role="tablist" aria-label="View presets">
      {VIEW_PRESETS.map((preset) => {
        const isActive = activeView === preset.id;
        return (
          <button
            key={preset.id}
            id={`watchlist-tab-${preset.id}`}
            role="tab"
            aria-selected={isActive}
            aria-controls="watchlist-tabpanel"
            onClick={() => onViewChange(preset.id)}
            className={cn(
              'px-3 py-1.5 text-sm font-medium rounded-lg whitespace-nowrap transition-colors',
              isActive
                ? 'bg-ic-blue text-white'
                : 'bg-ic-surface border border-ic-border text-ic-text-secondary hover:bg-ic-surface-hover'
            )}
            title={preset.description}
          >
            {preset.label}
            {preset.premium && (
              <svg
                className="inline-block ml-1 w-3 h-3 opacity-60"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-label="Premium"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
                />
              </svg>
            )}
          </button>
        );
      })}
    </div>
  );
}
