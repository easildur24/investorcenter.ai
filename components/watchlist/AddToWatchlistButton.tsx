'use client';

import { useState, useRef, useEffect } from 'react';
import { PlusIcon, CheckIcon, BookmarkIcon } from '@heroicons/react/24/outline';
import { BookmarkIcon as BookmarkSolidIcon } from '@heroicons/react/24/solid';
import { useAuth } from '@/lib/auth/AuthContext';
import { watchListAPI, WatchList } from '@/lib/api/watchlist';
import { useRouter } from 'next/navigation';

interface AddToWatchlistButtonProps {
  symbol: string;
}

export default function AddToWatchlistButton({ symbol }: AddToWatchlistButtonProps) {
  const { user } = useAuth();
  const router = useRouter();
  const [watchlists, setWatchlists] = useState<WatchList[]>([]);
  const [showPicker, setShowPicker] = useState(false);
  const [adding, setAdding] = useState<string | null>(null);
  const [addedTo, setAddedTo] = useState<Set<string>>(new Set());
  const [loaded, setLoaded] = useState(false);
  const pickerRef = useRef<HTMLDivElement>(null);

  // Close picker on outside click
  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (pickerRef.current && !pickerRef.current.contains(e.target as Node)) {
        setShowPicker(false);
      }
    }
    if (showPicker) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [showPicker]);

  const fetchWatchlists = async () => {
    if (loaded) return watchlists;
    try {
      const res = await watchListAPI.getWatchLists();
      const lists = res.watch_lists ?? [];
      setWatchlists(lists);
      setLoaded(true);
      return lists;
    } catch {
      return [];
    }
  };

  const handleClick = async () => {
    if (!user) {
      router.push('/auth/login');
      return;
    }

    const lists = await fetchWatchlists();

    if (lists.length === 0) {
      // No watchlists — create a default one and add
      try {
        setAdding('new');
        const newList = await watchListAPI.createWatchList({ name: 'My Watch List' });
        await watchListAPI.addTicker(newList.id, { symbol });
        setAddedTo(new Set([newList.id]));
        setWatchlists([newList]);
      } catch {
        // ignore
      } finally {
        setAdding(null);
      }
    } else if (lists.length === 1) {
      // One watchlist — add directly
      try {
        setAdding(lists[0].id);
        await watchListAPI.addTicker(lists[0].id, { symbol });
        setAddedTo(new Set([lists[0].id]));
      } catch {
        // ignore
      } finally {
        setAdding(null);
      }
    } else {
      // Multiple watchlists — show picker
      setShowPicker(true);
    }
  };

  const handleAddTo = async (listId: string) => {
    try {
      setAdding(listId);
      await watchListAPI.addTicker(listId, { symbol });
      setAddedTo((prev) => {
        const next = new Set(prev);
        next.add(listId);
        return next;
      });
    } catch {
      // ignore
    } finally {
      setAdding(null);
    }
  };

  const isAddedToAny = addedTo.size > 0;

  return (
    <div className="relative" ref={pickerRef}>
      <button
        onClick={handleClick}
        disabled={adding !== null}
        title={isAddedToAny ? 'Added to watchlist' : 'Add to watchlist'}
        className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium transition-colors
          ${
            isAddedToAny
              ? 'bg-ic-positive/10 text-ic-positive border border-ic-positive/30'
              : 'bg-ic-blue/10 text-ic-blue border border-ic-blue/30 hover:bg-ic-blue/20'
          }
          ${adding !== null ? 'opacity-50 cursor-wait' : ''}`}
      >
        {isAddedToAny ? (
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

      {showPicker && (
        <div className="absolute right-0 top-full mt-1 w-56 bg-ic-bg-primary rounded-lg shadow-xl border border-ic-border z-50 py-1">
          <div className="px-3 py-2 text-xs font-semibold text-ic-text-dim uppercase">
            Add to watch list
          </div>
          {watchlists.map((list) => {
            const isAdded = addedTo.has(list.id);
            const isAdding = adding === list.id;

            return (
              <button
                key={list.id}
                onClick={() => !isAdded && handleAddTo(list.id)}
                disabled={isAdded || isAdding}
                className={`w-full text-left px-3 py-2 text-sm flex items-center justify-between transition-colors
                  ${isAdded ? 'text-ic-positive' : 'text-ic-text-secondary hover:bg-ic-surface'}`}
              >
                <span className="truncate">{list.name}</span>
                {isAdded ? (
                  <CheckIcon className="w-4 h-4 flex-shrink-0" />
                ) : isAdding ? (
                  <span className="text-xs text-ic-text-dim">Adding...</span>
                ) : (
                  <PlusIcon className="w-4 h-4 flex-shrink-0 text-ic-text-dim" />
                )}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
