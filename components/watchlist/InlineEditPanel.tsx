'use client';

import React, { useState, useEffect, useRef, useCallback } from 'react';
import { WatchListItem } from '@/lib/api/watchlist';
import { AlertRuleWithDetails, CreateAlertRequest, UpdateAlertRequest } from '@/lib/api/alerts';
import TagChipInput from '@/components/watchlist/TagChipInput';
import InlineAlertSection from '@/components/watchlist/InlineAlertSection';

interface InlineEditPanelProps {
  /** The watchlist item being edited. */
  item: WatchListItem;
  /** Previously-used tags across all user watchlists (for autocomplete suggestions). */
  tagSuggestions: string[];
  /** Called on save with the symbol and updated metadata. */
  onSave: (
    symbol: string,
    data: {
      notes?: string;
      tags?: string[];
      target_buy_price?: number;
      target_sell_price?: number;
    }
  ) => Promise<void>;
  /** Called when the user cancels editing. */
  onCancel: () => void;
  /** Watchlist ID (for alert creation). */
  watchListId: string;
  /** Existing alert rule for this symbol, if any. */
  existingAlert?: AlertRuleWithDetails;
  /** Create a new alert rule. */
  onAlertCreate: (req: CreateAlertRequest) => Promise<any>;
  /** Update an existing alert rule. */
  onAlertUpdate: (alertId: string, req: UpdateAlertRequest) => Promise<void>;
  /** Delete an alert rule. */
  onAlertDelete: (alertId: string, symbol: string) => Promise<void>;
}

