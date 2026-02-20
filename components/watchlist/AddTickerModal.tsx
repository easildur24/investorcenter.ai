'use client';

import { useState, useEffect } from 'react';
import { apiClient } from '@/lib/api';

interface SearchResult {
  symbol: string;
  name: string;
  type: string;
  exchange: string;
}

interface AddTickerModalProps {
  onClose: () => void;
  onAdd: (
    symbol: string,
    notes?: string,
    tags?: string[],
    targetBuy?: number,
    targetSell?: number
  ) => Promise<void>;
}

export default function AddTickerModal({ onClose, onAdd }: AddTickerModalProps) {
  const [symbol, setSymbol] = useState('');
  const [notes, setNotes] = useState('');
  const [tags, setTags] = useState('');
  const [targetBuy, setTargetBuy] = useState('');
  const [targetSell, setTargetSell] = useState('');
  const [loading, setLoading] = useState(false);
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [showResults, setShowResults] = useState(false);

  // Debounced search effect
  useEffect(() => {
    const searchTickers = async () => {
      if (symbol.length < 1) {
        setSearchResults([]);
        setShowResults(false);
        return;
      }

      setIsSearching(true);
      try {
        const response = await apiClient.searchSecurities(symbol);
        setSearchResults(response.data);
        setShowResults(true);
      } catch (error) {
        console.error('Search failed:', error);
        setSearchResults([]);
      } finally {
        setIsSearching(false);
      }
    };

    const debounceTimer = setTimeout(searchTickers, 300);
    return () => clearTimeout(debounceTimer);
  }, [symbol]);

  const handleSelectTicker = (selectedSymbol: string) => {
    setSymbol(selectedSymbol);
    setShowResults(false);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      const tagArray = tags
        .split(',')
        .map((t) => t.trim())
        .filter((t) => t);
      await onAdd(
        symbol.toUpperCase(),
        notes || undefined,
        tagArray.length > 0 ? tagArray : undefined,
        targetBuy ? parseFloat(targetBuy) : undefined,
        targetSell ? parseFloat(targetSell) : undefined
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-ic-bg-primary bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-ic-surface rounded-lg p-6 w-full max-w-md max-h-[90vh] overflow-y-auto">
        <h2 className="text-2xl font-bold mb-4 text-ic-text-primary">Add Ticker</h2>

        <form onSubmit={handleSubmit}>
          <div className="mb-4 relative">
            <label
              className="block text-sm font-medium mb-2 text-ic-text-secondary"
              htmlFor="symbol"
            >
              Symbol *
            </label>
            <input
              id="symbol"
              type="text"
              value={symbol}
              onChange={(e) => setSymbol(e.target.value)}
              onFocus={() => symbol && setShowResults(true)}
              onBlur={() => setTimeout(() => setShowResults(false), 200)}
              placeholder="e.g., AAPL, TSLA, X:BTCUSD"
              className="w-full px-3 py-2 border border-ic-border rounded focus:outline-none focus:ring-2 focus:ring-blue-500 text-ic-text-primary"
              required
              autoComplete="off"
            />
            <p className="text-xs text-ic-text-muted mt-1">
              Use X: prefix for crypto (e.g., X:BTCUSD, X:ETHUSD)
            </p>

            {/* Autocomplete Dropdown */}
            {showResults && (
              <div className="absolute z-50 mt-1 w-full bg-ic-surface border border-ic-border max-h-60 rounded-lg py-1 text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none">
                {isSearching ? (
                  <div className="px-4 py-2 text-ic-text-muted text-sm">Searching...</div>
                ) : searchResults.length > 0 ? (
                  searchResults.map((result) => (
                    <button
                      key={result.symbol}
                      type="button"
                      onClick={() => handleSelectTicker(result.symbol)}
                      className="w-full text-left px-4 py-3 hover:bg-blue-50 focus:bg-blue-50 focus:outline-none transition-colors border-b border-ic-border last:border-b-0"
                    >
                      <div className="flex justify-between items-center">
                        <div className="flex-1 min-w-0">
                          <div className="font-semibold text-ic-text-primary text-sm">
                            {result.symbol}
                          </div>
                          <div className="text-sm text-ic-text-muted truncate">{result.name}</div>
                        </div>
                        <div className="ml-2 text-xs text-ic-text-dim bg-ic-bg-secondary px-2 py-1 rounded">
                          {result.exchange}
                        </div>
                      </div>
                    </button>
                  ))
                ) : symbol && !isSearching ? (
                  <div className="px-4 py-3 text-ic-text-muted text-sm">
                    No results found for &quot;{symbol}&quot;
                  </div>
                ) : null}
              </div>
            )}
          </div>

          <div className="mb-4">
            <label
              className="block text-sm font-medium mb-2 text-ic-text-secondary"
              htmlFor="notes"
            >
              Notes (optional)
            </label>
            <textarea
              id="notes"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="Add personal notes about this ticker..."
              className="w-full px-3 py-2 border border-ic-border rounded focus:outline-none focus:ring-2 focus:ring-blue-500 text-ic-text-primary"
              rows={2}
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-2 text-ic-text-secondary" htmlFor="tags">
              Tags (optional)
            </label>
            <input
              id="tags"
              type="text"
              value={tags}
              onChange={(e) => setTags(e.target.value)}
              placeholder="e.g., tech, growth, dividend"
              className="w-full px-3 py-2 border border-ic-border rounded focus:outline-none focus:ring-2 focus:ring-blue-500 text-ic-text-primary"
            />
            <p className="text-xs text-ic-text-muted mt-1">Separate multiple tags with commas</p>
          </div>

          <div className="grid grid-cols-2 gap-4 mb-6">
            <div>
              <label
                className="block text-sm font-medium mb-2 text-ic-text-secondary"
                htmlFor="targetBuy"
              >
                Target Buy Price
              </label>
              <input
                id="targetBuy"
                type="number"
                step="0.01"
                value={targetBuy}
                onChange={(e) => setTargetBuy(e.target.value)}
                placeholder="0.00"
                className="w-full px-3 py-2 border border-ic-border rounded focus:outline-none focus:ring-2 focus:ring-blue-500 text-ic-text-primary"
              />
            </div>
            <div>
              <label
                className="block text-sm font-medium mb-2 text-ic-text-secondary"
                htmlFor="targetSell"
              >
                Target Sell Price
              </label>
              <input
                id="targetSell"
                type="number"
                step="0.01"
                value={targetSell}
                onChange={(e) => setTargetSell(e.target.value)}
                placeholder="0.00"
                className="w-full px-3 py-2 border border-ic-border rounded focus:outline-none focus:ring-2 focus:ring-blue-500 text-ic-text-primary"
              />
            </div>
          </div>

          <div className="mb-4 p-3 bg-blue-50 border border-blue-200 rounded text-sm text-blue-800">
            <strong>💡 Tip:</strong> Set target prices to get visual alerts when the price reaches
            your buy or sell targets.
          </div>

          <div className="flex gap-2 justify-end">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border border-ic-border text-ic-text-secondary rounded hover:bg-ic-surface-hover"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading || !symbol}
              className="px-4 py-2 bg-ic-blue text-ic-text-primary rounded hover:bg-ic-blue-hover disabled:bg-ic-bg-tertiary disabled:cursor-not-allowed"
            >
              {loading ? 'Adding...' : 'Add Ticker'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
