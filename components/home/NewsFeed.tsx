'use client';

import { useState, useEffect } from 'react';
import { apiClient } from '@/lib/api';
import { useWidgetTracking } from '@/lib/hooks/useWidgetTracking';
import { formatRelativeTime } from '@/lib/utils';
import { NewspaperIcon } from '@heroicons/react/24/outline';

interface NewsArticle {
  id: number | string;
  title: string;
  summary: string;
  source: string;
  url: string;
  publishedAt: string;
  sentiment?: string;
  tickers?: string[];
  imageUrl?: string;
}

/**
 * NewsFeed — displays recent market news headlines.
 *
 * Fetches general market news from Polygon.io (no ticker filter)
 * to show macro-level, market-moving events rather than ticker-specific articles.
 */
export default function NewsFeed() {
  const { ref: widgetRef, trackInteraction } = useWidgetTracking('news_feed');
  const [articles, setArticles] = useState<NewsArticle[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let isMounted = true;

    const fetchNews = async () => {
      try {
        setLoading(true);
        // Fetch general market news (no ticker filter — returns macro/market-wide news)
        const response = await apiClient.getMarketNews(10);
        if (!isMounted) return;

        const newsArticles: NewsArticle[] = (response.data || []).map((article) => ({
          id: article.id || article.url,
          title: article.title,
          summary: article.summary || '',
          source: article.source,
          url: article.url,
          publishedAt: article.publishedAt,
          imageUrl: article.imageUrl,
        }));

        setArticles(newsArticles);
        setError(null);
      } catch (err) {
        if (!isMounted) return;
        setError(err instanceof Error ? err.message : 'Failed to fetch news');
        console.error('Error fetching news:', err);
      } finally {
        if (isMounted) setLoading(false);
      }
    };

    fetchNews();

    // Refresh every 5 minutes
    const interval = setInterval(fetchNews, 5 * 60 * 1000);

    return () => {
      isMounted = false;
      clearInterval(interval);
    };
  }, []);

  const isRecent = (publishedAt: string): boolean => {
    try {
      const published = new Date(publishedAt);
      const oneHourAgo = new Date(Date.now() - 60 * 60 * 1000);
      return published > oneHourAgo;
    } catch {
      return false;
    }
  };

  if (loading) {
    return (
      <div
        className="bg-ic-surface rounded-lg border border-ic-border p-6"
        style={{ boxShadow: 'var(--ic-shadow-card)' }}
      >
        <h2 className="text-lg font-semibold text-ic-text-primary mb-4">Market News</h2>
        <div className="animate-pulse space-y-4">
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className="space-y-2">
              <div className="h-4 bg-ic-bg-tertiary rounded w-full"></div>
              <div className="h-3 bg-ic-bg-tertiary rounded w-1/3"></div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div
        className="bg-ic-surface rounded-lg border border-ic-border p-6"
        style={{ boxShadow: 'var(--ic-shadow-card)' }}
      >
        <h2 className="text-lg font-semibold text-ic-text-primary mb-4">Market News</h2>
        <div className="text-ic-negative text-sm">
          <p>Error loading news: {error}</p>
          <p className="text-ic-text-muted mt-2">
            News will be available once the backend is running with market data access.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div
      ref={widgetRef}
      className="bg-ic-surface rounded-lg border border-ic-border p-6"
      style={{ boxShadow: 'var(--ic-shadow-card)' }}
    >
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-lg font-semibold text-ic-text-primary">Market News</h2>
        <NewspaperIcon className="h-5 w-5 text-ic-text-muted" />
      </div>

      <div className="space-y-3">
        {articles.length === 0 ? (
          <p className="text-ic-text-muted text-sm py-4 text-center">No news available</p>
        ) : (
          articles.map((article) => (
            <a
              key={article.id}
              href={article.url}
              target="_blank"
              rel="noopener noreferrer"
              className="block py-2.5 px-2 -mx-2 rounded-lg hover:bg-ic-surface-hover transition-colors group"
              onClick={() => trackInteraction('news_click', { source: article.source })}
            >
              <div className="flex items-start gap-2">
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-ic-text-primary group-hover:text-ic-blue transition-colors line-clamp-2 leading-snug">
                    {article.title}
                  </p>
                  <div className="flex items-center gap-2 mt-1.5">
                    <span className="text-xs text-ic-text-dim">{article.source}</span>
                    <span className="text-xs text-ic-text-dim">·</span>
                    <span className="text-xs text-ic-text-dim">
                      {formatRelativeTime(new Date(article.publishedAt))}
                    </span>
                    {isRecent(article.publishedAt) && (
                      <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-bold bg-red-500/10 text-red-500">
                        LIVE
                      </span>
                    )}
                  </div>
                </div>
              </div>
            </a>
          ))
        )}
      </div>
    </div>
  );
}
