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
        return 'text-green-700 bg-ic-positive-bg';
      case 'buy':
        return 'text-ic-positive bg-green-50';
      case 'hold':
        return 'text-yellow-600 bg-yellow-50';
      case 'sell':
        return 'text-ic-negative bg-red-50';
      case 'strong sell':
        return 'text-red-700 bg-ic-negative-bg';
      default:
        return 'text-ic-text-muted bg-ic-bg-secondary';
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
        <UserGroupIcon className="h-6 w-6 text-ic-text-dim mr-2" />
        <h3 className="text-lg font-semibold text-ic-text-primary">Analyst Ratings</h3>
      </div>

      {loading ? (
        <div className="space-y-4 animate-pulse">
          <div className="h-8 bg-ic-border rounded w-full"></div>
          <div className="h-4 bg-ic-border rounded w-3/4"></div>
          <div className="h-4 bg-ic-border rounded w-2/3"></div>
        </div>
      ) : ratings.length > 0 ? (
        <div className="space-y-6">
          {/* Show ratings directly without consensus */}
          <div>
            <h4 className="text-sm font-medium text-ic-text-secondary mb-4">Analyst Ratings ({ratings.length} analysts)</h4>
            <div className="space-y-3">
              {ratings.map((rating, index) => (
                <div key={index} className="flex justify-between items-center p-3 bg-ic-bg-secondary rounded-lg">
                  <div>
                    <div className="text-sm font-medium text-ic-text-primary">{rating.firm}</div>
                    <div className="text-xs text-ic-text-muted">{rating.analyst}</div>
                    <div className="text-xs text-ic-text-dim">{formatDate(rating.ratingDate)}</div>
                  </div>
                  <div className="text-right">
                    <div className={`inline-flex px-3 py-1 text-sm font-medium rounded-full ${getRatingColor(rating.rating)}`}>
                      {rating.rating}
                    </div>
                    {rating.priceTarget && (
                      <div className="text-sm text-ic-text-secondary mt-1 font-medium">
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
          <p className="text-ic-text-muted">No analyst data available</p>
        </div>
      )}
    </div>
  );
}
