'use client';

import { useEffect, useState, useCallback, useRef, useMemo } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { watchListAPI, WatchListWithItems, WatchListItem } from '@/lib/api/watchlist';
import { ProtectedRoute } from '@/components/auth/ProtectedRoute';
import { useToast } from '@/lib/hooks/useToast';
import { useWatchlistAlerts } from '@/lib/hooks/useWatchlistAlerts';
import {
  alertAPI,
  AlertRuleWithDetails,
  CreateAlertRequest,
  UpdateAlertRequest,
} from '@/lib/api/alerts';
import WatchListTable from '@/components/watchlist/WatchListTable';
import { useWatchlistPageStore } from '@/lib/stores/watchlistPageStore';
import WatchlistSearchInput from '@/components/watchlist/WatchlistSearchInput';
import InlineEditPanel from '@/components/watchlist/InlineEditPanel';
import AlertCard from '@/components/alerts/AlertCard';
import BulkAlertModal from '@/components/watchlist/BulkAlertModal';

export default function WatchListDetailPage() {
  const params = useParams();
  const router = useRouter();
  const watchListId = params.id as string;
  const toast = useToast();

  const [watchList, setWatchList] = useState<WatchListWithItems | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [editingSymbol, setEditingSymbol] = useState<string | null>(null);
  const [tagSuggestions, setTagSuggestions] = useState<string[]>([]);
  const [activeTab, setActiveTab] = useState<'table' | 'alerts'>('table');
  const [showBulkModal, setShowBulkModal] = useState(false);

  // ── Alerts tab state ──────────────────────────────────────────────
  const [tabAlerts, setTabAlerts] = useState<AlertRuleWithDetails[]>([]);
  const [tabAlertsLoading, setTabAlertsLoading] = useState(false);
  const [tabAlertsFilter, setTabAlertsFilter] = useState<'all' | 'active' | 'inactive'>('all');
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);

  // Zustand store for cross-component communication (header search → watchlist)
  const setActiveWatchlist = useWatchlistPageStore((s) => s.setActiveWatchlist);
  const clearActiveWatchlist = useWatchlistPageStore((s) => s.clearActiveWatchlist);
  const addSymbolToSet = useWatchlistPageStore((s) => s.addSymbolToSet);

  // Track the latest watchList ref for the Zustand callback
  const watchListRef = useRef(watchList);
  watchListRef.current = watchList;

  // ── Load watchlist data ─────────────────────────────────────────────

  const loadWatchList = useCallback(async () => {
    try {
      const data = await watchListAPI.getWatchList(watchListId);
      setWatchList(data);
    } catch (err: any) {
      setError(err.message || 'Failed to load watch list');
    } finally {
      setLoading(false);
    }
  }, [watchListId]);

  useEffect(() => {
    loadWatchList();
    // Set up auto-refresh for real-time prices
    const interval = setInterval(loadWatchList, 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, [loadWatchList]);

  // ── Load tag suggestions (for inline edit autocomplete) ─────────────

  useEffect(() => {
    watchListAPI
      .getUserTags()
      .then((res) => setTagSuggestions((res.tags ?? []).map((t) => t.name)))
      .catch(() => {
        /* tag suggestions are non-critical */
      });
  }, []);

  // ── Alerts for this watchlist ────────────────────────────────────────

  const {
    alertsBySymbol,
    createAlert,
    updateAlert,
    deleteAlert,
    refresh: refreshAlerts,
  } = useWatchlistAlerts(watchListId);

  const handleAlertCreate = useCallback(
    async (req: CreateAlertRequest) => {
      // Let errors propagate to AlertQuickPanel's catch handler for display.
      // Only refresh watchlist data (alert_count) on confirmed success.
      const result = await createAlert(req);
      toast.success('Alert created');
      loadWatchList(); // refresh alert_count in table (fire-and-forget)
      return result;
    },
    [createAlert, loadWatchList, toast]
  );

  const handleAlertUpdate = useCallback(
    async (alertId: string, req: UpdateAlertRequest) => {
      await updateAlert(alertId, req);
      toast.success('Alert updated');
    },
    [updateAlert, toast]
  );

  const handleAlertDelete = useCallback(
    async (alertId: string, symbol: string) => {
      // Let errors propagate so the caller knows the delete failed.
      // Only refresh watchlist data on confirmed success.
      await deleteAlert(alertId, symbol);
      toast.success('Alert removed');
      loadWatchList(); // refresh alert_count (fire-and-forget)
    },
    [deleteAlert, loadWatchList, toast]
  );

  // ── Quick add handler (inline search + header integration) ──────────

  const handleQuickAdd = useCallback(
    async (symbol: string) => {
      if (!watchListRef.current) return;

      // Optimistic update: add a placeholder item to the table immediately.
      // Explicitly sets all required WatchListItem fields to null/defaults so
      // downstream rendering handles missing data gracefully. Replaced by the
      // full server response within milliseconds via loadWatchList().
      const optimisticItem: WatchListItem = {
        id: `optimistic-${symbol}`,
        watch_list_id: watchListId,
        symbol,
        name: symbol,
        exchange: '',
        asset_type: '',
        tags: [],
        added_at: new Date().toISOString(),
        display_order: (watchListRef.current.items.length ?? 0) + 1,
        alert_count: 0,
        // Price and market data — populated after server round-trip
        current_price: undefined,
        price_change: undefined,
        price_change_pct: undefined,
        volume: undefined,
        market_cap: undefined,
        prev_close: undefined,
        // IC Score fields
        ic_score: null,
        ic_rating: null,
        value_score: null,
        growth_score: null,
        profitability_score: null,
        financial_health_score: null,
        momentum_score: null,
        analyst_consensus_score: null,
        insider_activity_score: null,
        institutional_score: null,
        news_sentiment_score: null,
        technical_score: null,
        sector_percentile: null,
        lifecycle_stage: null,
        // Fundamental fields
        pe_ratio: null,
        pb_ratio: null,
        ps_ratio: null,
        roe: null,
        roa: null,
        gross_margin: null,
        operating_margin: null,
        net_margin: null,
        debt_to_equity: null,
        current_ratio: null,
        revenue_growth: null,
        eps_growth: null,
        dividend_yield: null,
        payout_ratio: null,
      };

      // Apply optimistic update
      setWatchList((prev) => {
        if (!prev) return prev;
        return {
          ...prev,
          item_count: prev.item_count + 1,
          items: [...prev.items, optimisticItem],
        };
      });

      // Update Zustand store so header search shows "Added"
      addSymbolToSet(symbol);

      try {
        await watchListAPI.addTicker(watchListId, { symbol });
        // Refresh to get real data (prices, name, etc.)
        await loadWatchList();
        toast.success(`${symbol} added to watch list`);
      } catch (err: any) {
        // Rollback optimistic update — filter by the optimistic ID rather than
        // symbol to avoid accidentally removing a pre-existing real item.
        const optimisticId = `optimistic-${symbol}`;
        setWatchList((prev) => {
          if (!prev) return prev;
          return {
            ...prev,
            item_count: prev.item_count - 1,
            items: prev.items.filter((i) => i.id !== optimisticId),
          };
        });
        toast.error(err.message || `Failed to add ${symbol}`);
      }
    },
    [watchListId, loadWatchList, toast, addSymbolToSet]
  );

  // ── Register with Zustand store (for header search integration) ─────

  useEffect(() => {
    if (!watchList) return;

    const symbols = watchList.items.map((i) => i.symbol);
    setActiveWatchlist(watchListId, watchList.name, symbols, handleQuickAdd);

    return () => {
      clearActiveWatchlist();
    };
  }, [
    watchList?.name,
    watchList?.items.length,
    watchListId,
    handleQuickAdd,
    setActiveWatchlist,
    clearActiveWatchlist,
  ]);

  const handleRemoveTicker = async (symbol: string) => {
    if (!confirm(`Remove ${symbol} from watch list?`)) return;

    try {
      await watchListAPI.removeTicker(watchListId, symbol);
      await loadWatchList();
      toast.success(`${symbol} removed from watch list`);
    } catch (err: any) {
      toast.error(err.message || 'Failed to remove ticker');
    }
  };

  const handleUpdateTicker = async (
    symbol: string,
    data: { notes?: string; tags?: string[]; target_buy_price?: number; target_sell_price?: number }
  ) => {
    try {
      await watchListAPI.updateTicker(watchListId, symbol, data);
      await loadWatchList();
      setEditingSymbol(null);
      toast.success(`${symbol} updated successfully`);
    } catch (err: any) {
      toast.error(err.message || 'Failed to update ticker');
    }
  };

  // ── Existing symbols set (for search input "already added" display) ─

  const existingSymbols = useMemo(
    () => new Set(watchList?.items.map((i) => i.symbol) ?? []),
    [watchList?.items]
  );

  // ── Alerts tab: fetch alerts when tab is active ─────────────────────

  const loadTabAlerts = useCallback(async () => {
    setTabAlertsLoading(true);
    try {
      const data = await alertAPI.listAlerts({
        watch_list_id: watchListId,
        is_active: tabAlertsFilter === 'all' ? undefined : tabAlertsFilter === 'active',
      });
      setTabAlerts(data ?? []);
    } catch (err: unknown) {
      console.error('Failed to load alerts tab:', err);
    } finally {
      setTabAlertsLoading(false);
    }
  }, [watchListId, tabAlertsFilter]);

  useEffect(() => {
    if (activeTab === 'alerts') {
      loadTabAlerts();
    }
  }, [activeTab, loadTabAlerts]);

  const handleTabAlertToggle = async (alert: AlertRuleWithDetails) => {
    try {
      await alertAPI.updateAlert(alert.id, { is_active: !alert.is_active });
      await loadTabAlerts();
      await refreshAlerts(); // sync the bell icons in the table tab
      toast.success(alert.is_active ? 'Alert paused' : 'Alert resumed');
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to update alert';
      toast.error(msg);
    }
  };

  const handleTabAlertDelete = async (alertId: string) => {
    // First click sets confirmation state; second click (from inline confirm
    // button rendered in the Alerts tab) performs the actual delete.
    if (confirmDeleteId !== alertId) {
      setConfirmDeleteId(alertId);
      return;
    }
    setConfirmDeleteId(null);
    try {
      await alertAPI.deleteAlert(alertId);
      await loadTabAlerts();
      await refreshAlerts();
      loadWatchList(); // refresh alert_count
      toast.success('Alert deleted');
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to delete alert';
      toast.error(msg);
    }
  };

  // eslint-disable-next-line @typescript-eslint/no-unused-vars -- edit from alerts tab deferred to Phase 4
  const handleTabAlertEdit = (_alert: AlertRuleWithDetails) => {
    // Phase 4: open edit modal. For now, users edit via the bell popover on the Table tab.
    toast.info('Switch to the Table tab and click the bell icon to edit alerts');
  };

  const handleBulkSuccess = (message: string) => {
    setShowBulkModal(false);
    loadWatchList(); // refresh alert_count in the table
    refreshAlerts(); // refresh alertsBySymbol map
    if (activeTab === 'alerts') loadTabAlerts(); // refresh alerts list
    toast.success(message);
  };

  const tabActiveCount = tabAlerts.filter((a) => a.is_active).length;
  const tabInactiveCount = tabAlerts.filter((a) => !a.is_active).length;

  // ── Loading state ───────────────────────────────────────────────────

  if (loading) {
    return (
      <ProtectedRoute>
        <div className="flex items-center justify-center min-h-screen">
          <div className="text-xl">Loading...</div>
        </div>
      </ProtectedRoute>
    );
  }

  if (!watchList) {
    return (
      <ProtectedRoute>
        <div className="container mx-auto px-4 py-8">
          <div className="text-center">
            <p className="text-red-600">{error || 'Watch list not found'}</p>
          </div>
        </div>
      </ProtectedRoute>
    );
  }

  // ── Render ──────────────────────────────────────────────────────────

  return (
    <ProtectedRoute>
      <div className="container mx-auto px-4 py-8">
        <div className="flex justify-between items-center mb-6">
          <div>
            <h1 className="text-3xl font-bold">{watchList.name}</h1>
            {watchList.description && (
              <p className="text-ic-text-muted mt-2">{watchList.description}</p>
            )}
            <p className="text-sm text-ic-text-dim mt-1">{watchList.item_count} tickers</p>
          </div>
          <div className="flex gap-3">
            {watchList.items.length > 0 && (
              <button
                onClick={() => setShowBulkModal(true)}
                className="px-4 py-2 bg-ic-surface border border-ic-border text-ic-text-secondary rounded hover:bg-ic-surface-hover flex items-center gap-2"
              >
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
                  />
                </svg>
                Alert All
              </button>
            )}
            <button
              onClick={() => router.push(`/watchlist/${watchListId}/heatmap`)}
              className="px-4 py-2 bg-ic-purple text-ic-text-primary rounded hover:bg-ic-purple-hover flex items-center gap-2"
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
                />
              </svg>
              View Heatmap
            </button>
          </div>
        </div>

        {/* ── Tab bar ──────────────────────────────────────────────── */}
        <div className="flex gap-2 mb-4" role="tablist" aria-label="Watchlist views">
          <button
            role="tab"
            aria-selected={activeTab === 'table'}
            onClick={() => setActiveTab('table')}
            className={`px-4 py-2 text-sm font-medium rounded-lg transition-colors ${
              activeTab === 'table'
                ? 'bg-ic-blue text-white'
                : 'bg-ic-surface border border-ic-border text-ic-text-secondary hover:bg-ic-surface-hover'
            }`}
          >
            Table
          </button>
          <button
            role="tab"
            aria-selected={activeTab === 'alerts'}
            onClick={() => setActiveTab('alerts')}
            className={`px-4 py-2 text-sm font-medium rounded-lg transition-colors ${
              activeTab === 'alerts'
                ? 'bg-ic-blue text-white'
                : 'bg-ic-surface border border-ic-border text-ic-text-secondary hover:bg-ic-surface-hover'
            }`}
          >
            Alerts
          </button>
        </div>

        {error && (
          <div className="mb-4 p-3 bg-red-500/10 border border-red-500/30 text-ic-negative rounded">
            {error}
          </div>
        )}

        {/* ── Table tab ──────────────────────────────────────────── */}
        {activeTab === 'table' && (
          <>
            <WatchlistSearchInput
              onAdd={handleQuickAdd}
              existingSymbols={existingSymbols}
              itemCount={watchList.item_count}
              maxItems={50}
              className="mb-4"
            />

            {watchList.items.length === 0 ? (
              <div className="text-center py-12 bg-ic-bg-secondary rounded-lg">
                <svg
                  className="mx-auto h-12 w-12 text-ic-text-muted mb-4"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
                  />
                </svg>
                <h3 className="text-lg font-semibold text-ic-text-primary mb-2">No tickers yet</h3>
                <p className="text-ic-text-muted mb-6 max-w-md mx-auto">
                  Add stocks or cryptocurrencies to this watch list to track their real-time prices,
                  set target alerts, and monitor your investments.
                </p>
                <p className="text-ic-text-secondary text-sm">
                  Use the search bar above or press{' '}
                  <kbd className="border border-ic-border rounded px-1.5 py-0.5 font-mono bg-ic-bg-tertiary text-xs">
                    /
                  </kbd>{' '}
                  to start adding tickers.
                </p>
                <div className="mt-6 text-sm text-ic-text-dim">
                  <p>Examples: AAPL, TSLA, X:BTCUSD, X:ETHUSD</p>
                </div>
              </div>
            ) : (
              <WatchListTable
                items={watchList.items}
                onRemove={handleRemoveTicker}
                onEdit={setEditingSymbol}
                expandedSymbol={editingSymbol}
                watchListId={watchListId}
                alertsBySymbol={alertsBySymbol}
                onAlertCreate={handleAlertCreate}
                onAlertUpdate={handleAlertUpdate}
                onAlertDelete={handleAlertDelete}
                renderExpandedRow={(item: WatchListItem) => (
                  <InlineEditPanel
                    item={item}
                    tagSuggestions={tagSuggestions}
                    onSave={handleUpdateTicker}
                    onCancel={() => setEditingSymbol(null)}
                  />
                )}
              />
            )}
          </>
        )}

        {/* ── Alerts tab ─────────────────────────────────────────── */}
        {activeTab === 'alerts' && (
          <div>
            {/* Filter bar */}
            <div className="flex gap-2 mb-4">
              <button
                onClick={() => setTabAlertsFilter('all')}
                className={`px-3 py-1.5 text-sm font-medium rounded-lg transition-colors ${
                  tabAlertsFilter === 'all'
                    ? 'bg-ic-purple text-ic-text-primary'
                    : 'bg-ic-surface border border-ic-border text-ic-text-dim hover:text-ic-text-primary'
                }`}
              >
                All ({tabAlerts.length})
              </button>
              <button
                onClick={() => setTabAlertsFilter('active')}
                className={`px-3 py-1.5 text-sm font-medium rounded-lg transition-colors ${
                  tabAlertsFilter === 'active'
                    ? 'bg-ic-positive text-ic-text-primary'
                    : 'bg-ic-surface border border-ic-border text-ic-text-dim hover:text-ic-text-primary'
                }`}
              >
                Active ({tabActiveCount})
              </button>
              <button
                onClick={() => setTabAlertsFilter('inactive')}
                className={`px-3 py-1.5 text-sm font-medium rounded-lg transition-colors ${
                  tabAlertsFilter === 'inactive'
                    ? 'bg-ic-bg-tertiary text-ic-text-primary'
                    : 'bg-ic-surface border border-ic-border text-ic-text-dim hover:text-ic-text-primary'
                }`}
              >
                Inactive ({tabInactiveCount})
              </button>
            </div>

            {tabAlertsLoading ? (
              <div className="flex justify-center py-12">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-ic-blue"></div>
              </div>
            ) : tabAlerts.length === 0 ? (
              <div className="text-center py-12 bg-ic-bg-secondary rounded-lg">
                <svg
                  className="mx-auto h-12 w-12 text-ic-text-muted mb-4"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
                  />
                </svg>
                <h3 className="text-lg font-semibold text-ic-text-primary mb-2">
                  {tabAlertsFilter === 'all' ? 'No alerts yet' : `No ${tabAlertsFilter} alerts`}
                </h3>
                <p className="text-ic-text-muted mb-4 max-w-md mx-auto">
                  Click the bell icon on any ticker in the Table tab to create an alert, or use
                  &ldquo;Alert All&rdquo; to set alerts for every ticker at once.
                </p>
              </div>
            ) : (
              <div className="grid gap-4">
                {tabAlerts.map((alert) => (
                  <div key={alert.id} className="relative">
                    <AlertCard
                      alert={alert}
                      onEdit={handleTabAlertEdit}
                      onToggleActive={handleTabAlertToggle}
                      onDelete={handleTabAlertDelete}
                    />
                    {confirmDeleteId === alert.id && (
                      <div className="absolute inset-0 flex items-center justify-center bg-ic-bg-tertiary/80 rounded-lg backdrop-blur-sm z-10">
                        <div className="bg-ic-surface border border-ic-border rounded-lg p-4 shadow-lg text-center">
                          <p className="text-sm text-ic-text-primary mb-3">Delete this alert?</p>
                          <div className="flex gap-2 justify-center">
                            <button
                              onClick={() => setConfirmDeleteId(null)}
                              className="px-3 py-1.5 text-xs font-medium text-ic-text-secondary bg-ic-surface border border-ic-border rounded hover:bg-ic-surface-hover"
                            >
                              Cancel
                            </button>
                            <button
                              onClick={() => handleTabAlertDelete(alert.id)}
                              className="px-3 py-1.5 text-xs font-medium text-white bg-ic-negative rounded hover:bg-red-600"
                            >
                              Delete
                            </button>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* ── Bulk Alert Modal ───────────────────────────────────── */}
        {showBulkModal && watchList && (
          <BulkAlertModal
            watchListId={watchListId}
            watchListName={watchList.name}
            tickerCount={watchList.items.length}
            onClose={() => setShowBulkModal(false)}
            onSuccess={handleBulkSuccess}
          />
        )}
      </div>
    </ProtectedRoute>
  );
}
