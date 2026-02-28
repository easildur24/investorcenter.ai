'use client';

import React, { useState, useEffect } from 'react';
import { cn } from '@/lib/utils';
import { CalculationTooltip } from '@/components/ui/Tooltip';
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
  AnalystRatings,
  getZScoreColor,
  getFScoreColor,
  getPEGColor,
  getPayoutColor,
  getConsensusColor,
  getConsensusBgColor,
  calculateTargetUpside,
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
  analystRatingsConfigs,
  MetricDisplayConfig,
} from '@/types/metrics';
import { getComprehensiveMetrics } from '@/lib/api/metrics';
import type { RedFlag } from '@/lib/types/fundamentals';
import { useHealthSummary } from '@/lib/hooks/useHealthSummary';

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
  | 'quality'
  | 'analyst';

const categoryTabs: { id: MetricCategory; label: string }[] = [
  { id: 'valuation', label: 'Valuation' },
  { id: 'profitability', label: 'Profitability' },
  { id: 'financial_health', label: 'Financial Health' },
  { id: 'efficiency', label: 'Efficiency' },
  { id: 'growth', label: 'Growth' },
  { id: 'dividends', label: 'Dividends' },
  { id: 'quality', label: 'Quality Scores' },
  { id: 'analyst', label: 'Analyst Ratings' },
];

/** Small inline dot shown next to a metric that has a related red flag. */
function MetricRedFlagIndicator({
  metricKey,
  redFlags,
}: {
  metricKey: string;
  redFlags: RedFlag[];
}) {
  const match = redFlags.find((f) =>
    f.related_metrics.some((m) => m.toLowerCase() === metricKey.toLowerCase())
  );
  if (!match) return null;

  const dotColor =
    match.severity === 'high'
      ? 'bg-red-400'
      : match.severity === 'medium'
        ? 'bg-orange-400'
        : 'bg-yellow-400';

  return (
    <span
      className={cn('inline-block w-2 h-2 rounded-full ml-1 flex-shrink-0', dotColor)}
      title={match.title}
      aria-label={`Red flag: ${match.title}`}
    />
  );
}

