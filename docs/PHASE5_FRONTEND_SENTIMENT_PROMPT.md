# Phase 5: Frontend Components - Social Media Sentiment

## Overview
Build React components to display social media sentiment data for stock tickers. The backend API is already deployed at `/api/v1/sentiment/*`.

## File Structure
```
lib/
├── types/
│   └── sentiment.ts          # TypeScript interfaces
├── api.ts                    # Add sentiment methods to existing API client

components/
├── sentiment/
│   ├── SentimentGauge.tsx           # Visual score gauge (-1 to +1)
│   ├── SentimentBreakdownBar.tsx    # Horizontal stacked bar chart
│   ├── SentimentHistoryChart.tsx    # Line chart over time
│   ├── SentimentCard.tsx            # Summary card for ticker pages
│   ├── TrendingTickersList.tsx      # Trending tickers table
│   └── PostsList.tsx                # Sortable posts list

app/
├── sentiment/
│   └── page.tsx              # Standalone sentiment dashboard
├── ticker/[symbol]/
│   └── page.tsx              # Add SentimentCard to existing ticker page
```

## 1. TypeScript Types (`lib/types/sentiment.ts`)

```typescript
// Matches backend/models/social_sentiment.go exactly

export interface SentimentBreakdown {
  bullish: number;  // 0-100 percentage
  bearish: number;  // 0-100 percentage
  neutral: number;  // 0-100 percentage
}

export interface SubredditCount {
  subreddit: string;
  count: number;
}

// GET /api/v1/sentiment/:ticker
export interface SentimentResponse {
  ticker: string;
  company_name?: string;
  score: number;                    // -1 to +1
  label: 'bullish' | 'bearish' | 'neutral';
  breakdown: SentimentBreakdown;
  post_count_24h: number;
  post_count_7d: number;
  rank: number;
  rank_change: number;              // Positive = moved up, negative = moved down
  top_subreddits: SubredditCount[];
  last_updated: string;             // ISO 8601
}

// GET /api/v1/sentiment/:ticker/history?days=N
export interface SentimentHistoryPoint {
  date: string;       // YYYY-MM-DD
  score: number;      // -1 to +1
  post_count: number;
  bullish: number;    // Count (not percentage)
  bearish: number;
  neutral: number;
}

export interface SentimentHistoryResponse {
  ticker: string;
  period: string;     // "7d", "30d", "90d"
  history: SentimentHistoryPoint[];
}

// GET /api/v1/sentiment/trending?period=24h|7d&limit=N
export interface TrendingTicker {
  ticker: string;
  company_name?: string;
  score: number;          // -1 to +1
  label: 'bullish' | 'bearish' | 'neutral';
  post_count: number;
  mention_delta: number;  // % change from previous period
  rank: number;
}

export interface TrendingResponse {
  period: string;         // "24h" or "7d"
  tickers: TrendingTicker[];
  updated_at: string;     // ISO 8601
}

// GET /api/v1/sentiment/:ticker/posts?sort=recent|engagement|bullish|bearish&limit=N
export interface RepresentativePost {
  id: number;
  title: string;
  body_preview?: string;
  url: string;
  source: string;                   // "reddit"
  subreddit: string;
  upvotes: number;
  comment_count: number;
  award_count: number;
  sentiment: 'bullish' | 'bearish' | 'neutral';
  sentiment_confidence?: number;    // 0 to 1
  flair?: string;
  posted_at: string;                // ISO 8601
}

export interface RepresentativePostsResponse {
  ticker: string;
  posts: RepresentativePost[];
  total: number;
  sort: string;
}

export type PostSortOption = 'recent' | 'engagement' | 'bullish' | 'bearish';
export type TrendingPeriod = '24h' | '7d';
```

## 2. API Client Methods (`lib/api.ts`)

Add these methods to the existing `ApiClient` class:

```typescript
// Sentiment API
async getSentiment(ticker: string): Promise<SentimentResponse> {
  return this.get<SentimentResponse>(`/sentiment/${ticker.toUpperCase()}`);
}

async getSentimentHistory(ticker: string, days: number = 7): Promise<SentimentHistoryResponse> {
  return this.get<SentimentHistoryResponse>(`/sentiment/${ticker.toUpperCase()}/history?days=${days}`);
}

async getTrendingSentiment(period: TrendingPeriod = '24h', limit: number = 20): Promise<TrendingResponse> {
  return this.get<TrendingResponse>(`/sentiment/trending?period=${period}&limit=${limit}`);
}

async getSentimentPosts(
  ticker: string,
  sort: PostSortOption = 'recent',
  limit: number = 10
): Promise<RepresentativePostsResponse> {
  return this.get<RepresentativePostsResponse>(
    `/sentiment/${ticker.toUpperCase()}/posts?sort=${sort}&limit=${limit}`
  );
}
```

## 3. Components

