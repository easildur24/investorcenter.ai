import { WatchListItem } from '@/lib/api/watchlist';

// ---------------------------------------------------------------------------
// Column type determines how a cell value is formatted and rendered.
// ---------------------------------------------------------------------------

export type ColumnType =
  | 'symbol' // Link to /ticker/:symbol
  | 'text' // Plain text
  | 'currency' // $XX.XX
  | 'percent' // XX.X%  (with positive/negative coloring)
  | 'percentile' // XX% neutral (no green/red — rank-style metric)
  | 'number' // Numeric with configurable decimals
  | 'integer' // Whole number
  | 'change' // +X.XX (+Y.YY%) with green/red color
  | 'score' // IC Score colored pill badge
  | 'rating' // IC rating text (Strong Buy, Buy, etc.)
  | 'trend' // Reddit trend arrow + text
  | 'badge' // Target alert badge
  | 'actions'; // Edit / Remove buttons

// ---------------------------------------------------------------------------
// Column definition — drives header rendering, cell rendering, and sorting.
// ---------------------------------------------------------------------------

export interface ColumnDefinition {
  /** Unique identifier; maps to WatchListItem field name where applicable. */
  id: string;
  /** Display label for the column header. */
  label: string;
  /** Controls how the cell is formatted. */
  type: ColumnType;
  /** Text alignment in header and cells. */
  align: 'left' | 'right' | 'center';
  /** Whether clicking the header triggers sort. */
  sortable: boolean;
  /** Future: gate behind premium tier (Phase 4). */
  premium: boolean;
  /** Number of decimal places for number/currency/percent types. */
  decimals?: number;
  /** Optional Tailwind width class (e.g. 'min-w-[120px]'). */
  width?: string;
  /** Extracts the raw sortable value from a WatchListItem. */
  getValue: (item: WatchListItem) => string | number | null | undefined;
}

// ---------------------------------------------------------------------------
// Complete column registry — every displayable column.
// ---------------------------------------------------------------------------

