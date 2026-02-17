'use client';

import { Suspense, useCallback, useState, useEffect, useRef } from 'react';
import Link from 'next/link';
import { useQueryStates, parseAsFloat, parseAsInteger, parseAsString } from 'nuqs';
import { useScreener } from '@/lib/hooks/useScreener';
import { cn, safeToFixed, formatLargeNumber } from '@/lib/utils';
import type {
  ScreenerApiParams,
  ScreenerSortField,
  ScreenerPreset,
  ScreenerStock,
} from '@/lib/types/screener';

// ============================================================================
// Constants
// ============================================================================

const SECTORS = [
  'Technology',
  'Healthcare',
  'Financial Services',
  'Consumer Cyclical',
  'Consumer Defensive',
  'Industrials',
  'Energy',
  'Basic Materials',
  'Real Estate',
  'Communication Services',
  'Utilities',
];

const ITEMS_PER_PAGE = 25;

const PRESET_SCREENS: ScreenerPreset[] = [
  {
    id: 'value',
    name: 'Value Stocks',
    description: 'Low P/E, high dividend yield',
    params: { pe_max: 15, dividend_yield_min: 2 },
  },
  {
    id: 'growth',
    name: 'Growth Stocks',
    description: 'High revenue & EPS growth',
    params: { revenue_growth_min: 20, eps_growth_min: 15 },
  },
  {
    id: 'quality',
    name: 'Quality Stocks',
    description: 'High IC Score, large cap',
    params: { ic_score_min: 70, market_cap_min: 10e9 },
  },
  {
    id: 'dividend',
    name: 'Dividend Champions',
    description: 'High yield, low payout ratio',
    params: { dividend_yield_min: 3, payout_ratio_max: 80, market_cap_min: 10e9 },
  },
  {
    id: 'undervalued',
    name: 'Undervalued',
    description: 'High DCF upside, profitable',
    params: { dcf_upside_min: 20, roe_min: 10, de_max: 2 },
  },
  {
    id: 'momentum',
    name: 'Momentum Leaders',
    description: 'High momentum + technical scores',
    params: { momentum_score_min: 70, technical_score_min: 70 },
  },
];

// Filter definitions for the sidebar
interface FilterDef {
  id: string;
  label: string;
  type: 'range' | 'multiselect';
  group: string;
  minKey?: keyof ScreenerApiParams;
  maxKey?: keyof ScreenerApiParams;
  options?: { value: string; label: string }[];
  step?: number;
  suffix?: string;
  placeholder?: { min: string; max: string };
}

const FILTER_GROUPS: { id: string; label: string }[] = [
  { id: 'general', label: 'General' },
  { id: 'valuation', label: 'Valuation' },
  { id: 'profitability', label: 'Profitability' },
  { id: 'financial_health', label: 'Financial Health' },
  { id: 'growth', label: 'Growth' },
  { id: 'dividends', label: 'Dividends' },
  { id: 'risk', label: 'Risk & Value' },
  { id: 'ic_score', label: 'IC Score' },
];

