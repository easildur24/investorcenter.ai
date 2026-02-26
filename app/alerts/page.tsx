'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';

/**
 * /alerts redirect page — 30-day transition period.
 *
 * Alerts have moved into watchlists (Phase 3). This page shows a brief
 * explanation and auto-redirects to /watchlist after 5 seconds. After the
 * transition period, replace with: redirect('/watchlist') from next/navigation.
 */
export default function AlertsRedirectPage() {
  const router = useRouter();

  useEffect(() => {
    const timer = setTimeout(() => router.replace('/watchlist'), 5000);
    return () => clearTimeout(timer);
  }, [router]);

  return (
    <div className="min-h-screen flex items-center justify-center px-4">
      <div className="max-w-lg w-full p-8 bg-ic-surface rounded-lg border border-ic-border text-center">
        <svg
          className="mx-auto h-12 w-12 text-ic-blue mb-4"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
          />
        </svg>
        <h2 className="text-xl font-bold text-ic-text-primary mb-2">Alerts have moved</h2>
        <p className="text-ic-text-muted mb-6">
          Alerts are now managed directly from your watchlists. Each watchlist has its own Alerts
          tab where you can view, create, and manage all your alerts in context.
        </p>
        <p className="text-sm text-ic-text-dim mb-6">Redirecting in 5 seconds&hellip;</p>
        <Link
          href="/watchlist"
          className="inline-flex items-center gap-2 px-6 py-3 bg-ic-blue text-ic-text-primary rounded-lg hover:bg-ic-blue-hover font-semibold transition-colors"
        >
          Go to Watchlists
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M13 7l5 5m0 0l-5 5m5-5H6"
            />
          </svg>
        </Link>
      </div>
    </div>
  );
}
