/**
 * Sentiment API client
 *
 * Provides methods to interact with the social sentiment API endpoints.
 */

import { apiClient } from './client';
import { sentiment } from './routes';
import type {
  SentimentResponse,
  SentimentHistoryResponse,
  TrendingResponse,
  RepresentativePostsResponse,
  TrendingPeriod,
  PostSortOption,
} from '@/lib/types/sentiment';

/**
 * Get sentiment analysis for a specific ticker
 * @param ticker - Stock ticker symbol
 * @returns Sentiment data including score, breakdown, and top subreddits
 */
export async function getSentiment(ticker: string): Promise<SentimentResponse> {
  return apiClient.get<SentimentResponse>(sentiment.byTicker(ticker.toUpperCase()));
}

/**
 * Get historical sentiment data for a ticker
 * @param ticker - Stock ticker symbol
 * @param days - Number of days of history (7, 30, or 90)
 * @returns Array of daily sentiment data points
 */
export async function getSentimentHistory(
  ticker: string,
  days: number = 7
): Promise<SentimentHistoryResponse> {
  return apiClient.get<SentimentHistoryResponse>(
    `${sentiment.history(ticker.toUpperCase())}?days=${days}`
  );
}

/**
 * Get trending tickers by social media activity
 * @param period - Time period ('24h' or '7d')
 * @param limit - Maximum number of results
 * @returns List of trending tickers with sentiment scores
 */
export async function getTrendingSentiment(
  period: TrendingPeriod = '24h',
  limit: number = 20
): Promise<TrendingResponse> {
  return apiClient.get<TrendingResponse>(`${sentiment.trending}?period=${period}&limit=${limit}`);
}

/**
 * Get representative social media posts for a ticker
 * @param ticker - Stock ticker symbol
 * @param sort - Sort order (recent, engagement, bullish, bearish)
 * @param limit - Maximum number of posts
 * @returns List of curated posts
 */
export async function getSentimentPosts(
  ticker: string,
  sort: PostSortOption = 'recent',
  limit: number = 10
): Promise<RepresentativePostsResponse> {
  return apiClient.get<RepresentativePostsResponse>(
    `${sentiment.posts(ticker.toUpperCase())}?sort=${sort}&limit=${limit}`
  );
}
