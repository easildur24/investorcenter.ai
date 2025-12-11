'use client';

import { useState, useEffect, useCallback } from 'react';
import Link from 'next/link';
import { cn, safeToFixed, formatLargeNumber, formatPercent, safeParseNumber } from '@/lib/utils';

interface Stock {
  symbol: string;
  name: string;
  sector: string;
  industry: string;
  market_cap: number;
  price: number;
  change_percent: number;
  pe_ratio: number;
  pb_ratio: number;
  ps_ratio: number;
  roe: number;
  revenue_growth: number;
  earnings_growth: number;
  dividend_yield: number;
  beta: number;
  ic_score: number;
}

interface FilterConfig {
  id: string;
  label: string;
  field: string;
  type: 'range' | 'select' | 'multiselect';
  options?: { value: string; label: string }[];
  min?: number;
  max?: number;
  step?: number;
  suffix?: string;
}

interface FilterState {
  [key: string]: any;
}

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

const MARKET_CAP_OPTIONS = [
  { value: 'mega', label: 'Mega Cap ($200B+)' },
  { value: 'large', label: 'Large Cap ($10B-$200B)' },
  { value: 'mid', label: 'Mid Cap ($2B-$10B)' },
  { value: 'small', label: 'Small Cap ($300M-$2B)' },
  { value: 'micro', label: 'Micro Cap (<$300M)' },
];

const FILTERS: FilterConfig[] = [
  {
    id: 'sector',
    label: 'Sector',
    field: 'sector',
    type: 'multiselect',
    options: SECTORS.map(s => ({ value: s, label: s })),
  },
  {
    id: 'marketCap',
    label: 'Market Cap',
    field: 'market_cap',
    type: 'multiselect',
    options: MARKET_CAP_OPTIONS,
  },
  {
    id: 'peRatio',
    label: 'P/E Ratio',
    field: 'pe_ratio',
    type: 'range',
    min: 0,
    max: 100,
    step: 1,
  },
  {
    id: 'dividendYield',
    label: 'Dividend Yield',
    field: 'dividend_yield',
    type: 'range',
    min: 0,
    max: 10,
    step: 0.1,
    suffix: '%',
  },
  {
    id: 'revenueGrowth',
    label: 'Revenue Growth (YoY)',
    field: 'revenue_growth',
    type: 'range',
    min: -50,
    max: 100,
    step: 5,
    suffix: '%',
  },
  {
    id: 'icScore',
    label: 'IC Score',
    field: 'ic_score',
    type: 'range',
    min: 0,
    max: 100,
    step: 1,
  },
];

const PRESET_SCREENS = [
  {
    id: 'value',
    name: 'Value Stocks',
    description: 'Low P/E, high dividend yield',
    filters: { peRatio: { max: 15 }, dividendYield: { min: 2 } },
  },
  {
    id: 'growth',
    name: 'Growth Stocks',
    description: 'High revenue growth',
    filters: { revenueGrowth: { min: 20 } },
  },
  {
    id: 'quality',
    name: 'Quality Stocks',
    description: 'High IC Score, large cap',
    filters: { icScore: { min: 70 }, marketCap: ['mega', 'large'] },
  },
  {
    id: 'dividend',
    name: 'Dividend Champions',
    description: 'High yield, stable companies',
    filters: { dividendYield: { min: 3 }, marketCap: ['mega', 'large'] },
  },
];

type SortField = 'symbol' | 'name' | 'market_cap' | 'price' | 'change_percent' | 'pe_ratio' | 'dividend_yield' | 'revenue_growth' | 'ic_score';
type SortDirection = 'asc' | 'desc';

