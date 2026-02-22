'use client';

import React, { useState, useRef, useCallback, useEffect } from 'react';
import { MagnifyingGlassIcon, CheckIcon } from '@heroicons/react/24/outline';
import { apiClient } from '@/lib/api';
import { useSlashFocus } from '@/lib/hooks/useSlashFocus';
import AutocompleteDropdown, {
  type SearchResultItem,
} from '@/components/watchlist/AutocompleteDropdown';

interface WatchlistSearchInputProps {
  /** Called when a ticker is selected. Should handle API call + optimistic update. */
  onAdd: (symbol: string) => Promise<void>;
  /** Symbols already in the watchlist (used to mark results as "already added"). */
  existingSymbols: Set<string>;
  /** Current number of items in the watchlist. */
  itemCount?: number;
  /** Maximum items allowed in the watchlist. */
  maxItems?: number;
  /** Additional CSS classes for the container. */
  className?: string;
}

const DEBOUNCE_MS = 200;

export default function WatchlistSearchInput({
  onAdd,
  existingSymbols,
  itemCount = 0,
  maxItems = 10,
  className = '',
}: WatchlistSearchInputProps) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchResultItem[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isOpen, setIsOpen] = useState(false);
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const [justAdded, setJustAdded] = useState(false);
  const [searchError, setSearchError] = useState(false);

  const inputRef = useRef<HTMLInputElement>(null);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout>>();
  const containerRef = useRef<HTMLDivElement>(null);

  const isAtLimit = itemCount >= maxItems;

  // "/" shortcut to focus the search bar
  useSlashFocus(inputRef);

  // Cleanup debounce timer on unmount
  useEffect(() => {
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, []);

  const performSearch = useCallback(async (searchQuery: string) => {
    if (searchQuery.trim().length === 0) {
      setResults([]);
      setIsOpen(false);
      return;
    }

    setIsLoading(true);
    setSearchError(false);
    try {
      const response = await apiClient.searchSecurities(searchQuery.trim());
      setResults(response.data ?? []);
      setIsOpen(true);
    } catch {
      setResults([]);
      setSearchError(true);
      setIsOpen(true);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const handleInputChange = (value: string) => {
    setQuery(value);
    setHighlightedIndex(-1);

    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    if (value.trim().length === 0) {
      setResults([]);
      setIsOpen(false);
      return;
    }

    debounceTimerRef.current = setTimeout(() => {
      performSearch(value);
    }, DEBOUNCE_MS);
  };

  const handleRetry = () => {
    if (query.trim().length > 0) {
      performSearch(query);
    }
  };

  const addTicker = async (result: SearchResultItem) => {
    // Quick flash feedback
    setJustAdded(true);
    setTimeout(() => setJustAdded(false), 400);

    // Clear input and close dropdown, keep focus for batch adding
    setQuery('');
    setResults([]);
    setIsOpen(false);
    setHighlightedIndex(-1);

    await onAdd(result.symbol);

    // Re-focus input for rapid sequential adds
    inputRef.current?.focus();
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!isOpen || (results.length === 0 && !searchError)) return;

    const selectableResults = results.filter((r) => !existingSymbols.has(r.symbol));

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        if (selectableResults.length === 0) return;
        setHighlightedIndex((prev) => (prev < selectableResults.length - 1 ? prev + 1 : 0));
        break;
      case 'ArrowUp':
        e.preventDefault();
        if (selectableResults.length === 0) return;
        setHighlightedIndex((prev) => (prev > 0 ? prev - 1 : selectableResults.length - 1));
        break;
      case 'Enter':
        e.preventDefault();
        if (highlightedIndex >= 0 && selectableResults[highlightedIndex]) {
          addTicker(selectableResults[highlightedIndex]);
        } else if (selectableResults.length > 0) {
          addTicker(selectableResults[0]);
        }
        break;
      case 'Escape':
        setIsOpen(false);
        setHighlightedIndex(-1);
        inputRef.current?.blur();
        break;
    }
  };

  return (
    <div ref={containerRef} className={`relative ${className}`}>
      <div className="relative">
        {/* Search icon or success check */}
        <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
          {justAdded ? (
            <CheckIcon className="h-5 w-5 text-ic-positive" />
          ) : (
            <MagnifyingGlassIcon className="h-5 w-5 text-ic-text-dim" />
          )}
        </div>

        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={(e) => handleInputChange(e.target.value)}
          onFocus={() => {
            if (query.trim().length > 0 && results.length > 0) {
              setIsOpen(true);
            }
          }}
          onBlur={() => {
            // Dropdown items use onMouseDown + e.preventDefault() to keep focus,
            // so closing here is safe — no timeout needed.
            setIsOpen(false);
          }}
          onKeyDown={handleKeyDown}
          disabled={isAtLimit}
          placeholder={
            isAtLimit ? `Watchlist full (${itemCount}/${maxItems} tickers)` : 'Add a ticker...'
          }
          autoComplete="off"
          autoCorrect="off"
          spellCheck="false"
          role="combobox"
          aria-expanded={isOpen}
          aria-haspopup="listbox"
          aria-controls="watchlist-search-listbox"
          aria-activedescendant={
            highlightedIndex >= 0 ? `search-option-${highlightedIndex}` : undefined
          }
          aria-label="Search for a ticker to add to your watchlist"
          className={`block w-full pl-10 pr-10 py-2 text-sm border rounded-lg leading-5
            bg-ic-input-bg text-ic-text-primary placeholder-ic-text-dim
            focus:outline-none focus:ring-2 focus:ring-ic-blue focus:border-ic-blue
            transition-all
            ${isAtLimit ? 'opacity-50 cursor-not-allowed' : ''}
            ${justAdded ? 'border-ic-positive' : 'border-ic-input-border'}`}
        />

        {/* Keyboard shortcut badge (desktop only) */}
        {!isAtLimit && !query && (
          <div className="absolute inset-y-0 right-0 pr-3 hidden sm:flex items-center pointer-events-none">
            <kbd className="text-xs text-ic-text-dim border border-ic-border rounded px-1.5 py-0.5 font-mono bg-ic-bg-tertiary">
              /
            </kbd>
          </div>
        )}
      </div>

      {/* Live region for screen readers */}
      <div aria-live="polite" aria-atomic="true" className="sr-only">
        {results.length > 0
          ? `${results.length} results found. Use arrow keys to navigate.`
          : query.trim().length > 0 && !isLoading
            ? 'No results found.'
            : ''}
      </div>

      {/* Autocomplete dropdown */}
      {isOpen && (
        <AutocompleteDropdown
          results={results}
          existingSymbols={existingSymbols}
          highlightedIndex={highlightedIndex}
          isLoading={isLoading}
          query={query}
          searchError={searchError}
          onSelect={addTicker}
          onHighlightChange={setHighlightedIndex}
          onRetry={handleRetry}
        />
      )}
    </div>
  );
}
