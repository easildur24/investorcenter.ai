'use client';

import { useState, useEffect } from 'react';
import { cn } from '@/lib/utils';
import {
  ComprehensiveMetricsResponse,
  ValuationMetrics,
  ProfitabilityMetrics,
  LiquidityMetrics,
  LeverageMetrics,
  EfficiencyMetrics,
  GrowthMetrics,
  PerShareMetrics,
  DividendMetrics,
  QualityScores,
  ForwardEstimates,
  getZScoreColor,
  getFScoreColor,
  getPEGColor,
  getPayoutColor,
  formatMetricValue,
  valuationMetricConfigs,
  profitabilityMetricConfigs,
  liquidityMetricConfigs,
  leverageMetricConfigs,
  efficiencyMetricConfigs,
  growthMetricConfigs,
  perShareMetricConfigs,
  dividendMetricConfigs,
  qualityScoreConfigs,
  forwardEstimateConfigs,
  MetricDisplayConfig,
} from '@/types/metrics';
import { getComprehensiveMetrics } from '@/lib/api/metrics';

interface MetricsTabProps {
  symbol: string;
}

type MetricCategory =
  | 'valuation'
  | 'profitability'
  | 'financial_health'
  | 'efficiency'
  | 'growth'
  | 'dividends'
  | 'quality';

const categoryTabs: { id: MetricCategory; label: string }[] = [
  { id: 'valuation', label: 'Valuation' },
  { id: 'profitability', label: 'Profitability' },
  { id: 'financial_health', label: 'Financial Health' },
  { id: 'efficiency', label: 'Efficiency' },
  { id: 'growth', label: 'Growth' },
  { id: 'dividends', label: 'Dividends' },
  { id: 'quality', label: 'Quality Scores' },
];

export default function MetricsTab({ symbol }: MetricsTabProps) {
  const [data, setData] = useState<ComprehensiveMetricsResponse | null>(null);
  const [activeCategory, setActiveCategory] = useState<MetricCategory>('valuation');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        setError(null);
        const response = await getComprehensiveMetrics(symbol);
        if (response) {
          setData(response);
        } else {
          setError('No metrics data available');
        }
      } catch (err) {
        console.error('Error fetching metrics:', err);
        setError('Failed to load financial metrics');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [symbol]);

  if (loading) {
    return <MetricsLoadingSkeleton />;
  }

  if (error || !data) {
    return (
      <div className="p-6">
        <h3 className="text-lg font-semibold text-ic-text-primary mb-4">Financial Metrics</h3>
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
          <p className="text-yellow-800">{error || 'No metrics data available'}</p>
          <p className="text-sm text-yellow-600 mt-2">
            Financial metrics are sourced from Financial Modeling Prep (FMP) API.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="mb-6">
        <h3 className="text-lg font-semibold text-ic-text-primary">Financial Metrics</h3>
        <p className="text-sm text-ic-text-muted mt-1">
          Comprehensive TTM financial ratios and metrics
        </p>
      </div>

      {/* Category Tabs */}
      <div className="flex flex-wrap gap-2 mb-6 border-b border-ic-border pb-4">
        {categoryTabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveCategory(tab.id)}
            className={cn(
              'px-3 py-1.5 text-sm rounded-md transition-colors',
              activeCategory === tab.id
                ? 'bg-ic-blue text-white'
                : 'bg-ic-bg-secondary text-ic-text-muted hover:bg-ic-surface-hover'
            )}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Category Content */}
      <div className="min-h-[400px]">
        {activeCategory === 'valuation' && (
          <ValuationSection
            valuation={data.data.valuation}
            forwardEstimates={data.data.forward_estimates}
          />
        )}
        {activeCategory === 'profitability' && (
          <ProfitabilitySection profitability={data.data.profitability} />
        )}
        {activeCategory === 'financial_health' && (
          <FinancialHealthSection
            liquidity={data.data.liquidity}
            leverage={data.data.leverage}
          />
        )}
        {activeCategory === 'efficiency' && (
          <EfficiencySection efficiency={data.data.efficiency} />
        )}
        {activeCategory === 'growth' && (
          <GrowthSection growth={data.data.growth} />
        )}
        {activeCategory === 'dividends' && (
          <DividendsSection
            dividends={data.data.dividends}
            perShare={data.data.per_share}
          />
        )}
        {activeCategory === 'quality' && (
          <QualitySection
            qualityScores={data.data.quality_scores}
            valuation={data.data.valuation}
          />
        )}
      </div>

      {/* Data Source Footer */}
      <div className="mt-8 p-4 bg-blue-50 rounded-lg">
        <h4 className="text-sm font-medium text-blue-800 mb-1">About This Data</h4>
        <p className="text-sm text-blue-700">
          Financial metrics are sourced from Financial Modeling Prep (FMP) API.
          All values are trailing twelve months (TTM) unless otherwise specified.
          Data is updated in real-time.
        </p>
      </div>
    </div>
  );
}

