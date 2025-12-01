'use client';

import { useEffect, useState } from 'react';
import { alertAPI, AlertRuleWithDetails, ALERT_TYPES, ALERT_FREQUENCIES } from '@/lib/api/alerts';
import { subscriptionAPI, SubscriptionLimits } from '@/lib/api/subscriptions';
import CreateAlertModal from '@/components/alerts/CreateAlertModal';
// TODO: Implement these components (see REMAINING_FRONTEND_COMPONENTS.md)
// import EditAlertModal from '@/components/alerts/EditAlertModal';
// import UpgradeModal from '@/components/subscription/UpgradeModal';
// import ProtectedRoute from '@/components/ProtectedRoute';
import AlertCard from '@/components/alerts/AlertCard';

export default function AlertsPage() {
  // TODO: Add ProtectedRoute wrapper once component is implemented
  return <AlertsPageContent />;
}

function AlertsPageContent() {
  const [alerts, setAlerts] = useState<AlertRuleWithDetails[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<'all' | 'active' | 'inactive'>('all');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [showUpgradeModal, setShowUpgradeModal] = useState(false);
  const [selectedAlert, setSelectedAlert] = useState<AlertRuleWithDetails | null>(null);
  const [limits, setLimits] = useState<SubscriptionLimits | null>(null);

  useEffect(() => {
    loadAlerts();
    loadLimits();
  }, [filter]);

  const loadAlerts = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await alertAPI.listAlerts({
        is_active: filter === 'all' ? undefined : filter === 'active',
      });
      setAlerts(data);
    } catch (err: any) {
      setError(err.message || 'Failed to load alerts');
    } finally {
      setLoading(false);
    }
  };

  const loadLimits = async () => {
    try {
      const data = await subscriptionAPI.getLimits();
      setLimits(data);
    } catch (err) {
      console.error('Failed to load limits:', err);
    }
  };

  const handleCreateClick = () => {
    if (limits && limits.max_alert_rules !== -1 && limits.current_alert_rules >= limits.max_alert_rules) {
      setShowUpgradeModal(true);
    } else {
      setShowCreateModal(true);
    }
  };

  const handleAlertCreated = () => {
    setShowCreateModal(false);
    loadAlerts();
    loadLimits();
  };

  const handleAlertUpdated = () => {
    setShowEditModal(false);
    setSelectedAlert(null);
    loadAlerts();
  };

  const handleEditClick = (alert: AlertRuleWithDetails) => {
    setSelectedAlert(alert);
    setShowEditModal(true);
  };

  const handleToggleActive = async (alert: AlertRuleWithDetails) => {
    try {
      await alertAPI.updateAlert(alert.id, { is_active: !alert.is_active });
      loadAlerts();
    } catch (err: any) {
      window.alert(`Failed to update alert: ${err.message}`);
    }
  };

  const handleDelete = async (alertId: string) => {
    if (!confirm('Are you sure you want to delete this alert?')) return;

    try {
      await alertAPI.deleteAlert(alertId);
      loadAlerts();
      loadLimits();
    } catch (err: any) {
      window.alert(`Failed to delete alert: ${err.message}`);
    }
  };

  const filteredAlerts = alerts;

  const activeCount = alerts.filter((a) => a.is_active).length;
  const inactiveCount = alerts.filter((a) => !a.is_active).length;

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900 py-8 px-4">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h1 className="text-4xl font-bold text-ic-text-primary mb-2">Alert Rules</h1>
              <p className="text-gray-300">
                Manage your price, volume, and event alerts
              </p>
            </div>
            <button
              onClick={handleCreateClick}
              className="px-6 py-3 bg-gradient-to-r from-purple-600 to-blue-600 text-ic-text-primary rounded-lg hover:from-purple-700 hover:to-blue-700 font-semibold flex items-center gap-2 shadow-lg"
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
              Create Alert
            </button>
          </div>

          {/* Subscription Limits */}
          {limits && (
            <div className="bg-slate-800/50 backdrop-blur-sm border border-slate-700 rounded-lg p-4 flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div className="text-sm">
                  <span className="text-ic-text-dim">Active Alerts: </span>
                  <span className="text-ic-text-primary font-semibold">
                    {limits.current_alert_rules} / {limits.max_alert_rules === -1 ? '∞' : limits.max_alert_rules}
                  </span>
                </div>
                {limits.max_alert_rules !== -1 && limits.current_alert_rules >= limits.max_alert_rules * 0.8 && (
                  <div className="text-sm text-yellow-400 flex items-center gap-1">
                    <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                    </svg>
                    Approaching limit
                  </div>
                )}
              </div>
              {limits.max_alert_rules !== -1 && (
                <button
                  onClick={() => setShowUpgradeModal(true)}
                  className="text-sm text-purple-400 hover:text-purple-300 font-semibold"
                >
                  Upgrade Plan →
                </button>
              )}
            </div>
          )}
        </div>

        {/* Filter Tabs */}
        <div className="flex gap-2 mb-6">
          <button
            onClick={() => setFilter('all')}
            className={`px-4 py-2 rounded-lg font-medium transition-colors ${
              filter === 'all'
                ? 'bg-ic-purple text-ic-text-primary'
                : 'bg-slate-800/50 text-ic-text-dim hover:text-ic-text-primary border border-slate-700'
            }`}
          >
            All ({alerts.length})
          </button>
          <button
            onClick={() => setFilter('active')}
            className={`px-4 py-2 rounded-lg font-medium transition-colors ${
              filter === 'active'
                ? 'bg-ic-positive text-ic-text-primary'
                : 'bg-slate-800/50 text-ic-text-dim hover:text-ic-text-primary border border-slate-700'
            }`}
          >
            Active ({activeCount})
          </button>
          <button
            onClick={() => setFilter('inactive')}
            className={`px-4 py-2 rounded-lg font-medium transition-colors ${
              filter === 'inactive'
                ? 'bg-ic-bg-tertiary text-ic-text-primary'
                : 'bg-slate-800/50 text-ic-text-dim hover:text-ic-text-primary border border-slate-700'
            }`}
          >
            Inactive ({inactiveCount})
          </button>
        </div>

        {/* Loading State */}
        {loading && (
          <div className="flex justify-center items-center py-20">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-purple-500"></div>
          </div>
        )}

        {/* Error State */}
        {error && (
          <div className="bg-red-900/20 border border-red-500 rounded-lg p-4 mb-6">
            <div className="flex items-center gap-2">
              <svg className="w-5 h-5 text-red-500" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
              </svg>
              <span className="text-red-400">{error}</span>
            </div>
          </div>
        )}

        {/* Alert List */}
        {!loading && !error && (
          <>
            {filteredAlerts.length === 0 ? (
              <div className="text-center py-20">
                <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-slate-800 mb-4">
                  <svg className="w-8 h-8 text-ic-text-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
                  </svg>
                </div>
                <h3 className="text-xl font-semibold text-ic-text-primary mb-2">
                  {filter === 'all' ? 'No alerts yet' : `No ${filter} alerts`}
                </h3>
                <p className="text-ic-text-dim mb-6">
                  Create your first alert to get notified about price movements, volume spikes, or market events
                </p>
                <button
                  onClick={handleCreateClick}
                  className="px-6 py-3 bg-ic-purple text-ic-text-primary rounded-lg hover:bg-ic-purple-hover font-semibold"
                >
                  Create Your First Alert
                </button>
              </div>
            ) : (
              <div className="grid gap-4">
                {filteredAlerts.map((alert) => (
                  <AlertCard
                    key={alert.id}
                    alert={alert}
                    onEdit={handleEditClick}
                    onToggleActive={handleToggleActive}
                    onDelete={handleDelete}
                  />
                ))}
              </div>
            )}
          </>
        )}
      </div>

      {/* Modals */}
      {showCreateModal && (
        <CreateAlertModal
          onClose={() => setShowCreateModal(false)}
          onSuccess={handleAlertCreated}
        />
      )}

      {/* TODO: Implement EditAlertModal and UpgradeModal */}
      {/*
      {showEditModal && selectedAlert && (
        <EditAlertModal
          alert={selectedAlert}
          onClose={() => {
            setShowEditModal(false);
            setSelectedAlert(null);
          }}
          onSuccess={handleAlertUpdated}
        />
      )}

      {showUpgradeModal && (
        <UpgradeModal
          onClose={() => setShowUpgradeModal(false)}
          reason="alert_limit"
        />
      )}
      */}
    </div>
  );
}
