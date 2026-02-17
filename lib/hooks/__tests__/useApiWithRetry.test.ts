import { renderHook, waitFor, act } from '@testing-library/react';
import { useApiWithRetry, fetchWithRetry } from '../useApiWithRetry';

// ──────────────────────────────────────────────────────────────
// fetchWithRetry (standalone utility) tests
// ──────────────────────────────────────────────────────────────
describe('fetchWithRetry', () => {
  it('succeeds on first try', async () => {
    const fetchFn = jest.fn().mockResolvedValue('success');

    const result = await fetchWithRetry(fetchFn);

    expect(result).toBe('success');
    expect(fetchFn).toHaveBeenCalledTimes(1);
  });

  it('retries on failure and succeeds', async () => {
    const fetchFn = jest
      .fn()
      .mockRejectedValueOnce(new Error('fail 1'))
      .mockResolvedValue('success after retry');

    const result = await fetchWithRetry(fetchFn, {
      maxRetries: 3,
      retryDelay: 10,
    });

    expect(result).toBe('success after retry');
    expect(fetchFn).toHaveBeenCalledTimes(2);
  });

  it('exhausts retries and throws last error', async () => {
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
    const fetchFn = jest
      .fn()
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
    const fetchFn = jest.fn().mockRejectedValue('string error');

    await expect(fetchWithRetry(fetchFn, { maxRetries: 0, retryDelay: 10 })).rejects.toThrow(
      'string error'
    );
  });

  it('uses default options when none provided', async () => {
    const fetchFn = jest.fn().mockResolvedValue('data');

    const result = await fetchWithRetry(fetchFn);

    expect(result).toBe('data');
    expect(fetchFn).toHaveBeenCalledTimes(1);
  });

  it('applies exponential backoff timing', async () => {
    jest.useFakeTimers();

    const fetchFn = jest
      .fn()
      .mockRejectedValueOnce(new Error('fail 1'))
      .mockRejectedValueOnce(new Error('fail 2'))
      .mockRejectedValueOnce(new Error('fail 3'))
      .mockResolvedValue('ok');

    const baseDelay = 1000;
    const promise = fetchWithRetry(fetchFn, {
      maxRetries: 3,
      retryDelay: baseDelay,
    });

    // First call happens immediately
    await jest.advanceTimersByTimeAsync(0);
    expect(fetchFn).toHaveBeenCalledTimes(1);

    // Retry 1: delay = 1000 * 2^0 = 1000ms
    await jest.advanceTimersByTimeAsync(1000);
    expect(fetchFn).toHaveBeenCalledTimes(2);

    // Retry 2: delay = 1000 * 2^1 = 2000ms
    await jest.advanceTimersByTimeAsync(2000);
    expect(fetchFn).toHaveBeenCalledTimes(3);

    // Retry 3: delay = 1000 * 2^2 = 4000ms
    await jest.advanceTimersByTimeAsync(4000);
    expect(fetchFn).toHaveBeenCalledTimes(4);

    const result = await promise;
    expect(result).toBe('ok');

    jest.useRealTimers();
  });
});

