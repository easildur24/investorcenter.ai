import { apiClient } from './client';
import { watchlists } from './routes';

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
  // IC Score (from screener_data — always present, null when unavailable)
  ic_score: number | null;
  ic_rating: string | null;
  value_score: number | null;
  growth_score: number | null;
  profitability_score: number | null;
  financial_health_score: number | null;
  momentum_score: number | null;
  analyst_consensus_score: number | null;
  insider_activity_score: number | null;
  institutional_score: number | null;
  news_sentiment_score: number | null;
  technical_score: number | null;
  sector_percentile: number | null;
  lifecycle_stage: string | null;
  // Fundamentals (from screener_data — always present, null when unavailable)
  pe_ratio: number | null;
  pb_ratio: number | null;
  ps_ratio: number | null;
  roe: number | null;
  roa: number | null;
  gross_margin: number | null;
  operating_margin: number | null;
  net_margin: number | null;
  debt_to_equity: number | null;
  current_ratio: number | null;
  revenue_growth: number | null;
  eps_growth: number | null;
  dividend_yield: number | null;
  payout_ratio: number | null;
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
    return apiClient.get(watchlists.list);
  },

  // Create new watch list
  async createWatchList(data: { name: string; description?: string }): Promise<WatchList> {
    return apiClient.post(watchlists.create, data);
  },

  // Get single watch list with items
  async getWatchList(id: string): Promise<WatchListWithItems> {
    return apiClient.get(watchlists.byId(id));
  },

  // Update watch list metadata
  async updateWatchList(id: string, data: { name: string; description?: string }): Promise<void> {
    return apiClient.put(watchlists.byId(id), data);
  },

  // Delete watch list
  async deleteWatchList(id: string): Promise<void> {
    return apiClient.delete(watchlists.byId(id));
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
    return apiClient.post(watchlists.items.add(watchListId), data);
  },

  // Remove ticker from watch list
  async removeTicker(watchListId: string, symbol: string): Promise<void> {
    return apiClient.delete(watchlists.items.remove(watchListId, symbol));
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
    return apiClient.put(watchlists.items.update(watchListId, symbol), data);
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
    return apiClient.post(watchlists.bulk(watchListId), { symbols });
  },

  // Get all tags used across user's watchlists with usage counts, ordered by popularity
  async getUserTags(): Promise<{ tags: { name: string; count: number }[] }> {
    return apiClient.get(watchlists.tags);
  },

  // Reorder items
  async reorderItems(
    watchListId: string,
    itemOrders: Array<{ item_id: string; display_order: number }>
  ): Promise<void> {
    return apiClient.post(watchlists.reorder(watchListId), { item_orders: itemOrders });
  },
};
