'use client';

import MarketStatusBanner from './MarketStatusBanner';
import HeroMiniDashboard from './HeroMiniDashboard';
import Link from 'next/link';

export default function HeroSection() {
  return (
    <section className="relative overflow-hidden" style={{ background: 'var(--ic-hero-bg)' }}>
      {/* Decorative blur circles */}
      <div
        className="absolute top-20 left-10 w-72 h-72 rounded-full blur-3xl pointer-events-none"
        style={{ background: 'var(--ic-blur-circle)' }}
      />
      <div
        className="absolute bottom-10 right-20 w-96 h-96 rounded-full blur-3xl pointer-events-none"
        style={{ background: 'var(--ic-blur-circle)' }}
      />

      {/* Market Status Banner — full width at top */}
      <MarketStatusBanner />

      {/* Main hero content */}
      <div className="relative z-10 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="py-12 sm:py-16 lg:py-20">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-10 lg:gap-16 items-center">
            {/* Left column: Copy + CTAs */}
            <div className="text-center lg:text-left">
              <h1 className="text-4xl tracking-tight font-extrabold sm:text-5xl md:text-6xl">
                <span className="block text-ic-text-primary">Professional</span>
                <span className="block" style={{ color: 'var(--ic-hero-text-accent)' }}>
                  Financial Analytics
                </span>
              </h1>

              <p
                className="mt-4 text-base sm:text-lg md:text-xl max-w-xl mx-auto lg:mx-0"
                style={{ color: 'var(--ic-hero-text-body)' }}
              >
                Access comprehensive financial data, interactive charts, and powerful analytics
                tools. Make informed investment decisions with institutional-grade research and
                insights.
              </p>

              {/* CTA Buttons */}
              <div className="mt-8 flex flex-col sm:flex-row gap-3 sm:justify-center lg:justify-start">
                <Link
                  href="/signup"
                  className="inline-flex items-center justify-center px-8 py-3 text-base font-medium rounded-md bg-ic-blue hover:bg-ic-blue-hover text-white shadow-lg transition-colors md:py-4 md:text-lg md:px-10"
                >
                  Start Free Trial
                </Link>
                <Link
                  href="/screener"
                  className="inline-flex items-center justify-center px-8 py-3 text-base font-medium rounded-md border border-ic-border bg-ic-surface text-ic-text-secondary hover:bg-ic-surface-hover transition-colors md:py-4 md:text-lg md:px-10"
                  style={{ boxShadow: 'var(--ic-shadow-card)' }}
                >
                  View Markets
                </Link>
              </div>
            </div>

            {/* Right column: Mini dashboard */}
            <div className="flex flex-col items-center lg:items-end">
              <div className="w-full max-w-lg">
                <HeroMiniDashboard />
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
