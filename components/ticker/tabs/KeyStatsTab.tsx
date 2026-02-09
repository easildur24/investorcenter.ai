'use client';

import { useState, useEffect } from 'react';

interface KeyStatsTabProps {
  symbol: string;
}

// Format large numbers with abbreviations
function formatNumber(value: number | null | undefined): string {
  if (value === null || value === undefined) return 'N/A';
  const abs = Math.abs(value);
  if (abs >= 1e12) return `${(value / 1e12).toFixed(2)}T`;
  if (abs >= 1e9) return `${(value / 1e9).toFixed(2)}B`;
  if (abs >= 1e6) return `${(value / 1e6).toFixed(2)}M`;
  if (abs >= 1e3) return `${(value / 1e3).toFixed(1)}K`;
  return value.toLocaleString(undefined, { maximumFractionDigits: 2 });
}

// Format as percentage
function formatPct(value: number | null | undefined): string {
  if (value === null || value === undefined) return 'N/A';
  return `${(value * 100).toFixed(2)}%`;
}

// Format as decimal
function formatDec(value: number | null | undefined, digits: number = 2): string {
  if (value === null || value === undefined) return 'N/A';
  return value.toFixed(digits);
}

// Format currency
function formatCurrency(value: number | null | undefined): string {
  if (value === null || value === undefined) return 'N/A';
  return `$${formatNumber(value)}`;
}

// Format price
function formatPrice(value: number | null | undefined): string {
  if (value === null || value === undefined) return 'N/A';
  return `$${value.toFixed(2)}`;
}

// Format ratio (like PE, EV/EBITDA)
function formatRatio(value: number | null | undefined): string {
  if (value === null || value === undefined) return 'N/A';
  return `${value.toFixed(2)}x`;
}

// Color for positive/negative values
function valueColor(value: number | null | undefined): string {
  if (value === null || value === undefined) return 'text-ic-text-primary';
  if (value > 0) return 'text-ic-positive';
  if (value < 0) return 'text-ic-negative';
  return 'text-ic-text-primary';
}

// A single metric row
function MetricRow({ label, value, className = '' }: { label: string; value: string; className?: string }) {
  return (
    <div className="flex justify-between items-center py-2 border-b border-ic-border/50 last:border-b-0">
      <span className="text-sm text-ic-text-muted">{label}</span>
      <span className={`text-sm font-medium ${className || 'text-ic-text-primary'}`}>{value}</span>
    </div>
  );
}

// Section header
function SectionHeader({ title }: { title: string }) {
  return (
    <h4 className="text-sm font-semibold text-ic-text-secondary uppercase tracking-wide mb-3 mt-1">{title}</h4>
  );
}

// Group card
function GroupCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="bg-ic-bg-secondary rounded-lg p-4">
      <h3 className="text-base font-semibold text-ic-text-primary mb-4 pb-2 border-b border-ic-border">{title}</h3>
      {children}
    </div>
  );
}

