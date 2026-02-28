'use client';

import { useState, useEffect } from 'react';
import { API_BASE_URL } from '@/lib/api';
import { stocks } from '@/lib/api/routes';
import {
  SPARKLINE_METRICS,
  type MetricHistoryResponse,
  type SparklineDataMap,
} from '@/lib/types/fundamentals';

/**
 * useSparklineData — Batch-fetches metric history for sparkline display.
 *
 * Fetches all 11 configured metrics in parallel using Promise.allSettled
 * for graceful individual failure handling. Returns a map of metric key
 * to sparkline-ready data.
 */
export function useSparklineData(ticker: string) {
  const [data, setData] = useState<SparklineDataMap>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!ticker) {
      setLoading(false);
      return;
    }

    let cancelled = false;

    async function fetchAll() {
      try {
        setLoading(true);
        setError(null);

        const metricKeys = SPARKLINE_METRICS.map((m) => m.key);
        const metricConfigs = Object.fromEntries(SPARKLINE_METRICS.map((m) => [m.key, m]));

        const results = await Promise.allSettled(
          metricKeys.map((metric) =>
            fetch(`${API_BASE_URL}${stocks.metricHistory(ticker, metric)}?limit=20`)
              .then((r) => {
                if (!r.ok) throw new Error(`HTTP ${r.status}`);
                return r.json();
              })
              .then((json) => ({
                metric,
                data: (json.data || json) as MetricHistoryResponse,
              }))
          )
        );

        if (cancelled) return;

        const sparklineMap: SparklineDataMap = {};

        for (const result of results) {
          if (result.status !== 'fulfilled') continue;

          const { metric, data: historyData } = result.value;
          if (!historyData?.data_points || historyData.data_points.length < 2) continue;

          const points = historyData.data_points;
          const values = points.map((dp) => dp.value);
          const latest = points[points.length - 1];
          const config = metricConfigs[metric];

          // Build hover data from data_points
          const hoverData = points.map((dp) => ({
            label: `Q${dp.fiscal_quarter}'${String(dp.fiscal_year).slice(-2)}`,
            value: dp.value,
          }));

          sparklineMap[metric] = {
            values,
            trend: historyData.trend?.direction || 'flat',
            latestValue: latest.value,
            yoyChange: latest.yoy_change,
            hoverData,
            unit: config?.unit || 'USD',
            consecutiveGrowthQuarters: historyData.trend?.consecutive_growth_quarters || 0,
          };
        }

        setData(sparklineMap);
      } catch (err) {
        if (!cancelled) {
          console.error('Error fetching sparkline data:', err);
          setError('Failed to load trend data');
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    fetchAll();

    return () => {
      cancelled = true;
    };
  }, [ticker]);

  return { data, loading, error };
}
