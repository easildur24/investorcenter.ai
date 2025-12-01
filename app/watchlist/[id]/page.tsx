'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { watchListAPI, WatchListWithItems } from '@/lib/api/watchlist';
import { ProtectedRoute } from '@/components/auth/ProtectedRoute';
import { useToast } from '@/lib/hooks/useToast';
import WatchListTable from '@/components/watchlist/WatchListTable';
import AddTickerModal from '@/components/watchlist/AddTickerModal';
import EditTickerModal from '@/components/watchlist/EditTickerModal';

export default function WatchListDetailPage() {
  const params = useParams();
  const router = useRouter();
  const watchListId = params.id as string;
  const toast = useToast();

  const [watchList, setWatchList] = useState<WatchListWithItems | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showAddModal, setShowAddModal] = useState(false);
  const [editingSymbol, setEditingSymbol] = useState<string | null>(null);

  useEffect(() => {
    loadWatchList();
    // Set up auto-refresh for real-time prices
    const interval = setInterval(loadWatchList, 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, [watchListId]);

  const loadWatchList = async () => {
    try {
      const data = await watchListAPI.getWatchList(watchListId);
      setWatchList(data);
    } catch (err: any) {
      setError(err.message || 'Failed to load watch list');
    } finally {
      setLoading(false);
    }
  };

  const handleAddTicker = async (symbol: string, notes?: string, tags?: string[], targetBuy?: number, targetSell?: number) => {
    try {
      await watchListAPI.addTicker(watchListId, {
        symbol,
        notes,
        tags,
        target_buy_price: targetBuy,
        target_sell_price: targetSell,
      });
      await loadWatchList();
      setShowAddModal(false);
      toast.success(`${symbol} added to watch list`);
    } catch (err: any) {
      toast.error(err.message || 'Failed to add ticker');
    }
  };

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

  const handleUpdateTicker = async (symbol: string, data: any) => {
    try {
      await watchListAPI.updateTicker(watchListId, symbol, data);
      await loadWatchList();
      setEditingSymbol(null);
      toast.success(`${symbol} updated successfully`);
    } catch (err: any) {
      toast.error(err.message || 'Failed to update ticker');
    }
  };

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

  return (
    <ProtectedRoute>
      <div className="container mx-auto px-4 py-8">
        <div className="flex justify-between items-center mb-6">
          <div>
            <h1 className="text-3xl font-bold">{watchList.name}</h1>
            {watchList.description && (
              <p className="text-gray-600 mt-2">{watchList.description}</p>
            )}
            <p className="text-sm text-ic-text-dim mt-1">{watchList.item_count} tickers</p>
          </div>
          <div className="flex gap-3">
            <button
              onClick={() => router.push(`/watchlist/${watchListId}/heatmap`)}
              className="px-4 py-2 bg-purple-600 text-white rounded hover:bg-purple-700 flex items-center gap-2"
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
              </svg>
              View Heatmap
            </button>
            <button
              onClick={() => setShowAddModal(true)}
              className="px-4 py-2 bg-ic-blue text-ic-text-primary rounded hover:bg-ic-blue-hover"
            >
              + Add Ticker
            </button>
          </div>
        </div>

        {error && (
          <div className="mb-4 p-3 bg-red-100 border border-red-400 text-red-700 rounded">
            {error}
          </div>
        )}

        {watchList.items.length === 0 ? (
          <div className="text-center py-12 bg-gray-50 rounded-lg">
            <svg className="mx-auto h-12 w-12 text-ic-text-muted mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
            </svg>
            <h3 className="text-lg font-semibold text-gray-900 mb-2">No tickers yet</h3>
            <p className="text-gray-600 mb-6 max-w-md mx-auto">
              Add stocks or cryptocurrencies to this watch list to track their real-time prices,
              set target alerts, and monitor your investments.
            </p>
            <button
              onClick={() => setShowAddModal(true)}
              className="px-6 py-3 bg-ic-blue text-ic-text-primary rounded-lg hover:bg-ic-blue-hover font-medium"
            >
              + Add Your First Ticker
            </button>
            <div className="mt-6 text-sm text-ic-text-dim">
              <p>Examples: AAPL, TSLA, X:BTCUSD, X:ETHUSD</p>
            </div>
          </div>
        ) : (
          <WatchListTable
            items={watchList.items}
            onRemove={handleRemoveTicker}
            onEdit={setEditingSymbol}
          />
        )}

        {showAddModal && (
          <AddTickerModal
            onClose={() => setShowAddModal(false)}
            onAdd={handleAddTicker}
          />
        )}

        {editingSymbol && (
          <EditTickerModal
            symbol={editingSymbol}
            item={watchList.items.find(i => i.symbol === editingSymbol)!}
            onClose={() => setEditingSymbol(null)}
            onUpdate={handleUpdateTicker}
          />
        )}
      </div>
    </ProtectedRoute>
  );
}
