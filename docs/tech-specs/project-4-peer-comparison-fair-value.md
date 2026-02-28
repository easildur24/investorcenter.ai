# Tech Spec: Project 4 — Peer Comparison & Fair Value

**Parent PRD:** `docs/prd-enhanced-fundamentals-experience.md`
**Priority:** P1
**Estimated Effort:** 3 sprints
**Dependencies:** Project 1 (peers + fair-value API endpoints)

---

## 1. Overview

Build two major interactive features: (1) a `PeerComparisonPanel` showing the stock vs. its top 5 algorithmically-selected peers in a side-by-side comparison table, and (2) a `FairValueGauge` visualizing DCF, Graham Number, and EPV fair value estimates with a margin-of-safety indicator. Both are Premium-gated features that answer the core user question: "Is this stock cheap or expensive?"

## 2. Architecture Context

### Data Sources

**Peers API:** `GET /api/v1/stocks/:ticker/peers` (Project 1)
```typescript
interface PeersResponse {
  ticker: string;
  ic_score: number;
  peers: Array<{
    ticker: string;
    company_name: string;
    ic_score: number;
    similarity_score: number;
    metrics: {
      pe_ratio: number;
      roe: number;
      revenue_growth_yoy: number;
      net_margin: number;
      debt_to_equity: number;
      market_cap: number;
    };
  }>;
  stock_metrics: Record<string, number>;
  avg_peer_score: number;
  vs_peers_delta: number;
}
```

**Fair Value API:** `GET /api/v1/stocks/:ticker/fair-value` (Project 1)
```typescript
interface FairValueResponse {
  ticker: string;
  current_price: number;
  models: {
    dcf: { fair_value: number; upside_percent: number; confidence: string; inputs: Record<string, number> };
    graham_number: { fair_value: number; upside_percent: number; confidence: string };
    epv: { fair_value: number; upside_percent: number; confidence: string };
  };
  analyst_consensus: {
    target_price: number;
    upside_percent: number;
    num_analysts: number;
    consensus: string;
  };
  margin_of_safety: {
    avg_fair_value: number;
    zone: 'undervalued' | 'fairly_valued' | 'overvalued';
    description: string;
  };
  meta: {
    suppressed: boolean;
    suppression_reason: string | null;
  };
}
```

### Existing Related Backend

| Capability | Location | Status |
|---|---|---|
| Peer comparison algorithm (5-factor similarity) | `ic-score-service/pipelines/utils/peer_comparison.py` | Computed, stored in `stock_peers` table |
| Peer DB queries with IC Scores | `backend/database/ic_score_phase3.go` → `GetStockPeersWithScores()` | Ready |
| Peer comparison summary | `backend/database/ic_score_phase3.go` → `GetPeerComparisonSummary()` | Ready |
| DCF fair value | `ic-score-service/pipelines/fair_value_calculator.py` | Computed, stored in `fundamental_metrics_extended` |
| Graham Number | FMP API → `FMPRatiosTTM.GrahamNumberTTM` | Available via `fmpClient` |
| EPV | `ic-score-service/pipelines/fair_value_calculator.py` | Computed |
| Analyst consensus | `models.AnalystRatings` + FMP API | Available |

### Placement

Both components live on the ticker page. The `PeerComparisonPanel` appears as an expandable section below the TickerFundamentals sidebar or as a dedicated sub-section within the MetricsTab. The `FairValueGauge` appears in the TickerFundamentals sidebar area, below the Health Card.

---

## 3. Component Design: Peer Comparison

### 3.1 `PeerComparisonPanel`

**File:** `components/ticker/PeerComparisonPanel.tsx`

**Props:**

```typescript
interface PeerComparisonPanelProps {
  ticker: string;
  isPremium: boolean;
}
```

**Collapsed State (always visible):**

```
┌──────────────────────────────────────────────────────┐
│  Compare to 5 similar companies                  [▶] │
│  Peers: MSFT, GOOGL, META, ORCL, CRM   Avg IC: 75  │
└──────────────────────────────────────────────────────┘
```

**Expanded State (Premium):**

