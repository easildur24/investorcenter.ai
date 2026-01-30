/**
 * IC Score API Client
 *
 * Handles all API requests related to InvestorCenter's proprietary IC Score system.
 */

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

/**
 * Factor calculation metadata - shows the inputs used to calculate each factor score
 */
export interface FactorMetadata {
  // Value factor inputs
  pe_ratio?: number;
  pb_ratio?: number;
  ps_ratio?: number;
  pe_score?: number;
  pb_score?: number;
  ps_score?: number;

  // Growth factor inputs
  revenue_growth?: number;
  eps_growth?: number;
  revenue_score?: number;
  eps_score?: number;

  // Profitability factor inputs
  net_margin?: number;
  roe?: number;
  roa?: number;
  margin_score?: number;
  roe_score?: number;
  roa_score?: number;

  // Financial health factor inputs
  debt_to_equity?: number;
  current_ratio?: number;
  de_score?: number;
  cr_score?: number;

  // Momentum factor inputs
  return_1m?: number;
  return_3m?: number;
  return_6m?: number;
  return_12m?: number;

  // Technical factor inputs
  rsi?: number;
  macd_histogram?: number;
  price_vs_sma50?: number;
  price_vs_sma200?: number;

  // Analyst consensus inputs
  buy_count?: number;
  hold_count?: number;
  sell_count?: number;
  consensus_score?: number;
  price_target?: number;
  current_price?: number;
  upside_pct?: number;

  // Insider activity inputs
  net_shares?: number;
  buy_transactions?: number;
  sell_transactions?: number;

  // Institutional inputs
  institution_count?: number;
  total_shares?: number;
  holdings_change_pct?: number;

  // News sentiment inputs
  sentiment_avg?: number;
  article_count?: number;

  // Generic field for any additional data
  [key: string]: number | string | undefined;
}

/**
 * Calculation metadata stored with each IC Score
 */
export interface CalculationMetadata {
  factors: Record<string, FactorMetadata>;
  weights_used: Record<string, number>;
  calculated_at: string;
}

/**
 * IC Score Data Interface
 * Represents the complete IC Score data for a stock
 */
export interface ICScoreData {
  ticker: string;
  date: string;
  overall_score: number;
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
  rating: string;
  sector_percentile: number | null;
  confidence_level: string;
  data_completeness: number;
  calculated_at: string;
  factor_count: number;
  available_factors: string[];
  missing_factors: string[];
  calculation_metadata?: CalculationMetadata;
}

/**
 * IC Score List Item
 * Simplified view for admin/list displays
 */
export interface ICScoreListItem {
  ticker: string;
  overall_score: number;
  rating: string;
  data_completeness: number;
  calculated_at: string;
}

/**
 * IC Score Factor Details
 * Detailed information about a specific factor
 */
export interface ICScoreFactor {
  name: string;
  display_name: string;
  score: number | null;
  weight: number;
  grade: string;
  available: boolean;
  description: string;
}

/**
 * API Response wrapper
 */
interface APIResponse<T> {
  data: T;
  meta?: Record<string, any>;
}

/**
 * Fetch IC Score for a specific ticker
 *
 * @param ticker Stock symbol (e.g., "AAPL")
 * @returns IC Score data or null if not available
 */
export async function getICScore(ticker: string): Promise<ICScoreData | null> {
  try {
    const response = await fetch(`${API_BASE_URL}/stocks/${ticker.toUpperCase()}/ic-score`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      cache: 'no-store', // Always fetch fresh data
    });

    if (response.status === 404) {
      // Score not available for this ticker
      return null;
    }

    if (!response.ok) {
      console.error(`Failed to fetch IC Score for ${ticker}: ${response.status}`);
      return null;
    }

    const result: APIResponse<ICScoreData> = await response.json();
    return result.data;
  } catch (error) {
    console.error(`Error fetching IC Score for ${ticker}:`, error);
    return null;
  }
}

/**
 * Fetch IC Score history for a ticker
 *
 * @param ticker Stock symbol
 * @param days Number of days of history to fetch (default: 90)
 * @returns Array of historical IC Scores
 */