export default function InlineEditPanel({
  item,
  tagSuggestions,
  onSave,
  onCancel,
  watchListId,
  existingAlert,
  onAlertCreate,
  onAlertUpdate,
  onAlertDelete,
}: InlineEditPanelProps) {
  const [targetBuy, setTargetBuy] = useState(item.target_buy_price?.toString() ?? '');
  const [targetSell, setTargetSell] = useState(item.target_sell_price?.toString() ?? '');
  const [tags, setTags] = useState<string[]>(item.tags ?? []);
  const [notes, setNotes] = useState(item.notes ?? '');
  const [saving, setSaving] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);

  // Detect Mac for keyboard shortcut hint (avoids SSR hydration mismatch)
  const [modKey, setModKey] = useState('Ctrl');
  const panelRef = useRef<HTMLDivElement>(null);
  const firstInputRef = useRef<HTMLInputElement>(null);

  // Focus first field on mount + detect platform
  useEffect(() => {
    firstInputRef.current?.focus();
    if (/Mac/.test(navigator.userAgent)) {
      setModKey('⌘');
    }
  }, []);

  // Escape to cancel
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onCancel();
      }
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [onCancel]);

  const validate = useCallback((): boolean => {
    const buy = targetBuy ? parseFloat(targetBuy) : null;
    const sell = targetSell ? parseFloat(targetSell) : null;

    if (buy !== null && isNaN(buy)) {
      setValidationError('Target buy price must be a valid number.');
      return false;
    }
    if (sell !== null && isNaN(sell)) {
      setValidationError('Target sell price must be a valid number.');
      return false;
    }
    if (buy !== null && sell !== null && buy >= sell) {
      setValidationError('Target buy price must be less than target sell price.');
      return false;
    }
    if (buy !== null && buy < 0) {
      setValidationError('Target buy price cannot be negative.');
      return false;
    }
    if (sell !== null && sell < 0) {
      setValidationError('Target sell price cannot be negative.');
      return false;
    }

    setValidationError(null);
    return true;
  }, [targetBuy, targetSell]);

  const handleSave = async () => {
    if (!validate()) return;

    setSaving(true);
    try {
      await onSave(item.symbol, {
        notes: notes || undefined,
        tags,
        target_buy_price: targetBuy ? parseFloat(targetBuy) : undefined,
        target_sell_price: targetSell ? parseFloat(targetSell) : undefined,
      });
    } finally {
      setSaving(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    // Ctrl+Enter or Cmd+Enter to save
    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      handleSave();
    }
  };

  return (
    <div
      ref={panelRef}
      onKeyDown={handleKeyDown}
      className="px-6 py-4 bg-ic-bg-secondary border-t border-ic-border"
    >
      <div className="grid grid-cols-1 md:grid-cols-[1fr_1fr_2fr] gap-4">
        {/* Target Buy Price */}
        <div>
          <label
            htmlFor={`edit-buy-${item.symbol}`}
            className="block text-xs font-medium text-ic-text-secondary mb-1"
          >
            Target Buy Price
          </label>
          <div className="relative">
            <span className="absolute inset-y-0 left-0 pl-3 flex items-center text-ic-text-dim text-sm pointer-events-none">
              $
            </span>
            <input
              ref={firstInputRef}
              id={`edit-buy-${item.symbol}`}
              type="number"
              step="0.01"
              min="0"
              value={targetBuy}
              onChange={(e) => {
                setTargetBuy(e.target.value);
                setValidationError(null);
              }}
              placeholder="0.00"
              className="block w-full pl-7 pr-3 py-1.5 text-sm border rounded-lg
                bg-ic-input-bg text-ic-text-primary placeholder-ic-text-dim
                border-ic-input-border
                focus:outline-none focus:ring-2 focus:ring-ic-blue focus:border-ic-blue"
            />
          </div>
        </div>

        {/* Target Sell Price */}
        <div>
          <label
            htmlFor={`edit-sell-${item.symbol}`}
            className="block text-xs font-medium text-ic-text-secondary mb-1"
          >
            Target Sell Price
          </label>
          <div className="relative">
            <span className="absolute inset-y-0 left-0 pl-3 flex items-center text-ic-text-dim text-sm pointer-events-none">
              $
            </span>
            <input
              id={`edit-sell-${item.symbol}`}
              type="number"
              step="0.01"
              min="0"
              value={targetSell}
              onChange={(e) => {
                setTargetSell(e.target.value);
                setValidationError(null);
              }}
              placeholder="0.00"
              className="block w-full pl-7 pr-3 py-1.5 text-sm border rounded-lg
                bg-ic-input-bg text-ic-text-primary placeholder-ic-text-dim
                border-ic-input-border
                focus:outline-none focus:ring-2 focus:ring-ic-blue focus:border-ic-blue"
            />
          </div>
        </div>

        {/* Tags */}
        <div>
          <label className="block text-xs font-medium text-ic-text-secondary mb-1">Tags</label>
          <TagChipInput tags={tags} onChange={setTags} suggestions={tagSuggestions} maxTags={10} />
        </div>
      </div>

      {/* Notes */}
      <div className="mt-3">
        <label
          htmlFor={`edit-notes-${item.symbol}`}
          className="block text-xs font-medium text-ic-text-secondary mb-1"
        >
          Notes
        </label>
        <textarea
          id={`edit-notes-${item.symbol}`}
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
          rows={2}
          placeholder="Investment thesis, key observations..."
          className="block w-full px-3 py-1.5 text-sm border rounded-lg resize-y
            bg-ic-input-bg text-ic-text-primary placeholder-ic-text-dim
            border-ic-input-border
            focus:outline-none focus:ring-2 focus:ring-ic-blue focus:border-ic-blue"
        />
      </div>

      {/* Alert */}
      <div className="mt-3">
        <InlineAlertSection
          watchListId={watchListId}
          symbol={item.symbol}
          currentPrice={item.current_price}
          existingAlert={existingAlert}
          onCreate={onAlertCreate}
          onUpdate={onAlertUpdate}
          onDelete={onAlertDelete}
        />
      </div>

      {/* Validation error */}
      {validationError && (
        <div className="mt-2 text-sm text-ic-negative" role="alert">
          {validationError}
        </div>
      )}

      {/* Actions */}
      <div className="mt-3 flex items-center gap-2">
        <button
          type="button"
          onClick={handleSave}
          disabled={saving}
          className="px-3 py-1.5 text-sm font-medium bg-ic-blue text-ic-text-primary rounded-lg
            hover:bg-ic-blue-hover disabled:bg-ic-bg-tertiary disabled:cursor-not-allowed
            transition-colors"
        >
          {saving ? 'Saving...' : 'Save'}
        </button>
        <button
          type="button"
          onClick={onCancel}
          disabled={saving}
          className="px-3 py-1.5 text-sm font-medium border border-ic-border rounded-lg
            text-ic-text-secondary hover:bg-ic-surface-hover
            disabled:opacity-50 disabled:cursor-not-allowed
            transition-colors"
        >
          Cancel
        </button>
        <span className="ml-auto text-xs text-ic-text-dim hidden sm:inline">
          <kbd className="border border-ic-border rounded px-1 py-0.5 font-mono bg-ic-bg-tertiary text-[10px]">
            {modKey}
          </kbd>{' '}
          +{' '}
          <kbd className="border border-ic-border rounded px-1 py-0.5 font-mono bg-ic-bg-tertiary text-[10px]">
            Enter
          </kbd>{' '}
          to save
        </span>
      </div>
    </div>
  );
}
