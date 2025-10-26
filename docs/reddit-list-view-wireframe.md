# Reddit Trending Stocks - List View Design

## Navigation Integration

```
┌─────────────────────────────────────────────────────────────────┐
│  📊 InvestorCenter    [Home]  [Crypto]  [Reddit Trends]  ...   │
│                                          ^^^^^^^^^^^^^^          │
│                                          NEW SECTION             │
└─────────────────────────────────────────────────────────────────┘
```

## Page Layout: `/reddit` or `/reddit/trending`

```
┌─────────────────────────────────────────────────────────────────────────┐
│                                                                           │
│  🔥 Trending on Reddit                                                   │
│  Most mentioned stocks across r/wallstreetbets, r/stocks, r/investing   │
│                                                                           │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  Time Range:  [● Today]  [ 7 Days]  [ 14 Days]  [ 30 Days]      │   │
│  │  Last updated: 2 minutes ago  🔄                                 │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                           │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  #   TICKER    COMPANY              RANK   MENTIONS   SCORE      │   │
│  ├──────────────────────────────────────────────────────────────────┤   │
│  │  1   BYND      Beyond Meat          #1↑3   363       🔥 100.0    │   │
│  │      ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━   │   │
│  │                                                                   │   │
│  │  2   ASST      Asset Entities       #2↑5   185       🔥 100.0    │   │
│  │      ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━   │   │
│  │                                                                   │   │
│  │  3   SPY       SPDR S&P 500 ETF     #3→    69        📈 57.8     │   │
│  │      ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━         │   │
│  │                                                                   │   │
│  │  4   DTE       DTE Energy Co.       #4↑1   47        📈 48.3     │   │
│  │      ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━                 │   │
│  │                                                                   │   │
│  │  5   DFLI      Dragonfly Energy     #5↓2   46        📈 48.3     │   │
│  │      ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━                 │   │
│  │                                                                   │   │
│  │  6   GME       GameStop Corp.       #6↑2   45        📈 47.1     │   │
│  │      ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━                   │   │
│  │                                                                   │   │
│  │  7   TSLA      Tesla, Inc.          #7↓1   40        📈 44.6     │   │
│  │      ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━                     │   │
│  │                                                                   │   │
│  │  8   RR        Rolls-Royce          #8→    37        📈 42.9     │   │
│  │      ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━                         │   │
│  │                                                                   │   │
│  │  9   AMZN      Amazon.com Inc.      #9↑4   35        📈 42.1     │   │
│  │      ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━                         │   │
│  │                                                                   │   │
│  │  10  NVDA      NVIDIA Corp.         #10↓3  33        📈 40.8     │   │
│  │      ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━                           │   │
│  │                                                                   │   │
│  │  ... (show 20-50 items)                                          │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                           │
└─────────────────────────────────────────────────────────────────────────┘
```

## Mobile View (Simplified)

```
┌─────────────────────────────────┐
│  🔥 Trending on Reddit          │
│                                 │
│  [● Today]  [ 7D]  [ 30D]      │
│                                 │
│  ┌───────────────────────────┐ │
│  │ 1. BYND  #1↑3   🔥 100.0  │ │
│  │    Beyond Meat            │ │
│  │    363 mentions           │ │
│  │    ━━━━━━━━━━━━━━━━━━   │ │
│  └───────────────────────────┘ │
│                                 │
│  ┌───────────────────────────┐ │
│  │ 2. ASST  #2↑5   🔥 100.0  │ │
│  │    Asset Entities         │ │
│  │    185 mentions           │ │
│  │    ━━━━━━━━━━━━━━━━━━   │ │
│  └───────────────────────────┘ │
│                                 │
│  [... more items ...]           │
└─────────────────────────────────┘
```

## List Item Hover State

```
┌──────────────────────────────────────────────────────────────────┐
│  1   BYND      Beyond Meat          #1↑3   363       🔥 100.0    │
│      ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━   │
│      ↑ Background highlights on hover, cursor: pointer           │
│      ↑ Shows "Click to view details →" on right side             │
└──────────────────────────────────────────────────────────────────┘
```

## Component Structure

### 1. **Page** (`/app/reddit/page.tsx`)
```tsx
export default function RedditTrendingPage() {
  const [timeRange, setTimeRange] = useState<'1' | '7' | '14' | '30'>('7');
  const [data, setData] = useState<RedditHeatmapData[]>([]);

  // Fetch data when timeRange changes
  useEffect(() => {
    fetch(`/api/v1/reddit/heatmap?days=${timeRange}&top=50`)
      .then(res => res.json())
      .then(json => setData(json.data));
  }, [timeRange]);

  return (
    <div>
      <Header />
      <TimeRangeSelector value={timeRange} onChange={setTimeRange} />
      <TrendingList items={data} />
    </div>
  );
}
```

### 2. **TimeRangeSelector** (`components/reddit/TimeRangeSelector.tsx`)
```tsx
interface TimeRangeSelectorProps {
  value: '1' | '7' | '14' | '30';
  onChange: (value: '1' | '7' | '14' | '30') => void;
}

// Radio button group or tab-style buttons
// [● Today]  [ 7 Days]  [ 14 Days]  [ 30 Days]
```

### 3. **TrendingList** (`components/reddit/TrendingList.tsx`)
```tsx
interface TrendingListProps {
  items: RedditHeatmapData[];
  onItemClick?: (symbol: string) => void;
}

// Maps over items and renders TrendingListItem for each
```

