import { apiClient } from './client';
import { alerts } from './routes';

// Alert Types
export interface AlertRule {
  id: string;
  user_id: string;
  watch_list_id: string;
  watch_list_item_id?: string;
  symbol: string;
  alert_type: string;
  conditions: any;
  is_active: boolean;
  frequency: 'once' | 'daily' | 'always';
  notify_email: boolean;
  notify_in_app: boolean;
  name: string;
  description?: string;
  last_triggered_at?: string;
  trigger_count: number;
  created_at: string;
  updated_at: string;
}

export interface AlertRuleWithDetails extends AlertRule {
  watch_list_name: string;
  company_name?: string;
}

export interface AlertLog {
  id: string;
  alert_rule_id: string;
  user_id: string;
  symbol: string;
  triggered_at: string;
  alert_type: string;
  condition_met: any;
  market_data: any;
  notification_sent: boolean;
  notification_sent_at?: string;
  notification_error?: string;
  is_read: boolean;
  read_at?: string;
  is_dismissed: boolean;
  dismissed_at?: string;
}

export interface AlertLogWithRule extends AlertLog {
  rule_name: string;
}

export interface CreateAlertRequest {
  watch_list_id: string;
  symbol: string;
  alert_type: string;
  conditions: any;
  name: string;
  description?: string;
  frequency: 'once' | 'daily' | 'always';
  notify_email: boolean;
  notify_in_app: boolean;
}

export interface UpdateAlertRequest {
  name?: string;
  description?: string;
  conditions?: any;
  is_active?: boolean;
  frequency?: 'once' | 'daily' | 'always';
  notify_email?: boolean;
  notify_in_app?: boolean;
}

export interface BulkCreateAlertRequest {
  watch_list_id: string;
  alert_type: string;
  conditions: Record<string, unknown>;
  frequency: 'once' | 'daily' | 'always';
  notify_email: boolean;
  notify_in_app: boolean;
}

export interface BulkCreateAlertResponse {
  created: number;
  skipped: number;
}

// Alert API
export const alertAPI = {
  // List all alert rules
  async listAlerts(params?: {
    watch_list_id?: string;
    is_active?: boolean;
  }): Promise<AlertRuleWithDetails[]> {
    const queryParams = new URLSearchParams();
    if (params?.watch_list_id) queryParams.append('watch_list_id', params.watch_list_id);
    if (params?.is_active !== undefined)
      queryParams.append('is_active', params.is_active.toString());

    const query = queryParams.toString();
    return apiClient.get(`${alerts.list}${query ? `?${query}` : ''}`);
  },

  // Create new alert rule
  async createAlert(data: CreateAlertRequest): Promise<AlertRule> {
    return apiClient.post(alerts.create, data);
  },

  // Get alert rule by ID
  async getAlert(alertId: string): Promise<AlertRule> {
    return apiClient.get(alerts.byId(alertId));
  },

  // Update alert rule
  async updateAlert(alertId: string, data: UpdateAlertRequest): Promise<AlertRule> {
    return apiClient.put(alerts.byId(alertId), data);
  },

  // Delete alert rule
  async deleteAlert(alertId: string): Promise<void> {
    return apiClient.delete(alerts.byId(alertId));
  },

  // Get alert logs (trigger history)
  async getAlertLogs(params?: { limit?: number; offset?: number }): Promise<AlertLogWithRule[]> {
    const queryParams = new URLSearchParams();
    if (params?.limit) queryParams.append('limit', params.limit.toString());
    if (params?.offset) queryParams.append('offset', params.offset.toString());

    const query = queryParams.toString();
    return apiClient.get(`${alerts.logs.list}${query ? `?${query}` : ''}`);
  },

  // Mark alert log as read
  async markAlertLogRead(logId: string): Promise<void> {
    return apiClient.post(alerts.logs.read(logId), {});
  },

  // Dismiss alert log
  async dismissAlertLog(logId: string): Promise<void> {
    return apiClient.post(alerts.logs.dismiss(logId), {});
  },

  // Bulk create alerts for all tickers in a watchlist
  async bulkCreateAlerts(data: BulkCreateAlertRequest): Promise<BulkCreateAlertResponse> {
    return apiClient.post(alerts.bulk, data);
  },
};

// Alert Type Definitions
export const ALERT_TYPES = {
  // Price alerts
  price_above: { label: 'Price Above', icon: '↑', category: 'price' },
  price_below: { label: 'Price Below', icon: '↓', category: 'price' },
  price_change: { label: 'Price Change', icon: '±', category: 'price' },
  price_change_pct: { label: 'Price Change %', icon: '±', category: 'price' },
  price_change_amount: { label: 'Price Change $', icon: '$', category: 'price' },

  // Volume alerts
  volume_spike: { label: 'Volume Spike', icon: '📊', category: 'volume' },
  unusual_volume: { label: 'Unusual Volume', icon: '📈', category: 'volume' },
  volume_above: { label: 'Volume Above', icon: '⬆', category: 'volume' },
  volume_below: { label: 'Volume Below', icon: '⬇', category: 'volume' },

  // Event alerts
  news: { label: 'News Alert', icon: '📰', category: 'event' },
  earnings: { label: 'Earnings Report', icon: '💰', category: 'event' },
  dividend: { label: 'Dividend', icon: '💵', category: 'event' },
  sec_filing: { label: 'SEC Filing', icon: '📄', category: 'event' },
  analyst_rating: { label: 'Analyst Rating', icon: '⭐', category: 'event' },
} as const;

export const ALERT_FREQUENCIES = {
  once: { label: 'Once', description: 'Trigger once and auto-disable' },
  daily: { label: 'Daily', description: 'Once per day maximum' },
  always: { label: 'Always', description: 'Every time condition is met' },
} as const;
