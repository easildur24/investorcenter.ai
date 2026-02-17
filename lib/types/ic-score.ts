/**
 * TypeScript type definitions for IC Score feature
 *
 * IC Score is a proprietary 1-100 scoring system that evaluates stocks
 * based on 10 weighted factors across fundamental, technical, and sentiment analysis.
 */

/**
 * IC Score rating bands
 */
export type ICScoreRating = 'Strong Buy' | 'Buy' | 'Hold' | 'Underperform' | 'Sell';

/**
 * Individual factor within the IC Score calculation
 */
export interface ICScoreFactor {
  name: string;
  value: number;        // Factor score (0-100)
  weight: number;       // Weight percentage (0-1)
  contribution: number; // Weighted contribution to total score
}

/**
 * Main IC Score data structure
 */
export interface ICScore {
  ticker: string;
  score: number;                    // Overall IC Score (1-100)
  rating: ICScoreRating;            // Rating band
  calculatedAt: string;             // ISO 8601 timestamp

  // 10-factor breakdown
  factors: {
    value: ICScoreFactor;
    growth: ICScoreFactor;
    profitability: ICScoreFactor;
    financial_health: ICScoreFactor;
    momentum: ICScoreFactor;
    analyst_consensus: ICScoreFactor;
    insider_activity: ICScoreFactor;
    institutional: ICScoreFactor;
    news_sentiment: ICScoreFactor;
    technical: ICScoreFactor;
  };

  // Optional metadata
  sector?: string;
  industry?: string;
  marketCap?: number;

  // Comparison metrics
  percentile?: number;              // Percentile rank vs all stocks
  sectorPercentile?: number;        // Percentile rank within sector
  previousScore?: number;           // Score from previous calculation
  scoreChange?: number;             // Change from previous score
}

/**
 * Historical IC Score data point
 */
export interface ICScoreHistoryPoint {
  date: string;                     // ISO 8601 date
  score: number;                    // IC Score on that date
  rating: ICScoreRating;            // Rating on that date
}

/**
 * IC Score history response (30-day trend)
 */
export interface ICScoreHistory {
  ticker: string;
  history: ICScoreHistoryPoint[];
  startDate: string;
  endDate: string;
  averageScore: number;
  minScore: number;
  maxScore: number;
  volatility: number;               // Standard deviation of scores
}

/**
 * Helper function to get rating color class
 */
export function getICScoreRatingColor(rating: ICScoreRating): string {
  switch (rating) {
    case 'Strong Buy':
      return 'text-green-600 bg-green-50 border-green-200';
    case 'Buy':
      return 'text-green-500 bg-green-50 border-green-200';
    case 'Hold':
      return 'text-yellow-600 bg-yellow-50 border-yellow-200';
    case 'Underperform':
      return 'text-orange-600 bg-orange-50 border-orange-200';
    case 'Sell':
      return 'text-red-600 bg-red-50 border-red-200';
    default:
      return 'text-ic-text-muted bg-ic-surface border-ic-border-subtle';
  }
}

/**
 * Helper function to get rating from score
 */
export function getICScoreRating(score: number): ICScoreRating {
  if (score >= 80) return 'Strong Buy';
  if (score >= 65) return 'Buy';
  if (score >= 50) return 'Hold';
  if (score >= 35) return 'Underperform';
  return 'Sell';
}

/**
 * Helper function to get score color (for gauge)
 */
export function getICScoreColor(score: number): string {
  if (score >= 80) return '#10b981'; // green-500
  if (score >= 65) return '#84cc16'; // lime-500
  if (score >= 50) return '#eab308'; // yellow-500
  if (score >= 35) return '#f97316'; // orange-500
  return '#ef4444'; // red-500
}

/**
 * Helper function to format factor name for display
 */
export function formatFactorName(factorKey: string): string {
  const nameMap: Record<string, string> = {
    value: 'Value',
    growth: 'Growth',
    profitability: 'Profitability',
    financial_health: 'Financial Health',
    momentum: 'Momentum',
    analyst_consensus: 'Analyst Consensus',
    insider_activity: 'Insider Activity',
    institutional: 'Institutional',
    news_sentiment: 'News Sentiment',
    technical: 'Technical',
  };

  return nameMap[factorKey] || factorKey;
}
