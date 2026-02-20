/**
 * Tests for WatchListTable component.
 *
 * Covers rendering, price formatting, alert conditions, user interactions,
 * sorting (3-state cycle, nulls-last), filtering (search + asset type),
 * view switching, and localStorage persistence.
 */

import React from 'react';
import { render, screen, fireEvent, within } from '@testing-library/react';

// Mock next/link
jest.mock('next/link', () => {
  return function MockLink({
    href,
    children,
    ...props
  }: {
    href: string;
    children: React.ReactNode;
    className?: string;
  }) {
    return (
      <a href={href} {...props}>
        {children}
      </a>
    );
  };
});

import WatchListTable from '../watchlist/WatchListTable';
import type { WatchListItem } from '@/lib/api/watchlist';

// ---------------------------------------------------------------------------
// Mock factory
// ---------------------------------------------------------------------------

const makeItem = (overrides: Partial<WatchListItem> = {}): WatchListItem => ({
  id: 'item-1',
  watch_list_id: 'wl-1',
  symbol: 'AAPL',
  name: 'Apple Inc.',
  exchange: 'NASDAQ',
  asset_type: 'stock',
  tags: [],
  added_at: '2026-01-01T00:00:00Z',
  display_order: 0,
  current_price: 185.5,
  price_change: 2.3,
  price_change_pct: 1.25,
  target_buy_price: undefined,
  target_sell_price: undefined,
  alert_count: 0,
  // Screener fields default to null
  ic_score: null,
  ic_rating: null,
  value_score: null,
  growth_score: null,
  profitability_score: null,
  financial_health_score: null,
  momentum_score: null,
  analyst_consensus_score: null,
  insider_activity_score: null,
  institutional_score: null,
  news_sentiment_score: null,
  technical_score: null,
  sector_percentile: null,
  lifecycle_stage: null,
  pe_ratio: null,
  pb_ratio: null,
  ps_ratio: null,
  roe: null,
  roa: null,
  gross_margin: null,
  operating_margin: null,
  net_margin: null,
  debt_to_equity: null,
  current_ratio: null,
  revenue_growth: null,
  eps_growth: null,
  dividend_yield: null,
  payout_ratio: null,
  ...overrides,
});

// ---------------------------------------------------------------------------
// localStorage mock
// ---------------------------------------------------------------------------

const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: jest.fn((key: string) => store[key] ?? null),
    setItem: jest.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: jest.fn((key: string) => {
      delete store[key];
    }),
    clear: jest.fn(() => {
      store = {};
    }),
  };
})();

Object.defineProperty(window, 'localStorage', { value: localStorageMock });

// ===========================================================================
// Original tests — basic rendering & interactions
// ===========================================================================

