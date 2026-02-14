/**
 * Tests for useRealTimePrice hook
 *
 * We test the fetch logic (crypto vs stock fallback),
 * error handling, and data transformation.
 */

import { renderHook, waitFor, act } from '@testing-library/react';
import { useRealTimePrice } from '../useRealTimePrice';

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

    const { result } = renderHook(() =>
      useRealTimePrice({ symbol: 'AAPL', enabled: false })
    );

    expect(result.current.priceData).toBeNull();
    expect(result.current.error).toBeNull();
  });

  it('fetches crypto data when crypto endpoint succeeds', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        price: 45000,
        change_24h: 5.2,
        volume_24h: 1000000,
        last_updated: '2026-02-13T10:00:00Z',
        update_interval: 5000,
      }),
    });

    const { result } = renderHook(() =>
      useRealTimePrice({ symbol: 'X:BTCUSD' })
    );

    await waitFor(() => {
      expect(result.current.priceData).not.toBeNull();
    });

    expect(result.current.isCrypto).toBe(true);
    expect(result.current.isMarketOpen).toBe(true); // Crypto always open
    expect(result.current.priceData?.price).toBe('45000');
  });

  it('falls back to stock endpoint when crypto fails', async () => {
    // First call (crypto) fails, second call (stock) succeeds
    mockFetch
      .mockResolvedValueOnce({ ok: false, status: 404 })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          data: {
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

    const { result } = renderHook(() =>
      useRealTimePrice({ symbol: 'AAPL' })
    );

    await waitFor(() => {
      expect(result.current.priceData).not.toBeNull();
    });

    expect(result.current.isCrypto).toBe(false);
    expect(result.current.priceData?.price).toBe('150.25');
    expect(result.current.priceData?.change).toBe('2.50');
  });

  it('sets error when both endpoints fail', async () => {
    mockFetch
      .mockResolvedValueOnce({ ok: false, status: 404 }) // crypto
      .mockResolvedValueOnce({ ok: false, status: 500 }); // stock

    const { result } = renderHook(() =>
      useRealTimePrice({ symbol: 'INVALID' })
    );

    await waitFor(() => {
      expect(result.current.error).not.toBeNull();
    });

    expect(result.current.error).toContain('Failed to fetch price');
    expect(result.current.priceData).toBeNull();
  });

  it('does not fetch when disabled', () => {
    const { result } = renderHook(() =>
      useRealTimePrice({ symbol: 'AAPL', enabled: false })
    );

    expect(mockFetch).not.toHaveBeenCalled();
    expect(result.current.priceData).toBeNull();
  });

  it('does not fetch for empty symbol', () => {
    const { result } = renderHook(() =>
      useRealTimePrice({ symbol: '' })
    );

    expect(mockFetch).not.toHaveBeenCalled();
    expect(result.current.priceData).toBeNull();
  });

  it('calls crypto endpoint with correct URL', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        price: 100,
        change_24h: 1,
        volume_24h: 500,
      }),
    });

    renderHook(() => useRealTimePrice({ symbol: 'X:ETHUSD' }));

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalled();
    });

    expect(mockFetch.mock.calls[0][0]).toBe('/api/v1/crypto/X:ETHUSD/price');
  });

  it('cleans up interval on unmount', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        price: 100,
        change_24h: 1,
        volume_24h: 500,
        update_interval: 5000,
      }),
    });

    const { unmount } = renderHook(() =>
      useRealTimePrice({ symbol: 'X:BTCUSD' })
    );

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