```
┌──────────────────────────────────────────────────────────────────┐
│  Peer Comparison                                           [▲]  │
│  How peers are selected ⓘ                                       │
│                                                                  │
│  ┌────────┬────────┬────────┬────────┬────────┬────────┬───────┐│
│  │ Metric │ AAPL   │ MSFT   │ GOOGL  │ META   │ ORCL   │ CRM  ││
│  ├────────┼────────┼────────┼────────┼────────┼────────┼───────┤│
│  │IC Score│ 78.5   │ 82.1 ★ │ 76.3   │ 74.8   │ 71.2   │ 69.5 ││
│  │P/E     │ 28.5   │ 33.2   │ 24.1 ★ │ 22.8 ★ │ 35.1   │ 45.2 ││
│  │ROE     │ 147 ★  │ 38.5   │ 28.9   │ 32.1   │ 115    │ 8.2  ││
│  │Rev Grw │ 5.1%   │ 12.3% ★│ 14.2% ★│ 22.1% ★│ 8.3%   │ 11.2%││
│  │Net Mrg │ 25.3%  │ 35.1% ★│ 22.8%  │ 28.9%  │ 31.2%  │ 5.8% ││
│  │D/E     │ 1.87   │ 0.42 ★ │ 0.08 ★ │ 0.31 ★ │ 5.12   │ 0.55 ││
│  │Mkt Cap │ $2.8T  │ $3.1T  │ $2.1T  │ $1.4T  │ $380B  │ $290B││
│  └────────┴────────┴────────┴────────┴────────┴────────┴───────┘│
│                                                                  │
│  Similarity: MSFT 87% │ GOOGL 82% │ META 79% │ ORCL 65% │ CRM 61%│
│                                                                  │
│  Summary: AAPL has the highest ROE among peers but below-average │
│  revenue growth. IC Score ranks 2nd out of 6.                    │
└──────────────────────────────────────────────────────────────────┘
```

**Cell Color Coding:**
- Best value in row: `text-green-400 font-semibold` + star indicator (★)
- Worst value in row: `text-red-400`
- Stock's own column: `bg-ic-blue/5` background highlight

**Clickable Peer Tickers:** Each peer ticker in the header is a `<Link href={/ticker/${peerTicker}}>` that navigates to the peer's ticker page.

### 3.2 Lazy Loading

The panel data is fetched **only when the user expands it** (not on page load):

```typescript
function PeerComparisonPanel({ ticker, isPremium }: PeerComparisonPanelProps) {
  const [expanded, setExpanded] = useState(false);
  const [data, setData] = useState<PeersResponse | null>(null);
  const [loading, setLoading] = useState(false);

  const handleExpand = async () => {
    setExpanded(!expanded);
    if (!expanded && !data) {
      // First expand — fetch data
      setLoading(true);
      try {
        const response = await fetch(
          `${API_BASE_URL}${stocks.peers(ticker)}`,
          { cache: 'no-store' }
        );
        const result = await response.json();
        setData(result);
      } finally {
        setLoading(false);
      }
    }
  };
  // ...
}
```

### 3.3 "How Peers Are Selected" Tooltip

**Content:**
```
Peers are selected from the same sector using a 5-factor similarity algorithm:
• Market Cap (30% weight) — Similar company size
• Revenue Growth (20%) — Similar growth trajectory
• Net Margin (20%) — Similar profitability
• P/E Ratio (15%) — Similar valuation
• Beta (15%) — Similar risk profile

Candidates are filtered to companies between 0.25x and 4x the stock's market cap.
```

Rendered via the existing `Tooltip` component from `components/ui/Tooltip.tsx`.

### 3.4 Responsive Behavior

| Breakpoint | Layout |
|---|---|
| Desktop (≥1024px) | Full table with all 6 peers visible |
| Tablet (768-1023px) | Table with horizontal scroll (stock pinned left) |
| Mobile (<768px) | Card-based layout: each peer as a horizontal scroll card showing key metrics |

**Mobile card layout:**

```
┌─────────────────────────────────────────┐
│  [MSFT ▸]  [GOOGL ▸]  [META ▸]  ...   │  ← horizontal scroll
└─────────────────────────────────────────┘

Each card:
┌─────────────┐
│ MSFT        │
│ IC: 82.1    │
│ P/E: 33.2   │
│ ROE: 38.5%  │
│ Grw: 12.3%  │
│ Sim: 87%    │
└─────────────┘
```

---

## 4. Component Design: Fair Value

### 4.1 `FairValueGauge`

**File:** `components/ticker/FairValueGauge.tsx`

**Props:**

```typescript
interface FairValueGaugeProps {
  ticker: string;
  isPremium: boolean;
}
```

