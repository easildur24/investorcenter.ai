// Screener types — aligned with backend models/stock.go ScreenerStock + ScreenerParams

/** A single stock row returned by the screener API */
export interface ScreenerStock {
  // Core identity
  symbol: string;
  name: string;
  sector: string | null;
  industry: string | null;

  // Market data
  market_cap: number | null;
  price: number | null;

  // Valuation
  pe_ratio: number | null;
  pb_ratio: number | null;
  ps_ratio: number | null;

  // Profitability
  roe: number | null;
  roa: number | null;
  gross_margin: number | null;
  operating_margin: number | null;
  net_margin: number | null;

  // Financial health
  debt_to_equity: number | null;
  current_ratio: number | null;

  // Growth
  revenue_growth: number | null;
  eps_growth_yoy: number | null;

  // Dividends
  dividend_yield: number | null;
  payout_ratio: number | null;
  consecutive_dividend_years: number | null;

  // Risk
  beta: number | null;

  // Fair value
  dcf_upside_percent: number | null;

  // IC Score
  ic_score: number | null;
  ic_rating: string | null;

  // IC Score sub-factors (0-100)
  value_score: number | null;
  growth_score: number | null;
  profitability_score: number | null;
  financial_health_score: number | null;
  momentum_score: number | null;
  analyst_consensus_score: number | null;
  insider_activity_score: number | null;
  institutional_score: number | null;
  news_sentiment_score: number | null;
  technical_score: number | null;

  // IC Score metadata
  ic_sector_percentile: number | null;
  lifecycle_stage: string | null;
}

/** Pagination metadata from the screener API */
export interface ScreenerMeta {
  total: number;
  page: number;
  limit: number;
  total_pages: number;
  timestamp: string;
}

/** Full API response shape */
export interface ScreenerResponse {
  data: ScreenerStock[];
  meta: ScreenerMeta;
}

/** Query parameters sent to the screener API. All optional except page/limit/sort/order. */
export interface ScreenerApiParams {
  // Pagination & sorting
  page?: number;
  limit?: number;
  sort?: string;
  order?: 'asc' | 'desc';

  // Categorical
  sectors?: string;
  industries?: string;
  asset_type?: string;

  // Market data
  market_cap_min?: number;
  market_cap_max?: number;

  // Valuation
  pe_min?: number;
  pe_max?: number;
  pb_min?: number;
  pb_max?: number;
  ps_min?: number;
  ps_max?: number;

  // Profitability
  roe_min?: number;
  roe_max?: number;
  roa_min?: number;
  roa_max?: number;
  gross_margin_min?: number;
  gross_margin_max?: number;
  net_margin_min?: number;
  net_margin_max?: number;

  // Financial health
  de_min?: number;
  de_max?: number;
  current_ratio_min?: number;
  current_ratio_max?: number;

  // Growth
  revenue_growth_min?: number;
  revenue_growth_max?: number;
  eps_growth_min?: number;
  eps_growth_max?: number;

  // Dividends
  dividend_yield_min?: number;
  dividend_yield_max?: number;
  payout_ratio_min?: number;
  payout_ratio_max?: number;
  consec_div_years_min?: number;

  // Risk
  beta_min?: number;
  beta_max?: number;

  // Fair value
  dcf_upside_min?: number;
  dcf_upside_max?: number;

  // IC Score
  ic_score_min?: number;
  ic_score_max?: number;

  // IC Score sub-factors
  value_score_min?: number;
  value_score_max?: number;
  growth_score_min?: number;
  growth_score_max?: number;
  profitability_score_min?: number;
  profitability_score_max?: number;
  financial_health_score_min?: number;
  financial_health_score_max?: number;
  momentum_score_min?: number;
  momentum_score_max?: number;
  analyst_score_min?: number;
  analyst_score_max?: number;
  insider_score_min?: number;
  insider_score_max?: number;
  institutional_score_min?: number;
  institutional_score_max?: number;
  sentiment_score_min?: number;
  sentiment_score_max?: number;
  technical_score_min?: number;
  technical_score_max?: number;
}

/** Valid sort columns — same keys as backend ValidScreenerSortColumns */
export type ScreenerSortField =
  | 'symbol'
  | 'name'
  | 'market_cap'
  | 'price'
  | 'pe_ratio'
  | 'pb_ratio'
  | 'ps_ratio'
  | 'roe'
  | 'roa'
  | 'gross_margin'
  | 'operating_margin'
  | 'net_margin'
  | 'debt_to_equity'
  | 'current_ratio'
  | 'revenue_growth'
  | 'eps_growth_yoy'
  | 'dividend_yield'
  | 'payout_ratio'
  | 'consecutive_dividend_years'
  | 'beta'
  | 'dcf_upside_percent'
  | 'ic_score'
  | 'value_score'
  | 'growth_score'
  | 'profitability_score'
  | 'financial_health_score'
  | 'momentum_score'
  | 'analyst_consensus_score'
  | 'insider_activity_score'
  | 'institutional_score'
  | 'news_sentiment_score'
  | 'technical_score';

/** Preset screen definition */
export interface ScreenerPreset {
  id: string;
  name: string;
  description: string;
  params: Partial<ScreenerApiParams>;
}
