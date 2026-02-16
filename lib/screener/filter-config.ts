// Declarative filter configuration for the screener sidebar.
// Each filter definition maps a UI control to URL query param keys.

import type { ScreenerApiParams } from '@/lib/types/screener';

export interface FilterDef {
  id: string;
  label: string;
  type: 'range' | 'multiselect';
  group: string;
  minKey?: keyof ScreenerApiParams;
  maxKey?: keyof ScreenerApiParams;
  options?: { value: string; label: string }[];
  step?: number;
  suffix?: string;
  placeholder?: { min: string; max: string };
}

export interface FilterGroup {
  id: string;
  label: string;
  defaultOpen?: boolean;
}

export const SECTORS = [
  'Technology',
  'Healthcare',
  'Financial Services',
  'Consumer Cyclical',
  'Consumer Defensive',
  'Industrials',
  'Energy',
  'Basic Materials',
  'Real Estate',
  'Communication Services',
  'Utilities',
];

export const FILTER_GROUPS: FilterGroup[] = [
  { id: 'general', label: 'General', defaultOpen: true },
  { id: 'ic_score', label: 'IC Score', defaultOpen: true },
  { id: 'valuation', label: 'Valuation' },
  { id: 'profitability', label: 'Profitability' },
  { id: 'financial_health', label: 'Financial Health' },
  { id: 'growth', label: 'Growth' },
  { id: 'dividends', label: 'Dividends' },
  { id: 'risk', label: 'Risk & Value' },
  { id: 'ic_factors', label: 'IC Score Factors' },
];

