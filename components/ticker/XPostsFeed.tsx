'use client';

import { useState, useEffect } from 'react';
import { getXPosts, type XPost, type XPostsResponse } from '@/lib/api/x';
import { formatRelativeTime } from '@/lib/utils';

interface XPostsFeedProps {
  ticker: string;
}

/** Format large numbers compactly: 1200 -> "1.2K", 1500000 -> "1.5M" */
function formatCompactNumber(n: number | null): string {
  if (n === null || n === undefined) return '';
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1).replace(/\.0$/, '')}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1).replace(/\.0$/, '')}K`;
  return n.toString();
}

/** X logo SVG (simple version) */
function XLogo({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      className={className}
      fill="currentColor"
      aria-hidden="true"
    >
      <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z" />
    </svg>
  );
}

export default function XPostsFeed({ ticker }: XPostsFeedProps) {
  const [data, setData] = useState<XPostsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchPosts() {
      try {
        setLoading(true);
        setError(null);
        const result = await getXPosts(ticker);
        setData(result);
      } catch (err) {
        console.error('Error fetching X posts:', err);
        setError(err instanceof Error ? err.message : 'Failed to load posts');
      } finally {
        setLoading(false);
      }
    }

    fetchPosts();
  }, [ticker]);

  // Loading state
  if (loading) {
    return <XPostsFeedSkeleton />;
  }

  // Don't render anything if no posts or error
  if (error || !data || data.posts.length === 0) {
    return null;
  }

  return (
    <div className="bg-ic-surface rounded-lg shadow border border-ic-border overflow-hidden">
      {/* Header */}
      <div className="px-4 py-3 border-b border-ic-border flex items-center justify-between">
        <div className="flex items-center gap-2">
          <XLogo className="h-4 w-4 text-ic-text-primary" />
          <h3 className="text-sm font-semibold text-ic-text-primary">Latest on X</h3>
        </div>
        <span className="text-xs text-ic-text-dim">
          {data.posts.length} post{data.posts.length !== 1 ? 's' : ''}
        </span>
      </div>

      {/* Posts */}
      <div className="divide-y divide-ic-border/50">
        {data.posts.map((post, index) => (
          <PostItem key={post.post_url || index} post={post} />
        ))}
      </div>

      {/* Footer */}
      {data.updated_at && (
        <div className="px-4 py-2 border-t border-ic-border bg-ic-bg-secondary/30">
          <p className="text-xs text-ic-text-dim">
            Updated {formatRelativeTime(data.updated_at)}
          </p>
        </div>
      )}
    </div>
  );
}

function PostItem({ post }: { post: XPost }) {
  const content = (
    <div className="px-4 py-3 hover:bg-ic-bg-secondary/40 transition-colors">
      {/* Author line */}
      <div className="flex items-center gap-1.5 mb-1">
        {post.author_name && (
          <span className="text-sm font-medium text-ic-text-primary truncate max-w-[120px]">
            {post.author_name}
          </span>
        )}
        {post.author_verified && (
          <svg className="h-3.5 w-3.5 text-blue-500 flex-shrink-0" viewBox="0 0 22 22" fill="currentColor">
            <path d="M20.396 11c-.018-.646-.215-1.275-.57-1.816-.354-.54-.852-.972-1.438-1.246.223-.607.27-1.264.14-1.897-.131-.634-.437-1.218-.882-1.687-.47-.445-1.053-.75-1.687-.882-.633-.13-1.29-.083-1.897.14-.273-.587-.704-1.086-1.245-1.44S11.647 1.62 11 1.604c-.646.017-1.273.213-1.813.568s-.969.855-1.24 1.44c-.608-.223-1.267-.272-1.902-.14-.635.13-1.22.436-1.69.882-.445.47-.749 1.055-.878 1.69-.13.633-.08 1.29.144 1.896-.587.274-1.087.705-1.443 1.245-.356.54-.555 1.17-.574 1.817.02.647.218 1.276.574 1.817.356.54.856.972 1.443 1.245-.224.606-.274 1.263-.144 1.896.13.636.433 1.221.878 1.69.47.446 1.055.752 1.69.883.635.13 1.294.083 1.902-.143.271.586.702 1.084 1.24 1.438.54.354 1.167.551 1.813.568.647-.016 1.276-.213 1.817-.567s.972-.854 1.245-1.44c.604.225 1.261.272 1.893.143.636-.131 1.22-.437 1.69-.883.445-.47.75-1.055.88-1.69.131-.634.084-1.292-.139-1.9.584-.272 1.084-.705 1.439-1.246.354-.54.551-1.17.569-1.816zM9.662 14.85l-3.429-3.428 1.293-1.302 2.072 2.072 4.4-4.794 1.347 1.246z" />
          </svg>
        )}
        {post.author_handle && (
          <span className="text-xs text-ic-text-dim truncate">
            @{post.author_handle.replace('@', '')}
          </span>
        )}
        {post.timestamp && (
          <>
            <span className="text-xs text-ic-text-dim">·</span>
            <span className="text-xs text-ic-text-dim flex-shrink-0">{post.timestamp}</span>
          </>
        )}
      </div>

      {/* Content */}
      <p className="text-sm text-ic-text-secondary line-clamp-2 mb-2">
        {post.content}
      </p>

      {/* Engagement row */}
      <div className="flex items-center gap-4 text-xs text-ic-text-dim">
        {post.replies !== null && (
          <span className="flex items-center gap-1">
            <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 20.25c4.97 0 9-3.694 9-8.25s-4.03-8.25-9-8.25S3 7.444 3 12c0 2.104.859 4.023 2.273 5.48.432.447.74 1.04.586 1.641a4.483 4.483 0 01-.923 1.785A5.969 5.969 0 006 21c1.282 0 2.47-.402 3.445-1.087.81.22 1.668.337 2.555.337z" />
            </svg>
            {formatCompactNumber(post.replies)}
          </span>
        )}
        {post.reposts !== null && (
          <span className="flex items-center gap-1">
            <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M19.5 12c0-1.232-.046-2.453-.138-3.662a4.006 4.006 0 00-3.7-3.7 48.678 48.678 0 00-7.324 0 4.006 4.006 0 00-3.7 3.7c-.017.22-.032.441-.046.662M19.5 12l3-3m-3 3l-3-3m-12 3c0 1.232.046 2.453.138 3.662a4.006 4.006 0 003.7 3.7 48.656 48.656 0 007.324 0 4.006 4.006 0 003.7-3.7c.017-.22.032-.441.046-.662M4.5 12l3 3m-3-3l-3 3" />
            </svg>
            {formatCompactNumber(post.reposts)}
          </span>
        )}
        {post.likes !== null && (
          <span className="flex items-center gap-1">
            <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M21 8.25c0-2.485-2.099-4.5-4.688-4.5-1.935 0-3.597 1.126-4.312 2.733-.715-1.607-2.377-2.733-4.313-2.733C5.1 3.75 3 5.765 3 8.25c0 7.22 9 12 9 12s9-4.78 9-12z" />
            </svg>
            {formatCompactNumber(post.likes)}
          </span>
        )}
        {post.views !== null && (
          <span className="flex items-center gap-1">
            <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M2.036 12.322a1.012 1.012 0 010-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178z" />
              <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
            {formatCompactNumber(post.views)}
          </span>
        )}
      </div>
    </div>
  );

  if (post.post_url) {
    return (
      <a
        href={post.post_url}
        target="_blank"
        rel="noopener noreferrer"
        className="block"
      >
        {content}
      </a>
    );
  }

  return content;
}

/** Skeleton matching the card shape */
export function XPostsFeedSkeleton() {
  return (
    <div className="bg-ic-surface rounded-lg shadow border border-ic-border overflow-hidden animate-pulse">
      {/* Header skeleton */}
      <div className="px-4 py-3 border-b border-ic-border flex items-center justify-between">
        <div className="flex items-center gap-2">
          <div className="h-4 w-4 bg-ic-bg-secondary rounded" />
          <div className="h-4 w-20 bg-ic-bg-secondary rounded" />
        </div>
        <div className="h-3 w-12 bg-ic-bg-secondary rounded" />
      </div>
      {/* Post skeletons */}
      {[1, 2, 3].map((i) => (
        <div key={i} className="px-4 py-3 border-b border-ic-border/50 last:border-b-0">
          <div className="flex items-center gap-2 mb-2">
            <div className="h-3.5 w-20 bg-ic-bg-secondary rounded" />
            <div className="h-3.5 w-16 bg-ic-bg-secondary rounded" />
          </div>
          <div className="h-4 w-full bg-ic-bg-secondary rounded mb-1" />
          <div className="h-4 w-3/4 bg-ic-bg-secondary rounded mb-2" />
          <div className="flex gap-4">
            <div className="h-3 w-8 bg-ic-bg-secondary rounded" />
            <div className="h-3 w-8 bg-ic-bg-secondary rounded" />
            <div className="h-3 w-8 bg-ic-bg-secondary rounded" />
          </div>
        </div>
      ))}
    </div>
  );
}
