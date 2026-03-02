import { getMarketStatus, getCountdown, isTradingDay, isNYSEHoliday } from '../market-status';

describe('getMarketStatus', () => {
  it('returns regular during market hours', () => {
    // Tuesday Jan 6, 2026, 10:00 AM ET = 15:00 UTC (EST)
    const now = new Date('2026-01-06T15:00:00Z');
    const status = getMarketStatus(now);
    expect(status.state).toBe('regular');
    expect(status.label).toBe('Market Open');
    expect(status.color).toBe('green');
  });

  it('returns pre_market before market open', () => {
    // Wednesday Jan 7, 2026, 7:00 AM ET = 12:00 UTC (EST)
    const now = new Date('2026-01-07T12:00:00Z');
    const status = getMarketStatus(now);
    expect(status.state).toBe('pre_market');
    expect(status.label).toBe('Pre-Market');
    expect(status.color).toBe('amber');
  });

  it('returns after_hours after market close', () => {
    // Monday Jan 5, 2026, 5:00 PM ET = 22:00 UTC (EST)
    const now = new Date('2026-01-05T22:00:00Z');
    const status = getMarketStatus(now);
    expect(status.state).toBe('after_hours');
    expect(status.label).toBe('After Hours');
    expect(status.color).toBe('amber');
  });

  it('returns closed on Saturday', () => {
    // Saturday Jan 10, 2026, 2:00 PM ET = 19:00 UTC (EST)
    const now = new Date('2026-01-10T19:00:00Z');
    const status = getMarketStatus(now);
    expect(status.state).toBe('closed');
    expect(status.label).toBe('Market Closed');
  });

  it('returns closed on Sunday', () => {
    // Sunday Jan 11, 2026, 2:00 PM ET = 19:00 UTC (EST)
    const now = new Date('2026-01-11T19:00:00Z');
    const status = getMarketStatus(now);
    expect(status.state).toBe('closed');
  });

  it('returns closed on NYSE holiday', () => {
    // Thursday Jan 1, 2026, 10:00 AM ET = 15:00 UTC (EST)
    const now = new Date('2026-01-01T15:00:00Z');
    const status = getMarketStatus(now);
    expect(status.state).toBe('closed');
    expect(status.label).toContain('Holiday');
  });

  it('returns regular at exactly 9:30 AM ET (market open)', () => {
    // Tuesday Jan 6, 2026, 9:30 AM ET = 14:30 UTC (EST)
    const now = new Date('2026-01-06T14:30:00Z');
    const status = getMarketStatus(now);
    expect(status.state).toBe('regular');
  });

  it('returns after_hours at exactly 4:00 PM ET (market close)', () => {
    // Tuesday Jan 6, 2026, 4:00 PM ET = 21:00 UTC (EST)
    const now = new Date('2026-01-06T21:00:00Z');
    const status = getMarketStatus(now);
    expect(status.state).toBe('after_hours');
  });

  it('returns closed before pre-market (2:00 AM ET)', () => {
    // Tuesday Jan 6, 2026, 2:00 AM ET = 07:00 UTC (EST)
    const now = new Date('2026-01-06T07:00:00Z');
    const status = getMarketStatus(now);
    expect(status.state).toBe('closed');
  });

  it('returns closed after after-hours (9:00 PM ET)', () => {
    // Tuesday Jan 6, 2026, 9:00 PM ET = Jan 7 02:00 UTC (EST)
    const now = new Date('2026-01-07T02:00:00Z');
    const status = getMarketStatus(now);
    expect(status.state).toBe('closed');
  });
});

describe('isTradingDay', () => {
  it('returns true for a weekday non-holiday', () => {
    // Tuesday Jan 6, 2026
    expect(isTradingDay(new Date('2026-01-06T15:00:00Z'))).toBe(true);
  });

  it('returns false for Saturday', () => {
    expect(isTradingDay(new Date('2026-01-10T19:00:00Z'))).toBe(false);
  });

  it('returns false for Sunday', () => {
    expect(isTradingDay(new Date('2026-01-11T19:00:00Z'))).toBe(false);
  });

  it('returns false for an NYSE holiday', () => {
    // New Year's Day 2026
    expect(isTradingDay(new Date('2026-01-01T15:00:00Z'))).toBe(false);
  });
});

describe('isNYSEHoliday', () => {
  it('returns true for a known holiday', () => {
    expect(isNYSEHoliday('2026-01-01')).toBe(true);
  });

  it('returns false for a regular trading day', () => {
    expect(isNYSEHoliday('2026-01-06')).toBe(false);
  });
});

describe('getCountdown', () => {
  it('formats hours and minutes', () => {
    const now = new Date('2026-01-06T15:00:00Z');
    const target = new Date('2026-01-06T17:15:00Z'); // 2h 15m later
    expect(getCountdown(target, now)).toBe('2h 15m');
  });

  it('returns 0m for a past target time', () => {
    const now = new Date('2026-01-06T17:00:00Z');
    const target = new Date('2026-01-06T15:00:00Z'); // 2h earlier
    expect(getCountdown(target, now)).toBe('0m');
  });

  it('formats exact hours without minutes', () => {
    const now = new Date('2026-01-06T15:00:00Z');
    const target = new Date('2026-01-06T18:00:00Z'); // exactly 3h later
    expect(getCountdown(target, now)).toBe('3h');
  });

  it('formats minutes only when less than one hour', () => {
    const now = new Date('2026-01-06T15:00:00Z');
    const target = new Date('2026-01-06T15:45:00Z'); // 45m later
    expect(getCountdown(target, now)).toBe('45m');
  });
});
