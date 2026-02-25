'use client';

import { useState, useEffect, useCallback, useMemo } from 'react';
import {
  alertAPI,
  AlertRule,
  AlertRuleWithDetails,
  CreateAlertRequest,
  UpdateAlertRequest,
} from '@/lib/api/alerts';

interface UseWatchlistAlertsReturn {
  /** Map from symbol to alert rule (at most one per symbol due to 1:1 constraint) */
  alertsBySymbol: Map<string, AlertRuleWithDetails>;
  /** Loading state for initial fetch */
  loading: boolean;
  /** Create alert for a symbol; updates local cache on success */
  createAlert: (req: CreateAlertRequest) => Promise<AlertRule>;
  /** Update an existing alert; updates local cache on success */
  updateAlert: (alertId: string, req: UpdateAlertRequest) => Promise<void>;
  /** Delete an alert; removes from local cache on success */
  deleteAlert: (alertId: string, symbol: string) => Promise<void>;
  /** Re-fetch all alerts for this watchlist */
  refresh: () => Promise<void>;
}

export function useWatchlistAlerts(watchListId: string | null): UseWatchlistAlertsReturn {
  const [alerts, setAlerts] = useState<AlertRuleWithDetails[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchAlerts = useCallback(async () => {
    if (!watchListId) return;
    setLoading(true);
    try {
      const data = await alertAPI.listAlerts({ watch_list_id: watchListId });
      setAlerts(data ?? []);
    } catch {
      // Alert fetch is non-critical — table still renders without alert data
    } finally {
      setLoading(false);
    }
  }, [watchListId]);

  useEffect(() => {
    fetchAlerts();
  }, [fetchAlerts]);

  const alertsBySymbol = useMemo(() => {
    const map = new Map<string, AlertRuleWithDetails>();
    for (const alert of alerts) {
      map.set(alert.symbol, alert);
    }
    return map;
  }, [alerts]);

  const createAlert = useCallback(
    async (req: CreateAlertRequest) => {
      const created = await alertAPI.createAlert(req);
      // Re-fetch to get full AlertRuleWithDetails (with watch_list_name, company_name)
      await fetchAlerts();
      return created;
    },
    [fetchAlerts]
  );

  const updateAlert = useCallback(async (alertId: string, req: UpdateAlertRequest) => {
    await alertAPI.updateAlert(alertId, req);
    // Patch local cache
    setAlerts((prev) =>
      prev.map((a) => (a.id === alertId ? ({ ...a, ...req } as AlertRuleWithDetails) : a))
    );
  }, []);

  const deleteAlert = useCallback(async (alertId: string, _symbol: string) => {
    await alertAPI.deleteAlert(alertId);
    setAlerts((prev) => prev.filter((a) => a.id !== alertId));
  }, []);

  return {
    alertsBySymbol,
    loading,
    createAlert,
    updateAlert,
    deleteAlert,
    refresh: fetchAlerts,
  };
}
