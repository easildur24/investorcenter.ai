'use client';

import { useState, useEffect } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import { ArrowLeftIcon } from '@heroicons/react/24/outline';
import { tickers } from '@/lib/api/routes';
import { API_BASE_URL } from '@/lib/api';

interface NewsArticle {
  id?: string | number;
  title: string;
  description?: string;
  author?: string;
  article_url: string;
  published_utc: string;
  image_url?: string;
  keywords?: string[];
  tickers?: string[];
  publisher?: {
    name: string;
    homepage_url?: string;
    logo_url?: string;
    favicon_url?: string;
  };
  // Polygon API fields (fallback)
  insights?: Array<{
    ticker: string;
    sentiment: string;
    sentiment_reasoning: string;
  }>;
  // IC Score AI sentiment fields (primary)
  sentiment_score?: number; // -100 to +100 (from FinBERT AI analysis)
  sentiment_label?: string; // "Positive", "Negative", "Neutral"
  relevance_score?: number; // 0 to 100
  // Backward compatibility
  summary?: string;
  source?: string;
  url?: string;
  publishedAt?: string;
  sentiment?: string;
}

type Sentiment = 'Bullish' | 'Bearish' | 'Neutral';

function getSentimentFromApi(sentiment: string | undefined): Sentiment {
  if (!sentiment) return 'Neutral';
  const lower = sentiment.toLowerCase();
  if (lower === 'positive' || lower === 'bullish') return 'Bullish';
  if (lower === 'negative' || lower === 'bearish') return 'Bearish';
  return 'Neutral';
}

function getSentimentTextClass(sentiment: Sentiment): string {
  switch (sentiment) {
    case 'Bullish':
      return 'text-ic-positive';
    case 'Bearish':
      return 'text-ic-negative';
    default:
      return 'text-ic-text-muted';
  }
}

function getSentimentBgClass(sentiment: Sentiment): string {
  switch (sentiment) {
    case 'Bullish':
      return 'bg-ic-positive';
    case 'Bearish':
      return 'bg-ic-negative';
    default:
      return 'bg-ic-text-muted';
  }
}

function getConfidence(article: NewsArticle, symbol: string): number {
  // Primary: Use IC Score AI sentiment_score (-100 to +100)
  if (article.sentiment_score !== undefined && article.sentiment_score !== null) {
    return Math.round(Math.abs(article.sentiment_score));
  }

  // Fallback: Try Polygon insights
  const insight = article.insights?.find((i) => i.ticker?.toUpperCase() === symbol.toUpperCase());

  if (insight?.sentiment_reasoning) {
    const reasoning = insight.sentiment_reasoning;
    let confidence = 50;
    if (reasoning.includes('strong') || reasoning.includes('significant')) confidence += 15;
    if (reasoning.includes('clear') || reasoning.includes('definite')) confidence += 10;
    if (reasoning.length > 100) confidence += 10;
    return Math.min(95, Math.max(35, confidence));
  }

  const sentiment = insight?.sentiment || article.sentiment;
  if (sentiment) {
    const lower = sentiment.toLowerCase();
    if (lower === 'positive' || lower === 'negative') return 65;
  }

  return 50;
}

