'use client';

import { useState } from 'react';
import { ICScoreData, getFactorDetails, getScoreColor, getLetterGrade, getGradeTier, FactorMetadata } from '@/lib/api/ic-score';
import { ChevronDown, ChevronUp, Info, TrendingUp, TrendingDown, Minus } from 'lucide-react';

interface FactorBreakdownProps {
  icScore: ICScoreData;
}

// Factor metric configurations for displaying calculation details
const factorMetricConfig: Record<string, {
  label: string;
  metrics: Array<{
    key: string;
    label: string;
    format: (v: number) => string;
    scoreKey?: string;
    benchmark?: string;
  }>;
}> = {
  value: {
    label: 'Valuation Metrics',
    metrics: [
      { key: 'pe_ratio', label: 'P/E Ratio', format: (v) => v.toFixed(1), scoreKey: 'pe_score', benchmark: 'Market avg: ~20' },
      { key: 'pb_ratio', label: 'P/B Ratio', format: (v) => v.toFixed(2), scoreKey: 'pb_score', benchmark: 'Market avg: ~3' },
      { key: 'ps_ratio', label: 'P/S Ratio', format: (v) => v.toFixed(2), scoreKey: 'ps_score', benchmark: 'Market avg: ~2' },
    ],
  },
  growth: {
    label: 'Growth Metrics',
    metrics: [
      { key: 'revenue_growth', label: 'Revenue Growth (YoY)', format: (v) => `${v >= 0 ? '+' : ''}${v.toFixed(1)}%`, scoreKey: 'revenue_score' },
      { key: 'eps_growth', label: 'EPS Growth (YoY)', format: (v) => `${v >= 0 ? '+' : ''}${v.toFixed(1)}%`, scoreKey: 'eps_score' },
    ],
  },
  profitability: {
    label: 'Profitability Metrics',
    metrics: [
      { key: 'net_margin', label: 'Net Profit Margin', format: (v) => `${v.toFixed(1)}%`, scoreKey: 'margin_score' },
      { key: 'roe', label: 'Return on Equity (ROE)', format: (v) => `${v.toFixed(1)}%`, scoreKey: 'roe_score' },
      { key: 'roa', label: 'Return on Assets (ROA)', format: (v) => `${v.toFixed(1)}%`, scoreKey: 'roa_score' },
    ],
  },
  financial_health: {
    label: 'Financial Health Metrics',
    metrics: [
      { key: 'debt_to_equity', label: 'Debt-to-Equity', format: (v) => v.toFixed(2), scoreKey: 'de_score', benchmark: 'Lower is better' },
      { key: 'current_ratio', label: 'Current Ratio', format: (v) => v.toFixed(2), scoreKey: 'cr_score', benchmark: 'Optimal: ~2.0' },
    ],
  },
  momentum: {
    label: 'Price Momentum',
    metrics: [
      { key: 'return_1m', label: '1-Month Return', format: (v) => `${v >= 0 ? '+' : ''}${v.toFixed(1)}%` },
      { key: 'return_3m', label: '3-Month Return', format: (v) => `${v >= 0 ? '+' : ''}${v.toFixed(1)}%` },
      { key: 'return_6m', label: '6-Month Return', format: (v) => `${v >= 0 ? '+' : ''}${v.toFixed(1)}%` },
      { key: 'return_12m', label: '12-Month Return', format: (v) => `${v >= 0 ? '+' : ''}${v.toFixed(1)}%` },
    ],
  },
  technical: {
    label: 'Technical Indicators',
    metrics: [
      { key: 'rsi', label: 'RSI (14-day)', format: (v) => v.toFixed(0), benchmark: '30-70 neutral' },
      { key: 'macd_histogram', label: 'MACD Histogram', format: (v) => v.toFixed(3) },
      { key: 'price_vs_sma50', label: 'Price vs SMA-50', format: (v) => `${v >= 0 ? '+' : ''}${v.toFixed(1)}%` },
      { key: 'price_vs_sma200', label: 'Price vs SMA-200', format: (v) => `${v >= 0 ? '+' : ''}${v.toFixed(1)}%` },
    ],
  },
  analyst_consensus: {
    label: 'Analyst Ratings',
    metrics: [
      { key: 'buy_count', label: 'Buy Ratings', format: (v) => v.toFixed(0) },
      { key: 'hold_count', label: 'Hold Ratings', format: (v) => v.toFixed(0) },
      { key: 'sell_count', label: 'Sell Ratings', format: (v) => v.toFixed(0) },
      { key: 'upside_pct', label: 'Price Target Upside', format: (v) => `${v >= 0 ? '+' : ''}${v.toFixed(1)}%` },
    ],
  },
  insider_activity: {
    label: 'Insider Transactions (90 days)',
    metrics: [
      { key: 'buy_transactions', label: 'Buy Transactions', format: (v) => v.toFixed(0) },
      { key: 'sell_transactions', label: 'Sell Transactions', format: (v) => v.toFixed(0) },
      { key: 'net_shares', label: 'Net Shares', format: (v) => v >= 0 ? `+${v.toLocaleString()}` : v.toLocaleString() },
    ],
  },
  institutional: {
    label: 'Institutional Ownership',
    metrics: [
      { key: 'institution_count', label: 'Institutions', format: (v) => v.toLocaleString() },
      { key: 'holdings_change_pct', label: 'Holdings Change (QoQ)', format: (v) => `${v >= 0 ? '+' : ''}${v.toFixed(1)}%` },
    ],
  },
  news_sentiment: {
    label: 'News Analysis',
    metrics: [
      { key: 'sentiment_avg', label: 'Avg Sentiment', format: (v) => v.toFixed(2), benchmark: '-1 to +1 scale' },
      { key: 'article_count', label: 'Articles Analyzed', format: (v) => v.toFixed(0) },
    ],
  },
};