export default function MetricsTab({ symbol }: MetricsTabProps) {
  const [data, setData] = useState<ComprehensiveMetricsResponse | null>(null);
  const [activeCategory, setActiveCategory] = useState<MetricCategory>('valuation');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Health summary for inline red-flag indicators
  const { data: healthData } = useHealthSummary(symbol);
  const redFlags: RedFlag[] = healthData?.red_flags ?? [];

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
            redFlags={redFlags}
          />
        )}
        {activeCategory === 'profitability' && (
          <ProfitabilitySection profitability={data.data.profitability} redFlags={redFlags} />
        )}
        {activeCategory === 'financial_health' && (
          <FinancialHealthSection
            liquidity={data.data.liquidity}
            leverage={data.data.leverage}
            redFlags={redFlags}
          />
        )}
        {activeCategory === 'efficiency' && <EfficiencySection efficiency={data.data.efficiency} />}
        {activeCategory === 'growth' && <GrowthSection growth={data.data.growth} />}
        {activeCategory === 'dividends' && (
          <DividendsSection dividends={data.data.dividends} perShare={data.data.per_share} />
        )}
        {activeCategory === 'quality' && (
          <QualitySection
            qualityScores={data.data.quality_scores}
            valuation={data.data.valuation}
          />
        )}
        {activeCategory === 'analyst' && (
          <AnalystRatingsSection
            analystRatings={data.data.analyst_ratings}
            currentPrice={data.meta.current_price}
          />
        )}
      </div>

      {/* Data Source Footer */}
      <div className="mt-8 p-4 bg-blue-50 rounded-lg">
        <h4 className="text-sm font-medium text-blue-800 mb-1">About This Data</h4>
        <p className="text-sm text-blue-700">
          Financial metrics are sourced from Financial Modeling Prep (FMP) API. All values are
          trailing twelve months (TTM) unless otherwise specified. Data is updated in real-time.
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
  forwardEstimates,
  redFlags = [],
}: {
  valuation: ValuationMetrics;
  forwardEstimates: ForwardEstimates;
  redFlags?: RedFlag[];
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
            calculationTooltip={{
              formula: 'Share Price / EPS (TTM)',
              description:
                'Uses trailing 12-month diluted EPS. Price updates real-time; EPS updates quarterly after earnings. Source: FMP',
            }}
            flagIndicator={<MetricRedFlagIndicator metricKey="pe_ratio" redFlags={redFlags} />}
          />
          <MetricCard
            label="Forward P/E"
            value={valuation.forward_pe}
            format="ratio"
            tooltip="P/E based on estimated future earnings"
            calculationTooltip={{
              formula: 'Share Price / Forward EPS Estimate',
              description: 'Based on analyst earnings estimates',
            }}
          />
          <MetricCard
            label="PEG Ratio"
            value={valuation.peg_ratio}
            format="ratio"
            tooltip="P/E to Growth - <1 suggests undervalued relative to growth"
            calculationTooltip={{
              formula: 'P/E Ratio / Expected EPS Growth Rate (%)',
              description:
                'Growth rate based on 5-year historical EPS CAGR or analyst estimates. <1 suggests undervalued relative to growth. Source: FMP',
            }}
            interpretation={valuation.peg_interpretation}
            interpretationColorFn={getPEGColor}
            flagIndicator={<MetricRedFlagIndicator metricKey="peg_ratio" redFlags={redFlags} />}
          />
          <MetricCard
            label="P/B Ratio"
            value={valuation.pb_ratio}
            format="ratio"
            tooltip="Price to Book value"
            calculationTooltip={{
              formula: 'Share Price / Book Value Per Share',
              description: 'Compares market value to accounting value',
            }}
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
            calculationTooltip={{
              formula: 'Market Cap / Total Revenue',
              description: 'Useful for unprofitable companies',
            }}
          />
          <MetricCard
            label="P/FCF"
            value={valuation.price_to_fcf}
            format="ratio"
            tooltip="Price to Free Cash Flow"
            calculationTooltip={{
              formula: 'Market Cap / Free Cash Flow',
              description: 'FCF = Operating Cash Flow - CapEx',
            }}
          />
          <MetricCard
            label="EV/EBITDA"
            value={valuation.ev_to_ebitda}
            format="ratio"
            tooltip="Enterprise Value to EBITDA"
            calculationTooltip={{
              formula: 'Enterprise Value / EBITDA',
              description: 'Capital structure-neutral valuation',
            }}
          />
          <MetricCard
            label="EV/Sales"
            value={valuation.ev_to_sales}
            format="ratio"
            tooltip="Enterprise Value to Sales"
            calculationTooltip={{
              formula: 'Enterprise Value / Total Revenue',
              description: 'Compares total firm value to sales',
            }}
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
            calculationTooltip={{
              formula: 'EPS / Share Price × 100',
              description: 'Inverse of P/E ratio - higher is better',
            }}
          />
          <MetricCard
            label="FCF Yield"
            value={valuation.fcf_yield}
            format="percent"
            tooltip="Free Cash Flow Yield - higher is better"
            calculationTooltip={{
              formula: 'Free Cash Flow / Market Cap × 100',
              description: 'Cash return on investment - higher is better',
            }}
          />
          <MetricCard
            label="Market Cap"
            value={valuation.market_cap}
            format="currency"
            tooltip="Total market capitalization"
            calculationTooltip={{
              formula: 'Share Price × Shares Outstanding',
              description: 'Total market value of equity',
            }}
          />
          <MetricCard
            label="Enterprise Value"
            value={valuation.enterprise_value}
            format="currency"
            tooltip="Market Cap + Debt - Cash"
            calculationTooltip={{
              formula: 'Market Cap + Total Debt - Cash',
              description: 'Total firm value including debt',
            }}
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
              calculationTooltip={{
                formula: 'Average of Analyst EPS Estimates',
                description: 'Consensus estimate for next fiscal year',
              }}
            />
            <MetricCard
              label="EPS Range"
              value={
                forwardEstimates.forward_eps_low && forwardEstimates.forward_eps_high
                  ? `$${forwardEstimates.forward_eps_low?.toFixed(2)} - $${forwardEstimates.forward_eps_high?.toFixed(2)}`
                  : null
              }
              format="text"
              tooltip="Low to high analyst EPS estimates"
              calculationTooltip={{
                formula: 'Min(Analyst Estimates) - Max(Analyst Estimates)',
                description: 'Range shows analyst disagreement',
              }}
            />
            <MetricCard
              label="# Analysts (EPS)"
              value={forwardEstimates.num_analysts_eps}
              format="number"
              decimals={0}
              tooltip="Number of analysts providing EPS estimates"
              calculationTooltip={{
                formula: 'Count of Analyst Estimates',
                description: 'More analysts = more reliable consensus',
              }}
            />
            <MetricCard
              label="Forward Revenue"
              value={forwardEstimates.forward_revenue}
              format="currency"
              tooltip="Analyst revenue estimate"
              calculationTooltip={{
                formula: 'Average of Analyst Revenue Estimates',
                description: 'Consensus revenue estimate for next fiscal year',
              }}
            />
          </div>
        </div>
      )}
    </div>
  );
}

