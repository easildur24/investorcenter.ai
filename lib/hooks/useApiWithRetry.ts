import { useState, useCallback, useRef, useEffect } from 'react';

interface UseApiWithRetryOptions {
  maxRetries?: number;
  retryDelay?: number; // Base delay in ms (will use exponential backoff)
  onRetry?: (attempt: number, error: Error) => void;
  onSuccess?: (data: any) => void;
  onError?: (error: Error) => void;
}

interface UseApiWithRetryResult<T> {
  data: T | null;
  error: Error | null;
  loading: boolean;
  isRetrying: boolean;
  retryCount: number;
  execute: () => Promise<T | null>;
  retry: () => Promise<T | null>;
  reset: () => void;
}

export function useApiWithRetry<T>(
  apiCall: () => Promise<T>,
  options: UseApiWithRetryOptions = {}
): UseApiWithRetryResult<T> {
  const { maxRetries = 3, retryDelay = 1000, onRetry, onSuccess, onError } = options;

  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(false);
  const [isRetrying, setIsRetrying] = useState(false);
  const [retryCount, setRetryCount] = useState(0);

  const abortControllerRef = useRef<AbortController | null>(null);

  const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

  const execute = useCallback(async (): Promise<T | null> => {
    // Cancel any ongoing request
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }

    abortControllerRef.current = new AbortController();
    setLoading(true);
    setError(null);
    setRetryCount(0);
    setIsRetrying(false);

    let lastError: Error | null = null;

    for (let attempt = 0; attempt <= maxRetries; attempt++) {
      try {
        if (abortControllerRef.current?.signal.aborted) {
          break;
        }

        if (attempt > 0) {
          setIsRetrying(true);
          setRetryCount(attempt);
          onRetry?.(attempt, lastError!);

          // Exponential backoff: 1s, 2s, 4s, etc.
          const delay = retryDelay * Math.pow(2, attempt - 1);
          await sleep(delay);
        }

        const result = await apiCall();
        setData(result);
        setLoading(false);
        setIsRetrying(false);
        onSuccess?.(result);
        return result;
      } catch (err) {
        lastError = err instanceof Error ? err : new Error(String(err));

        // Don't retry on abort
        if (lastError.name === 'AbortError') {
          break;
        }
      }
    }

    // All retries failed
    setError(lastError);
    setLoading(false);
    setIsRetrying(false);
    onError?.(lastError!);
    return null;
  }, [apiCall, maxRetries, retryDelay, onRetry, onSuccess, onError]);

  const retry = useCallback(async (): Promise<T | null> => {
    return execute();
  }, [execute]);

  const reset = useCallback(() => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }
    setData(null);
    setError(null);
    setLoading(false);
    setIsRetrying(false);
    setRetryCount(0);
  }, []);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
    };
  }, []);

  return {
    data,
    error,
    loading,
    isRetrying,
    retryCount,
    execute,
    retry,
    reset,
  };
}

// Fetch wrapper with retry logic for use without hook
export async function fetchWithRetry<T>(
  fetchFn: () => Promise<T>,
  options: {
    maxRetries?: number;
    retryDelay?: number;
    onRetry?: (attempt: number) => void;
  } = {}
): Promise<T> {
  const { maxRetries = 3, retryDelay = 1000, onRetry } = options;

  let lastError: Error;

  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    try {
      if (attempt > 0) {
        onRetry?.(attempt);
        // Exponential backoff
        await new Promise((resolve) => setTimeout(resolve, retryDelay * Math.pow(2, attempt - 1)));
      }
      return await fetchFn();
    } catch (err) {
      lastError = err instanceof Error ? err : new Error(String(err));
    }
  }

  throw lastError!;
}
