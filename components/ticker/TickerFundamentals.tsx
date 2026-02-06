'use client';

import { useState, useEffect } from 'react';
import { safeToFixed, safeParseNumber, formatLargeNumber, formatPercent, formatRelativeTime } from '@/lib/utils';
import { useAuth } from '@/lib/auth/AuthContext';

interface TickerFundamentalsProps {
  symbol: string;
}

interface Fundamentals {
  pe: number | string;
  pb: number | string;
  ps: number | string;
  roe: number | string;
  roa: number | string;
  revenue: number | string;
  netIncome: number | string;
  eps: number | string;
  debtToEquity: number | string;
  currentRatio: number | string;
  grossMargin: number | string;
  operatingMargin: number | string;
  netMargin: number | string;
}

interface KeyMetrics {
  week52High: number | string;
  week52Low: number | string;
  ytdChange: number | string;
  beta: number | string;
  averageVolume: number | string;
  sharesOutstanding: number | string;
  revenueGrowth1Y: number | string;
  earningsGrowth1Y: number | string;
}

// Helper to check if value is valid (not null, undefined, 0, or 'N/A')
const isValidValue = (value: any): boolean => {
  if (value === null || value === undefined || value === 'N/A') return false;
  if (typeof value === 'number' && value === 0) return false;
  if (typeof value === 'string' && (value === '0' || value === '0.00')) return false;
  return true;
};

// Metric type definitions for contextual N/A messages
type MetricType = 'debt' | 'ratio' | 'growth' | 'margin' | 'valuation' | 'market' | 'default';

interface MetricConfig {
  zeroMessage: string;
  nullMessage: string;
  tooltip: string;
}

const metricConfigs: Record<MetricType, MetricConfig> = {
  debt: {
    zeroMessage: 'No debt',
    nullMessage: 'Not reported',
    tooltip: 'This company may have no debt, or the value was not reported in SEC filings',
  },
  ratio: {
    zeroMessage: 'N/A',
    nullMessage: 'Not reported',
    tooltip: 'This ratio could not be calculated from available SEC filings',
  },
  growth: {
    zeroMessage: '0.00%',
    nullMessage: 'Insufficient data',
    tooltip: 'Growth metrics require historical data that may not be available yet',
  },
  margin: {
    zeroMessage: '0.00%',
    nullMessage: 'Not reported',
    tooltip: 'Margin data may not be available in the latest SEC filings',
  },
  valuation: {
    zeroMessage: 'N/A',
    nullMessage: 'Not available',
    tooltip: 'Valuation metrics require both price and fundamental data',
  },
  market: {
    zeroMessage: '0',
    nullMessage: 'Not available',
    tooltip: 'Market data is refreshed during trading hours',
  },
  default: {
    zeroMessage: 'N/A',
    nullMessage: 'Not reported',
    tooltip: 'This data is not currently available',
  },
};

// Contextual N/A display component
// Debug sources interface for admin mode
interface DebugSources {
  pe_ratio?: string;
  pb_ratio?: string;
  ps_ratio?: string;
  gross_margin?: string;
  operating_margin?: string;
  net_margin?: string;
  roe?: string;
  roa?: string;
  current_ratio?: string;
  quick_ratio?: string;
  debt_to_equity?: string;
}

interface MetricValueProps {
  value: number | string;
  metricType?: MetricType;
  formatter?: (val: number | string) => string;
  colorClass?: string;
  dataSource?: string;
  isAdmin?: boolean;
}

