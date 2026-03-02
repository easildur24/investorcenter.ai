'use client';

import { useState, useEffect, useCallback } from 'react';
import Link from 'next/link';
import { GICS_SECTORS } from '@/config/gics-sectors';
import { useWidgetTracking } from '@/lib/hooks/useWidgetTracking';
import { apiClient } from '@/lib/api';
import type { ScreenerStock } from '@/lib/types/screener';
import { Squares2X2Icon } from '@heroicons/react/24/outline';

// ============================================================================
// Types
// ============================================================================

interface SectorData {
  id: string;
  label: string;
  avgChange: number | null; // null = no data available
  stockCount: number;
}

// ============================================================================
// Helpers
// ============================================================================

/**
 * Map a screener stock's sector string to a GICS sector id.
 * The screener returns sector names like "Technology", "Healthcare", etc.
 * We normalize and match against GICS_SECTORS ids and labels.
 */
function normalizeSectorId(sectorName: string): string | null {
  if (!sectorName) return null;
  const normalized = sectorName.toLowerCase().replace(/[\s-]+/g, '_');
  const match = GICS_SECTORS.find(
    (s) =>
      s.id === normalized ||
      s.label.toLowerCase().replace(/[\s.-]+/g, '_') === normalized ||
      sectorName.toLowerCase().includes(s.label.toLowerCase().split(' ')[0].toLowerCase())
  );
  return match?.id ?? null;
}

/**
 * Compute heatmap background color based on change percentage.
 * Green for positive, red for negative, with intensity proportional to magnitude.
 * Returns an rgba string for light/dark compatibility.
 */
function getHeatmapColor(change: number | null): string {
  if (change === null || change === 0) {
    return 'rgba(128, 128, 128, 0.08)';
  }

  // Clamp magnitude for intensity calculation (max out at 5% for full intensity)
  const magnitude = Math.min(Math.abs(change), 5);
  const intensity = 0.08 + (magnitude / 5) * 0.22; // range: 0.08 to 0.30

  if (change > 0) {
    return `rgba(34, 197, 94, ${intensity})`; // green-500
  }
  return `rgba(239, 68, 68, ${intensity})`; // red-500
}

/**
 * Get the text color class for the change value.
 */
function getChangeTextClass(change: number | null): string {
  if (change === null || change === 0) return 'text-ic-text-muted';
  return change > 0 ? 'text-ic-positive' : 'text-ic-negative';
}

/**
 * Format a change percentage for display.
 */
function formatChange(change: number | null): string {
  if (change === null) return 'No data';
  if (change === 0) return '0.00%';
  const sign = change > 0 ? '+' : '';
  return `${sign}${change.toFixed(2)}%`;
}

// ============================================================================
// Data Fetching
// ============================================================================

/**
 * Fetch screener data and group by sector to compute average price change.
 * Since the screener doesn't expose a direct "daily change" field, we use
 * revenue_growth as a proxy for demonstration. In the future, a dedicated
 * sector performance endpoint would provide actual daily/weekly/monthly changes.
 *
 * Falls back to null (no data) values if the API is unreachable.
 */
async function fetchSectorData(): Promise<SectorData[]> {
  // Initialize all sectors with null
  const sectorMap = new Map<string, { totalChange: number; count: number }>();

  try {
    // Fetch a broad set of stocks to get sector representation
    const response = await apiClient.getScreenerStocks({
      limit: 250,
      sort: 'market_cap',
      order: 'desc',
      asset_type: 'stock',
    });

    const stocks: ScreenerStock[] = response.data?.data ?? [];

    // Group stocks by sector and compute average revenue_growth as a proxy
    for (const stock of stocks) {
      if (!stock.sector) continue;
      const sectorId = normalizeSectorId(stock.sector);
      if (!sectorId) continue;

      // Use revenue_growth as a change proxy (it's what the screener provides)
      const change = stock.revenue_growth;
      if (change === null || change === undefined) continue;

      const existing = sectorMap.get(sectorId);
      if (existing) {
        existing.totalChange += change;
        existing.count += 1;
      } else {
        sectorMap.set(sectorId, { totalChange: change, count: 1 });
      }
    }
  } catch (err) {
    console.error('SectorHeatmap: failed to fetch screener data', err);
    // Fall through — sectors will show "No data"
  }

  // Build final sector data array
  return GICS_SECTORS.map((sector) => {
    const data = sectorMap.get(sector.id);
    return {
      id: sector.id,
      label: sector.label,
      avgChange: data ? data.totalChange / data.count : null,
      stockCount: data?.count ?? 0,
    };
  });
}

