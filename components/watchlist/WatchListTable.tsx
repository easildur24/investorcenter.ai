'use client';

import React, { useState, useMemo } from 'react';
import Link from 'next/link';
import { WatchListItem } from '@/lib/api/watchlist';
import { cn, safeToFixed, formatLargeNumber } from '@/lib/utils';
import {
  getColumnsForView,
  ColumnDefinition,
  ViewPresetId,
  VIEW_PRESETS,
  DEFAULT_VIEW,
  VIEW_STORAGE_KEY,
} from '@/lib/watchlist/columns';
import ViewSwitcher from './ViewSwitcher';

// ---------------------------------------------------------------------------
// Display label for asset type filter chips (uppercase abbreviations).
// ---------------------------------------------------------------------------

const ASSET_TYPE_LABELS: Record<string, string> = {
  etf: 'ETF',
  etn: 'ETN',
  reit: 'REIT',
  spac: 'SPAC',
  adr: 'ADR',
};

function formatAssetTypeLabel(type: string): string {
  return ASSET_TYPE_LABELS[type.toLowerCase()] ?? type.charAt(0).toUpperCase() + type.slice(1);
}

// ---------------------------------------------------------------------------
// Props — unchanged from original, no parent changes needed.
// ---------------------------------------------------------------------------

interface WatchListTableProps {
  items: WatchListItem[];
  onRemove: (symbol: string) => void;
  onEdit: (symbol: string) => void;
  /** When set, the row for this symbol will show an expanded panel below it. */
  expandedSymbol?: string | null;
  /** Render function for the expanded row content (e.g. InlineEditPanel). */
  renderExpandedRow?: (item: WatchListItem) => React.ReactNode;
}

// ---------------------------------------------------------------------------
// Sort state — 3-state cycle: asc → desc → null (unsorted)
// ---------------------------------------------------------------------------

type SortDirection = 'asc' | 'desc' | null;

function nextSortDirection(current: SortDirection): SortDirection {
  if (current === 'asc') return 'desc';
  if (current === 'desc') return null;
  return 'asc';
}

// ---------------------------------------------------------------------------
// Target alert helper (preserved from original)
// ---------------------------------------------------------------------------

function checkTargetAlert(item: WatchListItem) {
  if (!item.current_price) return null;

  if (item.target_buy_price && item.current_price <= item.target_buy_price) {
    return {
      type: 'buy' as const,
      message: 'At buy target',
      bgClass: 'bg-green-500/10 border-l-4 border-green-500',
    };
  }

  if (item.target_sell_price && item.current_price >= item.target_sell_price) {
    return {
      type: 'sell' as const,
      message: 'At sell target',
      bgClass: 'bg-blue-500/10 border-l-4 border-blue-500',
    };
  }

  return null;
}

// ---------------------------------------------------------------------------
// Cell renderer
// ---------------------------------------------------------------------------

