/**
 * TypeScript type definitions for Social Media Sentiment feature
 *
 * Sentiment analysis of social media posts (Reddit, etc.) to gauge
 * market sentiment for stock tickers.
 */

/**
 * Sentiment breakdown percentages
 */
export interface SentimentBreakdown {
  bullish: number;  // 0-100 percentage
  bearish: number;  // 0-100 percentage
  neutral: number;  // 0-100 percentage
}

/**
 * Post count per subreddit
 */
export interface SubredditCount {
  subreddit: string;
  count: number;
}

/**
 * Sentiment label type
 */
export type SentimentLabel = 'bullish' | 'bearish' | 'neutral';

/**
 * GET /api/v1/sentiment/:ticker
 */
export interface SentimentResponse {
  ticker: string;
  company_name?: string;            // Company name from stocks table
  score: number;                    // -1 to +1
  label: SentimentLabel;
  breakdown: SentimentBreakdown;
  post_count_24h: number;
  post_count_7d: number;
  rank: number;                     // Current rank by activity
  rank_change: number;              // Change from previous period (+/- or 0)
  top_subreddits: SubredditCount[];
  last_updated: string;             // ISO 8601
}

/**
 * Historical sentiment data point
 */
export interface SentimentHistoryPoint {
  date: string;       // YYYY-MM-DD
  score: number;      // -1 to +1
  post_count: number;
  bullish: number;    // Count (not percentage)
  bearish: number;
  neutral: number;
}

/**
 * GET /api/v1/sentiment/:ticker/history?days=N
 */
export interface SentimentHistoryResponse {
  ticker: string;
  period: string;     // "7d", "30d", "90d"
  history: SentimentHistoryPoint[];
}

/**
 * Trending ticker entry
 */
export interface TrendingTicker {
  ticker: string;
  company_name?: string;  // Company name from stocks table
  score: number;          // -1 to +1
  label: SentimentLabel;
  post_count: number;
  mention_delta: number;  // % change from previous period
  rank: number;
}

/**
 * GET /api/v1/sentiment/trending?period=24h|7d&limit=N
 */
export interface TrendingResponse {
  period: string;         // "24h" or "7d"
  tickers: TrendingTicker[];
  updated_at: string;     // ISO 8601
}

/**
 * Representative social media post
 */
export interface RepresentativePost {
  id: number;
  title: string;
  body_preview?: string;            // Preview of post body
  url: string;
  source: string;                   // "reddit"
  subreddit: string;
  upvotes: number;
  comment_count: number;
  award_count: number;
  sentiment: SentimentLabel;
  sentiment_confidence?: number;    // 0 to 1
  flair?: string;
  posted_at: string;                // ISO 8601
}

/**
 * GET /api/v1/sentiment/:ticker/posts?sort=recent|engagement|bullish|bearish&limit=N
 */
export interface RepresentativePostsResponse {
  ticker: string;
  posts: RepresentativePost[];
  total: number;
  sort: string;                     // Sort option that was applied
}

/**
 * Sort options for posts
 */
export type PostSortOption = 'recent' | 'engagement' | 'bullish' | 'bearish';

/**
 * Period options for trending
 */
export type TrendingPeriod = '24h' | '7d';

/**
 * Helper function to get sentiment label color classes
 */
export function getSentimentLabelColor(label: SentimentLabel): string {
  switch (label) {
    case 'bullish':
      return 'text-green-600 bg-green-50 border-green-200';
    case 'bearish':
      return 'text-red-600 bg-red-50 border-red-200';
    case 'neutral':
      return 'text-ic-text-muted bg-ic-surface border-ic-border-subtle';
    default:
      return 'text-ic-text-muted bg-ic-surface border-ic-border-subtle';
  }
}

/**
 * Helper function to get sentiment score color
 */
export function getSentimentScoreColor(score: number): string {
  if (score >= 0.2) return '#22c55e';  // green-500
  if (score <= -0.2) return '#ef4444'; // red-500
  return '#6b7280'; // gray-500
}

/**
 * Helper function to get sentiment label from score
 */
export function getSentimentLabelFromScore(score: number): SentimentLabel {
  if (score >= 0.2) return 'bullish';
  if (score <= -0.2) return 'bearish';
  return 'neutral';
}

/**
 * Helper function to format sentiment score as percentage
 */
export function formatSentimentScore(score: number): string {
  const percentage = Math.abs(score * 100).toFixed(0);
  if (score >= 0.2) return `${percentage}% Bullish`;
  if (score <= -0.2) return `${percentage}% Bearish`;
  return 'Neutral';
}

/**
 * Helper function to format relative time
 */
export function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / (1000 * 60));
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays === 1) return 'Yesterday';
  return `${diffDays}d ago`;
}

/**
 * Helper function to format numbers with K/M suffix
 */
export function formatCompactNumber(num: number): string {
  if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`;
  if (num >= 1000) return `${(num / 1000).toFixed(1)}K`;
  return num.toString();
}

/**
 * Helper function to format percentage change with sign
 */
export function formatPercentageChange(delta: number): string {
  const sign = delta >= 0 ? '+' : '';
  return `${sign}${delta.toFixed(0)}%`;
}

/**
 * Helper function to format rank change indicator
 */
export function formatRankChange(change: number): { text: string; color: string } {
  if (change > 0) {
    return { text: `↑${change}`, color: 'text-green-600' };
  } else if (change < 0) {
    return { text: `↓${Math.abs(change)}`, color: 'text-red-600' };
  }
  return { text: '—', color: 'text-ic-text-muted' };
}
