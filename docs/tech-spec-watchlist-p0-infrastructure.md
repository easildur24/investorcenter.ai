# Technical Specification: Watchlist P0 — Infrastructure & Reliability

**Document Version:** 1.0
**Author:** Engineering
**Date:** February 22, 2026
**Status:** Draft — Pending Engineering Review
**Companion Documents:** [PRD](./prd-watchlist-redesign.md) · [Full Tech Spec](./tech-spec-watchlist-redesign.md)

---

## Table of Contents

1. [Scope & Objectives](#1-scope--objectives)
2. [Current State Assessment](#2-current-state-assessment)
3. [Work Item 1: Batch Price Fetching](#3-work-item-1-batch-price-fetching)
4. [Work Item 2: Frontend Resilience — Error States & Retry](#4-work-item-2-frontend-resilience--error-states--retry)
5. [Work Item 3: Adaptive Polling & Market-Aware Refresh](#5-work-item-3-adaptive-polling--market-aware-refresh)
6. [Work Item 4: Production Deployment Verification](#6-work-item-4-production-deployment-verification)
7. [Testing Strategy](#7-testing-strategy)
8. [Deployment Plan](#8-deployment-plan)
9. [Performance Budget](#9-performance-budget)

---

## 1. Scope & Objectives

This spec covers the **P0 Infrastructure & Reliability** work required before the watchlist feature can be considered production-ready. The scope was determined by auditing the existing codebase against the PRD's P0 requirements, discarding items already implemented, and focusing on the gaps that pose real risk.

### What This Spec Covers

| # | Work Item | Why It's P0 |
|---|-----------|-------------|
| 1 | **Batch price fetching** | 50-ticker watchlist takes ~5s with sequential Polygon calls (5 concurrent × ~100ms each × 10 batches). Unacceptable latency. |
| 2 | **Frontend error states & retry** | "Failed to fetch" with no retry button or error classification is the #1 user complaint cited in the PRD. |
| 3 | **Adaptive polling & market-aware refresh** | Fixed 30s `setInterval` wastes API quota after-hours and doesn't leverage existing `useApiWithRetry` or `ErrorState` components. |
| 4 | **Production deployment verification** | Migrations may not be applied; Polygon API key may be missing; default watchlist trigger may not fire. Need a runbook. |

### What This Spec Does NOT Cover (Already Implemented)

The codebase audit revealed that several PRD P0 items are **already complete**:

| PRD Item | Status | Evidence |
|----------|--------|----------|
| P0-2: Fix test suite (`stocks` → `tickers`) | ✅ Done | `watchlist_handlers_test.go` correctly uses `INSERT INTO tickers` throughout. 26+ test cases pass. |
| P0-3: Default watchlist auto-creation | ✅ Done | Trigger `auto_create_default_watch_list` fires on `AFTER INSERT ON users`. Dashboard page (`app/watchlist/page.tsx`) also creates one if none exist. |
| P0-4: Core CRUD operations | ✅ Done | All 12 watchlist endpoints implemented and tested. Default watchlist delete protection present in handler. Watchlist count limit (3 free) enforced in `database.CreateWatchListAtomic()`. |
| Enriched data query (47-col JOIN) | ✅ Done | `GetWatchListItemsWithData` already JOINs `tickers`, `reddit_heatmap_daily`, `screener_data`, and `alert_rules`. |
| Summary metrics | ✅ Done | `computeSummaryMetrics()` computes avg IC Score, avg day change %, avg dividend yield, Reddit trending count. |
| Graceful degradation on Polygon failure | ✅ Done | `fetchRealTimePrices()` logs failures per-symbol but returns items without prices. |

---

## 2. Current State Assessment

### 2.1 Price Fetching — The N+1 Problem

**File:** `backend/services/watchlist_service.go`

The current `fetchRealTimePrices()` function creates a goroutine per item, bounded to 5 concurrent via a semaphore:

```
For 50 items, 5 concurrent: 10 rounds × ~100ms/round ≈ 1,000ms
For 100 items, 5 concurrent: 20 rounds × ~100ms/round ≈ 2,000ms
```

**Meanwhile, the codebase already has batch methods that aren't being used:**

| Method | File | Description | Used By |
|--------|------|-------------|---------|
| `GetMultipleQuotes(symbols)` | `polygon.go:296` | Separates stock/crypto, calls bulk APIs, returns `map[string]*QuoteData` | `AlertProcessor` only |
| `GetBulkStockSnapshots()` | `polygon.go:1193` | Fetches ALL US stock tickers in one call | `StockCache` only |
| `getBulkStockQuotes(symbols)` | `polygon.go:351` | Filters bulk snapshot for requested symbols | `GetMultipleQuotes` |
| `getBulkCryptoQuotes(symbols)` | `polygon.go:391` | Same for crypto | `GetMultipleQuotes` |

The `AlertProcessor` already uses `GetMultipleQuotes()` for batch fetching. The watchlist service should do the same.

### 2.2 Frontend Error Handling

**Existing infrastructure (unused by watchlist):**

| Component/Hook | File | Capability |
|----------------|------|------------|
| `ErrorState` | `components/ui/ErrorState.tsx` | 3 variants: inline, compact, full-page. Supports retry button, support link. |
| `useApiWithRetry` | `lib/hooks/useApiWithRetry.ts` | Exponential backoff (1s, 2s, 4s), abort support, retry count tracking. 26 tests. |
| `useToast` | `lib/hooks/useToast.tsx` | Toast notifications: success, error, warning, info. Auto-dismiss. |

**Current watchlist error handling:**

```typescript
// app/watchlist/[id]/page.tsx — current pattern
catch (err: any) {
  setError(err.message || 'Failed to load watch list');
}

// Rendered as:
{error && (
  <div className="mb-4 p-3 bg-red-500/10 border border-red-500/30 ...">
    {error}
  </div>
)}
```

Problems:
- No error classification (network vs auth vs server vs rate limit)
- No retry button — user must manually refresh the page
- No distinction between stale-data-with-error vs no-data
- Raw Tailwind colors instead of `ic-*` tokens in the error banner
- `useApiWithRetry` and `ErrorState` exist but aren't wired into the watchlist page

### 2.3 Polling Strategy

**Current:** Fixed `setInterval(loadWatchList, 30000)` regardless of:
- Market hours (stocks don't need 30s refresh at 2 AM)
- Asset composition (crypto-only watchlists should poll faster)
- Tab visibility (polling continues when tab is hidden)
- Error state (keeps polling even after auth expiry)

### 2.4 Production Readiness

The PRD's P0-1 flagged that watchlists return "Failed to fetch" in production. Likely causes:
- Migrations 010/012 not applied
- `POLYGON_API_KEY` not in `app-secrets` (code silently falls back to `"demo"`)
- Database in "mock mode" (server starts even if DB connection fails)

---

## 3. Work Item 1: Batch Price Fetching

**Priority:** P0 — Directly impacts page load time
**Estimated effort:** 0.5 day
**Risk:** Low — reuses existing tested code path

### 3.1 Change Summary

Replace `fetchRealTimePrices()` in `watchlist_service.go` to use the existing `GetMultipleQuotes()` batch method instead of per-item `GetQuote()` calls.

### 3.2 Implementation

**File:** `backend/services/watchlist_service.go`

Replace the current `fetchRealTimePrices` function:

```go
// fetchRealTimePrices populates price fields using batch Polygon API.
// Uses GetMultipleQuotes() for a single bulk request instead of N sequential calls.
// Failures are logged but do not prevent items from being returned (graceful degradation).
func fetchRealTimePrices(items []models.WatchListItemDetail, contextLabel string) {
    if len(items) == 0 {
        return
    }

    // Collect symbols
    symbols := make([]string, len(items))
    for i := range items {
        symbols[i] = items[i].Symbol
    }

    // Single batch call (handles stock/crypto split internally)
    polygonClient := NewPolygonClient()
    quotes, err := polygonClient.GetMultipleQuotes(symbols)
    if err != nil {
        log.Printf("Warning: Batch price fetch failed for %s: %v", contextLabel, err)
        return // graceful degradation — items returned without prices
    }

    // Merge quotes into items
    priceHits, priceMisses := 0, 0
    for i := range items {
        quote, ok := quotes[items[i].Symbol]
        if !ok || quote == nil {
            priceMisses++
            continue
        }
        priceHits++

        currentPrice := quote.Price.InexactFloat64()
        items[i].CurrentPrice = &currentPrice

        if quote.Change.IsPositive() || quote.Change.IsNegative() {
            change := quote.Change.InexactFloat64()
            changePercent := quote.ChangePercent.InexactFloat64()
            items[i].PriceChange = &change
            items[i].PriceChangePct = &changePercent

            prevClose := quote.Price.Sub(quote.Change).InexactFloat64()
            items[i].PrevClose = &prevClose
        }

        if quote.Volume > 0 {
            volume := int64(quote.Volume)
            items[i].Volume = &volume
        }
    }

    if priceMisses > 0 {
        log.Printf("Batch price (%s): %d hits, %d misses out of %d symbols",
            contextLabel, priceHits, priceMisses, len(symbols))
    }
}
```

### 3.3 What Changes

| Aspect | Before | After |
|--------|--------|-------|
| API calls for 50 tickers | 50 individual (5 concurrent) | 1 bulk snapshot + filter |
| Latency (50 tickers) | ~1,000ms | ~200ms |
| Latency (100 tickers) | ~2,000ms | ~200ms (same bulk call) |
| Crypto handling | Individual `GetQuote()` per symbol | `GetMultipleQuotes()` handles crypto separately via cache/bulk |
| Error granularity | Per-symbol logging | Aggregate hit/miss count |
| Code complexity | Goroutine + semaphore pattern | Single function call |

### 3.4 What Stays The Same

- `GetMultipleQuotes()` method — no changes needed, already tested
- `GetBulkStockSnapshots()` / `getBulkStockQuotes()` — used internally by `GetMultipleQuotes()`
- `QuoteData` struct — same price/change/volume fields
- Graceful degradation — batch failure returns items without prices, identical user experience

### 3.5 Risk: Bulk Snapshot Fetches All Tickers

`GetBulkStockSnapshots()` calls `/v2/snapshot/locale/us/markets/stocks/tickers` without a `?tickers=` filter, fetching **all** US stock snapshots (~5,600 tickers). It then filters in `getBulkStockQuotes()` to the requested symbols.

This is the same approach used by `AlertProcessor` and `StockCache` today. For a watchlist of 10-100 tickers, we're over-fetching. However:

- The Polygon response is ~2MB compressed, completes in ~200ms
- It's **one API call** vs 50 individual calls
- The `StockCache` already fetches this on a timer, so Polygon rate limits are not a concern
- If we need to optimize further (Phase 2), we can switch to the `?tickers=AAPL,MSFT,...` filter param — but that's a separate optimization with its own URL-length constraints

---

## 4. Work Item 2: Frontend Resilience — Error States & Retry

**Priority:** P0 — "Failed to fetch" with no recovery is the #1 PRD complaint
**Estimated effort:** 1-2 days
**Risk:** Low — leveraging existing `ErrorState` and `useApiWithRetry` components

### 4.1 Change Summary

Wire the existing `ErrorState` component and `useApiWithRetry` hook into the watchlist dashboard and detail pages. Classify errors and provide appropriate actions.

### 4.2 Error Classification

**New utility:** `lib/api/errorClassifier.ts`

```typescript
export type WatchlistErrorType =
  | 'network'
  | 'auth'
  | 'not_found'
  | 'rate_limit'
  | 'server';

export interface ClassifiedError {
  type: WatchlistErrorType;
  title: string;
  message: string;
  retryable: boolean;
}

export function classifyError(error: Error): ClassifiedError {
  const msg = error.message.toLowerCase();

  if (msg.includes('session expired') || msg.includes('401') || msg.includes('log in')) {
    return {
      type: 'auth',
      title: 'Session expired',
      message: 'Please sign in again to view your watchlists.',
      retryable: false,
    };
  }

  if (msg.includes('429') || msg.includes('too many')) {
    return {
      type: 'rate_limit',
      title: 'Too many requests',
      message: 'Please wait a moment before trying again.',
      retryable: true,
    };
  }

  if (msg.includes('404') || msg.includes('not found')) {
    return {
      type: 'not_found',
      title: 'Watchlist not found',
      message: 'This watchlist may have been deleted or you may not have access.',
      retryable: false,
    };
  }

  if (msg.includes('failed to fetch') || msg.includes('networkerror') || msg.includes('load failed')) {
    return {
      type: 'network',
      title: 'Unable to connect',
      message: 'Check your internet connection and try again.',
      retryable: true,
    };
  }

  return {
    type: 'server',
    title: 'Something went wrong',
    message: "We're looking into it. Please try again in a moment.",
    retryable: true,
  };
}
```

### 4.3 Watchlist Detail Page Changes

**File:** `app/watchlist/[id]/page.tsx`

Replace the raw `setInterval` + `try/catch` pattern with `useApiWithRetry` for the initial load, and preserve the existing polling for refresh cycles.

**Key changes:**

1. **Initial load uses `useApiWithRetry`** — provides automatic retry with exponential backoff on first load failure.

2. **Error state replaces raw string** — render `<ErrorState>` component with classified error, retry button, and (for auth errors) a sign-in link.

3. **Stale-while-error** — if data was previously loaded and a refresh fails, keep the stale data visible and show a non-blocking inline error banner instead of replacing the table.

4. **Remove raw Tailwind from error banner** — replace `bg-red-500/10 border border-red-500/30` with `ErrorState` component using `ic-*` tokens.

```typescript
// Pseudocode for the revised data-loading pattern:

const {
  data: watchListData,
  error: loadError,
  loading,
  execute: loadWatchList,
  isRetrying,
  retryCount,
} = useApiWithRetry(
  () => watchListAPI.getWatchList(watchListId),
  {
    maxRetries: 3,
    retryDelay: 1000,
    onSuccess: (data) => {
      setWatchList(data);
      setRefreshError(null);
    },
    onError: (err) => {
      // If we already have data, this is a refresh failure — don't replace the table
      if (watchList) {
        setRefreshError(err);
      }
    },
  }
);

// Render logic:
if (loading && !watchList) {
  return <LoadingSkeleton />;
}

if (!watchList && loadError) {
  const classified = classifyError(loadError);
  return (
    <ErrorState
      message={classified.message}
      onRetry={classified.retryable ? loadWatchList : undefined}
      variant="default"
    />
  );
}

// Table renders here — even if there's a refresh error
{refreshError && (
  <ErrorState
    message="Prices may be outdated. Retrying..."
    variant="inline"
    onRetry={() => loadWatchList()}
  />
)}
```

### 4.4 Watchlist Dashboard Page Changes

**File:** `app/watchlist/page.tsx`

Apply the same pattern: `useApiWithRetry` for the initial `getWatchLists()` call, with `ErrorState` for failure display.

### 4.5 Integration with Existing ErrorState Component

The existing `ErrorState` component at `components/ui/ErrorState.tsx` already supports:
- `variant="inline"` — for refresh errors above the table (stale-while-error)
- `variant="compact"` — for errors within modals or panels
- `variant="default"` — for full-page errors (no data loaded at all)
- `onRetry` prop — renders a "Try again" button
- `showSupport` prop — renders a "Contact support" link

No changes needed to the `ErrorState` component itself.

---

## 5. Work Item 3: Adaptive Polling & Market-Aware Refresh

**Priority:** P0 — Wastes API quota and provides poor UX
**Estimated effort:** 1 day
**Risk:** Low

### 5.1 Change Summary

Replace the fixed 30-second `setInterval` with a market-aware polling strategy that adjusts intervals based on market hours, asset composition, and tab visibility.

### 5.2 Market Hours Utility

**New file:** `lib/utils/marketHours.ts`

```typescript
export type MarketState = 'open' | 'pre_market' | 'after_hours' | 'closed';

export function getMarketState(): MarketState {
  const now = new Date();
  const et = new Intl.DateTimeFormat('en-US', {
    timeZone: 'America/New_York',
    hour: 'numeric',
    minute: 'numeric',
    hour12: false,
    weekday: 'short',
  }).formatToParts(now);

  const weekday = et.find((p) => p.type === 'weekday')?.value ?? '';
  const hour = parseInt(et.find((p) => p.type === 'hour')?.value ?? '0');
  const minute = parseInt(et.find((p) => p.type === 'minute')?.value ?? '0');
  const timeMinutes = hour * 60 + minute;

  // Weekend
  if (weekday === 'Sat' || weekday === 'Sun') return 'closed';

  // Pre-market: 4:00-9:30 ET
  if (timeMinutes >= 240 && timeMinutes < 570) return 'pre_market';

  // Market open: 9:30-16:00 ET
  if (timeMinutes >= 570 && timeMinutes < 960) return 'open';

  // After hours: 16:00-20:00 ET
  if (timeMinutes >= 960 && timeMinutes < 1200) return 'after_hours';

  return 'closed';
}

/**
 * Returns the polling interval in milliseconds based on market state
 * and whether the watchlist contains crypto assets.
 */
export function getPollingInterval(hasCrypto: boolean, hasStocks: boolean): number {
  const state = getMarketState();

  // Crypto-only watchlists: always fast polling
  if (hasCrypto && !hasStocks) {
    return state === 'open' ? 5_000 : 15_000;
  }

  // Stock or mixed watchlists
  switch (state) {
    case 'open':
      return 15_000;       // 15s during market hours
    case 'pre_market':
    case 'after_hours':
      return 30_000;       // 30s pre/post market
    case 'closed':
      return hasCrypto ? 15_000 : 60_000;  // 60s for stocks, 15s if crypto mixed in
  }
}
```

### 5.3 Polling Hook

**New file:** `lib/hooks/useWatchlistPolling.ts`

```typescript
import { useEffect, useRef, useCallback } from 'react';
import { getPollingInterval } from '@/lib/utils/marketHours';

interface UseWatchlistPollingOptions {
  /** Function to call on each poll cycle. */
  onPoll: () => Promise<void>;
  /** Whether the watchlist contains crypto assets. */
  hasCrypto: boolean;
  /** Whether the watchlist contains stock assets. */
  hasStocks: boolean;
  /** Disable polling entirely (e.g., during error state). */
  enabled?: boolean;
}

/**
 * Market-aware polling hook that:
 * - Adjusts interval based on market hours and asset composition
 * - Pauses when the tab is not visible
 * - Resumes with an immediate fetch when tab becomes visible
 * - Re-evaluates interval every cycle (market state changes during session)
 */
export function useWatchlistPolling({
  onPoll,
  hasCrypto,
  hasStocks,
  enabled = true,
}: UseWatchlistPollingOptions) {
  const timeoutRef = useRef<ReturnType<typeof setTimeout>>();
  const onPollRef = useRef(onPoll);
  onPollRef.current = onPoll;

  const scheduleNext = useCallback(() => {
    if (!enabled) return;
    const interval = getPollingInterval(hasCrypto, hasStocks);
    timeoutRef.current = setTimeout(async () => {
      await onPollRef.current();
      scheduleNext(); // Re-evaluate interval each cycle
    }, interval);
  }, [hasCrypto, hasStocks, enabled]);

  // Start polling
  useEffect(() => {
    if (!enabled) return;
    scheduleNext();
    return () => {
      if (timeoutRef.current) clearTimeout(timeoutRef.current);
    };
  }, [scheduleNext, enabled]);

  // Pause on tab hidden, resume + immediate fetch on visible
  useEffect(() => {
    const handleVisibility = () => {
      if (document.hidden) {
        if (timeoutRef.current) clearTimeout(timeoutRef.current);
      } else {
        // Tab became visible — immediate refresh then resume schedule
        onPollRef.current().then(scheduleNext);
      }
    };
    document.addEventListener('visibilitychange', handleVisibility);
    return () => document.removeEventListener('visibilitychange', handleVisibility);
  }, [scheduleNext]);
}
```

### 5.4 Integration into Watchlist Detail Page

**File:** `app/watchlist/[id]/page.tsx`

Replace:
```typescript
useEffect(() => {
  loadWatchList();
  const interval = setInterval(loadWatchList, 30000);
  return () => clearInterval(interval);
}, [loadWatchList]);
```

With:
```typescript
const hasCrypto = watchList?.items.some((i) => i.symbol.startsWith('X:')) ?? false;
const hasStocks = watchList?.items.some((i) => !i.symbol.startsWith('X:')) ?? false;

// Initial load via useApiWithRetry (see Work Item 2)
useEffect(() => { loadWatchList(); }, [loadWatchList]);

// Adaptive polling for refresh
useWatchlistPolling({
  onPoll: loadWatchList,
  hasCrypto,
  hasStocks,
  enabled: !!watchList && !loadError, // Don't poll if initial load failed
});
```

### 5.5 Market Status Display

**New component:** `components/watchlist/MarketStatusIndicator.tsx`

A small badge rendered in the watchlist table footer:

```typescript
interface MarketStatusIndicatorProps {
  lastUpdated: Date | null;
}

export default function MarketStatusIndicator({ lastUpdated }: MarketStatusIndicatorProps) {
  const state = getMarketState();

  const config = {
    open:         { dot: 'bg-ic-positive', label: 'Market Open' },
    pre_market:   { dot: 'bg-ic-warning',  label: 'Pre-Market' },
    after_hours:  { dot: 'bg-ic-warning',  label: 'After Hours' },
    closed:       { dot: 'bg-ic-text-dim', label: 'Market Closed' },
  }[state];

  return (
    <div className="flex items-center gap-2 text-xs text-ic-text-dim">
      <span className={`inline-block w-2 h-2 rounded-full ${config.dot}`} />
      <span>{config.label}</span>
      {lastUpdated && (
        <span>
          · Prices as of{' '}
          {lastUpdated.toLocaleTimeString('en-US', {
            hour: 'numeric',
            minute: '2-digit',
            second: '2-digit',
            timeZoneName: 'short',
          })}
        </span>
      )}
    </div>
  );
}
```

Rendered below the watchlist table:

```tsx
<MarketStatusIndicator lastUpdated={lastRefreshTime} />
```

---

## 6. Work Item 4: Production Deployment Verification

**Priority:** P0 — Blocking production readiness
**Estimated effort:** 0.5 day
**Risk:** Medium — may uncover issues requiring additional fixes

### 6.1 Pre-Deployment Verification Runbook

Execute these steps in order before deploying watchlist changes to production.

#### Step 1: Verify database migrations are applied

```bash
kubectl exec -it deploy/investorcenter-backend -n investorcenter -- \
  psql $DATABASE_URL -c "
    SELECT table_name FROM information_schema.tables
    WHERE table_schema = 'public'
    AND table_name IN (
      'watch_lists', 'watch_list_items',
      'alert_rules', 'alert_logs',
      'subscription_plans', 'user_subscriptions',
      'notification_preferences', 'notifications'
    )
    ORDER BY table_name;
  "
```

**Expected:** 8 tables. If any missing, apply migrations:

```bash
# Watchlist tables
kubectl exec -it deploy/investorcenter-backend -n investorcenter -- \
  psql $DATABASE_URL -f /app/migrations/010_watchlist_tables.sql

# Alert + subscription tables
kubectl exec -it deploy/investorcenter-backend -n investorcenter -- \
  psql $DATABASE_URL -f /app/migrations/012_alert_system.sql
```

#### Step 2: Verify trigger functions exist

```bash
kubectl exec -it deploy/investorcenter-backend -n investorcenter -- \
  psql $DATABASE_URL -c "
    SELECT routine_name FROM information_schema.routines
    WHERE routine_schema = 'public'
    AND routine_name IN (
      'update_watch_lists_updated_at',
      'check_watch_list_item_limit',
      'create_default_watch_list',
      'update_alert_rules_updated_at',
      'create_default_notification_preferences'
    )
    ORDER BY routine_name;
  "
```

**Expected:** 5 functions.

#### Step 3: Verify `tickers` table exists and is populated

```bash
kubectl exec -it deploy/investorcenter-backend -n investorcenter -- \
  psql $DATABASE_URL -c "
    SELECT
      (SELECT COUNT(*) FROM tickers) as ticker_count,
      (SELECT COUNT(*) FROM tickers WHERE asset_type = 'CS') as stock_count,
      (SELECT COUNT(*) FROM tickers WHERE symbol LIKE 'X:%') as crypto_count;
  "
```

**Expected:** ~25,000 total, ~5,600 stocks, ~4,200+ crypto.

#### Step 4: Verify Polygon API key is set

```bash
kubectl exec -it deploy/investorcenter-backend -n investorcenter -- \
  env | grep POLYGON_API_KEY
```

**Expected:** Non-empty value (not "demo"). If missing, verify `app-secrets` has `polygon-api-key`:

```bash
kubectl get secret app-secrets -n investorcenter -o jsonpath='{.data.polygon-api-key}' | base64 -d
```

**Critical:** The Go code silently falls back to `apiKey = "demo"` if the env var is missing (`polygon.go:31`). This means the server starts and returns data, but with demo/stale prices — a silent data quality issue.

#### Step 5: Verify screener_data materialized view exists and is fresh

```bash
kubectl exec -it deploy/investorcenter-backend -n investorcenter -- \
  psql $DATABASE_URL -c "
    SELECT COUNT(*) as row_count,
           MAX(refreshed_at) as last_refresh
    FROM screener_data;
  "
```

**Expected:** ~10,000 rows, `last_refresh` within the last 24 hours.

#### Step 6: Test API endpoints via curl

```bash
# Get a JWT token (replace with test credentials)
TOKEN=$(kubectl exec -it deploy/investorcenter-backend -n investorcenter -- \
  curl -s -X POST localhost:8080/api/v1/auth/login \
    -H 'Content-Type: application/json' \
    -d '{"email":"<test_email>","password":"<test_password>"}' | jq -r '.access_token')

# Test list watchlists
kubectl exec -it deploy/investorcenter-backend -n investorcenter -- \
  curl -s -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer $TOKEN" \
    localhost:8080/api/v1/watchlists

# Test create watchlist
kubectl exec -it deploy/investorcenter-backend -n investorcenter -- \
  curl -s -o /dev/null -w "%{http_code}" \
    -X POST -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d '{"name":"Deployment Test"}' \
    localhost:8080/api/v1/watchlists
```

**Expected:** HTTP 200 for list, HTTP 201 for create. If 500: check pod logs.

#### Step 7: Verify pod logs for errors

```bash
kubectl logs -n investorcenter deploy/investorcenter-backend --tail=200 \
  | grep -iE "watchlist|watch_list|error|panic|POLYGON"
```

**Look for:**
- `"database connection failed"` → DB connectivity issue
- `"POLYGON_API_KEY"` or `"demo"` → API key missing
- `"panic"` → Crash in handler initialization

### 6.2 Backfill Default Watchlists for Existing Users

The `create_default_watch_list` trigger only fires on new user creation. Existing users who signed up before migration 010 may not have a default watchlist.

```sql
-- One-time backfill: safe to run multiple times (INSERT ... WHERE NOT EXISTS)
INSERT INTO watch_lists (user_id, name, description, is_default, display_order)
SELECT u.id, 'My Watch List', 'Default watch list', TRUE, 0
FROM users u
WHERE NOT EXISTS (
    SELECT 1 FROM watch_lists wl
    WHERE wl.user_id = u.id AND wl.is_default = TRUE
);
```

Run this after verifying migrations are applied:

```bash
kubectl exec -it deploy/investorcenter-backend -n investorcenter -- \
  psql $DATABASE_URL -c "<above SQL>"
```

### 6.3 Health Check Enhancement

The current health check (`GET /health`) only verifies the database connection. Extend it to verify migration state:

**File:** `backend/main.go` — health endpoint

Add a check for critical table existence:

```go
// Inside the health endpoint handler, after database health check:
var tableCount int
err = database.DB.QueryRow(`
    SELECT COUNT(*) FROM information_schema.tables
    WHERE table_schema = 'public'
    AND table_name IN ('watch_lists', 'watch_list_items', 'alert_rules', 'tickers')
`).Scan(&tableCount)

if err != nil || tableCount < 4 {
    response["migrations"] = "incomplete"
    response["expected_tables"] = 4
    response["found_tables"] = tableCount
} else {
    response["migrations"] = "complete"
}
```

This gives deployment scripts a single endpoint to verify both connectivity and schema state.

---

## 7. Testing Strategy

### 7.1 Backend Tests

#### Batch Price Fetching

**File:** `backend/services/watchlist_service_test.go` (new or extend existing)

| Test | What It Validates |
|------|-------------------|
| `TestFetchRealTimePrices_BatchCall` | Uses `GetMultipleQuotes()` instead of per-item `GetQuote()` |
| `TestFetchRealTimePrices_EmptyItems` | No API call for empty slice |
| `TestFetchRealTimePrices_PartialFailure` | Items without prices still returned |
| `TestFetchRealTimePrices_TotalFailure` | All items returned without prices, no panic |
| `TestFetchRealTimePrices_MixedStockCrypto` | Both stock and crypto symbols handled |

#### Health Check Enhancement

| Test | What It Validates |
|------|-------------------|
| `TestHealthCheck_MigrationsComplete` | Returns `"migrations": "complete"` when all tables exist |
| `TestHealthCheck_MigrationsMissing` | Returns `"migrations": "incomplete"` with count |

### 7.2 Frontend Tests

#### Error Classification

**File:** `lib/api/__tests__/errorClassifier.test.ts`

| Test | Input | Expected Type |
|------|-------|---------------|
| Network error | `"Failed to fetch"` | `network` |
| Auth error | `"Session expired. Please log in again."` | `auth` |
| Not found | `"Watch list not found"` | `not_found` |
| Rate limit | `"429 Too many requests"` | `rate_limit` |
| Generic server error | `"Internal server error"` | `server` |
| Unknown error | `"Something unexpected"` | `server` |

#### Market Hours Utility

**File:** `lib/utils/__tests__/marketHours.test.ts`

| Test | What It Validates |
|------|-------------------|
| Market open at 10:00 ET weekday | Returns `'open'` |
| Pre-market at 7:00 ET weekday | Returns `'pre_market'` |
| After-hours at 17:00 ET weekday | Returns `'after_hours'` |
| Closed on Saturday | Returns `'closed'` |
| Polling interval: stocks during market hours | Returns `15_000` |
| Polling interval: crypto-only, closed | Returns `15_000` |
| Polling interval: stocks, closed | Returns `60_000` |
| Polling interval: mixed, closed | Returns `15_000` |

#### Polling Hook

**File:** `lib/hooks/__tests__/useWatchlistPolling.test.ts`

| Test | What It Validates |
|------|-------------------|
| Calls onPoll at market-appropriate interval | Uses fake timers |
| Pauses when document is hidden | `visibilitychange` event |
| Resumes with immediate fetch when visible | Immediate call then reschedule |
| Doesn't poll when `enabled=false` | No timer created |
| Cleans up on unmount | No dangling timers |

### 7.3 Existing Test Coverage (No Changes Needed)

The following tests already cover functionality that the original tech spec flagged as needing fixes:

- `TestCreateWatchListCountLimit` — 3-watchlist free tier limit ✅
- `TestDeleteDefaultWatchList` — default watchlist delete protection ✅
- `TestFreeTierLimit` — 10-item limit enforcement via DB trigger ✅
- `TestGetWatchListWithScreenerData` — enriched 47-column query ✅
- `TestGetWatchListWithAlertCount` — alert counts scoped to watchlist ✅
- `TestGetWatchListSummaryMetrics` — summary computation ✅
- `TestGetMultipleQuotes_EmptySymbols` — batch quote edge case ✅

---

## 8. Deployment Plan

### Phase 1: Backend (Batch Prices + Health Check)

```
1. Deploy backend with:
   - Updated fetchRealTimePrices() using GetMultipleQuotes()
   - Enhanced health check with migration verification
2. Run verification runbook (§6.1 Steps 1-7)
3. Apply default watchlist backfill (§6.2) if needed
4. Monitor: GET /health returns {"migrations": "complete"}
5. Monitor: API response times for GET /watchlists/:id
   - Target p95 < 800ms for 50-item watchlist (down from ~5s)
```

### Phase 2: Frontend (Error States + Adaptive Polling)

```
1. Deploy frontend with:
   - Error classification utility
   - useApiWithRetry integration in watchlist pages
   - ErrorState component wired into watchlist pages
   - useWatchlistPolling hook replacing setInterval
   - MarketStatusIndicator component
2. Verify: load watchlist page → see market status indicator
3. Verify: disconnect network → see ErrorState with retry button
4. Verify: reconnect → retry succeeds, data loads
5. Monitor: no "Failed to fetch" raw error messages in user-facing UI
```

### Rollback Plan

- **Backend:** `kubectl rollout undo deployment/investorcenter-backend -n investorcenter`
  - Reverts to per-item price fetching. Higher latency but functionally identical.
- **Frontend:** `kubectl rollout undo deployment/investorcenter-frontend -n investorcenter`
  - Reverts to 30s fixed polling and raw error display.
- **Database:** No schema changes — nothing to rollback.

---

## 9. Performance Budget

### API Response Times (After Batch Optimization)

| Endpoint | Watchlist Size | Target p50 | Target p95 | Current (est.) |
|----------|---------------|-----------|-----------|----------------|
| `GET /watchlists` (list) | N/A | 50ms | 200ms | ~50ms |
| `GET /watchlists/:id` (detail) | 10 items | 250ms | 500ms | ~300ms |
| `GET /watchlists/:id` (detail) | 50 items | 300ms | 600ms | ~1,200ms |
| `GET /watchlists/:id` (detail) | 100 items | 350ms | 800ms | ~2,200ms |

After batch optimization, the Polygon call becomes a constant ~200ms regardless of watchlist size. The variable factor is the DB query (scales linearly but all JOINs are index-backed, ~1ms per item).

### Polling Efficiency

| Scenario | Current | After |
|----------|---------|-------|
| Stocks during market hours | 30s fixed | 15s adaptive |
| Stocks after hours | 30s fixed | 60s adaptive |
| Crypto 24/7 | 30s fixed | 5-15s adaptive |
| Tab hidden | 30s (continues) | Paused |
| API quota per hour (50-item watchlist) | 120 calls | 40-240 (adaptive) |

### Frontend Bundle Size

| Addition | Estimated Size | Notes |
|----------|---------------|-------|
| `errorClassifier.ts` | ~0.5KB | Utility only |
| `marketHours.ts` | ~0.5KB | Utility only |
| `useWatchlistPolling.ts` | ~0.5KB | Hook only |
| `MarketStatusIndicator.tsx` | ~1KB | New component |
| **Total delta** | **~2.5KB** | Negligible |

`ErrorState` and `useApiWithRetry` are already in the bundle (used elsewhere) — zero additional cost.

---

*End of Technical Specification*