describe('WatchListTable', () => {
  const onRemove = jest.fn();
  const onEdit = jest.fn();

  beforeEach(() => {
    onRemove.mockClear();
    onEdit.mockClear();
    localStorageMock.clear();
  });

  it('renders all watchlist items with symbols and names', () => {
    const items = [
      makeItem({ symbol: 'AAPL', name: 'Apple Inc.' }),
      makeItem({ symbol: 'MSFT', name: 'Microsoft Corp.' }),
    ];

    render(<WatchListTable items={items} onRemove={onRemove} onEdit={onEdit} />);

    expect(screen.getByText('AAPL')).toBeInTheDocument();
    expect(screen.getByText('Apple Inc.')).toBeInTheDocument();
    expect(screen.getByText('MSFT')).toBeInTheDocument();
    expect(screen.getByText('Microsoft Corp.')).toBeInTheDocument();
  });

  it('formats prices as $XX.XX', () => {
    render(
      <WatchListTable
        items={[makeItem({ current_price: 185.5 })]}
        onRemove={onRemove}
        onEdit={onEdit}
      />
    );

    expect(screen.getByText('$185.50')).toBeInTheDocument();
  });

  it('shows dash for undefined price', () => {
    render(
      <WatchListTable
        items={[makeItem({ current_price: undefined })]}
        onRemove={onRemove}
        onEdit={onEdit}
      />
    );

    // The price cell should show a dash (em-dash)
    const dashes = screen.getAllByText('—');
    expect(dashes.length).toBeGreaterThan(0);
  });

  it('shows buy alert when price <= target buy price', () => {
    render(
      <WatchListTable
        items={[
          makeItem({
            current_price: 150,
            target_buy_price: 155,
          }),
        ]}
        onRemove={onRemove}
        onEdit={onEdit}
      />
    );

    expect(screen.getByText('At buy target')).toBeInTheDocument();
  });

  it('shows sell alert when price >= target sell price', () => {
    render(
      <WatchListTable
        items={[
          makeItem({
            current_price: 200,
            target_sell_price: 195,
          }),
        ]}
        onRemove={onRemove}
        onEdit={onEdit}
      />
    );

    expect(screen.getByText('At sell target')).toBeInTheDocument();
  });

  it('shows no alert when price is between targets', () => {
    render(
      <WatchListTable
        items={[
          makeItem({
            current_price: 185,
            target_buy_price: 150,
            target_sell_price: 200,
          }),
        ]}
        onRemove={onRemove}
        onEdit={onEdit}
      />
    );

    expect(screen.queryByText('At buy target')).not.toBeInTheDocument();
    expect(screen.queryByText('At sell target')).not.toBeInTheDocument();
  });

  it('calls onRemove when Remove button clicked', () => {
    render(
      <WatchListTable items={[makeItem({ symbol: 'AAPL' })]} onRemove={onRemove} onEdit={onEdit} />
    );

    fireEvent.click(screen.getByText('Remove'));

    expect(onRemove).toHaveBeenCalledWith('AAPL');
  });

  it('calls onEdit when Edit button clicked', () => {
    render(
      <WatchListTable items={[makeItem({ symbol: 'AAPL' })]} onRemove={onRemove} onEdit={onEdit} />
    );

    fireEvent.click(screen.getByText('Edit'));

    expect(onEdit).toHaveBeenCalledWith('AAPL');
  });
});

// ===========================================================================
// Sorting tests
// ===========================================================================

describe('WatchListTable — Sorting', () => {
  const onRemove = jest.fn();
  const onEdit = jest.fn();

  beforeEach(() => {
    localStorageMock.clear();
  });

  const threeItems = [
    makeItem({ symbol: 'MSFT', name: 'Microsoft Corp.', current_price: 400 }),
    makeItem({ symbol: 'AAPL', name: 'Apple Inc.', current_price: 185.5 }),
    makeItem({ symbol: 'GOOG', name: 'Alphabet Inc.', current_price: 150 }),
  ];

  const getSymbolsInOrder = (): string[] => {
    // Get all links in the table body (symbols are rendered as links)
    const rows = screen.getAllByRole('row');
    // Skip header row
    return rows.slice(1).map((row) => {
      const link = within(row).queryByRole('link');
      return link?.textContent ?? '';
    });
  };

  it('sorts by Price ascending on first click', () => {
    render(<WatchListTable items={threeItems} onRemove={onRemove} onEdit={onEdit} />);

    fireEvent.click(screen.getByText('Price'));

    const symbols = getSymbolsInOrder();
    // 150 < 185.5 < 400
    expect(symbols).toEqual(['GOOG', 'AAPL', 'MSFT']);
  });

  it('sorts by Price descending on second click', () => {
    render(<WatchListTable items={threeItems} onRemove={onRemove} onEdit={onEdit} />);

    fireEvent.click(screen.getByText('Price'));
    fireEvent.click(screen.getByText('Price'));

    const symbols = getSymbolsInOrder();
    // 400 > 185.5 > 150
    expect(symbols).toEqual(['MSFT', 'AAPL', 'GOOG']);
  });

  it('returns to original order on third click (unsorted)', () => {
    render(<WatchListTable items={threeItems} onRemove={onRemove} onEdit={onEdit} />);

    fireEvent.click(screen.getByText('Price'));
    fireEvent.click(screen.getByText('Price'));
    fireEvent.click(screen.getByText('Price'));

    const symbols = getSymbolsInOrder();
    // Original order: MSFT, AAPL, GOOG
    expect(symbols).toEqual(['MSFT', 'AAPL', 'GOOG']);
  });

  it('sorts nulls last regardless of direction (ascending)', () => {
    const items = [
      makeItem({ symbol: 'MSFT', current_price: 400, ic_score: null }),
      makeItem({ symbol: 'AAPL', current_price: 185.5, ic_score: 78.5 }),
      makeItem({ symbol: 'GOOG', current_price: 150, ic_score: 55 }),
    ];

    render(<WatchListTable items={items} onRemove={onRemove} onEdit={onEdit} />);

    // Click the IC Score column header (not the tab — use columnheader role)
    const header = screen.getByRole('columnheader', { name: /IC Score/ });
    fireEvent.click(header);

    const symbols = getSymbolsInOrder();
    // 55 < 78.5 then null last
    expect(symbols).toEqual(['GOOG', 'AAPL', 'MSFT']);
  });

  it('sorts nulls last regardless of direction (descending)', () => {
    const items = [
      makeItem({ symbol: 'MSFT', current_price: 400, ic_score: null }),
      makeItem({ symbol: 'AAPL', current_price: 185.5, ic_score: 78.5 }),
      makeItem({ symbol: 'GOOG', current_price: 150, ic_score: 55 }),
    ];

    render(<WatchListTable items={items} onRemove={onRemove} onEdit={onEdit} />);

    // Click the IC Score column header twice for descending
    const header = screen.getByRole('columnheader', { name: /IC Score/ });
    fireEvent.click(header);
    fireEvent.click(header);

    const symbols = getSymbolsInOrder();
    // 78.5 > 55 then null last
    expect(symbols).toEqual(['AAPL', 'GOOG', 'MSFT']);
  });

  it('sorts alphabetically by Symbol', () => {
    render(<WatchListTable items={threeItems} onRemove={onRemove} onEdit={onEdit} />);

    fireEvent.click(screen.getByText('Symbol'));

    const symbols = getSymbolsInOrder();
    expect(symbols).toEqual(['AAPL', 'GOOG', 'MSFT']);
  });

  it('shows sort indicator arrow when sorted', () => {
    render(<WatchListTable items={threeItems} onRemove={onRemove} onEdit={onEdit} />);

    fireEvent.click(screen.getByText('Price'));
    expect(screen.getByText('↑')).toBeInTheDocument();

    fireEvent.click(screen.getByText('Price'));
    expect(screen.getByText('↓')).toBeInTheDocument();
  });

  it('switching sort columns resets to ascending', () => {
    render(<WatchListTable items={threeItems} onRemove={onRemove} onEdit={onEdit} />);

    // Sort by Price descending
    fireEvent.click(screen.getByText('Price'));
    fireEvent.click(screen.getByText('Price'));

    // Now click Symbol — should start ascending
    fireEvent.click(screen.getByText('Symbol'));

    const symbols = getSymbolsInOrder();
    expect(symbols).toEqual(['AAPL', 'GOOG', 'MSFT']);
    expect(screen.getByText('↑')).toBeInTheDocument();
  });
});

