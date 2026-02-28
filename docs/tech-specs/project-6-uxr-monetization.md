# Tech Spec: Project 6 — UXR Program + Monetization / Paywall Implementation

**Parent PRD:** `docs/prd-enhanced-fundamentals-experience.md`
**Priority:** P1 (UXR runs in parallel), P2 (paywall implementation follows Projects 2-5)
**Estimated Effort:** 8 weeks (research) + 1-2 sprints (paywall engineering)
**Dependencies:** None for UXR; Projects 2-5 for paywall implementation

---

## 1. Overview

This project has two tracks:

**Track A — UXR Program (8 weeks):** Five research studies that validate design decisions for Projects 2-5. Runs in parallel with engineering work. Delivers findings that inform metric grouping, feature prioritization, and freemium boundary tuning.

**Track B — Paywall Engineering (1-2 sprints):** Implement the freemium gating infrastructure shared across all fundamentals features. Build reusable paywall components, event tracking, and A/B testing hooks.

---

## 2. Track A: UXR Program

### 2.1 Study Sequence & Timeline

| Week | Study | Method | Sample | Key Decision |
|---|---|---|---|---|
| 1-2 | Competitive Benchmarking Audit | Heuristic evaluation | 6 platforms | Visual approach for percentile bars, health cards |
| 2-3 | Current State Usability Testing | Remote moderated (60 min) | 8-10 users | Baseline metrics; identify pain points |
| 3-4 | Card Sort: Metric Organization | Hybrid card sort | 15-20 users | Validate metric groupings for MetricsTab |
| 4-5 | Survey: Metric Importance + WTP | Online survey (15 min) | 200+ users | Feature priority; free/premium boundary |
| 5-7 | Prototype Testing (A/B/C) | Unmoderated + moderated | 20 + 6 users | Final design direction for Phase 2+ |
| 8 | Synthesis & Recommendations | Internal | N/A | Final UXR report feeds design |

### 2.2 Study 1: Competitive Benchmarking Audit

**Platforms:** Yahoo Finance, Stock Analysis, Simply Wall St, Koyfin, Finviz, Morningstar

**Deliverables:**
- Feature comparison matrix (screenshot-annotated)
- Interaction pattern catalog (how each platform presents context on metrics)
- Gap analysis: what InvestorCenter is missing vs. competitors
- Design pattern recommendations for Projects 2-5

**Key Questions:**
- How do competitors color-code metrics (absolute thresholds vs. peer-relative)?
- What is the information hierarchy on their fundamentals pages?
- How do they handle data freshness and source attribution?
- What synthesis features exist (health scores, snowflakes, narrative summaries)?

### 2.3 Study 2: Current State Usability Testing

**Recruitment:** Existing InvestorCenter users with ≥5 ticker page views in last 30 days
**Tool:** Zoom + screen recording

**Task Scenarios:**
1. "Assess whether AAPL is a fundamentally strong company."
2. "Find out if AAPL is expensive or cheap compared to similar companies."
3. "Identify any financial concerns about MSFT."
4. "Compare GOOGL's profitability to its peers."

**Metrics:**
- Task completion rate
- Time on task
- Number of tabs/pages navigated
- SUS (System Usability Scale) score
- Post-task confidence rating (1-5)

**Deliverables:**
- Baseline metrics for comparison after Projects 2-5 ship
- Top 5 pain points ranked by frequency and severity
- Video clips of key moments (confusion, abandonment, delight)

### 2.4 Study 3: Card Sort — Metric Organization

**Tool:** Optimal Workshop (or similar)
**Cards:** 40 metric labels from the existing MetricsTab categories
**Pre-defined categories:** Valuation, Profitability, Growth, Financial Health, Risk, Quality, Dividends, Analyst Opinion

**Participants can:** Rename categories, create new ones, mark cards as "don't understand" or "not important"

**Deliverables:**
- Dendrogram showing user mental model clusters
- Agreement matrix for category assignments
- List of metrics users don't understand (need tooltips)
- Recommended category structure for MetricsTab redesign

### 2.5 Study 4: Survey — Metric Importance & WTP

**Distribution:** In-app banner + email to registered users

**Survey Sections:**

1. **Investor Profile:**
   - Experience level (beginner / intermediate / advanced)
   - Primary strategy (income / growth / value / momentum / diversified)
   - Portfolio size ($0-10K / $10K-100K / $100K-1M / $1M+)

2. **Metric Importance:** Rate 20 key metrics on a 5-point scale (Not important → Essential)

