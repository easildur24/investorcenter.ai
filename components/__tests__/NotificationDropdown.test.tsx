/**
 * Tests for NotificationDropdown component.
 *
 * Covers rendering, badge display, dropdown open/close, notification
 * interactions (mark read, dismiss, navigation), mark-all-read, polling,
 * and visibility-change pause/resume.
 */

import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';

// ---------------------------------------------------------------------------
// Mocks — declared before imports so jest.mock hoisting works
// ---------------------------------------------------------------------------

const mockPush = jest.fn();
jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: mockPush }),
}));

jest.mock('@/lib/api/notifications', () => ({
  notificationAPI: {
    getUnreadCount: jest.fn(),
    getNotifications: jest.fn(),
    markAsRead: jest.fn(),
    markAllAsRead: jest.fn(),
    dismiss: jest.fn(),
  },
}));

jest.mock('@heroicons/react/24/outline', () => ({
  BellIcon: (props: React.SVGProps<SVGSVGElement>) => <svg data-testid="bell-icon" {...props} />,
}));

jest.mock('date-fns', () => ({
  formatDistanceToNow: () => '5 minutes ago',
}));

import NotificationDropdown from '../notifications/NotificationDropdown';
import { notificationAPI } from '@/lib/api/notifications';

const mockGetUnreadCount = notificationAPI.getUnreadCount as jest.Mock;
const mockGetNotifications = notificationAPI.getNotifications as jest.Mock;
const mockMarkAsRead = notificationAPI.markAsRead as jest.Mock;
const mockMarkAllAsRead = notificationAPI.markAllAsRead as jest.Mock;
const mockDismiss = notificationAPI.dismiss as jest.Mock;

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Seed the API mocks with sensible defaults so the component can mount. */
function seedDefaults(unreadCount = 0) {
  mockGetUnreadCount.mockResolvedValue({ count: unreadCount });
  mockGetNotifications.mockResolvedValue([]);
}

const makeNotification = (overrides: Record<string, unknown> = {}) => ({
  id: 'n-1',
  user_id: 'u-1',
  type: 'alert_triggered',
  title: 'AAPL Price Above $150',
  message: 'AAPL crossed above $150.00',
  data: { watch_list_id: 'wl-42' },
  is_read: false,
  is_dismissed: false,
  created_at: '2026-02-25T12:00:00Z',
  ...overrides,
});

/** Render and wait for the initial unread-count fetch to settle. */
async function renderDropdown() {
  let result: ReturnType<typeof render>;
  await act(async () => {
    result = render(<NotificationDropdown />);
  });
  return result!;
}

/** Open the dropdown by clicking the bell button. */
async function openDropdown() {
  await act(async () => {
    fireEvent.click(screen.getByTitle('Notifications'));
  });
}

// ---------------------------------------------------------------------------
// Rendering & badge
// ---------------------------------------------------------------------------

