'use client';

import type { SentimentBreakdown } from '@/lib/types/sentiment';

interface SentimentBreakdownBarProps {
  breakdown: SentimentBreakdown;  // Values are 0-100 percentages
  showLabels?: boolean;
  showPercentages?: boolean;
  height?: 'sm' | 'md' | 'lg';
}

/**
 * Horizontal stacked bar showing bullish/bearish/neutral distribution
 */
export default function SentimentBreakdownBar({
  breakdown,
  showLabels = true,
  showPercentages = true,
  height = 'md',
}: SentimentBreakdownBarProps) {
  const { bullish, bearish, neutral } = breakdown;

  const heightClass = {
    sm: 'h-2',
    md: 'h-3',
    lg: 'h-4',
  }[height];

  // Ensure percentages add up to 100
  const total = bullish + bearish + neutral;
  const normalizedBullish = total > 0 ? (bullish / total) * 100 : 33.33;
  const normalizedBearish = total > 0 ? (bearish / total) * 100 : 33.33;
  const normalizedNeutral = total > 0 ? (neutral / total) * 100 : 33.34;

  return (
    <div className="w-full">
      {/* Bar */}
      <div className={`${heightClass} rounded-full overflow-hidden flex`}>
        {normalizedBullish > 0 && (
          <div
            className="bg-green-500 transition-all duration-300"
            style={{ width: `${normalizedBullish}%` }}
            title={`Bullish: ${bullish.toFixed(1)}%`}
          />
        )}
        {normalizedNeutral > 0 && (
          <div
            className="bg-gray-400 transition-all duration-300"
            style={{ width: `${normalizedNeutral}%` }}
            title={`Neutral: ${neutral.toFixed(1)}%`}
          />
        )}
        {normalizedBearish > 0 && (
          <div
            className="bg-red-500 transition-all duration-300"
            style={{ width: `${normalizedBearish}%` }}
            title={`Bearish: ${bearish.toFixed(1)}%`}
          />
        )}
      </div>

      {/* Labels */}
      {showLabels && (
        <div className="flex justify-between mt-2 text-xs">
          <div className="flex items-center gap-1">
            <div className="w-2 h-2 rounded-full bg-green-500" />
            <span className="text-gray-600">
              Bullish
              {showPercentages && (
                <span className="font-medium text-green-600 ml-1">
                  {bullish.toFixed(0)}%
                </span>
              )}
            </span>
          </div>

          <div className="flex items-center gap-1">
            <div className="w-2 h-2 rounded-full bg-gray-400" />
            <span className="text-gray-600">
              Neutral
              {showPercentages && (
                <span className="font-medium text-gray-500 ml-1">
                  {neutral.toFixed(0)}%
                </span>
              )}
            </span>
          </div>

          <div className="flex items-center gap-1">
            <div className="w-2 h-2 rounded-full bg-red-500" />
            <span className="text-gray-600">
              Bearish
              {showPercentages && (
                <span className="font-medium text-red-600 ml-1">
                  {bearish.toFixed(0)}%
                </span>
              )}
            </span>
          </div>
        </div>
      )}
    </div>
  );
}

/**
 * Compact version of the breakdown bar for tight spaces
 */
interface CompactBreakdownBarProps {
  breakdown: SentimentBreakdown;
}

export function CompactBreakdownBar({ breakdown }: CompactBreakdownBarProps) {
  const { bullish, bearish, neutral } = breakdown;
  const total = bullish + bearish + neutral;

  if (total === 0) {
    return (
      <div className="h-1.5 rounded-full bg-gray-200 w-full" />
    );
  }

  const normalizedBullish = (bullish / total) * 100;
  const normalizedBearish = (bearish / total) * 100;
  const normalizedNeutral = (neutral / total) * 100;

  return (
    <div className="h-1.5 rounded-full overflow-hidden flex w-full">
      {normalizedBullish > 0 && (
        <div
          className="bg-green-500"
          style={{ width: `${normalizedBullish}%` }}
        />
      )}
      {normalizedNeutral > 0 && (
        <div
          className="bg-gray-300"
          style={{ width: `${normalizedNeutral}%` }}
        />
      )}
      {normalizedBearish > 0 && (
        <div
          className="bg-red-500"
          style={{ width: `${normalizedBearish}%` }}
        />
      )}
    </div>
  );
}
