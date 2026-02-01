'use client';

import { useState, useEffect } from 'react';
import { getICScore, ICScoreData } from '@/lib/api/ic-score';
import {
  ICScoreDataV2,
  getCategoryScore,
  Categories,
} from '@/lib/types/ic-score-v2';
import ICScoreGauge from './ICScoreGauge';
import { CategoryScoreGrid } from './CategoryScoreBadge';
import { LifecycleBadge, ConfidenceBadge, RatingBadge, SectorRankBadge } from './Badges';
import FactorBreakdown from './FactorBreakdown';
import PeerComparisonPanel from './PeerComparisonPanel';
import CatalystTimeline, { CatalystCompact } from './CatalystTimeline';
import ScoreChangeExplainer, { ScoreChangeInline } from './ScoreChangeExplainer';
import GranularConfidenceDisplay from './GranularConfidenceDisplay';
import ICScoreExplainer, { ICScoreExplainerButton } from './ICScoreExplainer';
import { formatRelativeTime } from '@/lib/utils';

interface ICScoreCardV2Props {
  ticker: string;
  showPeers?: boolean;
  showCatalysts?: boolean;
  showExplanation?: boolean;
  variant?: 'full' | 'compact';
}

/**
 * ICScoreCardV2 - Enhanced IC Score card with v2.1 features
 *
 * Features:
 * - Category breakdown (Quality, Valuation, Signals)
 * - Lifecycle classification badge
 * - Peer comparison panel
 * - Catalyst timeline
 * - Score change explanations
 * - Granular confidence display
 */
export default function ICScoreCardV2({
  ticker,
  showPeers = true,
  showCatalysts = true,
  showExplanation = true,
  variant = 'full',
}: ICScoreCardV2Props) {
  const [icScore, setIcScore] = useState<ICScoreDataV2 | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedSection, setExpandedSection] = useState<string | null>(null);
  const [showExplainerModal, setShowExplainerModal] = useState(false);

  useEffect(() => {
    async function fetchData() {
      try {
        setLoading(true);
        setError(null);
        const score = await getICScore(ticker);
        // Cast to v2 type (API should return v2 data)
        setIcScore(score as unknown as ICScoreDataV2);
      } catch (err) {
        console.error('Error fetching IC Score:', err);
        setError(err instanceof Error ? err.message : 'Failed to load IC Score');
      } finally {
        setLoading(false);
      }
    }

    fetchData();
  }, [ticker]);

  if (loading) {
    return <LoadingSkeleton variant={variant} />;
  }

  if (error || !icScore) {
    return <ErrorState message={error || 'IC Score not available'} ticker={ticker} />;
  }

  // Calculate category scores
  const categories: Categories = {
    quality: getCategoryScore(icScore, 'quality'),
    valuation: getCategoryScore(icScore, 'valuation'),
    signals: getCategoryScore(icScore, 'signals'),
  };

  if (variant === 'compact') {
    return (
      <CompactCard
        icScore={icScore}
        categories={categories}
        onShowExplainer={() => setShowExplainerModal(true)}
        showExplainerModal={showExplainerModal}
        onCloseExplainer={() => setShowExplainerModal(false)}
      />
    );
  }

  return (
    <>
      <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden" id="ic-score">
        {/* Header with gauge and badges */}
        <div className="p-6 bg-gradient-to-br from-gray-50 to-white">
          <div className="flex items-start justify-between gap-6">
            {/* Left: Gauge and rating */}
            <div className="flex items-center gap-6">
              <ICScoreGauge
                score={icScore.overall_score}
                size="lg"
                showLabel
              />
              <div>
                <RatingBadge rating={icScore.rating} size="lg" />
                <ScoreChangeInline
                  previousScore={icScore.previous_score}
                  currentScore={icScore.overall_score}
                />
              </div>
            </div>

            {/* Right: Badges */}
            <div className="flex flex-col items-end gap-2">
              <div className="flex items-center gap-2">
                <LifecycleBadge stage={icScore.lifecycle_stage} />
                <ConfidenceBadge level={icScore.confidence_level} />
              </div>
              {icScore.sector_rank && icScore.sector_total && (
                <SectorRankBadge rank={icScore.sector_rank} total={icScore.sector_total} />
              )}
              <ICScoreExplainerButton onClick={() => setShowExplainerModal(true)} />
            </div>
          </div>
        </div>

        {/* Category breakdown */}
        <div className="p-4 border-t border-gray-100">
          <CategoryScoreGrid
            quality={categories.quality}
            valuation={categories.valuation}
            signals={categories.signals}
          />
        </div>

        {/* Expandable sections */}
        <div className="divide-y divide-gray-100">
          {/* Factor Breakdown */}
          <CollapsibleSection
            title="Factor Breakdown"
            isOpen={expandedSection === 'factors'}
            onToggle={() => setExpandedSection(expandedSection === 'factors' ? null : 'factors')}
          >
            <FactorBreakdown icScore={icScore as unknown as ICScoreData} />
          </CollapsibleSection>

          {/* Peer Comparison */}
          {showPeers && (icScore.peers || icScore.peer_comparison) && (
            <PeerComparisonPanel
              ticker={ticker}
              currentScore={icScore.overall_score}
              peerComparison={icScore.peer_comparison}
              peers={icScore.peers}
            />
          )}

          {/* Catalysts */}
          {showCatalysts && icScore.catalysts && icScore.catalysts.length > 0 && (
            <CatalystTimeline catalysts={icScore.catalysts} />
          )}

          {/* Score Explanation */}
          {showExplanation && icScore.explanation && (
            <ScoreChangeExplainer
              explanation={icScore.explanation}
              previousScore={icScore.previous_score}
              currentScore={icScore.overall_score}
            />
          )}

          {/* Granular Confidence */}
          {icScore.explanation?.confidence && (
            <GranularConfidenceDisplay
              confidence={{
                level: icScore.explanation.confidence.level as any,
                percentage: icScore.explanation.confidence.percentage,
                factors: {},
                warnings: icScore.explanation.confidence.warnings,
              }}
            />
          )}
        </div>

        {/* Footer */}
        <div className="px-4 py-3 bg-gray-50 border-t border-gray-100 text-xs text-gray-500 flex justify-between items-center">
          <span>Updated {formatRelativeTime(icScore.calculated_at)}</span>
          <span>{icScore.factor_count} of 13 factors available</span>
        </div>
      </div>

      {/* Explainer Modal */}
      {showExplainerModal && (
        <ICScoreExplainer
          icScore={icScore as unknown as ICScoreData}
          onClose={() => setShowExplainerModal(false)}
        />
      )}
    </>
  );
}