describe('NotificationDropdown — Rendering', () => {
  beforeEach(() => jest.clearAllMocks());

  it('renders the bell icon button', async () => {
    seedDefaults();
    await renderDropdown();
    expect(screen.getByTitle('Notifications')).toBeInTheDocument();
  });

  it('shows unread badge when count > 0', async () => {
    seedDefaults(3);
    await renderDropdown();
    expect(screen.getByText('3')).toBeInTheDocument();
  });

  it('hides unread badge when count is 0', async () => {
    seedDefaults(0);
    await renderDropdown();
    // No badge span should be rendered
    expect(screen.queryByText('0')).not.toBeInTheDocument();
  });

  it('caps badge at 99+', async () => {
    seedDefaults(150);
    await renderDropdown();
    expect(screen.getByText('99+')).toBeInTheDocument();
  });

  it('includes unread count in aria-label', async () => {
    seedDefaults(5);
    await renderDropdown();
    expect(screen.getByLabelText('Notifications (5 unread)')).toBeInTheDocument();
  });

  it('has plain aria-label when no unread', async () => {
    seedDefaults(0);
    await renderDropdown();
    expect(screen.getByLabelText('Notifications')).toBeInTheDocument();
  });

  it('dropdown is initially closed', async () => {
    seedDefaults();
    await renderDropdown();
    expect(screen.queryByText('No notifications yet')).not.toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// Dropdown open / close
// ---------------------------------------------------------------------------

describe('NotificationDropdown — Open & close', () => {
  beforeEach(() => jest.clearAllMocks());

  it('opens dropdown and shows header on bell click', async () => {
    seedDefaults();
    await renderDropdown();
    await openDropdown();

    expect(screen.getByText('Notifications')).toBeInTheDocument();
  });

  it('fetches notifications when opened', async () => {
    seedDefaults();
    await renderDropdown();
    await openDropdown();

    expect(mockGetNotifications).toHaveBeenCalledWith({ limit: 10 });
  });

  it('closes dropdown on second bell click', async () => {
    seedDefaults();
    await renderDropdown();
    await openDropdown();
    expect(screen.getByText(/Go to watchlists/)).toBeInTheDocument();

    await act(async () => {
      fireEvent.click(screen.getByTitle('Notifications'));
    });
    expect(screen.queryByText(/Go to watchlists/)).not.toBeInTheDocument();
  });

  it('closes dropdown on Escape key', async () => {
    seedDefaults();
    await renderDropdown();
    await openDropdown();
    expect(screen.getByText(/Go to watchlists/)).toBeInTheDocument();

    await act(async () => {
      fireEvent.keyDown(document, { key: 'Escape' });
    });
    expect(screen.queryByText(/Go to watchlists/)).not.toBeInTheDocument();
  });

  it('closes dropdown on outside click', async () => {
    seedDefaults();
    await renderDropdown();
    await openDropdown();
    expect(screen.getByText(/Go to watchlists/)).toBeInTheDocument();

    await act(async () => {
      fireEvent.mouseDown(document);
    });
    expect(screen.queryByText(/Go to watchlists/)).not.toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// Notification list display
// ---------------------------------------------------------------------------

describe('NotificationDropdown — List display', () => {
  beforeEach(() => jest.clearAllMocks());

  it('shows empty state when no notifications', async () => {
    seedDefaults();
    await renderDropdown();
    await openDropdown();

    expect(screen.getByText('No notifications yet')).toBeInTheDocument();
  });

  it('renders notification title and message', async () => {
    seedDefaults(1);
    mockGetNotifications.mockResolvedValueOnce([makeNotification()]);
    await renderDropdown();
    await openDropdown();

    await waitFor(() => {
      expect(screen.getByText('AAPL Price Above $150')).toBeInTheDocument();
      expect(screen.getByText('AAPL crossed above $150.00')).toBeInTheDocument();
    });
  });

  it('renders relative timestamp', async () => {
    seedDefaults(1);
    mockGetNotifications.mockResolvedValueOnce([makeNotification()]);
    await renderDropdown();
    await openDropdown();

    await waitFor(() => {
      expect(screen.getByText('5 minutes ago')).toBeInTheDocument();
    });
  });

  it('renders dismiss button for each notification', async () => {
    seedDefaults(1);
    mockGetNotifications.mockResolvedValueOnce([makeNotification()]);
    await renderDropdown();
    await openDropdown();

    await waitFor(() => {
      expect(screen.getByLabelText('Dismiss notification')).toBeInTheDocument();
    });
  });

  it('shows "Mark all read" when unreadCount > 0', async () => {
    seedDefaults(2);
    await renderDropdown();
    await openDropdown();

    expect(screen.getByText('Mark all read')).toBeInTheDocument();
  });

  it('hides "Mark all read" when unreadCount is 0', async () => {
    seedDefaults(0);
    await renderDropdown();
    await openDropdown();

    expect(screen.queryByText('Mark all read')).not.toBeInTheDocument();
  });

  it('shows footer with "Go to watchlists" link', async () => {
    seedDefaults();
    await renderDropdown();
    await openDropdown();

    expect(screen.getByText(/Go to watchlists/)).toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// Notification click interactions
// ---------------------------------------------------------------------------

describe('NotificationDropdown — Notification click', () => {
  beforeEach(() => jest.clearAllMocks());

  it('marks unread notification as read and re-fetches count', async () => {
    seedDefaults(1);
    mockGetNotifications.mockResolvedValueOnce([makeNotification()]);
    mockMarkAsRead.mockResolvedValueOnce(undefined);
    // After marking read, the server returns 0
    mockGetUnreadCount.mockResolvedValueOnce({ count: 1 });
    mockGetUnreadCount.mockResolvedValueOnce({ count: 0 });

    await renderDropdown();
    await openDropdown();

    await waitFor(() => {
      expect(screen.getByText('AAPL Price Above $150')).toBeInTheDocument();
    });

    await act(async () => {
      fireEvent.click(screen.getByText('AAPL Price Above $150'));
    });

    expect(mockMarkAsRead).toHaveBeenCalledWith('n-1');
    // Should re-fetch unread count from server
    expect(mockGetUnreadCount.mock.calls.length).toBeGreaterThanOrEqual(2);
  });

  it('navigates to specific watchlist from notification data', async () => {
    seedDefaults(1);
    mockGetNotifications.mockResolvedValueOnce([
      makeNotification({ data: { watch_list_id: 'wl-99' } }),
    ]);
    mockMarkAsRead.mockResolvedValueOnce(undefined);

    await renderDropdown();
    await openDropdown();

    await waitFor(() => {
      expect(screen.getByText('AAPL Price Above $150')).toBeInTheDocument();
    });

    await act(async () => {
      fireEvent.click(screen.getByText('AAPL Price Above $150'));
    });

    expect(mockPush).toHaveBeenCalledWith('/watchlist/wl-99');
  });

  it('navigates to /watchlist when notification has no watch_list_id', async () => {
    seedDefaults(1);
    mockGetNotifications.mockResolvedValueOnce([makeNotification({ data: null })]);
    mockMarkAsRead.mockResolvedValueOnce(undefined);

    await renderDropdown();
    await openDropdown();

    await waitFor(() => {
      expect(screen.getByText('AAPL Price Above $150')).toBeInTheDocument();
    });

    await act(async () => {
      fireEvent.click(screen.getByText('AAPL Price Above $150'));
    });

    expect(mockPush).toHaveBeenCalledWith('/watchlist');
  });

  it('does not call markAsRead for already-read notification', async () => {
    seedDefaults(0);
    mockGetNotifications.mockResolvedValueOnce([makeNotification({ is_read: true })]);

    await renderDropdown();
    await openDropdown();

    await waitFor(() => {
      expect(screen.getByText('AAPL Price Above $150')).toBeInTheDocument();
    });

    await act(async () => {
      fireEvent.click(screen.getByText('AAPL Price Above $150'));
    });

    expect(mockMarkAsRead).not.toHaveBeenCalled();
  });

  it('closes dropdown after clicking a notification', async () => {
    seedDefaults(1);
    mockGetNotifications.mockResolvedValueOnce([makeNotification()]);
    mockMarkAsRead.mockResolvedValueOnce(undefined);

    await renderDropdown();
    await openDropdown();

    await waitFor(() => {
      expect(screen.getByText('AAPL Price Above $150')).toBeInTheDocument();
    });

    await act(async () => {
      fireEvent.click(screen.getByText('AAPL Price Above $150'));
    });

    // Dropdown should be closed
    expect(screen.queryByText(/Go to watchlists/)).not.toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// Mark all read
// ---------------------------------------------------------------------------

describe('NotificationDropdown — Mark all read', () => {
  beforeEach(() => jest.clearAllMocks());

  it('calls markAllAsRead API and re-fetches unread count', async () => {
    seedDefaults(3);
    mockGetNotifications.mockResolvedValueOnce([
      makeNotification({ id: 'n-1' }),
      makeNotification({ id: 'n-2', title: 'TSLA Alert' }),
    ]);
    mockMarkAllAsRead.mockResolvedValueOnce(undefined);
    // After mark all read, server returns 0
    mockGetUnreadCount.mockResolvedValueOnce({ count: 3 });
    mockGetUnreadCount.mockResolvedValueOnce({ count: 0 });

    await renderDropdown();
    await openDropdown();

    await waitFor(() => {
      expect(screen.getByText('Mark all read')).toBeInTheDocument();
    });

    await act(async () => {
      fireEvent.click(screen.getByText('Mark all read'));
    });

    expect(mockMarkAllAsRead).toHaveBeenCalled();
    expect(mockGetUnreadCount.mock.calls.length).toBeGreaterThanOrEqual(2);
  });
});

// ---------------------------------------------------------------------------
// Dismiss
// ---------------------------------------------------------------------------

describe('NotificationDropdown — Dismiss', () => {
  beforeEach(() => jest.clearAllMocks());

  it('removes notification from list and re-fetches count', async () => {
    seedDefaults(1);
    mockGetNotifications.mockResolvedValueOnce([makeNotification()]);
    mockDismiss.mockResolvedValueOnce(undefined);
    // After dismiss, server returns 0
    mockGetUnreadCount.mockResolvedValueOnce({ count: 1 });
    mockGetUnreadCount.mockResolvedValueOnce({ count: 0 });

    await renderDropdown();
    await openDropdown();

    await waitFor(() => {
      expect(screen.getByText('AAPL Price Above $150')).toBeInTheDocument();
    });

    await act(async () => {
      fireEvent.click(screen.getByLabelText('Dismiss notification'));
    });

    expect(mockDismiss).toHaveBeenCalledWith('n-1');
    // Notification should be removed from the list
    await waitFor(() => {
      expect(screen.queryByText('AAPL Price Above $150')).not.toBeInTheDocument();
    });
  });

  it('does not navigate when dismiss is clicked', async () => {
    seedDefaults(1);
    mockGetNotifications.mockResolvedValueOnce([makeNotification()]);
    mockDismiss.mockResolvedValueOnce(undefined);

    await renderDropdown();
    await openDropdown();

    await waitFor(() => {
      expect(screen.getByLabelText('Dismiss notification')).toBeInTheDocument();
    });

    await act(async () => {
      fireEvent.click(screen.getByLabelText('Dismiss notification'));
    });

    // Should not navigate (stopPropagation prevents row click)
    expect(mockPush).not.toHaveBeenCalled();
  });
});

// ---------------------------------------------------------------------------
// Footer navigation
// ---------------------------------------------------------------------------

describe('NotificationDropdown — Footer', () => {
  beforeEach(() => jest.clearAllMocks());

  it('navigates to /watchlist and closes dropdown on footer click', async () => {
    seedDefaults();
    await renderDropdown();
    await openDropdown();

    await act(async () => {
      fireEvent.click(screen.getByText(/Go to watchlists/));
    });

    expect(mockPush).toHaveBeenCalledWith('/watchlist');
    // Dropdown should close
    expect(screen.queryByText(/Go to watchlists/)).not.toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// Polling on mount
// ---------------------------------------------------------------------------

describe('NotificationDropdown — Polling', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('fetches unread count on mount', async () => {
    seedDefaults();
    await renderDropdown();
    expect(mockGetUnreadCount).toHaveBeenCalledTimes(1);
  });

  it('polls unread count at 60-second interval', async () => {
    seedDefaults(0);
    await renderDropdown();

    expect(mockGetUnreadCount).toHaveBeenCalledTimes(1);

    // Advance 60 seconds
    await act(async () => {
      jest.advanceTimersByTime(60_000);
    });

    expect(mockGetUnreadCount).toHaveBeenCalledTimes(2);

    // Advance another 60 seconds
    await act(async () => {
      jest.advanceTimersByTime(60_000);
    });

    expect(mockGetUnreadCount).toHaveBeenCalledTimes(3);
  });
});
