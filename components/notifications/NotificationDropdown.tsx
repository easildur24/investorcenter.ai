'use client';

import { useState, useEffect, useRef, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { BellIcon } from '@heroicons/react/24/outline';
import { notificationAPI, InAppNotification } from '@/lib/api/notifications';
import { formatDistanceToNow } from 'date-fns';

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const POLL_INTERVAL_MS = 60_000; // Refresh unread count every 60 seconds
const FETCH_LIMIT = 10; // Show the 10 most recent notifications

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function NotificationDropdown() {
  const router = useRouter();
  const dropdownRef = useRef<HTMLDivElement>(null);

  const [isOpen, setIsOpen] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);
  const [notifications, setNotifications] = useState<InAppNotification[]>([]);
  const [loading, setLoading] = useState(false);

  // ── Fetch unread count (on mount + polling) ─────────────────────────

  const fetchUnreadCount = useCallback(async () => {
    try {
      const { count } = await notificationAPI.getUnreadCount();
      setUnreadCount(count);
    } catch {
      // Non-critical — silently retry on next interval
    }
  }, []);

  // Use a ref for the interval so the visibilitychange handler can
  // clear / restart it without re-running the effect.
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    const startPolling = () => {
      if (pollRef.current) clearInterval(pollRef.current);
      pollRef.current = setInterval(fetchUnreadCount, POLL_INTERVAL_MS);
    };

    // Initial fetch + start polling
    fetchUnreadCount();
    startPolling();

    // Pause polling when the browser tab is hidden to avoid wasted
    // requests, and resume (with an immediate refresh) when visible.
    const handleVisibilityChange = () => {
      if (document.hidden) {
        if (pollRef.current) {
          clearInterval(pollRef.current);
          pollRef.current = null;
        }
      } else {
        fetchUnreadCount();
        startPolling();
      }
    };
    document.addEventListener('visibilitychange', handleVisibilityChange);

    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [fetchUnreadCount]);

  // ── Fetch notifications when dropdown opens ─────────────────────────

  const fetchNotifications = useCallback(async () => {
    setLoading(true);
    try {
      const data = await notificationAPI.getNotifications({ limit: FETCH_LIMIT });
      setNotifications(data ?? []);
    } catch {
      // Non-critical — show empty state
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (isOpen) {
      fetchNotifications();
    }
  }, [isOpen, fetchNotifications]);

  // ── Click-outside to close ──────────────────────────────────────────

  useEffect(() => {
    if (!isOpen) return;
    const handler = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [isOpen]);

  // ── Escape key to close ─────────────────────────────────────────────

  useEffect(() => {
    if (!isOpen) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setIsOpen(false);
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [isOpen]);

  // ── Handlers ────────────────────────────────────────────────────────

  const toggleDropdown = () => setIsOpen((prev) => !prev);

  const handleMarkAllRead = async () => {
    try {
      await notificationAPI.markAllAsRead();
      // Re-fetch from server to stay in sync instead of local decrement
      await fetchUnreadCount();
      setNotifications((prev) =>
        prev.map((n) => ({ ...n, is_read: true, read_at: new Date().toISOString() }))
      );
    } catch {
      // Silently fail — user can retry
    }
  };

  /** Navigate to the watchlist associated with a notification, or fall back to /watchlist. */
  const getNotificationHref = (notification: InAppNotification): string => {
    const watchListId = notification.data?.watch_list_id;
    return watchListId ? `/watchlist/${watchListId}` : '/watchlist';
  };

  const handleNotificationClick = async (notification: InAppNotification) => {
    // Mark as read if unread
    if (!notification.is_read) {
      try {
        await notificationAPI.markAsRead(notification.id);
        // Re-fetch to stay in sync with the server
        await fetchUnreadCount();
        setNotifications((prev) =>
          prev.map((n) =>
            n.id === notification.id
              ? { ...n, is_read: true, read_at: new Date().toISOString() }
              : n
          )
        );
      } catch {
        // Non-critical
      }
    }

    setIsOpen(false);
    router.push(getNotificationHref(notification));
  };

  const handleDismiss = async (e: React.MouseEvent, notificationId: string) => {
    e.stopPropagation(); // Prevent triggering the row click
    try {
      await notificationAPI.dismiss(notificationId);
      setNotifications((prev) => prev.filter((n) => n.id !== notificationId));
      // Re-fetch from server to stay in sync
      await fetchUnreadCount();
    } catch {
      // Non-critical
    }
  };

  // ── Render ──────────────────────────────────────────────────────────

  return (
    <div ref={dropdownRef} className="relative">
      {/* Bell icon with badge */}
      <button
        onClick={toggleDropdown}
        className="relative p-2 text-ic-text-muted hover:text-ic-text-primary rounded-full hover:bg-ic-surface transition-colors"
        title="Notifications"
        aria-label={`Notifications${unreadCount > 0 ? ` (${unreadCount} unread)` : ''}`}
      >
        <BellIcon className="h-6 w-6" />
        {unreadCount > 0 && (
          <span className="absolute -top-0.5 -right-0.5 flex items-center justify-center min-w-[18px] h-[18px] px-1 text-[10px] font-bold bg-ic-negative text-white rounded-full leading-none">
            {unreadCount > 99 ? '99+' : unreadCount}
          </span>
        )}
      </button>

      {/* Dropdown panel */}
      {isOpen && (
        <div className="absolute right-0 mt-2 w-80 bg-ic-bg-primary rounded-lg shadow-lg border border-ic-border z-50 overflow-hidden">
          {/* Header */}
          <div className="flex items-center justify-between px-4 py-3 border-b border-ic-border">
            <h3 className="text-sm font-semibold text-ic-text-primary">Notifications</h3>
            {unreadCount > 0 && (
              <button
                onClick={handleMarkAllRead}
                className="text-xs text-ic-blue hover:text-ic-blue-hover font-medium"
              >
                Mark all read
              </button>
            )}
          </div>

          {/* Notification list */}
          <div className="max-h-[400px] overflow-y-auto">
            {loading ? (
              <div className="flex justify-center py-8">
                <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-ic-blue" />
              </div>
            ) : notifications.length === 0 ? (
              <div className="py-8 text-center">
                <BellIcon className="mx-auto h-8 w-8 text-ic-text-dim mb-2" />
                <p className="text-sm text-ic-text-dim">No notifications yet</p>
              </div>
            ) : (
              notifications.map((notification) => (
                <button
                  key={notification.id}
                  onClick={() => handleNotificationClick(notification)}
                  className={`w-full text-left px-4 py-3 border-b border-ic-border last:border-b-0 hover:bg-ic-surface transition-colors ${
                    !notification.is_read ? 'bg-ic-blue/5' : ''
                  }`}
                >
                  <div className="flex items-start gap-3">
                    {/* Unread indicator */}
                    <div className="flex-shrink-0 mt-1.5">
                      <div
                        className={`h-2 w-2 rounded-full ${
                          notification.is_read ? 'bg-transparent' : 'bg-ic-blue'
                        }`}
                      />
                    </div>

                    {/* Content */}
                    <div className="flex-1 min-w-0">
                      <p
                        className={`text-sm truncate ${
                          notification.is_read
                            ? 'text-ic-text-secondary'
                            : 'text-ic-text-primary font-medium'
                        }`}
                      >
                        {notification.title}
                      </p>
                      <p className="text-xs text-ic-text-dim mt-0.5 line-clamp-2">
                        {notification.message}
                      </p>
                      <p className="text-[10px] text-ic-text-dim mt-1">
                        {formatDistanceToNow(new Date(notification.created_at), {
                          addSuffix: true,
                        })}
                      </p>
                    </div>

                    {/* Dismiss button */}
                    <button
                      onClick={(e) => handleDismiss(e, notification.id)}
                      className="flex-shrink-0 p-1 text-ic-text-dim hover:text-ic-text-muted rounded"
                      aria-label="Dismiss notification"
                    >
                      <svg
                        className="h-3.5 w-3.5"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M6 18L18 6M6 6l12 12"
                        />
                      </svg>
                    </button>
                  </div>
                </button>
              ))
            )}
          </div>

          {/* Footer */}
          <div className="px-4 py-2.5 border-t border-ic-border bg-ic-bg-secondary">
            <button
              onClick={() => {
                setIsOpen(false);
                router.push('/watchlist');
              }}
              className="text-xs text-ic-blue hover:text-ic-blue-hover font-medium w-full text-center"
            >
              Go to watchlists &rarr;
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