**Desktop Layout:**

```
┌──────────────────────────────────────────────────┐
│  Fair Value Estimates                             │
│                                                   │
│  ┌─────────────────────────────────────────────┐  │
│  │ Undervalued    Fairly Valued    Overvalued  │  │
│  │ ████████████████|███●███████████████████████│  │
│  │ $142      $168  $179  $195          $210    │  │
│  │ Graham    EPV   Price  DCF      Analyst Tgt │  │
│  └─────────────────────────────────────────────┘  │
│                                                   │
│  Model Estimates:                                 │
│  ┌────────────────┬─────────┬──────────────────┐  │
│  │ Model          │ Fair Val│ vs. Price         │  │
│  ├────────────────┼─────────┼──────────────────┤  │
│  │ DCF            │ $195.30 │ ▲ +9.4% upside   │  │
│  │ Graham Number  │ $142.80 │ ▼ -20.0%         │  │
│  │ Earnings Power │ $168.40 │ ▼ -5.7%          │  │
│  │ Analyst Target │ $210.50 │ ▲ +17.9%         │  │
│  └────────────────┴─────────┴──────────────────┘  │
│                                                   │
│  Margin of Safety: Fairly Valued                  │
│  Average fair value: $168.83 (price is 5.7% above)│
│                                                   │
│  ⓘ Fair value models are estimates, not guarantees│
│  Confidence: Medium                               │
└──────────────────────────────────────────────────┘
```

### 4.2 Gauge Visualization

The horizontal gauge shows a spectrum from undervalued to overvalued:

```typescript
// Gauge zones:
// Green zone:  price < 0.85 * avg_fair_value  (>15% below → undervalued)
// Yellow zone: 0.85 * avg < price < 1.15 * avg (within 15% → fairly valued)
// Red zone:    price > 1.15 * avg_fair_value   (>15% above → overvalued)
```

**Implementation:** CSS-based horizontal bar with positioned markers:

```tsx
function FairValueGaugeBar({ currentPrice, models, analystTarget }: GaugeBarProps) {
  // Calculate price range for the gauge (min to max of all estimates + price + 20% padding)
  const allValues = [
    currentPrice,
    models.dcf?.fair_value,
    models.graham_number?.fair_value,
    models.epv?.fair_value,
    analystTarget?.target_price,
  ].filter(Boolean);

  const min = Math.min(...allValues) * 0.85;
  const max = Math.max(...allValues) * 1.15;

  const pricePosition = ((currentPrice - min) / (max - min)) * 100;

  return (
    <div className="relative h-8 rounded-full overflow-hidden">
      {/* Background gradient: green → yellow → red */}
      <div className="absolute inset-0 bg-gradient-to-r from-green-500/30 via-yellow-500/30 to-red-500/30" />

      {/* Model markers */}
      {Object.entries(models).map(([key, model]) => (
        <FairValueMarker
          key={key}
          position={((model.fair_value - min) / (max - min)) * 100}
          label={MODEL_LABELS[key]}
          value={model.fair_value}
        />
      ))}

      {/* Current price marker (prominent) */}
      <div
        className="absolute top-0 bottom-0 w-0.5 bg-white"
        style={{ left: `${pricePosition}%` }}
      >
        <div className="absolute -top-6 left-1/2 -translate-x-1/2 text-xs font-medium text-white">
          ${formatPrice(currentPrice)}
        </div>
      </div>
    </div>
  );
}
```

### 4.3 Suppression Logic

For pre-revenue, pre-profit, or very early-stage companies, fair value models are unreliable. The API returns `meta.suppressed = true` with a reason. In this case:

```tsx
{data.meta.suppressed && (
  <div className="bg-yellow-500/10 border border-yellow-500/20 rounded-lg p-3 text-sm text-yellow-300">
    <ExclamationTriangleIcon className="h-4 w-4 inline mr-1" />
    Fair value models are not available: {data.meta.suppression_reason}
  </div>
)}
```

### 4.4 Confidence Indicator

```typescript
type Confidence = 'high' | 'medium' | 'low';

// Displayed as colored text badge:
// high   → "High Confidence" green
// medium → "Medium Confidence" yellow
// low    → "Low Confidence — interpret with caution" orange
```

Confidence is determined by:
- Number of models with valid outputs (3 = high, 2 = medium, 1 = low)
- Analyst coverage breadth (>20 analysts = high, 5-20 = medium, <5 = low)
- Spread between model estimates (tight = high confidence, wide = low)

