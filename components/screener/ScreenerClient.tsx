'use client';

import { useCallback, useState, useEffect } from 'react';
import { useQueryStates, parseAsFloat, parseAsInteger, parseAsString } from 'nuqs';
import { useScreener } from '@/lib/hooks/useScreener';
import { loadVisibleColumns, saveVisibleColumns } from '@/lib/screener/column-config';
import type { ScreenerApiParams, ScreenerSortField, ScreenerPreset } from '@/lib/types/screener';
import { ScreenerToolbar } from './ScreenerToolbar';
import { FilterPanel } from './FilterPanel';
import { ResultsTable } from './ResultsTable';
import { ColumnPicker } from './ColumnPicker';
import { ExportButton } from './ExportButton';
import { Pagination } from './Pagination';

const ITEMS_PER_PAGE = 25;

/** Shared defaults for URL state — used by urlStateConfig, clearFilters, and applyPreset. */
const URL_DEFAULTS = {
  page: 1,
  limit: ITEMS_PER_PAGE,
  sort: 'market_cap',
  order: 'desc',
} as const;

// nuqs URL state configuration — one entry per filter/sort/pagination param
const urlStateConfig = {
  // Pagination & sorting
  page: parseAsInteger.withDefault(URL_DEFAULTS.page),
  limit: parseAsInteger.withDefault(URL_DEFAULTS.limit),
  sort: parseAsString.withDefault(URL_DEFAULTS.sort),
  order: parseAsString.withDefault(URL_DEFAULTS.order),

  // Categorical filters
  sectors: parseAsString,
  industries: parseAsString,

  // Market data
  market_cap_min: parseAsFloat,
  market_cap_max: parseAsFloat,

  // Valuation
  pe_min: parseAsFloat,
  pe_max: parseAsFloat,
  pb_min: parseAsFloat,
  pb_max: parseAsFloat,
  ps_min: parseAsFloat,
  ps_max: parseAsFloat,

  // Profitability
  roe_min: parseAsFloat,
  roe_max: parseAsFloat,
  roa_min: parseAsFloat,
  roa_max: parseAsFloat,
  gross_margin_min: parseAsFloat,
  gross_margin_max: parseAsFloat,
  net_margin_min: parseAsFloat,
  net_margin_max: parseAsFloat,

  // Financial health
  de_min: parseAsFloat,
  de_max: parseAsFloat,
  current_ratio_min: parseAsFloat,
  current_ratio_max: parseAsFloat,

  // Growth
  revenue_growth_min: parseAsFloat,
  revenue_growth_max: parseAsFloat,
  eps_growth_min: parseAsFloat,
  eps_growth_max: parseAsFloat,

  // Dividends
  dividend_yield_min: parseAsFloat,
  dividend_yield_max: parseAsFloat,
  payout_ratio_min: parseAsFloat,
  payout_ratio_max: parseAsFloat,
  consec_div_years_min: parseAsFloat,

  // Risk
  beta_min: parseAsFloat,
  beta_max: parseAsFloat,

  // Fair value
  dcf_upside_min: parseAsFloat,
  dcf_upside_max: parseAsFloat,

  // IC Score
  ic_score_min: parseAsFloat,
  ic_score_max: parseAsFloat,

  // IC Score sub-factors
  value_score_min: parseAsFloat,
  value_score_max: parseAsFloat,
  growth_score_min: parseAsFloat,
  growth_score_max: parseAsFloat,
  profitability_score_min: parseAsFloat,
  profitability_score_max: parseAsFloat,
  financial_health_score_min: parseAsFloat,
  financial_health_score_max: parseAsFloat,
  momentum_score_min: parseAsFloat,
  momentum_score_max: parseAsFloat,
  analyst_score_min: parseAsFloat,
  analyst_score_max: parseAsFloat,
  insider_score_min: parseAsFloat,
  insider_score_max: parseAsFloat,
  institutional_score_min: parseAsFloat,
  institutional_score_max: parseAsFloat,
  sentiment_score_min: parseAsFloat,
  sentiment_score_max: parseAsFloat,
  technical_score_min: parseAsFloat,
  technical_score_max: parseAsFloat,
};