// Compact version
interface CompactCardProps {
  icScore: ICScoreDataV2;
  categories: Categories;
  onShowExplainer: () => void;
  showExplainerModal: boolean;
  onCloseExplainer: () => void;
}

function CompactCard({
  icScore,
  categories,
  onShowExplainer,
  showExplainerModal,
  onCloseExplainer,
}: CompactCardProps) {
  return (
    <>
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-3">
            <ICScoreGauge score={icScore.overall_score} size="sm" />
            <div>
              <RatingBadge rating={icScore.rating} size="sm" />
              <ScoreChangeInline
                previousScore={icScore.previous_score}
                currentScore={icScore.overall_score}
              />
            </div>
          </div>
          <div className="flex flex-col items-end gap-1">
            <LifecycleBadge stage={icScore.lifecycle_stage} size="sm" />
            <ICScoreExplainerButton onClick={onShowExplainer} />
          </div>
        </div>

        {/* Mini category display */}
        <div className="flex gap-2 text-sm">
          <MiniCategoryBadge name="Q" score={categories.quality.score} />
          <MiniCategoryBadge name="V" score={categories.valuation.score} />
          <MiniCategoryBadge name="S" score={categories.signals.score} />
        </div>

        {/* Next catalyst */}
        {icScore.catalysts && icScore.catalysts.length > 0 && (
          <div className="mt-3 pt-3 border-t border-gray-100">
            <CatalystCompact catalysts={icScore.catalysts} />
          </div>
        )}
      </div>

      {showExplainerModal && (
        <ICScoreExplainer
          icScore={icScore as unknown as ICScoreData}
          onClose={onCloseExplainer}
        />
      )}
    </>
  );
}

function MiniCategoryBadge({ name, score }: { name: string; score: number }) {
  const bgColor =
    score >= 80
      ? 'bg-green-100 text-green-700'
      : score >= 65
      ? 'bg-green-50 text-green-600'
      : score >= 50
      ? 'bg-yellow-100 text-yellow-700'
      : score >= 35
      ? 'bg-orange-100 text-orange-700'
      : 'bg-red-100 text-red-700';

  return (
    <span className={`px-2 py-1 rounded text-xs font-medium ${bgColor}`}>
      {name}: {score}
    </span>
  );
}

// Collapsible section wrapper
interface CollapsibleSectionProps {
  title: string;
  isOpen: boolean;
  onToggle: () => void;
  children: React.ReactNode;
}

function CollapsibleSection({ title, isOpen, onToggle, children }: CollapsibleSectionProps) {
  return (
    <div>
      <button
        className="w-full px-4 py-3 flex items-center justify-between text-left hover:bg-gray-50 transition-colors"
        onClick={onToggle}
      >
        <span className="font-medium text-gray-900">{title}</span>
        <span className="text-gray-400">{isOpen ? '▲' : '▼'}</span>
      </button>
      {isOpen && <div className="px-4 pb-4">{children}</div>}
    </div>
  );
}

function LoadingSkeleton({ variant }: { variant: 'full' | 'compact' }) {
  if (variant === 'compact') {
    return (
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4 animate-pulse">
        <div className="flex items-center gap-3 mb-4">
          <div className="w-16 h-16 bg-gray-200 rounded-full" />
          <div className="flex-1">
            <div className="h-6 bg-gray-200 rounded w-24 mb-2" />
            <div className="h-4 bg-gray-200 rounded w-16" />
          </div>
        </div>
        <div className="flex gap-2">
          <div className="h-6 bg-gray-200 rounded w-12" />
          <div className="h-6 bg-gray-200 rounded w-12" />
          <div className="h-6 bg-gray-200 rounded w-12" />
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden animate-pulse">
      <div className="p-6">
        <div className="flex items-center gap-6">
          <div className="w-24 h-24 bg-gray-200 rounded-full" />
          <div>
            <div className="h-8 bg-gray-200 rounded w-32 mb-2" />
            <div className="h-4 bg-gray-200 rounded w-24" />
          </div>
        </div>
      </div>
      <div className="p-4 border-t">
        <div className="grid grid-cols-3 gap-4">
          <div className="h-24 bg-gray-200 rounded" />
          <div className="h-24 bg-gray-200 rounded" />
          <div className="h-24 bg-gray-200 rounded" />
        </div>
      </div>
    </div>
  );
}

function ErrorState({ message, ticker }: { message: string; ticker: string }) {
  return (
    <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-8 text-center">
      <div className="text-gray-400 text-5xl mb-4">📊</div>
      <h3 className="text-lg font-semibold text-gray-900 mb-2">IC Score Not Available</h3>
      <p className="text-gray-500 mb-4">
        IC Score for {ticker} hasn't been calculated yet.
      </p>
      <p className="text-sm text-gray-400">
        We're working on expanding coverage. Check back soon!
      </p>
    </div>
  );
}
