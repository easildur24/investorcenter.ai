'use client';

import { useEffect, useState } from 'react';
import { useAuth } from '@/lib/auth/AuthContext';
import { watchListAPI, WatchList } from '@/lib/api/watchlist';
import { ProtectedRoute } from '@/components/auth/ProtectedRoute';
import Link from 'next/link';
import CreateWatchListModal from '@/components/watchlist/CreateWatchListModal';

export default function WatchListDashboard() {
  const { user } = useAuth();
  const [watchLists, setWatchLists] = useState<WatchList[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    // Only load watch lists if user is authenticated
    if (user) {
      loadWatchLists();
    }
  }, [user]);

  const loadWatchLists = async () => {
    try {
      const data = await watchListAPI.getWatchLists();
      setWatchLists(data.watch_lists);
    } catch (err: any) {
      setError(err.message || 'Failed to load watch lists');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateWatchList = async (name: string, description?: string) => {
    try {
      await watchListAPI.createWatchList({ name, description });
      await loadWatchLists();
      setShowCreateModal(false);
    } catch (err: any) {
      setError(err.message || 'Failed to create watch list');
    }
  };

  const handleDeleteWatchList = async (id: string) => {
    if (!confirm('Are you sure you want to delete this watch list?')) return;

    try {
      await watchListAPI.deleteWatchList(id);
      await loadWatchLists();
    } catch (err: any) {
      setError(err.message || 'Failed to delete watch list');
    }
  };

  return (
    <ProtectedRoute>
      <div className="container mx-auto px-4 py-8">
        <div className="flex justify-between items-center mb-8">
          <h1 className="text-3xl font-bold">My Watch Lists</h1>
          <button
            onClick={() => setShowCreateModal(true)}
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
          >
            + Create Watch List
          </button>
        </div>

        {error && (
          <div className="mb-4 p-3 bg-red-100 border border-red-400 text-red-700 rounded">
            {error}
          </div>
        )}

        {loading ? (
          <div className="text-center py-12">Loading...</div>
        ) : watchLists.length === 0 ? (
          <div className="text-center py-12">
            <p className="text-gray-600 mb-4">You don't have any watch lists yet.</p>
            <button
              onClick={() => setShowCreateModal(true)}
              className="px-6 py-3 bg-blue-600 text-white rounded hover:bg-blue-700"
            >
              Create Your First Watch List
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {watchLists.map((watchList) => (
              <div key={watchList.id} className="bg-white p-6 rounded-lg shadow hover:shadow-lg transition">
                <div className="flex justify-between items-start mb-3">
                  <h3 className="text-xl font-semibold">{watchList.name}</h3>
                  {watchList.is_default && (
                    <span className="text-xs bg-blue-100 text-blue-800 px-2 py-1 rounded">Default</span>
                  )}
                </div>

                {watchList.description && (
                  <p className="text-gray-600 text-sm mb-4">{watchList.description}</p>
                )}

                <div className="flex items-center justify-between mb-4">
                  <span className="text-gray-500">{watchList.item_count} tickers</span>
                  <span className="text-xs text-gray-400">
                    {new Date(watchList.updated_at).toLocaleDateString()}
                  </span>
                </div>

                <div className="flex gap-2">
                  <Link
                    href={`/watchlist/${watchList.id}`}
                    className="flex-1 text-center px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
                  >
                    View
                  </Link>
                  <button
                    onClick={() => handleDeleteWatchList(watchList.id)}
                    className="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700"
                  >
                    Delete
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}

        {showCreateModal && (
          <CreateWatchListModal
            onClose={() => setShowCreateModal(false)}
            onCreate={handleCreateWatchList}
          />
        )}
      </div>
    </ProtectedRoute>
  );
}
