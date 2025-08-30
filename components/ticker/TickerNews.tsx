'use client';

import { useState, useEffect } from 'react';
import { ClockIcon, NewspaperIcon } from '@heroicons/react/24/outline';

interface TickerNewsProps {
  symbol: string;
}

interface NewsArticle {
  id: number;
  title: string;
  summary: string;
  author: string;
  source: string;
  url: string;
  sentiment: string;
  publishedAt: string;
}

export default function TickerNews({ symbol }: TickerNewsProps) {
  const [news, setNews] = useState<NewsArticle[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchNews = async () => {
      try {
        setLoading(true);
        const response = await fetch(`/api/v1/tickers/${symbol}/news`);
        const result = await response.json();
        setNews(result.data);
      } catch (error) {
        console.error('Error fetching news:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchNews();
  }, [symbol]);

  const getSentimentColor = (sentiment: string) => {
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
    const date = new Date(dateString);
    const now = new Date();
    const diffInHours = Math.floor((now.getTime() - date.getTime()) / (1000 * 60 * 60));
    
    if (diffInHours < 1) return 'Just now';
    if (diffInHours < 24) return `${diffInHours}h ago`;
    const diffInDays = Math.floor(diffInHours / 24);
    return `${diffInDays}d ago`;
  };

  return (
    <div className="p-6">
      <div className="flex items-center mb-6">
        <NewspaperIcon className="h-6 w-6 text-gray-400 mr-2" />
        <h3 className="text-lg font-semibold text-gray-900">News & Analysis</h3>
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
      ) : news.length > 0 ? (
        <div className="space-y-6">
          {news.map((article) => (
            <article key={article.id} className="border-b border-gray-100 pb-6 last:border-b-0">
              <div className="flex items-start justify-between mb-2">
                <h4 className="text-base font-medium text-gray-900 leading-6 hover:text-primary-600 cursor-pointer">
                  {article.title}
                </h4>
                <span className={`ml-2 px-2 py-1 text-xs font-medium rounded-full ${getSentimentColor(article.sentiment)}`}>
                  {article.sentiment}
                </span>
              </div>
              
              <p className="text-gray-600 text-sm mb-3 leading-relaxed">
                {article.summary}
              </p>
              
              <div className="flex items-center justify-between text-xs text-gray-500">
                <div className="flex items-center space-x-4">
                  <span className="font-medium">{article.source}</span>
                  {article.author && (
                    <>
                      <span>•</span>
                      <span>{article.author}</span>
                    </>
                  )}
                </div>
                <div className="flex items-center">
                  <ClockIcon className="h-3 w-3 mr-1" />
                  <span>{formatTimeAgo(article.publishedAt)}</span>
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