function renderCell(
  col: ColumnDefinition,
  item: WatchListItem,
  alert: ReturnType<typeof checkTargetAlert>,
  onRemove: (symbol: string) => void,
  onEdit: (symbol: string) => void
): React.ReactNode {
  const raw = col.getValue(item);

  switch (col.type) {
    // ── Symbol link ───────────────────────────────────────────────────
    case 'symbol':
      return (
        <Link href={`/ticker/${item.symbol}`} className="text-ic-blue hover:underline font-medium">
          {item.symbol}
        </Link>
      );

    // ── Plain text ────────────────────────────────────────────────────
    case 'text':
      return <span className="text-ic-text-primary truncate">{raw ?? '—'}</span>;

    // ── Currency ($XX.XX) ─────────────────────────────────────────────
    case 'currency': {
      if (raw == null) return <span className="text-ic-text-secondary">—</span>;
      const num = Number(raw);
      const dec = col.decimals ?? 2;

      // Special styling for target prices when alert triggered
      if (col.id === 'target_buy_price' && alert?.type === 'buy') {
        return <span className="font-bold text-ic-positive">${safeToFixed(num, dec)}</span>;
      }
      if (col.id === 'target_sell_price' && alert?.type === 'sell') {
        return <span className="font-bold text-ic-blue">${safeToFixed(num, dec)}</span>;
      }

      return <span className="font-medium text-ic-text-primary">${safeToFixed(num, dec)}</span>;
    }

    // ── Percent (XX.X%) ───────────────────────────────────────────────
    case 'percent': {
      if (raw == null) return <span className="text-ic-text-secondary">—</span>;
      const num = Number(raw);
      const dec = col.decimals ?? 1;
      const color =
        num > 0 ? 'text-ic-positive' : num < 0 ? 'text-ic-negative' : 'text-ic-text-primary';
      return <span className={color}>{safeToFixed(num, dec)}%</span>;
    }

    // ── Percentile (neutral color, always positive rank metric) ────────
    case 'percentile': {
      if (raw == null) return <span className="text-ic-text-secondary">—</span>;
      const num = Number(raw);
      const dec = col.decimals ?? 0;
      return <span>{safeToFixed(num, dec)}%</span>;
    }

    // ── Number (configurable decimals) ────────────────────────────────
    case 'number': {
      if (raw == null) return <span className="text-ic-text-secondary">—</span>;
      const num = Number(raw);
      // Large numbers like volume and market cap
      if (col.id === 'market_cap') return <span>{formatLargeNumber(num)}</span>;
      if (col.id === 'volume') {
        if (num >= 1e6) return <span>{(num / 1e6).toFixed(1)}M</span>;
        if (num >= 1e3) return <span>{(num / 1e3).toFixed(1)}K</span>;
        return <span>{num.toLocaleString()}</span>;
      }
      const dec = col.decimals ?? 2;
      return <span>{safeToFixed(num, dec)}</span>;
    }

    // ── Integer ───────────────────────────────────────────────────────
    case 'integer': {
      if (raw == null) return <span className="text-ic-text-secondary">—</span>;
      return <span>{Number(raw).toLocaleString()}</span>;
    }

    // ── Price change (+X.XX / +Y.YY%) ────────────────────────────────
    case 'change': {
      const change = item.price_change;
      const changePct = item.price_change_pct;
      if (change == null || changePct == null)
        return <span className="text-ic-text-secondary">—</span>;
      const color = change >= 0 ? 'text-ic-positive' : 'text-ic-negative';
      const sign = change >= 0 ? '+' : '';
      return (
        <span className={color}>
          {sign}
          {change.toFixed(2)} ({sign}
          {changePct.toFixed(2)}%)
        </span>
      );
    }

    // ── IC Score colored pill ─────────────────────────────────────────
    case 'score': {
      if (raw == null) return <span className="text-ic-text-secondary">—</span>;
      const score = Number(raw);
      const pillColor =
        score >= 70
          ? 'bg-green-500/20 text-ic-positive'
          : score >= 40
            ? 'bg-yellow-500/20 text-yellow-400'
            : 'bg-red-500/20 text-ic-negative';
      return (
        <span className={`inline-block px-2 py-0.5 text-xs font-semibold rounded ${pillColor}`}>
          {safeToFixed(score, 1)}
        </span>
      );
    }

    // ── IC Rating text ────────────────────────────────────────────────
    case 'rating':
      return <span className="text-sm text-ic-text-primary">{raw ?? '—'}</span>;

    // ── Reddit trend arrow ────────────────────────────────────────────
    case 'trend': {
      if (!raw) return <span className="text-ic-text-secondary">—</span>;
      const trend = String(raw).toLowerCase();
      let arrow = '→';
      let color = 'text-ic-text-secondary';
      if (trend === 'rising') {
        arrow = '↑';
        color = 'text-ic-positive';
      } else if (trend === 'falling') {
        arrow = '↓';
        color = 'text-ic-negative';
      }
      return (
        <span className={color} aria-label={`Trend: ${String(raw)}`}>
          <span aria-hidden="true">{arrow}</span> {String(raw)}
        </span>
      );
    }

    // ── Target alert badge ────────────────────────────────────────────
    case 'badge': {
      if (!alert) return null;
      return (
        <span
          className={`inline-block px-2 py-1 text-xs font-semibold rounded ${
            alert.type === 'buy'
              ? 'bg-green-500/20 text-ic-positive'
              : 'bg-blue-500/20 text-ic-blue'
          }`}
        >
          {alert.message}
        </span>
      );
    }

    // ── Actions (Edit / Remove) ───────────────────────────────────────
    case 'actions':
      return (
        <>
          <button
            onClick={() => onEdit(item.symbol)}
            className="text-ic-blue hover:underline text-sm mr-3"
            aria-label={`Edit ${item.symbol}`}
          >
            Edit
          </button>
          <button
            onClick={() => onRemove(item.symbol)}
            className="text-ic-negative hover:underline text-sm"
            aria-label={`Remove ${item.symbol}`}
          >
            Remove
          </button>
        </>
      );

    default:
      return <span>{raw != null ? String(raw) : '—'}</span>;
  }
}

