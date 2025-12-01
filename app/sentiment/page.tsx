'use client';

import { useState, useEffect, Suspense } from 'react';
import { useSearchParams, useRouter } from 'next/navigation';
import TrendingTickersList from '@/components/sentiment/TrendingTickersList';
import SentimentCard from '@/components/sentiment/SentimentCard';
import SentimentHistoryChart from '@/components/sentiment/SentimentHistoryChart';
import PostsList from '@/components/sentiment/PostsList';

function SentimentDashboardContent() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const [selectedTicker, setSelectedTicker] = useState<string | null>(
    searchParams.get('ticker')
  );
  const [searchInput, setSearchInput] = useState('');

  // Update URL when ticker changes
  useEffect(() => {
    if (selectedTicker) {
      router.push(`/sentiment?ticker=${selectedTicker}`, { scroll: false });
    } else {
      router.push('/sentiment', { scroll: false });
    }
  }, [selectedTicker, router]);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    if (searchInput.trim()) {
      setSelectedTicker(searchInput.trim().toUpperCase());
      setSearchInput('');
    }
  };

  const handleTickerSelect = (ticker: string) => {
    setSelectedTicker(ticker);
  };

  const handleClearSelection = () => {
    setSelectedTicker(null);
  };

  return (
    <div className="min-h-screen bg-ic-bg-primary">
      {/* Header */}
      <div className="bg-ic-surface border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <div className="flex items-center gap-3 mb-2">
            <span className="text-3xl">📊</span>
            <h1 className="text-3xl font-bold text-ic-text-primary">Social Sentiment</h1>
          </div>
          <p className="text-ic-text-muted mt-2">
            Track market sentiment from Reddit discussions across r/wallstreetbets, r/stocks, and more
          </p>
        </div>
      </div>

      {/* Main content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Search bar */}
        <div className="bg-ic-surface rounded-lg border border-ic-border p-4 mb-6">
          <form onSubmit={handleSearch} className="flex gap-3">
            <input
              type="text"
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              placeholder="Search ticker (e.g., AAPL, TSLA, GME)"
              className="flex-1 px-4 py-2 border border-ic-border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
            <button
              type="submit"
              className="px-6 py-2 bg-ic-blue text-ic-text-primary rounded-lg hover:bg-ic-blue-hover transition-colors"
            >
              Search
            </button>
            {selectedTicker && (
              <button
                type="button"
                onClick={handleClearSelection}
                className="px-4 py-2 border border-ic-border rounded-lg text-ic-text-muted hover:bg-ic-surface-hover transition-colors"
              >
                Clear
              </button>
            )}
          </form>
        </div>

        {/* Content grid */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Left column - Trending or Selected Ticker */}
          <div className="lg:col-span-2">
            {selectedTicker ? (
              <div className="space-y-6">
                {/* Selected ticker header */}
                <div className="flex items-center justify-between">
                  <h2 className="text-xl font-semibold text-ic-text-primary">
                    {selectedTicker} Sentiment Analysis
                  </h2>
                  <button
                    onClick={handleClearSelection}
                    className="text-sm text-ic-text-muted hover:text-ic-text-secondary"
                  >
                    ← Back to Trending
                  </button>
                </div>

                {/* Sentiment card */}
                <SentimentCard ticker={selectedTicker} variant="full" />

                {/* History chart */}
                <div className="bg-ic-surface rounded-lg border border-ic-border p-6">
                  <SentimentHistoryChart
                    ticker={selectedTicker}
                    initialDays={30}
                    height={350}
                    showPostCount={true}
                  />
                </div>

                {/* Posts list */}
                <PostsList ticker={selectedTicker} limit={15} />
              </div>
            ) : (
              <TrendingTickersList
                onTickerSelect={handleTickerSelect}
                limit={25}
              />
            )}
          </div>

          {/* Right column - Quick stats or secondary info */}
          <div className="space-y-6">
            {/* Quick search suggestions */}
            {!selectedTicker && (
              <div className="bg-ic-surface rounded-lg border border-ic-border p-6">
                <h3 className="text-lg font-semibold text-ic-text-primary mb-4">
                  Popular Tickers
                </h3>
                <div className="flex flex-wrap gap-2">
                  {['GME', 'AMC', 'AAPL', 'TSLA', 'NVDA', 'SPY', 'QQQ', 'PLTR'].map(
                    (ticker) => (
                      <button
                        key={ticker}
                        onClick={() => setSelectedTicker(ticker)}
                        className="px-3 py-1.5 text-sm bg-ic-bg-secondary text-ic-text-secondary rounded-lg hover:bg-ic-surface-hover transition-colors"
                      >
                        {ticker}
                      </button>
                    )
                  )}
                </div>
              </div>
            )}

            {/* Info card */}
            <div className="bg-ic-surface rounded-lg border border-ic-border p-6">
              <h3 className="text-lg font-semibold text-ic-text-primary mb-4">
                About Sentiment Analysis
              </h3>
              <div className="space-y-4 text-sm text-ic-text-muted">
                <p>
                  Our sentiment analysis tracks discussions across major financial
                  subreddits to gauge market sentiment for stocks.
                </p>
                <div>
                  <h4 className="font-medium text-ic-text-primary mb-2">Sentiment Scale</h4>
                  <ul className="space-y-1.5">
                    <li className="flex items-center gap-2">
                      <span className="w-3 h-3 bg-ic-positive rounded-full" />
                      <span>Bullish: Score {'>'}= 0.2</span>
                    </li>
                    <li className="flex items-center gap-2">
                      <span className="w-3 h-3 bg-ic-text-dim rounded-full" />
                      <span>Neutral: -0.2 to 0.2</span>
                    </li>
                    <li className="flex items-center gap-2">
                      <span className="w-3 h-3 bg-ic-negative rounded-full" />
                      <span>Bearish: Score {'<'}= -0.2</span>
                    </li>
                  </ul>
                </div>
                <div>
                  <h4 className="font-medium text-ic-text-primary mb-2">Data Sources</h4>
                  <ul className="space-y-1">
                    <li>• r/wallstreetbets</li>
                    <li>• r/stocks</li>
                    <li>• r/investing</li>
                    <li>• r/options</li>
                  </ul>
                </div>
              </div>
            </div>

            {/* Disclaimer */}
            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 text-sm text-yellow-800">
              <strong>Disclaimer:</strong> Social sentiment data is for
              informational purposes only and should not be considered financial
              advice. Always do your own research before making investment
              decisions.
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default function SentimentDashboard() {
  return (
    <Suspense fallback={<DashboardSkeleton />}>
      <SentimentDashboardContent />
    </Suspense>
  );
}

function DashboardSkeleton() {
  return (
    <div className="min-h-screen bg-ic-bg-primary">
      <div className="bg-ic-surface border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <div className="h-10 w-64 bg-ic-bg-secondary rounded animate-pulse" />
          <div className="h-6 w-96 bg-ic-bg-secondary rounded mt-2 animate-pulse" />
        </div>
      </div>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="h-14 bg-ic-bg-secondary rounded-lg mb-6 animate-pulse" />
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 h-96 bg-ic-bg-secondary rounded-lg animate-pulse" />
          <div className="h-64 bg-ic-bg-secondary rounded-lg animate-pulse" />
        </div>
      </div>
    </div>
  );
}
