import {
  getCronjobOverview,
  getCronjobHistory,
  getCronjobDetails,
  getCronjobMetrics,
  getCronjobSchedules,
  getStatusColor,
  getHealthStatusColor,
  getHealthStatusIcon,
  formatDuration,
  formatTimeAgo,
} from '../cronjobs';

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

beforeEach(() => {
  jest.clearAllMocks();
});

// =============================================================================
// API functions
// =============================================================================

describe('getCronjobOverview', () => {
  it('calls GET /admin/cronjobs/overview', async () => {
    mockGet.mockResolvedValueOnce({});
    await getCronjobOverview();
    expect(mockGet).toHaveBeenCalledWith('/admin/cronjobs/overview');
  });
});

describe('getCronjobHistory', () => {
  it('calls GET /admin/cronjobs/:jobName/history', async () => {
    mockGet.mockResolvedValueOnce({});
    await getCronjobHistory('daily-price-update');
    expect(mockGet).toHaveBeenCalledWith('/admin/cronjobs/daily-price-update/history');
  });

  it('appends limit and offset params', async () => {
    mockGet.mockResolvedValueOnce({});
    await getCronjobHistory('ic-score-calculator', { limit: 10, offset: 20 });
    const url = mockGet.mock.calls[0][0];
    expect(url).toContain('limit=10');
    expect(url).toContain('offset=20');
  });
});

describe('getCronjobDetails', () => {
  it('calls GET /admin/cronjobs/details/:executionId', async () => {
    mockGet.mockResolvedValueOnce({});
    await getCronjobDetails('exec-123');
    expect(mockGet).toHaveBeenCalledWith('/admin/cronjobs/details/exec-123');
  });
});

describe('getCronjobMetrics', () => {
  it('calls GET /admin/cronjobs/metrics with default period', async () => {
    mockGet.mockResolvedValueOnce({});
    await getCronjobMetrics();
    expect(mockGet).toHaveBeenCalledWith('/admin/cronjobs/metrics?period=7');
  });

  it('uses custom period', async () => {
    mockGet.mockResolvedValueOnce({});
    await getCronjobMetrics(30);
    expect(mockGet).toHaveBeenCalledWith('/admin/cronjobs/metrics?period=30');
  });
});

describe('getCronjobSchedules', () => {
  it('calls GET /admin/cronjobs/schedules', async () => {
    mockGet.mockResolvedValueOnce([]);
    await getCronjobSchedules();
    expect(mockGet).toHaveBeenCalledWith('/admin/cronjobs/schedules');
  });
});

// =============================================================================
// Pure helper functions
// =============================================================================

describe('getStatusColor', () => {
  it('returns green for success', () => {
    expect(getStatusColor('success')).toBe('text-green-600 bg-green-50');
  });

  it('returns red for failed', () => {
    expect(getStatusColor('failed')).toBe('text-red-600 bg-red-50');
  });

  it('returns red for timeout', () => {
    expect(getStatusColor('timeout')).toBe('text-red-600 bg-red-50');
  });

  it('returns yellow for running', () => {
    expect(getStatusColor('running')).toBe('text-yellow-600 bg-yellow-50');
  });

  it('returns muted for unknown', () => {
    expect(getStatusColor('unknown')).toBe('text-ic-text-muted bg-ic-surface');
  });
});

describe('getHealthStatusColor', () => {
  it('returns green for healthy', () => {
    expect(getHealthStatusColor('healthy')).toBe('text-green-600');
  });

  it('returns yellow for warning', () => {
    expect(getHealthStatusColor('warning')).toBe('text-yellow-600');
  });

  it('returns red for critical', () => {
    expect(getHealthStatusColor('critical')).toBe('text-red-600');
  });

  it('returns muted for unknown', () => {
    expect(getHealthStatusColor('unknown')).toBe('text-ic-text-muted');
  });
});

describe('getHealthStatusIcon', () => {
  it('returns correct icons', () => {
    expect(getHealthStatusIcon('healthy')).toBe('\uD83D\uDFE2');
    expect(getHealthStatusIcon('warning')).toBe('\uD83D\uDFE1');
    expect(getHealthStatusIcon('critical')).toBe('\uD83D\uDD34');
    expect(getHealthStatusIcon('unknown')).toBe('\u26AA');
  });
});

describe('formatDuration', () => {
  it('returns -- for undefined', () => {
    expect(formatDuration(undefined)).toBe('--');
  });

  it('returns -- for 0', () => {
    expect(formatDuration(0)).toBe('--');
  });

  it('formats seconds only', () => {
    expect(formatDuration(45)).toBe('45s');
  });

  it('formats minutes and seconds', () => {
    expect(formatDuration(125)).toBe('2m 5s');
  });

  it('formats hours, minutes, and seconds', () => {
    expect(formatDuration(3665)).toBe('1h 1m 5s');
  });
});

describe('formatTimeAgo', () => {
  it('returns "just now" for recent date', () => {
    const now = new Date().toISOString();
    expect(formatTimeAgo(now)).toBe('just now');
  });

  it('returns minutes ago', () => {
    const fiveMinAgo = new Date(Date.now() - 5 * 60 * 1000).toISOString();
    expect(formatTimeAgo(fiveMinAgo)).toBe('5 mins ago');
  });

  it('returns singular minute', () => {
    const oneMinAgo = new Date(Date.now() - 1 * 60 * 1000).toISOString();
    expect(formatTimeAgo(oneMinAgo)).toBe('1 min ago');
  });

  it('returns hours ago', () => {
    const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString();
    expect(formatTimeAgo(twoHoursAgo)).toBe('2 hours ago');
  });

  it('returns days ago', () => {
    const threeDaysAgo = new Date(Date.now() - 3 * 24 * 60 * 60 * 1000).toISOString();
    expect(formatTimeAgo(threeDaysAgo)).toBe('3 days ago');
  });

  it('returns formatted date for old dates', () => {
    const oldDate = new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString();
    const result = formatTimeAgo(oldDate);
    // Should return a locale date string rather than "X days ago"
    expect(result).toMatch(/\d+/);
    expect(result).not.toContain('ago');
  });
});
