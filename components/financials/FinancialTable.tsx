'use client';

import { useState, useEffect, useMemo } from 'react';
import { cn } from '@/lib/utils';
import {
  FinancialPeriod,
  FinancialRowConfig,
  StatementType,
  Timeframe,
  getRowConfigForStatementType,
  formatPeriodLabel,
} from '@/types/financials';
import { getFinancialStatements } from '@/lib/api/financials';
import {
  formatFinancialValue,
  formatYoYChange,
  exportToCSV,
  downloadFile,
} from '@/lib/formatters/financial';

interface FinancialTableProps {
  ticker: string;
  statementType: StatementType;
  timeframe: Timeframe;
  showYoY?: boolean;
  limit?: number;
}

export default function FinancialTable({
  ticker,
  statementType,
  timeframe,
  showYoY = true,
  limit = 8,
}: FinancialTableProps) {
  const [periods, setPeriods] = useState<FinancialPeriod[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const rowConfig = useMemo(
    () => getRowConfigForStatementType(statementType),
    [statementType]
  );

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        setError(null);

        const response = await getFinancialStatements(ticker, statementType, {
          timeframe,
          limit,
        });

        if (!response) {
          setError('No financial data available');
          setPeriods([]);
          return;
        }

        setPeriods(response.periods || []);
      } catch (err) {
        console.error('Error fetching financial data:', err);
        setError('Failed to load financial data');
        setPeriods([]);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [ticker, statementType, timeframe, limit]);

  const handleExport = () => {
    const csv = exportToCSV(periods, rowConfig, ticker, statementType);
    downloadFile(csv, `${ticker}-${statementType}-${timeframe}.csv`, 'text/csv');
  };

  if (loading) {
    return <FinancialTableSkeleton />;
  }

  if (error || periods.length === 0) {
    return (
      <div className="bg-gray-50 rounded-lg p-8 text-center">
        <p className="text-ic-text-dim">{error || 'No financial data available'}</p>
      </div>
    );
  }

  // Filter rows that have at least some data
  const rowsWithData = rowConfig.filter((row) =>
    periods.some((period) => period.data[row.key] !== undefined && period.data[row.key] !== null)
  );

  return (
    <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
      {/* Export Button */}
      <div className="flex justify-end p-2 border-b border-gray-100">
        <button
          onClick={handleExport}
          className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-ic-text-muted hover:text-ic-text-primary hover:bg-ic-surface-hover rounded-md transition-colors"
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
              d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
            />
          </svg>
          Export CSV
        </button>
      </div>

      {/* Table */}
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="bg-gray-50 border-b border-gray-200">
              <th className="sticky left-0 bg-gray-50 px-4 py-3 text-left text-sm font-semibold text-gray-900 min-w-[200px]">
                Metric
              </th>
              {periods.map((period, idx) => (
                <th
                  key={idx}
                  className="px-4 py-3 text-right text-sm font-semibold text-gray-900 min-w-[120px]"
                >
                  {formatPeriodLabel(period, timeframe)}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {rowsWithData.map((row) => (
              <FinancialTableRow
                key={row.key}
                row={row}
                periods={periods}
                showYoY={showYoY}
              />
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

interface FinancialTableRowProps {
  row: FinancialRowConfig;
  periods: FinancialPeriod[];
  showYoY: boolean;
}

function FinancialTableRow({ row, periods, showYoY }: FinancialTableRowProps) {
  return (
    <tr className={cn('hover:bg-ic-surface-hover transition-colors', row.bold && 'bg-gray-25')}>
      {/* Metric Name */}
      <td
        className={cn(
          'sticky left-0 bg-white px-4 py-3 text-sm text-gray-900',
          row.bold && 'font-semibold',
          row.indent && `pl-${4 + row.indent * 4}`
        )}
        style={{ paddingLeft: row.indent ? `${16 + row.indent * 16}px` : undefined }}
      >
        <div className="flex items-center gap-2">
          <span>{row.label}</span>
          {row.calculated && (
            <span
              className="text-xs text-blue-500 cursor-help"
              title={row.tooltip || 'Calculated field'}
            >
              calc
            </span>
          )}
        </div>
      </td>

      {/* Period Values */}
      {periods.map((period, idx) => {
        const value = period.data[row.key];
        const yoyChange = showYoY ? period.yoy_change?.[row.key] : undefined;
        const formattedYoY = formatYoYChange(yoyChange);

        return (
          <td key={idx} className="px-4 py-3 text-right">
            <div className={cn('text-sm', row.bold && 'font-semibold')}>
              {formatFinancialValue(value as number, row.format, row.decimals)}
            </div>
            {showYoY && yoyChange !== undefined && (
              <div className={cn('text-xs mt-0.5', formattedYoY.color)}>
                {formattedYoY.text}
              </div>
            )}
          </td>
        );
      })}
    </tr>
  );
}

function FinancialTableSkeleton() {
  return (
    <div className="bg-white rounded-lg border border-gray-200 overflow-hidden animate-pulse">
      <div className="p-2 border-b border-gray-100">
        <div className="h-8 w-24 bg-gray-200 rounded ml-auto" />
      </div>
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="bg-gray-50 border-b border-gray-200">
              <th className="px-4 py-3 text-left">
                <div className="h-4 w-24 bg-gray-200 rounded" />
              </th>
              {[1, 2, 3, 4, 5, 6, 7, 8].map((i) => (
                <th key={i} className="px-4 py-3 text-right">
                  <div className="h-4 w-16 bg-gray-200 rounded ml-auto" />
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {[1, 2, 3, 4, 5, 6, 7, 8, 9, 10].map((row) => (
              <tr key={row}>
                <td className="px-4 py-3">
                  <div className="h-4 w-32 bg-gray-200 rounded" />
                </td>
                {[1, 2, 3, 4, 5, 6, 7, 8].map((col) => (
                  <td key={col} className="px-4 py-3 text-right">
                    <div className="h-4 w-16 bg-gray-200 rounded ml-auto" />
                    <div className="h-3 w-12 bg-gray-100 rounded ml-auto mt-1" />
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
