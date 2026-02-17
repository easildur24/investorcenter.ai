/**
 * Tests for lib/api/metrics.ts
 *
 * Uses fetch directly (not apiClient), so we mock global.fetch.
 * Also tests the pure helper function extractMetricsCategory.
 */

import {
  getComprehensiveMetrics,
  hasComprehensiveMetrics,
  extractMetricsCategory,
} from '../metrics';

const mockFetch = global.fetch as jest.Mock;

beforeEach(() => {
  jest.clearAllMocks();
  mockFetch.mockReset();
});

describe('getComprehensiveMetrics', () => {
  it('calls correct endpoint and returns data', async () => {
    const mockResponse = { data: { valuation: {} }, meta: { fmp_available: true } };
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => mockResponse,
    });

    const result = await getComprehensiveMetrics('AAPL');
    expect(result).toEqual(mockResponse);
    expect(mockFetch.mock.calls[0][0]).toContain('/stocks/AAPL/metrics');
  });

  it('uppercases ticker', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({}),
    });
    await getComprehensiveMetrics('aapl');
    expect(mockFetch.mock.calls[0][0]).toContain('/stocks/AAPL/');
  });

  it('returns null on 404', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 404,
    });
    expect(await getComprehensiveMetrics('UNKNOWN')).toBeNull();
  });

  it('returns null on server error', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
    });
    expect(await getComprehensiveMetrics('AAPL')).toBeNull();
  });

  it('returns null on network error', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'));
    expect(await getComprehensiveMetrics('AAPL')).toBeNull();
  });
});

describe('hasComprehensiveMetrics', () => {
  it('returns true when fmp_available is true', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ meta: { fmp_available: true } }),
    });
    expect(await hasComprehensiveMetrics('AAPL')).toBe(true);
  });

  it('returns false when fmp_available is false', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ meta: { fmp_available: false } }),
    });
    expect(await hasComprehensiveMetrics('AAPL')).toBe(false);
  });

  it('returns false when meta is missing', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({}),
    });
    expect(await hasComprehensiveMetrics('AAPL')).toBe(false);
  });

  it('returns false on non-ok response', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 404,
    });
    expect(await hasComprehensiveMetrics('X')).toBe(false);
  });

  it('returns false on fetch error', async () => {
    mockFetch.mockRejectedValueOnce(new Error('fail'));
    expect(await hasComprehensiveMetrics('X')).toBe(false);
  });
});

describe('extractMetricsCategory', () => {
  it('extracts a specific category', () => {
    const response = {
      data: {
        valuation: { pe_ratio: 25 },
        growth: { revenue_growth: 0.1 },
      },
      meta: { fmp_available: true },
    } as any;

    expect(extractMetricsCategory(response, 'valuation')).toEqual({ pe_ratio: 25 });
    expect(extractMetricsCategory(response, 'growth')).toEqual({ revenue_growth: 0.1 });
  });

  it('returns null for null response', () => {
    expect(extractMetricsCategory(null, 'valuation' as any)).toBeNull();
  });

  it('returns null when data is missing', () => {
    const response = { meta: {} } as any;
    expect(extractMetricsCategory(response, 'valuation' as any)).toBeNull();
  });
});