// ──────────────────────────────────────────────────────────────
// useApiWithRetry (React hook) tests
// ──────────────────────────────────────────────────────────────
describe('useApiWithRetry', () => {
  it('returns initial idle state', () => {
    const apiCall = jest.fn().mockResolvedValue('data');

    const { result } = renderHook(() => useApiWithRetry(apiCall));

    expect(result.current.data).toBeNull();
    expect(result.current.error).toBeNull();
    expect(result.current.loading).toBe(false);
    expect(result.current.isRetrying).toBe(false);
    expect(result.current.retryCount).toBe(0);
    expect(typeof result.current.execute).toBe('function');
    expect(typeof result.current.retry).toBe('function');
    expect(typeof result.current.reset).toBe('function');
  });

  it('sets loading state during execution', async () => {
    let resolveFn: (val: string) => void;
    const apiCall = jest.fn(
      () =>
        new Promise<string>((resolve) => {
          resolveFn = resolve;
        })
    );

    const { result } = renderHook(() => useApiWithRetry(apiCall));

    // Start executing
    let executePromise: Promise<string | null>;
    act(() => {
      executePromise = result.current.execute();
    });

    // Should be loading
    expect(result.current.loading).toBe(true);
    expect(result.current.error).toBeNull();

    // Resolve the promise
    await act(async () => {
      resolveFn!('success');
      await executePromise!;
    });

    expect(result.current.loading).toBe(false);
    expect(result.current.data).toBe('success');
  });

  it('returns data on successful execute', async () => {
    const apiCall = jest.fn().mockResolvedValue({ id: 1, name: 'Test' });

    const { result } = renderHook(() => useApiWithRetry(apiCall));

    await act(async () => {
      const data = await result.current.execute();
      expect(data).toEqual({ id: 1, name: 'Test' });
    });

    expect(result.current.data).toEqual({ id: 1, name: 'Test' });
    expect(result.current.error).toBeNull();
    expect(result.current.loading).toBe(false);
  });

  it('calls onSuccess callback on successful execution', async () => {
    const apiCall = jest.fn().mockResolvedValue('result');
    const onSuccess = jest.fn();

    const { result } = renderHook(() => useApiWithRetry(apiCall, { onSuccess }));

    await act(async () => {
      await result.current.execute();
    });

    expect(onSuccess).toHaveBeenCalledWith('result');
  });

  it('retries on network error and eventually succeeds', async () => {
    const apiCall = jest
      .fn()
      .mockRejectedValueOnce(new Error('Network error'))
      .mockRejectedValueOnce(new Error('Timeout'))
      .mockResolvedValue('recovered');

    const onRetry = jest.fn();

    const { result } = renderHook(() =>
      useApiWithRetry(apiCall, {
        maxRetries: 3,
        retryDelay: 10,
        onRetry,
      })
    );

    await act(async () => {
      const data = await result.current.execute();
      expect(data).toBe('recovered');
    });

    expect(apiCall).toHaveBeenCalledTimes(3);
    expect(onRetry).toHaveBeenCalledTimes(2);
    expect(onRetry).toHaveBeenCalledWith(1, expect.any(Error));
    expect(onRetry).toHaveBeenCalledWith(2, expect.any(Error));
    expect(result.current.data).toBe('recovered');
    expect(result.current.error).toBeNull();
    expect(result.current.isRetrying).toBe(false);
  });

  it('sets error state when max retries exceeded', async () => {
    const apiCall = jest.fn().mockRejectedValue(new Error('Server down'));
    const onError = jest.fn();

    const { result } = renderHook(() =>
      useApiWithRetry(apiCall, {
        maxRetries: 2,
        retryDelay: 10,
        onError,
      })
    );

    await act(async () => {
      const data = await result.current.execute();
      expect(data).toBeNull();
    });

    // 1 initial + 2 retries = 3 calls
    expect(apiCall).toHaveBeenCalledTimes(3);
    expect(result.current.error).toBeInstanceOf(Error);
    expect(result.current.error?.message).toBe('Server down');
    expect(result.current.data).toBeNull();
    expect(result.current.loading).toBe(false);
    expect(result.current.isRetrying).toBe(false);
    expect(onError).toHaveBeenCalledWith(
      expect.objectContaining({
        message: 'Server down',
      })
    );
  });

  it('wraps non-Error thrown values into Error objects', async () => {
    const apiCall = jest.fn().mockRejectedValue('string rejection');

    const { result } = renderHook(() =>
      useApiWithRetry(apiCall, { maxRetries: 0, retryDelay: 10 })
    );

    await act(async () => {
      await result.current.execute();
    });

    expect(result.current.error).toBeInstanceOf(Error);
    expect(result.current.error?.message).toBe('string rejection');
  });

  it('does not retry on AbortError', async () => {
    const abortError = new DOMException('Aborted', 'AbortError');
    const apiCall = jest.fn().mockRejectedValue(abortError);
    const onRetry = jest.fn();

    const { result } = renderHook(() =>
      useApiWithRetry(apiCall, {
        maxRetries: 3,
        retryDelay: 10,
        onRetry,
      })
    );

    await act(async () => {
      await result.current.execute();
    });

    // Should only be called once (no retries for abort)
    expect(apiCall).toHaveBeenCalledTimes(1);
    expect(onRetry).not.toHaveBeenCalled();
  });

  it('reset clears all state', async () => {
    const apiCall = jest.fn().mockResolvedValue('data');

    const { result } = renderHook(() => useApiWithRetry(apiCall));

    // First, execute to get data
    await act(async () => {
      await result.current.execute();
    });

    expect(result.current.data).toBe('data');

    // Now reset
    act(() => {
      result.current.reset();
    });

    expect(result.current.data).toBeNull();
    expect(result.current.error).toBeNull();
    expect(result.current.loading).toBe(false);
    expect(result.current.isRetrying).toBe(false);
    expect(result.current.retryCount).toBe(0);
  });

  it('reset after error clears error state', async () => {
    const apiCall = jest.fn().mockRejectedValue(new Error('fail'));

    const { result } = renderHook(() =>
      useApiWithRetry(apiCall, { maxRetries: 0, retryDelay: 10 })
    );

    await act(async () => {
      await result.current.execute();
    });

    expect(result.current.error).not.toBeNull();

    act(() => {
      result.current.reset();
    });

    expect(result.current.error).toBeNull();
    expect(result.current.data).toBeNull();
  });

  it('retry calls execute again', async () => {
    const apiCall = jest
      .fn()
      .mockRejectedValueOnce(new Error('first failure'))
      .mockResolvedValue('second success');

    const { result } = renderHook(() =>
      useApiWithRetry(apiCall, { maxRetries: 0, retryDelay: 10 })
    );

    // First call fails
    await act(async () => {
      await result.current.execute();
    });

    expect(result.current.error).not.toBeNull();

    // Retry succeeds
    await act(async () => {
      const data = await result.current.retry();
      expect(data).toBe('second success');
    });

    expect(result.current.data).toBe('second success');
    expect(result.current.error).toBeNull();
  });

  it('cancels previous request when execute is called again', async () => {
    let callCount = 0;
    const apiCall = jest.fn(() => {
      callCount++;
      const currentCall = callCount;
      return new Promise<string>((resolve) => {
        setTimeout(() => resolve(`result-${currentCall}`), 100);
      });
    });

    const { result } = renderHook(() =>
      useApiWithRetry(apiCall, { maxRetries: 0, retryDelay: 10 })
    );

    // Start first execution but don't await
    act(() => {
      result.current.execute();
    });

    // Immediately start second execution
    await act(async () => {
      const data = await result.current.execute();
      // The second call should be the one that resolves
      expect(data).toBe('result-2');
    });

    expect(apiCall).toHaveBeenCalledTimes(2);
  });

  it('cleans up abort controller on unmount', async () => {
    const apiCall = jest.fn().mockResolvedValue('data');

    const { result, unmount } = renderHook(() => useApiWithRetry(apiCall));

    await act(async () => {
      await result.current.execute();
    });

    // Should not throw on unmount
    unmount();
  });

  it('uses default maxRetries of 3', async () => {
    const apiCall = jest.fn().mockRejectedValue(new Error('fail'));

    const { result } = renderHook(() => useApiWithRetry(apiCall, { retryDelay: 10 }));

    await act(async () => {
      await result.current.execute();
    });

    // 1 initial + 3 retries (default) = 4 total calls
    expect(apiCall).toHaveBeenCalledTimes(4);
  });

  it('handles zero maxRetries (no retries)', async () => {
    const apiCall = jest.fn().mockRejectedValue(new Error('fail'));

    const { result } = renderHook(() =>
      useApiWithRetry(apiCall, { maxRetries: 0, retryDelay: 10 })
    );

    await act(async () => {
      await result.current.execute();
    });

    expect(apiCall).toHaveBeenCalledTimes(1);
    expect(result.current.error?.message).toBe('fail');
  });

  it('tracks retryCount during retries', async () => {
    let resolveSecond: (val: string) => void;
    const apiCall = jest
      .fn()
      .mockRejectedValueOnce(new Error('fail'))
      .mockImplementation(
        () =>
          new Promise<string>((resolve) => {
            resolveSecond = resolve;
          })
      );

    const retryCounts: number[] = [];

    const { result } = renderHook(() =>
      useApiWithRetry(apiCall, {
        maxRetries: 2,
        retryDelay: 10,
        onRetry: (attempt) => {
          retryCounts.push(attempt);
        },
      })
    );

    const executePromise = act(async () => {
      const promise = result.current.execute();
      // Let the first attempt fail and retry start
      await new Promise((r) => setTimeout(r, 50));
      resolveSecond!('done');
      return promise;
    });

    await executePromise;

    expect(retryCounts).toEqual([1]);
    expect(result.current.data).toBe('done');
  });
});