### 4. **TrendingListItem** (`components/reddit/TrendingListItem.tsx`)
```tsx
interface TrendingListItemProps {
  rank: number;
  symbol: string;
  companyName: string;
  currentRank: number;
  rankChange: number;          // Positive = improved, negative = worsened
  mentions: number;
  popularityScore: number;
  trendDirection: 'rising' | 'falling' | 'stable';
  onClick?: () => void;
}

// Single row in the list
// Shows: Rank, Ticker, Company, Rank badge, Mentions, Score bar
```

## Visual Design Details

### Rank Change Indicators
```
↑3   Green  (rank improved - went from #4 to #1)
↓2   Red    (rank worsened - went from #3 to #5)
→    Gray   (stable, changed by 0-1)
```

### Popularity Score Display
```
🔥 90-100   Very hot (red fire emoji or red badge)
📈 70-89    Trending (chart emoji or yellow badge)
📊 50-69    Popular (bar chart emoji or blue badge)
💬 0-49     Mentioned (chat emoji or gray badge)
```

### Progress Bar (Optional)
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  (100.0)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━                  (57.8)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━                (48.3)
```

## Data Flow

```
1. User visits /reddit
2. Default: Fetch GET /api/v1/reddit/heatmap?days=7&top=50
3. Render TrendingList with 50 items
4. User clicks time range button (e.g., "Today")
5. Re-fetch: GET /api/v1/reddit/heatmap?days=1&top=50
6. Update list
7. User clicks on list item → Navigate to /ticker/:symbol
```

## Header Navigation Update

Add new navigation item in `/app/components/Header.tsx`:

```tsx
<nav>
  <Link href="/">Home</Link>
  <Link href="/crypto">Crypto</Link>
  <Link href="/reddit">Reddit Trends</Link>  {/* NEW */}
  <Link href="/login">Login</Link>
  <Link href="/signup">Get Started</Link>
</nav>
```

## Implementation Checklist (Week 3 - Simplified)

### Day 1-2: Core Components
- [ ] Create `/app/reddit/page.tsx`
- [ ] Create `TimeRangeSelector.tsx` component
- [ ] Create `TrendingList.tsx` container
- [ ] Create `TrendingListItem.tsx` component

### Day 3-4: Styling & Interactions
- [ ] Add hover states to list items
- [ ] Add rank change indicators (↑↓→)
- [ ] Add popularity score badges/bars
- [ ] Make list items clickable → navigate to ticker page

### Day 5: Navigation & Polish
- [ ] Add "Reddit Trends" to header navigation
- [ ] Add loading states
- [ ] Add empty state ("No data available")
- [ ] Add auto-refresh (every 5 minutes)
- [ ] Test responsive layout (mobile vs desktop)

### Day 6-7: Testing & Refinement
- [ ] Test with real data from production API
- [ ] Add error handling
- [ ] Performance optimization (memoization)
- [ ] Accessibility (keyboard navigation, ARIA labels)

## Sample Code Snippet

```tsx
// components/reddit/TrendingListItem.tsx
export function TrendingListItem({
  rank,
  symbol,
  companyName,
  currentRank,
  rankChange,
  mentions,
  popularityScore,
  onClick
}: TrendingListItemProps) {
  const getRankChangeIcon = () => {
    if (rankChange > 1) return <span className="text-green-600">↑{rankChange}</span>;
    if (rankChange < -1) return <span className="text-red-600">↓{Math.abs(rankChange)}</span>;
    return <span className="text-gray-400">→</span>;
  };

  const getScoreEmoji = () => {
    if (popularityScore >= 90) return '🔥';
    if (popularityScore >= 70) return '📈';
    if (popularityScore >= 50) return '📊';
    return '💬';
  };

  return (
    <div
      className="flex items-center justify-between p-4 hover:bg-gray-50 cursor-pointer border-b"
      onClick={onClick}
    >
      <div className="flex items-center gap-4 flex-1">
        <span className="text-gray-500 font-mono">{rank}</span>
        <div>
          <div className="font-bold">{symbol}</div>
          <div className="text-sm text-gray-600">{companyName}</div>
        </div>
      </div>

      <div className="flex items-center gap-4">
        <div className="flex items-center gap-1">
          <span className="text-sm">#{currentRank}</span>
          {getRankChangeIcon()}
        </div>

        <div className="text-sm text-gray-600">
          {mentions} mentions
        </div>

        <div className="flex items-center gap-2">
          <span>{getScoreEmoji()}</span>
          <span className="font-semibold">{popularityScore.toFixed(1)}</span>
        </div>
      </div>
    </div>
  );
}
```

## Advantages of List View vs Heatmap

1. **Simpler to implement** - Just a list, no complex grid layout
2. **More information per item** - Can show company name, exact numbers
3. **Better for mobile** - Scrollable list works great on small screens
4. **Easier to scan** - Natural top-to-bottom reading
5. **Clickable rows** - Entire row is clickable, not just a small cell
6. **Extensible** - Easy to add columns (upvotes, trend, etc.)

## Future Enhancements

- [ ] Sorting (by mentions, score, rank change)
- [ ] Search/filter by ticker
- [ ] Mini sparkline chart per row (7-day trend)
- [ ] Pagination or infinite scroll for 100+ items
- [ ] Export to CSV
- [ ] Comparison mode (Today vs 7 days ago side-by-side)

---

**This is the recommended starting point for Week 3 - much simpler than the full heatmap!**
