'use client';

import React from 'react';
import GaugeChart from 'react-gauge-chart';
import { getICScoreColor, getICScoreRating } from '@/lib/types/ic-score';
import type { ICScoreRating } from '@/lib/types/ic-score';

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
  const config = SIZE_CONFIG[size];
  const rating = getICScoreRating(score);
  const normalizedScore = Math.max(0, Math.min(100, score)) / 100; // 0-1 scale for gauge

  // Color segments for the gauge (5 bands)
  const colors = ['#ef4444', '#f97316', '#eab308', '#84cc16', '#10b981'];

  return (
    <div className="flex flex-col items-center">
      <div style={{ width: config.width }}>
        <GaugeChart
          id="ic-score-gauge"
          nrOfLevels={5}
          colors={colors}
          arcWidth={0.3}
          percent={normalizedScore}
          textColor="#1f2937"
          needleColor="#6b7280"
          needleBaseColor="#9ca3af"
          hideText={!showLabel}
          formatTextValue={(value) => Math.round(parseFloat(value)).toString()}
          animDelay={0}
          animateDuration={1500}
        />
      </div>

      {/* Score label below gauge */}
      {showLabel && (
        <div className="mt-2 text-center">
          <div
            className="font-bold text-gray-900"
            style={{ fontSize: config.fontSize }}
          >
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
        return 'bg-green-100 text-green-800 border-green-300';
      case 'Buy':
        return 'bg-green-50 text-green-700 border-green-200';
      case 'Hold':
        return 'bg-yellow-100 text-yellow-800 border-yellow-300';
      case 'Underperform':
        return 'bg-orange-100 text-orange-800 border-orange-300';
      case 'Sell':
        return 'bg-red-100 text-red-800 border-red-300';
      default:
        return 'bg-gray-100 text-gray-800 border-gray-300';
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
