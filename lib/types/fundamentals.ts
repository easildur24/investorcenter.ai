/**
 * Shared TypeScript interfaces for fundamental health data.
 *
 * Used by:
 *   - useHealthSummary hook
 *   - FundamentalHealthCard component
 *   - HealthBadge, LifecycleBadge, RedFlagBadge sub-components
 *   - MetricsTab inline red flag indicators
 */

// ─── Health Badge ────────────────────────────────────────────────────────────
export type HealthBadgeLevel = 'Strong' | 'Healthy' | 'Fair' | 'Weak' | 'Distressed';

// ─── Lifecycle Stage ─────────────────────────────────────────────────────────
export type LifecycleStage = 'hypergrowth' | 'growth' | 'mature' | 'value' | 'turnaround';

// ─── Red Flag Severity ───────────────────────────────────────────────────────
export type RedFlagSeverity = 'high' | 'medium' | 'low';

// ─── Component Scores ────────────────────────────────────────────────────────
export interface PiotroskiFScore {
  value: number;
  max: number;
  interpretation: string;
}

export interface AltmanZScore {
  value: number;
  zone: 'safe' | 'grey' | 'distress';
  interpretation: string;
}

export interface ICFinancialHealth {
  value: number;
  max: number;
}

export interface DebtPercentile {
  value: number;
  interpretation: string;
}

export interface HealthComponents {
  piotroski_f_score: PiotroskiFScore;
  altman_z_score: AltmanZScore;
  ic_financial_health: ICFinancialHealth;
  debt_percentile: DebtPercentile;
}

// ─── Health Summary ──────────────────────────────────────────────────────────
export interface HealthData {
  badge: HealthBadgeLevel;
  score: number;
  components: HealthComponents;
}

export interface LifecycleData {
  stage: LifecycleStage;
  description: string;
  classified_at: string;
}

export interface StrengthItem {
  metric: string;
  value: number;
  percentile: number;
  message: string;
}

export interface ConcernItem {
  metric: string;
  value: number;
  percentile: number;
  message: string;
}

export interface RedFlag {
  id: string;
  severity: RedFlagSeverity;
  title: string;
  description: string;
  related_metrics: string[];
}

export interface HealthSummaryResponse {
  ticker: string;
  health: HealthData;
  lifecycle: LifecycleData;
  strengths: StrengthItem[];
  concerns: ConcernItem[];
  red_flags: RedFlag[];
}