function ProfitabilitySection({
  profitability,
  redFlags = [],
}: {
  profitability: ProfitabilityMetrics;
  redFlags?: RedFlag[];
}) {
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
            calculationTooltip={{
              formula: '(Revenue - COGS) / Revenue × 100',
              description: 'Measures production efficiency',
            }}
            colorByValue
          />
          <MetricCard
            label="Operating Margin"
            value={profitability.operating_margin}
            format="percent"
            tooltip="Operating Income / Revenue"
            calculationTooltip={{
              formula: 'Operating Income / Revenue × 100',
              description: 'Profit from core operations',
            }}
            colorByValue
          />
          <MetricCard
            label="Net Margin"
            value={profitability.net_margin}
            format="percent"
            tooltip="Net Income / Revenue"
            calculationTooltip={{
              formula: 'Net Income / Revenue × 100',
              description: 'Bottom line profitability',
            }}
            colorByValue
            flagIndicator={<MetricRedFlagIndicator metricKey="net_margin" redFlags={redFlags} />}
          />
          <MetricCard
            label="EBITDA Margin"
            value={profitability.ebitda_margin}
            format="percent"
            tooltip="EBITDA / Revenue"
            calculationTooltip={{
              formula: 'EBITDA / Revenue × 100',
              description: 'Operating profitability before D&A',
            }}
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
            calculationTooltip={{
              formula: 'EBIT / Revenue × 100',
              description: 'Operating income margin',
            }}
            colorByValue
          />
          <MetricCard
            label="FCF Margin"
            value={profitability.fcf_margin}
            format="percent"
            tooltip="Free Cash Flow / Revenue"
            calculationTooltip={{
              formula: 'Free Cash Flow / Revenue × 100',
              description: 'Cash generation efficiency',
            }}
            colorByValue
          />
          <MetricCard
            label="Pre-tax Margin"
            value={profitability.pretax_margin}
            format="percent"
            tooltip="Pre-tax Income / Revenue"
            calculationTooltip={{
              formula: 'Pre-tax Income / Revenue × 100',
              description: 'Profitability before taxes',
            }}
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
            calculationTooltip={{
              formula: 'Net Income (TTM) / Avg Shareholders Equity × 100',
              description:
                'Return generated for shareholders. Very high ROE (>100%) may indicate significant stock buybacks reducing equity base. Source: FMP',
            }}
            colorByValue
          />
          <MetricCard
            label="ROA"
            value={profitability.roa}
            format="percent"
            tooltip="Return on Assets - Net Income / Total Assets"
            calculationTooltip={{
              formula: 'Net Income / Total Assets × 100',
              description: 'How efficiently assets generate profit',
            }}
            colorByValue
          />
          <MetricCard
            label="ROIC"
            value={profitability.roic}
            format="percent"
            tooltip="Return on Invested Capital"
            calculationTooltip={{
              formula: 'NOPAT / Invested Capital × 100',
              description: 'NOPAT = Net Operating Profit After Tax',
            }}
            colorByValue
          />
          <MetricCard
            label="ROCE"
            value={profitability.roce}
            format="percent"
            tooltip="Return on Capital Employed"
            calculationTooltip={{
              formula: 'EBIT / Capital Employed × 100',
              description: 'Capital Employed = Total Assets - Current Liabilities',
            }}
            colorByValue
          />
        </div>
      </div>
    </div>
  );
}

