/**
 * Tests for useScreener hook and buildScreenerUrl helper.
 *
 * We test URL construction, SWR fetch behavior, error handling,
 * and the shape of returned data.
 */

import { renderHook, waitFor } from '@testing-library/react';
import { buildScreenerUrl, useScreener } from '../useScreener';

const mockFetch = global.fetch as jest.Mock;

beforeEach(() => {
  mockFetch.mockReset();
});

// ---------------------------------------------------------------------------
// buildScreenerUrl
// ---------------------------------------------------------------------------

describe('buildScreenerUrl', () => {
  it('builds base URL with no params', () => {
    const url = buildScreenerUrl({});
    expect(url).toBe('/api/v1/screener/stocks');
  });

  it('includes page, limit, sort, order', () => {
    const url = buildScreenerUrl({ page: 2, limit: 25, sort: 'ic_score', order: 'desc' });
    expect(url).toContain('page=2');
    expect(url).toContain('limit=25');
    expect(url).toContain('sort=ic_score');
    expect(url).toContain('order=desc');
  });

  it('includes range filter params', () => {
    const url = buildScreenerUrl({ pe_min: 5, pe_max: 25, roe_min: 10 });
    expect(url).toContain('pe_min=5');
    expect(url).toContain('pe_max=25');
    expect(url).toContain('roe_min=10');
  });

  it('includes comma-separated sectors', () => {
    const url = buildScreenerUrl({ sectors: 'Technology,Healthcare' });
    expect(url).toContain('sectors=Technology%2CHealthcare');
  });

  it('omits null and undefined values', () => {
    const url = buildScreenerUrl({
      page: 1,
      pe_min: undefined,
      pe_max: undefined,
    });
    expect(url).not.toContain('pe_min');
    expect(url).not.toContain('pe_max');
    expect(url).toContain('page=1');
  });

  it('omits empty string values', () => {
    const url = buildScreenerUrl({ sectors: '' } as Record<string, unknown> as Parameters<typeof buildScreenerUrl>[0]);
    expect(url).not.toContain('sectors');
  });

  it('handles IC Score sub-factor filters', () => {
    const url = buildScreenerUrl({
      value_score_min: 60,
      momentum_score_max: 90,
      technical_score_min: 50,
    });
    expect(url).toContain('value_score_min=60');
    expect(url).toContain('momentum_score_max=90');
    expect(url).toContain('technical_score_min=50');
  });

  it('handles market_cap with scientific notation values', () => {
    const url = buildScreenerUrl({ market_cap_min: 10e9 });
    expect(url).toContain('market_cap_min=10000000000');
  });

  it('handles zero values (should include them)', () => {
    const url = buildScreenerUrl({ pe_min: 0 });
    expect(url).toContain('pe_min=0');
  });

  it('handles negative values', () => {
    const url = buildScreenerUrl({ revenue_growth_min: -50 });
    expect(url).toContain('revenue_growth_min=-50');
  });
});

// ---------------------------------------------------------------------------
// useScreener hook
// ---------------------------------------------------------------------------

const mockScreenerResponse = {
  data: [
    {
      symbol: 'AAPL',
      name: 'Apple Inc.',
      sector: 'Technology',
      industry: 'Consumer Electronics',
      market_cap: 3000000000000,
      price: 185.5,
      pe_ratio: 28.5,
      pb_ratio: 45.2,
      ps_ratio: 7.8,
      roe: 160.5,
      roa: 28.3,
      gross_margin: 46.2,
      operating_margin: 33.1,
      net_margin: 26.3,
      debt_to_equity: 1.8,
      current_ratio: 1.0,
      revenue_growth: 8.5,
      eps_growth_yoy: 12.3,
      dividend_yield: 0.5,
      payout_ratio: 15.0,
      consecutive_dividend_years: 12,
      beta: 1.2,
      dcf_upside_percent: 5.0,
      ic_score: 78,
      ic_rating: 'A',
      value_score: 55,
      growth_score: 72,
      profitability_score: 90,
      financial_health_score: 65,
      momentum_score: 80,
      analyst_consensus_score: 85,
      insider_activity_score: 50,
      institutional_score: 75,
      news_sentiment_score: 68,
      technical_score: 72,
      ic_sector_percentile: 82,
      lifecycle_stage: 'mature',
    },
  ],
  meta: {
    total: 1,
    page: 1,
    limit: 25,
    total_pages: 1,
    timestamp: '2026-02-15T12:00:00Z',
  },
};

