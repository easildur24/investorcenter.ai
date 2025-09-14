'use client';

import { useState, useEffect } from 'react';
import { UserGroupIcon } from '@heroicons/react/24/outline';
import { safeToFixed, safeParseNumber } from '@/lib/utils';

interface TickerAnalystsProps {
  symbol: string;
}

interface AnalystRating {
  firm: string;
  analyst: string;
  rating: string;
  priceTarget: number | string;
  ratingDate: string;
}

interface AnalystConsensus {
  rating: string;
  ratingScore: number | string;
  priceTarget: number | string;
  priceTargetHigh: number | string;
  priceTargetLow: number | string;
  upside: number | string;
  numberOfAnalysts: number;
  strongBuy: number;
  buy: number;
  hold: number;
  sell: number;
  strongSell: number;
}

export default function TickerAnalysts({ symbol }: TickerAnalystsProps) {
  const [ratings, setRatings] = useState<AnalystRating[]>([]);
  const [consensus, setConsensus] = useState<AnalystConsensus | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        const [ratingsResponse, overviewResponse] = await Promise.all([
          fetch(`/api/v1/tickers/${symbol}/analysts`),
          fetch(`/api/v1/tickers/${symbol}`)
        ]);
        
        const ratingsResult = await ratingsResponse.json();
        const overviewResult = await overviewResponse.json();
        
        setRatings(ratingsResult.data);
        setConsensus(overviewResult.data.summary.analystConsensus);
      } catch (error) {
        console.error('Error fetching analyst data:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [symbol]);

  const getRatingColor = (rating: string) => {
    switch (rating.toLowerCase()) {
      case 'strong buy':
        return 'text-green-700 bg-green-100';
      case 'buy':
        return 'text-green-600 bg-green-50';
      case 'hold':
        return 'text-yellow-600 bg-yellow-50';
      case 'sell':
        return 'text-red-600 bg-red-50';
      case 'strong sell':
        return 'text-red-700 bg-red-100';
      default:
        return 'text-gray-600 bg-gray-50';
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric'
    });
  };

  return (
    <div className="p-6">
      <div className="flex items-center mb-6">
        <UserGroupIcon className="h-6 w-6 text-gray-400 mr-2" />
        <h3 className="text-lg font-semibold text-gray-900">Analyst Ratings</h3>
      </div>

      {loading ? (
        <div className="space-y-4 animate-pulse">
          <div className="h-8 bg-gray-200 rounded w-full"></div>
          <div className="h-4 bg-gray-200 rounded w-3/4"></div>
          <div className="h-4 bg-gray-200 rounded w-2/3"></div>
        </div>
      ) : ratings.length > 0 ? (
        <div className="space-y-6">
          {/* Show ratings directly without consensus */}
          <div>
            <h4 className="text-sm font-medium text-gray-700 mb-4">Analyst Ratings ({ratings.length} analysts)</h4>
            <div className="space-y-3">
              {ratings.map((rating, index) => (
                <div key={index} className="flex justify-between items-center p-3 bg-gray-50 rounded-lg">
                  <div>
                    <div className="text-sm font-medium text-gray-900">{rating.firm}</div>
                    <div className="text-xs text-gray-500">{rating.analyst}</div>
                    <div className="text-xs text-gray-400">{formatDate(rating.ratingDate)}</div>
                  </div>
                  <div className="text-right">
                    <div className={`inline-flex px-3 py-1 text-sm font-medium rounded-full ${getRatingColor(rating.rating)}`}>
                      {rating.rating}
                    </div>
                    {rating.priceTarget && (
                      <div className="text-sm text-gray-700 mt-1 font-medium">
                        PT: ${safeToFixed(rating.priceTarget, 0)}
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      ) : (
        <div className="text-center py-8">
          <UserGroupIcon className="h-12 w-12 text-gray-300 mx-auto mb-4" />
          <p className="text-gray-500">No analyst data available</p>
        </div>
      )}
    </div>
  );
}
