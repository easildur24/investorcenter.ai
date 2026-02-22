/**
 * Lightweight client-side feature flag utility.
 *
 * Flags are read from Next.js public environment variables (NEXT_PUBLIC_*),
 * which are baked into the JS bundle at build time.
 *
 * For per-user rollout (phases 2-3 of the rollout plan), replace this with
 * server-side evaluation based on user ID.
 */

export const FF_INLINE_WATCHLIST_ADD = 'NEXT_PUBLIC_FF_INLINE_WATCHLIST_ADD';

/**
 * Returns true if the given feature flag environment variable is set to 'true'.
 */
export function isFeatureEnabled(flag: string): boolean {
  // process.env values are inlined by Next.js at build time for NEXT_PUBLIC_ vars.
  // We need to check each flag explicitly since dynamic access doesn't work with
  // Next.js dead-code elimination. For now, we use a simple map approach.
  switch (flag) {
    case FF_INLINE_WATCHLIST_ADD:
      return process.env.NEXT_PUBLIC_FF_INLINE_WATCHLIST_ADD === 'true';
    default:
      return false;
  }
}
