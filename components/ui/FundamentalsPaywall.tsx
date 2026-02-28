'use client';

import { useEffect, useRef } from 'react';
import { LockClosedIcon } from '@heroicons/react/24/solid';
import { useAuth } from '@/lib/auth/AuthContext';
import { trackFundamentalsEvent, FUNDAMENTALS_EVENTS } from '@/lib/utils/fundamentals-analytics';
import PaywallOverlay from '@/components/ui/PaywallOverlay';
import type { FundamentalsPaywallProps } from '@/lib/types/fundamentals';

/**
 * FundamentalsPaywall — Reusable gating wrapper component
 *
 * Wraps fundamentals content with premium gating logic.
 * Supports three display variants for non-premium users:
 * - 'blur': Content renders blurred with an overlay CTA
 * - 'lock': Content replaced with a lock icon and "Premium" text
 * - 'teaser': Content renders as-is (parent handles truncation)
 *
 * Premium users see children rendered directly with zero wrapper overhead.
 * Tracks paywall impressions on mount for analytics.
 */
export default function FundamentalsPaywall({
  feature,
  children,
  ctaText,
  ticker,
  variant = 'blur',
}: FundamentalsPaywallProps) {
  const { user } = useAuth();
  const hasTrackedImpression = useRef(false);

  const isPremium = user?.is_premium === true;

  // Track paywall impression on mount (only for non-premium users)
  useEffect(() => {
    if (!isPremium && !hasTrackedImpression.current) {
      trackFundamentalsEvent(FUNDAMENTALS_EVENTS.PAYWALL_IMPRESSION, ticker || '', {
        feature,
        variant,
      });
      hasTrackedImpression.current = true;
    }
  }, [isPremium, feature, variant, ticker]);

  // Premium users: render children directly, no wrapper overhead
  if (isPremium) {
    return <>{children}</>;
  }

  // Variant: lock — replace content with lock icon + "Premium" text
  if (variant === 'lock') {
    return (
      <div className="flex items-center gap-2 rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 dark:border-gray-700 dark:bg-gray-800/50">
        <LockClosedIcon className="h-4 w-4 text-amber-500" />
        <span className="text-sm font-medium text-gray-500 dark:text-gray-400">Premium</span>
      </div>
    );
  }

  // Variant: teaser — render children as-is (parent handles truncation + teaser text)
  if (variant === 'teaser') {
    return <>{children}</>;
  }

  // Variant: blur (default) — render blurred content with PaywallOverlay
  return (
    <div className="relative overflow-hidden rounded-lg">
      {/* Blurred content layer */}
      <div className="pointer-events-none select-none blur-sm" aria-hidden="true">
        {children}
      </div>

      {/* Overlay CTA */}
      <PaywallOverlay ctaText={ctaText} feature={feature} ticker={ticker} />
    </div>
  );
}
