'use client';

import { useState, useEffect } from 'react';
import { XMarkIcon } from '@heroicons/react/24/outline';
import { alertAPI, ALERT_TYPES, ALERT_FREQUENCIES } from '@/lib/api/alerts';
import { watchListAPI } from '@/lib/api/watchlist';

interface CreateAlertModalProps {
  onClose: () => void;
  onSuccess: () => void;
}

interface WatchListBasic {
  id: string;
  name: string;
  item_count: number;
}

interface WatchListItem {
  symbol: string;
  name: string;
}

export default function CreateAlertModal({ onClose, onSuccess }: CreateAlertModalProps) {
  const [watchLists, setWatchLists] = useState<WatchListBasic[]>([]);
  const [watchListItems, setWatchListItems] = useState<WatchListItem[]>([]);
  const [selectedWatchList, setSelectedWatchList] = useState('');
  const [symbol, setSymbol] = useState('');
  const [name, setName] = useState('');
  const [alertType, setAlertType] = useState('price_above');
  const [threshold, setThreshold] = useState('');
  const [frequency, setFrequency] = useState<'once' | 'daily' | 'always'>('daily');
  const [notifyEmail, setNotifyEmail] = useState(true);
  const [notifyInApp, setNotifyInApp] = useState(true);
  const [loading, setLoading] = useState(false);
  const [loadingItems, setLoadingItems] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadWatchLists();
  }, []);

  useEffect(() => {
    if (selectedWatchList) {
      loadWatchListItems(selectedWatchList);
    }
  }, [selectedWatchList]);

  const loadWatchLists = async () => {
    try {
      const response = await watchListAPI.getWatchLists();
      const lists = response.watch_lists || [];
      setWatchLists(lists);
      if (lists.length > 0) {
        setSelectedWatchList(lists[0].id);
      }
    } catch (err) {
      console.error('Failed to load watchlists:', err);
      setError('Failed to load watchlists');
    }
  };

  const loadWatchListItems = async (watchListId: string) => {
    try {
      setLoadingItems(true);
      const data = await watchListAPI.getWatchList(watchListId);
      setWatchListItems(data.items || []);
      setSymbol(''); // Reset symbol when changing watchlist
    } catch (err) {
      console.error('Failed to load watchlist items:', err);
      setWatchListItems([]);
    } finally {
      setLoadingItems(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!selectedWatchList || !symbol || !name || !threshold) {
      setError('Please fill in all required fields');
      return;
    }

    const thresholdNum = parseFloat(threshold);
    if (isNaN(thresholdNum) || thresholdNum <= 0) {
      setError('Please enter a valid threshold value');
      return;
    }

    try {
      setLoading(true);
      await alertAPI.createAlert({
        watch_list_id: selectedWatchList,
        symbol: symbol,
        name: name,
        alert_type: alertType,
        conditions: { threshold: thresholdNum },
        frequency: frequency,
        notify_email: notifyEmail,
        notify_in_app: notifyInApp,
      });
      onSuccess();
    } catch (err: any) {
      setError(err.message || 'Failed to create alert');
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      <div className="flex items-center justify-center min-h-screen px-4 pt-4 pb-20 text-center sm:block sm:p-0">
        {/* Background overlay */}
        <div
          className="fixed inset-0 transition-opacity bg-ic-bg-tertiary bg-opacity-75"
          onClick={onClose}
        />

        {/* Modal panel */}
        <div className="inline-block align-bottom bg-ic-surface rounded-lg text-left overflow-hidden border border-ic-border transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full">
          <div className="bg-ic-surface px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-ic-text-primary">Create Alert</h3>
              <button onClick={onClose} className="text-ic-text-dim hover:text-ic-text-muted">
                <XMarkIcon className="h-6 w-6" />
              </button>
            </div>

            {error && (
              <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-md">
                <p className="text-sm text-ic-negative">{error}</p>
              </div>
            )}

            <form onSubmit={handleSubmit} className="space-y-4">
              {/* Watch List Selection */}
              <div>
                <label className="block text-sm font-medium text-ic-text-secondary mb-1">
                  Watch List <span className="text-ic-negative">*</span>
                </label>
                <select
                  value={selectedWatchList}
                  onChange={(e) => setSelectedWatchList(e.target.value)}
                  className="w-full px-3 py-2 border border-ic-border rounded-md focus:ring-primary-500 focus:border-primary-500"
                  required
                >
                  {watchLists.map((wl) => (
                    <option key={wl.id} value={wl.id}>
                      {wl.name} ({wl.item_count} stocks)
                    </option>
                  ))}
                </select>
              </div>

              {/* Symbol Selection */}
              <div>
                <label className="block text-sm font-medium text-ic-text-secondary mb-1">
                  Symbol <span className="text-ic-negative">*</span>
                </label>
                <select
                  value={symbol}
                  onChange={(e) => setSymbol(e.target.value)}
                  className="w-full px-3 py-2 border border-ic-border rounded-md focus:ring-primary-500 focus:border-primary-500"
                  disabled={loadingItems}
                  required
                >
                  <option value="">
                    {loadingItems ? 'Loading symbols...' : 'Select a symbol...'}
                  </option>
                  {watchListItems.map((item) => (
                    <option key={item.symbol} value={item.symbol}>
                      {item.symbol} {item.name ? `- ${item.name}` : ''}
                    </option>
                  ))}
                </select>
              </div>

              {/* Alert Name */}
              <div>
                <label className="block text-sm font-medium text-ic-text-secondary mb-1">
                  Alert Name <span className="text-ic-negative">*</span>
                </label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="e.g., AAPL Price Above $150"
                  className="w-full px-3 py-2 border border-ic-border rounded-md focus:ring-primary-500 focus:border-primary-500"
                  required
                />
              </div>

              {/* Alert Type */}
              <div>
                <label className="block text-sm font-medium text-ic-text-secondary mb-1">
                  Alert Type
                </label>
                <select
                  value={alertType}
                  onChange={(e) => setAlertType(e.target.value)}
                  className="w-full px-3 py-2 border border-ic-border rounded-md focus:ring-primary-500 focus:border-primary-500"
                >
                  {Object.entries(ALERT_TYPES).map(([key, info]) => (
                    <option key={key} value={key}>
                      {info.icon} {info.label}
                    </option>
                  ))}
                </select>
              </div>

              {/* Threshold */}
              <div>
                <label className="block text-sm font-medium text-ic-text-secondary mb-1">
                  Threshold {alertType.includes('price') ? '(Price)' : '(Volume)'}{' '}
                  <span className="text-ic-negative">*</span>
                </label>
                <input
                  type="number"
                  step={alertType.includes('price') ? '0.01' : '1'}
                  value={threshold}
                  onChange={(e) => setThreshold(e.target.value)}
                  placeholder={alertType.includes('price') ? '150.00' : '1000000'}
                  className="w-full px-3 py-2 border border-ic-border rounded-md focus:ring-primary-500 focus:border-primary-500"
                  required
                />
              </div>

              {/* Frequency */}
              <div>
                <label className="block text-sm font-medium text-ic-text-secondary mb-1">
                  Frequency
                </label>
                <select
                  value={frequency}
                  onChange={(e) => setFrequency(e.target.value as 'once' | 'daily' | 'always')}
                  className="w-full px-3 py-2 border border-ic-border rounded-md focus:ring-primary-500 focus:border-primary-500"
                >
                  {Object.entries(ALERT_FREQUENCIES).map(([key, info]) => (
                    <option key={key} value={key}>
                      {info.label} - {info.description}
                    </option>
                  ))}
                </select>
              </div>

              {/* Notification Settings */}
              <div className="space-y-2">
                <label className="block text-sm font-medium text-ic-text-secondary">
                  Notifications
                </label>
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    id="notify-email"
                    checked={notifyEmail}
                    onChange={(e) => setNotifyEmail(e.target.checked)}
                    className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-ic-border rounded"
                  />
                  <label htmlFor="notify-email" className="ml-2 text-sm text-ic-text-secondary">
                    Email notifications
                  </label>
                </div>
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    id="notify-app"
                    checked={notifyInApp}
                    onChange={(e) => setNotifyInApp(e.target.checked)}
                    className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-ic-border rounded"
                  />
                  <label htmlFor="notify-app" className="ml-2 text-sm text-ic-text-secondary">
                    In-app notifications
                  </label>
                </div>
              </div>

              {/* Action Buttons */}
              <div className="flex items-center justify-end gap-3 pt-4">
                <button
                  type="button"
                  onClick={onClose}
                  className="px-4 py-2 text-sm font-medium text-ic-text-secondary bg-ic-surface border border-ic-border rounded-md hover:bg-ic-surface-hover"
                  disabled={loading}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 text-sm font-medium text-ic-text-primary bg-ic-blue rounded-md hover:bg-ic-blue-hover disabled:opacity-50 disabled:cursor-not-allowed"
                  disabled={loading}
                >
                  {loading ? 'Creating...' : 'Create Alert'}
                </button>
              </div>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
}
