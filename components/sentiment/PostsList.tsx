'use client';

import { useState, useEffect } from 'react';
import { getSentimentPosts } from '@/lib/api/sentiment';
import {
  formatRelativeTime,
  getSentimentLabelColor,
  formatCompactNumber,
} from '@/lib/types/sentiment';
import type {
  RepresentativePostsResponse,
  RepresentativePost,
  PostSortOption,
} from '@/lib/types/sentiment';

interface PostsListProps {
  ticker: string;
  initialSort?: PostSortOption;
  limit?: number;
}

/**
 * Sortable list of representative social media posts
 */
export default function PostsList({
  ticker,
  initialSort = 'recent',
  limit = 10,
}: PostsListProps) {
  const [data, setData] = useState<RepresentativePostsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [sort, setSort] = useState<PostSortOption>(initialSort);

  useEffect(() => {
    async function fetchPosts() {
      try {
        setLoading(true);
        setError(null);
        const result = await getSentimentPosts(ticker, sort, limit);
        setData(result);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load posts');
      } finally {
        setLoading(false);
      }
    }

    fetchPosts();
  }, [ticker, sort, limit]);

  const sortOptions: { value: PostSortOption; label: string }[] = [
    { value: 'recent', label: 'Recent' },
    { value: 'engagement', label: 'Most Engaged' },
    { value: 'bullish', label: 'Bullish' },
    { value: 'bearish', label: 'Bearish' },
  ];

  return (
    <div className="bg-ic-surface rounded-lg shadow-sm overflow-hidden">
      {/* Header */}
      <div className="px-4 py-4 border-b border-ic-border-subtle">
        <div className="flex items-center justify-between flex-wrap gap-3">
          <h3 className="text-lg font-semibold text-ic-text-primary">
            Social Media Posts
          </h3>
          <div className="flex items-center gap-2">
            <label className="text-sm text-ic-text-dim">Sort:</label>
            <select
              value={sort}
              onChange={(e) => setSort(e.target.value as PostSortOption)}
              className="text-sm border border-ic-border rounded-md px-2 py-1 focus:ring-blue-500 focus:border-blue-500"
            >
              {sortOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>
        </div>
      </div>

      {/* Loading state */}
      {loading && (
        <div className="p-8 text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-ic-blue mx-auto" />
          <p className="mt-4 text-ic-text-dim text-sm">Loading posts...</p>
        </div>
      )}

      {/* Error state */}
      {error && !loading && (
        <div className="p-8 text-center">
          <p className="text-red-600">{error}</p>
        </div>
      )}

      {/* Empty state */}
      {!loading && !error && (!data || data.posts.length === 0) && (
        <div className="p-12 text-center">
          <div className="text-ic-text-muted text-4xl mb-4">💬</div>
          <p className="text-ic-text-dim">No posts found for {ticker}</p>
        </div>
      )}

      {/* Posts list */}
      {!loading && !error && data && data.posts.length > 0 && (
        <div className="divide-y divide-ic-border-subtle">
          {data.posts.map((post) => (
            <PostCard key={post.id} post={post} />
          ))}
        </div>
      )}

      {/* Footer */}
      {data && data.total > 0 && (
        <div className="bg-ic-surface border-t border-ic-border-subtle px-4 py-3 text-sm text-ic-text-dim">
          Showing {data.posts.length} of {data.total} posts
        </div>
      )}
    </div>
  );
}

/**
 * Individual post card
 */
interface PostCardProps {
  post: RepresentativePost;
}

function PostCard({ post }: PostCardProps) {
  return (
    <div className="p-4 hover:bg-ic-surface-hover transition-colors">
      {/* Post header */}
      <div className="flex items-start justify-between gap-3 mb-2">
        <div className="flex items-center gap-2 flex-wrap">
          {/* Subreddit badge */}
          <span className="px-2 py-0.5 text-xs font-medium bg-orange-100 text-orange-700 rounded-full">
            r/{post.subreddit}
          </span>

          {/* Sentiment badge */}
          <span
            className={`px-2 py-0.5 text-xs font-medium rounded-full border ${getSentimentLabelColor(post.sentiment)}`}
          >
            {post.sentiment}
          </span>
        </div>

        {/* Time */}
        <span className="text-xs text-ic-text-muted whitespace-nowrap">
          {formatRelativeTime(post.posted_at)}
        </span>
      </div>

      {/* Title */}
      <a
        href={post.url}
        target="_blank"
        rel="noopener noreferrer"
        className="block text-ic-text-primary font-medium hover:text-ic-blue transition-colors mb-2"
      >
        {post.title}
      </a>

      {/* Engagement stats */}
      <div className="flex items-center gap-4 text-sm text-ic-text-dim">
        <span className="flex items-center gap-1">
          <UpvoteIcon className="w-4 h-4" />
          {formatCompactNumber(post.upvotes)}
        </span>
        <span className="flex items-center gap-1">
          <CommentIcon className="w-4 h-4" />
          {formatCompactNumber(post.comment_count)}
        </span>
        <a
          href={post.url}
          target="_blank"
          rel="noopener noreferrer"
          className="text-ic-blue hover:underline ml-auto"
        >
          View on Reddit
        </a>
      </div>
    </div>
  );
}

/**
 * Compact post list for widgets
 */
interface CompactPostsListProps {
  posts: RepresentativePost[];
  limit?: number;
}

export function CompactPostsList({ posts, limit = 3 }: CompactPostsListProps) {
  const displayPosts = posts.slice(0, limit);

  return (
    <div className="space-y-3">
      {displayPosts.map((post) => (
        <a
          key={post.id}
          href={post.url}
          target="_blank"
          rel="noopener noreferrer"
          className="block p-3 rounded-lg border border-ic-border-subtle hover:border-ic-border-subtle hover:bg-ic-surface-hover transition-colors"
        >
          <div className="flex items-center gap-2 mb-1">
            <span className="text-xs text-orange-600">r/{post.subreddit}</span>
            <span
              className={`text-xs px-1.5 py-0.5 rounded ${getSentimentLabelColor(post.sentiment)}`}
            >
              {post.sentiment}
            </span>
          </div>
          <p className="text-sm text-ic-text-primary line-clamp-2">{post.title}</p>
          <div className="flex items-center gap-3 mt-2 text-xs text-ic-text-dim">
            <span>{formatCompactNumber(post.upvotes)} upvotes</span>
            <span>{formatRelativeTime(post.posted_at)}</span>
          </div>
        </a>
      ))}
    </div>
  );
}

/**
 * Upvote icon
 */
function UpvoteIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      fill="none"
      stroke="currentColor"
      viewBox="0 0 24 24"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M5 15l7-7 7 7"
      />
    </svg>
  );
}

/**
 * Comment icon
 */
function CommentIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      fill="none"
      stroke="currentColor"
      viewBox="0 0 24 24"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"
      />
    </svg>
  );
}
