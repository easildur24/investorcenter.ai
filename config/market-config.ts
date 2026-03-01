/**
 * Market configuration — drives Market Overview tabs and navigation.
 * Config-driven design allows adding new asset categories without code changes.
 */

import type { MarketOverviewTab } from '@/lib/types/market';

// ─── Market Overview Tabs ────────────────────────────────────────────────────

export const MARKET_OVERVIEW_TABS: MarketOverviewTab[] = [
  {
    id: 'us_indices',
    label: 'US Indices',
    category: 'us_indices',
    description: 'Major US market indices',
  },
  {
    id: 'us_etfs',
    label: 'Major ETFs',
    category: 'us_etfs',
    description: 'Most-traded US ETFs',
  },
];

// ─── Navigation Market Items ─────────────────────────────────────────────────

export interface NavMarketItem {
  label: string;
  href: string;
  description?: string;
}

export const NAV_MARKET_ITEMS: NavMarketItem[] = [
  { label: 'Overview', href: '/markets', description: 'Market dashboard' },
  { label: 'Indices', href: '/markets/indices', description: 'US & global indices' },
  { label: 'ETFs', href: '/markets/etfs', description: 'Most-traded ETFs' },
  { label: 'Sectors', href: '/markets/sectors', description: 'Sector performance' },
];

// ─── Default Indices ─────────────────────────────────────────────────────────

export const DEFAULT_INDEX_SYMBOLS = ['I:SPX', 'I:DJI', 'I:COMP', 'I:RUT', 'I:VIX'];

export const INDEX_DISPLAY_NAMES: Record<string, string> = {
  'I:SPX': 'S&P 500',
  'I:DJI': 'Dow Jones',
  'I:COMP': 'NASDAQ',
  'I:RUT': 'Russell 2000',
  'I:VIX': 'VIX',
  // ETF fallbacks
  SPY: 'S&P 500',
  DIA: 'Dow Jones',
  QQQ: 'NASDAQ',
  IWM: 'Russell 2000',
  VIXY: 'VIX',
};