const FILTER_DEFS: FilterDef[] = [
  // General
  {
    id: 'sector',
    label: 'Sector',
    type: 'multiselect',
    group: 'general',
    options: SECTORS.map((s) => ({ value: s, label: s })),
  },
  {
    id: 'market_cap',
    label: 'Market Cap',
    type: 'range',
    group: 'general',
    minKey: 'market_cap_min',
    maxKey: 'market_cap_max',
    placeholder: { min: 'e.g. 1e9', max: 'e.g. 200e9' },
  },

  // Valuation
  {
    id: 'pe_ratio',
    label: 'P/E Ratio',
    type: 'range',
    group: 'valuation',
    minKey: 'pe_min',
    maxKey: 'pe_max',
    step: 1,
    placeholder: { min: '0', max: '100' },
  },
  {
    id: 'pb_ratio',
    label: 'P/B Ratio',
    type: 'range',
    group: 'valuation',
    minKey: 'pb_min',
    maxKey: 'pb_max',
    step: 0.1,
    placeholder: { min: '0', max: '20' },
  },
  {
    id: 'ps_ratio',
    label: 'P/S Ratio',
    type: 'range',
    group: 'valuation',
    minKey: 'ps_min',
    maxKey: 'ps_max',
    step: 0.1,
    placeholder: { min: '0', max: '30' },
  },

  // Profitability
  {
    id: 'roe',
    label: 'ROE',
    type: 'range',
    group: 'profitability',
    minKey: 'roe_min',
    maxKey: 'roe_max',
    step: 1,
    suffix: '%',
    placeholder: { min: '-50', max: '100' },
  },
  {
    id: 'roa',
    label: 'ROA',
    type: 'range',
    group: 'profitability',
    minKey: 'roa_min',
    maxKey: 'roa_max',
    step: 1,
    suffix: '%',
    placeholder: { min: '-20', max: '50' },
  },
  {
    id: 'gross_margin',
    label: 'Gross Margin',
    type: 'range',
    group: 'profitability',
    minKey: 'gross_margin_min',
    maxKey: 'gross_margin_max',
    step: 1,
    suffix: '%',
    placeholder: { min: '0', max: '100' },
  },
  {
    id: 'net_margin',
    label: 'Net Margin',
    type: 'range',
    group: 'profitability',
    minKey: 'net_margin_min',
    maxKey: 'net_margin_max',
    step: 1,
    suffix: '%',
    placeholder: { min: '-50', max: '80' },
  },

  // Financial Health
  {
    id: 'debt_to_equity',
    label: 'Debt/Equity',
    type: 'range',
    group: 'financial_health',
    minKey: 'de_min',
    maxKey: 'de_max',
    step: 0.1,
    placeholder: { min: '0', max: '5' },
  },
  {
    id: 'current_ratio',
    label: 'Current Ratio',
    type: 'range',
    group: 'financial_health',
    minKey: 'current_ratio_min',
    maxKey: 'current_ratio_max',
    step: 0.1,
    placeholder: { min: '0', max: '10' },
  },

  // Growth
  {
    id: 'revenue_growth',
    label: 'Revenue Growth (YoY)',
    type: 'range',
    group: 'growth',
    minKey: 'revenue_growth_min',
    maxKey: 'revenue_growth_max',
    step: 5,
    suffix: '%',
    placeholder: { min: '-50', max: '100' },
  },
  {
    id: 'eps_growth',
    label: 'EPS Growth (YoY)',
    type: 'range',
    group: 'growth',
    minKey: 'eps_growth_min',
    maxKey: 'eps_growth_max',
    step: 5,
    suffix: '%',
    placeholder: { min: '-50', max: '100' },
  },

  // Dividends
  {
    id: 'dividend_yield',
    label: 'Dividend Yield',
    type: 'range',
    group: 'dividends',
    minKey: 'dividend_yield_min',
    maxKey: 'dividend_yield_max',
    step: 0.1,
    suffix: '%',
    placeholder: { min: '0', max: '15' },
  },
  {
    id: 'payout_ratio',
    label: 'Payout Ratio',
    type: 'range',
    group: 'dividends',
    minKey: 'payout_ratio_min',
    maxKey: 'payout_ratio_max',
    step: 5,
    suffix: '%',
    placeholder: { min: '0', max: '100' },
  },

  // Risk & Value
  {
    id: 'beta',
    label: 'Beta',
    type: 'range',
    group: 'risk',
    minKey: 'beta_min',
    maxKey: 'beta_max',
    step: 0.1,
    placeholder: { min: '0', max: '3' },
  },
  {
    id: 'dcf_upside',
    label: 'DCF Upside',
    type: 'range',
    group: 'risk',
    minKey: 'dcf_upside_min',
    maxKey: 'dcf_upside_max',
    step: 5,
    suffix: '%',
    placeholder: { min: '-50', max: '200' },
  },

  // IC Score
  {
    id: 'ic_score',
    label: 'IC Score (Overall)',
    type: 'range',
    group: 'ic_score',
    minKey: 'ic_score_min',
    maxKey: 'ic_score_max',
    step: 1,
    placeholder: { min: '0', max: '100' },
  },
  {
    id: 'value_score',
    label: 'Value Score',
    type: 'range',
    group: 'ic_score',
    minKey: 'value_score_min',
    maxKey: 'value_score_max',
    step: 1,
    placeholder: { min: '0', max: '100' },
  },
  {
    id: 'growth_score',
    label: 'Growth Score',
    type: 'range',
    group: 'ic_score',
    minKey: 'growth_score_min',
    maxKey: 'growth_score_max',
    step: 1,
    placeholder: { min: '0', max: '100' },
  },
  {
    id: 'momentum_score',
    label: 'Momentum Score',
    type: 'range',
    group: 'ic_score',
    minKey: 'momentum_score_min',
    maxKey: 'momentum_score_max',
    step: 1,
    placeholder: { min: '0', max: '100' },
  },
  {
    id: 'technical_score',
    label: 'Technical Score',
    type: 'range',
    group: 'ic_score',
    minKey: 'technical_score_min',
    maxKey: 'technical_score_max',
    step: 1,
    placeholder: { min: '0', max: '100' },
  },
];

