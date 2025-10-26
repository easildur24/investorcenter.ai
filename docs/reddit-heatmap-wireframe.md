# Reddit Heatmap Wireframe & Design Spec

## Page Layout: `/reddit/heatmap`

```
┌─────────────────────────────────────────────────────────────────────────┐
│  HEADER (Existing Navigation)                                           │
│  InvestorCenter.ai   [Home] [Markets] [Portfolio] [Reddit Trends]       │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│                                                                           │
│  📊 Reddit Trending Stocks                                               │
│  Real-time popularity from r/wallstreetbets, r/stocks, r/investing      │
│                                                                           │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  Filters & Controls                                              │   │
│  │  ┌─────────┐  ┌──────────┐  ┌──────────────────┐                │   │
│  │  │ 7 Days▾ │  │ Top 50 ▾ │  │ Last updated: 2m ago 🔄         │   │
│  │  └─────────┘  └──────────┘  └──────────────────┘                │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                           │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  Color Legend                                                     │   │
│  │  🟩 90-100 Hot   🟨 70-89 Popular   🟧 50-69 Trending            │   │
│  │  ⬜ 0-49 Mentioned                                                │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                           │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  HEATMAP GRID (Responsive - 5 columns on desktop, 2 on mobile)  │   │
│  │                                                                   │   │
│  │  ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐                       │   │
│  │  │BYND │ │ASST │ │ SPY │ │DFLI │ │ DTE │                       │   │
│  │  │#1   │ │#2   │ │#3   │ │#5   │ │#4   │                       │   │
│  │  │🟩100│ │🟩100│ │🟨 58│ │🟨 48│ │🟨 48│  ← Popularity Score   │   │
│  │  │⬆ 3  │ │⬆ 5  │ │━ 0  │ │⬇ 2  │ │⬆ 1  │  ← Rank Change       │   │
│  │  └─────┘ └─────┘ └─────┘ └─────┘ └─────┘                       │   │
│  │                                                                   │   │
│  │  ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐                       │   │
│  │  │ GME │ │TSLA │ │ RR  │ │AMZN │ │NVDA │                       │   │
│  │  │#6   │ │#7   │ │#8   │ │#9   │ │#10  │                       │   │
│  │  │🟨 47│ │🟨 45│ │🟨 43│ │🟨 42│ │🟨 41│                       │   │
│  │  │⬆ 2  │ │⬇ 1  │ │━ 0  │ │⬆ 4  │ │⬇ 3  │                       │   │
│  │  └─────┘ └─────┘ └─────┘ └─────┘ └─────┘                       │   │
│  │                                                                   │   │
│  │  [... more rows ...]                                             │   │
│  │                                                                   │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                           │
└─────────────────────────────────────────────────────────────────────────┘
```

## Heatmap Cell Detail (on hover)

```
┌───────────────────────────────────┐
│  Hover Tooltip                    │
├───────────────────────────────────┤
│  BYND - Beyond Meat               │
│                                   │
│  🏆 Rank: #1 (↑3)                 │
│  💬 Mentions: 363                 │
│  ⬆ Upvotes: 4,299                │
│  📊 Score: 100.0                  │
│  📈 Trend: Rising                 │
│                                   │
│  [View Details →]                 │
└───────────────────────────────────┘
```

## Interactive Cell (on click) - Modal or Side Panel

