'use client';

import React, { useState, useEffect } from 'react';
import { icScoreApi } from '@/lib/api';
import { getICScoreRatingColor, getICScoreColor } from '@/lib/types/ic-score';
import type {
  ICScoreScreenerFilters,
  ICScoreStockEntry,
  ICScoreRating,
} from '@/lib/types/ic-score';

/**
 * IC Score stock screener component
 *
 * Full-page screener with:
 * - Filters by score range, rating, sector, market cap
 * - Sortable table with key metrics
 * - Pagination
 * - Export functionality
 */
export default function ICScoreScreener() {
  const [stocks, setStocks] = useState<ICScoreStockEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [total, setTotal] = useState(0);

  // Filter state
  const [filters, setFilters] = useState<ICScoreScreenerFilters>({
    minScore: undefined,
    maxScore: undefined,
    rating: [],
    sector: [],
    minMarketCap: undefined,
    maxMarketCap: undefined,
    sortBy: 'score',
    sortOrder: 'desc',
    limit: 50,
    offset: 0,
  });

  // Fetch stocks based on filters
  useEffect(() => {
    async function fetchStocks() {
      try {
        setLoading(true);
        setError(null);

        const response = await icScoreApi.runScreener(filters);
        setStocks(response.stocks);
        setTotal(response.total);
      } catch (err) {
        console.error('Error fetching stocks:', err);
        setError(err instanceof Error ? err.message : 'Failed to load stocks');
      } finally {
        setLoading(false);
      }
    }

    fetchStocks();
  }, [filters]);

  const handleFilterChange = (newFilters: Partial<ICScoreScreenerFilters>) => {
    setFilters((prev) => ({ ...prev, ...newFilters, offset: 0 }));
  };

  const handleSort = (column: 'score' | 'marketCap' | 'change' | 'volume') => {
    setFilters((prev) => ({
      ...prev,
      sortBy: column,
      sortOrder: prev.sortBy === column && prev.sortOrder === 'desc' ? 'asc' : 'desc',
      offset: 0,
    }));
  };

  const handlePageChange = (direction: 'prev' | 'next') => {
    setFilters((prev) => ({
      ...prev,
      offset:
        direction === 'next'
          ? prev.offset! + prev.limit!
          : Math.max(0, prev.offset! - prev.limit!),
    }));
  };

  return (
    <div className="min-h-screen bg-ic-bg-secondary">
      {/* Header */}
      <div className="bg-ic-surface border-b border-ic-border shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <h1 className="text-3xl font-bold text-ic-text-primary">IC Score Stock Screener</h1>
          <p className="mt-2 text-ic-text-muted">
            Filter and discover stocks based on our proprietary IC Score ranking system
          </p>
        </div>
      </div>

      {/* Main content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="grid lg:grid-cols-4 gap-6">
          {/* Sidebar: Filters */}
          <div className="lg:col-span-1">
            <FilterPanel filters={filters} onFilterChange={handleFilterChange} />
          </div>

          {/* Main: Results table */}
          <div className="lg:col-span-3">
            {/* Results count */}
            <div className="mb-4 flex items-center justify-between">
              <div className="text-sm text-ic-text-muted">
                {loading ? (
                  'Loading...'
                ) : (
                  <>
                    Showing {filters.offset! + 1}-
                    {Math.min(filters.offset! + filters.limit!, total)} of {total} stocks
                  </>
                )}
              </div>
              <button
                onClick={() => window.location.reload()}
                className="text-sm text-ic-blue hover:text-ic-blue font-medium"
              >
                Reset Filters
              </button>
            </div>

            {/* Error state */}
            {error && (
              <div className="bg-ic-negative-bg border border-ic-border rounded-lg p-4 text-ic-negative">
                {error}
              </div>
            )}

            {/* Loading state */}
            {loading && <LoadingTable />}

            {/* Results table */}
            {!loading && !error && (
              <>
                <ResultsTable
                  stocks={stocks}
                  sortBy={filters.sortBy!}
                  sortOrder={filters.sortOrder!}
                  onSort={handleSort}
                />

                {/* Pagination */}
                <Pagination
                  offset={filters.offset!}
                  limit={filters.limit!}
                  total={total}
                  onPageChange={handlePageChange}
                />
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

/**
 * Filter panel
 */
interface FilterPanelProps {
  filters: ICScoreScreenerFilters;
  onFilterChange: (filters: Partial<ICScoreScreenerFilters>) => void;
}

function FilterPanel({ filters, onFilterChange }: FilterPanelProps) {
  const ratingOptions: ICScoreRating[] = [
    'Strong Buy',
    'Buy',
    'Hold',
    'Underperform',
    'Sell',
  ];

  const sectorOptions = [
    'Technology',
    'Healthcare',
    'Financial Services',
    'Consumer Cyclical',
    'Industrials',
    'Energy',
    'Utilities',
    'Real Estate',
    'Basic Materials',
    'Consumer Defensive',
    'Communication Services',
  ];

  return (
    <div className="bg-ic-surface rounded-lg shadow border border-ic-border p-4 space-y-6 sticky top-4">
      <h3 className="font-semibold text-ic-text-primary">Filters</h3>

      {/* IC Score range */}
      <div>
        <label className="block text-sm font-medium text-ic-text-secondary mb-2">
          IC Score Range
        </label>
        <div className="grid grid-cols-2 gap-2">
          <input
            type="number"
            placeholder="Min"
            min="0"
            max="100"
            value={filters.minScore || ''}
            onChange={(e) =>
              onFilterChange({ minScore: e.target.value ? Number(e.target.value) : undefined })
            }
            className="px-3 py-2 border border-ic-border rounded-md text-sm focus:ring-blue-500 focus:border-blue-500"
          />
          <input
            type="number"
            placeholder="Max"
            min="0"
            max="100"
            value={filters.maxScore || ''}
            onChange={(e) =>
              onFilterChange({ maxScore: e.target.value ? Number(e.target.value) : undefined })
            }
            className="px-3 py-2 border border-ic-border rounded-md text-sm focus:ring-blue-500 focus:border-blue-500"
          />
        </div>
      </div>

      {/* Rating filter */}
      <div>
        <label className="block text-sm font-medium text-ic-text-secondary mb-2">Rating</label>
        <div className="space-y-2">
          {ratingOptions.map((rating) => (
            <label key={rating} className="flex items-center text-sm">
              <input
                type="checkbox"
                checked={filters.rating?.includes(rating)}
                onChange={(e) => {
                  const newRatings = e.target.checked
                    ? [...(filters.rating || []), rating]
                    : (filters.rating || []).filter((r) => r !== rating);
                  onFilterChange({ rating: newRatings });
                }}
                className="mr-2 rounded text-blue-600 focus:ring-blue-500"
              />
              <span className="text-ic-text-secondary">{rating}</span>
            </label>
          ))}
        </div>
      </div>

      {/* Sector filter */}
      <div>
        <label className="block text-sm font-medium text-ic-text-secondary mb-2">Sector</label>
        <select
          multiple
          size={6}
          value={filters.sector || []}
          onChange={(e) => {
            const selected = Array.from(e.target.selectedOptions, (option) => option.value);
            onFilterChange({ sector: selected });
          }}
          className="w-full px-3 py-2 border border-ic-border rounded-md text-sm focus:ring-blue-500 focus:border-blue-500"
        >
          {sectorOptions.map((sector) => (
            <option key={sector} value={sector}>
              {sector}
            </option>
          ))}
        </select>
        <p className="text-xs text-ic-text-muted mt-1">Hold Ctrl/Cmd to select multiple</p>
      </div>

      {/* Market cap filter */}
      <div>
        <label className="block text-sm font-medium text-ic-text-secondary mb-2">
          Market Cap ($B)
        </label>
        <div className="grid grid-cols-2 gap-2">
          <input
            type="number"
            placeholder="Min"
            value={filters.minMarketCap ? filters.minMarketCap / 1e9 : ''}
            onChange={(e) =>
              onFilterChange({
                minMarketCap: e.target.value ? Number(e.target.value) * 1e9 : undefined,
              })
            }
            className="px-3 py-2 border border-ic-border rounded-md text-sm focus:ring-blue-500 focus:border-blue-500"
          />
          <input
            type="number"
            placeholder="Max"
            value={filters.maxMarketCap ? filters.maxMarketCap / 1e9 : ''}
            onChange={(e) =>
              onFilterChange({
                maxMarketCap: e.target.value ? Number(e.target.value) * 1e9 : undefined,
              })
            }
            className="px-3 py-2 border border-ic-border rounded-md text-sm focus:ring-blue-500 focus:border-blue-500"
          />
        </div>
      </div>
    </div>
  );
}

/**
 * Results table
 */
interface ResultsTableProps {
  stocks: ICScoreStockEntry[];
  sortBy: string;
  sortOrder: string;
  onSort: (column: 'score' | 'marketCap' | 'change' | 'volume') => void;
}

function ResultsTable({ stocks, sortBy, sortOrder, onSort }: ResultsTableProps) {
  const SortIcon = ({ column }: { column: string }) => {
    if (sortBy !== column) return <span className="text-ic-text-dim">↕</span>;
    return sortOrder === 'desc' ? <span>↓</span> : <span>↑</span>;
  };

  return (
    <div className="bg-ic-surface rounded-lg shadow border border-ic-border overflow-hidden">
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-ic-border">
          <thead className="bg-ic-bg-secondary">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-ic-text-muted uppercase tracking-wider">
                Ticker
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-ic-text-muted uppercase tracking-wider">
                Company
              </th>
              <th
                className="px-6 py-3 text-left text-xs font-medium text-ic-text-muted uppercase tracking-wider cursor-pointer hover:text-ic-text-secondary"
                onClick={() => onSort('score')}
              >
                <span className="flex items-center gap-1">
                  IC Score <SortIcon column="score" />
                </span>
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-ic-text-muted uppercase tracking-wider">
                Rating
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-ic-text-muted uppercase tracking-wider">
                Price
              </th>
              <th
                className="px-6 py-3 text-left text-xs font-medium text-ic-text-muted uppercase tracking-wider cursor-pointer hover:text-ic-text-secondary"
                onClick={() => onSort('change')}
              >
                <span className="flex items-center gap-1">
                  Change <SortIcon column="change" />
                </span>
              </th>
              <th
                className="px-6 py-3 text-left text-xs font-medium text-ic-text-muted uppercase tracking-wider cursor-pointer hover:text-ic-text-secondary"
                onClick={() => onSort('marketCap')}
              >
                <span className="flex items-center gap-1">
                  Market Cap <SortIcon column="marketCap" />
                </span>
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-ic-text-muted uppercase tracking-wider">
                Sector
              </th>
            </tr>
          </thead>
          <tbody className="bg-ic-surface divide-y divide-ic-border">
            {stocks.map((stock) => (
              <tr key={stock.ticker} className="hover:bg-ic-surface-hover cursor-pointer">
                <td className="px-6 py-4 whitespace-nowrap">
                  <a
                    href={`/ticker/${stock.ticker}`}
                    className="text-ic-blue font-semibold hover:underline"
                  >
                    {stock.ticker}
                  </a>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-ic-text-primary">
                  {stock.companyName}
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <div className="flex items-center gap-2">
                    <div
                      className="w-12 h-2 rounded-full"
                      style={{ backgroundColor: getICScoreColor(stock.score) }}
                    />
                    <span className="font-bold text-ic-text-primary">{stock.score.toFixed(0)}</span>
                  </div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <span
                    className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${getICScoreRatingColor(
                      stock.rating
                    )}`}
                  >
                    {stock.rating}
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-ic-text-primary">
                  ${stock.price.toFixed(2)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm">
                  <span
                    className={
                      stock.changePercent >= 0 ? 'text-ic-positive' : 'text-ic-negative'
                    }
                  >
                    {stock.changePercent >= 0 ? '+' : ''}
                    {stock.changePercent.toFixed(2)}%
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-ic-text-primary">
                  ${(stock.marketCap / 1e9).toFixed(2)}B
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-ic-text-muted">
                  {stock.sector}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

/**
 * Pagination
 */
interface PaginationProps {
  offset: number;
  limit: number;
  total: number;
  onPageChange: (direction: 'prev' | 'next') => void;
}

function Pagination({ offset, limit, total, onPageChange }: PaginationProps) {
  const currentPage = Math.floor(offset / limit) + 1;
  const totalPages = Math.ceil(total / limit);

  return (
    <div className="mt-6 flex items-center justify-between">
      <button
        onClick={() => onPageChange('prev')}
        disabled={offset === 0}
        className="px-4 py-2 border border-ic-border rounded-md text-sm font-medium text-ic-text-secondary bg-ic-surface hover:bg-ic-surface-hover disabled:opacity-50 disabled:cursor-not-allowed"
      >
        Previous
      </button>
      <span className="text-sm text-ic-text-secondary">
        Page {currentPage} of {totalPages}
      </span>
      <button
        onClick={() => onPageChange('next')}
        disabled={offset + limit >= total}
        className="px-4 py-2 border border-ic-border rounded-md text-sm font-medium text-ic-text-secondary bg-ic-surface hover:bg-ic-surface-hover disabled:opacity-50 disabled:cursor-not-allowed"
      >
        Next
      </button>
    </div>
  );
}

/**
 * Loading state
 */
function LoadingTable() {
  return (
    <div className="bg-ic-surface rounded-lg shadow border border-ic-border p-8 animate-pulse">
      <div className="space-y-3">
        {[...Array(10)].map((_, i) => (
          <div key={i} className="h-12 bg-ic-bg-secondary rounded"></div>
        ))}
      </div>
    </div>
  );
}