// Table column definitions
interface ColumnDef {
  key: ScreenerSortField;
  label: string;
  align: 'left' | 'right';
  format: (stock: ScreenerStock) => string;
  colorize?: (stock: ScreenerStock) => string;
  width?: string;
}

const TABLE_COLUMNS: ColumnDef[] = [
  {
    key: 'symbol',
    label: 'Symbol',
    align: 'left',
    format: (s) => s.symbol,
  },
  {
    key: 'name',
    label: 'Name',
    align: 'left',
    format: (s) => s.name,
    width: 'max-w-xs truncate',
  },
  {
    key: 'market_cap',
    label: 'Market Cap',
    align: 'right',
    format: (s) => (s.market_cap != null ? formatLargeNumber(s.market_cap) : '—'),
  },
  {
    key: 'price',
    label: 'Price',
    align: 'right',
    format: (s) => (s.price != null ? `$${safeToFixed(s.price, 2)}` : '—'),
  },
  {
    key: 'pe_ratio',
    label: 'P/E',
    align: 'right',
    format: (s) => (s.pe_ratio != null ? safeToFixed(s.pe_ratio, 1) : '—'),
  },
  {
    key: 'roe',
    label: 'ROE',
    align: 'right',
    format: (s) => (s.roe != null ? `${safeToFixed(s.roe, 1)}%` : '—'),
    colorize: (s) => (s.roe != null ? (s.roe >= 0 ? 'text-ic-positive' : 'text-ic-negative') : ''),
  },
  {
    key: 'dividend_yield',
    label: 'Div Yield',
    align: 'right',
    format: (s) => (s.dividend_yield != null ? `${safeToFixed(s.dividend_yield, 2)}%` : '—'),
  },
  {
    key: 'revenue_growth',
    label: 'Rev Growth',
    align: 'right',
    format: (s) =>
      s.revenue_growth != null
        ? `${s.revenue_growth >= 0 ? '+' : ''}${safeToFixed(s.revenue_growth, 1)}%`
        : '—',
    colorize: (s) =>
      s.revenue_growth != null
        ? s.revenue_growth >= 0
          ? 'text-ic-positive'
          : 'text-ic-negative'
        : '',
  },
  {
    key: 'beta',
    label: 'Beta',
    align: 'right',
    format: (s) => (s.beta != null ? safeToFixed(s.beta, 2) : '—'),
  },
  {
    key: 'ic_score',
    label: 'IC Score',
    align: 'right',
    format: (s) => (s.ic_score != null ? String(Math.round(s.ic_score)) : '—'),
  },
];

// ============================================================================
// nuqs URL state configuration
// ============================================================================