export default function KeyStatsTab({ symbol }: KeyStatsTabProps) {
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [updatedAt, setUpdatedAt] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        setError(null);
        const response = await fetch(`/api/v1/tickers/${symbol}/keystats`);
        if (response.status === 404) {
          setError('No key stats data available for this ticker');
          return;
        }
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`);
        }
        const result = await response.json();
        setData(result.data);
        setUpdatedAt(result.meta?.updated_at || null);
      } catch (err) {
        console.error('Error fetching key stats:', err);
        setError('Failed to load key stats');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [symbol]);

  if (loading) {
    return (
      <div className="p-6 animate-pulse">
        <div className="h-6 bg-ic-bg-tertiary rounded w-48 mb-6"></div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {[1, 2, 3, 4, 5, 6].map((i) => (
            <div key={i} className="bg-ic-bg-secondary rounded-lg p-4">
              <div className="h-4 bg-ic-bg-tertiary rounded w-32 mb-4"></div>
              <div className="space-y-3">
                {[1, 2, 3, 4].map((j) => (
                  <div key={j} className="flex justify-between">
                    <div className="h-3 bg-ic-bg-tertiary rounded w-24"></div>
                    <div className="h-3 bg-ic-bg-tertiary rounded w-16"></div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <div className="text-center py-12">
          <p className="text-ic-text-muted text-lg">{error}</p>
          <p className="text-ic-text-dim text-sm mt-2">Key stats can be ingested via the API for this ticker.</p>
        </div>
      </div>
    );
  }

  if (!data) return null;

  const fin = data.financials || {};
  const income = fin.income_statement || {};
  const balance = fin.balance_sheet || {};
  const cashFlow = fin.cash_flow || {};
  const commonSize = fin.common_size_statements || {};
  const earningsQuality = fin.earnings_quality || {};
  const profitability = fin.profitability || {};

  const perf = data.performance_risk_estimates || {};
  const valuation = perf.current_valuation || {};
  const stockPerf = perf.stock_price_performance || {};
  const risk = perf.risk_metrics || {};
  const estimates = perf.estimates || {};
  const mgmt = perf.management_effectiveness || {};
  const dividends = perf.dividends_and_shares || {};

  const other = data.other_metrics || {};
  const liquidity = other.liquidity_and_solvency || {};
  const advanced = other.advanced_metrics || {};
  const employees = other.employee_count_metrics || {};

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-lg font-semibold text-ic-text-primary">Key Stats</h3>
        {updatedAt && (
          <span className="text-xs text-ic-text-dim">
            Updated {new Date(updatedAt).toLocaleDateString()}
          </span>
        )}
      </div>

      {/* Grid layout */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">

        {/* Current Valuation */}
        <GroupCard title="Valuation">
          <MetricRow label="Price" value={formatPrice(valuation.price)} />
          <MetricRow label="Market Cap" value={formatCurrency(valuation.market_cap)} />
          <MetricRow label="Enterprise Value" value={formatCurrency(valuation.enterprise_value)} />
          <MetricRow label="P/E Ratio" value={formatRatio(valuation.pe_ratio)} />
          <MetricRow label="Forward P/E" value={formatRatio(valuation.pe_ratio_forward)} />
          <MetricRow label="P/E Forward 1Y" value={formatRatio(valuation.pe_ratio_forward_1y)} />
          <MetricRow label="PEG Ratio" value={formatDec(valuation.peg_ratio)} />
          <MetricRow label="P/S Ratio" value={formatRatio(valuation.ps_ratio)} />
          <MetricRow label="P/S Forward" value={formatRatio(valuation.ps_ratio_forward)} />
          <MetricRow label="P/S Forward 1Y" value={formatRatio(valuation.ps_ratio_forward_1y)} />
          <MetricRow label="P/B Ratio" value={formatRatio(valuation.price_to_book_value)} />
          <MetricRow label="P/FCF" value={formatRatio(valuation.price_to_free_cash_flow)} />
          <MetricRow label="EV/EBIT" value={formatRatio(valuation.ev_to_ebit)} />
          <MetricRow label="EV/EBITDA" value={formatRatio(valuation.ev_to_ebitda)} />
          <MetricRow label="EV/EBITDA Forward" value={formatRatio(valuation.ev_to_ebitda_forward)} />
          <MetricRow label="EBIT Margin TTM" value={formatPct(valuation.ebit_margin_ttm)} />
        </GroupCard>

        {/* Income Statement */}
        <GroupCard title="Income Statement">
          <SectionHeader title="TTM" />
          <MetricRow label="Revenue TTM" value={formatCurrency(income.revenue_ttm)} />
          <MetricRow label="Net Income TTM" value={formatCurrency(income.net_income_ttm)} />
          <MetricRow label="EBIT TTM" value={formatCurrency(income.ebit_ttm)} />
          <MetricRow label="EBITDA TTM" value={formatCurrency(income.ebitda_ttm)} />
          <SectionHeader title="Quarterly" />
          <MetricRow label="Revenue" value={formatCurrency(income.revenue_quarterly)} />
          <MetricRow label="Net Income" value={formatCurrency(income.net_income_quarterly)} />
          <MetricRow label="EBIT" value={formatCurrency(income.ebit_quarterly)} />
          <MetricRow label="EBITDA" value={formatCurrency(income.ebitda_quarterly)} />
          <SectionHeader title="Growth" />
          <MetricRow label="Revenue QoQ" value={formatPct(income.revenue_quarterly_yoy_growth)} className={valueColor(income.revenue_quarterly_yoy_growth)} />
          <MetricRow label="EBITDA QoQ" value={formatPct(income.ebitda_quarterly_yoy_growth)} className={valueColor(income.ebitda_quarterly_yoy_growth)} />
          <MetricRow label="EPS Diluted QoQ" value={formatPct(income.eps_diluted_quarterly_yoy_growth)} className={valueColor(income.eps_diluted_quarterly_yoy_growth)} />
        </GroupCard>

        {/* Balance Sheet */}
        <GroupCard title="Balance Sheet">
          <MetricRow label="Total Assets" value={formatCurrency(balance.total_assets_quarterly)} />
          <MetricRow label="Total Liabilities" value={formatCurrency(balance.total_liabilities_quarterly)} />
          <MetricRow label="Shareholders' Equity" value={formatCurrency(balance.shareholders_equity_quarterly)} />
          <MetricRow label="Book Value" value={formatCurrency(balance.book_value_quarterly)} />
          <MetricRow label="Cash & Short-Term Inv." value={formatCurrency(balance.cash_and_short_term_investments_quarterly)} />
          <MetricRow label="Long-Term Assets" value={formatCurrency(balance.total_long_term_assets_quarterly)} />
          <MetricRow label="Long-Term Debt" value={formatCurrency(balance.total_long_term_debt_quarterly)} />
        </GroupCard>

        {/* Cash Flow */}
        <GroupCard title="Cash Flow">
          <MetricRow label="Operating Cash Flow TTM" value={formatCurrency(cashFlow.cash_from_operations_ttm)} />
          <MetricRow label="Investing Cash Flow TTM" value={formatCurrency(cashFlow.cash_from_investing_ttm)} />
          <MetricRow label="Financing Cash Flow TTM" value={formatCurrency(cashFlow.cash_from_financing_ttm)} />
          <MetricRow label="CapEx TTM" value={formatCurrency(cashFlow.capital_expenditures_ttm)} />
          <MetricRow label="Free Cash Flow" value={formatCurrency(cashFlow.free_cash_flow)} className={valueColor(cashFlow.free_cash_flow)} />
          <MetricRow label="Changes in Working Capital" value={formatCurrency(cashFlow.changes_in_working_capital_ttm)} />
        </GroupCard>

        {/* Profitability & Returns */}
        <GroupCard title="Profitability & Returns">
          <SectionHeader title="Margins" />
          <MetricRow label="Gross Profit Margin" value={formatPct(profitability.gross_profit_margin)} />
          <MetricRow label="Operating Margin TTM" value={formatPct(profitability.operating_margin_ttm)} />
          <SectionHeader title="Returns" />
          <MetricRow label="Return on Equity" value={formatPct(earningsQuality.return_on_equity)} />
          <MetricRow label="Return on Assets" value={formatPct(earningsQuality.return_on_assets)} />
          <MetricRow label="Return on Invested Capital" value={formatPct(earningsQuality.return_on_invested_capital)} />
          <SectionHeader title="Per Share" />
          <MetricRow label="EPS Basic TTM" value={formatPrice(commonSize.eps_basic_ttm)} />
          <MetricRow label="EPS Diluted TTM" value={formatPrice(commonSize.eps_diluted_ttm)} />
          <MetricRow label="Shares Outstanding" value={formatNumber(commonSize.shares_outstanding)} />
        </GroupCard>

        {/* Estimates */}
        <GroupCard title="Analyst Estimates">
          <SectionHeader title="EPS Estimates" />
          <MetricRow label="Current Quarter" value={formatPrice(estimates.eps_estimate_current_quarter)} />
          <MetricRow label="Next Quarter" value={formatPrice(estimates.eps_estimate_next_quarter)} />
          <MetricRow label="Current Fiscal Year" value={formatPrice(estimates.eps_estimate_current_fiscal_year)} />
          <MetricRow label="Next Fiscal Year" value={formatPrice(estimates.eps_estimate_next_fiscal_year)} />
          <SectionHeader title="Revenue Estimates" />
          <MetricRow label="Current Quarter" value={formatCurrency(estimates.revenue_estimate_current_quarter)} />
          <MetricRow label="Next Quarter" value={formatCurrency(estimates.revenue_estimate_next_quarter)} />
          <MetricRow label="Current Fiscal Year" value={formatCurrency(estimates.revenue_estimate_current_fiscal_year)} />
          <MetricRow label="Next Fiscal Year" value={formatCurrency(estimates.revenue_estimate_next_fiscal_year)} />
        </GroupCard>

        {/* Stock Price Performance */}
        <GroupCard title="Stock Price Performance">
          <MetricRow label="1 Month" value={formatPct(stockPerf['1_month_total_returns'])} className={valueColor(stockPerf['1_month_total_returns'])} />
          <MetricRow label="3 Months" value={formatPct(stockPerf['3_month_total_returns'])} className={valueColor(stockPerf['3_month_total_returns'])} />
          <MetricRow label="6 Months" value={formatPct(stockPerf['6_month_total_returns'])} className={valueColor(stockPerf['6_month_total_returns'])} />
          <MetricRow label="YTD" value={formatPct(stockPerf.ytd_total_returns)} className={valueColor(stockPerf.ytd_total_returns)} />
          <MetricRow label="1 Year" value={formatPct(stockPerf['1_year_total_returns'])} className={valueColor(stockPerf['1_year_total_returns'])} />
          <MetricRow label="3 Year (Ann.)" value={formatPct(stockPerf['3_year_annualized_returns'])} className={valueColor(stockPerf['3_year_annualized_returns'])} />
          <MetricRow label="5 Year (Ann.)" value={formatPct(stockPerf['5_year_annualized_returns'])} className={valueColor(stockPerf['5_year_annualized_returns'])} />
          <MetricRow label="52-Week High" value={formatPrice(stockPerf['52_week_high'])} />
          <MetricRow label="52-Week Low" value={formatPrice(stockPerf['52_week_low'])} />
        </GroupCard>

        {/* Risk Metrics */}
        <GroupCard title="Risk Metrics">
          <MetricRow label="Beta (5Y)" value={formatDec(risk.beta_5y)} />
          <MetricRow label="Alpha (5Y)" value={formatDec(risk.alpha_5y)} className={valueColor(risk.alpha_5y)} />
          <MetricRow label="Annualized Std Dev (5Y)" value={formatPct(risk.annualized_std_dev_5y)} />
          <MetricRow label="Sharpe Ratio (5Y)" value={formatDec(risk.historical_sharpe_ratio_5y, 4)} />
          <MetricRow label="Sortino Ratio (5Y)" value={formatDec(risk.historical_sortino_5y, 4)} />
          <MetricRow label="Max Drawdown (5Y)" value={formatPct(risk.max_drawdown_5y)} className="text-ic-negative" />
          <MetricRow label="Monthly VaR 5% (5Y)" value={formatPct(risk.monthly_var_5pct_5y)} />
        </GroupCard>

        {/* Liquidity & Solvency */}
        <GroupCard title="Liquidity & Solvency">
          <MetricRow label="Current Ratio" value={formatDec(liquidity.current_ratio)} />
          <MetricRow label="Quick Ratio" value={formatDec(liquidity.quick_ratio_quarterly)} />
          <MetricRow label="Debt to Equity" value={formatDec(liquidity.debt_to_equity_ratio)} />
          <MetricRow label="Altman Z-Score" value={formatDec(liquidity.altman_z_score_ttm)} />
          <MetricRow label="Free Cash Flow (Q)" value={formatCurrency(liquidity.free_cash_flow_quarterly)} className={valueColor(liquidity.free_cash_flow_quarterly)} />
        </GroupCard>

        {/* Management Effectiveness */}
        <GroupCard title="Management Effectiveness">
          <MetricRow label="Asset Utilization TTM" value={formatDec(mgmt.asset_utilization_ttm)} />
          <MetricRow label="Days Sales Outstanding" value={formatDec(mgmt.days_sales_outstanding_quarterly)} />
          <MetricRow label="Days Inventory Outstanding" value={formatDec(mgmt.days_inventory_outstanding_quarterly)} />
          <MetricRow label="Days Payable Outstanding" value={formatDec(mgmt.days_payable_outstanding_quarterly)} />
          <MetricRow label="Total Receivables (Q)" value={formatCurrency(mgmt.total_receivables_quarterly)} />
        </GroupCard>

        {/* Advanced Metrics */}
        <GroupCard title="Advanced Metrics">
          <MetricRow label="Piotroski F-Score" value={advanced.piotroski_f_score_ttm != null ? `${advanced.piotroski_f_score_ttm}/9` : 'N/A'} />
          <MetricRow label="Tobin's Q" value={formatDec(advanced.tobins_q_quarterly)} />
          <MetricRow label="Sustainable Growth Rate" value={formatPct(advanced.sustainable_growth_rate_ttm)} />
          <MetricRow label="Quality Ratio Score" value={advanced.quality_ratio_score != null ? `${advanced.quality_ratio_score}/10` : 'N/A'} />
          <MetricRow label="Momentum Score" value={advanced.momentum_score != null ? `${advanced.momentum_score}/10` : 'N/A'} />
          <MetricRow label="Market Cap Score" value={advanced.market_cap_score != null ? `${advanced.market_cap_score}/10` : 'N/A'} />
        </GroupCard>

        {/* Dividends & Shares */}
        <GroupCard title="Dividends & Shares">
          <MetricRow label="Dividend Yield" value={formatPct(dividends.dividend_yield)} />
          <MetricRow label="Forward Dividend Yield" value={dividends.dividend_yield_forward != null ? formatPct(dividends.dividend_yield_forward) : 'N/A'} />
          <MetricRow label="Payout Ratio TTM" value={formatPct(dividends.payout_ratio_ttm)} />
          <MetricRow label="Last Dividend" value={dividends.last_dividend_amount != null ? formatPrice(dividends.last_dividend_amount) : 'N/A'} />
          <MetricRow label="Last Ex-Dividend Date" value={dividends.last_ex_dividend_date || 'N/A'} />
        </GroupCard>

        {/* Employee Metrics */}
        {employees.total_employees_annual && (
          <GroupCard title="Employee Metrics">
            <MetricRow label="Total Employees" value={formatNumber(employees.total_employees_annual)} />
            <MetricRow label="Revenue per Employee" value={formatCurrency(employees.revenue_per_employee_annual)} />
            <MetricRow label="Net Income per Employee" value={formatCurrency(employees.net_income_per_employee_annual)} />
          </GroupCard>
        )}

      </div>
    </div>
  );
}
