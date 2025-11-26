'use client';

import { useState, useEffect } from 'react';
import { cn, safeToFixed, formatLargeNumber, formatPercent, safeParseNumber } from '@/lib/utils';

interface FinancialsTabProps {
  symbol: string;
}

interface FinancialData {
  revenue?: number;
  revenue_growth_yoy?: number;
  gross_profit?: number;
  gross_margin?: number;
  operating_income?: number;
  operating_margin?: number;
  net_income?: number;
  net_margin?: number;
  eps?: number;
  eps_diluted?: number;
  earnings_growth_yoy?: number;
  ebitda?: number;
  ebitda_margin?: number;
  total_assets?: number;
  total_liabilities?: number;
  total_equity?: number;
  cash_and_equivalents?: number;
  total_debt?: number;
  net_debt?: number;
  current_assets?: number;
  current_liabilities?: number;
  current_ratio?: number;
  quick_ratio?: number;
  debt_to_equity?: number;
  debt_to_assets?: number;
  interest_coverage?: number;
  roe?: number;
  roa?: number;
  roic?: number;
  asset_turnover?: number;
  inventory_turnover?: number;
  receivables_turnover?: number;
  operating_cash_flow?: number;
  investing_cash_flow?: number;
  financing_cash_flow?: number;
  free_cash_flow?: number;
  capex?: number;
  dividends_paid?: number;
  shares_outstanding?: number;
  book_value_per_share?: number;
  fiscal_year?: number;
  fiscal_quarter?: string;
  report_date?: string;
  pe_ratio?: number;
  pb_ratio?: number;
  ps_ratio?: number;
  ev_to_ebitda?: number;
  ev_to_revenue?: number;
  peg_ratio?: number;
  dividend_yield?: number;
}

type StatementType = 'income' | 'balance' | 'cash' | 'ratios';

