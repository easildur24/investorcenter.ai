'use client';

import { useMemo } from 'react';
import { FILTER_GROUPS, FILTER_DEFS } from '@/lib/screener/filter-config';
import { FilterSection } from './FilterSection';
import { RangeFilter } from './RangeFilter';
import { CheckboxFilter } from './CheckboxFilter';
import { IndustryFilter } from './IndustryFilter';

interface FilterPanelProps {
  urlState: Record<string, unknown>;
  activeFilterCount: number;
  selectedSectors: string[];
  selectedIndustries: string[];
  onRangeChange: (minKey: string, maxKey: string, min: number | null, max: number | null) => void;
  onSectorsChange: (sectors: string[]) => void;
  onIndustriesChange: (industries: string[]) => void;
  onClearFilters: () => void;
}

function getUrlStateValue(state: Record<string, unknown>, key: string): number | null {
  const val = state[key];
  return typeof val === 'number' ? val : null;
}

/** Sidebar filter panel with collapsible sections. */
export function FilterPanel({
  urlState,
  activeFilterCount,
  selectedSectors,
  selectedIndustries,
  onRangeChange,
  onSectorsChange,
  onIndustriesChange,
  onClearFilters,
}: FilterPanelProps) {
  // Count active filters per group
  const groupActiveCount = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const group of FILTER_GROUPS) {
      const groupFilters = FILTER_DEFS.filter(f => f.group === group.id);
      let count = 0;
      for (const filter of groupFilters) {
        if (filter.type === 'multiselect' && filter.id === 'sector') {
          count += selectedSectors.length > 0 ? 1 : 0;
        } else if (filter.type === 'multiselect' && filter.id === 'industry') {
          count += selectedIndustries.length > 0 ? 1 : 0;
        } else if (filter.type === 'range') {
          if (filter.minKey && getUrlStateValue(urlState, filter.minKey) != null) count++;
          if (filter.maxKey && getUrlStateValue(urlState, filter.maxKey) != null) count++;
        }
      }
      counts[group.id] = count;
    }
    return counts;
  }, [urlState, selectedSectors, selectedIndustries]);

  return (
    <div className="w-64 flex-shrink-0">
      <div className="bg-ic-surface rounded-lg border border-ic-border p-4 sticky top-4">
        <div className="flex items-center justify-between mb-4">
          <h3 className="font-semibold text-ic-text-primary">
            Filters
            {activeFilterCount > 0 && (
              <span className="ml-2 px-1.5 py-0.5 text-xs bg-ic-blue text-white rounded-full">
                {activeFilterCount}
              </span>
            )}
          </h3>
          {activeFilterCount > 0 && (
            <button
              onClick={onClearFilters}
              className="text-sm text-ic-blue hover:text-ic-blue-hover transition-colors"
            >
              Clear All
            </button>
          )}
        </div>

        <div className="space-y-5 max-h-[calc(100vh-200px)] overflow-y-auto pr-1">
          {FILTER_GROUPS.map(group => {
            const groupFilters = FILTER_DEFS.filter(f => f.group === group.id);
            if (groupFilters.length === 0) return null;

            return (
              <FilterSection
                key={group.id}
                label={group.label}
                activeCount={groupActiveCount[group.id] ?? 0}
                defaultOpen={group.defaultOpen}
              >
                {groupFilters.map(filter => {
                  if (filter.type === 'multiselect' && filter.id === 'sector') {
                    return (
                      <CheckboxFilter
                        key={filter.id}
                        label={filter.label}
                        options={filter.options!}
                        selected={selectedSectors}
                        onChange={onSectorsChange}
                      />
                    );
                  }
                  if (filter.type === 'multiselect' && filter.id === 'industry') {
                    return (
                      <IndustryFilter
                        key={filter.id}
                        selectedSectors={selectedSectors}
                        selectedIndustries={selectedIndustries}
                        onChange={onIndustriesChange}
                      />
                    );
                  }
                  if (filter.type === 'range' && filter.minKey && filter.maxKey) {
                    return (
                      <RangeFilter
                        key={filter.id}
                        label={filter.label}
                        minValue={getUrlStateValue(urlState, filter.minKey)}
                        maxValue={getUrlStateValue(urlState, filter.maxKey)}
                        onChange={(min, max) =>
                          onRangeChange(filter.minKey!, filter.maxKey!, min, max)
                        }
                        step={filter.step}
                        suffix={filter.suffix}
                        placeholder={filter.placeholder}
                      />
                    );
                  }
                  return null;
                })}
              </FilterSection>
            );
          })}
        </div>
      </div>
    </div>
  );
}