---

## 5. Data Fetching

### 5.1 `usePeerComparison` Hook

**File:** `lib/hooks/usePeerComparison.ts`

```typescript
function usePeerComparison(ticker: string, enabled: boolean): {
  data: PeersResponse | null;
  loading: boolean;
  error: string | null;
  fetch: () => Promise<void>;
}
```

Only fetches when `enabled` is true (lazy — triggered on panel expand).

### 5.2 `useFairValue` Hook

**File:** `lib/hooks/useFairValue.ts`

```typescript
function useFairValue(ticker: string): {
  data: FairValueResponse | null;
  loading: boolean;
  error: string | null;
}
```

Fetches on mount (fair value gauge is not lazy-loaded).

---

## 6. Freemium Gating

### Peer Comparison

| Feature | Free | Premium |
|---|---|---|
| Collapsed teaser (peer names + avg IC Score) | Yes | Yes |
| Expanded table | Blurred | Yes |
| Peer ticker links | No | Yes |
| Similarity scores | No | Yes |

**Free tier expanded view:** Show the table structure but blur all cell values. Overlay with upgrade CTA:

```tsx
{!isPremium && expanded && (
  <div className="relative">
    <div className="blur-sm pointer-events-none">
      <PeerComparisonTable data={data} />
    </div>
    <div className="absolute inset-0 flex items-center justify-center bg-ic-bg-primary/60">
      <div className="text-center">
        <LockClosedIcon className="h-8 w-8 mx-auto mb-2 text-ic-text-dim" />
        <p className="text-sm text-ic-text-secondary">Upgrade to compare {ticker} with peers</p>
        <UpgradeButton className="mt-2" />
      </div>
    </div>
  </div>
)}
```

### Fair Value

| Feature | Free | Premium |
|---|---|---|
| Gauge visualization | Blurred | Yes |
| Model estimates table | Blurred | Yes |
| Margin of safety assessment | No | Yes |
| Confidence indicator | No | Yes |

---

## 7. New Files

| File | Purpose | Est. LOC |
|---|---|---|
| `components/ticker/PeerComparisonPanel.tsx` | Expandable peer comparison table | ~300 |
| `components/ticker/FairValueGauge.tsx` | Fair value visualization with gauge | ~280 |
| `lib/hooks/usePeerComparison.ts` | Lazy-loaded peer data hook | ~50 |
| `lib/hooks/useFairValue.ts` | Fair value data hook | ~50 |

## 8. Modified Files

| File | Change |
|---|---|
| `components/ticker/TickerFundamentals.tsx` | Add FairValueGauge below health card |
| `components/ticker/TickerTabs.tsx` | Add PeerComparisonPanel as expandable section |
| `lib/types/fundamentals.ts` | Add `PeersResponse`, `FairValueResponse` types |
| `lib/api/routes.ts` | Add `peers`, `fairValue` routes (if not done in P1) |

## 9. Performance

| Metric | Target |
|---|---|
| Peer panel: data fetch on expand | <300ms |
| Peer panel: render time (6 columns × 7 rows) | <50ms |
| Fair value gauge: initial render | <200ms |
| No LCP impact (both are below fold or lazy) | 0ms added to LCP |

## 10. Acceptance Criteria

- [ ] Peer panel collapsed shows peer ticker names and average IC Score
- [ ] Peer panel expanded shows side-by-side comparison table for 7 metrics
- [ ] Best/worst values in each row are color-coded green/red
- [ ] Stock's column is visually distinguished (blue background tint)
- [ ] Peer tickers are clickable links to their ticker pages
- [ ] "How peers are selected" tooltip explains the 5-factor algorithm
- [ ] Peer panel data fetched only on expand (lazy loading)
- [ ] Fair value gauge shows DCF, Graham, EPV markers on a horizontal scale
- [ ] Current price marker is prominent on the gauge
- [ ] Analyst consensus target shown as additional marker
- [ ] Zone coloring: green (undervalued) → yellow (fair) → red (overvalued)
- [ ] Suppression message shown for pre-revenue/pre-profit companies
- [ ] Free tier: collapsed teaser visible, expanded content blurred with upgrade CTA
- [ ] Mobile: peer panel uses horizontal card scroll; gauge uses simplified view
- [ ] All text accessible to screen readers