// ============================================================================
// Component
// ============================================================================

export default function SectorHeatmap() {
  const { ref: widgetRef, trackInteraction } = useWidgetTracking('sector_heatmap');
  const [sectors, setSectors] = useState<SectorData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await fetchSectorData();
      setSectors(data);
    } catch (err) {
      console.error('SectorHeatmap: unexpected error', err);
      setError('Failed to load sector data');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // ── Loading State ──────────────────────────────────────────────────────────
  if (loading) {
    return (
      <div
        className="bg-ic-surface rounded-lg border border-ic-border p-6"
        style={{ boxShadow: 'var(--ic-shadow-card)' }}
      >
        <div className="flex items-center gap-2 mb-4">
          <Squares2X2Icon className="h-5 w-5 text-ic-text-muted" />
          <h2 className="text-lg font-semibold text-ic-text-primary">Sector Performance</h2>
        </div>
        <div className="animate-pulse grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3">
          {Array.from({ length: 11 }).map((_, i) => (
            <div key={i} className="rounded-lg p-4 space-y-2">
              <div className="h-4 bg-ic-bg-tertiary rounded w-24"></div>
              <div className="h-5 bg-ic-bg-tertiary rounded w-16"></div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  // ── Error State ────────────────────────────────────────────────────────────
  if (error) {
    return (
      <div
        className="bg-ic-surface rounded-lg border border-ic-border p-6"
        style={{ boxShadow: 'var(--ic-shadow-card)' }}
      >
        <div className="flex items-center gap-2 mb-4">
          <Squares2X2Icon className="h-5 w-5 text-ic-text-muted" />
          <h2 className="text-lg font-semibold text-ic-text-primary">Sector Performance</h2>
        </div>
        <div className="text-ic-negative text-sm">
          <p>{error}</p>
          <p className="text-ic-text-muted mt-2">
            This will work once the backend is running with market data.
          </p>
        </div>
      </div>
    );
  }

  // ── Main Render ────────────────────────────────────────────────────────────
  return (
    <div
      ref={widgetRef}
      className="bg-ic-surface rounded-lg border border-ic-border p-6"
      style={{ boxShadow: 'var(--ic-shadow-card)' }}
    >
      {/* Header with period toggle */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Squares2X2Icon className="h-5 w-5 text-ic-text-muted" />
          <h2 className="text-lg font-semibold text-ic-text-primary">Sector Performance</h2>
        </div>

        {/* TODO: Add period toggle (1D/1W/1M) when dedicated sector performance API is available */}
        <span className="text-xs text-ic-text-dim">Revenue Growth</span>
      </div>

      {/* Sector Grid */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3">
        {sectors.map((sector) => (
          <Link
            key={sector.id}
            href={`/screener?sectors=${encodeURIComponent(sector.label)}`}
            onClick={() =>
              trackInteraction('sector_click', {
                sector: sector.id,
                change: sector.avgChange,
              })
            }
            className="group relative rounded-lg p-4 transition-all hover:scale-[1.02] hover:shadow-md"
            style={{ backgroundColor: getHeatmapColor(sector.avgChange) }}
          >
            {/* Sector Name */}
            <p className="text-sm font-medium text-ic-text-primary truncate">{sector.label}</p>

            {/* Change Value */}
            <p
              className={`text-lg font-semibold tabular-nums mt-1 ${getChangeTextClass(sector.avgChange)}`}
            >
              {formatChange(sector.avgChange)}
            </p>

            {/* Stock count */}
            {sector.stockCount > 0 && (
              <p className="text-[10px] text-ic-text-dim mt-1">
                {sector.stockCount} stock{sector.stockCount !== 1 ? 's' : ''}
              </p>
            )}
          </Link>
        ))}
      </div>

      {/* Footer */}
      <div className="mt-4 pt-3 border-t border-ic-border-subtle">
        <Link
          href="/screener"
          onClick={() => trackInteraction('view_screener')}
          className="text-sm text-ic-blue hover:text-ic-blue-hover font-medium transition-colors"
        >
          Explore all sectors in Screener →
        </Link>
      </div>
    </div>
  );
}
