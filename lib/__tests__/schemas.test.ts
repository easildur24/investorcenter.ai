/**
 * Tests for Zod API response schemas.
 *
 * Validates that schemas correctly accept valid data and reject
 * malformed responses from the backend.
 */

import {
  ICScoreSchema,
  ICScoreScreenerResponseSchema,
  ICScoreTopStocksResponseSchema,
  ICScoreHistorySchema,
  ICScoreStatisticsSchema,
  ScreenerResponseSchema,
  MarketIndicesSchema,
  MarketMoversSchema,
} from '../api/schemas';
import { validateResponse } from '../api/validate';

// ---------------------------------------------------------------------------
// Test data factories
// ---------------------------------------------------------------------------

function makeICScoreFactor(overrides = {}) {
  return {
    name: 'Value',
    value: 72,
    weight: 0.15,
    contribution: 10.8,
    ...overrides,
  };
}

function makeICScore(overrides = {}) {
  return {
    ticker: 'AAPL',
    score: 78,
    rating: 'Buy' as const,
    calculatedAt: '2026-02-17T00:00:00Z',
    factors: {
      value: makeICScoreFactor({ name: 'Value' }),
      growth: makeICScoreFactor({ name: 'Growth' }),
      profitability: makeICScoreFactor({ name: 'Profitability' }),
      financial_health: makeICScoreFactor({ name: 'Financial Health' }),
      momentum: makeICScoreFactor({ name: 'Momentum' }),
      analyst_consensus: makeICScoreFactor({ name: 'Analyst Consensus' }),
      insider_activity: makeICScoreFactor({ name: 'Insider Activity' }),
      institutional: makeICScoreFactor({ name: 'Institutional' }),
      news_sentiment: makeICScoreFactor({ name: 'News Sentiment' }),
      technical: makeICScoreFactor({ name: 'Technical' }),
    },
    ...overrides,
  };
}

function makeScreenerStock(overrides = {}) {
  return {
    symbol: 'AAPL',
    name: 'Apple Inc.',
    sector: 'Technology',
    industry: 'Consumer Electronics',
    market_cap: 2850000000000,
    price: 185.0,
    pe_ratio: 28.5,
    pb_ratio: 45.0,
    ps_ratio: 7.5,
    roe: 23.0,
    roa: 8.5,
    gross_margin: 42.0,
    operating_margin: 28.0,
    net_margin: 22.0,
    debt_to_equity: 0.85,
    current_ratio: 1.8,
    revenue_growth: 12.5,
    eps_growth_yoy: 15.3,
    dividend_yield: 0.5,
    payout_ratio: 15.0,
    consecutive_dividend_years: 12,
    beta: 1.2,
    dcf_upside_percent: 8.1,
    ic_score: 78,
    ic_rating: 'Buy',
    value_score: 72,
    growth_score: 68,
    profitability_score: 85,
    financial_health_score: 70,
    momentum_score: 65,
    analyst_consensus_score: 75,
    insider_activity_score: 50,
    institutional_score: 60,
    news_sentiment_score: 55,
    technical_score: 62,
    ic_sector_percentile: 85,
    lifecycle_stage: 'mature',
    ...overrides,
  };
}

// ---------------------------------------------------------------------------
// ICScore schema tests
// ---------------------------------------------------------------------------

