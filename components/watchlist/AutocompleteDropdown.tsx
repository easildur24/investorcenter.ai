'use client';

import React from 'react';

export interface SearchResultItem {
  symbol: string;
  name: string;
  type: string;
  exchange: string;
  logo_url?: string;
}

interface AutocompleteDropdownProps {
  results: SearchResultItem[];
  existingSymbols: Set<string>;
  highlightedIndex: number;
  isLoading: boolean;
  query: string;
  searchError: boolean;
  onSelect: (result: SearchResultItem) => void;
  onHighlightChange: (index: number) => void;
  onRetry: () => void;
}

/** Returns a deterministic background color based on the first char of a string. */
function initialColor(str: string): string {
  const colors = [
    'bg-blue-500',
    'bg-green-500',
    'bg-purple-500',
    'bg-orange-500',
    'bg-pink-500',
    'bg-teal-500',
    'bg-indigo-500',
    'bg-red-500',
  ];
  const code = str.charCodeAt(0) || 0;
  return colors[code % colors.length];
}

function TickerLogo({ result }: { result: SearchResultItem }) {
  const [imgError, setImgError] = React.useState(false);

  if (result.logo_url && !imgError) {
    return (
      <img
        src={result.logo_url}
        alt=""
        width={24}
        height={24}
        className="w-6 h-6 rounded object-contain bg-white"
        onError={() => setImgError(true)}
      />
    );
  }

  // Fallback: first letter in a colored circle
  const letter = result.symbol.replace('X:', '').charAt(0).toUpperCase();
  return (
    <span
      className={`flex items-center justify-center w-6 h-6 rounded text-xs font-bold text-white ${initialColor(result.symbol)}`}
    >
      {letter}
    </span>
  );
}

export default function AutocompleteDropdown({
  results,
  existingSymbols,
  highlightedIndex,
  isLoading,
  query,
  searchError,
  onSelect,
  onHighlightChange,
  onRetry,
}: AutocompleteDropdownProps) {
  // Partition results: selectable first, already-added last
  const available = results.filter((r) => !existingSymbols.has(r.symbol));
  const alreadyAdded = results.filter((r) => existingSymbols.has(r.symbol));

  if (isLoading) {
    return (
      <div
        className="absolute z-50 mt-1 w-full bg-ic-bg-primary border border-ic-border
          rounded-lg shadow-xl shadow-black/20 py-2"
      >
        <div className="px-4 py-2 text-sm text-ic-text-muted">Searching...</div>
      </div>
    );
  }

  if (searchError) {
    return (
      <div
        className="absolute z-50 mt-1 w-full bg-ic-bg-primary border border-ic-border
          rounded-lg shadow-xl shadow-black/20 py-2"
      >
        <div className="px-4 py-2 text-sm text-ic-text-muted">
          Search unavailable. Check your connection and{' '}
          <button type="button" onClick={onRetry} className="text-ic-blue hover:underline">
            try again
          </button>
          .
        </div>
      </div>
    );
  }

  if (results.length === 0 && query.trim().length > 0) {
    return (
      <div
        className="absolute z-50 mt-1 w-full bg-ic-bg-primary border border-ic-border
          rounded-lg shadow-xl shadow-black/20 py-2"
      >
        <div className="px-4 py-2 text-sm text-ic-text-muted">
          No matching tickers found. Try a different name or symbol.
        </div>
      </div>
    );
  }

  if (results.length === 0) return null;

  return (
    <ul
      role="listbox"
      id="watchlist-search-listbox"
      className="absolute z-50 mt-1 w-full bg-ic-bg-primary border border-ic-border
        rounded-lg shadow-xl shadow-black/20 py-1 max-h-[400px] overflow-y-auto"
    >
      {available.map((result, index) => (
        <li
          key={result.symbol}
          role="option"
          id={`search-option-${index}`}
          aria-selected={index === highlightedIndex}
          className={`flex items-center gap-3 px-4 py-2.5 cursor-pointer transition-colors
            ${index === highlightedIndex ? 'bg-ic-surface' : 'hover:bg-ic-surface/50'}`}
          onClick={() => onSelect(result)}
          onMouseEnter={() => onHighlightChange(index)}
        >
          <TickerLogo result={result} />
          <div className="flex-1 min-w-0">
            <span className="font-semibold text-sm text-ic-text-primary">{result.symbol}</span>
            <span className="ml-2 text-sm text-ic-text-secondary truncate">{result.name}</span>
          </div>
          <span className="text-xs text-ic-text-dim bg-ic-bg-tertiary px-2 py-0.5 rounded flex-shrink-0">
            {result.exchange}
          </span>
        </li>
      ))}

      {alreadyAdded.length > 0 && (
        <>
          <li className="px-4 py-1.5 text-xs text-ic-text-dim uppercase tracking-wide border-t border-ic-border mt-1 pt-2">
            Already in watchlist
          </li>
          {alreadyAdded.map((result) => (
            <li
              key={result.symbol}
              role="option"
              aria-selected={false}
              aria-disabled="true"
              className="flex items-center gap-3 px-4 py-2.5 opacity-50 cursor-default"
            >
              <TickerLogo result={result} />
              <div className="flex-1 min-w-0">
                <span className="font-semibold text-sm text-ic-text-primary">{result.symbol}</span>
                <span className="ml-2 text-sm text-ic-text-secondary truncate">{result.name}</span>
              </div>
              <span className="text-xs text-ic-positive flex-shrink-0">&#10003;</span>
            </li>
          ))}
        </>
      )}
    </ul>
  );
}
