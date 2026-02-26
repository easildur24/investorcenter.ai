import { apiClient } from './client';
import { notifications } from './routes';

// Notification Types
export interface InAppNotification {
  id: string;
  user_id: string;
  type: string;
  title: string;
  message: string;
  data?: { watch_list_id?: string; [key: string]: unknown } | null;
  is_read: boolean;
  read_at?: string;
  is_dismissed: boolean;
  dismissed_at?: string;
  created_at: string;
}

export interface NotificationPreferences {
  id: string;
  user_id: string;
  email_enabled: boolean;
  in_app_enabled: boolean;
  push_enabled: boolean;
  quiet_hours_start?: string;
  quiet_hours_end?: string;
  digest_frequency: 'none' | 'daily' | 'weekly';
  alert_types: {
    price_alerts: boolean;
    volume_alerts: boolean;
    news_alerts: boolean;
    earnings_alerts: boolean;
  };
  created_at: string;
  updated_at: string;
}

export interface UpdatePreferencesRequest {
  email_enabled?: boolean;
  in_app_enabled?: boolean;
  push_enabled?: boolean;
  quiet_hours_start?: string;
  quiet_hours_end?: string;
  digest_frequency?: 'none' | 'daily' | 'weekly';
  alert_types?: {
    price_alerts?: boolean;
    volume_alerts?: boolean;
    news_alerts?: boolean;
    earnings_alerts?: boolean;
  };
}

// Notification API
export const notificationAPI = {
  // Get in-app notifications
  async getNotifications(params?: {
    limit?: number;
    offset?: number;
    unread_only?: boolean;
  }): Promise<InAppNotification[]> {
    const queryParams = new URLSearchParams();
    if (params?.limit) queryParams.append('limit', params.limit.toString());
    if (params?.offset) queryParams.append('offset', params.offset.toString());
    if (params?.unread_only) queryParams.append('unread_only', 'true');

    const query = queryParams.toString();
    return apiClient.get(`${notifications.list}${query ? `?${query}` : ''}`);
  },

  // Get unread count
  async getUnreadCount(): Promise<{ count: number }> {
    return apiClient.get(notifications.unreadCount);
  },

  // Mark notification as read
  async markAsRead(notificationId: string): Promise<void> {
    return apiClient.post(notifications.read(notificationId), {});
  },

  // Mark all notifications as read
  async markAllAsRead(): Promise<void> {
    return apiClient.post(notifications.readAll, {});
  },

  // Dismiss notification
  async dismiss(notificationId: string): Promise<void> {
    return apiClient.post(notifications.dismiss(notificationId), {});
  },

  // Get notification preferences
  async getPreferences(): Promise<NotificationPreferences> {
    return apiClient.get(notifications.preferences);
  },

  // Update notification preferences
  async updatePreferences(data: UpdatePreferencesRequest): Promise<NotificationPreferences> {
    return apiClient.put(notifications.preferences, data);
  },
};