describe('useScreener', () => {
  it('fetches data and returns stocks array', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => mockScreenerResponse,
    });

    const { result } = renderHook(() =>
      useScreener({ page: 1, limit: 25, sort: 'market_cap', order: 'desc' })
    );

    // Initially loading
    expect(result.current.isLoading).toBe(true);
    expect(result.current.stocks).toEqual([]);

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.stocks).toHaveLength(1);
    expect(result.current.stocks[0].symbol).toBe('AAPL');
    expect(result.current.meta?.total).toBe(1);
    expect(result.current.meta?.total_pages).toBe(1);
    expect(result.current.error).toBeUndefined();
  });

  it('calls correct URL with filter params', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => mockScreenerResponse,
    });

    renderHook(() =>
      useScreener({ page: 1, limit: 25, pe_max: 15, sectors: 'Technology' })
    );

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalled();
    });

    const calledUrl = mockFetch.mock.calls[0][0];
    expect(calledUrl).toContain('/screener/stocks');
    expect(calledUrl).toContain('pe_max=15');
    expect(calledUrl).toContain('sectors=Technology');
  });

  it('returns empty stocks array on error', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      json: async () => ({ error: 'Internal server error' }),
    });

    const { result } = renderHook(() =>
      useScreener({ page: 1, limit: 25 })
    );

    await waitFor(() => {
      expect(result.current.error).toBeDefined();
    });

    expect(result.current.stocks).toEqual([]);
    expect(result.current.meta).toBeNull();
    expect(result.current.error?.message).toContain('Internal server error');
  });

  it('handles network failure gracefully', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'));

    const { result } = renderHook(() =>
      useScreener({ page: 1, limit: 25 })
    );

    await waitFor(() => {
      expect(result.current.error).toBeDefined();
    });

    expect(result.current.stocks).toEqual([]);
    expect(result.current.error?.message).toBe('Network error');
  });

  it('handles JSON parse failure on error response', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 502,
      json: async () => { throw new Error('bad json'); },
    });

    const { result } = renderHook(() =>
      useScreener({ page: 1, limit: 25 })
    );

    await waitFor(() => {
      expect(result.current.error).toBeDefined();
    });

    expect(result.current.error?.message).toContain('HTTP 502');
  });

  it('returns meta with pagination info', async () => {
    const response = {
      ...mockScreenerResponse,
      meta: { total: 5600, page: 3, limit: 25, total_pages: 224, timestamp: '2026-02-15T12:00:00Z' },
    };
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => response,
    });

    const { result } = renderHook(() =>
      useScreener({ page: 3, limit: 25 })
    );

    await waitFor(() => {
      expect(result.current.meta).not.toBeNull();
    });

    expect(result.current.meta?.total).toBe(5600);
    expect(result.current.meta?.page).toBe(3);
    expect(result.current.meta?.total_pages).toBe(224);
  });

  it('returns empty data for empty response', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        data: [],
        meta: { total: 0, page: 1, limit: 25, total_pages: 0, timestamp: '2026-02-15T12:00:00Z' },
      }),
    });

    const { result } = renderHook(() =>
      useScreener({ page: 1, limit: 25, pe_max: 1 })
    );

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.stocks).toEqual([]);
    expect(result.current.meta?.total).toBe(0);
  });
});