function MetricValue({ value, metricType = 'default', formatter, colorClass = 'text-ic-text-primary', dataSource, isAdmin }: MetricValueProps) {
  const config = metricConfigs[metricType];

  // Check if value is null/undefined/N/A
  if (value === null || value === undefined || value === 'N/A' || value === '') {
    return (
      <span
        className="text-ic-text-dim cursor-help"
        title={config.tooltip}
      >
        {config.nullMessage}
      </span>
    );
  }

  // Check if value is zero (special handling for debt)
  const numValue = typeof value === 'string' ? parseFloat(value) : value;
  if (numValue === 0 && metricType === 'debt') {
    return (
      <span
        className="text-ic-positive cursor-help"
        title="Company has no reported debt - this is typically a positive indicator"
      >
        {config.zeroMessage}
      </span>
    );
  }

  // Format and display the value
  const displayValue = formatter ? formatter(value) : String(value);

  // If formatter returned N/A, show contextual message
  if (displayValue === 'N/A') {
    return (
      <span
        className="text-ic-text-dim cursor-help"
        title={config.tooltip}
      >
        {config.nullMessage}
      </span>
    );
  }

  // Show data source badge for admin users
  if (isAdmin && dataSource) {
    const sourceColor = dataSource === 'fmp' ? 'bg-green-500/20 text-green-400' : 'bg-blue-500/20 text-blue-400';
    const sourceLabel = dataSource === 'fmp' ? 'FMP' : 'DB';
    return (
      <span className="inline-flex items-center gap-1.5">
        <span className={`font-medium ${colorClass}`}>{displayValue}</span>
        <span
          className={`text-[10px] px-1 py-0.5 rounded ${sourceColor} cursor-help`}
          title={`Data source: ${dataSource === 'fmp' ? 'Financial Modeling Prep API (real-time)' : 'Database (SEC filings)'}`}
        >
          {sourceLabel}
        </span>
      </span>
    );
  }

  return <span className={`font-medium ${colorClass}`}>{displayValue}</span>;
}