function FinancialHealthSection({
  liquidity,
  leverage,
  redFlags = [],
}: {
  liquidity: LiquidityMetrics;
  leverage: LeverageMetrics;
  redFlags?: RedFlag[];
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
            calculationTooltip={{
              formula: 'Current Assets / Current Liabilities',
              description: '>1 indicates ability to pay short-term debts',
            }}
            flagIndicator={<MetricRedFlagIndicator metricKey="current_ratio" redFlags={redFlags} />}
          />
          <MetricCard
            label="Quick Ratio"
            value={liquidity.quick_ratio}
            format="ratio"
            tooltip="(Current Assets - Inventory) / Current Liabilities"
            calculationTooltip={{
              formula: '(Current Assets - Inventory) / Current Liabilities',
              description:
                "Also called 'Acid Test'. Excludes inventory as it may not be quickly liquidated. >1.0 indicates strong short-term liquidity. Source: FMP",
            }}
          />
          <MetricCard
            label="Cash Ratio"
            value={liquidity.cash_ratio}
            format="ratio"
            tooltip="Cash / Current Liabilities"
            calculationTooltip={{
              formula: 'Cash & Equivalents / Current Liabilities',
              description: 'Most conservative liquidity measure',
            }}
          />
          <MetricCard
            label="Working Capital"
            value={liquidity.working_capital}
            format="currency"
            tooltip="Current Assets - Current Liabilities"
            calculationTooltip={{
              formula: 'Current Assets - Current Liabilities',
              description: 'Short-term financial health buffer',
            }}
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
            calculationTooltip={{
              formula: 'Total Debt / Shareholders Equity',
              description: 'Lower ratio = less financial risk',
            }}
            flagIndicator={
              <MetricRedFlagIndicator metricKey="debt_to_equity" redFlags={redFlags} />
            }
          />
          <MetricCard
            label="Debt/Assets"
            value={leverage.debt_to_assets}
            format="ratio"
            tooltip="Total Debt / Total Assets"
            calculationTooltip={{
              formula: 'Total Debt / Total Assets',
              description: 'Proportion of assets financed by debt',
            }}
          />
          <MetricCard
            label="Debt/EBITDA"
            value={leverage.debt_to_ebitda}
            format="ratio"
            tooltip="Net Debt / EBITDA - <3 is generally healthy"
            calculationTooltip={{
              formula: 'Total Debt / EBITDA',
              description: 'Years to pay off debt with EBITDA (<3 is healthy)',
            }}
          />
          <MetricCard
            label="Interest Coverage"
            value={leverage.interest_coverage}
            format="ratio"
            tooltip="EBIT / Interest Expense - higher means more ability to pay interest"
            nullTooltip="Not applicable - company has net interest income (earns more from interest than it pays)"
            calculationTooltip={{
              formula: 'EBIT / Interest Expense',
              description: 'Ability to pay interest on debt',
            }}
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
            calculationTooltip={{
              formula: 'Total Debt / (Total Debt + Equity)',
              description: 'Debt as portion of total capitalization',
            }}
          />
          <MetricCard
            label="Net Debt/EBITDA"
            value={leverage.net_debt_to_ebitda}
            format="ratio"
            tooltip="Net Debt / EBITDA"
            calculationTooltip={{
              formula: '(Total Debt - Cash) / EBITDA',
              description: 'Net leverage after cash offset',
            }}
          />
          <MetricCard
            label="Net Debt"
            value={leverage.net_debt}
            format="currency"
            tooltip="Total Debt - Cash & Equivalents"
            calculationTooltip={{
              formula: 'Total Debt - Cash & Equivalents',
              description: 'Debt after subtracting cash on hand',
            }}
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
            calculationTooltip={{
              formula: 'Revenue / Average Total Assets',
              description: 'Sales generated per dollar of assets',
            }}
          />
          <MetricCard
            label="Inventory Turnover"
            value={efficiency.inventory_turnover}
            format="ratio"
            tooltip="COGS / Average Inventory - higher is better"
            calculationTooltip={{
              formula: 'Cost of Goods Sold / Average Inventory',
              description: 'How often inventory is sold and replaced',
            }}
          />
          <MetricCard
            label="Receivables Turnover"
            value={efficiency.receivables_turnover}
            format="ratio"
            tooltip="Revenue / Average Receivables"
            calculationTooltip={{
              formula: 'Revenue / Average Accounts Receivable',
              description: 'How quickly receivables are collected',
            }}
          />
          <MetricCard
            label="Payables Turnover"
            value={efficiency.payables_turnover}
            format="ratio"
            tooltip="COGS / Average Payables"
            calculationTooltip={{
              formula: 'COGS / Average Accounts Payable',
              description: 'How quickly company pays suppliers',
            }}
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
            calculationTooltip={{
              formula: '(Accounts Receivable / Revenue) × 365',
              description: 'Average days to collect payment',
            }}
          />
          <MetricCard
            label="DIO"
            value={efficiency.days_inventory_outstanding}
            format="days"
            tooltip="Days Inventory Outstanding - lower is better"
            calculationTooltip={{
              formula: '(Inventory / COGS) × 365',
              description: 'Average days inventory is held',
            }}
          />
          <MetricCard
            label="DPO"
            value={efficiency.days_payables_outstanding}
            format="days"
            tooltip="Days Payables Outstanding"
            calculationTooltip={{
              formula: '(Accounts Payable / COGS) × 365',
              description: 'Average days to pay suppliers',
            }}
          />
          <MetricCard
            label="Cash Conversion Cycle"
            value={efficiency.cash_conversion_cycle}
            format="days"
            tooltip="DSO + DIO - DPO - lower is better"
            calculationTooltip={{
              formula: 'DSO + DIO - DPO',
              description: 'Days to convert inventory to cash',
            }}
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
            calculationTooltip={{
              formula: 'Revenue / Net Property, Plant & Equipment',
              description: 'Efficiency of fixed asset utilization',
            }}
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
            calculationTooltip={{
              formula: '(Revenue₁ - Revenue₀) / Revenue₀ × 100',
              description: 'Change in revenue vs prior year',
            }}
            colorByValue
          />
          <MetricCard
            label="EPS Growth"
            value={growth.eps_growth_yoy}
            format="percent"
            tooltip="Year-over-year EPS growth"
            calculationTooltip={{
              formula: '(EPS₁ - EPS₀) / |EPS₀| × 100',
              description: 'Change in earnings per share vs prior year',
            }}
            colorByValue
          />
          <MetricCard
            label="Net Income Growth"
            value={growth.net_income_growth_yoy}
            format="percent"
            tooltip="Year-over-year net income growth"
            calculationTooltip={{
              formula: '(Net Income₁ - Net Income₀) / |Net Income₀| × 100',
              description: 'Change in net income vs prior year',
            }}
            colorByValue
          />
          <MetricCard
            label="FCF Growth"
            value={growth.fcf_growth_yoy}
            format="percent"
            tooltip="Year-over-year free cash flow growth"
            calculationTooltip={{
              formula: '(FCF₁ - FCF₀) / |FCF₀| × 100',
              description: 'Change in free cash flow vs prior year',
            }}
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
            calculationTooltip={{
              formula: '(Revenue₃ / Revenue₀)^(1/3) - 1',
              description: 'Annualized revenue growth over 3 years',
            }}
            colorByValue
          />
          <MetricCard
            label="Revenue 5Y CAGR"
            value={growth.revenue_growth_5y_cagr}
            format="percent"
            tooltip="5-year compound annual revenue growth"
            calculationTooltip={{
              formula: '(Revenue₅ / Revenue₀)^(1/5) - 1',
              description: 'Annualized revenue growth over 5 years',
            }}
            colorByValue
          />
          <MetricCard
            label="EPS 3Y CAGR"
            value={growth.eps_growth_3y_cagr}
            format="percent"
            tooltip="3-year compound annual EPS growth"
            calculationTooltip={{
              formula: '(EPS₃ / EPS₀)^(1/3) - 1',
              description: 'Annualized EPS growth over 3 years',
            }}
            colorByValue
          />
          <MetricCard
            label="EPS 5Y CAGR"
            value={growth.eps_growth_5y_cagr}
            format="percent"
            tooltip="5-year compound annual EPS growth"
            calculationTooltip={{
              formula: '(EPS₅ / EPS₀)^(1/5) - 1',
              description: 'Annualized EPS growth over 5 years',
            }}
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
            calculationTooltip={{
              formula: '(Gross Profit₁ - Gross Profit₀) / Gross Profit₀ × 100',
              description: 'Change in gross profit vs prior year',
            }}
            colorByValue
          />
          <MetricCard
            label="Operating Income Growth"
            value={growth.operating_income_growth_yoy}
            format="percent"
            tooltip="Year-over-year operating income growth"
            calculationTooltip={{
              formula: '(Op Income₁ - Op Income₀) / |Op Income₀| × 100',
              description: 'Change in operating income vs prior year',
            }}
            colorByValue
          />
          <MetricCard
            label="Book Value Growth"
            value={growth.book_value_growth_yoy}
            format="percent"
            tooltip="Year-over-year book value growth"
            calculationTooltip={{
              formula: '(Book Value₁ - Book Value₀) / Book Value₀ × 100',
              description: 'Change in shareholders equity vs prior year',
            }}
            colorByValue
          />
          <MetricCard
            label="Dividend 5Y CAGR"
            value={growth.dividend_growth_5y_cagr}
            format="percent"
            tooltip="5-year compound annual dividend growth"
            calculationTooltip={{
              formula: '(Dividend₅ / Dividend₀)^(1/5) - 1',
              description: 'Annualized dividend growth over 5 years',
            }}
            colorByValue
          />
        </div>
      </div>
    </div>
  );
}

