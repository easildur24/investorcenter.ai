import { watchListAPI } from '../watchlist';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
  },
}));

import { apiClient } from '../client';

const mockGet = apiClient.get as jest.Mock;
const mockPost = apiClient.post as jest.Mock;
const mockPut = apiClient.put as jest.Mock;
const mockDelete = apiClient.delete as jest.Mock;

beforeEach(() => {
  jest.clearAllMocks();
});

describe('watchListAPI', () => {
  describe('getWatchLists', () => {
    it('calls GET /watchlists', async () => {
      mockGet.mockResolvedValueOnce({ watch_lists: [] });
      await watchListAPI.getWatchLists();
      expect(mockGet).toHaveBeenCalledWith('/watchlists');
    });
  });

  describe('createWatchList', () => {
    it('calls POST /watchlists with data', async () => {
      const data = { name: 'My List', description: 'Test' };
      mockPost.mockResolvedValueOnce({});
      await watchListAPI.createWatchList(data);
      expect(mockPost).toHaveBeenCalledWith('/watchlists', data);
    });
  });

  describe('getWatchList', () => {
    it('calls GET /watchlists/:id', async () => {
      mockGet.mockResolvedValueOnce({});
      await watchListAPI.getWatchList('wl-1');
      expect(mockGet).toHaveBeenCalledWith('/watchlists/wl-1');
    });
  });

  describe('updateWatchList', () => {
    it('calls PUT /watchlists/:id', async () => {
      const data = { name: 'Updated', description: 'New desc' };
      mockPut.mockResolvedValueOnce(undefined);
      await watchListAPI.updateWatchList('wl-1', data);
      expect(mockPut).toHaveBeenCalledWith('/watchlists/wl-1', data);
    });
  });

  describe('deleteWatchList', () => {
    it('calls DELETE /watchlists/:id', async () => {
      mockDelete.mockResolvedValueOnce(undefined);
      await watchListAPI.deleteWatchList('wl-1');
      expect(mockDelete).toHaveBeenCalledWith('/watchlists/wl-1');
    });
  });

  describe('addTicker', () => {
    it('calls POST /watchlists/:id/items', async () => {
      const data = { symbol: 'AAPL', notes: 'Buy on dip' };
      mockPost.mockResolvedValueOnce({});
      await watchListAPI.addTicker('wl-1', data);
      expect(mockPost).toHaveBeenCalledWith('/watchlists/wl-1/items', data);
    });
  });

  describe('removeTicker', () => {
    it('calls DELETE /watchlists/:id/items/:symbol', async () => {
      mockDelete.mockResolvedValueOnce(undefined);
      await watchListAPI.removeTicker('wl-1', 'AAPL');
      expect(mockDelete).toHaveBeenCalledWith('/watchlists/wl-1/items/AAPL');
    });
  });

  describe('updateTicker', () => {
    it('calls PUT /watchlists/:id/items/:symbol', async () => {
      const data = { notes: 'Updated', tags: ['tech'] };
      mockPut.mockResolvedValueOnce({});
      await watchListAPI.updateTicker('wl-1', 'AAPL', data);
      expect(mockPut).toHaveBeenCalledWith('/watchlists/wl-1/items/AAPL', data);
    });
  });

  describe('bulkAddTickers', () => {
    it('calls POST /watchlists/:id/bulk', async () => {
      mockPost.mockResolvedValueOnce({ added: ['AAPL', 'MSFT'], failed: [], total: 2 });
      const result = await watchListAPI.bulkAddTickers('wl-1', ['AAPL', 'MSFT']);
      expect(mockPost).toHaveBeenCalledWith('/watchlists/wl-1/bulk', {
        symbols: ['AAPL', 'MSFT'],
      });
      expect(result.added).toEqual(['AAPL', 'MSFT']);
    });
  });

  describe('reorderItems', () => {
    it('calls POST /watchlists/:id/reorder', async () => {
      const orders = [
        { item_id: 'i-1', display_order: 0 },
        { item_id: 'i-2', display_order: 1 },
      ];
      mockPost.mockResolvedValueOnce(undefined);
      await watchListAPI.reorderItems('wl-1', orders);
      expect(mockPost).toHaveBeenCalledWith('/watchlists/wl-1/reorder', {
        item_orders: orders,
      });
    });
  });
});
