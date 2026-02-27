import {
  formatLargeNumber,
  formatRelativeTime,
  safeToFixed,
  safeParseNumber,
} from '../utils';

// ──────────────────────────────────────────────────────────────
// formatLargeNumber
// ──────────────────────────────────────────────────────────────
describe('formatLargeNumber', () => {
  it('returns N/A for null', () => {
    expect(formatLargeNumber(null)).toBe('N/A');
  });

  it('returns N/A for undefined', () => {
    expect(formatLargeNumber(undefined)).toBe('N/A');
  });

  it('returns N/A for non-numeric string', () => {
    expect(formatLargeNumber('abc')).toBe('N/A');
  });

  it('formats trillions', () => {
    expect(formatLargeNumber(1e12)).toBe('$1.0T');
    expect(formatLargeNumber(2.5e12)).toBe('$2.5T');
  });

  it('formats billions', () => {
    expect(formatLargeNumber(1e9)).toBe('$1.0B');
    expect(formatLargeNumber(3.75e9)).toBe('$3.8B');
  });

  it('formats millions', () => {
    expect(formatLargeNumber(1e6)).toBe('$1.0M');
    expect(formatLargeNumber(250e6)).toBe('$250.0M');
  });

  it('formats small numbers with two decimals', () => {
    expect(formatLargeNumber(1234.56)).toBe('$1234.56');
    expect(formatLargeNumber(0)).toBe('$0.00');
  });

  it('handles string input', () => {
    expect(formatLargeNumber('5000000000')).toBe('$5.0B');
    expect(formatLargeNumber('1500000')).toBe('$1.5M');
  });

  it('handles numeric string', () => {
    expect(formatLargeNumber('42.5')).toBe('$42.50');
  });
});

