# Technical Specification: Watchlist "Add Ticker" Redesign

**Service:** InvestorCenter.ai — Frontend + Backend
**Author:** Engineering
**Date:** 2026-02-21
**Status:** Draft — Pending Review

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [API Contracts](#2-api-contracts)
3. [Component Specifications](#3-component-specifications)
4. [State Management](#4-state-management)
5. [Keyboard Interaction & Accessibility](#5-keyboard-interaction--accessibility)
6. [Error Handling](#6-error-handling)
7. [Performance Considerations](#7-performance-considerations)
8. [Testing Plan](#8-testing-plan)
9. [Migration & Rollout](#9-migration--rollout)
10. [Open Questions](#10-open-questions)

---

## 1. Architecture Overview

### 1.1 Current Architecture

The watchlist detail page (`app/watchlist/[id]/page.tsx`) currently manages state with `useState` / `useEffect` and a 30-second polling interval. Adding a ticker opens `AddTickerModal`, a controlled modal with five fields (symbol, notes, tags, target buy, target sell). Editing opens `EditTickerModal`. Both modals call `watchListAPI` methods from `lib/api/watchlist.ts`, which delegates to the centralized `ApiClient` in `lib/api.ts`.

Backend routes live under `/api/v1/watchlists` in `backend/main.go`, handled by `backend/handlers/watchlist_handlers.go`, with business logic in `backend/services/watchlist_service.go` and database operations in `backend/database/watchlists.go`.

### 1.2 Proposed Component Tree

```
app/watchlist/[id]/page.tsx  (WatchListDetailPage)
│
├── WatchlistSearchInput          ← NEW (replaces "+ Add Ticker" button + AddTickerModal)
│   └── AutocompleteDropdown      ← NEW
│       └── AutocompleteRow       ← NEW (one per result)
│
├── WatchListTable                ← MODIFIED (add inline edit support)
│   ├── ViewSwitcher              (unchanged)
│   ├── TickerRow                 ← MODIFIED (display + edit modes)
│   │   └── InlineEditPanel       ← NEW (expandable panel below row)
│   │       └── TagChipInput      ← NEW (replaces comma-separated text)
│   └── EmptyState                ← NEW (shown when 0 items)
│
components/Header.tsx             ← MODIFIED (remove "Watch Lists" nav link)
│
components/TickerSearch.tsx       ← MODIFIED (add contextual "Add to watchlist" CTA)
```

### 1.3 Data Flow

```
                        ┌─────────────────────────────────┐
                        │   WatchListDetailPage            │
                        │   watchlistId: string            │
                        │   watchlist: WatchListWithItems  │
                        └──────┬──────────────┬────────────┘
                               │              │
              ┌────────────────┘              └──────────────────┐
              ▼                                                  ▼
   WatchlistSearchInput                                  WatchListTable
   (search query → API)                                  (items[], callbacks)
              │                                                  │
              │ onAdd(symbol)                                    │ onEdit(symbol, data)
              │                                                  │ onRemove(symbol)
              ▼                                                  ▼
   POST /watchlists/:id/items                        PUT  /watchlists/:id/items/:symbol
   { symbol: "AAPL" }                                DELETE /watchlists/:id/items/:symbol
              │                                                  │
              └──────────────────┬───────────────────────────────┘
                                 ▼
                    Optimistic cache update
                    + Background refetch on success
                    + Rollback on failure
```

### 1.4 State Management Strategy

| Layer | Tool | Purpose |
|-------|------|---------|
| Server state | Custom hook with `useState` + `useCallback` (current pattern) | Watchlist data, items, prices. Refetch every 30s. |
| Optimistic mutations | Local state update before API resolves | Instant row insert/update/remove |
| Search input state | `useState` local to `WatchlistSearchInput` | Query string, debounce timer, loading, results |
| Inline edit state | `useState` local to `TickerRow` | Edit mode flag, draft values for targets/tags/notes |
| Global context | React Context (`WatchlistPageContext`) | Pass `watchlistId` and `watchlistName` to `TickerSearch` in Header for contextual add CTA |

> **Note:** The existing codebase uses plain `useState`/`useEffect` rather than React Query or SWR. This spec preserves that pattern to minimize migration scope. A future refactor to React Query is recommended but out of scope.

---

## 2. API Contracts

### 2.1 Ticker Search / Autocomplete

**Existing endpoint — no changes required to the contract.**

```
GET /api/v1/markets/search?q={query}
```

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `q` | string | Yes | Search query. Min 1 character. |

**Response `200 OK`:**

```json
{
  "data": [
    {
      "symbol": "AAPL",
      "name": "Apple Inc.",
      "type": "stock",
      "exchange": "NASDAQ"
    },
    {
      "symbol": "X:BTCUSD",
      "name": "Bitcoin USD",
      "type": "crypto",
      "exchange": "CRYPTO"
    }
  ]
}
```

**Backend enhancement needed:** The current `SearchStocks` function in `backend/database/stocks.go` matches on symbol prefix and name substring, but does not strip the `X:` prefix for matching. When a user types `"BTC"`, the query should match `X:BTCUSD` by stripping the prefix.

**Required change in `backend/database/stocks.go` `SearchStocks()`:**

```go
// Add to the WHERE clause:
// For crypto symbols, also match the portion after "X:"
// e.g., query "BTC" should match symbol "X:BTCUSD"
//
// Current:  WHERE symbol ILIKE $1 || '%' OR name ILIKE '%' || $1 || '%'
// Proposed: WHERE symbol ILIKE $1 || '%'
//              OR REPLACE(symbol, 'X:', '') ILIKE $1 || '%'
//              OR name ILIKE '%' || $1 || '%'
```

**Additionally**, the response should include `logo_url` so the autocomplete dropdown can render company logos without a second request:

```go
// In searchSecurities handler (backend/main.go):
results[i] = gin.H{
    "symbol":   stock.Symbol,
    "name":     stock.Name,
    "type":     stock.AssetType,
    "exchange": stock.Exchange,
    "logo_url": stock.LogoURL,  // ADD THIS FIELD
}
```

The `Stock` model already has `LogoURL` from the `tickers` table. The search database query needs to include `logo_url` in its SELECT.

**Debounce strategy:** 200ms client-side debounce. No server-side rate limiting beyond the existing global rate limiter. If we observe excessive search traffic, a per-user rate limit of 20 req/s on `/markets/search` can be added.

---

### 2.2 Add Ticker to Watchlist

**Existing endpoint — contract unchanged, usage simplified.**

```
POST /api/v1/watchlists/:id/items
```

**Request body (minimal — only symbol required for the inline add flow):**

```json
{
  "symbol": "AAPL"
}
```

The fields `notes`, `tags`, `target_buy_price`, `target_sell_price` remain accepted but are no longer sent on the initial add. They are sent via the update endpoint (2.3) when the user edits inline.

**Response `201 Created`:**

```json
{
  "item": {
    "id": "a1b2c3d4-...",
    "watch_list_id": "...",
    "symbol": "AAPL",
    "notes": null,
    "tags": [],
    "target_buy_price": null,
    "target_sell_price": null,
    "added_at": "2026-02-21T15:30:00Z",
    "display_order": 0
  }
}
```

**Error responses:**

| Code | Condition | Body |
|------|-----------|------|
| `400` | Missing/invalid symbol | `{ "error": "Symbol is required" }` |
| `404` | Watchlist not found or not owned by user | `{ "error": "Watch list not found" }` |
| `404` | Ticker symbol doesn't exist in `tickers` table | `{ "error": "Ticker not found in database" }` |
| `409` | Ticker already in this watchlist | `{ "error": "Ticker already exists in this watch list" }` |
| `422` | Watchlist item limit reached (10 for free tier) | `{ "error": "Watch list item limit reached" }` |

**Optimistic update strategy:**

1. Client inserts a placeholder `WatchListItem` into local state immediately.
2. Placeholder uses the `symbol` from the search result plus `name`, `exchange`, `asset_type`, `logo_url` from the autocomplete response. All other fields (IC Score, fundamentals, price) default to `null`.
3. On `201`: replace placeholder with server response. The next 30-second polling cycle populates full data (real-time prices, IC Score, etc.).
4. On error: remove placeholder from local state, show toast with error message.

---

### 2.3 Update Ticker Metadata

**Existing endpoint — no contract changes.**

```
PUT /api/v1/watchlists/:id/items/:symbol
```

**Request body:**

```json
{
  "notes": "Watching for Q4 earnings beat",
  "tags": ["tech", "earnings-play"],
  "target_buy_price": 175.00,
  "target_sell_price": 210.00
}
```

All fields optional. Omitted fields remain unchanged. To clear a field, send `null` explicitly.

**Response `200 OK`:**

```json
{
  "item": {
    "id": "a1b2c3d4-...",
    "symbol": "AAPL",
    "notes": "Watching for Q4 earnings beat",
    "tags": ["tech", "earnings-play"],
    "target_buy_price": 175.00,
    "target_sell_price": 210.00,
    "added_at": "2026-02-21T15:30:00Z",
    "display_order": 0
  }
}
```

**Validation (existing, unchanged):**

- `notes`: max 10,000 characters
- `tags`: max 50 tags, each max 100 characters
- `target_buy_price`: must be ≥ 0
- `target_sell_price`: must be ≥ 0
- Cross-validation (buy < sell) is currently only enforced client-side in `AddTickerModal`. The backend should add this validation (see [Open Questions](#10-open-questions)).

---

### 2.4 Delete Ticker from Watchlist

**Existing endpoint — no changes.**

```
DELETE /api/v1/watchlists/:id/items/:symbol
```

**Response `200 OK`:**

```json
{
  "message": "Ticker removed from watch list"
}
```

---

### 2.5 New Endpoint: Get User's Tags

**New endpoint** to power tag suggestions in the chip input.

```
GET /api/v1/watchlists/tags
```

**Response `200 OK`:**

```json
{
  "tags": [
    { "name": "tech", "count": 5 },
    { "name": "earnings-play", "count": 3 },
    { "name": "dividend", "count": 2 },
    { "name": "long-term", "count": 1 }
  ]
}
```

**Implementation:** Query all `watch_list_items` for the authenticated user, `UNNEST(tags)`, `GROUP BY` tag value, `ORDER BY count DESC`, `LIMIT 50`.

```sql
SELECT tag, COUNT(*) as count
FROM watch_list_items wli
JOIN watch_lists wl ON wli.watch_list_id = wl.id
CROSS JOIN UNNEST(wli.tags) AS tag
WHERE wl.user_id = $1
GROUP BY tag
ORDER BY count DESC
LIMIT 50;
```

**Backend files to modify:**

- `backend/main.go` — add route `watchListRoutes.GET("/tags", handlers.GetUserTags)`
- `backend/handlers/watchlist_handlers.go` — new `GetUserTags` handler
- `backend/database/watchlists.go` — new `GetUserTags(userID string)` function

---

## 3. Component Specifications

### 3.1 TypeScript Interfaces

These types extend or refine the existing interfaces in `lib/api/watchlist.ts`:

```typescript
// ---- Search / Autocomplete ----

/** A single result from /markets/search. Extends current shape with logo_url. */
interface TickerSearchResult {
  symbol: string;       // e.g., "AAPL", "X:BTCUSD"
  name: string;         // e.g., "Apple Inc.", "Bitcoin USD"
  type: AssetType;      // e.g., "stock", "etf", "crypto"
  exchange: string;     // e.g., "NASDAQ", "CRYPTO"
  logo_url?: string;    // e.g., "https://logo.clearbit.com/apple.com"
}

type AssetType = 'stock' | 'etf' | 'crypto' | 'reit' | 'spac' | 'adr' | 'index';

// ---- Tags ----

interface TagSuggestion {
  name: string;   // lowercased tag text
  count: number;  // how many watchlist items use this tag
}

// ---- Inline Edit ----

/** Draft state for the inline edit panel. All fields optional — only changed fields sent to API. */
interface TickerEditDraft {
  notes: string | null;
  tags: string[];
  target_buy_price: number | null;
  target_sell_price: number | null;
}

// ---- Optimistic Placeholder ----

/** Minimal item inserted before server confirms. */
interface OptimisticWatchListItem extends Partial<WatchListItem> {
  symbol: string;
  name: string;
  exchange: string;
  asset_type: string;
  logo_url?: string;
  _optimistic: true;  // flag to distinguish from server-confirmed items
}
```

The existing `WatchListItem` interface in `lib/api/watchlist.ts` remains unchanged — it already carries all 47 fields from the backend response.

---

### 3.2 `WatchlistSearchInput`

**File:** `components/watchlist/WatchlistSearchInput.tsx` (new file)

**Props:**

```typescript
interface WatchlistSearchInputProps {
  watchlistId: string;
  /** Symbols already in the watchlist, used to mark results as "already added" */
  existingSymbols: Set<string>;
  /** Current item count, used to disable input at limit */
  itemCount: number;
  /** Max items allowed (currently 10 for free tier) */
  maxItems: number;
  /** Called after successful add. Parent should update its items list. */
  onTickerAdded: (item: WatchListItem) => void;
  /** Called when optimistic add needs rollback */
  onTickerAddFailed: (symbol: string, error: string) => void;
}
```

**Internal state:**

```typescript
const [query, setQuery] = useState('');
const [results, setResults] = useState<TickerSearchResult[]>([]);
const [isLoading, setIsLoading] = useState(false);
const [isOpen, setIsOpen] = useState(false);
const [highlightedIndex, setHighlightedIndex] = useState(-1);
const [justAdded, setJustAdded] = useState(false); // green check flash
const inputRef = useRef<HTMLInputElement>(null);
const debounceTimerRef = useRef<ReturnType<typeof setTimeout>>();
```

**Debounce logic:**

```typescript
const handleInputChange = (value: string) => {
  setQuery(value);
  setHighlightedIndex(-1);

  if (debounceTimerRef.current) {
    clearTimeout(debounceTimerRef.current);
  }

  if (value.trim().length === 0) {
    setResults([]);
    setIsOpen(false);
    return;
  }

  debounceTimerRef.current = setTimeout(async () => {
    setIsLoading(true);
    try {
      const response = await apiClient.searchSecurities(value.trim());
      setResults(response.data ?? []);
      setIsOpen(true);
    } catch {
      setResults([]);
    } finally {
      setIsLoading(false);
    }
  }, 200);
};
```

**Keyboard event handling:**

```typescript
const handleKeyDown = (e: React.KeyboardEvent) => {
  if (!isOpen || results.length === 0) return;

  const selectableResults = results.filter(
    r => !existingSymbols.has(r.symbol)
  );

  switch (e.key) {
    case 'ArrowDown':
      e.preventDefault();
      setHighlightedIndex(prev =>
        prev < selectableResults.length - 1 ? prev + 1 : 0
      );
      break;
    case 'ArrowUp':
      e.preventDefault();
      setHighlightedIndex(prev =>
        prev > 0 ? prev - 1 : selectableResults.length - 1
      );
      break;
    case 'Enter':
      e.preventDefault();
      if (highlightedIndex >= 0) {
        addTicker(selectableResults[highlightedIndex]);
      } else if (selectableResults.length > 0) {
        addTicker(selectableResults[0]);
      }
      break;
    case 'Escape':
      setIsOpen(false);
      inputRef.current?.blur();
      break;
  }
};
```

**Add ticker flow:**

```typescript
const addTicker = async (result: TickerSearchResult) => {
  // Optimistic flash
  setJustAdded(true);
  setTimeout(() => setJustAdded(false), 300);

  // Clear input, keep focus for batch adding
  setQuery('');
  setResults([]);
  setIsOpen(false);

  try {
    const item = await watchListAPI.addTicker(watchlistId, {
      symbol: result.symbol,
    });
    onTickerAdded(item);
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : 'Failed to add ticker';
    onTickerAddFailed(result.symbol, message);
  }
};
```

**Disabled state when limit reached:**

```typescript
const isAtLimit = itemCount >= maxItems;

// In JSX:
<input
  ref={inputRef}
  disabled={isAtLimit}
  placeholder={isAtLimit
    ? `Watchlist full (${itemCount}/${maxItems} tickers)`
    : 'Add a ticker...'
  }
  // ...
/>
```

---

### 3.3 `AutocompleteDropdown`

**File:** `components/watchlist/AutocompleteDropdown.tsx` (new file)

**Props:**

```typescript
interface AutocompleteDropdownProps {
  results: TickerSearchResult[];
  existingSymbols: Set<string>;
  highlightedIndex: number;
  isLoading: boolean;
  query: string;
  onSelect: (result: TickerSearchResult) => void;
  onHighlightChange: (index: number) => void;
}
```

**Rendering logic:**

```typescript
const AutocompleteDropdown: React.FC<AutocompleteDropdownProps> = ({
  results, existingSymbols, highlightedIndex, isLoading, query, onSelect, onHighlightChange,
}) => {
  // Partition: available results first, then already-added
  const available = results.filter(r => !existingSymbols.has(r.symbol));
  const alreadyAdded = results.filter(r => existingSymbols.has(r.symbol));

  if (isLoading) {
    return (
      <div role="listbox" className="...">
        <div className="px-4 py-3 text-sm text-gray-500">Searching...</div>
      </div>
    );
  }

  if (results.length === 0 && query.trim().length > 0) {
    return (
      <div role="listbox" className="...">
        <div className="px-4 py-3 text-sm text-gray-500">
          No matching tickers found. Try a different name or symbol.
        </div>
      </div>
    );
  }

  return (
    <ul role="listbox" id="watchlist-search-listbox" className="...">
      {available.map((result, index) => (
        <AutocompleteRow
          key={result.symbol}
          result={result}
          isHighlighted={index === highlightedIndex}
          isDisabled={false}
          onClick={() => onSelect(result)}
          onMouseEnter={() => onHighlightChange(index)}
          id={`search-option-${index}`}
        />
      ))}
      {alreadyAdded.length > 0 && (
        <>
          <li className="px-4 py-1 text-xs text-gray-400 uppercase tracking-wide">
            Already in watchlist
          </li>
          {alreadyAdded.map(result => (
            <AutocompleteRow
              key={result.symbol}
              result={result}
              isHighlighted={false}
              isDisabled={true}
              id={`search-option-${result.symbol}-disabled`}
            />
          ))}
        </>
      )}
    </ul>
  );
};
```

**`AutocompleteRow` sub-component:**

```typescript
interface AutocompleteRowProps {
  result: TickerSearchResult;
  isHighlighted: boolean;
  isDisabled: boolean;
  onClick?: () => void;
  onMouseEnter?: () => void;
  id: string;
}
```

Renders: logo (24×24, fallback to initial letter), bold symbol, name (truncated), exchange badge, and a checkmark if disabled.

**Virtualization:** Not needed. Max 10 results from API + a few already-added items. Total < 20 rows. Standard DOM rendering is sufficient.

---

### 3.4 `TickerRow` (Modified) + `InlineEditPanel`

**File:** `components/watchlist/WatchListTable.tsx` (modified) + `components/watchlist/InlineEditPanel.tsx` (new)

The existing `WatchListTable` renders rows in a `<tbody>`. Each row gains an `editingSymbol` check:

```typescript
// In WatchListTable, for each item:
<tr key={item.symbol} className={rowClasses}>
  {/* existing cell rendering unchanged */}
</tr>
{editingSymbol === item.symbol && (
  <tr>
    <td colSpan={columns.length}>
      <InlineEditPanel
        item={item}
        onSave={handleSave}
        onCancel={() => setEditingSymbol(null)}
      />
    </td>
  </tr>
)}
```

**`InlineEditPanel` props:**

```typescript
interface InlineEditPanelProps {
  item: WatchListItem;
  onSave: (symbol: string, data: TickerEditDraft) => Promise<void>;
  onCancel: () => void;
}
```

**Internal state:**

```typescript
const [draft, setDraft] = useState<TickerEditDraft>({
  notes: item.notes ?? null,
  tags: item.tags ?? [],
  target_buy_price: item.target_buy_price ?? null,
  target_sell_price: item.target_sell_price ?? null,
});
const [isSaving, setIsSaving] = useState(false);
const [error, setError] = useState<string | null>(null);
```

**Validation before save:**

```typescript
const validate = (): string | null => {
  const { target_buy_price, target_sell_price } = draft;
  if (
    target_buy_price != null &&
    target_sell_price != null &&
    target_buy_price >= target_sell_price
  ) {
    return 'Target buy price must be less than target sell price.';
  }
  return null;
};
```

**Save handler:**

```typescript
const handleSave = async () => {
  const validationError = validate();
  if (validationError) {
    setError(validationError);
    return;
  }
  setIsSaving(true);
  setError(null);
  try {
    await onSave(item.symbol, draft);
    // onSave closes the panel on success
  } catch {
    setError('Save failed. Try again.');
  } finally {
    setIsSaving(false);
  }
};
```

**Layout (Tailwind):**

```tsx
<div className="bg-gray-50 dark:bg-gray-800/50 border-t border-b border-gray-200
  dark:border-gray-700 px-6 py-4 space-y-4">
  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
    {/* Target Buy */}
    <div>
      <label className="block text-xs font-medium text-gray-500 mb-1">
        Target Buy Price
      </label>
      <div className="relative">
        <span className="absolute left-3 top-2 text-gray-400">$</span>
        <input
          type="number"
          step="0.01"
          min="0"
          className="pl-7 ..."
          value={draft.target_buy_price ?? ''}
          onChange={e => setDraft(d => ({
            ...d,
            target_buy_price: e.target.value ? parseFloat(e.target.value) : null,
          }))}
          placeholder="Buy target"
        />
      </div>
    </div>

    {/* Target Sell */}
    <div>
      <label className="...">Target Sell Price</label>
      <div className="relative">
        <span className="absolute left-3 top-2 text-gray-400">$</span>
        <input type="number" step="0.01" min="0" className="pl-7 ..." ... />
      </div>
    </div>

    {/* Tags */}
    <div>
      <label className="...">Tags</label>
      <TagChipInput
        tags={draft.tags}
        onChange={tags => setDraft(d => ({ ...d, tags }))}
      />
    </div>
  </div>

  {/* Notes */}
  <div>
    <label className="...">Notes</label>
    <textarea
      rows={2}
      maxLength={10000}
      className="..."
      value={draft.notes ?? ''}
      onChange={e => setDraft(d => ({
        ...d,
        notes: e.target.value || null,
      }))}
      placeholder="Add notes..."
    />
  </div>

  {/* Actions */}
  <div className="flex items-center justify-between">
    {error && <p className="text-sm text-red-600">{error}</p>}
    <div className="flex gap-2 ml-auto">
      <button onClick={onCancel} className="...">Cancel</button>
      <button onClick={handleSave} disabled={isSaving} className="...">
        {isSaving ? 'Saving...' : 'Save Changes'}
      </button>
    </div>
  </div>
</div>
```

---

### 3.5 `TagChipInput`

**File:** `components/watchlist/TagChipInput.tsx` (new file)

**Props:**

```typescript
interface TagChipInputProps {
  tags: string[];
  onChange: (tags: string[]) => void;
  maxTags?: number;          // default 50
  maxTagLength?: number;     // default 100
  placeholder?: string;      // default "Add a tag..."
}
```

**Internal state:**

```typescript
const [inputValue, setInputValue] = useState('');
const [suggestions, setSuggestions] = useState<TagSuggestion[]>([]);
const [showSuggestions, setShowSuggestions] = useState(false);
const [highlightedSuggestion, setHighlightedSuggestion] = useState(-1);
const inputRef = useRef<HTMLInputElement>(null);
```

**Tag creation logic:**

```typescript
const addTag = (rawTag: string) => {
  const tag = rawTag.trim().toLowerCase();
  if (tag.length === 0) return;
  if (tag.length > maxTagLength) return;
  if (tags.length >= maxTags) return;
  if (tags.includes(tag)) return; // deduplicate

  onChange([...tags, tag]);
  setInputValue('');
  setShowSuggestions(false);
};
```

**Key handlers:**

```typescript
const handleKeyDown = (e: React.KeyboardEvent) => {
  if (e.key === 'Enter' || e.key === ',') {
    e.preventDefault();
    if (highlightedSuggestion >= 0) {
      addTag(filteredSuggestions[highlightedSuggestion].name);
    } else {
      addTag(inputValue);
    }
  } else if (e.key === 'Backspace' && inputValue === '' && tags.length > 0) {
    // Remove last tag
    onChange(tags.slice(0, -1));
  } else if (e.key === 'ArrowDown') {
    e.preventDefault();
    setHighlightedSuggestion(prev =>
      prev < filteredSuggestions.length - 1 ? prev + 1 : 0
    );
  } else if (e.key === 'ArrowUp') {
    e.preventDefault();
    setHighlightedSuggestion(prev =>
      prev > 0 ? prev - 1 : filteredSuggestions.length - 1
    );
  } else if (e.key === 'Escape') {
    setShowSuggestions(false);
  }
};
```

**Tag suggestion fetching:**

```typescript
// Fetch once on mount, cache for session.
const [allUserTags, setAllUserTags] = useState<TagSuggestion[]>([]);

useEffect(() => {
  watchListAPI.getUserTags()
    .then(res => setAllUserTags(res.tags))
    .catch(() => {}); // silent — suggestions are optional
}, []);

// Filter locally as user types
const filteredSuggestions = useMemo(() => {
  if (inputValue.trim().length === 0) return [];
  const q = inputValue.trim().toLowerCase();
  return allUserTags
    .filter(t => t.name.includes(q) && !tags.includes(t.name))
    .slice(0, 6);
}, [inputValue, allUserTags, tags]);
```

**Chip rendering:**

```tsx
<div className="flex flex-wrap items-center gap-1.5 p-2 border rounded-lg
  bg-white dark:bg-gray-900 border-gray-300 dark:border-gray-600
  focus-within:ring-2 focus-within:ring-blue-500">
  {tags.map(tag => (
    <span
      key={tag}
      className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full
        text-xs font-medium bg-blue-100 text-blue-800
        dark:bg-blue-900 dark:text-blue-200"
    >
      {tag}
      <button
        type="button"
        onClick={() => onChange(tags.filter(t => t !== tag))}
        className="hover:text-blue-600 dark:hover:text-blue-100"
        aria-label={`Remove tag: ${tag}`}
      >
        ×
      </button>
    </span>
  ))}
  <input
    ref={inputRef}
    type="text"
    value={inputValue}
    onChange={e => {
      setInputValue(e.target.value);
      setShowSuggestions(true);
    }}
    onKeyDown={handleKeyDown}
    onFocus={() => setShowSuggestions(true)}
    onBlur={() => setTimeout(() => setShowSuggestions(false), 150)}
    placeholder={tags.length === 0 ? placeholder : ''}
    className="flex-1 min-w-[80px] outline-none bg-transparent text-sm"
    aria-label="Add a tag"
  />
</div>
```

---

### 3.6 `TickerSearch` (Global Search Bar) — Modifications

**File:** `components/TickerSearch.tsx` (modified)

**Change summary:** When the user is on a watchlist detail page, each search result row shows a `+ Add` button.

**How the component knows it's on a watchlist page:**

A new React context provides watchlist page state to the Header:

```typescript
// lib/contexts/WatchlistPageContext.tsx (new file)

interface WatchlistPageContextValue {
  watchlistId: string | null;
  watchlistName: string | null;
  existingSymbols: Set<string>;
  onQuickAdd: (symbol: string) => Promise<void>;
}

const WatchlistPageContext = createContext<WatchlistPageContextValue>({
  watchlistId: null,
  watchlistName: null,
  existingSymbols: new Set(),
  onQuickAdd: async () => {},
});

export const useWatchlistPage = () => useContext(WatchlistPageContext);
```

The `WatchListDetailPage` wraps its content in `<WatchlistPageProvider>`:

```tsx
// In app/watchlist/[id]/page.tsx:
<WatchlistPageContext.Provider value={{
  watchlistId: watchList.id,
  watchlistName: watchList.name,
  existingSymbols: new Set(watchList.items.map(i => i.symbol)),
  onQuickAdd: handleQuickAdd,
}}>
  <Header />  {/* Header renders TickerSearch */}
  {/* ... rest of page */}
</WatchlistPageContext.Provider>
```

> **Note:** This requires the `<Header />` to be rendered inside the provider. If `Header` is rendered in a layout above the page component, the provider must be lifted to the layout or use a global store. The simplest approach: keep `Header` in the root layout and use a lightweight Zustand store or a module-level event emitter instead of context. See [Open Questions](#10-open-questions).

**Modified rendering in `TickerSearch.tsx`:**

```tsx
const { watchlistId, watchlistName, existingSymbols, onQuickAdd } = useWatchlistPage();

// In the result row:
<li className="flex items-center justify-between px-4 py-2 ...">
  <div onClick={() => navigateToTicker(result.symbol)} className="flex-1 cursor-pointer">
    <span className="font-semibold">{result.symbol}</span>
    <span className="ml-2 text-gray-500">{result.name}</span>
  </div>
  {watchlistId && (
    existingSymbols.has(result.symbol) ? (
      <span className="text-xs text-gray-400">✓ Added</span>
    ) : (
      <button
        onClick={(e) => {
          e.stopPropagation();
          onQuickAdd(result.symbol);
        }}
        className="text-xs font-medium text-blue-600 hover:text-blue-800
          px-2 py-0.5 rounded hover:bg-blue-50"
        title={`Add to ${watchlistName}`}
      >
        + Add
      </button>
    )
  )}
</li>
```

---

### 3.7 `Header` / `TopNav` — Removal of "Watch Lists" Link

**File:** `components/Header.tsx` (modified)

**Current code (authenticated nav links):**

```tsx
{user && (
  <>
    <NavLink href="/watchlist">Watch Lists</NavLink>
    <NavLink href="/alerts">Alerts</NavLink>
  </>
)}
```

**Change:** Remove the `Watch Lists` NavLink. Keep `Alerts`.

```tsx
{user && (
  <>
    <NavLink href="/alerts">Alerts</NavLink>
  </>
)}
```

The profile dropdown already contains "My Watch Lists" which links to `/watchlist`. No other changes needed.

**Regression considerations:**

- Users who navigate via top nav will need to use the dropdown. This is a minor habit change.
- The `/watchlist` route remains valid and unchanged.
- Browser bookmarks to `/watchlist` are unaffected.
- No other components link to `/watchlist` via the top-nav item — they use direct hrefs.

---

## 4. State Management

### 4.1 Component-Level State

| Component | State | Type | Purpose |
|-----------|-------|------|---------|
| `WatchlistSearchInput` | `query` | `string` | Current search input value |
| | `results` | `TickerSearchResult[]` | API response for current query |
| | `isLoading` | `boolean` | Loading spinner |
| | `isOpen` | `boolean` | Dropdown visibility |
| | `highlightedIndex` | `number` | Keyboard-nav highlighted row |
| | `justAdded` | `boolean` | Green flash feedback (300ms) |
| `InlineEditPanel` | `draft` | `TickerEditDraft` | Uncommitted edits |
| | `isSaving` | `boolean` | Save in progress |
| | `error` | `string \| null` | Validation or API error |
| `TagChipInput` | `inputValue` | `string` | Current text in tag input |
| | `suggestions` | `TagSuggestion[]` | Filtered tag suggestions |
| | `showSuggestions` | `boolean` | Dropdown visibility |
| | `highlightedSuggestion` | `number` | Keyboard-nav highlighted suggestion |
| `WatchListTable` | `editingSymbol` | `string \| null` | Which row is in edit mode |

### 4.2 Page-Level State (Server Data)

The `WatchListDetailPage` currently manages the watchlist data:

```typescript
const [watchList, setWatchList] = useState<WatchListWithItems | null>(null);
const [loading, setLoading] = useState(true);
const [error, setError] = useState('');
```

This remains unchanged. The page refetches every 30 seconds via `setInterval`.

### 4.3 Optimistic Updates

**Adding a ticker:**

```typescript
// In WatchListDetailPage:
const handleTickerAdded = (newItem: WatchListItem) => {
  setWatchList(prev => {
    if (!prev) return prev;
    return {
      ...prev,
      item_count: prev.item_count + 1,
      items: [newItem, ...prev.items],
    };
  });
};

const handleTickerAddFailed = (symbol: string, errorMsg: string) => {
  // Remove the optimistic item if it was inserted
  setWatchList(prev => {
    if (!prev) return prev;
    return {
      ...prev,
      item_count: prev.item_count - 1,
      items: prev.items.filter(i => i.symbol !== symbol),
    };
  });
  toast.error(errorMsg);
};
```

**Saving inline edits:**

```typescript
const handleSaveEdit = async (symbol: string, data: TickerEditDraft) => {
  // Optimistic: update local state
  setWatchList(prev => {
    if (!prev) return prev;
    return {
      ...prev,
      items: prev.items.map(item =>
        item.symbol === symbol
          ? { ...item, ...data }
          : item
      ),
    };
  });

  try {
    await watchListAPI.updateTicker(watchList!.id, symbol, data);
    setEditingSymbol(null);
  } catch {
    // Rollback: refetch full watchlist
    await loadWatchList();
    toast.error(`Failed to save changes for ${symbol}.`);
  }
};
```

**Removing a ticker:**

```typescript
const handleRemoveTicker = async (symbol: string) => {
  const prevItems = watchList!.items;

  // Optimistic removal
  setWatchList(prev => {
    if (!prev) return prev;
    return {
      ...prev,
      item_count: prev.item_count - 1,
      items: prev.items.filter(i => i.symbol !== symbol),
    };
  });

  try {
    await watchListAPI.removeTicker(watchList!.id, symbol);
  } catch {
    // Rollback
    setWatchList(prev => {
      if (!prev) return prev;
      return { ...prev, item_count: prevItems.length, items: prevItems };
    });
    toast.error(`Failed to remove ${symbol}.`);
  }
};
```

### 4.4 Cross-Component Communication for Global Search CTA

The `Header` component is rendered in the root layout (`app/layout.tsx`), outside the watchlist page component tree. React context from the page won't reach it.

**Solution: Zustand micro-store.**

```typescript
// lib/stores/watchlistPageStore.ts (new file)

import { create } from 'zustand';

interface WatchlistPageState {
  activeWatchlistId: string | null;
  activeWatchlistName: string | null;
  existingSymbols: Set<string>;
  setActiveWatchlist: (
    id: string | null,
    name: string | null,
    symbols: Set<string>
  ) => void;
  addSymbol: (symbol: string) => void;
  clearActiveWatchlist: () => void;
}

export const useWatchlistPageStore = create<WatchlistPageState>((set) => ({
  activeWatchlistId: null,
  activeWatchlistName: null,
  existingSymbols: new Set(),
  setActiveWatchlist: (id, name, symbols) =>
    set({ activeWatchlistId: id, activeWatchlistName: name, existingSymbols: symbols }),
  addSymbol: (symbol) =>
    set((state) => ({
      existingSymbols: new Set([...state.existingSymbols, symbol]),
    })),
  clearActiveWatchlist: () =>
    set({ activeWatchlistId: null, activeWatchlistName: null, existingSymbols: new Set() }),
}));
```

The watchlist detail page calls `setActiveWatchlist` on mount and `clearActiveWatchlist` on unmount:

```typescript
// In app/watchlist/[id]/page.tsx:
const { setActiveWatchlist, clearActiveWatchlist, addSymbol } = useWatchlistPageStore();

useEffect(() => {
  if (watchList) {
    setActiveWatchlist(
      watchList.id,
      watchList.name,
      new Set(watchList.items.map(i => i.symbol))
    );
  }
  return () => clearActiveWatchlist();
}, [watchList?.id, watchList?.items.length]);
```

`TickerSearch` reads from the store:

```typescript
const { activeWatchlistId, activeWatchlistName, existingSymbols } =
  useWatchlistPageStore();
```

> **Zustand dependency:** If adding Zustand is undesirable, an alternative is a module-level event emitter or storing state in `window.__watchlistPageState`. Zustand is the cleanest option and is a single 1KB dependency. See [Open Questions](#10-open-questions).

---

## 5. Keyboard Interaction & Accessibility

### 5.1 `/` Shortcut Implementation

**File:** `app/watchlist/[id]/page.tsx` (or a custom hook `useSlashFocus`)

```typescript
// hooks/useSlashFocus.ts (new file)

export function useSlashFocus(inputRef: React.RefObject<HTMLInputElement>) {
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      // Skip if user is typing in an input, textarea, or contenteditable
      const target = e.target as HTMLElement;
      if (
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.tagName === 'SELECT' ||
        target.isContentEditable
      ) {
        return;
      }

      if (e.key === '/') {
        e.preventDefault();
        inputRef.current?.focus();
      }
    };

    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [inputRef]);
}
```

**Usage in `WatchlistSearchInput`:**

```typescript
useSlashFocus(inputRef);
```

**Conflict avoidance:** The guard checks `target.tagName` and `target.isContentEditable`. This prevents the shortcut from firing when the user is typing in any form field, the search bar itself, the inline edit panel, or any other text input.

---

### 5.2 ARIA Roles

**`WatchlistSearchInput` + `AutocompleteDropdown`:**

```tsx
<div role="combobox" aria-expanded={isOpen} aria-haspopup="listbox" aria-owns="watchlist-search-listbox">
  <input
    ref={inputRef}
    role="searchbox"
    aria-autocomplete="list"
    aria-controls="watchlist-search-listbox"
    aria-activedescendant={
      highlightedIndex >= 0 ? `search-option-${highlightedIndex}` : undefined
    }
    aria-label="Search for a ticker to add to your watchlist"
    // ...
  />
</div>

{isOpen && (
  <ul role="listbox" id="watchlist-search-listbox">
    {results.map((result, index) => (
      <li
        role="option"
        id={`search-option-${index}`}
        aria-selected={index === highlightedIndex}
        aria-disabled={existingSymbols.has(result.symbol)}
        // ...
      >
        {/* result content */}
      </li>
    ))}
  </ul>
)}
```

**Live region for result count:**

```tsx
<div aria-live="polite" aria-atomic="true" className="sr-only">
  {results.length > 0
    ? `${results.length} results found. Use arrow keys to navigate.`
    : query.trim().length > 0 && !isLoading
      ? 'No results found.'
      : ''}
</div>
```

**`TagChipInput`:**

```tsx
<div role="group" aria-label="Tags">
  {tags.map(tag => (
    <span role="listitem" key={tag}>
      {tag}
      <button aria-label={`Remove tag: ${tag}`} onClick={...}>×</button>
    </span>
  ))}
  <input aria-label="Add a tag. Type and press Enter to create." />
</div>
```

**`InlineEditPanel`:**

```tsx
<div role="region" aria-label={`Edit ${item.symbol} details`}>
  {/* form fields */}
</div>
```

### 5.3 Focus Management

| Action | Focus Destination |
|--------|-------------------|
| Ticker added via search | Focus remains in search input (enables batch adding) |
| Inline edit opened (via edit button click) | First field in edit panel (Target Buy Price) |
| Inline edit opened (via Tab on search result) | First field in edit panel |
| Inline edit saved | Edit button of the saved row |
| Inline edit cancelled (Escape or Cancel button) | Edit button of the row |
| Autocomplete dropdown closed (Escape) | Search input blurs |
| `/` shortcut pressed | Search input focuses |

Implementation pattern for returning focus after edit:

```typescript
// In WatchListTable:
const editButtonRefs = useRef<Map<string, HTMLButtonElement>>(new Map());

// When setting editingSymbol to null:
const closeEdit = (symbol: string) => {
  setEditingSymbol(null);
  // Return focus to the edit button after React re-renders
  requestAnimationFrame(() => {
    editButtonRefs.current.get(symbol)?.focus();
  });
};
```

---

## 6. Error Handling

### 6.1 Invalid or Unknown Ticker Symbol

**Scenario:** User types a query that returns 0 results from `/markets/search`.

**Handling:** The autocomplete dropdown displays a static message:

```
No matching tickers found. Try a different name or symbol.
```

No toast. No error state on the input. The user can keep typing.

**Scenario:** User manually submits a symbol that doesn't exist in the `tickers` table (edge case — the inline search UI should prevent this, but the API should handle it).

**Backend response:** `404 Not Found` with `{ "error": "Ticker not found in database" }`.

**Frontend handling:** Toast: "Ticker not recognized. Try searching by company name."

### 6.2 Duplicate Ticker Already in Watchlist

**Prevention (UI layer):** The `existingSymbols` set is checked in the autocomplete dropdown. Already-added tickers are visually muted with a `✓` badge and are not clickable. This makes duplicates nearly impossible from the UI.

**Backend safety net:** `409 Conflict` with `{ "error": "Ticker already exists in this watch list" }`.

**Frontend handling:** Toast: "AAPL is already in this watchlist." Remove any optimistic row that was inserted.

### 6.3 Network Failure During Add

**Scenario:** `POST /watchlists/:id/items` fails due to network timeout, 500, or other transport error.

**Handling:**

1. The optimistic row (if inserted) is removed from local state.
2. Toast displayed: "Couldn't add AAPL — check your connection and try again."
3. The toast includes a "Retry" button that re-attempts the `POST` for the same symbol.

```typescript
// Toast with retry action (using a toast library like react-hot-toast or sonner):
toast.error(
  `Couldn't add ${symbol} — check your connection and try again.`,
  {
    action: {
      label: 'Retry',
      onClick: () => addTicker({ symbol, name, type, exchange, logo_url }),
    },
  }
);
```

### 6.4 Network Failure During Search

**Scenario:** `GET /markets/search?q=...` returns a network error or 503.

**Handling:** Dropdown shows: "Search unavailable. Check your connection and try again."

The input remains editable. A "Retry" link in the dropdown re-fires the debounced search.

```typescript
const [searchError, setSearchError] = useState(false);

// In debounce callback:
try {
  const response = await apiClient.searchSecurities(value.trim());
  setResults(response.data ?? []);
  setSearchError(false);
} catch {
  setResults([]);
  setSearchError(true);
}
```

### 6.5 Rate Limiting on Autocomplete

The current backend does not rate-limit `/markets/search` specifically. The global rate limiter (if present) applies.

**Client-side mitigation:** The 200ms debounce is the primary defense. At maximum typing speed (~12 characters/second), this limits requests to ~5/second. Acceptable.

**If server-side rate limiting is added later:** The search handler should return `429 Too Many Requests`. Frontend treats this like a network error: "Search temporarily unavailable. Try again in a moment."

### 6.6 Watchlist Item Limit Reached

**Pre-check (UI layer):** The `WatchlistSearchInput` disables when `itemCount >= maxItems`.

**Race condition:** If the limit is reached between the time the user opens the dropdown and clicks a result (e.g., another tab added an item), the backend returns `422`.

**Handling:** Toast: "This watchlist is full (10/10). Remove a ticker to add more." Remove optimistic row.

### 6.7 Inline Edit Save Failure

**Handling:** The edit panel stays open. An inline error message appears below the Save button:

```
Save failed. Try again.
```

The Save button text changes from "Save Changes" to "Retry Save". The user can retry or cancel.

No optimistic rollback for edits — the panel stays open with the user's draft, so they don't lose their work.

---

## 7. Performance Considerations

### 7.1 Debounce Timing

**Recommendation: 200ms.**

Rationale:
- The current `AddTickerModal` and `TickerSearch` both use 300ms.
- 200ms feels more responsive for an inline search experience where speed is critical.
- The backend search query is a simple indexed lookup on the `tickers` table (~25K rows). P99 response time should be < 50ms.
- At 200ms debounce, worst case is 5 requests/second during continuous typing. Acceptable for a single user.

### 7.2 Avoiding Re-renders on Keystroke

**Problem:** If `WatchlistSearchInput` is a child of `WatchListDetailPage`, and the page holds the watchlist items in state, typing in the search input must NOT re-render the entire table.

**Solution:** The search input manages its own local state (`query`, `results`, `isLoading`). It does not lift these values to the parent. The parent only receives callbacks (`onTickerAdded`, `onTickerAddFailed`) which are stable refs.

```typescript
// Stable callbacks using useCallback:
const handleTickerAdded = useCallback((item: WatchListItem) => {
  setWatchList(prev => {
    if (!prev) return prev;
    return { ...prev, item_count: prev.item_count + 1, items: [item, ...prev.items] };
  });
}, []);
```

The `WatchListTable` should be wrapped in `React.memo` if not already, to prevent re-renders when the search input's parent state changes:

```typescript
const MemoizedWatchListTable = React.memo(WatchListTable);
```

### 7.3 Autocomplete Dropdown — No Virtualization Needed

The API returns a maximum of 10 results. Even with already-added items grouped at the bottom, the total is < 20 rows. Standard DOM rendering is sufficient. No `react-window` or `react-virtualized` needed.

### 7.4 Lazy Loading

The `AutocompleteDropdown` component is lightweight (< 2KB gzipped) and doesn't warrant dynamic import overhead. However, `TagChipInput` and `InlineEditPanel` are only needed when a user edits a row, which is infrequent.

**Recommendation:** Lazy-load `InlineEditPanel` (which contains `TagChipInput`):

```typescript
const InlineEditPanel = React.lazy(
  () => import('./InlineEditPanel')
);

// In WatchListTable:
{editingSymbol === item.symbol && (
  <Suspense fallback={<div className="p-4 text-sm text-gray-400">Loading editor...</div>}>
    <InlineEditPanel item={item} onSave={handleSave} onCancel={closeEdit} />
  </Suspense>
)}
```

### 7.5 Tag Suggestions — Single Fetch, Client-Side Filter

The `GET /watchlists/tags` endpoint is called once when `TagChipInput` mounts. Results are cached in component state and filtered client-side as the user types. This avoids repeated API calls for tag suggestions.

If a user has a very large number of unique tags (approaching the LIMIT 50), the client-side filter is still instant (50 items is trivial).

### 7.6 Search Input Cleanup on Unmount

The debounce timer must be cleared on unmount to prevent state updates on an unmounted component:

```typescript
useEffect(() => {
  return () => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }
  };
}, []);
```

---

## 8. Testing Plan

### 8.1 Unit Tests

| Component | Test | Description |
|-----------|------|-------------|
| `WatchlistSearchInput` | `renders with placeholder` | Renders input with "Add a ticker..." placeholder |
| | `shows disabled state at limit` | When `itemCount >= maxItems`, input is disabled with limit message |
| | `debounces API calls` | Typing rapidly fires only one API call after 200ms pause |
| | `clears input after add` | After selecting a result, input text clears |
| | `keeps focus after add` | After adding, `document.activeElement` is still the input |
| `AutocompleteDropdown` | `renders results` | Shows result rows with symbol, name, exchange |
| | `separates already-added items` | Already-added items appear below divider with ✓ badge |
| | `highlights on arrow keys` | ArrowDown/ArrowUp cycle through selectable items |
| | `skips disabled items` | Arrow keys skip already-added items |
| | `shows no-results message` | When `results=[]` and `query.length > 0`, shows message |
| | `shows loading state` | When `isLoading=true`, shows "Searching..." |
| `TagChipInput` | `creates chip on Enter` | Typing "tech" + Enter creates a "tech" chip |
| | `creates chip on comma` | Typing "tech," creates a "tech" chip |
| | `lowercases tags` | Typing "TECH" + Enter creates "tech" (lowercased) |
| | `prevents duplicates` | Adding "tech" twice results in only one chip |
| | `removes chip on × click` | Clicking × on a chip removes it |
| | `removes last chip on Backspace` | Backspace on empty input removes last chip |
| | `respects maxTags limit` | At limit, new chips are not created |
| | `filters suggestions` | Typing "te" shows suggestions matching "te" |
| `InlineEditPanel` | `populates with existing data` | Opens with item's current notes, tags, targets |
| | `validates buy < sell` | Shows error when buy ≥ sell |
| | `calls onSave with draft` | Save button sends updated data |
| | `calls onCancel` | Cancel button calls `onCancel` without saving |
| | `Ctrl+Enter saves` | Keyboard shortcut triggers save |
| `useSlashFocus` | `focuses input on /` | Pressing `/` calls `inputRef.focus()` |
| | `ignores / in text input` | Does not focus when user is in another input |
| | `ignores / in textarea` | Does not focus when user is in a textarea |

**Test framework:** Jest + React Testing Library (matches existing project setup).

### 8.2 Integration Tests

| Test | Steps | Assertions |
|------|-------|------------|
| **Add ticker end-to-end** | 1. Render watchlist page with mock data. 2. Focus search input. 3. Type "AAPL". 4. Wait for debounce. 5. Press Enter. | API `POST` called with `{ symbol: "AAPL" }`. AAPL row appears in table. Search input is cleared. |
| **Add ticker via global search** | 1. Render watchlist page. 2. Type in global search bar. 3. Click `+ Add` button on a result. | API `POST` called. Result shows `✓ Added`. |
| **Inline edit save** | 1. Click edit button on a row. 2. Edit panel opens. 3. Set target buy = 150. 4. Add tag "tech". 5. Click Save. | API `PUT` called with correct data. Panel closes. Row reflects updated data. |
| **Inline edit cancel** | 1. Open edit panel. 2. Make changes. 3. Click Cancel. | Panel closes. No API call. Row shows original data. |
| **Remove ticker** | 1. Click remove button. 2. Confirm if confirmation prompt exists. | API `DELETE` called. Row removed from table. |
| **Batch add** | 1. Focus search. 2. Type "AAPL" → Enter. 3. Type "MSFT" → Enter. 4. Type "GOOGL" → Enter. | 3 API calls. 3 rows in table. Search input focused after each add. |
| **Limit enforcement** | 1. Load watchlist with 10 items. | Search input is disabled with "Watchlist full" message. |

### 8.3 Accessibility Tests

| Test | Method | Criteria |
|------|--------|----------|
| **Keyboard-only add** | Tab to search → type → ArrowDown → Enter | Ticker added without mouse |
| **Keyboard-only edit** | Tab to edit button → Enter → Tab through fields → Ctrl+Enter | Edit saved without mouse |
| **Screen reader: search** | VoiceOver/NVDA on search component | Announces "Search for a ticker to add to your watchlist", result count, highlighted option |
| **Screen reader: tag chips** | VoiceOver/NVDA on tag input | Announces each chip, "Remove tag: tech" on × button |
| **Focus trap: edit panel** | Open edit panel, Tab through all fields | Focus cycles within panel; Escape closes |
| **Contrast** | axe-core automated check | All text passes WCAG 2.1 AA (4.5:1 normal, 3:1 large) |
| **Reduced motion** | Set `prefers-reduced-motion: reduce` | No slide/fade animations; instant show/hide |

**Tools:** `@testing-library/jest-dom`, `axe-core` via `jest-axe` for automated a11y checks. Manual testing with VoiceOver (macOS) and NVDA (Windows).

### 8.4 Edge Case Tests

| Scenario | Test |
|----------|------|
| **Duplicate ticker** | Mock API returns 409 → toast shows "already in watchlist" → optimistic row removed |
| **Empty search result** | Type "xyznotreal" → dropdown shows "No matching tickers found" |
| **API timeout on search** | Mock API times out → dropdown shows "Search unavailable" |
| **API error on add** | Mock API returns 500 → toast shows error → optimistic row removed |
| **Crypto prefix** | Type "BTC" → results include X:BTCUSD |
| **Long tag** | Type 101-character tag → chip is not created |
| **51st tag** | At 50 tags, adding another does nothing |
| **Concurrent edits** | Open edit on row A, click edit on row B → row A auto-saves, row B opens |
| **Rapid add** | Add 3 tickers in < 1 second → all 3 appear, no lost mutations |
| **Unmount during search** | Navigate away while search is in-flight → no state update on unmounted component |

---

## 9. Migration & Rollout

### 9.1 Deprecating the Existing Modal

**Approach: Feature flag with parallel code paths.**

The `AddTickerModal` component is imported in `app/watchlist/[id]/page.tsx`:

```typescript
// Current:
import AddTickerModal from '@/components/watchlist/AddTickerModal';

// ...

{showAddModal && (
  <AddTickerModal onClose={() => setShowAddModal(false)} onAdd={handleAddTicker} />
)}
```

**Step 1:** Introduce a feature flag.

```typescript
// lib/featureFlags.ts (new file, or add to existing config)
export const FEATURE_FLAGS = {
  INLINE_WATCHLIST_ADD: process.env.NEXT_PUBLIC_FF_INLINE_WATCHLIST_ADD === 'true',
};
```

**Step 2:** Conditional rendering in the watchlist detail page.

```typescript
import { FEATURE_FLAGS } from '@/lib/featureFlags';

// ...

{FEATURE_FLAGS.INLINE_WATCHLIST_ADD ? (
  <WatchlistSearchInput
    watchlistId={watchList.id}
    existingSymbols={existingSymbols}
    itemCount={watchList.items.length}
    maxItems={10}
    onTickerAdded={handleTickerAdded}
    onTickerAddFailed={handleTickerAddFailed}
  />
) : (
  <>
    <button onClick={() => setShowAddModal(true)}>+ Add Ticker</button>
    {showAddModal && (
      <AddTickerModal onClose={() => setShowAddModal(false)} onAdd={handleAddTicker} />
    )}
  </>
)}
```

**Step 3:** Similarly gate the `EditTickerModal` vs `InlineEditPanel`:

```typescript
// When INLINE_WATCHLIST_ADD is true, onEdit opens inline edit (sets editingSymbol).
// When false, onEdit opens EditTickerModal (existing behavior).
```

**Step 4: Cleanup (after flag is 100% on):**

- Remove `AddTickerModal.tsx`
- Remove `EditTickerModal.tsx`
- Remove feature flag conditionals
- Remove `showAddModal` state and its button

### 9.2 Rollout Plan

| Phase | % Users | Duration | Goal |
|-------|---------|----------|------|
| **1. Internal** | Team only | 1 week | Functional testing, a11y audit |
| **2. Beta** | 10% of authenticated users | 1 week | Monitor error rates, gather qualitative feedback |
| **3. Wide rollout** | 50% | 1 week | Compare add-ticker completion rate between cohorts |
| **4. Full rollout** | 100% | — | Remove flag, delete old modal code |

**Flag delivery:** Environment variable (`NEXT_PUBLIC_FF_INLINE_WATCHLIST_ADD`). For per-user rollout (phases 2–3), the flag would need to be evaluated server-side based on `user.id % 100 < rolloutPercent` and passed as a prop. Alternatively, use a feature flag service if available.

### 9.3 Database Schema Changes

**No schema changes required for the core feature.** The `watch_list_items` table already stores `tags` as a `TEXT[]` array. The chip input reads and writes the same format.

**New endpoint `GET /watchlists/tags`** requires no schema change — it queries existing data.

**Optional improvement (not blocking):** Add a GIN index on `watch_list_items.tags` to speed up the tag aggregation query:

```sql
CREATE INDEX idx_watch_list_items_tags ON watch_list_items USING GIN (tags);
```

This is only needed if the tag query becomes slow at scale. For the current user base (free tier: 3 watchlists × 10 items = max 30 items per user), the query is trivial.

### 9.4 Backend Changes Required

| File | Change | Effort |
|------|--------|--------|
| `backend/main.go` | Add route `GET /watchlists/tags` | S |
| `backend/handlers/watchlist_handlers.go` | Add `GetUserTags` handler | S |
| `backend/database/watchlists.go` | Add `GetUserTags(userID)` query | S |
| `backend/database/stocks.go` | Modify `SearchStocks` to strip `X:` prefix for matching | S |
| `backend/main.go` (search handler) | Include `logo_url` in search response | S |

All backend changes are small and backward-compatible.

### 9.5 Frontend Files — Full Change List

| File | Status | Change |
|------|--------|--------|
| `components/watchlist/WatchlistSearchInput.tsx` | **NEW** | Inline search component |
| `components/watchlist/AutocompleteDropdown.tsx` | **NEW** | Dropdown for search results |
| `components/watchlist/InlineEditPanel.tsx` | **NEW** | Expandable row editor |
| `components/watchlist/TagChipInput.tsx` | **NEW** | Chip-based tag input |
| `hooks/useSlashFocus.ts` | **NEW** | `/` keyboard shortcut hook |
| `lib/stores/watchlistPageStore.ts` | **NEW** | Zustand store for cross-component communication |
| `lib/featureFlags.ts` | **NEW** | Feature flag definitions |
| `app/watchlist/[id]/page.tsx` | **MODIFIED** | Integrate new components, optimistic updates, Zustand store setup |
| `components/watchlist/WatchListTable.tsx` | **MODIFIED** | Add `editingSymbol` state, render `InlineEditPanel` |
| `components/TickerSearch.tsx` | **MODIFIED** | Add contextual `+ Add` button when on watchlist page |
| `components/Header.tsx` | **MODIFIED** | Remove "Watch Lists" nav link |
| `lib/api/watchlist.ts` | **MODIFIED** | Add `getUserTags()` method |
| `lib/api.ts` | **MINOR** | Ensure `searchSecurities` returns `logo_url` (depends on backend change) |
| `components/watchlist/AddTickerModal.tsx` | **DELETE** (after rollout) | Replaced by inline search |
| `components/watchlist/EditTickerModal.tsx` | **DELETE** (after rollout) | Replaced by inline edit panel |

---

## 10. Open Questions

| # | Question | Context | Recommendation |
|---|----------|---------|----------------|
| **Q1** | Should we add Zustand as a dependency for the global search CTA, or use a simpler approach? | The `Header` is rendered in the root layout, outside the watchlist page's component tree. React context won't work without restructuring layouts. Zustand is 1KB gzipped. Alternatives: module-level event emitter, `window` global. | **Zustand.** It's the standard lightweight option for cross-tree state in React. Minimal footprint, good DX. |
| **Q2** | Should the backend enforce `target_buy_price < target_sell_price`? | Currently only enforced client-side in `AddTickerModal`. The inline edit panel will enforce it too, but a direct API caller could bypass it. | **Yes.** Add validation in `UpdateWatchListItem` handler: return `400` if both are set and buy ≥ sell. |
| **Q3** | Should the search debounce be 200ms or 300ms? | Current codebase uses 300ms in both `AddTickerModal` and `TickerSearch`. The search query is fast (indexed DB lookup). 200ms feels more responsive for the inline experience. | **200ms** for `WatchlistSearchInput`. Keep 300ms for `TickerSearch` to avoid changing existing behavior. |
| **Q4** | Should we add `logo_url` and `current_price` to the search endpoint response? | `logo_url` requires joining the `tickers` table (already done — the search queries `tickers`). `current_price` would require a Polygon API call per result, adding latency. | **Add `logo_url`** (free — already in the query). **Defer `current_price`** — adds latency and complexity. Autocomplete without price is acceptable for v1. |
| **Q5** | How should the global search "Add" button handle the API call — directly, or via the Zustand store callback? | Direct: `TickerSearch` calls `watchListAPI.addTicker()` itself. Via store: `TickerSearch` calls `store.onQuickAdd()` which is set by the watchlist page. The store approach keeps the watchlist page as the single source of truth for mutations. | **Via store callback.** The watchlist page sets `onQuickAdd` in the store, and `TickerSearch` calls it. This ensures optimistic updates and error handling are centralized. |
| **Q6** | Do we need to add the GIN index on `tags` now? | The `GET /watchlists/tags` query unnests and aggregates tags. For max 30 items per user, it's instant. The index only matters at scale. | **Defer.** Add if we observe slow queries in production. |
| **Q7** | Should the inline edit panel auto-save on blur, or require explicit Save? | Auto-save is faster but risks unintended saves. Explicit Save is safer but adds a click. | **Explicit Save** for v1. Auto-save can be explored in v2 after user feedback. |
| **Q8** | How to handle the `/` shortcut if the page has other keyboard shortcuts (e.g., from the table sort or view switcher)? | No conflicts exist today — the table and view switcher don't use keyboard shortcuts. Future-proofing: consider a keyboard shortcut registry. | **No conflict today.** Document the shortcut in the codebase so future keyboard features check for conflicts. |
| **Q9** | Should the `+ Add` button in global search also support keyboard selection (e.g., Tab to button + Enter)? | The current global search uses Enter to navigate to `/ticker/:symbol`. Adding a Tab target for `+ Add` changes the keyboard flow. | **Yes, support Tab + Enter** on the `+ Add` button. The primary Enter behavior (navigate) is preserved. Users Tab to reach the add button explicitly. |
| **Q10** | Feature flag implementation: environment variable vs user-based rollout? | `NEXT_PUBLIC_FF_*` is baked at build time (Next.js constraint). Per-user rollout requires server-side evaluation. | **Phase 1 (internal): env var.** Phase 2+ (gradual): evaluate server-side per `user.id` and pass as a prop or use a feature flag service. |

---

*End of document.*