// ===========================================================================
// Filtering tests
// ===========================================================================

describe('WatchListTable — Filtering', () => {
  const onRemove = jest.fn();
  const onEdit = jest.fn();

  beforeEach(() => {
    localStorageMock.clear();
  });

  const mixedItems = [
    makeItem({ symbol: 'AAPL', name: 'Apple Inc.', asset_type: 'stock' }),
    makeItem({ symbol: 'SPY', name: 'SPDR S&P 500 ETF', asset_type: 'etf' }),
    makeItem({ symbol: 'MSFT', name: 'Microsoft Corp.', asset_type: 'stock' }),
  ];

  it('filters by search query matching symbol', () => {
    render(<WatchListTable items={mixedItems} onRemove={onRemove} onEdit={onEdit} />);

    const searchInput = screen.getByPlaceholderText('Search by symbol or name...');
    fireEvent.change(searchInput, { target: { value: 'AAPL' } });

    expect(screen.getByText('AAPL')).toBeInTheDocument();
    expect(screen.queryByText('MSFT')).not.toBeInTheDocument();
    expect(screen.queryByText('SPY')).not.toBeInTheDocument();
  });

  it('filters by search query matching name (case-insensitive)', () => {
    render(<WatchListTable items={mixedItems} onRemove={onRemove} onEdit={onEdit} />);

    const searchInput = screen.getByPlaceholderText('Search by symbol or name...');
    fireEvent.change(searchInput, { target: { value: 'microsoft' } });

    expect(screen.getByText('MSFT')).toBeInTheDocument();
    expect(screen.queryByText('AAPL')).not.toBeInTheDocument();
  });

  it('filters by asset type chip', () => {
    render(<WatchListTable items={mixedItems} onRemove={onRemove} onEdit={onEdit} />);

    fireEvent.click(screen.getByText('etf'));

    expect(screen.getByText('SPY')).toBeInTheDocument();
    expect(screen.queryByText('AAPL')).not.toBeInTheDocument();
    expect(screen.queryByText('MSFT')).not.toBeInTheDocument();
  });

  it('shows "All" filter chip active by default', () => {
    render(<WatchListTable items={mixedItems} onRemove={onRemove} onEdit={onEdit} />);

    const allButton = screen.getByText('All');
    expect(allButton).toHaveClass('bg-ic-blue');
  });

  it('toggles asset type filter off when clicking the same chip again', () => {
    render(<WatchListTable items={mixedItems} onRemove={onRemove} onEdit={onEdit} />);

    // Click etf to filter
    fireEvent.click(screen.getByText('etf'));
    expect(screen.queryByText('AAPL')).not.toBeInTheDocument();

    // Click etf again to clear filter
    fireEvent.click(screen.getByText('etf'));
    expect(screen.getByText('AAPL')).toBeInTheDocument();
    expect(screen.getByText('MSFT')).toBeInTheDocument();
    expect(screen.getByText('SPY')).toBeInTheDocument();
  });

  it('shows result count "X of Y"', () => {
    render(<WatchListTable items={mixedItems} onRemove={onRemove} onEdit={onEdit} />);

    const count = screen.getByTestId('result-count');
    expect(count).toHaveTextContent('3 of 3');

    // Filter to etf
    fireEvent.click(screen.getByText('etf'));
    expect(count).toHaveTextContent('1 of 3');
  });

  it('combines search and asset type filter', () => {
    render(<WatchListTable items={mixedItems} onRemove={onRemove} onEdit={onEdit} />);

    // Filter by stock type first
    fireEvent.click(screen.getByText('stock'));

    // Then search for Apple
    const searchInput = screen.getByPlaceholderText('Search by symbol or name...');
    fireEvent.change(searchInput, { target: { value: 'apple' } });

    expect(screen.getByText('AAPL')).toBeInTheDocument();
    expect(screen.queryByText('MSFT')).not.toBeInTheDocument();
    expect(screen.queryByText('SPY')).not.toBeInTheDocument();

    const count = screen.getByTestId('result-count');
    expect(count).toHaveTextContent('1 of 3');
  });
});

