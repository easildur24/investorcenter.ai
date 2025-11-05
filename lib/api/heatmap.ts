import { apiClient } from './client';

export interface HeatmapConfig {
  id: string;
  user_id: string;
  watch_list_id: string;
  name: string;
  size_metric: 'market_cap' | 'volume' | 'avg_volume' | 'reddit_mentions' | 'reddit_popularity';
  color_metric: 'price_change_pct' | 'volume_change_pct' | 'reddit_rank' | 'reddit_trend';
  time_period: '1D' | '1W' | '1M' | '3M' | '6M' | 'YTD' | '1Y' | '5Y';
  color_scheme: 'red_green' | 'heatmap' | 'blue_red' | 'custom';
  label_display: 'symbol' | 'symbol_change' | 'full';
  layout_type: 'treemap' | 'grid';
  filters: Record<string, any>;
  color_gradient?: Record<string, string>;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface HeatmapTile {
  symbol: string;
  name: string;
  asset_type: string;
  size_value: number;
  size_label: string;
  color_value: number;
  color_label: string;
  current_price: number;
  price_change: number;
  price_change_pct: number;
  volume?: number;
  market_cap?: number;
  prev_close?: number;
  exchange: string;
  // Reddit data
  reddit_rank?: number;
  reddit_mentions?: number;
  reddit_popularity?: number;
  reddit_trend?: 'rising' | 'falling' | 'stable';
  reddit_rank_change?: number;
  // User watchlist data
  notes?: string;
  tags: string[];
  target_buy_price?: number;
  target_sell_price?: number;
}

export interface HeatmapData {
  watch_list_id: string;
  watch_list_name: string;
  config_id?: string;
  config_name?: string;
  size_metric: string;
  color_metric: string;
  time_period: string;
  color_scheme: string;
  tiles: HeatmapTile[];
  tile_count: number;
  min_color_value: number;
  max_color_value: number;
  generated_at: string;
}

export const heatmapAPI = {
  // Get heatmap data
  async getHeatmapData(
    watchListId: string,
    configId?: string,
    overrides?: {
      size_metric?: string;
      color_metric?: string;
      time_period?: string;
    }
  ): Promise<HeatmapData> {
    const params = new URLSearchParams();
    if (configId) params.append('config_id', configId);
    if (overrides?.size_metric) params.append('size_metric', overrides.size_metric);
    if (overrides?.color_metric) params.append('color_metric', overrides.color_metric);
    if (overrides?.time_period) params.append('time_period', overrides.time_period);

    const queryString = params.toString();
    const url = `/watchlists/${watchListId}/heatmap${queryString ? `?${queryString}` : ''}`;
    return apiClient.get(url);
  },

  // Get all configs for a watch list
  async getConfigs(watchListId: string): Promise<{ configs: HeatmapConfig[] }> {
    return apiClient.get(`/watchlists/${watchListId}/heatmap/configs`);
  },

  // Create new config
  async createConfig(
    watchListId: string,
    config: {
      name: string;
      size_metric: string;
      color_metric: string;
      time_period: string;
      color_scheme?: string;
      label_display?: string;
      layout_type?: string;
      filters?: Record<string, any>;
      color_gradient?: Record<string, string>;
      is_default?: boolean;
    }
  ): Promise<HeatmapConfig> {
    return apiClient.post(`/watchlists/${watchListId}/heatmap/configs`, {
      watch_list_id: watchListId,
      ...config,
    });
  },

  // Update config
  async updateConfig(
    watchListId: string,
    configId: string,
    config: Partial<HeatmapConfig>
  ): Promise<HeatmapConfig> {
    return apiClient.put(`/watchlists/${watchListId}/heatmap/configs/${configId}`, config);
  },

  // Delete config
  async deleteConfig(watchListId: string, configId: string): Promise<void> {
    return apiClient.delete(`/watchlists/${watchListId}/heatmap/configs/${configId}`);
  },
};
