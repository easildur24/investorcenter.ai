'use client';

import React, { useState, useMemo, useEffect, useCallback } from 'react';
import {
  AlertRuleWithDetails,
  UpdateAlertRequest,
  ALERT_TYPES,
  ALERT_FREQUENCIES,
} from '@/lib/api/alerts';

// ---------------------------------------------------------------------------
// Helpers (duplicated from AlertQuickPanel — small, pure utility functions)
// ---------------------------------------------------------------------------

function isPriceType(t: string) {
  return t === 'price_above' || t === 'price_below' || t === 'price_change';
}

function isVolumeType(t: string) {
  return t === 'volume_above' || t === 'volume_spike';
}

function needsThreshold(t: string) {
  return isPriceType(t) || isVolumeType(t);
}

/** Returns a human-readable label for the threshold input based on alert type. */
function getThresholdLabel(alertType: string): string {
  if (isPriceType(alertType)) return 'Price threshold ($)';
  if (alertType === 'volume_spike') return 'Volume multiplier (e.g., 2 = 2x average)';
  if (alertType === 'volume_above') return 'Volume threshold';
  return 'Threshold';
}

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface EditAlertModalProps {
  alert: AlertRuleWithDetails;
  onSave: (alertId: string, req: UpdateAlertRequest) => Promise<void>;
  onClose: () => void;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function EditAlertModal({ alert, onSave, onClose }: EditAlertModalProps) {
  // ── Parse existing conditions ───────────────────────────────────────
  const parsedConditions = useMemo(() => {
    try {
      return typeof alert.conditions === 'string' ? JSON.parse(alert.conditions) : alert.conditions;
    } catch {
      return {};
    }
  }, [alert.conditions]);

  // ── Form state (seeded from alert) ──────────────────────────────────
  const alertType = alert.alert_type; // immutable — display only
  const [threshold, setThreshold] = useState(
    () =>
      parsedConditions.threshold?.toString() ?? parsedConditions.volume_multiplier?.toString() ?? ''
  );
  const [frequency, setFrequency] = useState<'once' | 'daily' | 'always'>(alert.frequency);
  const [name, setName] = useState(alert.name);
  const [notifyEmail, setNotifyEmail] = useState(alert.notify_email);
  const [notifyInApp, setNotifyInApp] = useState(alert.notify_in_app);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // ── Auto-generated name (fallback when name field is cleared) ─────
  const autoName = useMemo(() => {
    const typeInfo = ALERT_TYPES[alertType as keyof typeof ALERT_TYPES];
    const label = typeInfo?.label ?? alertType;
    if (needsThreshold(alertType) && threshold) {
      const prefix = isPriceType(alertType) ? '$' : '';
      return `${alert.symbol} ${label} ${prefix}${threshold}`;
    }
    return `${alert.symbol} ${label}`;
  }, [alert.symbol, alertType, threshold]);

  // ── Close on Escape ─────────────────────────────────────────────────
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [onClose]);

  // ── Save handler ────────────────────────────────────────────────────
  const handleSave = useCallback(async () => {
    if (needsThreshold(alertType)) {
      const val = parseFloat(threshold);
      if (isNaN(val) || val <= 0) {
        setError('Please enter a valid threshold value');
        return;
      }
    }

    setSaving(true);
    setError(null);

    // Use the user-entered name, falling back to auto-generated name
    // when the name field is empty (e.g. the user cleared it).
    const finalName = name.trim() || autoName;
    const conditions = needsThreshold(alertType)
      ? alertType === 'volume_spike'
        ? { volume_multiplier: parseFloat(threshold) }
        : { threshold: parseFloat(threshold) }
      : {};

    try {
      await onSave(alert.id, {
        name: finalName,
        conditions,
        frequency,
        notify_email: notifyEmail,
        notify_in_app: notifyInApp,
      });
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to update alert');
    } finally {
      setSaving(false);
    }
  }, [alert.id, alertType, autoName, frequency, name, notifyEmail, notifyInApp, onSave, threshold]);

  // ── Render ──────────────────────────────────────────────────────────
  const typeInfo = ALERT_TYPES[alertType as keyof typeof ALERT_TYPES];

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      <div className="flex items-center justify-center min-h-screen px-4 pt-4 pb-20 text-center sm:block sm:p-0">
        {/* Background overlay — click to dismiss (keyboard dismiss via Escape) */}
        {/* eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-static-element-interactions */}
        <div
          className="fixed inset-0 transition-opacity bg-ic-bg-tertiary bg-opacity-75"
          data-testid="modal-overlay"
          onClick={onClose}
        />

        {/* Modal panel */}
        <div className="inline-block align-bottom bg-ic-surface rounded-lg text-left overflow-hidden border border-ic-border transform transition-all sm:my-8 sm:align-middle sm:max-w-md sm:w-full">
          <div className="bg-ic-surface px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
            {/* Header */}
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="text-lg font-semibold text-ic-text-primary">Edit Alert</h3>
                <p className="text-sm text-ic-text-dim mt-1">
                  {alert.symbol} &mdash; {typeInfo?.label ?? alertType}
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

            <form
              onSubmit={(e) => {
                e.preventDefault();
                handleSave();
              }}
              className="space-y-4"
            >
              {/* Alert Type (read-only) */}
              <div>
                <label
                  htmlFor="edit-alert-type"
                  className="block text-sm font-medium text-ic-text-secondary mb-1"
                >
                  Alert Type
                </label>
                <select
                  id="edit-alert-type"
                  value={alertType}
                  disabled
                  className="w-full px-3 py-2 bg-ic-bg-secondary border border-ic-border rounded-md text-ic-text-primary opacity-60 cursor-not-allowed"
                >
                  <option value={alertType}>
                    {typeInfo?.icon} {typeInfo?.label ?? alertType}
                  </option>
                </select>
                <p className="text-xs text-ic-text-dim mt-1">
                  To change type, delete and recreate the alert.
                </p>
              </div>

              {/* Threshold (only for price/volume types) */}
              {needsThreshold(alertType) && (
                <div>
                  <label
                    htmlFor="edit-alert-threshold"
                    className="block text-sm font-medium text-ic-text-secondary mb-1"
                  >
                    {getThresholdLabel(alertType)} <span className="text-ic-negative">*</span>
                  </label>
                  <input
                    id="edit-alert-threshold"
                    type="number"
                    step={isPriceType(alertType) ? '0.01' : '1'}
                    min="0"
                    value={threshold}
                    onChange={(e) => setThreshold(e.target.value)}
                    placeholder={
                      isPriceType(alertType)
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
              )}

              {/* Frequency */}
              <div>
                <label
                  htmlFor="edit-alert-frequency"
                  className="block text-sm font-medium text-ic-text-secondary mb-1"
                >
                  Frequency
                </label>
                <select
                  id="edit-alert-frequency"
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

              {/* Name */}
              <div>
                <label
                  htmlFor="edit-alert-name"
                  className="block text-sm font-medium text-ic-text-secondary mb-1"
                >
                  Name
                </label>
                <input
                  id="edit-alert-name"
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder={autoName}
                  className="w-full px-3 py-2 bg-ic-bg-secondary border border-ic-border rounded-md text-ic-text-primary focus:ring-ic-blue focus:border-ic-blue"
                />
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

              {/* Actions */}
              <div className="flex items-center justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={onClose}
                  className="px-4 py-2 text-sm font-medium text-ic-text-secondary bg-ic-surface border border-ic-border rounded-md hover:bg-ic-surface-hover"
                  disabled={saving}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 text-sm font-medium text-ic-text-primary bg-ic-blue rounded-md hover:bg-ic-blue-hover disabled:opacity-50 disabled:cursor-not-allowed"
                  disabled={saving}
                >
                  {saving ? 'Saving...' : 'Update Alert'}
                </button>
              </div>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
}
