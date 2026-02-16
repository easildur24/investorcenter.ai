'use client';

import { useState, useCallback, useEffect, useRef } from 'react';

interface RangeFilterProps {
  label: string;
  minValue: number | null;
  maxValue: number | null;
  onChange: (min: number | null, max: number | null) => void;
  step?: number;
  suffix?: string;
  placeholder?: { min: string; max: string };
}

/** Min/max number input pair. Commits on blur or Enter to avoid per-keystroke API calls. */
export function RangeFilter({
  label,
  minValue,
  maxValue,
  onChange,
  step,
  suffix,
  placeholder,
}: RangeFilterProps) {
  // Local state so we don't fire API calls on every keystroke.
  // Changes are committed on blur or Enter.
  const [localMin, setLocalMin] = useState<string>(minValue != null ? String(minValue) : '');
  const [localMax, setLocalMax] = useState<string>(maxValue != null ? String(maxValue) : '');
  const prevMinRef = useRef(minValue);
  const prevMaxRef = useRef(maxValue);

  // Sync local state when parent props change (e.g. preset applied, clear all)
  useEffect(() => {
    if (minValue !== prevMinRef.current) {
      setLocalMin(minValue != null ? String(minValue) : '');
      prevMinRef.current = minValue;
    }
    if (maxValue !== prevMaxRef.current) {
      setLocalMax(maxValue != null ? String(maxValue) : '');
      prevMaxRef.current = maxValue;
    }
  }, [minValue, maxValue]);

  const commit = useCallback(() => {
    const newMin = localMin !== '' ? Number(localMin) : null;
    const newMax = localMax !== '' ? Number(localMax) : null;
    // Only fire if values actually changed
    if (newMin !== minValue || newMax !== maxValue) {
      onChange(newMin, newMax);
    }
  }, [localMin, localMax, minValue, maxValue, onChange]);

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      commit();
    }
  }, [commit]);

  return (
    <div>
      <label className="block text-sm font-medium text-ic-text-secondary mb-1.5">
        {label}
      </label>
      <div className="flex gap-2 items-center">
        <input
          type="number"
          placeholder={placeholder?.min ?? 'Min'}
          value={localMin}
          onChange={(e) => setLocalMin(e.target.value)}
          onBlur={commit}
          onKeyDown={handleKeyDown}
          className="w-20 px-2 py-1 text-sm border border-ic-border rounded-md bg-ic-input-bg text-ic-text-primary placeholder:text-ic-text-dim"
          step={step}
        />
        <span className="text-ic-text-dim">&mdash;</span>
        <input
          type="number"
          placeholder={placeholder?.max ?? 'Max'}
          value={localMax}
          onChange={(e) => setLocalMax(e.target.value)}
          onBlur={commit}
          onKeyDown={handleKeyDown}
          className="w-20 px-2 py-1 text-sm border border-ic-border rounded-md bg-ic-input-bg text-ic-text-primary placeholder:text-ic-text-dim"
          step={step}
        />
        {suffix && <span className="text-sm text-ic-text-muted">{suffix}</span>}
      </div>
    </div>
  );
}