// ===========================================================================
// View switching tests
// ===========================================================================

describe('WatchListTable — View Switching', () => {
  const onRemove = jest.fn();
  const onEdit = jest.fn();

  beforeEach(() => {
    localStorageMock.clear();
  });

  const itemWithData = makeItem({
    symbol: 'AAPL',
    name: 'Apple Inc.',
    current_price: 185.5,
    pe_ratio: 32.5,
    pb_ratio: 18.2,
    roe: 35.2,
    roa: 18.7,
    ic_score: 78.5,
    ic_rating: 'Strong Buy',
    reddit_rank: 5,
    reddit_mentions: 120,
  });

  it('defaults to General view', () => {
    render(<WatchListTable items={[itemWithData]} onRemove={onRemove} onEdit={onEdit} />);

    // General view should show General tab as active
    const generalTab = screen.getByRole('tab', { name: /General/i });
    expect(generalTab).toHaveAttribute('aria-selected', 'true');
  });

  it('switches to Fundamentals view and shows P/E column', () => {
    render(<WatchListTable items={[itemWithData]} onRemove={onRemove} onEdit={onEdit} />);

    // Click Fundamentals tab
    fireEvent.click(screen.getByRole('tab', { name: /Fundamentals/i }));

    // P/E header should appear
    expect(screen.getByText('P/E')).toBeInTheDocument();
    expect(screen.getByText('P/B')).toBeInTheDocument();
    expect(screen.getByText('ROE')).toBeInTheDocument();
    expect(screen.getByText('ROA')).toBeInTheDocument();
  });

  it('switches to IC Score view and shows sub-factor columns', () => {
    render(<WatchListTable items={[itemWithData]} onRemove={onRemove} onEdit={onEdit} />);

    fireEvent.click(screen.getByRole('tab', { name: /IC Score/i }));

    // "IC Score" appears in both tab and column header — check column header exists
    expect(screen.getByRole('columnheader', { name: /IC Score/ })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: /IC Rating/ })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: /Value/ })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: /Growth/ })).toBeInTheDocument();
  });

  it('switches to Social view and shows Reddit columns', () => {
    render(<WatchListTable items={[itemWithData]} onRemove={onRemove} onEdit={onEdit} />);

    fireEvent.click(screen.getByRole('tab', { name: /Social/i }));

    expect(screen.getByText('Reddit Rank')).toBeInTheDocument();
    expect(screen.getByText('Mentions')).toBeInTheDocument();
  });

  it('persists view to localStorage', () => {
    render(<WatchListTable items={[itemWithData]} onRemove={onRemove} onEdit={onEdit} />);

    fireEvent.click(screen.getByRole('tab', { name: /Fundamentals/i }));

    expect(localStorageMock.setItem).toHaveBeenCalledWith('watchlist-view-preset', 'fundamentals');
  });

  it('restores view from localStorage on mount', () => {
    localStorageMock.getItem.mockReturnValueOnce('fundamentals');

    render(<WatchListTable items={[itemWithData]} onRemove={onRemove} onEdit={onEdit} />);

    // Fundamentals tab should be active
    const fundTab = screen.getByRole('tab', { name: /Fundamentals/i });
    expect(fundTab).toHaveAttribute('aria-selected', 'true');

    // P/E column should be visible
    expect(screen.getByText('P/E')).toBeInTheDocument();
  });

  it('Compact view has minimal columns', () => {
    render(<WatchListTable items={[itemWithData]} onRemove={onRemove} onEdit={onEdit} />);

    fireEvent.click(screen.getByRole('tab', { name: /Compact/i }));

    // Compact: symbol, price, change, ic_score, actions = 5 columns
    const headers = screen.getAllByRole('columnheader');
    expect(headers).toHaveLength(5);
  });
});

