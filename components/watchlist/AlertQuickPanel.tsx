'use client';

import React, { useState, useMemo, useEffect, useRef } from 'react';
import {
  AlertRuleWithDetails,
  ALERT_TYPES,
  ALERT_FREQUENCIES,
  CreateAlertRequest,
  UpdateAlertRequest,
} from '@/lib/api/alerts';

// Must match the backend binding validation in models.CreateAlertRuleRequest:
//   `binding:"required,oneof=price_above price_below price_change volume_above volume_spike news earnings sec_filing"`
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

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface AlertQuickPanelProps {
  watchListId: string;
  symbol: string;
  currentPrice?: number;
  existingAlert?: AlertRuleWithDetails;
  onCreate: (req: CreateAlertRequest) => Promise<any>;
  onUpdate: (alertId: string, req: UpdateAlertRequest) => Promise<void>;
  onDelete: (alertId: string, symbol: string) => Promise<void>;
  onClose: () => void;
  /** Ref to the trigger element for positioning */
  triggerRef: React.RefObject<HTMLElement | null>;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function AlertQuickPanel({
  watchListId,
  symbol,
  currentPrice,
  existingAlert,
  onCreate,
  onUpdate,
  onDelete,
  onClose,
  triggerRef,
}: AlertQuickPanelProps) {
  const panelRef = useRef<HTMLDivElement>(null);
  const [coords, setCoords] = useState({ top: 0, left: 0 });
  const [isPositioned, setIsPositioned] = useState(false);

  // ── Mode: view (has alert, not editing) vs form (create or edit) ────
  const [formOpen, setFormOpen] = useState(!existingAlert);
  const [isEditing, setIsEditing] = useState(false);
  const [confirmingDelete, setConfirmingDelete] = useState(false);

  // ── Form state ──────────────────────────────────────────────────────
  const [alertType, setAlertType] = useState<AllowedAlertType>('price_above');
  const [threshold, setThreshold] = useState('');
  const [frequency, setFrequency] = useState<'once' | 'daily' | 'always'>('daily');
  const [name, setName] = useState('');
  const [nameManuallyEdited, setNameManuallyEdited] = useState(false);
  const [notifyEmail, setNotifyEmail] = useState(true);
  const [notifyInApp, setNotifyInApp] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // ── Position panel below trigger ────────────────────────────────────
  useEffect(() => {
    if (!triggerRef.current || !panelRef.current) return;

    const updatePosition = () => {
      if (!triggerRef.current || !panelRef.current) return;
      const triggerRect = triggerRef.current.getBoundingClientRect();
      const panelRect = panelRef.current.getBoundingClientRect();
      const padding = 8;

      let top = triggerRect.bottom + 6;
      let left = triggerRect.left + triggerRect.width / 2 - panelRect.width / 2;

      // Keep within viewport horizontally
      if (left < padding) left = padding;
      if (left + panelRect.width > window.innerWidth - padding) {
        left = window.innerWidth - panelRect.width - padding;
      }

      // Flip above trigger if not enough space below
      if (top + panelRect.height > window.innerHeight - padding) {
        top = triggerRect.top - panelRect.height - 6;
      }

      setCoords({ top, left });
      setIsPositioned(true);
    };

    // Initial positioning (allow one render for panelRef to measure)
    requestAnimationFrame(updatePosition);

    // Close on scroll (table may scroll away from trigger)
    const handleScroll = () => onClose();
    const scrollParent = triggerRef.current.closest('.overflow-x-auto');
    scrollParent?.addEventListener('scroll', handleScroll);

    return () => {
      scrollParent?.removeEventListener('scroll', handleScroll);
    };
  }, [triggerRef, onClose]);

  // ── Click outside to close ──────────────────────────────────────────
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (
        panelRef.current &&
        !panelRef.current.contains(e.target as Node) &&
        triggerRef.current &&
        !triggerRef.current.contains(e.target as Node)
      ) {
        onClose();
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [onClose, triggerRef]);

  // ── Escape to close ─────────────────────────────────────────────────
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [onClose]);

  // ── Init form for create mode (no existing alert) ───────────────────
  useEffect(() => {
    if (!existingAlert) {
      setAlertType('price_above');
      setThreshold(currentPrice ? Math.ceil(currentPrice).toString() : '');
      setFrequency('daily');
      setName('');
      setNameManuallyEdited(false);
      setNotifyEmail(true);
      setNotifyInApp(true);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // ── Auto-generated name ─────────────────────────────────────────────
  const autoName = useMemo(() => {
    const typeInfo = ALERT_TYPES[alertType as keyof typeof ALERT_TYPES];
    const label = typeInfo?.label ?? alertType;
    if (needsThreshold(alertType) && threshold) {
      const prefix = isPriceType(alertType) ? '$' : '';
      return `${symbol} ${label} ${prefix}${threshold}`;
    }
    return `${symbol} ${label}`;
  }, [symbol, alertType, threshold]);

  // ── Open edit form for existing alert ───────────────────────────────
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

  // ── Save handler ────────────────────────────────────────────────────
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
      onClose();
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
    } catch (err) {
      console.error('[AlertQuickPanel] Failed to toggle active state:', err);
    }
  };

  const handleDelete = async () => {
    if (!existingAlert) return;
    try {
      await onDelete(existingAlert.id, symbol);
      onClose();
    } catch (err) {
      console.error('[AlertQuickPanel] Failed to delete alert:', err);
    }
  };

  // ── Render ──────────────────────────────────────────────────────────

  const renderViewMode = () => {
    if (!existingAlert) return null;
    const typeInfo = ALERT_TYPES[existingAlert.alert_type as keyof typeof ALERT_TYPES];
    const freqInfo = ALERT_FREQUENCIES[existingAlert.frequency as keyof typeof ALERT_FREQUENCIES];

    return (
      <div className="space-y-3">
        {/* Header */}
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold text-ic-text-primary">{symbol} Alert</span>
          {existingAlert.is_active ? (
            <span className="px-1.5 py-0.5 text-[10px] font-semibold bg-green-500/20 text-ic-positive rounded">
              Active
            </span>
          ) : (
            <span className="px-1.5 py-0.5 text-[10px] font-semibold bg-ic-bg-tertiary text-ic-text-dim rounded">
              Paused
            </span>
          )}
        </div>

        {/* Summary */}
        <div className="flex items-center gap-2">
          <svg
            className={`w-4 h-4 flex-shrink-0 ${existingAlert.is_active ? 'text-ic-blue' : 'text-ic-text-dim'}`}
            fill="currentColor"
            viewBox="0 0 20 20"
          >
            <path d="M10 2a6 6 0 00-6 6v3.586l-.707.707A1 1 0 004 14h12a1 1 0 00.707-1.707L16 11.586V8a6 6 0 00-6-6zM10 18a3 3 0 01-3-3h6a3 3 0 01-3 3z" />
          </svg>
          <span className="text-sm text-ic-text-primary">
            {typeInfo?.label ?? existingAlert.alert_type}{' '}
            <span className="font-medium">
              {formatConditions(existingAlert.alert_type, existingAlert.conditions)}
            </span>
          </span>
          <span className="text-xs text-ic-text-dim">
            {freqInfo?.label ?? existingAlert.frequency}
          </span>
        </div>

        {/* Trigger info */}
        {existingAlert.trigger_count > 0 && (
          <p className="text-[10px] text-ic-text-dim">
            Triggered {existingAlert.trigger_count} time{existingAlert.trigger_count !== 1 && 's'}
          </p>
        )}

        {/* Actions */}
        <div className="flex items-center gap-2 pt-1 border-t border-ic-border">
          <button
            type="button"
            onClick={handleToggleActive}
            className="px-2 py-1 text-xs font-medium border border-ic-border rounded-lg text-ic-text-secondary hover:bg-ic-surface-hover transition-colors"
          >
            {existingAlert.is_active ? 'Pause' : 'Resume'}
          </button>
          <button
            type="button"
            onClick={openEditForm}
            className="px-2 py-1 text-xs font-medium text-ic-blue hover:bg-ic-blue/10 rounded-lg transition-colors"
          >
            Edit
          </button>
          {confirmingDelete ? (
            <div className="flex items-center gap-1.5 ml-auto">
              <span className="text-xs text-ic-text-secondary">Delete?</span>
              <button
                type="button"
                onClick={handleDelete}
                className="px-1.5 py-0.5 text-[10px] font-medium bg-ic-negative/20 text-ic-negative rounded hover:bg-ic-negative/30 transition-colors"
              >
                Yes
              </button>
              <button
                type="button"
                onClick={() => setConfirmingDelete(false)}
                className="px-1.5 py-0.5 text-[10px] font-medium border border-ic-border text-ic-text-secondary rounded hover:bg-ic-surface-hover transition-colors"
              >
                No
              </button>
            </div>
          ) : (
            <button
              type="button"
              onClick={() => setConfirmingDelete(true)}
              className="px-2 py-1 text-xs font-medium text-ic-negative hover:bg-ic-negative/10 rounded-lg transition-colors ml-auto"
            >
              Delete
            </button>
          )}
        </div>
      </div>
    );
  };

  const renderFormMode = () => (
    <div className="space-y-3">
      {/* Header */}
      <div className="flex items-center justify-between">
        <span className="text-xs font-semibold text-ic-text-primary">
          {isEditing ? `Edit ${symbol} Alert` : `New Alert for ${symbol}`}
        </span>
      </div>

      {/* Type + Threshold row */}
      <div className="grid grid-cols-2 gap-2">
        <div>
          <label className="block text-[10px] text-ic-text-dim mb-0.5">Type</label>
          <select
            value={alertType}
            onChange={(e) => {
              setAlertType(e.target.value as AllowedAlertType);
              if (!nameManuallyEdited) setName('');
            }}
            disabled={isEditing}
            className="block w-full px-2 py-1.5 text-xs border rounded-lg
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
          {isEditing && (
            <p className="mt-0.5 text-[10px] text-ic-text-dim">
              To change type, delete and recreate.
            </p>
          )}
        </div>

        {needsThreshold(alertType) ? (
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
                <span className="absolute inset-y-0 left-0 pl-2 flex items-center text-ic-text-dim text-xs pointer-events-none">
                  $
                </span>
              )}
              {alertType === 'volume_spike' && (
                <span className="absolute inset-y-0 right-0 pr-2 flex items-center text-ic-text-dim text-[10px] pointer-events-none">
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
                className={`block w-full ${isPriceType(alertType) ? 'pl-5' : 'pl-2'} ${alertType === 'volume_spike' ? 'pr-10' : 'pr-2'} py-1.5 text-xs border rounded-lg
                  bg-ic-input-bg text-ic-text-primary border-ic-input-border
                  focus:outline-none focus:ring-2 focus:ring-ic-blue focus:border-ic-blue`}
              />
            </div>
          </div>
        ) : (
          <div>
            <label className="block text-[10px] text-ic-text-dim mb-0.5">Frequency</label>
            <select
              value={frequency}
              onChange={(e) => setFrequency(e.target.value as 'once' | 'daily' | 'always')}
              className="block w-full px-2 py-1.5 text-xs border rounded-lg
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
        )}
      </div>

      {/* Frequency (shown separately when threshold types are active) */}
      {needsThreshold(alertType) && (
        <div>
          <label className="block text-[10px] text-ic-text-dim mb-0.5">Frequency</label>
          <select
            value={frequency}
            onChange={(e) => setFrequency(e.target.value as 'once' | 'daily' | 'always')}
            className="block w-full px-2 py-1.5 text-xs border rounded-lg
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
      )}

      {/* Name */}
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
          className="block w-full px-2 py-1.5 text-xs border rounded-lg
            bg-ic-input-bg text-ic-text-primary placeholder-ic-text-dim border-ic-input-border
            focus:outline-none focus:ring-2 focus:ring-ic-blue focus:border-ic-blue"
        />
      </div>

      {/* Notifications */}
      <div className="flex items-center gap-4">
        <label className="flex items-center gap-1.5 text-xs text-ic-text-secondary cursor-pointer">
          <input
            type="checkbox"
            checked={notifyEmail}
            onChange={(e) => setNotifyEmail(e.target.checked)}
            className="h-3 w-3 rounded border-ic-border text-ic-blue focus:ring-ic-blue"
          />
          Email
        </label>
        <label className="flex items-center gap-1.5 text-xs text-ic-text-secondary cursor-pointer">
          <input
            type="checkbox"
            checked={notifyInApp}
            onChange={(e) => setNotifyInApp(e.target.checked)}
            className="h-3 w-3 rounded border-ic-border text-ic-blue focus:ring-ic-blue"
          />
          In-App
        </label>
      </div>

      {/* Error */}
      {error && (
        <div className="text-xs text-ic-negative" role="alert">
          {error}
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={handleSave}
          disabled={saving}
          className="px-3 py-1 text-xs font-medium bg-ic-blue text-ic-text-primary rounded-lg
            hover:bg-ic-blue-hover disabled:bg-ic-bg-tertiary disabled:cursor-not-allowed
            transition-colors"
        >
          {saving ? 'Saving...' : isEditing ? 'Update' : 'Create Alert'}
        </button>
        <button
          type="button"
          onClick={() => {
            if (isEditing && existingAlert) {
              setFormOpen(false);
              setIsEditing(false);
              setError(null);
            } else {
              onClose();
            }
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

  return (
    <div
      ref={panelRef}
      className={`fixed z-50 w-[340px] p-4 bg-ic-surface border border-ic-border rounded-xl shadow-xl transition-opacity duration-100 ${
        isPositioned ? 'opacity-100' : 'opacity-0'
      }`}
      style={{ top: coords.top, left: coords.left }}
    >
      {formOpen ? renderFormMode() : renderViewMode()}
    </div>
  );
}
