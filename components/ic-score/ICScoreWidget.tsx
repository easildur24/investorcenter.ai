'use client';

import { useState } from 'react';
import { ICScoreData, getScoreColor, getStarRating, getFactorDetails, getGradeTier } from '@/lib/api/ic-score';
import { ChevronDown, ChevronUp, Star, Info, Clock, TrendingUp, TrendingDown, Minus, BarChart3 } from 'lucide-react';
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

export default function ICScoreWidget({ icScore, showFactors = false, scoreChange }: ICScoreWidgetProps) {
  const [expanded, setExpanded] = useState(showFactors);
  const [showComposition, setShowComposition] = useState(false);
  const score = icScore.overall_score;
  const rating = icScore.rating;
  const stars = getStarRating(score);
  const gradeTier = getGradeTier(score);

  // Get factor details for composition display
  const factors = getFactorDetails(icScore);
  const availableFactors = factors.filter(f => f.available && f.score !== null);
  const weightsUsed = icScore.calculation_metadata?.weights_used || {};
  const totalWeight = availableFactors.reduce((sum, f) => sum + (weightsUsed[f.name] || f.weight), 0);

  // Calculate contributions
  const contributions = availableFactors.map(f => {
    const weight = weightsUsed[f.name] || f.weight;
    const normalizedWeight = weight / totalWeight;
    const contribution = (f.score! * normalizedWeight);
    return {
      ...f,
      actualWeight: weight,
      normalizedWeight,
      contribution,
      contributionPct: (contribution / score) * 100,
    };
  }).sort((a, b) => b.contribution - a.contribution);

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
            <h3 className="text-lg font-semibold text-ic-text-primary">IC Score</h3>
            <p className="text-sm text-ic-text-muted">{icScore.factor_count} of 10 factors available</p>
          </div>
          <div className="flex items-center gap-2">
            <a
              href="#"
              className="text-sm text-ic-blue hover:text-blue-700 hover:underline"
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
                // Open explainer modal - handled by parent
              }}
            >
              How it works
            </a>
          </div>
        </div>
      </div>

      {/* Score Display */}
      <div className="px-6 py-6 text-center">
        {/* Large Score Number with Grade */}
        <div className="mb-4">
          <div className={`text-6xl font-bold ${colors.text}`}>
            {Math.round(score)}
            <span className="text-2xl text-ic-text-muted">/100</span>
          </div>
          <div className="mt-2 text-lg font-semibold text-ic-text-secondary">
            {gradeTier.rating}
          </div>
        </div>

        {/* Progress Bar with markers */}
        <div className="max-w-md mx-auto mb-6">
          <div className="relative">
            <div className="h-3 bg-ic-bg-tertiary rounded-full overflow-hidden">
              <div
                className={`h-full ${colors.progress} transition-all duration-500`}
                style={{ width: `${score}%` }}
              />
            </div>
            {/* Scale markers */}
            <div className="flex justify-between mt-1 text-xs text-ic-text-dim">
              <span>0</span>
              <span className="absolute left-[35%] -translate-x-1/2">35</span>
              <span className="absolute left-[50%] -translate-x-1/2">50</span>
              <span className="absolute left-[65%] -translate-x-1/2">65</span>
              <span className="absolute left-[80%] -translate-x-1/2">80</span>
              <span>100</span>
            </div>
            {/* Rating labels */}
            <div className="flex justify-between mt-0.5 text-[10px] text-ic-text-dim">
              <span>Sell</span>
              <span className="absolute left-[17.5%] -translate-x-1/2">Underperform</span>
              <span className="absolute left-[57.5%] -translate-x-1/2">Hold</span>
              <span className="absolute left-[72.5%] -translate-x-1/2">Buy</span>
              <span>Strong Buy</span>
            </div>
          </div>
        </div>

        {/* Star Rating */}
        <div className="flex items-center justify-center gap-1 mb-3">
          {[...Array(5)].map((_, i) => (
            <Star
              key={i}
              className={`w-5 h-5 ${
                i < stars
                  ? `fill-current ${colors.text}`
                  : 'fill-ic-bg-tertiary text-ic-text-dim'
              }`}
            />
          ))}
        </div>

        {/* Score Change Indicator */}
        {scoreChange && scoreChange.change !== 0 && (
          <div className={`mt-3 inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm font-medium ${
            scoreChange.direction === 'up' ? 'bg-green-100 text-green-700' :
            scoreChange.direction === 'down' ? 'bg-red-100 text-red-700' : 'bg-ic-surface text-ic-text-muted'
          }`}>
            {scoreChange.direction === 'up' && <TrendingUp className="w-4 h-4" />}
            {scoreChange.direction === 'down' && <TrendingDown className="w-4 h-4" />}
            {scoreChange.direction === 'unchanged' && <Minus className="w-4 h-4" />}
            <span>
              {scoreChange.direction === 'up' ? '+' : scoreChange.direction === 'down' ? '-' : ''}
              {scoreChange.change} pts this month
            </span>
          </div>
        )}

        {/* Confidence & Data Info */}
        <div className="mt-4 flex items-center justify-center gap-4 text-sm text-ic-text-muted">
          {icScore.confidence_level && (
            <div>
              Confidence: <span className="font-medium">{icScore.confidence_level}</span>
            </div>
          )}
          {icScore.sector_percentile !== null && (
            <div>
              <span className="font-medium">{Math.round(icScore.sector_percentile)}</span>
              <sup>th</sup> percentile
            </div>
          )}
        </div>

        {/* Last Updated */}
        {icScore.calculated_at && (
          <div className="mt-3 flex items-center justify-center gap-1 text-xs text-ic-text-dim">
            <Clock className="w-3 h-3" />
            <span>Updated {formatRelativeTime(icScore.calculated_at)}</span>
          </div>
        )}
      </div>

      {/* Score Composition Toggle */}
      <div className="px-6 py-3 bg-ic-surface/50 border-t border-ic-border-subtle">
        <button
          onClick={(e) => {
            e.stopPropagation();
            setShowComposition(!showComposition);
          }}
          className="w-full flex items-center justify-between text-sm font-medium text-ic-text-secondary hover:text-ic-text-primary transition-colors"
        >
          <span className="flex items-center gap-2">
            <BarChart3 className="w-4 h-4" />
            Score Composition
          </span>
          {showComposition ? (
            <ChevronUp className="w-4 h-4" />
          ) : (
            <ChevronDown className="w-4 h-4" />
          )}
        </button>
      </div>

      {/* Score Composition Details */}
      {showComposition && (
        <div className="px-6 py-4 bg-ic-surface border-t border-ic-border-subtle">
          <div className="space-y-2">
            {contributions.map((factor) => (
              <div key={factor.name} className="flex items-center gap-3">
                <div className="w-28 text-xs text-ic-text-muted truncate">
                  {factor.display_name}
                </div>
                <div className="flex-1">
                  <div className="h-2 bg-ic-bg-tertiary rounded-full overflow-hidden">
                    <div
                      className={`h-full rounded-full ${
                        factor.score! >= 80 ? 'bg-green-500' :
                        factor.score! >= 65 ? 'bg-green-400' :
                        factor.score! >= 50 ? 'bg-yellow-500' :
                        factor.score! >= 35 ? 'bg-orange-500' : 'bg-red-500'
                      }`}
                      style={{ width: `${factor.score}%` }}
                    />
                  </div>
                </div>
                <div className="w-20 text-right">
                  <span className={`text-xs font-medium ${getScoreColor(factor.score)}`}>
                    {Math.round(factor.score!)}
                  </span>
                  <span className="text-xs text-ic-text-dim ml-1">
                    × {(factor.normalizedWeight * 100).toFixed(0)}%
                  </span>
                </div>
                <div className="w-16 text-right text-xs font-medium text-ic-text-secondary">
                  = {factor.contribution.toFixed(1)} pts
                </div>
              </div>
            ))}

            {/* Missing factors note */}
            {icScore.missing_factors.length > 0 && (
              <div className="mt-3 pt-3 border-t border-ic-border-subtle text-xs text-ic-text-dim">
                Missing: {icScore.missing_factors.map(f => f.replace(/_/g, ' ')).join(', ')}
              </div>
            )}

            {/* Total */}
            <div className="mt-3 pt-3 border-t border-ic-border flex items-center justify-between">
              <span className="text-sm font-medium text-ic-text-secondary">Total Score</span>
              <span className={`text-lg font-bold ${colors.text}`}>
                {Math.round(score)}/100
              </span>
            </div>
          </div>
        </div>
      )}

      {/* Toggle Factor Breakdown Button */}
      <div className="px-6 py-4 bg-ic-surface border-t border-ic-border-subtle">
        <button
          onClick={() => setExpanded(!expanded)}
          className="w-full flex items-center justify-between text-sm font-medium text-ic-text-secondary hover:text-ic-text-primary transition-colors"
        >
          <span>{expanded ? 'Hide' : 'View'} Factor Breakdown</span>
          {expanded ? (
            <ChevronUp className="w-5 h-5" />
          ) : (
            <ChevronDown className="w-5 h-5" />
          )}
        </button>
      </div>
    </div>
  );
}
