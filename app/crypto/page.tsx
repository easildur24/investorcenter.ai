'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { ArrowTrendingUpIcon, ArrowTrendingDownIcon } from '@heroicons/react/24/outline';

interface CryptoPrice {
  symbol: string;
  price: string;
  change: string;
  changePercent: string;
  volume: number;
  high: string;
  low: string;
  timestamp: number;
}

interface CryptoPageData {
  data: CryptoPrice[];
  meta: {
    page: number;
    perPage: number;
    total: number;
    totalPages: number;
    timestamp: string;
    source: string;
  };
}

export default function CryptoPage() {
  const [cryptoData, setCryptoData] = useState<CryptoPageData | null>(null);
  const [loading, setLoading] = useState(true);
  const [currentPage, setCurrentPage] = useState(1);
  const [error, setError] = useState<string | null>(null);
  const [previousPrices, setPreviousPrices] = useState<Map<string, number>>(new Map());
  const [flashColors, setFlashColors] = useState<Map<string, 'green' | 'red' | null>>(new Map());

  const fetchCryptoData = async (page: number = 1) => {
    try {
      setLoading(true);
      setError(null);

      const response = await fetch(`/api/v1/crypto?page=${page}`);
      const result = await response.json();

      if (result.data) {
        // Track price changes for flash animation
        const newFlashColors = new Map<string, 'green' | 'red' | null>();

        result.data.forEach((crypto: CryptoPrice) => {
          const currentPrice = parseFloat(crypto.price);
          const previousPrice = previousPrices.get(crypto.symbol);

          if (previousPrice !== undefined && previousPrice !== currentPrice) {
            const flashColor = currentPrice > previousPrice ? 'green' : 'red';
            newFlashColors.set(crypto.symbol, flashColor);

            // Clear flash after 1 second
            setTimeout(() => {
              setFlashColors(prev => {
                const updated = new Map(prev);
                updated.set(crypto.symbol, null);
                return updated;
              });
            }, 1000);
          }

          // Update previous prices
          setPreviousPrices(prev => {
            const updated = new Map(prev);
            updated.set(crypto.symbol, currentPrice);
            return updated;
          });
        });

        setFlashColors(newFlashColors);
        setCryptoData(result);
        setCurrentPage(page);
      } else {
        setError('Failed to load crypto data');
      }
    } catch (err) {
      setError('Failed to fetch crypto data');
      console.error('Error fetching crypto data:', err);
    } finally {
      setLoading(false);
    }
  };

  // Initial load
  useEffect(() => {
    fetchCryptoData(1);
  }, []);

  // Auto-refresh every 5 seconds
  useEffect(() => {
    const interval = setInterval(() => {
      if (!loading) {
        fetchCryptoData(currentPage);
      }
    }, 5000);

    return () => clearInterval(interval);
  }, [currentPage, loading]);

  const formatPrice = (price: string) => {
    const num = parseFloat(price);
    if (num >= 1) {
      return `$${num.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
    } else {
      return `$${num.toFixed(6)}`;
    }
  };

  const formatMarketCap = (symbol: string, price: string) => {
    // Rough market cap estimates for major cryptos (price × estimated circulating supply)
    const priceNum = parseFloat(price);
    let mcap = 0;

    const cleanSymbol = symbol.replace('X:', '');

    // Major crypto market caps (approximate circulating supply)
    switch (cleanSymbol) {
      case 'BTCUSD': mcap = priceNum * 19.8e6; break;  // ~19.8M BTC
      case 'ETHUSD': mcap = priceNum * 120e6; break;   // ~120M ETH
      case 'SOLUSD': mcap = priceNum * 470e6; break;   // ~470M SOL
      case 'XRPUSD': mcap = priceNum * 56e9; break;    // ~56B XRP
      case 'DOGEUSD': mcap = priceNum * 147e9; break;  // ~147B DOGE
      case 'ADAUSD': mcap = priceNum * 35e9; break;    // ~35B ADA
      case 'LTCUSD': mcap = priceNum * 75e6; break;    // ~75M LTC
      case 'LINKUSD': mcap = priceNum * 600e6; break;  // ~600M LINK
      case 'AVAXUSD': mcap = priceNum * 400e6; break;  // ~400M AVAX
      case 'MATICUSD': mcap = priceNum * 10e9; break;  // ~10B MATIC
      default:
        // For other cryptos, use volume as a rough proxy (not accurate but better than nothing)
        mcap = priceNum * 1e6; // Assume 1M supply as default
    }

    if (mcap >= 1e12) return `$${(mcap / 1e12).toFixed(2)}T`;
    if (mcap >= 1e9) return `$${(mcap / 1e9).toFixed(2)}B`;
    if (mcap >= 1e6) return `$${(mcap / 1e6).toFixed(2)}M`;
    return `$${mcap.toFixed(0)}`;
  };

  const formatVolume = (volume: number) => {
    if (volume >= 1e9) return `$${(volume / 1e9).toFixed(2)}B`;
    if (volume >= 1e6) return `$${(volume / 1e6).toFixed(2)}M`;
    if (volume >= 1e3) return `$${(volume / 1e3).toFixed(2)}K`;
    return `$${volume.toFixed(0)}`;
  };

  const goToPage = (page: number) => {
    fetchCryptoData(page);
  };

  if (loading && !cryptoData) {
    return (
      <div className="min-h-screen bg-ic-bg-primary">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <div className="flex justify-center items-center h-64">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-ic-blue"></div>
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-ic-bg-primary">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <div className="text-center">
            <h1 className="text-2xl font-bold text-ic-text-primary mb-4">Error Loading Crypto Data</h1>
            <p className="text-ic-text-muted">{error}</p>
            <button
              onClick={() => fetchCryptoData(currentPage)}
              className="mt-4 px-4 py-2 bg-ic-blue text-ic-text-primary rounded-md hover:bg-ic-blue/90"
            >
              Retry
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-ic-bg-primary">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-3xl font-bold text-ic-text-primary">Cryptocurrency Prices</h1>
            <p className="text-ic-text-muted mt-2">
              Live prices for {cryptoData?.meta.total || 0} cryptocurrencies, updated every 5 seconds
            </p>
          </div>

          {/* Pagination - Top Right */}
          {cryptoData && cryptoData.meta.totalPages > 1 && (
            <div className="flex items-center space-x-2">
              <span className="text-sm text-ic-text-muted">
                Page {cryptoData.meta.page} of {cryptoData.meta.totalPages}
              </span>
              <button
                onClick={() => goToPage(cryptoData.meta.page - 1)}
                disabled={cryptoData.meta.page === 1}
                className="px-3 py-1 text-sm font-medium text-ic-text-muted bg-ic-surface border border-ic-border rounded-md hover:bg-ic-surface-hover disabled:opacity-50 disabled:cursor-not-allowed"
              >
                ←
              </button>
              <button
                onClick={() => goToPage(cryptoData.meta.page + 1)}
                disabled={cryptoData.meta.page === cryptoData.meta.totalPages}
                className="px-3 py-1 text-sm font-medium text-ic-text-muted bg-ic-surface border border-ic-border rounded-md hover:bg-ic-surface-hover disabled:opacity-50 disabled:cursor-not-allowed"
              >
                →
              </button>
            </div>
          )}
        </div>

        {/* Crypto Table */}
        <div className="bg-ic-surface shadow-sm rounded-lg overflow-hidden">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-ic-border">
              <thead className="bg-ic-bg-secondary">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-ic-text-muted uppercase tracking-wider">
                    #
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-ic-text-muted uppercase tracking-wider">
                    Name
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-ic-text-muted uppercase tracking-wider">
                    Price
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-ic-text-muted uppercase tracking-wider">
                    24h Change
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-ic-text-muted uppercase tracking-wider">
                    Market Cap
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-ic-text-muted uppercase tracking-wider">
                    Volume (24h)
                  </th>
                </tr>
              </thead>
              <tbody className="bg-ic-surface divide-y divide-ic-border">
                {cryptoData?.data.map((crypto, index) => {
                  const rank = (cryptoData.meta.page - 1) * cryptoData.meta.perPage + index + 1;
                  const changeNum = parseFloat(crypto.changePercent);
                  const isPositive = changeNum >= 0;
                  const cleanSymbol = crypto.symbol.replace('X:', '');

                  return (
                    <tr key={crypto.symbol} className="hover:bg-ic-surface-hover">
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-ic-text-muted">
                        {rank}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Link
                          href={`/ticker/${crypto.symbol}`}
                          className="flex items-center hover:text-ic-blue"
                        >
                          <div>
                            <div className="text-sm font-medium text-ic-text-primary">
                              {cleanSymbol}
                            </div>
                            <div className="text-sm text-ic-text-muted">
                              {crypto.symbol}
                            </div>
                          </div>
                        </Link>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                        <span className={`transition-colors duration-1000 ${
                          flashColors.get(crypto.symbol) === 'green' ? 'text-ic-positive' :
                          flashColors.get(crypto.symbol) === 'red' ? 'text-ic-negative' :
                          'text-ic-text-primary'
                        }`}>
                          {formatPrice(crypto.price)}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-right text-sm">
                        <div className={`flex items-center justify-end space-x-1 ${
                          isPositive ? 'text-ic-positive' : 'text-ic-negative'
                        }`}>
                          {isPositive ? (
                            <ArrowTrendingUpIcon className="h-4 w-4" />
                          ) : (
                            <ArrowTrendingDownIcon className="h-4 w-4" />
                          )}
                          <span className="font-medium">
                            {isPositive ? '+' : ''}{changeNum.toFixed(2)}%
                          </span>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-right text-sm text-ic-text-primary">
                        {formatMarketCap(crypto.symbol, crypto.price)}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-right text-sm text-ic-text-primary">
                        {formatVolume(crypto.volume)}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>

        {/* Loading indicator during refresh */}
        {loading && cryptoData && (
          <div className="fixed bottom-4 right-4 bg-ic-blue text-ic-text-primary px-4 py-2 rounded-md shadow-lg">
            <div className="flex items-center space-x-2">
              <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
              <span className="text-sm">Updating prices...</span>
            </div>
          </div>
        )}

        {/* Bottom pagination */}
        {cryptoData && cryptoData.meta.totalPages > 1 && (
          <div className="flex items-center justify-center space-x-2 mt-8">
            <button
              onClick={() => goToPage(cryptoData.meta.page - 1)}
              disabled={cryptoData.meta.page === 1}
              className="px-4 py-2 text-sm font-medium text-ic-text-muted bg-ic-surface border border-ic-border rounded-md hover:bg-ic-surface-hover disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Previous
            </button>

            <div className="flex space-x-1">
              {Array.from({ length: Math.min(cryptoData.meta.totalPages, 5) }, (_, i) => {
                let pageNumber;
                if (cryptoData.meta.totalPages <= 5) {
                  pageNumber = i + 1;
                } else if (cryptoData.meta.page <= 3) {
                  pageNumber = i + 1;
                } else if (cryptoData.meta.page >= cryptoData.meta.totalPages - 2) {
                  pageNumber = cryptoData.meta.totalPages - 4 + i;
                } else {
                  pageNumber = cryptoData.meta.page - 2 + i;
                }

                return (
                  <button
                    key={pageNumber}
                    onClick={() => goToPage(pageNumber)}
                    className={`px-3 py-2 text-sm font-medium rounded-md ${
                      cryptoData.meta.page === pageNumber
                        ? 'bg-ic-blue text-ic-text-primary'
                        : 'text-ic-text-secondary bg-ic-surface border border-ic-border hover:bg-ic-surface-hover'
                    }`}
                  >
                    {pageNumber}
                  </button>
                );
              })}
            </div>

            <button
              onClick={() => goToPage(cryptoData.meta.page + 1)}
              disabled={cryptoData.meta.page === cryptoData.meta.totalPages}
              className="px-4 py-2 text-sm font-medium text-ic-text-muted bg-ic-surface border border-ic-border rounded-md hover:bg-ic-surface-hover disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Next
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
