// API client for communicating with Go backend

import type { ScreenerApiParams, ScreenerResponse } from '@/lib/types/screener';
import { validateResponse } from '@/lib/api/validate';
import {
  markets as marketsRoutes,
  tickers as tickersRoutes,
  screener as screenerRoutes,
  icScoreService,
} from '@/lib/api/routes';
import {
  ICScoreSchema,
  ICScoreScreenerResponseSchema,
  ICScoreTopStocksResponseSchema,
  ICScoreHistorySchema,
  ICScoreStatisticsSchema,
  MarketIndicesSchema,
  MarketMoversSchema,
  ScreenerResponseSchema,
} from '@/lib/api/schemas';

export const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || '/api/v1';
const IC_SCORE_API_BASE = process.env.NEXT_PUBLIC_IC_SCORE_API_URL || 'http://localhost:8001';

interface ApiResponse<T> {
  data: T;
  meta: {
    timestamp: string;
    count?: number;
  };
}

interface ApiError {
  error: string;
  message?: string;
}

class ApiClient {
  private baseURL: string;
  private token: string | null = null;

  constructor(baseURL: string = API_BASE_URL) {
    this.baseURL = baseURL;

    // Get token from localStorage if available
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('auth_token');
    }
  }

  async request<T>(endpoint: string, options: RequestInit = {}): Promise<ApiResponse<T>> {
    const url = `${this.baseURL}${endpoint}`;

    const headers: Record<string, string> = {
      ...(options.headers as Record<string, string>),
    };

    // Only set Content-Type for requests with a body to avoid CORS preflight
    const method = (options.method || 'GET').toUpperCase();
    if (method !== 'GET' && method !== 'HEAD') {
      headers['Content-Type'] = 'application/json';
    }

    if (this.token) {
      headers.Authorization = `Bearer ${this.token}`;
    }

    const config: RequestInit = {
      ...options,
      headers,
    };

    try {
      const response = await fetch(url, config);

      if (!response.ok) {
        const errorData: ApiError = await response.json();
        throw new Error(errorData.error || `HTTP ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('API request failed:', error);
      throw error;
    }
  }

  // Market data methods
  async getMarketIndices() {
    const result = await this.request<
      Array<{
        symbol: string;
        name: string;
        price: number;
        change: number;
        changePercent: number;
        lastUpdated: string;
      }>
    >(marketsRoutes.indices);
    result.data = validateResponse(MarketIndicesSchema, result.data, marketsRoutes.indices);
    return result;
  }

  async searchSecurities(query: string) {
    return this.request<
      Array<{
        symbol: string;
        name: string;
        type: string;
        exchange: string;
        logo_url?: string;
      }>
    >(`${marketsRoutes.search}?q=${encodeURIComponent(query)}`);
  }

  async getMarketMovers(limit: number = 5) {
    const result = await this.request<{
      gainers: Array<{
        symbol: string;
        name?: string;
        price: number;
        change: number;
        changePercent: number;
        volume: number;
      }>;
      losers: Array<{
        symbol: string;
        name?: string;
        price: number;
        change: number;
        changePercent: number;
        volume: number;
      }>;
      mostActive: Array<{
        symbol: string;
        name?: string;
        price: number;
        change: number;
        changePercent: number;
        volume: number;
      }>;
    }>(`${marketsRoutes.movers}?limit=${limit}`);
    result.data = validateResponse(MarketMoversSchema, result.data, marketsRoutes.movers);
    return result;
  }

  // Screener methods
  async getScreenerStocks(params?: ScreenerApiParams) {
    const queryParams = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null && value !== '') {
          queryParams.append(key, String(value));
        }
      });
    }
    const query = queryParams.toString() ? `?${queryParams.toString()}` : '';
    return this.request<ScreenerResponse>(`${screenerRoutes.stocks}${query}`);
  }

  // Volume data methods (hybrid: database + real-time)
  async getTickerVolume(symbol: string, realtime: boolean = false) {
    return this.request<{
      data: {
        symbol: string;
        volume: number;
        vwap: number;
        open: number;
        close: number;
        high: number;
        low: number;
        timestamp?: number;
        prevClose?: number;
        change?: number;
        changePercent?: number;
        avgVolume30d?: number;
        avgVolume90d?: number;
        week52High?: number;
        week52Low?: number;
        updatedAt: string;
      };
      source: 'database' | 'polygon';
      realtime: boolean;
    }>(`${tickersRoutes.volume(symbol)}${realtime ? '?realtime=true' : ''}`);
  }

  async getVolumeAggregates(symbol: string, days: number = 90) {
    return this.request<{
      data: {
        symbol: string;
        avgVolume30d: number;
        avgVolume90d: number;
        week52High: number;
        week52Low: number;
        volumeTrend: 'increasing' | 'decreasing' | 'stable';
      };
      source: 'database' | 'polygon';
    }>(`${tickersRoutes.volumeAggregates(symbol)}?days=${days}`);
  }
}

// Export singleton instance
export const apiClient = new ApiClient();

// IC Score API Client (separate service on port 8001)
import type {
  ICScore,
  ICScoreHistory,
  ICScoreScreenerFilters,
  ICScoreScreenerResponse,
  ICScoreTopStocksResponse,
  ICScoreStatistics,
} from './types/ic-score';

export const icScoreApi = {
  /**
   * Get IC Score for a specific ticker
   */
  async getScore(ticker: string): Promise<ICScore> {
    const response = await fetch(
      `${IC_SCORE_API_BASE}${icScoreService.score(ticker.toUpperCase())}`
    );

    if (!response.ok) {
      if (response.status === 404) {
        throw new Error(`IC Score not available for ${ticker}`);
      }
      const error = await response.json();
      throw new Error(error.error || `HTTP ${response.status}`);
    }

    const data = await response.json();
    return validateResponse(ICScoreSchema, data, icScoreService.score(ticker));
  },

  /**
   * Get IC Score history (30-day trend) for a ticker
   */
  async getHistory(ticker: string, days: number = 30): Promise<ICScoreHistory> {
    const response = await fetch(
      `${IC_SCORE_API_BASE}${icScoreService.history(ticker.toUpperCase())}?days=${days}`
    );

    if (!response.ok) {
      if (response.status === 404) {
        throw new Error(`IC Score history not available for ${ticker}`);
      }
      const error = await response.json();
      throw new Error(error.error || `HTTP ${response.status}`);
    }

    const data = await response.json();
    return validateResponse(ICScoreHistorySchema, data, icScoreService.history(ticker));
  },

  /**
   * Get top stocks by IC Score
   */
  async getTopStocks(limit: number = 50): Promise<ICScoreTopStocksResponse> {
    const response = await fetch(`${IC_SCORE_API_BASE}${icScoreService.top}?limit=${limit}`);

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || `HTTP ${response.status}`);
    }

    const data = await response.json();
    return validateResponse(ICScoreTopStocksResponseSchema, data, icScoreService.top);
  },

  /**
   * Run IC Score screener with filters
   */
  async runScreener(filters?: ICScoreScreenerFilters): Promise<ICScoreScreenerResponse> {
    const params = new URLSearchParams();

    if (filters) {
      if (filters.minScore !== undefined) params.append('min_score', filters.minScore.toString());
      if (filters.maxScore !== undefined) params.append('max_score', filters.maxScore.toString());
      if (filters.rating && filters.rating.length > 0) {
        filters.rating.forEach((r) => params.append('rating', r));
      }
      if (filters.sector && filters.sector.length > 0) {
        filters.sector.forEach((s) => params.append('sector', s));
      }
      if (filters.minMarketCap !== undefined)
        params.append('min_market_cap', filters.minMarketCap.toString());
      if (filters.maxMarketCap !== undefined)
        params.append('max_market_cap', filters.maxMarketCap.toString());
      if (filters.sortBy) params.append('sort_by', filters.sortBy);
      if (filters.sortOrder) params.append('sort_order', filters.sortOrder);
      if (filters.limit !== undefined) params.append('limit', filters.limit.toString());
      if (filters.offset !== undefined) params.append('offset', filters.offset.toString());
    }

    const queryString = params.toString();
    const url = `${IC_SCORE_API_BASE}${icScoreService.screener}${queryString ? '?' + queryString : ''}`;

    const response = await fetch(url);

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || `HTTP ${response.status}`);
    }

    const data = await response.json();
    return validateResponse(ICScoreScreenerResponseSchema, data, icScoreService.screener);
  },

  /**
   * Get IC Score statistics
   */
  async getStatistics(): Promise<ICScoreStatistics> {
    const response = await fetch(`${IC_SCORE_API_BASE}${icScoreService.statistics}`);

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || `HTTP ${response.status}`);
    }

    const data = await response.json();
    return validateResponse(ICScoreStatisticsSchema, data, icScoreService.statistics);
  },

  /**
   * Check IC Score API health
   */
  async checkHealth(): Promise<{ status: string; timestamp: string }> {
    const response = await fetch(`${IC_SCORE_API_BASE}${icScoreService.health}`);

    if (!response.ok) {
      throw new Error('IC Score API is unavailable');
    }

    return await response.json();
  },
};

// Export types for use in components
export type { ApiResponse, ApiError };