function DividendsSection({
  dividends,
  perShare,
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
            calculationTooltip={{
              formula: 'Annual Dividend Per Share / Share Price × 100',
              description: 'Income return on investment',
            }}
          />
          <MetricCard
            label="Dividend/Share"
            value={perShare.dividend_per_share}
            format="currency"
            decimals={2}
            tooltip="Annual Dividend Per Share"
            calculationTooltip={{
              formula: 'Total Dividends Paid / Shares Outstanding',
              description: 'Annual dividend amount per share',
            }}
          />
          <MetricCard
            label="Payout Ratio"
            value={dividends.payout_ratio}
            format="percent"
            tooltip="Dividends / Net Income"
            calculationTooltip={{
              formula: 'Dividends Paid / Net Income × 100',
              description: 'Proportion of earnings paid as dividends',
            }}
            interpretation={dividends.payout_interpretation}
            interpretationColorFn={getPayoutColor}
          />
          <MetricCard
            label="Dividend Streak"
            value={dividends.consecutive_dividend_years}
            format="years"
            tooltip="Years of consecutive dividend payments"
            calculationTooltip={{
              formula: 'Count of Consecutive Years with Dividends',
              description: 'Track record of dividend payments',
            }}
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
            calculationTooltip={{
              formula: 'Quarterly / Monthly / Annual / Semi-Annual',
              description: 'How frequently dividends are distributed',
            }}
          />
          <MetricCard
            label="Ex-Dividend Date"
            value={dividends.ex_dividend_date}
            format="date"
            tooltip="Last ex-dividend date"
            calculationTooltip={{
              formula: 'Date (from company announcement)',
              description: 'Must own before this date to receive dividend',
            }}
          />
          <MetricCard
            label="Payment Date"
            value={dividends.payment_date}
            format="date"
            tooltip="Last payment date"
            calculationTooltip={{
              formula: 'Date (from company announcement)',
              description: 'When dividend is actually paid',
            }}
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
            calculationTooltip={{
              formula: 'Net Income / Diluted Shares Outstanding',
              description: 'Earnings per share including stock options',
            }}
          />
          <MetricCard
            label="Book Value/Share"
            value={perShare.book_value_per_share}
            format="currency"
            decimals={2}
            tooltip="Book Value Per Share"
            calculationTooltip={{
              formula: 'Shareholders Equity / Shares Outstanding',
              description: 'Net asset value per share',
            }}
          />
          <MetricCard
            label="FCF/Share"
            value={perShare.fcf_per_share}
            format="currency"
            decimals={2}
            tooltip="Free Cash Flow Per Share"
            calculationTooltip={{
              formula: 'Free Cash Flow / Shares Outstanding',
              description: 'Cash generated per share',
            }}
          />
          <MetricCard
            label="Revenue/Share"
            value={perShare.revenue_per_share}
            format="currency"
            decimals={2}
            tooltip="Revenue Per Share"
            calculationTooltip={{
              formula: 'Total Revenue / Shares Outstanding',
              description: 'Sales per share',
            }}
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
            calculationTooltip={{
              formula: '√(22.5 × EPS × Book Value Per Share)',
              description: "Ben Graham's intrinsic value estimate",
            }}
          />
          <MetricCard
            label="Tangible Book/Share"
            value={perShare.tangible_book_per_share}
            format="currency"
            decimals={2}
            tooltip="Book Value excluding intangibles"
            calculationTooltip={{
              formula: '(Total Equity - Intangible Assets) / Shares',
              description: 'Physical asset value per share',
            }}
          />
          <MetricCard
            label="Cash/Share"
            value={perShare.cash_per_share}
            format="currency"
            decimals={2}
            tooltip="Cash Per Share"
            calculationTooltip={{
              formula: 'Total Cash & Equivalents / Shares Outstanding',
              description:
                'Uses total cash (not net cash after debt). For net cash position, subtract debt per share. Source: FMP',
            }}
          />
        </div>
      </div>
    </div>
  );
}

