/**
 * Tests for useRealTimePrice hook
 *
 * We test the fetch logic (single /price endpoint for stocks and crypto),
 * error handling, and data transformation.
 */

import { renderHook, waitFor, act } from '@testing-library/react';
import { useRealTimePrice } from '../useRealTimePrice';
import { tickers } from '@/lib/api/routes';
import { API_BASE_URL } from '@/lib/api';

const mockFetch = global.fetch as jest.Mock;

beforeEach(() => {
  jest.useFakeTimers();
  mockFetch.mockReset();
});

afterEach(() => {
  jest.useRealTimers();
});

describe('useRealTimePrice', () => {
  it('returns null data initially', () => {
    mockFetch.mockResolvedValue({ ok: false, status: 404 });

    const { result } = renderHook(() => useRealTimePrice({ symbol: 'AAPL', enabled: false }));

    expect(result.current.priceData).toBeNull();
    expect(result.current.error).toBeNull();
  });

  it('fetches crypto data from the price endpoint', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        data: {
          symbol: 'X:BTCUSD',
          price: '45000.00',
          change: '1500.00',
          changePercent: '3.45',
          volume: 1000000,
          lastUpdated: '2026-02-13T10:00:00Z',
          marketStatus: 'open',
          assetType: 'crypto',
        },
        meta: { timestamp: '2026-02-13T10:00:00Z', source: 'redis' },
      }),
    });

    const { result } = renderHook(() => useRealTimePrice({ symbol: 'X:BTCUSD' }));

    await waitFor(() => {
      expect(result.current.priceData).not.toBeNull();
    });

    expect(result.current.isCrypto).toBe(true);
    expect(result.current.isMarketOpen).toBe(true); // Crypto always open
    expect(result.current.priceData?.price).toBe('45000.00');
  });

  it('fetches stock data from the price endpoint', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        data: {
          symbol: 'AAPL',
          price: '150.25',
          change: '2.50',
          changePercent: '1.69',
          volume: 50000000,
          lastUpdated: '2026-02-13T16:00:00Z',
        },
        market: {
          isOpen: true,
          updateInterval: 15000,
        },
      }),
    });

    const { result } = renderHook(() => useRealTimePrice({ symbol: 'AAPL' }));

    await waitFor(() => {
      expect(result.current.priceData).not.toBeNull();
    });

    expect(result.current.isCrypto).toBe(false);
    expect(result.current.priceData?.price).toBe('150.25');
    expect(result.current.priceData?.change).toBe('2.50');
  });

  it('sets error when endpoint fails', async () => {
    mockFetch.mockResolvedValue({ ok: false, status: 500 });

    const { result } = renderHook(() => useRealTimePrice({ symbol: 'INVALID' }));

    await waitFor(() => {
      expect(result.current.error).not.toBeNull();
    });

    expect(result.current.error).toContain('Failed to fetch price');
    expect(result.current.priceData).toBeNull();
  });

  it('does not fetch when disabled', () => {
    const { result } = renderHook(() => useRealTimePrice({ symbol: 'AAPL', enabled: false }));

    expect(mockFetch).not.toHaveBeenCalled();
    expect(result.current.priceData).toBeNull();
  });

  it('does not fetch for empty symbol', () => {
    const { result } = renderHook(() => useRealTimePrice({ symbol: '' }));

    expect(mockFetch).not.toHaveBeenCalled();
    expect(result.current.priceData).toBeNull();
  });

  it('calls price endpoint with correct URL', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        data: {
          symbol: 'X:ETHUSD',
          price: '3000.00',
          change: '50.00',
          changePercent: '1.69',
          volume: 500,
          lastUpdated: '2026-02-13T10:00:00Z',
          assetType: 'crypto',
        },
        meta: { timestamp: '2026-02-13T10:00:00Z' },
      }),
    });

    renderHook(() => useRealTimePrice({ symbol: 'X:ETHUSD' }));

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalled();
    });

    expect(mockFetch.mock.calls[0][0]).toBe(`${API_BASE_URL}${tickers.price('X:ETHUSD')}`);
  });

  it('cleans up interval on unmount', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        data: {
          symbol: 'X:BTCUSD',
          price: '45000.00',
          change: '100.00',
          changePercent: '0.22',
          volume: 500,
          lastUpdated: '2026-02-13T10:00:00Z',
          assetType: 'crypto',
        },
        meta: { timestamp: '2026-02-13T10:00:00Z' },
      }),
    });

    const { unmount } = renderHook(() => useRealTimePrice({ symbol: 'X:BTCUSD' }));

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalled();
    });

    const fetchCountBefore = mockFetch.mock.calls.length;
    unmount();

    // Advance timers past the polling interval
    act(() => {
      jest.advanceTimersByTime(10000);
    });

    // No additional fetch calls after unmount
    expect(mockFetch.mock.calls.length).toBe(fetchCountBefore);
  });
});
