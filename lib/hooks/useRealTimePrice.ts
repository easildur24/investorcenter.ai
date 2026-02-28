'use client';

import { useState, useEffect, useRef } from 'react';
import { API_BASE_URL } from '@/lib/api';
import { tickers } from '@/lib/api/routes';

interface UseRealTimePriceProps {
  symbol: string;
  enabled?: boolean;
}

interface PriceData {
  price: string;
  change: string;
  changePercent: string;
  volume: number;
  lastUpdated: string;
}

export type MarketSession = 'regular' | 'pre_market' | 'after_hours' | 'closed';

interface RegularCloseData {
  price: string;
  change?: string;
  changePercent?: string;
}

export function useRealTimePrice({ symbol, enabled = true }: UseRealTimePriceProps) {
  const [priceData, setPriceData] = useState<PriceData | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [session, setSession] = useState<MarketSession>('closed');
  const [regularClose, setRegularClose] = useState<RegularCloseData | null>(null);
  const [isCrypto, setIsCrypto] = useState(false);
  const [updateInterval, setUpdateInterval] = useState(5000);
  const intervalRef = useRef<NodeJS.Timeout>();

  useEffect(() => {
    if (!enabled || !symbol) return;

    const fetchPrice = async () => {
      try {
        const response = await fetch(`${API_BASE_URL}${tickers.price(symbol)}`);

        if (!response.ok) {
          throw new Error(`Failed to fetch price: ${response.status}`);
        }

        const result = await response.json();
        const isCryptoAsset = result.data.assetType === 'crypto';

        setIsCrypto(isCryptoAsset);

        // Parse session from backend
        const backendSession = result.market?.session as MarketSession | undefined;
        if (isCryptoAsset) {
          setSession('regular');
        } else if (backendSession) {
          setSession(backendSession);
        } else {
          // Fallback for old response format
          setSession(result.market?.isOpen ? 'regular' : 'closed');
        }

        // Parse regular close data (present during extended hours)
        if (result.market?.regularClose) {
          setRegularClose(result.market.regularClose);
        } else {
          setRegularClose(null);
        }

        // API returns updateInterval in seconds — convert to ms with a 1s safety floor
        const rawInterval = isCryptoAsset ? 5 : (result.market?.updateInterval ?? 15);
        setUpdateInterval(Math.max(rawInterval * 1000, 1000));

        setPriceData({
          price: result.data.price,
          change: result.data.change,
          changePercent: result.data.changePercent,
          volume: result.data.volume,
          lastUpdated: result.data.lastUpdated,
        });

        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch price');
      }
    };

    // Initial fetch
    fetchPrice();

    // Set up polling
    intervalRef.current = setInterval(fetchPrice, updateInterval);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [symbol, enabled, updateInterval]);

  return {
    priceData,
    error,
    session,
    regularClose,
    isMarketOpen: session === 'regular',
    isCrypto,
    updateInterval,
  };
}