export const ALL_COLUMNS: ColumnDefinition[] = [
  // ── Identity ──────────────────────────────────────────────────────────
  {
    id: 'symbol',
    label: 'Symbol',
    type: 'symbol',
    align: 'left',
    sortable: true,
    premium: false,
    getValue: (i) => i.symbol,
  },
  {
    id: 'name',
    label: 'Name',
    type: 'text',
    align: 'left',
    sortable: true,
    premium: false,
    width: 'max-w-[200px]',
    getValue: (i) => i.name,
  },
  {
    id: 'exchange',
    label: 'Exchange',
    type: 'text',
    align: 'left',
    sortable: true,
    premium: false,
    getValue: (i) => i.exchange,
  },
  {
    id: 'asset_type',
    label: 'Type',
    type: 'text',
    align: 'left',
    sortable: true,
    premium: false,
    getValue: (i) => i.asset_type,
  },

  // ── Price & Market ────────────────────────────────────────────────────
  {
    id: 'current_price',
    label: 'Price',
    type: 'currency',
    align: 'right',
    sortable: true,
    premium: false,
    decimals: 2,
    getValue: (i) => i.current_price,
  },
  {
    id: 'price_change',
    label: 'Change',
    type: 'change',
    align: 'right',
    sortable: true,
    premium: false,
    // Sorting by percent change (not dollar change) — more meaningful across
    // different price ranges. The cell renderer reads both price_change and
    // price_change_pct directly from the item to display "$X.XX (Y.YY%)".
    getValue: (i) => i.price_change_pct,
  },
  {
    id: 'volume',
    label: 'Volume',
    type: 'number',
    align: 'right',
    sortable: true,
    premium: false,
    decimals: 0,
    getValue: (i) => i.volume,
  },
  {
    id: 'prev_close',
    label: 'Prev Close',
    type: 'currency',
    align: 'right',
    sortable: true,
    premium: false,
    decimals: 2,
    getValue: (i) => i.prev_close,
  },
  {
    id: 'market_cap',
    label: 'Market Cap',
    type: 'number',
    align: 'right',
    sortable: true,
    premium: false,
    getValue: (i) => i.market_cap,
  },

  // ── Targets & Alerts ─────────────────────────────────────────────────
  {
    id: 'target_buy_price',
    label: 'Target Buy',
    type: 'currency',
    align: 'right',
    sortable: true,
    premium: false,
    decimals: 2,
    getValue: (i) => i.target_buy_price,
  },
  {
    id: 'target_sell_price',
    label: 'Target Sell',
    type: 'currency',
    align: 'right',
    sortable: true,
    premium: false,
    decimals: 2,
    getValue: (i) => i.target_sell_price,
  },
  {
    id: 'alert',
    label: 'Alert',
    type: 'badge',
    align: 'center',
    sortable: false,
    premium: false,
    getValue: (i) => i.alert_count,
  },
  {
    id: 'alert_count',
    label: 'Alerts',
    type: 'integer',
    align: 'right',
    sortable: true,
    premium: false,
    getValue: (i) => i.alert_count,
  },

  // ── IC Score ──────────────────────────────────────────────────────────
  {
    id: 'ic_score',
    label: 'IC Score',
    type: 'score',
    align: 'right',
    sortable: true,
    premium: false,
    getValue: (i) => i.ic_score,
  },
  {
    id: 'ic_rating',
    label: 'IC Rating',
    type: 'rating',
    align: 'center',
    sortable: true,
    premium: false,
    getValue: (i) => i.ic_rating,
  },
  {
    id: 'value_score',
    label: 'Value',
    type: 'integer',
    align: 'right',
    sortable: true,
    premium: true,
    getValue: (i) => i.value_score,
  },
  {
    id: 'growth_score',
    label: 'Growth',
    type: 'integer',
    align: 'right',
    sortable: true,
    premium: true,
    getValue: (i) => i.growth_score,
  },
  {
    id: 'profitability_score',
    label: 'Profit',
    type: 'integer',
    align: 'right',
    sortable: true,
    premium: true,
    getValue: (i) => i.profitability_score,
  },
  {
    id: 'financial_health_score',
    label: 'Fin Health',
    type: 'integer',
    align: 'right',
    sortable: true,
    premium: true,
    getValue: (i) => i.financial_health_score,
  },
  {
    id: 'momentum_score',
    label: 'Momentum',
    type: 'integer',
    align: 'right',
    sortable: true,
    premium: true,
    getValue: (i) => i.momentum_score,
  },
  {
    id: 'analyst_consensus_score',
    label: 'Analyst',
    type: 'integer',
    align: 'right',
    sortable: true,
    premium: true,
    getValue: (i) => i.analyst_consensus_score,
  },
  {
    id: 'insider_activity_score',
    label: 'Insider',
    type: 'integer',
    align: 'right',
    sortable: true,
    premium: true,
    getValue: (i) => i.insider_activity_score,
  },
  {
    id: 'institutional_score',
    label: 'Institutional',
    type: 'integer',
    align: 'right',
    sortable: true,
    premium: true,
    getValue: (i) => i.institutional_score,
  },
  {
    id: 'news_sentiment_score',
    label: 'Sentiment',
    type: 'integer',
    align: 'right',
    sortable: true,
    premium: true,
    getValue: (i) => i.news_sentiment_score,
  },
  {
    id: 'technical_score',
    label: 'Technical',
    type: 'integer',
    align: 'right',
    sortable: true,
    premium: true,
    getValue: (i) => i.technical_score,
  },
  {
    id: 'sector_percentile',
    label: 'Sector %ile',
    type: 'percentile',
    align: 'right',
    sortable: true,
    premium: true,
    decimals: 0,
    getValue: (i) => i.sector_percentile,
  },
  {
    id: 'lifecycle_stage',
    label: 'Lifecycle',
    type: 'text',
    align: 'center',
    sortable: true,
    premium: true,
    getValue: (i) => i.lifecycle_stage,
  },

  // ── Fundamentals — Valuation ──────────────────────────────────────────
  {
    id: 'pe_ratio',
    label: 'P/E',
    type: 'number',
    align: 'right',
    sortable: true,
    premium: false,
    decimals: 1,
    getValue: (i) => i.pe_ratio,
  },
  {
    id: 'pb_ratio',
    label: 'P/B',
    type: 'number',
    align: 'right',
    sortable: true,
    premium: false,
    decimals: 1,
    getValue: (i) => i.pb_ratio,
  },
  {
    id: 'ps_ratio',
    label: 'P/S',
    type: 'number',
    align: 'right',
    sortable: true,
    premium: false,
    decimals: 1,
    getValue: (i) => i.ps_ratio,
  },

  // ── Fundamentals — Profitability ──────────────────────────────────────
  {
    id: 'roe',
    label: 'ROE',
    type: 'percent',
    align: 'right',
    sortable: true,
    premium: false,
    decimals: 1,
    getValue: (i) => i.roe,
  },
  {
    id: 'roa',
    label: 'ROA',
    type: 'percent',
    align: 'right',
    sortable: true,
    premium: false,
    decimals: 1,
    getValue: (i) => i.roa,
  },
  {
    id: 'gross_margin',
    label: 'Gross Margin',
    type: 'percent',
    align: 'right',
    sortable: true,
    premium: false,
    decimals: 1,
    getValue: (i) => i.gross_margin,
  },
  {
    id: 'operating_margin',
    label: 'Op Margin',
    type: 'percent',
    align: 'right',
    sortable: true,
    premium: false,
    decimals: 1,
    getValue: (i) => i.operating_margin,
  },
  {
    id: 'net_margin',
    label: 'Net Margin',
    type: 'percent',
    align: 'right',
    sortable: true,
    premium: false,
    decimals: 1,
    getValue: (i) => i.net_margin,
  },

  // ── Fundamentals — Financial Health ───────────────────────────────────
  {
    id: 'debt_to_equity',
    label: 'D/E',
    type: 'number',
    align: 'right',
    sortable: true,
    premium: false,
    decimals: 2,
    getValue: (i) => i.debt_to_equity,
  },
  {
    id: 'current_ratio',
    label: 'Current Ratio',
    type: 'number',
    align: 'right',
    sortable: true,
    premium: false,
    decimals: 2,
    getValue: (i) => i.current_ratio,
  },

  // ── Fundamentals — Growth ─────────────────────────────────────────────
  {
    id: 'revenue_growth',
    label: 'Rev Growth',
    type: 'percent',
    align: 'right',
    sortable: true,
    premium: false,
    decimals: 1,
    getValue: (i) => i.revenue_growth,
  },
  {
    id: 'eps_growth',
    label: 'EPS Growth',
    type: 'percent',
    align: 'right',
    sortable: true,
    premium: false,
    decimals: 1,
    getValue: (i) => i.eps_growth,
  },

  // ── Fundamentals — Dividends ──────────────────────────────────────────
  {
    id: 'dividend_yield',
    label: 'Div Yield',
    type: 'percent',
    align: 'right',
    sortable: true,
    premium: false,
    decimals: 2,
    getValue: (i) => i.dividend_yield,
  },
  {
    id: 'payout_ratio',
    label: 'Payout Ratio',
    type: 'percent',
    align: 'right',
    sortable: true,
    premium: false,
    decimals: 1,
    getValue: (i) => i.payout_ratio,
  },

  // ── Reddit / Social ───────────────────────────────────────────────────
  {
    id: 'reddit_rank',
    label: 'Reddit Rank',
    type: 'integer',
    align: 'right',
    sortable: true,
    premium: false,
    getValue: (i) => i.reddit_rank,
  },
  {
    id: 'reddit_mentions',
    label: 'Mentions',
    type: 'integer',
    align: 'right',
    sortable: true,
    premium: false,
    getValue: (i) => i.reddit_mentions,
  },
  {
    id: 'reddit_popularity',
    label: 'Popularity',
    type: 'number',
    align: 'right',
    sortable: true,
    premium: false,
    decimals: 1,
    getValue: (i) => i.reddit_popularity,
  },
  {
    id: 'reddit_trend',
    label: 'Trend',
    type: 'trend',
    align: 'center',
    sortable: true,
    premium: false,
    getValue: (i) => i.reddit_trend,
  },
  {
    id: 'reddit_rank_change',
    label: 'Rank Change',
    type: 'integer',
    align: 'right',
    sortable: true,
    premium: false,
    getValue: (i) => i.reddit_rank_change,
  },

  // ── Performance (TODO: backend doesn't provide these yet) ─────────────
  // { id: 'perf_1w', label: '1W', type: 'percent', ... },
  // { id: 'perf_1m', label: '1M', type: 'percent', ... },
  // { id: 'perf_3m', label: '3M', type: 'percent', ... },
  // { id: 'perf_6m', label: '6M', type: 'percent', ... },
  // { id: 'perf_ytd', label: 'YTD', type: 'percent', ... },
  // { id: 'perf_1y', label: '1Y', type: 'percent', ... },

  // ── Meta ──────────────────────────────────────────────────────────────
  {
    id: 'actions',
    label: 'Actions',
    type: 'actions',
    align: 'center',
    sortable: false,
    premium: false,
    getValue: () => null,
  },
];

