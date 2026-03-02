/**
 * Analytics module — provider-agnostic event tracking.
 *
 * Currently logs to console in development and no-ops in production.
 * Designed to be swapped with Mixpanel, GA4, or Amplitude later
 * without changing any calling code.
 *
 * Usage:
 *   import { analytics } from '@/lib/analytics';
 *   analytics.track('button_clicked', { button: 'Start Free Trial' });
 *   analytics.page('Home');
 *   analytics.identify('user-123');
 */

type EventProperties = Record<string, string | number | boolean | null | undefined>;

interface AnalyticsProvider {
  track(event: string, properties?: EventProperties): void;
  identify(userId: string, traits?: EventProperties): void;
  page(name: string, properties?: EventProperties): void;
}

// ─── Console Provider (development) ──────────────────────────────────────────

const consoleProvider: AnalyticsProvider = {
  track(event, properties) {
    if (process.env.NODE_ENV === 'development') {
      console.log(`[Analytics] track: ${event}`, properties || '');
    }
  },
  identify(userId, traits) {
    if (process.env.NODE_ENV === 'development') {
      console.log(`[Analytics] identify: ${userId}`, traits || '');
    }
  },
  page(name, properties) {
    if (process.env.NODE_ENV === 'development') {
      console.log(`[Analytics] page: ${name}`, properties || '');
    }
  },
};

// ─── No-op Provider (production fallback) ────────────────────────────────────

const noopProvider: AnalyticsProvider = {
  track() {},
  identify() {},
  page() {},
};

// ─── Singleton ───────────────────────────────────────────────────────────────

class Analytics {
  private provider: AnalyticsProvider;

  constructor() {
    // Default: console in dev, no-op in prod
    this.provider =
      typeof window !== 'undefined' && process.env.NODE_ENV === 'development'
        ? consoleProvider
        : noopProvider;

    // Expose analytics instance on window for runtime provider injection
    // Usage: window.__analytics.setProvider(myGA4Adapter)
    if (typeof window !== 'undefined') {
      (window as unknown as Record<string, unknown>).__analytics = this;
    }
  }

  /** Replace the analytics provider (e.g., with Mixpanel adapter) */
  setProvider(provider: AnalyticsProvider) {
    this.provider = provider;
  }

  /** Track a named event with optional properties */
  track(event: string, properties?: EventProperties) {
    this.provider.track(event, {
      ...properties,
      timestamp: new Date().toISOString(),
    });
  }

  /** Identify a user */
  identify(userId: string, traits?: EventProperties) {
    this.provider.identify(userId, traits);
  }

  /** Track a page view */
  page(name: string, properties?: EventProperties) {
    this.provider.page(name, properties);
  }
}

export const analytics = new Analytics();
