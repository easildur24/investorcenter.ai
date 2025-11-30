'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { MagnifyingGlassIcon, ClockIcon, XMarkIcon } from '@heroicons/react/24/outline';
import { apiClient } from '@/lib/api';

interface SearchResult {
  symbol: string;
  name: string;
  type: string;
  exchange: string;
}

interface RecentSearch {
  symbol: string;
  name: string;
  timestamp: number;
}

const RECENT_SEARCHES_KEY = 'ticker_recent_searches';
const MAX_RECENT_SEARCHES = 5;

// Helper functions for localStorage
const getRecentSearches = (): RecentSearch[] => {
  if (typeof window === 'undefined') return [];
  try {
    const stored = localStorage.getItem(RECENT_SEARCHES_KEY);
    return stored ? JSON.parse(stored) : [];
  } catch {
    return [];
  }
};

const saveRecentSearch = (symbol: string, name: string) => {
  if (typeof window === 'undefined') return;
  try {
    const searches = getRecentSearches();
    // Remove if already exists
    const filtered = searches.filter(s => s.symbol !== symbol);
    // Add to front
    const updated = [{ symbol, name, timestamp: Date.now() }, ...filtered].slice(0, MAX_RECENT_SEARCHES);
    localStorage.setItem(RECENT_SEARCHES_KEY, JSON.stringify(updated));
  } catch {
    // Ignore localStorage errors
  }
};

const clearRecentSearches = () => {
  if (typeof window === 'undefined') return;
  try {
    localStorage.removeItem(RECENT_SEARCHES_KEY);
  } catch {
    // Ignore
  }
};

