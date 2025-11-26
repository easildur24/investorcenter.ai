'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { MagnifyingGlassIcon } from '@heroicons/react/24/outline';
import { apiClient } from '@/lib/api';

interface SearchResult {
  symbol: string;
  name: string;
  type: string;
  exchange: string;
}

export default function TickerSearch() {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchResult[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [showResults, setShowResults] = useState(false);
  const router = useRouter();

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

  const handleSelectTicker = (symbol: string) => {
    setQuery('');
    setShowResults(false);
    router.push(`/ticker/${symbol}`);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (query.trim()) {
      router.push(`/ticker/${query.toUpperCase()}`);
    }
  };

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
            onFocus={() => query && setShowResults(true)}
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
        <div className="absolute z-50 mt-1 w-full bg-white shadow-xl max-h-60 rounded-lg py-1 text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none border border-gray-200">
          {isLoading ? (
            <div className="px-4 py-2 text-gray-500">Searching...</div>
          ) : results.length > 0 ? (
            results.map((result) => (
              <button
                key={result.symbol}
                onClick={() => handleSelectTicker(result.symbol)}
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
              No results found. Press Enter to view "{query.toUpperCase()}"
            </div>
          ) : !query && showResults ? (
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
