'use client';

import { AlertRuleWithDetails, ALERT_TYPES, ALERT_FREQUENCIES } from '@/lib/api/alerts';
import { formatDistanceToNow } from 'date-fns';

interface AlertCardProps {
  alert: AlertRuleWithDetails;
  onEdit: (alert: AlertRuleWithDetails) => void;
  onToggleActive: (alert: AlertRuleWithDetails) => void;
  onDelete: (alertId: string) => void;
}

export default function AlertCard({ alert, onEdit, onToggleActive, onDelete }: AlertCardProps) {
  const alertInfo = ALERT_TYPES[alert.alert_type as keyof typeof ALERT_TYPES];
  const frequencyInfo = ALERT_FREQUENCIES[alert.frequency as keyof typeof ALERT_FREQUENCIES];

  const formatConditions = (conditions: any) => {
    try {
      const cond = typeof conditions === 'string' ? JSON.parse(conditions) : conditions;

      switch (alert.alert_type) {
        case 'price_above':
        case 'price_below':
          return `$${cond.threshold?.toFixed(2) || 'N/A'}`;
        case 'price_change_pct':
          return `${cond.percent_change || 0}% ${cond.direction || 'any'}`;
        case 'volume_above':
        case 'volume_below':
          return formatVolume(cond.threshold || 0);
        case 'volume_spike':
          return `${cond.volume_multiplier || 2}x ${cond.baseline || 'avg_30d'}`;
        default:
          return 'Configured';
      }
    } catch {
      return 'Invalid conditions';
    }
  };

  const formatVolume = (vol: number) => {
    if (vol >= 1e9) return `${(vol / 1e9).toFixed(2)}B`;
    if (vol >= 1e6) return `${(vol / 1e6).toFixed(2)}M`;
    if (vol >= 1e3) return `${(vol / 1e3).toFixed(2)}K`;
    return vol.toString();
  };

  return (
    <div className={`bg-slate-800/50 backdrop-blur-sm border rounded-lg p-6 transition-all ${
      alert.is_active ? 'border-purple-500/50' : 'border-slate-700'
    }`}>
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-start gap-4 flex-1">
          {/* Alert Icon */}
          <div className={`w-12 h-12 rounded-lg flex items-center justify-center text-2xl ${
            alert.is_active
              ? 'bg-purple-600/20 border border-purple-500/50'
              : 'bg-slate-700/50 border border-slate-600'
          }`}>
            {alertInfo?.icon || '🔔'}
          </div>

          {/* Alert Info */}
          <div className="flex-1">
            <div className="flex items-center gap-3 mb-2">
              <h3 className="text-lg font-semibold text-ic-text-primary">{alert.name}</h3>
              {alert.is_active ? (
                <span className="px-2 py-1 text-xs font-semibold bg-green-600/20 text-ic-positive rounded border border-green-500/50">
                  Active
                </span>
              ) : (
                <span className="px-2 py-1 text-xs font-semibold bg-gray-600/20 text-ic-text-dim rounded border border-gray-500/50">
                  Inactive
                </span>
              )}
            </div>

            {alert.description && (
              <p className="text-ic-text-dim text-sm mb-3">{alert.description}</p>
            )}

            {/* Alert Details */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
              <div>
                <div className="text-ic-text-muted mb-1">Symbol</div>
                <div className="text-ic-text-primary font-medium">{alert.symbol}</div>
                {alert.company_name && (
                  <div className="text-ic-text-dim text-xs">{alert.company_name}</div>
                )}
              </div>

              <div>
                <div className="text-ic-text-muted mb-1">Type</div>
                <div className="text-ic-text-primary font-medium">{alertInfo?.label || alert.alert_type}</div>
              </div>

              <div>
                <div className="text-ic-text-muted mb-1">Condition</div>
                <div className="text-ic-text-primary font-medium">{formatConditions(alert.conditions)}</div>
              </div>

              <div>
                <div className="text-ic-text-muted mb-1">Frequency</div>
                <div className="text-ic-text-primary font-medium">{frequencyInfo?.label || alert.frequency}</div>
              </div>
            </div>

            {/* Notification Settings */}
            <div className="flex items-center gap-4 mt-3 text-sm">
              <div className="flex items-center gap-1">
                {alert.notify_email ? (
                  <svg className="w-4 h-4 text-ic-blue" fill="currentColor" viewBox="0 0 20 20">
                    <path d="M2.003 5.884L10 9.882l7.997-3.998A2 2 0 0016 4H4a2 2 0 00-1.997 1.884z" />
                    <path d="M18 8.118l-8 4-8-4V14a2 2 0 002 2h12a2 2 0 002-2V8.118z" />
                  </svg>
                ) : (
                  <svg className="w-4 h-4 text-ic-text-muted" fill="currentColor" viewBox="0 0 20 20">
                    <path d="M2.003 5.884L10 9.882l7.997-3.998A2 2 0 0016 4H4a2 2 0 00-1.997 1.884z" />
                    <path d="M18 8.118l-8 4-8-4V14a2 2 0 002 2h12a2 2 0 002-2V8.118z" />
                  </svg>
                )}
                <span className={alert.notify_email ? 'text-ic-blue' : 'text-ic-text-muted'}>Email</span>
              </div>

              <div className="flex items-center gap-1">
                {alert.notify_in_app ? (
                  <svg className="w-4 h-4 text-purple-400" fill="currentColor" viewBox="0 0 20 20">
                    <path d="M10 2a6 6 0 00-6 6v3.586l-.707.707A1 1 0 004 14h12a1 1 0 00.707-1.707L16 11.586V8a6 6 0 00-6-6zM10 18a3 3 0 01-3-3h6a3 3 0 01-3 3z" />
                  </svg>
                ) : (
                  <svg className="w-4 h-4 text-ic-text-muted" fill="currentColor" viewBox="0 0 20 20">
                    <path d="M10 2a6 6 0 00-6 6v3.586l-.707.707A1 1 0 004 14h12a1 1 0 00.707-1.707L16 11.586V8a6 6 0 00-6-6zM10 18a3 3 0 01-3-3h6a3 3 0 01-3 3z" />
                  </svg>
                )}
                <span className={alert.notify_in_app ? 'text-purple-400' : 'text-ic-text-muted'}>In-App</span>
              </div>
            </div>

            {/* Trigger Stats */}
            <div className="flex items-center gap-6 mt-3 text-xs text-ic-text-dim">
              <div>
                <span className="font-semibold text-ic-text-primary">{alert.trigger_count}</span> triggers
              </div>
              {alert.last_triggered_at && (
                <div>
                  Last triggered {formatDistanceToNow(new Date(alert.last_triggered_at), { addSuffix: true })}
                </div>
              )}
              <div>
                Created {formatDistanceToNow(new Date(alert.created_at), { addSuffix: true })}
              </div>
            </div>

            {/* Watch List */}
            <div className="mt-3 text-xs">
              <span className="text-ic-text-muted">Watch List:</span>{' '}
              <span className="text-gray-300">{alert.watch_list_name}</span>
            </div>
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex items-center gap-2">
          {/* Toggle Active */}
          <button
            onClick={() => onToggleActive(alert)}
            className={`p-2 rounded-lg transition-colors ${
              alert.is_active
                ? 'bg-green-600/20 text-ic-positive hover:bg-green-600/30'
                : 'bg-gray-600/20 text-ic-text-dim hover:bg-gray-600/30'
            }`}
            title={alert.is_active ? 'Deactivate' : 'Activate'}
          >
            {alert.is_active ? (
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            ) : (
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            )}
          </button>

          {/* Edit */}
          <button
            onClick={() => onEdit(alert)}
            className="p-2 bg-ic-blue/20 text-ic-blue rounded-lg hover:bg-ic-blue/30 transition-colors"
            title="Edit"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
            </svg>
          </button>

          {/* Delete */}
          <button
            onClick={() => onDelete(alert.id)}
            className="p-2 bg-red-600/20 text-ic-negative rounded-lg hover:bg-red-600/30 transition-colors"
            title="Delete"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  );
}
