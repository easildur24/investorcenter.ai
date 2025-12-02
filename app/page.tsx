import { ChartBarIcon, ArrowTrendingUpIcon, DocumentChartBarIcon, CurrencyDollarIcon } from '@heroicons/react/24/outline'
import MarketOverview from '@/components/MarketOverview'
import TopMovers from '@/components/home/TopMovers'

export default function Home() {
  return (
    <div className="min-h-screen bg-ic-bg-primary">

      {/* Hero Section */}
      <div className="relative overflow-hidden" style={{ background: 'var(--ic-hero-bg)' }}>
        {/* Decorative blur circles */}
        <div
          className="absolute top-20 left-10 w-72 h-72 rounded-full blur-3xl"
          style={{ background: 'var(--ic-blur-circle)' }}
        />
        <div
          className="absolute bottom-10 right-20 w-96 h-96 rounded-full blur-3xl"
          style={{ background: 'var(--ic-blur-circle)' }}
        />

        <div className="max-w-7xl mx-auto relative">
          <div className="relative z-10 pb-8 sm:pb-16 md:pb-20 lg:max-w-2xl lg:w-full lg:pb-28 xl:pb-32">
            <main className="mt-10 mx-auto max-w-7xl px-4 sm:mt-12 sm:px-6 md:mt-16 lg:mt-20 lg:px-8 xl:mt-28">
              <div className="sm:text-center lg:text-left">
                <h1 className="text-4xl tracking-tight font-extrabold sm:text-5xl md:text-6xl">
                  <span className="block xl:inline text-ic-text-primary">Professional</span>{' '}
                  <span className="block xl:inline" style={{ color: 'var(--ic-hero-text-accent)' }}>Financial Analytics</span>
                </h1>
                <p className="mt-3 text-base sm:mt-5 sm:text-lg sm:max-w-xl sm:mx-auto md:mt-5 md:text-xl lg:mx-0" style={{ color: 'var(--ic-hero-text-body)' }}>
                  Access comprehensive financial data, interactive charts, and powerful analytics tools.
                  Make informed investment decisions with institutional-grade research and insights.
                </p>
                <div className="mt-5 sm:mt-8 sm:flex sm:justify-center lg:justify-start">
                  <div className="rounded-md shadow-lg">
                    <button className="w-full flex items-center justify-center px-8 py-3 border border-transparent text-base font-medium rounded-md bg-ic-blue hover:bg-ic-blue-hover text-white md:py-4 md:text-lg md:px-10 transition-colors">
                      Start Free Trial
                    </button>
                  </div>
                  <div className="mt-3 sm:mt-0 sm:ml-3">
                    <button className="w-full flex items-center justify-center px-8 py-3 border border-ic-border text-base font-medium rounded-md bg-ic-surface text-ic-text-secondary hover:bg-ic-surface-hover md:py-4 md:text-lg md:px-10 transition-colors" style={{ boxShadow: 'var(--ic-shadow-card)' }}>
                      Watch Demo
                    </button>
                  </div>
                </div>
              </div>
            </main>
          </div>
        </div>
        <div className="lg:absolute lg:inset-y-0 lg:right-0 lg:w-1/2 flex items-center justify-center p-8">
          <div
            className="h-56 w-full sm:h-72 md:h-96 lg:h-full max-w-md rounded-2xl flex items-center justify-center backdrop-blur-sm border border-ic-border"
            style={{
              background: 'var(--ic-surface)',
              boxShadow: 'var(--ic-shadow-card)'
            }}
          >
            <div className="text-center p-8">
              <ArrowTrendingUpIcon className="h-24 w-24 mx-auto mb-4 text-ic-blue" />
              <p className="text-lg font-medium text-ic-text-primary">Interactive Charts & Analytics</p>
              <p className="text-sm text-ic-text-muted mt-2">Real-time data visualization</p>
            </div>
          </div>
        </div>
      </div>

      {/* Market Overview Section */}
      <div className="py-12 bg-ic-bg-secondary">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="lg:text-center mb-8">
            <h2 className="text-base text-ic-blue font-semibold tracking-wide uppercase">Live Data</h2>
            <p className="mt-2 text-3xl leading-8 font-extrabold tracking-tight text-ic-text-primary sm:text-4xl">
              Real-time market insights
            </p>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 max-w-5xl mx-auto">
            <MarketOverview />
            <TopMovers />
          </div>
        </div>
      </div>

      {/* Features Section */}
      <div className="py-12 bg-ic-bg-primary">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="lg:text-center">
            <h2 className="text-base text-ic-blue font-semibold tracking-wide uppercase">Features</h2>
            <p className="mt-2 text-3xl leading-8 font-extrabold tracking-tight text-ic-text-primary sm:text-4xl">
              Everything you need for investment research
            </p>
            <p className="mt-4 max-w-2xl text-xl text-ic-text-muted lg:mx-auto">
              Professional-grade tools and data to power your investment decisions
            </p>
          </div>

          <div className="mt-10">
            <div className="space-y-10 md:space-y-0 md:grid md:grid-cols-2 md:gap-x-8 md:gap-y-10">
              <div className="relative">
                <div className="absolute flex items-center justify-center h-12 w-12 rounded-md bg-ic-blue text-ic-text-primary">
                  <ChartBarIcon className="h-6 w-6" />
                </div>
                <p className="ml-16 text-lg leading-6 font-medium text-ic-text-primary">Interactive Charts</p>
                <p className="mt-2 ml-16 text-base text-ic-text-muted">
                  Advanced charting tools with technical indicators, multiple timeframes, and customizable layouts.
                </p>
              </div>

              <div className="relative">
                <div className="absolute flex items-center justify-center h-12 w-12 rounded-md bg-ic-blue text-ic-text-primary">
                  <DocumentChartBarIcon className="h-6 w-6" />
                </div>
                <p className="ml-16 text-lg leading-6 font-medium text-ic-text-primary">Financial Data</p>
                <p className="mt-2 ml-16 text-base text-ic-text-muted">
                  Real-time and historical data for stocks, bonds, commodities, and economic indicators.
                </p>
              </div>

              <div className="relative">
                <div className="absolute flex items-center justify-center h-12 w-12 rounded-md bg-ic-blue text-ic-text-primary">
                  <ArrowTrendingUpIcon className="h-6 w-6" />
                </div>
                <p className="ml-16 text-lg leading-6 font-medium text-ic-text-primary">Market Analytics</p>
                <p className="mt-2 ml-16 text-base text-ic-text-muted">
                  Comprehensive market analysis, sector performance, and trend identification tools.
                </p>
              </div>

              <div className="relative">
                <div className="absolute flex items-center justify-center h-12 w-12 rounded-md bg-ic-blue text-ic-text-primary">
                  <CurrencyDollarIcon className="h-6 w-6" />
                </div>
                <p className="ml-16 text-lg leading-6 font-medium text-ic-text-primary">Watch Lists</p>
                <p className="mt-2 ml-16 text-base text-ic-text-muted">
                  Create custom watch lists, track real-time prices, and set target price alerts for your favorite stocks and crypto.
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Footer */}
      <footer className="bg-ic-bg-secondary">
        <div className="max-w-7xl mx-auto py-12 px-4 sm:px-6 lg:py-16 lg:px-8">
          <div className="xl:grid xl:grid-cols-3 xl:gap-8">
            <div className="space-y-8 xl:col-span-1">
              <div className="flex items-center">
                <ChartBarIcon className="h-8 w-8 text-ic-blue" />
                <span className="ml-2 text-xl font-bold text-ic-text-primary">InvestorCenter</span>
              </div>
              <p className="text-ic-text-muted text-base">
                Professional financial data and analytics platform for informed investment decisions.
              </p>
            </div>
            <div className="mt-12 grid grid-cols-2 gap-8 xl:mt-0 xl:col-span-2">
              <div className="md:grid md:grid-cols-2 md:gap-8">
                <div>
                  <h3 className="text-sm font-semibold text-ic-text-dim tracking-wider uppercase">
                    Platform
                  </h3>
                  <ul className="mt-4 space-y-4">
                    <li><a href="#" className="text-base text-ic-text-muted hover:text-ic-text-primary transition-colors">Charts</a></li>
                    <li><a href="#" className="text-base text-ic-text-muted hover:text-ic-text-primary transition-colors">Data</a></li>
                    <li><a href="#" className="text-base text-ic-text-muted hover:text-ic-text-primary transition-colors">Analytics</a></li>
                  </ul>
                </div>
                <div className="mt-12 md:mt-0">
                  <h3 className="text-sm font-semibold text-ic-text-dim tracking-wider uppercase">
                    Company
                  </h3>
                  <ul className="mt-4 space-y-4">
                    <li><a href="#" className="text-base text-ic-text-muted hover:text-ic-text-primary transition-colors">About</a></li>
                    <li><a href="#" className="text-base text-ic-text-muted hover:text-ic-text-primary transition-colors">Contact</a></li>
                    <li><a href="#" className="text-base text-ic-text-muted hover:text-ic-text-primary transition-colors">Privacy</a></li>
                  </ul>
                </div>
              </div>
            </div>
          </div>
          <div className="mt-12 border-t border-ic-border pt-8">
            <p className="text-base text-ic-text-dim xl:text-center">
              &copy; 2024 InvestorCenter. All rights reserved.
            </p>
          </div>
        </div>
      </footer>
    </div>
  )
}
