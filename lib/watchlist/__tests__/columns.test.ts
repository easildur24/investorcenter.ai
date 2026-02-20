import { WatchListItem } from '@/lib/api/watchlist';
import {
  ALL_COLUMNS,
  VIEW_PRESETS,
  getColumnsForView,
  getColumn,
  ViewPresetId,
  DEFAULT_VIEW,
  VIEW_STORAGE_KEY,
} from '../columns';

// ---------------------------------------------------------------------------
// Mock WatchListItem factory
// ---------------------------------------------------------------------------

const mockItem = (overrides: Partial<WatchListItem> = {}): WatchListItem => ({
  id: 'item-1',
  watch_list_id: 'wl-1',
  symbol: 'AAPL',
  name: 'Apple Inc.',
  exchange: 'NASDAQ',
  asset_type: 'stock',
  tags: [],
  added_at: '2026-01-01T00:00:00Z',
  display_order: 0,
  current_price: 185.5,
  price_change: 2.3,
  price_change_pct: 1.25,
  volume: 52_000_000,
  market_cap: 3_000_000_000_000,
  prev_close: 183.2,
  target_buy_price: 150.0,
  target_sell_price: 200.0,
  alert_count: 2,
  ic_score: 78.5,
  ic_rating: 'Strong Buy',
  value_score: 65,
  growth_score: 82,
  profitability_score: 71,
  financial_health_score: 55,
  momentum_score: 90,
  analyst_consensus_score: 73,
  insider_activity_score: 40,
  institutional_score: 68,
  news_sentiment_score: 60,
  technical_score: 85,
  sector_percentile: 88.5,
  lifecycle_stage: 'growth',
  pe_ratio: 32.5,
  pb_ratio: 18.2,
  ps_ratio: 28.1,
  roe: 35.2,
  roa: 18.7,
  gross_margin: 72.1,
  operating_margin: 54.3,
  net_margin: 48.9,
  debt_to_equity: 0.41,
  current_ratio: 4.17,
  revenue_growth: 22.4,
  eps_growth: 18.1,
  dividend_yield: 0.55,
  payout_ratio: 15.2,
  reddit_rank: 5,
  reddit_mentions: 120,
  reddit_popularity: 85.5,
  reddit_trend: 'rising',
  reddit_rank_change: -3,
  ...overrides,
});

// ---------------------------------------------------------------------------
// ALL_COLUMNS tests
// ---------------------------------------------------------------------------

