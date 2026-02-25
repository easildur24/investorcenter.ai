'use client';

import React, { useState, useMemo } from 'react';
import {
  AlertRuleWithDetails,
  ALERT_TYPES,
  ALERT_FREQUENCIES,
  CreateAlertRequest,
  UpdateAlertRequest,
} from '@/lib/api/alerts';

// Only the alert types accepted by the backend binding validation
const ALLOWED_ALERT_TYPES = [
  'price_above',
  'price_below',
  'price_change',
  'volume_above',
  'volume_spike',
  'news',
  'earnings',
  'sec_filing',
] as const;

type AllowedAlertType = (typeof ALLOWED_ALERT_TYPES)[number];

function isPriceType(t: string) {
  return t === 'price_above' || t === 'price_below' || t === 'price_change';
}

function isVolumeType(t: string) {
  return t === 'volume_above' || t === 'volume_spike';
}

function needsThreshold(t: string) {
  return isPriceType(t) || isVolumeType(t);
}

function formatConditions(alertType: string, conditions: any): string {
  try {
    const cond = typeof conditions === 'string' ? JSON.parse(conditions) : conditions;
    if (isPriceType(alertType) && cond.threshold != null) {
      return `$${Number(cond.threshold).toFixed(2)}`;
    }
    if (alertType === 'volume_above' && cond.threshold != null) {
      return formatVolume(cond.threshold);
    }
    if (alertType === 'volume_spike' && cond.volume_multiplier != null) {
      return `${cond.volume_multiplier}x avg`;
    }
    return 'Configured';
  } catch {
    return 'Configured';
  }
}

function formatVolume(vol: number): string {
  if (vol >= 1e9) return `${(vol / 1e9).toFixed(1)}B`;
  if (vol >= 1e6) return `${(vol / 1e6).toFixed(1)}M`;
  if (vol >= 1e3) return `${(vol / 1e3).toFixed(0)}K`;
  return vol.toLocaleString();
}

interface InlineAlertSectionProps {
  watchListId: string;
  symbol: string;
  currentPrice?: number;
  existingAlert?: AlertRuleWithDetails;
  onCreate: (req: CreateAlertRequest) => Promise<any>;
  onUpdate: (alertId: string, req: UpdateAlertRequest) => Promise<void>;
  onDelete: (alertId: string, symbol: string) => Promise<void>;
}