describe('ICScoreSchema', () => {
  it('accepts valid IC Score data', () => {
    const result = ICScoreSchema.safeParse(makeICScore());
    expect(result.success).toBe(true);
  });

  it('rejects missing required ticker', () => {
    const { ticker, ...noTicker } = makeICScore();
    const result = ICScoreSchema.safeParse(noTicker);
    expect(result.success).toBe(false);
  });

  it('rejects missing required score', () => {
    const { score, ...noScore } = makeICScore();
    const result = ICScoreSchema.safeParse(noScore);
    expect(result.success).toBe(false);
  });

  it('rejects score out of range (> 100)', () => {
    const result = ICScoreSchema.safeParse(makeICScore({ score: 101 }));
    expect(result.success).toBe(false);
  });

  it('rejects score out of range (< 1)', () => {
    const result = ICScoreSchema.safeParse(makeICScore({ score: 0 }));
    expect(result.success).toBe(false);
  });

  it('rejects invalid rating string', () => {
    const result = ICScoreSchema.safeParse(makeICScore({ rating: 'Super Buy' }));
    expect(result.success).toBe(false);
  });

  it('accepts optional fields when present', () => {
    const result = ICScoreSchema.safeParse(
      makeICScore({
        sector: 'Technology',
        industry: 'Consumer Electronics',
        marketCap: 2850000000000,
        percentile: 92,
        previousScore: 75,
        scoreChange: 3,
      })
    );
    expect(result.success).toBe(true);
  });

  it('validates nested factor value range (0-100)', () => {
    const bad = makeICScore();
    bad.factors.value.value = 150;
    const result = ICScoreSchema.safeParse(bad);
    expect(result.success).toBe(false);
  });

  it('validates nested factor weight range (0-1)', () => {
    const bad = makeICScore();
    bad.factors.growth.weight = 2.0;
    const result = ICScoreSchema.safeParse(bad);
    expect(result.success).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// ScreenerResponse schema tests
// ---------------------------------------------------------------------------

describe('ScreenerResponseSchema', () => {
  it('accepts valid screener response', () => {
    const result = ScreenerResponseSchema.safeParse({
      data: [makeScreenerStock()],
      meta: {
        total: 1,
        page: 1,
        limit: 25,
        total_pages: 1,
        timestamp: '2026-02-17T00:00:00Z',
      },
    });
    expect(result.success).toBe(true);
  });

  it('accepts screener stock with all-null optional fields', () => {
    const result = ScreenerResponseSchema.safeParse({
      data: [
        makeScreenerStock({
          sector: null,
          industry: null,
          market_cap: null,
          price: null,
          pe_ratio: null,
          pb_ratio: null,
          ps_ratio: null,
          roe: null,
          roa: null,
          gross_margin: null,
          operating_margin: null,
          net_margin: null,
          debt_to_equity: null,
          current_ratio: null,
          revenue_growth: null,
          eps_growth_yoy: null,
          dividend_yield: null,
          payout_ratio: null,
          consecutive_dividend_years: null,
          beta: null,
          dcf_upside_percent: null,
          ic_score: null,
          ic_rating: null,
          value_score: null,
          growth_score: null,
          profitability_score: null,
          financial_health_score: null,
          momentum_score: null,
          analyst_consensus_score: null,
          insider_activity_score: null,
          institutional_score: null,
          news_sentiment_score: null,
          technical_score: null,
          ic_sector_percentile: null,
          lifecycle_stage: null,
        }),
      ],
      meta: {
        total: 1,
        page: 1,
        limit: 25,
        total_pages: 1,
        timestamp: '2026-02-17T00:00:00Z',
      },
    });
    expect(result.success).toBe(true);
  });

  it('rejects malformed meta (missing total)', () => {
    const result = ScreenerResponseSchema.safeParse({
      data: [],
      meta: {
        page: 1,
        limit: 25,
        total_pages: 1,
        timestamp: '2026-02-17T00:00:00Z',
      },
    });
    expect(result.success).toBe(false);
  });

  it('accepts empty data array', () => {
    const result = ScreenerResponseSchema.safeParse({
      data: [],
      meta: {
        total: 0,
        page: 1,
        limit: 25,
        total_pages: 0,
        timestamp: '2026-02-17T00:00:00Z',
      },
    });
    expect(result.success).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// MarketIndices / MarketMovers schema tests
// ---------------------------------------------------------------------------

describe('MarketIndicesSchema', () => {
  it('accepts valid market indices array', () => {
    const result = MarketIndicesSchema.safeParse([
      {
        symbol: 'SPY',
        name: 'S&P 500',
        price: 5100.5,
        change: 25.3,
        changePercent: 0.5,
        lastUpdated: '2026-02-17T16:00:00Z',
      },
    ]);
    expect(result.success).toBe(true);
  });

  it('accepts empty array', () => {
    const result = MarketIndicesSchema.safeParse([]);
    expect(result.success).toBe(true);
  });
});

describe('MarketMoversSchema', () => {
  it('accepts valid market movers', () => {
    const mover = {
      symbol: 'NVDA',
      name: 'NVIDIA Corp',
      price: 800.0,
      change: 50.0,
      changePercent: 6.6,
      volume: 15000000,
    };
    const result = MarketMoversSchema.safeParse({
      gainers: [mover],
      losers: [{ ...mover, change: -30, changePercent: -3.6 }],
      mostActive: [mover],
    });
    expect(result.success).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// validateResponse wrapper tests
// ---------------------------------------------------------------------------

describe('validateResponse', () => {
  const consoleSpy = jest.spyOn(console, 'warn').mockImplementation(() => {});

  afterEach(() => {
    consoleSpy.mockClear();
  });

  afterAll(() => {
    consoleSpy.mockRestore();
  });

  it('returns parsed data on valid input', () => {
    const data = makeICScore();
    const result = validateResponse(ICScoreSchema, data, '/test');
    expect(result.ticker).toBe('AAPL');
    expect(consoleSpy).not.toHaveBeenCalled();
  });

  it('logs warning and returns raw data on invalid input', () => {
    const badData = { ticker: 'AAPL' }; // missing required fields
    const result = validateResponse(ICScoreSchema, badData, '/test');
    expect(result).toBe(badData);
    expect(consoleSpy).toHaveBeenCalledWith('[API Validation] /test:', expect.any(Array));
  });

  it('handles non-object input gracefully', () => {
    const result = validateResponse(ICScoreSchema, 'not-an-object', '/test');
    expect(result).toBe('not-an-object');
    expect(consoleSpy).toHaveBeenCalled();
  });
});
