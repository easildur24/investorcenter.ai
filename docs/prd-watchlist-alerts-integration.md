# PRD: Alerts as Part of Watchlist

**Author:** Product
**Status:** Draft
**Date:** 2026-02-25

---

## 1. Problem

Creating an alert requires navigating to a separate /alerts page, selecting a watchlist, then selecting a symbol — even though the user was just looking at that symbol in their watchlist. This 3-click detour breaks flow and discourages alert adoption. The watchlist table already defines an `alert` column and returns `alert_count` per ticker in its API response, but neither is surfaced meaningfully — the alert column only fires on target price hits, and `alert_count` isn't shown in the default view. The data model is ready; the UX isn't.

## 2. Goals

- **Reduce alert creation from 5 steps (navigate to /alerts → open modal → select watchlist → select symbol → configure) to 2 steps** (click bell in watchlist row → configure). Measured by avg time-to-create-alert in analytics.
- **Surface alert status inline** so users see at a glance which tickers have active alerts without leaving the watchlist.
- **Increase alert adoption to 30% of active watchlist users** within 60 days of launch (baseline: current /alerts page usage).

## 3. Non-Goals

- Portfolio-level alerts (P&L thresholds, allocation drift).
- Mobile app or push notification channels — this is web UI only.
- Alert evaluation engine changes — the backend trigger/delivery system is untouched; this PRD only changes creation and display.
- Bulk alert templates or "smart alert" recommendations.

## 4. User Stories

1. **As a** watchlist user, **I want to** click a bell icon on any ticker row to create an alert for that symbol **so that** I never leave my watchlist to set up monitoring.
2. **As a** watchlist user, **I want to** see which tickers have active alerts (and how many) directly in the table **so that** I know my coverage at a glance.
3. **As a** watchlist user, **I want to** view, edit, and delete all alerts for a specific ticker from within the watchlist row **so that** I can manage alerts in context.
4. **As a** watchlist user, **I want to** apply a single alert condition to every ticker in my watchlist **so that** I can monitor an entire list without configuring each row individually.

## 5. UX Design Specification

**Nav change:** Remove "Alerts" from the top-level navigation. Add a 301 redirect: `/alerts` → `/watchlist?tab=alerts`. During the 30-day transition period, show a banner on the redirected page: "Alerts have moved into your watchlists."

**Alert column activation:** Replace the current `alert` badge column in the default "general" view with a unified `AlertCell` component:

| State | Display | Behavior |
|-------|---------|----------|
| No alerts | Empty bell icon (outline) | Click opens AlertQuickPanel in "create" mode |
| 1+ active alerts | Filled bell icon + count badge | Click opens AlertQuickPanel in "list" mode |
| Alert triggered (unread) | Filled bell icon, pulsing accent color | Click opens AlertQuickPanel with triggered alert highlighted |

**AlertQuickPanel** (popover, anchored to the bell icon):
- **Create mode:** Alert Type dropdown, Threshold input, Frequency selector, Notification toggles (email / in-app). Symbol and watchlist are auto-filled from the row context — not editable. "Create Alert" button.
- **List mode:** Compact list of existing alerts for that symbol-in-watchlist. Each row shows: type icon, condition summary, active toggle, edit/delete actions. "+ Add alert" link at bottom switches to create mode.
- Panel width: 360px. Max height: 400px with scroll.

**Watchlist-level bulk alert:** Add "Set alert for all tickers" to the existing watchlist header kebab/options menu. Opens a modal identical to the create form but without a symbol field — applies the configured alert to every ticker in the list. Skips tickers that already have an identical alert type active.

**All-alerts view:** Add an "Alerts" tab to the watchlist detail page (alongside the existing table/heatmap views). This tab renders a filtered version of the current AlertCard list, scoped to the active watchlist. Provides the "see everything" escape hatch without a separate page.

## 6. Technical Specification Guidance

### Data Model

No schema migration required. The `alert_rules` table already has `watch_list_id` (UUID FK), `watch_list_item_id` (nullable UUID FK), and `symbol` (VARCHAR) columns. The `alert_count` field is already computed via subquery in `database/watchlists.go` and returned in the `WatchListItem` response. The only backend data change: extend `GET /api/v1/watchlists/:id` to include a lightweight `alerts` array per ticker (id, alert_type, is_active, last_triggered_at) so the AlertQuickPanel list mode can render without a second fetch.