export async function getICScoreHistory(
  ticker: string,
  days: number = 90
): Promise<ICScoreData[]> {
  try {
    const response = await fetch(
      `${API_BASE_URL}/stocks/${ticker.toUpperCase()}/ic-score/history?days=${days}`,
      {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        cache: 'no-store',
      }
    );

    if (!response.ok) {
      console.error(`Failed to fetch IC Score history for ${ticker}: ${response.status}`);
      return [];
    }

    const result: APIResponse<ICScoreData[]> = await response.json();
    return result.data;
  } catch (error) {
    console.error(`Error fetching IC Score history for ${ticker}:`, error);
    return [];
  }
}

/**
 * Fetch all IC Scores with pagination
 *
 * @param params Query parameters for filtering and pagination
 * @returns Paginated list of IC Scores
 */
export async function getICScores(params?: {
  limit?: number;
  offset?: number;
  search?: string;
  sort?: string;
  order?: 'asc' | 'desc';
}): Promise<{
  data: ICScoreListItem[];
  meta: {
    total: number;
    limit: number;
    offset: number;
    total_stocks: number;
    coverage_percent: number;
  };
}> {
  try {
    const queryParams = new URLSearchParams();
    if (params?.limit) queryParams.set('limit', params.limit.toString());
    if (params?.offset) queryParams.set('offset', params.offset.toString());
    if (params?.search) queryParams.set('search', params.search);
    if (params?.sort) queryParams.set('sort', params.sort);
    if (params?.order) queryParams.set('order', params.order);

    const url = `${API_BASE_URL}/ic-scores${queryParams.toString() ? '?' + queryParams.toString() : ''}`;
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      cache: 'no-store',
    });

    if (!response.ok) {
      console.error(`Failed to fetch IC Scores: ${response.status}`);
      return {
        data: [],
        meta: {
          total: 0,
          limit: params?.limit || 20,
          offset: params?.offset || 0,
          total_stocks: 0,
          coverage_percent: 0,
        },
      };
    }

    return await response.json();
  } catch (error) {
    console.error('Error fetching IC Scores:', error);
    return {
      data: [],
      meta: {
        total: 0,
        limit: params?.limit || 20,
        offset: params?.offset || 0,
        total_stocks: 0,
        coverage_percent: 0,
      },
    };
  }
}

/**
 * Get letter grade for a score
 *
 * Aligned with IC Score rating system:
 * - A range (80-100): Strong Buy
 * - B range (65-79): Buy
 * - C range (50-64): Hold
 * - D range (35-49): Underperform
 * - F range (0-34): Sell
 *
 * @param score Numeric score (0-100)
 * @returns Letter grade (A+ to F)
 */
export function getLetterGrade(score: number | null): string {
  if (score === null) return 'N/A';

  // A range: Strong Buy (80-100)
  if (score >= 93) return 'A+';
  if (score >= 86) return 'A';
  if (score >= 80) return 'A-';

  // B range: Buy (65-79)
  if (score >= 75) return 'B+';
  if (score >= 70) return 'B';
  if (score >= 65) return 'B-';

  // C range: Hold (50-64)
  if (score >= 60) return 'C+';
  if (score >= 55) return 'C';
  if (score >= 50) return 'C-';

  // D range: Underperform (35-49)
  if (score >= 45) return 'D+';
  if (score >= 40) return 'D';
  if (score >= 35) return 'D-';

  // F range: Sell (0-34)
  return 'F';
}

/**
 * Get rating label from score
 */
export function getRating(score: number | null): string {
  if (score === null) return 'N/A';
  if (score >= 80) return 'Strong Buy';
  if (score >= 65) return 'Buy';
  if (score >= 50) return 'Hold';
  if (score >= 35) return 'Underperform';
  return 'Sell';
}

/**
 * Get grade tier description
 */
export function getGradeTier(score: number | null): { letter: string; rating: string; description: string } {
  if (score === null) return { letter: 'N/A', rating: 'N/A', description: 'Data not available' };

  if (score >= 80) return {
    letter: getLetterGrade(score),
    rating: 'Strong Buy',
    description: 'Exceptional investment profile'
  };
  if (score >= 65) return {
    letter: getLetterGrade(score),
    rating: 'Buy',
    description: 'Favorable investment profile'
  };
  if (score >= 50) return {
    letter: getLetterGrade(score),
    rating: 'Hold',
    description: 'Neutral investment profile'
  };
  if (score >= 35) return {
    letter: getLetterGrade(score),
    rating: 'Underperform',
    description: 'Below average investment profile'
  };
  return {
    letter: getLetterGrade(score),
    rating: 'Sell',
    description: 'Weak investment profile'
  };
}

