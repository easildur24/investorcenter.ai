# Tech Spec: Project 3 — Fundamental Health Card + Red Flags

**Parent PRD:** `docs/prd-enhanced-fundamentals-experience.md`
**Priority:** P0 — Hero feature of the fundamentals enhancement
**Estimated Effort:** 3 sprints
**Dependencies:** Project 1 (health-summary API endpoint)

---

## 1. Overview

Build the `FundamentalHealthCard` — a prominent summary card rendered at the top of the ticker fundamentals area. It synthesizes Piotroski F-Score, Altman Z-Score, IC Score financial health, and lifecycle classification into a single glanceable assessment. Includes auto-generated strengths, concerns, and red flag alerts with expandable explanations.

This is the single highest-impact UI deliverable: it transforms "50 raw numbers" into an actionable narrative a user can consume in <10 seconds.

## 2. Architecture Context

### Data Source

**API:** `GET /api/v1/stocks/:ticker/health-summary` (Project 1)

**Response shape:**

```typescript
interface HealthSummaryResponse {
  ticker: string;
  health: {
    badge: 'Strong' | 'Healthy' | 'Fair' | 'Weak' | 'Distressed';
    score: number; // 0-100 composite
    components: {
      piotroski_f_score: { value: number; max: 9; interpretation: string };
      altman_z_score: { value: number; zone: 'safe' | 'grey' | 'distress'; interpretation: string };
      ic_financial_health: { value: number; max: 100 };
      debt_percentile: { value: number; interpretation: string };
    };
  };
  lifecycle: {
    stage: 'hypergrowth' | 'growth' | 'mature' | 'value' | 'turnaround';
    description: string;
    classified_at: string;
  };
  strengths: Array<{
    metric: string;
    value: number;
    percentile: number;
    message: string;
  }>;
  concerns: Array<{
    metric: string;
    value: number;
    percentile: number;
    message: string;
  }>;
  red_flags: Array<{
    id: string;
    severity: 'high' | 'medium' | 'low';
    title: string;
    description: string;
    related_metrics: string[];
  }>;
}
```

### Placement

The card sits at the **top of the ticker fundamentals sidebar** (`TickerFundamentals.tsx`), above the existing key-value metric list. On the main ticker page, it's the first thing a user sees in the fundamentals area.

```
┌─────────────────────────────┐
│  Fundamental Health Card    │  ← NEW (this project)
│  [Strong] [Mature]          │
│  Strengths: ...             │
│  Concerns: ...              │
│  Red Flags: ⚠ ...           │
├─────────────────────────────┤
│  P/E Ratio     28.5  [═══] │  ← Existing metrics + percentile bars (Project 2)
│  ROE          147.2  [═══] │
│  ...                        │
└─────────────────────────────┘
```

---

## 3. Component Design

### 3.1 `FundamentalHealthCard` — Main Card

**File:** `components/ticker/FundamentalHealthCard.tsx`

**Props:**

```typescript
interface FundamentalHealthCardProps {
  ticker: string;
  /** Pre-fetched data (if available from parent) */
  data?: HealthSummaryResponse;
  /** Compact mode for smaller containers */
  variant?: 'full' | 'compact';
  /** Premium user flag (affects gating) */
  isPremium?: boolean;
}
```

**Layout (Desktop, `variant='full'`):**

```
┌─────────────────────────────────────────────────┐
│ Fundamental Health                               │
│                                                  │
│ ┌──────────┐  ┌────────────────┐                │
│ │ STRONG   │  │ Mature Company │                │
│ │ ██████░░ │  │ Stable ops     │                │
│ │ 82/100   │  └────────────────┘                │
│ └──────────┘                                    │
│                                                  │
│ Components:                                      │
│   F-Score: 7/9 ████████░      Strong            │
│   Z-Score: 5.8  ████████████  Safe Zone         │
│   IC Health: 85 ████████░░    Above Average     │
│                                                  │
│ ✓ Strengths                                      │
│   • Net margin ranks in top 10% of Tech sector   │
│   • ROE is exceptional vs. sector peers          │
│   • Revenue growth above sector median           │
│                                                  │
│ ⚠ Concerns                                       │
│   • Debt/Equity above sector median              │
│                                                  │
│ 🔴 Red Flags (1)                                 │
│ ┌─────────────────────────────────────────────┐  │
│ │ ⚠ Above-average leverage              [▼]  │  │
│ │   D/E of 1.87 exceeds 78% of Tech peers.   │  │
│ │   Interest coverage is adequate at 3.2x.    │  │
│ └─────────────────────────────────────────────┘  │
│                                                  │
│ Data as of Feb 27, 2026                          │
└─────────────────────────────────────────────────┘
```