export default function InlineAlertSection({
  watchListId,
  symbol,
  currentPrice,
  existingAlert,
  onCreate,
  onUpdate,
  onDelete,
}: InlineAlertSectionProps) {
  const [formOpen, setFormOpen] = useState(false);
  const [isEditing, setIsEditing] = useState(false);

  // Form state
  const [alertType, setAlertType] = useState<AllowedAlertType>('price_above');
  const [threshold, setThreshold] = useState('');
  const [frequency, setFrequency] = useState<'once' | 'daily' | 'always'>('daily');
  const [name, setName] = useState('');
  const [nameManuallyEdited, setNameManuallyEdited] = useState(false);
  const [notifyEmail, setNotifyEmail] = useState(true);
  const [notifyInApp, setNotifyInApp] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const autoName = useMemo(() => {
    const typeInfo = ALERT_TYPES[alertType as keyof typeof ALERT_TYPES];
    const label = typeInfo?.label ?? alertType;
    if (needsThreshold(alertType) && threshold) {
      const prefix = isPriceType(alertType) ? '$' : '';
      return `${symbol} ${label} ${prefix}${threshold}`;
    }
    return `${symbol} ${label}`;
  }, [symbol, alertType, threshold]);

  const openCreateForm = () => {
    setIsEditing(false);
    setAlertType('price_above');
    setThreshold(currentPrice ? Math.ceil(currentPrice).toString() : '');
    setFrequency('daily');
    setName('');
    setNameManuallyEdited(false);
    setNotifyEmail(true);
    setNotifyInApp(true);
    setError(null);
    setFormOpen(true);
  };

  const openEditForm = () => {
    if (!existingAlert) return;
    setIsEditing(true);
    setAlertType(existingAlert.alert_type as AllowedAlertType);
    const cond =
      typeof existingAlert.conditions === 'string'
        ? JSON.parse(existingAlert.conditions)
        : existingAlert.conditions;
    setThreshold(cond.threshold?.toString() ?? cond.volume_multiplier?.toString() ?? '');
    setFrequency(existingAlert.frequency);
    setName(existingAlert.name);
    setNameManuallyEdited(true);
    setNotifyEmail(existingAlert.notify_email);
    setNotifyInApp(existingAlert.notify_in_app);
    setError(null);
    setFormOpen(true);
  };

  const handleSave = async () => {
    if (needsThreshold(alertType)) {
      const val = parseFloat(threshold);
      if (isNaN(val) || val <= 0) {
        setError('Please enter a valid threshold value');
        return;
      }
    }

    setSaving(true);
    setError(null);

    const finalName = nameManuallyEdited && name ? name : autoName;
    const conditions = needsThreshold(alertType)
      ? alertType === 'volume_spike'
        ? { volume_multiplier: parseFloat(threshold) }
        : { threshold: parseFloat(threshold) }
      : {};

    try {
      if (isEditing && existingAlert) {
        await onUpdate(existingAlert.id, {
          name: finalName,
          conditions,
          frequency,
          notify_email: notifyEmail,
          notify_in_app: notifyInApp,
        });
      } else {
        await onCreate({
          watch_list_id: watchListId,
          symbol,
          alert_type: alertType,
          conditions,
          name: finalName,
          frequency,
          notify_email: notifyEmail,
          notify_in_app: notifyInApp,
        });
      }
      setFormOpen(false);
    } catch (err: any) {
      setError(err.message || 'Failed to save alert');
    } finally {
      setSaving(false);
    }
  };

  const handleToggleActive = async () => {
    if (!existingAlert) return;
    try {
      await onUpdate(existingAlert.id, { is_active: !existingAlert.is_active });
    } catch {
      // Swallow — toast is shown by parent
    }
  };

  const handleDelete = async () => {
    if (!existingAlert) return;
    if (!confirm(`Delete alert "${existingAlert.name}"?`)) return;
    try {
      await onDelete(existingAlert.id, symbol);
    } catch {
      // Swallow — toast is shown by parent
    }
  };

  // ── State A: No alert, form closed ────────────────────────────────
  if (!existingAlert && !formOpen) {
    return (
      <div className="flex items-center gap-2">
        <label className="block text-xs font-medium text-ic-text-secondary">Alert</label>
        <button
          type="button"
          onClick={openCreateForm}
          className="inline-flex items-center gap-1.5 text-xs text-ic-text-dim hover:text-ic-blue transition-colors"
        >
          <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1.5}
              d="M14.857 17.082a23.848 23.848 0 005.454-1.31A8.967 8.967 0 0118 9.75v-.7V9A6 6 0 006 9v.75a8.967 8.967 0 01-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 01-5.714 0m5.714 0a3 3 0 11-5.714 0"
            />
          </svg>
          Add Alert
        </button>
      </div>
    );
  }

  // ── State B: Has alert, form closed — compact summary ─────────────
  if (existingAlert && !formOpen) {
    const typeInfo = ALERT_TYPES[existingAlert.alert_type as keyof typeof ALERT_TYPES];
    const freqInfo = ALERT_FREQUENCIES[existingAlert.frequency as keyof typeof ALERT_FREQUENCIES];

    return (
      <div>
        <label className="block text-xs font-medium text-ic-text-secondary mb-1">Alert</label>
        <div className="flex items-center gap-3 flex-wrap">
          {/* Bell icon */}
          <svg
            className={`w-4 h-4 flex-shrink-0 ${existingAlert.is_active ? 'text-ic-blue' : 'text-ic-text-dim'}`}
            fill="currentColor"
            viewBox="0 0 20 20"
          >
            <path d="M10 2a6 6 0 00-6 6v3.586l-.707.707A1 1 0 004 14h12a1 1 0 00.707-1.707L16 11.586V8a6 6 0 00-6-6zM10 18a3 3 0 01-3-3h6a3 3 0 01-3 3z" />
          </svg>

          {/* Type + condition */}
          <span className="text-sm text-ic-text-primary">
            {typeInfo?.label ?? existingAlert.alert_type}{' '}
            <span className="font-medium">
              {formatConditions(existingAlert.alert_type, existingAlert.conditions)}
            </span>
          </span>

          {/* Frequency */}
          <span className="text-xs text-ic-text-dim">
            {freqInfo?.label ?? existingAlert.frequency}
          </span>

          {/* Active badge */}
          {existingAlert.is_active ? (
            <span className="px-1.5 py-0.5 text-[10px] font-semibold bg-green-500/20 text-ic-positive rounded">
              Active
            </span>
          ) : (
            <span className="px-1.5 py-0.5 text-[10px] font-semibold bg-ic-bg-tertiary text-ic-text-dim rounded">
              Paused
            </span>
          )}

          {/* Actions */}
          <div className="flex items-center gap-1 ml-auto">
            <button
              type="button"
              onClick={handleToggleActive}
              className="p-1 text-ic-text-dim hover:text-ic-text-secondary rounded transition-colors"
              title={existingAlert.is_active ? 'Pause' : 'Resume'}
            >
              {existingAlert.is_active ? (
                <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
              ) : (
                <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"
                  />
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
              )}
            </button>
            <button
              type="button"
              onClick={openEditForm}
              className="p-1 text-ic-text-dim hover:text-ic-blue rounded transition-colors"
              title="Edit alert"
            >
              <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                />
              </svg>
            </button>
            <button
              type="button"
              onClick={handleDelete}
              className="p-1 text-ic-text-dim hover:text-ic-negative rounded transition-colors"
              title="Delete alert"
            >
              <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                />
              </svg>
            </button>
          </div>
        </div>
      </div>
    );
  }

  // ── State C: Form (create or edit) ────────────────────────────────
  return (
    <div>
      <label className="block text-xs font-medium text-ic-text-secondary mb-1">
        {isEditing ? 'Edit Alert' : 'New Alert'}
      </label>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
        {/* Alert Type */}
        <div>
          <label className="block text-[10px] text-ic-text-dim mb-0.5">Type</label>
          <select
            value={alertType}
            onChange={(e) => {
              setAlertType(e.target.value as AllowedAlertType);
              if (!nameManuallyEdited) setName('');
            }}
            disabled={isEditing}
            className="block w-full px-2 py-1.5 text-sm border rounded-lg
              bg-ic-input-bg text-ic-text-primary border-ic-input-border
              focus:outline-none focus:ring-2 focus:ring-ic-blue focus:border-ic-blue
              disabled:opacity-50"
          >
            {ALLOWED_ALERT_TYPES.map((type) => {
              const info = ALERT_TYPES[type as keyof typeof ALERT_TYPES];
              return (
                <option key={type} value={type}>
                  {info?.label ?? type}
                </option>
              );
            })}
          </select>
        </div>

        {/* Threshold (hidden for event types) */}
        {needsThreshold(alertType) && (
          <div>
            <label className="block text-[10px] text-ic-text-dim mb-0.5">
              {alertType === 'volume_spike'
                ? 'Multiplier'
                : isPriceType(alertType)
                  ? 'Price'
                  : 'Volume'}
            </label>
            <div className="relative">
              {isPriceType(alertType) && (
                <span className="absolute inset-y-0 left-0 pl-2 flex items-center text-ic-text-dim text-sm pointer-events-none">
                  $
                </span>
              )}
              {alertType === 'volume_spike' && (
                <span className="absolute inset-y-0 right-0 pr-2 flex items-center text-ic-text-dim text-xs pointer-events-none">
                  x avg
                </span>
              )}
              <input
                type="number"
                step={isPriceType(alertType) ? '0.01' : '1'}
                min="0"
                value={threshold}
                onChange={(e) => setThreshold(e.target.value)}
                placeholder={isPriceType(alertType) ? '0.00' : '1000000'}
                className={`block w-full ${isPriceType(alertType) ? 'pl-6' : 'pl-2'} ${alertType === 'volume_spike' ? 'pr-12' : 'pr-2'} py-1.5 text-sm border rounded-lg
                  bg-ic-input-bg text-ic-text-primary border-ic-input-border
                  focus:outline-none focus:ring-2 focus:ring-ic-blue focus:border-ic-blue`}
              />
            </div>
          </div>
        )}

        {/* Frequency */}
        <div>
          <label className="block text-[10px] text-ic-text-dim mb-0.5">Frequency</label>
          <select
            value={frequency}
            onChange={(e) => setFrequency(e.target.value as 'once' | 'daily' | 'always')}
            className="block w-full px-2 py-1.5 text-sm border rounded-lg
              bg-ic-input-bg text-ic-text-primary border-ic-input-border
              focus:outline-none focus:ring-2 focus:ring-ic-blue focus:border-ic-blue"
          >
            {Object.entries(ALERT_FREQUENCIES).map(([key, info]) => (
              <option key={key} value={key}>
                {info.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Row 2: Name + Notification checkboxes */}
      <div className="grid grid-cols-1 md:grid-cols-[2fr_1fr] gap-3 mt-2">
        <div>
          <label className="block text-[10px] text-ic-text-dim mb-0.5">Name</label>
          <input
            type="text"
            value={nameManuallyEdited ? name : autoName}
            onChange={(e) => {
              setName(e.target.value);
              setNameManuallyEdited(true);
            }}
            onFocus={() => {
              if (!nameManuallyEdited) {
                setName(autoName);
                setNameManuallyEdited(true);
              }
            }}
            placeholder="Alert name"
            className="block w-full px-2 py-1.5 text-sm border rounded-lg
              bg-ic-input-bg text-ic-text-primary placeholder-ic-text-dim border-ic-input-border
              focus:outline-none focus:ring-2 focus:ring-ic-blue focus:border-ic-blue"
          />
        </div>

        <div className="flex items-end gap-4 pb-1">
          <label className="flex items-center gap-1.5 text-xs text-ic-text-secondary cursor-pointer">
            <input
              type="checkbox"
              checked={notifyEmail}
              onChange={(e) => setNotifyEmail(e.target.checked)}
              className="h-3.5 w-3.5 rounded border-ic-border text-ic-blue focus:ring-ic-blue"
            />
            Email
          </label>
          <label className="flex items-center gap-1.5 text-xs text-ic-text-secondary cursor-pointer">
            <input
              type="checkbox"
              checked={notifyInApp}
              onChange={(e) => setNotifyInApp(e.target.checked)}
              className="h-3.5 w-3.5 rounded border-ic-border text-ic-blue focus:ring-ic-blue"
            />
            In-App
          </label>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="mt-2 text-xs text-ic-negative" role="alert">
          {error}
        </div>
      )}

      {/* Actions */}
      <div className="mt-2 flex items-center gap-2">
        <button
          type="button"
          onClick={handleSave}
          disabled={saving}
          className="px-3 py-1 text-xs font-medium bg-ic-blue text-ic-text-primary rounded-lg
            hover:bg-ic-blue-hover disabled:bg-ic-bg-tertiary disabled:cursor-not-allowed
            transition-colors"
        >
          {saving ? 'Saving...' : isEditing ? 'Update Alert' : 'Create Alert'}
        </button>
        <button
          type="button"
          onClick={() => {
            setFormOpen(false);
            setError(null);
          }}
          disabled={saving}
          className="px-3 py-1 text-xs font-medium border border-ic-border rounded-lg
            text-ic-text-secondary hover:bg-ic-surface-hover
            disabled:opacity-50 disabled:cursor-not-allowed
            transition-colors"
        >
          Cancel
        </button>
      </div>
    </div>
  );
}
