'use client';

import SentimentCard from '@/components/sentiment/SentimentCard';
import SentimentHistoryChart from '@/components/sentiment/SentimentHistoryChart';
import PostsList from '@/components/sentiment/PostsList';

interface SentimentTabProps {
  symbol: string;
}

/**
 * Full sentiment analysis tab for the ticker detail page.
 * Shows sentiment card (which includes subreddit distribution chart),
 * history chart with price overlay, and posts feed.
 *
 * Note: SentimentCard (full variant) already renders SubredditDistributionChart
 * internally and fetches its own data, so we don't duplicate either here.
 */
export default function SentimentTab({ symbol }: SentimentTabProps) {
  return (
    <div className="space-y-6 p-6">
      {/* Sentiment Overview (includes gauge, breakdown, subreddit chart) */}
      <SentimentCard ticker={symbol} variant="full" />

      {/* Sentiment History Chart */}
      <div className="bg-ic-surface rounded-lg border border-ic-border p-6">
        <SentimentHistoryChart
          ticker={symbol}
          initialDays={30}
          height={350}
          showPostCount={true}
          showPriceOverlay={true}
        />
      </div>

      {/* Social Media Posts */}
      <PostsList ticker={symbol} limit={10} />
    </div>
  );
}
