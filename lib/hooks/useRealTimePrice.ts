'use client';

import { useState, useEffect, useRef } from 'react';

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

// Check if US stock market is currently open (9:30 AM - 4:00 PM EST, Mon-Fri)
function isMarketCurrentlyOpen(): boolean {
  const now = new Date();

  // Convert to Eastern Time
  const estTime = new Date(now.toLocaleString('en-US', { timeZone: 'America/New_York' }));

  // Check if it's a weekday (0 = Sunday, 6 = Saturday)
  const day = estTime.getDay();
  if (day === 0 || day === 6) {
    return false;
  }

  // Market hours: 9:30 AM - 4:00 PM EST
  const hours = estTime.getHours();
  const minutes = estTime.getMinutes();
  const timeInMinutes = hours * 60 + minutes;

  const marketOpenMinutes = 9 * 60 + 30;  // 9:30 AM
  const marketCloseMinutes = 16 * 60;      // 4:00 PM

  return timeInMinutes >= marketOpenMinutes && timeInMinutes < marketCloseMinutes;
}

export function useRealTimePrice({ symbol, enabled = true }: UseRealTimePriceProps) {
  const [priceData, setPriceData] = useState<PriceData | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isMarketOpen, setIsMarketOpen] = useState(isMarketCurrentlyOpen);
  const [isCrypto, setIsCrypto] = useState(false);
  const [updateInterval, setUpdateInterval] = useState(1000);
  const intervalRef = useRef<NodeJS.Timeout>();

  useEffect(() => {
    if (!enabled || !symbol) return;

    const fetchPrice = async () => {
      try {
        // First try crypto endpoint
        const cryptoResponse = await fetch(`/api/v1/crypto/${symbol}/price`);

        if (cryptoResponse.ok) {
          const data = await cryptoResponse.json();
          setIsCrypto(true);
          setIsMarketOpen(true); // Crypto markets are always open
          setUpdateInterval(data.update_interval || 1000);

          setPriceData({
            price: String(data.price),
            change: String(data.price * data.change_24h / 100),
            changePercent: String(data.change_24h / 100),
            volume: data.volume_24h || 0,
            lastUpdated: data.last_updated || new Date().toISOString()
          });

          setError(null);
          return;
        }

        // Fall back to regular ticker endpoint for stocks
        const response = await fetch(`/api/v1/tickers/${symbol}/realtime`);

        if (!response.ok) {
          throw new Error(`Failed to fetch price: ${response.status}`);
        }

        const result = await response.json();
        setIsCrypto(false);
        setIsMarketOpen(result.market?.isOpen ?? false);
        setUpdateInterval(result.market?.updateInterval || 1000);

        setPriceData({
          price: result.data.price,
          change: result.data.change,
          changePercent: result.data.changePercent,
          volume: result.data.volume,
          lastUpdated: result.data.lastUpdated
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
    isMarketOpen,
    isCrypto,
    updateInterval
  };
}