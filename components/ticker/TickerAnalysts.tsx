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
      ) : consensus ? (
        <div className="space-y-6">
          {/* Consensus Summary */}
          <div className="bg-gray-50 rounded-lg p-4">
            <div className="text-center mb-4">
              <div className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${getRatingColor(consensus.rating)}`}>
                {consensus.rating}
              </div>
              <div className="mt-2 text-2xl font-bold text-gray-900">
                ${safeToFixed(consensus.priceTarget, 2)}
              </div>
              <div className="text-sm text-gray-500">Price Target</div>
              {consensus.upside && (
                <div className={`text-sm font-medium mt-1 ${safeParseNumber(consensus.upside) > 0 ? 'text-green-600' : 'text-red-600'}`}>
                  {safeParseNumber(consensus.upside) > 0 ? '+' : ''}{safeToFixed(consensus.upside, 1)}% upside
                </div>
              )}
            </div>

            <div className="text-xs text-gray-500 text-center">
              Based on {consensus.numberOfAnalysts} analysts
            </div>
          </div>

          {/* Rating Distribution */}
          <div>
            <h4 className="text-sm font-medium text-gray-700 mb-3">Rating Distribution</h4>
            <div className="space-y-2">
              <div className="flex justify-between items-center">
                <span className="text-sm text-gray-600">Strong Buy</span>
                <div className="flex items-center">
                  <div className="w-24 bg-gray-200 rounded-full h-2 mr-2">
                    <div 
                      className="bg-green-600 h-2 rounded-full" 
                      style={{ width: `${(consensus.strongBuy / consensus.numberOfAnalysts) * 100}%` }}
                    ></div>
                  </div>
                  <span className="text-sm font-medium w-6">{consensus.strongBuy}</span>
                </div>
              </div>
              
              <div className="flex justify-between items-center">
                <span className="text-sm text-gray-600">Buy</span>
                <div className="flex items-center">
                  <div className="w-24 bg-gray-200 rounded-full h-2 mr-2">
                    <div 
                      className="bg-green-400 h-2 rounded-full" 
                      style={{ width: `${(consensus.buy / consensus.numberOfAnalysts) * 100}%` }}
                    ></div>
                  </div>
                  <span className="text-sm font-medium w-6">{consensus.buy}</span>
                </div>
              </div>
              
              <div className="flex justify-between items-center">
                <span className="text-sm text-gray-600">Hold</span>
                <div className="flex items-center">
                  <div className="w-24 bg-gray-200 rounded-full h-2 mr-2">
                    <div 
                      className="bg-yellow-400 h-2 rounded-full" 
                      style={{ width: `${(consensus.hold / consensus.numberOfAnalysts) * 100}%` }}
                    ></div>
                  </div>
                  <span className="text-sm font-medium w-6">{consensus.hold}</span>
                </div>
              </div>
              
              <div className="flex justify-between items-center">
                <span className="text-sm text-gray-600">Sell</span>
                <div className="flex items-center">
                  <div className="w-24 bg-gray-200 rounded-full h-2 mr-2">
                    <div 
                      className="bg-red-400 h-2 rounded-full" 
                      style={{ width: `${(consensus.sell / consensus.numberOfAnalysts) * 100}%` }}
                    ></div>
                  </div>
                  <span className="text-sm font-medium w-6">{consensus.sell}</span>
                </div>
              </div>
            </div>
          </div>

          {/* Price Target Range */}
          <div>
            <h4 className="text-sm font-medium text-gray-700 mb-3">Price Target Range</h4>
            <div className="space-y-2">
              <div className="flex justify-between">
                <span className="text-sm text-gray-600">High</span>
                <span className="font-medium">${safeToFixed(consensus.priceTargetHigh, 2)}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-gray-600">Mean</span>
                <span className="font-medium">${safeToFixed(consensus.priceTarget, 2)}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-gray-600">Low</span>
                <span className="font-medium">${safeToFixed(consensus.priceTargetLow, 2)}</span>
              </div>
            </div>
          </div>

          {/* Recent Ratings */}
          <div>
            <h4 className="text-sm font-medium text-gray-700 mb-3">Recent Ratings</h4>
            <div className="space-y-3">
              {ratings.slice(0, 5).map((rating, index) => (
                <div key={index} className="flex justify-between items-center">
                  <div>
                    <div className="text-sm font-medium text-gray-900">{rating.firm}</div>
                    <div className="text-xs text-gray-500">{formatDate(rating.ratingDate)}</div>
                  </div>
                  <div className="text-right">
                    <div className={`inline-flex px-2 py-1 text-xs font-medium rounded ${getRatingColor(rating.rating)}`}>
                      {rating.rating}
                    </div>
                    {rating.priceTarget && (
                      <div className="text-xs text-gray-500 mt-1">
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
