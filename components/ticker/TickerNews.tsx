'use client';

import { useState, useEffect } from 'react';
import { ClockIcon, NewspaperIcon } from '@heroicons/react/24/outline';

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
    if (!sentiment) return 'text-gray-600 bg-gray-50';
    
    switch (sentiment.toLowerCase()) {
      case 'positive':
        return 'text-green-600 bg-green-50';
      case 'negative':
        return 'text-red-600 bg-red-50';
      default:
        return 'text-gray-600 bg-gray-50';
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
          <NewspaperIcon className="h-6 w-6 text-gray-400 mr-2" />
          <h3 className="text-lg font-semibold text-gray-900">News & Analysis</h3>
          {totalArticles > 0 && (
            <span className="ml-3 text-sm text-gray-500">
              {totalArticles} articles
            </span>
          )}
        </div>
        
        {/* Pagination Controls - Top Right */}
        {totalPages > 1 && (
          <div className="flex items-center space-x-2">
            <span className="text-sm text-gray-500">Page {currentPage} of {totalPages}</span>
            <button
              onClick={() => goToPage(currentPage - 1)}
              disabled={currentPage === 1}
              className="px-3 py-1 text-sm font-medium text-gray-500 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              ←
            </button>
            <button
              onClick={() => goToPage(currentPage + 1)}
              disabled={currentPage === totalPages}
              className="px-3 py-1 text-sm font-medium text-gray-500 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              →
            </button>
          </div>
        )}
      </div>

      {loading ? (
        <div className="space-y-4">
          {[1, 2, 3].map((i) => (
            <div key={i} className="border-b border-gray-100 pb-4 animate-pulse">
              <div className="h-4 bg-gray-200 rounded w-3/4 mb-2"></div>
              <div className="h-3 bg-gray-200 rounded w-full mb-1"></div>
              <div className="h-3 bg-gray-200 rounded w-2/3"></div>
            </div>
          ))}
        </div>
      ) : currentArticles && currentArticles.length > 0 ? (
        <div className="space-y-8">
          {currentArticles.map((article) => (
            <article key={article.id} className="border border-gray-200 rounded-lg p-6 hover:shadow-md transition-shadow">
              {/* Article header with image */}
              <div className="flex gap-4 mb-4">
                {article.image_url && (
                  <div className="flex-shrink-0">
                    <img 
                      src={article.image_url} 
                      alt={article.title}
                      className="w-32 h-32 object-cover rounded-lg"
                      onError={(e) => {
                        e.currentTarget.style.display = 'none';
                      }}
                    />
                  </div>
                )}
                
                <div className="flex-1">
                  {/* Publisher info with logo */}
                  <div className="flex items-center space-x-2 mb-2">
                    {article.publisher?.logo_url && (
                      <img 
                        src={article.publisher.logo_url} 
                        alt={article.publisher.name}
                        className="w-5 h-5 object-contain"
                        onError={(e) => {
                          e.currentTarget.style.display = 'none';
                        }}
                      />
                    )}
                    <span className="text-sm font-medium text-gray-700">{article.publisher?.name || article.source || 'Unknown Source'}</span>
                    <span className="text-gray-400">•</span>
                    <span className="text-sm text-gray-500">{formatTimeAgo(article.published_utc || article.publishedAt || '')}</span>
                  </div>
                  
                  {/* Title with sentiment */}
                  <div className="flex items-start justify-between mb-2">
                    <a 
                    href={article.article_url || article.url} 
                    target="_blank" 
                    rel="noopener noreferrer"
                    className="text-lg font-semibold text-gray-900 hover:text-blue-600 transition-colors leading-tight"
                    >
                      {article.title}
                    </a>
                    {(() => {
                      // Get sentiment from insights for this symbol or fallback to article.sentiment
                      const insight = article.insights?.find(i => i.ticker?.toUpperCase() === symbol.toUpperCase());
                      const sentiment = insight?.sentiment || article.sentiment;
                      return sentiment ? (
                        <span className={`ml-3 px-3 py-1 text-xs font-bold rounded-full ${getSentimentColor(sentiment)}`}>
                          {sentiment.charAt(0).toUpperCase() + sentiment.slice(1)}
                        </span>
                      ) : null;
                    })()}
                  </div>
                  
                  {/* Author */}
                  {article.author && (
                    <div className="text-sm text-gray-600 mb-3">
                      By {article.author}
                    </div>
                  )}
                </div>
              </div>
              
              {/* Article description */}
              <p className="text-gray-700 text-sm leading-relaxed mb-4">
                {article.description || article.summary}
              </p>
              
              {/* Keywords tags */}
              {article.keywords && article.keywords.length > 0 && (
                <div className="mb-4">
                  <div className="text-xs text-gray-500 mb-2">Keywords:</div>
                  <div className="flex flex-wrap gap-1">
                    {article.keywords.slice(0, 6).map((keyword, index) => (
                      <span 
                        key={index}
                        className="bg-gray-100 text-gray-700 px-2 py-1 rounded text-xs font-medium"
                      >
                        {keyword}
                      </span>
                    ))}
                    {article.keywords.length > 6 && (
                      <span className="text-xs text-gray-500">+{article.keywords.length - 6} more</span>
                    )}
                  </div>
                </div>
              )}
              
              {/* Related tickers */}
              {article.tickers && article.tickers.length > 1 && (
                <div className="mb-4">
                  <div className="text-xs text-gray-500 mb-2">Related tickers:</div>
                  <div className="flex flex-wrap gap-2">
                    {article.tickers.map((ticker, index) => (
                      <span 
                        key={index}
                        className={`px-2 py-1 rounded text-xs font-bold ${
                          ticker === symbol ? 'bg-blue-100 text-blue-800' : 'bg-gray-100 text-gray-600'
                        }`}
                      >
                        {ticker}
                      </span>
                    ))}
                  </div>
                </div>
              )}
              
              {/* Sentiment insights for this specific ticker */}
              {article.insights && (
                <div className="mb-4">
                  {article.insights
                    .filter(insight => insight.ticker?.toUpperCase() === symbol.toUpperCase())
                    .map((insight, index) => (
                      <div key={index} className="bg-gray-50 rounded-lg p-3">
                        <p className="text-sm text-gray-700 italic">
                          "{insight.sentiment_reasoning || 'No reasoning provided'}"
                        </p>
                      </div>
                    ))}
                </div>
              )}
              
              {/* Action buttons */}
              <div className="flex items-center justify-between pt-3 border-t border-gray-100">
                <div className="flex items-center space-x-4">
                  {article.publisher?.homepage_url && (
                    <a 
                      href={article.publisher.homepage_url}
                      target="_blank"
                      rel="noopener noreferrer" 
                      className="text-xs text-blue-600 hover:text-blue-800 font-medium"
                    >
                      Visit {article.source}
                    </a>
                  )}
                  
                  <a 
                    href={article.article_url || article.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-xs text-blue-600 hover:text-blue-800 font-medium"
                  >
                    Read Full Article →
                  </a>
                </div>
                
                <div className="flex items-center text-xs text-gray-400">
                  <ClockIcon className="h-3 w-3 mr-1" />
                  <span>
                    {(article.published_utc || article.publishedAt) && !isNaN(new Date(article.published_utc || article.publishedAt || '').getTime()) 
                      ? new Date(article.published_utc || article.publishedAt || '').toLocaleDateString()
                      : 'Unknown date'
                    }
                  </span>
                </div>
              </div>
            </article>
          ))}
        </div>
      ) : (
        <div className="text-center py-8">
          <NewspaperIcon className="h-12 w-12 text-gray-300 mx-auto mb-4" />
          <p className="text-gray-500">No recent news available</p>
        </div>
      )}

    </div>
  );
}
