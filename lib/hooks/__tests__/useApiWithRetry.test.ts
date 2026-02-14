import { fetchWithRetry } from '../useApiWithRetry';

// Use fake timers for testing exponential backoff
beforeEach(() => {
  jest.useFakeTimers();
});

afterEach(() => {
  jest.useRealTimers();
});

describe('fetchWithRetry', () => {
  it('succeeds on first try', async () => {
    const fetchFn = jest.fn().mockResolvedValue('success');

    const result = await fetchWithRetry(fetchFn);

    expect(result).toBe('success');
    expect(fetchFn).toHaveBeenCalledTimes(1);
  });

  it('retries on failure and succeeds', async () => {
    const fetchFn = jest.fn()
      .mockRejectedValueOnce(new Error('fail 1'))
      .mockResolvedValue('success after retry');

    // Run with real timers since we need the promise to resolve
    jest.useRealTimers();

    const result = await fetchWithRetry(fetchFn, {
      maxRetries: 3,
      retryDelay: 10, // Short delay for testing
    });

    expect(result).toBe('success after retry');
    expect(fetchFn).toHaveBeenCalledTimes(2);
  });

  it('exhausts retries and throws last error', async () => {
    jest.useRealTimers();

    const fetchFn = jest.fn().mockRejectedValue(new Error('persistent failure'));

    await expect(
      fetchWithRetry(fetchFn, {
        maxRetries: 2,
        retryDelay: 10,
      })
    ).rejects.toThrow('persistent failure');

    // 1 initial + 2 retries = 3 total calls
    expect(fetchFn).toHaveBeenCalledTimes(3);
  });

  it('calls onRetry callback with attempt number', async () => {
    jest.useRealTimers();

    const fetchFn = jest.fn()
      .mockRejectedValueOnce(new Error('fail 1'))
      .mockRejectedValueOnce(new Error('fail 2'))
      .mockResolvedValue('success');

    const onRetry = jest.fn();

    await fetchWithRetry(fetchFn, {
      maxRetries: 3,
      retryDelay: 10,
      onRetry,
    });

    expect(onRetry).toHaveBeenCalledTimes(2);
    expect(onRetry).toHaveBeenCalledWith(1);
    expect(onRetry).toHaveBeenCalledWith(2);
  });

  it('wraps non-Error thrown values', async () => {
    jest.useRealTimers();

    const fetchFn = jest.fn().mockRejectedValue('string error');

    await expect(
      fetchWithRetry(fetchFn, { maxRetries: 0, retryDelay: 10 })
    ).rejects.toThrow('string error');
  });

  it('uses default options when none provided', async () => {
    const fetchFn = jest.fn().mockResolvedValue('data');

    const result = await fetchWithRetry(fetchFn);

    expect(result).toBe('data');
    expect(fetchFn).toHaveBeenCalledTimes(1);
  });
});