export default function FinancialsTab({ symbol }: FinancialsTabProps) {
  const [data, setData] = useState<FinancialData | null>(null);
  const [statementType, setStatementType] = useState<StatementType>('income');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        const response = await fetch(`/api/v1/stocks/${symbol}/financials`);
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`);
        }
        const result = await response.json();
        setData(result.data || {});
      } catch (err) {
        console.error('Error fetching financials:', err);
        setError('Failed to load financial data');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [symbol]);

  if (loading) {
    return (
      <div className="p-6 animate-pulse">
        <div className="h-6 bg-gray-200 rounded w-48 mb-6"></div>
        <div className="space-y-4">
          {[1, 2, 3, 4, 5, 6].map((i) => (
            <div key={i} className="flex justify-between">
              <div className="h-4 bg-gray-200 rounded w-32"></div>
              <div className="h-4 bg-gray-200 rounded w-24"></div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Financial Statements</h3>
        <p className="text-gray-500">{error || 'No financial data available'}</p>
      </div>
    );
  }

  const renderIncomeStatement = () => (
    <div className="space-y-6">
      {/* Revenue Section */}
      <div>
        <h4 className="text-sm font-medium text-gray-700 mb-3 uppercase tracking-wide">Revenue</h4>
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Total Revenue</span>
            <span className="font-semibold text-gray-900">{formatLargeNumber(data.revenue)}</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Revenue Growth (YoY)</span>
            <span className={cn(
              'font-semibold',
              safeParseNumber(data.revenue_growth_yoy) >= 0 ? 'text-green-600' : 'text-red-600'
            )}>
              {formatPercent(safeParseNumber(data.revenue_growth_yoy) * 100)}
            </span>
          </div>
        </div>
      </div>

      {/* Profitability Section */}
      <div>
        <h4 className="text-sm font-medium text-gray-700 mb-3 uppercase tracking-wide">Profitability</h4>
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Gross Profit</span>
            <span className="font-semibold text-gray-900">{formatLargeNumber(data.gross_profit)}</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Gross Margin</span>
            <span className="font-semibold text-gray-900">{formatPercent(safeParseNumber(data.gross_margin) * 100)}</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Operating Income</span>
            <span className="font-semibold text-gray-900">{formatLargeNumber(data.operating_income)}</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Operating Margin</span>
            <span className="font-semibold text-gray-900">{formatPercent(safeParseNumber(data.operating_margin) * 100)}</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Net Income</span>
            <span className={cn(
              'font-semibold',
              safeParseNumber(data.net_income) >= 0 ? 'text-green-600' : 'text-red-600'
            )}>
              {formatLargeNumber(data.net_income)}
            </span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Net Margin</span>
            <span className="font-semibold text-gray-900">{formatPercent(safeParseNumber(data.net_margin) * 100)}</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-gray-600">EBITDA</span>
            <span className="font-semibold text-gray-900">{formatLargeNumber(data.ebitda)}</span>
          </div>
        </div>
      </div>

      {/* EPS Section */}
      <div>
        <h4 className="text-sm font-medium text-gray-700 mb-3 uppercase tracking-wide">Per Share</h4>
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <span className="text-gray-600">EPS (Basic)</span>
            <span className="font-semibold text-gray-900">${safeToFixed(data.eps, 2)}</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-gray-600">EPS (Diluted)</span>
            <span className="font-semibold text-gray-900">${safeToFixed(data.eps_diluted, 2)}</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Earnings Growth (YoY)</span>
            <span className={cn(
              'font-semibold',
              safeParseNumber(data.earnings_growth_yoy) >= 0 ? 'text-green-600' : 'text-red-600'
            )}>
              {formatPercent(safeParseNumber(data.earnings_growth_yoy) * 100)}
            </span>
          </div>
        </div>
      </div>
    </div>
  );

  const renderBalanceSheet = () => (
    <div className="space-y-6">
      {/* Assets */}
      <div>
        <h4 className="text-sm font-medium text-gray-700 mb-3 uppercase tracking-wide">Assets</h4>
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Total Assets</span>
            <span className="font-semibold text-gray-900">{formatLargeNumber(data.total_assets)}</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Current Assets</span>
            <span className="font-semibold text-gray-900">{formatLargeNumber(data.current_assets)}</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Cash & Equivalents</span>
            <span className="font-semibold text-gray-900">{formatLargeNumber(data.cash_and_equivalents)}</span>
          </div>
        </div>
      </div>

      {/* Liabilities */}
      <div>
        <h4 className="text-sm font-medium text-gray-700 mb-3 uppercase tracking-wide">Liabilities</h4>
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Total Liabilities</span>
            <span className="font-semibold text-gray-900">{formatLargeNumber(data.total_liabilities)}</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Current Liabilities</span>
            <span className="font-semibold text-gray-900">{formatLargeNumber(data.current_liabilities)}</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Total Debt</span>
            <span className="font-semibold text-gray-900">{formatLargeNumber(data.total_debt)}</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Net Debt</span>
            <span className="font-semibold text-gray-900">{formatLargeNumber(data.net_debt)}</span>
          </div>
        </div>
      </div>

      {/* Equity */}
      <div>
        <h4 className="text-sm font-medium text-gray-700 mb-3 uppercase tracking-wide">Equity</h4>
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Total Equity</span>
            <span className="font-semibold text-gray-900">{formatLargeNumber(data.total_equity)}</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Shares Outstanding</span>
            <span className="font-semibold text-gray-900">
              {safeToFixed(safeParseNumber(data.shares_outstanding) / 1e9, 2)}B
            </span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Book Value Per Share</span>
            <span className="font-semibold text-gray-900">${safeToFixed(data.book_value_per_share, 2)}</span>
          </div>
        </div>
      </div>

      {/* Liquidity Ratios */}
      <div>
        <h4 className="text-sm font-medium text-gray-700 mb-3 uppercase tracking-wide">Liquidity</h4>
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Current Ratio</span>
            <span className="font-semibold text-gray-900">{safeToFixed(data.current_ratio, 2)}</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Quick Ratio</span>
            <span className="font-semibold text-gray-900">{safeToFixed(data.quick_ratio, 2)}</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Debt to Equity</span>
            <span className="font-semibold text-gray-900">{safeToFixed(data.debt_to_equity, 2)}</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Interest Coverage</span>
            <span className="font-semibold text-gray-900">{safeToFixed(data.interest_coverage, 1)}x</span>
          </div>
        </div>
      </div>
    </div>
  );

  const renderCashFlow = () => (
    <div className="space-y-6">
      {/* Operating Activities */}
      <div>
        <h4 className="text-sm font-medium text-gray-700 mb-3 uppercase tracking-wide">Operating Activities</h4>
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Operating Cash Flow</span>
            <span className={cn(
              'font-semibold',
              safeParseNumber(data.operating_cash_flow) >= 0 ? 'text-green-600' : 'text-red-600'
            )}>
              {formatLargeNumber(data.operating_cash_flow)}
            </span>
          </div>
        </div>
      </div>

      {/* Investing Activities */}
      <div>
        <h4 className="text-sm font-medium text-gray-700 mb-3 uppercase tracking-wide">Investing Activities</h4>
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Investing Cash Flow</span>
            <span className={cn(
              'font-semibold',
              safeParseNumber(data.investing_cash_flow) >= 0 ? 'text-green-600' : 'text-red-600'
            )}>
              {formatLargeNumber(data.investing_cash_flow)}
            </span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Capital Expenditures</span>
            <span className="font-semibold text-red-600">{formatLargeNumber(data.capex)}</span>
          </div>
        </div>
      </div>

      {/* Financing Activities */}
      <div>
        <h4 className="text-sm font-medium text-gray-700 mb-3 uppercase tracking-wide">Financing Activities</h4>
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Financing Cash Flow</span>
            <span className={cn(
              'font-semibold',
              safeParseNumber(data.financing_cash_flow) >= 0 ? 'text-green-600' : 'text-red-600'
            )}>
              {formatLargeNumber(data.financing_cash_flow)}
            </span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Dividends Paid</span>
            <span className="font-semibold text-red-600">{formatLargeNumber(data.dividends_paid)}</span>
          </div>
        </div>
      </div>

      {/* Free Cash Flow */}
      <div>
        <h4 className="text-sm font-medium text-gray-700 mb-3 uppercase tracking-wide">Free Cash Flow</h4>
        <div className="bg-blue-50 rounded-lg p-4">
          <div className="flex justify-between items-center">
            <span className="text-blue-800 font-medium">Free Cash Flow</span>
            <span className={cn(
              'text-xl font-bold',
              safeParseNumber(data.free_cash_flow) >= 0 ? 'text-green-600' : 'text-red-600'
            )}>
              {formatLargeNumber(data.free_cash_flow)}
            </span>
          </div>
          <p className="text-xs text-blue-600 mt-1">Operating Cash Flow - CapEx</p>
        </div>
      </div>
    </div>
  );

  const renderRatios = () => (
    <div className="space-y-6">
      {/* Valuation Ratios */}
      <div>
        <h4 className="text-sm font-medium text-gray-700 mb-3 uppercase tracking-wide">Valuation</h4>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <div className="bg-gray-50 rounded-lg p-4">
            <div className="text-sm text-gray-600 mb-1">P/E Ratio</div>
            <div className="text-xl font-semibold text-gray-900">{safeToFixed(data.pe_ratio, 1)}</div>
          </div>
          <div className="bg-gray-50 rounded-lg p-4">
            <div className="text-sm text-gray-600 mb-1">P/B Ratio</div>
            <div className="text-xl font-semibold text-gray-900">{safeToFixed(data.pb_ratio, 1)}</div>
          </div>
          <div className="bg-gray-50 rounded-lg p-4">
            <div className="text-sm text-gray-600 mb-1">P/S Ratio</div>
            <div className="text-xl font-semibold text-gray-900">{safeToFixed(data.ps_ratio, 1)}</div>
          </div>
          <div className="bg-gray-50 rounded-lg p-4">
            <div className="text-sm text-gray-600 mb-1">EV/EBITDA</div>
            <div className="text-xl font-semibold text-gray-900">{safeToFixed(data.ev_to_ebitda, 1)}</div>
          </div>
          <div className="bg-gray-50 rounded-lg p-4">
            <div className="text-sm text-gray-600 mb-1">EV/Revenue</div>
            <div className="text-xl font-semibold text-gray-900">{safeToFixed(data.ev_to_revenue, 1)}</div>
          </div>
          <div className="bg-gray-50 rounded-lg p-4">
            <div className="text-sm text-gray-600 mb-1">PEG Ratio</div>
            <div className="text-xl font-semibold text-gray-900">{safeToFixed(data.peg_ratio, 2)}</div>
          </div>
        </div>
      </div>

      {/* Profitability Ratios */}
      <div>
        <h4 className="text-sm font-medium text-gray-700 mb-3 uppercase tracking-wide">Profitability</h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="bg-gray-50 rounded-lg p-4">
            <div className="text-sm text-gray-600 mb-1">ROE</div>
            <div className="text-xl font-semibold text-gray-900">{formatPercent(safeParseNumber(data.roe) * 100)}</div>
          </div>
          <div className="bg-gray-50 rounded-lg p-4">
            <div className="text-sm text-gray-600 mb-1">ROA</div>
            <div className="text-xl font-semibold text-gray-900">{formatPercent(safeParseNumber(data.roa) * 100)}</div>
          </div>
          <div className="bg-gray-50 rounded-lg p-4">
            <div className="text-sm text-gray-600 mb-1">ROIC</div>
            <div className="text-xl font-semibold text-gray-900">{formatPercent(safeParseNumber(data.roic) * 100)}</div>
          </div>
          <div className="bg-gray-50 rounded-lg p-4">
            <div className="text-sm text-gray-600 mb-1">Dividend Yield</div>
            <div className="text-xl font-semibold text-gray-900">{formatPercent(safeParseNumber(data.dividend_yield) * 100)}</div>
          </div>
        </div>
      </div>

      {/* Efficiency Ratios */}
      <div>
        <h4 className="text-sm font-medium text-gray-700 mb-3 uppercase tracking-wide">Efficiency</h4>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <div className="bg-gray-50 rounded-lg p-4">
            <div className="text-sm text-gray-600 mb-1">Asset Turnover</div>
            <div className="text-xl font-semibold text-gray-900">{safeToFixed(data.asset_turnover, 2)}x</div>
          </div>
          <div className="bg-gray-50 rounded-lg p-4">
            <div className="text-sm text-gray-600 mb-1">Inventory Turnover</div>
            <div className="text-xl font-semibold text-gray-900">{safeToFixed(data.inventory_turnover, 1)}x</div>
          </div>
          <div className="bg-gray-50 rounded-lg p-4">
            <div className="text-sm text-gray-600 mb-1">Receivables Turnover</div>
            <div className="text-xl font-semibold text-gray-900">{safeToFixed(data.receivables_turnover, 1)}x</div>
          </div>
        </div>
      </div>
    </div>
  );

  const statementTabs: { id: StatementType; label: string }[] = [
    { id: 'income', label: 'Income Statement' },
    { id: 'balance', label: 'Balance Sheet' },
    { id: 'cash', label: 'Cash Flow' },
    { id: 'ratios', label: 'Ratios' },
  ];

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-lg font-semibold text-gray-900">Financial Statements</h3>
        {data.fiscal_year && data.fiscal_quarter && (
          <span className="text-sm text-gray-500">
            {data.fiscal_quarter} {data.fiscal_year}
          </span>
        )}
      </div>

      {/* Statement Type Tabs */}
      <div className="flex gap-2 mb-6 overflow-x-auto pb-2">
        {statementTabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setStatementType(tab.id)}
            className={cn(
              'px-4 py-2 text-sm font-medium rounded-lg whitespace-nowrap transition-colors',
              statementType === tab.id
                ? 'bg-blue-600 text-white'
                : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
            )}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Content */}
      {statementType === 'income' && renderIncomeStatement()}
      {statementType === 'balance' && renderBalanceSheet()}
      {statementType === 'cash' && renderCashFlow()}
      {statementType === 'ratios' && renderRatios()}
    </div>
  );
}
