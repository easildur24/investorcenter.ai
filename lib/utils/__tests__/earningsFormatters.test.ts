import {
  formatEPS,
  formatRevenue,
  formatSurprise,
  surpriseColor,
  parseDateLocal,
} from '../earningsFormatters';

// ──────────────────────────────────────────────────────────────
// formatEPS
// ──────────────────────────────────────────────────────────────
describe('formatEPS', () => {
  it('returns fallback for null', () => {
    expect(formatEPS(null)).toBe('—');
  });

  it('returns fallback for undefined', () => {
    expect(formatEPS(undefined)).toBe('—');
  });

  it('returns custom fallback when provided', () => {
    expect(formatEPS(null, 'N/A')).toBe('N/A');
  });

  it('formats positive EPS', () => {
    expect(formatEPS(1.23)).toBe('$1.23');
  });

  it('formats negative EPS', () => {
    expect(formatEPS(-0.45)).toBe('$-0.45');
  });

  it('formats zero EPS', () => {
    expect(formatEPS(0)).toBe('$0.00');
  });
});

// ──────────────────────────────────────────────────────────────
// formatRevenue
// ──────────────────────────────────────────────────────────────
describe('formatRevenue', () => {
  it('returns fallback for null', () => {
    expect(formatRevenue(null)).toBe('—');
  });

  it('returns fallback for undefined', () => {
    expect(formatRevenue(undefined)).toBe('—');
  });

  it('formats trillions', () => {
    expect(formatRevenue(1.5e12)).toBe('$1.5T');
  });

  it('formats billions', () => {
    expect(formatRevenue(2.3e9)).toBe('$2.3B');
  });

  it('formats millions', () => {
    expect(formatRevenue(450e6)).toBe('$450.0M');
  });

  it('formats thousands', () => {
    expect(formatRevenue(75_000)).toBe('$75K');
  });

  it('formats small numbers', () => {
    expect(formatRevenue(500)).toBe('$500');
  });

  it('handles negative values with correct suffix', () => {
    expect(formatRevenue(-3.2e9)).toBe('$-3.2B');
  });
});

// ──────────────────────────────────────────────────────────────
// formatSurprise
// ──────────────────────────────────────────────────────────────
describe('formatSurprise', () => {
  it('returns fallback for null', () => {
    expect(formatSurprise(null)).toBe('—');
  });

  it('returns fallback for undefined', () => {
    expect(formatSurprise(undefined)).toBe('—');
  });

  it('formats positive surprise with + sign', () => {
    expect(formatSurprise(5.3)).toBe('+5.3%');
  });

  it('formats negative surprise without + sign', () => {
    expect(formatSurprise(-2.1)).toBe('-2.1%');
  });

  it('formats zero surprise without + sign', () => {
    expect(formatSurprise(0)).toBe('0.0%');
  });
});

// ──────────────────────────────────────────────────────────────
// surpriseColor
// ──────────────────────────────────────────────────────────────
describe('surpriseColor', () => {
  it('returns dim for null', () => {
    expect(surpriseColor(null)).toBe('text-ic-text-dim');
  });

  it('returns dim for undefined', () => {
    expect(surpriseColor(undefined)).toBe('text-ic-text-dim');
  });

  it('returns green for positive surprise above threshold', () => {
    expect(surpriseColor(1.5)).toBe('text-green-400');
  });

  it('returns red for negative surprise below threshold', () => {
    expect(surpriseColor(-1.5)).toBe('text-red-400');
  });

  it('returns dim for near-zero surprise', () => {
    expect(surpriseColor(0.3)).toBe('text-ic-text-dim');
    expect(surpriseColor(-0.3)).toBe('text-ic-text-dim');
  });
});

// ──────────────────────────────────────────────────────────────
// parseDateLocal
// ──────────────────────────────────────────────────────────────
describe('parseDateLocal', () => {
  it('parses YYYY-MM-DD into a Date at noon', () => {
    const date = parseDateLocal('2026-03-15');
    expect(date.getHours()).toBe(12);
    expect(date.getMinutes()).toBe(0);
    expect(date.getDate()).toBe(15);
  });

  it('preserves the correct month and year', () => {
    const date = parseDateLocal('2026-12-25');
    expect(date.getFullYear()).toBe(2026);
    expect(date.getMonth()).toBe(11); // 0-indexed
    expect(date.getDate()).toBe(25);
  });
});
