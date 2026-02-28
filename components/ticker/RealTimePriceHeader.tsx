'use client';

import { useEffect, useState } from 'react';
import Image from 'next/image';
import { useRealTimePrice, MarketSession } from '@/lib/hooks/useRealTimePrice';
import { ArrowTrendingUpIcon, ArrowTrendingDownIcon } from '@heroicons/react/24/outline';
import { logos } from '@/lib/api/routes';
import { API_BASE_URL } from '@/lib/api';

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
      logoUrl?: string;
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
      session?: string;
      shouldUpdateRealtime: boolean;
      updateInterval: number;
      regularClose?: {
        price: string;
        change?: string;
        changePercent?: string;
      };
    };
  };
}

export default function RealTimePriceHeader({ symbol, initialData }: RealTimePriceHeaderProps) {
  const [currentPrice, setCurrentPrice] = useState(initialData.price);
  const [lastUpdate, setLastUpdate] = useState(new Date(initialData.price.lastUpdated));
  const [previousPrice, setPreviousPrice] = useState<string>(initialData.price.price);
  const [flashColor, setFlashColor] = useState<'green' | 'red' | null>(null);

  const { priceData, error, session, regularClose, isMarketOpen, isCrypto, updateInterval } = useRealTimePrice({
    symbol,
    enabled: initialData.market.shouldUpdateRealtime,
  });

  // Use initial session from server data until first poll completes
  const activeSession: MarketSession = session || (initialData.market.session as MarketSession) || (initialData.market.status === 'open' ? 'regular' : 'closed');
  const activeRegularClose = regularClose || initialData.market.regularClose || null;
  const isExtendedHours = activeSession === 'after_hours' || activeSession === 'pre_market';

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
    const changePercentNum = parseFloat(changePercent);

    const prefix = changeNum >= 0 ? '+' : '';
    return `${prefix}${changeNum.toFixed(2)} (${prefix}${changePercentNum.toFixed(2)}%)`;
  };

  const getSessionDisplay = () => {
    if (isCrypto) {
      return <span className="text-ic-positive text-sm">• Live (24/7)</span>;
    }

    switch (activeSession) {
      case 'regular':
        return <span className="text-ic-positive text-sm">• Market Open</span>;
      case 'pre_market':
        return <span className="text-blue-500 dark:text-blue-400 text-sm">• Pre-Market</span>;
      case 'after_hours':
        return <span className="text-amber-500 dark:text-amber-400 text-sm">• After Hours</span>;
      default:
        return <span className="text-ic-text-dim text-sm">• Market Closed</span>;
    }
  };

  const getPriceChangeColor = (change: string) => {
    const changeNum = parseFloat(change);
    if (changeNum > 0) return 'text-ic-positive';
    if (changeNum < 0) return 'text-ic-negative';
    return 'text-ic-text-muted';
  };

  const getPriceChangeIcon = (change: string) => {
    const changeNum = parseFloat(change);
    if (changeNum > 0) return <ArrowTrendingUpIcon className="w-4 h-4" />;
    if (changeNum < 0) return <ArrowTrendingDownIcon className="w-4 h-4" />;
    return null;
  };

  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-4">
        {/* Company Logo */}
        {initialData.stock.logoUrl ? (
          <div className="w-12 h-12 relative flex-shrink-0 bg-white dark:bg-gray-800 rounded-lg p-1">
            <Image
              src={`${API_BASE_URL}${logos.bySymbol(initialData.stock.symbol)}`}
              alt={`${initialData.stock.name} logo`}
              fill
              className="object-contain"
              unoptimized
            />
          </div>
        ) : (
          <div className="w-12 h-12 flex-shrink-0 bg-gradient-to-br from-blue-500 to-blue-700 rounded-lg flex items-center justify-center">
            <span className="text-lg font-bold text-white">
              {initialData.stock.symbol.length <= 2
                ? initialData.stock.symbol
                : initialData.stock.symbol.charAt(0)}
            </span>
          </div>
        )}
        <div>
          <h1 className="text-3xl font-bold text-ic-text-primary">
            {initialData.stock.name} ({initialData.stock.symbol})
          </h1>
          <div className="flex items-center space-x-2 text-ic-text-muted">
            <span>{initialData.stock.exchange}</span>
            {initialData.stock.sector && (
              <>
                <span>•</span>
                <span>{initialData.stock.sector}</span>
              </>
            )}
            {getSessionDisplay()}
            {initialData.stock.isCrypto && (
              <>
                <span>•</span>
                <span className="text-ic-blue font-medium">Crypto</span>
              </>
            )}
          </div>
        </div>
      </div>

      <div className="text-right">
        {isExtendedHours && activeRegularClose ? (
          <>
            {/* Close price — hero */}
            <div className="flex items-center justify-end">
              <div className="text-3xl font-bold text-ic-text-primary">
                ${formatPrice(activeRegularClose.price)}
              </div>
            </div>
            {activeRegularClose.change && activeRegularClose.changePercent ? (
              <div className={`flex items-center justify-end space-x-1 text-sm ${getPriceChangeColor(activeRegularClose.change)}`}>
                {getPriceChangeIcon(activeRegularClose.change)}
                <span>{formatChange(activeRegularClose.change, activeRegularClose.changePercent)}</span>
              </div>
            ) : (
              <div className="text-sm text-ic-text-dim text-right">Previous close</div>
            )}

            {/* Extended hours — compact single line */}
            <div className="flex items-center justify-end gap-2 mt-1">
              <span className="text-xs text-ic-text-dim">
                {activeSession === 'after_hours' ? 'After hours' : 'Pre-market'}
              </span>
              <span
                className={`text-sm font-medium transition-colors duration-1000 ${
                  flashColor === 'green'
                    ? 'text-ic-positive'
                    : flashColor === 'red'
                      ? 'text-ic-negative'
                      : 'text-ic-text-secondary'
                }`}
              >
                ${formatPrice(currentPrice.price)}
              </span>
              <span className={`text-xs ${getPriceChangeColor(currentPrice.change)}`}>
                {formatChange(currentPrice.change, currentPrice.changePercent)}
              </span>
            </div>
          </>
        ) : (
          <>
            {/* Regular hours / closed — single price (current behavior) */}
            <div className="flex items-center justify-end">
              <div
                className={`text-3xl font-bold transition-colors duration-1000 ${
                  flashColor === 'green'
                    ? 'text-ic-positive'
                    : flashColor === 'red'
                      ? 'text-ic-negative'
                      : 'text-ic-text-primary'
                }`}
              >
                ${formatPrice(currentPrice.price)}
              </div>
            </div>

            <div className={`flex items-center justify-end space-x-1 text-sm ${getPriceChangeColor(currentPrice.change)}`}>
              {getPriceChangeIcon(currentPrice.change)}
              <span>{formatChange(currentPrice.change, currentPrice.changePercent)}</span>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
