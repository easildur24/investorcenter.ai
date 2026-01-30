'use client';

import { useState, useEffect } from 'react';
import { UserGroupIcon } from '@heroicons/react/24/outline';
import { safeToFixed, safeParseNumber } from '@/lib/utils';
import { getComprehensiveMetrics } from '@/lib/api/metrics';
import { AnalystRatings } from '@/types/metrics';

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

export default function TickerAnalysts({ symbol }: TickerAnalystsProps) {
  const [ratings, setRatings] = useState<AnalystRating[]>([]);
  const [analystMetrics, setAnalystMetrics] = useState<AnalystRatings | null>(null);
  const [currentPrice, setCurrentPrice] = useState<number>(0);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);

        // Fetch both individual ratings and comprehensive metrics in parallel
        const [ratingsResponse, overviewResponse, metricsResponse] = await Promise.all([
          fetch(`/api/v1/tickers/${symbol}/analysts`),
          fetch(`/api/v1/tickers/${symbol}`),
          getComprehensiveMetrics(symbol).catch(() => null)
        ]);

        const ratingsResult = await ratingsResponse.json();
        const overviewResult = await overviewResponse.json();

        setRatings(ratingsResult.data || []);

        // Get current price from overview
        if (overviewResult.data?.summary?.price?.price) {
          setCurrentPrice(parseFloat(overviewResult.data.summary.price.price));
        }

        // Get analyst metrics from comprehensive metrics
        if (metricsResponse?.data?.analyst_ratings) {
          setAnalystMetrics(metricsResponse.data.analyst_ratings);
        }
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
      case 'buy':
        return 'text-ic-positive bg-ic-positive-bg';
      case 'hold':
        return 'text-ic-warning bg-ic-warning-bg';
      case 'sell':
      case 'strong sell':
        return 'text-ic-negative bg-ic-negative-bg';
      default:
        return 'text-ic-text-muted bg-ic-bg-secondary';
    }
  };

  const getConsensusColor = (consensus: string | null) => {
    if (!consensus) return 'text-ic-text-muted';
    switch (consensus.toLowerCase()) {
      case 'strong buy':
        return 'text-green-700 bg-green-100';
      case 'buy':
        return 'text-green-600 bg-green-50';
      case 'hold':
        return 'text-yellow-700 bg-yellow-100';
      case 'sell':
        return 'text-red-600 bg-red-50';
      case 'strong sell':
        return 'text-red-700 bg-red-100';
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

  // Calculate total ratings from metrics
  const totalRatings = analystMetrics
    ? (analystMetrics.analyst_rating_strong_buy ?? 0) +
      (analystMetrics.analyst_rating_buy ?? 0) +
      (analystMetrics.analyst_rating_hold ?? 0) +
      (analystMetrics.analyst_rating_sell ?? 0) +
      (analystMetrics.analyst_rating_strong_sell ?? 0)
    : 0;

  // Calculate upside percentage
  const upsidePercent = analystMetrics?.target_consensus && currentPrice > 0
    ? ((analystMetrics.target_consensus - currentPrice) / currentPrice) * 100
    : null;

  const hasData = ratings.length > 0 || (analystMetrics && (totalRatings > 0 || analystMetrics.target_consensus));

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
      ) : hasData ? (
        <div className="space-y-6">
          {/* Price Targets Section */}
          {analystMetrics?.target_consensus && (
            <div>
              <h4 className="text-sm font-medium text-ic-text-secondary mb-3">Price Targets</h4>
              <div className="bg-ic-bg-secondary rounded-lg p-4">
                {/* Current Price */}
                <div className="text-xs text-ic-text-muted mb-2">
                  Current Price: <span className="font-medium text-ic-text-primary">${currentPrice.toFixed(2)}</span>
                </div>

                {/* Price Target Range Visual */}
                {analystMetrics.target_low !== null && analystMetrics.target_high !== null && (
                  <div className="mb-4">
                    <div className="relative h-8 bg-ic-bg-tertiary rounded-lg">
                      {(() => {
                        const min = Math.min(analystMetrics.target_low!, currentPrice * 0.8);
                        const max = Math.max(analystMetrics.target_high!, currentPrice * 1.2);
                        const range = max - min;
                        const lowPos = ((analystMetrics.target_low! - min) / range) * 100;
                        const highPos = ((analystMetrics.target_high! - min) / range) * 100;
                        const currentPos = ((currentPrice - min) / range) * 100;
                        const consensusPos = analystMetrics.target_consensus
                          ? ((analystMetrics.target_consensus - min) / range) * 100
                          : null;

                        return (
                          <>
                            {/* Target range bar */}
                            <div
                              className="absolute h-3 top-2.5 bg-blue-200 rounded"
                              style={{
                                left: `${lowPos}%`,
                                width: `${highPos - lowPos}%`
                              }}
                            />
                            {/* Low target marker */}
                            <div
                              className="absolute w-0.5 h-6 top-1 bg-blue-400 rounded"
                              style={{ left: `${lowPos}%` }}
                              title={`Low: $${analystMetrics.target_low?.toFixed(2)}`}
                            />
                            {/* High target marker */}
                            <div
                              className="absolute w-0.5 h-6 top-1 bg-blue-400 rounded"
                              style={{ left: `${highPos}%` }}
                              title={`High: $${analystMetrics.target_high?.toFixed(2)}`}
                            />
                            {/* Consensus marker */}
                            {consensusPos !== null && (
                              <div
                                className="absolute w-1.5 h-7 top-0.5 bg-blue-600 rounded"
                                style={{ left: `calc(${consensusPos}% - 3px)` }}
                                title={`Consensus: $${analystMetrics.target_consensus?.toFixed(2)}`}
                              />
                            )}
                            {/* Current price marker */}
                            <div
                              className="absolute w-2 h-8 top-0 bg-ic-text-primary rounded"
                              style={{ left: `calc(${currentPos}% - 4px)` }}
                              title={`Current: $${currentPrice.toFixed(2)}`}
                            />
                          </>
                        );
                      })()}
                    </div>
                    <div className="flex justify-between text-xs text-ic-text-dim mt-1">
                      <span>${analystMetrics.target_low?.toFixed(0)} (Low)</span>
                      <span className="font-medium">${analystMetrics.target_consensus?.toFixed(0)} (Avg)</span>
                      <span>${analystMetrics.target_high?.toFixed(0)} (High)</span>
                    </div>
                  </div>
                )}

                {/* Key Metrics Grid */}
                <div className="grid grid-cols-2 gap-3">
                  <div className="text-center p-2 bg-ic-surface rounded">
                    <div className="text-lg font-bold text-ic-text-primary">
                      ${analystMetrics.target_consensus?.toFixed(0)}
                    </div>
                    <div className="text-xs text-ic-text-muted">Target Price</div>
                  </div>
                  <div className="text-center p-2 bg-ic-surface rounded">
                    <div className={`text-lg font-bold ${upsidePercent && upsidePercent >= 0 ? 'text-ic-positive' : 'text-ic-negative'}`}>
                      {upsidePercent !== null ? `${upsidePercent >= 0 ? '+' : ''}${upsidePercent.toFixed(1)}%` : '—'}
                    </div>
                    <div className="text-xs text-ic-text-muted">Upside</div>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Analyst Consensus Section */}
          {totalRatings > 0 && analystMetrics && (
            <div>
              <h4 className="text-sm font-medium text-ic-text-secondary mb-3">
                Consensus ({totalRatings} analysts)
              </h4>

              {/* Consensus Badge */}
              {analystMetrics.analyst_consensus && (
                <div className="mb-3">
                  <span className={`inline-flex px-3 py-1.5 text-sm font-semibold rounded-full ${getConsensusColor(analystMetrics.analyst_consensus)}`}>
                    {analystMetrics.analyst_consensus.toUpperCase()}
                  </span>
                </div>
              )}

              {/* Rating Distribution Bar */}
              <div className="bg-ic-bg-secondary rounded-lg p-3">
                <div className="flex h-6 rounded overflow-hidden mb-2">
                  {analystMetrics.analyst_rating_strong_buy !== null && analystMetrics.analyst_rating_strong_buy > 0 && (
                    <div
                      className="bg-green-600 flex items-center justify-center text-white text-xs font-medium"
                      style={{ width: `${(analystMetrics.analyst_rating_strong_buy / totalRatings) * 100}%` }}
                    >
                      {analystMetrics.analyst_rating_strong_buy}
                    </div>
                  )}
                  {analystMetrics.analyst_rating_buy !== null && analystMetrics.analyst_rating_buy > 0 && (
                    <div
                      className="bg-green-400 flex items-center justify-center text-white text-xs font-medium"
                      style={{ width: `${(analystMetrics.analyst_rating_buy / totalRatings) * 100}%` }}
                    >
                      {analystMetrics.analyst_rating_buy}
                    </div>
                  )}
                  {analystMetrics.analyst_rating_hold !== null && analystMetrics.analyst_rating_hold > 0 && (
                    <div
                      className="bg-yellow-400 flex items-center justify-center text-gray-800 text-xs font-medium"
                      style={{ width: `${(analystMetrics.analyst_rating_hold / totalRatings) * 100}%` }}
                    >
                      {analystMetrics.analyst_rating_hold}
                    </div>
                  )}
                  {analystMetrics.analyst_rating_sell !== null && analystMetrics.analyst_rating_sell > 0 && (
                    <div
                      className="bg-red-400 flex items-center justify-center text-white text-xs font-medium"
                      style={{ width: `${(analystMetrics.analyst_rating_sell / totalRatings) * 100}%` }}
                    >
                      {analystMetrics.analyst_rating_sell}
                    </div>
                  )}
                  {analystMetrics.analyst_rating_strong_sell !== null && analystMetrics.analyst_rating_strong_sell > 0 && (
                    <div
                      className="bg-red-600 flex items-center justify-center text-white text-xs font-medium"
                      style={{ width: `${(analystMetrics.analyst_rating_strong_sell / totalRatings) * 100}%` }}
                    >
                      {analystMetrics.analyst_rating_strong_sell}
                    </div>
                  )}
                </div>
                <div className="flex justify-between text-[10px] text-ic-text-dim">
                  <span>Strong Buy</span>
                  <span>Buy</span>
                  <span>Hold</span>
                  <span>Sell</span>
                </div>
              </div>
            </div>
          )}

          {/* Individual Analyst Ratings (if available) */}
          {ratings.length > 0 && (
            <div>
              <h4 className="text-sm font-medium text-ic-text-secondary mb-3">
                Recent Ratings
              </h4>
              <div className="space-y-2">
                {ratings.slice(0, 5).map((rating, index) => (
                  <div key={index} className="flex justify-between items-center p-2 bg-ic-bg-secondary rounded-lg">
                    <div>
                      <div className="text-sm font-medium text-ic-text-primary">{rating.firm}</div>
                      <div className="text-xs text-ic-text-dim">{formatDate(rating.ratingDate)}</div>
                    </div>
                    <div className="text-right">
                      <div className={`inline-flex px-2 py-0.5 text-xs font-medium rounded-full ${getRatingColor(rating.rating)}`}>
                        {rating.rating}
                      </div>
                      {rating.priceTarget && (
                        <div className="text-xs text-ic-text-secondary mt-0.5">
                          PT: ${safeToFixed(rating.priceTarget, 0)}
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      ) : (
        <div className="text-center py-8">
          <UserGroupIcon className="h-12 w-12 text-ic-text-dim mx-auto mb-4" />
          <p className="text-ic-text-muted">No analyst data available</p>
        </div>
      )}
    </div>
  );
}
