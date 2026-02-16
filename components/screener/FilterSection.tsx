'use client';

import { useState } from 'react';

interface FilterSectionProps {
  label: string;
  activeCount: number;
  defaultOpen?: boolean;
  children: React.ReactNode;
}

/** Collapsible filter group with active filter count badge. */
export function FilterSection({ label, activeCount, defaultOpen = false, children }: FilterSectionProps) {
  const [open, setOpen] = useState(defaultOpen);

  return (
    <div>
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex items-center justify-between w-full text-left"
      >
        <h4 className="text-xs font-semibold text-ic-text-muted uppercase tracking-wider">
          {label}
          {activeCount > 0 && (
            <span className="ml-1.5 px-1.5 py-0.5 text-[10px] bg-ic-blue text-white rounded-full normal-case tracking-normal font-medium">
              {activeCount}
            </span>
          )}
        </h4>
        <svg
          className={`w-4 h-4 text-ic-text-dim transition-transform ${open ? 'rotate-180' : ''}`}
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>
      {open && <div className="mt-2 space-y-3">{children}</div>}
    </div>
  );
}