### API Changes

| Endpoint | Change |
|----------|--------|
| `GET /api/v1/watchlists/:id` | Add `alerts: AlertSummary[]` per ticker item. New query param `?include_alerts=true` to opt in (keeps default response lean). |
| `POST /api/v1/alerts` | No change — already accepts `{ watch_list_id, symbol, alert_type, conditions, frequency, notify_email, notify_in_app, name }`. |
| `PUT /api/v1/alerts/:id` | No change — already supports partial update. |
| `DELETE /api/v1/alerts/:id` | No change. |
| `POST /api/v1/alerts/bulk` | **New.** Accepts `{ watch_list_id, alert_type, conditions, frequency, notify_email, notify_in_app }` — creates one alert per ticker in the watchlist. Returns `{ created: number, skipped: number }`. |

`AlertSummary` shape: `{ id, alert_type, is_active, last_triggered_at, trigger_count }`.

### Frontend Changes

**New components:**
- `components/watchlist/AlertCell.tsx` — Bell icon with badge. Manages popover open/close. Reads `alert_count` and `alerts[]` from the row data.
- `components/watchlist/AlertQuickPanel.tsx` — Popover body. Create form (simplified from `CreateAlertModal` — no watchlist/symbol selection step) and alert list with inline edit/delete.

**Modified files:**
- `lib/watchlist/columns.ts` — Replace `alert` badge column definition with new `AlertCell` renderer in the `general` view preset.
- `app/watchlist/[id]/page.tsx` — Add "Alerts" tab. Add "Set alert for all" to kebab menu.
- `lib/api/watchlist.ts` — Update `WatchListItem` interface to include optional `alerts: AlertSummary[]`. Add `?include_alerts=true` param to `getWatchList()`.
- `lib/api/alerts.ts` — Add `bulkCreateAlerts()` method.
- `components/layout/` (nav) — Remove "Alerts" nav item.
- `app/alerts/page.tsx` — Replace with redirect to `/watchlist?tab=alerts`.

### State Management

Alert state is owned by the watchlist ticker data, not a separate global store. On alert create/delete via the QuickPanel, apply optimistic updates to `alert_count` and the `alerts[]` array on the row. Revalidate on popover close or after 5 seconds, whichever comes first.

### Notification Delivery

Out of scope. The existing alert evaluation engine (if/when built) reads from `alert_rules` and writes to `alert_logs`. This PRD only changes how rules are created and displayed.

## 7. Success Metrics

1. **Alert creation rate:** 30% of active watchlist users create at least one alert within 60 days (up from current baseline on /alerts page).
2. **Time-to-create-alert:** Median drops from current ~45s (navigate + modal + selections) to <15s (bell click + configure).
3. **Alert column engagement:** >50% of watchlist page sessions include at least one hover or click on a bell icon within 30 days.

## 8. Migration Plan

- **Existing alerts are unaffected.** The `alert_rules` table already has `watch_list_id` — all existing alerts will appear in their corresponding watchlist's AlertQuickPanel and Alerts tab automatically.
- **30-day redirect period:** `/alerts` returns a 301 to `/watchlist?tab=alerts` with a dismissible banner explaining the move. After 30 days, remove the redirect route and banner code.
- **Feature flag rollout:** Ship behind a `watchlist-inline-alerts` flag. Enable for 10% of users for 1 week, monitor error rates and alert creation metrics, then roll to 100%.

## 9. Open Questions

1. **Subscription tier limits:** The current CreateAlertModal checks `CanCreateAlert()` for tier-based caps. Should the bulk "Set alert for all" action count as N alerts against the cap, or should bulk-created alerts have a separate limit? (Product decision.)
2. **Alert name auto-generation:** The current modal requires a user-provided `name` field. For the quick-create panel, should we auto-generate names (e.g., "AAPL Price Above $200") to reduce friction, or keep it required? (UX decision.)
3. **Real-time alert status:** Should the bell icon update in real-time (via WebSocket or polling) when an alert triggers while the user is viewing the watchlist, or only on page refresh? (Engineering cost/benefit.)
