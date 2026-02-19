/**
 * Tests for WatchListTable component.
 *
 * Verifies rendering of watchlist items, price formatting,
 * alert conditions, and user interactions.
 */

import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';

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
  // Screener fields default to null (stable schema)
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

describe('WatchListTable', () => {
  const onRemove = jest.fn();
  const onEdit = jest.fn();

  beforeEach(() => {
    onRemove.mockClear();
    onEdit.mockClear();
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

    // The price cell should show a dash
    const dashes = screen.getAllByText('-');
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
