/**
 * IC Score v2.1 API Types
 *
 * Type definitions for IC Score v2.1 with enhanced features:
 * - Sector-relative scoring
 * - Lifecycle classification
 * - Peer comparison
 * - Catalyst detection
 * - Score explanations
 * - Granular confidence
 */

// ===================
// Lifecycle Types
// ===================

export type LifecycleStage =
  | 'hypergrowth'
  | 'growth'
  | 'mature'
  | 'value'
  | 'turnaround';

export const LIFECYCLE_LABELS: Record<LifecycleStage, string> = {
  hypergrowth: 'Hypergrowth',
  growth: 'Growth',
  mature: 'Mature',
  value: 'Value',
  turnaround: 'Turnaround',
};

export const LIFECYCLE_COLORS: Record<LifecycleStage, string> = {
  hypergrowth: 'text-purple-600 bg-purple-100',
  growth: 'text-blue-600 bg-blue-100',
  mature: 'text-green-600 bg-green-100',
  value: 'text-amber-600 bg-amber-100',
  turnaround: 'text-orange-600 bg-orange-100',
};

// ===================
// Peer Comparison Types
// ===================

export interface PeerStock {
  ticker: string;
  company_name: string;
  ic_score: number | null;
  similarity_score: number;
}

export interface PeerComparison {
  peers: PeerStock[];
  avg_peer_score: number | null;
  vs_peers_delta: number | null;
  sector_rank: number | null;
  sector_total: number | null;
}

// ===================
// Catalyst Types
// ===================

export type CatalystImpact = 'Positive' | 'Negative' | 'Neutral' | 'Unknown';

export interface Catalyst {
  event_type: string;
  title: string;
  event_date: string | null;
  icon: string | null;
  impact: CatalystImpact;
  confidence: number | null;
  days_until: number | null;
}

export const CATALYST_TYPE_LABELS: Record<string, string> = {
  earnings: 'Earnings Report',
  ex_dividend: 'Ex-Dividend Date',
  analyst_rating: 'Analyst Rating',
  insider_trade: 'Insider Activity',
  technical: 'Technical Signal',
  '52_week_high': '52-Week High',
  '52_week_low': '52-Week Low',
};

export const CATALYST_ICONS: Record<string, string> = {
  earnings: '📊',
  ex_dividend: '💰',
  analyst_rating: '📈',
  insider_trade: '👔',
  technical: '📉',
  '52_week_high': '🔝',
  '52_week_low': '📉',
};

// ===================
// Score Change Types
// ===================

export interface FactorChange {
  factor: string;
  delta: number;
  contribution: number;
  explanation: string;
}

export interface ScoreExplanation {
  summary: string;
  delta: number;
  reasons: FactorChange[];
  confidence: {
    level: string;
    percentage: number;
    warnings: string[];
  };
}

// ===================
// Confidence Types
// ===================

export type ConfidenceLevel = 'High' | 'Medium' | 'Low';

export interface FactorDataStatus {
  available: boolean;
  freshness: 'fresh' | 'recent' | 'stale' | 'missing';
  freshness_days?: number;
  count?: number;
  warning?: string;
  reason?: string;
}

export interface GranularConfidence {
  level: ConfidenceLevel;
  percentage: number;
  factors: Record<string, FactorDataStatus>;
  warnings: string[];
}

// ===================
// Category Types
// ===================

export interface CategoryScore {
  score: number;
  grade: string;
  factors: string[];
}

export interface Categories {
  quality: CategoryScore;
  valuation: CategoryScore;
  signals: CategoryScore;
}

// ===================
// Main IC Score Data Type (v2.1)
// ===================

export interface ICScoreDataV2 {
  ticker: string;
  date: string;

  // Core scores
  overall_score: number;
  previous_score?: number | null;
  raw_score?: number | null;
  smoothing_applied?: boolean;
  rating: string;
  confidence_level: string;
  data_completeness: number;
  calculated_at: string;

  // Factor scores (v2.0)
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

  // Phase 2 factor scores
  earnings_revisions_score?: number | null;
  historical_value_score?: number | null;
  dividend_quality_score?: number | null;

  // v2.1: Lifecycle & sector context
  lifecycle_stage?: LifecycleStage | null;
  sector?: string | null;
  sector_rank?: number | null;
  sector_total?: number | null;
  sector_percentile: number | null;

  // Phase 3: Peer comparison
  peers?: PeerStock[];
  peer_comparison?: PeerComparison;

  // Phase 3: Catalysts
  catalysts?: Catalyst[];

  // Phase 3: Explanation
  explanation?: ScoreExplanation;

  // Legacy fields
  factor_count: number;
  available_factors: string[];
  missing_factors: string[];
}

// ===================
// Factor Configuration
// ===================

