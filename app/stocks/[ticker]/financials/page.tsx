'use client';

import { useState, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { cn } from '@/lib/utils';
import { StatementType, Timeframe, statementTypeLabels } from '@/types/financials';
import { FinancialsMetadata } from '@/types/financials';
import { getAllFinancials } from '@/lib/api/financials';
import StatementTabs from '@/components/financials/StatementTabs';
import TimeframePicker from '@/components/financials/TimeframePicker';
import FinancialTable from '@/components/financials/FinancialTable';

export default function FinancialsPage() {
  const params = useParams();
  const router = useRouter();
  const ticker = (params.ticker as string).toUpperCase();

  const [activeTab, setActiveTab] = useState<StatementType>('income');
  const [timeframe, setTimeframe] = useState<Timeframe>('quarterly');
  const [metadata, setMetadata] = useState<FinancialsMetadata | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchMetadata = async () => {
      try {
        setLoading(true);
        const response = await getAllFinancials(ticker, { limit: 1 });
        if (response) {
          setMetadata(response.metadata);
        }
      } catch (err) {
        console.error('Error fetching metadata:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchMetadata();
  }, [ticker]);

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          {/* Breadcrumb */}
          <nav className="flex items-center gap-2 text-sm text-gray-500 mb-4">
            <Link href="/" className="hover:text-gray-700">
              Home
            </Link>
            <span>/</span>
            <Link href={`/ticker/${ticker}`} className="hover:text-gray-700">
              {ticker}
            </Link>
            <span>/</span>
            <span className="text-gray-900">Financial Statements</span>
          </nav>

          {/* Title and Company Info */}
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">
                {ticker} Financial Statements
              </h1>
              {metadata?.company_name && (
                <p className="text-gray-500 mt-1">{metadata.company_name}</p>
              )}
              {metadata?.cik && (
                <p className="text-sm text-gray-400">CIK: {metadata.cik}</p>
              )}
            </div>

            {/* Back to Ticker Button */}
            <Link
              href={`/ticker/${ticker}`}
              className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
            >
              <svg
                className="w-4 h-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M10 19l-7-7m0 0l7-7m-7 7h18"
                />
              </svg>
              Back to {ticker}
            </Link>
          </div>
        </div>
      </div>

      {/* Controls */}
      <div className="bg-white border-b border-gray-200 sticky top-0 z-10">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <StatementTabs activeTab={activeTab} onChange={setActiveTab} />
            <TimeframePicker value={timeframe} onChange={setTimeframe} />
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <div className="mb-4">
          <h2 className="text-lg font-semibold text-gray-900">
            {statementTypeLabels[activeTab]}
          </h2>
          <p className="text-sm text-gray-500 mt-1">
            {getStatementDescription(activeTab)}
          </p>
        </div>

        <FinancialTable
          ticker={ticker}
          statementType={activeTab}
          timeframe={timeframe}
          showYoY={true}
          limit={8}
        />

        {/* Info Section */}
        <div className="mt-8 p-4 bg-blue-50 rounded-lg">
          <h3 className="text-sm font-medium text-blue-900 mb-2">
            About This Data
          </h3>
          <p className="text-sm text-blue-700">
            Financial data is sourced from SEC EDGAR filings via Polygon.io.
            Data includes 10-K (annual) and 10-Q (quarterly) filings.
            All values are in USD. YoY (Year-over-Year) changes compare the
            current period to the same period in the previous year.
          </p>
        </div>
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
