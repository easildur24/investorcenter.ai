'use client';

import { useState, useRef, useEffect, useCallback } from 'react';
import { BookmarkIcon } from '@heroicons/react/24/outline';
import { BookmarkIcon as BookmarkSolidIcon } from '@heroicons/react/24/solid';
import { useAuth } from '@/lib/auth/AuthContext';
import { watchListAPI, WatchList } from '@/lib/api/watchlist';
import { useToast } from '@/lib/hooks/useToast';
import { useRouter } from 'next/navigation';

interface AddToWatchlistButtonProps {
  symbol: string;
}

export default function AddToWatchlistButton({ symbol }: AddToWatchlistButtonProps) {
  const { user } = useAuth();
  const router = useRouter();
  const toast = useToast();
  const [watchlist, setWatchlist] = useState<WatchList | null>(null);
  const [isInWatchlist, setIsInWatchlist] = useState(false);
  const [adding, setAdding] = useState(false);
  const [checking, setChecking] = useState(false);
  const checkedRef = useRef(false);

  // On mount, check if this ticker is already in the user's watchlist
  const checkMembership = useCallback(async () => {
    if (!user || checkedRef.current) return;
    checkedRef.current = true;
    setChecking(true);
    try {
      const res = await watchListAPI.getWatchLists();
      const lists = res.watch_lists ?? [];
      if (lists.length === 0) {
        setChecking(false);
        return;
      }
      // Use first watchlist (one watchlist per account)
      const list = lists[0];
      setWatchlist(list);
      // Fetch full watchlist to check if symbol is in it
      const full = await watchListAPI.getWatchList(list.id);
      const found = full.items.some((item) => item.symbol === symbol);
      setIsInWatchlist(found);
    } catch {
      // Non-critical — button will just show default state
    } finally {
      setChecking(false);
    }
  }, [user, symbol]);

  useEffect(() => {
    checkMembership();
  }, [checkMembership]);

  const handleClick = async () => {
    if (!user) {
      router.push('/auth/login');
      return;
    }

    // Already in watchlist — navigate to it
    if (isInWatchlist && watchlist) {
      router.push(`/watchlist/${watchlist.id}`);
      return;
    }

    setAdding(true);
    try {
      let targetList = watchlist;

      // Auto-create watchlist if none exists
      if (!targetList) {
        targetList = await watchListAPI.createWatchList({ name: 'My Watch List' });
        setWatchlist(targetList);
      }

      await watchListAPI.addTicker(targetList.id, { symbol });
      setIsInWatchlist(true);
      toast.success(`${symbol} added to ${targetList.name}`);
    } catch (err: any) {
      const msg = err.message || '';
      if (msg.includes('already exists') || msg.includes('duplicate')) {
        setIsInWatchlist(true);
        toast.info(`${symbol} is already in your watchlist`);
      } else {
        toast.error(msg || `Failed to add ${symbol}`);
      }
    } finally {
      setAdding(false);
    }
  };

  // Don't render for logged-out users until they interact
  if (checking) {
    return (
      <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium bg-ic-bg-tertiary text-ic-text-dim border border-ic-border">
        <BookmarkIcon className="w-4 h-4" />
        Watchlist
      </div>
    );
  }

  return (
    <button
      onClick={handleClick}
      disabled={adding}
      title={
        isInWatchlist
          ? `View watchlist${watchlist ? ` — ${watchlist.name}` : ''}`
          : 'Add to watchlist'
      }
      className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium transition-colors
        ${
          isInWatchlist
            ? 'bg-ic-positive/10 text-ic-positive border border-ic-positive/30 hover:bg-ic-positive/20'
            : 'bg-ic-blue/10 text-ic-blue border border-ic-blue/30 hover:bg-ic-blue/20'
        }
        ${adding ? 'opacity-50 cursor-wait' : ''}`}
    >
      {adding ? (
        'Adding...'
      ) : isInWatchlist ? (
        <>
          <BookmarkSolidIcon className="w-4 h-4" />
          Watchlisted
        </>
      ) : (
        <>
          <BookmarkIcon className="w-4 h-4" />
          Watchlist
        </>
      )}
    </button>
  );
}
