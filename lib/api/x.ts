/**
 * X (Twitter) Posts API Client
 *
 * Fetches cached X/Twitter posts for a ticker from Redis via the backend.
 */

import { tickers } from './routes';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

/** A single X/Twitter post */
export interface XPost {
  author_handle: string | null;
  author_name: string | null;
  author_verified: boolean | null;
  content: string;
  timestamp: string | null;
  likes: number | null;
  reposts: number | null;
  replies: number | null;
  views: number | null;
  bookmarks: number | null;
  post_url: string | null;
  has_media: boolean | null;
  is_repost: boolean | null;
  is_reply: boolean | null;
}

/** Response from GET /api/v1/tickers/:symbol/x-posts */
export interface XPostsResponse {
  ticker: string;
  updated_at: string | null;
  posts: XPost[];
}

/**
 * Fetch latest X/Twitter posts for a ticker.
 * Returns top 5 posts cached in Redis (24h TTL).
 *
 * @param symbol Stock ticker symbol (e.g., "AAPL")
 * @returns Posts response or null on error
 */
export async function getXPosts(symbol: string): Promise<XPostsResponse | null> {
  try {
    const response = await fetch(`${API_BASE_URL}${tickers.xPosts(symbol.toUpperCase())}`, {
      method: 'GET',
      headers: { 'Content-Type': 'application/json' },
      cache: 'no-store',
    });

    if (!response.ok) {
      console.error(`Failed to fetch X posts for ${symbol}: ${response.status}`);
      return null;
    }

    const data: XPostsResponse = await response.json();
    return data;
  } catch (error) {
    console.error(`Error fetching X posts for ${symbol}:`, error);
    return null;
  }
}
