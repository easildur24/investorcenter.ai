import { apiClient } from './client';

export interface WatchList {
  id: string;
  user_id: string;
  name: string;
  description?: string;
  is_default: boolean;
  item_count: number;
  created_at: string;
  updated_at: string;
}

export interface WatchListItem {
  id: string;
  watch_list_id: string;
  symbol: string;
  notes?: string;
  tags: string[];
  target_buy_price?: number;
  target_sell_price?: number;
  added_at: string;
  display_order: number;
  // Ticker data
  name: string;
  exchange: string;
  asset_type: string;
  logo_url?: string;
  // Real-time price data
  current_price?: number;
  price_change?: number;
  price_change_pct?: number;
  volume?: number;
  market_cap?: number;
  prev_close?: number;
  // Reddit data
  reddit_rank?: number;
  reddit_mentions?: number;
  reddit_popularity?: number;
  reddit_trend?: string;
  reddit_rank_change?: number;
  // IC Score (from screener_data)
  ic_score?: number;
  ic_rating?: string;
  value_score?: number;
  growth_score?: number;
  profitability_score?: number;
  financial_health_score?: number;
  momentum_score?: number;
  analyst_consensus_score?: number;
  insider_activity_score?: number;
  institutional_score?: number;
  news_sentiment_score?: number;
  technical_score?: number;
  sector_percentile?: number;
  lifecycle_stage?: string;
  // Fundamentals (from screener_data)
  pe_ratio?: number;
  pb_ratio?: number;
  ps_ratio?: number;
  roe?: number;
  roa?: number;
  gross_margin?: number;
  operating_margin?: number;
  net_margin?: number;
  debt_to_equity?: number;
  current_ratio?: number;
  revenue_growth?: number;
  eps_growth?: number;
  dividend_yield?: number;
  payout_ratio?: number;
  // Alert metadata
  alert_count: number;
}

export interface WatchListSummaryMetrics {
  total_tickers: number;
  avg_ic_score?: number;
  avg_day_change_pct?: number;
  avg_dividend_yield?: number;
  reddit_trending_count: number;
}

export interface WatchListWithItems {
  id: string;
  name: string;
  description?: string;
  is_default: boolean;
  item_count: number;
  items: WatchListItem[];
  summary?: WatchListSummaryMetrics;
  created_at: string;
  updated_at: string;
}

export const watchListAPI = {
  // Get all watch lists for user
  async getWatchLists(): Promise<{ watch_lists: WatchList[] }> {
    return apiClient.get('/watchlists');
  },

  // Create new watch list
  async createWatchList(data: { name: string; description?: string }): Promise<WatchList> {
    return apiClient.post('/watchlists', data);
  },

  // Get single watch list with items
  async getWatchList(id: string): Promise<WatchListWithItems> {
    return apiClient.get(`/watchlists/${id}`);
  },

  // Update watch list metadata
  async updateWatchList(id: string, data: { name: string; description?: string }): Promise<void> {
    return apiClient.put(`/watchlists/${id}`, data);
  },

  // Delete watch list
  async deleteWatchList(id: string): Promise<void> {
    return apiClient.delete(`/watchlists/${id}`);
  },

  // Add ticker to watch list
  async addTicker(
    watchListId: string,
    data: {
      symbol: string;
      notes?: string;
      tags?: string[];
      target_buy_price?: number;
      target_sell_price?: number;
    }
  ): Promise<WatchListItem> {
    return apiClient.post(`/watchlists/${watchListId}/items`, data);
  },

  // Remove ticker from watch list
  async removeTicker(watchListId: string, symbol: string): Promise<void> {
    return apiClient.delete(`/watchlists/${watchListId}/items/${symbol}`);
  },

  // Update ticker metadata
  async updateTicker(
    watchListId: string,
    symbol: string,
    data: {
      notes?: string;
      tags?: string[];
      target_buy_price?: number;
      target_sell_price?: number;
    }
  ): Promise<WatchListItem> {
    return apiClient.put(`/watchlists/${watchListId}/items/${symbol}`, data);
  },

  // Bulk add tickers
  async bulkAddTickers(
    watchListId: string,
    symbols: string[]
  ): Promise<{
    added: string[];
    failed: string[];
    total: number;
  }> {
    return apiClient.post(`/watchlists/${watchListId}/bulk`, { symbols });
  },

  // Reorder items
  async reorderItems(
    watchListId: string,
    itemOrders: Array<{ item_id: string; display_order: number }>
  ): Promise<void> {
    return apiClient.post(`/watchlists/${watchListId}/reorder`, { item_orders: itemOrders });
  },
};