3. **Feature Desirability:** Rank features by desirability:
   - Sector percentile benchmarking
   - Trend sparklines
   - Peer comparison table
   - Fair value estimates
   - Red flag alerts
   - Fundamental health badge
   - Narrative summary

4. **Willingness to Pay:** "Which of these features would make you consider upgrading to Premium?" (select all that apply)

5. **Open Text:** "What's the hardest part about evaluating a stock's fundamentals?"

**Deliverables:**
- Metric importance ranking → informs "at-a-glance" vs. "detail view" placement
- Feature desirability ranking → informs development priority
- WTP analysis → informs free/premium boundary
- Qualitative themes from open text responses

### 2.6 Study 5: Prototype Testing

**Figma Prototypes (3 variants):**

**Variant A: Incremental Enhancement**
- Current layout preserved
- Percentile bars added inline to each metric
- Health badge at top of sidebar
- Minimal visual disruption

**Variant B: Dashboard Redesign**
- New "Fundamental Health Dashboard" at top
- Grouped cards with sparklines and peer comparison
- Red flag alerts prominently displayed
- More visual, less raw data

**Variant C: Narrative Approach**
- Simply Wall St-inspired visual summary
- Prose-style narrative summary at top
- Interactive "explore deeper" drill-down
- Most radical departure from current UX

**Metrics:**
- Preference ranking (A vs. B vs. C)
- Task completion rate (same scenarios as Study 2)
- Time on task comparison vs. Study 2 baseline
- Confidence rating comparison
- Qualitative feedback on most/least useful elements

**Deliverables:**
- Winning design direction with confidence level
- Specific UI elements to keep/discard from each variant
- Design recommendations for Projects 3-5 implementation

---

## 3. Track B: Paywall Engineering

### 3.1 Architecture Context

#### Existing Subscription Infrastructure

| Component | Location | Status |
|---|---|---|
| Subscription plans | `backend/services/subscription_service.go` | Implemented |
| User subscription status | `GET /api/v1/subscriptions/me` | Working endpoint |
| Subscription limits | `GET /api/v1/subscriptions/limits` | Working endpoint |
| Feature flags | `backend/services/feature_flags.go` | Implemented |
| Auth context (user, isPremium) | `lib/auth/AuthContext.tsx` → `useAuth()` | Working |

#### Current Auth Pattern

```typescript
// In lib/auth/AuthContext.tsx
const { user, isAuthenticated, isAdmin } = useAuth();

// Premium check pattern:
const isPremium = user?.subscription?.plan === 'premium' || user?.subscription?.plan === 'pro';
```

### 3.2 Fundamentals Feature Flag Configuration

**File:** `backend/services/feature_flags.go`

Add fundamentals-specific feature flags:

```go
var FundamentalsFeatureFlags = map[string]FeatureFlag{
    "fundamentals.percentile_bars.all": {
        Name:        "All sector percentile bars",
        FreeTier:    false,
        PremiumTier: true,
        Description: "Sector percentile bars on all 14+ metrics",
    },
    "fundamentals.percentile_bars.core": {
        Name:        "Core sector percentile bars (6 metrics)",
        FreeTier:    true,
        PremiumTier: true,
        Description: "Sector percentile bars on 6 core metrics",
    },
    "fundamentals.health_card.full": {
        Name:        "Full health card (all strengths/concerns)",
        FreeTier:    false,
        PremiumTier: true,
    },
    "fundamentals.health_card.basic": {
        Name:        "Basic health card (badge + 2 strengths/concerns)",
        FreeTier:    true,
        PremiumTier: true,
    },
    "fundamentals.red_flags.all": {
        Name:        "All red flag severities",
        FreeTier:    false,
        PremiumTier: true,
    },
    "fundamentals.red_flags.high": {
        Name:        "High-severity red flags only",
        FreeTier:    true,
        PremiumTier: true,
    },
    "fundamentals.peer_comparison": {
        Name:        "Peer comparison panel",
        FreeTier:    false,
        PremiumTier: true,
    },
    "fundamentals.sparklines": {
        Name:        "Trend sparklines",
        FreeTier:    false,
        PremiumTier: true,
    },
    "fundamentals.fair_value": {
        Name:        "Fair value gauge",
        FreeTier:    false,
        PremiumTier: true,
    },
    "fundamentals.metric_history": {
        Name:        "Full metric history charts",
        FreeTier:    false,
        PremiumTier: true,
    },
}
```