// ============================================================================
// Section Components
// ============================================================================

function ValuationSection({
  valuation,
  forwardEstimates
}: {
  valuation: ValuationMetrics;
  forwardEstimates: ForwardEstimates;
}) {
  return (
    <div className="space-y-6">
      {/* Key Valuation Multiples */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Price Multiples
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard
            label="P/E Ratio"
            value={valuation.pe_ratio}
            format="ratio"
            tooltip="Price to Earnings - lower may indicate undervaluation"
          />
          <MetricCard
            label="Forward P/E"
            value={valuation.forward_pe}
            format="ratio"
            tooltip="P/E based on estimated future earnings"
          />
          <MetricCard
            label="PEG Ratio"
            value={valuation.peg_ratio}
            format="ratio"
            tooltip="P/E to Growth - <1 suggests undervalued relative to growth"
            interpretation={valuation.peg_interpretation}
            interpretationColorFn={getPEGColor}
          />
          <MetricCard
            label="P/B Ratio"
            value={valuation.pb_ratio}
            format="ratio"
            tooltip="Price to Book value"
          />
        </div>
      </div>

      {/* Revenue & Cash Flow Multiples */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Revenue & Cash Flow Multiples
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard
            label="P/S Ratio"
            value={valuation.ps_ratio}
            format="ratio"
            tooltip="Price to Sales"
          />
          <MetricCard
            label="P/FCF"
            value={valuation.price_to_fcf}
            format="ratio"
            tooltip="Price to Free Cash Flow"
          />
          <MetricCard
            label="EV/EBITDA"
            value={valuation.ev_to_ebitda}
            format="ratio"
            tooltip="Enterprise Value to EBITDA"
          />
          <MetricCard
            label="EV/Sales"
            value={valuation.ev_to_sales}
            format="ratio"
            tooltip="Enterprise Value to Sales"
          />
        </div>
      </div>

      {/* Yields */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Yields
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard
            label="Earnings Yield"
            value={valuation.earnings_yield}
            format="percent"
            tooltip="Inverse of P/E - higher is better"
          />
          <MetricCard
            label="FCF Yield"
            value={valuation.fcf_yield}
            format="percent"
            tooltip="Free Cash Flow Yield - higher is better"
          />
          <MetricCard
            label="Market Cap"
            value={valuation.market_cap}
            format="currency"
            tooltip="Total market capitalization"
          />
          <MetricCard
            label="Enterprise Value"
            value={valuation.enterprise_value}
            format="currency"
            tooltip="Market Cap + Debt - Cash"
          />
        </div>
      </div>

      {/* Forward Estimates */}
      {(forwardEstimates.forward_eps || forwardEstimates.forward_revenue) && (
        <div>
          <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
            Analyst Estimates
          </h4>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <MetricCard
              label="Forward EPS"
              value={forwardEstimates.forward_eps}
              format="currency"
              decimals={2}
              tooltip="Average analyst EPS estimate"
            />
            <MetricCard
              label="EPS Range"
              value={forwardEstimates.forward_eps_low && forwardEstimates.forward_eps_high
                ? `$${forwardEstimates.forward_eps_low?.toFixed(2)} - $${forwardEstimates.forward_eps_high?.toFixed(2)}`
                : null}
              format="text"
              tooltip="Low to high analyst EPS estimates"
            />
            <MetricCard
              label="# Analysts (EPS)"
              value={forwardEstimates.num_analysts_eps}
              format="number"
              decimals={0}
              tooltip="Number of analysts providing EPS estimates"
            />
            <MetricCard
              label="Forward Revenue"
              value={forwardEstimates.forward_revenue}
              format="currency"
              tooltip="Analyst revenue estimate"
            />
          </div>
        </div>
      )}
    </div>
  );
}

function ProfitabilitySection({ profitability }: { profitability: ProfitabilityMetrics }) {
  return (
    <div className="space-y-6">
      {/* Margins */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Profit Margins
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard
            label="Gross Margin"
            value={profitability.gross_margin}
            format="percent"
            tooltip="Gross Profit / Revenue"
            colorByValue
          />
          <MetricCard
            label="Operating Margin"
            value={profitability.operating_margin}
            format="percent"
            tooltip="Operating Income / Revenue"
            colorByValue
          />
          <MetricCard
            label="Net Margin"
            value={profitability.net_margin}
            format="percent"
            tooltip="Net Income / Revenue"
            colorByValue
          />
          <MetricCard
            label="EBITDA Margin"
            value={profitability.ebitda_margin}
            format="percent"
            tooltip="EBITDA / Revenue"
            colorByValue
          />
        </div>
      </div>

      {/* Additional Margins */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Other Margins
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard
            label="EBIT Margin"
            value={profitability.ebit_margin}
            format="percent"
            tooltip="EBIT / Revenue"
            colorByValue
          />
          <MetricCard
            label="FCF Margin"
            value={profitability.fcf_margin}
            format="percent"
            tooltip="Free Cash Flow / Revenue"
            colorByValue
          />
          <MetricCard
            label="Pre-tax Margin"
            value={profitability.pretax_margin}
            format="percent"
            tooltip="Pre-tax Income / Revenue"
            colorByValue
          />
        </div>
      </div>

      {/* Returns */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Return Metrics
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard
            label="ROE"
            value={profitability.roe}
            format="percent"
            tooltip="Return on Equity - Net Income / Shareholders Equity"
            colorByValue
          />
          <MetricCard
            label="ROA"
            value={profitability.roa}
            format="percent"
            tooltip="Return on Assets - Net Income / Total Assets"
            colorByValue
          />
          <MetricCard
            label="ROIC"
            value={profitability.roic}
            format="percent"
            tooltip="Return on Invested Capital"
            colorByValue
          />
          <MetricCard
            label="ROCE"
            value={profitability.roce}
            format="percent"
            tooltip="Return on Capital Employed"
            colorByValue
          />
        </div>
      </div>
    </div>
  );
}

function FinancialHealthSection({
  liquidity,
  leverage
}: {
  liquidity: LiquidityMetrics;
  leverage: LeverageMetrics;
}) {
  return (
    <div className="space-y-6">
      {/* Liquidity Ratios */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Liquidity Ratios
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard
            label="Current Ratio"
            value={liquidity.current_ratio}
            format="ratio"
            tooltip="Current Assets / Current Liabilities - >1 is healthy"
          />
          <MetricCard
            label="Quick Ratio"
            value={liquidity.quick_ratio}
            format="ratio"
            tooltip="(Current Assets - Inventory) / Current Liabilities"
          />
          <MetricCard
            label="Cash Ratio"
            value={liquidity.cash_ratio}
            format="ratio"
            tooltip="Cash / Current Liabilities"
          />
          <MetricCard
            label="Working Capital"
            value={liquidity.working_capital}
            format="currency"
            tooltip="Current Assets - Current Liabilities"
          />
        </div>
      </div>

      {/* Leverage Ratios */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Leverage Ratios
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard
            label="Debt/Equity"
            value={leverage.debt_to_equity}
            format="ratio"
            tooltip="Total Debt / Shareholders Equity - lower is less risky"
          />
          <MetricCard
            label="Debt/Assets"
            value={leverage.debt_to_assets}
            format="ratio"
            tooltip="Total Debt / Total Assets"
          />
          <MetricCard
            label="Debt/EBITDA"
            value={leverage.debt_to_ebitda}
            format="ratio"
            tooltip="Net Debt / EBITDA - <3 is generally healthy"
          />
          <MetricCard
            label="Interest Coverage"
            value={leverage.interest_coverage}
            format="ratio"
            tooltip="EBIT / Interest Expense - higher means more ability to pay interest"
            nullTooltip="Not applicable - company has net interest income (earns more from interest than it pays)"
          />
        </div>
      </div>

      {/* Debt Metrics */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Debt Metrics
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard
            label="Debt/Capital"
            value={leverage.debt_to_capital}
            format="ratio"
            tooltip="Total Debt / Total Capital"
          />
          <MetricCard
            label="Net Debt/EBITDA"
            value={leverage.net_debt_to_ebitda}
            format="ratio"
            tooltip="Net Debt / EBITDA"
          />
          <MetricCard
            label="Net Debt"
            value={leverage.net_debt}
            format="currency"
            tooltip="Total Debt - Cash & Equivalents"
          />
        </div>
      </div>
    </div>
  );
}

function EfficiencySection({ efficiency }: { efficiency: EfficiencyMetrics }) {
  return (
    <div className="space-y-6">
      {/* Turnover Ratios */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Turnover Ratios
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard
            label="Asset Turnover"
            value={efficiency.asset_turnover}
            format="ratio"
            tooltip="Revenue / Total Assets - higher is more efficient"
          />
          <MetricCard
            label="Inventory Turnover"
            value={efficiency.inventory_turnover}
            format="ratio"
            tooltip="COGS / Average Inventory - higher is better"
          />
          <MetricCard
            label="Receivables Turnover"
            value={efficiency.receivables_turnover}
            format="ratio"
            tooltip="Revenue / Average Receivables"
          />
          <MetricCard
            label="Payables Turnover"
            value={efficiency.payables_turnover}
            format="ratio"
            tooltip="COGS / Average Payables"
          />
        </div>
      </div>

      {/* Days Metrics */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Working Capital Days
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard
            label="DSO"
            value={efficiency.days_sales_outstanding}
            format="days"
            tooltip="Days Sales Outstanding - lower is better"
          />
          <MetricCard
            label="DIO"
            value={efficiency.days_inventory_outstanding}
            format="days"
            tooltip="Days Inventory Outstanding - lower is better"
          />
          <MetricCard
            label="DPO"
            value={efficiency.days_payables_outstanding}
            format="days"
            tooltip="Days Payables Outstanding"
          />
          <MetricCard
            label="Cash Conversion Cycle"
            value={efficiency.cash_conversion_cycle}
            format="days"
            tooltip="DSO + DIO - DPO - lower is better"
          />
        </div>
      </div>

      {/* Fixed Asset */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Asset Efficiency
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard
            label="Fixed Asset Turnover"
            value={efficiency.fixed_asset_turnover}
            format="ratio"
            tooltip="Revenue / Fixed Assets"
          />
        </div>
      </div>
    </div>
  );
}

function GrowthSection({ growth }: { growth: GrowthMetrics }) {
  return (
    <div className="space-y-6">
      {/* Year-over-Year Growth */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Year-over-Year Growth
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard
            label="Revenue Growth"
            value={growth.revenue_growth_yoy}
            format="percent"
            tooltip="Year-over-year revenue growth"
            colorByValue
          />
          <MetricCard
            label="EPS Growth"
            value={growth.eps_growth_yoy}
            format="percent"
            tooltip="Year-over-year EPS growth"
            colorByValue
          />
          <MetricCard
            label="Net Income Growth"
            value={growth.net_income_growth_yoy}
            format="percent"
            tooltip="Year-over-year net income growth"
            colorByValue
          />
          <MetricCard
            label="FCF Growth"
            value={growth.fcf_growth_yoy}
            format="percent"
            tooltip="Year-over-year free cash flow growth"
            colorByValue
          />
        </div>
      </div>

      {/* Multi-Year CAGR */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Compound Annual Growth Rate (CAGR)
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard
            label="Revenue 3Y CAGR"
            value={growth.revenue_growth_3y_cagr}
            format="percent"
            tooltip="3-year compound annual revenue growth"
            colorByValue
          />
          <MetricCard
            label="Revenue 5Y CAGR"
            value={growth.revenue_growth_5y_cagr}
            format="percent"
            tooltip="5-year compound annual revenue growth"
            colorByValue
          />
          <MetricCard
            label="EPS 3Y CAGR"
            value={growth.eps_growth_3y_cagr}
            format="percent"
            tooltip="3-year compound annual EPS growth"
            colorByValue
          />
          <MetricCard
            label="EPS 5Y CAGR"
            value={growth.eps_growth_5y_cagr}
            format="percent"
            tooltip="5-year compound annual EPS growth"
            colorByValue
          />
        </div>
      </div>

      {/* Other Growth Metrics */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Other Growth Metrics
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard
            label="Gross Profit Growth"
            value={growth.gross_profit_growth_yoy}
            format="percent"
            tooltip="Year-over-year gross profit growth"
            colorByValue
          />
          <MetricCard
            label="Operating Income Growth"
            value={growth.operating_income_growth_yoy}
            format="percent"
            tooltip="Year-over-year operating income growth"
            colorByValue
          />
          <MetricCard
            label="Book Value Growth"
            value={growth.book_value_growth_yoy}
            format="percent"
            tooltip="Year-over-year book value growth"
            colorByValue
          />
          <MetricCard
            label="Dividend 5Y CAGR"
            value={growth.dividend_growth_5y_cagr}
            format="percent"
            tooltip="5-year compound annual dividend growth"
            colorByValue
          />
        </div>
      </div>
    </div>
  );
}

function DividendsSection({
  dividends,
  perShare
}: {
  dividends: DividendMetrics;
  perShare: PerShareMetrics;
}) {
  return (
    <div className="space-y-6">
      {/* Dividend Metrics */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Dividend Metrics
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard
            label="Dividend Yield"
            value={dividends.dividend_yield}
            format="percent"
            tooltip="Annual Dividend / Share Price"
          />
          <MetricCard
            label="Dividend/Share"
            value={dividends.dividend_per_share}
            format="currency"
            decimals={2}
            tooltip="Annual Dividend Per Share"
          />
          <MetricCard
            label="Payout Ratio"
            value={dividends.payout_ratio}
            format="percent"
            tooltip="Dividends / Net Income"
            interpretation={dividends.payout_interpretation}
            interpretationColorFn={getPayoutColor}
          />
          <MetricCard
            label="Dividend Streak"
            value={dividends.consecutive_dividend_years}
            format="years"
            tooltip="Years of consecutive dividend payments"
          />
        </div>
      </div>

      {/* Dividend Details */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Dividend Details
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard
            label="Payment Frequency"
            value={dividends.dividend_frequency}
            format="text"
            tooltip="How often dividends are paid"
          />
          <MetricCard
            label="Ex-Dividend Date"
            value={dividends.ex_dividend_date}
            format="date"
            tooltip="Last ex-dividend date"
          />
          <MetricCard
            label="Payment Date"
            value={dividends.payment_date}
            format="date"
            tooltip="Last payment date"
          />
        </div>
      </div>

      {/* Per Share Metrics */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Per Share Metrics
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard
            label="EPS (Diluted)"
            value={perShare.eps_diluted}
            format="currency"
            decimals={2}
            tooltip="Diluted Earnings Per Share"
          />
          <MetricCard
            label="Book Value/Share"
            value={perShare.book_value_per_share}
            format="currency"
            decimals={2}
            tooltip="Book Value Per Share"
          />
          <MetricCard
            label="FCF/Share"
            value={perShare.fcf_per_share}
            format="currency"
            decimals={2}
            tooltip="Free Cash Flow Per Share"
          />
          <MetricCard
            label="Revenue/Share"
            value={perShare.revenue_per_share}
            format="currency"
            decimals={2}
            tooltip="Revenue Per Share"
          />
        </div>
      </div>

      {/* Graham Number */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-4 uppercase tracking-wide">
          Value Investing
        </h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard
            label="Graham Number"
            value={perShare.graham_number}
            format="currency"
            decimals={2}
            tooltip="Fair value estimate based on Ben Graham formula"
          />
          <MetricCard
            label="Tangible Book/Share"
            value={perShare.tangible_book_per_share}
            format="currency"
            decimals={2}
            tooltip="Book Value excluding intangibles"
          />
          <MetricCard
            label="Cash/Share"
            value={perShare.cash_per_share}
            format="currency"
            decimals={2}
            tooltip="Cash Per Share"
          />
        </div>
      </div>
    </div>
  );
}

function QualitySection({
  qualityScores,
  valuation
}: {
  qualityScores: QualityScores;
  valuation: ValuationMetrics;
}) {
  return (
    <div className="space-y-6">
      {/* Altman Z-Score */}
      <div className="bg-ic-bg-secondary rounded-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h4 className="text-lg font-semibold text-ic-text-primary">Altman Z-Score</h4>
            <p className="text-sm text-ic-text-muted">Bankruptcy risk indicator</p>
          </div>
          {qualityScores.altman_z_interpretation && (
            <span className={cn(
              'px-3 py-1 rounded-full text-sm font-medium',
              getZScoreColor(qualityScores.altman_z_interpretation)
            )}>
              {qualityScores.altman_z_interpretation.toUpperCase()}
            </span>
          )}
        </div>
        <div className="text-4xl font-bold text-ic-text-primary mb-2">
          {qualityScores.altman_z_score?.toFixed(2) ?? '—'}
        </div>
        <p className="text-sm text-ic-text-muted">
          {qualityScores.altman_z_description || 'Z-Score helps predict bankruptcy probability within 2 years'}
        </p>
        <div className="mt-4 pt-4 border-t border-ic-border">
          <div className="text-xs text-ic-text-dim space-y-1">
            <p>&gt; 2.99: Safe Zone (Low risk)</p>
            <p>1.81 - 2.99: Grey Zone (Moderate risk)</p>
            <p>&lt; 1.81: Distress Zone (High risk)</p>
          </div>
        </div>
      </div>

      {/* Piotroski F-Score */}
      <div className="bg-ic-bg-secondary rounded-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h4 className="text-lg font-semibold text-ic-text-primary">Piotroski F-Score</h4>
            <p className="text-sm text-ic-text-muted">Financial strength score (0-9)</p>
          </div>
          {qualityScores.piotroski_f_interpretation && (
            <span className={cn(
              'px-3 py-1 rounded-full text-sm font-medium',
              getFScoreColor(qualityScores.piotroski_f_interpretation)
            )}>
              {qualityScores.piotroski_f_interpretation.toUpperCase()}
            </span>
          )}
        </div>
        <div className="text-4xl font-bold text-ic-text-primary mb-2">
          {qualityScores.piotroski_f_score ?? '—'} <span className="text-lg text-ic-text-muted">/ 9</span>
        </div>
        <p className="text-sm text-ic-text-muted">
          {qualityScores.piotroski_f_description || 'F-Score measures overall financial health'}
        </p>

        {/* F-Score Visual Bar */}
        {qualityScores.piotroski_f_score !== null && qualityScores.piotroski_f_score !== undefined && (
          <div className="mt-4">
            <div className="flex gap-1">
              {[1, 2, 3, 4, 5, 6, 7, 8, 9].map((n) => (
                <div
                  key={n}
                  className={cn(
                    'h-3 flex-1 rounded-sm',
                    n <= (qualityScores.piotroski_f_score ?? 0)
                      ? n <= 4 ? 'bg-red-400' : n <= 7 ? 'bg-yellow-400' : 'bg-green-400'
                      : 'bg-ic-border'
                  )}
                />
              ))}
            </div>
            <div className="flex justify-between text-xs text-ic-text-dim mt-1">
              <span>Weak</span>
              <span>Average</span>
              <span>Strong</span>
            </div>
          </div>
        )}

        <div className="mt-4 pt-4 border-t border-ic-border">
          <div className="text-xs text-ic-text-dim space-y-1">
            <p>8-9: Strong financial health</p>
            <p>5-7: Average financial health</p>
            <p>0-4: Weak financial health</p>
          </div>
        </div>
      </div>

      {/* PEG Interpretation */}
      {valuation.peg_ratio !== null && (
        <div className="bg-ic-bg-secondary rounded-lg p-6">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h4 className="text-lg font-semibold text-ic-text-primary">PEG Ratio Analysis</h4>
              <p className="text-sm text-ic-text-muted">Price/Earnings to Growth</p>
            </div>
            {valuation.peg_interpretation && (
              <span className={cn(
                'px-3 py-1 rounded-full text-sm font-medium',
                getPEGColor(valuation.peg_interpretation),
                'bg-ic-bg-tertiary'
              )}>
                {valuation.peg_interpretation.toUpperCase()}
              </span>
            )}
          </div>
          <div className="text-4xl font-bold text-ic-text-primary mb-2">
            {valuation.peg_ratio?.toFixed(2) ?? '—'}
          </div>
          <div className="mt-4 pt-4 border-t border-ic-border">
            <div className="text-xs text-ic-text-dim space-y-1">
              <p>&lt; 1.0: Undervalued (growing faster than P/E suggests)</p>
              <p>1.0 - 1.5: Fair value</p>
              <p>1.5 - 2.0: May be overvalued</p>
              <p>&gt; 2.0: Overvalued</p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Helper Components
// ============================================================================

interface MetricCardProps {
  label: string;
  value: number | string | null | undefined;
  format: 'ratio' | 'percent' | 'currency' | 'number' | 'days' | 'years' | 'date' | 'text';
  decimals?: number;
  tooltip?: string;
  nullTooltip?: string; // Tooltip to show when value is NULL
  colorByValue?: boolean;
  interpretation?: string | null;
  interpretationColorFn?: (interp: string | null) => string;
}

function MetricCard({
  label,
  value,
  format,
  decimals = 2,
  tooltip,
  nullTooltip,
  colorByValue = false,
  interpretation,
  interpretationColorFn,
}: MetricCardProps) {
  const formatValue = () => {
    if (value === null || value === undefined) return '—';
    if (typeof value === 'string' && format === 'text') return value;
    if (typeof value === 'string' && format === 'date') {
      try {
        return new Date(value).toLocaleDateString();
      } catch {
        return value;
      }
    }

    const numValue = typeof value === 'number' ? value : parseFloat(String(value));
    if (isNaN(numValue)) return '—';

    switch (format) {
      case 'percent':
        return `${numValue.toFixed(decimals)}%`;
      case 'currency':
        if (Math.abs(numValue) >= 1e12) return `$${(numValue / 1e12).toFixed(decimals)}T`;
        if (Math.abs(numValue) >= 1e9) return `$${(numValue / 1e9).toFixed(decimals)}B`;
        if (Math.abs(numValue) >= 1e6) return `$${(numValue / 1e6).toFixed(decimals)}M`;
        return `$${numValue.toFixed(decimals)}`;
      case 'days':
        return `${numValue.toFixed(0)} days`;
      case 'years':
        return `${numValue.toFixed(0)} years`;
      case 'ratio':
      case 'number':
      default:
        return numValue.toFixed(decimals);
    }
  };

  const getValueColor = () => {
    if (!colorByValue || value === null || value === undefined) return 'text-ic-text-primary';
    const numValue = typeof value === 'number' ? value : parseFloat(String(value));
    if (isNaN(numValue)) return 'text-ic-text-primary';
    return numValue >= 0 ? 'text-ic-positive' : 'text-ic-negative';
  };

  // Use nullTooltip when value is NULL, otherwise use regular tooltip
  const displayTooltip = (value === null || value === undefined) && nullTooltip ? nullTooltip : tooltip;

  return (
    <div className="bg-ic-bg-secondary rounded-lg p-4" title={displayTooltip}>
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm text-ic-text-muted">{label}</span>
        {interpretation && interpretationColorFn && (
          <span className={cn(
            'text-xs px-2 py-0.5 rounded-full',
            interpretationColorFn(interpretation),
            'bg-opacity-20'
          )}>
            {interpretation}
          </span>
        )}
      </div>
      <div className={cn('text-xl font-semibold', getValueColor())}>
        {formatValue()}
      </div>
    </div>
  );
}

function MetricsLoadingSkeleton() {
  return (
    <div className="p-6 animate-pulse">
      <div className="h-6 bg-ic-bg-tertiary rounded w-48 mb-2"></div>
      <div className="h-4 bg-ic-bg-tertiary rounded w-64 mb-6"></div>

      <div className="flex gap-2 mb-6">
        {[1, 2, 3, 4, 5, 6, 7].map((i) => (
          <div key={i} className="h-8 bg-ic-bg-tertiary rounded w-24"></div>
        ))}
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {[1, 2, 3, 4, 5, 6, 7, 8].map((i) => (
          <div key={i} className="bg-ic-bg-secondary rounded-lg p-4">
            <div className="h-4 bg-ic-bg-tertiary rounded w-20 mb-2"></div>
            <div className="h-6 bg-ic-bg-tertiary rounded w-16"></div>
          </div>
        ))}
      </div>
    </div>
  );
}