export const FILTER_DEFS: FilterDef[] = [
  // General
  {
    id: 'sector',
    label: 'Sector',
    type: 'multiselect',
    group: 'general',
    options: SECTORS.map(s => ({ value: s, label: s })),
  },
  {
    id: 'market_cap',
    label: 'Market Cap',
    type: 'range',
    group: 'general',
    minKey: 'market_cap_min',
    maxKey: 'market_cap_max',
    placeholder: { min: 'e.g. 1e9', max: 'e.g. 200e9' },
  },

  // IC Score
  {
    id: 'ic_score',
    label: 'IC Score (Overall)',
    type: 'range',
    group: 'ic_score',
    minKey: 'ic_score_min',
    maxKey: 'ic_score_max',
    step: 1,
    placeholder: { min: '0', max: '100' },
  },

  // Valuation
  {
    id: 'pe_ratio',
    label: 'P/E Ratio',
    type: 'range',
    group: 'valuation',
    minKey: 'pe_min',
    maxKey: 'pe_max',
    step: 1,
    placeholder: { min: '0', max: '100' },
  },
  {
    id: 'pb_ratio',
    label: 'P/B Ratio',
    type: 'range',
    group: 'valuation',
    minKey: 'pb_min',
    maxKey: 'pb_max',
    step: 0.1,
    placeholder: { min: '0', max: '20' },
  },
  {
    id: 'ps_ratio',
    label: 'P/S Ratio',
    type: 'range',
    group: 'valuation',
    minKey: 'ps_min',
    maxKey: 'ps_max',
    step: 0.1,
    placeholder: { min: '0', max: '30' },
  },

  // Profitability
  {
    id: 'roe',
    label: 'ROE',
    type: 'range',
    group: 'profitability',
    minKey: 'roe_min',
    maxKey: 'roe_max',
    step: 1,
    suffix: '%',
    placeholder: { min: '-50', max: '100' },
  },
  {
    id: 'roa',
    label: 'ROA',
    type: 'range',
    group: 'profitability',
    minKey: 'roa_min',
    maxKey: 'roa_max',
    step: 1,
    suffix: '%',
    placeholder: { min: '-20', max: '50' },
  },
  {
    id: 'gross_margin',
    label: 'Gross Margin',
    type: 'range',
    group: 'profitability',
    minKey: 'gross_margin_min',
    maxKey: 'gross_margin_max',
    step: 1,
    suffix: '%',
    placeholder: { min: '0', max: '100' },
  },
  {
    id: 'net_margin',
    label: 'Net Margin',
    type: 'range',
    group: 'profitability',
    minKey: 'net_margin_min',
    maxKey: 'net_margin_max',
    step: 1,
    suffix: '%',
    placeholder: { min: '-50', max: '80' },
  },

  // Financial Health
  {
    id: 'debt_to_equity',
    label: 'Debt/Equity',
    type: 'range',
    group: 'financial_health',
    minKey: 'de_min',
    maxKey: 'de_max',
    step: 0.1,
    placeholder: { min: '0', max: '5' },
  },
  {
    id: 'current_ratio',
    label: 'Current Ratio',
    type: 'range',
    group: 'financial_health',
    minKey: 'current_ratio_min',
    maxKey: 'current_ratio_max',
    step: 0.1,
    placeholder: { min: '0', max: '10' },
  },

  // Growth
  {
    id: 'revenue_growth',
    label: 'Revenue Growth (YoY)',
    type: 'range',
    group: 'growth',
    minKey: 'revenue_growth_min',
    maxKey: 'revenue_growth_max',
    step: 5,
    suffix: '%',
    placeholder: { min: '-50', max: '100' },
  },
  {
    id: 'eps_growth',
    label: 'EPS Growth (YoY)',
    type: 'range',
    group: 'growth',
    minKey: 'eps_growth_min',
    maxKey: 'eps_growth_max',
    step: 5,
    suffix: '%',
    placeholder: { min: '-50', max: '100' },
  },

  // Dividends
  {
    id: 'dividend_yield',
    label: 'Dividend Yield',
    type: 'range',
    group: 'dividends',
    minKey: 'dividend_yield_min',
    maxKey: 'dividend_yield_max',
    step: 0.1,
    suffix: '%',
    placeholder: { min: '0', max: '15' },
  },
  {
    id: 'payout_ratio',
    label: 'Payout Ratio',
    type: 'range',
    group: 'dividends',
    minKey: 'payout_ratio_min',
    maxKey: 'payout_ratio_max',
    step: 5,
    suffix: '%',
    placeholder: { min: '0', max: '100' },
  },

  // Risk & Value
  {
    id: 'beta',
    label: 'Beta',
    type: 'range',
    group: 'risk',
    minKey: 'beta_min',
    maxKey: 'beta_max',
    step: 0.1,
    placeholder: { min: '0', max: '3' },
  },
  {
    id: 'dcf_upside',
    label: 'DCF Upside',
    type: 'range',
    group: 'risk',
    minKey: 'dcf_upside_min',
    maxKey: 'dcf_upside_max',
    step: 5,
    suffix: '%',
    placeholder: { min: '-50', max: '200' },
  },

  // IC Score Factors
  {
    id: 'value_score',
    label: 'Value Score',
    type: 'range',
    group: 'ic_factors',
    minKey: 'value_score_min',
    maxKey: 'value_score_max',
    step: 1,
    placeholder: { min: '0', max: '100' },
  },
  {
    id: 'growth_score',
    label: 'Growth Score',
    type: 'range',
    group: 'ic_factors',
    minKey: 'growth_score_min',
    maxKey: 'growth_score_max',
    step: 1,
    placeholder: { min: '0', max: '100' },
  },
  {
    id: 'profitability_score',
    label: 'Profitability Score',
    type: 'range',
    group: 'ic_factors',
    minKey: 'profitability_score_min',
    maxKey: 'profitability_score_max',
    step: 1,
    placeholder: { min: '0', max: '100' },
  },
  {
    id: 'financial_health_score',
    label: 'Financial Health Score',
    type: 'range',
    group: 'ic_factors',
    minKey: 'financial_health_score_min',
    maxKey: 'financial_health_score_max',
    step: 1,
    placeholder: { min: '0', max: '100' },
  },
  {
    id: 'momentum_score',
    label: 'Momentum Score',
    type: 'range',
    group: 'ic_factors',
    minKey: 'momentum_score_min',
    maxKey: 'momentum_score_max',
    step: 1,
    placeholder: { min: '0', max: '100' },
  },
  {
    id: 'analyst_score',
    label: 'Analyst Consensus',
    type: 'range',
    group: 'ic_factors',
    minKey: 'analyst_score_min',
    maxKey: 'analyst_score_max',
    step: 1,
    placeholder: { min: '0', max: '100' },
  },
  {
    id: 'insider_score',
    label: 'Insider Activity',
    type: 'range',
    group: 'ic_factors',
    minKey: 'insider_score_min',
    maxKey: 'insider_score_max',
    step: 1,
    placeholder: { min: '0', max: '100' },
  },
  {
    id: 'institutional_score',
    label: 'Institutional Score',
    type: 'range',
    group: 'ic_factors',
    minKey: 'institutional_score_min',
    maxKey: 'institutional_score_max',
    step: 1,
    placeholder: { min: '0', max: '100' },
  },
  {
    id: 'sentiment_score',
    label: 'News Sentiment',
    type: 'range',
    group: 'ic_factors',
    minKey: 'sentiment_score_min',
    maxKey: 'sentiment_score_max',
    step: 1,
    placeholder: { min: '0', max: '100' },
  },
  {
    id: 'technical_score',
    label: 'Technical Score',
    type: 'range',
    group: 'ic_factors',
    minKey: 'technical_score_min',
    maxKey: 'technical_score_max',
    step: 1,
    placeholder: { min: '0', max: '100' },
  },
];
