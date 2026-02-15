# Screener Improvements: Implementation Plan

## Context

The InvestorCenter stock screener has 50+ metrics computed in the database but only exposes 6 filters and 9 table columns. IC Score and Beta are hardcoded to `0.0` due to placeholder query mappings. Filters don't persist in the URL, there's no CSV export, and no column customization. The product plan and technical spec are complete (see `docs/screener-product-plan.md` and `docs/screener-technical-spec.md`). This plan breaks implementation into shippable increments.

---

## Phase 1: Fix Critical Bugs (1 day)

**Goal:** IC Score and Beta display actual values. Zero backend schema changes.

### PR 1: Fix IC Score and Beta hardcoded zeros

**Files to modify:**
- `backend/database/screener.go` (lines 23-24, 166-167)
- `backend/handlers/screener.go` (no changes needed — structs already have `Beta`/`ICScore` as `*float64`)

**Changes:**
1. Fix sort column mapping (lines 23-24):
   - `"beta": "roe"` → `"beta": "beta"`
   - `"ic_score": "market_cap"` → `"ic_score": "ic_score"`
2. Fix SELECT query (lines 166-167):
   - `0.0 as beta` → `beta`
   - `0.0 as ic_score` → `ic_score`

**Tests to add:**
- `backend/database/screener_test.go` — table-driven tests for `ValidScreenerSortColumns` mapping correctness
- Verify IC Score and Beta columns are not nil for stocks that have computed scores

**Verification:** `cd backend && go test ./... -v` + `make check`

---

## Phase 2: Expand the Materialized View (2-3 days)

**Goal:** Make 20+ additional metrics available for screening by expanding the `screener_data` view.

### PR 2: Migration to expand screener_data materialized view

**Files to create:**
- `ic-score-service/migrations/017_expand_screener_materialized_view.sql`

**What it does:**
- Drops and recreates `screener_data` with additional columns from the existing `fundamental_metrics_extended` lateral join (same join, more columns): `roa`, `gross_margin`, `net_margin`, `debt_to_equity`, `current_ratio`, `eps_growth_yoy`, `payout_ratio`, `consecutive_dividend_years`, `dcf_upside_percent`
- Expands the `ic_scores` lateral join to include all 10 sub-factor scores: `value_score`, `growth_score`, `profitability_score`, `financial_health_score`, `momentum_score`, `analyst_consensus_score`, `insider_activity_score`, `institutional_score`, `news_sentiment_score`, `technical_score`, plus `rating`, `sector_percentile`, `lifecycle_stage`
- Adds partial indexes on new filterable columns (`WHERE col IS NOT NULL`)

**Verification:**
- Apply migration on local DB: `psql investorcenter_db -f ic-score-service/migrations/017_expand_screener_materialized_view.sql`
- Run `SELECT refresh_screener_data();` and verify refresh time <2 minutes
- Spot-check: `SELECT symbol, ic_score, beta, roe, gross_margin, value_score FROM screener_data WHERE ic_score IS NOT NULL LIMIT 10;`

---

## Phase 3: Backend Filter Registry + New Params (3-4 days)

**Goal:** Refactor the Go backend to support 20+ filters via a declarative registry pattern instead of hardcoded if-blocks.

### PR 3: Filter registry and expanded API params

**Files to create:**
- `backend/database/filter_registry.go` — `FilterDef` struct, `FilterRegistry` map, `BuildWhereClause()` function

**Files to modify:**
- `backend/models/stock.go` — Expand `ScreenerStock` struct with new fields (`Roa`, `GrossMargin`, `NetMargin`, `DebtToEquity`, `CurrentRatio`, `EpsGrowthYoy`, `PayoutRatio`, `ConsecutiveDividendYears`, `DcfUpsidePercent`, `ICRating`, plus 10 sub-factor score fields). Expand `ScreenerParams` with corresponding min/max params.
- `backend/handlers/screener.go` — Parse new query params (use a loop over the registry instead of 30 individual `ParseFloat` blocks). Add `industries` param (comma-separated, same pattern as `sectors`).
- `backend/database/screener.go` — Replace hardcoded WHERE clause building with `FilterRegistry.BuildWhereClause(params)`. Update SELECT to read all new columns from the view. Expand `ValidScreenerSortColumns` with all new columns.

