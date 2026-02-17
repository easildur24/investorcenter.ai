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
}

export interface WatchListWithItems {
  id: string;
  name: string;
  description?: string;
  is_default: boolean;
  item_count: number;
  items: WatchListItem[];
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