function QualitySection({
  qualityScores,
  valuation,
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
            <span
              className={cn(
                'px-3 py-1 rounded-full text-sm font-medium',
                getZScoreColor(qualityScores.altman_z_interpretation)
              )}
            >
              {qualityScores.altman_z_interpretation.toUpperCase()}
            </span>
          )}
        </div>
        <CalculationTooltip
          label="Altman Z-Score"
          formula="1.2×WC/TA + 1.4×RE/TA + 3.3×EBIT/TA + 0.6×MVE/TL + 1.0×S/TA"
          description="WC=Working Capital, RE=Retained Earnings, MVE=Market Value of Equity, TL=Total Liabilities, S=Sales, TA=Total Assets"
        >
          <div className="text-4xl font-bold text-ic-text-primary mb-2 cursor-help inline-block">
            {qualityScores.altman_z_score?.toFixed(2) ?? '—'}
          </div>
        </CalculationTooltip>
        <p className="text-sm text-ic-text-muted">
          {qualityScores.altman_z_description ||
            'Z-Score helps predict bankruptcy probability within 2 years'}
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
            <span
              className={cn(
                'px-3 py-1 rounded-full text-sm font-medium',
                getFScoreColor(qualityScores.piotroski_f_interpretation)
              )}
            >
              {qualityScores.piotroski_f_interpretation.toUpperCase()}
            </span>
          )}
        </div>
        <CalculationTooltip
          label="Piotroski F-Score"
          formula="Sum of 9 binary signals (0 or 1 each)"
          description="Profitability (4), Leverage/Liquidity (3), Operating Efficiency (2)"
        >
          <div className="text-4xl font-bold text-ic-text-primary mb-2 cursor-help inline-block">
            {qualityScores.piotroski_f_score ?? '—'}{' '}
            <span className="text-lg text-ic-text-muted">/ 9</span>
          </div>
        </CalculationTooltip>
        <p className="text-sm text-ic-text-muted">
          {qualityScores.piotroski_f_description || 'F-Score measures overall financial health'}
        </p>

        {/* F-Score Visual Bar */}
        {qualityScores.piotroski_f_score !== null &&
          qualityScores.piotroski_f_score !== undefined && (
            <div className="mt-4">
              <div className="flex gap-1">
                {[1, 2, 3, 4, 5, 6, 7, 8, 9].map((n) => (
                  <div
                    key={n}
                    className={cn(
                      'h-3 flex-1 rounded-sm',
                      n <= (qualityScores.piotroski_f_score ?? 0)
                        ? n <= 4
                          ? 'bg-red-400'
                          : n <= 7
                            ? 'bg-yellow-400'
                            : 'bg-green-400'
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
              <span
                className={cn(
                  'px-3 py-1 rounded-full text-sm font-medium',
                  getPEGColor(valuation.peg_interpretation),
                  'bg-ic-bg-tertiary'
                )}
              >
                {valuation.peg_interpretation.toUpperCase()}
              </span>
            )}
          </div>
          <CalculationTooltip
            label="PEG Ratio"
            formula="P/E Ratio / EPS Growth Rate"
            description="Values stock relative to earnings growth"
          >
            <div className="text-4xl font-bold text-ic-text-primary mb-2 cursor-help inline-block">
              {valuation.peg_ratio?.toFixed(2) ?? '—'}
            </div>
          </CalculationTooltip>
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

function AnalystRatingsSection({
  analystRatings,
  currentPrice,
}: {
  analystRatings: AnalystRatings;
  currentPrice: number;
}) {
  // Calculate total ratings
  const totalRatings =
    (analystRatings.analyst_rating_strong_buy ?? 0) +
    (analystRatings.analyst_rating_buy ?? 0) +
    (analystRatings.analyst_rating_hold ?? 0) +
    (analystRatings.analyst_rating_sell ?? 0) +
    (analystRatings.analyst_rating_strong_sell ?? 0);

  // Calculate upside/downside percentages
  const consensusUpside = calculateTargetUpside(currentPrice, analystRatings.target_consensus);
  const highUpside = calculateTargetUpside(currentPrice, analystRatings.target_high);
  const lowUpside = calculateTargetUpside(currentPrice, analystRatings.target_low);

  // Check if we have any data
  const hasRatings = totalRatings > 0;
  const hasPriceTargets = analystRatings.target_consensus !== null;

  if (!hasRatings && !hasPriceTargets) {
    return (
      <div className="p-8 text-center">
        <div className="text-ic-text-muted">
          <p className="text-lg font-medium">No Analyst Ratings Available</p>
          <p className="text-sm mt-2">Analyst coverage data is not available for this stock.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Analyst Consensus */}
      {hasRatings && (
        <div className="bg-ic-bg-secondary rounded-lg p-6">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h4 className="text-lg font-semibold text-ic-text-primary">Analyst Consensus</h4>
              <p className="text-sm text-ic-text-muted">Based on {totalRatings} analyst ratings</p>
            </div>
            {analystRatings.analyst_consensus && (
              <span
                className={cn(
                  'px-4 py-2 rounded-full text-sm font-semibold',
                  getConsensusColor(analystRatings.analyst_consensus),
                  getConsensusBgColor(analystRatings.analyst_consensus)
                )}
              >
                {analystRatings.analyst_consensus.toUpperCase()}
              </span>
            )}
          </div>

          {/* Rating Distribution Bar */}
          <div className="mt-4">
            <div className="flex h-8 rounded-lg overflow-hidden">
              {analystRatings.analyst_rating_strong_buy !== null &&
                analystRatings.analyst_rating_strong_buy > 0 && (
                  <div
                    className="bg-green-600 flex items-center justify-center text-white text-xs font-medium"
                    style={{
                      width: `${(analystRatings.analyst_rating_strong_buy / totalRatings) * 100}%`,
                    }}
                    title={`Strong Buy: ${analystRatings.analyst_rating_strong_buy}`}
                  >
                    {analystRatings.analyst_rating_strong_buy}
                  </div>
                )}
              {analystRatings.analyst_rating_buy !== null &&
                analystRatings.analyst_rating_buy > 0 && (
                  <div
                    className="bg-green-400 flex items-center justify-center text-white text-xs font-medium"
                    style={{
                      width: `${(analystRatings.analyst_rating_buy / totalRatings) * 100}%`,
                    }}
                    title={`Buy: ${analystRatings.analyst_rating_buy}`}
                  >
                    {analystRatings.analyst_rating_buy}
                  </div>
                )}
              {analystRatings.analyst_rating_hold !== null &&
                analystRatings.analyst_rating_hold > 0 && (
                  <div
                    className="bg-yellow-400 flex items-center justify-center text-gray-800 text-xs font-medium"
                    style={{
                      width: `${(analystRatings.analyst_rating_hold / totalRatings) * 100}%`,
                    }}
                    title={`Hold: ${analystRatings.analyst_rating_hold}`}
                  >
                    {analystRatings.analyst_rating_hold}
                  </div>
                )}
              {analystRatings.analyst_rating_sell !== null &&
                analystRatings.analyst_rating_sell > 0 && (
                  <div
                    className="bg-red-400 flex items-center justify-center text-white text-xs font-medium"
                    style={{
                      width: `${(analystRatings.analyst_rating_sell / totalRatings) * 100}%`,
                    }}
                    title={`Sell: ${analystRatings.analyst_rating_sell}`}
                  >
                    {analystRatings.analyst_rating_sell}
                  </div>
                )}
              {analystRatings.analyst_rating_strong_sell !== null &&
                analystRatings.analyst_rating_strong_sell > 0 && (
                  <div
                    className="bg-red-600 flex items-center justify-center text-white text-xs font-medium"
                    style={{
                      width: `${(analystRatings.analyst_rating_strong_sell / totalRatings) * 100}%`,
                    }}
                    title={`Strong Sell: ${analystRatings.analyst_rating_strong_sell}`}
                  >
                    {analystRatings.analyst_rating_strong_sell}
                  </div>
                )}
            </div>
            <div className="flex justify-between text-xs text-ic-text-dim mt-2">
              <span>Strong Buy</span>
              <span>Buy</span>
              <span>Hold</span>
              <span>Sell</span>
              <span>Strong Sell</span>
            </div>
          </div>

          {/* Rating Breakdown Grid */}
          <div className="grid grid-cols-5 gap-2 mt-6">
            <div className="text-center p-2 bg-green-100 rounded">
              <div className="text-lg font-bold text-green-700">
                {analystRatings.analyst_rating_strong_buy ?? 0}
              </div>
              <div className="text-xs text-green-600">Strong Buy</div>
            </div>
            <div className="text-center p-2 bg-green-50 rounded">
              <div className="text-lg font-bold text-green-600">
                {analystRatings.analyst_rating_buy ?? 0}
              </div>
              <div className="text-xs text-green-500">Buy</div>
            </div>
            <div className="text-center p-2 bg-yellow-50 rounded">
              <div className="text-lg font-bold text-yellow-600">
                {analystRatings.analyst_rating_hold ?? 0}
              </div>
              <div className="text-xs text-yellow-500">Hold</div>
            </div>
            <div className="text-center p-2 bg-red-50 rounded">
              <div className="text-lg font-bold text-red-500">
                {analystRatings.analyst_rating_sell ?? 0}
              </div>
              <div className="text-xs text-red-400">Sell</div>
            </div>
            <div className="text-center p-2 bg-red-100 rounded">
              <div className="text-lg font-bold text-red-700">
                {analystRatings.analyst_rating_strong_sell ?? 0}
              </div>
              <div className="text-xs text-red-600">Strong Sell</div>
            </div>
          </div>
        </div>
      )}

      {/* Price Targets */}
      {hasPriceTargets && (
        <div className="bg-ic-bg-secondary rounded-lg p-6">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h4 className="text-lg font-semibold text-ic-text-primary">Price Targets</h4>
              <p className="text-sm text-ic-text-muted">
                Current Price: ${currentPrice.toFixed(2)}
              </p>
            </div>
          </div>

          {/* Price Target Range Visual */}
          <div className="mt-4 mb-6">
            <div className="relative h-12 bg-ic-bg-tertiary rounded-lg">
              {/* Range bar */}
              {analystRatings.target_low !== null && analystRatings.target_high !== null && (
                <>
                  {/* Calculate positions */}
                  {(() => {
                    const min = Math.min(analystRatings.target_low, currentPrice * 0.8);
                    const max = Math.max(analystRatings.target_high, currentPrice * 1.2);
                    const range = max - min;
                    const lowPos = ((analystRatings.target_low - min) / range) * 100;
                    const highPos = ((analystRatings.target_high - min) / range) * 100;
                    const currentPos = ((currentPrice - min) / range) * 100;
                    const consensusPos = analystRatings.target_consensus
                      ? ((analystRatings.target_consensus - min) / range) * 100
                      : null;

                    return (
                      <>
                        {/* Target range bar */}
                        <div
                          className="absolute h-4 top-4 bg-blue-200 rounded"
                          style={{
                            left: `${lowPos}%`,
                            width: `${highPos - lowPos}%`,
                          }}
                        />
                        {/* Low target marker */}
                        <div
                          className="absolute w-1 h-8 top-2 bg-blue-400 rounded"
                          style={{ left: `${lowPos}%` }}
                          title={`Low: $${analystRatings.target_low?.toFixed(2)}`}
                        />
                        {/* High target marker */}
                        <div
                          className="absolute w-1 h-8 top-2 bg-blue-400 rounded"
                          style={{ left: `${highPos}%` }}
                          title={`High: $${analystRatings.target_high?.toFixed(2)}`}
                        />
                        {/* Consensus marker */}
                        {consensusPos !== null && (
                          <div
                            className="absolute w-2 h-10 top-1 bg-blue-600 rounded"
                            style={{ left: `calc(${consensusPos}% - 4px)` }}
                            title={`Consensus: $${analystRatings.target_consensus?.toFixed(2)}`}
                          />
                        )}
                        {/* Current price marker */}
                        <div
                          className="absolute w-3 h-12 top-0 bg-ic-text-primary rounded"
                          style={{ left: `calc(${currentPos}% - 6px)` }}
                          title={`Current: $${currentPrice.toFixed(2)}`}
                        />
                      </>
                    );
                  })()}
                </>
              )}
            </div>
            <div className="flex justify-between text-xs text-ic-text-dim mt-2">
              <span>${analystRatings.target_low?.toFixed(2) ?? '—'} (Low)</span>
              <span className="font-medium">
                ${analystRatings.target_consensus?.toFixed(2) ?? '—'} (Consensus)
              </span>
              <span>${analystRatings.target_high?.toFixed(2) ?? '—'} (High)</span>
            </div>
          </div>

          {/* Price Target Cards */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <MetricCard
              label="Consensus Target"
              value={analystRatings.target_consensus}
              format="currency"
              decimals={2}
              tooltip="Average analyst price target"
              calculationTooltip={{
                formula: 'Average of Analyst Price Targets',
                description: 'Mean price target from all analysts',
              }}
            />
            <MetricCard
              label="Upside/Downside"
              value={
                consensusUpside !== null
                  ? `${consensusUpside >= 0 ? '+' : ''}${consensusUpside.toFixed(1)}%`
                  : null
              }
              format="text"
              tooltip="Potential return to consensus target"
              calculationTooltip={{
                formula: '(Target - Current Price) / Current Price × 100',
                description: 'Potential gain/loss to reach consensus',
              }}
            />
            <MetricCard
              label="Target High"
              value={analystRatings.target_high}
              format="currency"
              decimals={2}
              tooltip="Highest analyst price target"
              calculationTooltip={{
                formula: 'Max(Analyst Price Targets)',
                description: `Upside: ${highUpside !== null ? `${highUpside >= 0 ? '+' : ''}${highUpside.toFixed(1)}%` : '—'}`,
              }}
            />
            <MetricCard
              label="Target Low"
              value={analystRatings.target_low}
              format="currency"
              decimals={2}
              tooltip="Lowest analyst price target"
              calculationTooltip={{
                formula: 'Min(Analyst Price Targets)',
                description: `Downside: ${lowUpside !== null ? `${lowUpside >= 0 ? '+' : ''}${lowUpside.toFixed(1)}%` : '—'}`,
              }}
            />
          </div>

          {/* Median Target */}
          {analystRatings.target_median !== null && (
            <div className="mt-4 grid grid-cols-2 md:grid-cols-4 gap-4">
              <MetricCard
                label="Median Target"
                value={analystRatings.target_median}
                format="currency"
                decimals={2}
                tooltip="Median analyst price target"
                calculationTooltip={{
                  formula: 'Median(Analyst Price Targets)',
                  description: 'Middle value of all price targets',
                }}
              />
            </div>
          )}
        </div>
      )}

      {/* Disclaimer */}
      <div className="text-xs text-ic-text-dim p-4 bg-ic-bg-tertiary rounded-lg">
        <p>
          <strong>Note:</strong> Analyst ratings and price targets are sourced from Financial
          Modeling Prep (FMP). These represent consensus views from multiple analysts and should not
          be considered investment advice. Past analyst performance does not guarantee future
          accuracy.
        </p>
      </div>
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
  calculationTooltip?: {
    formula: string;
    description?: string;
  };
  colorByValue?: boolean;
  interpretation?: string | null;
  interpretationColorFn?: (interp: string | null) => string;
  /** Optional React node rendered inline next to the label (e.g. red flag dot). */
  flagIndicator?: React.ReactNode;
}

function MetricCard({
  label,
  value,
  format,
  decimals = 2,
  tooltip,
  nullTooltip,
  calculationTooltip,
  colorByValue = false,
  interpretation,
  interpretationColorFn,
  flagIndicator,
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
  const displayTooltip =
    (value === null || value === undefined) && nullTooltip ? nullTooltip : tooltip;

  const valueContent = (
    <div className={cn('text-xl font-semibold', getValueColor())}>{formatValue()}</div>
  );

  return (
    <div
      className="bg-ic-bg-secondary rounded-lg p-4"
      title={!calculationTooltip ? displayTooltip : undefined}
    >
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm text-ic-text-muted inline-flex items-center">
          {label}
          {flagIndicator}
        </span>
        {interpretation && interpretationColorFn && (
          <span
            className={cn(
              'text-xs px-2 py-0.5 rounded-full',
              interpretationColorFn(interpretation),
              'bg-opacity-20'
            )}
          >
            {interpretation}
          </span>
        )}
      </div>
      {calculationTooltip ? (
        <CalculationTooltip
          label={label}
          formula={calculationTooltip.formula}
          description={calculationTooltip.description || displayTooltip}
        >
          <div className="cursor-help inline-block">{valueContent}</div>
        </CalculationTooltip>
      ) : (
        valueContent
      )}
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
