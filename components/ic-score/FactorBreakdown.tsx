'use client';

import { ICScoreData, getFactorDetails, getScoreColor, getLetterGrade } from '@/lib/api/ic-score';
import { TrendingUp, TrendingDown, Minus, Info } from 'lucide-react';

interface FactorBreakdownProps {
  icScore: ICScoreData;
}

export default function FactorBreakdown({ icScore }: FactorBreakdownProps) {
  const factors = getFactorDetails(icScore);

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
            <FactorCard key={factor.name} factor={factor} />
          ))}
        </div>

        {/* Legend */}
        <div className="mt-6 pt-6 border-t border-ic-border">
          <div className="flex items-start gap-2 text-xs text-ic-text-muted">
            <Info className="w-4 h-4 mt-0.5 flex-shrink-0" />
            <p>
              Scores are calculated using sector-relative percentile rankings and weighted by
              importance. Missing factors are excluded from the overall calculation and weights are
              redistributed proportionally.
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
}

function FactorCard({ factor }: FactorCardProps) {
  const { display_name, score, weight, grade, available, description } = factor;

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

  // Determine color based on score
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

  return (
    <div className={`p-4 rounded-lg border ${bgColor}`}>
      <div className="flex items-start justify-between mb-3">
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <h4 className="font-medium text-ic-text-primary">{display_name}</h4>
            <span className="text-xs text-ic-text-muted bg-ic-surface px-2 py-0.5 rounded border border-ic-border">
              Weight: {weight}%
            </span>
          </div>
          <p className="text-xs text-ic-text-muted mt-1">{description}</p>
        </div>
        <div className="ml-4 text-right flex-shrink-0">
          <div className="flex items-baseline gap-1">
            <span className={`text-2xl font-bold ${textColor}`}>{Math.round(score)}</span>
            <span className="text-sm text-ic-text-dim">/100</span>
          </div>
          <div className="text-sm font-medium text-ic-text-secondary mt-0.5">Grade: {grade}</div>
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

      {/* Trend Indicator - placeholder for future feature */}
      <div className="mt-2 flex items-center gap-1 text-xs text-ic-text-muted">
        <Minus className="w-3 h-3" />
        <span>Stable</span>
      </div>
    </div>
  );
}
