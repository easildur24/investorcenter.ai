'use client';

import { PRESET_SCREENS } from '@/lib/screener/presets';
import type { ScreenerPreset } from '@/lib/types/screener';

interface ScreenerToolbarProps {
  onApplyPreset: (preset: ScreenerPreset) => void;
}

/** Quick Screen preset buttons row. */
export function ScreenerToolbar({ onApplyPreset }: ScreenerToolbarProps) {
  return (
    <div className="mb-6">
      <h3 className="text-sm font-medium text-ic-text-secondary mb-3">Quick Screens</h3>
      <div className="flex flex-wrap gap-2">
        {PRESET_SCREENS.map(preset => (
          <button
            key={preset.id}
            onClick={() => onApplyPreset(preset)}
            className="px-4 py-2 bg-ic-surface border border-ic-border rounded-lg hover:bg-ic-surface-hover transition-colors text-left"
          >
            <div className="text-sm font-medium text-ic-text-primary">{preset.name}</div>
            <div className="text-xs text-ic-text-muted">{preset.description}</div>
          </button>
        ))}
      </div>
    </div>
  );
}
