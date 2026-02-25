# Tech Spec: Alerts as Part of Watchlist

**PRD:** [prd-watchlist-alerts-integration.md](./prd-watchlist-alerts-integration.md)
**Status:** Draft
**Date:** 2026-02-25

---

## 1. Overview

Integrate alert creation, display, and management into the watchlist UI. Remove the standalone /alerts page. The user flow becomes: create watchlist → add tickers → create alerts per ticker from within the watchlist row.

This spec covers backend API changes, frontend component work, and navigation updates. It does **not** cover the alert evaluation/trigger engine.

---

## 2. Backend Changes

### 2.1 Data Model — No Migration Required

The `alert_rules` table (migration `012_alert_tables.sql`) already has:
- `watch_list_id` UUID FK → `watch_lists.id`
- `watch_list_item_id` UUID FK → `watch_list_items.id` (nullable)
- `symbol` VARCHAR(20)

`alert_count` is already computed via a LATERAL subquery in `database/watchlists.go` (lines 305–311), counting active alerts per `(watch_list_id, symbol)`. No schema changes needed.

### 2.2 New Response Type: AlertSummary

Add to `backend/models/alert.go`:

```go
type AlertSummary struct {
    ID              string     `json:"id"`
    AlertType       string     `json:"alert_type"`
    IsActive        bool       `json:"is_active"`
    Conditions      json.RawMessage `json:"conditions"`
    Frequency       string     `json:"frequency"`
    LastTriggeredAt *time.Time `json:"last_triggered_at,omitempty"`
    TriggerCount    int        `json:"trigger_count"`
}
```

Add `Alerts []AlertSummary` field to `WatchListItemDetail` in `backend/models/watchlist.go` (line ~96, next to existing `AlertCount int`):

```go
AlertCount int            `json:"alert_count"`
Alerts     []AlertSummary `json:"alerts,omitempty"`
```

### 2.3 Extend GET /api/v1/watchlists/:id

**File:** `backend/database/watchlists.go`

Add a new function `GetAlertSummariesForWatchList(watchListID, userID string) (map[string][]AlertSummary, error)` that runs:

```sql
SELECT id, symbol, alert_type, is_active, conditions, frequency,
       last_triggered_at, trigger_count
FROM alert_rules
WHERE watch_list_id = $1 AND user_id = $2
ORDER BY created_at DESC
```

Returns a `map[symbol][]AlertSummary` for O(1) lookup when assembling the response.

**File:** `backend/handlers/watchlist_handlers.go` — `GetWatchList()`

Check for query param `?include_alerts=true`. If set, call `GetAlertSummariesForWatchList()` and attach the `Alerts` slice to each item before returning. This keeps the default response unchanged (backward compatible).

**File:** `backend/services/watchlist_service.go`

Add method `GetWatchListWithAlerts(watchListID, userID string)` that calls existing `GetWatchListWithItems()` then enriches with alert summaries. Only called when `include_alerts=true`.

### 2.4 New Endpoint: POST /api/v1/alerts/bulk

**File:** `backend/main.go` — Add route inside the existing `alerts` group (line ~270):

```go
alerts.POST("/bulk", alertHandler.BulkCreateAlertRules)
```

**File:** `backend/models/alert.go` — Add request/response types:

```go
type BulkCreateAlertRequest struct {
    WatchListID string          `json:"watch_list_id" binding:"required"`
    AlertType   string          `json:"alert_type" binding:"required"`
    Conditions  json.RawMessage `json:"conditions" binding:"required"`
    Frequency   string          `json:"frequency" binding:"required,oneof=once always daily"`
    NotifyEmail bool            `json:"notify_email"`
    NotifyInApp bool            `json:"notify_in_app"`
}

type BulkCreateAlertResponse struct {
    Created int `json:"created"`
    Skipped int `json:"skipped"`
}
```

**File:** `backend/handlers/alert_handlers.go` — Add `BulkCreateAlertRules()`:

1. Bind and validate `BulkCreateAlertRequest`
2. Call `ValidateWatchListOwnership(userID, watchListID)`
3. Fetch all tickers in the watchlist via `watchListService.GetWatchListItems(watchListID)`
4. For each ticker:
   - Check if an alert with the same `(watch_list_id, symbol, alert_type)` already exists and is active → skip
   - Check `CanCreateAlert(userID)` before each insert → stop with partial result if limit hit
   - Auto-generate name: `"{SYMBOL} {AlertTypeLabel}"` (e.g., "AAPL Price Above")
   - Call `alertService.CreateAlert()` with the ticker's symbol
