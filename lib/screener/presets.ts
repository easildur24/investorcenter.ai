// Quick Screen preset definitions for the stock screener.

import type { ScreenerPreset } from '@/lib/types/screener';

export const PRESET_SCREENS: ScreenerPreset[] = [
  {
    id: 'value',
    name: 'Value Stocks',
    description: 'Low P/E & P/B, high dividend yield, conservative leverage',
    params: { pe_max: 15, pb_max: 2, dividend_yield_min: 2, de_max: 1.5 },
  },
  {
    id: 'growth',
    name: 'Growth Stocks',
    description: 'High revenue & EPS growth with strong margins',
    params: { revenue_growth_min: 20, eps_growth_min: 15, gross_margin_min: 40 },
  },
  {
    id: 'quality',
    name: 'Quality Stocks',
    description: 'High IC Score, strong profitability, financially healthy',
    params: {
      ic_score_min: 70,
      roe_min: 15,
      net_margin_min: 10,
      current_ratio_min: 1.5,
    },
  },
  {
    id: 'dividend',
    name: 'Dividend Champions',
    description: 'High yield, long dividend track record, low leverage',
    params: {
      dividend_yield_min: 3,
      consec_div_years_min: 10,
      payout_ratio_max: 75,
      de_max: 1,
    },
  },
  {
    id: 'undervalued',
    name: 'Undervalued',
    description: 'Low valuation multiples, profitable, decent IC Score',
    params: { pe_max: 15, pb_max: 1.5, roe_min: 12, ic_score_min: 50 },
  },
  {
    id: 'momentum',
    name: 'Momentum Leaders',
    description: 'High momentum + technical scores',
    params: { momentum_score_min: 70, technical_score_min: 70 },
  },
];
