'use client';

import React, { useState, useEffect } from 'react';
import ICScoreGauge from './ICScoreGauge';
import ICScoreFactorChart, { ICScoreFactorList } from './ICScoreFactorChart';
import ICScoreTrendChart from './ICScoreTrendChart';
import { icScoreApi } from '@/lib/api';
import type { ICScore, ICScoreHistory } from '@/lib/types/ic-score';

interface ICScoreCardProps {
  ticker: string;
  variant?: 'full' | 'compact';
}

/**
 * Main IC Score card component
 *
 * Displays complete IC Score analysis with:
 * 1. Circular gauge showing overall score
 * 2. 10-factor breakdown chart
 * 3. 30-day historical trend
 *
 * Variants:
 * - full: Complete card with all sections (for dedicated IC Score page)
 * - compact: Condensed version (for ticker page integration)
 */
export default function ICScoreCard({ ticker, variant = 'full' }: ICScoreCardProps) {
  const [score, setScore] = useState<ICScore | null>(null);
  const [history, setHistory] = useState<ICScoreHistory | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'overview' | 'factors' | 'history'>('overview');

  useEffect(() => {
    async function fetchICScore() {
      try {
        setLoading(true);
        setError(null);

        // Fetch score and history in parallel
        const [scoreData, historyData] = await Promise.all([
          icScoreApi.getScore(ticker),
          icScoreApi.getHistory(ticker, 30),
        ]);

        setScore(scoreData);
        setHistory(historyData);
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

  if (error) {
    return <ErrorState message={error} ticker={ticker} />;
  }

  if (!score) {
    return <ErrorState message="IC Score not available" ticker={ticker} />;
  }

  if (variant === 'compact') {
    return <CompactCard score={score} history={history} />;
  }

  return (
    <div className="bg-white rounded-xl shadow-lg border border-gray-200 overflow-hidden">
      {/* Header */}
      <div className="bg-gradient-to-r from-blue-600 to-blue-700 px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-2xl font-bold text-white">IC Score Analysis</h2>
            <p className="text-blue-100 text-sm mt-1">
              Proprietary 10-factor scoring system for {ticker}
            </p>
          </div>
          <div className="text-right">
            <div className="text-xs text-blue-100">Last Updated</div>
            <div className="text-sm text-white font-medium">
              {new Date(score.calculatedAt).toLocaleDateString()}
            </div>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200 bg-gray-50">
        <div className="flex gap-1 px-6">
          <TabButton
            active={activeTab === 'overview'}
            onClick={() => setActiveTab('overview')}
          >
            Overview
          </TabButton>
          <TabButton
            active={activeTab === 'factors'}
            onClick={() => setActiveTab('factors')}
          >
            Factor Breakdown
          </TabButton>
          <TabButton
            active={activeTab === 'history'}
            onClick={() => setActiveTab('history')}
          >
            30-Day Trend
          </TabButton>
        </div>
      </div>

      {/* Content */}
      <div className="p-6">
        {activeTab === 'overview' && (
          <OverviewTab score={score} history={history} />
        )}
        {activeTab === 'factors' && (
          <FactorsTab factors={score.factors} />
        )}
        {activeTab === 'history' && history && (
          <HistoryTab history={history} />
        )}
      </div>

      {/* Footer */}
      <div className="bg-gray-50 px-6 py-4 border-t border-gray-200">
        <div className="flex items-center justify-between text-sm">
          <div className="text-gray-600">
            <span className="font-medium">Percentile Rank:</span>{' '}
            {score.percentile ? `Top ${(100 - score.percentile).toFixed(0)}%` : 'N/A'}
            {score.sectorPercentile && (
              <span className="ml-4">
                <span className="font-medium">Sector Rank:</span>{' '}
                Top {(100 - score.sectorPercentile).toFixed(0)}%
              </span>
            )}
          </div>
          <a
            href="/ic-score"
            className="text-blue-600 hover:text-blue-700 font-medium hover:underline"
          >
            View All IC Scores →
          </a>
        </div>
      </div>
    </div>
  );
}

/**
 * Tab components
 */
function OverviewTab({ score, history }: { score: ICScore; history: ICScoreHistory | null }) {
  return (
    <div className="grid md:grid-cols-2 gap-8">
      {/* Left: Gauge */}
      <div className="flex flex-col items-center justify-center">
        <ICScoreGauge score={score.score} size="lg" showLabel showRating />

        {/* Score change */}
        {score.previousScore !== undefined && score.scoreChange !== undefined && (
          <div className="mt-4 text-center">
            <div className="text-sm text-gray-600">Change from previous</div>
            <div
              className={`text-lg font-bold ${
                score.scoreChange >= 0 ? 'text-green-600' : 'text-red-600'
              }`}
            >
              {score.scoreChange >= 0 ? '+' : ''}
              {score.scoreChange.toFixed(1)}
            </div>
          </div>
        )}
      </div>

      {/* Right: Top factors */}
      <div>
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Top 5 Factors</h3>
        <TopFactorsList factors={score.factors} limit={5} />
      </div>
    </div>
  );
}

function FactorsTab({ factors }: { factors: ICScore['factors'] }) {
  return (
    <div>
      <h3 className="text-lg font-semibold text-gray-900 mb-4">10-Factor Breakdown</h3>
      <ICScoreFactorChart factors={factors} height={500} />
    </div>
  );
}

function HistoryTab({ history }: { history: ICScoreHistory }) {
  return (
    <div>
      <h3 className="text-lg font-semibold text-gray-900 mb-4">
        30-Day IC Score Trend
      </h3>
      <ICScoreTrendChart history={history} height={400} showStats />
    </div>
  );
}

/**
 * Compact card variant
 */
function CompactCard({ score, history }: { score: ICScore; history: ICScoreHistory | null }) {
  return (
    <div className="bg-white rounded-lg shadow border border-gray-200 p-6">
      <div className="flex items-start justify-between mb-4">
        <h3 className="text-lg font-semibold text-gray-900">IC Score</h3>
        <a
          href="/ic-score"
          className="text-sm text-blue-600 hover:text-blue-700 hover:underline"
        >
          View Details →
        </a>
      </div>

      <div className="grid md:grid-cols-2 gap-6">
        <div>
          <ICScoreGauge score={score.score} size="sm" showLabel showRating />
        </div>
        <div>
          <h4 className="text-sm font-medium text-gray-700 mb-3">Top 3 Factors</h4>
          <TopFactorsList factors={score.factors} limit={3} compact />
        </div>
      </div>

      {history && (
        <div className="mt-6 pt-4 border-t border-gray-200">
          <h4 className="text-sm font-medium text-gray-700 mb-3">30-Day Trend</h4>
          <ICScoreTrendChart history={history} height={150} showStats={false} />
        </div>
      )}
    </div>
  );
}

/**
 * Helper components
 */
interface TopFactorsListProps {
  factors: ICScore['factors'];
  limit?: number;
  compact?: boolean;
}

function TopFactorsList({ factors, limit = 5, compact = false }: TopFactorsListProps) {
  const factorArray = Object.entries(factors)
    .map(([key, factor]) => ({
      name: key.replace(/_/g, ' ').replace(/\b\w/g, (l) => l.toUpperCase()),
      ...factor,
    }))
    .sort((a, b) => b.contribution - a.contribution)
    .slice(0, limit);

  return (
    <div className={compact ? 'space-y-2' : 'space-y-3'}>
      {factorArray.map((factor, index) => (
        <div
          key={index}
          className={`flex items-center justify-between ${
            compact ? 'text-xs' : 'text-sm'
          }`}
        >
          <span className="text-gray-700 font-medium">{factor.name}</span>
          <span className="text-gray-900 font-bold">{factor.value.toFixed(1)}</span>
        </div>
      ))}
    </div>
  );
}

function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      className={`px-4 py-2 text-sm font-medium transition-colors border-b-2 ${
        active
          ? 'text-blue-600 border-blue-600'
          : 'text-gray-600 border-transparent hover:text-gray-900 hover:border-gray-300'
      }`}
    >
      {children}
    </button>
  );
}

function LoadingSkeleton({ variant }: { variant: 'full' | 'compact' }) {
  if (variant === 'compact') {
    return (
      <div className="bg-white rounded-lg shadow border border-gray-200 p-6 animate-pulse">
        <div className="h-6 bg-gray-200 rounded w-32 mb-4"></div>
        <div className="grid md:grid-cols-2 gap-6">
          <div className="h-48 bg-gray-200 rounded"></div>
          <div className="space-y-3">
            <div className="h-4 bg-gray-200 rounded"></div>
            <div className="h-4 bg-gray-200 rounded"></div>
            <div className="h-4 bg-gray-200 rounded"></div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-xl shadow-lg border border-gray-200 overflow-hidden animate-pulse">
      <div className="bg-gray-300 h-24"></div>
      <div className="p-6 space-y-4">
        <div className="h-64 bg-gray-200 rounded"></div>
        <div className="h-64 bg-gray-200 rounded"></div>
      </div>
    </div>
  );
}

function ErrorState({ message, ticker }: { message: string; ticker: string }) {
  return (
    <div className="bg-white rounded-lg shadow border border-gray-200 p-8 text-center">
      <div className="text-red-500 text-4xl mb-3">⚠️</div>
      <h3 className="text-lg font-semibold text-gray-900 mb-2">IC Score Unavailable</h3>
      <p className="text-gray-600 mb-4">{message}</p>
      <p className="text-sm text-gray-500">
        IC Score data for {ticker} may not be available yet. Please check back later.
      </p>
    </div>
  );
}
