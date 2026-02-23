/**
 * Comprehensive Financial Metrics API Client
 *
 * Handles API requests for comprehensive financial metrics from the
 * /api/v1/stocks/:ticker/metrics endpoint (FMP data).
 */

import { ComprehensiveMetricsResponse, ComprehensiveMetricsData } from '@/types/metrics';

import { stocks } from './routes';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

/**
 * Error response from API
 */
interface APIError {
  error: string;
  message: string;
  ticker?: string;
}

/**
 * Fetch comprehensive financial metrics for a ticker
 *
 * This endpoint returns TTM financial ratios, growth metrics, quality scores,
 * and forward analyst estimates from Financial Modeling Prep (FMP) API.
 *
 * @param ticker Stock symbol (e.g., "AAPL")
 * @returns Comprehensive metrics data or null if not available
 */
export async function getComprehensiveMetrics(
  ticker: string
): Promise<ComprehensiveMetricsResponse | null> {
  try {
    const response = await fetch(`${API_BASE_URL}${stocks.metrics(ticker.toUpperCase())}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      cache: 'no-store',
    });

    if (response.status === 404) {
      return null;
    }

    if (!response.ok) {
      console.error(`Failed to fetch comprehensive metrics for ${ticker}: ${response.status}`);
      return null;
    }

    const result: ComprehensiveMetricsResponse = await response.json();
    return result;
  } catch (error) {
    console.error(`Error fetching comprehensive metrics for ${ticker}:`, error);
    return null;
  }
}

/**
 * Check if comprehensive metrics data is available for a ticker
 *
 * @param ticker Stock symbol
 * @returns True if FMP data is available
 */
export async function hasComprehensiveMetrics(ticker: string): Promise<boolean> {
  try {
    const response = await fetch(`${API_BASE_URL}${stocks.metrics(ticker.toUpperCase())}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      return false;
    }

    const result: ComprehensiveMetricsResponse = await response.json();
    return result.meta?.fmp_available ?? false;
  } catch (error) {
    return false;
  }
}

/**
 * Helper to extract a specific category of metrics
 */
export function extractMetricsCategory<K extends keyof ComprehensiveMetricsData>(
  response: ComprehensiveMetricsResponse | null,
  category: K
): ComprehensiveMetricsData[K] | null {
  if (!response?.data) return null;
  return response.data[category];
}
