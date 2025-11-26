'use client';

import HybridChart from '@/components/ticker/HybridChart';
import TickerNews from '@/components/ticker/TickerNews';
import TickerEarnings from '@/components/ticker/TickerEarnings';

interface OverviewTabProps {
  symbol: string;
  chartData: any;
  currentPrice: number;
}

export default function OverviewTab({ symbol, chartData, currentPrice }: OverviewTabProps) {
  return (
    <div className="space-y-6 p-6">
      {/* Price Chart */}
      <div className="bg-gray-50 rounded-lg -mx-6 -mt-6 mb-6">
        <HybridChart
          symbol={symbol}
          initialData={chartData}
          currentPrice={currentPrice}
        />
      </div>

      {/* News & Earnings Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* News */}
        <div className="bg-gray-50 rounded-lg">
          <TickerNews symbol={symbol} />
        </div>

        {/* Earnings */}
        <div className="bg-gray-50 rounded-lg">
          <TickerEarnings symbol={symbol} />
        </div>
      </div>
    </div>
  );
}
