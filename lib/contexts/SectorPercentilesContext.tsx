'use client';

import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { API_BASE_URL } from '@/lib/api';
import { stocks } from '@/lib/api/routes';
import type { SectorPercentilesResponse, MetricPercentileData } from '@/lib/types/fundamentals';

interface SectorPercentilesContextValue {
  data: SectorPercentilesResponse | null;
  loading: boolean;
  error: string | null;
  getMetricPercentile: (metricKey: string) => MetricPercentileData | null;
}

const SectorPercentilesContext = createContext<SectorPercentilesContextValue | null>(null);

interface SectorPercentilesProviderProps {
  ticker: string;
  children: ReactNode;
}

export function SectorPercentilesProvider({ ticker, children }: SectorPercentilesProviderProps) {
  const [data, setData] = useState<SectorPercentilesResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchPercentiles = async () => {
      try {
        setLoading(true);
        setError(null);
        const response = await fetch(`${API_BASE_URL}${stocks.sectorPercentiles(ticker)}`);
        if (!response.ok) {
          if (response.status === 404) {
            // No percentile data available for this ticker — not an error
            setData(null);
            return;
          }
          throw new Error(`HTTP ${response.status}`);
        }
        const result = await response.json();
        setData(result.data || result);
      } catch (err) {
        console.error('Error fetching sector percentiles:', err);
        setError('Failed to load sector percentiles');
      } finally {
        setLoading(false);
      }
    };

    if (ticker) {
      fetchPercentiles();
    }
  }, [ticker]);

  const getMetricPercentile = (metricKey: string): MetricPercentileData | null => {
    if (!data?.metrics) return null;
    return data.metrics[metricKey] ?? null;
  };

  return (
    <SectorPercentilesContext.Provider value={{ data, loading, error, getMetricPercentile }}>
      {children}
    </SectorPercentilesContext.Provider>
  );
}

export function useSectorPercentiles(): SectorPercentilesContextValue {
  const context = useContext(SectorPercentilesContext);
  if (!context) {
    // Return a safe default when used outside of provider
    // This allows components to gracefully degrade
    return {
      data: null,
      loading: false,
      error: null,
      getMetricPercentile: () => null,
    };
  }
  return context;
}