export default function ScreenerPage() {
  const [stocks, setStocks] = useState<Stock[]>([]);
  const [filteredStocks, setFilteredStocks] = useState<Stock[]>([]);
  const [loading, setLoading] = useState(true);
  const [filters, setFilters] = useState<FilterState>({});
  const [sortField, setSortField] = useState<SortField>('market_cap');
  const [sortDirection, setSortDirection] = useState<SortDirection>('desc');
  const [showFilters, setShowFilters] = useState(true);
  const [currentPage, setCurrentPage] = useState(1);
  const itemsPerPage = 25;

  // Fetch stocks data
  useEffect(() => {
    const fetchStocks = async () => {
      try {
        setLoading(true);
        const response = await fetch('/api/v1/screener/stocks');
        if (!response.ok) {
          throw new Error('Failed to fetch stocks');
        }
        const result = await response.json();
        setStocks(result.data || []);
        setFilteredStocks(result.data || []);
      } catch (error) {
        console.error('Error fetching stocks:', error);
        setStocks([]);
        setFilteredStocks([]);
      } finally {
        setLoading(false);
      }
    };

    fetchStocks();
  }, []);

  // Apply filters and sorting
  const applyFilters = useCallback(() => {
    let result = [...stocks];

    // Apply filters
    Object.entries(filters).forEach(([key, value]) => {
      if (!value || (Array.isArray(value) && value.length === 0)) return;

      const filter = FILTERS.find(f => f.id === key);
      if (!filter) return;

      if (filter.type === 'multiselect' && Array.isArray(value)) {
        if (filter.id === 'marketCap') {
          result = result.filter(stock => {
            const cap = stock.market_cap;
            return value.some((v: string) => {
              switch (v) {
                case 'mega': return cap >= 200e9;
                case 'large': return cap >= 10e9 && cap < 200e9;
                case 'mid': return cap >= 2e9 && cap < 10e9;
                case 'small': return cap >= 300e6 && cap < 2e9;
                case 'micro': return cap < 300e6;
                default: return true;
              }
            });
          });
        } else if (filter.id === 'sector') {
          result = result.filter(stock => value.includes(stock.sector));
        }
      } else if (filter.type === 'range' && typeof value === 'object') {
        const { min, max } = value;
        result = result.filter(stock => {
          const fieldValue = (stock as any)[filter.field];
          if (min !== undefined && fieldValue < min) return false;
          if (max !== undefined && fieldValue > max) return false;
          return true;
        });
      }
    });

    // Apply sorting
    result.sort((a, b) => {
      const aValue = (a as any)[sortField];
      const bValue = (b as any)[sortField];

      if (typeof aValue === 'string') {
        return sortDirection === 'asc'
          ? aValue.localeCompare(bValue)
          : bValue.localeCompare(aValue);
      }

      return sortDirection === 'asc' ? aValue - bValue : bValue - aValue;
    });

    setFilteredStocks(result);
    setCurrentPage(1);
  }, [stocks, filters, sortField, sortDirection]);

  useEffect(() => {
    applyFilters();
  }, [applyFilters]);

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortDirection('desc');
    }
  };

  const handleFilterChange = (filterId: string, value: any) => {
    setFilters(prev => ({
      ...prev,
      [filterId]: value,
    }));
  };

  const applyPreset = (preset: typeof PRESET_SCREENS[0]) => {
    setFilters(preset.filters);
  };

  const clearFilters = () => {
    setFilters({});
  };

  // Pagination
  const totalPages = Math.ceil(filteredStocks.length / itemsPerPage);
  const paginatedStocks = filteredStocks.slice(
    (currentPage - 1) * itemsPerPage,
    currentPage * itemsPerPage
  );

  const SortIcon = ({ field }: { field: SortField }) => {
    if (sortField !== field) return null;
    return sortDirection === 'asc' ? (
      <span className="ml-1">↑</span>
    ) : (
      <span className="ml-1">↓</span>
    );
  };

  return (
    <div className="min-h-screen bg-ic-bg-primary">
      {/* Header */}
      <div className="bg-ic-surface border-b border-ic-border">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <h1 className="text-2xl font-bold text-ic-text-primary">Stock Screener</h1>
          <p className="mt-1 text-ic-text-muted">
            Filter and discover stocks based on fundamental metrics
          </p>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Preset Screens */}
        <div className="mb-6">
          <h3 className="text-sm font-medium text-ic-text-secondary mb-3">Quick Screens</h3>
          <div className="flex flex-wrap gap-2">
            {PRESET_SCREENS.map(preset => (
              <button
                key={preset.id}
                onClick={() => applyPreset(preset)}
                className="px-4 py-2 bg-ic-surface border border-ic-border rounded-lg hover:bg-ic-surface-hover transition-colors"
              >
                <div className="text-sm font-medium text-ic-text-primary">{preset.name}</div>
                <div className="text-xs text-ic-text-muted">{preset.description}</div>
              </button>
            ))}
          </div>
        </div>

        <div className="flex gap-6">
          {/* Filters Sidebar */}
          <div className={cn(
            'w-64 flex-shrink-0 transition-all',
            showFilters ? 'block' : 'hidden'
          )}>
            <div className="bg-ic-surface rounded-lg border border-ic-border p-4">
              <div className="flex items-center justify-between mb-4">
                <h3 className="font-semibold text-ic-text-primary">Filters</h3>
                <button
                  onClick={clearFilters}
                  className="text-sm text-ic-blue hover:text-ic-blue-hover transition-colors"
                >
                  Clear All
                </button>
              </div>

              <div className="space-y-6">
                {FILTERS.map(filter => (
                  <FilterInput
                    key={filter.id}
                    filter={filter}
                    value={filters[filter.id]}
                    onChange={(value) => handleFilterChange(filter.id, value)}
                  />
                ))}
              </div>
            </div>
          </div>

          {/* Results Table */}
          <div className="flex-1">
            <div className="bg-ic-surface rounded-lg border border-ic-border overflow-hidden">
              {/* Results Header */}
              <div className="px-4 py-3 border-b border-ic-border flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <button
                    onClick={() => setShowFilters(!showFilters)}
                    className="text-sm text-ic-text-muted hover:text-ic-text-primary transition-colors"
                  >
                    {showFilters ? 'Hide Filters' : 'Show Filters'}
                  </button>
                  <span className="text-sm text-ic-text-muted">
                    {filteredStocks.length} stocks found
                  </span>
                </div>
              </div>

              {/* Table */}
              {loading ? (
                <div className="p-8 text-center">
                  <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-ic-blue mx-auto"></div>
                  <p className="mt-4 text-ic-text-muted">Loading stocks...</p>
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full">
                    <thead className="bg-ic-bg-secondary">
                      <tr>
                        <th
                          className="px-4 py-3 text-left text-xs font-medium text-ic-text-muted uppercase tracking-wider cursor-pointer hover:bg-ic-bg-tertiary transition-colors"
                          onClick={() => handleSort('symbol')}
                        >
                          Symbol <SortIcon field="symbol" />
                        </th>
                        <th
                          className="px-4 py-3 text-left text-xs font-medium text-ic-text-muted uppercase tracking-wider cursor-pointer hover:bg-ic-bg-tertiary transition-colors"
                          onClick={() => handleSort('name')}
                        >
                          Name <SortIcon field="name" />
                        </th>
                        <th
                          className="px-4 py-3 text-right text-xs font-medium text-ic-text-muted uppercase tracking-wider cursor-pointer hover:bg-ic-bg-tertiary transition-colors"
                          onClick={() => handleSort('market_cap')}
                        >
                          Market Cap <SortIcon field="market_cap" />
                        </th>
                        <th
                          className="px-4 py-3 text-right text-xs font-medium text-ic-text-muted uppercase tracking-wider cursor-pointer hover:bg-ic-bg-tertiary transition-colors"
                          onClick={() => handleSort('price')}
                        >
                          Price <SortIcon field="price" />
                        </th>
                        <th
                          className="px-4 py-3 text-right text-xs font-medium text-ic-text-muted uppercase tracking-wider cursor-pointer hover:bg-ic-bg-tertiary transition-colors"
                          onClick={() => handleSort('change_percent')}
                        >
                          Change <SortIcon field="change_percent" />
                        </th>
                        <th
                          className="px-4 py-3 text-right text-xs font-medium text-ic-text-muted uppercase tracking-wider cursor-pointer hover:bg-ic-bg-tertiary transition-colors"
                          onClick={() => handleSort('pe_ratio')}
                        >
                          P/E <SortIcon field="pe_ratio" />
                        </th>
                        <th
                          className="px-4 py-3 text-right text-xs font-medium text-ic-text-muted uppercase tracking-wider cursor-pointer hover:bg-ic-bg-tertiary transition-colors"
                          onClick={() => handleSort('dividend_yield')}
                        >
                          Div Yield <SortIcon field="dividend_yield" />
                        </th>
                        <th
                          className="px-4 py-3 text-right text-xs font-medium text-ic-text-muted uppercase tracking-wider cursor-pointer hover:bg-ic-bg-tertiary transition-colors"
                          onClick={() => handleSort('revenue_growth')}
                        >
                          Rev Growth <SortIcon field="revenue_growth" />
                        </th>
                        <th
                          className="px-4 py-3 text-right text-xs font-medium text-ic-text-muted uppercase tracking-wider cursor-pointer hover:bg-ic-bg-tertiary transition-colors"
                          onClick={() => handleSort('ic_score')}
                        >
                          IC Score <SortIcon field="ic_score" />
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-ic-border-subtle">
                      {paginatedStocks.map(stock => (
                        <tr key={stock.symbol} className="hover:bg-ic-surface-hover transition-colors">
                          <td className="px-4 py-3">
                            <Link
                              href={`/ticker/${stock.symbol}`}
                              className="font-medium text-ic-blue hover:text-ic-blue-hover transition-colors"
                            >
                              {stock.symbol}
                            </Link>
                          </td>
                          <td className="px-4 py-3 text-ic-text-primary">
                            <div className="max-w-xs truncate">{stock.name}</div>
                            <div className="text-xs text-ic-text-muted">{stock.sector}</div>
                          </td>
                          <td className="px-4 py-3 text-right text-ic-text-primary">
                            {formatLargeNumber(stock.market_cap)}
                          </td>
                          <td className="px-4 py-3 text-right font-medium text-ic-text-primary">
                            ${safeToFixed(stock.price, 2)}
                          </td>
                          <td className={cn(
                            'px-4 py-3 text-right font-medium',
                            stock.change_percent >= 0 ? 'text-ic-positive' : 'text-ic-negative'
                          )}>
                            {stock.change_percent >= 0 ? '+' : ''}
                            {safeToFixed(stock.change_percent, 2)}%
                          </td>
                          <td className="px-4 py-3 text-right text-ic-text-primary">
                            {safeToFixed(stock.pe_ratio, 1)}
                          </td>
                          <td className="px-4 py-3 text-right text-ic-text-primary">
                            {stock.dividend_yield ? `${safeToFixed(stock.dividend_yield, 2)}%` : '—'}
                          </td>
                          <td className={cn(
                            'px-4 py-3 text-right',
                            stock.revenue_growth >= 0 ? 'text-ic-positive' : 'text-ic-negative'
                          )}>
                            {stock.revenue_growth ? `${stock.revenue_growth >= 0 ? '+' : ''}${safeToFixed(stock.revenue_growth, 1)}%` : '—'}
                          </td>
                          <td className="px-4 py-3 text-right">
                            <span className={cn(
                              'inline-flex px-2 py-0.5 rounded-full text-sm font-medium',
                              stock.ic_score >= 70 ? 'bg-ic-positive-bg text-ic-positive' :
                              stock.ic_score >= 40 ? 'bg-ic-warning-bg text-ic-warning' :
                              'bg-ic-negative-bg text-ic-negative'
                            )}>
                              {stock.ic_score}
                            </span>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}

              {/* Pagination */}
              {totalPages > 1 && (
                <div className="px-4 py-3 border-t border-ic-border flex items-center justify-between">
                  <div className="text-sm text-ic-text-muted">
                    Showing {(currentPage - 1) * itemsPerPage + 1} to{' '}
                    {Math.min(currentPage * itemsPerPage, filteredStocks.length)} of{' '}
                    {filteredStocks.length} results
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                      disabled={currentPage === 1}
                      className="px-3 py-1 border border-ic-border rounded-md text-sm text-ic-text-secondary disabled:opacity-50 disabled:cursor-not-allowed hover:bg-ic-surface-hover transition-colors"
                    >
                      Previous
                    </button>
                    <span className="px-3 py-1 text-sm text-ic-text-muted">
                      Page {currentPage} of {totalPages}
                    </span>
                    <button
                      onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
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

// Filter Input Component
function FilterInput({
  filter,
  value,
  onChange,
}: {
  filter: FilterConfig;
  value: any;
  onChange: (value: any) => void;
}) {
  if (filter.type === 'multiselect') {
    const selectedValues = Array.isArray(value) ? value : [];

    return (
      <div>
        <label className="block text-sm font-medium text-ic-text-secondary mb-2">
          {filter.label}
        </label>
        <div className="space-y-1 max-h-40 overflow-y-auto">
          {filter.options?.map(option => (
            <label key={option.value} className="flex items-center">
              <input
                type="checkbox"
                checked={selectedValues.includes(option.value)}
                onChange={(e) => {
                  if (e.target.checked) {
                    onChange([...selectedValues, option.value]);
                  } else {
                    onChange(selectedValues.filter((v: string) => v !== option.value));
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

  if (filter.type === 'range') {
    const rangeValue = typeof value === 'object' ? value : {};

    return (
      <div>
        <label className="block text-sm font-medium text-ic-text-secondary mb-2">
          {filter.label}
        </label>
        <div className="flex gap-2 items-center">
          <input
            type="number"
            placeholder="Min"
            value={rangeValue.min ?? ''}
            onChange={(e) => onChange({ ...rangeValue, min: e.target.value ? Number(e.target.value) : undefined })}
            className="w-20 px-2 py-1 text-sm border border-ic-border rounded-md bg-ic-input-bg text-ic-text-primary"
            step={filter.step}
          />
          <span className="text-ic-text-dim">-</span>
          <input
            type="number"
            placeholder="Max"
            value={rangeValue.max ?? ''}
            onChange={(e) => onChange({ ...rangeValue, max: e.target.value ? Number(e.target.value) : undefined })}
            className="w-20 px-2 py-1 text-sm border border-ic-border rounded-md bg-ic-input-bg text-ic-text-primary"
            step={filter.step}
          />
          {filter.suffix && <span className="text-sm text-ic-text-muted">{filter.suffix}</span>}
        </div>
      </div>
    );
  }

  return null;
}
