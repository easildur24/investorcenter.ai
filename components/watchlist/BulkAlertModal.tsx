'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { alertAPI, BulkCreateAlertRequest, ALERT_TYPES, ALERT_FREQUENCIES } from '@/lib/api/alerts';

// ---------------------------------------------------------------------------
// Bulk-create alert types supported by the backend binding validation.
// Event-based types (news, earnings, etc.) don't have a threshold and are
// excluded from the bulk flow for now.
// ---------------------------------------------------------------------------

const BULK_ALERT_TYPES = {
  price_above: ALERT_TYPES.price_above,
  price_below: ALERT_TYPES.price_below,
  volume_above: ALERT_TYPES.volume_above,
  volume_spike: ALERT_TYPES.volume_spike,
} as const;

type BulkAlertType = keyof typeof BULK_ALERT_TYPES;

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface BulkAlertModalProps {
  watchListId: string;
  watchListName: string;
  tickerCount: number;
  onClose: () => void;
  onSuccess: () => void;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function BulkAlertModal({
  watchListId,
  watchListName,
  tickerCount,
  onClose,
  onSuccess,
}: BulkAlertModalProps) {
  const [alertType, setAlertType] = useState<BulkAlertType>('price_above');
  const [threshold, setThreshold] = useState('');
  const [frequency, setFrequency] = useState<'once' | 'daily' | 'always'>('daily');
  const [notifyEmail, setNotifyEmail] = useState(true);
  const [notifyInApp, setNotifyInApp] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Close on Escape
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [onClose]);

  const buildConditions = useCallback((): Record<string, unknown> => {
    const t = parseFloat(threshold);
    switch (alertType) {
      case 'price_above':
      case 'price_below':
      case 'volume_above':
        return { threshold: t };
      case 'volume_spike':
        return { volume_multiplier: t, baseline: 'avg_30d' };
      default:
        return { threshold: t };
    }
  }, [alertType, threshold]);

  const getThresholdLabel = (): string => {
    switch (alertType) {
      case 'price_above':
        return 'Price threshold ($)';
      case 'price_below':
        return 'Price threshold ($)';
      case 'volume_above':
        return 'Volume threshold';
      case 'volume_spike':
        return 'Volume multiplier (e.g., 2 = 2x average)';
      default:
        return 'Threshold';
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!threshold) {
      setError('Please enter a threshold value');
      return;
    }

    const t = parseFloat(threshold);
    if (isNaN(t) || t <= 0) {
      setError('Please enter a valid positive number');
      return;
    }

    const req: BulkCreateAlertRequest = {
      watch_list_id: watchListId,
      alert_type: alertType,
      conditions: buildConditions(),
      frequency,
      notify_email: notifyEmail,
      notify_in_app: notifyInApp,
    };

    try {
      setLoading(true);
      const result = await alertAPI.bulkCreateAlerts(req);

      // Build success message
      const parts: string[] = [];
      if (result.created > 0)
        parts.push(`Created ${result.created} alert${result.created !== 1 ? 's' : ''}`);
      if (result.skipped > 0) parts.push(`skipped ${result.skipped} (already have alerts)`);
      const message = parts.join(', ') || 'No changes made';

      // Use window.alert as a simple feedback mechanism; the parent will also
      // show a toast via onSuccess callback.
      window.alert(message);
      onSuccess();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to create alerts';
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      <div className="flex items-center justify-center min-h-screen px-4 pt-4 pb-20 text-center sm:block sm:p-0">
        {/* Background overlay */}
        {/* eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-static-element-interactions -- overlay dismiss is a convenience; Escape key also closes */}
        <div
          className="fixed inset-0 transition-opacity bg-ic-bg-tertiary bg-opacity-75"
          onClick={onClose}
        />

        {/* Modal panel */}
        <div className="inline-block align-bottom bg-ic-surface rounded-lg text-left overflow-hidden border border-ic-border transform transition-all sm:my-8 sm:align-middle sm:max-w-md sm:w-full">
          <div className="bg-ic-surface px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="text-lg font-semibold text-ic-text-primary">
                  Set Alert for All Tickers
                </h3>
                <p className="text-sm text-ic-text-dim mt-1">
                  Apply to {tickerCount} ticker{tickerCount !== 1 ? 's' : ''} in &ldquo;
                  {watchListName}&rdquo;
                </p>
              </div>
              <button
                onClick={onClose}
                className="text-ic-text-dim hover:text-ic-text-muted p-1"
                aria-label="Close"
              >
                <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M6 18L18 6M6 6l12 12"
                  />
                </svg>
              </button>
            </div>

            {error && (
              <div className="mb-4 p-3 bg-red-500/10 border border-red-500/30 rounded-md">
                <p className="text-sm text-ic-negative">{error}</p>
              </div>
            )}

            <form onSubmit={handleSubmit} className="space-y-4">
              {/* Alert Type */}
              <div>
                <label
                  htmlFor="bulk-alert-type"
                  className="block text-sm font-medium text-ic-text-secondary mb-1"
                >
                  Alert Type
                </label>
                <select
                  id="bulk-alert-type"
                  value={alertType}
                  onChange={(e) => setAlertType(e.target.value as BulkAlertType)}
                  className="w-full px-3 py-2 bg-ic-bg-secondary border border-ic-border rounded-md text-ic-text-primary focus:ring-ic-blue focus:border-ic-blue"
                >
                  {Object.entries(BULK_ALERT_TYPES).map(([key, info]) => (
                    <option key={key} value={key}>
                      {info.icon} {info.label}
                    </option>
                  ))}
                </select>
              </div>

              {/* Threshold */}
              <div>
                <label
                  htmlFor="bulk-alert-threshold"
                  className="block text-sm font-medium text-ic-text-secondary mb-1"
                >
                  {getThresholdLabel()} <span className="text-ic-negative">*</span>
                </label>
                <input
                  id="bulk-alert-threshold"
                  type="number"
                  step={alertType.includes('price') ? '0.01' : '1'}
                  min="0"
                  value={threshold}
                  onChange={(e) => setThreshold(e.target.value)}
                  placeholder={
                    alertType.includes('price')
                      ? '150.00'
                      : alertType === 'volume_spike'
                        ? '2'
                        : '1000000'
                  }
                  className="w-full px-3 py-2 bg-ic-bg-secondary border border-ic-border rounded-md text-ic-text-primary focus:ring-ic-blue focus:border-ic-blue"
                  required
                />
                {alertType === 'volume_spike' && (
                  <p className="text-xs text-ic-text-dim mt-1">
                    Triggers when volume exceeds this multiple of the 30-day average
                  </p>
                )}
              </div>

              {/* Frequency */}
              <div>
                <label
                  htmlFor="bulk-alert-frequency"
                  className="block text-sm font-medium text-ic-text-secondary mb-1"
                >
                  Frequency
                </label>
                <select
                  id="bulk-alert-frequency"
                  value={frequency}
                  onChange={(e) => setFrequency(e.target.value as 'once' | 'daily' | 'always')}
                  className="w-full px-3 py-2 bg-ic-bg-secondary border border-ic-border rounded-md text-ic-text-primary focus:ring-ic-blue focus:border-ic-blue"
                >
                  {Object.entries(ALERT_FREQUENCIES).map(([key, info]) => (
                    <option key={key} value={key}>
                      {info.label} &mdash; {info.description}
                    </option>
                  ))}
                </select>
              </div>

              {/* Notifications */}
              <div className="space-y-2">
                <span className="block text-sm font-medium text-ic-text-secondary">
                  Notifications
                </span>
                <div className="flex items-center gap-4">
                  <label className="flex items-center gap-2 text-sm text-ic-text-secondary cursor-pointer">
                    <input
                      type="checkbox"
                      checked={notifyEmail}
                      onChange={(e) => setNotifyEmail(e.target.checked)}
                      className="h-4 w-4 rounded border-ic-border text-ic-blue focus:ring-ic-blue"
                    />
                    Email
                  </label>
                  <label className="flex items-center gap-2 text-sm text-ic-text-secondary cursor-pointer">
                    <input
                      type="checkbox"
                      checked={notifyInApp}
                      onChange={(e) => setNotifyInApp(e.target.checked)}
                      className="h-4 w-4 rounded border-ic-border text-ic-blue focus:ring-ic-blue"
                    />
                    In-App
                  </label>
                </div>
              </div>

              {/* Info note */}
              <div className="p-3 bg-ic-blue/10 border border-ic-blue/20 rounded-md text-xs text-ic-text-secondary">
                Tickers that already have an active alert will be skipped. Alert names are
                auto-generated (e.g., &ldquo;AAPL Price Above&rdquo;).
              </div>

              {/* Actions */}
              <div className="flex items-center justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={onClose}
                  className="px-4 py-2 text-sm font-medium text-ic-text-secondary bg-ic-surface border border-ic-border rounded-md hover:bg-ic-surface-hover"
                  disabled={loading}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 text-sm font-medium text-ic-text-primary bg-ic-blue rounded-md hover:bg-ic-blue-hover disabled:opacity-50 disabled:cursor-not-allowed"
                  disabled={loading}
                >
                  {loading ? 'Creating...' : `Create Alerts (${tickerCount})`}
                </button>
              </div>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
}
