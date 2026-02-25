'use client';

import { useState, useEffect } from 'react';
import SentimentCard from '@/components/sentiment/SentimentCard';
import SentimentHistoryChart from '@/components/sentiment/SentimentHistoryChart';
import SubredditDistributionChart from '@/components/sentiment/SubredditDistributionChart';
import PostsList from '@/components/sentiment/PostsList';
import { getSentiment } from '@/lib/api/sentiment';
import type { SentimentResponse } from '@/lib/types/sentiment';

interface SentimentTabProps {
  symbol: string;
}

/**
 * Full sentiment analysis tab for the ticker detail page.
 * Shows sentiment card, history chart, subreddit distribution, and posts feed.
 */
export default function SentimentTab({ symbol }: SentimentTabProps) {
  const [sentimentData, setSentimentData] = useState<SentimentResponse | null>(null);

  useEffect(() => {
    async function fetchSentiment() {
      try {
        const result = await getSentiment(symbol);
        setSentimentData(result);
      } catch {
        // SentimentCard handles its own error state; we just need subreddit data
      }
    }
    fetchSentiment();
  }, [symbol]);

  return (
    <div className="space-y-6 p-6">
      {/* Sentiment Overview */}
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

      {/* Subreddit Distribution (standalone section) */}
      {sentimentData && sentimentData.top_subreddits.length > 0 && (
        <div className="bg-ic-surface rounded-lg border border-ic-border p-6">
          <h3 className="text-lg font-semibold text-ic-text-primary mb-4">
            Subreddit Distribution
          </h3>
          <SubredditDistributionChart subreddits={sentimentData.top_subreddits} height={250} />
        </div>
      )}

      {/* Social Media Posts */}
      <PostsList ticker={symbol} limit={10} />
    </div>
  );
}
