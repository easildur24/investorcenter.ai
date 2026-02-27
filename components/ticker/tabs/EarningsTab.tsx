'use client';

import { useState, useEffect } from 'react';
import { API_BASE_URL } from '@/lib/api';
import { stocks } from '@/lib/api/routes';
import type { EarningsResponse } from '@/lib/types/earnings';
import NextEarningsCard from './earnings/NextEarningsCard';
import EarningsSummaryStats from './earnings/EarningsSummaryStats';
import EarningsTable from './earnings/EarningsTable';
import EarningsBarChart from './earnings/EarningsBarChart';

interface EarningsTabProps {
  symbol: string;
}

export default function EarningsTab({ symbol }: EarningsTabProps) {
  const [data, setData] = useState<EarningsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchEarnings = async () => {
      try {
        setLoading(true);
        setError(null);
        const response = await fetch(`${API_BASE_URL}${stocks.earnings(symbol.toUpperCase())}`, {
          cache: 'no-store',
        });
        if (response.status === 404) {
          setError('No earnings data available for this ticker');
          return;
        }
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`);
        }
        const result = await response.json();
        setData(result.data);
      } catch (err) {
        console.error('Error fetching earnings:', err);
        setError('Failed to load earnings data');
      } finally {
        setLoading(false);
      }
    };

    fetchEarnings();
  }, [symbol]);

  if (loading) return <EarningsTabSkeleton />;
  if (error || !data) {
    return (
      <div className="p-6">
        <div className="text-center py-12">
          <p className="text-ic-text-muted text-lg">{error || 'No earnings data available'}</p>
          <p className="text-ic-text-dim text-sm mt-2">
            Earnings data may not be available for all tickers.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      <NextEarningsCard
        nextEarnings={data.nextEarnings}
        mostRecentEarnings={data.mostRecentEarnings}
      />
      <EarningsSummaryStats beatRate={data.beatRate} earnings={data.earnings} />
      <EarningsTable earnings={data.earnings} />
      <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
        <EarningsBarChart data={data.earnings} type="eps" />
        <EarningsBarChart data={data.earnings} type="revenue" />
      </div>
    </div>
  );
}

function EarningsTabSkeleton() {
  return (
    <div className="p-6 animate-pulse space-y-6">
      {/* Next Earnings Card */}
      <div className="bg-ic-bg-secondary rounded-lg p-6">
        <div className="h-5 bg-ic-bg-tertiary rounded w-48 mb-3" />
        <div className="h-6 bg-ic-bg-tertiary rounded w-64 mb-2" />
        <div className="flex gap-8">
          <div className="h-5 bg-ic-bg-tertiary rounded w-24" />
          <div className="h-5 bg-ic-bg-tertiary rounded w-24" />
        </div>
      </div>
      {/* Beat Rate Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="bg-ic-bg-secondary rounded-lg p-4">
            <div className="h-3 bg-ic-bg-tertiary rounded w-20 mb-2" />
            <div className="h-6 bg-ic-bg-tertiary rounded w-16" />
          </div>
        ))}
      </div>
      {/* Table */}
      <div className="bg-ic-bg-secondary rounded-lg p-4">
        {[1, 2, 3, 4, 5].map((i) => (
          <div key={i} className="h-8 bg-ic-bg-tertiary rounded mb-2" />
        ))}
      </div>
      {/* Charts */}
      <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
        <div className="bg-ic-bg-secondary rounded-lg h-80" />
        <div className="bg-ic-bg-secondary rounded-lg h-80" />
      </div>
    </div>
  );
}
