'use client';

import { useEffect, useState } from 'react';
import { useRealTimePrice } from '@/lib/hooks/useRealTimePrice';
import { ArrowTrendingUpIcon, ArrowTrendingDownIcon } from '@heroicons/react/24/outline';

interface RealTimePriceHeaderProps {
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
      lastUpdated: string;
    };
    market: {
      status: string;
      shouldUpdateRealtime: boolean;
      updateInterval: number;
    };
  };
}

export default function RealTimePriceHeader({ symbol, initialData }: RealTimePriceHeaderProps) {
  const [currentPrice, setCurrentPrice] = useState(initialData.price);
  const [lastUpdate, setLastUpdate] = useState(new Date(initialData.price.lastUpdated));
  const [previousPrice, setPreviousPrice] = useState<string>(initialData.price.price);
  const [flashColor, setFlashColor] = useState<'green' | 'red' | null>(null);
  
  const { 
    priceData, 
    error, 
    isMarketOpen, 
    isCrypto,
    updateInterval 
  } = useRealTimePrice({ 
    symbol, 
    enabled: initialData.market.shouldUpdateRealtime 
  });

  // Update current price when real-time data comes in
  useEffect(() => {
    if (priceData) {
      const newPrice = priceData.price;
      const oldPrice = parseFloat(previousPrice);
      const newPriceNum = parseFloat(newPrice);
      
      // Determine flash color based on price change
      if (newPriceNum > oldPrice) {
        setFlashColor('green');
      } else if (newPriceNum < oldPrice) {
        setFlashColor('red');
      }
      
      // Update price data
      setCurrentPrice({
        price: priceData.price,
        change: priceData.change,
        changePercent: priceData.changePercent,
        volume: priceData.volume,
        lastUpdated: priceData.lastUpdated,
      });
      setLastUpdate(new Date(priceData.lastUpdated));
      setPreviousPrice(newPrice);
      
      // Clear flash after animation
      if (newPriceNum !== oldPrice) {
        setTimeout(() => setFlashColor(null), 1000); // 1 second flash
      }
    }
  }, [priceData, previousPrice]);

  const formatPrice = (price: string) => {
    const num = parseFloat(price);
    if (initialData.stock.isCrypto) {
      // For crypto, show more decimal places if price is low
      if (num < 1) {
        return num.toFixed(6);
      } else if (num < 100) {
        return num.toFixed(4);
      } else {
        return num.toFixed(2);
      }
    }
    return num.toFixed(2);
  };

  const formatChange = (change: string, changePercent: string) => {
    const changeNum = parseFloat(change);
    const changePercentNum = parseFloat(changePercent) * 100;
    
    const prefix = changeNum >= 0 ? '+' : '';
    return `${prefix}${changeNum.toFixed(2)} (${prefix}${changePercentNum.toFixed(2)}%)`;
  };

  const getMarketStatusDisplay = () => {
    if (isCrypto) {
      return (
        <span className="text-green-600 text-sm">
          • Live (24/7)
        </span>
      );
    }
    
    if (isMarketOpen) {
      return (
        <span className="text-green-600 text-sm">
          • Market Open
        </span>
      );
    }
    
    return (
      <span className="text-gray-500 text-sm">
        • Market Closed
      </span>
    );
  };

  const getPriceChangeColor = () => {
    const change = parseFloat(currentPrice.change);
    if (change > 0) return 'text-green-600';
    if (change < 0) return 'text-red-600';
    return 'text-gray-600';
  };

  const getPriceChangeIcon = () => {
    const change = parseFloat(currentPrice.change);
    if (change > 0) return <ArrowTrendingUpIcon className="w-4 h-4" />;
    if (change < 0) return <ArrowTrendingDownIcon className="w-4 h-4" />;
    return null;
  };

  return (
    <div className="flex items-center justify-between">
      <div>
        <h1 className="text-3xl font-bold text-gray-900">
          {initialData.stock.name} ({initialData.stock.symbol})
        </h1>
        <div className="flex items-center space-x-2 text-gray-600">
          <span>{initialData.stock.exchange}</span>
          {initialData.stock.sector && (
            <>
              <span>•</span>
              <span>{initialData.stock.sector}</span>
            </>
          )}
          {getMarketStatusDisplay()}
          {initialData.stock.isCrypto && (
            <>
              <span>•</span>
              <span className="text-blue-600 font-medium">Crypto</span>
            </>
          )}
        </div>
      </div>
      
      <div className="text-right">
        <div className="flex items-center justify-end">
          <div className={`text-3xl font-bold transition-colors duration-1000 ${
            flashColor === 'green' 
              ? 'text-green-600' 
              : flashColor === 'red' 
              ? 'text-red-600'
              : 'text-gray-900'
          }`}>
            ${formatPrice(currentPrice.price)}
          </div>
        </div>
        
        <div className={`flex items-center justify-end space-x-1 text-sm ${getPriceChangeColor()}`}>
          {getPriceChangeIcon()}
          <span>{formatChange(currentPrice.change, currentPrice.changePercent)}</span>
        </div>
        
      </div>
    </div>
  );
}
