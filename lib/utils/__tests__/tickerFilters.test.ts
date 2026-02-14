import {
  isDerivativeSecurity,
  getSecurityType,
  filterDerivatives,
  categorizeTickers,
} from '../tickerFilters';

describe('isDerivativeSecurity', () => {
  it('returns false for empty string', () => {
    expect(isDerivativeSecurity('')).toBe(false);
  });

  it('returns false for common stocks', () => {
    expect(isDerivativeSecurity('AAPL')).toBe(false);
    expect(isDerivativeSecurity('MSFT')).toBe(false);
    expect(isDerivativeSecurity('GOOG')).toBe(false);
    expect(isDerivativeSecurity('A')).toBe(false);
  });

  it('identifies warrants (W suffix)', () => {
    expect(isDerivativeSecurity('ABPW')).toBe(true);
    expect(isDerivativeSecurity('SPAKW')).toBe(true);
  });

  it('identifies warrants (WW suffix)', () => {
    expect(isDerivativeSecurity('ABPWW')).toBe(true);
  });

  it('identifies rights (R suffix)', () => {
    expect(isDerivativeSecurity('AACBR')).toBe(true);
  });

  it('identifies units (U suffix)', () => {
    expect(isDerivativeSecurity('AACBU')).toBe(true);
  });

  it('identifies preferred shares (N/O/P/Q suffix)', () => {
    expect(isDerivativeSecurity('ACGLN')).toBe(true);
    expect(isDerivativeSecurity('ACGLO')).toBe(true);
    expect(isDerivativeSecurity('ACGLP')).toBe(true);
    expect(isDerivativeSecurity('ACGLQ')).toBe(true);
  });

  it('does not flag short tickers ending in preferred letters', () => {
    // 4 chars or less should not be flagged as preferred
    expect(isDerivativeSecurity('AMZN')).toBe(false);
  });

  it('is case insensitive', () => {
    expect(isDerivativeSecurity('abpww')).toBe(true);
    expect(isDerivativeSecurity('aacbr')).toBe(true);
  });

  it('does not flag long R-suffix tickers (length > 6)', () => {
    expect(isDerivativeSecurity('LONGTICKERR')).toBe(false);
  });
});

describe('getSecurityType', () => {
  it('returns "Unknown" for empty string', () => {
    expect(getSecurityType('')).toBe('Unknown');
  });

  it('returns "Common Stock" for regular tickers', () => {
    expect(getSecurityType('AAPL')).toBe('Common Stock');
    expect(getSecurityType('TSLA')).toBe('Common Stock');
  });

  it('returns "Warrant" for W suffix', () => {
    expect(getSecurityType('ABPW')).toBe('Warrant');
  });

  it('returns "Warrant" for WW suffix', () => {
    expect(getSecurityType('ABPWW')).toBe('Warrant');
  });

  it('returns "Rights" for R suffix', () => {
    expect(getSecurityType('AACBR')).toBe('Rights');
  });

  it('returns "Unit (Stock + Warrant)" for U suffix', () => {
    expect(getSecurityType('AACBU')).toBe('Unit (Stock + Warrant)');
  });

  it('returns correct preferred series label', () => {
    expect(getSecurityType('ACGLN')).toBe('Preferred Stock (Series N)');
    expect(getSecurityType('ACGLP')).toBe('Preferred Stock (Series P)');
  });
});

describe('filterDerivatives', () => {
  it('filters out derivative securities', () => {
    const data = [
      { ticker: 'AAPL', price: 150 },
      { ticker: 'ABPWW', price: 2 },
      { ticker: 'MSFT', price: 300 },
      { ticker: 'AACBR', price: 1 },
    ];

    const filtered = filterDerivatives(data);
    expect(filtered).toHaveLength(2);
    expect(filtered.map(d => d.ticker)).toEqual(['AAPL', 'MSFT']);
  });

  it('keeps all items when no derivatives', () => {
    const data = [
      { ticker: 'AAPL', price: 150 },
      { ticker: 'GOOG', price: 140 },
    ];

    expect(filterDerivatives(data)).toHaveLength(2);
  });

  it('supports custom ticker field', () => {
    const data = [
      { symbol: 'AAPL', price: 150 },
      { symbol: 'ABPWW', price: 2 },
    ];

    const filtered = filterDerivatives(data, 'symbol');
    expect(filtered).toHaveLength(1);
    expect(filtered[0].symbol).toBe('AAPL');
  });

  it('keeps items without valid ticker field', () => {
    const data = [
      { ticker: 123 as any, price: 50 },
    ];

    expect(filterDerivatives(data)).toHaveLength(1);
  });

  it('handles empty array', () => {
    expect(filterDerivatives([])).toEqual([]);
  });
});

describe('categorizeTickers', () => {
  it('separates common stocks from derivatives', () => {
    const data = [
      { ticker: 'AAPL', price: 150 },
      { ticker: 'ABPWW', price: 2 },
      { ticker: 'MSFT', price: 300 },
    ];

    const { commonStocks, derivatives } = categorizeTickers(data);
    expect(commonStocks).toHaveLength(2);
    expect(derivatives).toHaveLength(1);
    expect(derivatives[0].ticker).toBe('ABPWW');
  });

  it('handles all common stocks', () => {
    const data = [
      { ticker: 'AAPL', price: 150 },
      { ticker: 'MSFT', price: 300 },
    ];

    const { commonStocks, derivatives } = categorizeTickers(data);
    expect(commonStocks).toHaveLength(2);
    expect(derivatives).toHaveLength(0);
  });

  it('handles empty array', () => {
    const { commonStocks, derivatives } = categorizeTickers([]);
    expect(commonStocks).toEqual([]);
    expect(derivatives).toEqual([]);
  });

  it('puts items without valid ticker into commonStocks', () => {
    const data = [
      { ticker: null as any, price: 50 },
    ];

    const { commonStocks } = categorizeTickers(data);
    expect(commonStocks).toHaveLength(1);
  });
});
