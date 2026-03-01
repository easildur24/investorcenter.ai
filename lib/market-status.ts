/**
 * Market status utilities — determines whether the NYSE is open/closed
 * and provides countdown to the next market event.
 */

import type { MarketState, MarketStatus } from '@/lib/types/market';
import { NYSE_HOLIDAYS } from '@/config/nyse-holidays';

// ─── Constants ───────────────────────────────────────────────────────────────

/** NYSE regular session: 9:30 AM – 4:00 PM ET */
const MARKET_OPEN_HOUR = 9;
const MARKET_OPEN_MINUTE = 30;
const MARKET_CLOSE_HOUR = 16;
const MARKET_CLOSE_MINUTE = 0;

/** Extended hours */
const PRE_MARKET_START_HOUR = 4;
const PRE_MARKET_START_MINUTE = 0;
const AFTER_HOURS_END_HOUR = 20;
const AFTER_HOURS_END_MINUTE = 0;

// ─── Helpers ─────────────────────────────────────────────────────────────────

/** Convert any Date to Eastern Time components */
function toET(date: Date): {
  hours: number;
  minutes: number;
  dayOfWeek: number;
  dateStr: string;
  etDate: Date;
} {
  // Use Intl to get Eastern Time components
  const etStr = date.toLocaleString('en-US', { timeZone: 'America/New_York' });
  const etDate = new Date(etStr);
  const dateOnly = date.toLocaleDateString('en-CA', { timeZone: 'America/New_York' }); // YYYY-MM-DD

  return {
    hours: etDate.getHours(),
    minutes: etDate.getMinutes(),
    dayOfWeek: etDate.getDay(), // 0=Sun, 6=Sat
    dateStr: dateOnly,
    etDate,
  };
}

/** Check if a date string (YYYY-MM-DD) is an NYSE holiday */
export function isNYSEHoliday(dateStr: string): boolean {
  return NYSE_HOLIDAYS.includes(dateStr);
}

/** Check if the given date is a trading day (weekday + not holiday) */
export function isTradingDay(date: Date): boolean {
  const { dayOfWeek, dateStr } = toET(date);
  if (dayOfWeek === 0 || dayOfWeek === 6) return false;
  return !isNYSEHoliday(dateStr);
}

/** Get the nearest upcoming trading day (or today if it is one) */
export function getNearestTradingDay(date: Date): Date {
  const d = new Date(date);
  let attempts = 0;
  while (!isTradingDay(d) && attempts < 10) {
    d.setDate(d.getDate() + 1);
    attempts++;
  }
  return d;
}

// ─── Time Math ───────────────────────────────────────────────────────────────

function minutesOfDay(hours: number, minutes: number): number {
  return hours * 60 + minutes;
}

function formatCountdown(ms: number): string {
  const totalMinutes = Math.max(0, Math.floor(ms / 60_000));
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;

  if (hours > 0 && minutes > 0) return `${hours}h ${minutes}m`;
  if (hours > 0) return `${hours}h`;
  return `${minutes}m`;
}

// ─── Main Function ───────────────────────────────────────────────────────────

/**
 * Determine the current NYSE market state and countdown to next event.
 * Uses `now` parameter for testability.
 */