**Tests to add:**
- `backend/database/filter_registry_test.go` — test `BuildWhereClause` with various param combinations, edge cases (nil params, empty arrays, conflicting min/max)
- `backend/handlers/screener_test.go` — test query param parsing for new filters, test invalid values are silently ignored

**Verification:** `cd backend && go test ./... -v` + manual curl tests:
```bash
curl "localhost:8080/api/v1/screener/stocks?roe_min=15&gross_margin_min=40&limit=10" | jq '.data[0]'
curl "localhost:8080/api/v1/screener/stocks?ic_score_min=70&sort=ic_score&order=desc&limit=5" | jq '.data[].ic_score'
```

---

## Phase 4: URL State Management (2-3 days)

**Goal:** Filters persist in the URL. Pages are bookmarkable and shareable.

### PR 4: Add nuqs for URL ↔ filter state sync

**Dependencies to install:** `nuqs`

**Files to create:**
- `lib/screener/url-params.ts` — `nuqs` parser definitions for all filter params
- `lib/screener/presets.ts` — Preset definitions (Value, Growth, Quality, Dividend, Undervalued) as param objects
- `lib/hooks/useScreenerParams.ts` — Custom hook wrapping `useQueryStates` with helpers: `applyPreset()`, `clearAll()`, `activeFilterCount`

**Files to modify:**
- `package.json` — Add `nuqs`
- `app/screener/page.tsx` — Replace `useState` filter management with `useScreenerParams()` hook. Remove client-side filtering logic entirely (filters now go to API).

**Key behavior:**
- `useQueryStates` with `{ shallow: true, throttleMs: 150, history: 'replace' }`
- Preset click uses `history: 'push'` (so Back goes to previous preset)
- URL format: `/screener?sectors=Technology,Healthcare&pe_max=15&sort=ic_score&order=desc`

**Tests to add:**
- `lib/hooks/useScreenerParams.test.ts` — test param serialization/deserialization, preset expansion, clearAll resets

**Verification:** Start dev server (`make dev`), apply filters in UI, refresh page — filters should persist. Copy URL, open in incognito — same filters applied.

---

## Phase 5: Server-Side Filtering Migration (2-3 days)

**Goal:** Replace the "fetch all 20K rows, filter client-side" pattern with server-side API calls per filter change.

### PR 5: Migrate to server-side filtering with SWR

**Dependencies to install:** `swr`

**Files to create:**
- `lib/screener/api.ts` — `fetchScreenerData(params)` function that builds the query string from URL params and calls `GET /api/v1/screener/stocks`

**Files to modify:**
- `package.json` — Add `swr`
- `app/screener/page.tsx` — Replace the single `useEffect` fetch + client-side `applyFilters()` with SWR:
  ```typescript
  const { data, isLoading } = useSWR(
    buildScreenerUrl(debouncedParams),
    fetcher,
    { keepPreviousData: true }
  );
  ```
- Remove: `stocks` state, `filteredStocks` state, `applyFilters` callback, client-side `.filter()` and `.sort()` logic
- Keep: pagination state synced with URL params

**Key behavior:**
- `keepPreviousData: true` — shows stale results while loading new ones (no flash of empty state)
- 150ms debounce on params (from `nuqs` throttle) prevents excessive API calls
- Optimistic UI: current results stay visible at reduced opacity during load
- Default limit changes from 20000 to 50 (server handles pagination)

**Tests to add:**
- `app/screener/screener.test.tsx` — test that filter changes trigger API calls with correct params, test loading state, test error state

**Verification:** Open Network tab, change a filter — should see `GET /api/v1/screener/stocks?...` with the filter params. No 20K-row payload on initial load.

---

## Phase 6: New Screener UI Components (4-5 days)

