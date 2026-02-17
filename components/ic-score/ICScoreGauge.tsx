'use client';

import React from 'react';
import GaugeChart from 'react-gauge-chart';
import { getICScoreColor, getICScoreRating } from '@/lib/types/ic-score';
import type { ICScoreRating } from '@/lib/types/ic-score';
import { useTheme } from '@/lib/contexts/ThemeContext';
import { themeColors } from '@/lib/theme';

interface ICScoreGaugeProps {
  score: number;
  size?: 'sm' | 'md' | 'lg';
  showLabel?: boolean;
  showRating?: boolean;
}

const SIZE_CONFIG = {
  sm: {
    width: 200,
    fontSize: '1.5rem',
    ratingSize: 'text-sm',
  },
  md: {
    width: 280,
    fontSize: '2.5rem',
    ratingSize: 'text-base',
  },
  lg: {
    width: 350,
    fontSize: '3rem',
    ratingSize: 'text-lg',
  },
};

/**
 * Circular gauge component for displaying IC Score
 *
 * Displays a 0-100 score with color-coded bands:
 * - Strong Buy (80-100): Green
 * - Buy (65-79): Lime
 * - Hold (50-64): Yellow
 * - Underperform (35-49): Orange
 * - Sell (1-34): Red
 */
export default function ICScoreGauge({
  score,
  size = 'md',
  showLabel = true,
  showRating = true,
}: ICScoreGaugeProps) {
  const { resolvedTheme } = useTheme();
  const config = SIZE_CONFIG[size];
  const rating = getICScoreRating(score);
  const normalizedScore = Math.max(0, Math.min(100, score)) / 100; // 0-1 scale for gauge

  // Theme-aware colors
  const isDark = resolvedTheme === 'dark';
  const textColor = isDark ? themeColors.dark.textPrimary : themeColors.light.textPrimary;
  const needleColor = isDark ? '#9ca3af' : '#6b7280';
  const needleBaseColor = isDark ? '#6b7280' : '#9ca3af';

  // Color segments for the gauge (5 bands) - using theme accent colors
  const colors = [
    themeColors.accent.negative,
    themeColors.accent.orange,
    themeColors.accent.warning,
    '#84cc16', // lime stays the same
    themeColors.accent.positive,
  ];

  return (
    <div className="flex flex-col items-center">
      <div style={{ width: config.width }}>
        <GaugeChart
          key={`ic-score-gauge-${resolvedTheme}`}
          id="ic-score-gauge"
          nrOfLevels={5}
          colors={colors}
          arcWidth={0.3}
          percent={normalizedScore}
          textColor={textColor}
          needleColor={needleColor}
          needleBaseColor={needleBaseColor}
          hideText={!showLabel}
          formatTextValue={(value) => Math.round(parseFloat(value)).toString()}
          animDelay={0}
          animateDuration={1500}
        />
      </div>

      {/* Score label below gauge */}
      {showLabel && (
        <div className="mt-2 text-center">
          <div className="font-bold text-ic-text-primary" style={{ fontSize: config.fontSize }}>
            {Math.round(score)}
          </div>
          <div className="text-sm text-ic-text-dim font-medium">IC Score</div>
        </div>
      )}

      {/* Rating badge */}
      {showRating && (
        <div className="mt-3">
          <RatingBadge rating={rating} size={config.ratingSize} />
        </div>
      )}
    </div>
  );
}

/**
 * Rating badge component
 */
interface RatingBadgeProps {
  rating: ICScoreRating;
  size?: string;
}

function RatingBadge({ rating, size = 'text-base' }: RatingBadgeProps) {
  const getColorClass = (rating: ICScoreRating): string => {
    switch (rating) {
      case 'Strong Buy':
        return 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-400 border-green-300 dark:border-green-700';
      case 'Buy':
        return 'bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-400 border-green-200 dark:border-green-800';
      case 'Hold':
        return 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-400 border-yellow-300 dark:border-yellow-700';
      case 'Underperform':
        return 'bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-400 border-orange-300 dark:border-orange-700';
      case 'Sell':
        return 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-400 border-red-300 dark:border-red-700';
      default:
        return 'bg-ic-surface text-ic-text-primary border-ic-border';
    }
  };

  return (
    <span
      className={`inline-flex items-center px-3 py-1 rounded-full font-semibold border ${size} ${getColorClass(
        rating
      )}`}
    >
      {rating}
    </span>
  );
}