export default function TickerSearch() {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchResult[]>([]);
  const [recentSearches, setRecentSearches] = useState<RecentSearch[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [showResults, setShowResults] = useState(false);
  const router = useRouter();

  // Load recent searches on mount
  useEffect(() => {
    setRecentSearches(getRecentSearches());
  }, []);

  useEffect(() => {
    const searchTickers = async () => {
      if (query.length < 1) {
        setResults([]);
        setShowResults(false);
        return;
      }

      setIsLoading(true);
      try {
        const response = await apiClient.searchSecurities(query);
        setResults(response.data);
        setShowResults(true);
      } catch (error) {
        console.error('Search failed:', error);
        setResults([]);
      } finally {
        setIsLoading(false);
      }
    };

    const debounceTimer = setTimeout(searchTickers, 300);
    return () => clearTimeout(debounceTimer);
  }, [query]);

  const handleSelectTicker = (symbol: string, name?: string) => {
    // Save to recent searches
    saveRecentSearch(symbol, name || symbol);
    setRecentSearches(getRecentSearches());

    setQuery('');
    setShowResults(false);
    router.push(`/ticker/${symbol}`);
  };

  const handleClearRecent = (e: React.MouseEvent) => {
    e.stopPropagation();
    clearRecentSearches();
    setRecentSearches([]);
  };

  const handleRemoveRecentItem = (e: React.MouseEvent, symbol: string) => {
    e.stopPropagation();
    const updated = recentSearches.filter(s => s.symbol !== symbol);
    localStorage.setItem(RECENT_SEARCHES_KEY, JSON.stringify(updated));
    setRecentSearches(updated);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (query.trim()) {
      handleSelectTicker(query.toUpperCase());
    }
  };

  const showRecentSearches = !query && recentSearches.length > 0;

  return (
    <div className="relative w-full max-w-md">
      <form onSubmit={handleSubmit}>
        <div className="relative">
          <div className="absolute inset-y-0 left-0 pl-2 sm:pl-3 flex items-center pointer-events-none">
            <MagnifyingGlassIcon className="h-4 w-4 sm:h-5 sm:w-5 text-gray-400" />
          </div>
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onFocus={() => setShowResults(true)}
            onBlur={() => setTimeout(() => setShowResults(false), 200)}
            placeholder="Search..."
            aria-label="Search stocks and crypto"
            autoComplete="off"
            autoCorrect="off"
            autoCapitalize="characters"
            spellCheck="false"
            enterKeyHint="search"
            className="block w-full pl-7 sm:pl-10 pr-2 sm:pr-3 py-1.5 sm:py-2 text-sm sm:text-base border border-gray-300 rounded-lg leading-5 bg-white text-gray-900 placeholder-gray-500 focus:outline-none focus:placeholder-gray-400 focus:ring-2 focus:ring-primary-500 focus:border-primary-500 transition-all"
          />
        </div>
      </form>

      {/* Search Results Dropdown */}
      {showResults && (
        <div className="absolute z-50 mt-1 w-full bg-white shadow-xl max-h-80 rounded-lg py-1 text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none border border-gray-200">
          {isLoading ? (
            <div className="px-4 py-2 text-gray-500">Searching...</div>
          ) : results.length > 0 ? (
            results.map((result) => (
              <button
                key={result.symbol}
                onClick={() => handleSelectTicker(result.symbol, result.name)}
                className="w-full text-left px-4 py-3 hover:bg-primary-50 focus:bg-primary-50 focus:outline-none transition-colors border-b border-gray-100 last:border-b-0"
              >
                <div className="flex justify-between items-center">
                  <div className="flex-1 min-w-0">
                    <div className="font-semibold text-gray-900 text-sm">{result.symbol}</div>
                    <div className="text-sm text-gray-600 truncate">{result.name}</div>
                  </div>
                  <div className="ml-2 text-xs text-gray-400 bg-gray-100 px-2 py-1 rounded">
                    {result.exchange}
                  </div>
                </div>
              </button>
            ))
          ) : query && !isLoading ? (
            <div className="px-4 py-3 text-gray-500">
              No results found. Press Enter to view &quot;{query.toUpperCase()}&quot;
            </div>
          ) : showRecentSearches ? (
            <div className="py-2">
              <div className="flex items-center justify-between px-4 py-1">
                <div className="flex items-center gap-1 text-xs text-gray-400">
                  <ClockIcon className="h-3 w-3" />
                  <span>Recent searches</span>
                </div>
                <button
                  onClick={handleClearRecent}
                  className="text-xs text-gray-400 hover:text-gray-600 transition-colors"
                >
                  Clear all
                </button>
              </div>
              {recentSearches.map((item) => (
                <button
                  key={item.symbol}
                  onClick={() => handleSelectTicker(item.symbol, item.name)}
                  className="w-full text-left px-4 py-2 hover:bg-primary-50 focus:bg-primary-50 focus:outline-none transition-colors group"
                >
                  <div className="flex justify-between items-center">
                    <div className="flex-1 min-w-0">
                      <div className="font-semibold text-gray-900 text-sm">{item.symbol}</div>
                      <div className="text-xs text-gray-500 truncate">{item.name}</div>
                    </div>
                    <button
                      onClick={(e) => handleRemoveRecentItem(e, item.symbol)}
                      className="ml-2 p-1 text-gray-300 hover:text-gray-500 opacity-0 group-hover:opacity-100 transition-opacity"
                      aria-label={`Remove ${item.symbol} from recent searches`}
                    >
                      <XMarkIcon className="h-4 w-4" />
                    </button>
                  </div>
                </button>
              ))}
            </div>
          ) : !query ? (
            <div className="px-4 py-3">
              <div className="text-xs text-gray-400 mb-2">Popular stocks:</div>
              <div className="flex flex-wrap gap-2">
                {['AAPL', 'GOOGL', 'MSFT', 'TSLA', 'AMZN'].map((symbol) => (
                  <button
                    key={symbol}
                    onClick={() => handleSelectTicker(symbol)}
                    className="text-xs bg-primary-100 text-primary-700 px-2 py-1 rounded hover:bg-primary-200 transition-colors"
                  >
                    {symbol}
                  </button>
                ))}
              </div>
            </div>
          ) : null}
        </div>
      )}
    </div>
  );
}