export interface FactorConfig {
  name: string;
  display_name: string;
  category: 'quality' | 'valuation' | 'signals';
  weight: number;
  description: string;
  icon?: string;
}

export const FACTOR_CONFIGS: FactorConfig[] = [
  // Quality (35%)
  {
    name: 'profitability',
    display_name: 'Profitability',
    category: 'quality',
    weight: 0.12,
    description: 'Net margin, ROE, ROA vs sector',
    icon: '💹',
  },
  {
    name: 'financial_health',
    display_name: 'Financial Health',
    category: 'quality',
    weight: 0.10,
    description: 'Debt ratios, liquidity position',
    icon: '🏦',
  },
  {
    name: 'growth',
    display_name: 'Growth',
    category: 'quality',
    weight: 0.13,
    description: 'Revenue, EPS growth vs sector',
    icon: '📈',
  },
  // Valuation (30%)
  {
    name: 'value',
    display_name: 'Value',
    category: 'valuation',
    weight: 0.12,
    description: 'P/E, P/B, P/S vs sector',
    icon: '💰',
  },
  {
    name: 'intrinsic_value',
    display_name: 'Intrinsic Value',
    category: 'valuation',
    weight: 0.10,
    description: 'DCF-based fair value estimate',
    icon: '🎯',
  },
  {
    name: 'historical_value',
    display_name: 'Historical Value',
    category: 'valuation',
    weight: 0.08,
    description: 'Current vs 5-year P/E range',
    icon: '📊',
  },
  // Signals (35%)
  {
    name: 'momentum',
    display_name: 'Momentum',
    category: 'signals',
    weight: 0.10,
    description: 'Price trends, relative strength',
    icon: '🚀',
  },
  {
    name: 'smart_money',
    display_name: 'Smart Money',
    category: 'signals',
    weight: 0.10,
    description: 'Analyst, insider, institutional signals',
    icon: '🧠',
  },
  {
    name: 'earnings_revisions',
    display_name: 'Earnings Revisions',
    category: 'signals',
    weight: 0.08,
    description: 'EPS estimate changes',
    icon: '📝',
  },
  {
    name: 'technical',
    display_name: 'Technical',
    category: 'signals',
    weight: 0.07,
    description: 'RSI, MACD, trend indicators',
    icon: '📉',
  },
];

// ===================
// Utility Functions
// ===================

export function getCategoryScore(
  icScore: ICScoreDataV2,
  category: 'quality' | 'valuation' | 'signals'
): CategoryScore {
  const categoryFactors = FACTOR_CONFIGS.filter((f) => f.category === category);
  const scores: number[] = [];
  const factors: string[] = [];

  categoryFactors.forEach((factor) => {
    const scoreKey = `${factor.name}_score` as keyof ICScoreDataV2;
    const score = icScore[scoreKey] as number | null;
    if (score !== null) {
      scores.push(score);
      factors.push(factor.name);
    }
  });

  const avgScore = scores.length > 0 ? scores.reduce((a, b) => a + b, 0) / scores.length : 0;

  return {
    score: Math.round(avgScore),
    grade: getGradeFromScore(avgScore),
    factors,
  };
}

export function getGradeFromScore(score: number): string {
  if (score >= 90) return 'A';
  if (score >= 80) return 'B';
  if (score >= 70) return 'C';
  if (score >= 60) return 'D';
  return 'F';
}

export function getScoreColorClass(score: number | null): string {
  if (score === null) return 'text-gray-400';
  if (score >= 80) return 'text-green-600';
  if (score >= 65) return 'text-green-500';
  if (score >= 50) return 'text-yellow-500';
  if (score >= 35) return 'text-orange-500';
  return 'text-red-500';
}

export function getScoreBgClass(score: number | null): string {
  if (score === null) return 'bg-gray-100';
  if (score >= 80) return 'bg-green-100';
  if (score >= 65) return 'bg-green-50';
  if (score >= 50) return 'bg-yellow-50';
  if (score >= 35) return 'bg-orange-50';
  return 'bg-red-50';
}

export function formatDaysUntil(days: number | null): string {
  if (days === null) return 'Unknown';
  if (days === 0) return 'Today';
  if (days === 1) return 'Tomorrow';
  if (days < 0) return `${Math.abs(days)} days ago`;
  return `In ${days} days`;
}

export function getImpactColor(impact: CatalystImpact): string {
  switch (impact) {
    case 'Positive':
      return 'text-green-600';
    case 'Negative':
      return 'text-red-600';
    case 'Neutral':
      return 'text-gray-600';
    default:
      return 'text-gray-400';
  }
}

export function getDeltaColor(delta: number): string {
  if (delta > 0) return 'text-green-600';
  if (delta < 0) return 'text-red-600';
  return 'text-gray-500';
}

export function formatDelta(delta: number): string {
  if (delta > 0) return `+${delta.toFixed(1)}`;
  return delta.toFixed(1);
}
