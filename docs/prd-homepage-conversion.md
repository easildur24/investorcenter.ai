# PRD: InvestorCenter Homepage Redesign for New User Conversion

**Author:** Product & UX
**Date:** 2026-03-01
**Status:** Draft
**Stakeholders:** Engineering, Design, Marketing, Growth

---

## Executive Summary

InvestorCenter's homepage currently functions as a **dashboard for existing users** rather than a **landing page for new visitors**. It leads with generic "Professional Financial Analytics" messaging, shows live data widgets that display "$0.00" outside market hours, and offers no social proof, pricing clarity, or product differentiation. The result is a homepage that fails the fundamental question every new visitor asks: *"Why should I sign up for this instead of what I already use?"*

This PRD proposes a conversion-focused homepage restructure built around three strategic moves:

1. **Lead with differentiation, not data.** Reframe the hero around what makes InvestorCenter unique (AI market summaries, Reddit sentiment analysis, IC Score ratings) instead of generic "financial data" messaging.
2. **Build trust before asking for commitment.** Add social proof, pricing transparency, and product previews so visitors can evaluate the product before hitting a signup wall.
3. **Restructure the scroll flow for conversion.** Reorder sections to follow the proven SaaS landing page pattern: Hook → Differentiate → Prove → Preview → Price → Convert.

**Expected outcome:** 2-3x improvement in homepage-to-signup conversion rate within 8 weeks, validated through A/B testing.

---

## Table of Contents

