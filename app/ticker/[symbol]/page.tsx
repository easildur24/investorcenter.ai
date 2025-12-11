import { Suspense } from 'react';
import TickerTabs, { TabSkeleton } from '@/components/ticker/TickerTabs';
import TickerFundamentals from '@/components/ticker/TickerFundamentals';
import TickerAnalysts from '@/components/ticker/TickerAnalysts';
import RealTimePriceHeader from '@/components/ticker/RealTimePriceHeader';
import CryptoTickerHeader from '@/components/ticker/CryptoTickerHeader';
import CryptoMainContent from '@/components/ticker/CryptoMainContent';
import ICScoreCard from '@/components/ic-score/ICScoreCard';
import SentimentCard from '@/components/sentiment/SentimentCard';
import OverviewTab from '@/components/ticker/tabs/OverviewTab';
import TechnicalTab from '@/components/ticker/tabs/TechnicalTab';
import RiskTab from '@/components/ticker/tabs/RiskTab';
import FinancialsTab from '@/components/ticker/tabs/FinancialsTab';
import OwnershipTab from '@/components/ticker/tabs/OwnershipTab';

interface PageProps {
  params: {
    symbol: string;
  };
  searchParams: {
    period?: string;
  };
}

// Force dynamic rendering and disable all caching
export const dynamic = 'force-dynamic';
export const revalidate = 0;

// Fetch ticker data server-side to avoid client hydration issues
async function getTickerData(symbol: string) {
  try {
    // Use internal backend service URL for server-side fetching
    const response = await fetch(`http://investorcenter-backend-service.investorcenter.svc.cluster.local:8080/api/v1/tickers/${symbol}`, {
      cache: 'no-store', // Always fetch fresh data
      next: { revalidate: 0 }, // Disable Next.js caching completely
    });

    if (!response.ok) {
      console.error(`❌ Failed to fetch ticker data: ${response.status} for ${symbol}`);
      return null;
    }

    const result = await response.json();
    const price = result.data?.summary?.price?.price;
    console.log(`✅ Server-side fetched ${symbol}: $${price || 'no price'} (${result.data?.summary?.stock?.assetType || 'unknown'})`);
    return result.data;
  } catch (error) {
    console.error(`❌ Error fetching ticker data server-side for ${symbol}:`, error);
    return null;
  }
}

// Fetch chart data server-side
async function getChartData(symbol: string, period: string = '1Y') {
  try {
    const response = await fetch(`http://investorcenter-backend-service.investorcenter.svc.cluster.local:8080/api/v1/tickers/${symbol}/chart?period=${period}`, {
      cache: 'no-store',
      next: { revalidate: 0 }, // Disable Next.js caching completely
    });

    if (!response.ok) {
      console.error(`❌ Failed to fetch chart data: ${response.status} for ${symbol}`);
      return null;
    }

    const result = await response.json();
    console.log(`✅ Server-side fetched chart data for ${symbol} (${period})`);
    return result.data;
  } catch (error) {
    console.error(`❌ Error fetching chart data server-side for ${symbol}:`, error);
    return null;
  }
}

// Tab configuration with icons
const stockTabs = [
  { id: 'overview', label: 'Overview' },
  { id: 'technical', label: 'Technical' },
  { id: 'risk', label: 'Risk' },
  { id: 'financials', label: 'Financials' },
  { id: 'ownership', label: 'Ownership' },
];

export default async function TickerPage({ params, searchParams }: PageProps) {
  const symbol = decodeURIComponent(params.symbol).toUpperCase();
  const period = searchParams.period || '1Y';

  // Fetch data server-side
  const [tickerData, chartData] = await Promise.all([
    getTickerData(symbol),
    getChartData(symbol, period)
  ]);

  if (!tickerData) {
    return (
      <div className="min-h-screen bg-ic-bg-primary flex items-center justify-center">
        <div className="bg-red-50 border border-red-200 rounded-lg p-6">
          <h2 className="text-red-800 font-semibold">Failed to Load Data</h2>
          <p className="text-ic-negative mt-2">Could not fetch data for {symbol}</p>
        </div>
      </div>
    );
  }

  const currentPrice = parseFloat(tickerData.summary.price.price);

  return (
    <div className="min-h-screen bg-ic-bg-primary">
      {/* Ticker Overview Header with Real-time Updates */}
      <div className="bg-ic-surface shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          {tickerData.summary.stock.isCrypto ? (
            <CryptoTickerHeader symbol={symbol} initialData={tickerData.summary} />
          ) : (
            <RealTimePriceHeader symbol={symbol} initialData={tickerData.summary} />
          )}
        </div>
      </div>

      {/* Main Content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {tickerData.summary.stock.isCrypto ? (
          /* CoinMarketCap-style crypto layout */
          <CryptoMainContent
            symbol={symbol}
            cryptoName={tickerData.summary.stock.name.split(' - ')[0]}
          />
        ) : (
          /* Stock layout with tabbed interface */
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
            {/* Left Column - Tabbed Content */}
            <div className="lg:col-span-2">
              <Suspense fallback={<TabSkeleton />}>
                <TickerTabs symbol={symbol} tabs={stockTabs} defaultTab="overview">
                  {/* Overview Tab */}
                  <OverviewTab
                    symbol={symbol}
                    chartData={chartData}
                    currentPrice={currentPrice}
                  />

                  {/* Technical Tab */}
                  <TechnicalTab symbol={symbol} />

                  {/* Risk Tab */}
                  <RiskTab symbol={symbol} />

                  {/* Financials Tab */}
                  <FinancialsTab symbol={symbol} />

                  {/* Ownership Tab */}
                  <OwnershipTab symbol={symbol} />
                </TickerTabs>
              </Suspense>
            </div>

            {/* Right Column - IC Score, Sentiment, Fundamentals and Analysis */}
            <div className="space-y-8">
              {/* IC Score Analysis - at top for visibility */}
              <Suspense fallback={<ICScoreSkeleton />}>
                <ICScoreCard ticker={symbol} variant="compact" />
              </Suspense>

              {/* Social Sentiment Analysis */}
              <Suspense fallback={<SentimentSkeleton />}>
                <SentimentCard ticker={symbol} variant="compact" />
              </Suspense>

              {/* Comprehensive Key Metrics */}
              <div className="bg-ic-surface rounded-lg shadow">
                <TickerFundamentals symbol={symbol} />
              </div>

              {/* Analyst Ratings */}
              <div className="bg-ic-surface rounded-lg shadow">
                <TickerAnalysts symbol={symbol} />
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// Loading Skeletons
function ICScoreSkeleton() {
  return (
    <div className="bg-ic-surface rounded-xl shadow-lg border border-ic-border overflow-hidden animate-pulse">
      <div className="bg-ic-bg-secondary h-24"></div>
      <div className="p-6 space-y-4">
        <div className="h-64 bg-ic-bg-secondary rounded"></div>
        <div className="h-64 bg-ic-bg-secondary rounded"></div>
      </div>
    </div>
  );
}

function SentimentSkeleton() {
  return (
    <div className="bg-ic-surface rounded-lg shadow border border-ic-border p-6 animate-pulse">
      <div className="h-6 w-32 bg-ic-bg-secondary rounded mb-4"></div>
      <div className="h-20 bg-ic-bg-secondary rounded mb-4"></div>
      <div className="h-3 bg-ic-bg-secondary rounded mb-4"></div>
      <div className="grid grid-cols-2 gap-4">
        <div className="h-12 bg-ic-bg-secondary rounded"></div>
        <div className="h-12 bg-ic-bg-secondary rounded"></div>
      </div>
    </div>
  );
}
