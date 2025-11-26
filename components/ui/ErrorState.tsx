'use client';

import { ExclamationTriangleIcon, ArrowPathIcon } from '@heroicons/react/24/outline';

interface ErrorStateProps {
  message: string;
  onRetry?: () => void;
  showSupport?: boolean;
  variant?: 'default' | 'compact' | 'inline';
  className?: string;
}

export default function ErrorState({
  message,
  onRetry,
  showSupport = false,
  variant = 'default',
  className = '',
}: ErrorStateProps) {
  if (variant === 'inline') {
    return (
      <div className={`flex items-center gap-2 text-sm text-red-600 ${className}`}>
        <ExclamationTriangleIcon className="h-4 w-4 flex-shrink-0" />
        <span>{message}</span>
        {onRetry && (
          <button
            onClick={onRetry}
            className="text-primary-600 hover:text-primary-700 underline"
          >
            Retry
          </button>
        )}
      </div>
    );
  }

  if (variant === 'compact') {
    return (
      <div className={`p-4 bg-red-50 border border-red-200 rounded-lg ${className}`}>
        <div className="flex items-start gap-3">
          <ExclamationTriangleIcon className="h-5 w-5 text-red-500 flex-shrink-0 mt-0.5" />
          <div className="flex-1">
            <p className="text-sm text-red-700">{message}</p>
            {onRetry && (
              <button
                onClick={onRetry}
                className="mt-2 flex items-center gap-1 text-sm text-red-600 hover:text-red-800 font-medium"
              >
                <ArrowPathIcon className="h-4 w-4" />
                Try again
              </button>
            )}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className={`p-8 text-center ${className}`}>
      <div className="mx-auto w-16 h-16 rounded-full bg-red-100 flex items-center justify-center mb-4">
        <ExclamationTriangleIcon className="h-8 w-8 text-red-500" />
      </div>

      <h3 className="text-lg font-semibold text-gray-900 mb-2">
        Something went wrong
      </h3>

      <p className="text-gray-600 mb-6 max-w-md mx-auto">
        {message}
      </p>

      <div className="flex flex-col sm:flex-row items-center justify-center gap-3">
        {onRetry && (
          <button
            onClick={onRetry}
            className="flex items-center gap-2 px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors"
          >
            <ArrowPathIcon className="h-5 w-5" />
            Try again
          </button>
        )}

        {showSupport && (
          <a
            href="mailto:support@investorcenter.ai"
            className="text-sm text-gray-500 hover:text-gray-700"
          >
            Contact support
          </a>
        )}
      </div>
    </div>
  );
}

// Loading state with retry capability
interface RetryLoadingStateProps {
  isLoading: boolean;
  isRetrying?: boolean;
  retryCount?: number;
  maxRetries?: number;
}

export function RetryLoadingState({
  isLoading,
  isRetrying = false,
  retryCount = 0,
  maxRetries = 3,
}: RetryLoadingStateProps) {
  if (!isLoading && !isRetrying) return null;

  return (
    <div className="flex items-center gap-2 text-sm text-gray-500">
      <div className="animate-spin h-4 w-4 border-2 border-primary-600 border-t-transparent rounded-full" />
      <span>
        {isRetrying
          ? `Retrying... (${retryCount}/${maxRetries})`
          : 'Loading...'}
      </span>
    </div>
  );
}
