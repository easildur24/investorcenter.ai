'use client';

import { useState, useEffect } from 'react';
import { getICScore, getICScoreHistory, ICScoreData } from '@/lib/api/ic-score';
import ICScoreWidget from './ICScoreWidget';
import FactorBreakdown from './FactorBreakdown';
import ICScoreExplainer, { ICScoreExplainerButton } from './ICScoreExplainer';
import ICScoreAIAnalysis from './ICScoreAIAnalysis';
import { formatRelativeTime } from '@/lib/utils';
import { ArrowUpIcon, ArrowDownIcon, MinusIcon } from '@heroicons/react/24/solid';

interface ICScoreCardProps {
  ticker: string;
  variant?: 'full' | 'compact';
}

interface ScoreChange {
  change: number;
  direction: 'up' | 'down' | 'unchanged';
  period: string;
}

/**
 * Main IC Score card component
 *
 * Displays complete IC Score analysis with widget and factor breakdown
 */
export default function ICScoreCard({ ticker, variant = 'full' }: ICScoreCardProps) {
  const [icScore, setIcScore] = useState<ICScoreData | null>(null);
  const [scoreChange, setScoreChange] = useState<ScoreChange | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showFactors, setShowFactors] = useState(false);
  const [showExplainer, setShowExplainer] = useState(false);

  useEffect(() => {
    async function fetchICScoreData() {
      try {
        setLoading(true);
        setError(null);

        // Fetch current score and history in parallel
        const [currentScore, history] = await Promise.all([
          getICScore(ticker),
          getICScoreHistory(ticker, 30).catch(() => []) // Fail silently if history not available
        ]);

        setIcScore(currentScore);

        // Calculate score change from 30 days ago
        if (currentScore && history.length > 0) {
          // Find the oldest score in the history (closest to 30 days ago)
          const oldestScore = history[history.length - 1];
          const change = Math.round(currentScore.overall_score - oldestScore.overall_score);

          setScoreChange({
            change: Math.abs(change),
            direction: change > 0 ? 'up' : change < 0 ? 'down' : 'unchanged',
            period: '30 days'
          });
        }
      } catch (err) {
        console.error('Error fetching IC Score:', err);
        setError(err instanceof Error ? err.message : 'Failed to load IC Score');
      } finally {
        setLoading(false);
      }
    }

    fetchICScoreData();
  }, [ticker]);

  if (loading) {
    return <LoadingSkeleton variant={variant} />;
  }

  if (error || !icScore) {
    return <ErrorState message={error || 'IC Score not available'} ticker={ticker} />;
  }

  if (variant === 'compact') {
    return (
      <>
        <div className="bg-ic-surface rounded-lg shadow border border-ic-border p-6">
          <div className="flex items-start justify-between mb-4">
            <div>
              <h3 className="text-lg font-semibold text-ic-text-primary">IC Score</h3>
              <p className="text-sm text-ic-text-muted">
                {icScore.factor_count} of 10 factors available
              </p>
            </div>
            <div className="flex flex-col items-end gap-1">
              <a
                href={`/ticker/${ticker}#ic-score`}
                className="text-sm text-ic-blue hover:text-blue-700 hover:underline"
              >
                View Details →
              </a>
              <ICScoreExplainerButton onClick={() => setShowExplainer(true)} />
            </div>
          </div>

          <div className="text-center mb-4">
            <div className="text-5xl font-bold text-ic-positive mb-2">
              {Math.round(icScore.overall_score)}
              <span className="text-xl text-ic-text-dim">/100</span>
            </div>
            <div className="text-sm font-medium text-ic-text-secondary mb-2">
              {icScore.rating}
            </div>

            {/* Score Change Indicator */}
            {scoreChange && scoreChange.change !== 0 && (
              <div className={`inline-flex items-center gap-1 text-sm font-medium ${
                scoreChange.direction === 'up' ? 'text-ic-positive' :
                scoreChange.direction === 'down' ? 'text-ic-negative' : 'text-ic-text-muted'
              }`}>
                {scoreChange.direction === 'up' && <ArrowUpIcon className="h-4 w-4" />}
                {scoreChange.direction === 'down' && <ArrowDownIcon className="h-4 w-4" />}
                {scoreChange.direction === 'unchanged' && <MinusIcon className="h-4 w-4" />}
                <span>
                  {scoreChange.direction === 'up' ? '+' : scoreChange.direction === 'down' ? '-' : ''}
                  {scoreChange.change} pts ({scoreChange.period})
                </span>
              </div>
            )}
          </div>

          {/* Progress bar */}
          <div className="h-2 bg-ic-bg-secondary rounded-full overflow-hidden">
            <div
              className="h-full bg-green-500 transition-all"
              style={{ width: `${icScore.overall_score}%` }}
            />
          </div>

          {/* Last updated timestamp */}
          <p className="text-xs text-ic-text-dim mt-3 text-center">
            Updated {formatRelativeTime(icScore.calculated_at)}
          </p>
        </div>

        {/* Explainer Modal */}
        {showExplainer && (
          <ICScoreExplainer icScore={icScore} onClose={() => setShowExplainer(false)} />
        )}
      </>
    );
  }

  return (
    <>
      <div className="space-y-6" id="ic-score">
        {/* Main Widget */}
        <div onClick={() => setShowFactors(!showFactors)} className="cursor-pointer">
          <ICScoreWidget icScore={icScore} showFactors={showFactors} scoreChange={scoreChange} />
        </div>

        {/* Explainer Link */}
        <div className="flex justify-center">
          <ICScoreExplainerButton onClick={() => setShowExplainer(true)} />
        </div>

        {/* Factor Breakdown - shown when expanded */}
        {showFactors && (
          <div className="animate-fadeIn">
            <FactorBreakdown icScore={icScore} />
          </div>
        )}

        {/* AI Analysis */}
        <ICScoreAIAnalysis icScore={icScore} />
      </div>

      {/* Explainer Modal */}
      {showExplainer && (
        <ICScoreExplainer icScore={icScore} onClose={() => setShowExplainer(false)} />
      )}
    </>
  );
}

function LoadingSkeleton({ variant }: { variant: 'full' | 'compact' }) {
  if (variant === 'compact') {
    return (
      <div className="bg-ic-surface rounded-lg shadow border border-ic-border p-6 animate-pulse">
        <div className="h-6 bg-ic-bg-secondary rounded w-32 mb-4"></div>
        <div className="h-24 bg-ic-bg-secondary rounded mb-4"></div>
        <div className="h-2 bg-ic-bg-secondary rounded"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6 animate-pulse">
      <div className="bg-ic-surface rounded-lg border border-ic-border overflow-hidden">
        <div className="h-64 bg-ic-bg-secondary"></div>
      </div>
    </div>
  );
}

function ErrorState({ message, ticker }: { message: string; ticker: string }) {
  return (
    <div className="bg-ic-surface rounded-lg shadow border border-ic-border p-8 text-center">
      <div className="text-ic-text-dim text-5xl mb-4">📊</div>
      <h3 className="text-lg font-semibold text-ic-text-primary mb-2">IC Score Not Available</h3>
      <p className="text-ic-text-muted mb-4">
        IC Score for {ticker} hasn't been calculated yet.
      </p>
      <p className="text-sm text-ic-text-muted">
        We're working on expanding coverage. Check back soon!
      </p>
    </div>
  );
}