5. Return `BulkCreateAlertResponse` with created/skipped counts
6. Status: 201 if any created, 200 if all skipped, 403 if limit reached mid-batch

**File:** `backend/services/alert_service.go` — Add `AlertExistsForSymbol(userID, watchListID, symbol, alertType string) (bool, error)` to support the skip-duplicate check.

**File:** `backend/database/alerts.go` — Add query:

```sql
SELECT EXISTS(
    SELECT 1 FROM alert_rules
    WHERE watch_list_id = $1 AND symbol = $2 AND alert_type = $3 AND is_active = true
)
```

### 2.5 Existing Endpoints — No Changes

These already work as needed:
- `POST /api/v1/alerts` — CreateAlertRule
- `GET /api/v1/alerts/:id` — GetAlertRule
- `PUT /api/v1/alerts/:id` — UpdateAlertRule
- `DELETE /api/v1/alerts/:id` — DeleteAlertRule (returns 204)
- `GET /api/v1/alerts` — ListAlertRules (supports `?watch_list_id=` filter)

---

## 3. Frontend Changes

### 3.1 New Component: AlertCell

**File:** `components/watchlist/AlertCell.tsx`

Replaces the current `alert` badge column renderer. Props:

```typescript
interface AlertCellProps {
  symbol: string;
  watchListId: string;
  alertCount: number;
  alerts: AlertSummary[];
  onAlertChange: () => void; // triggers refetch of watchlist data
}
```

**Rendering:**
- `alertCount === 0`: Outline `BellIcon` (from `@heroicons/react/24/outline`), muted color
- `alertCount > 0`: Solid `BellIcon` (from `@heroicons/react/24/solid`), primary color, with a count badge (`<span>` positioned top-right)
- Any alert with `last_triggered_at` newer than the last page load: pulsing animation via Tailwind `animate-pulse`

**Behavior:**
- `onClick`: Toggles `AlertQuickPanel` popover open/closed
- Uses a `useState` for popover visibility
- Popover positioned below the bell icon, aligned to start

### 3.2 New Component: AlertQuickPanel

**File:** `components/watchlist/AlertQuickPanel.tsx`

A popover panel (360px wide, max 400px tall with overflow scroll). Two modes:

**Create Mode** (shown when `alerts.length === 0` or user clicks "+ Add alert"):
- Alert Type: `<select>` dropdown using `ALERT_TYPES` from `lib/api/alerts.ts`
- Threshold: `<input type="number">` (label changes based on alert type)
- Frequency: `<select>` dropdown using `ALERT_FREQUENCIES`
- Notifications: Two checkboxes (Email, In-App), both default true
- "Create Alert" button
- Symbol and watchlist ID are passed as props — not shown as form fields
- Auto-generates `name` field: `"{SYMBOL} {ALERT_TYPES[type].label}"`
- On submit: call `alertAPI.createAlert()`, optimistic update `alertCount + 1` on the row, call `onAlertChange`

**List Mode** (shown when `alerts.length > 0`):
- Compact list of existing alerts. Each row:
  - Icon from `ALERT_TYPES[alert.alert_type].icon`
  - Type label + condition summary (e.g., "Price Above $200.00")
  - Toggle switch for `is_active` → calls `alertAPI.updateAlert(id, { is_active: !current })`
  - Delete icon (TrashIcon) → calls `alertAPI.deleteAlert(id)` with confirm
- Footer: "+ Add alert" link → switches to create mode

**Popover Implementation:**
No external UI library in the codebase. Implement as a positioned `<div>` with:
- `position: absolute`, anchored relative to the AlertCell
- Click-outside listener (`useEffect` with `mousedown` event)
- `z-50` Tailwind class to sit above table rows
- Arrow/caret pointing to the bell icon (CSS triangle)

### 3.3 Modify: Watchlist Column Definition

**File:** `lib/watchlist/columns.ts`

Replace the existing `alert` badge column (lines 165–173) with a custom renderer column:

```typescript
{
  id: 'alert',
  label: 'Alerts',
  type: 'custom',  // new type for component-rendered cells
  align: 'center',
  sortable: true,
  premium: false,
  getValue: (item) => item.alert_count,
}
```

Add `'custom'` to the column type union. In `WatchListTable.tsx`, handle `type === 'custom'` in the `renderCell` switch to render `<AlertCell>` when `column.id === 'alert'`.