**Goal:** Replace the monolithic 599-line `page.tsx` with a modular component tree. Add collapsible filter sections, column customization, and the expanded filter set.

### PR 6: Screener component architecture

**Files to create:**
- `components/screener/ScreenerClient.tsx` — Top-level client component (owns URL state + SWR data)
- `components/screener/ScreenerToolbar.tsx` — Preset buttons, Share URL, Export CSV
- `components/screener/FilterPanel.tsx` — Collapsible sidebar with filter sections
- `components/screener/FilterSection.tsx` — Collapsible group (uses `<details>`/`<summary>`) with active filter count badge
- `components/screener/RangeFilter.tsx` — Min/max number inputs with label, step, suffix, tooltip
- `components/screener/CheckboxFilter.tsx` — Checkbox list for categorical filters (sector, industry, market cap)
- `components/screener/ResultsTable.tsx` — Table with sortable headers, dynamic columns
- `components/screener/ColumnPicker.tsx` — Gear icon popover for show/hide columns (persisted to localStorage)
- `components/screener/Pagination.tsx` — Page controls + page size selector (25/50/100)
- `lib/screener/filter-config.ts` — Filter registry (sections, labels, types, param mappings)
- `lib/screener/column-config.ts` — Column registry (id, label, format function, sortable, default visibility)

**Files to modify:**
- `app/screener/page.tsx` — Replace monolithic component with `<ScreenerClient />`. Keep as thin server component wrapper.

**Filter sections in the sidebar:**
1. **Classification** (default open): Sector (11 checkboxes), Industry (dynamic dependent on sector), Market Cap (5 tiers)
2. **IC Score** (default open): IC Score range (0-100)
3. **Valuation**: P/E, P/B, P/S
4. **Profitability**: ROE, ROA, Gross Margin, Net Margin
5. **Financial Health**: Debt/Equity, Current Ratio
6. **Growth**: Revenue Growth YoY, EPS Growth YoY
7. **Dividends**: Dividend Yield, Payout Ratio, Consecutive Dividend Years
8. **Risk**: Beta

**Column customization:**
- Default visible: Symbol, Name, Market Cap, Price, Change, P/E, Div Yield, Rev Growth, IC Score (matches current)
- Available: All 25+ columns from the expanded API response
- Persist to `localStorage` key `ic_screener_columns`

**Tests to add:**
- `components/screener/RangeFilter.test.tsx` — test onChange fires with correct values
- `components/screener/CheckboxFilter.test.tsx` — test selection/deselection
- `components/screener/FilterPanel.test.tsx` — test section collapse/expand, active filter count

**Verification:** Visual comparison with current screener. All existing filters work. New filters produce results. Column picker persists across page refresh.

---

## Phase 7: CSV Export + Industry Sub-filtering (2 days)

**Goal:** Ship the remaining Phase 1 quick wins.

### PR 7: CSV export and industry filter

**Files to create:**
- `lib/screener/export.ts` — `exportScreenerCSV()` function: UTF-8 BOM, proper CSV escaping, Blob download

**Files to modify:**
- `components/screener/ScreenerToolbar.tsx` — Add "Export CSV" button
- `components/screener/FilterPanel.tsx` — Add industry checkbox filter under Sector (dynamic options fetched when sectors selected)

**Backend addition for industries:**
- `backend/handlers/screener.go` — Add `GET /api/v1/screener/industries?sectors=Technology` endpoint (returns `SELECT DISTINCT industry FROM screener_data WHERE sector IN (...)`)
- `backend/main.go` — Register the new route

**Export behavior:**
- Exports all rows matching current filters (up to 10K, paginated API calls if needed)
- Includes all visible columns
- Filename: `investorcenter-screener-YYYY-MM-DD.csv`
- Toast notification on completion

**Verification:** Export 50-row result, open in Excel/Google Sheets. Verify special characters in company names are escaped. Test industry filter narrows results correctly.

---

## Phase 8: Enhanced Presets + Polish (1-2 days)