### 3.3 `FundamentalsPaywall` — Reusable Gating Component

**File:** `components/ui/FundamentalsPaywall.tsx`

A reusable wrapper that handles blurring, lock overlay, and upgrade CTA for any gated fundamentals content.

```typescript
interface FundamentalsPaywallProps {
  /** Feature flag key */
  feature: string;
  /** What to show when gated (blurred content) */
  children: React.ReactNode;
  /** CTA text */
  ctaText?: string;
  /** Ticker for tracking context */
  ticker?: string;
  /** Display variant */
  variant?: 'blur' | 'lock' | 'teaser';
}
```

**Variants:**

1. **`blur`** — Content renders but is blurred with overlay lock icon + CTA. Used for peer comparison table, sparklines, fair value gauge.

2. **`lock`** — Content replaced with a lock icon and "Upgrade to unlock" message. Used for individual premium percentile bars.

3. **`teaser`** — Partial content visible (e.g., 2 of 5 strengths) with "Upgrade to see N more" teaser. Used for health card strengths/concerns.

```tsx
function FundamentalsPaywall({ feature, children, ctaText, ticker, variant = 'blur' }: FundamentalsPaywallProps) {
  const { user } = useAuth();
  const hasAccess = checkFeatureAccess(user, feature);

  if (hasAccess) return <>{children}</>;

  // Track paywall impression
  useEffect(() => {
    trackEvent('fundamentals_paywall_impression', { feature, ticker, variant });
  }, [feature, ticker]);

  if (variant === 'blur') {
    return (
      <div className="relative">
        <div className="blur-sm pointer-events-none select-none" aria-hidden="true">
          {children}
        </div>
        <PaywallOverlay ctaText={ctaText} feature={feature} ticker={ticker} />
      </div>
    );
  }

  if (variant === 'lock') {
    return (
      <div className="flex items-center justify-center py-2">
        <LockClosedIcon className="h-3.5 w-3.5 text-ic-text-dim mr-1" />
        <span className="text-xs text-ic-text-dim">Premium</span>
      </div>
    );
  }

  // variant === 'teaser'
  return <>{children}</>;
}
```

### 3.4 `PaywallOverlay` — Upgrade CTA Component

**File:** `components/ui/PaywallOverlay.tsx`

```typescript
interface PaywallOverlayProps {
  ctaText?: string;
  feature: string;
  ticker?: string;
}
```

**Layout:**

```
┌─────────────────────────────────────┐
│          🔒                          │
│   Upgrade to Premium                │
│   to compare AAPL with peers        │
│                                      │
│   [Upgrade Now]  [Learn More]       │
└─────────────────────────────────────┘
```

**CTA click behavior:**
1. Track click event: `fundamentals_paywall_cta_click`
2. Navigate to `/subscriptions` or open upgrade modal
3. Include `?source=fundamentals&feature=${feature}&ticker=${ticker}` query params for attribution

### 3.5 Event Tracking

Track fundamentals engagement and paywall interactions for analytics:

```typescript
// Event taxonomy for fundamentals features:

interface FundamentalsEvent {
  category: 'fundamentals';
  action: string;
  ticker: string;
  feature?: string;
  metadata?: Record<string, any>;
}

// Events to track:
const FUNDAMENTALS_EVENTS = {
  // Engagement
  'health_card_viewed': {},         // Health card entered viewport
  'health_card_expanded': {},       // Mobile: tapped to expand
  'percentile_bar_hovered': {},     // Hovered on a percentile bar
  'red_flag_expanded': {},          // Expanded a red flag description
  'peer_panel_expanded': {},        // Clicked to expand peer comparison
  'sparkline_clicked': {},          // Clicked sparkline to open full chart
  'metric_history_viewed': {},      // Full chart opened
  'fair_value_gauge_viewed': {},    // Fair value gauge viewed
  'peer_ticker_clicked': {},        // Clicked a peer ticker link

  // Paywall
  'paywall_impression': {},         // Gated content viewed
  'paywall_cta_clicked': {},        // Upgrade button clicked
  'paywall_dismissed': {},          // User scrolled past without clicking

  // Conversion
  'fundamentals_to_watchlist': {},  // Added to watchlist from fundamentals
  'fundamentals_to_screener': {},   // Navigated to screener from fundamentals
};
```

**Implementation:** Lightweight event function that sends to analytics backend:

