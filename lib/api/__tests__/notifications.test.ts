import { notificationAPI } from '../notifications';

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

beforeEach(() => {
  jest.clearAllMocks();
});

describe('notificationAPI', () => {
  describe('getNotifications', () => {
    it('calls GET /notifications with no params', async () => {
      mockGet.mockResolvedValueOnce([]);
      await notificationAPI.getNotifications();
      expect(mockGet).toHaveBeenCalledWith('/notifications');
    });

    it('appends limit and offset params', async () => {
      mockGet.mockResolvedValueOnce([]);
      await notificationAPI.getNotifications({ limit: 10, offset: 5 });
      const url = mockGet.mock.calls[0][0];
      expect(url).toContain('limit=10');
      expect(url).toContain('offset=5');
    });

    it('appends unread_only param', async () => {
      mockGet.mockResolvedValueOnce([]);
      await notificationAPI.getNotifications({ unread_only: true });
      expect(mockGet.mock.calls[0][0]).toContain('unread_only=true');
    });
  });

  describe('getUnreadCount', () => {
    it('calls GET /notifications/unread-count', async () => {
      mockGet.mockResolvedValueOnce({ count: 5 });
      const result = await notificationAPI.getUnreadCount();
      expect(mockGet).toHaveBeenCalledWith('/notifications/unread-count');
      expect(result.count).toBe(5);
    });
  });

  describe('markAsRead', () => {
    it('calls POST /notifications/:id/read', async () => {
      mockPost.mockResolvedValueOnce(undefined);
      await notificationAPI.markAsRead('n-1');
      expect(mockPost).toHaveBeenCalledWith('/notifications/n-1/read', {});
    });
  });

  describe('markAllAsRead', () => {
    it('calls POST /notifications/read-all', async () => {
      mockPost.mockResolvedValueOnce(undefined);
      await notificationAPI.markAllAsRead();
      expect(mockPost).toHaveBeenCalledWith('/notifications/read-all', {});
    });
  });

  describe('dismiss', () => {
    it('calls POST /notifications/:id/dismiss', async () => {
      mockPost.mockResolvedValueOnce(undefined);
      await notificationAPI.dismiss('n-1');
      expect(mockPost).toHaveBeenCalledWith('/notifications/n-1/dismiss', {});
    });
  });

  describe('getPreferences', () => {
    it('calls GET /notifications/preferences', async () => {
      mockGet.mockResolvedValueOnce({});
      await notificationAPI.getPreferences();
      expect(mockGet).toHaveBeenCalledWith('/notifications/preferences');
    });
  });

  describe('updatePreferences', () => {
    it('calls PUT /notifications/preferences', async () => {
      const data = { email_enabled: false };
      mockPut.mockResolvedValueOnce({});
      await notificationAPI.updatePreferences(data);
      expect(mockPut).toHaveBeenCalledWith('/notifications/preferences', data);
    });
  });
});
