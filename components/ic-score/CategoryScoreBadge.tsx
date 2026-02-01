'use client';

import { CategoryScore, getScoreColorClass, getScoreBgClass } from '@/lib/types/ic-score-v2';

interface CategoryScoreBadgeProps {
  name: string;
  category: CategoryScore;
  expanded?: boolean;
  onClick?: () => void;
}

/**
 * CategoryScoreBadge - Displays a category score (Quality, Valuation, Signals)
 *
 * Shows the category name, score, and grade with appropriate color coding.
 * Can be expanded to show contributing factors.
 */
export default function CategoryScoreBadge({
  name,
  category,
  expanded = false,
  onClick,
}: CategoryScoreBadgeProps) {
  const scoreColor = getScoreColorClass(category.score);
  const bgColor = getScoreBgClass(category.score);

  return (
    <div
      className={`rounded-lg p-4 ${bgColor} ${onClick ? 'cursor-pointer hover:opacity-90 transition-opacity' : ''}`}
      onClick={onClick}
    >
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm font-medium text-gray-600">{name}</span>
        <span className={`text-xl font-bold ${scoreColor}`}>{category.grade}</span>
      </div>

      <div className="flex items-baseline gap-1">
        <span className={`text-2xl font-bold ${scoreColor}`}>{category.score}</span>
        <span className="text-sm text-gray-500">/100</span>
      </div>

      {/* Progress bar */}
      <div className="mt-2 h-1.5 bg-white/50 rounded-full overflow-hidden">
        <div
          className={`h-full rounded-full transition-all ${
            category.score >= 80
              ? 'bg-green-500'
              : category.score >= 65
              ? 'bg-green-400'
              : category.score >= 50
              ? 'bg-yellow-400'
              : category.score >= 35
              ? 'bg-orange-400'
              : 'bg-red-400'
          }`}
          style={{ width: `${category.score}%` }}
        />
      </div>

      {/* Expanded: show contributing factors */}
      {expanded && category.factors.length > 0 && (
        <div className="mt-3 pt-3 border-t border-white/30">
          <div className="text-xs text-gray-500">
            Based on: {category.factors.join(', ')}
          </div>
        </div>
      )}
    </div>
  );
}

/**
 * CategoryScoreGrid - Displays all three category scores in a grid
 */
interface CategoryScoreGridProps {
  quality: CategoryScore;
  valuation: CategoryScore;
  signals: CategoryScore;
}

export function CategoryScoreGrid({ quality, valuation, signals }: CategoryScoreGridProps) {
  return (
    <div className="grid grid-cols-3 gap-4">
      <CategoryScoreBadge name="Quality" category={quality} />
      <CategoryScoreBadge name="Valuation" category={valuation} />
      <CategoryScoreBadge name="Signals" category={signals} />
    </div>
  );
}