/** Top-level client component that owns URL state, SWR data, and column visibility. */
export function ScreenerClient() {
  // URL state — all filters, sorting, and pagination live in the URL
  const [urlState, setUrlState] = useQueryStates(urlStateConfig, {
    history: 'push',
    throttleMs: 300,
    shallow: false,
  });

  // Column visibility (persisted in localStorage)
  const [visibleColumns, setVisibleColumns] = useState<ScreenerSortField[]>(loadVisibleColumns);

  // Save column changes to localStorage
  useEffect(() => {
    saveVisibleColumns(visibleColumns);
  }, [visibleColumns]);

  // Build API params from URL state (strip nulls)
  const apiParams: ScreenerApiParams = {};
  for (const [key, val] of Object.entries(urlState)) {
    if (val !== null && val !== undefined) {
      (apiParams as Record<string, unknown>)[key] = val;
    }
  }

  // Server-side data fetching via SWR
  const { stocks, meta, isLoading, isValidating, error } = useScreener(apiParams);

  // -------------------------------------------------------------------
  // Handlers
  // -------------------------------------------------------------------

  const handleSort = useCallback(
    (field: ScreenerSortField) => {
      setUrlState((prev) => ({
        ...prev,
        sort: field,
        order: prev.sort === field && prev.order === 'desc' ? 'asc' : 'desc',
        page: 1,
      }));
    },
    [setUrlState]
  );

  const handlePageChange = useCallback(
    (newPage: number) => {
      setUrlState((prev) => ({ ...prev, page: newPage }));
    },
    [setUrlState]
  );

  const handleRangeChange = useCallback(
    (minKey: string, maxKey: string, minVal: number | null, maxVal: number | null) => {
      setUrlState((prev) => ({
        ...prev,
        [minKey]: minVal,
        [maxKey]: maxVal,
        page: 1,
      }));
    },
    [setUrlState]
  );

  const handleSectorsChange = useCallback(
    (sectors: string[]) => {
      setUrlState((prev) => ({
        ...prev,
        sectors: sectors.length > 0 ? sectors.join(',') : null,
        industries: null, // Clear industries when sectors change
        page: 1,
      }));
    },
    [setUrlState]
  );

  const handleIndustriesChange = useCallback(
    (industries: string[]) => {
      setUrlState((prev) => ({
        ...prev,
        industries: industries.length > 0 ? industries.join(',') : null,
        page: 1,
      }));
    },
    [setUrlState]
  );

  const applyPreset = useCallback(
    (preset: ScreenerPreset) => {
      const cleared: Record<string, null> = {};
      for (const key of Object.keys(urlStateConfig)) {
        if (key !== 'limit') cleared[key] = null;
      }
      setUrlState({
        ...cleared,
        ...URL_DEFAULTS,
        ...preset.params,
      } as typeof urlState);
    },
    [setUrlState]
  );

  const clearFilters = useCallback(() => {
    const cleared: Record<string, null | number | string> = {};
    for (const key of Object.keys(urlStateConfig)) {
      cleared[key] = null;
    }
    Object.assign(cleared, URL_DEFAULTS);
    setUrlState(cleared as typeof urlState);
  }, [setUrlState]);

  // Derived state
  const activeFilterCount = Object.entries(urlState).filter(([key, val]) => {
    if (['page', 'limit', 'sort', 'order'].includes(key)) return false;
    return val !== null;
  }).length;

  const selectedSectors = urlState.sectors ? urlState.sectors.split(',') : [];
  const selectedIndustries = urlState.industries ? urlState.industries.split(',') : [];
  const totalPages = meta?.total_pages ?? 0;
  const total = meta?.total ?? 0;
  const currentPage = urlState.page ?? 1;

  // -------------------------------------------------------------------
  // Render
  // -------------------------------------------------------------------

  return (
    <div className="min-h-screen bg-ic-bg-primary">
      {/* Header */}
      <div className="bg-ic-surface border-b border-ic-border">
        <div className="max-w-[1400px] mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <h1 className="text-2xl font-bold text-ic-text-primary">Stock Screener</h1>
          <p className="mt-1 text-ic-text-muted">
            Filter and discover stocks across {total > 0 ? total.toLocaleString() : '5,600+'} US stocks
          </p>
        </div>
      </div>

      <div className="max-w-[1400px] mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <ScreenerToolbar onApplyPreset={applyPreset} />

        <div className="flex gap-6">
          <FilterPanel
            urlState={urlState as Record<string, unknown>}
            activeFilterCount={activeFilterCount}
            selectedSectors={selectedSectors}
            selectedIndustries={selectedIndustries}
            onRangeChange={handleRangeChange}
            onSectorsChange={handleSectorsChange}
            onIndustriesChange={handleIndustriesChange}
            onClearFilters={clearFilters}
          />

          {/* Results */}
          <div className="flex-1 min-w-0">
            <div className="bg-ic-surface rounded-lg border border-ic-border overflow-hidden">
              {/* Error banner */}
              {error && (
                <div className="px-4 py-3 bg-ic-negative-bg border-b border-ic-border text-ic-negative text-sm">
                  Failed to load stocks. Please try again later.
                </div>
              )}

              {/* Results Header */}
              <div className="px-4 py-3 border-b border-ic-border flex items-center justify-between">
                <span className="text-sm text-ic-text-muted">
                  {isLoading ? 'Loading...' : `${total.toLocaleString()} stocks found`}
                  {isValidating && !isLoading && (
                    <span className="ml-2 text-ic-text-dim">(updating...)</span>
                  )}
                </span>
                <div className="flex items-center gap-2">
                  <ExportButton params={apiParams} />
                  <ColumnPicker
                    visibleColumns={visibleColumns}
                    onChange={setVisibleColumns}
                  />
                </div>
              </div>

              <ResultsTable
                stocks={stocks}
                visibleColumns={visibleColumns}
                sortField={urlState.sort ?? 'market_cap'}
                sortOrder={urlState.order ?? 'desc'}
                isLoading={isLoading}
                onSort={handleSort}
              />

              <Pagination
                currentPage={currentPage}
                totalPages={totalPages}
                total={total}
                pageSize={urlState.limit ?? ITEMS_PER_PAGE}
                onPageChange={handlePageChange}
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
