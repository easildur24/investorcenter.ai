import { Suspense } from 'react';
import TickerOverview from '@/components/ticker/TickerOverview';
import TickerChart from '@/components/ticker/TickerChart';
import TickerFundamentals from '@/components/ticker/TickerFundamentals';
import TickerNews from '@/components/ticker/TickerNews';
import TickerEarnings from '@/components/ticker/TickerEarnings';
import TickerAnalysts from '@/components/ticker/TickerAnalysts';

interface PageProps {
  params: {
    symbol: string;
  };
}

export default function TickerPage({ params }: PageProps) {
  const symbol = params.symbol.toUpperCase();

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <Suspense fallback={<TickerHeaderSkeleton />}>
            <TickerOverview symbol={symbol} />
          </Suspense>
        </div>
      </div>

      {/* Main Content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Left Column - Chart and News */}
          <div className="lg:col-span-2 space-y-8">
            {/* Price Chart */}
            <div className="bg-white rounded-lg shadow">
              <Suspense fallback={<ChartSkeleton />}>
                <TickerChart symbol={symbol} />
              </Suspense>
            </div>

            {/* News & Analysis */}
            <div className="bg-white rounded-lg shadow">
              <Suspense fallback={<NewsSkeleton />}>
                <TickerNews symbol={symbol} />
              </Suspense>
            </div>

            {/* Earnings */}
            <div className="bg-white rounded-lg shadow">
              <Suspense fallback={<EarningsSkeleton />}>
                <TickerEarnings symbol={symbol} />
              </Suspense>
            </div>
          </div>

          {/* Right Column - Fundamentals and Analysis */}
          <div className="space-y-8">
            {/* Key Metrics */}
            <div className="bg-white rounded-lg shadow">
              <Suspense fallback={<FundamentalsSkeleton />}>
                <TickerFundamentals symbol={symbol} />
              </Suspense>
            </div>

            {/* Analyst Ratings */}
            <div className="bg-white rounded-lg shadow">
              <Suspense fallback={<AnalystsSkeleton />}>
                <TickerAnalysts symbol={symbol} />
              </Suspense>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// Loading Skeletons
function TickerHeaderSkeleton() {
  return (
    <div className="animate-pulse">
      <div className="flex items-center justify-between">
        <div>
          <div className="h-8 bg-gray-200 rounded w-48 mb-2"></div>
          <div className="h-4 bg-gray-200 rounded w-32"></div>
        </div>
        <div className="text-right">
          <div className="h-8 bg-gray-200 rounded w-24 mb-2"></div>
          <div className="h-4 bg-gray-200 rounded w-20"></div>
        </div>
      </div>
    </div>
  );
}

function ChartSkeleton() {
  return (
    <div className="p-6">
      <div className="h-6 bg-gray-200 rounded w-32 mb-4"></div>
      <div className="h-80 bg-gray-200 rounded"></div>
    </div>
  );
}

function NewsSkeleton() {
  return (
    <div className="p-6">
      <div className="h-6 bg-gray-200 rounded w-40 mb-4"></div>
      <div className="space-y-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="border-b border-gray-100 pb-4">
            <div className="h-4 bg-gray-200 rounded w-3/4 mb-2"></div>
            <div className="h-3 bg-gray-200 rounded w-full mb-1"></div>
            <div className="h-3 bg-gray-200 rounded w-2/3"></div>
          </div>
        ))}
      </div>
    </div>
  );
}

function EarningsSkeleton() {
  return (
    <div className="p-6">
      <div className="h-6 bg-gray-200 rounded w-32 mb-4"></div>
      <div className="space-y-3">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="flex justify-between items-center">
            <div className="h-4 bg-gray-200 rounded w-20"></div>
            <div className="h-4 bg-gray-200 rounded w-16"></div>
            <div className="h-4 bg-gray-200 rounded w-16"></div>
          </div>
        ))}
      </div>
    </div>
  );
}

function FundamentalsSkeleton() {
  return (
    <div className="p-6">
      <div className="h-6 bg-gray-200 rounded w-32 mb-4"></div>
      <div className="space-y-3">
        {[1, 2, 3, 4, 5, 6].map((i) => (
          <div key={i} className="flex justify-between">
            <div className="h-4 bg-gray-200 rounded w-24"></div>
            <div className="h-4 bg-gray-200 rounded w-16"></div>
          </div>
        ))}
      </div>
    </div>
  );
}

function AnalystsSkeleton() {
  return (
    <div className="p-6">
      <div className="h-6 bg-gray-200 rounded w-32 mb-4"></div>
      <div className="space-y-3">
        <div className="h-8 bg-gray-200 rounded w-full"></div>
        <div className="h-4 bg-gray-200 rounded w-3/4"></div>
        <div className="h-4 bg-gray-200 rounded w-2/3"></div>
      </div>
    </div>
  );
}
