/**
 * Tests for the ApiClient in lib/api.ts
 *
 * We test the core request method, auth header injection,
 * Content-Type behavior, and error handling.
 */

const mockFetch = global.fetch as jest.Mock;

beforeEach(() => {
  jest.clearAllMocks();
  mockFetch.mockReset();
});

describe('apiClient', () => {
  // Use dynamic import so we can reset modules between tests
  let apiClient: typeof import('../api')['apiClient'];
  beforeAll(async () => {
    const mod = await import('../api');
    apiClient = mod.apiClient;
  });

  describe('request method', () => {
    it('makes request to correct URL', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: [], meta: { timestamp: '2026-01-01' } }),
      });

      await apiClient.request('/test-endpoint');

      expect(mockFetch).toHaveBeenCalledTimes(1);
      const [url] = mockFetch.mock.calls[0];
      expect(url).toContain('/test-endpoint');
    });

    it('does not set Content-Type for GET requests', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: {}, meta: { timestamp: '2026-01-01' } }),
      });

      await apiClient.request('/endpoint');

      const [, options] = mockFetch.mock.calls[0];
      const headers = options.headers as Record<string, string>;
      expect(headers['Content-Type']).toBeUndefined();
    });

    it('sets Content-Type for POST requests', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: {}, meta: { timestamp: '2026-01-01' } }),
      });

      await apiClient.request('/endpoint', {
        method: 'POST',
        body: JSON.stringify({ test: true }),
      });

      const [, options] = mockFetch.mock.calls[0];
      const headers = options.headers as Record<string, string>;
      expect(headers['Content-Type']).toBe('application/json');
    });

    it('sets Content-Type for PUT requests', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: {}, meta: { timestamp: '2026-01-01' } }),
      });

      await apiClient.request('/endpoint', {
        method: 'PUT',
        body: JSON.stringify({ update: true }),
      });

      const [, options] = mockFetch.mock.calls[0];
      const headers = options.headers as Record<string, string>;
      expect(headers['Content-Type']).toBe('application/json');
    });

    it('throws on HTTP error response with error message', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        json: async () => ({ error: 'Not found' }),
      });

      await expect(
        apiClient.request('/nonexistent')
      ).rejects.toThrow('Not found');
    });

    it('throws with HTTP status when no error message in body', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: async () => ({}),
      });

      await expect(
        apiClient.request('/error')
      ).rejects.toThrow('HTTP 500');
    });

    it('returns parsed JSON on success', async () => {
      const expectedData = { data: { id: 1, name: 'Test' }, meta: { timestamp: '2026-01-01' } };
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => expectedData,
      });

      const result = await apiClient.request('/success');
      expect(result).toEqual(expectedData);
    });
  });

  describe('auth token handling', () => {
    it('includes Authorization header when token set in localStorage', async () => {
      // Set token in localStorage
      (window.localStorage.getItem as jest.Mock).mockReturnValue('test-jwt-token');

      // Re-import to pick up the token
      jest.resetModules();
      const freshMod = await import('../api');

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: {}, meta: { timestamp: '2026-01-01' } }),
      });

      await freshMod.apiClient.request('/authenticated');

      const [, options] = mockFetch.mock.calls[0];
      const headers = options.headers as Record<string, string>;
      expect(headers.Authorization).toBe('Bearer test-jwt-token');
    });
  });
});