**Layout (Mobile / `variant='compact'`):**

```
┌────────────────────────────────┐
│ Health: STRONG  │  Mature   [▼]│
│ F:7/9  Z:Safe  IC:85          │
│ ✓3 strengths  ⚠1 concern  🔴1 │
│ [Expand for details]          │
└────────────────────────────────┘
```

### 3.2 `HealthBadge` — Health Status Display

**File:** `components/ui/HealthBadge.tsx`

```typescript
interface HealthBadgeProps {
  badge: 'Strong' | 'Healthy' | 'Fair' | 'Weak' | 'Distressed';
  score: number;
  size?: 'sm' | 'md' | 'lg';
}
```

**Color mapping:**

| Badge | Background | Text | Icon |
|---|---|---|---|
| Strong | `bg-green-500/15` | `text-green-400` | Shield check |
| Healthy | `bg-green-400/15` | `text-green-300` | Check circle |
| Fair | `bg-yellow-500/15` | `text-yellow-400` | Minus circle |
| Weak | `bg-orange-500/15` | `text-orange-400` | Alert triangle |
| Distressed | `bg-red-500/15` | `text-red-400` | X circle |

**Score visualization:** Small horizontal progress bar inside the badge showing 0-100 score.

### 3.3 `LifecycleBadge` — Lifecycle Stage Indicator

**File:** `components/ui/LifecycleBadge.tsx`

```typescript
interface LifecycleBadgeProps {
  stage: 'hypergrowth' | 'growth' | 'mature' | 'value' | 'turnaround';
  description: string;
  size?: 'sm' | 'md';
}
```

**Color mapping (from PRD):**

| Stage | Color | Icon |
|---|---|---|
| Hypergrowth | `bg-purple-500/15 text-purple-400` | Rocket |
| Growth | `bg-blue-500/15 text-blue-400` | Trending up |
| Mature | `bg-slate-500/15 text-slate-300` | Building |
| Value | `bg-amber-500/15 text-amber-400` | Tag / Dollar |
| Turnaround | `bg-orange-500/15 text-orange-400` | Refresh |

**Hover tooltip:** Shows the full lifecycle description from the API response.

### 3.4 `RedFlagBadge` — Warning Indicator

**File:** `components/ui/RedFlagBadge.tsx`

```typescript
interface RedFlagBadgeProps {
  id: string;
  severity: 'high' | 'medium' | 'low';
  title: string;
  description: string;
  relatedMetrics: string[];
  /** Whether the full description is visible */
  defaultExpanded?: boolean;
}
```

**Behavior:**
- Collapsed by default: shows severity icon + title + expand chevron
- Click to expand: reveals full description paragraph
- Severity colors:
  - `high` → `border-red-500/30 bg-red-500/5`
  - `medium` → `border-orange-500/30 bg-orange-500/5`
  - `low` → `border-yellow-500/30 bg-yellow-500/5`
- `relatedMetrics` rendered as clickable badges that scroll to the metric in the sidebar

**Animation:** Expand/collapse with `transition-all duration-200 ease-in-out` using height animation via `overflow-hidden` + `max-height`.

### 3.5 `HealthComponentBar` — Score Component Visualization

**File:** Inline within `FundamentalHealthCard.tsx`

Small horizontal bars showing each health component:

```typescript
interface HealthComponentBarProps {
  label: string;          // "F-Score", "Z-Score", "IC Health"
  value: number;
  max: number;
  interpretation: string; // "Strong", "Safe Zone", "Above Average"
  colorZone: 'good' | 'neutral' | 'bad';
}
```

Rendered as:
```
F-Score: 7/9 ████████░      Strong
```

---

## 4. Data Fetching

### 4.1 `useHealthSummary` Hook

**File:** `lib/hooks/useHealthSummary.ts`

```typescript
interface UseHealthSummaryResult {
  data: HealthSummaryResponse | null;
  loading: boolean;
  error: string | null;
}

function useHealthSummary(ticker: string): UseHealthSummaryResult {
  const [data, setData] = useState<HealthSummaryResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        const response = await fetch(
          `${API_BASE_URL}${stocks.healthSummary(ticker.toUpperCase())}`,
          { cache: 'no-store' }
        );
        if (!response.ok) {
          if (response.status === 404) { setData(null); return; }
          throw new Error(`HTTP ${response.status}`);
        }
        const result = await response.json();
        setData(result);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch health summary');
      } finally {
        setLoading(false);
      }
    };

    const timer = setTimeout(fetchData, 50); // Slight delay to not block initial paint
    return () => clearTimeout(timer);
  }, [ticker]);

  return { data, loading, error };
}
```

