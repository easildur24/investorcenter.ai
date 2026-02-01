'use client';

import { useState, useEffect, useMemo } from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  ReferenceLine,
  ResponsiveContainer,
  Area,
  ComposedChart,
} from 'recharts';
import { getICScoreHistory, ICScoreData } from '@/lib/api/ic-score';
import { Catalyst, CATALYST_ICONS } from '@/lib/types/ic-score-v2';

interface ICScoreHistoryChartProps {
  ticker: string;
  days?: number;
  height?: number;
  showCategories?: boolean;
  events?: ScoreEvent[];
}

interface ScoreEvent {
  date: string;
  type: string;
  title: string;
  icon?: string;
  score_impact?: number;
}

interface ChartDataPoint {
  date: string;
  formattedDate: string;
  overall_score: number;
  quality_score?: number;
  valuation_score?: number;
  signals_score?: number;
  event?: ScoreEvent;
}

/**
 * ICScoreHistoryChart - Displays IC Score history over time
 *
 * Features:
 * - Line chart of overall score
 * - Optional category score lines
 * - Rating threshold reference lines
 * - Event annotations
 */
export default function ICScoreHistoryChart({
  ticker,
  days = 90,
  height = 300,
  showCategories = false,
  events = [],
}: ICScoreHistoryChartProps) {
  const [history, setHistory] = useState<ICScoreData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchHistory() {
      try {
        setLoading(true);
        const data = await getICScoreHistory(ticker, days);
        setHistory(data);
      } catch (err) {
        console.error('Error fetching IC Score history:', err);
        setError(err instanceof Error ? err.message : 'Failed to load history');
      } finally {
        setLoading(false);
      }
    }

    fetchHistory();
  }, [ticker, days]);

  const chartData = useMemo(() => {
    return history.map((score) => {
      const date = new Date(score.date);
      const matchingEvent = events.find(
        (e) => new Date(e.date).toDateString() === date.toDateString()
      );

      return {
        date: score.date,
        formattedDate: date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
        overall_score: score.overall_score,
        quality_score: calculateCategoryScore(score, 'quality'),
        valuation_score: calculateCategoryScore(score, 'valuation'),
        signals_score: calculateCategoryScore(score, 'signals'),
        event: matchingEvent,
      };
    });
  }, [history, events]);

  if (loading) {
    return <ChartSkeleton height={height} />;
  }

  if (error || history.length === 0) {
    return (
      <div className="flex items-center justify-center bg-gray-50 rounded-lg" style={{ height }}>
        <p className="text-gray-500">
          {error || 'No historical data available'}
        </p>
      </div>
    );
  }

  return (
    <div className="w-full">
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-medium text-gray-900">IC Score History ({days} Days)</h3>
        <div className="flex items-center gap-4 text-xs">
          <LegendItem color="#2563eb" label="Overall" />
          {showCategories && (
            <>
              <LegendItem color="#10b981" label="Quality" dashed />
              <LegendItem color="#f59e0b" label="Valuation" dashed />
              <LegendItem color="#8b5cf6" label="Signals" dashed />
            </>
          )}
        </div>
      </div>

      <ResponsiveContainer width="100%" height={height}>
        <ComposedChart data={chartData} margin={{ top: 5, right: 20, bottom: 5, left: 0 }}>
          <defs>
            <linearGradient id="scoreGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#2563eb" stopOpacity={0.1} />
              <stop offset="95%" stopColor="#2563eb" stopOpacity={0} />
            </linearGradient>
          </defs>

          <XAxis
            dataKey="formattedDate"
            axisLine={false}
            tickLine={false}
            tick={{ fontSize: 11, fill: '#9ca3af' }}
            interval="preserveStartEnd"
          />
          <YAxis
            domain={[0, 100]}
            axisLine={false}
            tickLine={false}
            tick={{ fontSize: 11, fill: '#9ca3af' }}
            ticks={[0, 35, 50, 65, 80, 100]}
          />

          {/* Rating threshold lines */}
          <ReferenceLine y={80} stroke="#10b981" strokeDasharray="3 3" strokeOpacity={0.5} />
          <ReferenceLine y={65} stroke="#84cc16" strokeDasharray="3 3" strokeOpacity={0.5} />
          <ReferenceLine y={50} stroke="#eab308" strokeDasharray="3 3" strokeOpacity={0.5} />
          <ReferenceLine y={35} stroke="#f97316" strokeDasharray="3 3" strokeOpacity={0.5} />

          {/* Score area fill */}
          <Area
            type="monotone"
            dataKey="overall_score"
            stroke="transparent"
            fill="url(#scoreGradient)"
          />

          {/* Main score line */}
          <Line
            type="monotone"
            dataKey="overall_score"
            stroke="#2563eb"
            strokeWidth={2}
            dot={false}
            activeDot={{ r: 6, fill: '#2563eb' }}
          />

          {/* Category lines */}
          {showCategories && (
            <>
              <Line
                type="monotone"
                dataKey="quality_score"
                stroke="#10b981"
                strokeWidth={1}
                strokeDasharray="5 5"
                dot={false}
              />
              <Line
                type="monotone"
                dataKey="valuation_score"
                stroke="#f59e0b"
                strokeWidth={1}
                strokeDasharray="5 5"
                dot={false}
              />
              <Line
                type="monotone"
                dataKey="signals_score"
                stroke="#8b5cf6"
                strokeWidth={1}
                strokeDasharray="5 5"
                dot={false}
              />
            </>
          )}

          <Tooltip content={<CustomTooltip />} />
        </ComposedChart>
      </ResponsiveContainer>

      {/* Key events */}
      {events.length > 0 && (
        <div className="mt-4 space-y-2">
          <h4 className="text-sm font-medium text-gray-700">Key Events</h4>
          <div className="space-y-1">
            {events.slice(0, 5).map((event, i) => (
              <EventRow key={i} event={event} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function CustomTooltip({ active, payload, label }: any) {
  if (!active || !payload || !payload.length) {
    return null;
  }

  const data = payload[0].payload as ChartDataPoint;

  return (
    <div className="bg-white border border-gray-200 rounded-lg shadow-lg p-3 text-sm">
      <div className="font-medium text-gray-900 mb-2">{label}</div>
      <div className="space-y-1">
        <div className="flex items-center justify-between gap-4">
          <span className="text-gray-500">Overall</span>
          <span className="font-semibold text-blue-600">{Math.round(data.overall_score)}</span>
        </div>
        {data.quality_score !== undefined && (
          <div className="flex items-center justify-between gap-4">
            <span className="text-gray-500">Quality</span>
            <span className="text-green-600">{Math.round(data.quality_score)}</span>
          </div>
        )}
        {data.valuation_score !== undefined && (
          <div className="flex items-center justify-between gap-4">
            <span className="text-gray-500">Valuation</span>
            <span className="text-amber-600">{Math.round(data.valuation_score)}</span>
          </div>
        )}
        {data.signals_score !== undefined && (
          <div className="flex items-center justify-between gap-4">
            <span className="text-gray-500">Signals</span>
            <span className="text-purple-600">{Math.round(data.signals_score)}</span>
          </div>
        )}
      </div>
      {data.event && (
        <div className="mt-2 pt-2 border-t border-gray-100">
          <div className="flex items-center gap-2">
            <span>{data.event.icon || '📌'}</span>
            <span className="text-gray-700">{data.event.title}</span>
          </div>
        </div>
      )}
    </div>
  );
}

function LegendItem({ color, label, dashed = false }: { color: string; label: string; dashed?: boolean }) {
  return (
    <div className="flex items-center gap-1">
      <div
        className="w-4 h-0.5"
        style={{
          backgroundColor: color,
          borderStyle: dashed ? 'dashed' : 'solid',
        }}
      />
      <span className="text-gray-600">{label}</span>
    </div>
  );
}

function EventRow({ event }: { event: ScoreEvent }) {
  const icon = event.icon || CATALYST_ICONS[event.type] || '📌';
  const date = new Date(event.date).toLocaleDateString('en-US', { month: 'short', day: 'numeric' });

  return (
    <div className="flex items-center gap-3 text-sm">
      <span>{icon}</span>
      <span className="text-gray-500">{date}:</span>
      <span className="text-gray-700">{event.title}</span>
      {event.score_impact !== undefined && (
        <span className={event.score_impact > 0 ? 'text-green-600' : 'text-red-600'}>
          {event.score_impact > 0 ? '+' : ''}{event.score_impact.toFixed(1)} pts
        </span>
      )}
    </div>
  );
}

function ChartSkeleton({ height }: { height: number }) {
  return (
    <div
      className="bg-gray-50 rounded-lg animate-pulse flex items-center justify-center"
      style={{ height }}
    >
      <div className="w-3/4 h-1/2 bg-gray-200 rounded" />
    </div>
  );
}

function calculateCategoryScore(score: ICScoreData, category: 'quality' | 'valuation' | 'signals'): number | undefined {
  const categoryFactors: Record<string, (keyof ICScoreData)[]> = {
    quality: ['profitability_score', 'financial_health_score', 'growth_score'],
    valuation: ['value_score'],
    signals: ['momentum_score', 'technical_score', 'analyst_consensus_score', 'insider_activity_score', 'institutional_score'],
  };

  const factors = categoryFactors[category];
  const scores = factors
    .map((f) => score[f] as number | null)
    .filter((s): s is number => s !== null);

  if (scores.length === 0) return undefined;
  return scores.reduce((a, b) => a + b, 0) / scores.length;
}

/**
 * ICScoreSparkline - Mini chart for compact displays
 */
interface ICScoreSparklineProps {
  ticker: string;
  days?: number;
  width?: number;
  height?: number;
}

export function ICScoreSparkline({ ticker, days = 30, width = 100, height = 30 }: ICScoreSparklineProps) {
  const [history, setHistory] = useState<ICScoreData[]>([]);

  useEffect(() => {
    getICScoreHistory(ticker, days).then(setHistory);
  }, [ticker, days]);

  if (history.length < 2) {
    return <div style={{ width, height }} className="bg-gray-100 rounded" />;
  }

  const scores = history.map((h) => h.overall_score);
  const min = Math.min(...scores);
  const max = Math.max(...scores);
  const range = max - min || 1;

  const points = scores
    .map((score, i) => {
      const x = (i / (scores.length - 1)) * width;
      const y = height - ((score - min) / range) * height;
      return `${x},${y}`;
    })
    .join(' ');

  const trend = scores[scores.length - 1] - scores[0];
  const color = trend > 0 ? '#10b981' : trend < 0 ? '#ef4444' : '#6b7280';

  return (
    <svg width={width} height={height} className="overflow-visible">
      <polyline
        points={points}
        fill="none"
        stroke={color}
        strokeWidth={1.5}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
