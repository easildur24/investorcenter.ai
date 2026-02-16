// Declarative column configuration for the screener results table.
// Each column maps a sort key to a label, alignment, and format function.

import type { ScreenerStock, ScreenerSortField } from '@/lib/types/screener';
import { safeToFixed, formatLargeNumber } from '@/lib/utils';

export interface ColumnDef {
  key: ScreenerSortField;
  label: string;
  align: 'left' | 'right';
  format: (stock: ScreenerStock) => string;
  colorize?: (stock: ScreenerStock) => string;
  defaultVisible?: boolean;
}

const STORAGE_KEY = 'ic_screener_columns';

// All available columns. `defaultVisible` marks which ones show by default.
export const ALL_COLUMNS: ColumnDef[] = [
  // Core identity
  { key: 'symbol', label: 'Symbol', align: 'left', format: (s) => s.symbol, defaultVisible: true },
  { key: 'name', label: 'Name', align: 'left', format: (s) => s.name, defaultVisible: true },

  // Market data
  { key: 'market_cap', label: 'Market Cap', align: 'right', format: (s) => s.market_cap != null ? formatLargeNumber(s.market_cap) : '—', defaultVisible: true },
  { key: 'price', label: 'Price', align: 'right', format: (s) => s.price != null ? `$${safeToFixed(s.price, 2)}` : '—', defaultVisible: true },

  // Valuation
  { key: 'pe_ratio', label: 'P/E', align: 'right', format: (s) => s.pe_ratio != null ? safeToFixed(s.pe_ratio, 1) : '—', defaultVisible: true },
  { key: 'pb_ratio', label: 'P/B', align: 'right', format: (s) => s.pb_ratio != null ? safeToFixed(s.pb_ratio, 1) : '—' },
  { key: 'ps_ratio', label: 'P/S', align: 'right', format: (s) => s.ps_ratio != null ? safeToFixed(s.ps_ratio, 1) : '—' },

  // Profitability
  {
    key: 'roe', label: 'ROE', align: 'right',
    format: (s) => s.roe != null ? `${safeToFixed(s.roe, 1)}%` : '—',
    colorize: (s) => s.roe != null ? (s.roe >= 0 ? 'text-ic-positive' : 'text-ic-negative') : '',
    defaultVisible: true,
  },
  {
    key: 'roa', label: 'ROA', align: 'right',
    format: (s) => s.roa != null ? `${safeToFixed(s.roa, 1)}%` : '—',
    colorize: (s) => s.roa != null ? (s.roa >= 0 ? 'text-ic-positive' : 'text-ic-negative') : '',
  },
  {
    key: 'gross_margin', label: 'Gross Margin', align: 'right',
    format: (s) => s.gross_margin != null ? `${safeToFixed(s.gross_margin, 1)}%` : '—',
  },
  {
    key: 'net_margin', label: 'Net Margin', align: 'right',
    format: (s) => s.net_margin != null ? `${safeToFixed(s.net_margin, 1)}%` : '—',
    colorize: (s) => s.net_margin != null ? (s.net_margin >= 0 ? 'text-ic-positive' : 'text-ic-negative') : '',
  },

  // Financial health
  { key: 'debt_to_equity', label: 'D/E', align: 'right', format: (s) => s.debt_to_equity != null ? safeToFixed(s.debt_to_equity, 2) : '—' },
  { key: 'current_ratio', label: 'Current Ratio', align: 'right', format: (s) => s.current_ratio != null ? safeToFixed(s.current_ratio, 2) : '—' },

  // Growth
  {
    key: 'revenue_growth', label: 'Rev Growth', align: 'right',
    format: (s) => s.revenue_growth != null ? `${s.revenue_growth >= 0 ? '+' : ''}${safeToFixed(s.revenue_growth, 1)}%` : '—',
    colorize: (s) => s.revenue_growth != null ? (s.revenue_growth >= 0 ? 'text-ic-positive' : 'text-ic-negative') : '',
    defaultVisible: true,
  },
  {
    key: 'eps_growth_yoy', label: 'EPS Growth', align: 'right',
    format: (s) => s.eps_growth_yoy != null ? `${s.eps_growth_yoy >= 0 ? '+' : ''}${safeToFixed(s.eps_growth_yoy, 1)}%` : '—',
    colorize: (s) => s.eps_growth_yoy != null ? (s.eps_growth_yoy >= 0 ? 'text-ic-positive' : 'text-ic-negative') : '',
  },

  // Dividends
  { key: 'dividend_yield', label: 'Div Yield', align: 'right', format: (s) => s.dividend_yield != null ? `${safeToFixed(s.dividend_yield, 2)}%` : '—', defaultVisible: true },
  { key: 'payout_ratio', label: 'Payout Ratio', align: 'right', format: (s) => s.payout_ratio != null ? `${safeToFixed(s.payout_ratio, 1)}%` : '—' },

  // Risk
  { key: 'beta', label: 'Beta', align: 'right', format: (s) => s.beta != null ? safeToFixed(s.beta, 2) : '—', defaultVisible: true },

  // Fair value
  {
    key: 'dcf_upside_percent', label: 'DCF Upside', align: 'right',
    format: (s) => s.dcf_upside_percent != null ? `${s.dcf_upside_percent >= 0 ? '+' : ''}${safeToFixed(s.dcf_upside_percent, 1)}%` : '—',
    colorize: (s) => s.dcf_upside_percent != null ? (s.dcf_upside_percent >= 0 ? 'text-ic-positive' : 'text-ic-negative') : '',
  },

  // IC Score (overall)
  { key: 'ic_score', label: 'IC Score', align: 'right', format: (s) => s.ic_score != null ? String(Math.round(s.ic_score)) : '—', defaultVisible: true },

  // IC Score sub-factors
  { key: 'value_score', label: 'Value', align: 'right', format: (s) => s.value_score != null ? String(Math.round(s.value_score)) : '—' },
  { key: 'growth_score', label: 'Growth', align: 'right', format: (s) => s.growth_score != null ? String(Math.round(s.growth_score)) : '—' },
  { key: 'profitability_score', label: 'Profitability', align: 'right', format: (s) => s.profitability_score != null ? String(Math.round(s.profitability_score)) : '—' },
  { key: 'financial_health_score', label: 'Fin. Health', align: 'right', format: (s) => s.financial_health_score != null ? String(Math.round(s.financial_health_score)) : '—' },
  { key: 'momentum_score', label: 'Momentum', align: 'right', format: (s) => s.momentum_score != null ? String(Math.round(s.momentum_score)) : '—' },
  { key: 'analyst_consensus_score', label: 'Analyst', align: 'right', format: (s) => s.analyst_consensus_score != null ? String(Math.round(s.analyst_consensus_score)) : '—' },
  { key: 'insider_activity_score', label: 'Insider', align: 'right', format: (s) => s.insider_activity_score != null ? String(Math.round(s.insider_activity_score)) : '—' },
  { key: 'institutional_score', label: 'Institutional', align: 'right', format: (s) => s.institutional_score != null ? String(Math.round(s.institutional_score)) : '—' },
  { key: 'news_sentiment_score', label: 'Sentiment', align: 'right', format: (s) => s.news_sentiment_score != null ? String(Math.round(s.news_sentiment_score)) : '—' },
  { key: 'technical_score', label: 'Technical', align: 'right', format: (s) => s.technical_score != null ? String(Math.round(s.technical_score)) : '—' },
];

export const DEFAULT_VISIBLE_KEYS = ALL_COLUMNS
  .filter(c => c.defaultVisible)
  .map(c => c.key);

/** Load visible column keys from localStorage, falling back to defaults. */
export function loadVisibleColumns(): ScreenerSortField[] {
  if (typeof window === 'undefined') return DEFAULT_VISIBLE_KEYS;
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      const parsed = JSON.parse(stored) as string[];
      // Validate all stored keys are still valid column keys
      const validKeys = new Set(ALL_COLUMNS.map(c => c.key));
      const filtered = parsed.filter(k => validKeys.has(k as ScreenerSortField)) as ScreenerSortField[];
      return filtered.length > 0 ? filtered : DEFAULT_VISIBLE_KEYS;
    }
  } catch {
    // ignore parse errors
  }
  return DEFAULT_VISIBLE_KEYS;
}

/** Save visible column keys to localStorage. */
export function saveVisibleColumns(keys: ScreenerSortField[]): void {
  if (typeof window === 'undefined') return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(keys));
  } catch {
    // ignore quota errors
  }
}