export default function FactorBreakdown({ icScore }: FactorBreakdownProps) {
  const factors = getFactorDetails(icScore);
  const metadata = icScore.calculation_metadata;
  const weightsUsed = metadata?.weights_used || {};

  // Calculate total weight of available factors for contribution calculation
  const totalWeight = factors
    .filter(f => f.available)
    .reduce((sum, f) => sum + (weightsUsed[f.name] || f.weight), 0);

  return (
    <div className="bg-ic-surface rounded-lg border border-ic-border">
      <div className="px-6 py-4 border-b border-ic-border">
        <h3 className="text-lg font-semibold text-ic-text-primary">Factor Breakdown</h3>
        <p className="text-sm text-ic-text-muted mt-1">
          Detailed analysis of all 10 scoring factors
        </p>
      </div>

      <div className="p-6">
        <div className="space-y-4">
          {factors.map((factor) => (
            <FactorCard
              key={factor.name}
              factor={factor}
              metadata={metadata?.factors?.[factor.name]}
              weight={weightsUsed[factor.name] || factor.weight}
              totalWeight={totalWeight}
              overallScore={icScore.overall_score}
            />
          ))}
        </div>

        {/* Legend */}
        <div className="mt-6 pt-6 border-t border-ic-border">
          <div className="flex items-start gap-2 text-xs text-ic-text-muted">
            <Info className="w-4 h-4 mt-0.5 flex-shrink-0" />
            <p>
              Click on any factor to see the underlying data and how it contributes to the overall score.
              Grades are aligned with ratings: A = Strong Buy, B = Buy, C = Hold, D = Underperform, F = Sell.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

interface FactorCardProps {
  factor: {
    name: string;
    display_name: string;
    score: number | null;
    weight: number;
    grade: string;
    available: boolean;
    description: string;
  };
  metadata?: FactorMetadata;
  weight: number;
  totalWeight: number;
  overallScore: number;
}

function FactorCard({ factor, metadata, weight, totalWeight, overallScore }: FactorCardProps) {
  const [expanded, setExpanded] = useState(false);
  const { display_name, score, available, description, name } = factor;

  if (!available || score === null) {
    return (
      <div className="p-4 rounded-lg bg-ic-bg-secondary border border-ic-border opacity-60">
        <div className="flex items-center justify-between">
          <div className="flex-1">
            <div className="flex items-center gap-2">
              <h4 className="font-medium text-ic-text-muted">{display_name}</h4>
              <span className="text-xs text-ic-text-dim bg-ic-bg-secondary px-2 py-0.5 rounded">
                Weight: {weight}%
              </span>
            </div>
            <p className="text-xs text-ic-text-dim mt-1">{description}</p>
          </div>
          <div className="ml-4">
            <span className="text-sm text-ic-text-dim font-medium">Data Not Available</span>
            <div className="text-xs text-ic-text-dim mt-1">Coming soon</div>
          </div>
        </div>
      </div>
    );
  }

  // Calculate contribution to overall score
  const normalizedWeight = weight / totalWeight;
  const contribution = (score * normalizedWeight);
  const contributionPct = (contribution / overallScore) * 100;

  // Get grade tier info
  const gradeTier = getGradeTier(score);

  // Determine colors based on score
  const getProgressColor = () => {
    if (score >= 80) return 'bg-green-500';
    if (score >= 65) return 'bg-green-400';
    if (score >= 50) return 'bg-yellow-500';
    if (score >= 35) return 'bg-orange-500';
    return 'bg-red-500';
  };

  const getBgColor = () => {
    if (score >= 80) return 'bg-ic-positive-bg border-green-200';
    if (score >= 65) return 'bg-ic-positive-bg border-green-100';
    if (score >= 50) return 'bg-ic-warning-bg border-yellow-200';
    if (score >= 35) return 'bg-orange-50 border-orange-200';
    return 'bg-ic-negative-bg border-red-200';
  };

  const progressColor = getProgressColor();
  const bgColor = getBgColor();
  const textColor = getScoreColor(score);

  const config = factorMetricConfig[name];

  return (
    <div className={`rounded-lg border ${bgColor} overflow-hidden`}>
      {/* Main card - clickable */}
      <div
        className="p-4 cursor-pointer hover:bg-black/5 transition-colors"
        onClick={() => setExpanded(!expanded)}
      >
        <div className="flex items-start justify-between mb-3">
          <div className="flex-1">
            <div className="flex items-center gap-2">
              <h4 className="font-medium text-ic-text-primary">{display_name}</h4>
              <span className="text-xs text-ic-text-muted bg-ic-surface px-2 py-0.5 rounded border border-ic-border">
                Weight: {weight}%
              </span>
              {expanded ? (
                <ChevronUp className="w-4 h-4 text-ic-text-muted" />
              ) : (
                <ChevronDown className="w-4 h-4 text-ic-text-muted" />
              )}
            </div>
            <p className="text-xs text-ic-text-muted mt-1">{description}</p>
          </div>
          <div className="ml-4 text-right flex-shrink-0">
            <div className="flex items-baseline gap-1">
              <span className={`text-2xl font-bold ${textColor}`}>
                {Math.round(score)}
              </span>
              <span className="text-sm text-ic-text-dim">/100</span>
            </div>
            <div className="text-sm font-medium text-ic-text-secondary mt-0.5">
              {gradeTier.letter} ({gradeTier.rating})
            </div>
          </div>
        </div>

        {/* Progress Bar */}
        <div className="mt-2">
          <div className="h-2 bg-ic-bg-secondary rounded-full overflow-hidden">
            <div
              className={`h-full ${progressColor} transition-all duration-500`}
              style={{ width: `${score}%` }}
            />
          </div>
        </div>

        {/* Contribution indicator */}
        <div className="mt-2 flex items-center justify-between text-xs text-ic-text-muted">
          <div className="flex items-center gap-1">
            <Minus className="w-3 h-3" />
            <span>Stable</span>
          </div>
          <div>
            Contributes <span className="font-medium text-ic-text-secondary">{contribution.toFixed(1)} pts</span> ({contributionPct.toFixed(0)}% of total)
          </div>
        </div>
      </div>

      {/* Expanded details */}
      {expanded && config && (
        <div className="px-4 pb-4 pt-2 border-t border-ic-border/50 bg-ic-surface/50">
          <div className="text-xs font-medium text-ic-text-muted uppercase mb-3">
            {config.label}
          </div>

          {metadata ? (
            <div className="space-y-2">
              {config.metrics.map((metric) => {
                const value = metadata[metric.key];
                if (value === undefined || value === null) return null;

                const metricScore = metric.scoreKey ? metadata[metric.scoreKey] : undefined;

                return (
                  <div key={metric.key} className="flex items-center justify-between py-1.5 border-b border-ic-border/30 last:border-0">
                    <div className="flex-1">
                      <span className="text-sm text-ic-text-secondary">{metric.label}</span>
                      {metric.benchmark && (
                        <span className="text-xs text-ic-text-dim ml-2">({metric.benchmark})</span>
                      )}
                    </div>
                    <div className="flex items-center gap-3">
                      <span className="text-sm font-medium text-ic-text-primary">
                        {metric.format(value as number)}
                      </span>
                      {metricScore !== undefined && (
                        <span className={`text-xs px-1.5 py-0.5 rounded ${getScoreColor(metricScore as number)} bg-ic-bg-secondary`}>
                          → {Math.round(metricScore as number)}
                        </span>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          ) : (
            <div className="text-sm text-ic-text-muted italic">
              Detailed calculation data not available for this factor.
            </div>
          )}

          {/* Score explanation */}
          <div className="mt-3 p-2 bg-ic-bg-secondary rounded text-xs text-ic-text-muted">
            <strong>How this score works:</strong> Individual metrics are scored 0-100, then averaged
            to produce the factor score. A score of 50 represents the market average (neutral).
          </div>
        </div>
      )}
    </div>
  );
}
