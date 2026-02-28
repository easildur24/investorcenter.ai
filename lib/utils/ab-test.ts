/**
 * Simple A/B test infrastructure
 *
 * Provides deterministic variant assignment based on user ID hashing.
 * Ensures consistent variant assignment across sessions for the same user.
 */

import type { ABTestConfig } from '@/lib/types/fundamentals';

/**
 * Get the A/B test variant for a given user.
 *
 * Uses a simple hash of the user ID combined with the test ID
 * to deterministically assign a variant. The same user always
 * gets the same variant for a given test.
 *
 * @param config - A/B test configuration
 * @param userId - Optional user ID for deterministic assignment
 * @returns The assigned variant name
 */
export function getABTestVariant(config: ABTestConfig, userId?: string): string {
  // If no user ID, return the default variant
  if (!userId) {
    return config.defaultVariant;
  }

  // Combine user ID and test ID for unique-per-test hashing
  const hashInput = `${userId}:${config.id}`;

  // Simple deterministic hash: sum of char codes
  let hash = 0;
  for (let i = 0; i < hashInput.length; i++) {
    hash = (hash * 31 + hashInput.charCodeAt(i)) | 0;
  }

  // Ensure positive value
  if (hash < 0) {
    hash = -hash;
  }

  // Map to variant index
  const variantIndex = hash % config.variants.length;
  return config.variants[variantIndex];
}

/**
 * Pre-defined A/B test configurations for fundamentals features
 */
export const AB_TESTS = {
  FREE_PERCENTILE_COUNT: {
    id: 'fundamentals_free_percentile_count',
    variants: ['6', '8', '10'],
    defaultVariant: '6',
  },
  PAYWALL_CTA_TEXT: {
    id: 'fundamentals_paywall_cta_text',
    variants: ['upgrade_now', 'try_premium', 'unlock_insights'],
    defaultVariant: 'upgrade_now',
  },
} as const;
