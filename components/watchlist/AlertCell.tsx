'use client';

import React, { useState, useRef } from 'react';
import { WatchListItem } from '@/lib/api/watchlist';
import {
  AlertRule,
  AlertRuleWithDetails,
  CreateAlertRequest,
  UpdateAlertRequest,
} from '@/lib/api/alerts';
import AlertQuickPanel from './AlertQuickPanel';

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface AlertCellProps {
  item: WatchListItem;
  /** Existing alert rule for this symbol, if any. */
  existingAlert?: AlertRuleWithDetails;
  /** Watchlist ID (auto-filled into alert create requests). */
  watchListId: string;
  /** Target-price alert from checkTargetAlert() — buy/sell target badge. */
  targetAlert: { type: 'buy' | 'sell'; message: string } | null;
  /** Create a new alert rule. Returns the created AlertRule from the API. */
  onAlertCreate: (req: CreateAlertRequest) => Promise<AlertRule>;
  /** Update an existing alert rule. */
  onAlertUpdate: (alertId: string, req: UpdateAlertRequest) => Promise<void>;
  /** Delete an alert rule. */
  onAlertDelete: (alertId: string, symbol: string) => Promise<void>;
}

// ---------------------------------------------------------------------------
// Component
//
// Three visual states: active rule pill, target-price badge, muted bell.
// triggerRef is always attached to a <button> in states 1 & 3 so the
// popover can position itself. State 2 (target badge) has no interactive
// trigger, but the ref stays defined to avoid conditional hook issues.
// ---------------------------------------------------------------------------

export default function AlertCell({
  item,
  existingAlert,
  watchListId,
  targetAlert,
  onAlertCreate,
  onAlertUpdate,
  onAlertDelete,
}: AlertCellProps) {
  const [popoverOpen, setPopoverOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);

  const handleTogglePopover = (e: React.MouseEvent) => {
    e.stopPropagation();
    setPopoverOpen((prev) => !prev);
  };

  // ── State 1: Has active alert rule — blue "Active" pill ──────────────
  if (existingAlert) {
    return (
      <>
        <button
          ref={triggerRef}
          className="inline-flex items-center gap-1 px-2 py-1 text-xs font-semibold rounded bg-ic-blue/20 text-ic-blue hover:bg-ic-blue/30 transition-colors"
          onClick={handleTogglePopover}
          title="Manage alert"
        >
          <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
            <path d="M10 2a6 6 0 00-6 6v3.586l-.707.707A1 1 0 004 14h12a1 1 0 00.707-1.707L16 11.586V8a6 6 0 00-6-6zM10 18a3 3 0 01-3-3h6a3 3 0 01-3 3z" />
          </svg>
          Active
        </button>
        {popoverOpen && (
          <AlertQuickPanel
            watchListId={watchListId}
            symbol={item.symbol}
            currentPrice={item.current_price}
            existingAlert={existingAlert}
            onCreate={onAlertCreate}
            onUpdate={onAlertUpdate}
            onDelete={onAlertDelete}
            onClose={() => setPopoverOpen(false)}
            triggerRef={triggerRef}
          />
        )}
      </>
    );
  }

  // ── State 2: Target price triggered (legacy badge, no popover) ───────
  // Derived from checkTargetAlert() when current_price crosses target_buy_price
  // or target_sell_price. Distinct from alert rules. triggerRef is not attached
  // here — if state oscillates rapidly between 2 ↔ 1/3, React reconciliation
  // remounts the button and the ref reattaches cleanly.
  if (targetAlert) {
    return (
      <span
        className={`inline-block px-2 py-1 text-xs font-semibold rounded ${
          targetAlert.type === 'buy'
            ? 'bg-green-500/20 text-ic-positive'
            : 'bg-blue-500/20 text-ic-blue'
        }`}
      >
        {targetAlert.message}
      </span>
    );
  }

  // ── State 3: No alert — muted bell, opens create mode ────────────────
  return (
    <>
      <button
        ref={triggerRef}
        className="text-ic-text-dim hover:text-ic-blue transition-colors"
        onClick={handleTogglePopover}
        title="Set alert"
      >
        <svg className="w-4 h-4 mx-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={1.5}
            d="M14.857 17.082a23.848 23.848 0 005.454-1.31A8.967 8.967 0 0118 9.75v-.7V9A6 6 0 006 9v.75a8.967 8.967 0 01-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 01-5.714 0m5.714 0a3 3 0 11-5.714 0"
          />
        </svg>
      </button>
      {popoverOpen && (
        <AlertQuickPanel
          watchListId={watchListId}
          symbol={item.symbol}
          currentPrice={item.current_price}
          onCreate={onAlertCreate}
          onUpdate={onAlertUpdate}
          onDelete={onAlertDelete}
          onClose={() => setPopoverOpen(false)}
          triggerRef={triggerRef}
        />
      )}
    </>
  );
}
