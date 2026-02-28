'use client';

import { LockClosedIcon } from '@heroicons/react/24/solid';
import { trackFundamentalsEvent, FUNDAMENTALS_EVENTS } from '@/lib/utils/fundamentals-analytics';
import type { PaywallOverlayProps } from '@/lib/types/fundamentals';

/**
 * PaywallOverlay — Upgrade CTA component
 *
 * Renders a centered absolute overlay on blurred content with a lock icon,
 * upgrade heading, custom CTA text, and action buttons.
 * Tracks CTA click events for attribution analytics.
 */
export default function PaywallOverlay({ ctaText, feature, ticker }: PaywallOverlayProps) {
  const upgradeUrl = `/subscriptions?source=fundamentals&feature=${encodeURIComponent(feature)}${ticker ? `&ticker=${encodeURIComponent(ticker)}` : ''}`;

  const handleUpgradeClick = () => {
    trackFundamentalsEvent(FUNDAMENTALS_EVENTS.PAYWALL_CTA_CLICKED, ticker || '', {
      feature,
      cta_type: 'upgrade_now',
    });
  };

  const handleLearnMoreClick = () => {
    trackFundamentalsEvent(FUNDAMENTALS_EVENTS.PAYWALL_CTA_CLICKED, ticker || '', {
      feature,
      cta_type: 'learn_more',
    });
  };

  return (
    <div className="absolute inset-0 z-10 flex items-center justify-center">
      <div className="flex flex-col items-center gap-3 rounded-xl bg-white/95 px-6 py-5 shadow-lg backdrop-blur-sm dark:bg-gray-900/95">
        {/* Lock icon */}
        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-amber-100 dark:bg-amber-900/30">
          <LockClosedIcon className="h-5 w-5 text-amber-600 dark:text-amber-400" />
        </div>

        {/* Heading */}
        <h3 className="text-base font-semibold text-gray-900 dark:text-gray-100">
          Upgrade to Premium
        </h3>

        {/* Custom CTA text */}
        {ctaText && (
          <p className="max-w-xs text-center text-sm text-gray-600 dark:text-gray-400">{ctaText}</p>
        )}

        {/* Action buttons */}
        <div className="flex items-center gap-3">
          <a
            href={upgradeUrl}
            onClick={handleUpgradeClick}
            className="inline-flex items-center rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-sm transition-colors hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
          >
            Upgrade Now
          </a>
          <a
            href={upgradeUrl}
            onClick={handleLearnMoreClick}
            className="inline-flex items-center rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 shadow-sm transition-colors hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"
          >
            Learn More
          </a>
        </div>
      </div>
    </div>
  );
}
