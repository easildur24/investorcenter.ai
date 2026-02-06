'use client';

import { BacktestSummary, formatPercent } from '@/lib/types/backtest';

interface StatisticalSummaryProps {
  summary: BacktestSummary;
}

export function StatisticalSummary({ summary }: StatisticalSummaryProps) {
  // Calculate additional statistics from the decile data
  const decileReturns = summary.decile_performance.map((dp) => dp.annualized_return);

  // Check monotonicity - do returns decrease with decile?
  let monotonicViolations = 0;
  for (let i = 0; i < decileReturns.length - 1; i++) {
    if (decileReturns[i] < decileReturns[i + 1]) {
      monotonicViolations++;
    }
  }

  // Calculate return spread statistics
  const avgReturn = decileReturns.reduce((a, b) => a + b, 0) / decileReturns.length;
  const variance =
    decileReturns.reduce((sum, r) => sum + Math.pow(r - avgReturn, 2), 0) / decileReturns.length;
  const stdDev = Math.sqrt(variance);

  return (
    <div className="bg-white border rounded-lg p-6">
      <h3 className="text-lg font-semibold mb-4">Statistical Analysis</h3>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Predictive Power */}
        <div className="space-y-4">
          <h4 className="font-medium text-gray-700">Predictive Power</h4>

          <div className="space-y-3">
            <StatRow
              label="Top vs Bottom Spread"
              value={formatPercent(summary.spread_cagr)}
              description="Annual return difference between D1 and D10"
              highlight={summary.spread_cagr > 0.10}
            />

            <StatRow
              label="Hit Rate"
              value={formatPercent(summary.hit_rate)}
              description={`D1 outperforms D10 in ${formatPercent(summary.hit_rate)} of periods`}
              highlight={summary.hit_rate > 0.6}
            />

            <StatRow
              label="Monotonicity Score"
              value={formatPercent(summary.monotonicity_score)}
              description={`${monotonicViolations} decile ordering violations`}
              highlight={summary.monotonicity_score > 0.8}
            />

            <StatRow
              label="Information Ratio"
              value={summary.information_ratio.toFixed(2)}
              description="Excess return per unit of tracking error"
              highlight={summary.information_ratio > 0.5}
            />
          </div>
        </div>

        {/* Risk Analysis */}
        <div className="space-y-4">
          <h4 className="font-medium text-gray-700">Risk Analysis</h4>

          <div className="space-y-3">
            <StatRow
              label="Top Decile Sharpe"
              value={summary.top_decile_sharpe.toFixed(2)}
              description="Risk-adjusted return of top decile"
              highlight={summary.top_decile_sharpe > 1}
            />

            <StatRow
              label="Top Decile Max Drawdown"
              value={`-${formatPercent(summary.top_decile_max_dd)}`}
              description="Worst peak-to-trough decline"
              highlight={summary.top_decile_max_dd < 0.3}
              negative
            />

            <StatRow
              label="Decile Return Std Dev"
              value={formatPercent(stdDev)}
              description="Dispersion of returns across deciles"
              highlight={stdDev > 0.05}
            />

            <StatRow
              label="vs Benchmark"
              value={formatPercent(summary.top_vs_benchmark)}
              description={`Excess return over ${summary.benchmark}`}
              highlight={summary.top_vs_benchmark > 0.05}
            />
          </div>
        </div>
      </div>

      {/* Interpretation */}
      <div className="mt-6 pt-4 border-t">
        <h4 className="font-medium text-gray-700 mb-3">Interpretation</h4>
        <div className="text-sm text-gray-600 space-y-2">
          {getInterpretation(summary)}
        </div>
      </div>

      {/* Confidence Level */}
      <div className="mt-4 pt-4 border-t">
        <div className="flex items-center gap-4">
          <span className="text-sm font-medium text-gray-700">Overall Confidence:</span>
          <ConfidenceMeter
            hitRate={summary.hit_rate}
            monotonicity={summary.monotonicity_score}
            spread={summary.spread_cagr}
          />
        </div>
      </div>
    </div>
  );
}

function StatRow({
  label,
  value,
  description,
  highlight,
  negative,
}: {
  label: string;
  value: string;
  description: string;
  highlight?: boolean;
  negative?: boolean;
}) {
  const valueColor = negative
    ? 'text-red-600'
    : highlight
    ? 'text-green-600'
    : 'text-gray-900';

  return (
    <div className="flex items-center justify-between">
      <div>
        <p className="text-sm font-medium text-gray-700">{label}</p>
        <p className="text-xs text-gray-500">{description}</p>
      </div>
      <span className={`text-lg font-semibold ${valueColor}`}>{value}</span>
    </div>
  );
}

