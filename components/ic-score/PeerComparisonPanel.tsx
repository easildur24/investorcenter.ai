'use client';

import Link from 'next/link';
import { PeerStock, PeerComparison, getScoreColorClass } from '@/lib/types/ic-score-v2';
import { DeltaBadge, SectorRankBadge } from './Badges';

interface PeerComparisonPanelProps {
  ticker: string;
  currentScore: number;
  peerComparison: PeerComparison | undefined;
  peers: PeerStock[] | undefined;
}

/**
 * PeerComparisonPanel - Displays peer comparison for a stock
 *
 * Shows:
 * - List of similar stocks with their IC Scores
 * - Average peer score and delta vs current stock
 * - Sector rank information
 */
export default function PeerComparisonPanel({
  ticker,
  currentScore,
  peerComparison,
  peers,
}: PeerComparisonPanelProps) {
  // Use either the dedicated peerComparison object or fall back to peers array
  const peerList = peerComparison?.peers || peers || [];

  if (peerList.length === 0) {
    return (
      <div className="border-t border-gray-200 p-4">
        <h3 className="font-medium text-gray-900 mb-3">Peer Comparison</h3>
        <p className="text-sm text-gray-500">No peer data available for this stock.</p>
      </div>
    );
  }

  const avgPeerScore = peerComparison?.avg_peer_score ?? calculateAvgScore(peerList);
  const vsPeersDelta = avgPeerScore ? currentScore - avgPeerScore : null;

  return (
    <div className="border-t border-gray-200 p-4">
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-medium text-gray-900">Peer Comparison</h3>
        {peerComparison?.sector_rank && peerComparison?.sector_total && (
          <SectorRankBadge
            rank={peerComparison.sector_rank}
            total={peerComparison.sector_total}
            size="sm"
          />
        )}
      </div>

      {/* Peer list */}
      <div className="space-y-3">
        {peerList.map((peer) => (
          <PeerRow key={peer.ticker} peer={peer} currentScore={currentScore} />
        ))}
      </div>

      {/* Summary footer */}
      {avgPeerScore !== null && (
        <div className="mt-4 pt-4 border-t border-gray-100">
          <div className="flex items-center justify-between text-sm">
            <span className="text-gray-500">Average Peer Score</span>
            <div className="flex items-center gap-2">
              <span className="font-medium">{Math.round(avgPeerScore)}</span>
              {vsPeersDelta !== null && (
                <span className="text-gray-400">
                  ({ticker} is{' '}
                  <span
                    className={
                      vsPeersDelta > 0
                        ? 'text-green-600'
                        : vsPeersDelta < 0
                          ? 'text-red-600'
                          : 'text-gray-500'
                    }
                  >
                    {vsPeersDelta > 0 ? '+' : ''}
                    {Math.round(vsPeersDelta)} pts
                  </span>{' '}
                  vs peers)
                </span>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

interface PeerRowProps {
  peer: PeerStock;
  currentScore: number;
}

function PeerRow({ peer, currentScore }: PeerRowProps) {
  const delta = peer.ic_score !== null ? currentScore - peer.ic_score : null;
  const scoreColor = getScoreColorClass(peer.ic_score);

  return (
    <div className="flex items-center justify-between py-2 hover:bg-gray-50 rounded px-2 -mx-2">
      <div className="flex items-center gap-3">
        <Link
          href={`/ticker/${peer.ticker}`}
          className="font-mono font-medium text-blue-600 hover:text-blue-800 hover:underline"
        >
          {peer.ticker}
        </Link>
        {peer.company_name && (
          <span className="text-sm text-gray-500 truncate max-w-[150px]">{peer.company_name}</span>
        )}
      </div>

      <div className="flex items-center gap-3">
        {peer.ic_score !== null ? (
          <>
            <span className={`text-lg font-semibold ${scoreColor}`}>
              {Math.round(peer.ic_score)}
            </span>
            {delta !== null && <DeltaBadge delta={delta} size="sm" />}
          </>
        ) : (
          <span className="text-sm text-gray-400">N/A</span>
        )}
      </div>
    </div>
  );
}

function calculateAvgScore(peers: PeerStock[]): number | null {
  const validScores = peers.filter((p) => p.ic_score !== null).map((p) => p.ic_score!);
  if (validScores.length === 0) return null;
  return validScores.reduce((a, b) => a + b, 0) / validScores.length;
}

/**
 * PeerComparisonCompact - Smaller version for compact displays
 */
interface PeerComparisonCompactProps {
  avgPeerScore: number | null;
  vsPeersDelta: number | null;
  sectorRank?: number | null;
  sectorTotal?: number | null;
}

export function PeerComparisonCompact({
  avgPeerScore,
  vsPeersDelta,
  sectorRank,
  sectorTotal,
}: PeerComparisonCompactProps) {
  return (
    <div className="flex items-center gap-4 text-sm">
      {avgPeerScore !== null && (
        <div className="flex items-center gap-1">
          <span className="text-gray-500">vs Peers:</span>
          {vsPeersDelta !== null && (
            <span
              className={
                vsPeersDelta > 0
                  ? 'text-green-600 font-medium'
                  : vsPeersDelta < 0
                    ? 'text-red-600 font-medium'
                    : 'text-gray-500'
              }
            >
              {vsPeersDelta > 0 ? '+' : ''}
              {Math.round(vsPeersDelta)} pts
            </span>
          )}
        </div>
      )}
      {sectorRank && sectorTotal && (
        <div className="flex items-center gap-1">
          <span className="text-gray-500">Sector:</span>
          <span className="font-medium">
            #{sectorRank}/{sectorTotal}
          </span>
        </div>
      )}
    </div>
  );
}
