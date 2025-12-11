'use client';

import { cn } from '@/lib/utils';
import { Timeframe, timeframeLabels } from '@/types/financials';

interface TimeframePickerProps {
  value: Timeframe;
  onChange: (timeframe: Timeframe) => void;
}

const timeframes: Timeframe[] = ['quarterly', 'annual', 'trailing_twelve_months'];

export default function TimeframePicker({ value, onChange }: TimeframePickerProps) {
  return (
    <div className="flex gap-2">
      {timeframes.map((timeframe) => (
        <button
          key={timeframe}
          onClick={() => onChange(timeframe)}
          className={cn(
            'px-3 py-1.5 text-sm font-medium rounded-md transition-all',
            value === timeframe
              ? 'bg-blue-100 text-blue-700 ring-1 ring-blue-200'
              : 'text-ic-text-muted hover:bg-ic-surface-hover'
          )}
        >
          {timeframeLabels[timeframe]}
        </button>
      ))}
    </div>
  );
}
