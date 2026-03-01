import {
  ChartBarIcon,
  ArrowTrendingUpIcon,
  DocumentChartBarIcon,
  CurrencyDollarIcon,
} from '@heroicons/react/24/outline';
import MarketOverview from '@/components/MarketOverview';
import TopMovers from '@/components/home/TopMovers';
import HeroSection from '@/components/home/HeroSection';
import NewsFeed from '@/components/home/NewsFeed';
import UpcomingEarnings from '@/components/home/UpcomingEarnings';
import SectorHeatmap from '@/components/home/SectorHeatmap';
import WatchlistPreview from '@/components/home/WatchlistPreview';
import MarketSummary from '@/components/home/MarketSummary';
import WidgetErrorBoundary from '@/components/ui/WidgetErrorBoundary';
import Footer from '@/components/Footer';

export default function Home() {
  return (
    <div className="min-h-screen bg-ic-bg-primary">
      {/* Hero Section with Market Status + Mini Dashboard */}
      <WidgetErrorBoundary widgetName="Hero">
        <HeroSection />
      </WidgetErrorBoundary>

      {/* AI Market Summary */}
      <div className="py-6 bg-ic-bg-secondary">
        <div className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
          <WidgetErrorBoundary widgetName="Market Summary">
            <MarketSummary />
          </WidgetErrorBoundary>
        </div>
      </div>

      {/* Market Overview + Top Movers */}
      <div className="py-12 bg-ic-bg-secondary">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="lg:text-center mb-8">
            <h2 className="text-base text-ic-blue font-semibold tracking-wide uppercase">
              Live Data
            </h2>
            <p className="mt-2 text-3xl leading-8 font-extrabold tracking-tight text-ic-text-primary sm:text-4xl">
              Real-time market insights
            </p>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 max-w-5xl mx-auto">
            <WidgetErrorBoundary widgetName="Market Overview">
              <MarketOverview />
            </WidgetErrorBoundary>
            <WidgetErrorBoundary widgetName="Top Movers">
              <TopMovers />
            </WidgetErrorBoundary>
          </div>
        </div>
      </div>

      {/* News + Upcoming Earnings */}
      <div className="py-12 bg-ic-bg-primary">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 max-w-5xl mx-auto">
            <WidgetErrorBoundary widgetName="News Feed">
              <NewsFeed />
            </WidgetErrorBoundary>
            <WidgetErrorBoundary widgetName="Upcoming Earnings">
              <UpcomingEarnings />
            </WidgetErrorBoundary>
          </div>
        </div>
      </div>

      {/* Sector Heatmap + Watchlist Preview */}
      <div className="py-12 bg-ic-bg-secondary">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 max-w-5xl mx-auto">
            <WidgetErrorBoundary widgetName="Sector Performance">
              <SectorHeatmap />
            </WidgetErrorBoundary>
            <WidgetErrorBoundary widgetName="Watchlist Preview">
              <WatchlistPreview />
            </WidgetErrorBoundary>
          </div>
        </div>
      </div>

      {/* Features Section */}
      <div className="py-12 bg-ic-bg-primary">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="lg:text-center">
            <h2 className="text-base text-ic-blue font-semibold tracking-wide uppercase">
              Features
            </h2>
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
                <p className="ml-16 text-lg leading-6 font-medium text-ic-text-primary">
                  Interactive Charts
                </p>
                <p className="mt-2 ml-16 text-base text-ic-text-muted">
                  Advanced charting tools with technical indicators, multiple timeframes, and
                  customizable layouts.
                </p>
              </div>

              <div className="relative">
                <div className="absolute flex items-center justify-center h-12 w-12 rounded-md bg-ic-blue text-ic-text-primary">
                  <DocumentChartBarIcon className="h-6 w-6" />
                </div>
                <p className="ml-16 text-lg leading-6 font-medium text-ic-text-primary">
                  Financial Data
                </p>
                <p className="mt-2 ml-16 text-base text-ic-text-muted">
                  Real-time and historical data for stocks, bonds, commodities, and economic
                  indicators.
                </p>
              </div>

              <div className="relative">
                <div className="absolute flex items-center justify-center h-12 w-12 rounded-md bg-ic-blue text-ic-text-primary">
                  <ArrowTrendingUpIcon className="h-6 w-6" />
                </div>
                <p className="ml-16 text-lg leading-6 font-medium text-ic-text-primary">
                  Market Analytics
                </p>
                <p className="mt-2 ml-16 text-base text-ic-text-muted">
                  Comprehensive market analysis, sector performance, and trend identification tools.
                </p>
              </div>

              <div className="relative">
                <div className="absolute flex items-center justify-center h-12 w-12 rounded-md bg-ic-blue text-ic-text-primary">
                  <CurrencyDollarIcon className="h-6 w-6" />
                </div>
                <p className="ml-16 text-lg leading-6 font-medium text-ic-text-primary">
                  Watch Lists
                </p>
                <p className="mt-2 ml-16 text-base text-ic-text-muted">
                  Create custom watch lists, track real-time prices, and set target price alerts for
                  your favorite stocks and crypto.
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Footer */}
      <Footer />
    </div>
  );
}
