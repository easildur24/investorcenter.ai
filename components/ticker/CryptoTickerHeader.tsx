'use client';

import { useEffect, useState } from 'react';
import { useRealTimePrice } from '@/lib/hooks/useRealTimePrice';
import { ArrowTrendingUpIcon, ArrowTrendingDownIcon } from '@heroicons/react/24/outline';
import HybridChart from './HybridChart';
import { tickers } from '@/lib/api/routes';
import { API_BASE_URL } from '@/lib/api';

interface CryptoTickerHeaderProps {
  symbol: string;
  initialData: {
    stock: {
      symbol: string;
      name: string;
      exchange: string;
      sector: string;
      assetType: string;
      isCrypto: boolean;
    };
    price: {
      price: string;
      change: string;
      changePercent: string;
      volume: number;
      high: string;
      low: string;
      lastUpdated: string;
    };
    market: {
      status: string;
      shouldUpdateRealtime: boolean;
      updateInterval: number;
    };
  };
}

export default function CryptoTickerHeader({ symbol, initialData }: CryptoTickerHeaderProps) {
  const [currentPrice, setCurrentPrice] = useState(initialData.price);
  const [previousPrice, setPreviousPrice] = useState<string>(initialData.price.price);
  const [flashColor, setFlashColor] = useState<'green' | 'red' | null>(null);
  const [chartData, setChartData] = useState<any>(null);
  const [chartLoading, setChartLoading] = useState(true);

  // Decode URL-encoded symbol for display
  const decodedSymbol = decodeURIComponent(symbol);
  const displaySymbol = decodedSymbol.replace('X:', '');

  const { priceData } = useRealTimePrice({
    symbol,
    enabled: initialData.market.shouldUpdateRealtime,
  });

  // Fetch chart data on mount
  useEffect(() => {
    const fetchChartData = async () => {
      try {
        setChartLoading(true);
        const response = await fetch(`${API_BASE_URL}${tickers.chart(symbol)}?period=1Y`);
        const data = await response.json();

        if (data.success && data.data) {
          setChartData(data.data);
        }
      } catch (error) {
        console.error('Failed to fetch chart data:', error);
      } finally {
        setChartLoading(false);
      }
    };

    fetchChartData();
  }, [symbol]);

  // Update current price when real-time data comes in
  useEffect(() => {
    if (priceData) {
      const newPrice = priceData.price;
      const oldPrice = parseFloat(previousPrice);
      const newPriceNum = parseFloat(newPrice);

      // Flash animation for price changes
      if (newPriceNum > oldPrice) {
        setFlashColor('green');
      } else if (newPriceNum < oldPrice) {
        setFlashColor('red');
      }

      setCurrentPrice({
        price: priceData.price,
        change: priceData.change,
        changePercent: priceData.changePercent,
        volume: priceData.volume,
        lastUpdated: priceData.lastUpdated,
        high: currentPrice.high,
        low: currentPrice.low,
      });
      setPreviousPrice(newPrice);

      // Clear flash after animation
      if (newPriceNum !== oldPrice) {
        setTimeout(() => setFlashColor(null), 1000);
      }
    }
  }, [priceData, previousPrice]);

  const formatPrice = (price: string) => {
    const num = parseFloat(price);
    if (num < 1) {
      return num.toFixed(6);
    } else if (num < 100) {
      return num.toFixed(4);
    } else {
      return num.toFixed(2);
    }
  };

  const formatChange = (change: string, changePercent: string) => {
    const changeNum = parseFloat(change);
    const changePercentNum = parseFloat(changePercent); // Backend already returns as percentage

    const prefix = changeNum >= 0 ? '+' : '';
    return `${prefix}${changePercentNum.toFixed(2)}%`;
  };

  const formatVolume = (volume: number) => {
    if (volume >= 1e9) return `$${(volume / 1e9).toFixed(2)}B`;
    if (volume >= 1e6) return `$${(volume / 1e6).toFixed(2)}M`;
    if (volume >= 1e3) return `$${(volume / 1e3).toFixed(2)}K`;
    return `$${volume.toFixed(2)}`;
  };

  const formatMarketCap = (price: string, supply: number = 19919928) => {
    const priceNum = parseFloat(price);
    const marketCap = priceNum * supply;
    if (marketCap >= 1e12) return `$${(marketCap / 1e12).toFixed(2)}T`;
    if (marketCap >= 1e9) return `$${(marketCap / 1e9).toFixed(2)}B`;
    return `$${(marketCap / 1e6).toFixed(2)}M`;
  };

  const getPriceChangeColor = () => {
    const change = parseFloat(currentPrice.change);
    if (change > 0) return 'text-ic-positive';
    if (change < 0) return 'text-ic-negative';
    return 'text-ic-text-muted';
  };

  const getPriceChangeIcon = () => {
    const change = parseFloat(currentPrice.change);
    if (change > 0) return <ArrowTrendingUpIcon className="w-5 h-5" />;
    if (change < 0) return <ArrowTrendingDownIcon className="w-5 h-5" />;
    return null;
  };

  // Extract crypto name without " - United States dollar"
  const cryptoName = initialData.stock.name.split(' - ')[0];

  return (
    <div className="bg-ic-surface">
      {/* EXACT CoinMarketCap header */}
      <div className="flex items-center space-x-3 mb-4">
        <span className="text-ic-text-muted text-lg">{displaySymbol}</span>
      </div>

      <div className="flex items-center space-x-4 mb-6">
        <h1 className="text-3xl font-bold text-ic-text-primary"># {cryptoName} price</h1>
        <span className="text-xl font-bold text-ic-text-secondary">{displaySymbol}</span>
        <span className="bg-orange-500 text-ic-text-primary px-2 py-1 rounded text-sm font-bold">
          #1
        </span>
        <button className="bg-ic-blue text-ic-text-primary px-3 py-1 rounded text-sm font-medium">
          6M
        </button>
      </div>

      {/* EXACT price display */}
      <div className="flex items-baseline space-x-4 mb-8">
        <div
          className={`text-6xl font-bold transition-colors duration-1000 ${
            flashColor === 'green'
              ? 'text-ic-positive'
              : flashColor === 'red'
                ? 'text-ic-negative'
                : 'text-ic-text-primary'
          }`}
        >
          ${formatPrice(currentPrice.price)}
        </div>

        <div className={`flex items-center space-x-1 text-xl ${getPriceChangeColor()}`}>
          {getPriceChangeIcon()}
          <span className="font-bold">
            {formatChange(currentPrice.change, currentPrice.changePercent)} (1d)
          </span>
        </div>
      </div>

      {/* Interactive Price Chart */}
      <div className="mb-8">
        <h2 className="text-xl font-semibold text-ic-text-primary mb-4">
          {cryptoName} to USD Chart
        </h2>
        {chartLoading ? (
          <div className="bg-ic-bg-secondary rounded-lg p-8 text-center">
            <div className="text-ic-text-muted">Loading chart data...</div>
            <div className="text-ic-text-dim text-sm mt-1">Please wait a moment.</div>
          </div>
        ) : chartData && chartData.dataPoints && chartData.dataPoints.length > 0 ? (
          <HybridChart
            symbol={symbol}
            initialData={chartData}
            currentPrice={parseFloat(currentPrice.price)}
          />
        ) : (
          <div className="bg-ic-bg-secondary rounded-lg p-8 text-center">
            <div className="text-ic-text-muted">Chart data temporarily unavailable</div>
            <div className="text-ic-text-dim text-sm mt-1">Please try again later.</div>
          </div>
        )}
      </div>

      {/* EXACT CoinMarketCap statistics section */}
      <div className="mb-8">
        <h2 className="text-xl font-semibold text-ic-text-primary mb-6">{cryptoName} statistics</h2>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-x-16 gap-y-6">
          {/* Left column - Financial metrics */}
          <div className="space-y-6">
            <div className="flex justify-between items-start">
              <span className="text-ic-text-muted font-medium">Market cap</span>
              <div className="text-right">
                <div className="text-xl font-bold text-ic-text-primary">
                  {formatMarketCap(currentPrice.price)}
                </div>
                <div className={`text-sm font-medium ${getPriceChangeColor()}`}>
                  {formatChange(currentPrice.change, currentPrice.changePercent)}
                </div>
              </div>
            </div>

            <div className="flex justify-between items-start">
              <span className="text-ic-text-muted font-medium">Volume (24h)</span>
              <div className="text-right">
                <div className="text-xl font-bold text-ic-text-primary">
                  {formatVolume(currentPrice.volume)}
                </div>
                <div className="text-sm font-medium text-ic-positive">5.67%</div>
              </div>
            </div>

            <div className="flex justify-between items-center">
              <span className="text-ic-text-muted font-medium">FDV</span>
              <div className="text-xl font-bold text-ic-text-primary">$2.43T</div>
            </div>

            <div className="flex justify-between items-center">
              <span className="text-ic-text-muted font-medium">Vol/Mkt Cap (24h)</span>
              <div className="text-xl font-bold text-ic-text-primary">2.14%</div>
            </div>
          </div>

          {/* Right column - Supply metrics */}
          <div className="space-y-6">
            <div className="flex justify-between items-center">
              <span className="text-ic-text-muted font-medium">Total supply</span>
              <div className="text-xl font-bold text-ic-text-primary">19.91M {displaySymbol}</div>
            </div>

            <div className="flex justify-between items-center">
              <span className="text-ic-text-muted font-medium">Max. supply</span>
              <div className="text-xl font-bold text-ic-text-primary">21M {displaySymbol}</div>
            </div>

            <div className="flex justify-between items-start">
              <span className="text-ic-text-muted font-medium">Circulating supply</span>
              <div className="text-right">
                <div className="text-xl font-bold text-ic-text-primary">19.91M {displaySymbol}</div>
                <div className="text-sm text-ic-text-muted font-medium">94.86%</div>
              </div>
            </div>

            {/* Additional CMC elements */}
            <div className="pt-4 border-t border-ic-border">
              <div className="flex justify-between items-center mb-3">
                <span className="text-ic-text-muted font-medium">Website</span>
                <div className="flex space-x-2">
                  <button className="text-ic-blue hover:text-ic-blue text-sm font-medium">
                    Website
                  </button>
                  <button className="text-ic-blue hover:text-ic-blue text-sm font-medium">
                    Whitepaper
                  </button>
                </div>
              </div>

              <div className="flex justify-between items-center mb-3">
                <span className="text-ic-text-muted font-medium">Socials</span>
                <div className="text-sm text-ic-text-muted">Rating 4.4 ⭐</div>
              </div>

              <div className="flex justify-between items-center">
                <span className="text-ic-text-muted font-medium">Explorers</span>
                <button className="text-ic-blue hover:text-ic-blue text-sm font-medium">
                  blockchain.info
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* BTC to USD converter - exact like CMC */}
      <div className="mb-8 p-6 bg-ic-bg-secondary rounded-lg">
        <h3 className="text-lg font-semibold text-ic-text-primary mb-4">
          {displaySymbol} to USD converter
        </h3>
        <div className="flex items-center space-x-4">
          <div className="flex-1">
            <div className="flex items-center space-x-2 bg-ic-surface border rounded-lg p-3">
              <span className="font-bold text-ic-text-secondary">{displaySymbol}</span>
              <input
                type="number"
                defaultValue="1"
                className="flex-1 border-none outline-none text-right font-semibold"
              />
            </div>
          </div>
          <span className="text-ic-text-dim">=</span>
          <div className="flex-1">
            <div className="flex items-center space-x-2 bg-ic-surface border rounded-lg p-3">
              <span className="font-bold text-ic-text-secondary">USD</span>
              <input
                type="text"
                value={formatPrice(currentPrice.price)}
                readOnly
                className="flex-1 border-none outline-none text-right font-semibold bg-ic-bg-secondary"
              />
            </div>
          </div>
        </div>
      </div>

      {/* Price performance - exact CMC layout */}
      <div className="mb-8">
        <h3 className="text-lg font-semibold text-ic-text-primary mb-6">Price performance</h3>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
          {/* 24h performance */}
          <div>
            <div className="text-base font-semibold text-ic-text-primary mb-4">24h</div>
            <div className="flex justify-between items-center mb-2">
              <div className="text-center">
                <div className="text-sm text-ic-text-muted">Low</div>
                <div className="font-bold text-ic-text-primary">
                  ${formatPrice(currentPrice.low)}
                </div>
              </div>
              <div className="text-center">
                <div className="text-sm text-ic-text-muted">High</div>
                <div className="font-bold text-ic-text-primary">
                  ${formatPrice(currentPrice.high)}
                </div>
              </div>
            </div>

            {/* 24h range slider */}
            <div className="relative h-2 bg-ic-bg-secondary rounded-full mb-4">
              <div className="absolute h-2 bg-gradient-to-r from-red-400 via-yellow-400 to-green-400 rounded-full w-full"></div>
              <div
                className="absolute w-4 h-4 bg-gray-800 rounded-full -mt-1 border-2 border-white shadow-lg"
                style={{
                  left: `${Math.max(
                    0,
                    Math.min(
                      100,
                      ((parseFloat(currentPrice.price) - parseFloat(currentPrice.low)) /
                        (parseFloat(currentPrice.high) - parseFloat(currentPrice.low))) *
                        100
                    )
                  )}%`,
                }}
              ></div>
            </div>
          </div>

          {/* All-time performance */}
          <div>
            <div className="mb-4">
              <div className="text-sm text-ic-text-muted mb-1">All-time high</div>
              <div className="text-sm text-ic-text-dim mb-2">Aug 14, 2025 (1 month ago)</div>
              <div className="text-xl font-bold text-ic-text-primary">$124,457.12</div>
              <div className="text-sm font-medium text-ic-negative">-7%</div>
            </div>

            <div>
              <div className="text-sm text-ic-text-muted mb-1">All-time low</div>
              <div className="text-sm text-ic-text-dim mb-2">Jul 14, 2010 (15 years ago)</div>
              <div className="text-xl font-bold text-ic-text-primary">$0.04865</div>
              <div className="text-sm font-medium text-ic-positive">+237,936,842.23%</div>
            </div>
          </div>
        </div>

        <button className="text-ic-blue hover:text-ic-blue text-sm font-medium mt-4">
          See historical data
        </button>
      </div>

      {/* Tags section - like CMC */}
      <div className="mb-8">
        <div className="text-sm text-ic-text-muted mb-2">Tags</div>
        <div className="flex flex-wrap gap-2">
          <span className="bg-blue-100 text-blue-800 px-3 py-1 rounded-full text-sm">
            Bitcoin Ecosystem
          </span>
          <span className="bg-blue-100 text-blue-800 px-3 py-1 rounded-full text-sm">Layer 1</span>
          <span className="bg-blue-100 text-blue-800 px-3 py-1 rounded-full text-sm">
            Store of Value
          </span>
          <button className="text-ic-blue text-sm font-medium">Show all</button>
        </div>
      </div>
    </div>
  );
}
