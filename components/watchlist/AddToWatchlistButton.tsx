'use client';

import { useState, useRef, useEffect } from 'react';
import { PlusIcon, CheckIcon, BookmarkIcon } from '@heroicons/react/24/outline';
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

  const fetchWatchlists = async (): Promise<WatchList[]> => {
    if (loaded) return watchlists;
    try {
      const res = await watchListAPI.getWatchLists();
      const lists = res.watch_lists ?? [];
      setWatchlists(lists);
      setLoaded(true);
      return lists;
    } catch (err: any) {
      toast.error(err.message || 'Failed to load watchlists');
      return [];
    }
  };

  const addToList = async (listId: string, listName: string) => {
    try {
      setAdding(listId);
      await watchListAPI.addTicker(listId, { symbol });
      setAddedTo((prev) => {
        const next = new Set(prev);
        next.add(listId);
        return next;
      });
      toast.success(`${symbol} added to ${listName}`);
    } catch (err: any) {
      const msg = err.message || '';
      if (msg.includes('already exists') || msg.includes('duplicate')) {
        toast.info(`${symbol} is already in ${listName}`);
        setAddedTo((prev) => {
          const next = new Set(prev);
          next.add(listId);
          return next;
        });
      } else {
        toast.error(msg || `Failed to add ${symbol}`);
      }
    } finally {
      setAdding(null);
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
        setWatchlists([newList]);
        await watchListAPI.addTicker(newList.id, { symbol });
        setAddedTo(new Set([newList.id]));
        toast.success(`${symbol} added to My Watch List`);
      } catch (err: any) {
        toast.error(err.message || 'Failed to create watchlist');
      } finally {
        setAdding(null);
      }
    } else if (lists.length === 1) {
      await addToList(lists[0].id, lists[0].name);
    } else {
      // Multiple watchlists — show picker
      setShowPicker(true);
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
        {adding !== null ? (
          'Adding...'
        ) : isAddedToAny ? (
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
        <div className="absolute left-0 top-full mt-1 w-56 bg-ic-bg-primary rounded-lg shadow-xl border border-ic-border z-50 py-1">
          <div className="px-3 py-2 text-xs font-semibold text-ic-text-dim uppercase">
            Add to watch list
          </div>
          {watchlists.map((list) => {
            const isAdded = addedTo.has(list.id);
            const isAdding = adding === list.id;

            return (
              <button
                key={list.id}
                onClick={() => !isAdded && addToList(list.id, list.name)}
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
