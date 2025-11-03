import { ChartBarIcon, ArrowTrendingUpIcon, DocumentChartBarIcon, CurrencyDollarIcon } from '@heroicons/react/24/outline'
import MarketOverview from '@/components/MarketOverview'

export default function Home() {
  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-blue-50">

      {/* Hero Section */}
      <div className="relative overflow-hidden">
        <div className="max-w-7xl mx-auto">
          <div className="relative z-10 pb-8 sm:pb-16 md:pb-20 lg:max-w-2xl lg:w-full lg:pb-28 xl:pb-32">
            <main className="mt-10 mx-auto max-w-7xl px-4 sm:mt-12 sm:px-6 md:mt-16 lg:mt-20 lg:px-8 xl:mt-28">
              <div className="sm:text-center lg:text-left">
                <h1 className="text-4xl tracking-tight font-extrabold text-gray-900 sm:text-5xl md:text-6xl">
                  <span className="block xl:inline">Professional</span>{' '}
                  <span className="block text-primary-600 xl:inline">Financial Analytics</span>
                </h1>
                <p className="mt-3 text-base text-gray-500 sm:mt-5 sm:text-lg sm:max-w-xl sm:mx-auto md:mt-5 md:text-xl lg:mx-0">
                  Access comprehensive financial data, interactive charts, and powerful analytics tools. 
                  Make informed investment decisions with institutional-grade research and insights.
                </p>
                <div className="mt-5 sm:mt-8 sm:flex sm:justify-center lg:justify-start">
                  <div className="rounded-md shadow">
                    <button className="w-full flex items-center justify-center px-8 py-3 border border-transparent text-base font-medium rounded-md text-white bg-primary-600 hover:bg-primary-700 md:py-4 md:text-lg md:px-10">
                      Start Free Trial
                    </button>
                  </div>
                  <div className="mt-3 sm:mt-0 sm:ml-3">
                    <button className="w-full flex items-center justify-center px-8 py-3 border border-transparent text-base font-medium rounded-md text-primary-700 bg-primary-100 hover:bg-primary-200 md:py-4 md:text-lg md:px-10">
                      Watch Demo
                    </button>
                  </div>
                </div>
              </div>
            </main>
          </div>
        </div>
        <div className="lg:absolute lg:inset-y-0 lg:right-0 lg:w-1/2">
          <div className="h-56 w-full bg-gradient-to-r from-primary-400 to-primary-600 sm:h-72 md:h-96 lg:w-full lg:h-full flex items-center justify-center">
            <div className="text-white text-center">
              <ArrowTrendingUpIcon className="h-24 w-24 mx-auto mb-4 opacity-80" />
              <p className="text-lg font-medium">Interactive Charts & Analytics</p>
            </div>
          </div>
        </div>
      </div>

      {/* Market Overview Section */}
      <div className="py-12 bg-gray-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="lg:text-center mb-8">
            <h2 className="text-base text-primary-600 font-semibold tracking-wide uppercase">Live Data</h2>
            <p className="mt-2 text-3xl leading-8 font-extrabold tracking-tight text-gray-900 sm:text-4xl">
              Real-time market insights
            </p>
          </div>
          
          <div className="max-w-3xl mx-auto">
            <MarketOverview />
          </div>
        </div>
      </div>

      {/* Features Section */}
      <div className="py-12 bg-white">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="lg:text-center">
            <h2 className="text-base text-primary-600 font-semibold tracking-wide uppercase">Features</h2>
            <p className="mt-2 text-3xl leading-8 font-extrabold tracking-tight text-gray-900 sm:text-4xl">
              Everything you need for investment research
            </p>
            <p className="mt-4 max-w-2xl text-xl text-gray-500 lg:mx-auto">
              Professional-grade tools and data to power your investment decisions
            </p>
          </div>

          <div className="mt-10">
            <div className="space-y-10 md:space-y-0 md:grid md:grid-cols-2 md:gap-x-8 md:gap-y-10">
              <div className="relative">
                <div className="absolute flex items-center justify-center h-12 w-12 rounded-md bg-primary-500 text-white">
                  <ChartBarIcon className="h-6 w-6" />
                </div>
                <p className="ml-16 text-lg leading-6 font-medium text-gray-900">Interactive Charts</p>
                <p className="mt-2 ml-16 text-base text-gray-500">
                  Advanced charting tools with technical indicators, multiple timeframes, and customizable layouts.
                </p>
              </div>

              <div className="relative">
                <div className="absolute flex items-center justify-center h-12 w-12 rounded-md bg-primary-500 text-white">
                  <DocumentChartBarIcon className="h-6 w-6" />
                </div>
                <p className="ml-16 text-lg leading-6 font-medium text-gray-900">Financial Data</p>
                <p className="mt-2 ml-16 text-base text-gray-500">
                  Real-time and historical data for stocks, bonds, commodities, and economic indicators.
                </p>
              </div>

              <div className="relative">
                <div className="absolute flex items-center justify-center h-12 w-12 rounded-md bg-primary-500 text-white">
                  <ArrowTrendingUpIcon className="h-6 w-6" />
                </div>
                <p className="ml-16 text-lg leading-6 font-medium text-gray-900">Market Analytics</p>
                <p className="mt-2 ml-16 text-base text-gray-500">
                  Comprehensive market analysis, sector performance, and trend identification tools.
                </p>
              </div>

              <div className="relative">
                <div className="absolute flex items-center justify-center h-12 w-12 rounded-md bg-primary-500 text-white">
                  <CurrencyDollarIcon className="h-6 w-6" />
                </div>
                <p className="ml-16 text-lg leading-6 font-medium text-gray-900">Watch Lists</p>
                <p className="mt-2 ml-16 text-base text-gray-500">
                  Create custom watch lists, track real-time prices, and set target price alerts for your favorite stocks and crypto.
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Footer */}
      <footer className="bg-gray-50">
        <div className="max-w-7xl mx-auto py-12 px-4 sm:px-6 lg:py-16 lg:px-8">
          <div className="xl:grid xl:grid-cols-3 xl:gap-8">
            <div className="space-y-8 xl:col-span-1">
              <div className="flex items-center">
                <ChartBarIcon className="h-8 w-8 text-primary-600" />
                <span className="ml-2 text-xl font-bold text-gray-900">InvestorCenter</span>
              </div>
              <p className="text-gray-500 text-base">
                Professional financial data and analytics platform for informed investment decisions.
              </p>
            </div>
            <div className="mt-12 grid grid-cols-2 gap-8 xl:mt-0 xl:col-span-2">
              <div className="md:grid md:grid-cols-2 md:gap-8">
                <div>
                  <h3 className="text-sm font-semibold text-gray-400 tracking-wider uppercase">
                    Platform
                  </h3>
                  <ul className="mt-4 space-y-4">
                    <li><a href="#" className="text-base text-gray-500 hover:text-gray-900">Charts</a></li>
                    <li><a href="#" className="text-base text-gray-500 hover:text-gray-900">Data</a></li>
                    <li><a href="#" className="text-base text-gray-500 hover:text-gray-900">Analytics</a></li>
                  </ul>
                </div>
                <div className="mt-12 md:mt-0">
                  <h3 className="text-sm font-semibold text-gray-400 tracking-wider uppercase">
                    Company
                  </h3>
                  <ul className="mt-4 space-y-4">
                    <li><a href="#" className="text-base text-gray-500 hover:text-gray-900">About</a></li>
                    <li><a href="#" className="text-base text-gray-500 hover:text-gray-900">Contact</a></li>
                    <li><a href="#" className="text-base text-gray-500 hover:text-gray-900">Privacy</a></li>
                  </ul>
                </div>
              </div>
            </div>
          </div>
          <div className="mt-12 border-t border-gray-200 pt-8">
            <p className="text-base text-gray-400 xl:text-center">
              &copy; 2024 InvestorCenter. All rights reserved.
            </p>
          </div>
        </div>
      </footer>
    </div>
  )
}
