'use client';

import { useState, useRef, useEffect } from 'react';
import { ALL_COLUMNS } from '@/lib/screener/column-config';
import type { ScreenerSortField } from '@/lib/types/screener';

interface ColumnPickerProps {
  visibleColumns: ScreenerSortField[];
  onChange: (columns: ScreenerSortField[]) => void;
}

/** Gear icon popover for toggling table column visibility. */
export function ColumnPicker({ visibleColumns, onChange }: ColumnPickerProps) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  // Close on outside click
  useEffect(() => {
    if (!open) return;
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [open]);

  const toggle = (key: ScreenerSortField) => {
    // Always keep symbol visible
    if (key === 'symbol') return;
    if (visibleColumns.includes(key)) {
      onChange(visibleColumns.filter(k => k !== key));
    } else {
      // Insert at the position matching ALL_COLUMNS order
      const ordered = ALL_COLUMNS
        .map(c => c.key)
        .filter(k => visibleColumns.includes(k) || k === key);
      onChange(ordered);
    }
  };

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="p-1.5 rounded-md text-ic-text-muted hover:text-ic-text-primary hover:bg-ic-surface-hover transition-colors"
        title="Customize columns"
      >
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
            d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.573-1.066z" />
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
        </svg>
      </button>

      {open && (
        <div className="absolute right-0 top-full mt-1 w-56 bg-ic-surface border border-ic-border rounded-lg shadow-lg z-50 max-h-80 overflow-y-auto">
          <div className="p-2">
            <div className="text-xs font-semibold text-ic-text-muted uppercase tracking-wider px-2 py-1 mb-1">
              Columns
            </div>
            {ALL_COLUMNS.map(col => (
              <label
                key={col.key}
                className="flex items-center px-2 py-1.5 rounded hover:bg-ic-surface-hover cursor-pointer"
              >
                <input
                  type="checkbox"
                  checked={visibleColumns.includes(col.key)}
                  onChange={() => toggle(col.key)}
                  disabled={col.key === 'symbol'}
                  className="rounded border-ic-border text-ic-blue focus:ring-ic-blue disabled:opacity-50"
                />
                <span className="ml-2 text-sm text-ic-text-primary">{col.label}</span>
              </label>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