### 4.2 Integration in `TickerFundamentals`

```tsx
// In components/ticker/TickerFundamentals.tsx

export default function TickerFundamentals({ symbol }: TickerFundamentalsProps) {
  const { data: healthData, loading: healthLoading } = useHealthSummary(symbol);

  return (
    <div className="space-y-4">
      {/* Health Card — rendered first, above existing metrics */}
      <FundamentalHealthCard
        ticker={symbol}
        data={healthData}
        variant="full"
        isPremium={isPremiumUser}
      />

      {/* Existing metric key-value pairs */}
      <div className="space-y-2">
        {/* ... existing metrics with percentile bars ... */}
      </div>
    </div>
  );
}
```

---

## 5. Freemium Gating

### Free Tier

- Health badge (Strong/Healthy/Fair/Weak/Distressed) — visible
- Lifecycle badge — visible
- Up to 2 strengths, 2 concerns — visible
- High-severity red flags only — visible
- Health component bars (F-Score, Z-Score, IC Health) — visible

### Premium Tier

- All strengths and concerns (unlimited)
- All red flag severities (medium, low)
- Red flag descriptions expanded by default
- Health score numeric value (82/100) — visible

### Gating Implementation

```tsx
const MAX_FREE_STRENGTHS = 2;
const MAX_FREE_CONCERNS = 2;

// In FundamentalHealthCard:
const visibleStrengths = isPremium
  ? data.strengths
  : data.strengths.slice(0, MAX_FREE_STRENGTHS);

const visibleConcerns = isPremium
  ? data.concerns
  : data.concerns.slice(0, MAX_FREE_CONCERNS);

const visibleRedFlags = isPremium
  ? data.red_flags
  : data.red_flags.filter(rf => rf.severity === 'high');

// Show "Upgrade to see N more strengths" teaser
{!isPremium && data.strengths.length > MAX_FREE_STRENGTHS && (
  <div className="text-xs text-ic-text-dim flex items-center gap-1">
    <LockClosedIcon className="h-3 w-3" />
    <span>{data.strengths.length - MAX_FREE_STRENGTHS} more strengths</span>
    <UpgradeLink />
  </div>
)}
```

---

## 6. Red Flag Integration with Metrics

Red flags should also appear as inline badges next to related metrics in the sidebar and MetricsTab. When the health summary API returns `red_flags[].related_metrics`, those metrics should display a small warning indicator.

### `MetricRedFlagIndicator`

```typescript
interface MetricRedFlagIndicatorProps {
  metricKey: string;
  redFlags: RedFlag[];
}
```

This small component checks if any red flag's `related_metrics` array includes the current metric key. If so, it renders a small colored dot next to the metric value:

```tsx
function MetricRedFlagIndicator({ metricKey, redFlags }: MetricRedFlagIndicatorProps) {
  const relevantFlags = redFlags.filter(rf => rf.related_metrics.includes(metricKey));
  if (relevantFlags.length === 0) return null;

  const highestSeverity = relevantFlags.reduce((max, rf) =>
    SEVERITY_ORDER[rf.severity] > SEVERITY_ORDER[max] ? rf.severity : max,
    'low' as RedFlag['severity']
  );

  return (
    <Tooltip content={relevantFlags[0].title}>
      <span className={cn(
        'inline-block w-2 h-2 rounded-full ml-1',
        highestSeverity === 'high' && 'bg-red-500',
        highestSeverity === 'medium' && 'bg-orange-500',
        highestSeverity === 'low' && 'bg-yellow-500',
      )} />
    </Tooltip>
  );
}
```

---

## 7. Responsive Behavior

| Breakpoint | Behavior |
|---|---|
| Desktop (≥1024px) | Full card with all sections visible, component bars, expanded red flags |
| Tablet (768-1023px) | Card collapses: badge + lifecycle visible, strengths/concerns/flags in accordion |
| Mobile (<768px) | Single-line summary: "Strong · Mature · 3✓ 1⚠ 1🔴" with tap-to-expand |

**Mobile accordion implementation:**