**Goal:** Update Quick Screen presets to leverage new filters. Clean up old code.

### PR 8: Updated presets and cleanup

**Files to modify:**
- `lib/screener/presets.ts` — Update preset definitions:
  - **Value Stocks:** pe_max=15, pb_max=2, div_yield_min=2, de_max=1.5
  - **Growth Stocks:** rev_growth_min=20, eps_growth_min=15, gross_margin_min=40
  - **Quality Stocks:** ic_score_min=70, roe_min=15, net_margin_min=10, current_ratio_min=1.5
  - **Dividend Champions:** div_yield_min=3, consec_div_years_min=10, payout_ratio_max=75, de_max=1
  - **NEW — Undervalued:** pe_max=15, pb_max=1.5, roe_min=12, ic_score_min=50

**Cleanup:**
- Remove the old monolithic screener code from `app/screener/page.tsx` (now just a thin wrapper)
- Remove any dead imports or unused utility functions

**Verification:** Click each preset. Verify URL updates with correct params. Verify result count is reasonable.

---

## Phase 9: IC Score Sub-Factor Filters (3-4 days)

**Goal:** Expose the 10 IC Score sub-factor scores as individual filters — the platform's key differentiator.

### PR 9: IC Score sub-factor filters and columns

**Files to modify:**
- `lib/screener/filter-config.ts` — Add "IC Score Factors" section with 10 range filters (value_score, growth_score, profitability_score, financial_health_score, momentum_score, analyst_consensus_score, insider_activity_score, institutional_score, news_sentiment_score, technical_score). Each 0-100, step 1.
- `lib/screener/column-config.ts` — Add 10 sub-factor columns (default hidden, available in column picker)
- `components/screener/FilterSection.tsx` — Add tooltips explaining each sub-factor
- `backend/handlers/screener.go` — Parse 10 new min/max params (already in registry from Phase 3)
- `backend/database/screener.go` — Already handled by filter registry if Phase 3 included them

**Verification:** Filter by `insider_activity_score_min=70&analyst_consensus_score_min=60`. Should return stocks with strong smart-money conviction. Cross-check a few results against the `/ticker/{symbol}` detail page.

---

## Phase 10: Saved Screeners (4-5 days)

**Goal:** Authenticated users can save, load, rename, and delete custom screen configurations.

### PR 10: Saved screeners CRUD

**Files to create:**
- `ic-score-service/migrations/018_create_saved_screens.sql` — `saved_screens` table (id UUID, user_id, name, filters JSONB, columns JSONB, sort JSONB, is_public, created_at, updated_at)
- `backend/handlers/saved_screens.go` — CRUD handlers (List, Create, Update, Delete)
- `backend/database/saved_screens.go` — SQL queries for saved_screens
- `backend/models/saved_screen.go` — `SavedScreen` struct
- `components/screener/SaveScreenModal.tsx` — Modal with name + description inputs
- `components/screener/MyScreensDropdown.tsx` — Dropdown listing user's saved screens

**Files to modify:**
- `backend/main.go` — Register `/api/v1/screens` routes under auth middleware
- `components/screener/ScreenerToolbar.tsx` — Add "Save" button + "My Screens" dropdown

**Verification:** Save a screen with filters. Navigate away. Come back to "My Screens" dropdown. Load the saved screen — filters, columns, and sort should restore. Rename and delete also work.

---

## Phase 11: Watchlist Integration (2-3 days)

**Goal:** Add/remove stocks to watchlists directly from screener results.

### PR 11: Watchlist integration + batch actions

**Files to create:**
- `components/screener/BatchActionBar.tsx` — Floating bar with "Add to Watchlist" when stocks selected

**Files to modify:**
- `components/screener/ResultsTable.tsx` — Add star icon (☆/★) per row for watchlist toggle. Add checkbox column for multi-select.
- `components/screener/ScreenerClient.tsx` — Fetch user's watchlist symbols on mount for membership check. Manage `selectedStocks: Set<string>` state.