describe('ALL_COLUMNS', () => {
  it('has unique ids', () => {
    const ids = ALL_COLUMNS.map((c) => c.id);
    expect(new Set(ids).size).toBe(ids.length);
  });

  it('every column has a getValue function', () => {
    ALL_COLUMNS.forEach((col) => {
      expect(typeof col.getValue).toBe('function');
    });
  });

  it('every column has required fields', () => {
    ALL_COLUMNS.forEach((col) => {
      expect(col.id).toBeTruthy();
      expect(col.label).toBeTruthy();
      expect(col.type).toBeTruthy();
      expect(['left', 'right', 'center']).toContain(col.align);
      expect(typeof col.sortable).toBe('boolean');
      expect(typeof col.premium).toBe('boolean');
    });
  });

  it('includes actions column', () => {
    const actionsCol = ALL_COLUMNS.find((c) => c.id === 'actions');
    expect(actionsCol).toBeDefined();
    expect(actionsCol!.type).toBe('actions');
    expect(actionsCol!.sortable).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// getValue extraction tests
// ---------------------------------------------------------------------------

describe('getValue extractors', () => {
  const item = mockItem();

  it('symbol column extracts symbol string', () => {
    const col = getColumn('symbol')!;
    expect(col.getValue(item)).toBe('AAPL');
  });

  it('name column extracts name', () => {
    const col = getColumn('name')!;
    expect(col.getValue(item)).toBe('Apple Inc.');
  });

  it('current_price extracts price number', () => {
    const col = getColumn('current_price')!;
    expect(col.getValue(item)).toBe(185.5);
  });

  it('price_change extracts price_change_pct for sorting', () => {
    const col = getColumn('price_change')!;
    expect(col.getValue(item)).toBe(1.25);
  });

  it('ic_score extracts numeric score', () => {
    const col = getColumn('ic_score')!;
    expect(col.getValue(item)).toBe(78.5);
  });

  it('ic_score returns null when unavailable', () => {
    const col = getColumn('ic_score')!;
    expect(col.getValue(mockItem({ ic_score: null }))).toBeNull();
  });

  it('pe_ratio extracts fundamentals value', () => {
    const col = getColumn('pe_ratio')!;
    expect(col.getValue(item)).toBe(32.5);
  });

  it('reddit_rank extracts integer', () => {
    const col = getColumn('reddit_rank')!;
    expect(col.getValue(item)).toBe(5);
  });

  it('reddit_trend extracts trend string', () => {
    const col = getColumn('reddit_trend')!;
    expect(col.getValue(item)).toBe('rising');
  });

  it('alert_count extracts count', () => {
    const col = getColumn('alert_count')!;
    expect(col.getValue(item)).toBe(2);
  });

  it('volume extracts large number', () => {
    const col = getColumn('volume')!;
    expect(col.getValue(item)).toBe(52_000_000);
  });

  it('getValue returns undefined for missing optional fields', () => {
    const col = getColumn('reddit_rank')!;
    expect(col.getValue(mockItem({ reddit_rank: undefined }))).toBeUndefined();
  });

  it('actions column getValue returns null', () => {
    const col = getColumn('actions')!;
    expect(col.getValue(item)).toBeNull();
  });
});

// ---------------------------------------------------------------------------
// getColumn helper tests
// ---------------------------------------------------------------------------

describe('getColumn', () => {
  it('returns column for valid id', () => {
    const col = getColumn('symbol');
    expect(col).toBeDefined();
    expect(col!.id).toBe('symbol');
  });

  it('returns undefined for invalid id', () => {
    expect(getColumn('nonexistent')).toBeUndefined();
  });
});

// ---------------------------------------------------------------------------
// VIEW_PRESETS tests
// ---------------------------------------------------------------------------

describe('VIEW_PRESETS', () => {
  it('has 7 presets', () => {
    expect(VIEW_PRESETS).toHaveLength(7);
  });

  it('has unique ids', () => {
    const ids = VIEW_PRESETS.map((p) => p.id);
    expect(new Set(ids).size).toBe(ids.length);
  });

  it('every preset has required fields', () => {
    VIEW_PRESETS.forEach((preset) => {
      expect(preset.id).toBeTruthy();
      expect(preset.label).toBeTruthy();
      expect(preset.description).toBeTruthy();
      expect(preset.columnIds.length).toBeGreaterThan(0);
      expect(typeof preset.premium).toBe('boolean');
    });
  });

  it('every preset includes symbol as first column', () => {
    VIEW_PRESETS.forEach((preset) => {
      expect(preset.columnIds[0]).toBe('symbol');
    });
  });

  it('every preset includes actions as last column', () => {
    VIEW_PRESETS.forEach((preset) => {
      expect(preset.columnIds[preset.columnIds.length - 1]).toBe('actions');
    });
  });

  it('every preset column id exists in ALL_COLUMNS', () => {
    VIEW_PRESETS.forEach((preset) => {
      preset.columnIds.forEach((colId) => {
        const col = getColumn(colId);
        expect(col).toBeDefined();
      });
    });
  });

  it('general preset is not premium', () => {
    const general = VIEW_PRESETS.find((p) => p.id === 'general');
    expect(general!.premium).toBe(false);
  });

  it('fundamentals preset is premium', () => {
    const fundamentals = VIEW_PRESETS.find((p) => p.id === 'fundamentals');
    expect(fundamentals!.premium).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// getColumnsForView tests
// ---------------------------------------------------------------------------

describe('getColumnsForView', () => {
  it('returns correct columns for general view', () => {
    const cols = getColumnsForView('general');
    const ids = cols.map((c) => c.id);
    expect(ids).toContain('symbol');
    expect(ids).toContain('name');
    expect(ids).toContain('current_price');
    expect(ids).toContain('price_change');
    expect(ids).toContain('ic_score');
    expect(ids).toContain('actions');
  });

  it('always starts with symbol and ends with actions', () => {
    const presetIds: ViewPresetId[] = [
      'general',
      'performance',
      'fundamentals',
      'dividends',
      'ic_score',
      'social',
      'compact',
    ];
    presetIds.forEach((presetId) => {
      const cols = getColumnsForView(presetId);
      expect(cols[0].id).toBe('symbol');
      expect(cols[cols.length - 1].id).toBe('actions');
    });
  });

  it('fundamentals view includes valuation and profitability columns', () => {
    const cols = getColumnsForView('fundamentals');
    const ids = cols.map((c) => c.id);
    expect(ids).toContain('pe_ratio');
    expect(ids).toContain('pb_ratio');
    expect(ids).toContain('roe');
    expect(ids).toContain('roa');
    expect(ids).toContain('debt_to_equity');
    expect(ids).toContain('current_ratio');
    expect(ids).toContain('revenue_growth');
  });

  it('social view includes reddit columns', () => {
    const cols = getColumnsForView('social');
    const ids = cols.map((c) => c.id);
    expect(ids).toContain('reddit_rank');
    expect(ids).toContain('reddit_mentions');
    expect(ids).toContain('reddit_popularity');
    expect(ids).toContain('reddit_trend');
    expect(ids).toContain('reddit_rank_change');
  });

  it('ic_score view includes all sub-factor scores', () => {
    const cols = getColumnsForView('ic_score');
    const ids = cols.map((c) => c.id);
    expect(ids).toContain('ic_score');
    expect(ids).toContain('ic_rating');
    expect(ids).toContain('value_score');
    expect(ids).toContain('growth_score');
    expect(ids).toContain('profitability_score');
    expect(ids).toContain('financial_health_score');
    expect(ids).toContain('momentum_score');
    expect(ids).toContain('technical_score');
    expect(ids).toContain('sector_percentile');
  });

  it('dividends view includes yield and payout', () => {
    const cols = getColumnsForView('dividends');
    const ids = cols.map((c) => c.id);
    expect(ids).toContain('dividend_yield');
    expect(ids).toContain('payout_ratio');
  });

  it('compact view has minimal columns', () => {
    const cols = getColumnsForView('compact');
    // symbol + price + change + ic_score + actions = 5
    expect(cols).toHaveLength(5);
  });

  it('falls back to general for invalid preset id', () => {
    const cols = getColumnsForView('nonexistent' as ViewPresetId);
    const generalCols = getColumnsForView('general');
    expect(cols.map((c) => c.id)).toEqual(generalCols.map((c) => c.id));
  });

  it('returns full ColumnDefinition objects, not just ids', () => {
    const cols = getColumnsForView('general');
    cols.forEach((col) => {
      expect(col.id).toBeTruthy();
      expect(col.label).toBeTruthy();
      expect(col.type).toBeTruthy();
      expect(typeof col.getValue).toBe('function');
    });
  });
});

// ---------------------------------------------------------------------------
// Constants tests
// ---------------------------------------------------------------------------

describe('constants', () => {
  it('DEFAULT_VIEW is general', () => {
    expect(DEFAULT_VIEW).toBe('general');
  });

  it('VIEW_STORAGE_KEY is defined', () => {
    expect(VIEW_STORAGE_KEY).toBe('watchlist-view-preset');
  });
});