```tsx
const [expanded, setExpanded] = useState(false);

// Mobile: collapsed view
<div className="md:hidden">
  <button onClick={() => setExpanded(!expanded)} className="w-full">
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-2">
        <HealthBadge badge={data.health.badge} score={data.health.score} size="sm" />
        <LifecycleBadge stage={data.lifecycle.stage} size="sm" />
      </div>
      <div className="flex items-center gap-2 text-xs text-ic-text-dim">
        <span>✓{data.strengths.length}</span>
        <span>⚠{data.concerns.length}</span>
        {data.red_flags.length > 0 && (
          <span className="text-red-400">🔴{data.red_flags.length}</span>
        )}
        <ChevronDownIcon className={cn('h-4 w-4 transition-transform', expanded && 'rotate-180')} />
      </div>
    </div>
  </button>
  {expanded && (
    <div className="mt-3 space-y-3">
      {/* Full card content */}
    </div>
  )}
</div>
```

## 8. Accessibility

- **`HealthBadge`:** `role="status"` with `aria-label="Fundamental health: Strong, score 82 out of 100"`
- **`LifecycleBadge`:** `aria-label="Company lifecycle: Mature company with stable operations"`
- **`RedFlagBadge`:** `role="alert"` with `aria-live="polite"` for dynamic content; expand/collapse uses `aria-expanded`
- **Strengths/Concerns:** Rendered as `<ul>` with proper list semantics
- **Color + text:** All severity indicators use text labels in addition to color (not color-only)
- **Focus management:** When expanding a red flag, focus moves to the description content

## 9. New Files

| File | Purpose | Est. LOC |
|---|---|---|
| `components/ticker/FundamentalHealthCard.tsx` | Main health card component | ~350 |
| `components/ui/HealthBadge.tsx` | Health status badge | ~60 |
| `components/ui/LifecycleBadge.tsx` | Lifecycle stage badge | ~60 |
| `components/ui/RedFlagBadge.tsx` | Expandable red flag warning | ~100 |
| `lib/hooks/useHealthSummary.ts` | Health summary data hook | ~50 |

## 10. Modified Files

| File | Change |
|---|---|
| `components/ticker/TickerFundamentals.tsx` | Add `FundamentalHealthCard` at top; pass red flags for inline indicators |
| `components/ticker/tabs/MetricsTab.tsx` | Add `MetricRedFlagIndicator` next to flagged metrics |
| `lib/types/fundamentals.ts` | Add `HealthSummaryResponse` types (if not in Project 2) |

## 11. Performance

- **Health Card rendering:** Must complete within 200ms of data availability (server response + React render)
- **API call:** Not blocking initial page paint. Card shows skeleton loader while data fetches.
- **Skeleton loader:** Matches card dimensions exactly — badge skeleton (rounded rectangle), 3 lines for strengths, 1 for concerns

```tsx
function HealthCardSkeleton() {
  return (
    <div className="bg-ic-surface rounded-lg border border-ic-border p-4 space-y-3 animate-pulse">
      <div className="flex items-center gap-3">
        <div className="h-10 w-24 bg-ic-border rounded-lg" />
        <div className="h-6 w-32 bg-ic-border rounded-full" />
      </div>
      <div className="space-y-2">
        <div className="h-4 w-full bg-ic-border rounded" />
        <div className="h-4 w-3/4 bg-ic-border rounded" />
        <div className="h-4 w-5/6 bg-ic-border rounded" />
      </div>
    </div>
  );
}
```

## 12. Acceptance Criteria

- [ ] Health card renders on all stock ticker pages with skeleton while loading
- [ ] Health badge accurately reflects F-Score + Z-Score + IC Score health composite
- [ ] Lifecycle badge shows correct stage with matching color and icon
- [ ] Auto-generated strengths reference specific sector percentile rankings
- [ ] Auto-generated concerns identify below-average metrics
- [ ] Red flag rules trigger correctly (test with known edge cases: high-yield REIT, pre-profit growth stock)
- [ ] Red flags expandable with descriptive explanations
- [ ] Inline red flag indicators appear next to related metrics in sidebar
- [ ] Free tier: badge + lifecycle + 2 strengths + 2 concerns + high-severity flags only
- [ ] Premium tier: all strengths/concerns + all flag severities
- [ ] Mobile: collapsed view with tap-to-expand
- [ ] Accessible: screen readers can consume all health information
- [ ] No layout shift when health data loads (skeleton placeholder)