/**
 * Get color class for a score
 *
 * @param score Numeric score (0-100)
 * @returns Tailwind color class
 */
export function getScoreColor(score: number | null): string {
  if (score === null) return 'text-ic-text-muted';

  if (score >= 80) return 'text-green-600';
  if (score >= 65) return 'text-green-500';
  if (score >= 50) return 'text-yellow-500';
  if (score >= 35) return 'text-orange-500';
  return 'text-red-500';
}

/**
 * Get background color class for a score
 */
export function getScoreBgColor(score: number | null): string {
  if (score === null) return 'bg-ic-surface';

  if (score >= 80) return 'bg-green-100';
  if (score >= 65) return 'bg-green-50';
  if (score >= 50) return 'bg-yellow-50';
  if (score >= 35) return 'bg-orange-50';
  return 'bg-red-50';
}

/**
 * Get all factor details with metadata
 */
export function getFactorDetails(icScore: ICScoreData): ICScoreFactor[] {
  const factors: ICScoreFactor[] = [
    {
      name: 'value',
      display_name: 'Value',
      score: icScore.value_score,
      weight: 12,
      grade: getLetterGrade(icScore.value_score),
      available: icScore.value_score !== null,
      description: 'P/E, P/B, P/S ratios vs sector',
    },
    {
      name: 'growth',
      display_name: 'Growth',
      score: icScore.growth_score,
      weight: 15,
      grade: getLetterGrade(icScore.growth_score),
      available: icScore.growth_score !== null,
      description: 'Revenue, EPS growth trends',
    },
    {
      name: 'profitability',
      display_name: 'Profitability',
      score: icScore.profitability_score,
      weight: 12,
      grade: getLetterGrade(icScore.profitability_score),
      available: icScore.profitability_score !== null,
      description: 'Margins, ROE, ROA',
    },
    {
      name: 'financial_health',
      display_name: 'Financial Health',
      score: icScore.financial_health_score,
      weight: 10,
      grade: getLetterGrade(icScore.financial_health_score),
      available: icScore.financial_health_score !== null,
      description: 'Debt ratios, liquidity',
    },
    {
      name: 'momentum',
      display_name: 'Momentum',
      score: icScore.momentum_score,
      weight: 8,
      grade: getLetterGrade(icScore.momentum_score),
      available: icScore.momentum_score !== null,
      description: 'Price trends, relative strength',
    },
    {
      name: 'analyst_consensus',
      display_name: 'Analyst Consensus',
      score: icScore.analyst_consensus_score,
      weight: 10,
      grade: getLetterGrade(icScore.analyst_consensus_score),
      available: icScore.analyst_consensus_score !== null,
      description: 'Buy/sell ratings, price targets',
    },
    {
      name: 'insider_activity',
      display_name: 'Insider Activity',
      score: icScore.insider_activity_score,
      weight: 8,
      grade: getLetterGrade(icScore.insider_activity_score),
      available: icScore.insider_activity_score !== null,
      description: 'Insider buying/selling trends',
    },
    {
      name: 'institutional',
      display_name: 'Institutional',
      score: icScore.institutional_score,
      weight: 10,
      grade: getLetterGrade(icScore.institutional_score),
      available: icScore.institutional_score !== null,
      description: 'Institutional ownership changes',
    },
    {
      name: 'news_sentiment',
      display_name: 'News Sentiment',
      score: icScore.news_sentiment_score,
      weight: 7,
      grade: getLetterGrade(icScore.news_sentiment_score),
      available: icScore.news_sentiment_score !== null,
      description: 'News analysis and sentiment',
    },
    {
      name: 'technical',
      display_name: 'Technical',
      score: icScore.technical_score,
      weight: 8,
      grade: getLetterGrade(icScore.technical_score),
      available: icScore.technical_score !== null,
      description: 'RSI, MACD, trend indicators',
    },
  ];

  return factors;
}

/**
 * Get star rating (0-5 stars) based on score
 */
export function getStarRating(score: number): number {
  return Math.round((score / 100) * 5);
}
