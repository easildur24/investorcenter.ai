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

type SortField = 'symbol' | 'name' | 'market_cap' | 'price' | 'change_percent' | 'pe_ratio' | 'ic_score';
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
        // Use mock data for now
        const mockStocks = generateMockStocks();
        setStocks(mockStocks);
        setFilteredStocks(mockStocks);
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
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <h1 className="text-2xl font-bold text-gray-900">Stock Screener</h1>
          <p className="mt-1 text-gray-500">
            Filter and discover stocks based on fundamental metrics
          </p>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Preset Screens */}
        <div className="mb-6">
          <h3 className="text-sm font-medium text-gray-700 mb-3">Quick Screens</h3>
          <div className="flex flex-wrap gap-2">
            {PRESET_SCREENS.map(preset => (
              <button
                key={preset.id}
                onClick={() => applyPreset(preset)}
                className="px-4 py-2 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
              >
                <div className="text-sm font-medium text-gray-900">{preset.name}</div>
                <div className="text-xs text-gray-500">{preset.description}</div>
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
            <div className="bg-white rounded-lg shadow p-4">
              <div className="flex items-center justify-between mb-4">
                <h3 className="font-semibold text-gray-900">Filters</h3>
                <button
                  onClick={clearFilters}
                  className="text-sm text-blue-600 hover:text-blue-700"
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
            <div className="bg-white rounded-lg shadow overflow-hidden">
              {/* Results Header */}
              <div className="px-4 py-3 border-b border-gray-200 flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <button
                    onClick={() => setShowFilters(!showFilters)}
                    className="text-sm text-gray-600 hover:text-gray-900"
                  >
                    {showFilters ? 'Hide Filters' : 'Show Filters'}
                  </button>
                  <span className="text-sm text-gray-500">
                    {filteredStocks.length} stocks found
                  </span>
                </div>
              </div>

              {/* Table */}
              {loading ? (
                <div className="p-8 text-center">
                  <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
                  <p className="mt-4 text-gray-500">Loading stocks...</p>
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full">
                    <thead className="bg-gray-50">
                      <tr>
                        <th
                          className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100"
                          onClick={() => handleSort('symbol')}
                        >
                          Symbol <SortIcon field="symbol" />
                        </th>
                        <th
                          className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100"
                          onClick={() => handleSort('name')}
                        >
                          Name <SortIcon field="name" />
                        </th>
                        <th
                          className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100"
                          onClick={() => handleSort('market_cap')}
                        >
                          Market Cap <SortIcon field="market_cap" />
                        </th>
                        <th
                          className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100"
                          onClick={() => handleSort('price')}
                        >
                          Price <SortIcon field="price" />
                        </th>
                        <th
                          className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100"
                          onClick={() => handleSort('change_percent')}
                        >
                          Change <SortIcon field="change_percent" />
                        </th>
                        <th
                          className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100"
                          onClick={() => handleSort('pe_ratio')}
                        >
                          P/E <SortIcon field="pe_ratio" />
                        </th>
                        <th
                          className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100"
                          onClick={() => handleSort('ic_score')}
                        >
                          IC Score <SortIcon field="ic_score" />
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200">
                      {paginatedStocks.map(stock => (
                        <tr key={stock.symbol} className="hover:bg-gray-50">
                          <td className="px-4 py-3">
                            <Link
                              href={`/ticker/${stock.symbol}`}
                              className="font-medium text-blue-600 hover:text-blue-700"
                            >
                              {stock.symbol}
                            </Link>
                          </td>
                          <td className="px-4 py-3 text-gray-900">
                            <div className="max-w-xs truncate">{stock.name}</div>
                            <div className="text-xs text-gray-500">{stock.sector}</div>
                          </td>
                          <td className="px-4 py-3 text-right text-gray-900">
                            {formatLargeNumber(stock.market_cap)}
                          </td>
                          <td className="px-4 py-3 text-right font-medium text-gray-900">
                            ${safeToFixed(stock.price, 2)}
                          </td>
                          <td className={cn(
                            'px-4 py-3 text-right font-medium',
                            stock.change_percent >= 0 ? 'text-green-600' : 'text-red-600'
                          )}>
                            {stock.change_percent >= 0 ? '+' : ''}
                            {safeToFixed(stock.change_percent, 2)}%
                          </td>
                          <td className="px-4 py-3 text-right text-gray-900">
                            {safeToFixed(stock.pe_ratio, 1)}
                          </td>
                          <td className="px-4 py-3 text-right">
                            <span className={cn(
                              'inline-flex px-2 py-0.5 rounded-full text-sm font-medium',
                              stock.ic_score >= 70 ? 'bg-green-100 text-green-700' :
                              stock.ic_score >= 40 ? 'bg-yellow-100 text-yellow-700' :
                              'bg-red-100 text-red-700'
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
                <div className="px-4 py-3 border-t border-gray-200 flex items-center justify-between">
                  <div className="text-sm text-gray-500">
                    Showing {(currentPage - 1) * itemsPerPage + 1} to{' '}
                    {Math.min(currentPage * itemsPerPage, filteredStocks.length)} of{' '}
                    {filteredStocks.length} results
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                      disabled={currentPage === 1}
                      className="px-3 py-1 border border-gray-200 rounded-md text-sm disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50"
                    >
                      Previous
                    </button>
                    <span className="px-3 py-1 text-sm text-gray-600">
                      Page {currentPage} of {totalPages}
                    </span>
                    <button
                      onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                      disabled={currentPage === totalPages}
                      className="px-3 py-1 border border-gray-200 rounded-md text-sm disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50"
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
        <label className="block text-sm font-medium text-gray-700 mb-2">
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
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm text-gray-600">{option.label}</span>
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
        <label className="block text-sm font-medium text-gray-700 mb-2">
          {filter.label}
        </label>
        <div className="flex gap-2 items-center">
          <input
            type="number"
            placeholder="Min"
            value={rangeValue.min ?? ''}
            onChange={(e) => onChange({ ...rangeValue, min: e.target.value ? Number(e.target.value) : undefined })}
            className="w-20 px-2 py-1 text-sm border border-gray-200 rounded-md"
            step={filter.step}
          />
          <span className="text-gray-400">-</span>
          <input
            type="number"
            placeholder="Max"
            value={rangeValue.max ?? ''}
            onChange={(e) => onChange({ ...rangeValue, max: e.target.value ? Number(e.target.value) : undefined })}
            className="w-20 px-2 py-1 text-sm border border-gray-200 rounded-md"
            step={filter.step}
          />
          {filter.suffix && <span className="text-sm text-gray-500">{filter.suffix}</span>}
        </div>
      </div>
    );
  }

  return null;
}

// Mock data generator for development
function generateMockStocks(): Stock[] {
  const symbols = [
    'AAPL', 'MSFT', 'GOOGL', 'AMZN', 'META', 'NVDA', 'TSLA', 'BRK.B', 'JPM', 'JNJ',
    'V', 'PG', 'UNH', 'HD', 'MA', 'DIS', 'PYPL', 'NFLX', 'ADBE', 'CRM',
    'INTC', 'CSCO', 'PEP', 'ABT', 'TMO', 'NKE', 'MRK', 'WMT', 'VZ', 'T',
  ];

  const sectors = [
    'Technology', 'Healthcare', 'Financial Services', 'Consumer Cyclical',
    'Consumer Defensive', 'Industrials', 'Communication Services',
  ];

  return symbols.map(symbol => ({
    symbol,
    name: `${symbol} Inc.`,
    sector: sectors[Math.floor(Math.random() * sectors.length)],
    industry: 'Various',
    market_cap: Math.random() * 2e12 + 10e9,
    price: Math.random() * 500 + 50,
    change_percent: (Math.random() - 0.5) * 10,
    pe_ratio: Math.random() * 50 + 5,
    pb_ratio: Math.random() * 10 + 1,
    ps_ratio: Math.random() * 15 + 1,
    roe: Math.random() * 40,
    revenue_growth: (Math.random() - 0.3) * 50,
    earnings_growth: (Math.random() - 0.3) * 60,
    dividend_yield: Math.random() * 5,
    beta: Math.random() * 1.5 + 0.5,
    ic_score: Math.floor(Math.random() * 60 + 40),
  }));
}
