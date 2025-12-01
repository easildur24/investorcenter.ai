'use client';

import Link from 'next/link';
import { useState, useEffect } from 'react';

interface TrendingListItemProps {
  rank: number;
  symbol: string;
  companyName?: string;
  currentRank: number;
  rankChange?: number;
  mentions: number;
  upvotes?: number;
  popularityScore: number;
  trendDirection: string;
}

export default function TrendingListItem({
  rank,
  symbol,
  companyName,
  currentRank,
  rankChange = 0,
  mentions,
  upvotes = 0,
  popularityScore,
  trendDirection,
}: TrendingListItemProps) {
  const [showTooltip, setShowTooltip] = useState(false);

  const getRankChangeDisplay = () => {
    if (rankChange > 1) {
      return (
        <span className="text-green-600 font-medium">
          ↑{rankChange}
        </span>
      );
    }
    if (rankChange < -1) {
      return (
        <span className="text-red-600 font-medium">
          ↓{Math.abs(rankChange)}
        </span>
      );
    }
    return <span className="text-ic-text-muted">→</span>;
  };

  const getScoreEmoji = () => {
    if (popularityScore >= 90) return '🔥';
    if (popularityScore >= 70) return '📈';
    if (popularityScore >= 50) return '📊';
    return '💬';
  };

  const getScoreBadgeColor = () => {
    if (popularityScore >= 90) return 'bg-red-100 text-red-800';
    if (popularityScore >= 70) return 'bg-yellow-100 text-yellow-800';
    if (popularityScore >= 50) return 'bg-blue-100 text-blue-800';
    return 'bg-ic-surface text-gray-800';
  };

  const getProgressBarColor = () => {
    if (popularityScore >= 90) return 'bg-red-500';
    if (popularityScore >= 70) return 'bg-yellow-500';
    if (popularityScore >= 50) return 'bg-blue-500';
    return 'bg-gray-400';
  };

  return (
    <Link href={`/ticker/${symbol}`}>
      <div
        className="flex items-center justify-between p-4 hover:bg-gray-50 cursor-pointer border-b border-gray-200 transition-colors duration-150 group"
        onMouseEnter={() => setShowTooltip(true)}
        onMouseLeave={() => setShowTooltip(false)}
      >
        {/* Left Section: Rank, Ticker, Company */}
        <div className="flex items-center gap-4 flex-1 min-w-0">
          {/* Rank Number */}
          <span className="text-ic-text-dim font-mono text-sm w-8 flex-shrink-0">
            {rank}
          </span>

          {/* Ticker & Company */}
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <span className="font-bold text-gray-900 text-lg">{symbol}</span>
              <span className="text-xs px-2 py-0.5 rounded-full bg-ic-surface text-ic-text-muted">
                #{currentRank}
              </span>
              {getRankChangeDisplay()}
            </div>
            {companyName && (
              <div className="text-sm text-ic-text-muted truncate">{companyName}</div>
            )}
          </div>
        </div>

        {/* Right Section: Stats */}
        <div className="flex items-center gap-6 ml-4">
          {/* Mentions */}
          <div className="text-right hidden sm:block">
            <div className="text-sm font-semibold text-gray-900">{mentions.toLocaleString()}</div>
            <div className="text-xs text-ic-text-dim">mentions</div>
          </div>

          {/* Upvotes - Only show on larger screens */}
          {upvotes > 0 && (
            <div className="text-right hidden md:block">
              <div className="text-sm font-semibold text-gray-900">{upvotes.toLocaleString()}</div>
              <div className="text-xs text-ic-text-dim">upvotes</div>
            </div>
          )}

          {/* Popularity Score */}
          <div className="flex items-center gap-2">
            <span className="text-xl">{getScoreEmoji()}</span>
            <div className="text-right">
              <div className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getScoreBadgeColor()}`}>
                {popularityScore.toFixed(1)}
              </div>
              <div className="mt-1 w-20 bg-gray-200 rounded-full h-1.5">
                <div
                  className={`h-1.5 rounded-full ${getProgressBarColor()}`}
                  style={{ width: `${Math.min(popularityScore, 100)}%` }}
                ></div>
              </div>
            </div>
          </div>
        </div>

        {/* Hover Arrow */}
        <div className="ml-4 opacity-0 group-hover:opacity-100 transition-opacity">
          <svg className="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
        </div>
      </div>
    </Link>
  );
}
