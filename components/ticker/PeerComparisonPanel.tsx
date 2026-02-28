'use client';

import { useState } from 'react';
import Link from 'next/link';
import { cn } from '@/lib/utils';
import { useAuth } from '@/lib/auth/AuthContext';
import { usePeerComparison } from '@/lib/hooks/usePeerComparison';
import { Tooltip } from '@/components/ui/Tooltip';
import type { Peer, PeersResponse } from '@/lib/types/fundamentals';

// ── Format helpers ────────────────────────────────────────────────────────────

function formatPrice(n: number): string {
  if (n == null || isNaN(n)) return 'N/A';
  return `$${n.toFixed(2)}`;
}

function formatPercent(n: number): string {
  if (n == null || isNaN(n)) return 'N/A';
  const sign = n >= 0 ? '+' : '';
  return `${sign}${n.toFixed(1)}%`;
}

function formatLargeNumber(n: number): string {
  if (n == null || isNaN(n)) return 'N/A';
  if (n >= 1e12) return `$${(n / 1e12).toFixed(1)}T`;
  if (n >= 1e9) return `$${(n / 1e9).toFixed(1)}B`;
  if (n >= 1e6) return `$${(n / 1e6).toFixed(1)}M`;
  return `$${n.toFixed(2)}`;
}

function formatMetricValue(key: string, value: number): string {
  switch (key) {
    case 'ic_score':
      return value.toFixed(0);
    case 'pe_ratio':
      return value.toFixed(1);
    case 'roe':
    case 'revenue_growth_yoy':
    case 'net_margin':
      return formatPercent(value);
    case 'debt_to_equity':
      return value.toFixed(2);
    case 'market_cap':
      return formatLargeNumber(value);
    default:
      return value.toFixed(2);
  }
}

// ── Metric config ─────────────────────────────────────────────────────────────

interface MetricDef {
  key: string;
  label: string;
  higherIsBetter: boolean;
}

const METRICS: MetricDef[] = [
  { key: 'ic_score', label: 'IC Score', higherIsBetter: true },
  { key: 'pe_ratio', label: 'P/E', higherIsBetter: false },
  { key: 'roe', label: 'ROE', higherIsBetter: true },
  { key: 'revenue_growth_yoy', label: 'Rev Growth', higherIsBetter: true },
  { key: 'net_margin', label: 'Net Margin', higherIsBetter: true },
  { key: 'debt_to_equity', label: 'D/E', higherIsBetter: false },
  { key: 'market_cap', label: 'Market Cap', higherIsBetter: true },
];

// ── Helpers to extract metric values ──────────────────────────────────────────

function getStockMetricValue(data: PeersResponse, metricKey: string): number | null {
  if (metricKey === 'ic_score') return data.ic_score;
  return data.stock_metrics[metricKey] ?? null;
}

function getPeerMetricValue(peer: Peer, metricKey: string): number | null {
  if (metricKey === 'ic_score') return peer.ic_score;
  return (peer.metrics as Record<string, number>)[metricKey] ?? null;
}

// Determine best & worst indices for a row (stock at index 0, peers at 1..N)
function getBestWorstIndices(
  values: (number | null)[],
  higherIsBetter: boolean
): { bestIdx: number; worstIdx: number } {
  let bestIdx = -1;
  let worstIdx = -1;
  let bestVal = higherIsBetter ? -Infinity : Infinity;
  let worstVal = higherIsBetter ? Infinity : -Infinity;

  values.forEach((v, i) => {
    if (v == null) return;
    if (higherIsBetter) {
      if (v > bestVal) {
        bestVal = v;
        bestIdx = i;
      }
      if (v < worstVal) {
        worstVal = v;
        worstIdx = i;
      }
    } else {
      if (v < bestVal) {
        bestVal = v;
        bestIdx = i;
      }
      if (v > worstVal) {
        worstVal = v;
        worstIdx = i;
      }
    }
  });

  return { bestIdx, worstIdx };
}

// ── Skeleton ──────────────────────────────────────────────────────────────────

