'use client';

import { memo } from 'react';

interface CheckboxFilterProps {
  label: string;
  options: { value: string; label: string }[];
  selected: string[];
  onChange: (values: string[]) => void;
}

/** Checkbox list for categorical filters (sector, industry). */
export const CheckboxFilter = memo(function CheckboxFilter({ label, options, selected, onChange }: CheckboxFilterProps) {
  return (
    <div>
      <label className="block text-sm font-medium text-ic-text-secondary mb-1.5">
        {label}
        {selected.length > 0 && (
          <span className="ml-1 text-xs text-ic-text-dim">({selected.length})</span>
        )}
      </label>
      <div className="space-y-1 max-h-40 overflow-y-auto">
        {options.map(option => (
          <label key={option.value} className="flex items-center">
            <input
              type="checkbox"
              checked={selected.includes(option.value)}
              onChange={(e) => {
                if (e.target.checked) {
                  onChange([...selected, option.value]);
                } else {
                  onChange(selected.filter(v => v !== option.value));
                }
              }}
              className="rounded border-ic-border text-ic-blue focus:ring-ic-blue"
            />
            <span className="ml-2 text-sm text-ic-text-muted">{option.label}</span>
          </label>
        ))}
      </div>
    </div>
  );
});