// ---------------------------------------------------------------------------
// Sort indicator
// ---------------------------------------------------------------------------

function SortIndicator({ direction }: { direction: SortDirection }) {
  if (!direction) return null;
  return (
    <span className="ml-1 text-xs" aria-hidden="true">
      {direction === 'asc' ? '↑' : '↓'}
    </span>
  );
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export default function WatchListTable({
  items,
  onRemove,
  onEdit,
  expandedSymbol,
  renderExpandedRow,
}: WatchListTableProps) {
  // ── View preset (persisted to localStorage) ───────────────────────
  const [activeView, setActiveView] = useState<ViewPresetId>(() => {
    if (typeof window === 'undefined') return DEFAULT_VIEW;
    try {
      const stored = localStorage.getItem(VIEW_STORAGE_KEY);
      const validIds = VIEW_PRESETS.map((p) => p.id) as string[];
      return stored && validIds.includes(stored) ? (stored as ViewPresetId) : DEFAULT_VIEW;
    } catch {
      return DEFAULT_VIEW;
    }
  });

  const handleViewChange = (view: ViewPresetId) => {
    setActiveView(view);
    try {
      localStorage.setItem(VIEW_STORAGE_KEY, view);
    } catch {
      // localStorage may be unavailable (private browsing, quota exceeded)
    }
  };

  // ── Sort state — 3-state cycle ────────────────────────────────────
  const [sortColumn, setSortColumn] = useState<string | null>(null);
  const [sortDirection, setSortDirection] = useState<SortDirection>(null);

  const handleSort = (colId: string) => {
    if (sortColumn === colId) {
      const next = nextSortDirection(sortDirection);
      setSortDirection(next);
      if (next === null) setSortColumn(null);
    } else {
      setSortColumn(colId);
      setSortDirection('asc');
    }
  };

  // ── Search & filter state ─────────────────────────────────────────
  const [searchQuery, setSearchQuery] = useState('');
  const [assetTypeFilter, setAssetTypeFilter] = useState<string | null>(null);

  // ── Derived columns for active view ───────────────────────────────
  const columns = useMemo(() => getColumnsForView(activeView), [activeView]);

  // ── Data pipeline: search → asset type filter → sort ──────────────
  const processedItems = useMemo(() => {
    let result = [...items];

    // Search filter (symbol + name)
    if (searchQuery) {
      const q = searchQuery.toLowerCase();
      result = result.filter(
        (item) => item.symbol.toLowerCase().includes(q) || item.name.toLowerCase().includes(q)
      );
    }

    // Asset type filter
    if (assetTypeFilter) {
      result = result.filter((item) => item.asset_type === assetTypeFilter);
    }

    // Sort
    if (sortColumn && sortDirection) {
      const col = columns.find((c) => c.id === sortColumn);
      if (col) {
        result.sort((a, b) => {
          const aVal = col.getValue(a);
          const bVal = col.getValue(b);

          // Nulls always sort last regardless of direction
          if (aVal == null && bVal == null) return 0;
          if (aVal == null) return 1;
          if (bVal == null) return -1;

          let cmp: number;
          if (typeof aVal === 'string' && typeof bVal === 'string') {
            cmp = aVal.localeCompare(bVal);
          } else {
            cmp = Number(aVal) - Number(bVal);
          }

          return sortDirection === 'desc' ? -cmp : cmp;
        });
      }
    }

    return result;
  }, [items, searchQuery, assetTypeFilter, sortColumn, sortDirection, columns]);

  // ── Unique asset types for filter chips ────────────────────────────
  const assetTypes = useMemo(() => {
    const types = new Set(items.map((i) => i.asset_type));
    return Array.from(types).sort();
  }, [items]);

  // ── Render ─────────────────────────────────────────────────────────
  return (
    <div className="space-y-4">
      {/* View preset tabs */}
      <ViewSwitcher activeView={activeView} onViewChange={handleViewChange} />

      {/* Search + filter bar */}
      <div className="flex flex-wrap items-center gap-3">
        <input
          type="text"
          placeholder="Search by symbol or name..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="px-3 py-1.5 text-sm rounded-lg border border-ic-border bg-ic-surface text-ic-text-primary placeholder:text-ic-text-secondary focus:outline-none focus:ring-1 focus:ring-ic-blue w-56"
          aria-label="Search watchlist"
        />

        {/* Asset type chips */}
        <div className="flex gap-1.5" role="group" aria-label="Filter by asset type">
          <button
            onClick={() => setAssetTypeFilter(null)}
            aria-pressed={assetTypeFilter === null}
            className={cn(
              'px-2.5 py-1 text-xs rounded-lg transition-colors',
              assetTypeFilter === null
                ? 'bg-ic-blue text-white'
                : 'bg-ic-surface border border-ic-border text-ic-text-secondary hover:bg-ic-surface-hover'
            )}
          >
            All
          </button>
          {assetTypes.map((type) => (
            <button
              key={type}
              onClick={() => setAssetTypeFilter(assetTypeFilter === type ? null : type)}
              aria-pressed={assetTypeFilter === type}
              className={cn(
                'px-2.5 py-1 text-xs rounded-lg transition-colors',
                assetTypeFilter === type
                  ? 'bg-ic-blue text-white'
                  : 'bg-ic-surface border border-ic-border text-ic-text-secondary hover:bg-ic-surface-hover'
              )}
            >
              {formatAssetTypeLabel(type)}
            </button>
          ))}
        </div>

        {/* Result count */}
        <span className="text-xs text-ic-text-secondary ml-auto" data-testid="result-count">
          {processedItems.length} of {items.length} {items.length === 1 ? 'ticker' : 'tickers'}
        </span>
      </div>

      {/* Table */}
      <div
        className="overflow-x-auto"
        role="tabpanel"
        id="watchlist-tabpanel"
        aria-labelledby={`watchlist-tab-${activeView}`}
      >
        <table className="w-full bg-ic-surface rounded-lg border border-ic-border">
          <thead className="bg-ic-bg-secondary">
            <tr>
              {columns.map((col) => {
                const isSorted = sortColumn === col.id;
                return (
                  <th
                    key={col.id}
                    className={cn(
                      'px-4 py-3 text-sm font-semibold text-ic-text-primary whitespace-nowrap',
                      col.align === 'left'
                        ? 'text-left'
                        : col.align === 'right'
                          ? 'text-right'
                          : 'text-center',
                      col.sortable && 'cursor-pointer select-none hover:text-ic-blue',
                      col.width
                    )}
                    onClick={col.sortable ? () => handleSort(col.id) : undefined}
                    aria-sort={
                      isSorted && sortDirection
                        ? sortDirection === 'asc'
                          ? 'ascending'
                          : 'descending'
                        : undefined
                    }
                  >
                    {col.label}
                    {col.sortable && isSorted && <SortIndicator direction={sortDirection} />}
                  </th>
                );
              })}
            </tr>
          </thead>
          <tbody className="divide-y">
            {processedItems.length === 0 ? (
              <tr>
                <td colSpan={columns.length} className="px-4 py-12 text-center text-ic-text-muted">
                  {searchQuery
                    ? `No results found for "${searchQuery}"`
                    : assetTypeFilter
                      ? `No ${formatAssetTypeLabel(assetTypeFilter)} tickers found`
                      : 'No tickers in this watchlist'}
                </td>
              </tr>
            ) : (
              processedItems.map((item) => {
                const alert = checkTargetAlert(item);
                const isExpanded = expandedSymbol === item.symbol;
                return (
                  <React.Fragment key={item.symbol}>
                    <tr className={cn('hover:bg-ic-surface-hover', alert ? alert.bgClass : '')}>
                      {columns.map((col) => (
                        <td
                          key={col.id}
                          className={cn(
                            'px-4 py-3 text-sm',
                            col.align === 'left'
                              ? 'text-left'
                              : col.align === 'right'
                                ? 'text-right'
                                : 'text-center',
                            col.width
                          )}
                        >
                          {renderCell(col, item, alert, onRemove, onEdit)}
                        </td>
                      ))}
                    </tr>
                    {isExpanded && renderExpandedRow && (
                      <tr>
                        <td colSpan={columns.length} className="p-0">
                          {renderExpandedRow(item)}
                        </td>
                      </tr>
                    )}
                  </React.Fragment>
                );
              })
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
