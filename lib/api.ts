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
    
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers,
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

  async getStockData(symbol: string) {
    return this.request<{
      symbol: string;
      name: string;
      price: number;
      change: number;
      changePercent: number;
      volume: number;
      marketCap: number;
      pe: number;
      eps: number;
      dividend: number;
      dividendYield: number;
      '52WeekHigh': number;
      '52WeekLow': number;
      lastUpdated: string;
    }>(`/markets/stocks/${symbol}`);
  }

  async getStockChart(symbol: string, period: string = '1d') {
    return this.request<{
      symbol: string;
      period: string;
      dataPoints: Array<{
        timestamp: string;
        open: number;
        high: number;
        low: number;
        close: number;
        volume: number;
      }>;
    }>(`/markets/stocks/${symbol}/chart?period=${period}`);
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
