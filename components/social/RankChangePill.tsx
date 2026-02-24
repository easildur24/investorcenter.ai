'use client';

interface RankChangePillProps {
  change: number | null | undefined;
}

/**
 * Displays rank change as an integer with directional arrow.
 * BUG-001 fix: Always rounds to eliminate floating-point display.
 */
export default function RankChangePill({ change }: RankChangePillProps) {
  if (change === null || change === undefined) {
    return (
      <span className="text-gray-400 text-sm" data-testid="rank-change">
        NEW
      </span>
    );
  }

  // BUG-001: Math.round to eliminate floating-point display
  const rounded = Math.round(change);

  if (rounded === 0) {
    return (
      <span className="text-gray-400 text-sm" data-testid="rank-change">
        —
      </span>
    );
  }

  const isUp = rounded > 0;
  return (
    <span
      className={`text-sm font-medium ${isUp ? 'text-green-600' : 'text-red-600'}`}
      data-testid="rank-change"
    >
      {isUp ? '\u2191' : '\u2193'}
      {Math.abs(rounded)}
    </span>
  );
}