function PeerComparisonSkeleton() {
  return (
    <div className="animate-pulse space-y-3 p-4">
      <div className="h-5 bg-ic-border rounded w-40" />
      <div className="overflow-x-auto">
        <div className="min-w-[600px]">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="flex gap-2 py-2">
              <div className="h-4 bg-ic-border rounded w-20" />
              {Array.from({ length: 6 }).map((_, j) => (
                <div key={j} className="h-4 bg-ic-border rounded w-16 flex-1" />
              ))}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// ── Mobile card ───────────────────────────────────────────────────────────────

function PeerCard({
  peer,
  data,
  isStock,
}: {
  peer: Peer | null;
  data: PeersResponse;
  isStock: boolean;
}) {
  const ticker = isStock ? data.ticker : peer!.ticker;
  const name = isStock ? data.ticker : peer!.company_name;

  return (
    <div
      className={cn(
        'min-w-[200px] rounded-lg border p-3 flex-shrink-0 snap-start',
        isStock ? 'border-ic-blue/40 bg-ic-blue/5' : 'border-ic-border bg-ic-bg-secondary'
      )}
    >
      <div className="flex items-center justify-between mb-3">
        {isStock ? (
          <span className="text-sm font-bold text-ic-blue">{ticker}</span>
        ) : (
          <Link
            href={`/ticker/${ticker}`}
            className="text-sm font-bold text-ic-text-primary hover:text-ic-blue transition-colors"
          >
            {ticker}
          </Link>
        )}
        {!isStock && peer && (
          <span className="text-[10px] text-ic-text-dim">
            {(peer.similarity_score * 100).toFixed(0)}% match
          </span>
        )}
      </div>
      <div className="space-y-2">
        {METRICS.map((m) => {
          const value = isStock
            ? getStockMetricValue(data, m.key)
            : peer
              ? getPeerMetricValue(peer, m.key)
              : null;
          return (
            <div key={m.key} className="flex justify-between text-xs">
              <span className="text-ic-text-dim">{m.label}</span>
              <span className="text-ic-text-primary font-medium">
                {value != null ? formatMetricValue(m.key, value) : 'N/A'}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

// ── Main component ────────────────────────────────────────────────────────────

interface PeerComparisonPanelProps {
  ticker: string;
  isPremium?: boolean;
}

export default function PeerComparisonPanel({
  ticker,
  isPremium: isPremiumProp,
}: PeerComparisonPanelProps) {
  const { user } = useAuth();
  const isPremium = isPremiumProp ?? user?.is_premium ?? false;
  const [expanded, setExpanded] = useState(false);
  const {
    data,
    loading,
    error,
    fetch: fetchPeers,
  } = usePeerComparison({
    ticker,
    enabled: expanded,
  });

  const handleToggle = () => {
    setExpanded((prev) => !prev);
    if (!expanded) {
      fetchPeers();
    }
  };

  const peerCount = data?.peers?.length ?? 5;
  const peerNames = data?.peers?.map((p) => p.ticker).join(', ') ?? 'Loading...';

  return (
    <div className="bg-ic-surface rounded-lg shadow border border-ic-border overflow-hidden">
      {/* Collapsed header — always visible */}
      <button
        onClick={handleToggle}
        className="w-full flex items-center justify-between p-4 hover:bg-ic-bg-secondary/50 transition-colors text-left"
      >
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-semibold text-ic-text-primary">
              Compare to {peerCount} similar companies
            </h3>
            {data && (
              <span className="text-xs text-ic-text-dim px-2 py-0.5 bg-ic-bg-secondary rounded-full">
                Avg IC Score: {data.avg_peer_score.toFixed(0)}
              </span>
            )}
          </div>
          {data && <p className="text-xs text-ic-text-dim mt-1 truncate">{peerNames}</p>}
        </div>
        <svg
          className={cn(
            'w-5 h-5 text-ic-text-dim transition-transform flex-shrink-0 ml-2',
            expanded && 'rotate-180'
          )}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {/* Expanded content */}
      {expanded && (
        <div className="border-t border-ic-border">
          {loading && <PeerComparisonSkeleton />}

          {error && <div className="p-4 text-sm text-ic-negative">{error}</div>}

          {!loading && !error && data && (
            <div className="relative">
              {/* Free tier overlay */}
              {!isPremium && (
                <div className="absolute inset-0 z-10 flex items-center justify-center bg-ic-bg-primary/60 backdrop-blur-sm rounded-b-lg">
                  <div className="text-center p-6">
                    <svg
                      className="w-8 h-8 mx-auto mb-2 text-ic-text-dim"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
                      />
                    </svg>
                    <p className="text-sm font-medium text-ic-text-primary mb-1">
                      Upgrade to compare
                    </p>
                    <p className="text-xs text-ic-text-dim mb-3">
                      Unlock side-by-side peer comparisons
                    </p>
                    <Link
                      href="/pricing"
                      className="inline-block text-sm font-medium px-4 py-2 bg-ic-blue text-white rounded-lg hover:bg-ic-blue/90 transition-colors"
                    >
                      See Plans
                    </Link>
                  </div>
                </div>
              )}

              {/* Desktop table */}
              <div
                className={cn(
                  'hidden md:block overflow-x-auto',
                  !isPremium && 'blur-[6px] select-none pointer-events-none'
                )}
              >
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-ic-border">
                      <th className="text-left text-xs font-medium text-ic-text-dim p-3 sticky left-0 bg-ic-surface z-[1]">
                        Metric
                      </th>
                      <th className="text-center text-xs font-bold p-3 bg-ic-blue/5 text-ic-blue">
                        {data.ticker}
                      </th>
                      {data.peers.map((peer) => (
                        <th key={peer.ticker} className="text-center text-xs font-medium p-3">
                          <Link
                            href={`/ticker/${peer.ticker}`}
                            className="text-ic-text-primary hover:text-ic-blue transition-colors"
                          >
                            {peer.ticker}
                          </Link>
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {METRICS.map((metric) => {
                      const stockVal = getStockMetricValue(data, metric.key);
                      const peerVals = data.peers.map((p) => getPeerMetricValue(p, metric.key));
                      const allVals = [stockVal, ...peerVals];
                      const { bestIdx, worstIdx } = getBestWorstIndices(
                        allVals,
                        metric.higherIsBetter
                      );

                      return (
                        <tr
                          key={metric.key}
                          className="border-b border-ic-border/50 hover:bg-ic-bg-secondary/30"
                        >
                          <td className="text-xs text-ic-text-dim p-3 sticky left-0 bg-ic-surface z-[1]">
                            {metric.label}
                          </td>
                          <td
                            className={cn(
                              'text-center p-3 font-medium bg-ic-blue/5',
                              bestIdx === 0 && 'text-green-400',
                              worstIdx === 0 && bestIdx !== 0 && 'text-red-400'
                            )}
                          >
                            {stockVal != null ? formatMetricValue(metric.key, stockVal) : 'N/A'}
                            {bestIdx === 0 && (
                              <span className="ml-1 text-green-400" aria-label="Best">
                                ★
                              </span>
                            )}
                          </td>
                          {data.peers.map((peer, i) => {
                            const val = peerVals[i];
                            const colIdx = i + 1;
                            return (
                              <td
                                key={peer.ticker}
                                className={cn(
                                  'text-center p-3 text-sm',
                                  bestIdx === colIdx && 'text-green-400 font-medium',
                                  worstIdx === colIdx && bestIdx !== colIdx && 'text-red-400'
                                )}
                              >
                                {val != null ? formatMetricValue(metric.key, val) : 'N/A'}
                                {bestIdx === colIdx && (
                                  <span className="ml-1 text-green-400" aria-label="Best">
                                    ★
                                  </span>
                                )}
                              </td>
                            );
                          })}
                        </tr>
                      );
                    })}

                    {/* Similarity row */}
                    <tr className="bg-ic-bg-secondary/20">
                      <td className="text-xs text-ic-text-dim p-3 sticky left-0 bg-ic-surface z-[1]">
                        Similarity
                      </td>
                      <td className="text-center p-3 text-xs text-ic-text-dim bg-ic-blue/5">--</td>
                      {data.peers.map((peer) => (
                        <td key={peer.ticker} className="text-center p-3 text-xs text-ic-text-dim">
                          {(peer.similarity_score * 100).toFixed(0)}%
                        </td>
                      ))}
                    </tr>
                  </tbody>
                </table>
              </div>

              {/* Mobile horizontal card scroll */}
              <div
                className={cn(
                  'md:hidden flex gap-3 overflow-x-auto snap-x snap-mandatory p-4 scrollbar-hide',
                  !isPremium && 'blur-[6px] select-none pointer-events-none'
                )}
              >
                <PeerCard peer={null} data={data} isStock />
                {data.peers.map((peer) => (
                  <PeerCard key={peer.ticker} peer={peer} data={data} isStock={false} />
                ))}
              </div>

              {/* Tooltip: How peers are selected */}
              <div className="p-3 border-t border-ic-border/50 flex items-center justify-end">
                <Tooltip
                  content={
                    <div className="space-y-1.5 text-xs">
                      <p className="font-medium text-ic-text-primary">How peers are selected</p>
                      <p className="text-ic-text-muted">
                        5 most similar companies based on a weighted score:
                      </p>
                      <ul className="list-disc pl-4 space-y-0.5 text-ic-text-muted">
                        <li>Market Cap (30%)</li>
                        <li>Revenue Growth (20%)</li>
                        <li>Net Margin (20%)</li>
                        <li>P/E Ratio (15%)</li>
                        <li>Beta (15%)</li>
                      </ul>
                    </div>
                  }
                  position="top"
                >
                  <span className="text-xs text-ic-text-dim hover:text-ic-text-muted cursor-help flex items-center gap-1">
                    <svg
                      className="w-3.5 h-3.5"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                      />
                    </svg>
                    How peers are selected
                  </span>
                </Tooltip>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
