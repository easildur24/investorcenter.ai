import { alertAPI, ALERT_TYPES, ALERT_FREQUENCIES } from '../alerts';

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

describe('alertAPI', () => {
  describe('listAlerts', () => {
    it('calls GET /alerts with no params', async () => {
      mockGet.mockResolvedValueOnce([]);
      await alertAPI.listAlerts();
      expect(mockGet).toHaveBeenCalledWith('/alerts');
    });

    it('appends watch_list_id query param', async () => {
      mockGet.mockResolvedValueOnce([]);
      await alertAPI.listAlerts({ watch_list_id: 'wl-1' });
      expect(mockGet).toHaveBeenCalledWith('/alerts?watch_list_id=wl-1');
    });

    it('appends is_active query param', async () => {
      mockGet.mockResolvedValueOnce([]);
      await alertAPI.listAlerts({ is_active: true });
      expect(mockGet).toHaveBeenCalledWith('/alerts?is_active=true');
    });

    it('appends multiple query params', async () => {
      mockGet.mockResolvedValueOnce([]);
      await alertAPI.listAlerts({ watch_list_id: 'wl-2', is_active: false });
      expect(mockGet).toHaveBeenCalledWith('/alerts?watch_list_id=wl-2&is_active=false');
    });
  });

  describe('createAlert', () => {
    it('calls POST /alerts with data', async () => {
      const data = {
        watch_list_id: 'wl-1',
        symbol: 'AAPL',
        alert_type: 'price_above',
        conditions: { price: 200 },
        name: 'AAPL above 200',
        frequency: 'once' as const,
        notify_email: true,
        notify_in_app: true,
      };
      mockPost.mockResolvedValueOnce({});
      await alertAPI.createAlert(data);
      expect(mockPost).toHaveBeenCalledWith('/alerts', data);
    });
  });

  describe('getAlert', () => {
    it('calls GET /alerts/:id', async () => {
      mockGet.mockResolvedValueOnce({});
      await alertAPI.getAlert('alert-123');
      expect(mockGet).toHaveBeenCalledWith('/alerts/alert-123');
    });
  });

  describe('updateAlert', () => {
    it('calls PUT /alerts/:id with data', async () => {
      const data = { name: 'Updated', is_active: false };
      mockPut.mockResolvedValueOnce({});
      await alertAPI.updateAlert('alert-123', data);
      expect(mockPut).toHaveBeenCalledWith('/alerts/alert-123', data);
    });
  });

  describe('deleteAlert', () => {
    it('calls DELETE /alerts/:id', async () => {
      mockDelete.mockResolvedValueOnce(undefined);
      await alertAPI.deleteAlert('alert-123');
      expect(mockDelete).toHaveBeenCalledWith('/alerts/alert-123');
    });
  });

  describe('getAlertLogs', () => {
    it('calls GET /alerts/logs with no params', async () => {
      mockGet.mockResolvedValueOnce([]);
      await alertAPI.getAlertLogs();
      expect(mockGet).toHaveBeenCalledWith('/alerts/logs');
    });

    it('appends limit and offset params', async () => {
      mockGet.mockResolvedValueOnce([]);
      await alertAPI.getAlertLogs({ limit: 10, offset: 20 });
      expect(mockGet).toHaveBeenCalledWith('/alerts/logs?limit=10&offset=20');
    });
  });

  describe('markAlertLogRead', () => {
    it('calls POST /alerts/logs/:id/read', async () => {
      mockPost.mockResolvedValueOnce(undefined);
      await alertAPI.markAlertLogRead('log-1');
      expect(mockPost).toHaveBeenCalledWith('/alerts/logs/log-1/read', {});
    });
  });

  describe('dismissAlertLog', () => {
    it('calls POST /alerts/logs/:id/dismiss', async () => {
      mockPost.mockResolvedValueOnce(undefined);
      await alertAPI.dismissAlertLog('log-1');
      expect(mockPost).toHaveBeenCalledWith('/alerts/logs/log-1/dismiss', {});
    });
  });
});

describe('ALERT_TYPES constant', () => {
  it('has price alert types', () => {
    expect(ALERT_TYPES.price_above).toBeDefined();
    expect(ALERT_TYPES.price_above.category).toBe('price');
  });

  it('has volume alert types', () => {
    expect(ALERT_TYPES.volume_spike).toBeDefined();
    expect(ALERT_TYPES.volume_spike.category).toBe('volume');
  });

  it('has event alert types', () => {
    expect(ALERT_TYPES.news).toBeDefined();
    expect(ALERT_TYPES.news.category).toBe('event');
  });
});

describe('ALERT_FREQUENCIES constant', () => {
  it('has once, daily, always', () => {
    expect(ALERT_FREQUENCIES.once.label).toBe('Once');
    expect(ALERT_FREQUENCIES.daily.label).toBe('Daily');
    expect(ALERT_FREQUENCIES.always.label).toBe('Always');
  });
});
