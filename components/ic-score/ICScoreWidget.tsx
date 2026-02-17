'use client';

import { useState } from 'react';
import { ICScoreData, getScoreColor, getStarRating } from '@/lib/api/ic-score';
import {
  ChevronDown,
  ChevronUp,
  Star,
  Info,
  Clock,
  TrendingUp,
  TrendingDown,
  Minus,
} from 'lucide-react';
import { formatRelativeTime } from '@/lib/utils';

interface ScoreChange {
  change: number;
  direction: 'up' | 'down' | 'unchanged';
  period: string;
}

interface ICScoreWidgetProps {
  icScore: ICScoreData;
  showFactors?: boolean;
  scoreChange?: ScoreChange | null;
}

export default function ICScoreWidget({
  icScore,
  showFactors = false,
  scoreChange,
}: ICScoreWidgetProps) {
  const [expanded, setExpanded] = useState(showFactors);
  const score = icScore.overall_score;
  const rating = icScore.rating;
  const stars = getStarRating(score);

  // Determine color based on score
  const getColorClasses = () => {
    if (score >= 80) {
      return {
        bg: 'bg-green-50',
        border: 'border-green-200',
        text: 'text-green-700',
        progress: 'bg-green-500',
        badge: 'bg-green-100 text-green-800',
      };
    } else if (score >= 65) {
      return {
        bg: 'bg-green-50',
        border: 'border-green-200',
        text: 'text-green-600',
        progress: 'bg-green-400',
        badge: 'bg-green-100 text-green-700',
      };
    } else if (score >= 50) {
      return {
        bg: 'bg-yellow-50',
        border: 'border-yellow-200',
        text: 'text-yellow-700',
        progress: 'bg-yellow-500',
        badge: 'bg-yellow-100 text-yellow-800',
      };
    } else if (score >= 35) {
      return {
        bg: 'bg-orange-50',
        border: 'border-orange-200',
        text: 'text-orange-700',
        progress: 'bg-orange-500',
        badge: 'bg-orange-100 text-orange-800',
      };
    } else {
      return {
        bg: 'bg-red-50',
        border: 'border-red-200',
        text: 'text-red-700',
        progress: 'bg-red-500',
        badge: 'bg-red-100 text-red-800',
      };
    }
  };

  const colors = getColorClasses();

  return (
    <div className={`rounded-lg border ${colors.border} ${colors.bg} overflow-hidden`}>
      {/* Header */}
      <div className="px-6 py-4 bg-ic-surface border-b border-ic-border-subtle">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-semibold text-ic-text-primary">InvestorCenter Score</h3>
            <p className="text-sm text-ic-text-muted">Proprietary 10-factor analysis</p>
          </div>
          <div className="text-right">
            <div className="text-xs text-ic-text-dim mb-1">
              {icScore.factor_count} of 10 factors available
            </div>
            <div className="text-xs text-ic-text-dim">
              {Math.round(icScore.data_completeness)}% data completeness
            </div>
          </div>
        </div>
      </div>

      {/* Score Display */}
      <div className="px-6 py-8 text-center">
        {/* Large Score Number */}
        <div className={`text-6xl font-bold ${colors.text} mb-4`}>
          {Math.round(score)}
          <span className="text-2xl text-ic-text-muted">/100</span>
        </div>

        {/* Progress Bar */}
        <div className="max-w-md mx-auto mb-6">
          <div className="h-3 bg-ic-bg-tertiary rounded-full overflow-hidden">
            <div
              className={`h-full ${colors.progress} transition-all duration-500`}
              style={{ width: `${score}%` }}
            />
          </div>
        </div>

        {/* Star Rating */}
        <div className="flex items-center justify-center gap-1 mb-3">
          {[...Array(5)].map((_, i) => (
            <Star
              key={i}
              className={`w-6 h-6 ${
                i < stars ? `fill-current ${colors.text}` : 'fill-ic-bg-tertiary text-ic-text-dim'
              }`}
            />
          ))}
        </div>

        {/* Rating Label */}
        <div
          className={`inline-flex items-center px-4 py-2 rounded-full ${colors.badge} font-semibold text-sm`}
        >
          {rating}
        </div>

        {/* Score Change Indicator */}
        {scoreChange && scoreChange.change !== 0 && (
          <div
            className={`mt-3 inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm font-medium ${
              scoreChange.direction === 'up'
                ? 'bg-green-100 text-green-700'
                : scoreChange.direction === 'down'
                  ? 'bg-red-100 text-red-700'
                  : 'bg-ic-surface text-ic-text-muted'
            }`}
          >
            {scoreChange.direction === 'up' && <TrendingUp className="w-4 h-4" />}
            {scoreChange.direction === 'down' && <TrendingDown className="w-4 h-4" />}
            {scoreChange.direction === 'unchanged' && <Minus className="w-4 h-4" />}
            <span>
              {scoreChange.direction === 'up' ? '+' : scoreChange.direction === 'down' ? '-' : ''}
              {scoreChange.change} pts this month
            </span>
          </div>
        )}

        {/* Confidence Level */}
        {icScore.confidence_level && (
          <div className="mt-4 text-sm text-ic-text-muted">
            Confidence: <span className="font-medium">{icScore.confidence_level}</span>
          </div>
        )}

        {/* Sector Percentile */}
        {icScore.sector_percentile !== null && (
          <div className="mt-2 text-sm text-ic-text-muted">
            <span className="font-medium">{Math.round(icScore.sector_percentile)}</span>
            <sup>th</sup> percentile in sector
          </div>
        )}

        {/* Last Updated Timestamp */}
        {icScore.calculated_at && (
          <div className="mt-4 flex items-center justify-center gap-1 text-xs text-ic-text-muted">
            <Clock className="w-3 h-3" />
            <span>Updated {formatRelativeTime(icScore.calculated_at)}</span>
          </div>
        )}
      </div>

      {/* Toggle Factor Breakdown Button */}
      <div className="px-6 py-4 bg-ic-surface border-t border-ic-border-subtle">
        <button
          onClick={() => setExpanded(!expanded)}
          className="w-full flex items-center justify-between text-sm font-medium text-ic-text-secondary hover:text-ic-text-primary transition-colors"
        >
          <span>View Factor Breakdown</span>
          {expanded ? <ChevronUp className="w-5 h-5" /> : <ChevronDown className="w-5 h-5" />}
        </button>
      </div>

      {/* Expanded Factor Breakdown - handled by parent */}
    </div>
  );
}
