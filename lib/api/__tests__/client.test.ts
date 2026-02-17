/**
 * Tests for lib/api/client.ts — apiClient with auth and token refresh
 */
export {};

const mockFetch = global.fetch as jest.Mock;

beforeEach(() => {
  jest.clearAllMocks();
  mockFetch.mockReset();
  (window.localStorage.getItem as jest.Mock).mockReturnValue(null);
  (window.localStorage.setItem as jest.Mock).mockClear();
  (window.localStorage.removeItem as jest.Mock).mockClear();
});

describe('apiClient', () => {
  let apiClient: (typeof import('../client'))['apiClient'];

  beforeEach(async () => {
    jest.resetModules();
    const mod = await import('../client');
    apiClient = mod.apiClient;
  });

  describe('get()', () => {
    it('calls fetch with correct URL and GET method', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ data: [] }),
      });

      await apiClient.get('/test');

      expect(mockFetch).toHaveBeenCalledTimes(1);
      const [url, options] = mockFetch.mock.calls[0];
      expect(url).toContain('/test');
      expect(options.method).toBe('GET');
    });

    it('returns parsed JSON', async () => {
      const expected = { data: [1, 2, 3] };
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => expected,
      });

      const result = await apiClient.get('/items');
      expect(result).toEqual(expected);
    });
  });

  describe('post()', () => {
    it('sends JSON body with POST method', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ id: 1 }),
      });

      await apiClient.post('/create', { name: 'test' });

      const [, options] = mockFetch.mock.calls[0];
      expect(options.method).toBe('POST');
      expect(options.body).toBe(JSON.stringify({ name: 'test' }));
    });
  });

  describe('put()', () => {
    it('sends JSON body with PUT method', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ id: 1 }),
      });

      await apiClient.put('/update', { name: 'updated' });

      const [, options] = mockFetch.mock.calls[0];
      expect(options.method).toBe('PUT');
      expect(options.body).toBe(JSON.stringify({ name: 'updated' }));
    });
  });

  describe('delete()', () => {
    it('calls fetch with DELETE method', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({}),
      });

      await apiClient.delete('/items/1');

      const [url, options] = mockFetch.mock.calls[0];
      expect(url).toContain('/items/1');
      expect(options.method).toBe('DELETE');
    });
  });

  describe('auth header', () => {
    it('includes Authorization Bearer header when token exists', async () => {
      (window.localStorage.getItem as jest.Mock).mockImplementation((key: string) => {
        if (key === 'access_token') return 'my-jwt-token';
        return null;
      });

      jest.resetModules();
      const mod = await import('../client');

      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ data: {} }),
      });

      await mod.apiClient.get('/protected');

      const [, options] = mockFetch.mock.calls[0];
      expect(options.headers.Authorization).toBe('Bearer my-jwt-token');
    });

    it('does not include Authorization header when no token', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ data: {} }),
      });

      await apiClient.get('/public');

      const [, options] = mockFetch.mock.calls[0];
      expect(options.headers.Authorization).toBeUndefined();
    });
  });

  describe('401 handling', () => {
    it('attempts token refresh on 401 and retries', async () => {
      (window.localStorage.getItem as jest.Mock).mockImplementation((key: string) => {
        if (key === 'access_token') return 'expired-token';
        if (key === 'refresh_token') return 'valid-refresh';
        return null;
      });

      jest.resetModules();
      const mod = await import('../client');

      // First call: 401
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        json: async () => ({ error: 'Unauthorized' }),
      });

      // Refresh call: success
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ access_token: 'new-token' }),
      });

      // Retry call: success
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ data: 'success' }),
      });

      const result = await mod.apiClient.get('/protected');
      expect(result).toEqual({ data: 'success' });
      expect(mockFetch).toHaveBeenCalledTimes(3);

      // Verify refresh endpoint was called
      const refreshCall = mockFetch.mock.calls[1];
      expect(refreshCall[0]).toContain('/auth/refresh');
    });

    it('clears tokens and throws on refresh failure', async () => {
      (window.localStorage.getItem as jest.Mock).mockImplementation((key: string) => {
        if (key === 'access_token') return 'expired-token';
        if (key === 'refresh_token') return 'invalid-refresh';
        return null;
      });

      jest.resetModules();
      const mod = await import('../client');

      // First call: 401
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        json: async () => ({ error: 'Unauthorized' }),
      });

      // Refresh call: failure
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        json: async () => ({ error: 'Invalid refresh token' }),
      });

      await expect(mod.apiClient.get('/protected')).rejects.toThrow(
        'Session expired. Please log in again.'
      );

      // Verify tokens were cleared
      expect(window.localStorage.removeItem).toHaveBeenCalledWith('access_token');
      expect(window.localStorage.removeItem).toHaveBeenCalledWith('refresh_token');
      expect(window.localStorage.removeItem).toHaveBeenCalledWith('user');
    });

    it('throws on 401 when no refresh token available', async () => {
      (window.localStorage.getItem as jest.Mock).mockImplementation((key: string) => {
        if (key === 'access_token') return 'expired-token';
        return null; // no refresh_token
      });

      jest.resetModules();
      const mod = await import('../client');

      // First call: 401
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        json: async () => ({ error: 'Unauthorized' }),
      });

      await expect(mod.apiClient.get('/protected')).rejects.toThrow(
        'Session expired. Please log in again.'
      );
    });
  });

  describe('error handling', () => {
    it('throws with error message from response body', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        json: async () => ({ error: 'Bad request: invalid symbol' }),
      });

      await expect(apiClient.get('/invalid')).rejects.toThrow('Bad request: invalid symbol');
    });

    it('throws "Request failed" when no error in body', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: async () => ({}),
      });

      await expect(apiClient.get('/error')).rejects.toThrow('Request failed');
    });

    it('throws "Request failed" when json parsing fails', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: async () => {
          throw new Error('Invalid JSON');
        },
      });

      await expect(apiClient.get('/error')).rejects.toThrow('Request failed');
    });
  });

  describe('Content-Type header', () => {
    it('always sets Content-Type to application/json', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({}),
      });

      await apiClient.get('/test');

      const [, options] = mockFetch.mock.calls[0];
      expect(options.headers['Content-Type']).toBe('application/json');
    });
  });
});
