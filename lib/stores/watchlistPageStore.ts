import { create } from 'zustand';

/**
 * Cross-component store for watchlist page state.
 *
 * The Header (with TickerSearch) renders in app/layout.tsx, outside any page-level
 * React Context provider. This Zustand store bridges the watchlist detail page and
 * the global search bar so TickerSearch can show a contextual "+ Add" button when
 * the user is viewing a watchlist.
 */

interface WatchlistPageState {
  /** ID of the watchlist currently being viewed, or null if not on a watchlist page. */
  activeWatchlistId: string | null;
  /** Name of the active watchlist (for tooltip: "Add to My Watchlist"). */
  activeWatchlistName: string | null;
  /** Symbols already in the active watchlist (used to show "Added" state in search). */
  existingSymbols: Set<string>;
  /** Callback to add a ticker from the global search bar. Set by the watchlist page. */
  addTickerFn: ((symbol: string) => Promise<void>) | null;

  // Actions
  setActiveWatchlist: (
    id: string,
    name: string,
    symbols: string[],
    addFn: (symbol: string) => Promise<void>
  ) => void;
  clearActiveWatchlist: () => void;
  addSymbolToSet: (symbol: string) => void;
}

export const useWatchlistPageStore = create<WatchlistPageState>((set) => ({
  activeWatchlistId: null,
  activeWatchlistName: null,
  existingSymbols: new Set(),
  addTickerFn: null,

  setActiveWatchlist: (id, name, symbols, addFn) =>
    set({
      activeWatchlistId: id,
      activeWatchlistName: name,
      existingSymbols: new Set(symbols),
      addTickerFn: addFn,
    }),

  clearActiveWatchlist: () =>
    set({
      activeWatchlistId: null,
      activeWatchlistName: null,
      existingSymbols: new Set(),
      addTickerFn: null,
    }),

  addSymbolToSet: (symbol) =>
    set((state) => {
      const updated = new Set(Array.from(state.existingSymbols));
      updated.add(symbol);
      return { existingSymbols: updated };
    }),
}));
