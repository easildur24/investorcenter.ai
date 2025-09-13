'use client';

import { useEffect, useState } from 'react';
import { useRealTimePrice } from '@/lib/hooks/useRealTimePrice';
import { ArrowTrendingUpIcon, ArrowTrendingDownIcon } from '@heroicons/react/24/outline';

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
  
  const { priceData } = useRealTimePrice({ 
    symbol, 
    enabled: initialData.market.shouldUpdateRealtime 
  });

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
    const changePercentNum = parseFloat(changePercent) * 100;
    
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
    if (change > 0) return 'text-green-600';
    if (change < 0) return 'text-red-600';
    return 'text-gray-600';
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
    <div className="bg-white">
      {/* Compact header like CoinMarketCap */}
      <div className="flex items-center space-x-3 mb-6">
        <h1 className="text-2xl font-bold text-gray-900">{cryptoName} price</h1>
        <span className="text-lg font-semibold text-gray-600">{symbol.replace('X:', '')}</span>
        <span className="bg-orange-100 text-orange-800 px-2 py-1 rounded text-sm font-medium">
          #1
        </span>
      </div>

      {/* Large price with change - CoinMarketCap style */}
      <div className="flex items-baseline space-x-6 mb-8">
        <div className={`text-5xl font-bold transition-colors duration-1000 ${
          flashColor === 'green' 
            ? 'text-green-600' 
            : flashColor === 'red' 
            ? 'text-red-600'
            : 'text-gray-900'
        }`}>
          ${formatPrice(currentPrice.price)}
        </div>
        
        <div className={`flex items-center space-x-2 ${getPriceChangeColor()}`}>
          {getPriceChangeIcon()}
          <span className="text-lg font-semibold">
            {formatChange(currentPrice.change, currentPrice.changePercent)}
          </span>
          <span className="text-gray-500 text-base">(1d)</span>
        </div>
      </div>

      {/* Statistics section - exact CoinMarketCap layout */}
      <div className="border-t border-gray-200 pt-6">
        <h2 className="text-xl font-semibold text-gray-900 mb-6">{cryptoName} statistics</h2>
        
        <div className="grid grid-cols-1 md:grid-cols-2 gap-x-12 gap-y-6">
          {/* Left column */}
          <div className="space-y-4">
            <div className="flex justify-between items-center">
              <span className="text-gray-600">Market cap</span>
              <div className="text-right">
                <div className="font-semibold text-gray-900">{formatMarketCap(currentPrice.price)}</div>
                <div className={`text-sm ${getPriceChangeColor()}`}>
                  {formatChange(currentPrice.change, currentPrice.changePercent)}
                </div>
              </div>
            </div>
            
            <div className="flex justify-between items-center">
              <span className="text-gray-600">Volume (24h)</span>
              <div className="text-right">
                <div className="font-semibold text-gray-900">{formatVolume(currentPrice.volume)}</div>
                <div className="text-sm text-green-600">5.67%</div>
              </div>
            </div>
            
            <div className="flex justify-between items-center">
              <span className="text-gray-600">FDV</span>
              <div className="font-semibold text-gray-900">$2.43T</div>
            </div>
            
            <div className="flex justify-between items-center">
              <span className="text-gray-600">Vol/Mkt Cap (24h)</span>
              <div className="font-semibold text-gray-900">2.14%</div>
            </div>
          </div>

          {/* Right column */}
          <div className="space-y-4">
            <div className="flex justify-between items-center">
              <span className="text-gray-600">Total supply</span>
              <div className="font-semibold text-gray-900">19.91M {symbol.replace('X:', '')}</div>
            </div>
            
            <div className="flex justify-between items-center">
              <span className="text-gray-600">Max. supply</span>
              <div className="font-semibold text-gray-900">21M {symbol.replace('X:', '')}</div>
            </div>
            
            <div className="flex justify-between items-center">
              <span className="text-gray-600">Circulating supply</span>
              <div className="text-right">
                <div className="font-semibold text-gray-900">19.91M {symbol.replace('X:', '')}</div>
                <div className="text-sm text-gray-500">94.86%</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Price performance section */}
      <div className="border-t border-gray-200 mt-6 pt-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Price performance</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div>
            <div className="text-sm text-gray-500 mb-2">24h</div>
            <div className="flex justify-between items-center">
              <div>
                <div className="text-sm text-gray-500">Low</div>
                <div className="font-semibold">${formatPrice(currentPrice.low)}</div>
              </div>
              <div className="flex-1 mx-4">
                <div className="h-2 bg-gray-200 rounded-full relative">
                  <div className="absolute h-2 bg-gradient-to-r from-red-400 to-green-400 rounded-full w-full"></div>
                  <div 
                    className="absolute w-3 h-3 bg-gray-900 rounded-full -mt-0.5"
                    style={{
                      left: `${((parseFloat(currentPrice.price) - parseFloat(currentPrice.low)) / 
                               (parseFloat(currentPrice.high) - parseFloat(currentPrice.low))) * 100}%`
                    }}
                  ></div>
                </div>
              </div>
              <div>
                <div className="text-sm text-gray-500">High</div>
                <div className="font-semibold">${formatPrice(currentPrice.high)}</div>
              </div>
            </div>
          </div>
          
          <div>
            <div className="text-sm text-gray-500 mb-2">All-time high</div>
            <div className="text-lg font-semibold text-gray-900">$124,457.12</div>
            <div className="text-sm text-red-600">-7% (1 month ago)</div>
          </div>
        </div>
      </div>
    </div>
  );
}
