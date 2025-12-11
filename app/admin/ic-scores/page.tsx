'use client';

import { useState, useEffect } from 'react';
import { getICScores, getICScore, ICScoreListItem, ICScoreData, getScoreColor, getFactorDetails } from '@/lib/api/ic-score';
import { Search, ChevronLeft, ChevronRight, ArrowUpDown, X } from 'lucide-react';
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

  // Modal state
  const [selectedTicker, setSelectedTicker] = useState<string | null>(null);
  const [detailedScore, setDetailedScore] = useState<ICScoreData | null>(null);
  const [loadingDetails, setLoadingDetails] = useState(false);

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
      // Handle null or undefined data gracefully
      setScores(result.data || []);
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

  async function handleViewDetails(ticker: string) {
    setSelectedTicker(ticker);
    setLoadingDetails(true);
    try {
      const data = await getICScore(ticker);
      setDetailedScore(data);
    } catch (error) {
      console.error('Error fetching IC Score details:', error);
    } finally {
      setLoadingDetails(false);
    }
  }

  function closeModal() {
    setSelectedTicker(null);
    setDetailedScore(null);
  }

  return (
    <div className="min-h-screen bg-ic-bg-primary">
      {/* Header */}
      <div className="bg-ic-surface shadow-sm border-b border-ic-border">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div>
            <h1 className="text-3xl font-bold text-ic-text-primary">IC Scores Admin</h1>
            <p className="text-ic-text-muted mt-1">
              Development view of all calculated IC Scores
            </p>
          </div>

          {/* Stats */}
          <div className="mt-6 grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
              <div className="text-sm text-ic-blue font-medium">Scores Calculated</div>
              <div className="text-2xl font-bold text-blue-900 mt-1">
                {meta.total.toLocaleString()}
              </div>
            </div>
            <div className="bg-ic-bg-secondary border border-ic-border rounded-lg p-4">
              <div className="text-sm text-ic-text-muted font-medium">Total Stocks</div>
              <div className="text-2xl font-bold text-ic-text-primary mt-1">
                {meta.total_stocks.toLocaleString()}
              </div>
            </div>
            <div className="bg-green-50 border border-green-200 rounded-lg p-4">
              <div className="text-sm text-ic-positive font-medium">Coverage</div>
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
        <div className="bg-ic-surface rounded-lg shadow border border-ic-border p-6 mb-6">
          <form onSubmit={handleSearchSubmit} className="flex gap-4">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-ic-text-dim w-5 h-5" />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search by ticker symbol..."
                className="w-full pl-10 pr-4 py-2 border border-ic-border rounded-lg focus:ring-2 focus:ring-ic-blue focus:border-blue-500 text-ic-text-primary"
              />
            </div>
            <button
              type="submit"
              className="px-6 py-2 bg-ic-blue text-ic-text-primary rounded-lg hover:bg-ic-blue-hover transition-colors font-medium"
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
                className="px-4 py-2 text-ic-text-muted hover:text-ic-text-primary transition-colors"
              >
                Clear
              </button>
            )}
          </form>
        </div>

        {/* Table */}
        <div className="bg-ic-surface rounded-lg shadow border border-ic-border overflow-hidden">
          {loading ? (
            <div className="p-12 text-center">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-ic-blue mx-auto"></div>
              <p className="text-ic-text-muted mt-4">Loading IC Scores...</p>
            </div>
          ) : scores.length === 0 ? (
            <div className="p-12 text-center">
              <p className="text-ic-text-muted">No IC Scores found</p>
            </div>
          ) : (
            <>
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead className="bg-ic-bg-secondary border-b border-ic-border">
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
                      <th className="px-6 py-3 text-left text-xs font-semibold text-ic-text-secondary uppercase tracking-wider">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-ic-border">
                    {scores.map((score) => (
                      <tr key={score.ticker} className="hover:bg-ic-surface-hover">
                        <td className="px-6 py-4">
                          <Link
                            href={`/ticker/${score.ticker}`}
                            className="font-semibold text-ic-blue hover:text-blue-800 hover:underline"
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
                              <div className="h-2 bg-ic-bg-secondary rounded-full overflow-hidden">
                                <div
                                  className={`h-full ${
                                    score.overall_score >= 70
                                      ? 'bg-ic-positive'
                                      : score.overall_score >= 50
                                      ? 'bg-ic-warning'
                                      : 'bg-ic-negative'
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
                                ? 'bg-ic-positive-bg text-green-800'
                                : score.rating === 'Hold'
                                ? 'bg-ic-warning-bg text-yellow-800'
                                : 'bg-ic-negative-bg text-red-800'
                            }`}
                          >
                            {score.rating}
                          </span>
                        </td>
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-medium text-ic-text-primary">
                              {Math.round(score.data_completeness)}%
                            </span>
                            <div className="flex-1 max-w-20 h-1.5 bg-ic-bg-secondary rounded-full overflow-hidden">
                              <div
                                className="h-full bg-blue-500"
                                style={{ width: `${score.data_completeness}%` }}
                              />
                            </div>
                          </div>
                        </td>
                        <td className="px-6 py-4 text-sm text-ic-text-muted">
                          {new Date(score.calculated_at).toLocaleDateString()}
                        </td>
                        <td className="px-6 py-4">
                          <button
                            onClick={() => handleViewDetails(score.ticker)}
                            className="text-sm text-ic-blue hover:text-blue-800 hover:underline"
                          >
                            View Details
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {/* Pagination */}
              <div className="px-6 py-4 bg-ic-bg-secondary border-t border-ic-border">
                <div className="flex items-center justify-between">
                  <div className="text-sm text-ic-text-muted">
                    Showing {meta.offset + 1} to {Math.min(meta.offset + meta.limit, meta.total)} of{' '}
                    {meta.total} scores
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => handlePageChange(currentPage - 1)}
                      disabled={currentPage === 1}
                      className="p-2 rounded hover:bg-ic-bg-secondary disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      <ChevronLeft className="w-5 h-5" />
                    </button>
                    <span className="text-sm text-ic-text-secondary">
                      Page {currentPage} of {totalPages}
                    </span>
                    <button
                      onClick={() => handlePageChange(currentPage + 1)}
                      disabled={currentPage === totalPages}
                      className="p-2 rounded hover:bg-ic-bg-secondary disabled:opacity-50 disabled:cursor-not-allowed"
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

      {/* Details Modal */}
      {selectedTicker && (
        <div className="fixed inset-0 bg-ic-bg-primary bg-opacity-50 z-50 flex items-center justify-center p-4">
          <div className="bg-ic-surface rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] overflow-y-auto">
            {/* Modal Header */}
            <div className="sticky top-0 bg-ic-surface border-b border-ic-border px-6 py-4 flex items-center justify-between">
              <h2 className="text-2xl font-bold text-ic-text-primary">
                {selectedTicker} - IC Score Breakdown
              </h2>
              <button
                onClick={closeModal}
                className="text-ic-text-dim hover:text-ic-text-muted transition-colors"
              >
                <X className="w-6 h-6" />
              </button>
            </div>

            {/* Modal Content */}
            <div className="p-6">
              {loadingDetails ? (
                <div className="flex items-center justify-center py-12">
                  <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-ic-blue"></div>
                </div>
              ) : detailedScore ? (
                <div className="space-y-6">
                  {/* Overall Score Summary */}
                  <div className="bg-gradient-to-br from-blue-50 to-blue-100 rounded-lg p-6 border border-blue-200">
                    <div className="flex items-center justify-between">
                      <div>
                        <div className="text-sm text-ic-blue font-medium">Overall IC Score</div>
                        <div className={`text-5xl font-bold ${getScoreColor(detailedScore.overall_score)} mt-2`}>
                          {Math.round(detailedScore.overall_score)}
                        </div>
                        <div className="mt-2 inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-ic-surface border border-blue-200">
                          {detailedScore.rating}
                        </div>
                      </div>
                      <div className="text-right">
                        <div className="text-sm text-ic-text-muted">Data Completeness</div>
                        <div className="text-3xl font-bold text-ic-text-primary mt-1">
                          {Math.round(detailedScore.data_completeness)}%
                        </div>
                        <div className="text-sm text-ic-text-muted mt-1">
                          {detailedScore.factor_count} of 10 factors
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Factor Breakdown */}
                  <div>
                    <h3 className="text-lg font-semibold text-ic-text-primary mb-4">Factor Breakdown</h3>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      {getFactorDetails(detailedScore).map((factor) => (
                        <div
                          key={factor.name}
                          className={`border rounded-lg p-4 ${
                            factor.available
                              ? 'bg-ic-surface border-ic-border'
                              : 'bg-ic-bg-secondary border-ic-border'
                          }`}
                        >
                          <div className="flex items-center justify-between mb-2">
                            <div className="font-medium text-ic-text-primary">{factor.display_name}</div>
                            {factor.available ? (
                              <span className={`text-2xl font-bold ${getScoreColor(factor.score)}`}>
                                {Math.round(factor.score!)}
                              </span>
                            ) : (
                              <span className="text-sm text-ic-text-dim font-medium">N/A</span>
                            )}
                          </div>
                          {factor.available ? (
                            <>
                              <div className="mb-2">
                                <div className="h-2 bg-ic-bg-secondary rounded-full overflow-hidden">
                                  <div
                                    className={`h-full ${
                                      factor.score! >= 70
                                        ? 'bg-ic-positive'
                                        : factor.score! >= 50
                                        ? 'bg-ic-warning'
                                        : 'bg-ic-negative'
                                    }`}
                                    style={{ width: `${factor.score}%` }}
                                  />
                                </div>
                              </div>
                              <div className="text-xs text-ic-text-muted">{factor.description}</div>
                            </>
                          ) : (
                            <div className="text-xs text-ic-text-dim">Data not available</div>
                          )}
                        </div>
                      ))}
                    </div>
                  </div>

                  {/* Additional Info */}
                  <div className="bg-ic-bg-secondary rounded-lg p-4 border border-ic-border">
                    <div className="grid grid-cols-2 gap-4 text-sm">
                      <div>
                        <span className="text-ic-text-muted">Calculated:</span>{' '}
                        <span className="font-medium text-ic-text-primary">
                          {new Date(detailedScore.calculated_at).toLocaleString()}
                        </span>
                      </div>
                      <div>
                        <span className="text-ic-text-muted">Confidence:</span>{' '}
                        <span className="font-medium text-ic-text-primary">{detailedScore.confidence_level}</span>
                      </div>
                    </div>
                  </div>
                </div>
              ) : (
                <div className="text-center py-12 text-ic-text-muted">
                  Failed to load IC Score details
                </div>
              )}
            </div>

            {/* Modal Footer */}
            <div className="sticky bottom-0 bg-ic-bg-secondary border-t border-ic-border px-6 py-4 flex items-center justify-between">
              <Link
                href={`/ticker/${selectedTicker}`}
                className="text-ic-blue hover:text-blue-800 text-sm font-medium"
              >
                View Full Ticker Page →
              </Link>
              <button
                onClick={closeModal}
                className="px-4 py-2 bg-ic-bg-secondary text-ic-text-secondary rounded-lg hover:bg-ic-border transition-colors font-medium"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}
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
      className="flex items-center gap-2 text-xs font-semibold text-ic-text-secondary uppercase tracking-wider hover:text-ic-text-primary transition-colors"
    >
      {children}
      <ArrowUpDown
        className={`w-4 h-4 ${isActive ? 'text-ic-blue' : 'text-ic-text-dim'}`}
      />
    </button>
  );
}
