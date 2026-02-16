'use client';

import { useState, useCallback } from 'react';
import useSWR from 'swr';
import { API_BASE_URL } from '@/lib/api';

interface IndustryFilterProps {
  selectedSectors: string[];
  selectedIndustries: string[];
  onChange: (industries: string[]) => void;
}

async function industriesFetcher(url: string): Promise<string[]> {
  const res = await fetch(url);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  const json = await res.json();
  return json.data ?? [];
}

/** Searchable industry checkbox list that dynamically loads industries from the API. */
export function IndustryFilter({ selectedSectors, selectedIndustries, onChange }: IndustryFilterProps) {
  const [search, setSearch] = useState('');

  // Build URL with optional sector filter
  const sectorsParam = selectedSectors.length > 0 ? `?sectors=${encodeURIComponent(selectedSectors.join(','))}` : '';
  const url = `${API_BASE_URL}/screener/industries${sectorsParam}`;

  const { data: industries = [], isLoading } = useSWR(url, industriesFetcher, {
    revalidateOnFocus: false,
    dedupingInterval: 5000,
  });

  const filtered = search
    ? industries.filter(i => i.toLowerCase().includes(search.toLowerCase()))
    : industries;

  const toggle = useCallback((industry: string) => {
    if (selectedIndustries.includes(industry)) {
      onChange(selectedIndustries.filter(i => i !== industry));
    } else {
      onChange([...selectedIndustries, industry]);
    }
  }, [selectedIndustries, onChange]);

  return (
    <div>
      <label className="block text-sm font-medium text-ic-text-secondary mb-1.5">
        Industry
        {selectedIndustries.length > 0 && (
          <span className="ml-1 text-xs text-ic-text-dim">({selectedIndustries.length})</span>
        )}
      </label>
      <input
        type="text"
        placeholder="Search industries..."
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="w-full px-2 py-1 text-sm border border-ic-border rounded-md bg-ic-input-bg text-ic-text-primary placeholder:text-ic-text-dim mb-1"
      />
      <div className="space-y-1 max-h-40 overflow-y-auto">
        {isLoading && (
          <div className="text-xs text-ic-text-dim py-2">Loading industries...</div>
        )}
        {!isLoading && filtered.length === 0 && (
          <div className="text-xs text-ic-text-dim py-2">
            {search ? 'No matching industries' : 'No industries available'}
          </div>
        )}
        {filtered.map(industry => (
          <label key={industry} className="flex items-center">
            <input
              type="checkbox"
              checked={selectedIndustries.includes(industry)}
              onChange={() => toggle(industry)}
              className="rounded border-ic-border text-ic-blue focus:ring-ic-blue"
            />
            <span className="ml-2 text-sm text-ic-text-muted truncate">{industry}</span>
          </label>
        ))}
      </div>
    </div>
  );
}