**Uses existing API:** `POST /api/v1/watchlists/:id/stocks` (already exists)

**Verification:** Click star on a stock → appears in watchlist. Select 3 stocks → batch add to watchlist. Star fills for stocks already in a watchlist.

---

## Phase 12: Fair Value + Technical + Risk Filters (5-7 days)

**Goal:** Ship the remaining Phase 2 differentiators — DCF fair value screening, technical indicator filters, and risk metric filters.

### PR 12a: Fair value filters
- Add `dcf_upside_percent` range filter (already in view from Phase 2 migration)
- Column: "DCF Upside" showing percentage above/below fair value

### PR 12b: Technical indicator filters
- Expand materialized view (migration 019) to add `rsi_14`, `sma_50`, `sma_200`, `macd_histogram` via LATERAL join on `technical_indicators` (pivot from key-value format)
- Add filters: RSI range, `above_sma50` boolean, `above_sma200` boolean
- Backend: Boolean filters map to `price > sma_50` in SQL

### PR 12c: Risk metric filters
- Expand materialized view to add `sharpe_ratio`, `max_drawdown`, `alpha` via LATERAL join on `risk_metrics` (period='1Y')
- Add filters: Sharpe Ratio, Max Drawdown, Alpha ranges

**Verification:** Filter by `rsi_max=30` — should return oversold stocks. Filter by `dcf_upside_min=20` — should return undervalued stocks. Cross-check against ticker detail pages.

---

## Phase 13: IC Score Screener Consolidation (1-2 days)

**Goal:** Merge the two parallel screener implementations into one.

### PR 13: Redirect /ic-score to /screener

**Files to modify:**
- `app/ic-score/page.tsx` — Replace content with redirect to `/screener?sort=ic_score&order=desc`
- `middleware.ts` — Add 301 redirect rule for `/ic-score` → `/screener?sort=ic_score&order=desc`

**Files to deprecate (don't delete yet):**
- `components/ic-score/ICScoreScreener.tsx` — Mark as deprecated, remove after one release cycle

**Verification:** Navigate to `/ic-score` — should redirect to `/screener` with IC Score sort. All IC Score screening functionality available in the unified screener.

---

## Testing Strategy (Across All Phases)

**Go backend:** Table-driven tests using `testify/assert`. Test filter registry, param parsing, sort mapping. Integration tests gated by `RUN_INTEGRATION_TESTS=true`.

**Frontend:** Jest + React Testing Library. Test component rendering, filter interactions, URL param sync. Mock API calls with `jest.fn()`.

**CI must pass:** `go test ./...`, `go vet`, `gofmt`, `npm test`, pre-commit hooks.

**Manual QA per PR:** Test on both desktop and mobile viewport. Verify dark theme styling matches existing `ic-*` design tokens.

---

## Estimated Timeline

| Phase | Effort | Cumulative |
|-------|--------|------------|
| Phase 1: Fix IC Score/Beta bug | 1 day | Day 1 |
| Phase 2: Expand materialized view | 2-3 days | Day 4 |
| Phase 3: Backend filter registry | 3-4 days | Day 8 |
| Phase 4: URL state management | 2-3 days | Day 11 |
| Phase 5: Server-side filtering | 2-3 days | Day 14 |
| Phase 6: New UI components | 4-5 days | Day 19 |
| Phase 7: CSV export + industry filter | 2 days | Day 21 |
| Phase 8: Enhanced presets + polish | 1-2 days | Day 23 |
| **— Phase 1 Complete (shippable) —** | | **~4-5 weeks** |
| Phase 9: IC Score sub-factors | 3-4 days | Day 27 |
| Phase 10: Saved screeners | 4-5 days | Day 32 |
| Phase 11: Watchlist integration | 2-3 days | Day 35 |
| Phase 12: Fair value + technical + risk | 5-7 days | Day 42 |
| Phase 13: Screener consolidation | 1-2 days | Day 44 |
| **— Phase 2 Complete —** | | **~9-10 weeks** |
