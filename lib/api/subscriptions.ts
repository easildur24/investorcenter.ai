import { apiClient } from './client';
import { subscriptions } from './routes';

// Subscription Types
export interface SubscriptionPlan {
  id: string;
  name: string;
  display_name: string;
  description?: string;
  price_monthly: number;
  price_yearly: number;
  max_watch_lists: number;
  max_items_per_watch_list: number;
  max_alert_rules: number;
  max_heatmap_configs: number;
  features: {
    realtime_data?: boolean;
    advanced_alerts?: boolean;
    priority_support?: boolean;
    api_access?: boolean;
    custom_dashboards?: boolean;
    export_data?: boolean;
    white_label?: boolean;
    dedicated_account_manager?: boolean;
  };
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface UserSubscription {
  id: string;
  user_id: string;
  plan_id: string;
  status:
    | 'active'
    | 'past_due'
    | 'canceled'
    | 'trialing'
    | 'incomplete'
    | 'incomplete_expired'
    | 'unpaid';
  billing_period: 'monthly' | 'yearly';
  started_at: string;
  current_period_start: string;
  current_period_end?: string;
  canceled_at?: string;
  ended_at?: string;
  stripe_subscription_id?: string;
  stripe_customer_id?: string;
  payment_method?: string;
  last_payment_date?: string;
  next_payment_date?: string;
  created_at: string;
  updated_at: string;
  plan?: SubscriptionPlan; // Joined plan details
}

export interface SubscriptionLimits {
  max_watch_lists: number;
  max_items_per_watch_list: number;
  max_alert_rules: number;
  max_heatmap_configs: number;
  current_watch_lists: number;
  current_alert_rules: number;
  features: any;
}

export interface PaymentHistory {
  id: string;
  user_id: string;
  subscription_id?: string;
  amount: number;
  currency: string;
  status: 'succeeded' | 'pending' | 'failed' | 'refunded' | 'canceled';
  payment_method?: string;
  stripe_payment_intent_id?: string;
  stripe_invoice_id?: string;
  description?: string;
  receipt_url?: string;
  created_at: string;
}

export interface CreateSubscriptionRequest {
  plan_id: string;
  billing_period: 'monthly' | 'yearly';
  payment_method_id: string;
}

export interface UpdateSubscriptionRequest {
  plan_id?: string;
  billing_period?: 'monthly' | 'yearly';
}

// Subscription API
export const subscriptionAPI = {
  // List all subscription plans
  async listPlans(): Promise<SubscriptionPlan[]> {
    return apiClient.get(subscriptions.plans);
  },

  // Get specific plan details
  async getPlan(planId: string): Promise<SubscriptionPlan> {
    return apiClient.get(subscriptions.planById(planId));
  },

  // Get user's current subscription
  async getMySubscription(): Promise<UserSubscription> {
    return apiClient.get(subscriptions.me);
  },

  // Create new subscription
  async createSubscription(data: CreateSubscriptionRequest): Promise<UserSubscription> {
    return apiClient.post(subscriptions.create, data);
  },

  // Update subscription (upgrade/downgrade)
  async updateSubscription(data: UpdateSubscriptionRequest): Promise<UserSubscription> {
    return apiClient.put(subscriptions.me, data);
  },

  // Cancel subscription
  async cancelSubscription(): Promise<void> {
    return apiClient.post(subscriptions.cancel, {});
  },

  // Get subscription limits
  async getLimits(): Promise<SubscriptionLimits> {
    return apiClient.get(subscriptions.limits);
  },

  // Get payment history
  async getPaymentHistory(): Promise<PaymentHistory[]> {
    return apiClient.get(subscriptions.payments);
  },
};

// Helper function to check if user has reached limit
export function hasReachedLimit(
  limits: SubscriptionLimits,
  type: 'watch_lists' | 'alert_rules'
): boolean {
  if (type === 'watch_lists') {
    return limits.max_watch_lists !== -1 && limits.current_watch_lists >= limits.max_watch_lists;
  }
  if (type === 'alert_rules') {
    return limits.max_alert_rules !== -1 && limits.current_alert_rules >= limits.max_alert_rules;
  }
  return false;
}

// Helper to format price
export function formatPrice(price: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(price);
}