// ──────────────────────────────────────────────────────────────
// formatRelativeTime
// ──────────────────────────────────────────────────────────────
describe('formatRelativeTime', () => {
  beforeEach(() => {
    // Fix "now" to a known time for predictable tests
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2025-06-15T12:00:00Z'));
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('returns Unknown for null', () => {
    expect(formatRelativeTime(null)).toBe('Unknown');
  });

  it('returns Unknown for undefined', () => {
    expect(formatRelativeTime(undefined)).toBe('Unknown');
  });

  it('returns Unknown for empty string', () => {
    expect(formatRelativeTime('')).toBe('Unknown');
  });

  it('returns Unknown for invalid date string', () => {
    expect(formatRelativeTime('not-a-date')).toBe('Unknown');
  });

  it('returns "Just now" for dates less than 60 seconds ago', () => {
    const date = new Date('2025-06-15T11:59:30Z'); // 30 seconds ago
    expect(formatRelativeTime(date)).toBe('Just now');
  });

  it('returns minutes ago', () => {
    const date = new Date('2025-06-15T11:55:00Z'); // 5 minutes ago
    expect(formatRelativeTime(date)).toBe('5 minutes ago');
  });

  it('returns singular "minute" for 1 minute', () => {
    const date = new Date('2025-06-15T11:59:00Z'); // 1 minute ago
    expect(formatRelativeTime(date)).toBe('1 minute ago');
  });

  it('returns hours ago', () => {
    const date = new Date('2025-06-15T09:00:00Z'); // 3 hours ago
    expect(formatRelativeTime(date)).toBe('3 hours ago');
  });

  it('returns singular "hour" for 1 hour', () => {
    const date = new Date('2025-06-15T11:00:00Z'); // 1 hour ago
    expect(formatRelativeTime(date)).toBe('1 hour ago');
  });

  it('returns "Yesterday" for 1 day ago', () => {
    const date = new Date('2025-06-14T12:00:00Z'); // exactly 1 day ago
    expect(formatRelativeTime(date)).toBe('Yesterday');
  });

  it('returns days ago for 2-6 days', () => {
    const date = new Date('2025-06-12T12:00:00Z'); // 3 days ago
    expect(formatRelativeTime(date)).toBe('3 days ago');
  });

  it('returns weeks ago for 7-29 days', () => {
    const date = new Date('2025-06-01T12:00:00Z'); // 14 days ago
    expect(formatRelativeTime(date)).toBe('2 weeks ago');
  });

  it('returns singular "week" for 1 week', () => {
    const date = new Date('2025-06-08T12:00:00Z'); // 7 days ago
    expect(formatRelativeTime(date)).toBe('1 week ago');
  });

  it('returns months ago for 30-364 days', () => {
    const date = new Date('2025-03-15T12:00:00Z'); // ~3 months ago
    expect(formatRelativeTime(date)).toBe('3 months ago');
  });

  it('returns singular "month" for 1 month', () => {
    const date = new Date('2025-05-15T12:00:00Z'); // ~1 month ago (31 days)
    expect(formatRelativeTime(date)).toBe('1 month ago');
  });

  it('returns years ago for 365+ days', () => {
    const date = new Date('2023-06-15T12:00:00Z'); // 2 years ago
    expect(formatRelativeTime(date)).toBe('2 years ago');
  });

  it('returns singular "year" for 1 year', () => {
    const date = new Date('2024-06-15T12:00:00Z'); // exactly 1 year ago
    expect(formatRelativeTime(date)).toBe('1 year ago');
  });

  it('accepts a string date input', () => {
    expect(formatRelativeTime('2025-06-15T11:55:00Z')).toBe('5 minutes ago');
  });

  it('accepts a Date object', () => {
    const date = new Date('2025-06-15T11:55:00Z');
    expect(formatRelativeTime(date)).toBe('5 minutes ago');
  });
});

// ──────────────────────────────────────────────────────────────
// safeToFixed
// ──────────────────────────────────────────────────────────────
describe('safeToFixed', () => {
  it('returns N/A for null', () => {
    expect(safeToFixed(null)).toBe('N/A');
  });

  it('returns N/A for undefined', () => {
    expect(safeToFixed(undefined)).toBe('N/A');
  });

  it('returns N/A for "N/A" string', () => {
    expect(safeToFixed('N/A')).toBe('N/A');
  });

  it('returns N/A for empty string', () => {
    expect(safeToFixed('')).toBe('N/A');
  });

  it('returns N/A for non-numeric string', () => {
    expect(safeToFixed('abc')).toBe('N/A');
  });

  it('formats number with default 2 decimals', () => {
    expect(safeToFixed(12.3456)).toBe('12.35');
  });

  it('formats number with custom decimal places', () => {
    expect(safeToFixed(12.3456, 3)).toBe('12.346');
  });

  it('formats integer with decimals', () => {
    expect(safeToFixed(42)).toBe('42.00');
  });

  it('formats zero', () => {
    expect(safeToFixed(0)).toBe('0.00');
  });

  it('formats string number', () => {
    expect(safeToFixed('3.14159')).toBe('3.14');
  });

  it('formats with 0 decimal places', () => {
    expect(safeToFixed(9.87, 0)).toBe('10');
  });
});

// ──────────────────────────────────────────────────────────────
// safeParseNumber
// ──────────────────────────────────────────────────────────────
describe('safeParseNumber', () => {
  it('returns default for null', () => {
    expect(safeParseNumber(null)).toBe(0);
  });

  it('returns default for undefined', () => {
    expect(safeParseNumber(undefined)).toBe(0);
  });

  it('returns default for empty string', () => {
    expect(safeParseNumber('')).toBe(0);
  });

  it('returns custom default for null', () => {
    expect(safeParseNumber(null, 42)).toBe(42);
  });

  it('returns default for non-numeric string', () => {
    expect(safeParseNumber('abc')).toBe(0);
  });

  it('parses integer string', () => {
    expect(safeParseNumber('42')).toBe(42);
  });

  it('parses float string', () => {
    expect(safeParseNumber('3.14')).toBe(3.14);
  });

  it('parses a number directly', () => {
    expect(safeParseNumber(99.9)).toBe(99.9);
  });

  it('returns 0 for NaN value', () => {
    expect(safeParseNumber(NaN)).toBe(0);
  });

  it('parses negative numbers', () => {
    expect(safeParseNumber('-5.5')).toBe(-5.5);
  });

  it('returns custom default for NaN', () => {
    expect(safeParseNumber('not-a-number', -1)).toBe(-1);
  });
});
