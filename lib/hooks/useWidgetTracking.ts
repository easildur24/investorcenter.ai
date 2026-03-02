'use client';

import { useEffect, useRef, useCallback } from 'react';
import { analytics } from '@/lib/analytics';

type InteractionMetadata = Record<string, string | number | boolean | null | undefined>;

/**
 * Hook for tracking widget visibility and interactions.
 *
 * Features:
 * - Fires `widget_visible` event when widget enters viewport (IntersectionObserver)
 * - Tracks `time_on_widget` on unmount
 * - Returns `trackInteraction()` callback for manual event tracking
 *
 * Usage:
 *   const { ref, trackInteraction } = useWidgetTracking('market_overview');
 *   // Attach ref to the widget container div
 *   // Call trackInteraction('tab_click', { tab: 'US Indices' }) on actions
 */
export function useWidgetTracking(widgetName: string) {
  const ref = useRef<HTMLDivElement>(null);
  const visibleTimeRef = useRef<number | null>(null);
  const hasBeenVisible = useRef(false);

  // Track when widget becomes visible
  useEffect(() => {
    const element = ref.current;
    if (!element) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting && !hasBeenVisible.current) {
          hasBeenVisible.current = true;
          visibleTimeRef.current = Date.now();
          analytics.track('widget_visible', { widget: widgetName });
        }
      },
      { threshold: 0.5 } // 50% of widget must be visible
    );

    observer.observe(element);

    return () => {
      observer.disconnect();

      // Track time on widget when unmounting
      if (visibleTimeRef.current) {
        const timeOnWidget = Math.round((Date.now() - visibleTimeRef.current) / 1000);
        if (timeOnWidget > 0) {
          analytics.track('time_on_widget', {
            widget: widgetName,
            seconds: timeOnWidget,
          });
        }
      }
    };
  }, [widgetName]);

  // Track manual interactions
  const trackInteraction = useCallback(
    (action: string, metadata?: InteractionMetadata) => {
      analytics.track('widget_interaction', {
        widget: widgetName,
        action,
        ...metadata,
      });
    },
    [widgetName]
  );

  return { ref, trackInteraction };
}
