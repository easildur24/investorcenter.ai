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
    <div className="relative max-w-md mx-auto">
      <form onSubmit={handleSubmit}>
        <div className="relative">
          <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
            <MagnifyingGlassIcon className="h-5 w-5 text-gray-400" />
          </div>
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onFocus={() => query && setShowResults(true)}
            onBlur={() => setTimeout(() => setShowResults(false), 200)}
            placeholder="Search stocks (e.g., AAPL, Apple)"
            className="block w-full pl-10 pr-3 py-2 border border-gray-300 rounded-md leading-5 bg-white placeholder-gray-500 focus:outline-none focus:placeholder-gray-400 focus:ring-1 focus:ring-primary-500 focus:border-primary-500"
          />
        </div>
      </form>

      {/* Search Results Dropdown */}
      {showResults && (
        <div className="absolute z-10 mt-1 w-full bg-white shadow-lg max-h-60 rounded-md py-1 text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none">
          {isLoading ? (
            <div className="px-4 py-2 text-gray-500">Searching...</div>
          ) : results.length > 0 ? (
            results.map((result) => (
              <button
                key={result.symbol}
                onClick={() => handleSelectTicker(result.symbol)}
                className="w-full text-left px-4 py-2 hover:bg-gray-100 focus:bg-gray-100 focus:outline-none"
              >
                <div className="flex justify-between items-center">
                  <div>
                    <div className="font-medium text-gray-900">{result.symbol}</div>
                    <div className="text-sm text-gray-500 truncate">{result.name}</div>
                  </div>
                  <div className="text-xs text-gray-400">{result.exchange}</div>
                </div>
              </button>
            ))
          ) : query && !isLoading ? (
            <div className="px-4 py-2 text-gray-500">
              No results found. Press Enter to view "{query.toUpperCase()}"
            </div>
          ) : null}
        </div>
      )}
    </div>
  );
}
