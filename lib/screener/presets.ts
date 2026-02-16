// Quick Screen preset definitions for the stock screener.

import type { ScreenerPreset } from '@/lib/types/screener';

export const PRESET_SCREENS: ScreenerPreset[] = [
  {
    id: 'value',
    name: 'Value Stocks',
    description: 'Low P/E, high dividend yield',
    params: { pe_max: 15, dividend_yield_min: 2 },
  },
  {
    id: 'growth',
    name: 'Growth Stocks',
    description: 'High revenue & EPS growth',
    params: { revenue_growth_min: 20, eps_growth_min: 15 },
  },
  {
    id: 'quality',
    name: 'Quality Stocks',
    description: 'High IC Score, large cap',
    params: { ic_score_min: 70, market_cap_min: 10e9 },
  },
  {
    id: 'dividend',
    name: 'Dividend Champions',
    description: 'High yield, low payout ratio',
    params: { dividend_yield_min: 3, payout_ratio_max: 80, market_cap_min: 10e9 },
  },
  {
    id: 'undervalued',
    name: 'Undervalued',
    description: 'High DCF upside, profitable',
    params: { dcf_upside_min: 20, roe_min: 10, de_max: 2 },
  },
  {
    id: 'momentum',
    name: 'Momentum Leaders',
    description: 'High momentum + technical scores',
    params: { momentum_score_min: 70, technical_score_min: 70 },
  },
];
