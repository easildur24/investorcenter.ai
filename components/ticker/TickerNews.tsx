'use client';

import { useState, useEffect } from 'react';
import { NewspaperIcon } from '@heroicons/react/24/outline';

interface TickerNewsProps {
  symbol: string;
}

interface NewsArticle {
  // Raw Polygon API fields
  id?: string;
  title: string;
  description: string;
  author: string;
  article_url: string;
  published_utc: string;
  image_url?: string;
  keywords?: string[];
  tickers?: string[];
  publisher?: {
    name: string;
    homepage_url: string;
    logo_url: string;
    favicon_url: string;
  };
  insights?: Array<{
    ticker: string;
    sentiment: string;
    sentiment_reasoning: string;
  }>;
  // For backward compatibility with mock data
  summary?: string;
  source?: string;
  url?: string;
  publishedAt?: string;
  sentiment?: string;
}

export default function TickerNews({ symbol }: TickerNewsProps) {
  const [news, setNews] = useState<NewsArticle[]>([]);
  const [loading, setLoading] = useState(true);
  const [currentPage, setCurrentPage] = useState(1);
  const [totalArticles, setTotalArticles] = useState(0);
  const articlesPerPage = 5;

  useEffect(() => {
    const fetchNews = async () => {
      try {
        setLoading(true);
        const response = await fetch(`/api/v1/tickers/${symbol}/news`);
        const result = await response.json();
        if (result.data && Array.isArray(result.data)) {
          setNews(result.data);
          setTotalArticles(result.data.length);
        } else {
          setNews([]);
          setTotalArticles(0);
        }
      } catch (error) {
        console.error('Error fetching news:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchNews();
  }, [symbol]);

  const getSentimentColor = (sentiment: string | undefined) => {
    if (!sentiment) return 'text-ic-text-muted bg-ic-bg-secondary';

    switch (sentiment.toLowerCase()) {
      case 'positive':
        return 'text-ic-positive bg-ic-positive-bg';
      case 'negative':
        return 'text-ic-negative bg-ic-negative-bg';
      default:
        return 'text-ic-text-muted bg-ic-bg-secondary';
    }
  };

  const formatTimeAgo = (dateString: string) => {
    if (!dateString) return 'Unknown';

    const date = new Date(dateString);
    if (isNaN(date.getTime())) return 'Unknown';

    const now = new Date();
    const diffInMs = now.getTime() - date.getTime();
    const diffInHours = Math.floor(diffInMs / (1000 * 60 * 60));

    if (diffInHours < 1) return 'Just now';
    if (diffInHours < 24) return `${diffInHours}h ago`;
    const diffInDays = Math.floor(diffInHours / 24);
    if (diffInDays < 30) return `${diffInDays}d ago`;
    const diffInMonths = Math.floor(diffInDays / 30);
    if (diffInMonths < 12) return `${diffInMonths}mo ago`;
    const diffInYears = Math.floor(diffInDays / 365);
    return `${diffInYears}y ago`;
  };

  // Pagination logic
  const totalPages = Math.ceil(totalArticles / articlesPerPage);
  const startIndex = (currentPage - 1) * articlesPerPage;
  const endIndex = startIndex + articlesPerPage;
  const currentArticles = news.slice(startIndex, endIndex);


  const goToPage = (page: number) => {
    setCurrentPage(page);
    // Scroll to top of news section
    document.getElementById('news-section')?.scrollIntoView({ behavior: 'smooth' });
  };

  return (
    <div className="p-6" id="news-section">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center">
          <NewspaperIcon className="h-6 w-6 text-ic-text-dim mr-2" />
          <h3 className="text-lg font-semibold text-ic-text-primary">News & Analysis</h3>
          {totalArticles > 0 && (
            <span className="ml-3 text-sm text-ic-text-muted">
              {totalArticles} articles
            </span>
          )}
        </div>

        {/* Pagination Controls - Top Right */}
        {totalPages > 1 && (
          <div className="flex items-center space-x-2">
            <span className="text-sm text-ic-text-muted">Page {currentPage} of {totalPages}</span>
            <button
              onClick={() => goToPage(currentPage - 1)}
              disabled={currentPage === 1}
              className="px-3 py-1 text-sm font-medium text-ic-text-muted bg-ic-surface border border-ic-border rounded-md hover:bg-ic-surface-hover disabled:opacity-50 disabled:cursor-not-allowed"
            >
              ←
            </button>
            <button
              onClick={() => goToPage(currentPage + 1)}
              disabled={currentPage === totalPages}
              className="px-3 py-1 text-sm font-medium text-ic-text-muted bg-ic-surface border border-ic-border rounded-md hover:bg-ic-surface-hover disabled:opacity-50 disabled:cursor-not-allowed"
            >
              →
            </button>
          </div>
        )}
      </div>

      {loading ? (
        <div className="space-y-4">
          {[1, 2, 3].map((i) => (
            <div key={i} className="border-b border-ic-border pb-4 animate-pulse">
              <div className="h-4 bg-ic-border rounded w-3/4 mb-2"></div>
              <div className="h-3 bg-ic-border rounded w-full mb-1"></div>
              <div className="h-3 bg-ic-border rounded w-2/3"></div>
            </div>
          ))}
        </div>
      ) : currentArticles && currentArticles.length > 0 ? (
        <div className="space-y-4">
          {currentArticles.map((article) => (
            <article key={article.id} className="border border-ic-border rounded-lg p-4 hover:shadow-sm transition-shadow">
              {/* Compact article layout */}
              <div className="flex gap-3">
                {article.image_url && (
                  <div className="flex-shrink-0">
                    <img
                      src={article.image_url}
                      alt={article.title}
                      className="w-20 h-20 object-cover rounded"
                      onError={(e) => {
                        e.currentTarget.style.display = 'none';
                      }}
                    />
                  </div>
                )}

                <div className="flex-1 min-w-0">
                  {/* Publisher and time */}
                  <div className="flex items-center gap-2 mb-1 text-xs">
                    {article.publisher?.logo_url && (
                      <img
                        src={article.publisher.logo_url}
                        alt={article.publisher.name}
                        className="w-4 h-4 object-contain"
                        onError={(e) => {
                          e.currentTarget.style.display = 'none';
                        }}
                      />
                    )}
                    <span className="font-medium text-ic-text-muted">{article.publisher?.name || article.source || 'Unknown'}</span>
                    <span className="text-ic-text-dim">•</span>
                    <span className="text-ic-text-muted">{formatTimeAgo(article.published_utc || article.publishedAt || '')}</span>
                    {(() => {
                      const insight = article.insights?.find(i => i.ticker?.toUpperCase() === symbol.toUpperCase());
                      const sentiment = insight?.sentiment || article.sentiment;
                      return sentiment ? (
                        <span className={`ml-auto px-2 py-0.5 text-xs font-semibold rounded ${getSentimentColor(sentiment)}`}>
                          {sentiment.charAt(0).toUpperCase() + sentiment.slice(1)}
                        </span>
                      ) : null;
                    })()}
                  </div>

                  {/* Title */}
                  <a
                    href={article.article_url || article.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-sm font-semibold text-ic-text-primary hover:text-ic-blue transition-colors line-clamp-2"
                  >
                    {article.title}
                  </a>

                  {/* Description - truncated */}
                  <p className="text-xs text-ic-text-muted mt-1 line-clamp-2">
                    {article.description || article.summary}
                  </p>
                </div>
              </div>

              {/* Compact footer */}
              <div className="flex items-center justify-between mt-3 pt-2 border-t border-ic-border">
                {/* Related tickers - compact */}
                <div className="flex flex-wrap gap-1">
                  {article.tickers?.slice(0, 4).map((ticker, index) => (
                    <span
                      key={index}
                      className={`px-1.5 py-0.5 rounded text-xs font-medium ${
                        ticker === symbol ? 'bg-blue-100 text-blue-700' : 'bg-ic-bg-secondary text-ic-text-muted'
                      }`}
                    >
                      {ticker}
                    </span>
                  ))}
                </div>

                <a
                  href={article.article_url || article.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-xs text-ic-blue hover:text-blue-800 font-medium whitespace-nowrap"
                >
                  Read →
                </a>
              </div>
            </article>
          ))}
        </div>
      ) : (
        <div className="text-center py-8">
          <NewspaperIcon className="h-12 w-12 text-gray-300 mx-auto mb-4" />
          <p className="text-ic-text-muted">No recent news available</p>
        </div>
      )}

    </div>
  );
}
