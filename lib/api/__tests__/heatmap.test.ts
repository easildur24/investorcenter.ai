import { heatmapAPI } from '../heatmap';

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

describe('heatmapAPI', () => {
  describe('getHeatmapData', () => {
    it('calls GET /watchlists/:id/heatmap with no overrides', async () => {
      mockGet.mockResolvedValueOnce({});
      await heatmapAPI.getHeatmapData('wl-1');
      expect(mockGet).toHaveBeenCalledWith('/watchlists/wl-1/heatmap');
    });

    it('appends config_id param', async () => {
      mockGet.mockResolvedValueOnce({});
      await heatmapAPI.getHeatmapData('wl-1', 'cfg-1');
      expect(mockGet).toHaveBeenCalledWith('/watchlists/wl-1/heatmap?config_id=cfg-1');
    });

    it('appends override params', async () => {
      mockGet.mockResolvedValueOnce({});
      await heatmapAPI.getHeatmapData('wl-1', undefined, {
        size_metric: 'market_cap',
        color_metric: 'price_change_pct',
        time_period: '1D',
      });
      const url = mockGet.mock.calls[0][0];
      expect(url).toContain('size_metric=market_cap');
      expect(url).toContain('color_metric=price_change_pct');
      expect(url).toContain('time_period=1D');
    });

    it('combines config_id with overrides', async () => {
      mockGet.mockResolvedValueOnce({});
      await heatmapAPI.getHeatmapData('wl-1', 'cfg-1', { size_metric: 'volume' });
      const url = mockGet.mock.calls[0][0];
      expect(url).toContain('config_id=cfg-1');
      expect(url).toContain('size_metric=volume');
    });
  });

  describe('getConfigs', () => {
    it('calls GET /watchlists/:id/heatmap/configs', async () => {
      mockGet.mockResolvedValueOnce({ configs: [] });
      await heatmapAPI.getConfigs('wl-1');
      expect(mockGet).toHaveBeenCalledWith('/watchlists/wl-1/heatmap/configs');
    });
  });

  describe('createConfig', () => {
    it('calls POST /watchlists/:id/heatmap/configs', async () => {
      const config = {
        name: 'My Config',
        size_metric: 'market_cap',
        color_metric: 'price_change_pct',
        time_period: '1D',
      };
      mockPost.mockResolvedValueOnce({});
      await heatmapAPI.createConfig('wl-1', config);
      expect(mockPost).toHaveBeenCalledWith('/watchlists/wl-1/heatmap/configs', {
        watch_list_id: 'wl-1',
        ...config,
      });
    });
  });

  describe('updateConfig', () => {
    it('calls PUT /watchlists/:id/heatmap/configs/:configId', async () => {
      const updates = { name: 'Updated' };
      mockPut.mockResolvedValueOnce({});
      await heatmapAPI.updateConfig('wl-1', 'cfg-1', updates);
      expect(mockPut).toHaveBeenCalledWith('/watchlists/wl-1/heatmap/configs/cfg-1', updates);
    });
  });

  describe('deleteConfig', () => {
    it('calls DELETE /watchlists/:id/heatmap/configs/:configId', async () => {
      mockDelete.mockResolvedValueOnce(undefined);
      await heatmapAPI.deleteConfig('wl-1', 'cfg-1');
      expect(mockDelete).toHaveBeenCalledWith('/watchlists/wl-1/heatmap/configs/cfg-1');
    });
  });
});
