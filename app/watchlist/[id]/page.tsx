'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { watchListAPI, WatchListWithItems } from '@/lib/api/watchlist';
import { ProtectedRoute } from '@/components/auth/ProtectedRoute';
import WatchListTable from '@/components/watchlist/WatchListTable';
import AddTickerModal from '@/components/watchlist/AddTickerModal';
import EditTickerModal from '@/components/watchlist/EditTickerModal';

export default function WatchListDetailPage() {
  const params = useParams();
  const watchListId = params.id as string;

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
    } catch (err: any) {
      alert(err.message || 'Failed to add ticker');
    }
  };

  const handleRemoveTicker = async (symbol: string) => {
    if (!confirm(`Remove ${symbol} from watch list?`)) return;

    try {
      await watchListAPI.removeTicker(watchListId, symbol);
      await loadWatchList();
    } catch (err: any) {
      alert(err.message || 'Failed to remove ticker');
    }
  };

  const handleUpdateTicker = async (symbol: string, data: any) => {
    try {
      await watchListAPI.updateTicker(watchListId, symbol, data);
      await loadWatchList();
      setEditingSymbol(null);
    } catch (err: any) {
      alert(err.message || 'Failed to update ticker');
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
            <p className="text-sm text-gray-500 mt-1">{watchList.item_count} tickers</p>
          </div>
          <button
            onClick={() => setShowAddModal(true)}
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
          >
            + Add Ticker
          </button>
        </div>

        {error && (
          <div className="mb-4 p-3 bg-red-100 border border-red-400 text-red-700 rounded">
            {error}
          </div>
        )}

        {watchList.items.length === 0 ? (
          <div className="text-center py-12 bg-gray-50 rounded-lg">
            <p className="text-gray-600 mb-4">No tickers in this watch list yet.</p>
            <button
              onClick={() => setShowAddModal(true)}
              className="px-6 py-3 bg-blue-600 text-white rounded hover:bg-blue-700"
            >
              Add Your First Ticker
            </button>
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