1. [Problem Statement](#1-problem-statement)
2. [Goals & Success Metrics](#2-goals--success-metrics)
3. [UX Research Recommendations](#3-ux-research-recommendations)
4. [Feature Improvements — Detailed Requirements](#4-feature-improvements--detailed-requirements)
5. [UX Design Principles & Visual Direction](#5-ux-design-principles--visual-direction)
6. [Competitive Positioning Summary](#6-competitive-positioning-summary)
7. [Implementation Roadmap](#7-implementation-roadmap)
8. [Open Questions & Assumptions](#8-open-questions--assumptions)

---

## 1. Problem Statement

### Core Conversion Problem

A first-time visitor landing on InvestorCenter's homepage sees a wall of financial widgets — market indices at $0.00 (on weekends/off-hours), a news feed with "Unknown" sources, and sector performance tiles all reading "No data." The hero says "Professional Financial Analytics," which describes every financial platform from Yahoo Finance to Bloomberg. There is no reason for a new visitor to believe InvestorCenter offers anything they can't get elsewhere, and the single "Start Free Trial" CTA asks for commitment before demonstrating value.

### Key Gaps

| Gap | Description |
|-----|-------------|
| **"Why should I care?"** | The value proposition is generic. "Comprehensive financial data" and "interactive charts" describe TradingView, Yahoo Finance, and dozens of other platforms. Nothing communicates what is unique to InvestorCenter. |
| **"Why should I trust you?"** | Zero social proof — no user count, no testimonials, no press logos, no ratings. Footer links to About, Contact, and Privacy all lead to "Coming Soon" pages. The platform feels early-stage and unvalidated. |
| **"What will I get?"** | "Start Free Trial" with no explanation of trial duration, what's included vs. locked, or what happens after. No pricing page exists. Visitors cannot evaluate the cost-benefit before committing. |
| **"Does it actually work?"** | No product screenshots, demos, or interactive previews. The live widgets are the only "proof," but they show empty states outside market hours — the exact times when many retail investors browse for new tools. |

### Jobs-To-Be-Done by Persona

#### Persona A: Retail Investor / Beginner (25-35yo, uses Robinhood)

| JTBD | Current Gap |
|------|-------------|
| "Help me understand what's happening in the market today without reading 10 articles" | The AI Market Summary does this, but it's buried below the fold, labeled generically, and has no CTA to sign up for daily summaries |
| "Show me which stocks are trending and why" | Top Movers and Reddit Trends exist but aren't positioned as a unique advantage — Reddit Trends isn't even mentioned on the homepage |
| "Give me an easy way to track my portfolio" | Watchlist section only appears for logged-in users; new visitors see nothing about this capability |

#### Persona B: Active / Intermediate Trader (35-50yo, uses TradingView or Seeking Alpha)

| JTBD | Current Gap |
|------|-------------|
| "Give me a tool that combines charting, screening, and research in one place" | The Features section lists capabilities but shows no screenshots or interactive previews to demonstrate depth |
| "Help me find undervalued or high-momentum stocks faster" | IC Score (1-100 proprietary rating) is nowhere on the homepage — the platform's strongest differentiator is invisible |
| "Show me data I can't get elsewhere" | Reddit sentiment, AI summaries, and IC Scores are all absent from the homepage narrative |

#### Persona C: Finance Professional / Power User (30-50yo, Bloomberg/FactSet user)

| JTBD | Current Gap |
|------|-------------|
| "Give me institutional-quality data at a fraction of the cost" | The headline says "institutional-grade" but provides no evidence — no data coverage stats, no methodology explanation, no comparison to expensive alternatives |
| "Let me evaluate the platform's data depth before committing" | No screener preview, no sample ticker page, no data quality examples visible without signup |
| "Integrate into my existing workflow" | No mention of API access, export capabilities, or how IC Score methodology works |

---

## 2. Goals & Success Metrics

### Primary Goal

**Increase homepage-to-signup conversion rate by 2-3x** through a conversion-focused redesign that addresses trust, differentiation, and value clarity for new visitors.

### KPIs

| # | KPI | Baseline (Assumed) | Target | Measurement Method |
|---|-----|-------------------|--------|-------------------|
| 1 | **Hero CTA click-through rate** (clicks on "Start Free Trial" or primary CTA / unique visitors) | ~2-3% | 6-8% | Analytics event tracking on CTA clicks |
| 2 | **Homepage-to-signup conversion rate** (completed signups / unique homepage visitors) | ~0.5-1% | 2-3% | Funnel analytics: homepage → signup page → signup complete |
| 3 | **Scroll depth to 50%** (% of visitors who scroll past the fold to mid-page) | ~35-40% | 55-65% | Scroll depth tracking via analytics |
| 4 | **Average time on page** (for new visitors, first session) | ~25-35 seconds | 50-70 seconds | Analytics with session segmentation (new vs. returning) |
| 5 | **7-day retention after signup** (users who return within 7 days of signup originating from homepage) | ~15-20% | 30-40% | Cohort analysis of homepage-originated signups |

### Secondary Metrics

- Bounce rate reduction (target: -15-20% from baseline)
- Signup page abandonment rate (target: <40%)
- Widget interaction rate for new visitors (target: 20%+ interact with at least one widget)
- Secondary CTA engagement (View Markets, feature section links)

---

## 3. UX Research Recommendations

### Pre-Implementation Research Plan

Research should begin **before** implementation to validate assumptions in this PRD. Budget: 2-3 weeks.

#### Qualitative Methods

**Method 1: 5-Second Test (Remote, Unmoderated)**

- **Sample:** 30-40 participants across 3 personas (recruited via UserTesting or Maze)
- **Protocol:** Show the current homepage for 5 seconds, then ask:
  - "What does this website do?"
  - "Who is it for?"
  - "What makes it different from other financial tools you use?"
  - "Would you sign up? Why or why not?"
- **Hypothesis to validate:** New visitors cannot articulate InvestorCenter's differentiation within 5 seconds. They default to "it's a stock market website."
- **Success criteria:** <20% of participants can name a unique feature (AI summaries, Reddit trends, IC Score) — confirming the need for clearer differentiation messaging.

**Method 2: Moderated Usability Test (Remote, 45 minutes each)**

- **Sample:** 6-8 participants (2-3 per persona), recruited with screener questions about current tools and investing experience
- **Protocol:**
  - Task 1: "You heard about InvestorCenter from a friend. Spend 2 minutes on the homepage and tell me what you'd use it for."
  - Task 2: "Find out how much it costs to use InvestorCenter." (Tests pricing clarity)
  - Task 3: "Sign up for a free account and tell me what you expect to get."
  - Post-task interview: "What would make you switch from [their current tool]?" / "What's missing from this page?"
- **Hypothesis to validate:** Users cannot determine pricing or trial details, and they cannot distinguish InvestorCenter from competitors.
- **Persona-specific questions:**
  - Beginner: "Does this feel approachable or intimidating?"
  - Active trader: "What data or tools would you check first?"
  - Professional: "Does this feel serious enough for professional use?"

**Method 3: Competitor Comparison Interview (Remote, 30 minutes each)**

- **Sample:** 6 participants who actively use TradingView, Seeking Alpha, or Yahoo Finance
- **Protocol:** Show them InvestorCenter's homepage alongside their current tool's homepage. Ask:
  - "What does [competitor] do better at explaining itself?"
  - "What would InvestorCenter need to show you to make you try it?"
  - "Rate your trust level for each site on a 1-10 scale and explain why."
- **Hypothesis to validate:** Competitor homepages build trust faster through social proof, product previews, and specific value claims.

#### Quantitative Methods

**Method 1: Heatmap & Session Recording Analysis**

- **Tool:** Hotjar, Microsoft Clarity, or PostHog
- **Duration:** 2 weeks of data collection on current homepage
- **What to measure:**
  - Click heatmap: Where do users click? Do they click on widgets expecting interactivity?
  - Scroll heatmap: Where is the drop-off point? What's the last section most visitors see?
  - Session recordings: Watch 50+ new-visitor sessions to identify confusion points, rage clicks, and abandonment patterns
- **Hypothesis to validate:** Most new visitors never scroll past the Market Overview section. The Features section (conversion-relevant) has <10% visibility.

**Method 2: A/B Test Design (Post-Implementation)**

- **Test 1 — Hero messaging:** Current generic headline vs. differentiation-focused headline (test primary CTA CTR)
- **Test 2 — Social proof placement:** Hero section social proof bar vs. dedicated section below fold (test scroll depth and signup rate)
- **Test 3 — Content hierarchy:** Current data-first layout vs. restructured conversion-first layout (test overall homepage-to-signup rate)
- **Statistical requirements:** 95% confidence, minimum 1,000 unique visitors per variant, 2-week minimum test duration

#### Insight Synthesis Framework

Use an **Opportunity Solution Tree** (Teresa Torres model):

```
Desired Outcome: Increase homepage-to-signup conversion rate
│
├── Opportunity: Visitors don't understand differentiation
│   ├── Solution: Rewrite hero with specific value claims
│   ├── Solution: AI feature spotlight section
│   └── Solution: Reddit Trends callout
│
├── Opportunity: Visitors don't trust the platform
│   ├── Solution: Social proof section
│   ├── Solution: Pricing transparency
│   └── Solution: Product screenshots/demo
│
├── Opportunity: Visitors can't evaluate the product
│   ├── Solution: Interactive demo preview
│   ├── Solution: Feature tour with screenshots
│   └── Solution: Live widget improvements
│
└── Opportunity: CTAs are insufficient
    ├── Solution: Multiple CTA placements
    ├── Solution: Sticky nav signup button
    └── Solution: Footer CTA banner
```

Synthesize qualitative findings via **affinity mapping** — group user quotes and observations into clusters, then map clusters to the opportunity tree branches above. This ensures every design decision traces back to a validated user need.

---

## 4. Feature Improvements — Detailed Requirements

### 4.1 Hero Section Redesign

**Priority:** P0

**User Story:**
As a new visitor who has never heard of InvestorCenter, I want to immediately understand what makes this platform different from Yahoo Finance or TradingView so that I can decide in under 10 seconds whether it's worth exploring further.

**Current State:**
- Headline: "Professional Financial Analytics"
- Subheadline: "Access comprehensive financial data, interactive charts, and powerful analytics tools. Make informed investment decisions with institutional-grade research and insights."
- CTA: "Start Free Trial" / "View Markets"
- Mini dashboard showing 3 index tiles (often $0.00 on weekends)

**Proposed Changes:**

**Headline (Option A — Differentiation-led):**
> "The Smartest Way to Research Stocks"

**Headline (Option B — Specificity-led):**
> "AI-Powered Stock Research for 5,600+ Stocks and ETFs"

**Subheadline:**
> "Daily AI market summaries, Reddit sentiment signals, proprietary IC Score ratings, and a powerful stock screener — all in one platform. Free to start."

**Social Proof Bar** (directly below subheadline):
> "[X,XXX] investors already use InvestorCenter" + average rating if available

**Primary CTA:**
> "Get Started Free" with microcopy below: "No credit card required. Free forever for core features."

**Secondary CTA:**
> "See How It Works" (scrolls to the product tour section)

**Mini Dashboard Improvement:**
- Keep the index tiles but add a fallback state for off-hours: show the previous close with a "Market Closed — Opens Mon 9:30 AM ET" label instead of $0.00
- This is critical because weekends and evenings are peak browsing times for retail investors evaluating new tools

**Acceptance Criteria:**

```
GIVEN a new visitor lands on the homepage for the first time
WHEN the hero section loads
THEN they see a differentiation-focused headline, a subheadline that names
     at least 3 specific features, a social proof metric, a primary CTA with
     "no credit card" microcopy, and a secondary "See How It Works" CTA.

GIVEN the market is closed (weekends, holidays, after-hours)
WHEN the hero mini dashboard renders
THEN it shows the most recent closing prices (not $0.00) with a
     "Market Closed" indicator and countdown to next open.

GIVEN a user clicks "See How It Works"
WHEN the page scrolls
THEN it smooth-scrolls to the Product Tour section (4.4) with
     the section highlighted briefly.
```

**UX Design Guidance:**
- Hero should occupy 85-100vh on desktop to create a focused first impression
- Social proof bar: horizontal row with a subtle separator, using muted text — not a full-width banner
- CTA button hierarchy: primary button in brand blue (`ic-blue`), secondary as ghost/outline button
- The mini dashboard should feel like a "live preview" of the platform, not the focal point — reduce its visual weight by 20-30% compared to current
- On mobile, stack: headline → subheadline → social proof → CTAs → mini dashboard (scrollable)

---

### 4.2 AI Feature Spotlight Section (NEW)

**Priority:** P0

**User Story:**
As a retail investor who is overwhelmed by financial news, I want to see a live example of InvestorCenter's AI market summary so that I can understand the value of getting AI-curated insights before I sign up.

**Proposed Section:**

A dedicated section immediately below the hero that showcases the AI Market Summary as the platform's primary differentiator.

**Section Layout:**

| Left Column (60%) | Right Column (40%) |
|---|---|
| **Section label:** "AI-Powered Insights" | Live or recent AI Market Summary card |
| **Headline:** "Your Daily Market Brief, Written by AI" | Actual summary content with timestamp |
| **Body:** "Every day, InvestorCenter's AI analyzes market movements, top movers, and sector trends to deliver a plain-English summary of what happened and why it matters. No more reading 10 articles to understand the market." | Styled as a preview card with a subtle glass/blur effect suggesting "unlock more" |
| **CTA:** "Sign Up for Daily AI Summaries →" | |

**How the AI Works Explanation:**
Below the two-column layout, a brief 3-step visual:
1. "We analyze real-time price movements across 5,600+ stocks"
2. "Our AI identifies the key stories, movers, and trends"
3. "You get a plain-English summary delivered daily"

**Acceptance Criteria:**

```
GIVEN a new visitor scrolls past the hero section
WHEN the AI Feature Spotlight section enters the viewport
THEN they see a live or cached version of the most recent AI market summary
     displayed as a styled preview card.

GIVEN the AI summary is unavailable or errored
WHEN the section renders
THEN it shows a static example summary with a label "Example summary from
     [recent date]" rather than an empty or error state.

GIVEN a visitor clicks the "Sign Up for Daily AI Summaries" CTA
WHEN the signup page loads
THEN a query parameter (e.g., ?source=ai_summary) is passed so that
     post-signup onboarding can reference their interest in AI summaries.
```

**UX Design Guidance:**
- The live summary card should look polished — use a card with subtle shadow, the blue left-border accent already in the MarketSummary component, and a slight frosted-glass effect
- Add a faint animated shimmer on the card edge to draw attention without being distracting
- The 3-step "how it works" strip should use small icons (not hero-sized), inline with text, horizontally on desktop and vertically stacked on mobile
- Keep the background neutral (`ic-bg-secondary`) to contrast with the hero above

---

### 4.3 Reddit Trends Differentiator Callout (NEW)

**Priority:** P1

**User Story:**
As an active retail investor, I want to know that InvestorCenter tracks Reddit sentiment so that I can use crowd-sourced signals as part of my research — something my current tools don't offer.

**Proposed Section:**

A compact callout section positioned after the AI Spotlight, designed to highlight the Reddit Trends feature as a unique edge.

**Section Layout:**

| Left Column (40%) | Right Column (60%) |
|---|---|
| Preview mockup: a styled card showing 3-4 trending Reddit tickers with sentiment scores and post counts | **Section label:** "Retail Sentiment Signals" |
| | **Headline:** "See What Reddit Is Talking About Before Everyone Else" |
| | **Body:** "InvestorCenter monitors r/wallstreetbets, r/stocks, and r/investing in real-time. Our AI categorizes posts by ticker, sentiment, and momentum — giving you a unique signal that traditional platforms miss." |
| | **CTA:** "Explore Reddit Trends →" (links to `/reddit`) |

**Acceptance Criteria:**

```
GIVEN a new visitor scrolls to the Reddit Trends section
WHEN the section renders
THEN they see a preview card showing 3-4 real or example trending tickers
     with Reddit mention counts and AI-classified sentiment (bullish/bearish/neutral).

GIVEN the visitor clicks "Explore Reddit Trends"
WHEN the Reddit Trends page loads
THEN the page is accessible without login (public page for conversion purposes).
```

**UX Design Guidance:**
- Invert the column layout from the AI section (preview on left, text on right) for visual variety
- The preview card should look like a real product screenshot, slightly tilted or with a perspective shadow to suggest depth
- Use a subtle Reddit-brand-orange accent (#FF4500) in the section sparingly (e.g., one icon or label) to create brand association
- Keep the section compact — no more than 40vh on desktop
- This section can be A/B tested against removal to measure incremental conversion impact

---

### 4.4 Product Screenshot / Interactive Demo Preview

**Priority:** P1

**User Story:**
As an intermediate trader evaluating whether to switch from TradingView, I want to see what InvestorCenter's screener, charts, and analytics actually look like so that I can judge the product's depth without signing up first.

**Proposed Section:**

A tabbed or segmented product tour showing 3-4 key features with screenshots and brief descriptions.

**Tabs/Segments:**

| Tab | Screenshot Content | Caption |
|-----|-------------------|---------|
| Stock Screener | Screener page with filters active and results visible | "Filter 5,600+ stocks by IC Score, valuation, momentum, and 50+ metrics" |
| AI Market Summary | Full market summary view with detail | "Get AI-generated daily market summaries explaining what moved and why" |
| Ticker Research | A ticker detail page showing IC Score, financials, chart | "Deep-dive into any stock with proprietary IC Score ratings (1-100)" |
| Watchlist & Alerts | Watchlist view with price alerts | "Track your portfolio with real-time watchlists and custom price alerts" |

**Acceptance Criteria:**

```
GIVEN a new visitor reaches the Product Tour section
WHEN they click a tab (e.g., "Stock Screener")
THEN a corresponding product screenshot or animated preview loads
     with a brief feature description below the image.

GIVEN the section loads on mobile
WHEN the viewport is <768px
THEN the tabs collapse into a vertical accordion or swipeable carousel
     instead of horizontal tabs.

GIVEN a visitor clicks "See How It Works" from the hero
WHEN the page scrolls to this section
THEN the first tab (Stock Screener) is pre-selected and visible.
```

**UX Design Guidance:**
- Screenshots should be actual product images, slightly cropped and annotated with subtle callout arrows or highlights pointing to key features
- Use a browser-frame mockup wrapper around screenshots to create a "this is real software" feel
- Tab bar should be sticky within the section as users scroll through longer content
- Add a "Try It Free →" CTA at the bottom of the section
- Consider a subtle auto-advance (every 5 seconds) with manual tab selection overriding the timer
- Screenshot images should be optimized (WebP, lazy-loaded) to avoid impacting page load

---

### 4.5 Social Proof & Trust Section (NEW)

**Priority:** P0

**User Story:**
As a new visitor who found InvestorCenter through a search engine, I want to see evidence that other people use and trust this platform so that I feel confident providing my email to sign up.

**Proposed Section:**

A dedicated trust section combining three elements: metrics bar, testimonials, and security signals.

**Metrics Bar:**

| Metric | Display |
|--------|---------|
| Stocks & ETFs covered | "5,600+ US Stocks & 4,200+ ETFs" |
| Data points | "Updated every 30 seconds during market hours" |
| IC Score rated companies | "[X,XXX] companies rated by IC Score" |

**Testimonials (2-3):**

Each testimonial card includes:
- Quote (2-3 sentences max)
- User avatar (or initials), first name, and persona descriptor (e.g., "Retail Investor" / "Day Trader" / "Financial Analyst")
- Star rating if applicable

*Note: Testimonials must be collected from real users. If real testimonials are not yet available, use the metrics bar and data-driven trust signals only. Do NOT fabricate testimonials.*

**Security / Data Trust Signals:**

- "Data sourced from SEC EDGAR, Polygon.io, and official exchanges"
- "Bank-level encryption (HTTPS/TLS)"
- "No credit card required to start"

**Acceptance Criteria:**

```
GIVEN a new visitor scrolls to the Social Proof section
WHEN the section loads
THEN they see at least 3 data coverage metrics, a data sources attribution,
     and a security/encryption statement.

GIVEN real user testimonials are available
WHEN the section renders
THEN 2-3 testimonial cards display with real user quotes, first names,
     and persona descriptors. No stock photos or fabricated quotes.

GIVEN testimonials are not yet collected
WHEN the section renders
THEN the testimonials area is omitted entirely (not shown with
     placeholder content), and the metrics bar + trust signals fill the section.
```

**UX Design Guidance:**
- Metrics bar: horizontal on desktop, vertical stack on mobile. Use large bold numbers with descriptive labels below
- Testimonial cards: subtle background, rounded corners, with a quotation mark icon. Arranged in a 3-column grid on desktop, carousel on mobile
- Trust signals: small row at the bottom of the section with icons (lock icon, shield icon) and muted text
- The overall section should feel understated and credible — avoid "marketing-speak" visual treatments

---

### 4.6 Pricing Transparency

**Priority:** P1

**User Story:**
As a new visitor considering signup, I want to know exactly what I get for free vs. what costs money so that I can make an informed decision about whether the free tier meets my needs or whether I need to budget for a paid plan.

**Proposed Section (Homepage Teaser):**

A brief pricing preview on the homepage with a link to a full pricing page.

**Homepage Pricing Teaser:**

| Element | Content |
|---------|---------|
| **Headline** | "Free to Start. Upgrade When You're Ready." |
| **Subheadline** | "Core features are free forever. Unlock advanced analytics with Pro." |

**Pricing Tiers (3-tier comparison):**

| | Free | Pro | Institutional |
|--|------|-----|---------------|
| **Price** | $0/month | $XX/month | Contact Us |
| **Stock Screener** | Basic filters | All 50+ filters | All + API access |
| **AI Market Summary** | Daily summary | Daily + ticker-level AI | Custom reports |
| **IC Score** | Top 10 preview | Full access (all stocks) | Full + methodology docs |
| **Watchlists** | 1 watchlist, 10 stocks | Unlimited | Unlimited + team sharing |
| **Reddit Trends** | Trending tickers | Full sentiment data | Full + historical data |
| **Price Alerts** | 3 alerts | Unlimited | Unlimited + webhook |
| **Data Export** | No | CSV export | CSV + API |

*Note: Specific pricing and tier limits are pending stakeholder input. The structure above is a recommendation — actual limits should be set based on product strategy and competitive analysis.*

**Acceptance Criteria:**

```
GIVEN a new visitor scrolls to the pricing section on the homepage
WHEN the section renders
THEN they see a 2-3 tier comparison table showing Free, Pro,
     and optionally Institutional tiers with clear feature differentiation.

GIVEN the free tier is displayed
WHEN the visitor reads the free tier column
THEN it clearly states "$0/month" with no asterisks, time limits,
     or hidden conditions (unless a genuine trial limitation exists).

GIVEN a visitor wants full pricing details
WHEN they click "See Full Pricing" or the "Pricing" nav item
THEN they are navigated to a dedicated /pricing page with
     complete tier comparison, FAQ, and signup CTA per tier.
```

**UX Design Guidance:**
- Homepage teaser: compact, no more than 50vh. Show the 3 tier headers with 3-4 key differentiating features only — link to full page for details
- Highlight the "Pro" tier as "Most Popular" with a visual badge
- Free tier should feel generous, not restrictive — frame limitations as "upgrade for more" rather than "locked features"
- Use checkmarks and dashes for feature availability — avoid complex footnotes
- CTA per tier: "Get Started" (Free), "Start Pro Trial" (Pro), "Contact Sales" (Institutional)

---

### 4.7 Homepage Content Hierarchy Restructure

**Priority:** P0

**User Story:**
As a new visitor, I want the homepage to guide me through a logical narrative — from understanding what the product does, to seeing it in action, to trusting it, to knowing the cost — so that I arrive at the signup decision naturally rather than being asked to commit before I understand the product.

**Current Section Order (Dashboard-First):**
1. Hero (generic headline + CTAs)
2. AI Market Summary
3. Market Overview + Top Movers
4. News Feed + Upcoming Earnings
5. Sector Heatmap + Watchlist Preview
6. Features (text descriptions)
7. Footer

**Proposed Section Order (Conversion-First):**

```
┌──────────────────────────────────────────────────────────────────┐
│ 1. HERO SECTION (Redesigned)                                     │
│    - Differentiation headline + social proof bar                 │
│    - Primary CTA + "See How It Works" secondary CTA              │
│    - Mini dashboard (with off-hours fallback)                    │
├──────────────────────────────────────────────────────────────────┤
│ 2. AI FEATURE SPOTLIGHT (NEW)                                    │
│    - Live AI summary preview + "how it works" strip              │
│    - CTA: "Sign Up for Daily AI Summaries"                       │
├──────────────────────────────────────────────────────────────────┤
│ 3. LIVE MARKET DATA                                              │
│    - Market Overview + Top Movers (condensed to single row)      │
│    - Purpose: demonstrate real-time data quality                 │
├──────────────────────────────────────────────────────────────────┤
│ 4. PRODUCT TOUR / SCREENSHOTS (NEW)                              │
│    - Tabbed: Screener, AI Summary, Ticker Research, Watchlist    │
│    - CTA: "Try It Free"                                          │
├──────────────────────────────────────────────────────────────────┤
│ 5. REDDIT TRENDS CALLOUT (NEW)                                   │
│    - Preview card + explanation + CTA                            │
├──────────────────────────────────────────────────────────────────┤
│ 6. SOCIAL PROOF & TRUST (NEW)                                    │
│    - Data coverage metrics + testimonials + trust signals        │
├──────────────────────────────────────────────────────────────────┤
│ 7. MARKET NEWS + EARNINGS CALENDAR                               │
│    - Condensed: 4 news items + 5 earnings (reduced from current) │
│    - Purpose: show freshness and data breadth                    │
├──────────────────────────────────────────────────────────────────┤
│ 8. PRICING TEASER (NEW)                                          │
│    - 3-tier comparison + "See Full Pricing" link                 │
│    - CTA per tier                                                │
├──────────────────────────────────────────────────────────────────┤
│ 9. FULL-WIDTH FOOTER CTA BANNER (NEW)                            │
│    - Final conversion push: "Start Your Free Account Today"      │
├──────────────────────────────────────────────────────────────────┤
│ 10. FOOTER                                                       │
│     - Platform links, company links, legal                       │
└──────────────────────────────────────────────────────────────────┘
```

**Sections Removed or Relocated:**
- **Sector Heatmap:** Removed from homepage. Currently shows "No data" for all sectors, undermining trust. Move to Screener page as a feature.
- **Watchlist Preview:** Removed from homepage for logged-out users (already returns `null`). Keep as a post-login dashboard element.
- **Features Section (text-only):** Replaced by the Product Tour section (4.4) which shows rather than tells.

**Sections Condensed:**
- **Market Overview + Top Movers:** Condense to a single row. Keep Top Movers as primary (more engaging for new visitors), move full Market Overview to a "View All Indices" expandable or link.
- **News + Earnings:** Reduce item count (4 news headlines, 5 earnings) to keep the section tight. These serve as "freshness proof" rather than primary content for new visitors.

**Acceptance Criteria:**

```
GIVEN a new visitor loads the homepage
WHEN the page renders for a logged-out user
THEN the sections appear in the order defined above (1-10),
     and the Sector Heatmap and Watchlist Preview sections are not visible.

GIVEN a returning logged-in user loads the homepage
WHEN the page renders
THEN the Watchlist Preview appears as an additional card alongside
     the Market News section (Section 7), and the rest of the
     conversion-focused layout remains intact.

GIVEN the page is viewed on mobile (<768px)
WHEN sections render
THEN each section stacks vertically with appropriate padding,
     and no horizontal scrolling is required except within
     explicitly scrollable components (mini dashboard, earnings list).
```

---

### 4.8 Secondary & Footer CTAs

**Priority:** P1

**User Story:**
As a visitor who has scrolled through the entire homepage, I want a clear final prompt to sign up so that I don't have to scroll back up to find the CTA if I've been convinced by the content below the fold.

**Proposed Changes:**

**Repeated CTA after Product Tour (Section 4):**
- Inline CTA: "Ready to try it? Get Started Free →" (text link, not a full banner)
- Positioned as the last element of the Product Tour section

**Full-Width Footer CTA Banner (above site footer):**

| Element | Content |
|---------|---------|
| **Background** | Gradient from `ic-blue` to a darker shade |
| **Headline** | "Start Making Smarter Investment Decisions Today" |
| **Subheadline** | "Join [X,XXX] investors using InvestorCenter. Free forever for core features." |
| **CTA Button** | "Create Free Account" (white button on blue background) |
| **Microcopy** | "No credit card required" |

**Acceptance Criteria:**

```
GIVEN a visitor scrolls to the bottom of the homepage
WHEN the footer CTA banner enters the viewport
THEN they see a full-width banner with headline, subheadline,
     CTA button, and "no credit card" microcopy.

GIVEN a visitor clicks the footer CTA
WHEN the signup page loads
THEN a query parameter (e.g., ?source=footer_cta) is passed
     for attribution tracking.
```

**UX Design Guidance:**
- The footer CTA banner should be visually distinct from all other sections — use the brand blue gradient background with white text
- Banner height: ~200px on desktop, ~250px on mobile (give breathing room)
- CTA button: white background, blue text, large (min 48px height)
- This banner should have a subtle parallax or fade-in animation as it enters the viewport

---

### 4.9 Navigation Enhancement

**Priority:** P1

**User Story:**
As a visitor exploring the homepage, I want a persistent way to sign up from anywhere on the page, and I want to quickly find pricing information, so that I can take action the moment I'm convinced.

**Proposed Changes:**

1. **Sticky "Sign Up Free" button** in the header for logged-out users
   - Replaces the current "Get Started" link with a more prominent button
   - Remains visible on scroll (header is already sticky)
   - On mobile, show as a compact icon-button or slide-in CTA after scrolling past the hero

2. **Add "Pricing" to main navigation**
   - Position: between "Earnings" and the search bar
   - Links to `/pricing` (new page to be created)

3. **Add "Product" dropdown or "Why InvestorCenter" link**
   - Optional: a mega-menu or simple dropdown with links to feature-specific pages (AI Summary, Screener, Reddit Trends, IC Score)
   - Alternative: a single "Why InvestorCenter" link pointing to a product overview page or scrolling to the Product Tour section on the homepage

**Acceptance Criteria:**

```
GIVEN a logged-out visitor is on any page of the site
WHEN the header renders
THEN a "Sign Up Free" button is visible in the top navigation
     bar, styled as a primary action button (blue background, white text).

GIVEN a visitor clicks "Pricing" in the navigation
WHEN the pricing page loads
THEN they see the full pricing comparison table with Free, Pro,
     and Institutional tiers, FAQ, and signup CTAs.

GIVEN a logged-in user is on any page
WHEN the header renders
THEN the "Sign Up Free" button is hidden and replaced with
     the user menu (current behavior).
```

**UX Design Guidance:**
- The "Sign Up Free" nav button should be slightly smaller than the hero CTA but clearly a button (not a text link) — `px-4 py-2 rounded-lg bg-ic-blue text-white`
- "Pricing" nav link should be a standard text nav item matching the existing nav style
- On mobile (hamburger menu), "Sign Up Free" should be the first item in the menu, visually separated from navigation links
- Consider reducing the current nav items on mobile — "Home" can be the logo, reducing the list to: Screener, Crypto, Reddit Trends, Earnings, Pricing, Sign Up Free

---

## 5. UX Design Principles & Visual Direction

### Design Principles

| # | Principle | Application |
|---|-----------|-------------|
| 1 | **Show, don't tell.** | Replace text descriptions of features with live previews, screenshots, and interactive demos. The current Features section says "Interactive Charts" — the redesigned homepage should show a chart. |
| 2 | **Earn trust before asking for commitment.** | Place social proof, product previews, and pricing before the signup CTA. No visitor should encounter a signup form before they understand what they're signing up for. |
| 3 | **Lead with what's unique, not what's expected.** | Every financial platform has charts and watchlists. Lead with AI summaries, Reddit sentiment, and IC Score — the things only InvestorCenter offers. |
| 4 | **Design for the skeptic.** | Assume the visitor has tried 3 other platforms and been disappointed. Every claim needs evidence. "Professional-grade" means nothing without a screenshot proving it. |
| 5 | **Respect the visitor's time.** | A new visitor will give you 10-15 seconds before deciding to scroll or bounce. The hero must communicate differentiation in that window. Everything below the fold should progressively deepen interest, not repeat it. |

### Visual Direction

**Color:**
- Primary palette: keep the current `ic-blue` (#2563EB or similar) as the primary action color
- Backgrounds: maintain the light neutral alternating pattern (`ic-bg-primary` / `ic-bg-secondary`) for section rhythm
- Accent: introduce a warm secondary accent (amber or orange) for attention-drawing elements like badges, "NEW" labels, and the Reddit Trends section — this breaks the current monotone blue and adds visual energy
- Data visualization colors: green/red for positive/negative (already in use) — ensure WCAG contrast ratios are met (avoid pure green/red for colorblind users; add directional arrows or +/- prefixes)

**Typography:**
- Headlines: current font at 2.5-3.5rem range with tight tracking (`tracking-tight`) — consider a bold weight (700-800) for hero headlines to create visual impact
- Body text: 1rem-1.125rem with 1.6-1.75 line height for readability
- Subheadlines and section labels: use uppercase letter-spacing (`tracking-wider uppercase text-sm`) for section labels (e.g., "AI-POWERED INSIGHTS", "LIVE DATA") to create visual hierarchy — this pattern is already used for "FEATURES" and "LIVE DATA" labels
- Numbers and data: use tabular/monospace numerals for financial data to ensure column alignment

**Motion & Animation:**
- Page load: sections should fade-in with a subtle upward translate (8-12px) as they enter the viewport. Use `IntersectionObserver` with staggered delays (100-200ms between elements)
- Micro-interactions: CTA buttons should have a subtle scale transform on hover (1.02-1.05x). The AI summary card should have a shimmer/pulse effect to suggest "live" data
- Avoid: autoplay videos, aggressive parallax, or anything that competes with the data widgets for attention
- Performance: all animations should use CSS transforms and opacity only (GPU-accelerated). No layout-triggering animations. Respect `prefers-reduced-motion` media query

**Accessibility (WCAG 2.1 AA Minimum):**
- Color contrast: all text must meet 4.5:1 ratio for normal text, 3:1 for large text (18pt+) — audit current green-on-white and red-on-white data colors
- Focus indicators: all interactive elements must have visible focus outlines (current Tailwind ring utilities are appropriate)
- Keyboard navigation: entire page must be navigable via Tab/Shift+Tab with logical focus order
- Screen reader support: all images/screenshots must have descriptive alt text; section landmarks (`<section>`, `<nav>`, `<main>`) must be semantically correct
- Touch targets: all buttons and links must have minimum 44x44px touch targets on mobile
- Reduced motion: respect `prefers-reduced-motion` — disable all animations and transitions when enabled

---

## 6. Competitive Positioning Summary

### Competitive Landscape

| Competitor | Strengths (for new users) | Weaknesses (that IC can exploit) |
|------------|--------------------------|----------------------------------|
| **TradingView** | Massive community, best-in-class charting, freemium model well-understood | Overwhelming for beginners, no AI summaries, no proprietary ratings, expensive Pro tiers |
| **Yahoo Finance** | Universal brand recognition, free, good news aggregation | Dated UI, ad-heavy, no screening depth, no AI features, no proprietary scoring |
| **Seeking Alpha** | Strong editorial content, earnings analysis, community | Paywall-heavy, focused on articles (not tools), no real-time screener, no Reddit/social signals |
| **Bloomberg Terminal** | Gold standard for professionals, deepest data | $24K+/year, inaccessible to retail investors, no consumer-facing web product |

### InvestorCenter's Unique Positioning

**"Only" Statement:**

> InvestorCenter is the **only** stock research platform that combines AI-generated market summaries, Reddit retail sentiment signals, and proprietary IC Score ratings (1-100) in a single free-to-start platform designed for investors who want institutional-quality insights without the institutional price tag.

**Positioning Framework:**

- **For** retail investors and active traders
- **Who** want smarter stock research without paying Bloomberg prices
- **InvestorCenter is** an AI-powered research platform
- **That** combines proprietary IC Score ratings, daily AI market summaries, and Reddit sentiment analysis
- **Unlike** TradingView (charting-focused) or Seeking Alpha (editorial-focused)
- **We** deliver institutional-grade analytics with AI-powered insights in a single, free-to-start platform

---

## 7. Implementation Roadmap

### Phase 1: Quick Wins (Week 1-2)

**Theme:** Messaging + CTAs — highest conversion impact with lowest engineering effort.

| Feature | Effort | Expected Conversion Impact | Dependencies |
|---------|--------|---------------------------|--------------|
| 4.1 Hero Section Redesign (copy, CTA microcopy, social proof bar) | S | High | Copy finalized, user count available |
| 4.8 Footer CTA Banner | S | Medium | None — pure frontend |
| 4.9 Navigation Enhancement (sticky signup button, Pricing link) | S | Medium | Pricing page stub (can link to anchor on homepage initially) |
| 4.7 Content Hierarchy — Remove Sector Heatmap + Watchlist from logged-out view | S | Medium | Feature flag or auth-conditional rendering |
| 4.1 Hero Mini Dashboard — Off-hours fallback (show last close instead of $0.00) | M | Medium | Backend change: API should return last close when current price is 0 |

**Phase 1 Deliverables:**
- Rewritten hero with new headline, subheadline, social proof bar, updated CTAs
- Footer CTA banner
- Sticky nav signup button + Pricing nav link
- Sector Heatmap and Watchlist hidden for logged-out users
- Mini dashboard off-hours improvement

---

### Phase 2: Core UX Restructure (Week 3-5)

**Theme:** New sections + section reordering — the structural conversion improvement.

| Feature | Effort | Expected Conversion Impact | Dependencies |
|---------|--------|---------------------------|--------------|
| 4.2 AI Feature Spotlight Section | M | High | Design mockup, AI summary API stable |
| 4.4 Product Tour / Screenshots Section | M | High | Product screenshots captured and annotated |
| 4.5 Social Proof & Trust Section | S | Medium | Real metrics compiled; testimonials if available |
| 4.7 Full Content Hierarchy Restructure (reorder all sections) | M | High | Phase 1 complete, all new sections built |
| 4.3 Reddit Trends Differentiator Callout | M | Medium | Reddit Trends page accessible without login |
| 4.6 Pricing Teaser Section + /pricing page | L | High | Pricing strategy decided by stakeholders |

**Phase 2 Deliverables:**
- AI Feature Spotlight section live
- Product Tour section with 4 tabs and screenshots
- Social Proof section with metrics and trust signals
- Reddit Trends callout section
- Pricing teaser on homepage + full /pricing page
- Complete section reordering per 4.7

---

### Phase 3: Polish & Optimization (Week 6-8)

**Theme:** A/B testing, animation, and data-driven refinement.

| Feature | Effort | Expected Conversion Impact | Dependencies |
|---------|--------|---------------------------|--------------|
| A/B tests: Hero headline (Option A vs. B) | S | High (validates messaging) | Analytics tooling, minimum traffic threshold |
| A/B test: With vs. without Reddit Trends section | S | Medium (validates section value) | Phase 2 complete |
| Viewport animations (fade-in, section transitions) | S | Low | Design review |
| Accessibility audit and remediation | M | N/A (compliance) | WCAG tooling (axe, Lighthouse) |
| Testimonial collection and integration | M | Medium | User outreach, consent process |
| Analytics dashboard for homepage KPIs | M | N/A (measurement) | Analytics events from Phase 1-2 |
| Performance optimization (image optimization, lazy loading) | S | Low (indirect via bounce rate) | Lighthouse audit |

**Phase 3 Deliverables:**
- 2-3 A/B tests running with statistical significance monitoring
- Animations and micro-interactions implemented
- WCAG 2.1 AA compliance verified
- Real testimonials integrated (if collected)
- Homepage analytics dashboard tracking all 5 KPIs

---

## 8. Open Questions & Assumptions

### Open Questions

| # | Question | Stakeholder | Impact if Unanswered |
|---|----------|-------------|---------------------|
| 1 | **What is the current homepage-to-signup conversion rate?** Analytics baseline needed to measure improvement. | Engineering / Analytics | Cannot define realistic targets or measure success |
| 2 | **What is the pricing strategy?** What features are in Free vs. Pro? What does Pro cost? Is there a trial period for Pro? | Product / Business | Blocks Section 4.6 (Pricing). Can launch Phase 1 without this but Phase 2 is blocked |
| 3 | **What is the current registered user count?** Needed for social proof bar. If <500, consider alternative proof points (data coverage stats, uptime). | Engineering / Database | Determines whether to show user count or data coverage metrics |
| 4 | **Do we have real user testimonials or NPS data?** Fabricated testimonials destroy trust. If none exist, the Social Proof section should rely on data metrics only. | Marketing / Support | Determines Social Proof section design |
| 5 | **Is the Reddit Trends page accessible without login?** If it requires auth, the callout section CTA needs to lead to a preview or the signup page instead. | Engineering | Determines Reddit section CTA behavior |
| 6 | **What analytics tooling is in place?** Heatmaps, funnel analytics, and A/B testing require specific tools (PostHog, Mixpanel, LaunchDarkly, etc.). | Engineering | Blocks Phase 3 and all measurement |
| 7 | **What is the target audience priority?** Should we optimize primarily for Persona A (retail/beginner), B (active trader), or C (professional)? This affects messaging tone and feature emphasis. | Product / Business | Affects hero copy and section prioritization |
| 8 | **Is there budget for UX research?** The pre-implementation research plan (Section 3) requires recruiting participants and potentially tool subscriptions. | Product / Budget | Determines whether to skip or modify the research plan |

### Key Assumptions

| # | Assumption | Risk if Wrong |
|---|-----------|---------------|
| 1 | The primary conversion bottleneck is the homepage, not SEO/traffic or the signup flow itself. | If signup form friction is the real blocker, homepage changes won't move the needle — audit the signup funnel separately. |
| 2 | New visitors are the primary growth lever (vs. retention of existing users). | If retention is the bigger problem, resources should go to post-login experience instead. |
| 3 | The AI Market Summary, Reddit Trends, and IC Score are genuine differentiators that matter to target users. | If users don't care about these features, leading with them will miss the mark. UX research (Section 3) should validate this. |
| 4 | A "Free forever" tier exists or will exist — the CTA "No credit card required" assumes this. | If everything is trial-based with a hard paywall, all CTA microcopy and pricing section need revision. |
| 5 | Weekend and after-hours traffic is significant (retail investors browse evenings/weekends). | If nearly all traffic is during market hours, the $0.00 off-hours issue is less urgent. Analytics should confirm. |
| 6 | The current visitor volume is sufficient for A/B testing (need ~1,000 visitors per variant per test). | If traffic is <500/week, A/B tests will take months to reach significance. Consider qualitative methods instead. |
| 7 | Mobile traffic is a significant share (30%+). | If nearly all traffic is desktop, mobile-specific optimizations can be deprioritized. |
| 8 | The backend team can implement a "last close price" fallback for the market indices API within Phase 1 timeline. | If this requires significant backend work, the mini dashboard improvement moves to Phase 2. |

---

*End of PRD. This document should be reviewed by Engineering, Design, and Business stakeholders before implementation begins. UX Research (Section 3) should ideally complete before Phase 2 starts.*