```
┌─────────────────────────────────────────────────────────────────┐
│  BYND - Beyond Meat Inc.                                    [✕] │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  📊 Reddit Popularity (Last 7 Days)                             │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  Line Chart: Popularity Score Over Time                  │  │
│  │  100 ┤                                                    │  │
│  │   80 ┤     ╭──╮                                          │  │
│  │   60 ┤   ╭─╯  ╰─╮                                        │  │
│  │   40 ┤  ╭╯      ╰╮                                       │  │
│  │   20 ┤╭─╯        ╰─╮                                     │  │
│  │    0 ┴───────────────────────────────────────────────    │  │
│  │      10/19  10/21  10/23  10/25                          │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  📈 Key Metrics                                                 │
│  ┌──────────────┬──────────────┬──────────────┐                │
│  │ Current Rank │ Avg Mentions │ Total Period │                │
│  │      #1      │   363/day    │   7 days     │                │
│  └──────────────┴──────────────┴──────────────┘                │
│                                                                  │
│  📊 Rank History                                                │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  Bar Chart: Daily Rank (Lower is Better)                 │  │
│  │  #1  █                                                    │  │
│  │  #5  ███                                                  │  │
│  │  #10 █████                                                │  │
│  │      10/19  10/21  10/23  10/25                          │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  [View Full Stock Details →]                                    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Component Breakdown

### 1. **Page Container** (`/app/reddit/heatmap/page.tsx`)
- Layout wrapper
- Fetches data from API
- Manages state (filters, selected ticker)

### 2. **Filter Controls** (`components/reddit/HeatmapFilters.tsx`)
```tsx
interface HeatmapFilters {
  days: number;        // 1, 7, 14, 30
  limit: number;       // 10, 20, 50, 100
  onFilterChange: (filters: { days: number; limit: number }) => void;
}
```

### 3. **Color Legend** (`components/reddit/HeatmapLegend.tsx`)
```tsx
// Static component showing color scale
// Green (90-100), Yellow (70-89), Orange (50-69), Gray (0-49)
```

### 4. **Main Heatmap Grid** (`components/reddit/TickerHeatmap.tsx`)
```tsx
interface TickerHeatmapProps {
  data: RedditHeatmapData[];
  days: number;
  onTickerClick: (symbol: string) => void;
}
```

### 5. **Heatmap Cell** (`components/reddit/HeatmapCell.tsx`)
```tsx
interface HeatmapCellProps {
  symbol: string;
  rank: number;
  popularityScore: number;
  rankChange?: number;      // Change from previous period
  trendDirection: 'rising' | 'falling' | 'stable';
  mentions: number;
  upvotes: number;
  onClick: () => void;
}
```

### 6. **Hover Tooltip** (`components/reddit/HeatmapTooltip.tsx`)
```tsx
interface HeatmapTooltipProps {
  symbol: string;
  companyName: string;
  rank: number;
  rankChange: number;
  mentions: number;
  upvotes: number;
  score: number;
  trend: string;
}
```

### 7. **Ticker Detail Modal** (`components/reddit/TickerDetailModal.tsx`)
```tsx
interface TickerDetailModalProps {
  symbol: string;
  isOpen: boolean;
  onClose: () => void;
  historyDays: number;
}
// Fetches /api/v1/reddit/ticker/:symbol/history
// Shows line chart + metrics
```

## Color Scheme

### Popularity Score → Background Color
```css
100-90:  bg-green-500    text-white   /* Hot */
89-70:   bg-yellow-400   text-gray-900 /* Popular */
69-50:   bg-orange-400   text-gray-900 /* Trending */
49-0:    bg-gray-200     text-gray-700 /* Mentioned */
```

### Trend Indicators
```
⬆ Rising:   text-green-600  (rank improved by 5+)
⬇ Falling:  text-red-600    (rank worsened by 5+)
━ Stable:   text-gray-500   (rank changed by < 5)
```

## Responsive Breakpoints

```css
/* Mobile: 2 columns */
@media (max-width: 640px) {
  grid-template-columns: repeat(2, 1fr);
  gap: 0.5rem;
}

/* Tablet: 3 columns */
@media (min-width: 641px) and (max-width: 1024px) {
  grid-template-columns: repeat(3, 1fr);
  gap: 0.75rem;
}

/* Desktop: 5 columns */
@media (min-width: 1025px) {
  grid-template-columns: repeat(5, 1fr);
  gap: 1rem;
}
```

## Data Flow

```
1. User visits /reddit/heatmap
2. Page fetches: GET /api/v1/reddit/heatmap?days=7&top=50
3. TickerHeatmap component renders grid of HeatmapCell components
4. On hover: Show HeatmapTooltip with details
5. On click: Open TickerDetailModal
   └─> Fetch: GET /api/v1/reddit/ticker/:symbol/history?days=7
   └─> Render charts and metrics
6. User can filter by days/limit → Re-fetch data
```

## User Interactions

### Primary Actions:
- **Hover over cell**: Show tooltip with rank, mentions, score
- **Click cell**: Open modal with 7-day history charts
- **Change filters**: Update heatmap data (days: 1/7/14/30, limit: 10/20/50/100)
- **Click "View Details"**: Navigate to `/ticker/:symbol` page

### Auto-refresh:
- Poll API every 5 minutes for updated data
- Show "Last updated: Xm ago" timestamp
- Manual refresh button

## Accessibility

- Keyboard navigation (Tab through cells, Enter to open)
- ARIA labels for screen readers
- Color-blind friendly (use patterns + text, not just color)
- Focus indicators on interactive elements

## Performance Considerations

- Virtualize grid if showing 100+ cells
- Lazy load TickerDetailModal
- Cache API responses (5 min TTL)
- Debounce filter changes
- Use React.memo for HeatmapCell to prevent unnecessary re-renders

## Future Enhancements (Post-MVP)

- [ ] Compare view (side-by-side heatmaps for different periods)
- [ ] Export to PNG/CSV
- [ ] Share specific heatmap via URL
- [ ] Filter by subreddit (r/wallstreetbets only, etc.)
- [ ] Alert notifications for specific tickers entering top 10
- [ ] Mobile app swipe gestures

---

**Design Philosophy**: Clean, data-dense, instantly readable. Users should be able to scan the heatmap in 3 seconds and identify the hottest stocks. Click for details only if interested.