export function getMarketStatus(now: Date = new Date()): MarketStatus {
  const { hours, minutes, dayOfWeek, dateStr, etDate } = toET(now);
  const currentMinutes = minutesOfDay(hours, minutes);

  const openMinutes = minutesOfDay(MARKET_OPEN_HOUR, MARKET_OPEN_MINUTE);
  const closeMinutes = minutesOfDay(MARKET_CLOSE_HOUR, MARKET_CLOSE_MINUTE);
  const preMarketMinutes = minutesOfDay(PRE_MARKET_START_HOUR, PRE_MARKET_START_MINUTE);
  const afterHoursEndMinutes = minutesOfDay(AFTER_HOURS_END_HOUR, AFTER_HOURS_END_MINUTE);

  // Weekend or holiday → closed
  const isWeekend = dayOfWeek === 0 || dayOfWeek === 6;
  const isHoliday = isNYSEHoliday(dateStr);

  if (isWeekend || isHoliday) {
    // Find next trading day open
    const nextTradingDay = getNearestTradingDay(
      isWeekend
        ? new Date(etDate.getTime() + (dayOfWeek === 6 ? 2 : 1) * 86_400_000)
        : new Date(etDate.getTime() + 86_400_000)
    );
    nextTradingDay.setHours(MARKET_OPEN_HOUR, MARKET_OPEN_MINUTE, 0, 0);

    return {
      state: 'closed',
      label: isHoliday ? 'Market Closed (Holiday)' : 'Market Closed',
      color: 'grey',
      nextEvent: `Opens ${formatCountdown(nextTradingDay.getTime() - etDate.getTime())}`,
      nextEventTime: nextTradingDay,
    };
  }

  // Pre-market: 4:00 AM – 9:30 AM ET
  if (currentMinutes >= preMarketMinutes && currentMinutes < openMinutes) {
    const openTime = new Date(etDate);
    openTime.setHours(MARKET_OPEN_HOUR, MARKET_OPEN_MINUTE, 0, 0);
    const ms = openTime.getTime() - etDate.getTime();

    return {
      state: 'pre_market',
      label: 'Pre-Market',
      color: 'amber',
      nextEvent: `Opens in ${formatCountdown(ms)}`,
      nextEventTime: openTime,
    };
  }

  // Regular session: 9:30 AM – 4:00 PM ET
  if (currentMinutes >= openMinutes && currentMinutes < closeMinutes) {
    const closeTime = new Date(etDate);
    closeTime.setHours(MARKET_CLOSE_HOUR, MARKET_CLOSE_MINUTE, 0, 0);
    const ms = closeTime.getTime() - etDate.getTime();

    return {
      state: 'regular',
      label: 'Market Open',
      color: 'green',
      nextEvent: `Closes in ${formatCountdown(ms)}`,
      nextEventTime: closeTime,
    };
  }

  // After hours: 4:00 PM – 8:00 PM ET
  if (currentMinutes >= closeMinutes && currentMinutes < afterHoursEndMinutes) {
    const endTime = new Date(etDate);
    endTime.setHours(AFTER_HOURS_END_HOUR, AFTER_HOURS_END_MINUTE, 0, 0);
    const ms = endTime.getTime() - etDate.getTime();

    return {
      state: 'after_hours',
      label: 'After Hours',
      color: 'amber',
      nextEvent: `Ends in ${formatCountdown(ms)}`,
      nextEventTime: endTime,
    };
  }

  // Before pre-market (midnight – 4:00 AM) or after after-hours (8 PM+)
  // → "closed" until next pre-market or next day
  if (currentMinutes < preMarketMinutes) {
    // Before 4 AM → pre-market starts today
    const preMarketTime = new Date(etDate);
    preMarketTime.setHours(PRE_MARKET_START_HOUR, PRE_MARKET_START_MINUTE, 0, 0);
    const ms = preMarketTime.getTime() - etDate.getTime();

    return {
      state: 'closed',
      label: 'Market Closed',
      color: 'grey',
      nextEvent: `Pre-market in ${formatCountdown(ms)}`,
      nextEventTime: preMarketTime,
    };
  }

  // After 8 PM → next trading day
  const nextDay = new Date(etDate);
  nextDay.setDate(nextDay.getDate() + 1);
  const nextTradingDay = getNearestTradingDay(nextDay);
  nextTradingDay.setHours(PRE_MARKET_START_HOUR, PRE_MARKET_START_MINUTE, 0, 0);
  const ms = nextTradingDay.getTime() - etDate.getTime();

  return {
    state: 'closed',
    label: 'Market Closed',
    color: 'grey',
    nextEvent: `Pre-market in ${formatCountdown(ms)}`,
    nextEventTime: nextTradingDay,
  };
}

/**
 * Get a countdown string to a target time, auto-updating.
 * Returns formatted string like "2h 15m".
 */
export function getCountdown(targetTime: Date, now: Date = new Date()): string {
  const ms = targetTime.getTime() - now.getTime();
  if (ms <= 0) return '0m';
  return formatCountdown(ms);
}