Remove the separate `alert_count` column (lines 174–182) since the new `alert` column covers both display and count.

### 3.4 Modify: WatchListTable

**File:** `components/watchlist/WatchListTable.tsx`

In the `renderCell` function, add a case for the `alert` column:

```typescript
case 'alert':
  return (
    <AlertCell
      symbol={item.symbol}
      watchListId={watchListId}
      alertCount={item.alert_count}
      alerts={item.alerts || []}
      onAlertChange={onRefresh}
    />
  );
```

Pass `watchListId` and `onRefresh` as new props to `WatchListTable`.

### 3.5 Modify: Watchlist Detail Page — Add Alerts Tab

**File:** `app/watchlist/[id]/page.tsx`

Add a tab bar above the table. Three tabs:

```typescript
const [activeTab, setActiveTab] = useState<'table' | 'heatmap' | 'alerts'>(
  searchParams.get('tab') as any || 'table'
);
```

- **Table** (default): existing `WatchListTable` component
- **Heatmap**: move existing heatmap button into a tab; render heatmap inline instead of navigating to a sub-route
- **Alerts**: render a list of `AlertCard` components (reuse existing `components/alerts/AlertCard.tsx`) filtered to `watch_list_id`. Fetch via `alertAPI.listAlerts({ watch_list_id: id })`.

Tab bar styling: horizontal pills using Tailwind, consistent with the existing `ViewSwitcher` pattern.

### 3.6 Modify: Watchlist Detail Page — Bulk Alert Action

**File:** `app/watchlist/[id]/page.tsx`

Add "Set alert for all tickers" to the existing kebab/options menu in the watchlist header. On click, open a modal (reuse the same form layout as `AlertQuickPanel` create mode, but in a centered modal overlay). On submit, call the new `alertAPI.bulkCreateAlerts()` method. Show the `{ created, skipped }` result as a toast or inline message.

### 3.7 Modify: Frontend API Layer

**File:** `lib/api/alerts.ts` — Add:

```typescript
export interface AlertSummary {
  id: string;
  alert_type: string;
  is_active: boolean;
  conditions: any;
  frequency: string;
  last_triggered_at?: string;
  trigger_count: number;
}

export interface BulkCreateAlertRequest {
  watch_list_id: string;
  alert_type: string;
  conditions: any;
  frequency: 'once' | 'daily' | 'always';
  notify_email: boolean;
  notify_in_app: boolean;
}

export interface BulkCreateAlertResponse {
  created: number;
  skipped: number;
}

// Add to alertAPI object:
bulkCreateAlerts: async (data: BulkCreateAlertRequest): Promise<BulkCreateAlertResponse> => {
  const response = await api.post('/alerts/bulk', data);
  return response.data;
}
```

**File:** `lib/api/watchlist.ts` — Update `WatchListItem` interface to include:

```typescript
alerts?: AlertSummary[];
```

Update `getWatchList()` to pass `?include_alerts=true` by default (the AlertCell needs this data).

### 3.8 Modify: Navigation

**File:** `components/Header.tsx`

1. Remove the "Alerts" `<Link>` in the top nav (lines 52–59)
2. Remove the "My Alerts" link in the user dropdown menu (line ~111)
3. Keep the `BellIcon` in the header but repurpose it: link to `/watchlist?tab=alerts` instead of `/alerts`. Add unread alert count badge (fetch via `alertAPI.getAlertLogs({ is_read: false })` — count only, lightweight).

### 3.9 Add: /alerts Redirect

**File:** `app/alerts/page.tsx`

Replace the full page component with a redirect:

```typescript
import { redirect } from 'next/navigation';

export default function AlertsPage() {
  redirect('/watchlist?tab=alerts');
}
```

During the 30-day transition period, instead of instant redirect, render a banner:

```typescript
'use client';
import { useEffect } from 'react';
import { useRouter } from 'next/navigation';

export default function AlertsRedirectPage() {
  const router = useRouter();
  useEffect(() => {
    const timer = setTimeout(() => router.replace('/watchlist'), 5000);
    return () => clearTimeout(timer);
  }, [router]);

  return (
    <div className="max-w-2xl mx-auto mt-20 p-6 bg-ic-surface rounded-lg border border-ic-border text-center">
      <h2 className="text-lg font-semibold mb-2">Alerts have moved</h2>
      <p className="text-ic-text-muted mb-4">
        Alerts are now managed directly from your watchlists.
        Redirecting in 5 seconds...
      </p>
      <Link href="/watchlist" className="text-ic-primary hover:underline">
        Go to Watchlists now →
      </Link>
    </div>
  );
}
```

