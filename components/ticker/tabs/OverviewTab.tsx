'use client';

import HybridChart from '@/components/ticker/HybridChart';
import NewsSentiment from '@/components/ticker/NewsSentiment';

interface OverviewTabProps {
  symbol: string;
  chartData: any;
  currentPrice: number;
}

export default function OverviewTab({ symbol, chartData, currentPrice }: OverviewTabProps) {
  return (
    <div className="space-y-6 p-6">
      {/* Price Chart */}
      <div className="bg-ic-surface rounded-lg -mx-6 -mt-6 mb-6">
        <HybridChart symbol={symbol} initialData={chartData} currentPrice={currentPrice} />
      </div>

      {/* News & Sentiment - Full Width */}
      <NewsSentiment symbol={symbol} />
    </div>
  );
}
