'use client';

import { useCallback, useState, useEffect, useRef } from 'react';
import { useQueryStates, parseAsFloat, parseAsInteger, parseAsString } from 'nuqs';
import { useScreener } from '@/lib/hooks/useScreener';
import { loadVisibleColumns, saveVisibleColumns } from '@/lib/screener/column-config';
import { apiClient } from '@/lib/api';
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

  // -------------------------------------------------------------------
  // NLP Search
  // -------------------------------------------------------------------

  const [nlpQuery, setNlpQuery] = useState('');
  const [nlpLoading, setNlpLoading] = useState(false);
  const [nlpExplanation, setNlpExplanation] = useState<string | null>(null);
  const [nlpError, setNlpError] = useState<string | null>(null);
  const nlpQueryRef = useRef(nlpQuery);
  nlpQueryRef.current = nlpQuery;
  const nlpErrorTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (nlpErrorTimerRef.current) clearTimeout(nlpErrorTimerRef.current);
    };
  }, []);

  const handleNlpSubmit = useCallback(async () => {
    const trimmed = nlpQueryRef.current.trim();
    if (!trimmed || nlpLoading) return;

    setNlpLoading(true);
    setNlpError(null);
    setNlpExplanation(null);

    try {
      const result = await apiClient.nlpScreenerQuery(trimmed);

      // Apply returned params the same way applyPreset does
      const cleared: Record<string, null> = {};
      for (const key of Object.keys(urlStateConfig)) {
        if (key !== 'limit') cleared[key] = null;
      }
      setUrlState({
        ...cleared,
        ...URL_DEFAULTS,
        ...result.params,
      } as typeof urlState);

      setNlpExplanation(result.explanation);
    } catch {
      setNlpError('Could not interpret query. Try rephrasing.');
      if (nlpErrorTimerRef.current) clearTimeout(nlpErrorTimerRef.current);
      nlpErrorTimerRef.current = setTimeout(() => setNlpError(null), 5000);
    } finally {
      setNlpLoading(false);
    }
  }, [nlpLoading, setUrlState]);

  const clearNlpExplanation = useCallback(() => {
    setNlpExplanation(null);
  }, []);

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
        {/* NLP Search */}
        <div className="mb-6">
          <div className="flex items-center gap-3">
            <div className="relative flex-1 max-w-2xl">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                <svg className="h-5 w-5 text-ic-text-muted" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M9.813 15.904 9 18.75l-.813-2.846a4.5 4.5 0 0 0-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 0 0 3.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 0 0 3.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 0 0-3.09 3.09ZM18.259 8.715 18 9.75l-.259-1.035a3.375 3.375 0 0 0-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 0 0 2.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 0 0 2.455 2.456L21.75 6l-1.036.259a3.375 3.375 0 0 0-2.455 2.456Z" />
                </svg>
              </div>
              <input
                type="text"
                value={nlpQuery}
                onChange={(e) => setNlpQuery(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') handleNlpSubmit(); }}
                placeholder='Ask AI: e.g. "show me tech companies with more than 2T market cap"'
                className="w-full pl-10 pr-4 py-2.5 bg-ic-surface border border-ic-border rounded-lg text-sm text-ic-text-primary placeholder-ic-text-muted focus:outline-none focus:ring-2 focus:ring-ic-blue focus:border-transparent"
                disabled={nlpLoading}
              />
            </div>
            <button
              onClick={handleNlpSubmit}
              disabled={!nlpQuery.trim() || nlpLoading}
              className="px-4 py-2.5 bg-ic-blue text-white rounded-lg text-sm font-medium hover:bg-ic-blue-hover transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
            >
              {nlpLoading ? (
                <div className="animate-spin rounded-full h-4 w-4 border-2 border-white border-t-transparent" />
              ) : (
                <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M13.5 4.5 21 12m0 0-7.5 7.5M21 12H3" />
                </svg>
              )}
              Search
            </button>
          </div>

          {/* NLP explanation banner */}
          {nlpExplanation && (
            <div className="mt-2 flex items-center gap-2 px-3 py-2 bg-ic-blue/10 border border-ic-blue/20 rounded-lg text-sm text-ic-blue">
              <svg className="h-4 w-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="m11.25 11.25.041-.02a.75.75 0 0 1 1.063.852l-.708 2.836a.75.75 0 0 0 1.063.853l.041-.021M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9-3.75h.008v.008H12V8.25Z" />
              </svg>
              <span className="flex-1">{nlpExplanation}</span>
              <button
                onClick={clearNlpExplanation}
                className="text-ic-blue hover:text-ic-blue-hover flex-shrink-0"
              >
                <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M6 18 18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          )}

          {/* NLP error */}
          {nlpError && (
            <div className="mt-2 px-3 py-2 bg-ic-negative-bg border border-ic-negative/20 rounded-lg text-sm text-ic-negative">
              {nlpError}
            </div>
          )}
        </div>

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
