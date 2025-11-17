'use client';

import { useState, useEffect } from 'react';
import { getICScores, ICScoreListItem, getScoreColor } from '@/lib/api/ic-score';
import { Search, ChevronLeft, ChevronRight, ArrowUpDown } from 'lucide-react';
import Link from 'next/link';

export default function ICScoresAdminPage() {
  const [scores, setScores] = useState<ICScoreListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [meta, setMeta] = useState({
    total: 0,
    limit: 20,
    offset: 0,
    total_stocks: 0,
    coverage_percent: 0,
  });
  const [search, setSearch] = useState('');
  const [sort, setSort] = useState('overall_score');
  const [order, setOrder] = useState<'asc' | 'desc'>('desc');

  const currentPage = Math.floor(meta.offset / meta.limit) + 1;
  const totalPages = Math.ceil(meta.total / meta.limit);

  useEffect(() => {
    fetchScores();
  }, [search, sort, order, meta.offset, meta.limit]);

  async function fetchScores() {
    try {
      setLoading(true);
      const result = await getICScores({
        limit: meta.limit,
        offset: meta.offset,
        search: search || undefined,
        sort,
        order,
      });
      setScores(result.data);
      setMeta(result.meta);
    } catch (error) {
      console.error('Error fetching IC Scores:', error);
    } finally {
      setLoading(false);
    }
  }

  function handleSort(column: string) {
    if (sort === column) {
      setOrder(order === 'asc' ? 'desc' : 'asc');
    } else {
      setSort(column);
      setOrder('desc');
    }
  }

  function handlePageChange(newPage: number) {
    setMeta({
      ...meta,
      offset: (newPage - 1) * meta.limit,
    });
  }

  function handleSearchSubmit(e: React.FormEvent) {
    e.preventDefault();
    setMeta({ ...meta, offset: 0 }); // Reset to first page
    fetchScores();
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow-sm border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div>
            <h1 className="text-3xl font-bold text-gray-900">IC Scores Admin</h1>
            <p className="text-gray-600 mt-1">
              Development view of all calculated IC Scores
            </p>
          </div>

          {/* Stats */}
          <div className="mt-6 grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
              <div className="text-sm text-blue-600 font-medium">Scores Calculated</div>
              <div className="text-2xl font-bold text-blue-900 mt-1">
                {meta.total.toLocaleString()}
              </div>
            </div>
            <div className="bg-gray-50 border border-gray-200 rounded-lg p-4">
              <div className="text-sm text-gray-600 font-medium">Total Stocks</div>
              <div className="text-2xl font-bold text-gray-900 mt-1">
                {meta.total_stocks.toLocaleString()}
              </div>
            </div>
            <div className="bg-green-50 border border-green-200 rounded-lg p-4">
              <div className="text-sm text-green-600 font-medium">Coverage</div>
              <div className="text-2xl font-bold text-green-900 mt-1">
                {meta.coverage_percent.toFixed(1)}%
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Search and Filters */}
        <div className="bg-white rounded-lg shadow border border-gray-200 p-6 mb-6">
          <form onSubmit={handleSearchSubmit} className="flex gap-4">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search by ticker symbol..."
                className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
            </div>
            <button
              type="submit"
              className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium"
            >
              Search
            </button>
            {search && (
              <button
                type="button"
                onClick={() => {
                  setSearch('');
                  setMeta({ ...meta, offset: 0 });
                }}
                className="px-4 py-2 text-gray-600 hover:text-gray-900 transition-colors"
              >
                Clear
              </button>
            )}
          </form>
        </div>

        {/* Table */}
        <div className="bg-white rounded-lg shadow border border-gray-200 overflow-hidden">
          {loading ? (
            <div className="p-12 text-center">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
              <p className="text-gray-600 mt-4">Loading IC Scores...</p>
            </div>
          ) : scores.length === 0 ? (
            <div className="p-12 text-center">
              <p className="text-gray-600">No IC Scores found</p>
            </div>
          ) : (
            <>
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead className="bg-gray-50 border-b border-gray-200">
                    <tr>
                      <th className="px-6 py-3 text-left">
                        <SortButton
                          column="ticker"
                          currentSort={sort}
                          currentOrder={order}
                          onClick={() => handleSort('ticker')}
                        >
                          Ticker
                        </SortButton>
                      </th>
                      <th className="px-6 py-3 text-left">
                        <SortButton
                          column="overall_score"
                          currentSort={sort}
                          currentOrder={order}
                          onClick={() => handleSort('overall_score')}
                        >
                          Score
                        </SortButton>
                      </th>
                      <th className="px-6 py-3 text-left">
                        <SortButton
                          column="rating"
                          currentSort={sort}
                          currentOrder={order}
                          onClick={() => handleSort('rating')}
                        >
                          Rating
                        </SortButton>
                      </th>
                      <th className="px-6 py-3 text-left">
                        <SortButton
                          column="data_completeness"
                          currentSort={sort}
                          currentOrder={order}
                          onClick={() => handleSort('data_completeness')}
                        >
                          Completeness
                        </SortButton>
                      </th>
                      <th className="px-6 py-3 text-left">
                        <SortButton
                          column="created_at"
                          currentSort={sort}
                          currentOrder={order}
                          onClick={() => handleSort('created_at')}
                        >
                          Calculated
                        </SortButton>
                      </th>
                      <th className="px-6 py-3 text-left">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200">
                    {scores.map((score) => (
                      <tr key={score.ticker} className="hover:bg-gray-50">
                        <td className="px-6 py-4">
                          <Link
                            href={`/ticker/${score.ticker}`}
                            className="font-semibold text-blue-600 hover:text-blue-800 hover:underline"
                          >
                            {score.ticker}
                          </Link>
                        </td>
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-3">
                            <span
                              className={`text-2xl font-bold ${getScoreColor(score.overall_score)}`}
                            >
                              {Math.round(score.overall_score)}
                            </span>
                            <div className="flex-1 max-w-24">
                              <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
                                <div
                                  className={`h-full ${
                                    score.overall_score >= 70
                                      ? 'bg-green-500'
                                      : score.overall_score >= 50
                                      ? 'bg-yellow-500'
                                      : 'bg-red-500'
                                  }`}
                                  style={{ width: `${score.overall_score}%` }}
                                />
                              </div>
                            </div>
                          </div>
                        </td>
                        <td className="px-6 py-4">
                          <span
                            className={`inline-flex px-3 py-1 rounded-full text-sm font-medium ${
                              score.rating.includes('Buy')
                                ? 'bg-green-100 text-green-800'
                                : score.rating === 'Hold'
                                ? 'bg-yellow-100 text-yellow-800'
                                : 'bg-red-100 text-red-800'
                            }`}
                          >
                            {score.rating}
                          </span>
                        </td>
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-medium text-gray-900">
                              {Math.round(score.data_completeness)}%
                            </span>
                            <div className="flex-1 max-w-20 h-1.5 bg-gray-200 rounded-full overflow-hidden">
                              <div
                                className="h-full bg-blue-500"
                                style={{ width: `${score.data_completeness}%` }}
                              />
                            </div>
                          </div>
                        </td>
                        <td className="px-6 py-4 text-sm text-gray-600">
                          {new Date(score.calculated_at).toLocaleDateString()}
                        </td>
                        <td className="px-6 py-4">
                          <Link
                            href={`/ticker/${score.ticker}#ic-score`}
                            className="text-sm text-blue-600 hover:text-blue-800 hover:underline"
                          >
                            View Details
                          </Link>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {/* Pagination */}
              <div className="px-6 py-4 bg-gray-50 border-t border-gray-200">
                <div className="flex items-center justify-between">
                  <div className="text-sm text-gray-600">
                    Showing {meta.offset + 1} to {Math.min(meta.offset + meta.limit, meta.total)} of{' '}
                    {meta.total} scores
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => handlePageChange(currentPage - 1)}
                      disabled={currentPage === 1}
                      className="p-2 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      <ChevronLeft className="w-5 h-5" />
                    </button>
                    <span className="text-sm text-gray-700">
                      Page {currentPage} of {totalPages}
                    </span>
                    <button
                      onClick={() => handlePageChange(currentPage + 1)}
                      disabled={currentPage === totalPages}
                      className="p-2 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      <ChevronRight className="w-5 h-5" />
                    </button>
                  </div>
                </div>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

interface SortButtonProps {
  column: string;
  currentSort: string;
  currentOrder: 'asc' | 'desc';
  onClick: () => void;
  children: React.ReactNode;
}

function SortButton({ column, currentSort, currentOrder, onClick, children }: SortButtonProps) {
  const isActive = currentSort === column;

  return (
    <button
      onClick={onClick}
      className="flex items-center gap-2 text-xs font-semibold text-gray-700 uppercase tracking-wider hover:text-gray-900 transition-colors"
    >
      {children}
      <ArrowUpDown
        className={`w-4 h-4 ${isActive ? 'text-blue-600' : 'text-gray-400'}`}
      />
    </button>
  );
}
