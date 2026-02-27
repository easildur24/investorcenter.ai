'use client';

import { useState, useEffect, useMemo, useCallback } from 'react';
import Link from 'next/link';
import { API_BASE_URL } from '@/lib/api';
import { earningsCalendar } from '@/lib/api/routes';
import type { EarningsResult, EarningsCalendarResponse } from '@/lib/types/earnings';
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  MagnifyingGlassIcon,
} from '@heroicons/react/24/outline';

// ============================================================================
// Date Helpers
// ============================================================================

function getMonday(date: Date): Date {
  const d = new Date(date);
  const day = d.getDay();
  const diff = day === 0 ? -6 : 1 - day; // Monday = 1
  d.setDate(d.getDate() + diff);
  d.setHours(0, 0, 0, 0);
  return d;
}

function formatDateShort(date: Date): string {
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

function formatDateISO(date: Date): string {
  return date.toISOString().split('T')[0];
}

function getWeekDays(monday: Date): Date[] {
  return Array.from({ length: 5 }, (_, i) => {
    const d = new Date(monday);
    d.setDate(monday.getDate() + i);
    return d;
  });
}

const DAY_NAMES = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri'];

// ============================================================================
// Formatting
// ============================================================================

function formatEPS(value: number | null): string {
  if (value == null) return '—';
  return `$${value.toFixed(2)}`;
}

function formatRevenue(value: number | null): string {
  if (value == null) return '—';
  const abs = Math.abs(value);
  if (abs >= 1e12) return `$${(value / 1e12).toFixed(1)}T`;
  if (abs >= 1e9) return `$${(value / 1e9).toFixed(1)}B`;
  if (abs >= 1e6) return `$${(value / 1e6).toFixed(1)}M`;
  if (abs >= 1e3) return `$${(value / 1e3).toFixed(0)}K`;
  return `$${value.toFixed(0)}`;
}

function formatSurprise(value: number | null): string {
  if (value == null) return '—';
  const sign = value >= 0 ? '+' : '';
  return `${sign}${value.toFixed(2)}%`;
}

// ============================================================================
// Sort
// ============================================================================

type SortField =
  | 'symbol'
  | 'date'
  | 'epsEstimated'
  | 'epsActual'
  | 'epsSurprise'
  | 'revenueEstimated'
  | 'revenueActual';
type SortOrder = 'asc' | 'desc';

function sortEarnings(
  earnings: EarningsResult[],
  field: SortField,
  order: SortOrder
): EarningsResult[] {
  return [...earnings].sort((a, b) => {
    let cmp = 0;
    switch (field) {
      case 'symbol':
        cmp = a.symbol.localeCompare(b.symbol);
        break;
      case 'date':
        cmp = a.date.localeCompare(b.date);
        break;
      case 'epsEstimated':
        cmp = (a.epsEstimated ?? -Infinity) - (b.epsEstimated ?? -Infinity);
        break;
      case 'epsActual':
        cmp = (a.epsActual ?? -Infinity) - (b.epsActual ?? -Infinity);
        break;
      case 'epsSurprise':
        cmp = (a.epsSurprisePercent ?? -Infinity) - (b.epsSurprisePercent ?? -Infinity);
        break;
      case 'revenueEstimated':
        cmp = (a.revenueEstimated ?? -Infinity) - (b.revenueEstimated ?? -Infinity);
        break;
      case 'revenueActual':
        cmp = (a.revenueActual ?? -Infinity) - (b.revenueActual ?? -Infinity);
        break;
    }
    return order === 'asc' ? cmp : -cmp;
  });
}

// ============================================================================
// Main Page Component
// ============================================================================

export default function EarningsCalendarPage() {
  const today = new Date();
  const [weekMonday, setWeekMonday] = useState<Date>(() => getMonday(today));
  const [selectedDate, setSelectedDate] = useState<string>(formatDateISO(today));
  const [data, setData] = useState<EarningsCalendarResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [viewMode, setViewMode] = useState<'daily' | 'weekly'>('daily');
  const [sortField, setSortField] = useState<SortField>('symbol');
  const [sortOrder, setSortOrder] = useState<SortOrder>('asc');

  const weekDays = useMemo(() => getWeekDays(weekMonday), [weekMonday]);
  const fromDate = formatDateISO(weekMonday);
  const toDate = formatDateISO(weekDays[4]);

  // Fetch earnings for the entire week
  useEffect(() => {
    const fetchCalendar = async () => {
      try {
        setLoading(true);
        setError(null);
        const response = await fetch(
          `${API_BASE_URL}${earningsCalendar.list}?from=${fromDate}&to=${toDate}`,
          { cache: 'no-store' }
        );
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`);
        }
        const result = await response.json();
        setData(result.data);
      } catch (err) {
        console.error('Error fetching earnings calendar:', err);
        setError('Failed to load earnings calendar');
      } finally {
        setLoading(false);
      }
    };

    fetchCalendar();
  }, [fromDate, toDate]);

  // Filter earnings by selected date (daily mode) and search query
  const filteredEarnings = useMemo(() => {
    if (!data?.earnings) return [];
    let filtered = data.earnings;

    // In daily mode, filter to selected date only
    if (viewMode === 'daily') {
      filtered = filtered.filter((e) => e.date === selectedDate);
    }

    // Search filter
    if (search.trim()) {
      const q = search.trim().toLowerCase();
      filtered = filtered.filter((e) => e.symbol.toLowerCase().includes(q));
    }

    return sortEarnings(filtered, sortField, sortOrder);
  }, [data, selectedDate, search, viewMode, sortField, sortOrder]);

  const handlePrevWeek = useCallback(() => {
    setWeekMonday((prev) => {
      const d = new Date(prev);
      d.setDate(d.getDate() - 7);
      return d;
    });
  }, []);

  const handleNextWeek = useCallback(() => {
    setWeekMonday((prev) => {
      const d = new Date(prev);
      d.setDate(d.getDate() + 7);
      return d;
    });
  }, []);

  const handleSort = useCallback((field: SortField) => {
    setSortField((prev) => {
      if (prev === field) {
        setSortOrder((o) => (o === 'asc' ? 'desc' : 'asc'));
        return prev;
      }
      setSortOrder('asc');
      return field;
    });
  }, []);

  return (
    <div className="min-h-screen bg-ic-bg-primary">
      {/* Header */}
      <div className="bg-ic-surface border-b border-ic-border">
        <div className="max-w-[1400px] mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <h1 className="text-2xl font-bold text-ic-text-primary">Earnings Calendar</h1>
          <p className="mt-1 text-ic-text-muted">Track upcoming and recent earnings reports</p>
        </div>
      </div>

      <div className="max-w-[1400px] mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-4">
        {/* Date Strip */}
        <div className="bg-ic-surface rounded-lg border border-ic-border p-4">
          <div className="flex items-center justify-between">
            <button
              onClick={handlePrevWeek}
              className="p-2 rounded-md hover:bg-ic-bg-secondary text-ic-text-muted hover:text-ic-text-primary transition-colors"
            >
              <ChevronLeftIcon className="h-5 w-5" />
            </button>

            <div className="flex gap-2 overflow-x-auto">
              {weekDays.map((day, i) => {
                const iso = formatDateISO(day);
                const count = data?.earningsCounts?.[iso] ?? 0;
                const isSelected = viewMode === 'daily' && selectedDate === iso;
                const isToday = iso === formatDateISO(today);

                return (
                  <button
                    key={iso}
                    onClick={() => {
                      setSelectedDate(iso);
                      setViewMode('daily');
                    }}
                    className={`flex flex-col items-center px-4 py-2 rounded-lg min-w-[80px] transition-colors ${
                      isSelected
                        ? 'bg-ic-blue text-white'
                        : 'hover:bg-ic-bg-secondary text-ic-text-secondary'
                    }`}
                  >
                    <span className="text-xs font-medium">{DAY_NAMES[i]}</span>
                    <span
                      className={`text-sm font-semibold ${isToday && !isSelected ? 'text-ic-blue' : ''}`}
                    >
                      {formatDateShort(day)}
                    </span>
                    {count > 0 && (
                      <span
                        className={`mt-1 text-xs px-2 py-0.5 rounded-full font-medium ${
                          isSelected ? 'bg-white/20 text-white' : 'bg-ic-blue/10 text-ic-blue'
                        }`}
                      >
                        {count}
                      </span>
                    )}
                  </button>
                );
              })}
            </div>

            <button
              onClick={handleNextWeek}
              className="p-2 rounded-md hover:bg-ic-bg-secondary text-ic-text-muted hover:text-ic-text-primary transition-colors"
            >
              <ChevronRightIcon className="h-5 w-5" />
            </button>
          </div>
        </div>

        {/* Filters */}
        <div className="flex flex-wrap items-center gap-3">
          {/* Search */}
          <div className="relative flex-1 min-w-[200px] max-w-sm">
            <MagnifyingGlassIcon className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-ic-text-dim" />
            <input
              type="text"
              placeholder="Search by symbol..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full pl-9 pr-3 py-2 bg-ic-surface border border-ic-border rounded-lg text-sm text-ic-text-primary placeholder-ic-text-dim focus:outline-none focus:ring-2 focus:ring-ic-blue/50 focus:border-ic-blue"
            />
          </div>

          {/* View Mode Toggle */}
          <div className="flex rounded-lg border border-ic-border overflow-hidden">
            <button
              onClick={() => setViewMode('daily')}
              className={`px-4 py-2 text-sm font-medium transition-colors ${
                viewMode === 'daily'
                  ? 'bg-ic-blue text-white'
                  : 'bg-ic-surface text-ic-text-muted hover:bg-ic-bg-secondary'
              }`}
            >
              Daily
            </button>
            <button
              onClick={() => setViewMode('weekly')}
              className={`px-4 py-2 text-sm font-medium transition-colors ${
                viewMode === 'weekly'
                  ? 'bg-ic-blue text-white'
                  : 'bg-ic-surface text-ic-text-muted hover:bg-ic-bg-secondary'
              }`}
            >
              Weekly
            </button>
          </div>

          {/* Result count */}
          <span className="text-sm text-ic-text-muted">
            {filteredEarnings.length} result{filteredEarnings.length !== 1 ? 's' : ''}
          </span>
        </div>

        {/* Table */}
        <div className="bg-ic-surface rounded-lg border border-ic-border overflow-hidden">
          {loading ? (
            <CalendarTableSkeleton />
          ) : error ? (
            <div className="p-8 text-center">
              <p className="text-ic-text-muted text-lg">{error}</p>
            </div>
          ) : filteredEarnings.length === 0 ? (
            <div className="p-8 text-center">
              <p className="text-ic-text-muted text-lg">No earnings found</p>
              <p className="text-ic-text-dim text-sm mt-1">
                {search
                  ? 'Try a different search term'
                  : 'No earnings reports scheduled for this period'}
              </p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-ic-border">
                    <SortableHeader
                      label="Symbol"
                      field="symbol"
                      currentField={sortField}
                      currentOrder={sortOrder}
                      onClick={handleSort}
                      align="left"
                    />
                    {viewMode === 'weekly' && (
                      <SortableHeader
                        label="Date"
                        field="date"
                        currentField={sortField}
                        currentOrder={sortOrder}
                        onClick={handleSort}
                        align="left"
                      />
                    )}
                    <SortableHeader
                      label="EPS Est"
                      field="epsEstimated"
                      currentField={sortField}
                      currentOrder={sortOrder}
                      onClick={handleSort}
                      align="right"
                    />
                    <SortableHeader
                      label="EPS Actual"
                      field="epsActual"
                      currentField={sortField}
                      currentOrder={sortOrder}
                      onClick={handleSort}
                      align="right"
                    />
                    <SortableHeader
                      label="EPS Surprise"
                      field="epsSurprise"
                      currentField={sortField}
                      currentOrder={sortOrder}
                      onClick={handleSort}
                      align="right"
                    />
                    <SortableHeader
                      label="Rev Est"
                      field="revenueEstimated"
                      currentField={sortField}
                      currentOrder={sortOrder}
                      onClick={handleSort}
                      align="right"
                    />
                    <SortableHeader
                      label="Rev Actual"
                      field="revenueActual"
                      currentField={sortField}
                      currentOrder={sortOrder}
                      onClick={handleSort}
                      align="right"
                    />
                  </tr>
                </thead>
                <tbody className="divide-y divide-ic-border">
                  {filteredEarnings.map((e, idx) => (
                    <tr
                      key={`${e.symbol}-${e.date}-${idx}`}
                      className="hover:bg-ic-bg-secondary transition-colors"
                    >
                      <td className="px-4 py-3 text-sm">
                        <Link
                          href={`/ticker/${e.symbol}?tab=earnings`}
                          className="font-semibold text-ic-blue hover:underline"
                        >
                          {e.symbol}
                        </Link>
                      </td>
                      {viewMode === 'weekly' && (
                        <td className="px-4 py-3 text-sm text-ic-text-secondary whitespace-nowrap">
                          {new Date(e.date + 'T12:00:00').toLocaleDateString('en-US', {
                            month: 'short',
                            day: 'numeric',
                          })}
                        </td>
                      )}
                      <td className="px-4 py-3 text-sm text-ic-text-secondary text-right tabular-nums">
                        {formatEPS(e.epsEstimated)}
                      </td>
                      <td className="px-4 py-3 text-sm text-right tabular-nums">
                        {e.epsActual != null ? (
                          <span
                            className={
                              e.epsBeat
                                ? 'text-green-400'
                                : e.epsBeat === false
                                  ? 'text-red-400'
                                  : 'text-ic-text-secondary'
                            }
                          >
                            {formatEPS(e.epsActual)}
                          </span>
                        ) : (
                          <span className="text-ic-text-dim">—</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-sm text-right tabular-nums">
                        {e.epsSurprisePercent != null ? (
                          <span
                            className={
                              e.epsSurprisePercent >= 0 ? 'text-green-400' : 'text-red-400'
                            }
                          >
                            {formatSurprise(e.epsSurprisePercent)}
                          </span>
                        ) : (
                          <span className="text-ic-text-dim">—</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-sm text-ic-text-secondary text-right tabular-nums">
                        {formatRevenue(e.revenueEstimated)}
                      </td>
                      <td className="px-4 py-3 text-sm text-right tabular-nums">
                        {e.revenueActual != null ? (
                          <span
                            className={
                              e.revenueBeat
                                ? 'text-green-400'
                                : e.revenueBeat === false
                                  ? 'text-red-400'
                                  : 'text-ic-text-secondary'
                            }
                          >
                            {formatRevenue(e.revenueActual)}
                          </span>
                        ) : (
                          <span className="text-ic-text-dim">—</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// ============================================================================
// Sub-components
// ============================================================================

function SortableHeader({
  label,
  field,
  currentField,
  currentOrder,
  onClick,
  align,
}: {
  label: string;
  field: SortField;
  currentField: SortField;
  currentOrder: SortOrder;
  onClick: (field: SortField) => void;
  align: 'left' | 'right';
}) {
  const isActive = currentField === field;
  return (
    <th
      className={`px-4 py-3 text-xs font-medium uppercase tracking-wider cursor-pointer hover:bg-ic-bg-tertiary transition-colors whitespace-nowrap ${
        align === 'right' ? 'text-right' : 'text-left'
      } ${isActive ? 'text-ic-text-primary' : 'text-ic-text-muted'}`}
      onClick={() => onClick(field)}
    >
      {label}
      {isActive && <span className="ml-1">{currentOrder === 'asc' ? '↑' : '↓'}</span>}
    </th>
  );
}

function CalendarTableSkeleton() {
  return (
    <div className="animate-pulse p-4 space-y-3">
      {Array.from({ length: 10 }, (_, i) => (
        <div key={i} className="h-10 bg-ic-bg-tertiary rounded" />
      ))}
    </div>
  );
}
