// API client for communicating with Go backend

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || '/api/v1';

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

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<ApiResponse<T>> {
    const url = `${this.baseURL}${endpoint}`;
    
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string>),
    };

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

  // Authentication methods
  async login(email: string, password: string) {
    const response = await this.request<{
      token: string;
      user: {
        id: number;
        email: string;
        firstName: string;
        lastName: string;
      };
    }>('/users/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });

    this.token = response.data.token;
    if (typeof window !== 'undefined') {
      localStorage.setItem('auth_token', this.token);
    }

    return response;
  }

  async register(userData: {
    email: string;
    password: string;
    firstName: string;
    lastName: string;
  }) {
    return this.request('/users/register', {
      method: 'POST',
      body: JSON.stringify(userData),
    });
  }

  logout() {
    this.token = null;
    if (typeof window !== 'undefined') {
      localStorage.removeItem('auth_token');
    }
  }

  // Market data methods
  async getMarketIndices() {
    return this.request<Array<{
      symbol: string;
      name: string;
      price: number;
      change: number;
      changePercent: number;
      lastUpdated: string;
    }>>('/markets/indices');
  }

  async searchSecurities(query: string) {
    return this.request<Array<{
      symbol: string;
      name: string;
      type: string;
      exchange: string;
    }>>(`/markets/search?q=${encodeURIComponent(query)}`);
  }

  // Portfolio methods
  async getPortfolios() {
    return this.request<Array<{
      id: number;
      name: string;
      description: string;
      value: number;
      change: number;
      changePercent: number;
      createdAt: string;
    }>>('/portfolios');
  }

  async createPortfolio(data: { name: string; description?: string }) {
    return this.request('/portfolios', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getPortfolio(id: string) {
    return this.request(`/portfolios/${id}`);
  }

  async updatePortfolio(id: string, data: { name?: string; description?: string }) {
    return this.request(`/portfolios/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deletePortfolio(id: string) {
    return this.request(`/portfolios/${id}`, {
      method: 'DELETE',
    });
  }

  async getPortfolioPerformance(id: string, period: string = '1m') {
    return this.request(`/portfolios/${id}/performance?period=${period}`);
  }

  // Analytics methods
  async getSectorPerformance() {
    return this.request<Array<{
      name: string;
      change: number;
      changePercent: number;
    }>>('/analytics/sectors');
  }

  async getMarketTrends() {
    return this.request<{
      bullishSentiment: number;
      bearishSentiment: number;
      volatilityIndex: number;
      topGainers: Array<{
        symbol: string;
        change: number;
        changePercent: number;
      }>;
      topLosers: Array<{
        symbol: string;
        change: number;
        changePercent: number;
      }>;
    }>('/analytics/trends');
  }

  async runStockScreener(criteria?: any) {
    const query = criteria ? `?${new URLSearchParams(criteria).toString()}` : '';
    return this.request(`/analytics/screener${query}`);
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
    }>(`/tickers/${symbol}/volume${realtime ? '?realtime=true' : ''}`);
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
    }>(`/tickers/${symbol}/volume/aggregates?days=${days}`);
  }

  async getBulkVolume(symbols: string[]) {
    return this.request<{
      data: Array<{
        symbol: string;
        volume: number;
        avgVolume30d: number;
        avgVolume90d: number;
        vwap: number;
        currentPrice: number;
        dayOpen: number;
        dayHigh: number;
        dayLow: number;
        previousClose: number;
        week52High: number;
        week52Low: number;
        lastUpdated: string;
      }>;
      source: 'database';
      count: number;
    }>('/volume/bulk', {
      method: 'POST',
      body: JSON.stringify({ symbols }),
    });
  }

  async getTopVolumeStocks(limit: number = 20, assetType: string = 'all') {
    return this.request<{
      data: Array<{
        symbol: string;
        volume: number;
        avgVolume30d: number;
        avgVolume90d: number;
        vwap: number;
        currentPrice: number;
        dayOpen: number;
        dayHigh: number;
        dayLow: number;
        previousClose: number;
        week52High: number;
        week52Low: number;
        lastUpdated: string;
      }>;
      source: 'database';
      count: number;
    }>(`/volume/top?limit=${limit}&type=${assetType}`);
  }

  // Ticker page methods
  async getTickerOverview(symbol: string) {
    return this.request(`/tickers/${symbol}`);
  }

  async getTickerChart(symbol: string, period: string = '1Y') {
    return this.request(`/tickers/${symbol}/chart?period=${period}`);
  }

  async getTickerFundamentals(symbol: string, years: number = 5) {
    return this.request(`/tickers/${symbol}/fundamentals?years=${years}`);
  }

  async getTickerNews(symbol: string, limit: number = 20) {
    return this.request(`/tickers/${symbol}/news?limit=${limit}`);
  }

  async getTickerEarnings(symbol: string) {
    return this.request(`/tickers/${symbol}/earnings`);
  }

  async getTickerDividends(symbol: string) {
    return this.request(`/tickers/${symbol}/dividends`);
  }

  async getTickerAnalysts(symbol: string) {
    return this.request(`/tickers/${symbol}/analysts`);
  }

  async getTickerInsiders(symbol: string) {
    return this.request(`/tickers/${symbol}/insiders`);
  }

  async getTickerPeers(symbol: string) {
    return this.request(`/tickers/${symbol}/peers`);
  }

  // User profile methods
  async getUserProfile() {
    return this.request('/users/profile');
  }

  async updateUserProfile(data: { firstName?: string; lastName?: string }) {
    return this.request('/users/profile', {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }
}

// Export singleton instance
export const apiClient = new ApiClient();

// Export types for use in components
export type { ApiResponse, ApiError };