// Lookup map for O(1) column access by id
const COLUMN_MAP = new Map(ALL_COLUMNS.map((col) => [col.id, col]));

/** Look up a single column definition by id. */
export function getColumn(id: string): ColumnDefinition | undefined {
  return COLUMN_MAP.get(id);
}

// ---------------------------------------------------------------------------
// View presets — each defines which columns to display and in what order.
// ---------------------------------------------------------------------------

export type ViewPresetId =
  | 'general'
  | 'performance'
  | 'fundamentals'
  | 'dividends'
  | 'ic_score'
  | 'social'
  | 'compact';

export interface ViewPreset {
  id: ViewPresetId;
  label: string;
  description: string;
  columnIds: string[];
  /** Future: gate entire view behind premium tier (Phase 4). */
  premium: boolean;
}

export const VIEW_PRESETS: ViewPreset[] = [
  {
    id: 'general',
    label: 'General',
    description: 'Price, change, targets, IC Score',
    columnIds: [
      'symbol',
      'name',
      'current_price',
      'price_change',
      'target_buy_price',
      'target_sell_price',
      'alert',
      'ic_score',
      'actions',
    ],
    premium: false,
  },
  {
    id: 'performance',
    label: 'Performance',
    description: 'Price, volume, market cap',
    // TODO: Add perf_1w, perf_1m, perf_3m, perf_6m, perf_ytd, perf_1y when backend supports them
    columnIds: [
      'symbol',
      'name',
      'current_price',
      'price_change',
      'volume',
      'market_cap',
      'actions',
    ],
    premium: false,
  },
  {
    id: 'fundamentals',
    label: 'Fundamentals',
    description: 'Valuation, profitability, financial health',
    columnIds: [
      'symbol',
      'name',
      'current_price',
      'pe_ratio',
      'pb_ratio',
      'ps_ratio',
      'roe',
      'roa',
      'gross_margin',
      'net_margin',
      'debt_to_equity',
      'current_ratio',
      'revenue_growth',
      'eps_growth',
      'actions',
    ],
    premium: true,
  },
  {
    id: 'dividends',
    label: 'Dividends',
    description: 'Yield, payout ratio, growth',
    columnIds: [
      'symbol',
      'name',
      'current_price',
      'dividend_yield',
      'payout_ratio',
      'pe_ratio',
      'roe',
      'eps_growth',
      'actions',
    ],
    premium: true,
  },
  {
    id: 'ic_score',
    label: 'IC Score',
    description: 'IC Score breakdown by factor',
    columnIds: [
      'symbol',
      'name',
      'ic_score',
      'ic_rating',
      'value_score',
      'growth_score',
      'profitability_score',
      'financial_health_score',
      'momentum_score',
      'technical_score',
      'sector_percentile',
      'actions',
    ],
    premium: true,
  },
  {
    id: 'social',
    label: 'Social',
    description: 'Reddit rank, mentions, trends',
    columnIds: [
      'symbol',
      'name',
      'current_price',
      'price_change',
      'reddit_rank',
      'reddit_mentions',
      'reddit_popularity',
      'reddit_trend',
      'reddit_rank_change',
      'actions',
    ],
    premium: false,
  },
  {
    id: 'compact',
    label: 'Compact',
    description: 'Minimal columns for quick glance',
    columnIds: ['symbol', 'current_price', 'price_change', 'ic_score', 'actions'],
    premium: false,
  },
];

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Resolve a view preset's columnIds to full ColumnDefinition objects. */
export function getColumnsForView(presetId: ViewPresetId): ColumnDefinition[] {
  const preset = VIEW_PRESETS.find((p) => p.id === presetId);
  if (!preset) return getColumnsForView('general');

  const cols: ColumnDefinition[] = [];
  for (const colId of preset.columnIds) {
    const col = COLUMN_MAP.get(colId);
    if (col) cols.push(col);
  }
  return cols;
}

/** localStorage key for persisting the selected view preset. */
export const VIEW_STORAGE_KEY = 'watchlist-view-preset';

/** Default view preset. */
export const DEFAULT_VIEW: ViewPresetId = 'general';