// ===========================================================================
// ViewSwitcher rendering tests
// ===========================================================================

describe('WatchListTable — ViewSwitcher rendering', () => {
  const onRemove = jest.fn();
  const onEdit = jest.fn();

  beforeEach(() => {
    localStorageMock.clear();
  });

  it('renders all 7 view preset tabs', () => {
    render(<WatchListTable items={[makeItem()]} onRemove={onRemove} onEdit={onEdit} />);

    const tabs = screen.getAllByRole('tab');
    expect(tabs).toHaveLength(7);
  });

  it('General tab is active (blue) by default', () => {
    render(<WatchListTable items={[makeItem()]} onRemove={onRemove} onEdit={onEdit} />);

    const generalTab = screen.getByRole('tab', { name: /General/i });
    expect(generalTab).toHaveClass('bg-ic-blue');
    expect(generalTab).toHaveClass('text-white');
  });

  it('inactive tabs have surface background', () => {
    render(<WatchListTable items={[makeItem()]} onRemove={onRemove} onEdit={onEdit} />);

    const compactTab = screen.getByRole('tab', { name: /Compact/i });
    expect(compactTab).toHaveClass('bg-ic-surface');
    expect(compactTab).not.toHaveClass('bg-ic-blue');
  });
});

// ===========================================================================
// Cell rendering tests
// ===========================================================================

