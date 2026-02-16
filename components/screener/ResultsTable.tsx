'use client';

import Link from 'next/link';
import { cn } from '@/lib/utils';
import { ALL_COLUMNS } from '@/lib/screener/column-config';
import type { ScreenerStock, ScreenerSortField } from '@/lib/types/screener';

interface ResultsTableProps {
  stocks: ScreenerStock[];
  visibleColumns: ScreenerSortField[];
  sortField: string;
  sortOrder: string;
  isLoading: boolean;
  onSort: (field: ScreenerSortField) => void;
}

/** Sortable results table with dynamic column visibility. */
export function ResultsTable({
  stocks,
  visibleColumns,
  sortField,
  sortOrder,
  isLoading,
  onSort,
}: ResultsTableProps) {
  const columns = ALL_COLUMNS.filter(c => visibleColumns.includes(c.key));

  if (isLoading && stocks.length === 0) {
    return (
      <div className="p-8 text-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-ic-blue mx-auto"></div>
        <p className="mt-4 text-ic-text-muted">Loading stocks...</p>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full">
        <thead className="bg-ic-bg-secondary">
          <tr>
            {columns.map(col => (
              <th
                key={col.key}
                className={cn(
                  'px-4 py-3 text-xs font-medium text-ic-text-muted uppercase tracking-wider cursor-pointer hover:bg-ic-bg-tertiary transition-colors whitespace-nowrap',
                  col.align === 'right' ? 'text-right' : 'text-left'
                )}
                onClick={() => onSort(col.key)}
              >
                {col.label}
                {sortField === col.key && (
                  <span className="ml-1">
                    {sortOrder === 'asc' ? '\u2191' : '\u2193'}
                  </span>
                )}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-ic-border-subtle">
          {stocks.map(stock => (
            <tr key={stock.symbol} className="hover:bg-ic-surface-hover transition-colors">
              {columns.map(col => {
                // Symbol — linked to ticker page
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

                // Name — with sector subtitle
                if (col.key === 'name') {
                  return (
                    <td key={col.key} className="px-4 py-3 text-ic-text-primary">
                      <div className="max-w-xs truncate">{stock.name}</div>
                      <div className="text-xs text-ic-text-muted">{stock.sector}</div>
                    </td>
                  );
                }

                // IC Score — colored badge
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
                        <span className="text-ic-text-dim">&mdash;</span>
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
                      'px-4 py-3 whitespace-nowrap',
                      col.align === 'right' ? 'text-right' : 'text-left',
                      colorClass || 'text-ic-text-primary'
                    )}
                  >
                    {formatted}
                  </td>
                );
              })}
            </tr>
          ))}
          {stocks.length === 0 && !isLoading && (
            <tr>
              <td
                colSpan={columns.length}
                className="px-4 py-12 text-center text-ic-text-muted"
              >
                No stocks match your filters. Try adjusting or clearing filters.
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
