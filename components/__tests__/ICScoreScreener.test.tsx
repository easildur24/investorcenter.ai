/**
 * Tests for ICScoreScreener component.
 *
 * Verifies loading states, data rendering, error handling,
 * and filter UI presence.
 */

import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';

// Mock the icScoreApi
const mockRunScreener = jest.fn();
jest.mock('@/lib/api', () => ({
  icScoreApi: {
    runScreener: (...args: unknown[]) => mockRunScreener(...args),
  },
}));

// Mock IC Score type helpers
jest.mock('@/lib/types/ic-score', () => ({
  getICScoreRatingColor: (rating: string) => {
    if (rating === 'Strong Buy') return 'text-green-600 bg-green-50 border-green-200';
    if (rating === 'Buy') return 'text-green-500 bg-green-50 border-green-200';
    return 'text-yellow-600 bg-yellow-50 border-yellow-200';
  },
  getICScoreColor: (score: number) => {
    if (score >= 80) return '#10b981';
    if (score >= 65) return '#84cc16';
    return '#eab308';
  },
}));

import ICScoreScreener from '../ic-score/ICScoreScreener';

const makeStock = (overrides = {}) => ({
  ticker: 'AAPL',
  companyName: 'Apple Inc.',
  score: 78,
  rating: 'Buy' as const,
  sector: 'Technology',
  marketCap: 2850000000000,
  price: 185.5,
  change: 2.3,
  changePercent: 1.25,
  volume: 50000000,
  calculatedAt: '2026-02-17T00:00:00Z',
  ...overrides,
});

describe('ICScoreScreener', () => {
  beforeEach(() => {
    mockRunScreener.mockReset();
  });

  it('shows loading state initially', () => {
    mockRunScreener.mockReturnValue(new Promise(() => {})); // Never resolves

    render(<ICScoreScreener />);

    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('renders screener header', async () => {
    mockRunScreener.mockResolvedValue({
      stocks: [makeStock()],
      total: 1,
      filters: {},
      timestamp: '2026-02-17T00:00:00Z',
    });

    render(<ICScoreScreener />);

    expect(screen.getByText('IC Score Stock Screener')).toBeInTheDocument();
  });

  it('renders stock data after loading', async () => {
    mockRunScreener.mockResolvedValue({
      stocks: [
        makeStock({ ticker: 'AAPL', companyName: 'Apple Inc.', score: 78 }),
        makeStock({ ticker: 'MSFT', companyName: 'Microsoft', score: 82 }),
      ],
      total: 2,
      filters: {},
      timestamp: '2026-02-17T00:00:00Z',
    });

    render(<ICScoreScreener />);

    await waitFor(() => {
      expect(screen.getByText('AAPL')).toBeInTheDocument();
    });

    expect(screen.getByText('MSFT')).toBeInTheDocument();
  });

  it('shows error state when API fails', async () => {
    mockRunScreener.mockRejectedValue(new Error('Network error'));

    render(<ICScoreScreener />);

    await waitFor(() => {
      expect(screen.getByText('Network error')).toBeInTheDocument();
    });
  });

  it('handles empty results', async () => {
    mockRunScreener.mockResolvedValue({
      stocks: [],
      total: 0,
      filters: {},
      timestamp: '2026-02-17T00:00:00Z',
    });

    render(<ICScoreScreener />);

    await waitFor(() => {
      expect(screen.getByText(/Showing 1-0 of 0 stocks/)).toBeInTheDocument();
    });
  });

  it('renders filter description text', async () => {
    mockRunScreener.mockResolvedValue({
      stocks: [makeStock()],
      total: 1,
      filters: {},
      timestamp: '2026-02-17T00:00:00Z',
    });

    render(<ICScoreScreener />);

    expect(screen.getByText(/Filter and discover stocks/)).toBeInTheDocument();
  });
});
