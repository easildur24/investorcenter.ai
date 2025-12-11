'use client';

import { useState, useEffect } from 'react';
import { cn, safeToFixed, formatLargeNumber, formatPercent, safeParseNumber } from '@/lib/utils';

interface OwnershipTabProps {
  symbol: string;
}

interface InsiderTrade {
  transaction_date: string;
  insider_name: string;
  insider_title: string;
  transaction_type: string;
  shares: number;
  price_per_share: number;
  total_value: number;
  shares_owned_after: number;
}

interface InstitutionalHolder {
  holder_name: string;
  shares: number;
  value: number;
  percent_of_shares: number;
  change_shares?: number;
  change_percent?: number;
  report_date: string;
}

interface OwnershipData {
  insider_trades?: InsiderTrade[];
  institutional_holders?: InstitutionalHolder[];
  insider_ownership_percent?: number;
  institutional_ownership_percent?: number;
  total_shares_outstanding?: number;
  insider_net_activity_30d?: number;
  institutional_net_activity_90d?: number;
}

type ViewType = 'insider' | 'institutional';

export default function OwnershipTab({ symbol }: OwnershipTabProps) {
  const [data, setData] = useState<OwnershipData | null>(null);
  const [viewType, setViewType] = useState<ViewType>('insider');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        // Fetch both insider trades and institutional holdings
        const [insiderRes, institutionalRes] = await Promise.all([
          fetch(`/api/v1/stocks/${symbol}/insider-trades`).catch(() => null),
          fetch(`/api/v1/stocks/${symbol}/institutional-holdings`).catch(() => null)
        ]);

        const ownershipData: OwnershipData = {};

        if (insiderRes?.ok) {
          const insiderResult = await insiderRes.json();
          ownershipData.insider_trades = insiderResult.data?.trades || [];
          ownershipData.insider_ownership_percent = insiderResult.data?.insider_ownership_percent;
          ownershipData.insider_net_activity_30d = insiderResult.data?.net_activity_30d;
        }

        if (institutionalRes?.ok) {
          const institutionalResult = await institutionalRes.json();
          ownershipData.institutional_holders = institutionalResult.data?.holders || [];
          ownershipData.institutional_ownership_percent = institutionalResult.data?.institutional_ownership_percent;
          ownershipData.institutional_net_activity_90d = institutionalResult.data?.net_activity_90d;
          ownershipData.total_shares_outstanding = institutionalResult.data?.total_shares_outstanding;
        }

        setData(ownershipData);
      } catch (err) {
        console.error('Error fetching ownership data:', err);
        setError('Failed to load ownership data');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [symbol]);

  if (loading) {
    return (
      <div className="p-6 animate-pulse">
        <div className="h-6 bg-ic-bg-secondary rounded w-48 mb-6"></div>
        <div className="space-y-4">
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className="bg-ic-bg-secondary rounded-lg p-4">
              <div className="h-4 bg-ic-bg-secondary rounded w-3/4 mb-2"></div>
              <div className="h-4 bg-ic-bg-secondary rounded w-1/2"></div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="p-6">
        <h3 className="text-lg font-semibold text-ic-text-primary mb-4">Ownership</h3>
        <p className="text-ic-text-muted">{error || 'No ownership data available'}</p>
      </div>
    );
  }

  const renderSummaryCards = () => (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
      <div className="bg-ic-bg-secondary rounded-lg p-4">
        <div className="text-sm text-ic-text-muted mb-1">Insider Ownership</div>
        <div className="text-xl font-semibold text-ic-text-primary">
          {formatPercent(safeParseNumber(data.insider_ownership_percent))}
        </div>
      </div>
      <div className="bg-ic-bg-secondary rounded-lg p-4">
        <div className="text-sm text-ic-text-muted mb-1">Institutional Ownership</div>
        <div className="text-xl font-semibold text-ic-text-primary">
          {formatPercent(safeParseNumber(data.institutional_ownership_percent))}
        </div>
      </div>
      <div className="bg-ic-bg-secondary rounded-lg p-4">
        <div className="text-sm text-ic-text-muted mb-1">Insider Activity (30d)</div>
        <div className={cn(
          'text-xl font-semibold',
          safeParseNumber(data.insider_net_activity_30d) >= 0 ? 'text-ic-positive' : 'text-ic-negative'
        )}>
          {safeParseNumber(data.insider_net_activity_30d) >= 0 ? '+' : ''}
          {safeToFixed(safeParseNumber(data.insider_net_activity_30d) / 1000, 1)}K
        </div>
      </div>
      <div className="bg-ic-bg-secondary rounded-lg p-4">
        <div className="text-sm text-ic-text-muted mb-1">Institutional Activity (90d)</div>
        <div className={cn(
          'text-xl font-semibold',
          safeParseNumber(data.institutional_net_activity_90d) >= 0 ? 'text-ic-positive' : 'text-ic-negative'
        )}>
          {safeParseNumber(data.institutional_net_activity_90d) >= 0 ? '+' : ''}
          {formatLargeNumber(data.institutional_net_activity_90d)}
        </div>
      </div>
    </div>
  );

  const renderInsiderTrades = () => {
    const trades = data.insider_trades || [];

    if (trades.length === 0) {
      return (
        <div className="text-center py-8 text-ic-text-muted">
          No recent insider trades available
        </div>
      );
    }

    return (
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="text-left text-sm text-ic-text-muted border-b border-ic-border">
              <th className="pb-3 font-medium">Date</th>
              <th className="pb-3 font-medium">Insider</th>
              <th className="pb-3 font-medium">Type</th>
              <th className="pb-3 font-medium text-right">Shares</th>
              <th className="pb-3 font-medium text-right">Price</th>
              <th className="pb-3 font-medium text-right">Value</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-ic-border">
            {trades.map((trade, index) => (
              <tr key={index} className="hover:bg-ic-surface-hover">
                <td className="py-3 text-ic-text-primary">
                  {new Date(trade.transaction_date).toLocaleDateString()}
                </td>
                <td className="py-3">
                  <div className="text-ic-text-primary font-medium">{trade.insider_name}</div>
                  <div className="text-xs text-ic-text-muted">{trade.insider_title}</div>
                </td>
                <td className="py-3">
                  <span className={cn(
                    'px-2 py-0.5 rounded-full text-xs font-medium',
                    trade.transaction_type.toLowerCase().includes('buy') || trade.transaction_type.toLowerCase().includes('acquisition')
                      ? 'bg-green-100 text-green-700'
                      : 'bg-red-100 text-red-700'
                  )}>
                    {trade.transaction_type}
                  </span>
                </td>
                <td className="py-3 text-right font-medium text-ic-text-primary">
                  {trade.shares.toLocaleString()}
                </td>
                <td className="py-3 text-right text-ic-text-primary">
                  ${safeToFixed(trade.price_per_share, 2)}
                </td>
                <td className="py-3 text-right font-medium text-ic-text-primary">
                  {formatLargeNumber(trade.total_value)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  };

  const renderInstitutionalHolders = () => {
    const holders = data.institutional_holders || [];

    if (holders.length === 0) {
      return (
        <div className="text-center py-8 text-ic-text-muted">
          No institutional holdings data available
        </div>
      );
    }

    return (
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="text-left text-sm text-ic-text-muted border-b border-ic-border">
              <th className="pb-3 font-medium">Institution</th>
              <th className="pb-3 font-medium text-right">Shares</th>
              <th className="pb-3 font-medium text-right">Value</th>
              <th className="pb-3 font-medium text-right">% of Shares</th>
              <th className="pb-3 font-medium text-right">Change</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-ic-border">
            {holders.map((holder, index) => (
              <tr key={index} className="hover:bg-ic-surface-hover">
                <td className="py-3">
                  <div className="text-ic-text-primary font-medium">{holder.holder_name}</div>
                  <div className="text-xs text-ic-text-muted">
                    Filed: {new Date(holder.report_date).toLocaleDateString()}
                  </div>
                </td>
                <td className="py-3 text-right font-medium text-ic-text-primary">
                  {holder.shares.toLocaleString()}
                </td>
                <td className="py-3 text-right text-ic-text-primary">
                  {formatLargeNumber(holder.value)}
                </td>
                <td className="py-3 text-right text-ic-text-primary">
                  {formatPercent(holder.percent_of_shares)}
                </td>
                <td className="py-3 text-right">
                  {holder.change_shares !== undefined && (
                    <div className={cn(
                      'font-medium',
                      holder.change_shares >= 0 ? 'text-ic-positive' : 'text-ic-negative'
                    )}>
                      {holder.change_shares >= 0 ? '+' : ''}
                      {holder.change_shares.toLocaleString()}
                      {holder.change_percent !== undefined && (
                        <span className="text-xs ml-1">
                          ({holder.change_percent >= 0 ? '+' : ''}{safeToFixed(holder.change_percent, 1)}%)
                        </span>
                      )}
                    </div>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  };

  return (
    <div className="p-6">
      <h3 className="text-lg font-semibold text-ic-text-primary mb-6">Ownership Analysis</h3>

      {/* Summary Cards */}
      {renderSummaryCards()}

      {/* View Toggle */}
      <div className="flex gap-2 mb-6">
        <button
          onClick={() => setViewType('insider')}
          className={cn(
            'px-4 py-2 text-sm font-medium rounded-lg transition-colors',
            viewType === 'insider'
              ? 'bg-ic-blue text-ic-text-primary'
              : 'bg-ic-bg-secondary text-ic-text-muted hover:bg-ic-surface-hover'
          )}
        >
          Insider Trades
        </button>
        <button
          onClick={() => setViewType('institutional')}
          className={cn(
            'px-4 py-2 text-sm font-medium rounded-lg transition-colors',
            viewType === 'institutional'
              ? 'bg-ic-blue text-ic-text-primary'
              : 'bg-ic-bg-secondary text-ic-text-muted hover:bg-ic-surface-hover'
          )}
        >
          Institutional Holdings
        </button>
      </div>

      {/* Content */}
      {viewType === 'insider' ? renderInsiderTrades() : renderInstitutionalHolders()}
    </div>
  );
}