describe('WatchListTable — Cell rendering', () => {
  const onRemove = jest.fn();
  const onEdit = jest.fn();

  beforeEach(() => {
    localStorageMock.clear();
  });

  it('renders IC Score as colored pill badge', () => {
    const item = makeItem({ ic_score: 78.5 });
    render(<WatchListTable items={[item]} onRemove={onRemove} onEdit={onEdit} />);

    // Score >= 70 gets green pill
    const pill = screen.getByText('78.5');
    expect(pill).toHaveClass('bg-green-100');
    expect(pill).toHaveClass('text-green-800');
  });

  it('renders low IC Score with red pill', () => {
    const item = makeItem({ ic_score: 25 });
    render(<WatchListTable items={[item]} onRemove={onRemove} onEdit={onEdit} />);

    const pill = screen.getByText('25.0');
    expect(pill).toHaveClass('bg-red-100');
    expect(pill).toHaveClass('text-red-800');
  });

  it('renders medium IC Score with yellow pill', () => {
    const item = makeItem({ ic_score: 55 });
    render(<WatchListTable items={[item]} onRemove={onRemove} onEdit={onEdit} />);

    const pill = screen.getByText('55.0');
    expect(pill).toHaveClass('bg-yellow-100');
    expect(pill).toHaveClass('text-yellow-800');
  });

  it('renders positive change in green', () => {
    const item = makeItem({ price_change: 2.3, price_change_pct: 1.25 });
    render(<WatchListTable items={[item]} onRemove={onRemove} onEdit={onEdit} />);

    const changeCell = screen.getByText('+2.30 (+1.25%)');
    expect(changeCell).toHaveClass('text-ic-positive');
  });

  it('renders negative change in red', () => {
    const item = makeItem({ price_change: -3.5, price_change_pct: -1.89 });
    render(<WatchListTable items={[item]} onRemove={onRemove} onEdit={onEdit} />);

    const changeCell = screen.getByText('-3.50 (-1.89%)');
    expect(changeCell).toHaveClass('text-ic-negative');
  });

  it('renders symbol as link to ticker page', () => {
    render(
      <WatchListTable items={[makeItem({ symbol: 'AAPL' })]} onRemove={onRemove} onEdit={onEdit} />
    );

    const link = screen.getByRole('link', { name: 'AAPL' });
    expect(link).toHaveAttribute('href', '/ticker/AAPL');
  });

  it('renders trend column with arrow and color', () => {
    // Switch to Social view to see trend column
    localStorageMock.getItem.mockReturnValueOnce('social');
    const item = makeItem({ reddit_trend: 'rising' });
    render(<WatchListTable items={[item]} onRemove={onRemove} onEdit={onEdit} />);

    const trendCell = screen.getByText(/↑ rising/);
    expect(trendCell).toHaveClass('text-ic-positive');
  });

  it('renders falling trend with down arrow', () => {
    localStorageMock.getItem.mockReturnValueOnce('social');
    const item = makeItem({ reddit_trend: 'falling' });
    render(<WatchListTable items={[item]} onRemove={onRemove} onEdit={onEdit} />);

    const trendCell = screen.getByText(/↓ falling/);
    expect(trendCell).toHaveClass('text-ic-negative');
  });

  it('buy alert highlights target buy price in bold green', () => {
    const item = makeItem({
      current_price: 150,
      target_buy_price: 155,
    });
    render(<WatchListTable items={[item]} onRemove={onRemove} onEdit={onEdit} />);

    // Target buy price should be bold green
    const buyPrice = screen.getByText('$155.00');
    expect(buyPrice).toHaveClass('font-bold');
    expect(buyPrice).toHaveClass('text-green-700');
  });

  it('sell alert highlights target sell price in bold blue', () => {
    const item = makeItem({
      current_price: 200,
      target_sell_price: 195,
    });
    render(<WatchListTable items={[item]} onRemove={onRemove} onEdit={onEdit} />);

    const sellPrice = screen.getByText('$195.00');
    expect(sellPrice).toHaveClass('font-bold');
    expect(sellPrice).toHaveClass('text-blue-700');
  });
});

// ===========================================================================
// Edge cases
// ===========================================================================

describe('WatchListTable — Edge cases', () => {
  const onRemove = jest.fn();
  const onEdit = jest.fn();

  beforeEach(() => {
    localStorageMock.clear();
  });

  it('renders empty table when no items', () => {
    render(<WatchListTable items={[]} onRemove={onRemove} onEdit={onEdit} />);

    // Should have headers but no data rows
    const rows = screen.getAllByRole('row');
    expect(rows).toHaveLength(1); // Only header row
  });

  it('handles items with all null scores gracefully', () => {
    const item = makeItem({
      ic_score: null,
      pe_ratio: null,
      volume: undefined,
    });

    // Should not throw
    render(<WatchListTable items={[item]} onRemove={onRemove} onEdit={onEdit} />);

    // Null IC Score shows dash
    const dashes = screen.getAllByText('—');
    expect(dashes.length).toBeGreaterThan(0);
  });

  it('search with no matches shows zero results', () => {
    render(
      <WatchListTable
        items={[makeItem({ symbol: 'AAPL', name: 'Apple Inc.' })]}
        onRemove={onRemove}
        onEdit={onEdit}
      />
    );

    const searchInput = screen.getByPlaceholderText('Search by symbol or name...');
    fireEvent.change(searchInput, { target: { value: 'ZZZZZ' } });

    const count = screen.getByTestId('result-count');
    expect(count).toHaveTextContent('0 of 1');

    // Only header row, no data rows
    const rows = screen.getAllByRole('row');
    expect(rows).toHaveLength(1);
  });
});
