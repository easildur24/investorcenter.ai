import { renderHook, waitFor, act } from '@testing-library/react';
import { useWatchlistAlerts } from '../useWatchlistAlerts';
import { alertAPI, AlertRuleWithDetails } from '@/lib/api/alerts';

// Mock the alertAPI module
jest.mock('@/lib/api/alerts', () => ({
  alertAPI: {
    listAlerts: jest.fn(),
    createAlert: jest.fn(),
    updateAlert: jest.fn(),
    deleteAlert: jest.fn(),
  },
}));

const mockListAlerts = alertAPI.listAlerts as jest.Mock;
const mockCreateAlert = alertAPI.createAlert as jest.Mock;
const mockUpdateAlert = alertAPI.updateAlert as jest.Mock;
const mockDeleteAlert = alertAPI.deleteAlert as jest.Mock;

beforeEach(() => {
  jest.clearAllMocks();
});

function buildAlert(overrides: Partial<AlertRuleWithDetails> = {}): AlertRuleWithDetails {
  return {
    id: 'alert-1',
    user_id: 'user-1',
    watch_list_id: 'wl-1',
    symbol: 'AAPL',
    alert_type: 'price_above',
    conditions: { price: 200 },
    is_active: true,
    frequency: 'once',
    notify_email: true,
    notify_in_app: true,
    name: 'AAPL above 200',
    trigger_count: 0,
    created_at: '2025-01-01T00:00:00Z',
    updated_at: '2025-01-01T00:00:00Z',
    watch_list_name: 'My Watchlist',
    company_name: 'Apple Inc.',
    ...overrides,
  };
}

