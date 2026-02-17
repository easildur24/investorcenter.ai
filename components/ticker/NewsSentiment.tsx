'use client';

import { useState, useEffect } from 'react';

interface NewsSentimentProps {
  symbol: string;
}

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

// Get confidence from sentiment_score or generate fallback
function getConfidence(article: NewsArticle, symbol: string): number {
  // Primary: Use IC Score AI sentiment_score (-100 to +100)
  // Convert absolute value to percentage (0-100 scale)
  if (article.sentiment_score !== undefined && article.sentiment_score !== null) {
    // |sentiment_score| gives us the confidence level
    // -80 = 80% confident it's negative, +60 = 60% confident it's positive
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

  // Last resort: generate based on sentiment presence
  const sentiment = insight?.sentiment || article.sentiment;
  if (sentiment) {
    const lower = sentiment.toLowerCase();
    if (lower === 'positive' || lower === 'negative') return 65;
  }

  return 50; // Default neutral confidence
}

export default function NewsSentiment({ symbol }: NewsSentimentProps) {
  const [news, setNews] = useState<NewsArticle[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchNews = async () => {
      try {
        setLoading(true);
        const response = await fetch(`/api/v1/tickers/${symbol}/news`);
        const result = await response.json();
        if (result.data && Array.isArray(result.data)) {
          setNews(result.data.slice(0, 5)); // Show only 3-5 items
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
    if (diffInHours < 24) return `${diffInHours}h`;
    const diffInDays = Math.floor(diffInHours / 24);
    if (diffInDays < 30) return `${diffInDays}d`;
    const diffInMonths = Math.floor(diffInDays / 30);
    if (diffInMonths < 12) return `${diffInMonths}mo`;
    const diffInYears = Math.floor(diffInDays / 365);
    return `${diffInYears}y`;
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

  if (loading) {
    return (
      <div className="bg-ic-surface border border-ic-border rounded-2xl p-6 shadow-[var(--ic-shadow-card)]">
        <div className="flex justify-between items-center mb-5">
          <div className="h-6 bg-ic-border rounded w-40 animate-pulse"></div>
          <div className="h-5 bg-ic-border rounded w-24 animate-pulse"></div>
        </div>
        <div className="space-y-4">
          {[1, 2, 3].map((i) => (
            <div
              key={i}
              className="py-4 border-t border-ic-border-subtle first:border-t-0 animate-pulse"
            >
              <div className="h-4 bg-ic-border rounded w-1/3 mb-2"></div>
              <div className="h-5 bg-ic-border rounded w-full mb-2"></div>
              <div className="h-4 bg-ic-border rounded w-2/3 mb-3"></div>
              <div className="h-1.5 bg-ic-border rounded w-full"></div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (news.length === 0) {
    return (
      <div className="bg-ic-surface border border-ic-border rounded-2xl p-6 shadow-[var(--ic-shadow-card)]">
        <div className="flex justify-between items-center mb-5">
          <h2 className="text-xl font-semibold text-ic-text-primary">News & Sentiment</h2>
          <div className="flex items-center gap-1.5 text-ic-positive text-sm font-medium">
            AI Analysis
            <span className="w-2 h-2 rounded-full bg-ic-positive" />
          </div>
        </div>
        <div className="text-center py-8">
          <p className="text-ic-text-muted">No recent news available</p>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-ic-surface border border-ic-border rounded-2xl p-6 shadow-[var(--ic-shadow-card)]">
      {/* Header */}
      <div className="flex justify-between items-center mb-5">
        <h2 className="text-xl font-semibold text-ic-text-primary">News & Sentiment</h2>
        <div className="flex items-center gap-1.5 text-ic-positive text-sm font-medium">
          AI Analysis
          <span className="w-2 h-2 rounded-full bg-ic-positive" />
        </div>
      </div>

      {/* News Items */}
      {news.map((article, i) => {
        const sentiment = getArticleSentiment(article);
        const confidence = getConfidence(article, symbol);
        const source = article.publisher?.name || article.source || 'Unknown';
        const time = formatTimeAgo(article.published_utc || article.publishedAt || '');

        return (
          <div
            key={article.id || i}
            className="py-4 border-t border-ic-border-subtle first:border-t-0"
          >
            {/* Source & Sentiment */}
            <div className="flex justify-between items-center mb-2">
              <span className="text-ic-text-muted text-sm">
                {source} {time && `· ${time}`}
              </span>
              <span className={`font-semibold text-sm ${getSentimentTextClass(sentiment)}`}>
                {sentiment}
              </span>
            </div>

            {/* Title */}
            <a
              href={article.article_url || article.url}
              target="_blank"
              rel="noopener noreferrer"
              className="block text-ic-text-primary font-medium mb-1.5 hover:text-ic-blue cursor-pointer transition-colors"
            >
              {article.title}
            </a>

            {/* Description */}
            <p className="text-ic-text-muted text-sm mb-3 line-clamp-2">
              {article.description || article.summary}
            </p>

            {/* Confidence Bar */}
            <div className="flex items-center gap-3">
              <div className="flex-1 h-1.5 bg-ic-border-subtle rounded-full overflow-hidden">
                <div
                  className={`h-full rounded-full transition-all ${getSentimentBgClass(sentiment)}`}
                  style={{ width: `${confidence}%` }}
                />
              </div>
              <span className="text-ic-text-muted text-sm tabular-nums">{confidence}%</span>
            </div>
          </div>
        );
      })}

      {/* Footer */}
      <div className="mt-4 pt-4 border-t border-ic-border-subtle">
        <a
          href={`/ticker/${symbol}/news`}
          className="text-ic-blue text-sm font-medium hover:underline"
        >
          View all articles →
        </a>
      </div>
    </div>
  );
}
