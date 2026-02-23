'use client';

import { useEffect, useState } from 'react';
import { useAuth } from '@/lib/auth/AuthContext';
import { watchListAPI } from '@/lib/api/watchlist';
import { ProtectedRoute } from '@/components/auth/ProtectedRoute';
import { useRouter } from 'next/navigation';

export default function WatchListDashboard() {
  const { user } = useAuth();
  const router = useRouter();
  const [error, setError] = useState('');

  useEffect(() => {
    if (!user) return;

    (async () => {
      try {
        const data = await watchListAPI.getWatchLists();
        const lists = data.watch_lists ?? [];

        if (lists.length > 0) {
          // Redirect to the user's watchlist
          router.replace(`/watchlist/${lists[0].id}`);
        } else {
          // Auto-create a default watchlist
          const newList = await watchListAPI.createWatchList({ name: 'My Watch List' });
          router.replace(`/watchlist/${newList.id}`);
        }
      } catch (err: any) {
        setError(err.message || 'Failed to load watch list');
      }
    })();
  }, [user, router]);

  return (
    <ProtectedRoute>
      <div className="container mx-auto px-4 py-8">
        {error ? (
          <div className="text-center py-12">
            <p className="text-ic-negative">{error}</p>
          </div>
        ) : (
          <div className="text-center py-12">
            <div className="text-xl text-ic-text-muted">Loading your watch list...</div>
          </div>
        )}
      </div>
    </ProtectedRoute>
  );
}