describe('useWatchlistAlerts', () => {
  describe('initial fetch', () => {
    it('does not fetch when watchListId is null', async () => {
      const { result } = renderHook(() => useWatchlistAlerts(null));

      // Wait a tick to ensure no async operations fire
      await act(async () => {});

      expect(mockListAlerts).not.toHaveBeenCalled();
      expect(result.current.loading).toBe(false);
      expect(result.current.alertsBySymbol.size).toBe(0);
    });

    it('fetches alerts when watchListId is provided', async () => {
      const alerts = [buildAlert(), buildAlert({ id: 'alert-2', symbol: 'MSFT' })];
      mockListAlerts.mockResolvedValueOnce(alerts);

      const { result } = renderHook(() => useWatchlistAlerts('wl-1'));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(mockListAlerts).toHaveBeenCalledWith({ watch_list_id: 'wl-1' });
      expect(result.current.alertsBySymbol.size).toBe(2);
      expect(result.current.alertsBySymbol.get('AAPL')).toEqual(alerts[0]);
      expect(result.current.alertsBySymbol.get('MSFT')).toEqual(alerts[1]);
    });

    it('handles fetch error gracefully', async () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
      mockListAlerts.mockRejectedValueOnce(new Error('Network error'));

      const { result } = renderHook(() => useWatchlistAlerts('wl-1'));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.alertsBySymbol.size).toBe(0);
      expect(consoleSpy).toHaveBeenCalled();
      consoleSpy.mockRestore();
    });

    it('handles null response from listAlerts', async () => {
      mockListAlerts.mockResolvedValueOnce(null);

      const { result } = renderHook(() => useWatchlistAlerts('wl-1'));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.alertsBySymbol.size).toBe(0);
    });
  });

  describe('alertsBySymbol map', () => {
    it('builds a map keyed by symbol', async () => {
      const alerts = [
        buildAlert({ symbol: 'AAPL' }),
        buildAlert({ id: 'alert-2', symbol: 'GOOG' }),
      ];
      mockListAlerts.mockResolvedValueOnce(alerts);

      const { result } = renderHook(() => useWatchlistAlerts('wl-1'));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.alertsBySymbol.has('AAPL')).toBe(true);
      expect(result.current.alertsBySymbol.has('GOOG')).toBe(true);
      expect(result.current.alertsBySymbol.has('TSLA')).toBe(false);
    });
  });

  describe('createAlert', () => {
    it('creates alert and re-fetches the list', async () => {
      // Initial fetch
      mockListAlerts.mockResolvedValueOnce([]);

      const { result } = renderHook(() => useWatchlistAlerts('wl-1'));
      await waitFor(() => expect(result.current.loading).toBe(false));

      const newAlertRule = { id: 'alert-new', symbol: 'TSLA' };
      mockCreateAlert.mockResolvedValueOnce(newAlertRule);
      // Re-fetch after create
      const updatedAlerts = [buildAlert({ id: 'alert-new', symbol: 'TSLA' })];
      mockListAlerts.mockResolvedValueOnce(updatedAlerts);

      const createReq = {
        watch_list_id: 'wl-1',
        symbol: 'TSLA',
        alert_type: 'price_above',
        conditions: { price: 300 },
        name: 'TSLA above 300',
        frequency: 'once' as const,
        notify_email: true,
        notify_in_app: true,
      };

      let created: any;
      await act(async () => {
        created = await result.current.createAlert(createReq);
      });

      expect(mockCreateAlert).toHaveBeenCalledWith(createReq);
      expect(created).toEqual(newAlertRule);
      // Should have re-fetched
      expect(mockListAlerts).toHaveBeenCalledTimes(2);
    });
  });

  describe('updateAlert', () => {
    it('updates alert and re-fetches the list', async () => {
      const alerts = [buildAlert()];
      mockListAlerts.mockResolvedValueOnce(alerts);

      const { result } = renderHook(() => useWatchlistAlerts('wl-1'));
      await waitFor(() => expect(result.current.loading).toBe(false));

      mockUpdateAlert.mockResolvedValueOnce(undefined);
      const updatedAlerts = [buildAlert({ name: 'Updated Alert' })];
      mockListAlerts.mockResolvedValueOnce(updatedAlerts);

      await act(async () => {
        await result.current.updateAlert('alert-1', { name: 'Updated Alert' });
      });

      expect(mockUpdateAlert).toHaveBeenCalledWith('alert-1', { name: 'Updated Alert' });
      expect(mockListAlerts).toHaveBeenCalledTimes(2);
    });
  });

  describe('deleteAlert', () => {
    it('deletes alert and removes it from local state', async () => {
      const alerts = [
        buildAlert({ id: 'alert-1', symbol: 'AAPL' }),
        buildAlert({ id: 'alert-2', symbol: 'MSFT' }),
      ];
      mockListAlerts.mockResolvedValueOnce(alerts);

      const { result } = renderHook(() => useWatchlistAlerts('wl-1'));
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.alertsBySymbol.size).toBe(2);

      mockDeleteAlert.mockResolvedValueOnce(undefined);

      await act(async () => {
        await result.current.deleteAlert('alert-1', 'AAPL');
      });

      expect(mockDeleteAlert).toHaveBeenCalledWith('alert-1');
      expect(result.current.alertsBySymbol.size).toBe(1);
      expect(result.current.alertsBySymbol.has('AAPL')).toBe(false);
      expect(result.current.alertsBySymbol.has('MSFT')).toBe(true);
    });
  });

  describe('refresh', () => {
    it('re-fetches alerts when refresh is called', async () => {
      mockListAlerts.mockResolvedValueOnce([buildAlert()]);

      const { result } = renderHook(() => useWatchlistAlerts('wl-1'));
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.alertsBySymbol.size).toBe(1);

      // On refresh, return updated list
      const updatedAlerts = [buildAlert(), buildAlert({ id: 'alert-3', symbol: 'NVDA' })];
      mockListAlerts.mockResolvedValueOnce(updatedAlerts);

      await act(async () => {
        await result.current.refresh();
      });

      expect(result.current.alertsBySymbol.size).toBe(2);
      expect(mockListAlerts).toHaveBeenCalledTimes(2);
    });
  });

  describe('watchListId changes', () => {
    it('re-fetches when watchListId changes', async () => {
      mockListAlerts.mockResolvedValueOnce([buildAlert({ symbol: 'AAPL' })]);

      const { result, rerender } = renderHook(
        ({ id }: { id: string | null }) => useWatchlistAlerts(id),
        { initialProps: { id: 'wl-1' } }
      );

      await waitFor(() => expect(result.current.loading).toBe(false));
      expect(mockListAlerts).toHaveBeenCalledWith({ watch_list_id: 'wl-1' });

      mockListAlerts.mockResolvedValueOnce([buildAlert({ symbol: 'TSLA' })]);

      rerender({ id: 'wl-2' });

      await waitFor(() => {
        expect(result.current.alertsBySymbol.has('TSLA')).toBe(true);
      });

      expect(mockListAlerts).toHaveBeenLastCalledWith({ watch_list_id: 'wl-2' });
    });
  });
});
