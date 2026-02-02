'use client';

import {
  ScoreExplanation,
  FactorChange,
  getDeltaColor,
  formatDelta,
  FACTOR_CONFIGS,
} from '@/lib/types/ic-score-v2';
import { DeltaBadge, ConfidenceBadge } from './Badges';

interface ScoreChangeExplainerProps {
  explanation: ScoreExplanation | undefined;
  previousScore?: number | null;
  currentScore: number;
}

/**
 * ScoreChangeExplainer - Shows why a score changed
 *
 * Displays:
 * - Summary of score change
 * - Factor-level breakdown with contributions
 * - Confidence information
 */
export default function ScoreChangeExplainer({
  explanation,
  previousScore,
  currentScore,
}: ScoreChangeExplainerProps) {
  if (!explanation) {
    return null;
  }

  const { summary, delta, reasons, confidence } = explanation;

  // No significant change
  if (Math.abs(delta) < 0.5 && reasons.length === 0) {
    return (
      <div className="border-t border-gray-200 p-4">
        <h3 className="font-medium text-gray-900 mb-2">Score Analysis</h3>
        <p className="text-sm text-gray-500">
          Score unchanged since last calculation.
        </p>
      </div>
    );
  }

  return (
    <div className="border-t border-gray-200 p-4">
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-medium text-gray-900">Score Analysis</h3>
        <ConfidenceBadge level={confidence.level} percentage={confidence.percentage} size="sm" />
      </div>

      {/* Summary */}
      <div className="mb-4 p-3 bg-gray-50 rounded-lg">
        <div className="flex items-center gap-3 mb-2">
          {previousScore != null && (
            <span className="text-gray-500">
              {Math.round(previousScore)} → {Math.round(currentScore)}
            </span>
          )}
          <DeltaBadge delta={delta} />
        </div>
        <p className="text-sm text-gray-700">{summary}</p>
      </div>

      {/* Factor breakdown */}
      {reasons.length > 0 && (
        <div className="space-y-2">
          <h4 className="text-sm font-medium text-gray-700 mb-2">Key Drivers</h4>
          {reasons.map((reason, index) => (
            <FactorChangeRow key={reason.factor || index} change={reason} />
          ))}
        </div>
      )}

      {/* Warnings */}
      {confidence.warnings.length > 0 && (
        <div className="mt-4 p-3 bg-yellow-50 border border-yellow-100 rounded-lg">
          <h4 className="text-sm font-medium text-yellow-800 mb-1">Data Notes</h4>
          <ul className="text-sm text-yellow-700 space-y-1">
            {confidence.warnings.map((warning, index) => (
              <li key={index} className="flex items-start gap-2">
                <span className="text-yellow-500">⚠</span>
                <span>{warning}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}

interface FactorChangeRowProps {
  change: FactorChange;
}

function FactorChangeRow({ change }: FactorChangeRowProps) {
  const factorConfig = FACTOR_CONFIGS.find((f) => f.name === change.factor);
  const displayName = factorConfig?.display_name || formatFactorName(change.factor);
  const icon = factorConfig?.icon || '📊';
  const deltaColor = getDeltaColor(change.delta);

  return (
    <div className="flex items-center gap-3 py-2 px-3 bg-gray-50 rounded">
      <span className="text-lg">{icon}</span>

      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="font-medium text-gray-900">{displayName}</span>
          <span className={`text-sm font-medium ${deltaColor}`}>
            {formatDelta(change.delta)}
          </span>
        </div>
        <p className="text-xs text-gray-500 truncate">{change.explanation}</p>
      </div>

      <div className="text-right flex-shrink-0">
        <div className={`text-sm font-medium ${deltaColor}`}>
          {change.contribution > 0 ? '+' : ''}{change.contribution.toFixed(1)} pts
        </div>
        <div className="text-xs text-gray-400">contribution</div>
      </div>
    </div>
  );
}

function formatFactorName(name: string): string {
  return name
    .replace(/_/g, ' ')
    .replace(/\b\w/g, (l) => l.toUpperCase());
}

/**
 * ScoreChangeSummary - Compact version showing just the summary
 */
interface ScoreChangeSummaryProps {
  explanation: ScoreExplanation | undefined;
}

export function ScoreChangeSummary({ explanation }: ScoreChangeSummaryProps) {
  if (!explanation || Math.abs(explanation.delta) < 0.5) {
    return null;
  }

  const topReason = explanation.reasons[0];
  const deltaColor = getDeltaColor(explanation.delta);

  return (
    <div className="flex items-center gap-2 text-sm">
      <span className={deltaColor}>
        {explanation.delta > 0 ? '↑' : '↓'} {formatDelta(explanation.delta)}
      </span>
      {topReason && (
        <span className="text-gray-500 truncate max-w-[200px]">
          {topReason.explanation}
        </span>
      )}
    </div>
  );
}

/**
 * ScoreChangeInline - Very compact inline display
 */
interface ScoreChangeInlineProps {
  previousScore: number | null | undefined;
  currentScore: number;
}

export function ScoreChangeInline({ previousScore, currentScore }: ScoreChangeInlineProps) {
  if (previousScore === null || previousScore === undefined) {
    return null;
  }

  const delta = currentScore - previousScore;

  if (Math.abs(delta) < 0.5) {
    return (
      <span className="text-gray-400 text-sm">unchanged</span>
    );
  }

  const deltaColor = getDeltaColor(delta);

  return (
    <span className={`text-sm font-medium ${deltaColor}`}>
      {delta > 0 ? '↑' : '↓'} {formatDelta(delta)}
    </span>
  );
}
