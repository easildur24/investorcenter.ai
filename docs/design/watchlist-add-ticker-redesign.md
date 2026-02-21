# Product Design Document: Watchlist "Add Ticker" Redesign

**Product:** InvestorCenter.ai
**Feature:** Watchlist Add Ticker Experience
**Version:** 1.0
**Date:** 2026-02-21
**Status:** Draft — Pending Review

---

## Table of Contents

1. [Overview](#1-overview)
2. [User Research Insights](#2-user-research-insights)
3. [User Personas](#3-user-personas)
4. [User Journey Maps](#4-user-journey-maps)
5. [Design Principles](#5-design-principles)
6. [Feature Specifications](#6-feature-specifications)
7. [Edge Cases & Error States](#7-edge-cases--error-states)
8. [Accessibility Considerations](#8-accessibility-considerations)
9. [Success Metrics](#9-success-metrics)
10. [Out of Scope](#10-out-of-scope)

---

## 1. Overview

### Problem Statement

Adding a ticker to a watchlist on InvestorCenter currently requires opening a heavy modal dialog that presents five fields simultaneously (Symbol, Notes, Tags, Target Buy Price, Target Sell Price). The modal breaks the user's visual context, demands exact symbol knowledge with no autocomplete guidance, and exposes rarely-used fields upfront. Comma-separated tag entry is unintuitive. The "Watch Lists" top-nav link duplicates functionality already present in the user profile dropdown.

The result: a high-friction flow for what should be a 1–2 second action. Users who want to quickly track a stock are forced through a form designed for power-user metadata entry.

### Opportunity

Watchlist addition is the single highest-frequency action on the watchlist page. Reducing the interaction cost from "open modal → type symbol → skip 4 fields → submit" to "type → select → done" directly improves engagement, return visits, and watchlist utilization. Every major competitor (TradingView, Yahoo Finance, Robinhood) treats "add to watchlist" as a near-instant action.

### Goals

| # | Goal | Measure |
|---|------|---------|
| G1 | Replace modal with inline search-to-add | Modal removed; inline component ships |
| G2 | Live autocomplete with rich ticker metadata | Results show logo, name, exchange, price |
| G3 | Progressive disclosure for secondary fields | Notes, tags, targets move to post-add row edit |
| G4 | Chip-based tag input | Comma-separated text replaced with pill UI |
| G5 | Keyboard shortcut to focus search | `/` key activates search bar |
| G6 | Remove "Watch Lists" from top nav | Nav item removed; accessible via profile dropdown only |
| G7 | Global search integration | "Add to watchlist" CTA appears in global search when on a watchlist page |

---

## 2. User Research Insights

### Key Friction Points

**F1: Symbol Knowledge Barrier**
The current free-text symbol input assumes users know exact ticker symbols. This is especially problematic for crypto assets, which require an `X:` prefix (e.g., `X:BTCUSD`), and for international ADRs or less-known small-caps. Users who know a company by name but not its symbol must leave the page to look it up.

**F2: Context Disruption**
The modal overlay obscures the watchlist table. Users comparing prices, scanning IC Scores, or evaluating their portfolio lose visual context the moment they click "+ Add Ticker." After adding, they must re-orient themselves in the table.

**F3: Premature Field Exposure**
Notes, Tags, Target Buy Price, and Target Sell Price are shown on the same screen as the symbol input. Behavioral data from similar SaaS products suggests fewer than 15% of "add" actions include metadata entry on first pass. Users typically add a ticker first, then enrich it later.

**F4: Tag Discoverability**
Comma-separated text input provides no affordance for tag behavior. Users don't know tags exist, don't know the delimiter, and can't see existing tags to maintain consistency. There is no suggestion of previously-used tags.

**F5: Redundant Navigation**
"Watch Lists" appears in both the top nav bar (for authenticated users) and the profile dropdown under "My Watch Lists." This wastes horizontal nav space and creates ambiguity about which is the canonical path.

### User Mental Models

- **"Search, then act"**: Users expect to find a stock by typing its name or symbol, then take an action (add, view, compare). They do not expect to fill out a form.
- **"Add now, organize later"**: The primary intent when adding a ticker is tracking. Organization (tags, notes, price targets) is a secondary task performed during review sessions.
- **"I shouldn't have to remember anything"**: Autocomplete is table stakes. Users expect the system to resolve partial company names, handle symbol ambiguity, and display enough context (logo, exchange, price) to confirm they selected the right asset.

---

## 3. User Personas

### Persona 1: Sarah — Casual Retail Investor

| Attribute | Detail |
|-----------|--------|
| **Age** | 34 |
| **Experience** | 2 years investing; holds 8 positions across 2 brokerages |
| **Behavior** | Checks portfolio 2–3× per week. Reads headlines, acts on tips from podcasts and social media. Tracks ~15 stocks across 2 watchlists. |
| **Tools** | iPhone, occasionally laptop. Uses Yahoo Finance and Reddit for research. |
| **Pain Points** | Doesn't memorize ticker symbols. Knows "the company that makes Ozempic" but not `NVO`. Finds modals disruptive on mobile. Never uses tags or price targets — just wants to track a stock she heard about. |
| **Goal** | Add a ticker in under 3 seconds after hearing about it. |

### Persona 2: Marcus — Active Trader & Power User

| Attribute | Detail |
|-----------|--------|
| **Age** | 41 |
| **Experience** | 12 years trading; manages a personal portfolio of ~60 positions |
| **Behavior** | Checks markets multiple times daily. Maintains 3 watchlists: "Earnings Plays," "Swing Trades," "Long-Term Holds." Uses target prices and tags extensively. |
| **Tools** | Dual monitor desktop setup. Uses TradingView charts, InvestorCenter for IC Scores and screener. Keyboard-first workflow. |
| **Pain Points** | The modal interrupts his scan-and-add workflow. He wants to rapidly add 5–10 tickers from a screener session without leaving context. Tags via comma-separated text means he can't see what tags he's already used, leading to inconsistent naming ("tech" vs "Tech" vs "technology"). |
| **Goal** | Batch-add tickers with keyboard shortcuts, then enrich metadata inline without opening another modal. |

### Persona 3: Priya — Crypto-Curious Crossover Investor

| Attribute | Detail |
|-----------|--------|
| **Age** | 28 |
| **Experience** | 3 years in crypto, 1 year in equities. Holds BTC, ETH, SOL and a few meme coins alongside AAPL and VOO. |
| **Behavior** | Mixes crypto and stock tracking in a single watchlist. Checks prices daily, mostly on mobile. Active on Reddit — uses the Social view preset. |
| **Pain Points** | Doesn't know about the `X:` prefix convention. Tried typing "BTC" and "Bitcoin" — neither found the right result on first attempt. Confused by the distinction between `X:BTCUSD` and `BTC`. |
| **Goal** | Search by common crypto names ("Bitcoin", "Ethereum") and add them without needing to know internal symbol conventions. |

---

## 4. User Journey Maps

### Journey: Adding a Single Ticker

#### Current Flow (Modal)

```
Sarah hears about Novo Nordisk on a podcast
  │
  ├─ 1. Opens InvestorCenter, navigates to her watchlist
  ├─ 2. Clicks "+ Add Ticker" button
  ├─ 3. Modal opens, covering the watchlist table
  ├─ 4. Types "Novo" into the Symbol field — no results (search requires ≥1 char, but returns symbols, not company name matches reliably)
  ├─ 5. Tries "NVO" — autocomplete shows a result
  ├─ 6. Selects NVO from dropdown
  ├─ 7. Sees Notes, Tags, Target Buy, Target Sell fields — skips all of them
  ├─ 8. Clicks "Add Ticker"
  ├─ 9. Modal closes, page reloads with new ticker in the table
  └─ Total: ~12 seconds, 8 interactions, 1 context break
```

**Failure Mode:** If Sarah doesn't know "NVO," she opens a new tab, searches Google, finds the symbol, returns to InvestorCenter, and resumes at step 4. Total time: 30+ seconds.

#### Proposed Flow (Inline Search)

```
Sarah hears about Novo Nordisk on a podcast
  │
  ├─ 1. Opens InvestorCenter, navigates to her watchlist
  ├─ 2. Clicks the inline search bar (or presses "/")
  ├─ 3. Types "novo" — autocomplete shows:
  │      ┌──────────────────────────────────────────────────────┐
  │      │  🏢 NVO   Novo Nordisk A/S        NYSE    $128.45   │
  │      │  🏢 NOVN  Novan Inc               NASDAQ    $3.12   │
  │      └──────────────────────────────────────────────────────┘
  ├─ 4. Presses Enter or clicks NVO
  ├─ 5. NVO appears in watchlist table immediately with a subtle highlight
  ├─ 6. (Optional) She clicks the row's edit icon later to add notes or a target price
  └─ Total: ~3 seconds, 3 interactions, 0 context breaks
```

---

#### Current Flow: Marcus Batch-Adding Tickers

```
Marcus finishes a screener session with 6 candidates
  │
  ├─ For each ticker (×6):
  │   ├─ Click "+ Add Ticker" → modal opens
  │   ├─ Type symbol → select from dropdown
  │   ├─ Optionally: type tag "earnings-play", set target buy/sell
  │   ├─ Click "Add Ticker" → modal closes, page reloads
  │   └─ Wait for reload, then repeat
  └─ Total: ~90 seconds for 6 tickers, 6 modal open/close cycles
```

#### Proposed Flow: Marcus Batch-Adding Tickers

```
Marcus finishes a screener session with 6 candidates
  │
  ├─ 1. Presses "/" to focus inline search
  ├─ 2. Types "CRWD" → Enter → added (row highlights briefly)
  ├─ 3. Search bar stays focused — types "PANW" → Enter → added
  ├─ 4. Repeats for remaining 4 tickers
  ├─ 5. Scrolls to newly added rows, clicks edit icon on each
  ├─ 6. Adds "earnings-play" tag via chip selector (sees existing tags suggested)
  ├─ 7. Sets target buy/sell prices inline
  └─ Total: ~25 seconds for 6 tickers, 0 modal cycles, keyboard-only
```

---

#### Current Flow: Priya Adding Bitcoin

```
Priya wants to track Bitcoin
  │
  ├─ 1. Clicks "+ Add Ticker"
  ├─ 2. Types "Bitcoin" — no results
  ├─ 3. Types "BTC" — no results (symbol is X:BTCUSD)
  ├─ 4. Confused. Closes modal.
  ├─ 5. Googles "InvestorCenter bitcoin ticker symbol"
  ├─ 6. Finds nothing helpful
  ├─ 7. Tries "X:BTC" — sees X:BTCUSD in dropdown
  ├─ 8. Selects it, submits
  └─ Total: ~45 seconds, significant frustration, near-abandonment
```

#### Proposed Flow: Priya Adding Bitcoin

```
Priya wants to track Bitcoin
  │
  ├─ 1. Types "bitcoin" in inline search
  ├─ 2. Autocomplete shows:
  │      ┌──────────────────────────────────────────────────────────┐
  │      │  ₿ X:BTCUSD   Bitcoin / USD          Crypto   $97,412  │
  │      │  🏢 BITO      ProShares Bitcoin ETF   NYSE      $24.31  │
  │      │  🏢 IBIT      iShares Bitcoin Trust   NASDAQ    $52.80  │
  │      └──────────────────────────────────────────────────────────┘
  ├─ 3. Selects X:BTCUSD → added instantly
  └─ Total: ~4 seconds, no prefix knowledge required
```

---

## 5. Design Principles

### P1: Search-First, Form-Never

The primary add-ticker interaction is a search. Not a form. The user types a query, sees results, and selects. No form fields, no submit button, no modal. The search bar is the only required input.

### P2: Progressive Disclosure

Show only what's needed at each step. Adding a ticker requires one field: the ticker. Notes, tags, and price targets are surfaced only when the user explicitly chooses to enrich a row after adding.

### P3: Zero Unnecessary Typing

Autocomplete resolves partial names, common abbreviations, and crypto names. The system does the work of matching "bitcoin" to `X:BTCUSD` so the user doesn't have to learn internal conventions.

### P4: Maintain Visual Context

The watchlist table is never obscured. The inline search bar lives above or within the table header. Adding a ticker inserts a row into the visible table without navigation or overlay.

### P5: Keyboard-Navigable Throughout

Every interaction — focus, search, select, add, edit, tag — is achievable via keyboard. Power users should never need to reach for the mouse.

### P6: Instant Feedback

Adding a ticker produces an immediate visual response: the row appears in the table with a brief highlight animation. No loading spinners for the table insert (optimistic UI with rollback on error).

---

## 6. Feature Specifications

### 6.1 Inline Search-to-Add Component

**Location:** Replaces the current "+ Add Ticker" button. Positioned at the top of the watchlist detail page (`/watchlist/[id]`), inside the toolbar area above the table, left-aligned alongside existing filter controls.

**Component Structure:**

```
┌─────────────────────────────────────────────────────────────────────────┐
│  🔍  Add a ticker...                                          / ⌨     │
└─────────────────────────────────────────────────────────────────────────┘
```

- Left icon: magnifying glass (search affordance)
- Placeholder text: `"Add a ticker..."` — communicates both purpose and interaction model
- Right badge: keyboard shortcut hint (`/`) — shown only on desktop, hidden on touch devices
- Width: 320px default, expands to 420px on focus (smooth transition, 150ms ease-out)

**States:**

| State | Behavior |
|-------|----------|
| **Empty / Idle** | Placeholder visible. Right-side `/` shortcut badge shown. Muted border. |
| **Focused** | Border becomes primary blue (`#3B82F6`). Placeholder persists until first keystroke. Input width expands. If the user has previously used tags, a subtle hint appears below: "Press Enter to add, Tab to edit details." |
| **Typing (no results yet)** | Debounce of 200ms before API call fires. No spinner until 200ms elapsed. |
| **Loading** | Subtle inline spinner replaces the magnifying glass icon. Input remains editable. |
| **Results Shown** | Autocomplete dropdown opens below (see 6.2). |
| **No Results** | Dropdown shows: "No matching tickers found. Try a different name or symbol." |
| **Error (network)** | Dropdown shows: "Search unavailable. Check your connection and try again." with a "Retry" link. |
| **Already Added** | Result row shows a checkmark badge and muted text: "Already in watchlist." Row is not clickable. |
| **Ticker Added (success)** | Input clears. Brief green checkmark flash in the input (300ms). Focus remains in input for rapid sequential adds. |
| **Limit Reached** | Input disabled. Placeholder changes to: "Watchlist full (10/10 tickers)." Tooltip on hover: "Remove a ticker to add more, or upgrade for higher limits." |

**Behavior:**

- On focus: fetch no initial results (wait for input).
- On input: debounce 200ms, then call `GET /api/v1/markets/search?q={query}`.
- On Escape: close dropdown, blur input.
- On blur (without selection): close dropdown, clear input.
- After successful add: clear input text, keep focus, close dropdown.
- Search matches against: symbol (exact prefix match, weighted highest), company name (substring match), and common aliases (e.g., "bitcoin" → `X:BTCUSD`).

### 6.2 Autocomplete Dropdown

**Layout per result row:**

```
┌──────────────────────────────────────────────────────────────┐
│  [Logo]  AAPL    Apple Inc                NASDAQ    $189.42  │
│  [Logo]  AAPD    Direxion Daily AAPL...   NYSE       $14.80  │
│  [Logo]  AAPU    Direxion Daily AAPL...   NYSE       $32.15  │
│                                                              │
│  ── Already in watchlist ──────────────────────────────────  │
│  [Logo]  AAPL    Apple Inc          ✓     NASDAQ    $189.42  │
└──────────────────────────────────────────────────────────────┘
```

**Fields per row:**

| Field | Source | Format |
|-------|--------|--------|
| Logo | `tickers.logo_url` | 24×24px rounded square. Fallback: first letter of symbol in a colored circle (color derived from symbol hash). Crypto: currency icon (₿, Ξ, etc.) or generic crypto icon. |
| Symbol | `tickers.symbol` | Bold, monospace. Max 12 chars. Highlight matching substring in the query color. |
| Name | `tickers.name` | Regular weight, truncated with ellipsis at 28 chars. |
| Exchange | `tickers.exchange` | Small badge/pill: `NYSE`, `NASDAQ`, `CRYPTO`, `OTC`. Color-coded by exchange. |
| Price | Real-time or last close | Right-aligned. Format: `$X.XX` for stocks/ETFs, `$XX,XXX` for crypto. Omitted if unavailable. |
| Already-added indicator | Client-side check | Checkmark icon + row muted to 50% opacity. Row not clickable. Grouped at bottom under a divider labeled "Already in watchlist." |

**Ordering Logic:**

1. Exact symbol match (e.g., query "AAPL" → AAPL first)
2. Symbol prefix match, sorted by market cap descending
3. Name substring match, sorted by market cap descending
4. Already-in-watchlist results grouped last, separated by divider

**Maximum results:** 8 (to prevent excessive scrolling in the dropdown)

**Keyboard Navigation:**

| Key | Action |
|-----|--------|
| `↓` / `↑` | Move highlight through results. Skips already-added rows. |
| `Enter` | Add highlighted ticker to watchlist. If no highlight, add first non-added result. |
| `Tab` | Add highlighted ticker and immediately open inline row edit for that ticker. |
| `Escape` | Close dropdown, blur search input. |
| `Meta+Enter` / `Ctrl+Enter` | Add ticker and open inline row edit (same as Tab — alternative for users who expect modifier keys). |

**Visual Details:**

- Dropdown shadow: `0 4px 16px rgba(0, 0, 0, 0.12)` (light mode), `0 4px 16px rgba(0, 0, 0, 0.4)` (dark mode)
- Max height: 400px, scrollable if more than 8 results
- Active/highlighted row: light blue background (`#EFF6FF` light / `#1E3A5F` dark)
- Border radius: 8px, matching existing card styles
- Appears 4px below the search input (no gap)

### 6.3 Instant Add Behavior

**On Enter / Click of a result:**

1. **Optimistic insert**: Immediately add a new row to the watchlist table at the top (or at the current sort position). Row appears with a subtle slide-down animation (200ms) and a brief background highlight pulse (green tint, fades over 1.5s).
2. **API call**: `POST /api/v1/watchlists/{id}/items` with `{ symbol: "AAPL" }`. No other fields sent.
3. **On success**: Row persists. Real-time price data populates within the next polling cycle (≤30 seconds) or via immediate fetch.
4. **On failure**: Row is removed with a slide-up animation. Toast notification appears: "Failed to add AAPL. Please try again." Toast includes a "Retry" action button.
5. **Input behavior post-add**: Input text clears. Focus remains in the search input. Dropdown closes. User can immediately type the next ticker.

**Microcopy — Toast Messages:**

| Scenario | Message |
|----------|---------|
| Success | No toast (the row appearing is sufficient feedback). |
| Duplicate | "AAPL is already in this watchlist." (also prevented by disabled row in dropdown) |
| Network error | "Couldn't add AAPL — check your connection and try again." [Retry] |
| Server error | "Something went wrong adding AAPL. Please try again." [Retry] |
| Limit reached | "This watchlist is full (10/10). Remove a ticker to add more." |
| Ticker not found | "Ticker not recognized. Try searching by company name." |

### 6.4 Inline Row Editing for Secondary Fields

**Trigger:** Click the edit icon (pencil) on any watchlist row, or press `Tab`/`Ctrl+Enter` when adding a ticker.

**Behavior:** The table row expands downward to reveal an inline edit panel, pushing subsequent rows down. No modal. No navigation.

**Edit Panel Layout:**

```
┌──────────────────────────────────────────────────────────────────────┐
│  AAPL   Apple Inc   $189.42   +1.23 (+0.65%)   ...   [✏️] [🗑️]     │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Target Buy     Target Sell       Tags                               │
│  ┌──────────┐   ┌──────────┐     ┌───────────────────────────────┐  │
│  │ $ 175.00 │   │ $ 210.00 │     │ [tech ×] [long-term ×]  + ▏  │  │
│  └──────────┘   └──────────┘     └───────────────────────────────┘  │
│                                                                      │
│  Notes                                                               │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │ Watching for Q4 earnings beat. AI revenue segment growing.   │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│                                            [Cancel]  [Save Changes] │
└──────────────────────────────────────────────────────────────────────┘
```

**Field Specifications:**

| Field | Input Type | Validation | Placeholder |
|-------|-----------|------------|-------------|
| Target Buy Price | Numeric input with `$` prefix | Must be > 0. Must be < Target Sell (if set). Max 2 decimal places. | `"Buy target"` |
| Target Sell Price | Numeric input with `$` prefix | Must be > 0. Must be > Target Buy (if set). Max 2 decimal places. | `"Sell target"` |
| Tags | Chip/pill multi-select (see 6.5) | Max 50 tags, each max 100 chars. | `"Add a tag..."` |
| Notes | Auto-expanding textarea | Max 10,000 chars. | `"Add notes..."` |

**Save Behavior:**

- "Save Changes" calls `PUT /api/v1/watchlists/{id}/items/{symbol}` with updated fields.
- On save: panel collapses, row updates in-place.
- "Cancel" discards changes and collapses the panel.
- Clicking another row's edit button saves the current row automatically, then opens the new row.
- Unsaved changes warning: if the user clicks away (blur outside the panel), show a subtle prompt: "You have unsaved changes. Save or discard?"

**Keyboard Shortcuts in Edit Panel:**

| Key | Action |
|-----|--------|
| `Ctrl+Enter` / `Meta+Enter` | Save changes |
| `Escape` | Cancel and collapse |
| `Tab` | Move between fields |

### 6.5 Chip/Pill Tag Input Component

**Replaces:** The current comma-separated text input in both `AddTickerModal` and `EditTickerModal`.

**Appearance:**

```
┌───────────────────────────────────────────────────────┐
│  [tech ×]  [earnings-play ×]  [long-term ×]  Add...▏  │
└───────────────────────────────────────────────────────┘
```

**Each chip:**
- Background: `bg-blue-100` (light) / `bg-blue-900` (dark)
- Text: `text-blue-800` (light) / `text-blue-200` (dark)
- Remove button: `×` icon, visible on hover or focus. On click/Enter: removes tag with a brief fade-out (150ms).
- Max display width: truncate long tag names with ellipsis at 20 chars. Full name in tooltip.

**Input Behavior:**

| Action | Result |
|--------|--------|
| Type text + press `Enter` | Creates a new chip with the entered text. Lowercased and trimmed. |
| Type text + press `,` | Creates a new chip (familiar behavior for users accustomed to comma-separated entry). |
| Type text + press `Tab` | Creates a new chip and moves focus to the next form field. |
| `Backspace` on empty input | Selects the last chip (highlighted state). Second `Backspace` removes it. |
| Click `×` on a chip | Removes that tag. |
| Type in input | Filters a suggestion dropdown of previously-used tags across all watchlists. |

**Tag Suggestions Dropdown:**

- Source: `GET /api/v1/watchlists/tags` (new endpoint — returns all unique tags the user has ever used, sorted by frequency).
- Shows up to 6 suggestions, filtered by current input text.
- Keyboard navigable (↑/↓ to highlight, Enter to select).
- Displays tag name and usage count: `tech (5 tickers)`.
- If the typed text doesn't match any existing tag, show: `Create "new-tag"` as the first option.

**Normalization:**
- Tags are lowercased on creation.
- Leading/trailing whitespace trimmed.
- Duplicate tags (case-insensitive) are rejected silently (chip doesn't appear a second time).
- Empty strings are ignored.

### 6.6 Keyboard Shortcut: `/` to Focus Search

**Behavior:**

- Pressing `/` anywhere on the watchlist detail page (`/watchlist/[id]`) focuses the inline search input.
- The shortcut is suppressed when focus is already inside an input, textarea, or contenteditable element (to avoid capturing normal typing).
- The `/` character is NOT typed into the search input (it's consumed by the shortcut handler).
- On focus, if the dropdown was previously dismissed, it does not reopen until the user types.

**Implementation Notes:**

- Attach a global `keydown` listener at the page level.
- Guard: `if (event.key === '/' && !isEditableElement(document.activeElement)) { event.preventDefault(); searchInputRef.focus(); }`
- The `/` badge in the search bar serves as a discoverability affordance. It is rendered as:

```
<kbd class="text-xs text-gray-400 border border-gray-300 rounded px-1.5 py-0.5 font-mono">/</kbd>
```

- On mobile viewports (< 768px): hide the `kbd` badge. The shortcut still works if a hardware keyboard is connected but is not advertised.

### 6.7 Global Search Integration

**Context:** InvestorCenter has a global search bar in the header (`components/TickerSearch.tsx`) that currently navigates to `/ticker/{symbol}` on selection. When the user is on a watchlist detail page, the global search should additionally surface an "Add to watchlist" action.

**Behavior:**

When the current route matches `/watchlist/[id]`:

- Each result row in the global search dropdown gains a secondary action button: `+ Add` (right-aligned, small pill button).
- Clicking `+ Add` calls `POST /api/v1/watchlists/{id}/items` for the current watchlist.
- Clicking the result row itself (outside the `+ Add` button) navigates to `/ticker/{symbol}` as before.
- If the ticker is already in the current watchlist, the button shows `✓ Added` (disabled, muted style).

**Dropdown Layout (on watchlist page):**

```
┌──────────────────────────────────────────────────────────────┐
│  🔍  Search tickers...                                       │
├──────────────────────────────────────────────────────────────┤
│  [Logo]  AAPL   Apple Inc          NASDAQ   $189.42  [+ Add] │
│  [Logo]  AAPD   Direxion AAPL...   NYSE      $14.80  [+ Add] │
│  [Logo]  AAPL   Apple Inc          NASDAQ   $189.42  [✓ Added]│
├──────────────────────────────────────────────────────────────┤
│  Recent Searches                                              │
│  TSLA · MSFT · NVDA                                           │
└──────────────────────────────────────────────────────────────┘
```

**Microcopy:**

- Button default: `+ Add` (16px, semibold, primary blue)
- Button on hover: `+ Add to [watchlist name]` (tooltip, 200ms delay)
- Button after add: `✓ Added` (green text, disabled)
- Button on error: flash red briefly, revert to `+ Add`

**Non-watchlist pages:** The `+ Add` button does not appear. The global search behaves exactly as it does today.

### 6.8 Navigation Change: Remove "Watch Lists" from Top Nav

**Current state:** The `Header.tsx` component renders a "Watch Lists" nav link for authenticated users in the top navigation bar, alongside Home, Screener, Crypto, and Reddit Trends. The same destination is accessible via the user profile dropdown under "My Watch Lists."

**Change:**

- Remove the `Watch Lists` item from the top nav bar (`components/Header.tsx`).
- Retain "My Watch Lists" in the user profile dropdown (no change).
- The freed horizontal space improves nav readability, especially on smaller desktop viewports and tablets.

**Migration Concern:**

- Users who currently rely on the top-nav link will need to adapt. This is low-risk because:
  - The profile dropdown path exists and is identical.
  - Direct URL access (`/watchlist`) remains functional.
  - No functionality is removed — only the redundant nav link.

---

## 7. Edge Cases & Error States

### 7.1 Invalid or Unknown Symbol

| Scenario | Handling |
|----------|----------|
| User types a string that matches no tickers | Dropdown shows: "No matching tickers found. Try a different name or symbol." |
| User types a valid company name but an inactive/delisted ticker | If the ticker exists in the `tickers` table with a delisted status, show it with a "Delisted" badge and allow adding. If not in the table, show "No matching tickers." |
| User pastes a URL or long string | Truncate input at 30 characters. Search runs on truncated value. |

### 7.2 Duplicate Ticker

| Scenario | Handling |
|----------|----------|
| Ticker already in current watchlist | Dropdown: row appears with `✓` badge, muted, non-clickable. Grouped under "Already in watchlist" divider. |
| User somehow bypasses UI and sends duplicate via API | Backend returns `409 Conflict`. Toast: "AAPL is already in this watchlist." |

### 7.3 Network Errors

| Scenario | Handling |
|----------|----------|
| Search API fails (timeout, 500) | Dropdown shows: "Search unavailable. Check your connection and try again." [Retry]. |
| Add API fails after optimistic insert | Row slides out of table. Toast: "Couldn't add AAPL — check your connection and try again." [Retry]. |
| Inline edit save fails | Keep edit panel open. Inline error message below Save button: "Save failed. Try again." Button text changes to "Retry Save". |

### 7.4 Crypto Prefix Handling

| Scenario | Handling |
|----------|----------|
| User types "bitcoin", "BTC", "btc" | Search backend resolves to `X:BTCUSD`. Displayed in results with the full symbol `X:BTCUSD` and label `Bitcoin / USD`. |
| User types "X:BTC" | Matched as symbol prefix. Results show `X:BTCUSD`, `X:BTCEUR`, etc. |
| User types "ETH" | Matches both `ETH` (if a stock ticker exists) and `X:ETHUSD`. Both shown, ordered by relevance (exact stock match first, crypto second, unless crypto is more popular by market cap). |

**Backend requirement:** The `/markets/search` endpoint must be enhanced to:
1. Match the `name` field of crypto tickers (e.g., "Bitcoin" → `X:BTCUSD`).
2. Strip `X:` prefix for matching purposes (so "BTC" matches `X:BTCUSD`).
3. Include `asset_type` in results so the frontend can render appropriate icons.

### 7.5 Watchlist Limits

| Scenario | Handling |
|----------|----------|
| Watchlist has 10/10 tickers (limit reached) | Search input is disabled. Placeholder: "Watchlist full (10/10 tickers)." Tooltip: "Remove a ticker to add more." |
| User reaches limit mid-batch-add (adds #10 then tries #11) | Add #10 succeeds. Input re-renders as disabled. Toast: "Watchlist full. Remove a ticker or upgrade for more." |

### 7.6 Concurrent Edits

| Scenario | Handling |
|----------|----------|
| Two tabs open on same watchlist; user adds ticker in tab A | Tab B will not reflect the change until next page load or polling cycle. No real-time sync in v1. |
| User adds a ticker that was removed by another session | Backend processes the add normally (it's a new insert). No conflict. |

### 7.7 Empty Watchlist State

When the watchlist has 0 tickers, the inline search bar is prominently displayed with an empty state illustration and message:

```
┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│              📊                                                  │
│                                                                  │
│    Your watchlist is empty                                       │
│    Search for a ticker to start tracking                         │
│                                                                  │
│    🔍  Add a ticker...                                    /      │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

The search bar is always visible regardless of empty state — it just gets more visual prominence when the table is empty.

---

## 8. Accessibility Considerations

### 8.1 Keyboard Navigation

All interactions are fully keyboard-accessible:

| Component | Keyboard Support |
|-----------|------------------|
| Inline search input | Focusable via `Tab` or `/` shortcut. Standard text input behavior. |
| Autocomplete dropdown | `↓`/`↑` arrow keys navigate results. `Enter` selects. `Escape` dismisses. |
| Inline edit panel | `Tab` cycles through fields. `Ctrl+Enter` saves. `Escape` cancels. |
| Chip/pill tags | `Backspace` on empty input selects last chip. Arrow keys navigate between chips. `Delete`/`Backspace` on selected chip removes it. |
| Edit/Remove buttons | Focusable via `Tab`. Activated via `Enter` or `Space`. |

### 8.2 Screen Reader Support

**Search Input:**
- `role="combobox"` with `aria-expanded`, `aria-controls`, `aria-activedescendant`
- `aria-label="Search for a ticker to add to your watchlist"`
- Live region announces result count: `aria-live="polite"` → "5 results found" / "No results found"

**Autocomplete Dropdown:**
- `role="listbox"` on the container
- Each result: `role="option"` with `aria-selected` on highlighted item
- Already-added items: `aria-disabled="true"` with `aria-label="AAPL, Apple Inc, already in watchlist"`

**Chip Tags:**
- Each chip: `role="option"` within a `role="listbox"` container, or `role="group"` with individual `role="button"` for remove
- Remove button: `aria-label="Remove tag: tech"`
- Tag input: `aria-label="Add a tag. Type and press Enter to create."`

**Inline Edit Panel:**
- `aria-expanded="true"` on the row when panel is open
- Panel: `role="region"` with `aria-label="Edit AAPL details"`
- Save/Cancel buttons: standard `role="button"` with clear labels

**Toast Notifications:**
- `role="alert"` with `aria-live="assertive"` for errors
- `aria-live="polite"` for success/info messages

### 8.3 Focus Management

- When the autocomplete dropdown opens, focus remains in the input. `aria-activedescendant` tracks the visually highlighted item.
- When inline edit opens, focus moves to the first field (Target Buy Price).
- When inline edit is saved/cancelled, focus returns to the edit button of that row.
- When a ticker is added, focus remains in the search input (for batch adding).
- When a toast appears, it does not steal focus.

### 8.4 Color and Contrast

- All text meets WCAG 2.1 AA contrast ratios (4.5:1 for normal text, 3:1 for large text).
- Exchange badges and chip tags use both color and text labels (never color alone to convey meaning).
- Price change indicators use both color (green/red) and a `+`/`−` prefix.
- Dark mode support: all states tested against both light and dark palettes.
- Focus rings: visible `outline` on all interactive elements (2px solid, offset 2px, primary blue).

### 8.5 Motion and Animation

- All animations respect `prefers-reduced-motion`. When enabled:
  - Row insert: no slide animation, instant appearance.
  - Highlight pulse: no animation, static highlight for 2 seconds then removed.
  - Dropdown open: no transition, instant display.

---

## 9. Success Metrics

### Primary KPIs

| Metric | Current Baseline (Estimated) | Target | Measurement |
|--------|------------------------------|--------|-------------|
| **Time to add a ticker** | ~12 seconds (modal flow) | < 4 seconds (inline flow) | Frontend event timing: search focus → successful add |
| **Add-ticker completion rate** | ~65% (users who open modal and complete the add) | > 90% | Funnel: search input focused → ticker added |
| **Tickers added per session** | 1.2 avg | > 2.0 avg | Count of add events per unique session on watchlist pages |
| **Watchlist feature adoption** | Baseline TBD | +25% increase in users with ≥1 non-empty watchlist | Weekly cohort: users who added ≥1 ticker / total authenticated users |

### Secondary KPIs

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Tag adoption rate** | > 30% of watchlist items have ≥1 tag (up from est. ~5%) | Database query: items with non-empty tags / total items |
| **Keyboard shortcut usage** | > 15% of add-ticker flows initiated via `/` | Frontend event: shortcut triggered / total search focuses |
| **Target price fill rate** | > 20% of watchlist items have a target buy or sell price | Database query |
| **Inline edit engagement** | > 40% of users use inline edit within first week post-add | Frontend event: edit panel opened within 7 days of add |
| **Global search add-to-watchlist CTR** | > 5% of searches on watchlist pages result in an add | Frontend event: `+ Add` clicked / global search sessions on watchlist pages |
| **Crypto add success rate** | > 85% of users who search for a crypto name find and add it on first attempt | Funnel: crypto name typed → crypto ticker added (no backspace/retry) |

### Guardrail Metrics (Should Not Worsen)

| Metric | Threshold |
|--------|-----------|
| Watchlist page load time | No regression > 200ms |
| API error rate on `/markets/search` | < 1% |
| Watchlist deletion rate | Should not increase (validates that easier adding doesn't lead to noise/regret) |

---

## 10. Out of Scope

The following are explicitly **not** addressed in this iteration:

| Item | Rationale |
|------|-----------|
| **Bulk import redesign** | The existing CSV bulk import (`/watchlists/{id}/bulk`) works and is used by < 2% of users. Will revisit if adoption grows. |
| **Multi-watchlist add** | Adding a ticker to multiple watchlists at once (e.g., "Add AAPL to Earnings Plays AND Long-Term Holds"). Adds complexity to the inline flow. Potential v2 feature. |
| **Drag-and-drop reordering** | Row reordering exists via API (`/reorder`) but has no UI. Separate initiative. |
| **Watchlist sharing / collaboration** | Social features (share watchlist with a friend, public watchlists) are a separate product decision. |
| **Mobile-native app** | This redesign targets the responsive web app. Native mobile (iOS/Android) is not in scope. |
| **Real-time price in autocomplete** | v1 will show the last available price (last close or cached real-time). Streaming live prices into the dropdown is deferred. |
| **Watchlist limit increase / pricing tier** | The 10-ticker and 3-watchlist limits are product/business decisions outside this design scope. Messaging for limits is in-scope; changing them is not. |
| **"Star" icon on ticker detail pages** | Adding a one-click "Add to default watchlist" star/bookmark on `/ticker/{symbol}` pages (inspired by Google Finance) is a strong idea but is a separate feature with its own edge cases (which watchlist? what if limit reached?). Recommended for v2. |
| **Autocomplete for tags** | v1 ships with tag suggestions from previously-used tags. Auto-generating tags from sector/industry data (e.g., auto-tagging AAPL as "tech, mega-cap") is out of scope. |
| **Undo add** | An "Undo" toast after adding a ticker (like Gmail's "Undo send") is a polish feature. v1 relies on the existing Remove button. |
| **Search analytics / popular tickers** | Showing "Trending tickers" or "Popular on InvestorCenter" in the empty search dropdown is deferred. The current global search shows popular stocks (AAPL, GOOGL, MSFT, TSLA, AMZN) — this can be replicated in v2 if validated. |

---

*End of document. For questions or feedback, contact the product design team.*