export default function TickerFundamentals({ symbol }: TickerFundamentalsProps) {
  const [fundamentals, setFundamentals] = useState<Fundamentals | null>(null);
  const [keyMetrics, setKeyMetrics] = useState<KeyMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [isCrypto, setIsCrypto] = useState(false);
  const [dataFetchedAt, setDataFetchedAt] = useState<Date | null>(null);
  const [icScoreDataDate, setIcScoreDataDate] = useState<string | null>(null);
  const [debugSources, setDebugSources] = useState<DebugSources | null>(null);

  // Get admin status from auth context
  const { user } = useAuth();
  const isAdmin = user?.is_admin ?? false;

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        console.log(`🔥 Fetching fundamentals for ${symbol}...`);

        // Fetch all data sources in parallel (including manual fundamentals)
        const [tickerResponse, manualFundamentalsResponse, financialsResponse, riskResponse] = await Promise.all([
          fetch(`/api/v1/tickers/${symbol}`),
          fetch(`/api/v1/tickers/${symbol}/manual-fundamentals`).catch(() => null),
          fetch(`/api/v1/stocks/${symbol}/financials`).catch(() => null),
          fetch(`/api/v1/stocks/${symbol}/risk?period=1Y`).catch(() => null)
        ]);

        if (!tickerResponse.ok) {
          throw new Error(`HTTP ${tickerResponse.status}: Failed to fetch ticker data`);
        }

        const tickerResult = await tickerResponse.json();

        // Check if this is a crypto asset
        if (tickerResult.data.summary.stock.isCrypto) {
          console.log('🪙 This is a crypto asset, skipping stock fundamentals');
          setIsCrypto(true);
          setLoading(false);
          return;
        }

        // Extract data from main ticker endpoint (Polygon.io)
        const polygonFundamentals = tickerResult.data.summary.fundamentals || {};
        const polygonKeyMetrics = tickerResult.data.summary.keyMetrics || {};

        // Extract manual fundamentals if available (HIGHEST PRIORITY)
        let manualFundamentals: any = {};
        if (manualFundamentalsResponse?.ok) {
          const manualResult = await manualFundamentalsResponse.json();
          manualFundamentals = manualResult.data || {};
          console.log('✨ Manual Fundamentals:', manualFundamentals);
          // Capture the update date
          if (manualResult.meta?.updated_at) {
            setIcScoreDataDate(manualResult.meta.updated_at);
          }
        }

        // Extract IC Score financial metrics if available
        let icScoreFinancials: any = {};
        if (financialsResponse?.ok) {
          const financialsResult = await financialsResponse.json();
          icScoreFinancials = financialsResult.data || {};
          console.log('📊 IC Score Financials:', icScoreFinancials);
          // Capture the filing date if available (only if no manual data)
          if (!manualFundamentals || Object.keys(manualFundamentals).length === 0) {
            if (icScoreFinancials.period_end_date || icScoreFinancials.filing_date) {
              setIcScoreDataDate(icScoreFinancials.period_end_date || icScoreFinancials.filing_date);
            }
          }
          // Capture debug sources for admin mode
          if (financialsResult.debug?.sources) {
            setDebugSources(financialsResult.debug.sources);
            console.log('🔍 Debug sources:', financialsResult.debug.sources);
          }
        }

        // Extract IC Score risk metrics if available
        let icScoreRisk: any = {};
        if (riskResponse?.ok) {
          const riskResult = await riskResponse.json();
          icScoreRisk = riskResult.data || {};
          console.log('📊 IC Score Risk:', icScoreRisk);
        }

        // Merge data - PRIORITY: Manual > IC Score > Polygon
        const mappedFundamentals: Fundamentals = {
          pe: manualFundamentals?.pe_ratio || polygonFundamentals?.pe || icScoreFinancials?.pe_ratio || 'N/A',
          pb: manualFundamentals?.pb_ratio || polygonFundamentals?.pb || icScoreFinancials?.pb_ratio || 'N/A',
          ps: manualFundamentals?.ps_ratio || polygonFundamentals?.ps || icScoreFinancials?.ps_ratio || 'N/A',
          // Prefer Manual > IC Score > Polygon
          roe: isValidValue(manualFundamentals?.roe) ? manualFundamentals.roe : (isValidValue(icScoreFinancials?.roe) ? icScoreFinancials.roe : (polygonFundamentals?.roe || 'N/A')),
          roa: isValidValue(manualFundamentals?.roa) ? manualFundamentals.roa : (isValidValue(icScoreFinancials?.roa) ? icScoreFinancials.roa : (polygonFundamentals?.roa || 'N/A')),
          revenue: manualFundamentals?.revenue_ttm || polygonFundamentals?.revenue || '0',
          netIncome: manualFundamentals?.net_income_ttm || polygonFundamentals?.netIncome || '0',
          eps: manualFundamentals?.eps_diluted_ttm || polygonFundamentals?.eps || 'N/A',
          debtToEquity: isValidValue(manualFundamentals?.debt_to_equity) ? manualFundamentals.debt_to_equity : (isValidValue(icScoreFinancials?.debt_to_equity) ? icScoreFinancials.debt_to_equity : (polygonKeyMetrics?.debtToEquity || 'N/A')),
          currentRatio: isValidValue(manualFundamentals?.current_ratio) ? manualFundamentals.current_ratio : (isValidValue(icScoreFinancials?.current_ratio) ? icScoreFinancials.current_ratio : (polygonKeyMetrics?.currentRatio || 'N/A')),
          grossMargin: isValidValue(manualFundamentals?.gross_margin) ? manualFundamentals.gross_margin : (isValidValue(icScoreFinancials?.gross_margin) ? icScoreFinancials.gross_margin : (polygonFundamentals?.grossMargin || 'N/A')),
          operatingMargin: isValidValue(manualFundamentals?.operating_margin) ? manualFundamentals.operating_margin : (isValidValue(icScoreFinancials?.operating_margin) ? icScoreFinancials.operating_margin : (polygonFundamentals?.operatingMargin || 'N/A')),
          netMargin: isValidValue(manualFundamentals?.net_margin) ? manualFundamentals.net_margin : (isValidValue(icScoreFinancials?.net_margin) ? icScoreFinancials.net_margin : (polygonFundamentals?.netMargin || 'N/A'))
        };

        const mappedKeyMetrics: KeyMetrics = {
          week52High: polygonKeyMetrics?.week52High || '0',
          week52Low: polygonKeyMetrics?.week52Low || '0',
          ytdChange: polygonKeyMetrics?.ytdChange || '0',
          // Prefer Manual > IC Score Risk > Polygon
          beta: isValidValue(manualFundamentals?.beta) ? manualFundamentals.beta : (isValidValue(icScoreRisk?.beta) ? icScoreRisk.beta : (polygonKeyMetrics?.beta || '1.0')),
          averageVolume: polygonKeyMetrics?.averageVolume || '0',
          // Prefer Manual > IC Score > Polygon
          sharesOutstanding: isValidValue(manualFundamentals?.shares_outstanding) ? manualFundamentals.shares_outstanding : (isValidValue(icScoreFinancials?.shares_outstanding) ? icScoreFinancials.shares_outstanding : (polygonKeyMetrics?.sharesOutstanding || '0')),
          // Prefer Manual > IC Score growth metrics
          revenueGrowth1Y: isValidValue(manualFundamentals?.revenue_growth_yoy) ? manualFundamentals.revenue_growth_yoy : (isValidValue(icScoreFinancials?.revenue_growth_yoy) ? icScoreFinancials.revenue_growth_yoy : (polygonKeyMetrics?.revenueGrowth1Y || '0')),
          earningsGrowth1Y: isValidValue(manualFundamentals?.earnings_growth_yoy) ? manualFundamentals.earnings_growth_yoy : (isValidValue(icScoreFinancials?.earnings_growth_yoy) ? icScoreFinancials.earnings_growth_yoy : (polygonKeyMetrics?.earningsGrowth1Y || '0'))
        };

        console.log('✅ Merged Fundamentals:', mappedFundamentals);
        console.log('✅ Merged Key Metrics:', mappedKeyMetrics);
        setFundamentals(mappedFundamentals);
        setKeyMetrics(mappedKeyMetrics);
        setDataFetchedAt(new Date());
      } catch (error) {
        console.error('❌ Error fetching fundamentals:', error);
      } finally {
        setLoading(false);
        console.log(`🏁 Fundamentals loading complete for ${symbol}`);
      }
    };

    // Add delay for proper mounting
    const timer = setTimeout(fetchData, 100);
    return () => clearTimeout(timer);
  }, [symbol]);

  // Don't render stock fundamentals for crypto assets
  if (isCrypto) {
    return null;
  }

  if (loading) {
    return (
      <div className="p-6">
        <div className="h-6 bg-ic-border rounded w-32 mb-4 animate-pulse"></div>
        <div className="space-y-3">
          {[1, 2, 3, 4, 5, 6].map((i) => (
            <div key={i} className="flex justify-between animate-pulse">
              <div className="h-4 bg-ic-border rounded w-24"></div>
              <div className="h-4 bg-ic-border rounded w-16"></div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (!fundamentals || !keyMetrics) {
    return (
      <div className="p-6">
        <h3 className="text-lg font-semibold text-ic-text-primary mb-4">Key Metrics</h3>
        <p className="text-ic-text-muted">No fundamental data available</p>
      </div>
    );
  }



  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-lg font-semibold text-ic-text-primary">Key Metrics</h3>
        {dataFetchedAt && (
          <span className="text-xs text-ic-text-dim" title={dataFetchedAt.toLocaleString()}>
            Updated {formatRelativeTime(dataFetchedAt)}
          </span>
        )}
      </div>

      {/* Valuation Metrics */}
      <div className="mb-6">
        <h4 className="text-sm font-medium text-ic-text-secondary mb-3 uppercase tracking-wide">Valuation</h4>
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <span className="text-ic-text-muted">P/E Ratio</span>
            <MetricValue value={fundamentals.pe} metricType="valuation" formatter={(v) => safeToFixed(v, 1)} dataSource={debugSources?.pe_ratio} isAdmin={isAdmin} />
          </div>
          <div className="flex justify-between items-center">
            <span className="text-ic-text-muted">Price/Book</span>
            <MetricValue value={fundamentals.pb} metricType="valuation" formatter={(v) => safeToFixed(v, 1)} dataSource={debugSources?.pb_ratio} isAdmin={isAdmin} />
          </div>
          <div className="flex justify-between items-center">
            <span className="text-ic-text-muted">Price/Sales</span>
            <MetricValue value={fundamentals.ps} metricType="valuation" formatter={(v) => safeToFixed(v, 1)} dataSource={debugSources?.ps_ratio} isAdmin={isAdmin} />
          </div>
        </div>
      </div>

      {/* Profitability */}
      <div className="mb-6">
        <h4 className="text-sm font-medium text-ic-text-secondary mb-3 uppercase tracking-wide">Profitability</h4>
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <span className="text-ic-text-muted">ROE</span>
            <MetricValue value={fundamentals.roe} metricType="ratio" formatter={formatPercent} dataSource={debugSources?.roe} isAdmin={isAdmin} />
          </div>
          <div className="flex justify-between items-center">
            <span className="text-ic-text-muted">ROA</span>
            <MetricValue value={fundamentals.roa} metricType="ratio" formatter={formatPercent} dataSource={debugSources?.roa} isAdmin={isAdmin} />
          </div>
          <div className="flex justify-between items-center">
            <span className="text-ic-text-muted">Gross Margin</span>
            <MetricValue value={fundamentals.grossMargin} metricType="margin" formatter={formatPercent} dataSource={debugSources?.gross_margin} isAdmin={isAdmin} />
          </div>
          <div className="flex justify-between items-center">
            <span className="text-ic-text-muted">Net Margin</span>
            <MetricValue value={fundamentals.netMargin} metricType="margin" formatter={formatPercent} dataSource={debugSources?.net_margin} isAdmin={isAdmin} />
          </div>
        </div>
      </div>

      {/* Financial Health */}
      <div className="mb-6">
        <h4 className="text-sm font-medium text-ic-text-secondary mb-3 uppercase tracking-wide">Financial Health</h4>
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <span className="text-ic-text-muted">Debt/Equity</span>
            <MetricValue value={fundamentals.debtToEquity} metricType="debt" formatter={(v) => safeToFixed(v, 1)} dataSource={debugSources?.debt_to_equity} isAdmin={isAdmin} />
          </div>
          <div className="flex justify-between items-center">
            <span className="text-ic-text-muted">Current Ratio</span>
            <MetricValue value={fundamentals.currentRatio} metricType="ratio" formatter={(v) => safeToFixed(v, 1)} dataSource={debugSources?.current_ratio} isAdmin={isAdmin} />
          </div>
        </div>
      </div>

      {/* Performance */}
      <div className="mb-6">
        <h4 className="text-sm font-medium text-ic-text-secondary mb-3 uppercase tracking-wide">Performance</h4>
        <div className="space-y-3">
          <div className="flex justify-between">
            <span className="text-ic-text-muted">YTD Change</span>
            <MetricValue
              value={keyMetrics.ytdChange}
              metricType="growth"
              formatter={formatPercent}
              colorClass={safeParseNumber(keyMetrics.ytdChange) >= 0 ? 'text-ic-positive' : 'text-ic-negative'}
            />
          </div>
          <div className="flex justify-between">
            <span className="text-ic-text-muted">Revenue Growth</span>
            <MetricValue
              value={keyMetrics.revenueGrowth1Y}
              metricType="growth"
              formatter={formatPercent}
              colorClass={safeParseNumber(keyMetrics.revenueGrowth1Y) >= 0 ? 'text-ic-positive' : 'text-ic-negative'}
            />
          </div>
          <div className="flex justify-between">
            <span className="text-ic-text-muted">Earnings Growth</span>
            <MetricValue
              value={keyMetrics.earningsGrowth1Y}
              metricType="growth"
              formatter={formatPercent}
              colorClass={safeParseNumber(keyMetrics.earningsGrowth1Y) >= 0 ? 'text-ic-positive' : 'text-ic-negative'}
            />
          </div>
        </div>
      </div>

      {/* Market Data */}
      <div>
        <h4 className="text-sm font-medium text-ic-text-secondary mb-3 uppercase tracking-wide">Market Data</h4>
        <div className="space-y-3">
          <div className="flex justify-between">
            <span className="text-ic-text-muted">Beta</span>
            <MetricValue value={keyMetrics.beta} metricType="market" formatter={(v) => safeToFixed(v, 2)} />
          </div>
          <div className="flex justify-between">
            <span className="text-ic-text-muted">Avg Volume</span>
            <MetricValue
              value={keyMetrics.averageVolume}
              metricType="market"
              formatter={(v) => `${safeToFixed(safeParseNumber(v) / 1000000, 1)}M`}
            />
          </div>
          <div className="flex justify-between">
            <span className="text-ic-text-muted">Shares Out</span>
            <MetricValue
              value={keyMetrics.sharesOutstanding}
              metricType="market"
              formatter={(v) => `${safeToFixed(safeParseNumber(v) / 1000000000, 1)}B`}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
