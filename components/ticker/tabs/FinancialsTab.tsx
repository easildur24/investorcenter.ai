'use client';

import { useState, useEffect } from 'react';
import { StatementType, Timeframe, statementTypeLabels } from '@/types/financials';
import { FinancialsMetadata } from '@/types/financials';
import { getAllFinancials } from '@/lib/api/financials';
import StatementTabs from '@/components/financials/StatementTabs';
import TimeframePicker from '@/components/financials/TimeframePicker';
import FinancialTable from '@/components/financials/FinancialTable';

interface FinancialsTabProps {
  symbol: string;
}

export default function FinancialsTab({ symbol }: FinancialsTabProps) {
  const [activeTab, setActiveTab] = useState<StatementType>('income');
  const [timeframe, setTimeframe] = useState<Timeframe>('quarterly');
  const [metadata, setMetadata] = useState<FinancialsMetadata | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchMetadata = async () => {
      try {
        setLoading(true);
        setError(null);
        const response = await getAllFinancials(symbol, { limit: 1 });
        if (response) {
          setMetadata(response.metadata);
        }
      } catch (err) {
        console.error('Error fetching metadata:', err);
        setError('Failed to load financial data');
      } finally {
        setLoading(false);
      }
    };

    fetchMetadata();
  }, [symbol]);

  if (loading) {
    return (
      <div className="p-6 animate-pulse">
        <div className="h-6 bg-ic-bg-tertiary rounded w-48 mb-6"></div>
        <div className="flex gap-2 mb-6">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="h-10 bg-ic-bg-tertiary rounded w-28"></div>
          ))}
        </div>
        <div className="space-y-4">
          {[1, 2, 3, 4, 5, 6].map((i) => (
            <div key={i} className="flex justify-between">
              <div className="h-4 bg-ic-bg-tertiary rounded w-32"></div>
              <div className="h-4 bg-ic-bg-tertiary rounded w-24"></div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
        <div>
          <h3 className="text-lg font-semibold text-ic-text-primary">Financial Statements</h3>
          {metadata?.company_name && (
            <p className="text-sm text-ic-text-dim mt-1">{metadata.company_name}</p>
          )}
        </div>
        <TimeframePicker value={timeframe} onChange={setTimeframe} />
      </div>

      {/* Statement Type Tabs */}
      <div className="mb-6">
        <StatementTabs activeTab={activeTab} onChange={setActiveTab} />
      </div>

      {/* Statement Description */}
      <div className="mb-4">
        <h4 className="text-base font-medium text-ic-text-primary">
          {statementTypeLabels[activeTab]}
        </h4>
        <p className="text-sm text-ic-text-dim mt-1">
          {getStatementDescription(activeTab)}
        </p>
      </div>

      {/* Financial Table */}
      <FinancialTable
        ticker={symbol}
        statementType={activeTab}
        timeframe={timeframe}
        showYoY={true}
        limit={8}
      />

      {/* Info Footer */}
      <div className="mt-6 p-4 bg-blue-50 rounded-lg">
        <h4 className="text-sm font-medium text-blue-900 mb-1">About This Data</h4>
        <p className="text-sm text-blue-700">
          Financial data is sourced from SEC EDGAR filings via Polygon.io.
          Includes 10-K (annual) and 10-Q (quarterly) filings.
          All values are in USD.
        </p>
      </div>
    </div>
  );
}

function getStatementDescription(statementType: StatementType): string {
  switch (statementType) {
    case 'income':
      return 'Revenue, expenses, and profitability over the reporting period';
    case 'balance_sheet':
      return 'Assets, liabilities, and equity at a point in time';
    case 'cash_flow':
      return 'Cash generated and used in operating, investing, and financing activities';
    case 'ratios':
      return 'Key financial ratios for valuation, profitability, and efficiency';
    default:
      return '';
  }
}
