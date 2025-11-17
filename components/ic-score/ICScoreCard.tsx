'use client';

import { useState, useEffect } from 'react';
import { getICScore, ICScoreData } from '@/lib/api/ic-score';
import ICScoreWidget from './ICScoreWidget';
import FactorBreakdown from './FactorBreakdown';

interface ICScoreCardProps {
  ticker: string;
  variant?: 'full' | 'compact';
}

/**
 * Main IC Score card component
 *
 * Displays complete IC Score analysis with widget and factor breakdown
 */
export default function ICScoreCard({ ticker, variant = 'full' }: ICScoreCardProps) {
  const [icScore, setIcScore] = useState<ICScoreData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showFactors, setShowFactors] = useState(false);

  useEffect(() => {
    async function fetchICScore() {
      try {
        setLoading(true);
        setError(null);
        const data = await getICScore(ticker);
        setIcScore(data);
      } catch (err) {
        console.error('Error fetching IC Score:', err);
        setError(err instanceof Error ? err.message : 'Failed to load IC Score');
      } finally {
        setLoading(false);
      }
    }

    fetchICScore();
  }, [ticker]);

  if (loading) {
    return <LoadingSkeleton variant={variant} />;
  }

  if (error || !icScore) {
    return <ErrorState message={error || 'IC Score not available'} ticker={ticker} />;
  }

  if (variant === 'compact') {
    return (
      <div className="bg-white rounded-lg shadow border border-gray-200 p-6">
        <div className="flex items-start justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-gray-900">IC Score</h3>
            <p className="text-sm text-gray-600">
              {icScore.factor_count} of 10 factors available
            </p>
          </div>
          <a
            href={`/ticker/${ticker}#ic-score`}
            className="text-sm text-blue-600 hover:text-blue-700 hover:underline"
          >
            View Details →
          </a>
        </div>

        <div className="text-center mb-4">
          <div className="text-5xl font-bold text-green-600 mb-2">
            {Math.round(icScore.overall_score)}
            <span className="text-xl text-gray-400">/100</span>
          </div>
          <div className="text-sm font-medium text-gray-700">
            {icScore.rating}
          </div>
        </div>

        {/* Progress bar */}
        <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
          <div
            className="h-full bg-green-500 transition-all"
            style={{ width: `${icScore.overall_score}%` }}
          />
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6" id="ic-score">
      {/* Main Widget */}
      <div onClick={() => setShowFactors(!showFactors)} className="cursor-pointer">
        <ICScoreWidget icScore={icScore} showFactors={showFactors} />
      </div>

      {/* Factor Breakdown - shown when expanded */}
      {showFactors && (
        <div className="animate-fadeIn">
          <FactorBreakdown icScore={icScore} />
        </div>
      )}
    </div>
  );
}

function LoadingSkeleton({ variant }: { variant: 'full' | 'compact' }) {
  if (variant === 'compact') {
    return (
      <div className="bg-white rounded-lg shadow border border-gray-200 p-6 animate-pulse">
        <div className="h-6 bg-gray-200 rounded w-32 mb-4"></div>
        <div className="h-24 bg-gray-200 rounded mb-4"></div>
        <div className="h-2 bg-gray-200 rounded"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6 animate-pulse">
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <div className="h-64 bg-gray-200"></div>
      </div>
    </div>
  );
}

function ErrorState({ message, ticker }: { message: string; ticker: string }) {
  return (
    <div className="bg-white rounded-lg shadow border border-gray-200 p-8 text-center">
      <div className="text-gray-400 text-5xl mb-4">📊</div>
      <h3 className="text-lg font-semibold text-gray-900 mb-2">IC Score Not Available</h3>
      <p className="text-gray-600 mb-4">
        IC Score for {ticker} hasn't been calculated yet.
      </p>
      <p className="text-sm text-gray-500">
        We're working on expanding coverage. Check back soon!
      </p>
    </div>
  );
}