### SentimentGauge.tsx
Visual gauge displaying sentiment score from -1 (bearish) to +1 (bullish).

```typescript
interface SentimentGaugeProps {
  score: number;        // -1 to +1
  label: string;        // "bullish", "bearish", "neutral"
  size?: 'sm' | 'md' | 'lg';
}
```

**Display logic:**
- Score of 0.5 → "50% Bullish"
- Score of -0.3 → "30% Bearish"
- Use green gradient for bullish, red for bearish, gray for neutral
- Show needle/marker at score position on -1 to +1 scale

### SentimentBreakdownBar.tsx
Horizontal stacked bar showing bullish/bearish/neutral distribution.

```typescript
interface SentimentBreakdownBarProps {
  breakdown: SentimentBreakdown;  // Values are already 0-100 percentages
  showLabels?: boolean;
}
```

**Colors:**
- Bullish: `#22c55e` (green-500)
- Bearish: `#ef4444` (red-500)
- Neutral: `#6b7280` (gray-500)

### SentimentCard.tsx
Summary card for ticker detail pages.

```typescript
interface SentimentCardProps {
  ticker: string;
}
```

**Features:**
- Fetch data using `getSentiment(ticker)`
- Display: score gauge, breakdown bar, post counts, rank with change indicator
- Rank change: ↑ green for positive, ↓ red for negative, — gray for zero
- Show top subreddits as badges
- Link to full posts list

### SentimentHistoryChart.tsx
Line chart showing sentiment over time.

```typescript
interface SentimentHistoryChartProps {
  ticker: string;
  days?: number;  // 7, 30, or 90
}
```

**Features:**
- Use recharts or similar library
- X-axis: dates, Y-axis: score (-1 to +1)
- Optional: overlay post count as bar chart
- Period selector tabs: 7D | 30D | 90D

### TrendingTickersList.tsx
Table of trending tickers by social activity.

```typescript
interface TrendingTickersListProps {
  period?: TrendingPeriod;
  limit?: number;
}
```

**Table columns:**
| Rank | Ticker | Company | Sentiment | Posts | Change |
|------|--------|---------|-----------|-------|--------|
| 1 | NVDA | NVIDIA Corp | 🟢 Bullish (0.45) | 127 | +52% |
| 2 | TSLA | Tesla Inc | 🔴 Bearish (-0.23) | 98 | -12% |

**Features:**
- Period toggle: 24h | 7d
- Click row to navigate to `/ticker/{symbol}`
- Color-code sentiment label
- Format mention_delta as percentage with +/- sign

### PostsList.tsx
Sortable list of representative posts.

```typescript
interface PostsListProps {
  ticker: string;
  initialSort?: PostSortOption;
}
```

**Features:**
- Sort dropdown: Recent | Most Engaged | Bullish | Bearish
- Each post card shows:
  - Title (link to URL)
  - Subreddit badge (r/wallstreetbets)
  - Sentiment badge with confidence if available
  - Engagement: ⬆️ {upvotes} 💬 {comment_count} 🏆 {award_count}
  - Relative time (e.g., "2 hours ago")
  - Flair tag if present
  - Body preview (truncated)

## 4. Sentiment Dashboard Page (`app/sentiment/page.tsx`)

Full-page sentiment dashboard with:
1. **Trending Section**: TrendingTickersList with period toggle
2. **Search**: Ticker search to view specific stock sentiment
3. **Selected Ticker Panel** (when ticker selected):
   - SentimentCard
   - SentimentHistoryChart
   - PostsList

## 5. Integration with Ticker Page

Add SentimentCard to existing ticker detail page (`app/ticker/[symbol]/page.tsx`):

```tsx
// In the ticker page component
<div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
  {/* Existing content */}
  <div className="lg:col-span-2">
    {/* Price chart, fundamentals, etc. */}
  </div>

  {/* New sentiment sidebar */}
  <div className="space-y-6">
    <SentimentCard ticker={symbol} />
    {/* Other sidebar content */}
  </div>
</div>
```

## 6. Styling Guidelines

- Use existing Tailwind classes from the codebase
- Match existing card styles (`bg-white dark:bg-gray-800 rounded-lg shadow`)
- Use existing color scheme for consistency
- Responsive: Stack on mobile, side-by-side on desktop

## 7. Error Handling

- Show skeleton loaders while fetching
- Handle empty state: "No sentiment data available for {ticker}"
- Handle API errors gracefully with retry option
- For tickers with no posts, show "No social media activity in the last 7 days"

## 8. API Endpoints Reference

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/sentiment/trending` | GET | Trending tickers (query: period, limit) |
| `/api/v1/sentiment/:ticker` | GET | Ticker sentiment summary |
| `/api/v1/sentiment/:ticker/history` | GET | Historical sentiment (query: days) |
| `/api/v1/sentiment/:ticker/posts` | GET | Representative posts (query: sort, limit) |
