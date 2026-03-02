'use client';

import React, { Component, type ReactNode } from 'react';
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline';

interface Props {
  children: ReactNode;
  /** Widget name for error reporting */
  widgetName?: string;
  /** Optional fallback UI */
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

/**
 * Error boundary for individual home page widgets.
 * Prevents one broken widget from crashing the entire page.
 * Shows a compact error message with retry option.
 */
export default class WidgetErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, error: null };

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error(`[WidgetErrorBoundary] ${this.props.widgetName || 'Unknown'}:`, error, errorInfo);
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <div
          className="bg-ic-surface rounded-lg border border-ic-border p-6"
          style={{ boxShadow: 'var(--ic-shadow-card)' }}
        >
          <div className="flex items-start gap-3">
            <ExclamationTriangleIcon className="h-5 w-5 text-ic-negative flex-shrink-0 mt-0.5" />
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-ic-text-primary">
                {this.props.widgetName || 'Widget'} failed to load
              </p>
              <p className="text-xs text-ic-text-muted mt-1">
                {this.state.error?.message || 'An unexpected error occurred.'}
              </p>
              <button
                onClick={this.handleRetry}
                className="mt-3 text-xs text-ic-blue hover:text-ic-blue-hover font-medium transition-colors"
              >
                Try again
              </button>
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