export default function TickerNewsPage() {
  const params = useParams();
  const symbol = (params.symbol as string).toUpperCase();

  const [news, setNews] = useState<NewsArticle[]>([]);
  const [loading, setLoading] = useState(true);
  const [currentPage, setCurrentPage] = useState(1);
  const articlesPerPage = 10;

  useEffect(() => {
    const fetchNews = async () => {
      try {
        setLoading(true);
        const response = await fetch(`${API_BASE_URL}${tickers.news(symbol)}`);
        const result = await response.json();
        if (result.data && Array.isArray(result.data)) {
          setNews(result.data);
        } else {
          setNews([]);
        }
      } catch (error) {
        console.error('Error fetching news:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchNews();
  }, [symbol]);

  const formatTimeAgo = (dateString: string) => {
    if (!dateString) return '';
    const date = new Date(dateString);
    if (isNaN(date.getTime())) return '';

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

  const formatDate = (dateString: string) => {
    if (!dateString) return '';
    const date = new Date(dateString);
    if (isNaN(date.getTime())) return '';
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: 'numeric',
      minute: '2-digit',
    });
  };

  const getArticleSentiment = (article: NewsArticle): Sentiment => {
    // Primary: Use IC Score AI sentiment_label
    if (article.sentiment_label) {
      return getSentimentFromApi(article.sentiment_label);
    }
    // Fallback: Try Polygon insights
    const insight = article.insights?.find((i) => i.ticker?.toUpperCase() === symbol.toUpperCase());
    return getSentimentFromApi(insight?.sentiment || article.sentiment);
  };

  // Pagination
  const totalPages = Math.ceil(news.length / articlesPerPage);
  const startIndex = (currentPage - 1) * articlesPerPage;
  const currentArticles = news.slice(startIndex, startIndex + articlesPerPage);

  return (
    <div className="min-h-screen bg-ic-bg-primary">
      {/* Header */}
      <div className="bg-ic-surface border-b border-ic-border">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <Link
            href={`/ticker/${symbol}`}
            className="inline-flex items-center gap-2 text-ic-text-muted hover:text-ic-text-primary mb-4 transition-colors"
          >
            <ArrowLeftIcon className="w-4 h-4" />
            Back to {symbol}
          </Link>

          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold text-ic-text-primary">{symbol} News & Sentiment</h1>
              <p className="text-ic-text-muted mt-1">
                {news.length} articles with AI-powered sentiment analysis
              </p>
            </div>
            <div className="flex items-center gap-1.5 text-ic-positive text-sm font-medium">
              AI Analysis
              <span className="w-2 h-2 rounded-full bg-ic-positive" />
            </div>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {loading ? (
          <div className="space-y-4">
            {[1, 2, 3, 4, 5].map((i) => (
              <div
                key={i}
                className="bg-ic-surface border border-ic-border rounded-2xl p-6 animate-pulse"
              >
                <div className="h-4 bg-ic-border rounded w-1/3 mb-3"></div>
                <div className="h-6 bg-ic-border rounded w-full mb-2"></div>
                <div className="h-4 bg-ic-border rounded w-2/3 mb-4"></div>
                <div className="h-1.5 bg-ic-border rounded w-full"></div>
              </div>
            ))}
          </div>
        ) : news.length === 0 ? (
          <div className="bg-ic-surface border border-ic-border rounded-2xl p-12 text-center">
            <p className="text-ic-text-muted text-lg">No news articles found for {symbol}</p>
          </div>
        ) : (
          <>
            {/* News List */}
            <div className="space-y-4">
              {currentArticles.map((article, i) => {
                const sentiment = getArticleSentiment(article);
                const confidence = getConfidence(article, symbol);
                const source = article.publisher?.name || article.source || 'Unknown';
                const timeAgo = formatTimeAgo(article.published_utc || article.publishedAt || '');
                const fullDate = formatDate(article.published_utc || article.publishedAt || '');

                return (
                  <article
                    key={article.id || i}
                    className="bg-ic-surface border border-ic-border rounded-2xl p-6 shadow-[var(--ic-shadow-card)] hover:shadow-[var(--ic-shadow-card-hover)] transition-shadow"
                  >
                    {/* Header: Source, Time, Sentiment */}
                    <div className="flex justify-between items-start mb-3">
                      <div className="flex items-center gap-2">
                        {article.publisher?.favicon_url && (
                          <img
                            src={article.publisher.favicon_url}
                            alt=""
                            className="w-4 h-4 rounded"
                            onError={(e) => {
                              e.currentTarget.style.display = 'none';
                            }}
                          />
                        )}
                        <span className="text-ic-text-muted text-sm font-medium">{source}</span>
                        <span className="text-ic-text-dim">·</span>
                        <span className="text-ic-text-muted text-sm" title={fullDate}>
                          {timeAgo}
                        </span>
                      </div>
                      <span className={`font-semibold text-sm ${getSentimentTextClass(sentiment)}`}>
                        {sentiment}
                      </span>
                    </div>

                    {/* Title */}
                    <a
                      href={article.article_url || article.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="block text-lg font-semibold text-ic-text-primary hover:text-ic-blue transition-colors mb-2"
                    >
                      {article.title}
                    </a>

                    {/* Description */}
                    <p className="text-ic-text-muted text-sm mb-4 line-clamp-3">
                      {article.description || article.summary}
                    </p>

                    {/* Image (if available) */}
                    {article.image_url && (
                      <a
                        href={article.article_url || article.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="block mb-4"
                      >
                        <img
                          src={article.image_url}
                          alt={article.title}
                          className="w-full h-48 object-cover rounded-lg"
                          onError={(e) => {
                            e.currentTarget.style.display = 'none';
                          }}
                        />
                      </a>
                    )}

                    {/* Confidence Bar */}
                    <div className="flex items-center gap-3 mb-4">
                      <span className="text-ic-text-dim text-xs">Confidence</span>
                      <div className="flex-1 h-1.5 bg-ic-border-subtle rounded-full overflow-hidden">
                        <div
                          className={`h-full rounded-full transition-all ${getSentimentBgClass(sentiment)}`}
                          style={{ width: `${confidence}%` }}
                        />
                      </div>
                      <span className="text-ic-text-muted text-sm tabular-nums">{confidence}%</span>
                    </div>

                    {/* Related Tickers */}
                    {article.tickers && article.tickers.length > 0 && (
                      <div className="flex flex-wrap gap-2 pt-4 border-t border-ic-border-subtle">
                        {article.tickers.slice(0, 8).map((ticker, idx) => (
                          <Link
                            key={idx}
                            href={`/ticker/${ticker}`}
                            className={`px-2 py-1 rounded text-xs font-medium transition-colors ${
                              ticker === symbol
                                ? 'bg-ic-blue/20 text-ic-blue'
                                : 'bg-ic-bg-secondary text-ic-text-muted hover:bg-ic-bg-tertiary'
                            }`}
                          >
                            {ticker}
                          </Link>
                        ))}
                      </div>
                    )}
                  </article>
                );
              })}
            </div>

            {/* Pagination */}
            {totalPages > 1 && (
              <div className="flex items-center justify-center gap-2 mt-8">
                <button
                  onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                  disabled={currentPage === 1}
                  className="px-4 py-2 text-sm font-medium text-ic-text-muted bg-ic-surface border border-ic-border rounded-lg hover:bg-ic-surface-hover disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  Previous
                </button>

                <div className="flex items-center gap-1">
                  {Array.from({ length: totalPages }, (_, i) => i + 1)
                    .filter((page) => {
                      // Show first, last, current, and adjacent pages
                      return page === 1 || page === totalPages || Math.abs(page - currentPage) <= 1;
                    })
                    .map((page, idx, arr) => (
                      <span key={page} className="flex items-center">
                        {idx > 0 && arr[idx - 1] !== page - 1 && (
                          <span className="px-2 text-ic-text-dim">...</span>
                        )}
                        <button
                          onClick={() => setCurrentPage(page)}
                          className={`w-10 h-10 text-sm font-medium rounded-lg transition-colors ${
                            currentPage === page
                              ? 'bg-ic-blue text-white'
                              : 'text-ic-text-muted hover:bg-ic-surface-hover'
                          }`}
                        >
                          {page}
                        </button>
                      </span>
                    ))}
                </div>

                <button
                  onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
                  disabled={currentPage === totalPages}
                  className="px-4 py-2 text-sm font-medium text-ic-text-muted bg-ic-surface border border-ic-border rounded-lg hover:bg-ic-surface-hover disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  Next
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