function getInterpretation(summary: BacktestSummary): React.ReactNode[] {
  const points: React.ReactNode[] = [];

  // Spread interpretation
  if (summary.spread_cagr > 0.15) {
    points.push(
      <p key="spread-strong">
        <span className="text-green-600 font-medium">Strong predictive signal:</span> The {formatPercent(summary.spread_cagr)} annual spread between top and bottom deciles demonstrates significant differentiation power.
      </p>
    );
  } else if (summary.spread_cagr > 0.08) {
    points.push(
      <p key="spread-moderate">
        <span className="text-blue-600 font-medium">Moderate predictive signal:</span> The {formatPercent(summary.spread_cagr)} spread is meaningful but leaves room for improvement.
      </p>
    );
  } else if (summary.spread_cagr > 0) {
    points.push(
      <p key="spread-weak">
        <span className="text-yellow-600 font-medium">Weak predictive signal:</span> The {formatPercent(summary.spread_cagr)} spread is positive but relatively small.
      </p>
    );
  } else {
    points.push(
      <p key="spread-none">
        <span className="text-red-600 font-medium">No predictive signal:</span> The negative spread suggests the scoring methodology may need refinement.
      </p>
    );
  }

  // Hit rate interpretation
  if (summary.hit_rate > 0.65) {
    points.push(
      <p key="hitrate">
        <span className="text-green-600 font-medium">Consistent outperformance:</span> Top decile beat bottom decile in {formatPercent(summary.hit_rate)} of periods, well above the 50% random expectation.
      </p>
    );
  } else if (summary.hit_rate > 0.55) {
    points.push(
      <p key="hitrate">
        <span className="text-blue-600 font-medium">Moderate consistency:</span> Hit rate of {formatPercent(summary.hit_rate)} shows some persistence but with notable exceptions.
      </p>
    );
  }

  // Monotonicity interpretation
  if (summary.monotonicity_score > 0.9) {
    points.push(
      <p key="mono">
        <span className="text-green-600 font-medium">Excellent rank ordering:</span> Returns decrease nearly monotonically across deciles, validating the scoring methodology.
      </p>
    );
  } else if (summary.monotonicity_score < 0.7) {
    points.push(
      <p key="mono">
        <span className="text-yellow-600 font-medium">Imperfect ordering:</span> Some middle deciles show unexpected return patterns.
      </p>
    );
  }

  // Benchmark comparison
  if (summary.top_vs_benchmark > 0.05) {
    points.push(
      <p key="bench">
        <span className="text-green-600 font-medium">Alpha generation:</span> Top decile outperformed {summary.benchmark} by {formatPercent(summary.top_vs_benchmark)} annually.
      </p>
    );
  }

  return points;
}

function ConfidenceMeter({
  hitRate,
  monotonicity,
  spread,
}: {
  hitRate: number;
  monotonicity: number;
  spread: number;
}) {
  // Calculate overall confidence score (0-100)
  const hitRateScore = Math.min(100, Math.max(0, (hitRate - 0.5) / 0.3 * 100));
  const monoScore = monotonicity * 100;
  const spreadScore = Math.min(100, Math.max(0, spread / 0.15 * 100));

  const overall = (hitRateScore * 0.3 + monoScore * 0.4 + spreadScore * 0.3);

  const getLabel = (score: number) => {
    if (score >= 80) return { label: 'High', color: 'bg-green-500' };
    if (score >= 60) return { label: 'Moderate', color: 'bg-blue-500' };
    if (score >= 40) return { label: 'Low', color: 'bg-yellow-500' };
    return { label: 'Very Low', color: 'bg-red-500' };
  };

  const { label, color } = getLabel(overall);

  return (
    <div className="flex items-center gap-3">
      <div className="flex-1 h-2 bg-gray-200 rounded-full overflow-hidden max-w-[200px]">
        <div
          className={`h-full ${color} transition-all duration-500`}
          style={{ width: `${overall}%` }}
        />
      </div>
      <span className={`text-sm font-medium ${color.replace('bg-', 'text-')}`}>
        {label} ({Math.round(overall)}%)
      </span>
    </div>
  );
}
