/**
 * Zod schemas for runtime validation of API responses.
 *
 * These mirror the TypeScript interfaces in lib/types/ but add
 * runtime checks to catch backend data mismatches early.
 */

import { z } from 'zod';

// ---------------------------------------------------------------------------
// IC Score schemas (mirrors lib/types/ic-score.ts)
// ---------------------------------------------------------------------------

const ICScoreRatingSchema = z.enum(['Strong Buy', 'Buy', 'Hold', 'Underperform', 'Sell']);

const ICScoreFactorSchema = z.object({
  name: z.string(),
  value: z.number().min(0).max(100),
  weight: z.number().min(0).max(1),
  contribution: z.number(),
});

export const ICScoreSchema = z.object({
  ticker: z.string().min(1),
  score: z.number().min(1).max(100),
  rating: ICScoreRatingSchema,
  calculatedAt: z.string(),

  factors: z.object({
    value: ICScoreFactorSchema,
    growth: ICScoreFactorSchema,
    profitability: ICScoreFactorSchema,
    financial_health: ICScoreFactorSchema,
    momentum: ICScoreFactorSchema,
    analyst_consensus: ICScoreFactorSchema,
    insider_activity: ICScoreFactorSchema,
    institutional: ICScoreFactorSchema,
    news_sentiment: ICScoreFactorSchema,
    technical: ICScoreFactorSchema,
  }),

  sector: z.string().optional(),
  industry: z.string().optional(),
  marketCap: z.number().optional(),
  percentile: z.number().optional(),
  sectorPercentile: z.number().optional(),
  previousScore: z.number().optional(),
  scoreChange: z.number().optional(),
});

const ICScoreStockEntrySchema = z.object({
  ticker: z.string().min(1),
  companyName: z.string(),
  score: z.number().min(1).max(100),
  rating: ICScoreRatingSchema,
  sector: z.string(),
  marketCap: z.number(),
  price: z.number(),
  change: z.number(),
  changePercent: z.number(),
  volume: z.number(),
  calculatedAt: z.string(),
  topFactors: z
    .array(
      z.object({
        name: z.string(),
        value: z.number(),
      })
    )
    .optional(),
});

export const ICScoreScreenerResponseSchema = z.object({
  stocks: z.array(ICScoreStockEntrySchema),
  total: z.number(),
  filters: z.object({}).passthrough(),
  timestamp: z.string(),
});

export const ICScoreTopStocksResponseSchema = z.object({
  stocks: z.array(ICScoreStockEntrySchema),
  limit: z.number(),
  timestamp: z.string(),
});

export const ICScoreHistorySchema = z.object({
  ticker: z.string().min(1),
  history: z.array(
    z.object({
      date: z.string(),
      score: z.number(),
      rating: ICScoreRatingSchema,
    })
  ),
  startDate: z.string(),
  endDate: z.string(),
  averageScore: z.number(),
  minScore: z.number(),
  maxScore: z.number(),
  volatility: z.number(),
});

export const ICScoreStatisticsSchema = z.object({
  totalStocks: z.number(),
  averageScore: z.number(),
  strongBuyCount: z.number(),
  buyCount: z.number(),
  holdCount: z.number(),
  underperformCount: z.number(),
  sellCount: z.number(),
  lastUpdated: z.string(),
});

// ---------------------------------------------------------------------------
// Screener schemas (mirrors lib/types/screener.ts)
// ---------------------------------------------------------------------------

const ScreenerStockSchema = z.object({
  symbol: z.string().min(1),
  name: z.string(),
  sector: z.string().nullable(),
  industry: z.string().nullable(),
  market_cap: z.number().nullable(),
  price: z.number().nullable(),
  pe_ratio: z.number().nullable(),
  pb_ratio: z.number().nullable(),
  ps_ratio: z.number().nullable(),
  roe: z.number().nullable(),
  roa: z.number().nullable(),
  gross_margin: z.number().nullable(),
  operating_margin: z.number().nullable(),
  net_margin: z.number().nullable(),
  debt_to_equity: z.number().nullable(),
  current_ratio: z.number().nullable(),
  revenue_growth: z.number().nullable(),
  eps_growth_yoy: z.number().nullable(),
  dividend_yield: z.number().nullable(),
  payout_ratio: z.number().nullable(),
  consecutive_dividend_years: z.number().nullable(),
  beta: z.number().nullable(),
  dcf_upside_percent: z.number().nullable(),
  ic_score: z.number().nullable(),
  ic_rating: z.string().nullable(),
  value_score: z.number().nullable(),
  growth_score: z.number().nullable(),
  profitability_score: z.number().nullable(),
  financial_health_score: z.number().nullable(),
  momentum_score: z.number().nullable(),
  analyst_consensus_score: z.number().nullable(),
  insider_activity_score: z.number().nullable(),
  institutional_score: z.number().nullable(),
  news_sentiment_score: z.number().nullable(),
  technical_score: z.number().nullable(),
  ic_sector_percentile: z.number().nullable(),
  lifecycle_stage: z.string().nullable(),
});

const ScreenerMetaSchema = z.object({
  total: z.number(),
  page: z.number(),
  limit: z.number(),
  total_pages: z.number(),
  timestamp: z.string(),
});

export const ScreenerResponseSchema = z.object({
  data: z.array(ScreenerStockSchema),
  meta: ScreenerMetaSchema,
});

// ---------------------------------------------------------------------------
// Market data schemas (mirrors inline types in lib/api.ts)
// ---------------------------------------------------------------------------

const MarketIndexSchema = z.object({
  symbol: z.string(),
  name: z.string(),
  price: z.number(),
  change: z.number(),
  changePercent: z.number(),
  lastUpdated: z.string(),
});

export const MarketIndicesSchema = z.array(MarketIndexSchema);

const MarketMoverSchema = z.object({
  symbol: z.string(),
  name: z.string().optional(),
  price: z.number(),
  change: z.number(),
  changePercent: z.number(),
  volume: z.number(),
});

export const MarketMoversSchema = z.object({
  gainers: z.array(MarketMoverSchema),
  losers: z.array(MarketMoverSchema),
  mostActive: z.array(MarketMoverSchema),
});