const urlStateConfig = {
  // Pagination & sorting
  page: parseAsInteger.withDefault(1),
  limit: parseAsInteger.withDefault(ITEMS_PER_PAGE),
  sort: parseAsString.withDefault('market_cap'),
  order: parseAsString.withDefault('desc'),

  // Categorical filters
  sectors: parseAsString, // comma-separated

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

// ============================================================================
// Page Component
// ============================================================================

// Helper to safely read a numeric value from urlState by key
function getUrlStateValue(state: Record<string, unknown>, key: string): number | null {
  const val = state[key];
  return typeof val === 'number' ? val : null;
}

// Wrap in Suspense because nuqs uses useSearchParams internally,
// which requires a Suspense boundary in Next.js App Router.
export default function ScreenerPage() {
  return (
    <Suspense
      fallback={
        <div className="min-h-screen bg-ic-bg-primary flex items-center justify-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-ic-blue"></div>
        </div>
      }
    >
      <ScreenerContent />
    </Suspense>
  );
}

function ScreenerContent() {
  // URL state — all filters, sorting, and pagination live in the URL
  const [urlState, setUrlState] = useQueryStates(urlStateConfig, {
    history: 'push',
    throttleMs: 300,
    shallow: false,
  });

  // Build API params from URL state (strip nulls)
  const apiParams: ScreenerApiParams = {};
  for (const [key, val] of Object.entries(urlState)) {
    if (val !== null && val !== undefined) {
      (apiParams as Record<string, unknown>)[key] = val;
    }
  }

  // Server-side data fetching via SWR
  const { stocks, meta, isLoading, isValidating } = useScreener(apiParams);

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
        page: 1,
      }));
    },
    [setUrlState]
  );

  const applyPreset = useCallback(
    (preset: ScreenerPreset) => {
      // Clear all filter params, then apply preset params
      const cleared: Record<string, null> = {};
      for (const key of Object.keys(urlStateConfig)) {
        if (key !== 'limit') cleared[key] = null;
      }
      setUrlState({
        ...cleared,
        page: 1,
        sort: 'market_cap',
        order: 'desc',
        limit: ITEMS_PER_PAGE,
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
    cleared.page = 1;
    cleared.limit = ITEMS_PER_PAGE;
    cleared.sort = 'market_cap';
    cleared.order = 'desc';
    setUrlState(cleared as typeof urlState);
  }, [setUrlState]);

  // Count active filters (non-default, non-pagination)
  const activeFilterCount = Object.entries(urlState).filter(([key, val]) => {
    if (['page', 'limit', 'sort', 'order'].includes(key)) return false;
    return val !== null;
  }).length;

  // Parse sectors from comma-separated string
  const selectedSectors = urlState.sectors ? urlState.sectors.split(',') : [];

  // Pagination
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
            Filter and discover stocks across {total > 0 ? total.toLocaleString() : '5,600+'} US
            stocks
          </p>
        </div>
      </div>

      <div className="max-w-[1400px] mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Preset Screens */}
        <div className="mb-6">
          <h3 className="text-sm font-medium text-ic-text-secondary mb-3">Quick Screens</h3>
          <div className="flex flex-wrap gap-2">
            {PRESET_SCREENS.map((preset) => (
              <button
                key={preset.id}
                onClick={() => applyPreset(preset)}
                className="px-4 py-2 bg-ic-surface border border-ic-border rounded-lg hover:bg-ic-surface-hover transition-colors text-left"
              >
                <div className="text-sm font-medium text-ic-text-primary">{preset.name}</div>
                <div className="text-xs text-ic-text-muted">{preset.description}</div>
              </button>
            ))}
          </div>
        </div>

        <div className="flex gap-6">
          {/* Filters Sidebar */}
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
                    onClick={clearFilters}
                    className="text-sm text-ic-blue hover:text-ic-blue-hover transition-colors"
                  >
                    Clear All
                  </button>
                )}
              </div>

              <div className="space-y-5 max-h-[calc(100vh-200px)] overflow-y-auto pr-1">
                {FILTER_GROUPS.map((group) => {
                  const groupFilters = FILTER_DEFS.filter((f) => f.group === group.id);
                  if (groupFilters.length === 0) return null;

                  return (
                    <div key={group.id}>
                      <h4 className="text-xs font-semibold text-ic-text-muted uppercase tracking-wider mb-2">
                        {group.label}
                      </h4>
                      <div className="space-y-3">
                        {groupFilters.map((filter) => {
                          if (filter.type === 'multiselect' && filter.id === 'sector') {
                            return (
                              <SectorFilter
                                key={filter.id}
                                label={filter.label}
                                options={filter.options!}
                                selected={selectedSectors}
                                onChange={handleSectorsChange}
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
                                  handleRangeChange(filter.minKey!, filter.maxKey!, min, max)
                                }
                                step={filter.step}
                                suffix={filter.suffix}
                                placeholder={filter.placeholder}
                              />
                            );
                          }
                          return null;
                        })}
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>

          {/* Results Table */}
          <div className="flex-1 min-w-0">
            <div className="bg-ic-surface rounded-lg border border-ic-border overflow-hidden">
              {/* Results Header */}
              <div className="px-4 py-3 border-b border-ic-border flex items-center justify-between">
                <span className="text-sm text-ic-text-muted">
                  {isLoading ? 'Loading...' : `${total.toLocaleString()} stocks found`}
                  {isValidating && !isLoading && (
                    <span className="ml-2 text-ic-text-dim">(updating...)</span>
                  )}
                </span>
              </div>

              {/* Table */}
              {isLoading && stocks.length === 0 ? (
                <div className="p-8 text-center">
                  <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-ic-blue mx-auto"></div>
                  <p className="mt-4 text-ic-text-muted">Loading stocks...</p>
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full">
                    <thead className="bg-ic-bg-secondary">
                      <tr>
                        {TABLE_COLUMNS.map((col) => (
                          <th
                            key={col.key}
                            className={cn(
                              'px-4 py-3 text-xs font-medium text-ic-text-muted uppercase tracking-wider cursor-pointer hover:bg-ic-bg-tertiary transition-colors whitespace-nowrap',
                              col.align === 'right' ? 'text-right' : 'text-left'
                            )}
                            onClick={() => handleSort(col.key)}
                          >
                            {col.label}
                            {urlState.sort === col.key && (
                              <span className="ml-1">{urlState.order === 'asc' ? '↑' : '↓'}</span>
                            )}
                          </th>
                        ))}
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-ic-border-subtle">
                      {stocks.map((stock) => (
                        <tr
                          key={stock.symbol}
                          className="hover:bg-ic-surface-hover transition-colors"
                        >
                          {TABLE_COLUMNS.map((col) => {
                            // Special rendering for symbol (link) and name (with sector subtitle)
                            if (col.key === 'symbol') {
                              return (
                                <td key={col.key} className="px-4 py-3">
                                  <Link
                                    href={`/ticker/${stock.symbol}`}
                                    className="font-medium text-ic-blue hover:text-ic-blue-hover transition-colors"
                                  >
                                    {stock.symbol}
                                  </Link>
                                </td>
                              );
                            }
                            if (col.key === 'name') {
                              return (
                                <td key={col.key} className="px-4 py-3 text-ic-text-primary">
                                  <div className="max-w-xs truncate">{stock.name}</div>
                                  <div className="text-xs text-ic-text-muted">{stock.sector}</div>
                                </td>
                              );
                            }
                            if (col.key === 'ic_score') {
                              return (
                                <td key={col.key} className="px-4 py-3 text-right">
                                  {stock.ic_score != null ? (
                                    <span
                                      className={cn(
                                        'inline-flex px-2 py-0.5 rounded-full text-sm font-medium',
                                        stock.ic_score >= 70
                                          ? 'bg-ic-positive-bg text-ic-positive'
                                          : stock.ic_score >= 40
                                            ? 'bg-ic-warning-bg text-ic-warning'
                                            : 'bg-ic-negative-bg text-ic-negative'
                                      )}
                                    >
                                      {Math.round(stock.ic_score)}
                                    </span>
                                  ) : (
                                    <span className="text-ic-text-dim">—</span>
                                  )}
                                </td>
                              );
                            }

                            // Generic column rendering
                            const formatted = col.format(stock);
                            const colorClass = col.colorize ? col.colorize(stock) : '';
                            return (
                              <td
                                key={col.key}
                                className={cn(
                                  'px-4 py-3',
                                  col.align === 'right' ? 'text-right' : 'text-left',
                                  colorClass || 'text-ic-text-primary'
                                )}
                              >
                                <span className={col.width}>{formatted}</span>
                              </td>
                            );
                          })}
                        </tr>
                      ))}
                      {stocks.length === 0 && !isLoading && (
                        <tr>
                          <td
                            colSpan={TABLE_COLUMNS.length}
                            className="px-4 py-12 text-center text-ic-text-muted"
                          >
                            No stocks match your filters. Try adjusting or clearing filters.
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              )}

              {/* Pagination */}
              {totalPages > 1 && (
                <div className="px-4 py-3 border-t border-ic-border flex items-center justify-between">
                  <div className="text-sm text-ic-text-muted">
                    Showing {(currentPage - 1) * (urlState.limit ?? ITEMS_PER_PAGE) + 1} to{' '}
                    {Math.min(currentPage * (urlState.limit ?? ITEMS_PER_PAGE), total)} of{' '}
                    {total.toLocaleString()} results
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={() => handlePageChange(Math.max(1, currentPage - 1))}
                      disabled={currentPage === 1}
                      className="px-3 py-1 border border-ic-border rounded-md text-sm text-ic-text-secondary disabled:opacity-50 disabled:cursor-not-allowed hover:bg-ic-surface-hover transition-colors"
                    >
                      Previous
                    </button>
                    <span className="px-3 py-1 text-sm text-ic-text-muted">
                      Page {currentPage} of {totalPages}
                    </span>
                    <button
                      onClick={() => handlePageChange(Math.min(totalPages, currentPage + 1))}
                      disabled={currentPage === totalPages}
                      className="px-3 py-1 border border-ic-border rounded-md text-sm text-ic-text-secondary disabled:opacity-50 disabled:cursor-not-allowed hover:bg-ic-surface-hover transition-colors"
                    >
                      Next
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// ============================================================================
// Filter Components
// ============================================================================

function SectorFilter({
  label,
  options,
  selected,
  onChange,
}: {
  label: string;
  options: { value: string; label: string }[];
  selected: string[];
  onChange: (values: string[]) => void;
}) {
  return (
    <div>
      <label className="block text-sm font-medium text-ic-text-secondary mb-1.5">
        {label}
        {selected.length > 0 && (
          <span className="ml-1 text-xs text-ic-text-dim">({selected.length})</span>
        )}
      </label>
      <div className="space-y-1 max-h-40 overflow-y-auto">
        {options.map((option) => (
          <label key={option.value} className="flex items-center">
            <input
              type="checkbox"
              checked={selected.includes(option.value)}
              onChange={(e) => {
                if (e.target.checked) {
                  onChange([...selected, option.value]);
                } else {
                  onChange(selected.filter((v) => v !== option.value));
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
}

function RangeFilter({
  label,
  minValue,
  maxValue,
  onChange,
  step,
  suffix,
  placeholder,
}: {
  label: string;
  minValue: number | null;
  maxValue: number | null;
  onChange: (min: number | null, max: number | null) => void;
  step?: number;
  suffix?: string;
  placeholder?: { min: string; max: string };
}) {
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

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') {
        commit();
      }
    },
    [commit]
  );

  return (
    <div>
      <label className="block text-sm font-medium text-ic-text-secondary mb-1.5">{label}</label>
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
        <span className="text-ic-text-dim">—</span>
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