```typescript
// lib/utils/analytics.ts
function trackFundamentalsEvent(action: string, ticker: string, metadata?: Record<string, any>) {
  // Send to backend analytics endpoint or third-party (e.g., Mixpanel, PostHog)
  if (typeof window !== 'undefined') {
    // Queue event for batch sending
    window.__fundamentalsEvents = window.__fundamentalsEvents || [];
    window.__fundamentalsEvents.push({
      category: 'fundamentals',
      action,
      ticker,
      metadata,
      timestamp: new Date().toISOString(),
    });
  }
}
```

### 3.6 A/B Testing Infrastructure

For testing the free/premium boundary (e.g., 6 vs. 8 free percentile bars), implement a simple A/B test mechanism:

```typescript
// lib/utils/ab-test.ts
interface ABTestConfig {
  id: string;
  variants: string[];
  defaultVariant: string;
}

function getABTestVariant(config: ABTestConfig, userId?: string): string {
  // Deterministic assignment based on user ID hash
  // Consistent across sessions for the same user
  if (!userId) return config.defaultVariant;

  const hash = simpleHash(userId + config.id);
  const variantIndex = hash % config.variants.length;
  return config.variants[variantIndex];
}

// Usage:
const freePercentileCount = getABTestVariant({
  id: 'fundamentals_free_percentile_count',
  variants: ['6', '8', '10'],
  defaultVariant: '6',
}, user?.id);
```

### 3.7 Free/Premium Boundary Summary

| Feature | Free Tier | Premium Tier |
|---|---|---|
| Sector percentile bars | 6 core metrics | All 14+ metrics |
| Health card badge + lifecycle | Yes | Yes |
| Health card strengths/concerns | 2 each | Unlimited |
| Red flags (high severity) | Yes | Yes |
| Red flags (medium, low) | No | Yes |
| Red flag descriptions | Collapsed | Expanded by default |
| Peer comparison (collapsed teaser) | Yes | Yes |
| Peer comparison (expanded table) | Blurred | Yes |
| Trend sparklines | Blurred placeholder | Yes |
| Full metric history charts | No | Yes |
| Fair value gauge | Blurred | Yes |
| Health score number (82/100) | No (badge only) | Yes |

---

## 4. New Files

### Track B (Engineering)

| File | Purpose | Est. LOC |
|---|---|---|
| `components/ui/FundamentalsPaywall.tsx` | Reusable paywall wrapper | ~120 |
| `components/ui/PaywallOverlay.tsx` | Upgrade CTA overlay | ~80 |
| `lib/utils/analytics.ts` | Fundamentals event tracking | ~60 |
| `lib/utils/ab-test.ts` | Simple A/B test infrastructure | ~40 |
| `lib/types/fundamentals.ts` | Add feature flag types | ~20 |

## 5. Modified Files

| File | Change |
|---|---|
| `backend/services/feature_flags.go` | Add fundamentals feature flags |
| `components/ticker/TickerFundamentals.tsx` | Wrap gated content in `FundamentalsPaywall` |
| `components/ticker/tabs/MetricsTab.tsx` | Wrap premium sparklines in paywall |
| `components/ticker/FundamentalHealthCard.tsx` | Use teaser variant for strengths/concerns |
| `components/ticker/PeerComparisonPanel.tsx` | Use blur variant for expanded table |
| `components/ticker/FairValueGauge.tsx` | Use blur variant for gauge |
| `components/ui/SectorPercentileBar.tsx` | Use lock variant for premium metrics |

## 6. Acceptance Criteria

### Track A (UXR)

- [ ] Competitive audit covers 6 platforms with annotated screenshots
- [ ] Usability testing establishes baseline metrics (task completion, time, confidence)
- [ ] Card sort produces validated metric category structure
- [ ] Survey reaches 200+ responses with statistically significant results
- [ ] Prototype test identifies winning design direction with >65% preference
- [ ] Final UXR report delivered with actionable recommendations for each project

### Track B (Paywall Engineering)

- [ ] `FundamentalsPaywall` component works with all 3 variants (blur, lock, teaser)
- [ ] Feature flag system correctly gates content based on user subscription
- [ ] Paywall impressions and CTA clicks tracked in analytics
- [ ] Upgrade CTA navigates to subscription page with attribution params
- [ ] Free tier users see teaser content (not blank) — always demonstrate value
- [ ] Premium users see no paywall components (zero visual noise)
- [ ] A/B test infrastructure assigns consistent variants per user
- [ ] Free/premium boundaries match the specification table above
- [ ] Mobile paywall overlays are touch-friendly and don't break layout
