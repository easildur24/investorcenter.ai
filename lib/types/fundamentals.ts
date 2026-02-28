/**
 * TypeScript type definitions for Fundamentals feature
 *
 * Shared interfaces for fundamentals paywall gating, analytics,
 * A/B testing, and free/premium feature boundaries.
 */

/**
 * Paywall variant determines how gated content is presented to free users.
 * - 'blur': Content renders with blur overlay + upgrade CTA
 * - 'lock': Content replaced with lock icon + "Premium" text
 * - 'teaser': Content renders as-is (parent handles truncation)
 */
export type PaywallVariant = 'blur' | 'lock' | 'teaser';

/**
 * Props for the FundamentalsPaywall gating wrapper component
 */
export interface FundamentalsPaywallProps {
  /** Feature identifier for tracking and attribution */
  feature: string;
  /** Content to gate behind the paywall */
  children: React.ReactNode;
  /** Custom CTA text displayed on the paywall overlay */
  ctaText?: string;
  /** Ticker symbol for tracking context */
  ticker?: string;
  /** How gated content is displayed to free users */
  variant?: PaywallVariant;
}

/**
 * Props for the PaywallOverlay upgrade CTA component
 */
export interface PaywallOverlayProps {
  /** Custom CTA text (e.g., "to compare AAPL with peers") */
  ctaText?: string;
  /** Feature identifier for attribution tracking */
  feature: string;
  /** Ticker symbol for tracking context */
  ticker?: string;
}

/**
 * A fundamentals analytics event queued for batch sending
 */
export interface FundamentalsEvent {
  /** Event action name (e.g., 'paywall_impression') */
  action: string;
  /** Ticker symbol associated with the event */
  ticker: string;
  /** ISO 8601 timestamp of when the event occurred */
  timestamp: string;
  /** Additional metadata for the event */
  metadata?: Record<string, unknown>;
}

/**
 * A/B test configuration for deterministic variant assignment
 */
export interface ABTestConfig {
  /** Unique identifier for the test */
  id: string;
  /** List of variant names */
  variants: string[];
  /** Default variant when no user ID is available */
  defaultVariant: string;
}

/**
 * Free/premium feature access levels for fundamentals
 */
export interface FundamentalsAccessConfig {
  /** Number of sector percentile bars visible to free users */
  freePercentileBarCount: number;
  /** Maximum strengths/concerns shown in health card for free users */
  freeHealthCardDetailCount: number;
  /** Whether the feature is available to free users at all */
  isFreeFeature: boolean;
  /** Whether premium unlocks additional content */
  hasPremiumContent: boolean;
}

/**
 * Map of fundamentals features to their access configuration
 */
export type FundamentalsFeatureMap = Record<string, FundamentalsAccessConfig>;

/**
 * Fundamentals feature identifiers used for tracking and gating
 */
export const FUNDAMENTALS_FEATURES = {
  PERCENTILE_BARS: 'percentile_bars',
  HEALTH_CARD: 'health_card',
  RED_FLAGS: 'red_flags',
  PEER_COMPARISON: 'peer_comparison',
  SPARKLINES: 'sparklines',
  FAIR_VALUE: 'fair_value',
  METRIC_HISTORY: 'metric_history',
} as const;

export type FundamentalsFeature =
  (typeof FUNDAMENTALS_FEATURES)[keyof typeof FUNDAMENTALS_FEATURES];

/**
 * Default free/premium boundaries for fundamentals features
 */
export const FUNDAMENTALS_ACCESS: Record<string, FundamentalsAccessConfig> = {
  [FUNDAMENTALS_FEATURES.PERCENTILE_BARS]: {
    freePercentileBarCount: 6,
    freeHealthCardDetailCount: 0,
    isFreeFeature: true,
    hasPremiumContent: true,
  },
  [FUNDAMENTALS_FEATURES.HEALTH_CARD]: {
    freePercentileBarCount: 0,
    freeHealthCardDetailCount: 2,
    isFreeFeature: true,
    hasPremiumContent: true,
  },
  [FUNDAMENTALS_FEATURES.RED_FLAGS]: {
    freePercentileBarCount: 0,
    freeHealthCardDetailCount: 0,
    isFreeFeature: true, // High severity only
    hasPremiumContent: true, // Medium + low severity
  },
  [FUNDAMENTALS_FEATURES.PEER_COMPARISON]: {
    freePercentileBarCount: 0,
    freeHealthCardDetailCount: 0,
    isFreeFeature: false,
    hasPremiumContent: true,
  },
  [FUNDAMENTALS_FEATURES.SPARKLINES]: {
    freePercentileBarCount: 0,
    freeHealthCardDetailCount: 0,
    isFreeFeature: false,
    hasPremiumContent: true,
  },
  [FUNDAMENTALS_FEATURES.FAIR_VALUE]: {
    freePercentileBarCount: 0,
    freeHealthCardDetailCount: 0,
    isFreeFeature: false,
    hasPremiumContent: true,
  },
  [FUNDAMENTALS_FEATURES.METRIC_HISTORY]: {
    freePercentileBarCount: 0,
    freeHealthCardDetailCount: 0,
    isFreeFeature: false,
    hasPremiumContent: true,
  },
};
