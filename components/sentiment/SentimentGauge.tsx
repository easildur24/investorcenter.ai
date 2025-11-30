'use client';

import { getSentimentScoreColor, formatSentimentScore, SentimentLabel } from '@/lib/types/sentiment';

interface SentimentGaugeProps {
  score: number;        // -1 to +1
  label: SentimentLabel;
  size?: 'sm' | 'md' | 'lg';
  showLabel?: boolean;
}

/**
 * Visual gauge displaying sentiment score from -1 (bearish) to +1 (bullish)
 */
export default function SentimentGauge({
  score,
  label,
  size = 'md',
  showLabel = true,
}: SentimentGaugeProps) {
  const color = getSentimentScoreColor(score);

  // Normalize score from -1 to +1 range to 0 to 100 for gauge position
  const normalizedPosition = ((score + 1) / 2) * 100;

  // Size configurations
  const sizeConfig = {
    sm: {
      height: 'h-2',
      text: 'text-sm',
      labelText: 'text-xs',
      markerSize: 'w-3 h-3',
    },
    md: {
      height: 'h-3',
      text: 'text-base',
      labelText: 'text-sm',
      markerSize: 'w-4 h-4',
    },
    lg: {
      height: 'h-4',
      text: 'text-lg',
      labelText: 'text-base',
      markerSize: 'w-5 h-5',
    },
  };

  const config = sizeConfig[size];

  return (
    <div className="w-full">
      {/* Score display */}
      {showLabel && (
        <div className="flex items-center justify-between mb-2">
          <span className={`font-semibold ${config.text}`} style={{ color }}>
            {formatSentimentScore(score)}
          </span>
          <span className={`${config.labelText} text-gray-500 capitalize`}>
            {label}
          </span>
        </div>
      )}

      {/* Gauge bar */}
      <div className="relative">
        {/* Background gradient bar */}
        <div className={`${config.height} rounded-full overflow-hidden bg-gradient-to-r from-red-500 via-gray-300 to-green-500`}>
        </div>

        {/* Marker */}
        <div
          className="absolute top-1/2 transform -translate-y-1/2 -translate-x-1/2 transition-all duration-300"
          style={{ left: `${normalizedPosition}%` }}
        >
          <div
            className={`${config.markerSize} rounded-full border-2 border-white shadow-lg`}
            style={{ backgroundColor: color }}
          />
        </div>

        {/* Scale labels */}
        <div className="flex justify-between mt-1">
          <span className="text-xs text-red-500">Bearish</span>
          <span className="text-xs text-gray-400">Neutral</span>
          <span className="text-xs text-green-500">Bullish</span>
        </div>
      </div>
    </div>
  );
}

/**
 * Compact inline sentiment indicator
 */
interface SentimentIndicatorProps {
  score: number;
  label: SentimentLabel;
}

export function SentimentIndicator({ score, label }: SentimentIndicatorProps) {
  const color = getSentimentScoreColor(score);

  return (
    <div className="inline-flex items-center gap-1.5">
      <div
        className="w-2 h-2 rounded-full"
        style={{ backgroundColor: color }}
      />
      <span
        className="text-sm font-medium capitalize"
        style={{ color }}
      >
        {label}
      </span>
    </div>
  );
}
