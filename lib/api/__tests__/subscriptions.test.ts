import { subscriptionAPI, hasReachedLimit, formatPrice } from '../subscriptions';
import type { SubscriptionLimits } from '../subscriptions';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
  },
}));

import { apiClient } from '../client';

const mockGet = apiClient.get as jest.Mock;
const mockPost = apiClient.post as jest.Mock;
const mockPut = apiClient.put as jest.Mock;

beforeEach(() => {
  jest.clearAllMocks();
});

// =============================================================================
// Pure helper functions
// =============================================================================

describe('hasReachedLimit', () => {
  it('returns true when watch_lists at max', () => {
    const limits: SubscriptionLimits = {
      max_watch_lists: 5,
      max_items_per_watch_list: 50,
      max_alert_rules: 10,
      max_heatmap_configs: 3,
      current_watch_lists: 5,
      current_alert_rules: 2,
      features: {},
    };
    expect(hasReachedLimit(limits, 'watch_lists')).toBe(true);
  });

  it('returns false when watch_lists below max', () => {
    const limits: SubscriptionLimits = {
      max_watch_lists: 5,
      max_items_per_watch_list: 50,
      max_alert_rules: 10,
      max_heatmap_configs: 3,
      current_watch_lists: 3,
      current_alert_rules: 2,
      features: {},
    };
    expect(hasReachedLimit(limits, 'watch_lists')).toBe(false);
  });

  it('returns false when max_watch_lists is -1 (unlimited)', () => {
    const limits: SubscriptionLimits = {
      max_watch_lists: -1,
      max_items_per_watch_list: 50,
      max_alert_rules: 10,
      max_heatmap_configs: 3,
      current_watch_lists: 100,
      current_alert_rules: 2,
      features: {},
    };
    expect(hasReachedLimit(limits, 'watch_lists')).toBe(false);
  });

  it('returns true when alert_rules at max', () => {
    const limits: SubscriptionLimits = {
      max_watch_lists: 5,
      max_items_per_watch_list: 50,
      max_alert_rules: 10,
      max_heatmap_configs: 3,
      current_watch_lists: 3,
      current_alert_rules: 10,
      features: {},
    };
    expect(hasReachedLimit(limits, 'alert_rules')).toBe(true);
  });

  it('returns false when alert_rules below max', () => {
    const limits: SubscriptionLimits = {
      max_watch_lists: 5,
      max_items_per_watch_list: 50,
      max_alert_rules: 10,
      max_heatmap_configs: 3,
      current_watch_lists: 3,
      current_alert_rules: 5,
      features: {},
    };
    expect(hasReachedLimit(limits, 'alert_rules')).toBe(false);
  });

  it('returns false when max_alert_rules is -1 (unlimited)', () => {
    const limits: SubscriptionLimits = {
      max_watch_lists: 5,
      max_items_per_watch_list: 50,
      max_alert_rules: -1,
      max_heatmap_configs: 3,
      current_watch_lists: 3,
      current_alert_rules: 100,
      features: {},
    };
    expect(hasReachedLimit(limits, 'alert_rules')).toBe(false);
  });

  it('returns false for unknown type', () => {
    const limits: SubscriptionLimits = {
      max_watch_lists: 5,
      max_items_per_watch_list: 50,
      max_alert_rules: 10,
      max_heatmap_configs: 3,
      current_watch_lists: 5,
      current_alert_rules: 10,
      features: {},
    };
    expect(hasReachedLimit(limits, 'unknown' as any)).toBe(false);
  });
});

describe('formatPrice', () => {
  it('formats whole dollar amount', () => {
    expect(formatPrice(10)).toBe('$10.00');
  });

  it('formats cents', () => {
    expect(formatPrice(9.99)).toBe('$9.99');
  });

  it('formats zero', () => {
    expect(formatPrice(0)).toBe('$0.00');
  });

  it('formats large values with commas', () => {
    const result = formatPrice(1234.56);
    expect(result).toContain('1,234.56');
  });
});

// =============================================================================
// API wrapper methods
// =============================================================================

describe('subscriptionAPI', () => {
  describe('listPlans', () => {
    it('calls apiClient.get with correct endpoint', async () => {
      mockGet.mockResolvedValueOnce([]);
      await subscriptionAPI.listPlans();
      expect(mockGet).toHaveBeenCalledWith('/subscriptions/plans');
    });
  });

  describe('getPlan', () => {
    it('calls apiClient.get with plan id', async () => {
      mockGet.mockResolvedValueOnce({});
      await subscriptionAPI.getPlan('plan-123');
      expect(mockGet).toHaveBeenCalledWith('/subscriptions/plans/plan-123');
    });
  });

  describe('getMySubscription', () => {
    it('calls apiClient.get with correct endpoint', async () => {
      mockGet.mockResolvedValueOnce({});
      await subscriptionAPI.getMySubscription();
      expect(mockGet).toHaveBeenCalledWith('/subscriptions/me');
    });
  });

  describe('createSubscription', () => {
    it('calls apiClient.post with data', async () => {
      const data = {
        plan_id: 'plan-1',
        billing_period: 'monthly' as const,
        payment_method_id: 'pm-1',
      };
      mockPost.mockResolvedValueOnce({});
      await subscriptionAPI.createSubscription(data);
      expect(mockPost).toHaveBeenCalledWith('/subscriptions', data);
    });
  });

  describe('updateSubscription', () => {
    it('calls apiClient.put with data', async () => {
      const data = { plan_id: 'plan-2' };
      mockPut.mockResolvedValueOnce({});
      await subscriptionAPI.updateSubscription(data);
      expect(mockPut).toHaveBeenCalledWith('/subscriptions/me', data);
    });
  });

  describe('cancelSubscription', () => {
    it('calls apiClient.post with empty body', async () => {
      mockPost.mockResolvedValueOnce(undefined);
      await subscriptionAPI.cancelSubscription();
      expect(mockPost).toHaveBeenCalledWith('/subscriptions/me/cancel', {});
    });
  });

  describe('getLimits', () => {
    it('calls apiClient.get with correct endpoint', async () => {
      mockGet.mockResolvedValueOnce({});
      await subscriptionAPI.getLimits();
      expect(mockGet).toHaveBeenCalledWith('/subscriptions/limits');
    });
  });

  describe('getPaymentHistory', () => {
    it('calls apiClient.get with correct endpoint', async () => {
      mockGet.mockResolvedValueOnce([]);
      await subscriptionAPI.getPaymentHistory();
      expect(mockGet).toHaveBeenCalledWith('/subscriptions/payments');
    });
  });
});