### 3.10 State Management

Alert state is **not** a global store. It lives on the watchlist data:

- `WatchListItem.alerts[]` and `WatchListItem.alert_count` are the source of truth
- `AlertCell` reads from row data, `AlertQuickPanel` writes via API then triggers `onAlertChange` → parent refetches watchlist data
- **Optimistic updates:** On create, immediately increment `alert_count` and append a placeholder to `alerts[]`. On delete, immediately decrement and filter. Revert on API error.
- The 30-second auto-refresh in `app/watchlist/[id]/page.tsx` (line 50) already handles stale state.

---

## 4. Implementation Plan

### Phase 1 — Backend (est. 2–3 days)

| Task | Files |
|------|-------|
| Add `AlertSummary` struct | `models/alert.go` |
| Add `Alerts` field to `WatchListItemDetail` | `models/watchlist.go` |
| Add `GetAlertSummariesForWatchList()` query | `database/alerts.go` |
| Wire `?include_alerts=true` into GetWatchList handler | `handlers/watchlist_handlers.go`, `services/watchlist_service.go` |
| Add `AlertExistsForSymbol()` query | `database/alerts.go` |
| Add `BulkCreateAlertRules` handler + route | `handlers/alert_handlers.go`, `main.go` |
| Tests for new queries and bulk endpoint | `services/alert_service_test.go`, `handlers/alert_handlers_test.go` |

### Phase 2 — Frontend Core (est. 3–4 days)

| Task | Files |
|------|-------|
| Build `AlertCell` component | `components/watchlist/AlertCell.tsx` |
| Build `AlertQuickPanel` component | `components/watchlist/AlertQuickPanel.tsx` |
| Add `custom` column type + wire AlertCell | `lib/watchlist/columns.ts`, `components/watchlist/WatchListTable.tsx` |
| Add `AlertSummary`, `bulkCreateAlerts` to API layer | `lib/api/alerts.ts`, `lib/api/watchlist.ts` |
| Update `getWatchList()` to pass `include_alerts=true` | `lib/api/watchlist.ts` |

### Phase 3 — Frontend Navigation & Tabs (est. 2 days)

| Task | Files |
|------|-------|
| Add tab bar (table / heatmap / alerts) to watchlist detail page | `app/watchlist/[id]/page.tsx` |
| Add "Alerts" tab content (list of AlertCards) | `app/watchlist/[id]/page.tsx` |
| Add "Set alert for all" to kebab menu | `app/watchlist/[id]/page.tsx` |
| Build `BulkAlertModal` component | `components/watchlist/BulkAlertModal.tsx` |
| Update Header: remove Alerts nav, update BellIcon link | `components/Header.tsx` |
| Replace /alerts page with redirect | `app/alerts/page.tsx` |

### Phase 4 — Polish & Ship (est. 1–2 days)

| Task | Files |
|------|-------|
| Feature flag: `watchlist-inline-alerts` | Config / env |
| Unread alert badge on header BellIcon | `components/Header.tsx` |
| Optimistic UI for create/delete | `AlertQuickPanel.tsx`, `AlertCell.tsx` |
| Transition banner on /alerts redirect page | `app/alerts/page.tsx` |
| E2E testing | — |

**Total estimate: 8–11 days** for one full-stack engineer.

---

## 5. Testing Strategy

**Backend:**
- Unit tests for `GetAlertSummariesForWatchList` — verify map grouping by symbol
- Unit tests for `BulkCreateAlertRules` — verify skip-duplicate logic, tier limit mid-batch stop
- Integration test: create watchlist → add tickers → bulk create alerts → GET watchlist with `include_alerts=true` → verify `alerts[]` populated

**Frontend:**
- Component tests for `AlertCell` — verify bell icon states (empty, filled+badge, pulsing)
- Component tests for `AlertQuickPanel` — verify create mode form submission, list mode edit/delete
- E2E: navigate to watchlist → click bell → create alert → verify bell state changes → delete alert → verify bell reverts

---

## 6. Rollback Plan

- Feature flag `watchlist-inline-alerts` controls all frontend changes. Disable flag → revert to current behavior instantly.
- Backend changes are additive (new query param, new endpoint). Old `/api/v1/alerts` endpoints remain unchanged. No rollback needed for backend — new code is dead if frontend doesn't call it.
- `/alerts` redirect page can be reverted to the original page component in a single commit.
